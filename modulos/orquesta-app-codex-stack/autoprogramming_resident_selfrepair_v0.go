package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	autoprogrammingResidentRepairEvidenceV0 = "evidence-ref-autoprogramming-resident-selfrepair-prepared"
	autoprogrammingResidentNoActionV0       = "evidence-ref-autoprogramming-resident-selfrepair-no-action"
)

func maybePrepareAutoprogrammingResidentSelfRepairV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	stack StackV0,
	supervised CodexSupervisorResultV0,
	result orquestamcp.MCPRunSupervisorToolResultV0,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	if !input.ResidentMode || result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
		return result
	}
	if !autoprogrammingResidentShouldRepairV0(input, supervised, result) {
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, autoprogrammingResidentNoActionV0))
		return result
	}
	if autoprogrammingResidentRunAlreadySelfRepairV0(ctx, stack, result.RunRef) {
		result.NextActions = compactStringsV0(append(result.NextActions,
			"preserve_blocker_evidence",
			"ask_human_review_or_director_before_new_repair",
		))
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs,
			autoprogrammingResidentLoopGuardEvidenceV0,
		))
		if result.Last.Status != "" {
			result.Last.EvidenceRefs = compactStringsV0(append(result.Last.EvidenceRefs, result.EvidenceRefs...))
		}
		return result
	}
	proposal, ok := autoprogrammingResidentSelfRepairProposalV0(ctx, stack, result.RunRef, result.EvidenceRefs)
	if !ok {
		result.NextActions = compactStringsV0(append(result.NextActions,
			"preserve_blocker_evidence",
			"ask_director_for_repair_write_set_or_isolated_refs",
		))
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs,
			"evidence-ref-autoprogramming-resident-selfrepair-needs-director-query",
		))
		return result
	}
	prepared := autoprogrammingResidentPrepareSelfRepairV0(ctx, input, stack, proposal)
	if prepared.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0 || !prepared.Accepted {
		result.NextActions = compactStringsV0(append(result.NextActions,
			"preserve_blocker_evidence",
			"repair_autoprogramming_prepare_run_result",
		))
		result.Errores = append(result.Errores, prepared.Errores...)
		return result
	}
	result.RepairRunRefs = compactStringsV0(append(result.RepairRunRefs, prepared.RunRef))
	result.NextActions = compactStringsV0(append(result.NextActions,
		"continue_resident_backlog_after_repair_followup",
	))
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs,
		autoprogrammingResidentRepairEvidenceV0,
		"autoprogramming-resident-selfrepair-run:"+prepared.RunRef,
	))
	if result.Last.Status != "" {
		result.Last.EvidenceRefs = compactStringsV0(append(result.Last.EvidenceRefs, result.EvidenceRefs...))
	}
	return result
}

func autoprogrammingResidentShouldRepairV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
	supervised CodexSupervisorResultV0,
	result orquestamcp.MCPRunSupervisorToolResultV0,
) bool {
	if !input.ResidentMode || result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
		return false
	}
	if strings.TrimSpace(result.RunRef) == "" {
		return false
	}
	if supervised.Last.Status == CodexSupervisorRuntimeFailedV0 {
		return true
	}
	if supervised.Last.Status != CodexSupervisorRuntimeStoppedV0 {
		return false
	}
	for _, ref := range result.EvidenceRefs {
		value := strings.TrimSpace(ref)
		if strings.Contains(value, "blocked") ||
			strings.Contains(value, "required-tests-failed") ||
			strings.Contains(value, "required-tests-evidence-missing") {
			return true
		}
	}
	return false
}

