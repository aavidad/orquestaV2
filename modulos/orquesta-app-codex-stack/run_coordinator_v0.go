package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func (stack StackV0) RunGlobalTickV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) (orquestaruncoordinator.RunCoordinatorTickResultV0, error) {
	command = stack.normalizeRunCoordinatorCommandV0(command)
	if err := stack.recoverQueuedStoppedActiveRunsV0(ctx, command); err != nil {
		return orquestaruncoordinator.RunCoordinatorTickResultV0{}, err
	}
	if err := stack.recoverQueuedControlledDomainWorkArtifactsV0(ctx, command); err != nil {
		return orquestaruncoordinator.RunCoordinatorTickResultV0{}, err
	}
	return orquestaruncoordinator.CoordinateRunsTickV0(
		ctx,
		orquestaruncoordinator.RunCoordinatorDepsV0{
			QueueReader:   stack.Stores.RunQueue,
			QueueUpdater:  stack.Stores.RunQueue,
			ControlReader: stack.Stores.RunControl,
			Drainer:       stackRunDrainerV0{stack: stack},
		},
		command,
	)
}

func (stack StackV0) normalizeRunCoordinatorCommandV0(
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) orquestaruncoordinator.RunCoordinatorTickCommandV0 {
	queue := normalizeRunQueueConfigV0(stack.RunQueue)
	command.QueueRef = firstNonEmptyQueuedSourceV0(command.QueueRef, queue.QueueRef)
	if command.QueueLimit <= 0 {
		command.QueueLimit = queue.QueueLimit
	}
	if command.MaxRuns <= 0 {
		command.MaxRuns = queue.MaxRunsPerTick
	}
	if command.OccurredAt.IsZero() {
		command.OccurredAt = stackNowV0(stack.Clock)
	}
	command.DrainLimits = normalizeGlobalDrainLimitsV0(command.DrainLimits)
	return command
}

type stackRunDrainerV0 struct {
	stack StackV0
}

func (drainer stackRunDrainerV0) DrainRunV0(
	ctx context.Context,
	request orquestaruncoordinator.RunDrainRequestV0,
) (orquestaruncoordinator.RunDrainResultV0, error) {
	drainRequest, err := drainer.stack.stackDrainRequestFromCoordinatorV0(ctx, request)
	if err != nil {
		result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{}
		return stackDrainCoordinatorResultV0(request, result, "error", "", err.Error()), err
	}
	result, err := drainer.stack.DrainRunV0(ctx, drainRequest)
	if err != nil {
		return stackDrainCoordinatorResultV0(request, result, "error", "", err.Error()), err
	}
	queueStatus, err := drainer.stack.stackDrainQueueStatusForCoordinatorV0(ctx, result)
	if err != nil {
		return stackDrainCoordinatorResultV0(request, result, "error", "", err.Error()), err
	}
	return stackDrainCoordinatorResultV0(request, result, stackDrainOutcomeV0(result), queueStatus, ""), nil
}

func stackDrainCoordinatorResultV0(
	request orquestaruncoordinator.RunDrainRequestV0,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
	outcome string,
	queueStatus string,
	errorMessage string,
) orquestaruncoordinator.RunDrainResultV0 {
	diagnostics := stackDrainDiagnosticsV0(result)
	if strings.TrimSpace(errorMessage) != "" {
		diagnostics = stackDrainDiagnosticsWithErrorV0(result, diagnostics, request.RunRef, errorMessage)
	}
	return orquestaruncoordinator.RunDrainResultV0{
		RunRef:       strings.TrimSpace(request.RunRef),
		AppRef:       strings.TrimSpace(request.AppRef),
		Outcome:      strings.TrimSpace(outcome),
		QueueStatus:  strings.TrimSpace(queueStatus),
		EvidenceRefs: stackDrainEvidenceRefsV0(result),
		Diagnostics:  diagnostics,
	}
}

func (stack StackV0) stackDrainRequestFromCoordinatorV0(
	ctx context.Context,
	request orquestaruncoordinator.RunDrainRequestV0,
) (DrainRunRequestV0, error) {
	drainRequest := stackDrainRequestFromCoordinatorV0(request)
	return stack.enrichQueuedOperationalDirectorDrainRequestV0(ctx, drainRequest)
}

func stackDrainRequestFromCoordinatorV0(
	request orquestaruncoordinator.RunDrainRequestV0,
) DrainRunRequestV0 {
	return DrainRunRequestV0{
		RunRef:               strings.TrimSpace(request.RunRef),
		OccurredAt:           formatStackCoordinatorTimeV0(request.OccurredAt),
		CorrelationID:        strings.TrimSpace(request.CorrelationID),
		MaxBursts:            request.Limits.MaxBursts,
		MaxStepsPerBurst:     request.Limits.MaxStepsPerBurst,
		MaxDispatchesPerWait: request.Limits.MaxDispatchesPerWait,
		MaxCommands:          request.Limits.MaxCommands,
		MaxOutboxPerCycle:    request.Limits.MaxOutboxPerCycle,
		MaxDecisionCycles:    request.Limits.MaxDecisionCycles,
		MaxExternalWaits:     request.Limits.MaxExternalWaits,
	}
}

