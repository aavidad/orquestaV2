package orquestaappdirectorservice

import (
	"context"
	"errors"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func operationalDirectorPlanStateEventReaderV0(
	ports StartAppDirectorPortsV0,
) orquestacionnucleoapp.RunEventReaderPortV0 {
	if ports.EventReader != nil {
		return ports.EventReader
	}
	reader, _ := ports.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	return reader
}

func operationalDirectorPlanAcceptedReviewMatchForTaskV0(
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	taskRef string,
) (operationalDirectorPlanAcceptedReviewMatchV0, bool) {
	taskRef = strings.TrimSpace(taskRef)
	for _, deliveryRef := range trace.DeliveryRefs {
		delivery := trace.Deliveries[deliveryRef]
		if strings.TrimSpace(delivery.TaskID) != taskRef {
			continue
		}
		if len(activeStep.AgentRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.AgentRefs, delivery.AgentRef) {
			continue
		}
		if len(activeStep.DeliveryRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.DeliveryRefs, delivery.DeliveryRef) {
			continue
		}
		if !startAppDirectorStringInSetV0(run.Deliveries, delivery.DeliveryRef) ||
			!startAppDirectorStringInSetV0(run.DeliveredTasks, taskRef) {
			continue
		}
		if strings.TrimSpace(delivery.AgentRef) != "" &&
			!startAppDirectorStringInSetV0(run.DeliveredAgents, delivery.AgentRef) {
			continue
		}
		match, ok := operationalDirectorPlanAcceptedReviewMatchForDeliveryV0(run, trace, delivery)
		if !ok {
			continue
		}
		if len(activeStep.ReviewResultRefs) > 0 &&
			!startAppDirectorStringInSetV0(activeStep.ReviewResultRefs, match.ReviewResultRef) {
			continue
		}
		match.TaskRef = taskRef
		match.AgentRef = strings.TrimSpace(delivery.AgentRef)
		return match, true
	}
	return operationalDirectorPlanAcceptedReviewMatchV0{}, false
}

func operationalDirectorPlanAcceptedReviewMatchForDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
) (operationalDirectorPlanAcceptedReviewMatchV0, bool) {
	deliveryRef := strings.TrimSpace(delivery.DeliveryRef)
	for _, reviewRequestID := range trace.ReviewRequestRefs {
		reviewRequest := trace.ReviewRequests[reviewRequestID]
		if strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef ||
			!startAppDirectorStringInSetV0(run.Reviews, reviewRequest.ReviewRequestID) {
			continue
		}
		reviewResult, ok := operationalDirectorPlanAcceptedReviewResultForRequestV0(run, trace, reviewRequest, deliveryRef)
		if !ok {
			continue
		}
		acceptedReview, ok := operationalDirectorPlanAcceptedReviewForRequestV0(run, trace, reviewRequest, deliveryRef)
		if !ok {
			continue
		}
		return operationalDirectorPlanAcceptedReviewMatchV0{
			DeliveryRef:       deliveryRef,
			ReviewRequestID:   strings.TrimSpace(reviewRequest.ReviewRequestID),
			ReviewResultRef:   strings.TrimSpace(reviewResult.ReviewResultRef),
			AcceptedReviewRef: strings.TrimSpace(acceptedReview.AcceptedReviewRef),
			EvidenceRefs: compactServiceRefsV0(append(append(append(
				append([]string(nil), delivery.EvidenceRefs...),
				reviewRequest.EvidenceRefs...),
				reviewResult.EvidenceRefs...),
				acceptedReview.EvidenceRefs...)),
		}, true
	}
	return operationalDirectorPlanAcceptedReviewMatchV0{}, false
}

func operationalDirectorPlanNegativeReviewResultForRequestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	reviewRequestID := strings.TrimSpace(reviewRequest.ReviewRequestID)
	for _, reviewResultRef := range trace.ReviewResultRefs {
		reviewResult := trace.ReviewResults[reviewResultRef]
		status := orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(reviewResult.Status)))
		if strings.TrimSpace(reviewResult.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(reviewResult.DeliveryRef) == deliveryRef &&
			(status == orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
				status == orquestacoreworkflow.ReviewResultStatusRejectedV0) &&
			operationalDirectorPlanProjectionReflectedV0(reviewResult.ReviewResultRef, run.ReviewResults) {
			return reviewResult, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorPlanReworkForReviewResultV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewResult orquestacoreworkflow.ReviewResultV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReworkRequestedPayloadV0, bool) {
	for _, reworkRequestRef := range trace.ReworkRequestRefs {
		rework := trace.ReworkRequests[reworkRequestRef]
		if strings.TrimSpace(rework.ReviewResultRef) == strings.TrimSpace(reviewResult.ReviewResultRef) &&
			strings.TrimSpace(rework.ReviewRequestID) == strings.TrimSpace(reviewRequest.ReviewRequestID) &&
			strings.TrimSpace(rework.DeliveryRef) == strings.TrimSpace(deliveryRef) &&
			operationalDirectorPlanProjectionReflectedV0(rework.ReworkRequestRef, run.ReworkRequests) {
			return rework, true
		}
	}
	return orquestacoreworkflow.ReworkRequestedPayloadV0{}, false
}

func operationalDirectorPlanReplanForReworkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	rework orquestacoreworkflow.ReworkRequestedPayloadV0,
	taskRef string,
) (orquestacoreworkflow.ReplanDecisionRecordedPayloadV0, bool) {
	for _, replanDecisionRef := range trace.ReplanDecisionRefs {
		replan := trace.ReplanDecisions[replanDecisionRef]
		if strings.TrimSpace(replan.SourceRef) == strings.TrimSpace(rework.ReworkRequestRef) &&
			strings.TrimSpace(replan.TaskRef) == strings.TrimSpace(taskRef) &&
			strings.TrimSpace(replan.RunRef) == strings.TrimSpace(run.RunID) &&
			operationalDirectorPlanProjectionReflectedV0(replan.ReplanRef, run.ReplanDecisions) {
			return replan, true
		}
	}
	return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, false
}

