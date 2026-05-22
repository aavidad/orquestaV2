package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type codexStackOperationalClosureSourceV0 struct {
	TaskStore                 orquestacionnucleoapp.WorkflowTaskStorePortV0
	EventReader               orquestacionnucleoapp.RunEventReaderPortV0
	RequiredTestEvidenceStore orquestacionnucleoapp.RequiredTestEvidenceReaderPortV0
}

var _ orquestaappdirectorservice.AppDirectorOperationalClosureSourcePortV0 = codexStackOperationalClosureSourceV0{}

func operationalClosureSourceV0(
	config ConfigV0,
) orquestaappdirectorservice.AppDirectorOperationalClosureSourcePortV0 {
	reader, _ := config.Stores.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	if config.Stores.TaskStore == nil || reader == nil {
		return nil
	}
	return codexStackOperationalClosureSourceV0{
		TaskStore:                 config.Stores.TaskStore,
		EventReader:               reader,
		RequiredTestEvidenceStore: requiredTestEvidenceStoreV0(config),
	}
}

func eventReaderV0(config ConfigV0) orquestacionnucleoapp.RunEventReaderPortV0 {
	reader, _ := config.Stores.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	return reader
}

func (source codexStackOperationalClosureSourceV0) BuildOperationalDirectorClosureRequestV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	if source.TaskStore == nil || source.EventReader == nil ||
		strings.TrimSpace(request.Run.RunID) == "" ||
		request.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(request.Run.Tasks) == 0 {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
	}
	tasks, err := source.TaskStore.LoadWorkflowTasksV0(ctx, request.Run.RunID, request.Run.Tasks)
	if err != nil {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
	}
	events, err := source.EventReader.LoadRunEventsV0(ctx, request.Run.RunID)
	if err != nil {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
	}
	trace := codexStackOperationalClosureTraceFromEventsV0(events)
	for _, task := range codexStackOperationalClosureCandidateTasksV0(request, tasks) {
		closureRequest, ok, err := source.codexStackOperationalClosureRequestForTaskV0(ctx, request, trace, task)
		if err != nil {
			return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
		}
		if ok {
			return closureRequest, true, nil
		}
	}
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
}

type codexStackOperationalClosureTraceV0 struct {
	Deliveries      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0
	ReviewRequests  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0
	ReviewResults   map[string]orquestacoreworkflow.ReviewResultV0
	AcceptedReviews map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0
}

func codexStackOperationalClosureCandidateTasksV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []orquestacoreworkflow.WorkflowTaskV0 {
	scopeAgents := codexStackOperationalClosureSetV0(request.WaitAgentRefs)
	if request.WaitScopeApplied && len(scopeAgents) == 0 {
		return nil
	}
	open := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	closed := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		if !codexStackOperationalClosureTaskIsOperationalDirectorV0(task) {
			continue
		}
		if !codexStackOperationalClosureChildrenClosedV0(request, task, tasks) {
			continue
		}
		if len(scopeAgents) > 0 && !scopeAgents[orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID)] {
			continue
		}
		if codexStackOperationalClosureContainsV0(request.Run.ClosedTasks, task.TaskID) {
			closed = append(closed, task)
			continue
		}
		open = append(open, task)
	}
	if len(open) > 0 {
		return open
	}
	return closed
}

func codexStackOperationalClosureTaskIsOperationalDirectorV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	for _, criterion := range task.AcceptanceCriteria {
		if strings.Contains(strings.TrimSpace(criterion), "operational_director.") {
			return true
		}
	}
	for _, ref := range task.FunctionContractRefs {
		if strings.Contains(strings.TrimSpace(ref.ContractRef), "operational-director") ||
			strings.Contains(strings.TrimSpace(ref.FunctionName), "OperationalDirector") {
			return true
		}
	}
	return false
}

