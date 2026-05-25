package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func operationalDirectorPlanStateAfterRequiredTestsReplanV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
	failedRefs []string,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if len(compactServiceRefsV0(activeStep.TaskRefs)) == 0 || len(matches) == 0 {
		return state, false, nil
	}
	replan, gateRef, ok, err := operationalDirectorPlanRequiredTestsReplanDecisionV0(ctx, request, ports, run, activeStep, matches, failedRefs)
	if err != nil || !ok {
		return state, false, err
	}
	return operationalDirectorPlanStateAfterQualityGateReplanFollowupsV0(
		ctx,
		request,
		ports,
		state,
		activeStep,
		run,
		replan,
		gateRef,
		operationalDirectorPlanQualityGateReplanTransitionV0{
			SetRequiredTestEvidenceRefs: true,
			RequiredTestEvidenceRefs:    failedRefs,
			ActiveStepReason:            "required-tests-failed-replan-recorded",
			ActiveStepBlockerRefs:       []string{"required-tests-failed"},
			WaitFollowupsBlocker:        "wait-subagents-required-tests-replan-followups",
			WaitFollowupsReason:         "required-tests-failed-replan-followups-waiting",
			WaitAgentsBlocker:           "wait-subagents-required-tests-replan-followup-agents",
			WaitAgentsReason:            "required-tests-failed-replan-followup-agents-waiting",
			EvidenceRefs: []string{
				"evidence-ref-app-director-operational-plan-state-required-tests-failed-replan-v0",
			},
		},
	)
}

func operationalDirectorPlanRecordRequiredTestsEvidenceMissingQualityGateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
	requiredTests []string,
) error {
	if ports.RunStore == nil || ports.EventSink == nil ||
		!operationalDirectorPlanRequiredTestsReplanScopeSupportedV0(activeStep, matches) {
		return nil
	}
	requiredTests = compactServiceRefsV0(requiredTests)
	if len(requiredTests) == 0 {
		return nil
	}
	match := matches[0]
	if strings.TrimSpace(match.TaskRef) == "" || strings.TrimSpace(match.TaskRef) != strings.TrimSpace(activeStep.TaskRefs[0]) {
		return nil
	}
	refs := operationalDirectorRequiredTestsEvidenceMissingGateRefsV0(request, match, requiredTests)
	storedRun, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(storedRun.RunID) != strings.TrimSpace(run.RunID) {
		return nil
	}
	storedRun, ready, err := operationalDirectorPlanEnsureProgrammingPhaseForRequiredTestsReplanV0(ctx, request, ports, storedRun, refs)
	if err != nil || !ready || !operationalDirectorRunProgrammingPhaseActiveV0(storedRun) {
		return err
	}
	payload := orquestacoreworkflow.RecordQualityGateCommandPayloadV0{
		RunRef:       request.RunRef,
		GateRef:      refs.GateRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		SubjectRef:   match.TaskRef,
		Decision:     orquestacoreworkflow.QualityGateDecisionBlockedV0,
		IssueRefs:    []string{"required-tests-evidence-missing"},
		Summary:      "Tests requeridos sin evidencia durable.",
		EvidenceRefs: operationalDirectorRequiredTestsAutoReplanEvidenceRefsV0(match, []string{"required-tests-evidence-missing"}),
	}
	command, err := orquestacoreworkflow.NewRecordQualityGateCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-" + refs.GateRef,
			RunID:          request.RunRef,
			IdempotencyKey: "idem-" + refs.GateRef,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    request.RequestedBy,
			OccurredAt:     request.OccurredAt,
		},
		payload,
	)
	if err != nil {
		return err
	}
	_, err = orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, command)
	return err
}

func operationalDirectorPlanRecordRequiredTestsPassedQualityGateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
	passedRefs []string,
) error {
	if ports.RunStore == nil || ports.EventSink == nil || len(matches) == 0 {
		return nil
	}
	passedRefs = compactServiceRefsV0(passedRefs)
	if len(passedRefs) == 0 {
		return nil
	}
	storedRun, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(storedRun.RunID) != strings.TrimSpace(run.RunID) {
		return nil
	}
	for _, match := range matches {
		taskRef := strings.TrimSpace(match.TaskRef)
		if taskRef == "" ||
			!startAppDirectorStringInSetV0(activeStep.TaskRefs, taskRef) ||
			len(orquestacoreworkflow.PendingBlockingQualityGateRefsForSubjectV0(storedRun, taskRef)) == 0 {
			continue
		}
		if strings.TrimSpace(string(storedRun.CurrentPhase)) == "" {
			continue
		}
		refs := operationalDirectorRequiredTestsPassedGateRefsV0(request, match, passedRefs)
		payload := orquestacoreworkflow.RecordQualityGateCommandPayloadV0{
			RunRef:       request.RunRef,
			GateRef:      refs.GateRef,
			PhaseID:      string(storedRun.CurrentPhase),
			SubjectRef:   taskRef,
			Decision:     orquestacoreworkflow.QualityGateDecisionAcceptedV0,
			Summary:      "Tests requeridos pasados tras followup causal.",
			EvidenceRefs: operationalDirectorRequiredTestsPassedGateEvidenceRefsV0(match, passedRefs),
		}
		command, err := orquestacoreworkflow.NewRecordQualityGateCommandV0(
			orquestacoreworkflow.OrchestrationCommandMetaV0{
				CommandID:      "cmd-" + refs.GateRef,
				RunID:          request.RunRef,
				IdempotencyKey: "idem-" + refs.GateRef,
				CorrelationID:  request.CorrelationID,
				RequestedBy:    request.RequestedBy,
				OccurredAt:     request.OccurredAt,
			},
			payload,
		)
		if err != nil {
			return err
		}
		if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, command); err != nil {
			return err
		}
		storedRun, err = ports.RunStore.LoadRunV0(ctx, request.RunRef)
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureOperationalDirectorPassedRequiredTestsQualityGateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) error {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" ||
		ports.OperationalPlanStateStore == nil ||
		ports.RunStore == nil ||
		ports.EventSink == nil {
		return nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return err
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		activeStep.Reason != "required-tests-passed" {
		return nil
	}
	passedRefs := compactServiceRefsV0(activeStep.RequiredTestEvidenceRefs)
	if len(passedRefs) == 0 {
		passedRefs = operationalDirectorPlanStateRequiredTestEvidenceRefsV0(state)
	}
	if len(passedRefs) == 0 || !operationalDirectorPlanStepHasPendingQualityGateV0(ctx, request, ports, activeStep) {
		return nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return err
	}
	matches, complete, err := operationalDirectorPlanAcceptedReviewMatchesV0(ctx, request, ports, activeStep, run)
	if err != nil || !complete {
		return err
	}
	return operationalDirectorPlanRecordRequiredTestsPassedQualityGateV0(
		ctx,
		request,
		ports,
		run,
		activeStep,
		matches,
		passedRefs,
	)
}

func operationalDirectorPlanStateRequiredTestEvidenceRefsV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) []string {
	refs := []string(nil)
	for _, step := range state.Steps {
		refs = append(refs, step.RequiredTestEvidenceRefs...)
	}
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanStepHasPendingQualityGateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) bool {
	if ports.RunStore == nil {
		return false
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return false
	}
	for _, taskRef := range compactServiceRefsV0(activeStep.TaskRefs) {
		if len(orquestacoreworkflow.PendingBlockingQualityGateRefsForSubjectV0(run, taskRef)) > 0 {
			return true
		}
	}
	return false
}
