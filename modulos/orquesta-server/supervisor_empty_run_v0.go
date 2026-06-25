package orquestaserver

import (
	"strings"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func supervisorResultHasExternalEmptyRunV0(result orquestarunsupervisor.RunSupervisorResultV0) bool {
	if supervisorTerminalEmptyRunEvidenceV0(
		result.StopReason,
		result.StopProjection.PublicReason,
		result.StopProjection.Category,
		result.StopProjection.EvidenceRefs,
		result.Diagnostics,
	) {
		return true
	}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			if supervisorTerminalEmptyRunEvidenceV0(
				execution.Outcome,
				execution.QueueStatus,
				"",
				execution.EvidenceRefs,
				execution.Diagnostics,
			) {
				return true
			}
		}
		for _, skip := range tick.Result.Skips {
			if supervisorTerminalEmptyRunEvidenceV0(skip.Reason, skip.Status, "", nil, nil) {
				return true
			}
		}
	}
	return false
}

func supervisorTerminalEmptyRunEvidenceV0(
	primary string,
	secondary string,
	tertiary string,
	evidenceRefs []string,
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
) bool {
	if supervisorExplicitEmptyRunValueV0(primary, secondary, tertiary) {
		return true
	}
	if !supervisorTerminalRunValueV0(primary, secondary, tertiary) {
		return false
	}
	return supervisorEvidenceRefsDescribeEmptyRunV0(evidenceRefs) ||
		supervisorDiagnosticsDescribeEmptyRunV0(diagnostics)
}

func supervisorDiagnosticsDescribeEmptyRunV0(diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0) bool {
	refs := []string{}
	for _, diagnostic := range diagnostics {
		if supervisorExplicitEmptyRunValueV0(diagnostic.Kind, diagnostic.Status, diagnostic.FinalAction) {
			return true
		}
		refs = append(refs, diagnostic.EvidenceRefs...)
	}
	return supervisorEvidenceRefsDescribeEmptyRunV0(refs)
}

func supervisorEvidenceRefsDescribeEmptyRunV0(evidenceRefs []string) bool {
	tasksZero := false
	openTasksZero := false
	agentsZero := false
	for _, ref := range evidenceRefs {
		switch strings.TrimSpace(ref) {
		case "external_work_empty_run", "failed_empty_run", "empty_run", "quiescent_without_domain_progress":
			return true
		case "projection-tasks-0", "tasks-0", "tasks_total-0":
			tasksZero = true
		case "projection-open-tasks-0", "open_tasks-0", "tasks_open-0":
			openTasksZero = true
		case "projection-requested-agents-0", "requested_agents-0", "agents_total-0":
			agentsZero = true
		}
	}
	return tasksZero && openTasksZero && agentsZero
}

func supervisorExplicitEmptyRunValueV0(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "failed_empty_run", "empty_run", "external_work_empty_run", "quiescent_without_domain_progress":
			return true
		}
	}
	return false
}

func supervisorTerminalRunValueV0(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "done", "complete", "completed", "closed", "quiescent", "stop_quiescent":
			return true
		}
	}
	return false
}
