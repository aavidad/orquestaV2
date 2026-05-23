package orquestaappcodexstack

import (
	"context"
	"strings"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func (stack StackV0) RunGlobalTickV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) (orquestaruncoordinator.RunCoordinatorTickResultV0, error) {
	command = stack.normalizeRunCoordinatorCommandV0(command)
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
	result, err := drainer.stack.DrainRunV0(ctx, stackDrainRequestFromCoordinatorV0(request))
	if err != nil {
		return orquestaruncoordinator.RunDrainResultV0{}, err
	}
	queueStatus, err := drainer.stack.stackDrainQueueStatusForCoordinatorV0(ctx, result)
	if err != nil {
		return orquestaruncoordinator.RunDrainResultV0{}, err
	}
	return orquestaruncoordinator.RunDrainResultV0{
		RunRef:       strings.TrimSpace(request.RunRef),
		AppRef:       strings.TrimSpace(request.AppRef),
		Outcome:      stackDrainOutcomeV0(result),
		QueueStatus:  queueStatus,
		EvidenceRefs: stackDrainEvidenceRefsV0(result),
	}, nil
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
