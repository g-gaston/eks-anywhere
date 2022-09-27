//go:build option-1

package workload

// TODO: maybe this package name is confusing?
// I was afraid it might be mistaken for tasks for "eks-a workload" clusters, while this is intended
// for both management and workload clusters

import (
	"context"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/workflow/contextutil"
)

type ProviderClusterCreator interface {
	// CreateCluster uses a CAPI management cluster to create a provider specific cluster
	// based on an EKS-A cluster Spec. It installs all the necessary components to make the conformant cluster (CNI, CPI, etc.)
	CreateCluster(ctx context.Context, managementCluster types.Cluster, spec *cluster.Spec) (types.Cluster, error)
}

type Create struct {
	Spec    *cluster.Spec
	Creator ProviderClusterCreator
}

func (t Create) RunTask(ctx context.Context) (context.Context, error) {
	cluster, err := t.Creator.CreateCluster(ctx, contextutil.BootstrapCluster(ctx), t.Spec)
	if err != nil {
		return nil, err
	}

	return contextutil.WithTargetCluster(ctx, cluster), nil
}
