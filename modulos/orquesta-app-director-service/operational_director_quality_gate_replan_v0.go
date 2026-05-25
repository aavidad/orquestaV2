package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

type operationalDirectorPlanQualityGateReplanTransitionV0 struct {
	SetRequiredTestEvidenceRefs bool
	RequiredTestEvidenceRefs    []string
	ActiveStepReason            string
	ActiveStepBlockerRefs       []string
	WaitFollowupsBlocker        string
	WaitFollowupsReason         string
	WaitAgentsBlocker           string
	WaitAgentsReason            string
	EvidenceRefs                []string
}

func operationalDirectorPlanStateAfterQualityGateReplanFollowupsV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	replan orquestacoreworkflow.ReplanDecisionRecordedPayloadV0,
	gateRef string,
	transition operationalDirectorPlanQualityGateReplanTransitionV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	match := operationalDirectorPlanReviewReworkReplanMatchV0{
		TaskRef:           replan.TaskRef,
		ReplanDecisionRef: replan.ReplanRef,
		AcceptedAction:    replan.AcceptedAction,
		FollowupRefs:      append([]string(nil), replan.FollowupRefs...),
	}
	followupTasks, followupsReady, err := operationalDirectorPlanReplanFollowupTasksV0(ctx, request, ports, run, match)
	if err != nil {
		return state, false, err
	}
	followupTaskRefs := continueOperationalDirectorTaskRefsV0(followupTasks)
	followupAgentRefs := continueOperationalDirectorAgentRefsV0(followupTasks)
	followupWaveRef, followupCohortRef, followupParentTaskRef, _ := operationalDirectorPlanFollowupScopeV0(followupTasks)
	followupAgentOnlyRefs := operationalDirectorPlanReplanFollowupAgentRefsV0(run, activeStep, match)
	waitStepID := ""
	if followupsReady || len(followupAgentOnlyRefs) > 0 {
		waitStepID = operationalDirectorPlanStateStepIDByKindV0(state, orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0)
		if waitStepID == "" {
			followupsReady = false
			followupAgentOnlyRefs = nil
		}
	}
	if !followupsReady && len(followupAgentOnlyRefs) == 0 {
		return state, false, nil
	}
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		switch {
		case step.StepID == activeStep.StepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			if transition.SetRequiredTestEvidenceRefs {
				nextStep.RequiredTestEvidenceRefs = append([]string(nil), transition.RequiredTestEvidenceRefs...)
			}
			nextStep.ReplanDecisionRefs = []string{replan.ReplanRef}
			nextStep.BlockerRefs = compactServiceRefsV0(append(append([]string(nil), transition.ActiveStepBlockerRefs...), gateRef))
			nextStep.Reason = transition.ActiveStepReason
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
			nextStep.BlockerRefs = []string{transition.WaitFollowupsBlocker}
			nextStep.Reason = transition.WaitFollowupsReason
		case len(followupAgentOnlyRefs) > 0 && step.StepID == waitStepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			nextStep.WaveRef = ""
			nextStep.CohortRef = ""
			nextStep.ParentTaskRef = ""
			nextStep.TaskRefs = []string{replan.TaskRef}
			nextStep.AgentRefs = append([]string(nil), followupAgentOnlyRefs...)
			nextStep.PendingAgentRefs = append([]string(nil), followupAgentOnlyRefs...)
			nextStep.WaitRefs = []string{appDirectorWaitRefV0(request.RunRef, appDirectorWaitFilterV0{}, request.CorrelationID)}
			nextStep.BlockerRefs = []string{transition.WaitAgentsBlocker}
			nextStep.Reason = transition.WaitAgentsReason
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
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0
	state.ActiveStepID = waitStepID
	state.ReplanAttempts++
	state.BlockerRefs = nil
	state.ClosureReason = ""
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(
		append(state.EvidenceRefs, transition.EvidenceRefs...),
		gateRef,
		replan.ReplanRef,
	))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

func operationalDirectorPlanRequiredTestsReplanScopeSupportedV0(
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) bool {
	return len(compactServiceRefsV0(activeStep.TaskRefs)) == 1 && len(matches) == 1
}

func operationalDirectorPlanRequiredTestsReplanDecisionV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
	failedRefs []string,
) (orquestacoreworkflow.ReplanDecisionRecordedPayloadV0, string, bool, error) {
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, nil
	}
	events, err := reader.LoadRunEventsV0(ctx, request.RunRef)
	if err != nil {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	for _, match := range matches {
		gate, ok := operationalDirectorPlanRequiredTestsQualityGateV0(run, trace, activeStep, match.TaskRef, failedRefs)
		if !ok {
			continue
		}
		replan, ok := operationalDirectorPlanReplanForQualityGateV0(run, trace, gate.GateRef, match.TaskRef)
		if ok {
			return replan, strings.TrimSpace(gate.GateRef), true, nil
		}
	}
	if !operationalDirectorPlanRequiredTestsReplanScopeSupportedV0(activeStep, matches) {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, nil
	}
	return operationalDirectorPlanEmitRequiredTestsReplanDecisionV0(
		ctx,
		request,
		ports,
		run,
		activeStep,
		matches,
		failedRefs,
	)
}
