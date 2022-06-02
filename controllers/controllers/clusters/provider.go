package clusters

import (
	"context"
	"fmt"
	"time"

	eksdv1alpha1 "github.com/aws/eks-distro-build-tooling/release/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/aws/eks-anywhere/controllers/controllers/reconciler"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/clusterapi"
	"github.com/aws/eks-anywhere/pkg/constants"
	"github.com/aws/eks-anywhere/pkg/providers/vsphere"
	releasev1alpha1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
)

type ProviderClusterReconciler interface {
	Reconcile(ctx context.Context, cluster *anywherev1.Cluster) (reconciler.Result, error)
}

func BuildProviderReconciler(datacenterKind string, client client.Client, log logr.Logger, validator *vsphere.Validator, defaulter *vsphere.Defaulter, tracker *remote.ClusterCacheTracker) (ProviderClusterReconciler, error) {
	switch datacenterKind {
	case anywherev1.VSphereDatacenterKind:
		return NewVSphereReconciler(client, log, validator, defaulter, tracker), nil
	}
	return nil, fmt.Errorf("invalid data center type %s", datacenterKind)
}

func newProviderClusterReconciler(client client.Client, tracker *remote.ClusterCacheTracker, ciliumReconciler CiliumReconciler) *providerClusterReconciler {
	return &providerClusterReconciler{
		client:           client,
		tracker:          tracker,
		ciliumReconciler: ciliumReconciler,
	}
}

type providerClusterReconciler struct {
	client           client.Client
	tracker          *remote.ClusterCacheTracker
	ciliumReconciler CiliumReconciler
}

type CiliumReconciler interface {
	Reconcile(ctx context.Context, log logr.Logger, client client.Client, clusterSpec *cluster.Spec) (reconciler.Result, error)
}

func (p *providerClusterReconciler) eksdRelease(ctx context.Context, name, namespace string) (*eksdv1alpha1.Release, error) {
	eksd := &eksdv1alpha1.Release{}
	releaseName := types.NamespacedName{Namespace: namespace, Name: name}

	if err := p.client.Get(ctx, releaseName, eksd); err != nil {
		return nil, err
	}

	return eksd, nil
}

func (p *providerClusterReconciler) bundles(ctx context.Context, name, namespace string) (*releasev1alpha1.Bundles, error) {
	clusterBundle := &releasev1alpha1.Bundles{}
	bundleName := types.NamespacedName{Namespace: namespace, Name: name}

	if err := p.client.Get(ctx, bundleName, clusterBundle); err != nil {
		return nil, err
	}

	return clusterBundle, nil
}

func (p *providerClusterReconciler) GetClusterSpec(ctx context.Context, log logr.Logger, cs *anywherev1.Cluster) (*cluster.Spec, error) {
	managementCluster := cs
	if cs.IsManaged() {
		// TODO: super hacky: since we look for the bundle based on name (bundle has the same name as the management cluster)
		// we create a copy of the cluster and rename it to the management cluster name, then undo the changes before returning
		managementCluster = cs.DeepCopy()
		managementCluster.Name = cs.Spec.ManagementCluster.Name
	}

	spec, err := cluster.BuildSpecForCluster(ctx, managementCluster, p.bundles, p.eksdRelease, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	spec.Cluster = cs
	return p.populateClusterSpec(ctx, log, spec)
}

func (p *providerClusterReconciler) getCAPICluster(ctx context.Context, cluster *anywherev1.Cluster) (*clusterv1.Cluster, error) {
	capiClusterName := clusterapi.ClusterName(cluster)

	capiCluster := &clusterv1.Cluster{}
	key := types.NamespacedName{Namespace: constants.EksaSystemNamespace, Name: capiClusterName}

	err := p.client.Get(ctx, key, capiCluster)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	return capiCluster, nil
}

func (p *providerClusterReconciler) checkControlPlaneReady(ctx context.Context, log logr.Logger, clusterSpec *cluster.Spec) (reconciler.Result, error) {
	log = log.WithValues("phase", "checkControlPlaneReady")
	capiCluster, err := p.getCAPICluster(ctx, clusterSpec.Cluster)
	if err != nil {
		return reconciler.Result{}, err
	}

	if capiCluster == nil {
		log.Info("CAPI cluster does not exist yet, requeuing")
		return reconciler.ResultWithRequeue(5 * time.Second), nil
	}

	if !conditions.IsTrue(capiCluster, controlPlaneReadyCondition) {
		log.Info("CAPI control plane is not ready yet")
		return reconciler.ResultWithReturn(), nil
	}

	log.Info("CAPI control plane is ready!")
	return reconciler.Result{}, nil
}

func (p *providerClusterReconciler) reconcileCilium(ctx context.Context, log logr.Logger, clusterSpec *cluster.Spec) (reconciler.Result, error) {
	log = log.WithValues("phase", "reconcileCilium")
	capiCluster, err := p.getCAPICluster(ctx, clusterSpec.Cluster)
	if err != nil {
		return reconciler.Result{}, err
	}

	log.Info("Getting remote client", "capiCluster", capiCluster.Name)
	key := client.ObjectKey{
		Namespace: capiCluster.Namespace,
		Name:      capiCluster.Name,
	}
	remoteClient, err := p.tracker.GetClient(ctx, key)
	if err != nil {
		return reconciler.Result{}, err
	}

	return p.ciliumReconciler.Reconcile(ctx, log, remoteClient, clusterSpec)
}

func (p *providerClusterReconciler) populateClusterSpec(ctx context.Context, log logr.Logger, clusterSpec *cluster.Spec) (*cluster.Spec, error) {
	log.Info("Populating clusterSpec with cluster objects")
	if clusterSpec.Cluster.Spec.DatacenterRef.Kind != anywherev1.SnowDatacenterKind {
		log.Info("ClusterSpec building not supported for provider, reconcile might fail", "kind", clusterSpec.Cluster.Spec.DatacenterRef)
	}
	machineConfigRefs := clusterSpec.Cluster.MachineConfigRefs()
	if clusterSpec.Config.SnowMachineConfigs == nil {
		clusterSpec.Config.SnowMachineConfigs = make(map[string]*anywherev1.SnowMachineConfig, len(machineConfigRefs))
	}

	log.Info("Populating clusterSpec with Snow machine configs")
	for _, m := range machineConfigRefs {
		machine := &anywherev1.SnowMachineConfig{}
		key := types.NamespacedName{Namespace: clusterSpec.Cluster.Namespace, Name: m.Name}
		if err := p.client.Get(ctx, key, machine); err != nil {
			return nil, err
		}

		clusterSpec.Config.SnowMachineConfigs[machine.Name] = machine
	}

	return clusterSpec, nil
}
