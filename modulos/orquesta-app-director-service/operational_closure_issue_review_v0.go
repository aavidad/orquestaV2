package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func operationalDirectorClosureIssueAcceptedReviewMatchV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	closureRequest orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) (operationalDirectorPlanAcceptedReviewMatchV0, operationalDirectorPlanReviewTraceV0, bool, error) {
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, operationalDirectorPlanReviewTraceV0{}, false, nil
	}
	events, err := loadOperationalRunEventsV0(ctx, reader, request.RunRef)
	if err != nil {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, operationalDirectorPlanReviewTraceV0{}, false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	taskRef := strings.TrimSpace(closureRequest.TaskID)
	deliveryRef := strings.TrimSpace(closureRequest.DeliveryRef)
	acceptedReviewRef := strings.TrimSpace(closureRequest.AcceptedReviewRef)
	if taskRef == "" || deliveryRef == "" || acceptedReviewRef == "" {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	if len(activeStep.TaskRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.TaskRefs, taskRef) {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	if len(activeStep.DeliveryRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.DeliveryRefs, deliveryRef) {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	delivery := trace.Deliveries[deliveryRef]
	if strings.TrimSpace(delivery.TaskID) != taskRef ||
		!startAppDirectorStringInSetV0(run.Deliveries, deliveryRef) ||
		!startAppDirectorStringInSetV0(run.AcceptedReviews, acceptedReviewRef) {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	if agentRef := strings.TrimSpace(delivery.AgentRef); agentRef != "" &&
		len(activeStep.AgentRefs) > 0 &&
		!startAppDirectorStringInSetV0(activeStep.AgentRefs, agentRef) {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	accepted := trace.AcceptedReviews[acceptedReviewRef]
	reviewRequestID := strings.TrimSpace(accepted.ReviewRequestID)
	if reviewRequestID == "" {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	reviewRequest := trace.ReviewRequests[reviewRequestID]
	if strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	reviewResult, ok := operationalDirectorClosureIssueAcceptedReviewResultV0(trace, reviewRequestID, deliveryRef)
	if !ok {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	return operationalDirectorPlanAcceptedReviewMatchV0{
		TaskRef:           taskRef,
		AgentRef:          strings.TrimSpace(delivery.AgentRef),
		DeliveryRef:       deliveryRef,
		ReviewRequestID:   reviewRequestID,
		ReviewResultRef:   strings.TrimSpace(reviewResult.ReviewResultRef),
		AcceptedReviewRef: acceptedReviewRef,
		EvidenceRefs: compactServiceRefsV0(append(append(append(
			append([]string(nil), delivery.EvidenceRefs...),
			reviewRequest.EvidenceRefs...),
			reviewResult.EvidenceRefs...),
			acceptedReviewRef)),
	}, trace, true, nil
}

func operationalDirectorClosureIssueAcceptedReviewResultV0(
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequestID string,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	for _, result := range trace.ReviewResults {
		if strings.TrimSpace(result.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(result.DeliveryRef) == deliveryRef &&
			result.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
			return result, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorPlanEmitClosureIssueReplanDecisionV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	issueRefs []string,
) ([]string, error) {
	refs := operationalDirectorClosureIssueAutoReplanRefsV0(request, match, issueRefs)
	storedRun, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(storedRun.RunID) != strings.TrimSpace(run.RunID) {
		return nil, nil
	}
	if operationalDirectorClosureIssueReplanReflectedV0(storedRun, refs, match) {
		return compactServiceRefsV0([]string{refs.GateRef, refs.ReplanRef, refs.CapacityRef, refs.AgentRef}), nil
	}
	storedRun, ready, err := operationalDirectorPlanEnsureProgrammingPhaseForRequiredTestsReplanV0(ctx, request, ports, storedRun, refs)
	if err != nil || !ready || !operationalDirectorRunProgrammingPhaseActiveV0(storedRun) {
		return nil, err
	}
	gatePayload := orquestacoreworkflow.RecordQualityGateCommandPayloadV0{
		RunRef:       request.RunRef,
		GateRef:      refs.GateRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		SubjectRef:   match.TaskRef,
		Decision:     orquestacoreworkflow.QualityGateDecisionBlockedV0,
		IssueRefs:    append([]string(nil), issueRefs...),
		Summary:      operationalDirectorClosureIssueGateSummaryV0(issueRefs),
		EvidenceRefs: operationalDirectorRequiredTestsAutoReplanEvidenceRefsV0(match, issueRefs),
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
		return nil, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, gateCommand); err != nil {
		return nil, err
	}
	replanPayload := orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
		ReplanRef:      refs.ReplanRef,
		RunRef:         request.RunRef,
		TaskRef:        match.TaskRef,
		SourceRef:      refs.GateRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		FollowupRefs:   []string{refs.CapacityRef, refs.AgentRef},
		Summary:        operationalDirectorClosureIssueReplanSummaryV0(issueRefs),
		EvidenceRefs:   compactServiceRefsV0(append([]string{refs.GateRef}, issueRefs...)),
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
		return nil, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, replanCommand); err != nil {
		return nil, err
	}
	return compactServiceRefsV0([]string{refs.GateRef, refs.ReplanRef, refs.CapacityRef, refs.AgentRef}), nil
}
