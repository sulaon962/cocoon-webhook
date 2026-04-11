package admission

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/projecteru2/core/log"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	commonadmission "github.com/cocoonstack/cocoon-common/k8s/admission"
	"github.com/cocoonstack/cocoon-common/meta"
	"github.com/cocoonstack/cocoon-webhook/metrics"
)

// scalable is the small subset of appsv1.Deployment / appsv1.StatefulSet
// that the workload validator needs: replica count + the pod
// template's tolerations. The generic helper below uses it as the
// type constraint so the two scale-down code paths share one
// implementation.
type scalable interface {
	appsv1.Deployment | appsv1.StatefulSet
}

// validateWorkload is the admission entry point for Deployment and
// StatefulSet UPDATE. It rejects scale-down on cocoon workloads
// because the agents are stateful VMs — reducing replicas would
// destroy state. Operators should use a CocoonHibernation CR to
// suspend an individual agent instead.
//
// Two request shapes reach this handler:
//
//   - Direct PUT/PATCH on the parent: req.Kind = Deployment/StatefulSet,
//     req.SubResource = "". The request body carries the full object,
//     including the pod template tolerations we use to filter cocoon
//     workloads.
//   - kubectl scale: req.Resource = deployments/statefulsets,
//     req.SubResource = "scale", req.Kind = autoscaling/v1 Scale. The
//     request body is just a Scale, so we have to fetch the parent
//     out of band to read its tolerations.
func (s *Server) validateWorkload(ctx context.Context, review *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	req := review.Request
	if req.Operation != admissionv1.Update {
		return commonadmission.Allow()
	}
	if req.SubResource == "scale" {
		return s.validateScaleSubresource(ctx, req)
	}
	switch req.Kind.Kind {
	case "Deployment":
		return validateScaleDown[appsv1.Deployment](ctx, req)
	case "StatefulSet":
		return validateScaleDown[appsv1.StatefulSet](ctx, req)
	default:
		return commonadmission.Allow()
	}
}

// validateScaleSubresource handles UPDATEs to deployments/scale and
// statefulsets/scale. The Scale object only carries spec.replicas, so
// we look up the parent workload from the apiserver to read its
// tolerations and decide whether the cocoon scale-down rule applies.
//
// Failure modes are deliberately permissive (Allow + warning log)
// rather than Fail-closed: an unreachable apiserver would otherwise
// take down `kubectl scale` for every workload in the cluster, not
// just cocoon ones. The mutating-webhook side is the only place that
// has to be strict.
func (s *Server) validateScaleSubresource(ctx context.Context, req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	logger := log.WithFunc("validateScaleSubresource")

	var oldScale, newScale autoscalingv1.Scale
	if err := json.Unmarshal(req.OldObject.Raw, &oldScale); err != nil {
		logger.Warnf(ctx, "decode old Scale %s/%s: %v", req.Namespace, req.Name, err)
		return commonadmission.Allow()
	}
	if err := json.Unmarshal(req.Object.Raw, &newScale); err != nil {
		logger.Warnf(ctx, "decode new Scale %s/%s: %v", req.Namespace, req.Name, err)
		return commonadmission.Allow()
	}

	tolerations, ok := s.fetchParentTolerations(ctx, req)
	if !ok {
		// Fetch failed; we already logged the underlying cause. Allow
		// the request rather than wedging cluster scaling on a
		// transient apiserver hiccup.
		return commonadmission.Allow()
	}
	if !meta.HasCocoonToleration(tolerations) {
		return commonadmission.Allow()
	}
	return checkScaleDown(ctx, req, oldScale.Spec.Replicas, newScale.Spec.Replicas)
}

// fetchParentTolerations resolves req.Resource to the parent
// Deployment / StatefulSet and returns its pod-template tolerations.
// Returns ok=false if the resource is not one we know about or the
// API call failed for any reason other than NotFound (NotFound also
// returns false: nothing to enforce against a missing parent).
func (s *Server) fetchParentTolerations(ctx context.Context, req *admissionv1.AdmissionRequest) ([]corev1.Toleration, bool) {
	logger := log.WithFunc("fetchParentTolerations")
	switch req.Resource.Resource {
	case "deployments":
		dep, err := s.client.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Warnf(ctx, "get parent Deployment %s/%s: %v", req.Namespace, req.Name, err)
			}
			return nil, false
		}
		return dep.Spec.Template.Spec.Tolerations, true
	case "statefulsets":
		sts, err := s.client.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Warnf(ctx, "get parent StatefulSet %s/%s: %v", req.Namespace, req.Name, err)
			}
			return nil, false
		}
		return sts.Spec.Template.Spec.Tolerations, true
	default:
		return nil, false
	}
}

// validateScaleDown decodes the old and new objects, applies the
// cocoon-toleration filter, and runs checkScaleDown.
func validateScaleDown[T scalable](ctx context.Context, req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	logger := log.WithFunc("validateScaleDown")
	var oldObj, newObj T
	if err := json.Unmarshal(req.OldObject.Raw, &oldObj); err != nil {
		logger.Warnf(ctx, "decode old %s %s/%s: %v", req.Kind.Kind, req.Namespace, req.Name, err)
		return commonadmission.Allow()
	}
	if err := json.Unmarshal(req.Object.Raw, &newObj); err != nil {
		logger.Warnf(ctx, "decode new %s %s/%s: %v", req.Kind.Kind, req.Namespace, req.Name, err)
		return commonadmission.Allow()
	}

	if !meta.HasCocoonToleration(workloadTolerations(&newObj)) {
		return commonadmission.Allow()
	}
	return checkScaleDown(ctx, req, workloadReplicas(&oldObj), workloadReplicas(&newObj))
}

// workloadReplicas extracts the replica count from either Deployment
// or StatefulSet via a type switch on the generic argument. The
// apps controllers default a nil pointer to 1, so we do too.
func workloadReplicas[T scalable](obj *T) int32 {
	switch v := any(obj).(type) {
	case *appsv1.Deployment:
		return ptr.Deref(v.Spec.Replicas, 1)
	case *appsv1.StatefulSet:
		return ptr.Deref(v.Spec.Replicas, 1)
	default:
		return 0
	}
}

// workloadTolerations returns the pod-template tolerations of the
// workload — used to filter out non-cocoon workloads early.
func workloadTolerations[T scalable](obj *T) []corev1.Toleration {
	switch v := any(obj).(type) {
	case *appsv1.Deployment:
		return v.Spec.Template.Spec.Tolerations
	case *appsv1.StatefulSet:
		return v.Spec.Template.Spec.Tolerations
	default:
		return nil
	}
}

func checkScaleDown(ctx context.Context, req *admissionv1.AdmissionRequest, oldReplicas, newReplicas int32) *admissionv1.AdmissionResponse {
	if newReplicas >= oldReplicas {
		metrics.RecordAdmission(metrics.HandlerValidate, metrics.DecisionAllow)
		return commonadmission.Allow()
	}
	msg := fmt.Sprintf(
		"cocoon-webhook: scale-down blocked for cocoon %s %s/%s (%d -> %d). "+
			"Use a CocoonHibernation CR to suspend individual agents.",
		req.Kind.Kind, req.Namespace, req.Name, oldReplicas, newReplicas)
	log.WithFunc("checkScaleDown").Warn(ctx, msg)
	metrics.RecordAdmission(metrics.HandlerValidate, metrics.DecisionDeny)
	return commonadmission.Deny(msg)
}
