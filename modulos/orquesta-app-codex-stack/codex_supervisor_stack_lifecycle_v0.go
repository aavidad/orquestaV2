package orquestaappcodexstack

import (
	"context"
	"errors"
	"strconv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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
	enriched, err := lifecycle.Stack.enrichQueuedOperationalDirectorDrainRequestV0(ctx, request)
	if err != nil {
		return CodexSupervisorRuntimeSnapshotV0{
			Status:     CodexSupervisorRuntimeFailedV0,
			SessionRef: strings.TrimSpace(request.RunRef),
		}, err
	}
	request = enriched
	result, err := lifecycle.Stack.DrainRunV0(ctx, request)
	if err == nil {
		err = lifecycle.syncDirectDrainTerminalQueueStatusV0(ctx, result)
	}
	snapshot := lifecycle.codexSupervisorSnapshotFromDrainV0(ctx, request.RunRef, request.OperationalDirectorPlanRef, result)
	if err != nil {
		if codexSupervisorDrainStopPendingErrorV0(err, result) {
			snapshot.Status = CodexSupervisorRuntimeStopPendingV0
			snapshot.EvidenceRefs = compactStringsV0(append(
				snapshot.EvidenceRefs,
				"evidence-ref-codex-supervisor-stop-pending-runtime-not-confirmed",
			))
			return snapshot, nil
		}
		if codexSupervisorSnapshotNeedsOperationalReplanV0(snapshot) {
			snapshot.Status = CodexSupervisorRuntimeNeedsReplanV0
			snapshot.EvidenceRefs = compactStringsV0(append(
				snapshot.EvidenceRefs,
				"evidence-ref-codex-supervisor-operational-plan-needs-replan",
			))
			if syncErr := lifecycle.syncDirectDrainRecoverableBlockedQueueStatusV0(ctx, request.RunRef); syncErr != nil {
				return snapshot, syncErr
			}
			return snapshot, nil
		}
		if codexSupervisorDrainRecoverableBlockedV0(result) ||
			codexSupervisorSnapshotRecoverablePlanBlockedV0(snapshot) {
			snapshot.Status = CodexSupervisorRuntimeStoppedV0
			snapshot.EvidenceRefs = compactStringsV0(append(
				snapshot.EvidenceRefs,
				"evidence-ref-codex-supervisor-stack-drain-recoverable-blocked",
			))
			if syncErr := lifecycle.syncDirectDrainRecoverableBlockedQueueStatusV0(ctx, request.RunRef); syncErr != nil {
				return snapshot, syncErr
			}
			return snapshot, nil
		}
		snapshot.Status = CodexSupervisorRuntimeFailedV0
	}
	return snapshot, err
}

func (lifecycle CodexSupervisorStackLifecycleV0) syncDirectDrainTerminalQueueStatusV0(
	ctx context.Context,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) error {
	if lifecycle.Stack.Stores.RunQueue == nil {
		return nil
	}
	queueStatus, err := lifecycle.Stack.stackDrainQueueStatusForCoordinatorV0(ctx, result)
	if err != nil || strings.TrimSpace(queueStatus) == "" || orquestarunqueue.IsExecutableRunStatusV0(queueStatus) {
		return nil
	}
	queueConfig := normalizeRunQueueConfigV0(lifecycle.Stack.RunQueue)
	candidates, err := lifecycle.Stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             queueConfig.QueueRef,
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		return err
	}
	runRef := strings.TrimSpace(result.Final.Run.RunID)
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.RunRef) != runRef {
			continue
		}
		_, err = lifecycle.Stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
			RunRef:           candidate.RunRef,
			QueueRef:         queueConfig.QueueRef,
			AppRef:           candidate.AppRef,
			Status:           queueStatus,
			PriorityScore:    candidate.PriorityScore,
			UpdatedAt:        stackNowV0(lifecycle.Stack.Clock),
			FairnessGroupRef: candidate.FairnessGroupRef,
			AttemptGroup:     candidate.AttemptGroup,
			ParentRunRef:     candidate.ParentRunRef,
			SupersedesRunRef: candidate.SupersedesRunRef,
			RescueReason:     candidate.RescueReason,
			RequestedBy:      "codex-supervisor-direct-drain",
			Reason:           "supervise directo sincronizo estado terminal de cola",
			IdempotencyKey:   "codex-supervisor-direct-drain-terminal:" + runRef + ":" + queueStatus,
			EvidenceRefs: compactStringsV0(append(
				candidate.EvidenceRefs,
				"evidence-ref-codex-supervisor-direct-drain-terminal-queue-sync",
			)),
			WorksetClaims: candidate.WorksetClaims,
		})
		return err
	}
	return nil
}

