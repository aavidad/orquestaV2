package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type codexStackReviewGateReworkProjectionV0 struct {
	ReworkRequestRef string
	ReviewResultRef  string
	ReviewRequestID  string
	DeliveryRef      string
}

type codexStackReviewGateReplanProjectionV0 struct {
	ReplanRef      string
	SourceRef      string
	TaskRef        string
	AcceptedAction string
	FollowupRefs   []string
}

func codexStackReviewGateReworkAcceptanceObservationsV0(
	request orquestacionnucleoapp.ReviewGateObservationRequestV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []orquestacionnucleoapp.ReviewGateObservationV0 {
	taskByDelivery, deliveryByTask := codexStackReviewGateDescriptorTaskMapsV0(descriptors)
	observations := make([]orquestacionnucleoapp.ReviewGateObservationV0, 0)
	for _, rawRework := range request.Run.ReworkRequests {
		rework, ok := codexStackReviewGateParseReworkProjectionV0(rawRework)
		if !ok || codexStackReviewGateReworkAcceptanceAlreadyDoneV0(request.Run, rework.DeliveryRef) {
			continue
		}
		parentTaskRef := taskByDelivery[rework.DeliveryRef]
		if parentTaskRef == "" {
			continue
		}
		followups, replanRefs := codexStackReviewGateFollowupsForReworkV0(request.Run, rework, parentTaskRef)
		if len(followups) == 0 || !codexStackReviewGateFollowupsAcceptedV0(request.Run, deliveryByTask, followups) {
			continue
		}
		observations = append(observations, codexStackReviewGateReworkAcceptanceObservationV0(
			rework,
			followups,
			replanRefs,
			deliveryByTask,
		))
	}
	return observations
}

func codexStackReviewGateDescriptorTaskMapsV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (map[string]string, map[string]string) {
	taskByDelivery := map[string]string{}
	deliveryByTask := map[string]string{}
	for _, descriptor := range descriptors {
		taskRef := strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef)
		deliveryRef := strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
		if taskRef == "" || deliveryRef == "" {
			continue
		}
		taskByDelivery[deliveryRef] = taskRef
		deliveryByTask[taskRef] = deliveryRef
	}
	return taskByDelivery, deliveryByTask
}

func codexStackReviewGateFollowupsForReworkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	rework codexStackReviewGateReworkProjectionV0,
	parentTaskRef string,
) ([]string, []string) {
	followups := []string{}
	replanRefs := []string{}
	for _, rawReplan := range run.ReplanDecisions {
		replan, ok := codexStackReviewGateParseReplanProjectionV0(rawReplan)
		if !ok ||
			replan.SourceRef != rework.ReworkRequestRef ||
			replan.TaskRef != parentTaskRef ||
			replan.AcceptedAction != string(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0) {
			continue
		}
		followups = append(followups, replan.FollowupRefs...)
		replanRefs = append(replanRefs, replan.ReplanRef)
	}
	return compactStringsV0(followups), compactStringsV0(replanRefs)
}

func codexStackReviewGateFollowupsAcceptedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryByTask map[string]string,
	followups []string,
) bool {
	for _, followupRef := range compactStringsV0(followups) {
		deliveryRef := strings.TrimSpace(deliveryByTask[followupRef])
		if deliveryRef == "" ||
			!codexStackStringInSetV0(run.ClosedTasks, followupRef) ||
			!codexStackStringInSetV0(run.Deliveries, deliveryRef) ||
			!codexStackStringInSetV0(run.AcceptedReviews, "accepted-review-ref-"+deliveryRef) {
			return false
		}
	}
	return len(compactStringsV0(followups)) > 0
}

