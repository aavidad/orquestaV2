package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type CodexSupervisorStackLifecycleV0 struct {
	Stack             StackV0
	RunRef            string
	DrainRequest      DrainRunRequestV0
	SupervisorCommand orquestarunsupervisor.RunSupervisorCommandV0
}

var _ CodexSupervisorAgentLifecyclePortV0 = CodexSupervisorStackLifecycleV0{}

func (lifecycle CodexSupervisorStackLifecycleV0) LaunchV0(
	ctx context.Context,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	return lifecycle.stepV0(ctx)
}

func (lifecycle CodexSupervisorStackLifecycleV0) ContinueV0(
	ctx context.Context,
	_ string,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	return lifecycle.stepV0(ctx)
}

func (lifecycle CodexSupervisorStackLifecycleV0) stepV0(
	ctx context.Context,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	if strings.TrimSpace(lifecycle.RunRef) != "" ||
		strings.TrimSpace(lifecycle.DrainRequest.RunRef) != "" {
		return lifecycle.drainRunV0(ctx)
	}
	return lifecycle.runGlobalSupervisorV0(ctx)
}

func (lifecycle CodexSupervisorStackLifecycleV0) drainRunV0(
	ctx context.Context,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	request := lifecycle.DrainRequest
	if strings.TrimSpace(request.RunRef) == "" {
		request.RunRef = strings.TrimSpace(lifecycle.RunRef)
	}
	result, err := lifecycle.Stack.DrainRunV0(ctx, request)
	snapshot := codexSupervisorSnapshotFromDrainV0(request.RunRef, result)
	if err != nil {
		snapshot.Status = CodexSupervisorRuntimeFailedV0
	}
	return snapshot, err
}

func (lifecycle CodexSupervisorStackLifecycleV0) runGlobalSupervisorV0(
	ctx context.Context,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	result, err := lifecycle.Stack.RunGlobalSupervisorV0(ctx, lifecycle.SupervisorCommand)
	snapshot := codexSupervisorSnapshotFromGlobalSupervisorV0(result)
	if err != nil {
		snapshot.Status = CodexSupervisorRuntimeFailedV0
	}
	return snapshot, err
}

func codexSupervisorSnapshotFromDrainV0(
	runRef string,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) CodexSupervisorRuntimeSnapshotV0 {
	return CodexSupervisorRuntimeSnapshotV0{
		Status:     codexSupervisorRuntimeStateFromLoopV0(result.Final.Status, result.Final.Run.Status),
		SessionRef: strings.TrimSpace(runRef),
		ProcessRef: codexSupervisorLastRefV0(result.Final.Run.StartedAgents),
		EvidenceRefs: compactStringsV0(append(
			[]string{"evidence-ref-codex-supervisor-stack-drain", string(result.Status)},
			stackDrainEvidenceRefsV0(result)...,
		)),
	}
}

func codexSupervisorSnapshotFromGlobalSupervisorV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) CodexSupervisorRuntimeSnapshotV0 {
	return CodexSupervisorRuntimeSnapshotV0{
		Status:       codexSupervisorRuntimeStateFromGlobalSupervisorV0(result),
		SessionRef:   codexSupervisorLastGlobalExecutionRunRefV0(result),
		EvidenceRefs: codexSupervisorGlobalEvidenceRefsV0(result),
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
	seenExecution := false
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			seenExecution = true
			state := codexSupervisorRuntimeStateFromOutcomeV0(execution.Outcome)
			switch state {
			case CodexSupervisorRuntimeFailedV0:
				return state
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
	if seenStopped {
		return CodexSupervisorRuntimeStoppedV0
	}
	if !seenExecution {
		return CodexSupervisorRuntimeDoneV0
	}
	return CodexSupervisorRuntimeDoneV0
}

func codexSupervisorRuntimeStateFromOutcomeV0(
	outcome string,
) CodexSupervisorRuntimeStateV0 {
	return codexSupervisorRuntimeStateFromLoopV0(
		orquestacionnucleoapp.ProgressiveLoopStatusV0(strings.TrimSpace(outcome)),
		"",
	)
}

func codexSupervisorRuntimeStateFromLoopV0(
	status orquestacionnucleoapp.ProgressiveLoopStatusV0,
	runStatus orquestacoreworkflow.OrchestrationRunStatusV0,
) CodexSupervisorRuntimeStateV0 {
	if runStatus == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return CodexSupervisorRuntimeDoneV0
	}
	switch status {
	case orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		orquestacionnucleoapp.ProgressiveLoopStatusRunTerminalV0:
		return CodexSupervisorRuntimeDoneV0
	case orquestacionnucleoapp.ProgressiveLoopStatusStopErrorV0,
		orquestacionnucleoapp.ProgressiveLoopStatusRunCanceledV0:
		return CodexSupervisorRuntimeFailedV0
	case orquestacionnucleoapp.ProgressiveLoopStatusNeedsDirectorV0,
		orquestacionnucleoapp.ProgressiveLoopStatusBlockedV0,
		orquestacionnucleoapp.ProgressiveLoopStatusRunPausedV0,
		orquestacionnucleoapp.ProgressiveLoopStatusRunStopRequestedV0:
		return CodexSupervisorRuntimeStoppedV0
	default:
		return CodexSupervisorRuntimeRunningV0
	}
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
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			refs = append(refs, execution.EvidenceRefs...)
			refs = append(refs, strings.TrimSpace(execution.Outcome))
		}
	}
	return compactStringsV0(refs)
}

func codexSupervisorLastRefV0(values []string) string {
	for index := len(values) - 1; index >= 0; index-- {
		if ref := strings.TrimSpace(values[index]); ref != "" {
			return ref
		}
	}
	return ""
}
