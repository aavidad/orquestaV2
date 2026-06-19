package orquestacionnucleoapp

import (
	"encoding/json"
	"sort"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	operationalDirectorClosureReplanProjectionSourceSeparatorV0    = "#source:"
	operationalDirectorClosureReplanProjectionTaskSeparatorV0      = "#task:"
	operationalDirectorClosureReplanProjectionActionSeparatorV0    = "#action:"
	operationalDirectorClosureReplanProjectionFollowupsSeparatorV0 = "#followups:"
	operationalDirectorClosureReplanProjectionFollowupsJoinerV0    = "+"
)

type operationalDirectorClosureTraceV0 struct {
	Deliveries            map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0
	ReviewRequests        map[string]orquestacoreworkflow.ReviewRequestedPayloadV0
	ReviewResults         map[string]orquestacoreworkflow.ReviewResultV0
	ReviewResultRefs      []string
	AcceptedReviews       map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0
	ReplanFollowupsByTask map[string][]string
}

type operationalDirectorClosureReplanProjectionPartsV0 struct {
	TaskRef        string
	AcceptedAction orquestacoreworkflow.ReplanDecisionActionV0
	FollowupRefs   []string
}

func operationalDirectorClosureCausalIssuesV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	request OperationalDirectorClosureRequestV0,
) []ErrorV0 {
	trace := operationalDirectorClosureTraceFromEventsV0(events)
	issues := make([]ErrorV0, 0)
	delivery, ok := trace.Deliveries[request.DeliveryRef]
	if !ok {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_ref", "entrega no encontrada en eventos"))
	} else if !operationalDirectorClosureDeliveryMatchesTaskOrDescendantV0(delivery, tasks, trace, request.TaskID) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_ref", "entrega no corresponde a la microtarea"))
	}
	accepted, ok := trace.AcceptedReviews[request.AcceptedReviewRef]
	if !ok {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "accepted_review_ref", "review aceptada no encontrada en eventos"))
		return issues
	}
	if strings.TrimSpace(accepted.DeliveryRef) != request.DeliveryRef {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "accepted_review_ref", "review aceptada no corresponde a la entrega"))
	}
	reviewRequest, ok := trace.ReviewRequests[strings.TrimSpace(accepted.ReviewRequestID)]
	if !ok {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "review_request_id", "solicitud de review no encontrada"))
	} else if strings.TrimSpace(reviewRequest.DeliveryRef) != request.DeliveryRef {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "review_request_id", "solicitud de review no corresponde a la entrega"))
	}
	if !operationalDirectorClosureHasAcceptedResultV0(trace, accepted, request.DeliveryRef) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "review_result_ref", "resultado aceptado no encontrado para la entrega"))
	}
	return issues
}

func operationalDirectorClosureDeliveryMatchesTaskOrDescendantV0(
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	trace operationalDirectorClosureTraceV0,
	taskID string,
) bool {
	deliveryTaskID := strings.TrimSpace(delivery.TaskID)
	taskID = strings.TrimSpace(taskID)
	if deliveryTaskID == "" || taskID == "" {
		return false
	}
	if deliveryTaskID == taskID {
		return true
	}
	childrenByParent := make(map[string][]string, len(tasks))
	for _, task := range tasks {
		parentRef := strings.TrimSpace(task.ParentTaskRef)
		childRef := strings.TrimSpace(task.TaskID)
		if parentRef != "" && childRef != "" {
			childrenByParent[parentRef] = append(childrenByParent[parentRef], childRef)
		}
		for _, childRef := range task.ChildTaskRefs {
			childRef = strings.TrimSpace(childRef)
			if childRef != "" && strings.TrimSpace(task.TaskID) != "" {
				childrenByParent[strings.TrimSpace(task.TaskID)] = append(childrenByParent[strings.TrimSpace(task.TaskID)], childRef)
			}
		}
	}
	for parentRef, followupRefs := range trace.ReplanFollowupsByTask {
		parentRef = strings.TrimSpace(parentRef)
		if parentRef == "" {
			continue
		}
		childrenByParent[parentRef] = append(childrenByParent[parentRef], compactStringsV0(followupRefs)...)
	}
	queue := append([]string(nil), childrenByParent[taskID]...)
	seen := map[string]bool{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		if current == deliveryTaskID {
			return true
		}
		queue = append(queue, childrenByParent[current]...)
	}
	return false
}

func operationalDirectorClosureTraceFromEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) operationalDirectorClosureTraceV0 {
	trace := newOperationalDirectorClosureTraceV0()
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0:
			var payload orquestacoreworkflow.DeliveryRegisteredPayloadV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				trace.Deliveries[strings.TrimSpace(payload.DeliveryRef)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewRequestedV0:
			var payload orquestacoreworkflow.ReviewRequestedPayloadV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				trace.ReviewRequests[strings.TrimSpace(payload.ReviewRequestID)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0:
			var payload orquestacoreworkflow.ReviewResultV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				resultRef := strings.TrimSpace(payload.ReviewResultRef)
				trace.ReviewResults[resultRef] = payload
				trace.ReviewResultRefs = append(trace.ReviewResultRefs, resultRef)
			}
		case orquestacoreworkflow.OrchestrationEventReviewAcceptedV0:
			var payload orquestacoreworkflow.ReviewAcceptedPayloadV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				trace.AcceptedReviews[strings.TrimSpace(payload.AcceptedReviewRef)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			var payload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				operationalDirectorClosureAppendSplitTaskFollowupsV0(trace, payload.TaskRef, payload.AcceptedAction, payload.FollowupRefs)
			}
		}
	}
	return trace
}

