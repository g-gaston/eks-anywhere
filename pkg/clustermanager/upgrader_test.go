package clustermanager_test

import (
	"testing"

	"github.com/aws/eks-anywhere/pkg/clustermanager"
	"github.com/aws/eks-anywhere/pkg/clustermanager/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
)

type upgraderTest struct {
	*WithT
	client   *mocks.MockClusterClient
	upgrader *clustermanager.Upgrader
}

func newUpgraderTest(t *testing.T) *upgraderTest {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockClusterClient(ctrl)
	return &upgraderTest{
		WithT:    NewWithT(t),
		client:   client,
		upgrader: clustermanager.NewUpgrader(client),
	}
}

func TestUpgraderUpgradeSuccess(t *testing.T) {
	_ = newUpgraderTest(t)
}