func codexStackOperationalClosureChildrenClosedV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) bool {
	childRefs := codexStackOperationalClosureCompactRefsV0(task.ChildTaskRefs)
	if len(childRefs) == 0 {
		return true
	}
	tasksByRef := make(map[string]bool, len(tasks))
	for _, item := range tasks {
		tasksByRef[strings.TrimSpace(item.TaskID)] = true
	}
	for _, childRef := range childRefs {
		if !tasksByRef[childRef] || !codexStackOperationalClosureContainsV0(request.Run.ClosedTasks, childRef) {
			return false
		}
	}
	return true
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureRequestForTaskV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	trace codexStackOperationalClosureTraceV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	for _, delivery := range codexStackOperationalClosureDeliveriesForTaskV0(request, trace, task) {
		reviewRequest, accepted, result, ok := codexStackOperationalClosureAcceptedReviewForDeliveryV0(request, trace, delivery.DeliveryRef)
		if !ok {
			continue
		}
		requiredTestEvidenceRefs := []string(nil)
		if len(task.RequiredTests) > 0 {
			if source.RequiredTestEvidenceStore == nil {
				continue
			}
			evidence, err := source.codexStackOperationalClosureLoadRequiredTestEvidenceForResultV0(ctx, request, result)
			if err != nil {
				return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
			}
			requiredTestEvidenceRefs = codexStackOperationalClosurePassedTestEvidenceRefsV0(task, evidence, accepted, result)
			if len(requiredTestEvidenceRefs) == 0 {
				continue
			}
		}
		evidenceRefs := codexStackOperationalClosureEvidenceRefsV0(request, delivery, reviewRequest, accepted, result)
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			RunRef:                   request.Run.RunID,
			TaskID:                   task.TaskID,
			DeliveryRef:              delivery.DeliveryRef,
			AcceptedReviewRef:        accepted.AcceptedReviewRef,
			ValidationRef:            "validation-ref-operational-director-" + codexStackOperationalClosureSafeRefV0(task.TaskID),
			ClosureRef:               "closure-ref-operational-director-" + codexStackOperationalClosureSafeRefV0(request.Run.RunID),
			OccurredAt:               request.OccurredAt,
			CorrelationID:            request.CorrelationID,
			RequestedBy:              request.RequestedBy,
			Summary:                  "Cierre causal generado desde el stack de Orquesta.",
			RequiredTestEvidenceRefs: requiredTestEvidenceRefs,
			EvidenceRefs:             evidenceRefs,
		}, true, nil
	}
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureLoadRequiredTestEvidenceForResultV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	result orquestacoreworkflow.ReviewResultV0,
) ([]orquestacionnucleoapp.RequiredTestEvidenceV0, error) {
	candidateRefs := append([]string(nil), request.RequiredTestEvidenceRefs...)
	candidateRefs = append(candidateRefs, result.EvidenceRefs...)
	evidence := make([]orquestacionnucleoapp.RequiredTestEvidenceV0, 0, len(candidateRefs))
	for _, evidenceRef := range codexStackOperationalClosureCompactRefsV0(candidateRefs) {
		items, err := source.RequiredTestEvidenceStore.LoadRequiredTestEvidenceV0(ctx, request.Run.RunID, []string{evidenceRef})
		if err != nil {
			if codexStackOperationalClosureMissingTestEvidenceV0(err) {
				continue
			}
			return nil, err
		}
		evidence = append(evidence, items...)
	}
	return evidence, nil
}

func codexStackOperationalClosureMissingTestEvidenceV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "required_test_evidence"
}

