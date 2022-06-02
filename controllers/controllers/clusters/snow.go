package clusters

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/eks-anywhere/controllers/controllers/clients"
	"github.com/aws/eks-anywhere/controllers/controllers/reconciler"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/providers/snow"
)

type SnowClusterReconciler struct {
	*providerClusterReconciler
}

func NewSnowClusterReconciler(client client.Client, tracker *remote.ClusterCacheTracker, ciliumReconciler CiliumReconciler) *SnowClusterReconciler {
	return &SnowClusterReconciler{
		providerClusterReconciler: newProviderClusterReconciler(client, tracker, ciliumReconciler),
	}
}

func (s *SnowClusterReconciler) Reconcile(ctx context.Context, log logr.Logger, cluster *anywherev1.Cluster) (reconciler.Result, error) {
	log = log.WithValues("provider", "snow")
	clusterSpec, err := s.GetClusterSpec(ctx, log, cluster)
	if err != nil {
		return reconciler.Result{}, err
	}

	return newRunner().register(
		s.reconcileControlPlane,
		s.checkControlPlaneReady,
		s.reconcileCilium,
		s.reconcileWorkers,
	).run(ctx, log, clusterSpec)
}

func (s *SnowClusterReconciler) reconcileControlPlane(ctx context.Context, log logr.Logger, clusterSpec *cluster.Spec) (reconciler.Result, error) {
	log = log.WithValues("phase", "reconcileControlPlane")
	log.Info("Generating control plane CAPI objects")
	controlPlaneObjs, err := snow.ControlPlaneObjects(ctx, clusterSpec, clients.NewKubeClient(s.client))
	if err != nil {
		return reconciler.Result{}, err
	}

	log.Info("Applying control plane objects")
	if err = reconciler.ReconcileObjects(ctx, s.client, controlPlaneObjs.ClientObjects()); err != nil {
		return reconciler.Result{}, err
	}

	return reconciler.Result{}, nil
}

func (s *SnowClusterReconciler) reconcileWorkers(ctx context.Context, log logr.Logger, clusterSpec *cluster.Spec) (reconciler.Result, error) {
	log = log.WithValues("phase", "reconcileWorkers")
	log.Info("Generating worker CAPI objects")
	workerObjs, err := snow.WorkersObjects(ctx, clusterSpec, clients.NewKubeClient(s.client))
	if err != nil {
		return reconciler.Result{}, err
	}

	log.Info("Applying worker CAPI objects")
	if err = reconciler.ReconcileObjects(ctx, s.client, workerObjs); err != nil {
		return reconciler.Result{}, err
	}

	return reconciler.Result{}, nil
}
