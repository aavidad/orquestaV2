package orquestaappdirectorservice

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func operationalDirectorPlanStateWaitStepCanRecoverByTaskDeliveryV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) bool {
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 {
		return false
	}
	if activeStep.Reason != "wait-subagents-terminal-without-delivery" &&
		activeStep.Reason != "external-wait-exhausted" {
		return false
	}
	for _, step := range state.Steps {
		if step.Kind == orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 {
			if step.Status == orquestadirectoroperativo.OperationalDirectorStepPendingV0 {
				return true
			}
			return step.Status == orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 &&
				operationalDirectorPlanStateWaitStepLooksReworkFollowupV0(activeStep, step)
		}
	}
	return false
}

func operationalDirectorPlanStateWaitStepLooksReworkFollowupV0(
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	reviewStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) bool {
	if startAppDirectorStringInSetV0(step.BlockerRefs, "wait-subagents-replan-followups") ||
		startAppDirectorStringInSetV0(step.BlockerRefs, "wait-subagents-replan-followup-agents") {
		return true
	}
	for _, taskRef := range compactServiceRefsV0(step.TaskRefs) {
		if !startAppDirectorStringInSetV0(reviewStep.TaskRefs, taskRef) {
			return true
		}
	}
	return false
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
			nextStep.TaskRefs = append([]string(nil), activeStep.TaskRefs...)
			nextStep.AgentRefs = append([]string(nil), activeStep.AgentRefs...)
			nextStep.WaveRef = activeStep.WaveRef
			nextStep.CohortRef = activeStep.CohortRef
			nextStep.ParentTaskRef = activeStep.ParentTaskRef
			nextStep.DeliveryRefs = nil
			nextStep.ReviewResultRefs = nil
			nextStep.AcceptedReviewRefs = nil
			nextStep.ReworkRequestRefs = nil
			nextStep.ReplanDecisionRefs = nil
			nextStep.BlockerRefs = nil
			nextStep.Reason = "wait-subagents-consumed"
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
