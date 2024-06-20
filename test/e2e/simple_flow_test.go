//go:build e2e
// +build e2e

package e2e

import (
	"testing"

	"github.com/aws/eks-anywhere/test/framework"
)

var _ = runFor("TestSimpleFlow",
	framework.TestGroups{
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
	},
)

func TestSimpleFlow(t *testing.T) {
	for _, tc := range testsFor(t) {
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
