package v1alpha1

import (
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/util/version"
)

const supportedMinorVersionIncrement = 1

// ValidateVersionSkew validates Kubernetes version skew between valid non-nil versions.
func ValidateVersionSkew(oldVersion, newVersion *version.Version) error {
	newVersionMajor := newVersion.Major()
	newVersionMinor := newVersion.Minor()
	oldVersionMajor := oldVersion.Major()
	oldVersionMinor := oldVersion.Minor()

	majorVersionDifference := math.Abs(float64(newVersionMajor) - float64(oldVersionMajor))

	if majorVersionDifference > 0 {
		return fmt.Errorf("MAJOR version upgrades are not supported. Major version difference between upgrade version (%d.%d) and server version (%d.%d) is %f", newVersionMajor, newVersionMinor, oldVersionMajor, oldVersionMinor, majorVersionDifference)
	}

	minorVersionDifference := float64(newVersionMinor) - float64(oldVersionMinor)

	if minorVersionDifference < 0 {
		return fmt.Errorf("MINOR version downgrade is not supported. Difference between upgrade version (%d.%d) and server version (%d.%d) should meet the supported version increment of +%d", newVersionMajor, newVersionMinor, oldVersionMajor, oldVersionMinor, supportedMinorVersionIncrement)
	}

	if minorVersionDifference > supportedMinorVersionIncrement {
		return fmt.Errorf("MINOR version difference between upgrade version (%d.%d) and server version (%d.%d) does not meet the supported version increment of +%d",
			newVersionMajor, newVersionMinor, oldVersionMajor, oldVersionMinor, supportedMinorVersionIncrement)
	}

	return nil
}
