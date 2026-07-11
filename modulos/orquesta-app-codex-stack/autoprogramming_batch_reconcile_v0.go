package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
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

func (stack StackV0) reconcileQueuedClosedAutoprogrammingBatchRunsV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) error {
	if stack.Stores.RunQueue == nil || stack.Ports.RunStore == nil ||
		stack.Stores.AutoprogrammingBatchStore == nil {
		return nil
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             strings.TrimSpace(command.QueueRef),
		AppRefs:              append([]string(nil), command.AppRefs...),
		IncludeNonExecutable: true,
	})
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.Status) != orquestarunqueue.RunStatusClosedV0 {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		run, err := stack.Ports.RunStore.LoadRunV0(ctx, strings.TrimSpace(candidate.RunRef))
		if err != nil {
			if orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
				continue
			}
			return err
		}
		if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
			continue
		}
		if _, _, _, err := stack.ReconcileClosedAutoprogrammingBatchRunV0(ctx, run); err != nil {
			return err
		}
	}
	return nil
}
