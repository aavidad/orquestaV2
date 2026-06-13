package orquestaappcodexstack

import (
	"strings"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func codexSupervisorSnapshotFromGlobalSupervisorV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) CodexSupervisorRuntimeSnapshotV0 {
	return CodexSupervisorRuntimeSnapshotV0{
		Status:       codexSupervisorRuntimeStateFromGlobalSupervisorV0(result),
		SessionRef:   codexSupervisorLastGlobalExecutionRunRefV0(result),
		EvidenceRefs: codexSupervisorGlobalEvidenceRefsV0(result),
		Diagnostics:  append([]orquestaruncoordinator.RunDrainDiagnosticV0(nil), result.Diagnostics...),
	}
}

func codexSupervisorRuntimeStateFromGlobalSupervisorV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) CodexSupervisorRuntimeStateV0 {
	switch result.StopReason {
	case orquestarunsupervisor.RunSupervisorStopTickErrorV0,
		orquestarunsupervisor.RunSupervisorStopNoTickerV0:
		return CodexSupervisorRuntimeFailedV0
	}
	seenStopped := false
	seenRunning := false
	seenWaitingOutbox := false
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			if strings.TrimSpace(execution.QueueStatus) == orquestarunqueue.RunStatusRunningV0 {
				seenRunning = true
				continue
			}
			state := codexSupervisorRuntimeStateFromOutcomeV0(execution.Outcome)
			switch state {
			case CodexSupervisorRuntimeFailedV0:
				return state
			case CodexSupervisorRuntimeWaitingOutboxV0:
				seenWaitingOutbox = true
			case CodexSupervisorRuntimeRunningV0:
				seenRunning = true
			case CodexSupervisorRuntimeStoppedV0:
				seenStopped = true
			}
		}
	}
	if seenRunning {
		return CodexSupervisorRuntimeRunningV0
	}
	if seenWaitingOutbox {
		return CodexSupervisorRuntimeWaitingOutboxV0
	}
	if seenStopped {
		return CodexSupervisorRuntimeStoppedV0
	}
	return CodexSupervisorRuntimeDoneV0
}

func codexSupervisorLastGlobalExecutionRunRefV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) string {
	for tickIndex := len(result.Ticks) - 1; tickIndex >= 0; tickIndex-- {
		executions := result.Ticks[tickIndex].Result.Executions
		for executionIndex := len(executions) - 1; executionIndex >= 0; executionIndex-- {
			if ref := strings.TrimSpace(executions[executionIndex].RunRef); ref != "" {
				return ref
			}
		}
	}
	return ""
}

func codexSupervisorGlobalEvidenceRefsV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) []string {
	refs := []string{
		"evidence-ref-codex-supervisor-stack-global",
		strings.TrimSpace(result.StopReason),
	}
	refs = append(refs, result.ErrorRunRefs...)
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			refs = append(refs, execution.EvidenceRefs...)
			refs = append(refs, strings.TrimSpace(execution.Outcome))
		}
	}
	for _, diagnostic := range result.Diagnostics {
		refs = append(refs,
			diagnostic.RunRef,
			diagnostic.Kind,
			diagnostic.MessageID,
			diagnostic.DispatchRef,
		)
	}
	return compactStringsV0(refs)
}