func operationalDirectorClosureTraceFromRunProjectionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) operationalDirectorClosureTraceV0 {
	trace := newOperationalDirectorClosureTraceV0()
	for _, projection := range compactStringsV0(run.ReplanDecisions) {
		parts, ok := operationalDirectorClosureReplanProjectionPartsFromRefV0(projection)
		if !ok {
			continue
		}
		operationalDirectorClosureAppendSplitTaskFollowupsV0(trace, parts.TaskRef, parts.AcceptedAction, parts.FollowupRefs)
	}
	return trace
}

func newOperationalDirectorClosureTraceV0() operationalDirectorClosureTraceV0 {
	return operationalDirectorClosureTraceV0{
		Deliveries:            map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0{},
		ReviewRequests:        map[string]orquestacoreworkflow.ReviewRequestedPayloadV0{},
		ReviewResults:         map[string]orquestacoreworkflow.ReviewResultV0{},
		ReviewResultRefs:      []string{},
		AcceptedReviews:       map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0{},
		ReplanFollowupsByTask: map[string][]string{},
	}
}

func operationalDirectorClosureAppendSplitTaskFollowupsV0(
	trace operationalDirectorClosureTraceV0,
	taskRef string,
	action orquestacoreworkflow.ReplanDecisionActionV0,
	followupRefs []string,
) {
	if action != orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 {
		return
	}
	taskRef = strings.TrimSpace(taskRef)
	if taskRef == "" {
		return
	}
	trace.ReplanFollowupsByTask[taskRef] = append(trace.ReplanFollowupsByTask[taskRef], compactStringsV0(followupRefs)...)
}

func operationalDirectorClosureReplanProjectionPartsFromRefV0(
	ref string,
) (operationalDirectorClosureReplanProjectionPartsV0, bool) {
	_, tail, ok := strings.Cut(strings.TrimSpace(ref), operationalDirectorClosureReplanProjectionSourceSeparatorV0)
	if !ok {
		return operationalDirectorClosureReplanProjectionPartsV0{}, false
	}
	_, tail, ok = strings.Cut(tail, operationalDirectorClosureReplanProjectionTaskSeparatorV0)
	if !ok {
		return operationalDirectorClosureReplanProjectionPartsV0{}, false
	}
	taskRef, tail, ok := strings.Cut(tail, operationalDirectorClosureReplanProjectionActionSeparatorV0)
	if !ok {
		return operationalDirectorClosureReplanProjectionPartsV0{}, false
	}
	action, followups, ok := strings.Cut(tail, operationalDirectorClosureReplanProjectionFollowupsSeparatorV0)
	if !ok {
		return operationalDirectorClosureReplanProjectionPartsV0{}, false
	}
	parts := operationalDirectorClosureReplanProjectionPartsV0{
		TaskRef:        strings.TrimSpace(taskRef),
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionV0(strings.TrimSpace(action)),
		FollowupRefs:   compactStringsV0(strings.Split(strings.TrimSpace(followups), operationalDirectorClosureReplanProjectionFollowupsJoinerV0)),
	}
	return parts, parts.TaskRef != "" && len(parts.FollowupRefs) > 0
}

func operationalDirectorDecodeEventPayloadV0(event orquestacoreworkflow.OrchestrationEventV0, out any) bool {
	return json.Unmarshal(event.Payload, out) == nil
}

func operationalDirectorClosureHasAcceptedResultV0(
	trace operationalDirectorClosureTraceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	deliveryRef string,
) bool {
	_, ok := operationalDirectorClosureAcceptedResultV0(trace, accepted, deliveryRef)
	return ok
}

func operationalDirectorClosureAcceptedResultV0(
	trace operationalDirectorClosureTraceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	reviewRequestID := strings.TrimSpace(accepted.ReviewRequestID)
	deliveryRef = strings.TrimSpace(deliveryRef)
	acceptedEvidence := operationalDirectorClosureRefSetV0(accepted.EvidenceRefs)
	var fallback orquestacoreworkflow.ReviewResultV0
	haveFallback := false
	for _, resultRef := range operationalDirectorClosureReviewResultRefsV0(trace) {
		result, ok := trace.ReviewResults[resultRef]
		if !ok {
			continue
		}
		if strings.TrimSpace(result.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(result.DeliveryRef) == deliveryRef &&
			result.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
			if operationalDirectorClosureReviewResultMatchesAcceptedV0(result, acceptedEvidence) {
				return result, true
			}
			if !haveFallback {
				fallback = result
				haveFallback = true
			}
		}
	}
	if haveFallback {
		return fallback, true
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorClosureReviewResultRefsV0(trace operationalDirectorClosureTraceV0) []string {
	refs := compactStringsV0(trace.ReviewResultRefs)
	if len(refs) > 0 {
		return refs
	}
	refs = make([]string, 0, len(trace.ReviewResults))
	for ref := range trace.ReviewResults {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

func operationalDirectorClosureReviewResultMatchesAcceptedV0(
	result orquestacoreworkflow.ReviewResultV0,
	acceptedEvidence map[string]bool,
) bool {
	if len(acceptedEvidence) == 0 {
		return false
	}
	if acceptedEvidence[strings.TrimSpace(result.ReviewResultRef)] {
		return true
	}
	for _, evidenceRef := range result.EvidenceRefs {
		if acceptedEvidence[strings.TrimSpace(evidenceRef)] {
			return true
		}
	}
	return false
}

func operationalDirectorClosureRefSetV0(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range compactStringsV0(values) {
		out[value] = true
	}
	return out
}
