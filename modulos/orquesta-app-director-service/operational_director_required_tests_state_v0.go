package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

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
	requiredTestsByTask, err := operationalDirectorPlanRequiredTestsByTaskV0(
		ctx,
		request.RunRef,
		ports.DirectorTaskStore,
		activeStep,
		requiredTests,
		matches,
	)
	if err != nil {
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
	testStatus := operationalDirectorPlanEvaluateRequiredTestEvidenceV0(request.RunRef, requiredTests, matches, evidence, requiredTestsByTask)
	if len(testStatus.FailedRefs) > 0 {
		replanned, replannedChanged, err := operationalDirectorPlanStateAfterRequiredTestsReplanV0(
			ctx,
			request,
			ports,
			state,
			activeStep,
			loop.Run,
			matches,
			testStatus.FailedRefs,
		)
		if err != nil || replannedChanged {
			return replanned, replannedChanged, err
		}
		return operationalDirectorPlanStateWithRequiredTestsBlockedV0(request, state, activeStep, testStatus.FailedRefs)
	}
	if !testStatus.Complete {
		updatedStep, generated, err := operationalDirectorPlanRunRequiredTestsV0(
			ctx,
			request,
			ports,
			activeStep,
			requiredTests,
			requiredTestsByTask,
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
			testStatus = operationalDirectorPlanEvaluateRequiredTestEvidenceV0(request.RunRef, requiredTests, matches, evidence, requiredTestsByTask)
			if len(testStatus.FailedRefs) > 0 {
				replanned, replannedChanged, err := operationalDirectorPlanStateAfterRequiredTestsReplanV0(
					ctx,
					request,
					ports,
					state,
					activeStep,
					loop.Run,
					matches,
					testStatus.FailedRefs,
				)
				if err != nil || replannedChanged {
					return replanned, replannedChanged, err
				}
				return operationalDirectorPlanStateWithRequiredTestsBlockedV0(request, state, activeStep, testStatus.FailedRefs)
			}
		}
	}
	if !testStatus.Complete {
		if err := operationalDirectorPlanRecordRequiredTestsEvidenceMissingQualityGateV0(
			ctx,
			request,
			ports,
			loop.Run,
			activeStep,
			matches,
			operationalDirectorPlanRequiredTestsForMatchV0(requiredTestsByTask, requiredTests, matches[0]),
		); err != nil {
			return state, false, err
		}
		return operationalDirectorPlanStateWithRequiredTestsEvidenceMissingV0(request, state, activeStep)
	}
	if err := operationalDirectorPlanRecordRequiredTestsPassedQualityGateV0(
		ctx,
		request,
		ports,
		loop.Run,
		activeStep,
		matches,
		testStatus.PassedRefs,
	); err != nil {
		return state, false, err
	}
	return operationalDirectorPlanStateWithRequiredTestsPassedV0(request, state, activeStep, testStatus.PassedRefs)
}

func operationalDirectorPlanStateWithRequiredTestsPassedV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	passedRefs []string,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	passedRefs = compactServiceRefsV0(passedRefs)
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
			nextStep.RequiredTestEvidenceRefs = append([]string(nil), passedRefs...)
			nextStep.BlockerRefs = nil
			nextStep.Reason = "required-tests-passed"
		case replanStepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			nextStep.TaskRefs = append([]string(nil), activeStep.TaskRefs...)
			nextStep.AgentRefs = append([]string(nil), activeStep.AgentRefs...)
			nextStep.DeliveryRefs = append([]string(nil), activeStep.DeliveryRefs...)
			nextStep.ReviewResultRefs = append([]string(nil), activeStep.ReviewResultRefs...)
			nextStep.RequiredTestEvidenceRefs = append([]string(nil), passedRefs...)
			nextStep.WaveRef = activeStep.WaveRef
			nextStep.CohortRef = activeStep.CohortRef
			nextStep.ParentTaskRef = activeStep.ParentTaskRef
			nextStep.BlockerRefs = nil
			nextStep.Reason = "required-tests-passed"
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0
	state.ActiveStepID = replanStepID
	state.PendingAgentRefs = nil
	state.BlockerRefs = nil
	state.ClosureReason = ""
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-required-tests-passed-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

func operationalDirectorPlanStateWithRequiredTestsEvidenceMissingV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == activeStep.StepID {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			nextStep.BlockerRefs = []string{"required-tests-evidence-missing"}
			nextStep.Reason = "required-tests-evidence-missing"
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.PendingAgentRefs = nil
	state.BlockerRefs = []string{"required-tests-evidence-missing"}
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-required-tests-evidence-missing-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}
