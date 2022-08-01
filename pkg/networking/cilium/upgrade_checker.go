package cilium

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/aws/eks-anywhere/pkg/cluster"
)

// InstallationUpgradeInfo contains information about a Cilium installation upgrade
type InstallationUpgradeInfo struct {
	DaemonSet ComponentUpgradeInfo
	Operator  ComponentUpgradeInfo
}

// Needed determines if an upgrade is needed or not
// Returns true if any of the installation components needs an upgrade
func (c InstallationUpgradeInfo) Needed() bool {
	return c.DaemonSet.Needed() || c.Operator.Needed()
}

// Reason returns the reason why an upgrade might be needed
// If no upgrade needed, returns empty string
// For multiple components with needed upgrades, it composes their reasons into one
func (c InstallationUpgradeInfo) Reason() string {
	if !c.Needed() {
		return ""
	}

	s := make([]string, 0, 2)
	if c.DaemonSet.Reason != "" {
		s = append(s, c.DaemonSet.Reason)
	}
	if c.Operator.Reason != "" {
		s = append(s, c.Operator.Reason)
	}

	return strings.Join(s, " - ")
}

// ComponentUpgradeInfo contains upgrade information for a Cilium component
type ComponentUpgradeInfo struct {
	Reason   string
	OldImage string
	NewImage string
}

// Needed determines if an upgrade is needed or not
func (c ComponentUpgradeInfo) Needed() bool {
	return c.Reason != ""
}

// AnalyzeInstallationUpgrade builds the upgrade information for a cilium installation by comparing it
// with a desired cluster Spec
func AnalyzeInstallationUpgrade(installation *Installation, clusterSpec *cluster.Spec) InstallationUpgradeInfo {
	return InstallationUpgradeInfo{
		DaemonSet: daemonSetUpgradeInfo(installation.DaemonSet, clusterSpec),
		Operator:  operatorSetUpgradeInfo(installation.Operator, clusterSpec),
	}
}

func daemonSetUpgradeInfo(ds *appsv1.DaemonSet, clusterSpec *cluster.Spec) ComponentUpgradeInfo {
	dsImage := clusterSpec.VersionsBundle.Cilium.Cilium.VersionedImage()
	i := ComponentUpgradeInfo{
		NewImage: dsImage,
	}

	if ds == nil {
		i.Reason = "Cilium needs upgrade, DS doesn't exist"
		return i
	}

	oldDSImage := ds.Spec.Template.Spec.Containers[0].Image
	i.OldImage = oldDSImage

	containers := make([]corev1.Container, 0, len(ds.Spec.Template.Spec.Containers)+len(ds.Spec.Template.Spec.InitContainers))
	containers = append(containers, ds.Spec.Template.Spec.Containers...)
	containers = append(containers, ds.Spec.Template.Spec.InitContainers...)
	for _, c := range containers {
		if c.Image != dsImage {
			i.OldImage = c.Image
			i.Reason = fmt.Sprintf("Cilium needs upgrade, DS container %s doesn't match image", c.Name)
			return i
		}
	}

	return i
}

func operatorSetUpgradeInfo(operator *appsv1.Deployment, clusterSpec *cluster.Spec) ComponentUpgradeInfo {
	newImage := clusterSpec.VersionsBundle.Cilium.Operator.VersionedImage()
	i := ComponentUpgradeInfo{
		NewImage: newImage,
	}

	if operator == nil {
		i.Reason = "Operator deployment doesn't exist"
		return i
	}

	if len(operator.Spec.Template.Spec.Containers) == 0 {
		i.Reason = "Operator deployment doesn't have any containers"
		return i
	}

	oldImage := operator.Spec.Template.Spec.Containers[0].Image
	i.OldImage = oldImage

	if oldImage != newImage {
		i.Reason = "Operator container doesn't match image"
		return i
	}

	return i
}
