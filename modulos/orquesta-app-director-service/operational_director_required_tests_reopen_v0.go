package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func continueOperationalDirectorPlanStateAfterBlockedRequiredTestsReplanV0(
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
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		activeStep.Reason != "required-tests-failed" {
		return state, false, nil
	}
	failedRefs := compactServiceRefsV0(activeStep.RequiredTestEvidenceRefs)
	if len(failedRefs) == 0 {
		return state, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return state, false, err
	}
	matches, complete, err := operationalDirectorPlanAcceptedReviewMatchesV0(ctx, request, ports, activeStep, run)
	if err != nil || !complete {
		return state, false, err
	}
	return operationalDirectorPlanStateAfterRequiredTestsReplanV0(
		ctx,
		request,
		ports,
		state,
		activeStep,
		run,
		matches,
		failedRefs,
	)
}

func continueOperationalDirectorPlanStateAfterBlockedRequiredTestsEvidenceV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if ports.OperationalPlanStateWriter == nil || ports.RunStore == nil || ports.RequiredTestEvidenceStore == nil {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		activeStep.Reason != "required-tests-evidence-missing" {
		return state, false, nil
	}
	requiredTests := compactServiceRefsV0(state.RequiredTestRefs)
	if len(requiredTests) == 0 {
		return state, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return state, false, err
	}
	matches, complete, err := operationalDirectorPlanAcceptedReviewMatchesV0(ctx, request, ports, activeStep, run)
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
		}
	}
	if len(testStatus.FailedRefs) > 0 {
		replanned, replannedChanged, err := operationalDirectorPlanStateAfterRequiredTestsReplanV0(
			ctx,
			request,
			ports,
			state,
			activeStep,
			run,
			matches,
			testStatus.FailedRefs,
		)
		if err != nil || replannedChanged {
			return replanned, replannedChanged, err
		}
		return operationalDirectorPlanStateWithRequiredTestsBlockedV0(request, state, activeStep, testStatus.FailedRefs)
	}
	if testStatus.Complete {
		if err := operationalDirectorPlanRecordRequiredTestsPassedQualityGateV0(
			ctx,
			request,
			ports,
			run,
			activeStep,
			matches,
			testStatus.PassedRefs,
		); err != nil {
			return state, false, err
		}
		return operationalDirectorPlanStateWithRequiredTestsPassedV0(request, state, activeStep, testStatus.PassedRefs)
	}
	return state, false, nil
}
