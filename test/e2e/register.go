package e2e

import (
	"testing"

	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/test/framework"
)

var testRegistry = map[string]framework.TestGroup{}

// TestCases returns the test cases registered for the given test.
// If the test hasn't been registered to be run with multiple test
// cases, it returns nil.
func TestCases(testName string) []framework.TestCase {
	tg, ok := testRegistry[testName]
	if !ok {
		return nil
	}

	return tg.GenerateTestCases()
}

// runFor registers a test to be run with a collection of test cases.
// This needs to be run either at the package level, so the test cases
// for a given test can be inspected from outside this package.
func runFor(testName string, testGroup framework.TestGroup) struct{} {
	testRegistry[testName] = testGroup
	return struct{}{}
}

// testsFor returns the registered test cases for the given test.
// Intended to be run from each test to generate sub tests.
func testsFor(tb testing.TB) []framework.TestCase {
	tb.Helper()
	tg, ok := testRegistry[tb.Name()]

	if !ok {
		tb.Fatalf("Test %s is not registered", tb.Name())
	}

	return tg.GenerateTestCases()
}

var allKubeVersions = []anywherev1.KubernetesVersion{
	anywherev1.Kube126,
	anywherev1.Kube127,
	anywherev1.Kube128,
	anywherev1.Kube129,
	anywherev1.Kube130,
}