func (lifecycle CodexSupervisorStackLifecycleV0) syncDirectDrainRecoverableBlockedQueueStatusV0(
	ctx context.Context,
	runRef string,
) error {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" || lifecycle.Stack.Stores.RunQueue == nil {
		return nil
	}
	queueConfig := normalizeRunQueueConfigV0(lifecycle.Stack.RunQueue)
	candidates, err := lifecycle.Stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             queueConfig.QueueRef,
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		return err
	}
	queueStatus := orquestarunqueue.RunStatusStoppedV0
	if lifecycle.Stack.Stores.RunStore != nil {
		run, err := lifecycle.Stack.Stores.RunStore.LoadRunV0(ctx, runRef)
		if err == nil && run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
			queueStatus = orquestarunqueue.RunStatusClosedV0
		}
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.RunRef) != runRef {
			continue
		}
		_, err = lifecycle.Stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
			RunRef:           candidate.RunRef,
			QueueRef:         queueConfig.QueueRef,
			AppRef:           candidate.AppRef,
			Status:           queueStatus,
			PriorityScore:    candidate.PriorityScore,
			UpdatedAt:        stackNowV0(lifecycle.Stack.Clock),
			FairnessGroupRef: candidate.FairnessGroupRef,
			AttemptGroup:     candidate.AttemptGroup,
			ParentRunRef:     candidate.ParentRunRef,
			SupersedesRunRef: candidate.SupersedesRunRef,
			RescueReason:     candidate.RescueReason,
			RequestedBy:      "codex-supervisor-direct-drain",
			Reason:           "supervise directo reconcilio bloqueo operativo recuperable",
			IdempotencyKey:   "codex-supervisor-direct-drain-recoverable-blocked:" + runRef + ":" + queueStatus,
			EvidenceRefs: compactStringsV0(append(
				candidate.EvidenceRefs,
				"evidence-ref-codex-supervisor-direct-drain-recoverable-blocked-queue-sync",
			)),
			WorksetClaims: candidate.WorksetClaims,
		})
		return err
	}
	return nil
}

func (lifecycle CodexSupervisorStackLifecycleV0) runGlobalSupervisorV0(
	ctx context.Context,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	result, err := lifecycle.Stack.RunGlobalSupervisorV0(ctx, lifecycle.SupervisorCommand)
	snapshot := codexSupervisorSnapshotFromGlobalSupervisorV0(result)
	if err == nil && lifecycle.codexSupervisorGlobalHasDeliveredOpenRunV0(ctx, result) {
		snapshot.Status = CodexSupervisorRuntimeRunningV0
		snapshot.EvidenceRefs = compactStringsV0(append(
			snapshot.EvidenceRefs,
			"evidence-ref-codex-supervisor-global-delivered-open",
		))
	}
	if err != nil {
		snapshot.Status = CodexSupervisorRuntimeFailedV0
	}
	return snapshot, err
}

func (lifecycle CodexSupervisorStackLifecycleV0) codexSupervisorGlobalHasDeliveredOpenRunV0(
	ctx context.Context,
	result orquestarunsupervisor.RunSupervisorResultV0,
) bool {
	if strings.TrimSpace(result.StopReason) != orquestarunsupervisor.RunSupervisorStopNoExecutionV0 ||
		lifecycle.Stack.Stores.RunQueue == nil ||
		lifecycle.Stack.Stores.RunStore == nil {
		return false
	}
	queueConfig := normalizeRunQueueConfigV0(lifecycle.Stack.RunQueue)
	candidates, err := lifecycle.Stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             queueConfig.QueueRef,
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		return false
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.Status) != orquestarunqueue.RunStatusDeliveredV0 {
			continue
		}
		run, err := lifecycle.Stack.Stores.RunStore.LoadRunV0(ctx, candidate.RunRef)
		if err == nil &&
			run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 &&
			codexSupervisorRunHasOpenDeliveredTasksV0(run) {
			return true
		}
	}
	return false
}

