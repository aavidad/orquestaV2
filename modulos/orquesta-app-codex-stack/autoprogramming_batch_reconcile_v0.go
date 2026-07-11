package orquestaappcodexstack

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

// ReconcileClosedAutoprogrammingBatchRunV0 advances batch effects after one
// goal-first member has reached a durable closed run.
func (stack StackV0) ReconcileClosedAutoprogrammingBatchRunV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (handled bool, complete bool, evidenceRefs []string, err error) {
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return false, false, nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return stack.maybeFinalizeClosedAutoprogrammingBatchV0(ctx, run)
}
