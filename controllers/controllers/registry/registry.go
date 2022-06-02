package registry

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/aws/eks-anywhere/controllers/controllers/reconciler"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
)

type ProviderClusterReconciler interface {
	Reconcile(ctx context.Context, log logr.Logger, cluster *anywherev1.Cluster) (reconciler.Result, error)
}

type ClusterReconcilerRegistry struct {
	reconcilers map[string]ProviderClusterReconciler
}

func NewClusterReconcilerRegistry() ClusterReconcilerRegistry {
	return ClusterReconcilerRegistry{
		reconcilers: map[string]ProviderClusterReconciler{},
	}
}

func (r *ClusterReconcilerRegistry) add(kind string, reconciler ProviderClusterReconciler) {
	r.reconcilers[kind] = reconciler
}

func (r *ClusterReconcilerRegistry) Get(kind string) ProviderClusterReconciler {
	return r.reconcilers[kind]
}
