package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestOperationalDirectorClosureV0CierraRunConReviewYTests(t *testing.T) {
	runRef := "run-operational-director-closure-001"
	runStore, sink, _ := mustOperationalDirectorClosureReadyStoreV0(t, runRef)
	taskStore := NewInMemoryWorkflowTaskStoreV0(
		operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", []string{"go test ./..."}),
	)

	result, err := (OperationalDirectorClosureV0{
		RunStore:                  runStore,
		EventSink:                 sink,
		TaskStore:                 taskStore,
		RequiredTestEvidenceStore: NewInMemoryRequiredTestEvidenceStoreV0(requiredTestEvidenceForTestV0(runRef, "task-ref-nucleo-001", "go test ./...")),
		RequestedBy:               "operational-director-closure-test",
	}).CloseOperationalDirectorRunV0(context.Background(), operationalDirectorClosureRequestForTestV0(runRef))
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if len(result.Issues) > 0 {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!containsNucleoRefV0(result.Run.ClosedTasks, "task-ref-nucleo-001") ||
		!containsNucleoRefV0(result.Run.Validations, "validation-ref-operational-director-001") ||
		!containsNucleoRefV0(result.Run.Closures, "closure-ref-operational-director-001") {
		t.Fatalf("run no cerrado causalmente: %+v", result.Run)
	}
	if result.EventsCount != 5 || len(result.Commands) != 5 {
		t.Fatalf("events=%d commands=%d", result.EventsCount, len(result.Commands))
	}
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventTaskClosedV0,
		orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0,
		orquestacoreworkflow.OrchestrationEventRunClosedV0,
	} {
		if !sinkHasEventTypeV0(sink, eventType) {
			t.Fatalf("sink sin %s: %+v", eventType, sink.EventsV0())
		}
	}
}

func TestOperationalDirectorClosureV0ExigeEvidenciaTestsRequeridos(t *testing.T) {
	runRef := "run-operational-director-closure-tests-001"
	runStore, sink, _ := mustOperationalDirectorClosureReadyStoreV0(t, runRef)
	request := operationalDirectorClosureRequestForTestV0(runRef)
	request.RequiredTestEvidenceRefs = nil

	result, err := (OperationalDirectorClosureV0{
		RunStore:  runStore,
		EventSink: sink,
		TaskStore: NewInMemoryWorkflowTaskStoreV0(
			operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", []string{"go test ./..."}),
		),
	}).CloseOperationalDirectorRunV0(context.Background(), request)
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if !operationalDirectorClosureHasIssueV0(result.Issues, "required_test_evidence_refs") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if len(result.Run.ClosedTasks) != 0 || result.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("no debe cerrar sin tests: %+v", result.Run)
	}
}

func TestOperationalDirectorClosureV0ExigeStoreDeTestsDurables(t *testing.T) {
	runRef := "run-operational-director-closure-test-store-001"
	runStore, sink, _ := mustOperationalDirectorClosureReadyStoreV0(t, runRef)

	result, err := (OperationalDirectorClosureV0{
		RunStore:  runStore,
		EventSink: sink,
		TaskStore: NewInMemoryWorkflowTaskStoreV0(
			operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", []string{"go test ./..."}),
		),
	}).CloseOperationalDirectorRunV0(context.Background(), operationalDirectorClosureRequestForTestV0(runRef))
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if !operationalDirectorClosureHasIssueV0(result.Issues, "required_test_evidence_store") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if len(result.Run.ClosedTasks) != 0 || result.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("no debe cerrar sin store de tests: %+v", result.Run)
	}
}

func TestOperationalDirectorClosureV0ExigeEventSink(t *testing.T) {
	runRef := "run-operational-director-closure-event-sink-001"
	runStore, _, _ := mustOperationalDirectorClosureReadyStoreV0(t, runRef)

	result, err := (OperationalDirectorClosureV0{
		RunStore:    runStore,
		EventReader: NewInMemoryEventSinkV0(),
		TaskStore: NewInMemoryWorkflowTaskStoreV0(
			operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", nil),
		),
	}).CloseOperationalDirectorRunV0(context.Background(), operationalDirectorClosureRequestForTestV0(runRef))
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if !operationalDirectorClosureHasIssueV0(result.Issues, "event_sink") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	loaded, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if loaded.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("no debe cerrar sin event sink: %+v", loaded)
	}
}

