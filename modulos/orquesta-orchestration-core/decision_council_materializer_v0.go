package orquestacionnucleoapp

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

type DecisionCouncilPlanMaterializerV0 struct {
	RunStore    RunStorePortV0
	EventSink   EventSinkPortV0
	TaskWriter  WorkflowTaskWriterPortV0
	RequestedBy string
}

type DecisionCouncilPlanMaterializeRequestV0 struct {
	Plan                 orquestadecisioncouncil.DecisionCouncilPlanV0
	FunctionContractRefs []orquestacoreworkflow.WorkflowFunctionContractRefV0
	OccurredAt           string
	CorrelationID        string
	MaxAssignments       int
}

type DecisionCouncilPlanMaterializeResultV0 struct {
	Rounds      []orquestadecisioncouncil.DecisionCouncilOperationalRoundV0
	Tasks       []orquestacoreworkflow.WorkflowTaskV0
	Commands    []orquestacoreworkflow.OrchestrationCommandV0
	EventsCount int
	Issues      []ErrorV0
}

func (materializer DecisionCouncilPlanMaterializerV0) MaterializeDecisionCouncilPlanV0(
	ctx context.Context,
	request DecisionCouncilPlanMaterializeRequestV0,
) (DecisionCouncilPlanMaterializeResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if issues := materializer.validateDecisionCouncilMaterializerV0(request); len(issues) > 0 {
		return DecisionCouncilPlanMaterializeResultV0{Issues: issues}, nil
	}
	rounds, err := orquestadecisioncouncil.BuildDecisionCouncilOperationalRoundsV0(request.Plan)
	if err != nil {
		return DecisionCouncilPlanMaterializeResultV0{
			Issues: []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "decision_council_plan", err.Error())},
		}, nil
	}
	tasks, err := decisionCouncilWorkflowTasksV0(request, rounds)
	if err != nil {
		return DecisionCouncilPlanMaterializeResultV0{}, err
	}
	if request.MaxAssignments > 0 && len(tasks) > request.MaxAssignments {
		tasks = tasks[:request.MaxAssignments]
	}
	result := DecisionCouncilPlanMaterializeResultV0{
		Rounds: rounds.Rounds,
		Tasks:  make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks)),
	}
	for _, task := range tasks {
		command, err := materializer.decisionCouncilCreateMicrotaskCommandV0(request, task)
		if err != nil {
			result.Issues = append(result.Issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "decision_council.command", err.Error()))
			return result, nil
		}
		if err := materializer.preflightDecisionCouncilCreateMicrotaskCommandV0(ctx, command); err != nil {
			return result, err
		}
		if err := materializer.TaskWriter.SaveWorkflowTaskV0(ctx, task); err != nil {
			return result, err
		}
		applied, err := HandleStoredWorkflowCommandV0(ctx, materializer.RunStore, materializer.EventSink, command)
		if err != nil {
			return result, err
		}
		result.Tasks = append(result.Tasks, task)
		result.Commands = append(result.Commands, command)
		result.EventsCount += len(applied.Events)
	}
	return result, nil
}

func (materializer DecisionCouncilPlanMaterializerV0) validateDecisionCouncilMaterializerV0(
	request DecisionCouncilPlanMaterializeRequestV0,
) []ErrorV0 {
	issues := []ErrorV0{}
	if materializer.RunStore == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido"))
	}
	if materializer.EventSink == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "event_sink", "event_sink requerido"))
	}
	if materializer.TaskWriter == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "task_writer", "task_writer requerido"))
	}
	if len(request.FunctionContractRefs) == 0 {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "function_contract_refs", "function_contract_refs requerido"))
	}
	if request.OccurredAt == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido"))
	}
	return issues
}

func (materializer DecisionCouncilPlanMaterializerV0) preflightDecisionCouncilCreateMicrotaskCommandV0(
	ctx context.Context,
	command orquestacoreworkflow.OrchestrationCommandV0,
) error {
	run, err := materializer.RunStore.LoadRunV0(ctx, command.RunID)
	if err != nil {
		return err
	}
	_, err = orquestacoreworkflow.HandleCommandV0(run, command)
	return err
}

func (materializer DecisionCouncilPlanMaterializerV0) decisionCouncilCreateMicrotaskCommandV0(
	request DecisionCouncilPlanMaterializeRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	suffix := operationalDirectorSafeRefPartV0(task.TaskID)
	return orquestacoreworkflow.NewCreateMicrotaskCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-create-" + suffix,
			RunID:          request.Plan.RunRef,
			IdempotencyKey: "idem-create-" + suffix,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    operationalDirectorRequestedByV0(materializer.RequestedBy),
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{Task: task},
	)
}

func decisionCouncilWorkflowTasksV0(
	request DecisionCouncilPlanMaterializeRequestV0,
	rounds orquestadecisioncouncil.DecisionCouncilOperationalRoundsV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	assignmentTaskRefs := decisionCouncilAssignmentTaskRefsV0(request.Plan)
	critiqueTaskRefs := decisionCouncilTaskRefsForRoleV0(request.Plan, assignmentTaskRefs, orquestadecisioncouncil.CouncilRoleCritiqueV0)
	tasks := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(request.Plan.Assignments))
	roleOrdinals := map[string]int{}
	for _, assignment := range request.Plan.Assignments {
		roleOrdinals[assignment.Role]++
		task, err := decisionCouncilWorkflowTaskV0(request, rounds, assignment, roleOrdinals[assignment.Role], assignmentTaskRefs, critiqueTaskRefs)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
