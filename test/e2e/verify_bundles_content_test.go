//go:build e2e
// +build e2e

package e2e

import (
	"testing"

	"github.com/aws/eks-anywhere/test/framework"
)

func TestVerifyBundles(t *testing.T) {
	test := framework.NewClusterE2ETest(t, framework.NewDocker(t))
	test.GenerateClusterConfig()

	testCommand := []string{"verify-setup", "-f", test.ClusterConfigLocation, "-v", "4"}
	testCommand = framework.ProcessBundlesOverride(testCommand)

	test.RunEKSA(testCommand)
}
