package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

type continueOperationalDirectorMaterializedV0 struct {
	Tasks     []orquestacoreworkflow.WorkflowTaskV0
	WaveRef   string
	CohortRef string
}

func materializeContinueOperationalDirectorPlanV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (continueOperationalDirectorMaterializedV0, error) {
	plan := request.OperationalDirectorPlan
	if !continueHasOperationalDirectorPlanV0(plan) {
		return continueOperationalDirectorMaterializedV0{}, nil
	}
	if strings.TrimSpace(plan.RunRef) != "" && strings.TrimSpace(plan.RunRef) != request.RunRef {
		return continueOperationalDirectorMaterializedV0{}, AppDirectorServiceIssueV0{Field: "operational_director_plan.run_ref"}
	}
	if ports.DirectorTaskStore == nil {
		return continueOperationalDirectorMaterializedV0{}, AppDirectorServiceIssueV0{Field: "ports.director_task_store"}
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return continueOperationalDirectorMaterializedV0{}, err
	}
	refs, err := continueOperationalDirectorFunctionRefsV0(request, run)
	if err != nil {
		return continueOperationalDirectorMaterializedV0{}, err
	}
	result, err := (orquestacionnucleoapp.OperationalDirectorPlanMaterializerV0{
		RunStore:    ports.RunStore,
		EventSink:   ports.EventSink,
		TaskWriter:  ports.DirectorTaskStore,
		RequestedBy: request.RequestedBy,
	}).MaterializeOperationalDirectorPlanV0(ctx, orquestacionnucleoapp.OperationalDirectorPlanMaterializeRequestV0{
		Plan:                 plan,
		FunctionContractRefs: refs,
		TargetPhaseID:        continueOperationalDirectorTargetPhaseIDV0(request, run),
		OccurredAt:           request.OccurredAt,
		CorrelationID:        request.CorrelationID,
		MaxItems:             request.OperationalDirectorMaxItems,
	})
	if err != nil {
		return continueOperationalDirectorMaterializedV0{}, err
	}
	if len(result.Issues) > 0 {
		return continueOperationalDirectorMaterializedV0{}, result.Issues[0]
	}
	if err := saveContinueOperationalDirectorPlanStateV0(ctx, request, plan, result.Tasks, ports); err != nil {
		return continueOperationalDirectorMaterializedV0{}, err
	}
	return continueOperationalDirectorMaterializedV0{
		Tasks:     append([]orquestacoreworkflow.WorkflowTaskV0(nil), result.Tasks...),
		WaveRef:   firstOperationalDirectorTaskWaveRefV0(result.Tasks),
		CohortRef: firstOperationalDirectorTaskCohortRefV0(result.Tasks),
	}, nil
}

func saveContinueOperationalDirectorPlanStateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	ports StartAppDirectorPortsV0,
) error {
	if ports.OperationalPlanStateWriter == nil || len(tasks) == 0 {
		return nil
	}
	state, err := continueOperationalDirectorPlanStateV0(request, plan, tasks)
	if err != nil {
		return err
	}
	return ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, state)
}

func continueOperationalDirectorPlanStateV0(
	request ContinueAppDirectorRequestV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, error) {
	waveRef := firstOperationalDirectorTaskWaveRefV0(tasks)
	cohortRef := firstOperationalDirectorTaskCohortRefV0(tasks)
	parentTaskRef := firstOperationalDirectorTaskParentTaskRefV0(tasks)
	activeStepID := continueOperationalDirectorActivePlanStepIDV0(plan, orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0)
	if activeStepID == "" {
		activeStepID = continueOperationalDirectorActivePlanStepIDV0(plan, orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0)
	}
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "operational-director-plan-state-" + appDirectorSafeRefPartV0(plan.PlanRef),
		PlanRef:             plan.PlanRef,
		RequestRef:          plan.RequestRef,
		RunRef:              plan.RunRef,
		ProjectRef:          plan.ProjectRef,
		Mode:                plan.Mode,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        activeStepID,
		ActiveWaveRef:       waveRef,
		ActiveCohortRef:     cohortRef,
		ActiveParentTaskRef: parentTaskRef,
		Steps:               continueOperationalDirectorPlanStepStatesV0(request, plan, tasks, waveRef, cohortRef, parentTaskRef),
		PendingAgentRefs:    continueOperationalDirectorAgentRefsV0(tasks),
		RequiredTestRefs:    append([]string(nil), plan.RequiredTests...),
		CorrelationID:       request.CorrelationID,
		EvidenceRefs:        []string{"evidence-ref-app-director-operational-plan-state-v0"},
		ObservedAt:          request.OccurredAt,
	}
	return orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
}

