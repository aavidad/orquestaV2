package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func operationalDirectorPlanEmitRequiredTestsReplanDecisionV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
	failedRefs []string,
) (orquestacoreworkflow.ReplanDecisionRecordedPayloadV0, string, bool, error) {
	if ports.RunStore == nil || ports.EventSink == nil ||
		!operationalDirectorPlanRequiredTestsReplanScopeSupportedV0(activeStep, matches) {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, nil
	}
	match := matches[0]
	taskRef := strings.TrimSpace(match.TaskRef)
	if taskRef == "" || taskRef != strings.TrimSpace(activeStep.TaskRefs[0]) {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, nil
	}
	failedRefs = compactServiceRefsV0(failedRefs)
	if len(failedRefs) == 0 {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, nil
	}
	refs := operationalDirectorRequiredTestsAutoReplanRefsV0(request, match, failedRefs)
	storedRun, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, err
	}
	if strings.TrimSpace(storedRun.RunID) != strings.TrimSpace(run.RunID) {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, nil
	}
	storedRun, ready, err := operationalDirectorPlanEnsureProgrammingPhaseForRequiredTestsReplanV0(ctx, request, ports, storedRun, refs)
	if err != nil {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, err
	}
	if !ready || !operationalDirectorRunProgrammingPhaseActiveV0(storedRun) {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, nil
	}

	gatePayload := orquestacoreworkflow.RecordQualityGateCommandPayloadV0{
		RunRef:       request.RunRef,
		GateRef:      refs.GateRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		SubjectRef:   taskRef,
		Decision:     orquestacoreworkflow.QualityGateDecisionBlockedV0,
		IssueRefs:    append([]string(nil), failedRefs...),
		Summary:      "Tests requeridos fallidos para una tarea causal.",
		EvidenceRefs: operationalDirectorRequiredTestsAutoReplanEvidenceRefsV0(match, failedRefs),
	}
	gateCommand, err := orquestacoreworkflow.NewRecordQualityGateCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-" + refs.GateRef,
			RunID:          request.RunRef,
			IdempotencyKey: "idem-" + refs.GateRef,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    request.RequestedBy,
			OccurredAt:     request.OccurredAt,
		},
		gatePayload,
	)
	if err != nil {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, gateCommand); err != nil {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, err
	}

	replanPayload := orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
		ReplanRef:      refs.ReplanRef,
		RunRef:         request.RunRef,
		TaskRef:        taskRef,
		SourceRef:      refs.GateRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		FollowupRefs:   []string{refs.CapacityRef, refs.AgentRef},
		Summary:        "Reintentar tarea tras tests requeridos fallidos.",
		EvidenceRefs:   compactServiceRefsV0(append([]string{refs.GateRef}, failedRefs...)),
	}
	replanCommand, err := orquestacoreworkflow.NewRecordReplanDecisionCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-" + refs.ReplanRef,
			RunID:          request.RunRef,
			IdempotencyKey: "idem-" + refs.ReplanRef,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    request.RequestedBy,
			OccurredAt:     request.OccurredAt,
		},
		replanPayload,
	)
	if err != nil {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, replanCommand); err != nil {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, err
	}
	return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0(replanPayload), refs.GateRef, true, nil
}

func operationalDirectorPlanEnsureProgrammingPhaseForRequiredTestsReplanV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	refs operationalDirectorRequiredTestsAutoReplanRefSetV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if operationalDirectorRunProgrammingPhaseActiveV0(run) {
		return run, true, nil
	}
	if !operationalDirectorRunContainsPhaseV0(run, orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		return run, false, nil
	}
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-open-phase-" + refs.ReplanRef,
			RunID:          request.RunRef,
			IdempotencyKey: "idem-open-phase-" + refs.ReplanRef,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    request.RequestedBy,
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "required-tests-failed-replan",
		},
	)
	if err != nil {
		return run, false, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, command); err != nil {
		return run, false, err
	}
	updated, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return run, false, err
	}
	return updated, operationalDirectorRunProgrammingPhaseActiveV0(updated), nil
}
