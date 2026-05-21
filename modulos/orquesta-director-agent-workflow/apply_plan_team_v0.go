package orquestadirectoragentworkflow

import (
	"context"
	"reflect"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func applyDirectorAgentPlanTeamDecisionV0(
	ctx context.Context,
	request ApplyDirectorAgentDecisionRequestV0,
	ports ApplyDirectorAgentDecisionPortsV0,
) (ApplyDirectorAgentDecisionResultV0, error) {
	if issues := validateApplyDirectorAgentDecisionPortsV0(ports); len(issues) > 0 {
		return ApplyDirectorAgentDecisionResultV0{Issues: issues}, nil
	}
	if ports.TaskStore == nil {
		return ApplyDirectorAgentDecisionResultV0{
			Issues: []DirectorAgentWorkflowIssueV0{
				directorAgentWorkflowIssueV0("director_agent_workflow_required", "ports.task_store"),
			},
		}, nil
	}
	commandRequest := normalizeDirectorAgentWorkflowRequestV0(DirectorAgentWorkflowCommandRequestV0{
		Decision:      request.Decision,
		OccurredAt:    request.OccurredAt,
		CorrelationID: request.CorrelationID,
		RequestedBy:   request.RequestedBy,
	})
	if issues := validateDirectorAgentWorkflowRequestV0(commandRequest); len(issues) > 0 {
		return ApplyDirectorAgentDecisionResultV0{Issues: issues}, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, commandRequest.Decision.RunID)
	if err != nil {
		return ApplyDirectorAgentDecisionResultV0{}, err
	}
	tasks, issues := directorAgentPlanTeamWorkflowTasksV0(commandRequest.Decision)
	if len(issues) > 0 {
		return ApplyDirectorAgentDecisionResultV0{Run: run, Issues: issues}, nil
	}
	next := run
	commands := make([]orquestacoreworkflow.OrchestrationCommandV0, 0, len(tasks))
	events := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(tasks))
	for _, task := range tasks {
		command, issues := directorAgentPlanTeamCreateMicrotaskCommandV0(commandRequest, task)
		if len(issues) > 0 {
			return ApplyDirectorAgentDecisionResultV0{
				Run:      run,
				Commands: commands,
				Issues:   issues,
			}, nil
		}
		commandResult, err := orquestacoreworkflow.HandleCommandV0(next, command)
		if err != nil {
			return ApplyDirectorAgentDecisionResultV0{
				Command:  command,
				Commands: append(commands, command),
				Run:      next,
			}, err
		}
		next, err = applyDirectorAgentWorkflowEventsV0(next, commandResult.Events)
		if err != nil {
			return ApplyDirectorAgentDecisionResultV0{
				Command:  command,
				Commands: append(commands, command),
				Run:      next,
			}, err
		}
		commands = append(commands, command)
		events = append(events, commandResult.Events...)
	}
	for _, task := range tasks {
		if err := ports.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
			return ApplyDirectorAgentDecisionResultV0{
				Command:  firstDirectorAgentWorkflowCommandV0(commands),
				Commands: commands,
				Run:      run,
			}, err
		}
	}
	if len(events) > 0 {
		if err := ports.EventSink.AppendRunEventsV0(ctx, commandRequest.Decision.RunID, events); err != nil {
			return ApplyDirectorAgentDecisionResultV0{
				Command:  firstDirectorAgentWorkflowCommandV0(commands),
				Commands: commands,
				Run:      run,
			}, err
		}
	}
	if err := ports.RunStore.SaveRunV0(ctx, next); err != nil {
		return ApplyDirectorAgentDecisionResultV0{
			Command:  firstDirectorAgentWorkflowCommandV0(commands),
			Commands: commands,
			Run:      run,
		}, err
	}
	return ApplyDirectorAgentDecisionResultV0{
		Command:     firstDirectorAgentWorkflowCommandV0(commands),
		Commands:    commands,
		Run:         next,
		EventsCount: len(events),
		Idempotent:  len(events) == 0,
	}, nil
}

func directorAgentPlanTeamWorkflowTasksV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, []DirectorAgentWorkflowIssueV0) {
	if decision.ProposePlanTeam == nil {
		return nil, []DirectorAgentWorkflowIssueV0{
			directorAgentWorkflowIssueV0("director_agent_workflow_required", "decision.propose_autonomous_plan_team"),
		}
	}
	plan := decision.ProposePlanTeam.Plan
	tasks := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(plan.WorkUnits))
	for _, unit := range plan.WorkUnits {
		task, err := orquestacoreworkflow.NewWorkflowTaskV0(orquestacoreworkflow.WorkflowTaskV0{
			SchemaVersion:        orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
			TaskID:               unit.WorkUnitRef,
			RunID:                plan.RunID,
			PhaseID:              orquestacoreworkflow.OrchestrationPhaseIDV0(unit.PhaseID),
			Title:                unit.Title,
			Summary:              unit.Summary,
			WriteSet:             unit.WriteSet,
			AcceptanceCriteria:   directorAgentPlanTeamAcceptanceCriteriaV0(unit),
			RequiredTests:        directorAgentAutonomousWorkUnitRequiredTestsV0(unit),
			DependsOn:            unit.DependsOn,
			FunctionContractRefs: directorAgentWorkflowFunctionRefsV0(unit.FunctionContractRefs),
		})
		if err != nil {
			return nil, []DirectorAgentWorkflowIssueV0{
				directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "decision.propose_autonomous_plan_team.plan.work_units"),
			}
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func directorAgentPlanTeamAcceptanceCriteriaV0(
	unit orquestadirectoragent.DirectorAgentAutonomousWorkUnitV0,
) []string {
	return directorAgentWorkflowOperationalAcceptanceCriteriaV0(unit.PhaseID, unit.AcceptanceCriteria)
}

func directorAgentPlanTeamCreateMicrotaskCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	command, err := orquestacoreworkflow.NewCreateMicrotaskCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      directorAgentPlanTeamCommandRefV0(request.Decision.CommandRef, task.TaskID),
			RunID:          request.Decision.RunID,
			IdempotencyKey: "idem-" + directorAgentPlanTeamCommandRefV0(request.Decision.CommandRef, task.TaskID),
			CorrelationID:  request.CorrelationID,
			RequestedBy:    request.RequestedBy,
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{Task: task},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{}, []DirectorAgentWorkflowIssueV0{
			directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command"),
		}
	}
	return command, nil
}

func directorAgentAutonomousWorkUnitRequiredTestsV0(unit any) []string {
	value := reflect.ValueOf(unit)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}
	field := value.FieldByName("RequiredTests")
	if !field.IsValid() || field.Kind() != reflect.Slice || field.Type().Elem().Kind() != reflect.String {
		return nil
	}
	tests := make([]string, 0, field.Len())
	for i := 0; i < field.Len(); i++ {
		tests = append(tests, field.Index(i).String())
	}
	return tests
}

func directorAgentPlanTeamCommandRefV0(commandRef string, taskRef string) string {
	return strings.TrimSpace(commandRef) + "-" + directorAgentPlanTeamSafeRefPartV0(taskRef)
}

func directorAgentPlanTeamSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func firstDirectorAgentWorkflowCommandV0(
	commands []orquestacoreworkflow.OrchestrationCommandV0,
) orquestacoreworkflow.OrchestrationCommandV0 {
	if len(commands) == 0 {
		return orquestacoreworkflow.OrchestrationCommandV0{}
	}
	return commands[0]
}
