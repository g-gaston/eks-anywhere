package validations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/config"
	"github.com/aws/eks-anywhere/pkg/logger"
	"github.com/aws/eks-anywhere/pkg/providers"
	"github.com/aws/eks-anywhere/pkg/semver"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/version"
)

// ValidateOSForRegistryMirror checks if the OS is valid for the provided registry mirror configuration.
func ValidateOSForRegistryMirror(clusterSpec *cluster.Spec, provider providers.Provider) error {
	cluster := clusterSpec.Cluster
	if cluster.Spec.RegistryMirrorConfiguration == nil {
		return nil
	}

	machineConfigs := provider.MachineConfigs(clusterSpec)
	if !cluster.Spec.RegistryMirrorConfiguration.InsecureSkipVerify || machineConfigs == nil {
		return nil
	}

	for _, mc := range machineConfigs {
		if mc.OSFamily() == v1alpha1.Bottlerocket {
			return errors.New("InsecureSkipVerify is not supported for bottlerocket")
		}
	}
	return nil
}

func ValidateCertForRegistryMirror(clusterSpec *cluster.Spec, tlsValidator TlsValidator) error {
	cluster := clusterSpec.Cluster
	if cluster.Spec.RegistryMirrorConfiguration == nil {
		return nil
	}

	if cluster.Spec.RegistryMirrorConfiguration.InsecureSkipVerify {
		logger.V(1).Info("Warning: skip registry certificate verification is enabled", "registryMirrorConfiguration.insecureSkipVerify", true)
		return nil
	}

	host, port := cluster.Spec.RegistryMirrorConfiguration.Endpoint, cluster.Spec.RegistryMirrorConfiguration.Port
	authorityUnknown, err := tlsValidator.IsSignedByUnknownAuthority(host, port)
	if err != nil {
		return fmt.Errorf("validating registry mirror endpoint: %v", err)
	}
	if authorityUnknown {
		logger.V(1).Info(fmt.Sprintf("Warning: registry mirror endpoint %s is using self-signed certs", cluster.Spec.RegistryMirrorConfiguration.Endpoint))
	}

	certContent := cluster.Spec.RegistryMirrorConfiguration.CACertContent
	if certContent == "" && authorityUnknown {
		return fmt.Errorf("registry %s is using self-signed certs, please provide the certificate using caCertContent field. Or use insecureSkipVerify field to skip registry certificate verification", cluster.Spec.RegistryMirrorConfiguration.Endpoint)
	}

	if certContent != "" {
		if err = tlsValidator.ValidateCert(host, port, certContent); err != nil {
			return fmt.Errorf("invalid registry certificate: %v", err)
		}
	}

	return nil
}

// ValidateAuthenticationForRegistryMirror checks if REGISTRY_USERNAME and REGISTRY_PASSWORD is set if authenticated registry mirrors are used.
func ValidateAuthenticationForRegistryMirror(clusterSpec *cluster.Spec) error {
	cluster := clusterSpec.Cluster
	if cluster.Spec.RegistryMirrorConfiguration != nil && cluster.Spec.RegistryMirrorConfiguration.Authenticate {
		_, _, err := config.ReadCredentials()
		if err != nil {
			return err
		}
	}
	return nil
}

// ValidateManagementClusterName checks if the management cluster specified in the workload cluster spec is valid.
func ValidateManagementClusterName(ctx context.Context, k KubectlClient, mgmtCluster *types.Cluster, mgmtClusterName string) error {
	cluster, err := k.GetEksaCluster(ctx, mgmtCluster, mgmtClusterName)
	if err != nil {
		return err
	}
	if cluster.IsManaged() {
		return fmt.Errorf("%s is not a valid management cluster", mgmtClusterName)
	}
	return nil
}

// ValidateManagementClusterBundlesVersion checks if management cluster's bundle version
// is greater than or equal to the bundle version used to upgrade a workload cluster.
func ValidateManagementClusterBundlesVersion(ctx context.Context, k KubectlClient, mgmtCluster *types.Cluster, workload *cluster.Spec) error {
	cluster, err := k.GetEksaCluster(ctx, mgmtCluster, mgmtCluster.Name)
	if err != nil {
		return err
	}

	if cluster.Spec.BundlesRef == nil {
		return fmt.Errorf("management cluster bundlesRef cannot be nil")
	}

	mgmtBundles, err := k.GetBundles(ctx, mgmtCluster.KubeconfigFile, cluster.Spec.BundlesRef.Name, cluster.Spec.BundlesRef.Namespace)
	if err != nil {
		return err
	}

	if mgmtBundles.Spec.Number < workload.Bundles.Spec.Number {
		return fmt.Errorf("cannot upgrade workload cluster with bundle spec.number %d while management cluster %s is on older bundle spec.number %d", workload.Bundles.Spec.Number, mgmtCluster.Name, mgmtBundles.Spec.Number)
	}

	return nil
}