func (lifecycle CodexSupervisorStackLifecycleV0) codexSupervisorSnapshotFromDrainV0(
	ctx context.Context,
	runRef string,
	planRef string,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) CodexSupervisorRuntimeSnapshotV0 {
	run, projectionEvidenceRefs := lifecycle.codexSupervisorDrainRunForProjectionV0(ctx, runRef, result.Final.Run)
	evidenceRefs := compactStringsV0(append(
		[]string{"evidence-ref-codex-supervisor-stack-drain", string(result.Status)},
		append(
			stackDrainEvidenceRefsV0(result),
			lifecycle.codexSupervisorOperationalBlockedEvidenceRefsV0(ctx, runRef, planRef, result)...,
		)...,
	))
	evidenceRefs = compactStringsV0(append(evidenceRefs,
		append(projectionEvidenceRefs, codexSupervisorDrainRunProjectionEvidenceRefsV0(run)...)...,
	))
	status := codexSupervisorRuntimeStateFromDrainLoopV0(result.Final)
	if status == CodexSupervisorRuntimeDoneV0 &&
		lifecycle.codexSupervisorDrainHasOpenRunWorkV0(ctx, run) {
		status = CodexSupervisorRuntimeRunningV0
		evidenceRefs = compactStringsV0(append(
			evidenceRefs,
			"evidence-ref-codex-supervisor-stack-drain-open-run-work",
		))
	}
	if codexSupervisorEvidenceRefsContainPartV0(evidenceRefs, "operational-director-plan-state:blocked") {
		status = CodexSupervisorRuntimeStoppedV0
	}
	agentRef := codexSupervisorLastRefV0(stackDrainPendingStartedAgentRefsV0(run))
	processRef, processStatus, processEvidenceRefs := lifecycle.codexSupervisorLiveProcessProjectionV0(ctx, run, status, agentRef)
	if processStatus != "" {
		status = processStatus
	}
	evidenceRefs = compactStringsV0(append(evidenceRefs, processEvidenceRefs...))
	return CodexSupervisorRuntimeSnapshotV0{
		Status:       status,
		SessionRef:   strings.TrimSpace(runRef),
		AgentRef:     strings.TrimSpace(agentRef),
		ProcessRef:   strings.TrimSpace(processRef),
		EvidenceRefs: evidenceRefs,
		Diagnostics:  append([]orquestaruncoordinator.RunDrainDiagnosticV0(nil), stackDrainDiagnosticsV0(result)...),
	}
}

func (lifecycle CodexSupervisorStackLifecycleV0) codexSupervisorDrainRunForProjectionV0(
	ctx context.Context,
	runRef string,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, []string) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" || lifecycle.Stack.Stores.RunStore == nil {
		return run, nil
	}
	loaded, err := lifecycle.Stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return run, []string{"evidence-ref-codex-supervisor-drain-runstore-load-failed"}
	}
	return loaded, []string{"evidence-ref-codex-supervisor-drain-runstore-loaded"}
}

func codexSupervisorDrainRunProjectionEvidenceRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	return []string{
		"evidence-ref-codex-supervisor-drain-projection-tasks-" + strconv.Itoa(len(compactStringsV0(run.Tasks))),
		"evidence-ref-codex-supervisor-drain-projection-open-tasks-" + strconv.Itoa(len(stackDrainOpenTaskRefsV0(run))),
		"evidence-ref-codex-supervisor-drain-projection-requested-agents-" + strconv.Itoa(len(stackDrainRequestedAgentRefsV0(run))),
	}
}

