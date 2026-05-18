package orquestadirectoragentworkflow

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func ApplyDirectorAgentDecisionV0(
	ctx context.Context,
	request ApplyDirectorAgentDecisionRequestV0,
	ports ApplyDirectorAgentDecisionPortsV0,
) (ApplyDirectorAgentDecisionResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if request.Decision.CommandType == orquestadirectoragent.DirectorAgentCommandProposePlanTeamV0 {
		return applyDirectorAgentPlanTeamDecisionV0(ctx, request, ports)
	}
	if issues := validateApplyDirectorAgentDecisionPortsV0(ports); len(issues) > 0 {
		return ApplyDirectorAgentDecisionResultV0{Issues: issues}, nil
	}
	command, issues := BuildDirectorAgentWorkflowCommandV0(DirectorAgentWorkflowCommandRequestV0{
		Decision:      request.Decision,
		OccurredAt:    request.OccurredAt,
		CorrelationID: request.CorrelationID,
		RequestedBy:   request.RequestedBy,
	})
	if len(issues) > 0 {
		return ApplyDirectorAgentDecisionResultV0{Issues: issues}, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, command.RunID)
	if err != nil {
		return ApplyDirectorAgentDecisionResultV0{Command: command}, err
	}
	commandResult, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		return ApplyDirectorAgentDecisionResultV0{Command: command, Run: run}, err
	}
	next, err := applyDirectorAgentWorkflowEventsV0(run, commandResult.Events)
	if err != nil {
		return ApplyDirectorAgentDecisionResultV0{Command: command, Run: run}, err
	}
	if issues, err := materializeDirectorAgentDecisionV0(ctx, request, ports); err != nil {
		return ApplyDirectorAgentDecisionResultV0{Command: command, Run: run}, err
	} else if len(issues) > 0 {
		return ApplyDirectorAgentDecisionResultV0{Command: command, Run: run, Issues: issues}, nil
	}
	if len(commandResult.Events) > 0 {
		if err := ports.EventSink.AppendRunEventsV0(ctx, command.RunID, commandResult.Events); err != nil {
			return ApplyDirectorAgentDecisionResultV0{Command: command, Run: run}, err
		}
	}
	if err := ports.RunStore.SaveRunV0(ctx, next); err != nil {
		return ApplyDirectorAgentDecisionResultV0{Command: command, Run: run}, err
	}
	return ApplyDirectorAgentDecisionResultV0{
		Command:     command,
		Run:         next,
		EventsCount: len(commandResult.Events),
		Idempotent:  commandResult.Idempotent,
	}, nil
}

func validateApplyDirectorAgentDecisionPortsV0(
	ports ApplyDirectorAgentDecisionPortsV0,
) []DirectorAgentWorkflowIssueV0 {
	issues := make([]DirectorAgentWorkflowIssueV0, 0)
	if ports.RunStore == nil {
		issues = append(issues, directorAgentWorkflowIssueV0("director_agent_workflow_required", "ports.run_store"))
	}
	if ports.EventSink == nil {
		issues = append(issues, directorAgentWorkflowIssueV0("director_agent_workflow_required", "ports.event_sink"))
	}
	return issues
}

func materializeDirectorAgentDecisionV0(
	ctx context.Context,
	request ApplyDirectorAgentDecisionRequestV0,
	ports ApplyDirectorAgentDecisionPortsV0,
) ([]DirectorAgentWorkflowIssueV0, error) {
	if request.Decision.CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 {
		return nil, nil
	}
	if ports.TaskStore == nil {
		return []DirectorAgentWorkflowIssueV0{
			directorAgentWorkflowIssueV0("director_agent_workflow_required", "ports.task_store"),
		}, nil
	}
	if request.Decision.CreateMicrotask == nil {
		return []DirectorAgentWorkflowIssueV0{
			directorAgentWorkflowIssueV0("director_agent_workflow_required", "decision.create_microtask"),
		}, nil
	}
	task, err := orquestacoreworkflow.NewWorkflowTaskV0(
		directorAgentWorkflowTaskV0(request.Decision.CreateMicrotask.Task),
	)
	if err != nil {
		return []DirectorAgentWorkflowIssueV0{
			directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "decision.create_microtask.task"),
		}, nil
	}
	return nil, ports.TaskStore.SaveWorkflowTaskV0(ctx, task)
}

func applyDirectorAgentWorkflowEventsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	next := run
	var err error
	for _, event := range events {
		next, err = orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			return run, err
		}
	}
	return next, nil
}
