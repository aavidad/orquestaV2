package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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

func continueRequestWithOperationalDirectorWaitV0(
	request ContinueAppDirectorRequestV0,
	materialized continueOperationalDirectorMaterializedV0,
) ContinueAppDirectorRequestV0 {
	if len(materialized.Tasks) == 0 {
		return request
	}
	if strings.TrimSpace(request.WaitWaveRef) == "" && strings.TrimSpace(materialized.WaveRef) != "" {
		request.WaitWaveRef = materialized.WaveRef
	}
	if strings.TrimSpace(request.WaitCohortRef) == "" && strings.TrimSpace(materialized.CohortRef) != "" {
		request.WaitCohortRef = materialized.CohortRef
	}
	return request
}

func continueRequestWithOperationalDirectorPlanStateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorRequestV0, error) {
	if continueRequestHasWaitScopeV0(request) {
		return request, nil
	}
	ensured, err := ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(ctx, request, ports)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	request = ensured
	explicitPlanRef := strings.TrimSpace(request.OperationalDirectorPlanRef)
	if explicitPlanRef == "" && ports.OperationalPlanStateStore == nil {
		return request, nil
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" {
		return request, nil
	}
	if ports.OperationalPlanStateStore == nil {
		return ContinueAppDirectorRequestV0{}, AppDirectorServiceIssueV0{Field: "ports.operational_plan_state_store"}
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	next, applied := continueRequestWithLoadedOperationalDirectorPlanStateV0(request, state)
	if explicitPlanRef != "" && (!applied || !continueRequestHasWaitScopeV0(next)) {
		return ContinueAppDirectorRequestV0{}, AppDirectorServiceIssueV0{Field: "operational_director_plan_state.active_step"}
	}
	return next, nil
}

func continueRequestWithLoadedOperationalDirectorPlanStateV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (ContinueAppDirectorRequestV0, bool) {
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		return request, false
	}
	switch step.Kind {
	case orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0:
		if step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			return request, false
		}
		request = continueRequestWithOperationalDirectorPlanStepScopeV0(request, state, step)
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, step.PendingAgentRefs...))
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, state.PendingAgentRefs...))
		return request, true
	case orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0:
		if step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			return request, false
		}
		if len(step.AgentRefs) == 0 {
			return request, false
		}
		request = continueRequestWithOperationalDirectorPlanStepScopeV0(request, state, step)
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, step.AgentRefs...))
		return request, true
	case orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0:
		if step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			return request, false
		}
		if len(step.AgentRefs) == 0 {
			return request, false
		}
		request = continueRequestWithOperationalDirectorPlanStepScopeV0(request, state, step)
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, step.AgentRefs...))
		return request, true
	case orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0:
		if step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			return request, false
		}
		if len(step.AgentRefs) == 0 {
			return request, false
		}
		request = continueRequestWithOperationalDirectorPlanStepScopeV0(request, state, step)
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, step.AgentRefs...))
		return request, true
	default:
		return request, false
	}
}

func continueRequestWithOperationalDirectorPlanStepScopeV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) ContinueAppDirectorRequestV0 {
	if strings.TrimSpace(request.WaitWaveRef) == "" {
		request.WaitWaveRef = state.ActiveWaveRef
		if request.WaitWaveRef == "" {
			request.WaitWaveRef = step.WaveRef
		}
	}
	if strings.TrimSpace(request.WaitCohortRef) == "" {
		request.WaitCohortRef = state.ActiveCohortRef
		if request.WaitCohortRef == "" {
			request.WaitCohortRef = step.CohortRef
		}
	}
	if strings.TrimSpace(request.WaitParentTaskRef) == "" {
		request.WaitParentTaskRef = state.ActiveParentTaskRef
		if request.WaitParentTaskRef == "" {
			request.WaitParentTaskRef = step.ParentTaskRef
		}
	}
	return request
}

func operationalDirectorPlanStateActiveStepV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, bool) {
	activeStepID := strings.TrimSpace(state.ActiveStepID)
	for _, step := range state.Steps {
		if activeStepID != "" && strings.TrimSpace(step.StepID) != activeStepID {
			continue
		}
		if activeStepID == "" && step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			continue
		}
		return step, true
	}
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}, false
}

func continueRequestHasWaitScopeV0(request ContinueAppDirectorRequestV0) bool {
	return len(request.WaitAgentRefs) > 0 ||
		strings.TrimSpace(request.WaitCohortRef) != "" ||
		strings.TrimSpace(request.WaitWaveRef) != "" ||
		strings.TrimSpace(request.WaitParentTaskRef) != ""
}

func continueOperationalDirectorPlanRefV0(request ContinueAppDirectorRequestV0) string {
	if value := strings.TrimSpace(request.OperationalDirectorPlanRef); value != "" {
		return value
	}
	return strings.TrimSpace(request.OperationalDirectorPlan.PlanRef)
}

func updateOperationalDirectorPlanStateAfterLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) error {
	if ports.OperationalPlanStateStore == nil ||
		ports.OperationalPlanStateWriter == nil ||
		loop.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		return nil
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" {
		return nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return err
	}
	next := state
	changed := false
	afterWait, waitChanged, err := operationalDirectorPlanStateAfterWaitConsumedV0(request, next, loop.Run)
	if err != nil {
		return err
	}
	if waitChanged {
		next = afterWait
		changed = true
	}
	afterReview, reviewChanged, err := operationalDirectorPlanStateAfterReviewAcceptedV0(ctx, request, ports, next, loop)
	if err != nil {
		return err
	}
	if reviewChanged {
		next = afterReview
		changed = true
	}
	afterReviewReplan, reviewReplanChanged, err := operationalDirectorPlanStateAfterReviewReworkReplanV0(ctx, request, ports, next, loop)
	if err != nil {
		return err
	}
	if reviewReplanChanged {
		next = afterReviewReplan
		changed = true
	}
	afterTests, testsChanged, err := operationalDirectorPlanStateAfterRequiredTestsV0(ctx, request, ports, next, loop)
	if err != nil {
		return err
	}
	if testsChanged {
		next = afterTests
		changed = true
	}
	if !changed {
		return nil
	}
	return ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func operationalDirectorPlanStateAfterWaitConsumedV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	pendingAgentRefs := compactServiceRefsV0(append(activeStep.PendingAgentRefs, state.PendingAgentRefs...))
	if len(pendingAgentRefs) == 0 || !allServiceRefsInSetV0(pendingAgentRefs, run.DeliveredAgents) {
		return state, false, nil
	}
	reviewStepID := ""
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		switch {
		case step.StepID == activeStep.StepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			nextStep.PendingAgentRefs = nil
			nextStep.Reason = "wait-subagents-consumed"
		case step.Kind == orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 &&
			(reviewStepID == "" || step.StepID == state.ActiveStepID):
			reviewStepID = step.StepID
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			if len(nextStep.TaskRefs) == 0 {
				nextStep.TaskRefs = append([]string(nil), activeStep.TaskRefs...)
			}
			if len(nextStep.AgentRefs) == 0 {
				nextStep.AgentRefs = append([]string(nil), activeStep.AgentRefs...)
			}
			if nextStep.WaveRef == "" {
				nextStep.WaveRef = activeStep.WaveRef
			}
			if nextStep.CohortRef == "" {
				nextStep.CohortRef = activeStep.CohortRef
			}
			if nextStep.ParentTaskRef == "" {
				nextStep.ParentTaskRef = activeStep.ParentTaskRef
			}
		}
		nextSteps = append(nextSteps, nextStep)
	}
	if reviewStepID == "" {
		return state, false, nil
	}
	state.ActiveStepID = reviewStepID
	state.PendingAgentRefs = nil
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-wait-consumed-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

type operationalDirectorPlanReviewTraceV0 struct {
	Deliveries         map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0
	DeliveryRefs       []string
	ReviewRequests     map[string]orquestacoreworkflow.ReviewRequestedPayloadV0
	ReviewRequestRefs  []string
	ReviewResults      map[string]orquestacoreworkflow.ReviewResultV0
	ReviewResultRefs   []string
	AcceptedReviews    map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0
	AcceptedReviewRefs []string
	ReworkRequests     map[string]orquestacoreworkflow.ReworkRequestedPayloadV0
	ReworkRequestRefs  []string
	ReplanDecisions    map[string]orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
	ReplanDecisionRefs []string
}

type operationalDirectorPlanAcceptedReviewMatchV0 struct {
	TaskRef           string
	AgentRef          string
	DeliveryRef       string
	ReviewRequestID   string
	ReviewResultRef   string
	AcceptedReviewRef string
	EvidenceRefs      []string
}

type operationalDirectorPlanReviewReworkReplanMatchV0 struct {
	TaskRef           string
	AgentRef          string
	DeliveryRef       string
	ReviewRequestID   string
	ReviewResultRef   string
	ReworkRequestRef  string
	ReplanDecisionRef string
	EvidenceRefs      []string
}

