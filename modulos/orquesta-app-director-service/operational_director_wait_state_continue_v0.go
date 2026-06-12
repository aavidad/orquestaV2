package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func continueRequestWithLoadedOperationalDirectorPlanStateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorRequestV0, bool, error) {
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		return request, false, nil
	}
	switch step.Kind {
	case orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0:
		if step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			return request, false, nil
		}
		recovered, ok, err := continueRequestWithRecoveredWorkflowTaskWaitStateV0(ctx, request, state, step, ports)
		if err != nil || ok {
			return recovered, ok, err
		}
		request = continueRequestWithOperationalDirectorPlanStepScopeV0(request, state, step)
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, step.PendingAgentRefs...))
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, state.PendingAgentRefs...))
		if len(request.WaitAgentRefs) == 0 {
			request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, step.AgentRefs...))
		}
		return request, true, nil
	case orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0:
		if step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			return request, false, nil
		}
		if len(step.AgentRefs) == 0 {
			return request, false, nil
		}
		request = continueRequestWithOperationalDirectorPlanStepScopeV0(request, state, step)
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, step.AgentRefs...))
		return request, true, nil
	case orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0:
		if step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			return request, false, nil
		}
		if len(step.AgentRefs) == 0 {
			return request, false, nil
		}
		request = continueRequestWithOperationalDirectorPlanStepScopeV0(request, state, step)
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, step.AgentRefs...))
		return request, true, nil
	case orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0:
		if step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			return request, false, nil
		}
		request = continueRequestWithOperationalDirectorPlanStepScopeV0(request, state, step)
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, step.AgentRefs...))
		return request, true, nil
	default:
		return request, false, nil
	}
}

func continueOperationalDirectorPlanStateAfterBlockedWaitExpiredDeliveryV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		ports.RunStore == nil ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		activeStep.Reason != "external-wait-exhausted" {
		return state, false, nil
	}
	agentRefs := compactServiceRefsV0(append(activeStep.PendingAgentRefs, state.PendingAgentRefs...))
	if len(agentRefs) == 0 {
		agentRefs = compactServiceRefsV0(activeStep.AgentRefs)
	}
	if len(agentRefs) == 0 {
		return state, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return state, false, err
	}
	if !allServiceRefsInSetV0(agentRefs, run.DeliveredAgents) {
		return state, false, nil
	}
	reopened := state
	reopened.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0(nil), state.Steps...)
	reopened.Status = orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0
	reopened.PendingAgentRefs = append([]string(nil), agentRefs...)
	reopened.BlockerRefs = nil
	reopened.ClosureReason = ""
	reopened.EvidenceRefs = compactServiceRefsV0(append(
		reopened.EvidenceRefs,
		"evidence-ref-app-director-operational-plan-state-wait-expired-late-delivery-v0",
	))
	for index := range reopened.Steps {
		step := &reopened.Steps[index]
		if step.StepID != activeStep.StepID {
			continue
		}
		step.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
		step.PendingAgentRefs = append([]string(nil), agentRefs...)
		step.BlockerRefs = nil
		step.Reason = "late-delivery-after-wait-expired"
		step.EvidenceRefs = compactServiceRefsV0(append(
			step.EvidenceRefs,
			"evidence-ref-app-director-wait-subagents-late-delivery-v0",
		))
	}
	next, changed, err := operationalDirectorPlanStateAfterWaitConsumedV0(request, reopened, run)
	if err != nil || !changed {
		return state, false, err
	}
	return next, true, nil
}

func continueOperationalDirectorPlanStateAfterBlockedWaitDeliveredTasksV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		ports.RunStore == nil ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		(activeStep.Reason != "wait-subagents-terminal-without-delivery" &&
			activeStep.Reason != "external-wait-exhausted") {
		return state, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return state, false, err
	}
	return operationalDirectorPlanStateAfterWaitDeliveredTasksV0(request, state, run)
}

func continueRequestWithRecoveredWorkflowTaskWaitStateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorRequestV0, bool, error) {
	if ports.WaitStateStore == nil {
		return request, false, nil
	}
	waitRefs := compactServiceRefsV0(step.WaitRefs)
	if len(waitRefs) == 0 {
		return request, false, nil
	}
	waitState, err := ports.WaitStateStore.LoadWorkflowTaskWaitStateV0(ctx, request.RunRef, waitRefs[0])
	if err != nil {
		if appDirectorWorkflowTaskWaitStateNotFoundV0(err) {
			return request, false, nil
		}
		return request, false, err
	}
	if !appDirectorWorkflowTaskWaitStateMatchesPlanStepV0(request.RunRef, waitState, state, step) {
		return request, false, nil
	}
	request.WaitWaveRef = ""
	request.WaitCohortRef = ""
	request.WaitParentTaskRef = ""
	request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, waitState.PendingAgentRefs...))
	if len(request.WaitAgentRefs) == 0 {
		request.WaitAgentRefs = compactServiceRefsV0(append(request.WaitAgentRefs, waitState.AgentRefs...))
	}
	return request, len(request.WaitAgentRefs) > 0, nil
}

func appDirectorWorkflowTaskWaitStateMatchesPlanStepV0(
	runRef string,
	waitState orquestacionnucleoapp.WorkflowTaskWaitStateV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) bool {
	if strings.TrimSpace(waitState.RunRef) != strings.TrimSpace(runRef) {
		return false
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0 {
		return false
	}
	if !appDirectorWaitStateScopeMatchesV0(waitState.WaveRef, state.ActiveWaveRef, step.WaveRef) ||
		!appDirectorWaitStateScopeMatchesV0(waitState.CohortRef, state.ActiveCohortRef, step.CohortRef) ||
		!appDirectorWaitStateScopeMatchesV0(waitState.ParentTaskRef, state.ActiveParentTaskRef, step.ParentTaskRef) {
		return false
	}
	for _, taskRef := range waitState.TaskRefs {
		if !startAppDirectorStringInSetV0(step.TaskRefs, taskRef) {
			return false
		}
	}
	for _, agentRef := range waitState.AgentRefs {
		if !startAppDirectorStringInSetV0(step.AgentRefs, agentRef) {
			return false
		}
	}
	for _, agentRef := range waitState.PendingAgentRefs {
		if !startAppDirectorStringInSetV0(waitState.AgentRefs, agentRef) {
			return false
		}
	}
	return len(compactServiceRefsV0(append(waitState.PendingAgentRefs, waitState.AgentRefs...))) > 0
}

func appDirectorWaitStateScopeMatchesV0(waitValue string, stateValue string, stepValue string) bool {
	waitValue = strings.TrimSpace(waitValue)
	stateValue = strings.TrimSpace(stateValue)
	stepValue = strings.TrimSpace(stepValue)
	if waitValue == "" {
		return stateValue == "" && stepValue == ""
	}
	return (stateValue == "" || waitValue == stateValue) &&
		(stepValue == "" || waitValue == stepValue)
}

func appDirectorWorkflowTaskWaitStateNotFoundV0(err error) bool {
	issue, ok := err.(orquestacionnucleoapp.ErrorV0)
	return ok &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "workflow_task_wait_state"
}