func (stack StackV0) enrichQueuedOperationalDirectorDrainRequestV0(
	ctx context.Context,
	request DrainRunRequestV0,
) (DrainRunRequestV0, error) {
	if strings.TrimSpace(request.RunRef) == "" {
		return request, nil
	}
	if err := stack.repairQueuedAutoprogrammingWorkflowTasksV0(ctx, request.RunRef); err != nil {
		return DrainRunRequestV0{}, err
	}
	if strings.TrimSpace(request.OperationalDirectorPlanRef) == "" {
		ensured, err := orquestaappdirectorservice.EnsureOperationalDirectorPlanStateFromWorkflowTasksV0(
			ctx,
			orquestaappdirectorservice.ContinueAppDirectorRequestV0{
				RunRef:            request.RunRef,
				OccurredAt:        request.OccurredAt,
				CorrelationID:     request.CorrelationID,
				RequestedBy:       "orquesta-run-coordinator",
				MaxBursts:         request.MaxBursts,
				MaxStepsPerBurst:  request.MaxStepsPerBurst,
				MaxCommands:       request.MaxCommands,
				MaxOutboxPerCycle: request.MaxOutboxPerCycle,
			},
			stack.Ports,
		)
		if err != nil {
			return DrainRunRequestV0{}, err
		}
		request.OperationalDirectorPlanRef = strings.TrimSpace(ensured.OperationalDirectorPlanRef)
	}
	if len(compactStringsV0(request.WaitAgentRefs)) == 0 {
		refs, err := stack.queuedOperationalDirectorWaitAgentRefsV0(ctx, request.RunRef)
		if err != nil {
			return DrainRunRequestV0{}, err
		}
		request.WaitAgentRefs = refs
	}
	return request, nil
}

func (stack StackV0) repairQueuedAutoprogrammingWorkflowTasksV0(
	ctx context.Context,
	runRef string,
) error {
	if stack.Ports.RunStore == nil || stack.Ports.DirectorTaskStore == nil {
		return nil
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return err
	}
	openTaskRefs := stackDrainOpenTaskRefsV0(run)
	if len(openTaskRefs) == 0 {
		openTaskRefs = compactStringsV0(run.Tasks)
	}
	if !codexStackRunLooksAutoprogrammingV0(run, openTaskRefs) {
		return nil
	}
	tasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, run.RunID, openTaskRefs)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if codexStackWorkflowTaskLooksOperationalDirectorV0(task) ||
			!codexStackWorkflowTaskLooksAutoprogrammingV0(task) {
			continue
		}
		task.ContextRefs = compactStringsV0(append(
			[]string{autoprogrammingBridgeOperationalTaskSourceRefV0},
			task.ContextRefs...,
		))
		if err := stack.Ports.DirectorTaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
			if codexStackWorkflowTaskImmutableConflictV0(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func codexStackWorkflowTaskImmutableConflictV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "workflow_task" &&
		strings.Contains(issue.Message, "microtarea existente con contrato distinto")
}

func (stack StackV0) queuedOperationalDirectorWaitAgentRefsV0(
	ctx context.Context,
	runRef string,
) ([]string, error) {
	if stack.Ports.RunStore == nil || stack.Ports.DirectorTaskStore == nil {
		return nil, nil
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return nil, err
	}
	openTaskRefs := stackDrainOpenTaskRefsV0(run)
	if len(openTaskRefs) == 0 {
		openTaskRefs = compactStringsV0(run.Tasks)
	}
	tasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, run.RunID, openTaskRefs)
	if err != nil {
		return nil, err
	}
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if !codexStackWorkflowTaskLooksOperationalDirectorV0(task) {
			continue
		}
		refs = append(refs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	return compactStringsV0(refs), nil
}

func stackDrainOpenTaskRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	closed := map[string]bool{}
	for _, taskRef := range compactStringsV0(run.ClosedTasks) {
		closed[taskRef] = true
	}
	refs := make([]string, 0, len(run.Tasks))
	for _, taskRef := range compactStringsV0(run.Tasks) {
		if closed[taskRef] {
			continue
		}
		refs = append(refs, taskRef)
	}
	return refs
}