func operationalDirectorPlanStateAfterReviewAcceptedV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if loop.PendingOutboxCount > 0 {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	matches, complete, err := operationalDirectorPlanAcceptedReviewMatchesV0(ctx, request, ports, activeStep, loop.Run)
	if err != nil || !complete {
		return state, false, err
	}
	nextStepKind := orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0
	if state.Mode == orquestadirectoroperativo.OperationalDirectorModeProgrammingV0 && len(state.RequiredTestRefs) > 0 {
		nextStepKind = orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0
	}
	nextStepID := operationalDirectorPlanStateStepIDByKindV0(state, nextStepKind)
	if nextStepID == "" && nextStepKind == orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0 {
		nextStepKind = orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0
		nextStepID = operationalDirectorPlanStateStepIDByKindV0(state, nextStepKind)
	}
	if nextStepID == "" {
		return state, false, nil
	}
	deliveryRefs := operationalDirectorPlanReviewMatchDeliveryRefsV0(matches)
	reviewResultRefs := operationalDirectorPlanReviewMatchReviewResultRefsV0(matches)
	acceptedReviewRefs := operationalDirectorPlanReviewMatchAcceptedReviewRefsV0(matches)
	evidenceRefs := operationalDirectorPlanReviewMatchEvidenceRefsV0(matches)
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		switch {
		case step.StepID == activeStep.StepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			nextStep.DeliveryRefs = deliveryRefs
			nextStep.ReviewResultRefs = reviewResultRefs
			nextStep.AcceptedReviewRefs = acceptedReviewRefs
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, evidenceRefs...))
			nextStep.BlockerRefs = nil
			nextStep.Reason = "review-deliveries-accepted"
		case step.StepID == nextStepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			if len(nextStep.TaskRefs) == 0 {
				nextStep.TaskRefs = append([]string(nil), activeStep.TaskRefs...)
			}
			if len(nextStep.AgentRefs) == 0 {
				nextStep.AgentRefs = append([]string(nil), activeStep.AgentRefs...)
			}
			if len(nextStep.DeliveryRefs) == 0 {
				nextStep.DeliveryRefs = append([]string(nil), deliveryRefs...)
			}
			if len(nextStep.ReviewResultRefs) == 0 {
				nextStep.ReviewResultRefs = append([]string(nil), reviewResultRefs...)
			}
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, evidenceRefs...))
			if nextStep.WaveRef == "" {
				nextStep.WaveRef = activeStep.WaveRef
			}
			if nextStep.CohortRef == "" {
				nextStep.CohortRef = activeStep.CohortRef
			}
			if nextStep.ParentTaskRef == "" {
				nextStep.ParentTaskRef = activeStep.ParentTaskRef
			}
			if nextStepKind == orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0 {
				nextStep.BlockerRefs = []string{"required-tests-pending"}
				nextStep.Reason = "required-tests-pending"
			} else {
				nextStep.BlockerRefs = nil
				nextStep.Reason = "review-accepted"
			}
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.ActiveStepID = nextStepID
	state.PendingAgentRefs = nil
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-accepted-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

func operationalDirectorPlanStateAfterReviewReworkReplanV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if loop.PendingOutboxCount > 0 {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	match, ok, err := operationalDirectorPlanReviewReworkReplanMatchForActiveStepV0(ctx, request, ports, activeStep, loop.Run)
	if err != nil || !ok {
		return state, false, err
	}
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == activeStep.StepID {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0
			nextStep.DeliveryRefs = []string{match.DeliveryRef}
			nextStep.ReviewResultRefs = []string{match.ReviewResultRef}
			nextStep.AcceptedReviewRefs = nil
			nextStep.ReworkRequestRefs = []string{match.ReworkRequestRef}
			nextStep.ReplanDecisionRefs = []string{match.ReplanDecisionRef}
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, match.EvidenceRefs...))
			nextStep.BlockerRefs = []string{"review-rework-replan-recorded"}
			nextStep.Reason = "review-rework-replan-recorded"
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.ActiveStepID = activeStep.StepID
	state.PendingAgentRefs = nil
	state.ReplanAttempts++
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

type operationalDirectorPlanRequiredTestEvidenceEvaluationV0 struct {
	PassedRefs []string
	FailedRefs []string
	Complete   bool
}

