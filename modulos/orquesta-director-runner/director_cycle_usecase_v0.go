package orquestadirectorrunner

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func RunDirectorCycleV0(
	ctx context.Context,
	input DirectorCycleInputV0,
) (DirectorCycleResultV0, error) {
	input = normalizeDirectorCycleInputV0(input)
	result := newDirectorCycleResultV0(input)
	if err := validateDirectorCycleInputV0(ctx, input); err != nil {
		return resultWithDirectorCycleErrorV0(result, err)
	}
	if err := ctx.Err(); err != nil {
		return resultWithDirectorCycleIssueV0(result, input, ErrDirectorRunnerCycleInvalidoV0, "context", "context cancelado", true)
	}
	plan, err := input.Scheduler.BuildDirectorSchedulerTickV0(ctx, input.SchedulerInput)
	if err != nil {
		return resultWithDirectorCycleSchedulerErrorV0(result, input, err)
	}
	copySchedulerPlanToResultV0(&result, plan)
	if err := validateDirectorCyclePlanV0(input, plan); err != nil {
		return resultWithDirectorCycleErrorV0(result, err)
	}
	if len(plan.Commands) == 0 {
		return finishDirectorCycleWithoutCommandsV0(result, plan), nil
	}
	return applyDirectorCycleCommandsV0(ctx, input, result, plan)
}

func applyDirectorCycleCommandsV0(
	ctx context.Context,
	input DirectorCycleInputV0,
	result DirectorCycleResultV0,
	plan orquestadirectorscheduler.DirectorSchedulerTickPlanV0,
) (DirectorCycleResultV0, error) {
	for _, command := range plan.Commands {
		if err := ctx.Err(); err != nil {
			return resultWithDirectorCycleIssueV0(result, input, ErrDirectorRunnerCycleInvalidoV0, "context", "context cancelado", true)
		}
		commandResult, err := input.Workflow.HandleWorkflowCommandV0(ctx, command)
		if err != nil {
			return resultWithDirectorCycleWorkflowErrorV0(result, input, command, err)
		}
		result.AppliedCommands = append(result.AppliedCommands, appliedCommandFromResultV0(command, commandResult))
		result.EventsCount += len(commandResult.Events)
		if len(commandResult.Outbox) > 0 {
			if err := validateDirectorCycleOutboxV0(input, commandResult.Outbox); err != nil {
				return resultWithDirectorCycleErrorV0(result, err)
			}
			result.Outbox = append(result.Outbox, cloneDirectorCycleOutboxV0(commandResult.Outbox)...)
			if len(result.Outbox) >= input.MaxOutbox {
				return finishDirectorCycleWithOutboxV0(result), nil
			}
		}
	}
	if len(result.Outbox) > 0 {
		return finishDirectorCycleWithOutboxV0(result), nil
	}
	return finishDirectorCycleAfterCommandsV0(result, plan), nil
}

func finishDirectorCycleWithOutboxV0(
	result DirectorCycleResultV0,
) DirectorCycleResultV0 {
	result.Status = DirectorCycleStatusOutboxPendingV0
	result.StopReason = DirectorCycleStopOutboxGeneratedV0
	return result
}

func finishDirectorCycleWithoutCommandsV0(
	result DirectorCycleResultV0,
	plan orquestadirectorscheduler.DirectorSchedulerTickPlanV0,
) DirectorCycleResultV0 {
	switch plan.Status {
	case orquestadirectorscheduler.SchedulerTickStatusWaitingV0:
		result.Status = DirectorCycleStatusWaitingV0
		result.StopReason = DirectorCycleStopSchedulerWaitingV0
	case orquestadirectorscheduler.SchedulerTickStatusBlockedV0:
		result.Status = DirectorCycleStatusBlockedV0
		result.StopReason = DirectorCycleStopSchedulerBlockedV0
	case orquestadirectorscheduler.SchedulerTickStatusNeedsDirectorV0:
		result.Status = DirectorCycleStatusNeedsDirectorV0
		result.StopReason = DirectorCycleStopNeedsDirectorV0
	default:
		result.Status = DirectorCycleStatusQuiescentV0
		result.StopReason = DirectorCycleStopQuiescentV0
	}
	return result
}

func finishDirectorCycleAfterCommandsV0(
	result DirectorCycleResultV0,
	plan orquestadirectorscheduler.DirectorSchedulerTickPlanV0,
) DirectorCycleResultV0 {
	switch plan.Status {
	case orquestadirectorscheduler.SchedulerTickStatusBlockedV0:
		result.Status = DirectorCycleStatusBlockedV0
		result.StopReason = DirectorCycleStopSchedulerBlockedV0
	case orquestadirectorscheduler.SchedulerTickStatusNeedsDirectorV0:
		result.Status = DirectorCycleStatusNeedsDirectorV0
		result.StopReason = DirectorCycleStopNeedsDirectorV0
	default:
		result.Status = DirectorCycleStatusCommandsAppliedV0
		result.StopReason = DirectorCycleStopCommandsExhaustedV0
	}
	return result
}

func validateDirectorCycleOutboxV0(
	input DirectorCycleInputV0,
	messages []orquestacoreworkflow.OutboxMessageV0,
) error {
	for _, message := range messages {
		if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
			return directorCycleErrorV0(input, ErrDirectorRunnerWorkflowV0, "outbox invalido", "workflow.outbox", true, nil)
		}
	}
	return nil
}