func TestOperationalDirectorClosureV0RechazaTestEvidenceDeOtraTask(t *testing.T) {
	runRef := "run-operational-director-closure-test-foreign-001"
	runStore, sink, _ := mustOperationalDirectorClosureReadyStoreV0(t, runRef)
	foreign := requiredTestEvidenceForTestV0(runRef, "task-ref-otra", "go test ./...")
	foreign.EvidenceRef = "test-evidence-ref-task-ref-nucleo-001"

	result, err := (OperationalDirectorClosureV0{
		RunStore:                  runStore,
		EventSink:                 sink,
		TaskStore:                 NewInMemoryWorkflowTaskStoreV0(operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", []string{"go test ./..."})),
		RequiredTestEvidenceStore: NewInMemoryRequiredTestEvidenceStoreV0(foreign),
	}).CloseOperationalDirectorRunV0(context.Background(), operationalDirectorClosureRequestForTestV0(runRef))
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if !operationalDirectorClosureHasIssueV0(result.Issues, "required_test_evidence_refs") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if len(result.Run.ClosedTasks) != 0 || result.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("no debe cerrar con evidencia de otra task: %+v", result.Run)
	}
}

func TestOperationalDirectorClosureV0ExigeTodosLosTestsRequeridos(t *testing.T) {
	runRef := "run-operational-director-closure-multiple-tests-001"
	runStore, sink, _ := mustOperationalDirectorClosureReadyStoreV0(t, runRef)
	request := operationalDirectorClosureRequestForTestV0(runRef)
	request.RequiredTestEvidenceRefs = []string{"test-evidence-ref-task-ref-nucleo-001"}

	result, err := (OperationalDirectorClosureV0{
		RunStore:  runStore,
		EventSink: sink,
		TaskStore: NewInMemoryWorkflowTaskStoreV0(
			operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", []string{"go test ./...", "go test ./modulos/orquesta-orchestration-core"}),
		),
		RequiredTestEvidenceStore: NewInMemoryRequiredTestEvidenceStoreV0(requiredTestEvidenceForTestV0(runRef, "task-ref-nucleo-001", "go test ./...")),
	}).CloseOperationalDirectorRunV0(context.Background(), request)
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if !operationalDirectorClosureHasIssueV0(result.Issues, "required_test_evidence_refs") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if len(result.Run.ClosedTasks) != 0 || result.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("no debe cerrar con test requerido faltante: %+v", result.Run)
	}
}

func TestOperationalDirectorClosureV0RechazaTestFailed(t *testing.T) {
	runRef := "run-operational-director-closure-test-failed-001"
	runStore, sink, _ := mustOperationalDirectorClosureReadyStoreV0(t, runRef)
	failed := requiredTestEvidenceForTestV0(runRef, "task-ref-nucleo-001", "go test ./...")
	failed.Status = RequiredTestEvidenceStatusFailedV0

	result, err := (OperationalDirectorClosureV0{
		RunStore:                  runStore,
		EventSink:                 sink,
		TaskStore:                 NewInMemoryWorkflowTaskStoreV0(operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", []string{"go test ./..."})),
		RequiredTestEvidenceStore: NewInMemoryRequiredTestEvidenceStoreV0(failed),
	}).CloseOperationalDirectorRunV0(context.Background(), operationalDirectorClosureRequestForTestV0(runRef))
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if !operationalDirectorClosureHasIssueV0(result.Issues, "required_test_evidence_refs") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if len(result.Run.ClosedTasks) != 0 || result.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("no debe cerrar con test fallido: %+v", result.Run)
	}
}

