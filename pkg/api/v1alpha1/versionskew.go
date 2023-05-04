package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/version"
)

const supportedMinorVersionIncrement = 1

// ValidateVersionSkew validates Kubernetes version skew between valid non-nil versions.
func ValidateVersionSkew(oldVersion, newVersion *version.Version) error {
	if newVersion.LessThan(oldVersion) {
		return fmt.Errorf("kubernetes version downgrade is not supported (%s) -> (%s)", oldVersion, newVersion)
	}

	newVersionMinor := newVersion.Minor()
	oldVersionMinor := oldVersion.Minor()

	minorVersionDifference := int(newVersionMinor) - int(oldVersionMinor)

	if minorVersionDifference > supportedMinorVersionIncrement {
		return fmt.Errorf("MINOR version difference between upgrade version (%d.%d) and server version (%d.%d) does not meet the supported version increment of +%d",
			newVersionMajor, newVersionMinor, oldVersionMajor, oldVersionMinor, supportedMinorVersionIncrement)
	}

	return nil
}