func autoprogrammingResidentSelfRepairProposalV0(
	ctx context.Context,
	stack StackV0,
	runRef string,
	evidenceRefs []string,
) (orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0, bool) {
	if stack.Ports.RunStore == nil || stack.Ports.DirectorTaskStore == nil {
		return orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{}, false
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, strings.TrimSpace(runRef))
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{}, false
	}
	task, ok := autoprogrammingResidentRepairTaskV0(ctx, stack, run)
	if !ok {
		return orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{}, false
	}
	worktreeRef := autoprogrammingResidentContextRefV0(task.ContextRefs, "worktree_ref:")
	branchRef := autoprogrammingResidentContextRefV0(task.ContextRefs, "branch_ref:")
	if worktreeRef == "" || branchRef == "" || len(task.WriteSet) == 0 || len(task.RequiredTests) == 0 {
		return orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{}, false
	}
	summary := autoprogrammingResidentFailureSummaryV0(run.RunID, task.TaskID, evidenceRefs)
	return orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{
		RequestRef:         "request-ref-autoprogramming-resident-selfrepair-" + autoprogrammingBridgeHashRefV0(summary),
		ProjectRef:         run.ProjectRef,
		WorktreeRef:        worktreeRef,
		WorktreeIsolated:   true,
		BranchRef:          branchRef,
		ObservedBy:         "orquesta-autoprogramming-resident",
		SourceRunRef:       run.RunID,
		SourceTaskRef:      task.TaskID,
		FailureKind:        "resident_blocker",
		FailureSummary:     summary,
		SuggestedArea:      autoprogrammingResidentAreaV0(task),
		SuggestedWriteSet:  append([]string(nil), task.WriteSet...),
		RequiredTests:      append([]string(nil), task.RequiredTests...),
		AcceptanceCriteria: autoprogrammingResidentAcceptanceCriteriaV0(task),
		CompactRules:       []string{"comunicacion breve", "por defecto reparar y seguir"},
		ContextRefs:        []string{"source_run_ref:" + run.RunID, "source_task_ref:" + task.TaskID},
		EvidenceRefs:       compactStringsV0(evidenceRefs),
		PriorityScore:      orquestaautoprogramming.AutoprogrammingSelfImprovementDefaultPriorityScoreV0,
	}, true
}

func autoprogrammingResidentRepairTaskV0(
	ctx context.Context,
	stack StackV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.WorkflowTaskV0, bool) {
	taskRefs := autoprogrammingResidentOpenTaskRefsV0(run)
	tasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, run.RunID, taskRefs)
	if err != nil {
		return orquestacoreworkflow.WorkflowTaskV0{}, false
	}
	for _, task := range tasks {
		if len(compactStringsV0(task.WriteSet)) > 0 && len(compactStringsV0(task.RequiredTests)) > 0 {
			return task, true
		}
	}
	return orquestacoreworkflow.WorkflowTaskV0{}, false
}

func autoprogrammingResidentOpenTaskRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	refs := []string{}
	for _, taskRef := range run.Tasks {
		if !codexStackStringInSetV0(run.ClosedTasks, taskRef) {
			refs = append(refs, taskRef)
		}
	}
	if len(refs) == 0 {
		refs = append(refs, run.Tasks...)
	}
	return compactStringsV0(refs)
}

func autoprogrammingResidentPrepareSelfRepairV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	stack StackV0,
	proposal orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0 {
	prepare := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		&stack,
		input.OccurredAt,
		"orquesta-autoprogramming-resident",
		stack.Stores.RunQueue,
		stack.RunQueue,
		stack.Clock,
	)
	self := orquestamcp.NewMCPAutoprogrammingSelfImprovementToolExecutorV0(prepare)
	out, err := self.Execute(ctx, orquestamcp.MCPAutoprogrammingSelfImprovementToolInputV0{
		RequestID:      proposal.RequestRef,
		CorrelationID:  firstNonEmptyQueuedSourceV0(input.CorrelationID, input.RequestID, proposal.RequestRef),
		AutoPrepareRun: true,
		Proposal:       proposal,
	})
	if err != nil || out.PreparedRun == nil {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(
			orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{RequestID: proposal.RequestRef},
			"autoprogramming_resident_selfrepair_error",
			"self_improvement",
			fmt.Sprint(err),
		)
	}
	return *out.PreparedRun
}

func autoprogrammingResidentContextRefV0(values []string, prefix string) string {
	for _, value := range values {
		if tail, ok := strings.CutPrefix(strings.TrimSpace(value), prefix); ok {
			return strings.TrimSpace(tail)
		}
	}
	return ""
}

func autoprogrammingResidentFailureSummaryV0(runRef string, taskRef string, evidenceRefs []string) string {
	evidenceCount := len(compactStringsV0(evidenceRefs))
	return "Bloqueo recuperable en modo residente; run_ref=" + strings.TrimSpace(runRef) +
		"; task_ref=" + strings.TrimSpace(taskRef) + fmt.Sprintf("; evidence_count=%d", evidenceCount)
}

func autoprogrammingResidentAreaV0(task orquestacoreworkflow.WorkflowTaskV0) string {
	for _, path := range task.WriteSet {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 2 && parts[0] == "modulos" {
			return strings.TrimPrefix(parts[1], "orquesta-")
		}
	}
	return "automejora"
}

func autoprogrammingResidentAcceptanceCriteriaV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) []string {
	criteria := append([]string(nil), task.AcceptanceCriteria...)
	criteria = append(criteria,
		"bloqueo recuperable produce tarea durable de reparacion",
		"entregas utiles se conservan como evidencia",
	)
	return compactStringsV0(criteria)
}