func TestOperationalDirectorClosureV0NoValidaConOtraTaskAbierta(t *testing.T) {
	runRef := "run-operational-director-closure-open-task-001"
	runStore, sink, run := mustOperationalDirectorClosureReadyStoreV0(t, runRef)
	run.Tasks = append(run.Tasks, "task-ref-nucleo-002")
	if err := runStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	result, err := (OperationalDirectorClosureV0{
		RunStore:  runStore,
		EventSink: sink,
		TaskStore: NewInMemoryWorkflowTaskStoreV0(
			operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", nil),
			operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-002", nil),
		),
	}).CloseOperationalDirectorRunV0(context.Background(), operationalDirectorClosureRequestForTestV0(runRef))
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if !operationalDirectorClosureHasIssueV0(result.Issues, "run.open_tasks") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if !containsNucleoRefV0(result.Run.ClosedTasks, "task-ref-nucleo-001") {
		t.Fatalf("debe cerrar la task objetivo antes de bloquear validacion: %+v", result.Run)
	}
	if len(result.Run.Validations) != 0 || len(result.Run.Closures) != 0 {
		t.Fatalf("no debe validar/cerrar con task abierta: %+v", result.Run)
	}
}

func TestOperationalDirectorClosureV0NoCierraSinReviewAceptada(t *testing.T) {
	runRef := "run-operational-director-closure-no-review-001"
	run := mustReviewReworkReadyRunV0(t, runRef, orquestacoreworkflow.ReviewResultStatusChangesRequestedV0)

	result, err := (OperationalDirectorClosureV0{
		RunStore:  NewInMemoryRunStoreV0(run),
		EventSink: NewInMemoryEventSinkV0(),
		TaskStore: NewInMemoryWorkflowTaskStoreV0(
			operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", nil),
		),
	}).CloseOperationalDirectorRunV0(context.Background(), operationalDirectorClosureRequestForTestV0(runRef))
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if !operationalDirectorClosureHasIssueV0(result.Issues, "accepted_review_ref") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if len(result.Run.ClosedTasks) != 0 || result.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("no debe cerrar sin review aceptada: %+v", result.Run)
	}
}

func TestOperationalDirectorClosureV0RechazaReviewAceptadaDeOtraEntrega(t *testing.T) {
	runRef := "run-operational-director-closure-cross-review-001"
	runStore, sink, run := mustOperationalDirectorClosureReadyStoreV0(t, runRef)
	run.AcceptedReviews = append(run.AcceptedReviews, "accepted-review-ref-foreign-delivery")
	if err := runStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	foreign, err := orquestacoreworkflow.NewReviewAcceptedEventV0(
		orquestacoreworkflow.OrchestrationEventMetaV0{
			EventID:        "evt-foreign-accepted-review",
			RunID:          runRef,
			Sequence:       run.LastSequence + 1,
			IdempotencyKey: "idem-foreign-accepted-review",
			CorrelationID:  "corr-foreign-accepted-review",
			CausationID:    "cmd-foreign-accepted-review",
			OccurredAt:     "2026-05-17T13:05:00Z",
		},
		orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: "accepted-review-ref-foreign-delivery",
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   "review-request-ref-foreign-delivery",
			DeliveryRef:       "delivery-ref-foreign-delivery",
			Summary:           "Review aceptada de otra entrega.",
		},
	)
	if err != nil {
		t.Fatalf("foreign accepted event: %v", err)
	}
	if err := sink.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{foreign}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	request := operationalDirectorClosureRequestForTestV0(runRef)
	request.AcceptedReviewRef = "accepted-review-ref-foreign-delivery"

	result, err := (OperationalDirectorClosureV0{
		RunStore:  runStore,
		EventSink: sink,
		TaskStore: NewInMemoryWorkflowTaskStoreV0(
			operationalDirectorClosureTaskForTestV0(runRef, "task-ref-nucleo-001", nil),
		),
	}).CloseOperationalDirectorRunV0(context.Background(), request)
	if err != nil {
		t.Fatalf("CloseOperationalDirectorRunV0: %v", err)
	}
	if !operationalDirectorClosureHasIssueV0(result.Issues, "accepted_review_ref") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if containsNucleoRefV0(result.Run.ClosedTasks, "task-ref-nucleo-001") {
		t.Fatalf("no debe cerrar con review cruzada: %+v", result.Run)
	}
}

