package framework

import (
	"testing"

	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
)

type TestCase struct {
	NewProvider  func(t *testing.T) Provider
	ProviderName string
	KubeVersion  anywherev1.KubernetesVersion
	OS           OS
}

func (tc TestCase) Name() string {
	return tc.ProviderName + "-" + string(tc.OS) + "-" + string(tc.KubeVersion)
}

type TestGroups []TestGroup

func (tcs TestGroups) GenerateTestCases() []TestCase {
	var testCases []TestCase
	for _, tc := range tcs {
		testCases = append(testCases, tc.GenerateTestCases()...)
	}

	return testCases
}

type TestGroup interface {
	GenerateTestCases() []TestCase
}

type DockerTests struct {
	KubeVersions []anywherev1.KubernetesVersion
}

func (d DockerTests) GenerateTestCases() []TestCase {
	tcs := make([]TestCase, 0, len(d.KubeVersions))
	for _, kubeVersion := range d.KubeVersions {
		tcs = append(tcs, TestCase{
			NewProvider:  NewDockerProvider,
			ProviderName: "docker",
			KubeVersion:  kubeVersion,
		})
	}

	return tcs
}

type VSphereTests struct {
	KubeVersions []anywherev1.KubernetesVersion
	OSs          []OS
}

func (v VSphereTests) GenerateTestCases() []TestCase {
	tcs := make([]TestCase, 0, len(v.KubeVersions)*len(v.OSs))
	for _, kubeVersion := range v.KubeVersions {
		for _, os := range v.OSs {
			tcs = append(tcs, TestCase{
				NewProvider:  NewVSphereProvider,
				ProviderName: "vsphere",
				KubeVersion:  kubeVersion,
				OS:           os,
			})
		}
	}

	return tcs
}

type CloudStackTests struct {
	KubeVersions []anywherev1.KubernetesVersion
	OSs          []OS
}

func (c CloudStackTests) GenerateTestCases() []TestCase {
	tcs := make([]TestCase, 0, len(c.KubeVersions)*len(c.OSs))
	for _, kubeVersion := range c.KubeVersions {
		for _, os := range c.OSs {
			tcs = append(tcs, TestCase{
				NewProvider:  NewCloudStackProvider,
				ProviderName: "cloudstack",
				KubeVersion:  kubeVersion,
				OS:           os,
			})
		}
	}

	return tcs
}
