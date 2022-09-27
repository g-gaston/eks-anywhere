package contextutil

import (
	"context"

	"github.com/aws/eks-anywhere/pkg/types"
)

// bootstrapCluster is used to store and retrieve a target cluster kubeconfig.
var bootstrapCluster contextKey = "bootstrap-cluster"
var targetCluster contextKey = "target-cluster"

// WithBootstrapCluster returns a context based on ctx containing the target cluster kubeconfig.
func WithBootstrapCluster(ctx context.Context, cluster types.Cluster) context.Context {
	return context.WithValue(ctx, bootstrapCluster, cluster)
}

// BootstrapCluster retrieves the bootstrap cluster configured in ctx or returns an empty string.
func BootstrapCluster(ctx context.Context) types.Cluster {
	return ctx.Value(bootstrapCluster).(types.Cluster)
}

// TODO: better name
func WithTargetCluster(ctx context.Context, cluster types.Cluster) context.Context {
	return context.WithValue(ctx, targetCluster, cluster)
}

func TargetCluster(ctx context.Context) types.Cluster {
	return ctx.Value(targetCluster).(types.Cluster)
}
