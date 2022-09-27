//go:build option-2

package workload

// TODO: maybe this package name is confusing?
// I was afraid it might be mistaken for tasks for "eks-a workload" clusters, while this is intended
// for both management and workload clusters

import (
	"context"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/filewriter"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/workflow/contextutil"
)

type ProviderClusterCreator interface {
	// CreateControlPlane triggers the creation of a k8s ControlPlane using CAPI. It doesn't wait for such process to succeed
	CreateControlPlane(ctx context.Context, managementCluster types.Cluster, spec *cluster.Spec) (*clusterv1.Cluster, error)
	// CreateControlPlane triggers the creation of k8s worker nodes using CAPI. It doesn't wait for such process to succeed
	CreateDataPlane(ctx context.Context, managementCluster types.Cluster, targetCluster *clusterv1.Cluster, spec *cluster.Spec) error
}

type ProviderNamespaceGetter interface {
	GetNamespaces() []string
}

// TODO: better name?
type CAPIClusterWatcher interface {
	WaitForEtcdReady(ctx context.Context, managementCluster types.Cluster, cluster *clusterv1.Cluster) error
	WaitForControlPlaneReady(ctx context.Context, managementCluster types.Cluster, cluster *clusterv1.Cluster) error
}

type KubeconfigRetriever interface {
	GetKubeconfig(ctx context.Context, managementCluster types.Cluster, cluster *clusterv1.Cluster) ([]byte, error)
}

type CNIInstaller interface {
	// Install configures a CNI in a kubernetes cluster
	// allowedNamespaces permits specifying a list of namespaces to/from which traffic is allowed. This only
	// applies if the CNI is configured with some kind of default network policy restriction (eg. Cilium with PolicyEnforcementMode)
	Install(ctx context.Context, cluster types.Cluster, spec *cluster.Spec, allowedNamespaces []string) error
}

// Create spins up a conformant cluster based on a EKS-A spec using CAPI
type Create struct {
	Spec                *cluster.Spec
	Creator             ProviderClusterCreator
	NamespaceGetter     ProviderNamespaceGetter
	Watcher             CAPIClusterWatcher
	KubeconfigRetriever KubeconfigRetriever
	CNI                 CNIInstaller
	Writer              filewriter.FileWriter
}

func (t Create) RunTask(ctx context.Context) (context.Context, error) {
	// TODO: validate management cluster exists
	// TODO: figure out a way to retrieve the management cluster, no matter if it's a bootstrap cluster or eks-a management cluster
	// That will make this task reusable across workflows
	managementCluster := contextutil.BootstrapCluster(ctx)
	capiCluster, err := t.Creator.CreateControlPlane(ctx, managementCluster, t.Spec)
	if err != nil {
		return nil, err
	}

	if err = t.Watcher.WaitForEtcdReady(ctx, managementCluster, capiCluster); err != nil {
		return nil, err
	}

	if err = t.Watcher.WaitForControlPlaneReady(ctx, managementCluster, capiCluster); err != nil {
		return nil, err
	}

	kubeconfigContent, err := t.KubeconfigRetriever.GetKubeconfig(ctx, managementCluster, capiCluster)
	if err != nil {
		return nil, err
	}

	// TODO: change path
	path, err := t.Writer.Write("config.kubeconfig", kubeconfigContent)
	if err != nil {
		return nil, err
	}

	targetCluster := types.Cluster{
		Name:       capiCluster.Name,
		Kubeconfig: path,
	}

	ctx = contextutil.WithTargetCluster(ctx, targetCluster)

	if err = t.CNI.Install(ctx, targetCluster, t.Spec, t.NamespaceGetter.GetNamespaces()); err != nil {
		return nil, err
	}

	if err = t.Creator.CreateDataPlane(ctx, managementCluster, capiCluster, t.Spec); err != nil {
		return nil, err
	}

	return ctx, nil
}