func mustOperationalDirectorClosureReadyRunV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := mustReviewGateReadyRunV0(t, runRef)
	run = mustApplyCommandV0(t, run, mustReviewReworkRequestReviewCommandV0(t, runRef))
	run = mustApplyCommandV0(t, run, mustReviewReworkResultCommandV0(t, runRef, orquestacoreworkflow.ReviewResultStatusAcceptedV0))
	return mustApplyCommandV0(t, run, mustOperationalDirectorAcceptReviewCommandV0(t, runRef))
}

func mustOperationalDirectorClosureReadyStoreV0(
	t *testing.T,
	runRef string,
) (*InMemoryRunStoreV0, *InMemoryEventSinkV0, orquestacoreworkflow.OrchestrationRunV0) {
	t.Helper()
	run := mustDeliveryReadyRunV0(t, runRef)
	store := NewInMemoryRunStoreV0(run)
	sink := NewInMemoryEventSinkV0()
	for _, command := range []orquestacoreworkflow.OrchestrationCommandV0{
		mustReviewGateDeliveryCommandV0(t, runRef),
		mustOpenRevisionCommandV0(t, runRef),
		mustReviewReworkRequestReviewCommandV0(t, runRef),
		mustReviewReworkResultCommandV0(t, runRef, orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		mustOperationalDirectorAcceptReviewCommandV0(t, runRef),
	} {
		if _, err := HandleStoredWorkflowCommandV0(context.Background(), store, sink, command); err != nil {
			t.Fatalf("HandleStoredWorkflowCommandV0(%s): %v", command.CommandType, err)
		}
	}
	ready, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	return store, sink, ready
}

func mustOperationalDirectorAcceptReviewCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewAcceptReviewCommandV0(
		commandMetaV0(runRef, "cmd-accept-review-operational-director-001", "idem-accept-review-operational-director-001"),
		orquestacoreworkflow.AcceptReviewCommandPayloadV0{
			AcceptedReviewRef: "accepted-review-ref-nucleo-review-001",
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   "review-request-ref-nucleo-review-001",
			DeliveryRef:       "delivery-ref-nucleo-review-001",
			Summary:           "Review aceptada para cierre operativo.",
			EvidenceRefs:      []string{"evidence-ref-accepted-review-operational-director-001"},
		},
	)
	if err != nil {
		t.Fatalf("accept review command: %v", err)
	}
	return command
}

func operationalDirectorClosureTaskForTestV0(
	runRef string,
	taskRef string,
	requiredTests []string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Task cierre operativo",
		Summary:            "Task usada por cierre operativo.",
		WriteSet:           []string{"internal/module"},
		AcceptanceCriteria: []string{"entrega revisada"},
		RequiredTests:      append([]string(nil), requiredTests...),
	}
}

func operationalDirectorClosureRequestForTestV0(
	runRef string,
) OperationalDirectorClosureRequestV0 {
	return OperationalDirectorClosureRequestV0{
		RunRef:                   runRef,
		TaskID:                   "task-ref-nucleo-001",
		DeliveryRef:              "delivery-ref-nucleo-review-001",
		AcceptedReviewRef:        "accepted-review-ref-nucleo-review-001",
		ValidationRef:            "validation-ref-operational-director-001",
		ClosureRef:               "closure-ref-operational-director-001",
		OccurredAt:               "2026-05-17T13:00:00Z",
		CorrelationID:            "corr-operational-director-closure-001",
		Summary:                  "Cierre generico causal del Director Operativo.",
		RequiredTestEvidenceRefs: []string{"test-evidence-ref-task-ref-nucleo-001"},
		EvidenceRefs:             []string{"evidence-ref-operational-director-closure-001"},
	}
}

func operationalDirectorClosureHasIssueV0(issues []ErrorV0, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
