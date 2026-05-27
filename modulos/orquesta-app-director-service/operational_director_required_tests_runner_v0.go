package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func operationalDirectorPlanRunRequiredTestsV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	requiredTests []string,
	requiredTestsByTask map[string][]string,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, bool, error) {
	if ports.RequiredTestRunner == nil {
		return activeStep, false, nil
	}
	evidenceRefs := make([]string, 0, len(requiredTests)*len(matches))
	for _, match := range matches {
		matchRequiredTests := operationalDirectorPlanRequiredTestsForMatchV0(requiredTestsByTask, requiredTests, match)
		if len(matchRequiredTests) == 0 {
			continue
		}
		result, err := ports.RequiredTestRunner.RunRequiredTestsV0(ctx, orquestacionnucleoapp.RequiredTestExecutionRequestV0{
			RunRef:            request.RunRef,
			TaskRef:           match.TaskRef,
			TestCommands:      matchRequiredTests,
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
