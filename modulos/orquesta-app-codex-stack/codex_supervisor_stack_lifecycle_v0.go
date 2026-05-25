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
	snapshot := lifecycle.codexSupervisorSnapshotFromDrainV0(ctx, request.RunRef, request.OperationalDirectorPlanRef, result)
	if err != nil {
		if codexSupervisorDrainRecoverableBlockedV0(result) {
			snapshot.Status = CodexSupervisorRuntimeStoppedV0
			return snapshot, nil
		}
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

func (lifecycle CodexSupervisorStackLifecycleV0) codexSupervisorSnapshotFromDrainV0(
	ctx context.Context,
	runRef string,
	planRef string,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) CodexSupervisorRuntimeSnapshotV0 {
	evidenceRefs := compactStringsV0(append(
		[]string{"evidence-ref-codex-supervisor-stack-drain", string(result.Status)},
		append(
			stackDrainEvidenceRefsV0(result),
			lifecycle.codexSupervisorOperationalBlockedEvidenceRefsV0(ctx, runRef, planRef, result)...,
		)...,
	))
	status := codexSupervisorRuntimeStateFromDrainLoopV0(result.Final)
	if status == CodexSupervisorRuntimeDoneV0 &&
		lifecycle.codexSupervisorDrainHasOpenAutoprogrammingWorkV0(ctx, result.Final.Run) {
		status = CodexSupervisorRuntimeRunningV0
	}
	if codexSupervisorEvidenceRefsContainPartV0(evidenceRefs, "operational-director-plan-state:blocked") {
		status = CodexSupervisorRuntimeStoppedV0
	}
	return CodexSupervisorRuntimeSnapshotV0{
		Status:       status,
		SessionRef:   strings.TrimSpace(runRef),
		ProcessRef:   codexSupervisorLastRefV0(result.Final.Run.StartedAgents),
		EvidenceRefs: evidenceRefs,
	}
}

func codexSupervisorDrainRecoverableBlockedV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) bool {
	return result.Status == orquestacionnucleoapp.ProgressiveLoopStatusBlockedV0 ||
		result.Final.Status == orquestacionnucleoapp.ProgressiveLoopStatusBlockedV0
}

func (lifecycle CodexSupervisorStackLifecycleV0) codexSupervisorOperationalBlockedEvidenceRefsV0(
	ctx context.Context,
	runRef string,
	planRef string,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) []string {
	refs := []string{}
	if codexSupervisorDrainRecoverableBlockedV0(result) {
		refs = append(refs, "evidence-ref-codex-supervisor-stack-drain-blocked")
	}
	refs = append(refs, codexSupervisorRunOperationalBlockerRefsV0(result.Final.Run)...)
	refs = append(refs, lifecycle.codexSupervisorPlanStateEvidenceRefsV0(ctx, runRef, planRef)...)
	return compactStringsV0(refs)
}

func codexSupervisorRunOperationalBlockerRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	refs := []string{}
	for _, taskRef := range compactStringsV0(run.Tasks) {
		refs = append(refs, orquestacoreworkflow.PendingBlockingQualityGateRefsForSubjectV0(run, taskRef)...)
	}
	if len(refs) == 0 {
		for _, gate := range run.QualityGates {
			gate = strings.TrimSpace(gate)
			if gate == "" || strings.Contains(gate, "#subject:") {
				continue
			}
			if strings.Contains(gate, "required-tests-evidence-missing") ||
				strings.Contains(gate, "#decision:blocked") {
				refs = append(refs, gate)
			}
		}
	}
	return compactStringsV0(refs)
}

func codexSupervisorEvidenceRefsContainPartV0(values []string, part string) bool {
	part = strings.TrimSpace(part)
	if part == "" {
		return false
	}
	for _, value := range values {
		if strings.Contains(strings.TrimSpace(value), part) {
			return true
		}
	}
	return false
}

func (lifecycle CodexSupervisorStackLifecycleV0) codexSupervisorPlanStateEvidenceRefsV0(
	ctx context.Context,
	runRef string,
	planRef string,
) []string {
	runRef = strings.TrimSpace(runRef)
	planRef = strings.TrimSpace(planRef)
	if runRef == "" || planRef == "" || lifecycle.Stack.Ports.OperationalPlanStateStore == nil {
		return nil
	}
	state, err := lifecycle.Stack.Ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, runRef, planRef)
	if err != nil || state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 {
		return nil
	}
	refs := []string{
		"operational-director-plan-state:" + strings.TrimSpace(string(state.Status)),
	}
	refs = append(refs, state.BlockerRefs...)
	refs = append(refs, state.EvidenceRefs...)
	for _, step := range state.Steps {
		if step.Status != "blocked" {
			continue
		}
		refs = append(refs, step.BlockerRefs...)
		refs = append(refs, step.Reason)
		refs = append(refs, step.EvidenceRefs...)
	}
	return compactStringsV0(refs)
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

func codexSupervisorRuntimeStateFromDrainLoopV0(
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) CodexSupervisorRuntimeStateV0 {
	return codexSupervisorRuntimeStateFromLoopV0(loop.Status, loop.Run.Status)
}

func (lifecycle CodexSupervisorStackLifecycleV0) codexSupervisorDrainHasOpenAutoprogrammingWorkV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!codexSupervisorRunHasOpenDeliveredTasksV0(run) {
		return false
	}
	hold, err := RunHasOpenAutoprogrammingTasksV0(ctx, lifecycle.Stack.Stores.TaskStore, run)
	return err == nil && hold
}

func codexSupervisorRunHasOpenDeliveredTasksV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	for _, taskRef := range compactStringsV0(run.DeliveredTasks) {
		if !codexStackStringInSetV0(run.ClosedTasks, taskRef) {
			return true
		}
	}
	return false
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

func codexSupervisorLastRefV0(values []string) string {
	for index := len(values) - 1; index >= 0; index-- {
		if ref := strings.TrimSpace(values[index]); ref != "" {
			return ref
		}
	}
	return ""
}