func (stack StackV0) recoverQueuedStoppedActiveRunsV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) error {
	if stack.Stores.RunQueue == nil || stack.Stores.RunControl == nil || stack.Ports.RunStore == nil {
		return nil
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             strings.TrimSpace(command.QueueRef),
			AppRefs:              append([]string(nil), command.AppRefs...),
			Limit:                command.QueueLimit,
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		state, err := stack.Stores.RunControl.ReadRunControlStateV0(
			ctx,
			orquestaruncontrol.RunControlReadRequestV0{RunRef: candidate.RunRef},
		)
		if err != nil {
			var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
			if errors.As(err, &notFound) {
				state = orquestaruncontrol.DefaultRunControlStateV0(candidate.RunRef)
			} else {
				return err
			}
		}
		run, err := stack.Ports.RunStore.LoadRunV0(ctx, candidate.RunRef)
		if err != nil {
			return err
		}
		if !stackRunIsActiveV0(run) {
			continue
		}
		if state.Status == orquestaruncontrol.RunControlStatusStoppedV0 &&
			stackRunControlCanAutoResumeQueuedCandidateV0(state) {
			state, err = stack.Stores.RunControl.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
				RunRef:         candidate.RunRef,
				RequestedBy:    "orquesta-app-codex-stack-run-control-reconciler",
				Reason:         "queued_ready_active_run_reconciled",
				IdempotencyKey: "idem-run-control-reconcile-ready-" + codexStackOperationalClosureSafeRefV0(candidate.RunRef),
				EvidenceRefs: []string{
					"evidence-ref-run-control-queued-ready-active-reconciled",
					"evidence-ref-run-control-stopped-state-preserved-in-history",
				},
			})
			if err != nil {
				return err
			}
		}
		evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
		if !evaluation.DispatchAllowed || orquestarunqueue.IsExecutableRunStatusV0(candidate.Status) {
			continue
		}
		if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
			RunRef:         candidate.RunRef,
			QueueRef:       command.QueueRef,
			AppRef:         candidate.AppRef,
			Status:         orquestarunqueue.RunStatusReadyV0,
			PriorityScore:  candidate.PriorityScore,
			UpdatedAt:      command.OccurredAt,
			RequestedBy:    "orquesta-app-codex-stack-run-control-reconciler",
			Reason:         "queued_non_executable_active_run_reconciled",
			IdempotencyKey: "idem-run-queue-reconcile-ready-" + codexStackOperationalClosureSafeRefV0(candidate.RunRef),
			EvidenceRefs: []string{
				"evidence-ref-run-queue-non-executable-active-reconciled",
				"evidence-ref-run-control-dispatch-allowed",
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func stackRunIsActiveV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	return run.Status == orquestacoreworkflow.OrchestrationRunStatusActiveV0
}

func stackRunControlCanAutoResumeQueuedCandidateV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	if stackRunControlLooksServerShutdownV0(state) {
		return false
	}
	idempotency := strings.TrimSpace(state.Meta.IdempotencyKey)
	reason := strings.TrimSpace(state.Meta.Reason)
	requestedBy := strings.TrimSpace(state.Meta.RequestedBy)
	if requestedBy == "orquesta-run-control" &&
		strings.HasPrefix(idempotency, "idem-run-control-complete-") &&
		strings.HasSuffix(idempotency, "-stopped") &&
		strings.Contains(reason, "agentes drenados por control de run") {
		return true
	}
	return false
}

func stackRunControlLooksServerShutdownV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	idempotency := strings.TrimSpace(state.Meta.IdempotencyKey)
	reason := strings.TrimSpace(state.Meta.Reason)
	requestedBy := strings.TrimSpace(state.Meta.RequestedBy)
	if strings.HasPrefix(idempotency, "idem-orquesta-server-stop") ||
		strings.HasPrefix(idempotency, "idem-run-control-complete-idem-orquesta-server-stop") {
		return true
	}
	return requestedBy == "orquesta-director" &&
		(strings.Contains(reason, "apagado controlado solicitado por CLI") ||
			strings.Contains(strings.ToLower(reason), "shutdown"))
}

func normalizeGlobalDrainLimitsV0(
	limits orquestaruncoordinator.RunDrainLimitsV0,
) orquestaruncoordinator.RunDrainLimitsV0 {
	if limits.MaxBursts <= 0 {
		limits.MaxBursts = 4
	}
	if limits.MaxStepsPerBurst <= 0 {
		limits.MaxStepsPerBurst = 6
	}
	if limits.MaxDispatchesPerWait <= 0 {
		limits.MaxDispatchesPerWait = 4
	}
	if limits.MaxCommands <= 0 {
		limits.MaxCommands = 20
	}
	if limits.MaxOutboxPerCycle <= 0 {
		limits.MaxOutboxPerCycle = 4
	}
	if limits.MaxDecisionCycles <= 0 {
		limits.MaxDecisionCycles = 1
	}
	if limits.MaxExternalWaits <= 0 {
		limits.MaxExternalWaits = defaultDrainRunMaxExternalWaitsV0
	}
	return limits
}

func stackDrainOutcomeV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) string {
	if result.Status != "" {
		return string(result.Status)
	}
	return string(result.Final.Status)
}

func stackDrainEvidenceRefsV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) []string {
	refs := append([]string{"evidence-ref-run-coordinator-drain"}, result.Final.FirstPendingRefs...)
	for _, wait := range result.ExternalWaits {
		refs = append(refs, wait.EvidenceRefs...)
	}
	return compactCodexStackStringsV0(refs)
}

func stackDrainRunHasAllTasksDeliveredOrClosedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	tasks := compactCodexStackStringsV0(run.Tasks)
	if len(tasks) == 0 {
		return false
	}
	for _, taskRef := range tasks {
		if !codexStackStringInSetV0(run.DeliveredTasks, taskRef) &&
			!codexStackStringInSetV0(run.ClosedTasks, taskRef) {
			return false
		}
	}
	return true
}

func formatStackCoordinatorTimeV0(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
