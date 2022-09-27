package cni

import (
	"context"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/workflow/contextutil"
)

type CNIInstaller interface {
	// Install configures a CNI in a kubernetes cluster
	// allowedNamespaces permits specifying a list of namespaces to/from which traffic is allowed. This only
	// applies if the CNI is configured with some kind of default network policy restriction (eg. Cilium with PolicyEnforcementMode)
	Install(ctx context.Context, cluster types.Cluster, spec *cluster.Spec, allowedNamespaces []string) error
}

type ProviderNamespaceGetter interface {
	GetNamespaces() []string
}

type Install struct {
	Spec            *cluster.Spec
	NamespaceGetter ProviderNamespaceGetter
	CNI             CNIInstaller
}

func (t Install) RunTask(ctx context.Context) (context.Context, error) {
	targetCluster := contextutil.TargetCluster(ctx)
	return ctx, t.CNI.Install(ctx, targetCluster, t.Spec, t.NamespaceGetter.GetNamespaces())
}