func (lifecycle CodexSupervisorStackLifecycleV0) codexSupervisorLiveProcessProjectionV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	status CodexSupervisorRuntimeStateV0,
	agentRef string,
) (string, CodexSupervisorRuntimeStateV0, []string) {
	agentRef = strings.TrimSpace(agentRef)
	if status == CodexSupervisorRuntimeWaitingOutboxV0 || agentRef == "" {
		return "", "", nil
	}
	if !codexSupervisorRuntimeNeedsLiveProcessV0(status) {
		return "", "", nil
	}
	refs := []string{"evidence-ref-codex-supervisor-live-process-check"}
	if lifecycle.Stack.Stores.ProcessRegistry == nil {
		return "", CodexSupervisorRuntimeStalledV0, append(refs, "evidence-ref-codex-supervisor-process-registry-missing")
	}
	if lifecycle.Stack.CodexSnapshotSource == nil {
		return "", CodexSupervisorRuntimeStalledV0, append(refs, "evidence-ref-codex-supervisor-snapshot-source-missing")
	}
	record, err := lifecycle.Stack.Stores.ProcessRegistry.ResolveAgentProcessV0(ctx, strings.TrimSpace(run.RunID), agentRef)
	if err != nil {
		return "", CodexSupervisorRuntimeStalledV0, append(refs, "evidence-ref-codex-supervisor-process-record-missing")
	}
	processRef := strings.TrimSpace(record.ProcessRef)
	if processRef == "" {
		return "", CodexSupervisorRuntimeStalledV0, append(refs, "evidence-ref-codex-supervisor-process-ref-missing")
	}
	snapshot, err := lifecycle.Stack.CodexSnapshotSource.SnapshotV0(processRef)
	if err != nil {
		if codexStackProcessRuntimeMissingV0(err) {
			return processRef, CodexSupervisorRuntimeStalledV0, append(refs, "evidence-ref-codex-supervisor-process-snapshot-missing")
		}
		return processRef, CodexSupervisorRuntimeStalledV0, append(refs, "evidence-ref-codex-supervisor-process-snapshot-error")
	}
	if snapshot.Status == orquestaruntime.ProcessRuntimeRunningV0 ||
		snapshot.Status == orquestaruntime.ProcessRuntimeStoppingV0 {
		return processRef, CodexSupervisorRuntimeRunningLiveV0, append(refs, "evidence-ref-codex-supervisor-process-live")
	}
	return processRef, CodexSupervisorRuntimeStalledV0, append(refs, "evidence-ref-codex-supervisor-process-not-live")
}

func codexSupervisorRuntimeNeedsLiveProcessV0(status CodexSupervisorRuntimeStateV0) bool {
	switch status {
	case CodexSupervisorRuntimeRunningV0, CodexSupervisorRuntimeRunningLiveV0:
		return true
	default:
		return false
	}
}

func codexSupervisorDrainRecoverableBlockedV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) bool {
	return result.Status == orquestacionnucleoapp.ProgressiveLoopStatusBlockedV0 ||
		result.Final.Status == orquestacionnucleoapp.ProgressiveLoopStatusBlockedV0
}

func codexSupervisorDrainStopPendingErrorV0(
	err error,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) bool {
	if err == nil {
		return false
	}
	var coreErr orquestacionnucleoapp.ErrorV0
	if !errors.As(err, &coreErr) {
		return false
	}
	return coreErr.Code == orquestacionnucleoapp.ErrNucleoOrquestacionInvalidoV0 &&
		strings.TrimSpace(coreErr.Field) == "process_runtime.status"
}

func codexSupervisorSnapshotRecoverablePlanBlockedV0(
	snapshot CodexSupervisorRuntimeSnapshotV0,
) bool {
	return codexSupervisorEvidenceRefsContainPartV0(snapshot.EvidenceRefs, "operational-director-plan-state:blocked") &&
		codexSupervisorEvidenceRefsContainPartV0(snapshot.EvidenceRefs, "operational-closure-run-not-active")
}

func codexSupervisorSnapshotNeedsOperationalReplanV0(
	snapshot CodexSupervisorRuntimeSnapshotV0,
) bool {
	if !codexSupervisorEvidenceRefsContainPartV0(snapshot.EvidenceRefs, "operational-director-plan-state:blocked") ||
		!codexSupervisorEvidenceRefsContainPartV0(snapshot.EvidenceRefs, "external-wait-exhausted") {
		return false
	}
	openTasks, hasOpenTasks := codexSupervisorEvidenceCountByPrefixV0(
		snapshot.EvidenceRefs,
		"evidence-ref-codex-supervisor-drain-projection-open-tasks-",
	)
	requestedAgents, hasRequestedAgents := codexSupervisorEvidenceCountByPrefixV0(
		snapshot.EvidenceRefs,
		"evidence-ref-codex-supervisor-drain-projection-requested-agents-",
	)
	return hasOpenTasks && openTasks > 0 && hasRequestedAgents && requestedAgents == 0
}

