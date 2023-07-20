package cluster

import (
	"context"
	"fmt"

	eksdv1alpha1 "github.com/aws/eks-distro-build-tooling/release/api/v1alpha1"

	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/constants"
	v1alpha1release "github.com/aws/eks-anywhere/release/api/v1alpha1"
)

func bundlesNamespacedKey(cluster *v1alpha1.Cluster) (name, namespace string) {
	if cluster.Spec.BundlesRef != nil {
		name = cluster.Spec.BundlesRef.Name
		namespace = cluster.Spec.BundlesRef.Namespace
	} else {
		// Handles old clusters that don't contain a reference yet to the Bundles
		// For those clusters, the Bundles was created with the same name as the cluster
		// and in the same namespace
		name = cluster.Name
		namespace = cluster.Namespace
	}

	return name, namespace
}

// GetVersionsBundle gets the VersionsBundle that corresponds to KubernetesVersion.
func GetVersionsBundle(version v1alpha1.KubernetesVersion, bundles *v1alpha1release.Bundles) (*v1alpha1release.VersionsBundle, error) {
	return getVersionsBundleForKubernetesVersion(version, bundles)
}

func getVersionsBundleForKubernetesVersion(kubernetesVersion v1alpha1.KubernetesVersion, bundles *v1alpha1release.Bundles) (*v1alpha1release.VersionsBundle, error) {
	for _, versionsBundle := range bundles.Spec.VersionsBundles {
		if versionsBundle.KubeVersion == string(kubernetesVersion) {
			return &versionsBundle, nil
		}
	}
	return nil, fmt.Errorf("kubernetes version %s is not supported by bundles manifest %d", kubernetesVersion, bundles.Spec.Number)
}

// BuildSpec constructs a cluster.Spec for an eks-a cluster by retrieving all
// necessary objects from the cluster using a kubernetes client.
func BuildSpec(ctx context.Context, client Client, cluster *v1alpha1.Cluster) (*Spec, error) {
	configBuilder := NewDefaultConfigClientBuilder()
	config, err := configBuilder.Build(ctx, client, cluster)
	if err != nil {
		return nil, err
	}

	return BuildSpecFromConfig(ctx, client, config)
}

// BuildSpecFromConfig constructs a cluster.Spec for an eks-a cluster config by retrieving all dependencies objects from the cluster using a kubernetes client.
func BuildSpecFromConfig(ctx context.Context, client Client, config *Config) (*Spec, error) {
	bundlesName, bundlesNamespace := bundlesNamespacedKey(config.Cluster)
	bundles := &v1alpha1release.Bundles{}
	if err := client.Get(ctx, bundlesName, bundlesNamespace, bundles); err != nil {
		return nil, err
	}

	eksdReleases, err := fetchAllEksdReleases(ctx, client, config.Cluster, bundles)
	if err != nil {
		return nil, err
	}

	return NewSpec(config, bundles, eksdReleases)
}

func fetchAllEksdReleases(ctx context.Context, client Client, cluster *v1alpha1.Cluster, bundles *v1alpha1release.Bundles) ([]eksdv1alpha1.Release, error) {
	m := make([]eksdv1alpha1.Release, 0)
	visited := map[string]bool{}

	version := cluster.Spec.KubernetesVersion
	visited[string(version)] = true
	eksd, err := getEksdRelease(ctx, client, version, bundles)
	if err != nil && eksd == nil {
		return nil, err
	}
	m = append(m, *eksd)

	for _, wng := range cluster.Spec.WorkerNodeGroupConfigurations {
		if wng.KubernetesVersion != nil {
			version := *wng.KubernetesVersion
			if _, ok := visited[string(version)]; ok {
				continue
			}
			visited[string(version)] = true
			eksd, err = getEksdRelease(ctx, client, version, bundles)
			if err != nil {
				return nil, err
			}
			m = append(m, *eksd)
		}
	}

	return m, nil
}

func getEksdRelease(ctx context.Context, client Client, version v1alpha1.KubernetesVersion, bundles *v1alpha1release.Bundles) (*eksdv1alpha1.Release, error) {
	versionsBundle, err := GetVersionsBundle(version, bundles)
	if err != nil {
		return nil, err
	}

	// Ideally we would use the same namespace as the Bundles, but Bundles can be in any namespace and
	// the eksd release is always in eksa-system
	eksdRelease := &eksdv1alpha1.Release{}
	if err = client.Get(ctx, versionsBundle.EksD.Name, constants.EksaSystemNamespace, eksdRelease); err != nil {
		return nil, err
	}

	return eksdRelease, nil
}