func operationalDirectorPlanStateAfterRequiredTestsV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if loop.PendingOutboxCount > 0 || ports.RequiredTestEvidenceStore == nil {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	requiredTests := compactServiceRefsV0(state.RequiredTestRefs)
	if len(requiredTests) == 0 {
		return state, false, nil
	}
	matches, complete, err := operationalDirectorPlanAcceptedReviewMatchesV0(ctx, request, ports, activeStep, loop.Run)
	if err != nil || !complete {
		return state, false, err
	}
	evidence, err := operationalDirectorPlanRequiredTestEvidenceForMatchesV0(
		ctx,
		request.RunRef,
		ports.RequiredTestEvidenceStore,
		activeStep,
		matches,
	)
	if err != nil {
		return state, false, err
	}
	testStatus := operationalDirectorPlanEvaluateRequiredTestEvidenceV0(request.RunRef, requiredTests, matches, evidence)
	if len(testStatus.FailedRefs) > 0 {
		return operationalDirectorPlanStateWithRequiredTestsBlockedV0(request, state, activeStep, testStatus.FailedRefs)
	}
	if !testStatus.Complete {
		updatedStep, generated, err := operationalDirectorPlanRunRequiredTestsV0(
			ctx,
			request,
			ports,
			activeStep,
			requiredTests,
			matches,
		)
		if err != nil {
			return state, false, err
		}
		if generated {
			activeStep = updatedStep
			evidence, err = operationalDirectorPlanRequiredTestEvidenceForMatchesV0(
				ctx,
				request.RunRef,
				ports.RequiredTestEvidenceStore,
				activeStep,
				matches,
			)
			if err != nil {
				return state, false, err
			}
			testStatus = operationalDirectorPlanEvaluateRequiredTestEvidenceV0(request.RunRef, requiredTests, matches, evidence)
			if len(testStatus.FailedRefs) > 0 {
				return operationalDirectorPlanStateWithRequiredTestsBlockedV0(request, state, activeStep, testStatus.FailedRefs)
			}
		}
	}
	if !testStatus.Complete {
		return state, false, nil
	}
	replanStepID := operationalDirectorPlanStateStepIDByKindV0(state, orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0)
	if replanStepID == "" {
		return state, false, nil
	}
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		switch step.StepID {
		case activeStep.StepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			nextStep.RequiredTestEvidenceRefs = append([]string(nil), testStatus.PassedRefs...)
			nextStep.BlockerRefs = nil
			nextStep.Reason = "required-tests-passed"
		case replanStepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			if len(nextStep.TaskRefs) == 0 {
				nextStep.TaskRefs = append([]string(nil), activeStep.TaskRefs...)
			}
			if len(nextStep.AgentRefs) == 0 {
				nextStep.AgentRefs = append([]string(nil), activeStep.AgentRefs...)
			}
			if len(nextStep.DeliveryRefs) == 0 {
				nextStep.DeliveryRefs = append([]string(nil), activeStep.DeliveryRefs...)
			}
			if len(nextStep.ReviewResultRefs) == 0 {
				nextStep.ReviewResultRefs = append([]string(nil), activeStep.ReviewResultRefs...)
			}
			if len(nextStep.RequiredTestEvidenceRefs) == 0 {
				nextStep.RequiredTestEvidenceRefs = append([]string(nil), testStatus.PassedRefs...)
			}
			if nextStep.WaveRef == "" {
				nextStep.WaveRef = activeStep.WaveRef
			}
			if nextStep.CohortRef == "" {
				nextStep.CohortRef = activeStep.CohortRef
			}
			if nextStep.ParentTaskRef == "" {
				nextStep.ParentTaskRef = activeStep.ParentTaskRef
			}
			nextStep.BlockerRefs = nil
			nextStep.Reason = "required-tests-passed"
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.ActiveStepID = replanStepID
	state.PendingAgentRefs = nil
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-required-tests-passed-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

func operationalDirectorPlanRunRequiredTestsV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	requiredTests []string,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, bool, error) {
	if ports.RequiredTestRunner == nil {
		return activeStep, false, nil
	}
	evidenceRefs := make([]string, 0, len(requiredTests)*len(matches))
	for _, match := range matches {
		result, err := ports.RequiredTestRunner.RunRequiredTestsV0(ctx, orquestacionnucleoapp.RequiredTestExecutionRequestV0{
			RunRef:            request.RunRef,
			TaskRef:           match.TaskRef,
			TestCommands:      requiredTests,
			DeliveryRef:       match.DeliveryRef,
			ReviewRequestID:   match.ReviewRequestID,
			ReviewResultRef:   match.ReviewResultRef,
			AcceptedReviewRef: match.AcceptedReviewRef,
			OccurredAt:        request.OccurredAt,
			CorrelationID:     request.CorrelationID,
			EvidenceRefs:      match.EvidenceRefs,
		})
		if err != nil {
			return activeStep, false, err
		}
		if len(result.Issues) > 0 {
			return activeStep, false, result.Issues[0]
		}
		evidenceRefs = append(evidenceRefs, result.EvidenceRefs...)
	}
	evidenceRefs = compactServiceRefsV0(evidenceRefs)
	if len(evidenceRefs) == 0 {
		return activeStep, false, nil
	}
	activeStep.RequiredTestEvidenceRefs = compactServiceRefsV0(append(activeStep.RequiredTestEvidenceRefs, evidenceRefs...))
	return activeStep, true, nil
}

func operationalDirectorPlanStateWithRequiredTestsBlockedV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	failedRefs []string,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == activeStep.StepID {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			nextStep.RequiredTestEvidenceRefs = append([]string(nil), failedRefs...)
			nextStep.BlockerRefs = []string{"required-tests-failed"}
			nextStep.Reason = "required-tests-failed"
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-required-tests-failed-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

func operationalDirectorPlanAcceptedReviewMatchesV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]operationalDirectorPlanAcceptedReviewMatchV0, bool, error) {
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return nil, false, nil
	}
	events, err := reader.LoadRunEventsV0(ctx, request.RunRef)
	if err != nil {
		return nil, false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	taskRefs := compactServiceRefsV0(activeStep.TaskRefs)
	if len(taskRefs) == 0 {
		return nil, false, nil
	}
	matches := make([]operationalDirectorPlanAcceptedReviewMatchV0, 0, len(taskRefs))
	for _, taskRef := range taskRefs {
		match, ok := operationalDirectorPlanAcceptedReviewMatchForTaskV0(activeStep, run, trace, taskRef)
		if !ok {
			return nil, false, nil
		}
		matches = append(matches, match)
	}
	return matches, true, nil
}