func continueOperationalDirectorPlanStepStatesV0(
	request ContinueAppDirectorRequestV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	waveRef string,
	cohortRef string,
	parentTaskRef string,
) []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	taskRefs := continueOperationalDirectorTaskRefsV0(tasks)
	agentRefs := continueOperationalDirectorAgentRefsV0(tasks)
	waitRef := appDirectorWaitRefV0(plan.RunRef, appDirectorWaitFilterV0{
		WaveRef:       waveRef,
		CohortRef:     cohortRef,
		ParentTaskRef: parentTaskRef,
	}, request.CorrelationID)
	steps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		state := orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			StepID:        step.StepID,
			Kind:          step.Kind,
			Status:        step.Status,
			WaveRef:       waveRef,
			CohortRef:     cohortRef,
			ParentTaskRef: parentTaskRef,
		}
		switch step.Kind {
		case orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0:
			state.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.TaskRefs = append([]string(nil), taskRefs...)
			state.AgentRefs = append([]string(nil), agentRefs...)
			state.EvidenceRefs = append([]string(nil), step.EvidenceRefs...)
		case orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0:
			state.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.TaskRefs = append([]string(nil), taskRefs...)
			state.WaitRefs = []string{waitRef}
			state.AgentRefs = append([]string(nil), agentRefs...)
			state.PendingAgentRefs = append([]string(nil), agentRefs...)
			state.BlockerRefs = []string{"wait-subagents"}
		}
		steps = append(steps, state)
	}
	return steps
}

func continueOperationalDirectorActivePlanStepIDV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	kind orquestadirectoroperativo.OperationalDirectorStepKindV0,
) string {
	for _, step := range plan.Steps {
		if step.Kind == kind {
			return strings.TrimSpace(step.StepID)
		}
	}
	return ""
}

func continueOperationalDirectorTaskRefsV0(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.TaskID)
	}
	return compactServiceRefsV0(refs)
}

func continueOperationalDirectorAgentRefsV0(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	return compactServiceRefsV0(refs)
}

func continueHasOperationalDirectorPlanV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
) bool {
	return strings.TrimSpace(plan.PlanRef) != "" ||
		strings.TrimSpace(plan.RequestRef) != "" ||
		strings.TrimSpace(plan.RunRef) != "" ||
		len(plan.Steps) > 0
}

func continueOperationalDirectorFunctionRefsV0(
	request ContinueAppDirectorRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]orquestacoreworkflow.WorkflowFunctionContractRefV0, error) {
	if len(request.OperationalDirectorFunctionContractRefs) > 0 {
		refs := append([]orquestacoreworkflow.WorkflowFunctionContractRefV0(nil), request.OperationalDirectorFunctionContractRefs...)
		if !continueOperationalDirectorFunctionRefsPublishedV0(refs, run.FunctionContracts) {
			return nil, AppDirectorServiceIssueV0{Field: "operational_director_function_contract_refs"}
		}
		return refs, nil
	}
	refs := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(run.FunctionContracts))
	for _, contractRef := range compactServiceRefsV0(run.FunctionContracts) {
		refs = append(refs, orquestacoreworkflow.WorkflowFunctionContractRefV0{ContractRef: contractRef})
	}
	return refs, nil
}

func continueOperationalDirectorTargetPhaseIDV0(
	request ContinueAppDirectorRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestacoreworkflow.OrchestrationPhaseIDV0 {
	if strings.TrimSpace(string(request.OperationalDirectorTargetPhaseID)) != "" {
		return request.OperationalDirectorTargetPhaseID
	}
	return run.CurrentPhase
}

func continueOperationalDirectorFunctionRefsPublishedV0(
	refs []orquestacoreworkflow.WorkflowFunctionContractRefV0,
	published []string,
) bool {
	available := map[string]bool{}
	for _, ref := range compactServiceRefsV0(published) {
		available[ref] = true
	}
	for _, ref := range refs {
		contractRef := strings.TrimSpace(ref.ContractRef)
		if contractRef == "" || !available[contractRef] {
			return false
		}
	}
	return len(refs) > 0
}