func codexStackReviewGateReworkAcceptanceObservationV0(
	rework codexStackReviewGateReworkProjectionV0,
	followups []string,
	replanRefs []string,
	deliveryByTask map[string]string,
) orquestacionnucleoapp.ReviewGateObservationV0 {
	deliveryRef := strings.TrimSpace(rework.DeliveryRef)
	safe := codexStackOperationalClosureSafeRefV0(deliveryRef)
	evidenceRefs := []string{
		deliveryRef,
		rework.ReworkRequestRef,
		rework.ReviewResultRef,
		rework.ReviewRequestID,
		"evidence-ref-review-gate-rework-accepted",
	}
	evidenceRefs = append(evidenceRefs, replanRefs...)
	for _, followupRef := range compactStringsV0(followups) {
		evidenceRefs = append(evidenceRefs, followupRef, deliveryByTask[followupRef], "accepted-review-ref-"+deliveryByTask[followupRef])
	}
	return orquestacionnucleoapp.ReviewGateObservationV0{
		CandidateRef:      "review-gate-candidate-ref-rework-accepted-" + safe,
		ReviewRequestID:   rework.ReviewRequestID,
		ReviewResultRef:   "review-result-ref-rework-accepted-" + safe,
		AcceptedReviewRef: "accepted-review-ref-rework-accepted-" + safe,
		DeliveryRef:       deliveryRef,
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		Status:            orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		Summary:           "Entrega aceptada tras rework causal cerrado.",
		QualityGateRef:    "quality-gate-ref-rework-accepted-" + safe,
		EvidenceRefs:      compactStringsV0(evidenceRefs),
	}
}

func codexStackReviewGateReworkAcceptanceAlreadyDoneV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) bool {
	safe := codexStackOperationalClosureSafeRefV0(deliveryRef)
	return codexStackStringInSetV0(run.AcceptedReviews, "accepted-review-ref-rework-accepted-"+safe)
}

func codexStackReviewGateParseReworkProjectionV0(value string) (codexStackReviewGateReworkProjectionV0, bool) {
	reworkRef, tail, ok := strings.Cut(strings.TrimSpace(value), "#review_result:")
	if !ok {
		return codexStackReviewGateReworkProjectionV0{}, false
	}
	resultRef, tail, ok := strings.Cut(tail, "#review_request:")
	if !ok {
		return codexStackReviewGateReworkProjectionV0{}, false
	}
	reviewRequestID, deliveryRef, ok := strings.Cut(tail, "#delivery:")
	if !ok {
		return codexStackReviewGateReworkProjectionV0{}, false
	}
	out := codexStackReviewGateReworkProjectionV0{
		ReworkRequestRef: strings.TrimSpace(reworkRef),
		ReviewResultRef:  strings.TrimSpace(resultRef),
		ReviewRequestID:  strings.TrimSpace(reviewRequestID),
		DeliveryRef:      strings.TrimSpace(deliveryRef),
	}
	return out, out.ReworkRequestRef != "" && out.ReviewResultRef != "" &&
		out.ReviewRequestID != "" && out.DeliveryRef != ""
}

func codexStackReviewGateParseReplanProjectionV0(value string) (codexStackReviewGateReplanProjectionV0, bool) {
	replanRef, tail, ok := strings.Cut(strings.TrimSpace(value), "#source:")
	if !ok {
		return codexStackReviewGateReplanProjectionV0{}, false
	}
	sourceRef, tail, ok := strings.Cut(tail, "#task:")
	if !ok {
		return codexStackReviewGateReplanProjectionV0{}, false
	}
	taskRef, tail, ok := strings.Cut(tail, "#action:")
	if !ok {
		return codexStackReviewGateReplanProjectionV0{}, false
	}
	action, joinedFollowups, ok := strings.Cut(tail, "#followups:")
	if !ok {
		return codexStackReviewGateReplanProjectionV0{}, false
	}
	out := codexStackReviewGateReplanProjectionV0{
		ReplanRef:      strings.TrimSpace(replanRef),
		SourceRef:      strings.TrimSpace(sourceRef),
		TaskRef:        strings.TrimSpace(taskRef),
		AcceptedAction: strings.TrimSpace(action),
		FollowupRefs:   compactStringsV0(strings.Split(strings.TrimSpace(joinedFollowups), "+")),
	}
	return out, out.ReplanRef != "" && out.SourceRef != "" && out.TaskRef != "" &&
		out.AcceptedAction != "" && len(out.FollowupRefs) > 0
}