func operationalDirectorPlanAcceptedReviewResultForRequestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	reviewRequestID := strings.TrimSpace(reviewRequest.ReviewRequestID)
	for _, reviewResultRef := range trace.ReviewResultRefs {
		reviewResult := trace.ReviewResults[reviewResultRef]
		if strings.TrimSpace(reviewResult.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(reviewResult.DeliveryRef) == deliveryRef &&
			orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(reviewResult.Status))) == orquestacoreworkflow.ReviewResultStatusAcceptedV0 &&
			operationalDirectorPlanProjectionReflectedV0(reviewResult.ReviewResultRef, run.ReviewResults) {
			return reviewResult, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorPlanAcceptedReviewForRequestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewAcceptedPayloadV0, bool) {
	reviewRequestID := strings.TrimSpace(reviewRequest.ReviewRequestID)
	for _, acceptedReviewRef := range trace.AcceptedReviewRefs {
		acceptedReview := trace.AcceptedReviews[acceptedReviewRef]
		if strings.TrimSpace(acceptedReview.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(acceptedReview.DeliveryRef) == deliveryRef &&
			startAppDirectorStringInSetV0(run.AcceptedReviews, acceptedReview.AcceptedReviewRef) {
			return acceptedReview, true
		}
	}
	return orquestacoreworkflow.ReviewAcceptedPayloadV0{}, false
}

func operationalDirectorPlanRequiredTestEvidenceForMatchesV0(
	ctx context.Context,
	runRef string,
	store orquestacionnucleoapp.RequiredTestEvidenceReaderPortV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) ([]orquestacionnucleoapp.RequiredTestEvidenceV0, error) {
	candidateRefs := append([]string(nil), activeStep.RequiredTestEvidenceRefs...)
	if len(candidateRefs) == 0 {
		for _, match := range matches {
			candidateRefs = append(candidateRefs, match.EvidenceRefs...)
		}
	}
	evidence := make([]orquestacionnucleoapp.RequiredTestEvidenceV0, 0, len(candidateRefs))
	for _, evidenceRef := range compactServiceRefsV0(candidateRefs) {
		items, err := store.LoadRequiredTestEvidenceV0(ctx, runRef, []string{evidenceRef})
		if err != nil {
			if operationalDirectorPlanMissingRequiredTestEvidenceV0(err) {
				continue
			}
			return nil, err
		}
		evidence = append(evidence, items...)
	}
	return evidence, nil
}

func operationalDirectorPlanMissingRequiredTestEvidenceV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "required_test_evidence"
}

func operationalDirectorPlanEvaluateRequiredTestEvidenceV0(
	runRef string,
	requiredTests []string,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
) operationalDirectorPlanRequiredTestEvidenceEvaluationV0 {
	passedRefs := []string(nil)
	failedRefs := []string(nil)
	for _, match := range matches {
		for _, required := range requiredTests {
			passedRef, failedRef := operationalDirectorPlanRequiredTestEvidenceRefV0(runRef, required, match, evidence)
			if passedRef != "" {
				passedRefs = append(passedRefs, passedRef)
				continue
			}
			if failedRef != "" {
				failedRefs = append(failedRefs, failedRef)
			}
		}
	}
	expected := len(compactServiceRefsV0(requiredTests)) * len(matches)
	passedRefs = compactServiceRefsV0(passedRefs)
	return operationalDirectorPlanRequiredTestEvidenceEvaluationV0{
		PassedRefs: passedRefs,
		FailedRefs: compactServiceRefsV0(failedRefs),
		Complete:   expected > 0 && len(passedRefs) == expected,
	}
}

func operationalDirectorPlanRequiredTestEvidenceRefV0(
	runRef string,
	required string,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
) (string, string) {
	passedRef := ""
	failedRef := ""
	for _, item := range evidence {
		if strings.TrimSpace(item.RunRef) != strings.TrimSpace(runRef) ||
			strings.TrimSpace(item.TaskRef) != strings.TrimSpace(match.TaskRef) ||
			strings.TrimSpace(item.TestCommand) != strings.TrimSpace(required) ||
			strings.TrimSpace(item.DeliveryRef) != strings.TrimSpace(match.DeliveryRef) ||
			strings.TrimSpace(item.ReviewRequestID) != strings.TrimSpace(match.ReviewRequestID) ||
			strings.TrimSpace(item.ReviewResultRef) != strings.TrimSpace(match.ReviewResultRef) ||
			strings.TrimSpace(item.AcceptedReviewRef) != strings.TrimSpace(match.AcceptedReviewRef) {
			continue
		}
		switch item.Status {
		case orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0:
			passedRef = strings.TrimSpace(item.EvidenceRef)
		case orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0:
			failedRef = strings.TrimSpace(item.EvidenceRef)
		}
	}
	return passedRef, failedRef
}