func codexSupervisorEvidenceCountByPrefixV0(
	values []string,
	prefix string,
) (int, bool) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return 0, false
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !strings.HasPrefix(value, prefix) {
			continue
		}
		count, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(value, prefix)))
		if err != nil {
			return 0, false
		}
		return count, true
	}
	return 0, false
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

func (lifecycle CodexSupervisorStackLifecycleV0) codexSupervisorDrainHasOpenRunWorkV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return false
	}
	if len(stackDrainOpenTaskRefsV0(run)) > 0 {
		if open, err := RunHasOpenProgrammingOrAutonomyTasksV0(ctx, lifecycle.Stack.Stores.TaskStore, run); err != nil || open {
			return true
		}
	}
	if codexSupervisorRunHasOpenNonOperationalDirectorTasksV0(ctx, lifecycle.Stack.Stores.TaskStore, run) {
		return true
	}
	return codexSupervisorRunHasOpenDeliveredNonOperationalDirectorTasksV0(ctx, lifecycle.Stack.Stores.TaskStore, run)
}

func codexSupervisorRunHasOpenNonOperationalDirectorTasksV0(
	ctx context.Context,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	openRefs := stackDrainOpenTaskRefsV0(run)
	if len(openRefs) == 0 {
		return false
	}
	if taskStore == nil {
		return true
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(ctx, run.RunID, openRefs)
	if err != nil {
		return true
	}
	for _, task := range tasks {
		if !codexSupervisorWorkflowTaskIsInternalOperationalDirectorWorkV0(task) {
			return true
		}
	}
	return false
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

func codexSupervisorRunHasOpenDeliveredNonOperationalDirectorTasksV0(
	ctx context.Context,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	refs := []string{}
	for _, taskRef := range compactStringsV0(run.DeliveredTasks) {
		if !codexStackStringInSetV0(run.ClosedTasks, taskRef) {
			refs = append(refs, taskRef)
		}
	}
	if len(refs) == 0 {
		return false
	}
	if taskStore == nil {
		return true
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(ctx, run.RunID, refs)
	if err != nil {
		return true
	}
	for _, task := range tasks {
		if !codexSupervisorWorkflowTaskIsInternalOperationalDirectorWorkV0(task) {
			return true
		}
	}
	return false
}

func codexSupervisorWorkflowTaskIsInternalOperationalDirectorWorkV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	return codexStackWorkflowTaskHasOperationalDirectorFunctionContractV0(task) ||
		codexStackWorkflowTaskHasOperationalDirectorContextRefV0(task)
}

func codexSupervisorRuntimeStateFromLoopV0(
	status orquestacionnucleoapp.ProgressiveLoopStatusV0,
	runStatus orquestacoreworkflow.OrchestrationRunStatusV0,
) CodexSupervisorRuntimeStateV0 {
	if runStatus == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return CodexSupervisorRuntimeDoneV0
	}
	switch status {
	case orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0:
		return CodexSupervisorRuntimeDoneV0
	case orquestacionnucleoapp.ProgressiveLoopStatusRunTerminalV0:
		return CodexSupervisorRuntimeStoppedV0
	case orquestacionnucleoapp.ProgressiveLoopStatusStopErrorV0,
		orquestacionnucleoapp.ProgressiveLoopStatusRunCanceledV0:
		return CodexSupervisorRuntimeFailedV0
	case orquestacionnucleoapp.ProgressiveLoopStatusWaitUnhandledOutboxV0:
		return CodexSupervisorRuntimeWaitingOutboxV0
	case orquestacionnucleoapp.ProgressiveLoopStatusNeedsDirectorV0:
		return CodexSupervisorRuntimeRunningV0
	case orquestacionnucleoapp.ProgressiveLoopStatusBlockedV0,
		orquestacionnucleoapp.ProgressiveLoopStatusRunPausedV0:
		return CodexSupervisorRuntimeStoppedV0
	case orquestacionnucleoapp.ProgressiveLoopStatusRunStopRequestedV0:
		return CodexSupervisorRuntimeStopPendingV0
	default:
		return CodexSupervisorRuntimeRunningV0
	}
}

func codexSupervisorLastRefV0(values []string) string {
	for index := len(values) - 1; index >= 0; index-- {
		if ref := strings.TrimSpace(values[index]); ref != "" {
			return ref
		}
	}
	return ""
}
