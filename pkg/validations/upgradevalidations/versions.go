package upgradevalidations

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/version"

	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/logger"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/validations"
)

func ValidateServerVersionSkew(ctx context.Context, compareVersion v1alpha1.KubernetesVersion, cluster *types.Cluster, kubectl validations.KubectlClient) error {
	versions, err := kubectl.Version(ctx, cluster)
	if err != nil {
		return fmt.Errorf("fetching cluster version: %v", err)
	}

	parsedInputVersion, err := version.ParseGeneric(string(compareVersion))
	if err != nil {
		return fmt.Errorf("parsing comparison version: %v", err)
	}

	parsedServerVersion, err := version.ParseSemantic(versions.ServerVersion.GitVersion)
	if err != nil {
		return fmt.Errorf("parsing cluster version: %v", err)
	}

	logger.V(3).Info("calculating version differences", "inputVersion", parsedInputVersion, "clusterVersion", parsedServerVersion)

	if err := v1alpha1.ValidateVersionSkew(parsedServerVersion, parsedInputVersion); err != nil {
		return err
	}
	return nil
}
