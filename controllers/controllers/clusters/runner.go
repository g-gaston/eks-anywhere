package clusters

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/aws/eks-anywhere/controllers/controllers/reconciler"
	"github.com/aws/eks-anywhere/pkg/cluster"
)

type phase func(ctx context.Context, log logr.Logger, clusterSpec *cluster.Spec) (reconciler.Result, error)

type phaseRunner struct {
	phases []phase
}

func newRunner() phaseRunner {
	return phaseRunner{}
}

func (r phaseRunner) register(phases ...phase) phaseRunner {
	r.phases = append(r.phases, phases...)
	return r
}

func (r phaseRunner) run(ctx context.Context, log logr.Logger, clusterSpec *cluster.Spec) (reconciler.Result, error) {
	for _, p := range r.phases {
		if r, err := p(ctx, log, clusterSpec); r.Return() {
			return r, nil
		} else if err != nil {
			return reconciler.Result{}, err
		}
	}

	return reconciler.Result{}, nil
}
