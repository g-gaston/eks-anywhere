// +build e2e

package e2e

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/aws/eks-anywhere/internal/pkg/api"
	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/eksctl"
	"github.com/aws/eks-anywhere/pkg/executables"
	"github.com/aws/eks-anywhere/pkg/features"
	"github.com/aws/eks-anywhere/pkg/validations"
	"github.com/aws/eks-anywhere/test/framework"

	releasev1alpha1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
)

const (
	prodReleasesManifest = "https://anywhere-assets.eks.amazonaws.com/releases/eks-a/manifest.yaml"
	latestReleasePath    = "bin/latest-release"
	releaseBinaryName    = "eksctl-anywhere"
)

func runUpgradeFromLatestCLIFlow(test *framework.E2ETest) {
	latestReleaseBinaryPath := setupLatestReleaseBinary(test)
	// Force eks-a in dev mode. Hacky but should work for now
	os.Setenv(eksctl.VersionEnvVar, "no-version")

	generateClusterConfig(test, latestReleaseBinaryPath)
	createCluster(test, latestReleaseBinaryPath)
	// Enable core component upgrades
	os.Setenv(features.ComponentsUpgradesEnvVar, "true")
	logPodImages(test)
	test.UpgradeCluster()
	logPodImages(test)
	test.DeleteCluster()
}

func setupLatestReleaseBinary(test *framework.E2ETest) (binaryPath string) {
	reader := cluster.NewManifestReader("e2e")
	test.T.Logf("Reading prod release manifest %s", prodReleasesManifest)
	releases, err := reader.GetReleases(prodReleasesManifest)
	if err != nil {
		test.T.Fatal(err)
	}
	var latestRelease *releasev1alpha1.EksARelease
	for _, release := range releases.Spec.Releases {
		if release.Version == releases.Spec.LatestVersion {
			latestRelease = &release
			break
		}
	}

	if latestRelease == nil {
		test.T.Fatalf("Releases manifest doesn't contain latest release %s", releases.Spec.LatestVersion)
	}

	latestReleaseBinaryFolder := filepath.Join(latestReleasePath, latestRelease.Version)
	latestReleaseBinaryPath := filepath.Join(latestReleaseBinaryFolder, releaseBinaryName)

	if !validations.FileExists(latestReleaseBinaryPath) {
		test.T.Logf("Reading prod latest release tarball %s", latestRelease.EksABinary.LinuxBinary.URI)
		latestReleaseTar, err := reader.ReadFile(latestRelease.EksABinary.LinuxBinary.URI)
		if err != nil {
			test.T.Fatalf("Failed downloading tar for latest release: %s", err)
		}

		test.T.Log("Untaring prod latest release tarball")

		err = untar(latestReleaseBinaryFolder, bytes.NewReader(latestReleaseTar))
		if err != nil {
			test.T.Fatalf("Failed untaring latest release: %s", err)
		}
	}

	return latestReleaseBinaryPath
}

func untar(destinationFolder string, r io.Reader) error {
	err := os.MkdirAll(destinationFolder, os.ModePerm)
	if err != nil {
		return err
	}

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var header *tar.Header
	for {
		header, err = tr.Next()
		if err == io.EOF {
			return fmt.Errorf("Binary [%s] not found in tarball", releaseBinaryName)
		}

		if err != nil {
			return err
		}

		if header != nil && strings.TrimPrefix(header.Name, "./") == releaseBinaryName {
			break
		}
	}

	target := filepath.Join(destinationFolder, header.Name)
	if header.Typeflag != tar.TypeReg {
		return fmt.Errorf("Invalid type flag [%b] for binary [%s]", header.Typeflag, releaseBinaryName)
	}

	f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, tr); err != nil {
		return err
	}

	return nil
}

func generateClusterConfig(test *framework.E2ETest, eksaBinaryPath string) {
	test.Run(eksaBinaryPath, "generate", "clusterconfig", test.ClusterName, "-p", test.Provider.Name(), ">", test.ClusterConfigLocation)
	test.FillClusterConfig()
}

func createCluster(test *framework.E2ETest, eksaBinaryPath string) {
	test.Run(eksaBinaryPath, "create", "cluster", "-f", test.ClusterConfigLocation, "-v", "3")
	test.Cleanup(func() {
		os.RemoveAll(test.ClusterName)
	})
}

