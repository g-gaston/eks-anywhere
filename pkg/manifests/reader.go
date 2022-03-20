package manifests

import (
	"fmt"

	"github.com/aws/eks-anywhere/pkg/releases"
	releasev1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
)

type FileReader interface {
	ReadFile(url string) ([]byte, error)
}

type Reader struct {
	FileReader
}

func NewReader(filereader FileReader) *Reader {
	return &Reader{FileReader: filereader}
}

func (r *Reader) ReadBundlesForVersion(version string) (*releasev1.Bundles, error) {
	rls, err := releases.ReadReleases(r)
	if err != nil {
		return nil, err
	}

	release, err := releases.ReleaseForVersion(rls, version)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, fmt.Errorf("invalid version %s, no matching release found", version)
	}

	return releases.ReadBundlesForRelease(r, release)
}
