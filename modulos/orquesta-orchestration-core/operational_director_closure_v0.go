package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type OperationalDirectorClosureV0 struct {
	RunStore                  RunStorePortV0
	EventSink                 EventSinkPortV0
	EventReader               RunEventReaderPortV0
	TaskStore                 WorkflowTaskStorePortV0
	RequiredTestEvidenceStore RequiredTestEvidenceReaderPortV0
	RequestedBy               string
}

type OperationalDirectorClosureRequestV0 struct {
	RunRef                   string
	TaskID                   string
	DeliveryRef              string
	AcceptedReviewRef        string
	ValidationRef            string
	ClosureRef               string
	OccurredAt               string
	CorrelationID            string
	RequestedBy              string
	Summary                  string
	RequiredTestEvidenceRefs []string
	EvidenceRefs             []string
}

type OperationalDirectorClosureResultV0 struct {
	Run         orquestacoreworkflow.OrchestrationRunV0
	Commands    []orquestacoreworkflow.OrchestrationCommandV0
	EventsCount int
	Issues      []ErrorV0
}

func (closer OperationalDirectorClosureV0) CloseOperationalDirectorRunV0(
	ctx context.Context,
	request OperationalDirectorClosureRequestV0,
) (OperationalDirectorClosureResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeOperationalDirectorClosureRequestV0(request)
	if request.RequestedBy == "" {
		request.RequestedBy = strings.TrimSpace(closer.RequestedBy)
	}
	if issues := closer.validateOperationalDirectorClosureV0(request); len(issues) > 0 {
		return OperationalDirectorClosureResultV0{Issues: issues}, nil
	}
	run, err := closer.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return OperationalDirectorClosureResultV0{}, err
	}
	result := OperationalDirectorClosureResultV0{Run: run}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		result.Issues = append(result.Issues, operationalDirectorClosedRunRequestIssuesV0(run, request)...)
		return result, nil
	}
	tasks, issues, err := closer.operationalDirectorClosureTasksV0(ctx, run, request)
	if err != nil || len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, err
	}
	if issues := operationalDirectorClosureReadinessIssuesV0(run, tasks, request); len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, nil
	}
	events, issues, err := closer.operationalDirectorClosureEventsV0(ctx, request)
	if err != nil || len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, err
	}
	if issues := operationalDirectorClosureCausalIssuesV0(events, request); len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, nil
	}
	trace := operationalDirectorClosureTraceFromEventsV0(events)
	if issues, err := closer.operationalDirectorClosureRequiredTestIssuesV0(ctx, tasks, trace, request); err != nil || len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, err
	}
	if !operationalDirectorClosureReflectedV0(run.ClosedTasks, request.TaskID) {
		if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorCloseTaskCommandV0); err != nil {
			return result, err
		}
	}
	run = result.Run
	if openTasks := operationalDirectorClosureOpenTasksV0(run); len(openTasks) > 0 {
		result.Issues = append(result.Issues, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"run.open_tasks",
			fmt.Sprintf("microtareas abiertas: %s", strings.Join(openTasks, ",")),
		))
		return result, nil
	}
	if !operationalDirectorClosureReflectedV0(run.Validations, request.ValidationRef) {
		if operationalDirectorClosurePhaseV0(run.CurrentPhase) != orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0 {
			if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorOpenFinalValidationCommandV0); err != nil {
				return result, err
			}
		}
		if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorRegisterFinalValidationCommandV0); err != nil {
			return result, err
		}
	}
	run = result.Run
	if operationalDirectorClosurePhaseV0(run.CurrentPhase) != orquestacoreworkflow.OrchestrationPhaseCierreV0 {
		if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorOpenClosureCommandV0); err != nil {
			return result, err
		}
	}
	if !operationalDirectorClosureReflectedV0(result.Run.Closures, request.ClosureRef) {
		if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorCloseRunCommandV0); err != nil {
			return result, err
		}
	}
	return result, nil
}

type operationalDirectorClosureCommandBuilderV0 func(
	OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error)

func (closer OperationalDirectorClosureV0) applyOperationalDirectorClosureCommandV0(
	ctx context.Context,
	request OperationalDirectorClosureRequestV0,
	result *OperationalDirectorClosureResultV0,
	build operationalDirectorClosureCommandBuilderV0,
) error {
	command, err := build(request)
	if err != nil {
		return err
	}
	commandResult, err := HandleStoredWorkflowCommandV0(ctx, closer.RunStore, closer.EventSink, command)
	if err != nil {
		return err
	}
	run, err := closer.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return err
	}
	result.Run = run
	result.Commands = append(result.Commands, command)
	result.EventsCount += len(commandResult.Events)
	return nil
}

