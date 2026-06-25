package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func (stack StackV0) RunGlobalTickV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) (orquestaruncoordinator.RunCoordinatorTickResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command = stack.normalizeRunCoordinatorCommandV0(command)
	if err := ctx.Err(); err != nil {
		return orquestaruncoordinator.RunCoordinatorTickResultV0{}, err
	}
	if _, err := stack.prepareRunCoordinatorTickV0(ctx, command); err != nil {
		return orquestaruncoordinator.RunCoordinatorTickResultV0{}, err
	}
	return stack.coordinateRunsTickV0(
		ctx,
		command,
		stackRunDrainerV0{stack: stack},
	)
}

type runCoordinatorPreparationResultV0 struct {
	ProcessRuntimeMissing bool
}

func (stack StackV0) prepareRunCoordinatorTickV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) (runCoordinatorPreparationResultV0, error) {
	if err := ctx.Err(); err != nil {
		return runCoordinatorPreparationResultV0{}, err
	}
	if err := stack.reconcileQueuedOrphanExecutableRunsV0(ctx, command); err != nil {
		return runCoordinatorPreparationResultV0{}, err
	}
	if err := stack.recoverQueuedStoppedActiveRunsV0(ctx, command); err != nil {
		if codexStackProcessRuntimeMissingV0(err) {
			if err := ctx.Err(); err != nil {
				return runCoordinatorPreparationResultV0{}, err
			}
			return runCoordinatorPreparationResultV0{ProcessRuntimeMissing: true}, nil
		}
		return runCoordinatorPreparationResultV0{}, err
	}
	if err := ctx.Err(); err != nil {
		return runCoordinatorPreparationResultV0{}, err
	}
	if err := stack.recoverQueuedControlledDomainWorkArtifactsV0(ctx, command); err != nil {
		return runCoordinatorPreparationResultV0{}, err
	}
	if err := ctx.Err(); err != nil {
		return runCoordinatorPreparationResultV0{}, err
	}
	return runCoordinatorPreparationResultV0{}, nil
}

func (stack StackV0) coordinateRunsTickV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	drainer orquestaruncoordinator.RunDrainerPortV0,
) (orquestaruncoordinator.RunCoordinatorTickResultV0, error) {
	return orquestaruncoordinator.CoordinateRunsTickV0(
		ctx,
		orquestaruncoordinator.RunCoordinatorDepsV0{
			QueueReader:   stack.Stores.RunQueue,
			QueueUpdater:  stack.Stores.RunQueue,
			ControlReader: stack.Stores.RunControl,
			Drainer:       drainer,
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
	stack := drainer.stack
	stack.Ports.ExternalWaiter = nil
	drainRequest, err := stack.stackDrainRequestFromCoordinatorV0(ctx, request)
	if err != nil {
		result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{}
		if codexSupervisorRecoverableOperationalPlanStateErrorV0(err) {
			return stackDrainOperationalPlanStateNeedsReplanCoordinatorResultV0(request, result, err), nil
		}
		return stackDrainCoordinatorResultV0(request, result, "error", "", err.Error()), err
	}
	result, err := stack.DrainRunV0(ctx, drainRequest)
	if err != nil {
		if codexSupervisorRecoverableOperationalPlanStateErrorV0(err) {
			return stackDrainOperationalPlanStateNeedsReplanCoordinatorResultV0(request, result, err), nil
		}
		return stackDrainCoordinatorResultV0(request, result, "error", "", err.Error()), err
	}
	queueStatus, err := stack.stackDrainQueueStatusForCoordinatorV0(ctx, result)
	if err != nil {
		return stackDrainCoordinatorResultV0(request, result, "error", "", err.Error()), err
	}
	return stackDrainCoordinatorResultV0(request, result, stackDrainOutcomeV0(result), queueStatus, ""), nil
}

func stackDrainOperationalPlanStateNeedsReplanCoordinatorResultV0(
	request orquestaruncoordinator.RunDrainRequestV0,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
	err error,
) orquestaruncoordinator.RunDrainResultV0 {
	diagnostics := stackDrainDiagnosticsV0(result)
	diagnostics = append(
		diagnostics,
		codexSupervisorOperationalPlanStateNeedsReplanDiagnosticV0(request.RunRef, err),
	)
	return orquestaruncoordinator.RunDrainResultV0{
		RunRef:      strings.TrimSpace(request.RunRef),
		AppRef:      strings.TrimSpace(request.AppRef),
		Outcome:     codexSupervisorOperationalPlanStateNeedsReplanOutcomeV0,
		QueueStatus: orquestarunqueue.RunStatusStoppedV0,
		EvidenceRefs: compactStringsV0(append(
			stackDrainEvidenceRefsV0(result),
			"evidence-ref-codex-supervisor-operational-plan-state-active-step-needs-replan",
			"evidence-ref-codex-supervisor-operational-plan-needs-replan",
		)),
		Diagnostics: diagnostics,
	}
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
	if _, err := stack.recoverBlockedDomainWorkOpenReviewRunV0(
		ctx,
		request.RunRef,
		request.CorrelationID,
		request.OccurredAt,
	); err != nil {
		return DrainRunRequestV0{}, err
	}
	if _, err := stack.recoverBlockedAutoprogrammingOpenReviewRunV0(
		ctx,
		request.RunRef,
		request.CorrelationID,
		request.OccurredAt,
	); err != nil {
		return DrainRunRequestV0{}, err
	}
	if _, err := stack.recoverPartialDomainWorkReviewPhaseV0(
		ctx,
		request.RunRef,
		request.CorrelationID,
		request.OccurredAt,
	); err != nil {
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
