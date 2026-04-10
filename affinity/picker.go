package affinity

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"

	"github.com/cocoonstack/cocoon-common/meta"
)

// ByNodeIndex is the informer-cache index name used to look up pods
// by spec.nodeName. Registered once in main before the factory starts.
const ByNodeIndex = "byNode"

// NodeNameIndexFunc is the cache.IndexFunc for ByNodeIndex.
// Pass it to podInformer.AddIndexers so the LeastUsedPicker can
// look up pods by spec.nodeName.
func NodeNameIndexFunc(obj any) ([]string, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, nil
	}
	if pod.Spec.NodeName == "" {
		return nil, nil
	}
	return []string{pod.Spec.NodeName}, nil
}

// PodIndexer looks up pods by the node they are scheduled on.
// The informer's ByIndex method satisfies this when wired to the
// ByNodeIndex indexer registered at startup.
type PodIndexer interface {
	ByIndex(indexName, indexedValue string) ([]any, error)
}

var _ NodePicker = (*LeastUsedPicker)(nil)

// LeastUsedPicker picks the cocoon node in a pool that currently
// hosts the fewest pods. Ties are broken alphabetically so the
// outcome is deterministic across multiple webhook replicas.
//
// Both the node lookup (by pool label) and the pod count are served
// from shared informer caches, so a Pick is a couple of in-memory
// scans rather than two apiserver round trips on the admission hot
// path.
type LeastUsedPicker struct {
	PodIndexer PodIndexer
	NodeLister corelisters.NodeLister
}

// NewLeastUsedPicker constructs a LeastUsedPicker that reads from
// the supplied informer-backed listers. Callers are responsible for
// starting the informer factory and waiting for cache sync before
// calling Pick.
func NewLeastUsedPicker(pods PodIndexer, nodes corelisters.NodeLister) *LeastUsedPicker {
	return &LeastUsedPicker{PodIndexer: pods, NodeLister: nodes}
}

// Pick returns the name of the cocoon node in the pool that has the
// fewest pods scheduled to it. Returns "" (with no error) when the
// pool has no nodes — callers should treat that as "let the
// scheduler decide".
func (p *LeastUsedPicker) Pick(_ context.Context, pool string) (string, error) {
	if pool == "" {
		return "", fmt.Errorf("pick node: pool must not be empty")
	}

	poolSelector := labels.SelectorFromSet(labels.Set{meta.LabelNodePool: pool})
	nodes, err := p.NodeLister.List(poolSelector)
	if err != nil {
		return "", fmt.Errorf("list nodes for pool %s: %w", pool, err)
	}
	if len(nodes) == 0 {
		return "", nil
	}

	counts, err := p.podsPerNode(nodes)
	if err != nil {
		return "", err
	}

	slices.SortStableFunc(nodes, func(a, b *corev1.Node) int {
		return cmp.Or(
			cmp.Compare(counts[a.Name], counts[b.Name]),
			cmp.Compare(a.Name, b.Name),
		)
	})
	return nodes[0].Name, nil
}

// podsPerNode counts the live pods currently scheduled to each
// candidate node. Excluded: pods in Succeeded / Failed phases (they
// no longer consume capacity).
//
// Each lookup is O(pods-on-that-node) via the ByNodeIndex informer
// index, so the total cost is proportional to the pods on the
// candidate nodes — not every pod in the cluster.
func (p *LeastUsedPicker) podsPerNode(nodes []*corev1.Node) (map[string]int, error) {
	counts := make(map[string]int, len(nodes))
	for _, n := range nodes {
		objs, err := p.PodIndexer.ByIndex(ByNodeIndex, n.Name)
		if err != nil {
			return nil, fmt.Errorf("index lookup for node %s: %w", n.Name, err)
		}
		live := 0
		for _, obj := range objs {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				continue
			}
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				continue
			}
			live++
		}
		counts[n.Name] = live
	}
	return counts, nil
}