func (closer OperationalDirectorClosureV0) operationalDirectorClosureTasksV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	request OperationalDirectorClosureRequestV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, []ErrorV0, error) {
	if closer.TaskStore == nil {
		return nil, []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "task_store", "task_store requerido")}, nil
	}
	tasks, err := closer.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
	if err != nil {
		return nil, nil, err
	}
	if !operationalDirectorClosureTaskLoadedV0(tasks, request.TaskID) {
		return tasks, []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "task_id", "microtarea no encontrada")}, nil
	}
	return tasks, nil, nil
}

func (closer OperationalDirectorClosureV0) operationalDirectorClosureEventsV0(
	ctx context.Context,
	request OperationalDirectorClosureRequestV0,
) ([]orquestacoreworkflow.OrchestrationEventV0, []ErrorV0, error) {
	reader := closer.EventReader
	if reader == nil {
		reader, _ = closer.EventSink.(RunEventReaderPortV0)
	}
	if reader == nil {
		return nil, []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "event_reader", "event_reader requerido")}, nil
	}
	events, err := reader.LoadRunEventsV0(ctx, request.RunRef)
	if err != nil {
		return nil, nil, err
	}
	if len(events) == 0 {
		return nil, []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "events", "historial de eventos requerido")}, nil
	}
	return events, nil, nil
}

func operationalDirectorClosureReadinessIssuesV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	request OperationalDirectorClosureRequestV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if !operationalDirectorClosureReflectedV0(run.Tasks, request.TaskID) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "task_id", "microtarea no pertenece al run"))
	}
	if !operationalDirectorClosureReflectedV0(run.Deliveries, request.DeliveryRef) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_ref", "entrega no registrada"))
	}
	if !operationalDirectorClosureReflectedV0(run.AcceptedReviews, request.AcceptedReviewRef) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "accepted_review_ref", "review aceptada no registrada"))
	}
	return issues
}

type operationalDirectorClosureTraceV0 struct {
	Deliveries      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0
	ReviewRequests  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0
	ReviewResults   map[string]orquestacoreworkflow.ReviewResultV0
	AcceptedReviews map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0
}

func operationalDirectorClosureCausalIssuesV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	request OperationalDirectorClosureRequestV0,
) []ErrorV0 {
	trace := operationalDirectorClosureTraceFromEventsV0(events)
	issues := make([]ErrorV0, 0)
	delivery, ok := trace.Deliveries[request.DeliveryRef]
	if !ok {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_ref", "entrega no encontrada en eventos"))
	} else if strings.TrimSpace(delivery.TaskID) != request.TaskID {
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

func (closer OperationalDirectorClosureV0) operationalDirectorClosureRequiredTestIssuesV0(
	ctx context.Context,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	trace operationalDirectorClosureTraceV0,
	request OperationalDirectorClosureRequestV0,
) ([]ErrorV0, error) {
	requiredTests := operationalDirectorClosureRequiredTestsForTaskV0(tasks, request.TaskID)
	if len(requiredTests) == 0 {
		return nil, nil
	}
	if len(request.RequiredTestEvidenceRefs) == 0 {
		return []ErrorV0{errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"required_test_evidence_refs",
			"evidencia durable de tests requerida",
		)}, nil
	}
	if closer.RequiredTestEvidenceStore == nil {
		return []ErrorV0{errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"required_test_evidence_store",
			"store de evidencias de tests requerido",
		)}, nil
	}
	accepted := trace.AcceptedReviews[request.AcceptedReviewRef]
	reviewResult, ok := operationalDirectorClosureAcceptedResultV0(trace, accepted, request.DeliveryRef)
	if !ok {
		return []ErrorV0{errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"review_result_ref",
			"resultado aceptado no encontrado para tests requeridos",
		)}, nil
	}
	evidence, err := closer.RequiredTestEvidenceStore.LoadRequiredTestEvidenceV0(
		ctx,
		request.RunRef,
		request.RequiredTestEvidenceRefs,
	)
	if err != nil {
		return nil, err
	}
	if !operationalDirectorClosureRequiredTestsSatisfiedV0(requiredTests, evidence, reviewResult, request) {
		return []ErrorV0{errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"required_test_evidence_refs",
			"tests requeridos sin evidencia passed causal",
		)}, nil
	}
	return nil, nil
}

func operationalDirectorClosureRequiredTestsForTaskV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	taskID string,
) []string {
	taskID = strings.TrimSpace(taskID)
	for _, task := range tasks {
		if strings.TrimSpace(task.TaskID) == taskID {
			return compactStringsV0(task.RequiredTests)
		}
	}
	return nil
}

