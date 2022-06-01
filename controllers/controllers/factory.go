package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

		clusterapiv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/aws/eks-anywhere/controllers/controllers/cilium"
	"github.com/aws/eks-anywhere/controllers/controllers/clusters"
	"github.com/aws/eks-anywhere/controllers/controllers/reconciler"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/dependencies"
)

type ProviderClusterReconciler interface {
	Reconcile(ctx context.Context, log logr.Logger, cluster *anywherev1.Cluster) (reconciler.Result, error)
}

type ClusterReconcilerRegistry struct {
	reconcilers map[string]ProviderClusterReconciler
}

func (r *ClusterReconcilerRegistry) add(kind string, reconciler ProviderClusterReconciler) {
}

func BuildClusterReconcilerRegistry(ctx context.Context, client client.Client) (*ClusterReconcilerRegistry, error) {
	client.List(ctx, clusterapiv1.ProviderLabelName)

	return nil, nil
}

type buildStep func(ctx context.Context) error

type clusterReconcilerRegistryFactory struct {
	reconcilerRegistry *ClusterReconcilerRegistry
	dependencyFactory  *dependencies.Factory
	ciliumReconciler   clusters.CiliumReconciler
	tracker            *remote.ClusterCacheTracker
	client             client.Client
	manager            manager.Manager
	log                logr.Logger
	buildSteps         []buildStep
	deps               *dependencies.Dependencies
}

func (f *clusterReconcilerRegistryFactory) withSnowReconciler() *clusterReconcilerRegistryFactory {
	f.dependencyFactory.WithHelm()
	f.withTracker()

	f.buildSteps = append(f.buildSteps, func(ctx context.Context) error {
		f.reconcilerRegistry.add(anywherev1.SnowDatacenterKind, clusters.NewSnowClusterReconciler(
			f.client,
			f.tracker,
			f.ciliumReconciler,
		))
		return nil
	})
	return f
}

func (f *clusterReconcilerRegistryFactory) withVSphereReconciler() *clusterReconcilerRegistryFactory {
	f.dependencyFactory.WithGovc()
	return f
}

func (f *clusterReconcilerRegistryFactory) withCiliumReconciler() *clusterReconcilerRegistryFactory {
	f.dependencyFactory.WithCilium()

	f.buildSteps = append(f.buildSteps, func(ctx context.Context) error {
		if f.ciliumReconciler != nil {
			return nil
		}

		f.ciliumReconciler = cilium.NewReconciler(f.deps.Cilium)
		return nil
	})
	return f
}

func (f *clusterReconcilerRegistryFactory) withTracker() *clusterReconcilerRegistryFactory {
	f.buildSteps = append(f.buildSteps, func(ctx context.Context) error {
		if f.tracker != nil {
			return nil
		}

		logger := f.log.WithName("remote").WithName("ClusterCacheTracker")
		tracker, err := remote.NewClusterCacheTracker(
			f.manager,
			remote.ClusterCacheTrackerOptions{
				Log:     &logger,
				Indexes: remote.DefaultIndexes,
			},
		)
		if err != nil {
			return err
		}

		f.tracker = tracker

		return nil
	})
	return f
}

func (f *clusterReconcilerRegistryFactory) build(ctx context.Context) (*ClusterReconcilerRegistry, error) {
	deps, err := f.dependencyFactory.Build(ctx)
	if err != nil {
		return nil, err
	}

	f.deps = deps

	for _, step := range f.buildSteps {
		if err := step(ctx); err != nil {
			return nil, err
		}
	}

	f.buildSteps = make([]buildStep, 0)

	return f.reconcilerRegistry, nil
}
