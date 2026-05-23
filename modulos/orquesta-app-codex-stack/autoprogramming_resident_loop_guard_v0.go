package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const autoprogrammingResidentLoopGuardEvidenceV0 = "evidence-ref-autoprogramming-resident-selfrepair-loop-guard"

func autoprogrammingResidentRunAlreadySelfRepairV0(
	ctx context.Context,
	stack StackV0,
	runRef string,
) bool {
	if stack.Ports.RunStore == nil || stack.Ports.DirectorTaskStore == nil {
		return false
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, strings.TrimSpace(runRef))
	if err != nil {
		return false
	}
	task, ok := autoprogrammingResidentRepairTaskV0(ctx, stack, run)
	if !ok {
		return false
	}
	return autoprogrammingResidentTaskHasSelfRepairLineageV0(task)
}

func autoprogrammingResidentTaskHasSelfRepairLineageV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	for _, ref := range task.ContextRefs {
		ref = strings.TrimSpace(ref)
		switch {
		case strings.HasPrefix(ref, "source_run_ref:"):
			return true
		case strings.HasPrefix(ref, "request_ref:request-ref-autoprogramming-resident-selfrepair-"):
			return true
		case strings.Contains(ref, "autoprogramming-resident-selfrepair"):
			return true
		}
	}
	return false
}
