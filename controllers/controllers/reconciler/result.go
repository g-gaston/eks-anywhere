package reconciler

import (
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

type Result struct {
	Result *ctrl.Result
}

func (r *Result) ToCtrlResult() ctrl.Result {
	if r.Result == nil {
		return ctrl.Result{}
	}

	return *r.Result
}

func (r *Result) Return() bool {
	return r.Result != nil
}

func ResultWithReturn() Result {
	return Result{Result: &ctrl.Result{}}
}

func ResultWithRequeue(after time.Duration) Result {
	return Result{Result: &ctrl.Result{RequeueAfter: after}}
}
