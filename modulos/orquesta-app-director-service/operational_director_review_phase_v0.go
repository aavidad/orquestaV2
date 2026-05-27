package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strconv"
	"strings"
)

func ensureOperationalDirectorReviewPhaseV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (bool, error) {
	state, activeStep, ok, err := operationalDirectorActiveReviewDeliveriesStepV0(ctx, request, ports)
	if err != nil || !ok {
		return false, err
	}
	if ports.RunStore == nil || ports.EventSink == nil {
		return false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return false, err
	}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		operationalDirectorRunPhaseActiveV0(run, orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		return false, nil
	}
	if !operationalDirectorRunContainsPhaseV0(run, orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		return false, AppDirectorServiceIssueV0{Field: "run.phases.revision"}
	}
	suffix := operationalDirectorReviewPhaseCommandSuffixV0(request, state, activeStep)
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-open-review-" + suffix,
			RunID:          request.RunRef,
			IdempotencyKey: "idem-open-review-" + suffix,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    request.RequestedBy,
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			Reason:  "operational-director-review-deliveries",
		},
	)
	if err != nil {
		return false, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, command); err != nil {
		return false, err
	}
	return true, nil
}

func operationalDirectorActiveReviewDeliveriesStepV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (
	orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	bool,
	error,
) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{},
			orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{},
			false,
			nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{},
			orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{},
			false,
			err
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, activeStep, false, nil
	}
	return state, activeStep, true, nil
}

func operationalDirectorReviewPhaseCommandSuffixV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) string {
	return appDirectorSafeRefPartV0(strings.Join(compactServiceRefsV0([]string{
		request.RunRef,
		state.PlanRef,
		activeStep.StepID,
		activeStep.WaveRef,
		activeStep.CohortRef,
		activeStep.ParentTaskRef,
		"replan-attempt-" + strconv.Itoa(state.ReplanAttempts),
	}), "-"))
}

func continueOperationalDirectorPlanStateCanProgressBlockedRequiredTestsReplanV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (bool, error) {
	if ports.RunStore == nil {
		return false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		activeStep.Reason != "required-tests-failed" {
		return false, nil
	}
	failedRefs := compactServiceRefsV0(activeStep.RequiredTestEvidenceRefs)
	if len(failedRefs) == 0 {
		return false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return false, err
	}
	matches, complete, err := operationalDirectorPlanAcceptedReviewMatchesV0(ctx, request, ports, activeStep, run)
	if err != nil || !complete {
		return false, err
	}
	if !operationalDirectorPlanRequiredTestsReplanScopeSupportedV0(activeStep, matches) {
		return false, nil
	}
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return false, nil
	}
	events, err := loadOperationalRunEventsV0(ctx, reader, request.RunRef)
	if err != nil {
		return false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	for _, match := range matches {
		gate, ok := operationalDirectorPlanRequiredTestsQualityGateV0(run, trace, activeStep, match.TaskRef, failedRefs)
		if !ok {
			continue
		}
		if _, ok := operationalDirectorPlanReplanForQualityGateV0(run, trace, gate.GateRef, match.TaskRef); ok {
			return true, nil
		}
	}
	return false, nil
}
