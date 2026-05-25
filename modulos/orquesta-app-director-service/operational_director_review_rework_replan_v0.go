package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

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
	followupTasks, followupsReady, err := operationalDirectorPlanReplanFollowupTasksV0(ctx, request, ports, loop.Run, match)
	if err != nil {
		return state, false, err
	}
	followupTaskRefs := continueOperationalDirectorTaskRefsV0(followupTasks)
	followupAgentRefs := continueOperationalDirectorAgentRefsV0(followupTasks)
	followupWaveRef, followupCohortRef, followupParentTaskRef, _ := operationalDirectorPlanFollowupScopeV0(followupTasks)
	followupAgentOnlyRefs := operationalDirectorPlanReplanFollowupAgentRefsV0(loop.Run, activeStep, match)
	waitStepID := ""
	if followupsReady || len(followupAgentOnlyRefs) > 0 {
		waitStepID = operationalDirectorPlanStateStepIDByKindV0(state, orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0)
		if waitStepID == "" {
			followupsReady = false
			followupAgentOnlyRefs = nil
		}
	}
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		switch {
		case step.StepID == activeStep.StepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0
			nextStep.DeliveryRefs = []string{match.DeliveryRef}
			nextStep.ReviewResultRefs = []string{match.ReviewResultRef}
			nextStep.AcceptedReviewRefs = nil
			nextStep.ReworkRequestRefs = []string{match.ReworkRequestRef}
			nextStep.ReplanDecisionRefs = []string{match.ReplanDecisionRef}
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, match.EvidenceRefs...))
			nextStep.BlockerRefs = []string{"review-rework-replan-recorded"}
			nextStep.Reason = "review-rework-replan-recorded"
		case followupsReady && step.StepID == waitStepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			nextStep.WaveRef = followupWaveRef
			nextStep.CohortRef = followupCohortRef
			nextStep.ParentTaskRef = followupParentTaskRef
			nextStep.TaskRefs = append([]string(nil), followupTaskRefs...)
			nextStep.AgentRefs = append([]string(nil), followupAgentRefs...)
			nextStep.PendingAgentRefs = append([]string(nil), followupAgentRefs...)
			nextStep.WaitRefs = []string{appDirectorWaitRefV0(
				request.RunRef,
				appDirectorWaitFilterV0{
					WaveRef:       followupWaveRef,
					CohortRef:     followupCohortRef,
					ParentTaskRef: followupParentTaskRef,
				},
				request.CorrelationID,
			)}
			nextStep.BlockerRefs = []string{"wait-subagents-replan-followups"}
			nextStep.Reason = "review-rework-replan-followups-waiting"
		case len(followupAgentOnlyRefs) > 0 && step.StepID == waitStepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			nextStep.WaveRef = ""
			nextStep.CohortRef = ""
			nextStep.ParentTaskRef = ""
			nextStep.TaskRefs = []string{match.TaskRef}
			nextStep.AgentRefs = append([]string(nil), followupAgentOnlyRefs...)
			nextStep.PendingAgentRefs = append([]string(nil), followupAgentOnlyRefs...)
			nextStep.WaitRefs = []string{appDirectorWaitRefV0(request.RunRef, appDirectorWaitFilterV0{}, request.CorrelationID)}
			nextStep.BlockerRefs = []string{"wait-subagents-replan-followup-agents"}
			nextStep.Reason = "review-rework-replan-followup-agents-waiting"
		}
		nextSteps = append(nextSteps, nextStep)
	}
	if followupsReady {
		state.ActiveStepID = waitStepID
		state.ActiveWaveRef = followupWaveRef
		state.ActiveCohortRef = followupCohortRef
		state.ActiveParentTaskRef = followupParentTaskRef
		state.PendingAgentRefs = append([]string(nil), followupAgentRefs...)
	} else if len(followupAgentOnlyRefs) > 0 {
		state.ActiveStepID = waitStepID
		state.ActiveWaveRef = ""
		state.ActiveCohortRef = ""
		state.ActiveParentTaskRef = ""
		state.PendingAgentRefs = append([]string(nil), followupAgentOnlyRefs...)
	} else {
		state.ActiveStepID = activeStep.StepID
		state.PendingAgentRefs = nil
	}
	state.ReplanAttempts++
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-v0"))
	if followupsReady {
		state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followups-v0"))
	}
	if len(followupAgentOnlyRefs) > 0 {
		state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followup-agents-v0"))
	}
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}