// ValidateEksaVersion ensures that the version matches EKS-A CLI.
func ValidateEksaVersion(ctx context.Context, k KubectlClient, cliVersion string, workload *cluster.Spec) error {
	v := workload.Cluster.Spec.EksaVersion

	if cliVersion == "" {
		cliVersion = version.Get().GitVersion
	}

	if v != nil {
		parsedVersion, err := semver.New(string(*v))
		if err != nil {
			return fmt.Errorf("parsing cluster eksa version: %v", err)
		}

		parsedCLIVersion, err := semver.New(cliVersion)
		if err != nil {
			return fmt.Errorf("parsing eksa cli version: %v", err)
		}

		if !parsedVersion.SamePatch(parsedCLIVersion) {
			return fmt.Errorf("cluster's eksaVersion does not match EKS-A")
		}
	}
	return nil
}

// ValidateEksaVersionSkew ensures that upgrades are sequential by CLI minor versions.
func ValidateEksaVersionSkew(ctx context.Context, k KubectlClient, cluster *types.Cluster, workload *cluster.Spec) error {
	c, err := k.GetEksaCluster(ctx, cluster, cluster.Name)
	if err != nil {
		return err
	}

	v := c.Spec.EksaVersion

	// try getting version from bundle's eksa component's cluster controller image
	if v == nil {
		version, err := getVersionFromBundle(ctx, k, c.Spec.BundlesRef, cluster)
		if err != nil {
			return fmt.Errorf("could not get cluster's version: %w", err)
		}
		v = version
	}

	parsedClusterVersion, err := semver.New(string(*v))
	if err != nil {
		return fmt.Errorf("parsing cluster eksa version: %v", err)
	}

	uv := workload.Cluster.Spec.EksaVersion
	if uv == nil {
		return fmt.Errorf("upgrade cluster version cannot be nil")
	}

	parsedUpgradeVersion, err := semver.New(string(*uv))
	if err != nil {
		return fmt.Errorf("parsing upgrade cli version: %v", err)
	}

	majorVersionDifference := int64(parsedUpgradeVersion.Major) - int64(parsedClusterVersion.Major)
	minorVersionDifference := int64(parsedUpgradeVersion.Minor) - int64(parsedClusterVersion.Minor)
	var supportedMinorVersionIncrement int64 = 1

	if majorVersionDifference > 0 || !(minorVersionDifference <= supportedMinorVersionIncrement && minorVersionDifference >= 0) {
		msg := fmt.Sprintf("WARNING: version difference between upgrade version (%d.%d) and cluster version (%d.%d) do not meet the supported version increment of +%f",
			parsedUpgradeVersion.Major, parsedUpgradeVersion.Minor, parsedClusterVersion.Major, parsedClusterVersion.Minor, supportedMinorVersionIncrement)
		return fmt.Errorf(msg)
	}

	return nil
}

func getVersionFromBundle(ctx context.Context, k KubectlClient, br *v1alpha1.BundlesRef, cluster *types.Cluster) (*v1alpha1.EksaVersion, error) {
	if br == nil {
		return nil, fmt.Errorf("bundlesRef and eksa version cannot be nil")
	}

	name := br.Name
	ns := br.Namespace
	bundle, err := k.GetBundles(ctx, cluster.KubeconfigFile, name, ns)
	if err != nil {
		return nil, err
	}

	uri := bundle.Spec.VersionsBundles[0].Eksa.ClusterController.URI
	if !strings.Contains(uri, ":") {
		return nil, fmt.Errorf("could not find tag in Eksa Cluster Controller Image")
	}

	tag := strings.Split(uri, ":")[1]
	v := strings.Split(tag, "-")[0]
	version := v1alpha1.EksaVersion(v)
	return &version, nil
}

// ValidateManagementClusterEksaVersion ensures workload cluster isn't created by a newer version than management cluster.
func ValidateManagementClusterEksaVersion(ctx context.Context, k KubectlClient, mgmtCluster *types.Cluster, workload *cluster.Spec) error {
	cluster, err := k.GetEksaCluster(ctx, mgmtCluster, mgmtCluster.Name)
	if err != nil {
		return err
	}

	v := cluster.Spec.EksaVersion
	if v == nil {
		return fmt.Errorf("mgmt cluster eksaVersion cannot be nil")
	}

	mVersion, err := semver.New(string(*v))
	if err != nil {
		return fmt.Errorf("management cluster eksaVersion is invalid: %w", err)
	}

	wv := workload.Cluster.Spec.EksaVersion
	if wv == nil {
		return fmt.Errorf("workload cluster eksaVersion cannot be nil")
	}

	wVersion, err := semver.New(string(*workload.Cluster.Spec.EksaVersion))
	if err != nil {
		return fmt.Errorf("workload cluster eksaVersion is invalid: %w", err)
	}

	if wVersion.GreaterThan(mVersion) {
		return fmt.Errorf("cannot upgrade workload cluster with version %v while management cluster is an older version %v", wVersion, mVersion)
	}

	return nil
}
