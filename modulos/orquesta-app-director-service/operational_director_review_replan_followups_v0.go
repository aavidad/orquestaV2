package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func continueOperationalDirectorPlanStateAfterReviewReworkReplanFollowupsV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if ports.OperationalPlanStateWriter == nil || ports.RunStore == nil {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		activeStep.Reason != "review-rework-replan-recorded" ||
		len(compactServiceRefsV0(activeStep.ReworkRequestRefs)) == 0 ||
		len(compactServiceRefsV0(activeStep.ReplanDecisionRefs)) == 0 {
		return state, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return state, false, err
	}
	match, ok, err := operationalDirectorPlanReviewReworkReplanMatchForActiveStepV0(ctx, request, ports, activeStep, run)
	if err != nil || !ok {
		return state, false, err
	}
	if !startAppDirectorStringInSetV0(activeStep.ReworkRequestRefs, match.ReworkRequestRef) ||
		!startAppDirectorStringInSetV0(activeStep.ReplanDecisionRefs, match.ReplanDecisionRef) {
		return state, false, nil
	}
	followupTasks, followupsReady, err := operationalDirectorPlanReplanFollowupTasksV0(ctx, request, ports, run, match)
	if err != nil {
		return state, false, err
	}
	followupTaskRefs := continueOperationalDirectorTaskRefsV0(followupTasks)
	followupAgentRefs := continueOperationalDirectorAgentRefsV0(followupTasks)
	followupWaveRef, followupCohortRef, followupParentTaskRef, _ := operationalDirectorPlanFollowupScopeV0(followupTasks)
	followupAgentOnlyRefs := operationalDirectorPlanReplanFollowupAgentRefsV0(run, activeStep, match)
	if !followupsReady && len(followupAgentOnlyRefs) == 0 {
		return state, false, nil
	}
	waitStepID := operationalDirectorPlanStateStepIDByKindV0(state, orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0)
	if waitStepID == "" {
		return state, false, nil
	}
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		switch {
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
		state.ActiveWaveRef = followupWaveRef
		state.ActiveCohortRef = followupCohortRef
		state.ActiveParentTaskRef = followupParentTaskRef
		state.PendingAgentRefs = append([]string(nil), followupAgentRefs...)
	} else {
		state.ActiveWaveRef = ""
		state.ActiveCohortRef = ""
		state.ActiveParentTaskRef = ""
		state.PendingAgentRefs = append([]string(nil), followupAgentOnlyRefs...)
	}
	state.ActiveStepID = waitStepID
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followups-late-v0"))
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