func operationalDirectorClosureRequiredTestsSatisfiedV0(
	requiredTests []string,
	evidence []RequiredTestEvidenceV0,
	reviewResult orquestacoreworkflow.ReviewResultV0,
	request OperationalDirectorClosureRequestV0,
) bool {
	for _, required := range compactStringsV0(requiredTests) {
		if !operationalDirectorClosureRequiredTestSatisfiedV0(required, evidence, reviewResult, request) {
			return false
		}
	}
	return len(requiredTests) > 0
}

func operationalDirectorClosureRequiredTestSatisfiedV0(
	required string,
	evidence []RequiredTestEvidenceV0,
	reviewResult orquestacoreworkflow.ReviewResultV0,
	request OperationalDirectorClosureRequestV0,
) bool {
	required = strings.TrimSpace(required)
	for _, item := range evidence {
		if item.Status == RequiredTestEvidenceStatusPassedV0 &&
			strings.TrimSpace(item.RunRef) == request.RunRef &&
			operationalDirectorClosureReflectedV0(request.RequiredTestEvidenceRefs, item.EvidenceRef) &&
			strings.TrimSpace(item.TaskRef) == request.TaskID &&
			strings.TrimSpace(item.TestCommand) == required &&
			strings.TrimSpace(item.DeliveryRef) == request.DeliveryRef &&
			strings.TrimSpace(item.ReviewRequestID) == strings.TrimSpace(reviewResult.ReviewRequestID) &&
			strings.TrimSpace(item.ReviewResultRef) == strings.TrimSpace(reviewResult.ReviewResultRef) &&
			strings.TrimSpace(item.AcceptedReviewRef) == request.AcceptedReviewRef {
			return true
		}
	}
	return false
}

func operationalDirectorClosureTraceFromEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) operationalDirectorClosureTraceV0 {
	trace := operationalDirectorClosureTraceV0{
		Deliveries:      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0{},
		ReviewRequests:  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0{},
		ReviewResults:   map[string]orquestacoreworkflow.ReviewResultV0{},
		AcceptedReviews: map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0{},
	}
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
				trace.ReviewResults[strings.TrimSpace(payload.ReviewResultRef)] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewAcceptedV0:
			var payload orquestacoreworkflow.ReviewAcceptedPayloadV0
			if operationalDirectorDecodeEventPayloadV0(event, &payload) {
				trace.AcceptedReviews[strings.TrimSpace(payload.AcceptedReviewRef)] = payload
			}
		}
	}
	return trace
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
	for _, result := range trace.ReviewResults {
		if strings.TrimSpace(result.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(result.DeliveryRef) == deliveryRef &&
			result.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
			return result, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorClosedRunRequestIssuesV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	request OperationalDirectorClosureRequestV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	for _, check := range []struct {
		field  string
		values []string
		want   string
	}{
		{field: "task_id", values: run.ClosedTasks, want: request.TaskID},
		{field: "delivery_ref", values: run.Deliveries, want: request.DeliveryRef},
		{field: "accepted_review_ref", values: run.AcceptedReviews, want: request.AcceptedReviewRef},
		{field: "validation_ref", values: run.Validations, want: request.ValidationRef},
		{field: "closure_ref", values: run.Closures, want: request.ClosureRef},
	} {
		if !operationalDirectorClosureReflectedV0(check.values, check.want) {
			issues = append(issues, errorV0(
				ErrNucleoOrquestacionInvalidoV0,
				check.field,
				"request de cierre no corresponde al run cerrado",
			))
		}
	}
	return issues
}

func operationalDirectorCloseTaskCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewCloseTaskCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "close-task", request.TaskID),
		orquestacoreworkflow.CloseTaskCommandPayloadV0{
			TaskID:            request.TaskID,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:       request.DeliveryRef,
			AcceptedReviewRef: request.AcceptedReviewRef,
			Summary:           operationalDirectorClosureSummaryV0(request, "Cierre causal de microtarea del Director Operativo."),
			EvidenceRefs:      operationalDirectorClosureEvidenceRefsV0(request),
		},
	)
}

func operationalDirectorOpenFinalValidationCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewOpenPhaseCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "open-final-validation", request.ValidationRef),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
			Reason:  "Validar cierre causal del Director Operativo.",
		},
	)
}

func operationalDirectorRegisterFinalValidationCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewRegisterFinalValidationCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "register-final-validation", request.ValidationRef),
		orquestacoreworkflow.RegisterFinalValidationCommandPayloadV0{
			ValidationRef: request.ValidationRef,
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
			ClosedTaskRef: request.TaskID,
			Summary:       operationalDirectorClosureSummaryV0(request, "Validacion final con review y tests requeridos."),
			EvidenceRefs:  operationalDirectorClosureEvidenceRefsV0(request),
		},
	)
}

func operationalDirectorOpenClosureCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewOpenPhaseCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "open-closure", request.ClosureRef),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
			Reason:  "Cerrar run del Director Operativo.",
		},
	)
}

func operationalDirectorCloseRunCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewCloseRunCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "close-run", request.ClosureRef),
		orquestacoreworkflow.CloseRunCommandPayloadV0{
			ClosureRef:    request.ClosureRef,
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
			ValidationRef: request.ValidationRef,
			Summary:       operationalDirectorClosureSummaryV0(request, "Run cerrado por Director Operativo."),
			EvidenceRefs:  operationalDirectorClosureEvidenceRefsV0(request),
		},
	)
}

func operationalDirectorClosureCommandMetaV0(
	request OperationalDirectorClosureRequestV0,
	action string,
	ref string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	ref = strings.TrimSpace(ref)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-operational-director-" + action + "-" + ref,
		RunID:          request.RunRef,
		IdempotencyKey: "idem-operational-director-" + action + "-" + ref,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    operationalDirectorClosureRequestedByV0(request),
		OccurredAt:     request.OccurredAt,
	}
}

func normalizeOperationalDirectorClosureRequestV0(
	request OperationalDirectorClosureRequestV0,
) OperationalDirectorClosureRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.TaskID = strings.TrimSpace(request.TaskID)
	request.DeliveryRef = strings.TrimSpace(request.DeliveryRef)
	request.AcceptedReviewRef = strings.TrimSpace(request.AcceptedReviewRef)
	request.ValidationRef = strings.TrimSpace(request.ValidationRef)
	request.ClosureRef = strings.TrimSpace(request.ClosureRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.Summary = strings.TrimSpace(request.Summary)
	request.RequiredTestEvidenceRefs = compactStringsV0(request.RequiredTestEvidenceRefs)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	return request
}

func (closer OperationalDirectorClosureV0) validateOperationalDirectorClosureV0(
	request OperationalDirectorClosureRequestV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if closer.RunStore == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido"))
	}
	if closer.EventSink == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "event_sink", "event_sink requerido"))
	}
	if strings.TrimSpace(request.RunRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido"))
	}
	if strings.TrimSpace(request.TaskID) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "task_id", "task_id requerido"))
	}
	if strings.TrimSpace(request.DeliveryRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_ref", "delivery_ref requerido"))
	}
	if strings.TrimSpace(request.AcceptedReviewRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "accepted_review_ref", "accepted_review_ref requerido"))
	}
	if strings.TrimSpace(request.ValidationRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "validation_ref", "validation_ref requerido"))
	}
	if strings.TrimSpace(request.ClosureRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "closure_ref", "closure_ref requerido"))
	}
	if strings.TrimSpace(request.OccurredAt) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido"))
	}
	return issues
}

func operationalDirectorClosureTaskLoadedV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	taskID string,
) bool {
	taskID = strings.TrimSpace(taskID)
	for _, task := range tasks {
		if strings.TrimSpace(task.TaskID) == taskID {
			return true
		}
	}
	return false
}

func operationalDirectorClosureTaskRequiresTestsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	taskID string,
) bool {
	taskID = strings.TrimSpace(taskID)
	for _, task := range tasks {
		if strings.TrimSpace(task.TaskID) == taskID {
			return len(compactStringsV0(task.RequiredTests)) > 0
		}
	}
	return false
}

func operationalDirectorClosureOpenTasksV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	open := make([]string, 0, len(run.Tasks))
	closed := compactStringsV0(run.ClosedTasks)
	for _, taskRef := range compactStringsV0(run.Tasks) {
		if !operationalDirectorClosureReflectedV0(closed, taskRef) {
			open = append(open, taskRef)
		}
	}
	return open
}

func operationalDirectorClosureReflectedV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func operationalDirectorClosureEvidenceRefsV0(
	request OperationalDirectorClosureRequestV0,
) []string {
	return compactStringsV0(append(append([]string(nil), request.EvidenceRefs...), request.RequiredTestEvidenceRefs...))
}

func operationalDirectorClosureSummaryV0(
	request OperationalDirectorClosureRequestV0,
	fallback string,
) string {
	if request.Summary != "" {
		return request.Summary
	}
	return fallback
}

func operationalDirectorClosureRequestedByV0(
	request OperationalDirectorClosureRequestV0,
) string {
	requestedBy := strings.TrimSpace(request.RequestedBy)
	if requestedBy != "" {
		return requestedBy
	}
	return "orquesta-operational-director-closure"
}

func operationalDirectorClosurePhaseV0(
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestacoreworkflow.OrchestrationPhaseIDV0 {
	return orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(string(phase)))
}