func operationalDirectorPlanReviewReworkReplanMatchForActiveStepV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (operationalDirectorPlanReviewReworkReplanMatchV0, bool, error) {
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return operationalDirectorPlanReviewReworkReplanMatchV0{}, false, nil
	}
	events, err := reader.LoadRunEventsV0(ctx, request.RunRef)
	if err != nil {
		return operationalDirectorPlanReviewReworkReplanMatchV0{}, false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	for _, taskRef := range compactServiceRefsV0(activeStep.TaskRefs) {
		match, ok := operationalDirectorPlanReviewReworkReplanMatchForTaskV0(activeStep, run, trace, taskRef)
		if ok {
			return match, true, nil
		}
	}
	return operationalDirectorPlanReviewReworkReplanMatchV0{}, false, nil
}

func operationalDirectorPlanReviewReworkReplanMatchForTaskV0(
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	taskRef string,
) (operationalDirectorPlanReviewReworkReplanMatchV0, bool) {
	taskRef = strings.TrimSpace(taskRef)
	for _, deliveryRef := range trace.DeliveryRefs {
		delivery := trace.Deliveries[deliveryRef]
		if strings.TrimSpace(delivery.TaskID) != taskRef {
			continue
		}
		if len(activeStep.AgentRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.AgentRefs, delivery.AgentRef) {
			continue
		}
		if len(activeStep.DeliveryRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.DeliveryRefs, delivery.DeliveryRef) {
			continue
		}
		if !startAppDirectorStringInSetV0(run.Deliveries, delivery.DeliveryRef) ||
			!startAppDirectorStringInSetV0(run.DeliveredTasks, taskRef) {
			continue
		}
		if strings.TrimSpace(delivery.AgentRef) != "" &&
			!startAppDirectorStringInSetV0(run.DeliveredAgents, delivery.AgentRef) {
			continue
		}
		match, ok := operationalDirectorPlanReviewReworkReplanMatchForDeliveryV0(run, trace, delivery, taskRef)
		if !ok {
			continue
		}
		if len(activeStep.ReviewResultRefs) > 0 &&
			!startAppDirectorStringInSetV0(activeStep.ReviewResultRefs, match.ReviewResultRef) {
			continue
		}
		match.AgentRef = strings.TrimSpace(delivery.AgentRef)
		return match, true
	}
	return operationalDirectorPlanReviewReworkReplanMatchV0{}, false
}

func operationalDirectorPlanReviewReworkReplanMatchForDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	taskRef string,
) (operationalDirectorPlanReviewReworkReplanMatchV0, bool) {
	deliveryRef := strings.TrimSpace(delivery.DeliveryRef)
	for _, reviewRequestID := range trace.ReviewRequestRefs {
		reviewRequest := trace.ReviewRequests[reviewRequestID]
		if strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef ||
			!startAppDirectorStringInSetV0(run.Reviews, reviewRequest.ReviewRequestID) {
			continue
		}
		reviewResult, ok := operationalDirectorPlanNegativeReviewResultForRequestV0(run, trace, reviewRequest, deliveryRef)
		if !ok {
			continue
		}
		rework, ok := operationalDirectorPlanReworkForReviewResultV0(run, trace, reviewResult, reviewRequest, deliveryRef)
		if !ok {
			continue
		}
		replan, ok := operationalDirectorPlanReplanForReworkV0(run, trace, rework, taskRef)
		if !ok {
			continue
		}
		return operationalDirectorPlanReviewReworkReplanMatchV0{
			TaskRef:           strings.TrimSpace(taskRef),
			DeliveryRef:       deliveryRef,
			ReviewRequestID:   strings.TrimSpace(reviewRequest.ReviewRequestID),
			ReviewResultRef:   strings.TrimSpace(reviewResult.ReviewResultRef),
			ReworkRequestRef:  strings.TrimSpace(rework.ReworkRequestRef),
			ReplanDecisionRef: strings.TrimSpace(replan.ReplanRef),
			EvidenceRefs: compactServiceRefsV0(append(append(append(
				append([]string(nil), delivery.EvidenceRefs...),
				reviewRequest.EvidenceRefs...),
				reviewResult.EvidenceRefs...),
				append(rework.EvidenceRefs, replan.EvidenceRefs...)...)),
		}, true
	}
	return operationalDirectorPlanReviewReworkReplanMatchV0{}, false
}

func operationalDirectorPlanStateEventReaderV0(
	ports StartAppDirectorPortsV0,
) orquestacionnucleoapp.RunEventReaderPortV0 {
	if ports.EventReader != nil {
		return ports.EventReader
	}
	reader, _ := ports.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	return reader
}

func operationalDirectorPlanAcceptedReviewMatchForTaskV0(
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	taskRef string,
) (operationalDirectorPlanAcceptedReviewMatchV0, bool) {
	taskRef = strings.TrimSpace(taskRef)
	for _, deliveryRef := range trace.DeliveryRefs {
		delivery := trace.Deliveries[deliveryRef]
		if strings.TrimSpace(delivery.TaskID) != taskRef {
			continue
		}
		if len(activeStep.AgentRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.AgentRefs, delivery.AgentRef) {
			continue
		}
		if len(activeStep.DeliveryRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.DeliveryRefs, delivery.DeliveryRef) {
			continue
		}
		if !startAppDirectorStringInSetV0(run.Deliveries, delivery.DeliveryRef) ||
			!startAppDirectorStringInSetV0(run.DeliveredTasks, taskRef) {
			continue
		}
		if strings.TrimSpace(delivery.AgentRef) != "" &&
			!startAppDirectorStringInSetV0(run.DeliveredAgents, delivery.AgentRef) {
			continue
		}
		match, ok := operationalDirectorPlanAcceptedReviewMatchForDeliveryV0(run, trace, delivery)
		if !ok {
			continue
		}
		if len(activeStep.ReviewResultRefs) > 0 &&
			!startAppDirectorStringInSetV0(activeStep.ReviewResultRefs, match.ReviewResultRef) {
			continue
		}
		match.TaskRef = taskRef
		match.AgentRef = strings.TrimSpace(delivery.AgentRef)
		return match, true
	}
	return operationalDirectorPlanAcceptedReviewMatchV0{}, false
}

