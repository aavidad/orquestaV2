package orquestaappcodexstack

import (
	"context"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func (stack StackV0) RunGlobalSupervisorV0(
	ctx context.Context,
	command orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	command = stack.normalizeRunSupervisorCommandV0(command)
	return orquestarunsupervisor.SuperviseRunsV0(
		ctx,
		orquestarunsupervisor.RunSupervisorDepsV0{Ticker: stack},
		command,
	)
}

func (stack StackV0) normalizeRunSupervisorCommandV0(
	command orquestarunsupervisor.RunSupervisorCommandV0,
) orquestarunsupervisor.RunSupervisorCommandV0 {
	queue := normalizeRunQueueConfigV0(stack.RunQueue)
	supervisor := normalizeRunSupervisorConfigV0(stack.RunSupervisor)
	command.QueueRef = firstNonEmptyQueuedSourceV0(command.QueueRef, queue.QueueRef)
	if command.QueueLimit <= 0 {
		command.QueueLimit = queue.QueueLimit
	}
	if command.MaxRunsPerTick <= 0 {
		command.MaxRunsPerTick = queue.MaxRunsPerTick
	}
	if command.MaxTicks <= 0 {
		command.MaxTicks = supervisor.MaxTicks
	}
	if command.MaxExecutions <= 0 {
		command.MaxExecutions = supervisor.MaxExecutions
	}
	if !command.StopOnNoExecution {
		command.StopOnNoExecution = supervisor.StopOnNoExecution
	}
	if !command.AllowRepeatedRuns {
		command.AllowRepeatedRuns = supervisor.AllowRepeatedRuns
	}
	if command.OccurredAt.IsZero() {
		command.OccurredAt = stackNowV0(stack.Clock)
	}
	return command
}
