package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func (stack StackV0) recoverQueuedStoppedAgentProgressV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	waitAgentRefs := stackDrainPendingStartedAgentRefsV0(run)
	if len(waitAgentRefs) == 0 {
		return stack.recoverQueuedTerminalAgentAssessmentV0(ctx, command, run)
	}
	_, _, err := stack.reconcileStoppedSnapshotsV0(
		ctx,
		DrainRunRequestV0{
			RunRef:            run.RunID,
			OccurredAt:        formatStackCoordinatorTimeV0(command.OccurredAt),
			CorrelationID:     command.CorrelationID,
			MaxBursts:         command.DrainLimits.MaxBursts,
			MaxStepsPerBurst:  command.DrainLimits.MaxStepsPerBurst,
			MaxCommands:       command.DrainLimits.MaxCommands,
			MaxOutboxPerCycle: command.DrainLimits.MaxOutboxPerCycle,
			WaitAgentRefs:     waitAgentRefs,
		},
		run,
	)
	return err
}

func (stack StackV0) recoverQueuedTerminalAgentAssessmentV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	if stack.Ports.AssessmentReplanSource == nil || !stackRunHasRecoverableTerminalAssessmentV0(run) {
		return nil
	}
	ports := stack.directorPortsWithClosureSourceV0(stack.Ports)
	ports.DeliverySource = nil
	ports.ProgressSource = nil
	ports.LeaseSource = nil
	ports.DirectorDecisionSource = nil
	ports.ReviewGateSource = nil
	ports.ReviewReworkReplanSource = nil
	_, err := stack.continueDrainRunControlAfterExternalWithPortsV0(
		ctx,
		DrainRunRequestV0{
			RunRef:               run.RunID,
			OccurredAt:           formatStackCoordinatorTimeV0(command.OccurredAt),
			CorrelationID:        command.CorrelationID,
			MaxBursts:            command.DrainLimits.MaxBursts,
			MaxStepsPerBurst:     command.DrainLimits.MaxStepsPerBurst,
			MaxCommands:          command.DrainLimits.MaxCommands,
			MaxOutboxPerCycle:    command.DrainLimits.MaxOutboxPerCycle,
			MaxDispatchesPerWait: command.DrainLimits.MaxDispatchesPerWait,
			MaxDecisionCycles:    command.DrainLimits.MaxDecisionCycles,
		},
		ports,
	)
	return err
}

func stackRunHasRecoverableTerminalAssessmentV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		len(stackOpenTaskRefsV0(run)) == 0 {
		return false
	}
	terminal := stackTerminalAgentRefSetV0(run)
	for _, raw := range run.AgentAssessments {
		projection, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(raw)
		if !ok || !stackAssessmentActionCanTriggerRecoveryV0(projection.Action) {
			continue
		}
		if terminal[strings.TrimSpace(projection.AgentRequestID)] {
			return true
		}
	}
	return false
}

func stackOpenTaskRefsV0(run orquestacoreworkflow.OrchestrationRunV0) []string {
	closed := map[string]bool{}
	for _, taskRef := range compactStringsV0(run.ClosedTasks) {
		closed[taskRef] = true
	}
	open := make([]string, 0, len(run.Tasks))
	for _, taskRef := range compactStringsV0(run.Tasks) {
		if closed[taskRef] {
			continue
		}
		open = append(open, taskRef)
	}
	return open
}

func stackTerminalAgentRefSetV0(run orquestacoreworkflow.OrchestrationRunV0) map[string]bool {
	terminal := map[string]bool{}
	for _, values := range [][]string{
		run.DeliveredAgents,
		run.FailedAgents,
		run.LostAgents,
		run.StoppedAgents,
		run.ConfirmedStoppedAgents,
	} {
		for _, agentRef := range compactStringsV0(values) {
			terminal[agentRef] = true
		}
	}
	return terminal
}

func stackAssessmentActionCanTriggerRecoveryV0(action string) bool {
	switch strings.TrimSpace(action) {
	case orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
		orquestacoreworkflow.AgentAssessmentActionStopAgentV0:
		return true
	default:
		return false
	}
}