func operationalDirectorPlanAcceptedReviewMatchForDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
) (operationalDirectorPlanAcceptedReviewMatchV0, bool) {
	deliveryRef := strings.TrimSpace(delivery.DeliveryRef)
	for _, reviewRequestID := range trace.ReviewRequestRefs {
		reviewRequest := trace.ReviewRequests[reviewRequestID]
		if strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef ||
			!startAppDirectorStringInSetV0(run.Reviews, reviewRequest.ReviewRequestID) {
			continue
		}
		reviewResult, ok := operationalDirectorPlanAcceptedReviewResultForRequestV0(run, trace, reviewRequest, deliveryRef)
		if !ok {
			continue
		}
		acceptedReview, ok := operationalDirectorPlanAcceptedReviewForRequestV0(run, trace, reviewRequest, deliveryRef)
		if !ok {
			continue
		}
		return operationalDirectorPlanAcceptedReviewMatchV0{
			DeliveryRef:       deliveryRef,
			ReviewRequestID:   strings.TrimSpace(reviewRequest.ReviewRequestID),
			ReviewResultRef:   strings.TrimSpace(reviewResult.ReviewResultRef),
			AcceptedReviewRef: strings.TrimSpace(acceptedReview.AcceptedReviewRef),
			EvidenceRefs: compactServiceRefsV0(append(append(append(
				append([]string(nil), delivery.EvidenceRefs...),
				reviewRequest.EvidenceRefs...),
				reviewResult.EvidenceRefs...),
				acceptedReview.EvidenceRefs...)),
		}, true
	}
	return operationalDirectorPlanAcceptedReviewMatchV0{}, false
}

func operationalDirectorPlanNegativeReviewResultForRequestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	reviewRequestID := strings.TrimSpace(reviewRequest.ReviewRequestID)
	for _, reviewResultRef := range trace.ReviewResultRefs {
		reviewResult := trace.ReviewResults[reviewResultRef]
		status := orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(reviewResult.Status)))
		if strings.TrimSpace(reviewResult.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(reviewResult.DeliveryRef) == deliveryRef &&
			(status == orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
				status == orquestacoreworkflow.ReviewResultStatusRejectedV0) &&
			operationalDirectorPlanProjectionReflectedV0(reviewResult.ReviewResultRef, run.ReviewResults) {
			return reviewResult, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorPlanReworkForReviewResultV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewResult orquestacoreworkflow.ReviewResultV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReworkRequestedPayloadV0, bool) {
	for _, reworkRequestRef := range trace.ReworkRequestRefs {
		rework := trace.ReworkRequests[reworkRequestRef]
		if strings.TrimSpace(rework.ReviewResultRef) == strings.TrimSpace(reviewResult.ReviewResultRef) &&
			strings.TrimSpace(rework.ReviewRequestID) == strings.TrimSpace(reviewRequest.ReviewRequestID) &&
			strings.TrimSpace(rework.DeliveryRef) == strings.TrimSpace(deliveryRef) &&
			operationalDirectorPlanProjectionReflectedV0(rework.ReworkRequestRef, run.ReworkRequests) {
			return rework, true
		}
	}
	return orquestacoreworkflow.ReworkRequestedPayloadV0{}, false
}

func operationalDirectorPlanReplanForReworkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	rework orquestacoreworkflow.ReworkRequestedPayloadV0,
	taskRef string,
) (orquestacoreworkflow.ReplanDecisionRecordedPayloadV0, bool) {
	for _, replanDecisionRef := range trace.ReplanDecisionRefs {
		replan := trace.ReplanDecisions[replanDecisionRef]
		if strings.TrimSpace(replan.SourceRef) == strings.TrimSpace(rework.ReworkRequestRef) &&
			strings.TrimSpace(replan.TaskRef) == strings.TrimSpace(taskRef) &&
			strings.TrimSpace(replan.RunRef) == strings.TrimSpace(run.RunID) &&
			operationalDirectorPlanProjectionReflectedV0(replan.ReplanRef, run.ReplanDecisions) {
			return replan, true
		}
	}
	return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, false
}

