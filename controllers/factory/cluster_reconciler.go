package factory

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"

	"github.com/aws/eks-anywhere/controllers/controllers/cilium"
	"github.com/aws/eks-anywhere/controllers/controllers/clusters"
	"github.com/aws/eks-anywhere/controllers/controllers/registry"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/dependencies"
)

const (
	snowProviderName    = "snow"
	vSphereProviderName = "vsphere"
)

func NewFactory() *dependencies.Factory {
	// TODO: hacky, add ability to pass this option to the dep factory and the executable builder
	os.Setenv("MR_TOOLS_DISABLE", "true")
	return dependencies.NewFactory().UseExecutableImage("test.com/fake-image:1.0")
}

func BuildClusterReconcilerRegistry(ctx context.Context, log logr.Logger, client client.Client, manager manager.Manager, dependencyFactory *dependencies.Factory) (registry.ClusterReconcilerRegistry, error) {
	factory := newClusterReconcilerRegistryFactory(log, client, manager, dependencyFactory)
	factory.withSnowReconciler()
	return factory.build(ctx)

	// TODO: figure out how to use the API before the client cached in initialized (which only happens after the manager is started)
	// We want to build the reconcilers before the manager is started so we can inject it in the cluster reconciler constructor
	// And this happens before we call manager.Start()

	providers := &clusterctlv1.ProviderList{}
	err := client.List(ctx, providers)
	if err != nil {
		return registry.ClusterReconcilerRegistry{}, err
	}

	for _, p := range providers.Items {
		if p.Type != string(clusterctlv1.InfrastructureProviderType) {
			continue
		}

		switch p.ProviderName {
		case snowProviderName:
			factory.withSnowReconciler()
		case vSphereProviderName:
			factory.withVSphereReconciler()
		default:
			log.Info("Found unknown CAPI provider, ignoring", "providerType", p.ProviderName)
		}
	}

	return factory.build(ctx)
}

type buildStep func(ctx context.Context) error

type clusterReconcilerRegistryFactory struct {
	builder           registry.Builder
	dependencyFactory *dependencies.Factory
	log               logr.Logger
	client            client.Client
	manager           manager.Manager
	ciliumReconciler  clusters.CiliumReconciler
	tracker           *remote.ClusterCacheTracker
	buildSteps        []buildStep
	deps              *dependencies.Dependencies
}

func newClusterReconcilerRegistryFactory(log logr.Logger, client client.Client, manager manager.Manager, dependencyFactory *dependencies.Factory) *clusterReconcilerRegistryFactory {
	return &clusterReconcilerRegistryFactory{
		builder:           registry.NewBuilder(),
		dependencyFactory: dependencyFactory,
		log:               log,
		client:            client,
		manager:           manager,
	}
}

func (f *clusterReconcilerRegistryFactory) withSnowReconciler() *clusterReconcilerRegistryFactory {
	f.dependencyFactory.WithHelm()
	f.withTracker().withCiliumReconciler()

	f.buildSteps = append(f.buildSteps, func(ctx context.Context) error {
		f.builder.Add(anywherev1.SnowDatacenterKind, clusters.NewSnowClusterReconciler(
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
	// TODO: implement
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

func (f *clusterReconcilerRegistryFactory) build(ctx context.Context) (registry.ClusterReconcilerRegistry, error) {
	deps, err := f.dependencyFactory.Build(ctx)
	if err != nil {
		return registry.ClusterReconcilerRegistry{}, err
	}

	f.deps = deps

	for _, step := range f.buildSteps {
		if err := step(ctx); err != nil {
			return registry.ClusterReconcilerRegistry{}, err
		}
	}

	f.buildSteps = make([]buildStep, 0)

	return f.builder.Build(), nil
}
