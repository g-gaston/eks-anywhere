//go:build base-option

package cluster

import (
	"context"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/workflow/contextutil"
)

// TODO: maybe this package name is too generic? I also thought about calling this `workload`
// But I was afraid it might be mistaken for tasks for "eks-a workload" clusters, while this is intended
// for both management and workload clusters

type ProviderClusterCreator interface {
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
