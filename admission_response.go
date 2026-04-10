package main

import (
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// allowResponse returns a permissive AdmissionResponse with no patch.
func allowResponse() *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{Allowed: true}
}

// denyResponse returns a forbidden AdmissionResponse carrying the
// supplied human-readable message.
func denyResponse(msg string) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: msg,
			Reason:  metav1.StatusReasonForbidden,
		},
	}
}
