package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func continueOperationalDirectorPlanStateAfterBlockedClosureIssuesReplanV0(
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
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		activeStep.Reason != "operational-closure-issues" {
		return state, false, nil
	}
	issueRefs := operationalDirectorClosureReplannableBlockerRefsV0(append(state.BlockerRefs, activeStep.BlockerRefs...))
	if len(issueRefs) == 0 {
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
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return state, false, nil
	}
	events, err := loadOperationalRunEventsV0(ctx, reader, request.RunRef)
	if err != nil {
		return state, false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	var selectedMatch operationalDirectorPlanAcceptedReviewMatchV0
	var selectedReplan orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
	selectedGateRef := ""
	for _, match := range matches {
		gate, ok := operationalDirectorPlanRequiredTestsQualityGateV0(run, trace, activeStep, match.TaskRef, issueRefs)
		if !ok {
			continue
		}
		replan, ok := operationalDirectorPlanReplanForQualityGateV0(run, trace, gate.GateRef, match.TaskRef)
		if !ok {
			continue
		}
		if selectedGateRef != "" {
			return state, false, nil
		}
		selectedMatch = match
		selectedReplan = replan
		selectedGateRef = strings.TrimSpace(gate.GateRef)
	}
	if selectedGateRef == "" {
		replan, gateRef, emitted, err := operationalDirectorPlanEmitBlockedClosureIssuesReplanDecisionV0(
			ctx,
			request,
			ports,
			run,
			activeStep,
			matches,
			issueRefs,
		)
		if err != nil || !emitted {
			return state, false, err
		}
		updatedRun, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
		if err != nil {
			return state, false, err
		}
		selectedReplan = replan
		selectedGateRef = gateRef
		run = updatedRun
	}
	if selectedGateRef == "" || strings.TrimSpace(selectedMatch.TaskRef) == "" {
		if strings.TrimSpace(selectedReplan.TaskRef) == "" {
			return state, false, nil
		}
		selectedMatch.TaskRef = selectedReplan.TaskRef
	}
	return operationalDirectorPlanStateAfterQualityGateReplanFollowupsV0(
		ctx,
		request,
		ports,
		state,
		activeStep,
		run,
		selectedReplan,
		selectedGateRef,
		operationalDirectorPlanQualityGateReplanTransitionV0{
			ActiveStepReason:      "operational-closure-issues-replan-recorded",
			ActiveStepBlockerRefs: append([]string{"operational-closure-issues"}, issueRefs...),
			WaitFollowupsBlocker:  "wait-subagents-operational-closure-replan-followups",
			WaitFollowupsReason:   "operational-closure-issues-replan-followups-waiting",
			WaitAgentsBlocker:     "wait-subagents-operational-closure-replan-followup-agents",
			WaitAgentsReason:      "operational-closure-issues-replan-followup-agents-waiting",
			EvidenceRefs: []string{
				"evidence-ref-app-director-operational-plan-state-closure-issues-replan-v0",
			},
		},
	)
}

func operationalDirectorPlanEmitBlockedClosureIssuesReplanDecisionV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
	issueRefs []string,
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
	refs := operationalDirectorClosureIssueAutoReplanRefsV0(request, match, issueRefs)
	replanRefs, err := operationalDirectorPlanEmitClosureIssueReplanDecisionV0(ctx, request, ports, run, match, issueRefs)
	if err != nil {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, err
	}
	if !startAppDirectorStringInSetV0(replanRefs, refs.GateRef) ||
		!startAppDirectorStringInSetV0(replanRefs, refs.ReplanRef) {
		return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, "", false, nil
	}
	return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
		ReplanRef:      refs.ReplanRef,
		RunRef:         request.RunRef,
		TaskRef:        taskRef,
		SourceRef:      refs.GateRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		FollowupRefs:   []string{refs.CapacityRef, refs.AgentRef},
		Summary:        operationalDirectorClosureIssueReplanSummaryV0(issueRefs),
		EvidenceRefs:   compactServiceRefsV0(append([]string{refs.GateRef}, issueRefs...)),
	}, refs.GateRef, true, nil
}

func continueOperationalDirectorPlanStateCanProgressBlockedClosureIssuesReplanV0(
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
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		activeStep.Reason != "operational-closure-issues" {
		return false, nil
	}
	issueRefs := operationalDirectorClosureReplannableBlockerRefsV0(append(state.BlockerRefs, activeStep.BlockerRefs...))
	if len(issueRefs) == 0 {
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
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return false, nil
	}
	events, err := loadOperationalRunEventsV0(ctx, reader, request.RunRef)
	if err != nil {
		return false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	matchedReplans := 0
	for _, match := range matches {
		gate, ok := operationalDirectorPlanRequiredTestsQualityGateV0(run, trace, activeStep, match.TaskRef, issueRefs)
		if !ok {
			if operationalDirectorClosureIssueReplanReflectedV0(
				run,
				operationalDirectorClosureIssueAutoReplanRefsV0(request, match, issueRefs),
				match,
			) {
				matchedReplans++
			}
			continue
		}
		if _, ok := operationalDirectorPlanReplanForQualityGateV0(run, trace, gate.GateRef, match.TaskRef); ok {
			matchedReplans++
		}
	}
	return matchedReplans == 1, nil
}