func codexStackOperationalClosureDeliveriesForTaskV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	trace codexStackOperationalClosureTraceV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) []orquestacoreworkflow.DeliveryRegisteredPayloadV0 {
	deliveries := make([]orquestacoreworkflow.DeliveryRegisteredPayloadV0, 0)
	for _, deliveryRef := range request.Run.Deliveries {
		delivery, ok := trace.Deliveries[strings.TrimSpace(deliveryRef)]
		if !ok {
			continue
		}
		if strings.TrimSpace(delivery.TaskID) != strings.TrimSpace(task.TaskID) ||
			!codexStackOperationalClosureContainsV0(request.Run.Deliveries, delivery.DeliveryRef) {
			continue
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries
}

func codexStackOperationalClosureAcceptedReviewForDeliveryV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	trace codexStackOperationalClosureTraceV0,
	deliveryRef string,
) (
	orquestacoreworkflow.ReviewRequestedPayloadV0,
	orquestacoreworkflow.ReviewAcceptedPayloadV0,
	orquestacoreworkflow.ReviewResultV0,
	bool,
) {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, acceptedReviewRef := range request.Run.AcceptedReviews {
		accepted, ok := trace.AcceptedReviews[strings.TrimSpace(acceptedReviewRef)]
		if !ok {
			continue
		}
		if strings.TrimSpace(accepted.DeliveryRef) != deliveryRef ||
			!codexStackOperationalClosureContainsV0(request.Run.AcceptedReviews, accepted.AcceptedReviewRef) {
			continue
		}
		reviewRequest, ok := trace.ReviewRequests[strings.TrimSpace(accepted.ReviewRequestID)]
		if !ok || strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef {
			continue
		}
		result, ok := codexStackOperationalClosureAcceptedResultV0(trace, accepted)
		if ok {
			return reviewRequest, accepted, result, true
		}
	}
	return orquestacoreworkflow.ReviewRequestedPayloadV0{}, orquestacoreworkflow.ReviewAcceptedPayloadV0{}, orquestacoreworkflow.ReviewResultV0{}, false
}

func codexStackOperationalClosureAcceptedResultV0(
	trace codexStackOperationalClosureTraceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	for _, result := range trace.ReviewResults {
		if strings.TrimSpace(result.ReviewRequestID) == strings.TrimSpace(accepted.ReviewRequestID) &&
			strings.TrimSpace(result.DeliveryRef) == strings.TrimSpace(accepted.DeliveryRef) &&
			result.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
			return result, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func codexStackOperationalClosurePassedTestEvidenceRefsV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) []string {
	refs := make([]string, 0, len(task.RequiredTests))
	for _, required := range codexStackOperationalClosureCompactRefsV0(task.RequiredTests) {
		ref, ok := codexStackOperationalClosurePassedTestEvidenceRefV0(task, required, evidence, accepted, result)
		if !ok {
			return nil
		}
		refs = append(refs, ref)
	}
	return codexStackOperationalClosureCompactRefsV0(refs)
}

func codexStackOperationalClosurePassedTestEvidenceRefV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	required string,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) (string, bool) {
	for _, item := range evidence {
		if item.Status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 &&
			strings.TrimSpace(item.TaskRef) == strings.TrimSpace(task.TaskID) &&
			strings.TrimSpace(item.TestCommand) == strings.TrimSpace(required) &&
			strings.TrimSpace(item.DeliveryRef) == strings.TrimSpace(result.DeliveryRef) &&
			strings.TrimSpace(item.ReviewRequestID) == strings.TrimSpace(result.ReviewRequestID) &&
			strings.TrimSpace(item.ReviewResultRef) == strings.TrimSpace(result.ReviewResultRef) &&
			strings.TrimSpace(item.AcceptedReviewRef) == strings.TrimSpace(accepted.AcceptedReviewRef) {
			return item.EvidenceRef, true
		}
	}
	return "", false
}

func codexStackOperationalClosureEvidenceRefsV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) []string {
	refs := append([]string(nil), request.EvidenceRefs...)
	refs = append(refs, delivery.EvidenceRefs...)
	refs = append(refs, reviewRequest.EvidenceRefs...)
	refs = append(refs, result.EvidenceRefs...)
	refs = append(refs, accepted.EvidenceRefs...)
	refs = append(refs, "evidence-ref-operational-director-closure-source-"+codexStackOperationalClosureSafeRefV0(delivery.DeliveryRef))
	return codexStackOperationalClosureCompactRefsV0(refs)
}

func codexStackOperationalClosureTraceFromEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) codexStackOperationalClosureTraceV0 {
	trace := codexStackOperationalClosureTraceV0{
		Deliveries:      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0{},
		ReviewRequests:  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0{},
		ReviewResults:   map[string]orquestacoreworkflow.ReviewResultV0{},
		AcceptedReviews: map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0{},
	}
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0:
			var payload orquestacoreworkflow.DeliveryRegisteredPayloadV0
			if codexStackOperationalClosureDecodeEventPayloadV0(event, &payload) {
				trace.Deliveries[strings.TrimSpace(payload.DeliveryRef)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewRequestedV0:
			var payload orquestacoreworkflow.ReviewRequestedPayloadV0
			if codexStackOperationalClosureDecodeEventPayloadV0(event, &payload) {
				trace.ReviewRequests[strings.TrimSpace(payload.ReviewRequestID)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0:
			var payload orquestacoreworkflow.ReviewResultV0
			if codexStackOperationalClosureDecodeEventPayloadV0(event, &payload) {
				trace.ReviewResults[strings.TrimSpace(payload.ReviewResultRef)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewAcceptedV0:
			var payload orquestacoreworkflow.ReviewAcceptedPayloadV0
			if codexStackOperationalClosureDecodeEventPayloadV0(event, &payload) {
				trace.AcceptedReviews[strings.TrimSpace(payload.AcceptedReviewRef)] = payload
			}
		}
	}
	return trace
}

func codexStackOperationalClosureDecodeEventPayloadV0(event orquestacoreworkflow.OrchestrationEventV0, out any) bool {
	return json.Unmarshal(event.Payload, out) == nil
}

func codexStackOperationalClosureSetV0(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range codexStackOperationalClosureCompactRefsV0(values) {
		out[value] = true
	}
	return out
}

func codexStackOperationalClosureContainsV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func codexStackOperationalClosureCompactRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func codexStackOperationalClosureSafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
