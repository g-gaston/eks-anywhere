package clustermanager

import (
	"context"

	"github.com/aws/eks-anywhere/pkg/cluster"
)

type Upgrader struct {
	client ClusterClient
}

func NewUpgrader(client ClusterClient) *Upgrader {
	return &Upgrader{client: client}
}

func (u *Upgrader) Upgrade(ctx context.Context, currentSpec *cluster.Spec, newSpec *cluster.Spec) error {
	return nil
}
