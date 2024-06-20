package e2e

import (
	"testing"

	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/test/framework"
)

var allKubeVersions = []anywherev1.KubernetesVersion{
	anywherev1.Kube126,
	anywherev1.Kube127,
	anywherev1.Kube128,
	anywherev1.Kube129,
	anywherev1.Kube130,
}

func TestSimpleFlow(t *testing.T) {
	tests := framework.TestCases{
		framework.DockerTests{
			KubeVersions: allKubeVersions,
		},
		framework.VSphereTests{
			KubeVersions: allKubeVersions,
			OSs:          []framework.OS{framework.Ubuntu2004, framework.Ubuntu2204, framework.Bottlerocket1},
		},
		framework.CloudStackTests{
			KubeVersions: allKubeVersions,
			OSs:          []framework.OS{framework.RedHat8, framework.RedHat9},
		},
	}

	for _, tc := range tests.GenerateTestCases() {
		t.Run(tc.Name(), func(t *testing.T) {
			provider := tc.NewProvider(t)
			test := framework.NewClusterE2ETest(
				t,
				provider,
			).WithClusterConfig(
				provider.WithKubeVersionAndOS(tc.KubeVersion, tc.OS, nil),
			)
			runSimpleFlowWithoutClusterConfigGeneration(test)
		})
	}
}
