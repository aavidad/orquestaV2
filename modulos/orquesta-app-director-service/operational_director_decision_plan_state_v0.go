package orquestaappdirectorservice

import (
	"context"
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const operationalDirectorDecisionTaskSourceMarkerV0 = "operational_director.task_source"

func startRequestWithOperationalDirectorPlanStateV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (StartAppDirectorRequestV0, error) {
	next, err := ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(ctx, ContinueAppDirectorRequestV0{
		RunRef:                     request.RunRef,
		OccurredAt:                 request.OccurredAt,
		CorrelationID:              request.CorrelationID,
		RequestedBy:                request.RequestedBy,
		MaxBursts:                  request.MaxBursts,
		MaxStepsPerBurst:           request.MaxStepsPerBurst,
		MaxDispatchesPerWait:       request.MaxDispatchesPerWait,
		WaitAgentRefs:              append([]string(nil), request.WaitAgentRefs...),
		WaitCohortRef:              request.WaitCohortRef,
		WaitWaveRef:                request.WaitWaveRef,
		WaitParentTaskRef:          request.WaitParentTaskRef,
		MaxCommands:                request.MaxCommands,
		MaxOutboxPerCycle:          request.MaxOutboxPerCycle,
		MaxDecisionCycles:          request.MaxDecisionCycles,
		MaxExternalWaits:           request.MaxExternalWaits,
		OperationalDirectorPlanRef: request.OperationalDirectorPlanRef,
	}, ports)
	if err != nil {
		return StartAppDirectorRequestV0{}, err
	}
	if ports.OperationalPlanStateStore != nil {
		next, err = continueRequestWithOperationalDirectorPlanStateV0(ctx, next, ports)
		if err != nil {
			return StartAppDirectorRequestV0{}, err
		}
	}
	request.WaitAgentRefs = append([]string(nil), next.WaitAgentRefs...)
	request.WaitCohortRef = next.WaitCohortRef
	request.WaitWaveRef = next.WaitWaveRef
	request.WaitParentTaskRef = next.WaitParentTaskRef
	request.OperationalDirectorPlanRef = next.OperationalDirectorPlanRef
	return request, nil
}

func ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorRequestV0, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" {
		planRef = defaultOperationalDirectorDecisionPlanRefV0(request.RunRef)
	}
	if planRef == "" {
		return request, nil
	}
	if ports.OperationalPlanStateStore != nil {
		state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
		if err == nil {
			if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
				request.OperationalDirectorPlanRef = state.PlanRef
			}
			return request, nil
		}
		if !operationalDirectorPlanStateMissingV0(err) {
			return ContinueAppDirectorRequestV0{}, err
		}
	}
	if ports.OperationalPlanStateWriter == nil ||
		ports.DirectorTaskStore == nil ||
		ports.RunStore == nil {
		return request, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	tasks, err := ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, request.RunRef, workflowTaskOpenRefsForDirectorDecisionPlanStateV0(run))
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	tasks = directorDecisionOperationalTasksV0(tasks)
	if len(tasks) == 0 {
		return request, nil
	}
	state, err := directorDecisionOperationalPlanStateV0(request, run, planRef, tasks)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	if err := ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, state); err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	request.OperationalDirectorPlanRef = planRef
	return request, nil
}

func directorDecisionOperationalPlanStateV0(
	request ContinueAppDirectorRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	planRef string,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, error) {
	plan := orquestadirectoroperativo.OperationalDirectorPlanV0{
		PlanRef:       planRef,
		RequestRef:    "request-ref-director-decisions-" + appDirectorSafeRefPartV0(request.RunRef),
		RunRef:        request.RunRef,
		ProjectRef:    run.ProjectRef,
		Mode:          orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:        orquestadirectoroperativo.OperationalDirectorPlanReadyV0,
		Objective:     "Continuar decisiones aplicadas por el Director.",
		RequiredTests: directorDecisionRequiredTestsV0(tasks),
		Steps: []orquestadirectoroperativo.OperationalDirectorStepV0{
			{
				StepID: "step-launch-subagents",
				Kind:   orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status: orquestadirectoroperativo.OperationalDirectorStepPendingV0,
			},
			{
				StepID: "step-wait-subagents",
				Kind:   orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status: orquestadirectoroperativo.OperationalDirectorStepPendingV0,
			},
			{
				StepID: "step-review-deliveries",
				Kind:   orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status: orquestadirectoroperativo.OperationalDirectorStepPendingV0,
			},
			{
				StepID: "step-run-required-tests",
				Kind:   orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status: orquestadirectoroperativo.OperationalDirectorStepPendingV0,
			},
			{
				StepID: "step-replan-or-close",
				Kind:   orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status: orquestadirectoroperativo.OperationalDirectorStepPendingV0,
			},
		},
	}
	return continueOperationalDirectorPlanStateV0(request, plan, tasks)
}

func workflowTaskOpenRefsForDirectorDecisionPlanStateV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	closed := compactServiceRefsV0(run.ClosedTasks)
	refs := make([]string, 0, len(run.Tasks))
	for _, taskRef := range compactServiceRefsV0(run.Tasks) {
		if startAppDirectorStringInSetV0(closed, taskRef) {
			continue
		}
		refs = append(refs, taskRef)
	}
	return refs
}

func directorDecisionOperationalTasksV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []orquestacoreworkflow.WorkflowTaskV0 {
	out := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		if task.PhaseID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
			!workflowTaskHasOperationalDirectorDecisionMarkerV0(task) {
			continue
		}
		out = append(out, task)
	}
	return out
}

func workflowTaskHasOperationalDirectorDecisionMarkerV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	for _, criterion := range task.AcceptanceCriteria {
		if strings.Contains(strings.TrimSpace(criterion), operationalDirectorDecisionTaskSourceMarkerV0) {
			return true
		}
	}
	return false
}

func directorDecisionRequiredTestsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.RequiredTests...)
	}
	return compactServiceRefsV0(refs)
}

func defaultOperationalDirectorDecisionPlanRefV0(runRef string) string {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return ""
	}
	return "operational-director-plan-director-decisions-" + appDirectorSafeRefPartV0(runRef)
}

func operationalDirectorPlanStateMissingV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "operational_director_plan_state"
}
