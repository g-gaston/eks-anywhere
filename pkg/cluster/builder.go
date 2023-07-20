package cluster

import (
	eksdv1 "github.com/aws/eks-distro-build-tooling/release/api/v1alpha1"
	"github.com/pkg/errors"

	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/constants"
	"github.com/aws/eks-anywhere/pkg/manifests"
	"github.com/aws/eks-anywhere/pkg/manifests/bundles"
	"github.com/aws/eks-anywhere/pkg/version"
	releasev1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
)

// FileSpecBuilder allows to build [Spec] by reading from files.
type FileSpecBuilder struct {
	// TODO(g-gaston): this is very much a CLI thing. Move to `pkg/cli` when available.
	reader              manifests.FileReader
	cliVersion          version.Info
	releasesManifestURL string
	bundlesManifestURL  string
}

// FileSpecBuilderOpt allows to configure [FileSpecBuilder].
type FileSpecBuilderOpt func(*FileSpecBuilder)

// WithReleasesManifest configures the URL to read the Releases manifest.
func WithReleasesManifest(url string) FileSpecBuilderOpt {
	return func(b *FileSpecBuilder) {
		b.releasesManifestURL = url
	}
}

// WithOverrideBundlesManifest configures the URL to read the Bundles manifest.
// This overrides the Bundles declared in the Releases so reading the Releases
// manifest is skipped.
func WithOverrideBundlesManifest(url string) FileSpecBuilderOpt {
	return func(b *FileSpecBuilder) {
		b.bundlesManifestURL = url
	}
}

// NewFileSpecBuilder builds a new [FileSpecBuilder].
// cliVersion is used to chose the right Bundles from the the Release manifest.
func NewFileSpecBuilder(reader manifests.FileReader, cliVersion version.Info, opts ...FileSpecBuilderOpt) FileSpecBuilder {
	f := &FileSpecBuilder{
		cliVersion: cliVersion,
		reader:     reader,
	}

	for _, opt := range opts {
		opt(f)
	}

	return *f
}

// Build constructs a new [Spec] by reading the cluster config in yaml from a file and
// Releases, Bundles and EKS-D manifests from the configured URLs.
func (b FileSpecBuilder) Build(clusterConfigURL string) (*Spec, error) {
	config, err := b.getConfig(clusterConfigURL)
	if err != nil {
		return nil, err
	}
	bundlesManifest, err := b.getBundles()
	if err != nil {
		return nil, errors.Wrapf(err, "getting Bundles file")
	}

	bundlesManifest.Namespace = constants.EksaSystemNamespace

	configManager, err := NewDefaultConfigManager()
	if err != nil {
		return nil, err
	}
	configManager.RegisterDefaulters(BundlesRefDefaulter())

	if err = configManager.SetDefaults(config); err != nil {
		return nil, err
	}

	// We are pulling the latest available Bundles, so making sure we update the ref to make the spec consistent
	config.Cluster.Spec.BundlesRef.Name = bundlesManifest.Name
	config.Cluster.Spec.BundlesRef.Namespace = bundlesManifest.Namespace
	config.Cluster.Spec.BundlesRef.APIVersion = releasev1.GroupVersion.String()

	eksdReleases, err := getAllEksdReleases(config.Cluster, bundlesManifest, b.reader)
	if err != nil {
		return nil, err
	}

	return NewSpec(config, bundlesManifest, eksdReleases)
}

func getAllEksdReleases(cluster *v1alpha1.Cluster, bundlesManifest *releasev1.Bundles, reader bundles.Reader) ([]eksdv1.Release, error) {
	m := make([]eksdv1.Release, 0)
	visited := map[string]bool{}
	version := cluster.Spec.KubernetesVersion
	visited[string(version)] = true
	eksd, err := getEksdReleases(version, bundlesManifest, reader)
	if err != nil && eksd == nil {
		return nil, err
	}
	m = append(m, *eksd)

	for _, wng := range cluster.Spec.WorkerNodeGroupConfigurations {
		if wng.KubernetesVersion != nil {
			version = *wng.KubernetesVersion
			if _, ok := visited[string(version)]; ok {
				continue
			}
			visited[string(version)] = true
			eksd, err = getEksdReleases(version, bundlesManifest, reader)
			if err != nil && eksd == nil {
				return nil, err
			}
			m = append(m, *eksd)
		}
	}
	return m, nil
}

func getEksdReleases(version v1alpha1.KubernetesVersion, bundlesManifest *releasev1.Bundles, reader bundles.Reader) (*eksdv1.Release, error) {
	versionsBundle, err := GetVersionsBundle(version, bundlesManifest)
	if err != nil {
		return nil, err
	}

	eksd, err := bundles.ReadEKSD(reader, *versionsBundle)
	if err != nil {
		return nil, err
	}

	return eksd, nil
}

func (b FileSpecBuilder) getConfig(clusterConfigURL string) (*Config, error) {
	yaml, err := b.reader.ReadFile(clusterConfigURL)
	if err != nil {
		return nil, errors.Wrapf(err, "reading cluster config file")
	}

	return ParseConfig(yaml)
}

func (b FileSpecBuilder) getBundles() (*releasev1.Bundles, error) {
	bundlesURL := b.bundlesManifestURL
	if bundlesURL == "" {
		var opts []manifests.ReaderOpt
		if b.releasesManifestURL != "" {
			opts = append(opts, manifests.WithReleasesManifest(b.releasesManifestURL))
		}
		manifestReader := manifests.NewReader(b.reader, opts...)
		return manifestReader.ReadBundlesForVersion(b.cliVersion.GitVersion)
	}

	return bundles.Read(b.reader, bundlesURL)
}