func logPodImages(test *framework.E2ETest) {
	pods, err := test.KubectlClient.GetPods(context.Background(),
		executables.WithKubeconfig(test.KubeconfigFilePath()),
		executables.WithAllNamespaces(),
	)
	if err != nil {
		test.T.Fatal(err)
	}

	podImagesSet := make(map[string]struct{}, len(pods))
	for _, p := range pods {
		for _, container := range p.Spec.Containers {
			podImagesSet[container.Image] = struct{}{}
		}
	}

	podImages := make([]string, 0, len(podImagesSet))
	for image := range podImagesSet {
		podImages = append(podImages, image)
	}

	sort.Strings(podImages)
	test.T.Logf("Pod images:\n%s", strings.Join(podImages, "\n"))
}

func TestDockerKubernetes120UpgradeFromLatestCli(t *testing.T) {
	test := framework.NewE2ETest(t,
		framework.NewDocker(t),
		framework.WithVLevel(3),
		framework.WithClusterName("Docker120CoreUpgrade"),
		framework.WithClusterFiller(
			api.WithKubernetesVersion(v1alpha1.Kube120),
			api.WithControlPlaneCount(1),
			api.WithWorkerNodeCount(1),
			api.WithStackedEtcdTopology(),
		),
	)
	runUpgradeFromLatestCLIFlow(test)
}

func TestDockerKubernetes120ExternalEtcdUpgradeFromLatestCli(t *testing.T) {
	test := framework.NewE2ETest(t,
		framework.NewDocker(t),
		framework.WithVLevel(3),
		framework.WithClusterName("Docker120ExtEtcdCoreUpgrade"),
		framework.WithClusterFiller(
			api.WithKubernetesVersion(v1alpha1.Kube120),
			api.WithControlPlaneCount(1),
			api.WithWorkerNodeCount(1),
			api.WithExternalEtcdTopology(1),
		),
	)
	runUpgradeFromLatestCLIFlow(test)
}

func TestVSphereKubernetes121UbuntuUpgradeFromLatestCli(t *testing.T) {
	test := framework.NewE2ETest(t,
		framework.NewVSphere(t, framework.WithUbuntu121()),
		framework.WithVLevel(3),
		framework.WithClusterName("Ubuntu121CoreUpgradeLastCli"),
		framework.WithClusterFiller(
			api.WithKubernetesVersion(v1alpha1.Kube121),
			api.WithControlPlaneCount(1),
			api.WithWorkerNodeCount(1),
			api.WithStackedEtcdTopology(),
		),
	)
	runUpgradeFromLatestCLIFlow(test)
}

func TestVSphereKubernetes120UbuntuExternalEtcdUpgradeFromLatestCli(t *testing.T) {
	test := framework.NewE2ETest(t,
		framework.NewVSphere(t, framework.WithUbuntu120()),
		framework.WithVLevel(3),
		framework.WithClusterName("Ubuntu120ExtEtcdCoreUpgradeLastCli"),
		framework.WithClusterFiller(
			api.WithKubernetesVersion(v1alpha1.Kube120),
			api.WithControlPlaneCount(1),
			api.WithWorkerNodeCount(1),
			api.WithExternalEtcdTopology(1),
		),
	)
	runUpgradeFromLatestCLIFlow(test)
}

func TestVSphereKubernetes120BottlerocketUpgradeFromLatestCli(t *testing.T) {
	test := framework.NewE2ETest(t,
		framework.NewVSphere(t, framework.WithBottleRocket120()),
		framework.WithVLevel(3),
		framework.WithClusterName("BR120CoreUpgradeLastCli"),
		framework.WithClusterFiller(
			api.WithKubernetesVersion(v1alpha1.Kube120),
			api.WithControlPlaneCount(1),
			api.WithWorkerNodeCount(1),
			api.WithStackedEtcdTopology(),
		),
	)
	runUpgradeFromLatestCLIFlow(test)
}

func TestVSphereKubernetes121BottlerocketExternalEtcdUpgradeFromLatestCli(t *testing.T) {
	test := framework.NewE2ETest(t,
		framework.NewVSphere(t, framework.WithBottleRocket121()),
		framework.WithVLevel(3),
		framework.WithClusterName("BR121ExtEtcdCoreUpgradeLastCli"),
		framework.WithClusterFiller(
			api.WithKubernetesVersion(v1alpha1.Kube121),
			api.WithControlPlaneCount(1),
			api.WithWorkerNodeCount(1),
			api.WithExternalEtcdTopology(1),
		),
	)
	runUpgradeFromLatestCLIFlow(test)
}
