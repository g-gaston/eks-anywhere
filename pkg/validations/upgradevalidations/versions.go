package upgradevalidations

import (
	"context"
	"fmt"

	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/validations"
)

const supportedMinorVersionIncrement = 1

func ValidateServerVersionSkew(ctx context.Context, newCluster *anywherev1.Cluster, cluster *types.Cluster, kubectl validations.KubectlClient) error {
	oldCluster, err := kubectl.GetEksaCluster(ctx, cluster, newCluster.Name)
	if err != nil {
		return fmt.Errorf("fetching old cluster: %v", err)
	}

	return anywherev1.ValidateServerVersionSkew(newCluster, oldCluster).ToAggregate()
}
