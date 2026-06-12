package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

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
	QualityGates       map[string]orquestacoreworkflow.QualityGateRecordedPayloadV0
	QualityGateRefs    []string
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
	AcceptedAction    orquestacoreworkflow.ReplanDecisionActionV0
	FollowupRefs      []string
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
			if nextStepKind == orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0 {
				nextStep.TaskRefs = append([]string(nil), activeStep.TaskRefs...)
				nextStep.AgentRefs = append([]string(nil), activeStep.AgentRefs...)
				nextStep.DeliveryRefs = append([]string(nil), deliveryRefs...)
				nextStep.ReviewResultRefs = append([]string(nil), reviewResultRefs...)
				nextStep.AcceptedReviewRefs = append([]string(nil), acceptedReviewRefs...)
				nextStep.RequiredTestEvidenceRefs = nil
			} else {
				nextStep.TaskRefs = append([]string(nil), activeStep.TaskRefs...)
				nextStep.AgentRefs = append([]string(nil), activeStep.AgentRefs...)
				nextStep.DeliveryRefs = append([]string(nil), deliveryRefs...)
				nextStep.ReviewResultRefs = append([]string(nil), reviewResultRefs...)
				nextStep.AcceptedReviewRefs = append([]string(nil), acceptedReviewRefs...)
			}
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, evidenceRefs...))
			nextStep.WaveRef = activeStep.WaveRef
			nextStep.CohortRef = activeStep.CohortRef
			nextStep.ParentTaskRef = activeStep.ParentTaskRef
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