func operationalDirectorPlanAcceptedReviewResultForRequestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	reviewRequestID := strings.TrimSpace(reviewRequest.ReviewRequestID)
	for _, reviewResultRef := range trace.ReviewResultRefs {
		reviewResult := trace.ReviewResults[reviewResultRef]
		if strings.TrimSpace(reviewResult.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(reviewResult.DeliveryRef) == deliveryRef &&
			orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(reviewResult.Status))) == orquestacoreworkflow.ReviewResultStatusAcceptedV0 &&
			operationalDirectorPlanProjectionReflectedV0(reviewResult.ReviewResultRef, run.ReviewResults) {
			return reviewResult, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorPlanAcceptedReviewForRequestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewAcceptedPayloadV0, bool) {
	reviewRequestID := strings.TrimSpace(reviewRequest.ReviewRequestID)
	for _, acceptedReviewRef := range trace.AcceptedReviewRefs {
		acceptedReview := trace.AcceptedReviews[acceptedReviewRef]
		if strings.TrimSpace(acceptedReview.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(acceptedReview.DeliveryRef) == deliveryRef &&
			startAppDirectorStringInSetV0(run.AcceptedReviews, acceptedReview.AcceptedReviewRef) {
			return acceptedReview, true
		}
	}
	return orquestacoreworkflow.ReviewAcceptedPayloadV0{}, false
}

func operationalDirectorPlanRequiredTestEvidenceForMatchesV0(
	ctx context.Context,
	runRef string,
	store orquestacionnucleoapp.RequiredTestEvidenceReaderPortV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) ([]orquestacionnucleoapp.RequiredTestEvidenceV0, error) {
	candidateRefs := append([]string(nil), activeStep.RequiredTestEvidenceRefs...)
	if len(candidateRefs) == 0 {
		for _, match := range matches {
			candidateRefs = append(candidateRefs, match.EvidenceRefs...)
		}
	}
	evidence := make([]orquestacionnucleoapp.RequiredTestEvidenceV0, 0, len(candidateRefs))
	for _, evidenceRef := range compactServiceRefsV0(candidateRefs) {
		items, err := store.LoadRequiredTestEvidenceV0(ctx, runRef, []string{evidenceRef})
		if err != nil {
			if operationalDirectorPlanMissingRequiredTestEvidenceV0(err) {
				continue
			}
			return nil, err
		}
		evidence = append(evidence, items...)
	}
	return evidence, nil
}

func operationalDirectorPlanMissingRequiredTestEvidenceV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "required_test_evidence"
}

func operationalDirectorPlanEvaluateRequiredTestEvidenceV0(
	runRef string,
	requiredTests []string,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
) operationalDirectorPlanRequiredTestEvidenceEvaluationV0 {
	passedRefs := []string(nil)
	failedRefs := []string(nil)
	for _, match := range matches {
		for _, required := range requiredTests {
			passedRef, failedRef := operationalDirectorPlanRequiredTestEvidenceRefV0(runRef, required, match, evidence)
			if passedRef != "" {
				passedRefs = append(passedRefs, passedRef)
				continue
			}
			if failedRef != "" {
				failedRefs = append(failedRefs, failedRef)
			}
		}
	}
	expected := len(compactServiceRefsV0(requiredTests)) * len(matches)
	passedRefs = compactServiceRefsV0(passedRefs)
	return operationalDirectorPlanRequiredTestEvidenceEvaluationV0{
		PassedRefs: passedRefs,
		FailedRefs: compactServiceRefsV0(failedRefs),
		Complete:   expected > 0 && len(passedRefs) == expected,
	}
}

func operationalDirectorPlanRequiredTestEvidenceRefV0(
	runRef string,
	required string,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
) (string, string) {
	passedRef := ""
	failedRef := ""
	for _, item := range evidence {
		if strings.TrimSpace(item.RunRef) != strings.TrimSpace(runRef) ||
			strings.TrimSpace(item.TaskRef) != strings.TrimSpace(match.TaskRef) ||
			strings.TrimSpace(item.TestCommand) != strings.TrimSpace(required) ||
			strings.TrimSpace(item.DeliveryRef) != strings.TrimSpace(match.DeliveryRef) ||
			strings.TrimSpace(item.ReviewRequestID) != strings.TrimSpace(match.ReviewRequestID) ||
			strings.TrimSpace(item.ReviewResultRef) != strings.TrimSpace(match.ReviewResultRef) ||
			strings.TrimSpace(item.AcceptedReviewRef) != strings.TrimSpace(match.AcceptedReviewRef) {
			continue
		}
		switch item.Status {
		case orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0:
			passedRef = strings.TrimSpace(item.EvidenceRef)
		case orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0:
			failedRef = strings.TrimSpace(item.EvidenceRef)
		}
	}
	return passedRef, failedRef
}

func operationalDirectorPlanReviewResultReflectedV0(reviewResultRef string, reviewResults []string) bool {
	return operationalDirectorPlanProjectionReflectedV0(reviewResultRef, reviewResults)
}

func operationalDirectorPlanProjectionReflectedV0(ref string, values []string) bool {
	ref = strings.TrimSpace(ref)
	for _, existing := range values {
		existing = strings.TrimSpace(existing)
		if existing == ref || strings.HasPrefix(existing, ref+"#") {
			return true
		}
	}
	return false
}

func operationalDirectorPlanReviewTraceFromEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) operationalDirectorPlanReviewTraceV0 {
	trace := operationalDirectorPlanReviewTraceV0{
		Deliveries:      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0{},
		ReviewRequests:  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0{},
		ReviewResults:   map[string]orquestacoreworkflow.ReviewResultV0{},
		AcceptedReviews: map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0{},
		ReworkRequests:  map[string]orquestacoreworkflow.ReworkRequestedPayloadV0{},
		ReplanDecisions: map[string]orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{},
	}
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0:
			var payload orquestacoreworkflow.DeliveryRegisteredPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.DeliveryRef)
				if ref != "" && trace.Deliveries[ref].DeliveryRef == "" {
					trace.DeliveryRefs = append(trace.DeliveryRefs, ref)
				}
				trace.Deliveries[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewRequestedV0:
			var payload orquestacoreworkflow.ReviewRequestedPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.ReviewRequestID)
				if ref != "" && trace.ReviewRequests[ref].ReviewRequestID == "" {
					trace.ReviewRequestRefs = append(trace.ReviewRequestRefs, ref)
				}
				trace.ReviewRequests[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0:
			var payload orquestacoreworkflow.ReviewResultV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.ReviewResultRef)
				if ref != "" && trace.ReviewResults[ref].ReviewResultRef == "" {
					trace.ReviewResultRefs = append(trace.ReviewResultRefs, ref)
				}
				trace.ReviewResults[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewAcceptedV0:
			var payload orquestacoreworkflow.ReviewAcceptedPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.AcceptedReviewRef)
				if ref != "" && trace.AcceptedReviews[ref].AcceptedReviewRef == "" {
					trace.AcceptedReviewRefs = append(trace.AcceptedReviewRefs, ref)
				}
				trace.AcceptedReviews[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReworkRequestedV0:
			var payload orquestacoreworkflow.ReworkRequestedPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.ReworkRequestRef)
				if ref != "" && trace.ReworkRequests[ref].ReworkRequestRef == "" {
					trace.ReworkRequestRefs = append(trace.ReworkRequestRefs, ref)
				}
				trace.ReworkRequests[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			var payload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.ReplanRef)
				if ref != "" && trace.ReplanDecisions[ref].ReplanRef == "" {
					trace.ReplanDecisionRefs = append(trace.ReplanDecisionRefs, ref)
				}
				trace.ReplanDecisions[ref] = payload
			}
		}
	}
	return trace
}

func operationalDirectorPlanDecodeEventPayloadV0(
	event orquestacoreworkflow.OrchestrationEventV0,
	out any,
) bool {
	return json.Unmarshal(event.Payload, out) == nil
}

func operationalDirectorPlanStateStepIDByKindV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	kind orquestadirectoroperativo.OperationalDirectorStepKindV0,
) string {
	for _, step := range state.Steps {
		if step.Kind == kind {
			return strings.TrimSpace(step.StepID)
		}
	}
	return ""
}

func operationalDirectorPlanReviewMatchDeliveryRefsV0(
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) []string {
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, match.DeliveryRef)
	}
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanReviewMatchReviewResultRefsV0(
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) []string {
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, match.ReviewResultRef)
	}
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanReviewMatchAcceptedReviewRefsV0(
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) []string {
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, match.AcceptedReviewRef)
	}
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanReviewMatchEvidenceRefsV0(
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) []string {
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, match.EvidenceRefs...)
	}
	return compactServiceRefsV0(refs)
}

func allServiceRefsInSetV0(values []string, available []string) bool {
	for _, value := range compactServiceRefsV0(values) {
		if !startAppDirectorStringInSetV0(available, value) {
			return false
		}
	}
	return len(compactServiceRefsV0(values)) > 0
}

func firstOperationalDirectorTaskWaveRefV0(tasks []orquestacoreworkflow.WorkflowTaskV0) string {
	for _, task := range tasks {
		if value := strings.TrimSpace(task.WaveRef); value != "" {
			return value
		}
	}
	return ""
}

func firstOperationalDirectorTaskCohortRefV0(tasks []orquestacoreworkflow.WorkflowTaskV0) string {
	for _, task := range tasks {
		if value := strings.TrimSpace(task.CohortRef); value != "" {
			return value
		}
	}
	return ""
}

func firstOperationalDirectorTaskParentTaskRefV0(tasks []orquestacoreworkflow.WorkflowTaskV0) string {
	for _, task := range tasks {
		if value := strings.TrimSpace(task.ParentTaskRef); value != "" {
			return value
		}
	}
	return ""
}

func normalizeServiceWorkflowFunctionContractRefsV0(
	refs []orquestacoreworkflow.WorkflowFunctionContractRefV0,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	out := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(refs))
	for _, ref := range refs {
		compact := orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef:  strings.TrimSpace(ref.ContractRef),
			FunctionName: strings.TrimSpace(ref.FunctionName),
		}
		if compact.ContractRef == "" && compact.FunctionName == "" {
			continue
		}
		out = append(out, compact)
	}
	return out
}
