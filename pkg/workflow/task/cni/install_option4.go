//go:build option-4

package cni

import (
	"context"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/workflow/contextutil"
)

type CNIInstaller interface {
	// Install configures a CNI in a kubernetes cluster
	Install(ctx context.Context, cluster types.Cluster, spec *cluster.Spec) error
}

type Install struct {
	Spec *cluster.Spec
	CNI  CNIInstaller
}

func (t Install) RunTask(ctx context.Context) (context.Context, error) {
	targetCluster := contextutil.TargetCluster(ctx)
	return ctx, t.CNI.Install(ctx, targetCluster, t.Spec)
}
