package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestOperationalClosureSourceV0ConstruyeRequestCausalDentroDeScope(t *testing.T) {
	runRef := "run-stack-operational-closure-source-001"
	task := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-a", []string{"go test ./..."})
	other := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-b", nil)
	run := stackOperationalClosureRunForTestV0(runRef, task.TaskID, other.TaskID)
	run.Deliveries = []string{
		"delivery-ref-stack-operational-closure-a",
		"delivery-ref-stack-operational-closure-b",
	}
	run.AcceptedReviews = []string{
		"accepted-review-ref-delivery-ref-stack-operational-closure-a",
		"accepted-review-ref-delivery-ref-stack-operational-closure-b",
	}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, task.TaskID, "delivery-ref-stack-operational-closure-a"),
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 2, other.TaskID, "delivery-ref-stack-operational-closure-b"),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 3, "delivery-ref-stack-operational-closure-a"),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 4, "delivery-ref-stack-operational-closure-a", orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 5, "delivery-ref-stack-operational-closure-a"),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 6, "delivery-ref-stack-operational-closure-b"),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 7, "delivery-ref-stack-operational-closure-b", orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 8, "delivery-ref-stack-operational-closure-b"),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:                 orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task, other),
		EventReader:               reader,
		RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(stackOperationalClosureRequiredTestEvidenceForTestV0(runRef, task.TaskID, "delivery-ref-stack-operational-closure-a")),
	}

	got, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:              run,
			LoopStatus:       orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			OccurredAt:       "2026-05-17T14:00:00Z",
			CorrelationID:    "corr-stack-operational-closure-source-001",
			RequestedBy:      "orquesta-app-codex-stack-test",
			WaitScopeApplied: true,
			WaitAgentRefs:    []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID)},
			EvidenceRefs:     []string{"evidence-ref-stack-operational-closure-source-001"},
		},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if !ok {
		t.Fatalf("source no produjo cierre")
	}
	if got.TaskID != task.TaskID ||
		got.DeliveryRef != "delivery-ref-stack-operational-closure-a" ||
		got.AcceptedReviewRef != "accepted-review-ref-delivery-ref-stack-operational-closure-a" {
		t.Fatalf("request fuera de scope o no causal: %+v", got)
	}
	if want := []string{"test-evidence-ref-delivery-ref-stack-operational-closure-a"}; !reflect.DeepEqual(got.RequiredTestEvidenceRefs, want) {
		t.Fatalf("required_test_evidence_refs=%v, want %v", got.RequiredTestEvidenceRefs, want)
	}
	if codexStackOperationalClosureContainsV0(got.EvidenceRefs, "evidence-ref-delivery-ref-stack-operational-closure-b") {
		t.Fatalf("evidence_refs arrastra evidencia fuera de scope: %+v", got.EvidenceRefs)
	}
}

func TestOperationalClosureSourceV0CierraConEvidenceRefsMixtosEnReviewResult(t *testing.T) {
	runRef := "run-stack-operational-closure-source-mixed-evidence-001"
	task := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-mixed-evidence", []string{"go test ./..."})
	deliveryRef := "delivery-ref-stack-operational-closure-mixed-evidence"
	run := stackOperationalClosureRunForTestV0(runRef, task.TaskID)
	run.Deliveries = []string{deliveryRef}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, task.TaskID, deliveryRef),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 2, deliveryRef),
		stackOperationalClosureReviewResultEventWithEvidenceRefsForTestV0(t, runRef, 3, deliveryRef, orquestacoreworkflow.ReviewResultStatusAcceptedV0, []string{
			"transcript-ref-" + deliveryRef,
			"test-evidence-ref-" + deliveryRef,
		}),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 4, deliveryRef),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:                 orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		EventReader:               reader,
		RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(stackOperationalClosureRequiredTestEvidenceForTestV0(runRef, task.TaskID, deliveryRef)),
	}

	got, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: run, OccurredAt: "2026-05-17T14:15:00Z"},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if !ok || !reflect.DeepEqual(got.RequiredTestEvidenceRefs, []string{"test-evidence-ref-" + deliveryRef}) {
		t.Fatalf("source no cerro con evidencia mixta: ok=%v request=%+v", ok, got)
	}
}

func TestOperationalClosureSourceV0UsaRequiredTestEvidenceRefsDelRequest(t *testing.T) {
	runRef := "run-stack-operational-closure-source-request-evidence-001"
	task := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-request-evidence", []string{"go test ./..."})
	deliveryRef := "delivery-ref-stack-operational-closure-request-evidence"
	run := stackOperationalClosureRunForTestV0(runRef, task.TaskID)
	run.Deliveries = []string{deliveryRef}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, task.TaskID, deliveryRef),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 2, deliveryRef),
		stackOperationalClosureReviewResultEventWithEvidenceRefsForTestV0(t, runRef, 3, deliveryRef, orquestacoreworkflow.ReviewResultStatusAcceptedV0, []string{
			"transcript-ref-" + deliveryRef,
		}),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 4, deliveryRef),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:                 orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		EventReader:               reader,
		RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(stackOperationalClosureRequiredTestEvidenceForTestV0(runRef, task.TaskID, deliveryRef)),
	}

	got, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:                      run,
			OccurredAt:               "2026-05-17T14:16:00Z",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-" + deliveryRef},
		},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if !ok || !reflect.DeepEqual(got.RequiredTestEvidenceRefs, []string{"test-evidence-ref-" + deliveryRef}) {
		t.Fatalf("source no cerro con refs explicitas del request: ok=%v request=%+v", ok, got)
	}
}

func TestOperationalClosureSourceV0UsaDeliveryAceptadaAunqueHayaDeliveryPreviaNoAceptada(t *testing.T) {
	runRef := "run-stack-operational-closure-source-multi-delivery-001"
	task := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-multi-delivery", nil)
	oldDelivery := "delivery-ref-stack-operational-closure-old"
	acceptedDelivery := "delivery-ref-stack-operational-closure-new"
	run := stackOperationalClosureRunForTestV0(runRef, task.TaskID)
	run.Deliveries = []string{oldDelivery, acceptedDelivery}
	run.AcceptedReviews = []string{"accepted-review-ref-" + acceptedDelivery}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, task.TaskID, oldDelivery),
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 2, task.TaskID, acceptedDelivery),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 3, acceptedDelivery),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 4, acceptedDelivery, orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 5, acceptedDelivery),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:   orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		EventReader: reader,
	}

	got, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: run, OccurredAt: "2026-05-17T14:20:00Z"},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if !ok || got.DeliveryRef != acceptedDelivery {
		t.Fatalf("delivery elegida=%s ok=%v", got.DeliveryRef, ok)
	}
}

func TestOperationalClosureSourceV0NoCierraPadreConHijosAbiertos(t *testing.T) {
	runRef := "run-stack-operational-closure-source-parent-open-child-001"
	parent := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-parent", nil)
	child := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-child", nil)
	parent.ChildTaskRefs = []string{child.TaskID}
	child.ParentTaskRef = parent.TaskID
	child.DelegationDepth = parent.DelegationDepth + 1
	run := stackOperationalClosureRunForTestV0(runRef, parent.TaskID, child.TaskID)
	deliveryRef := "delivery-ref-stack-operational-closure-parent-open-child"
	run.Deliveries = []string{deliveryRef}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, parent.TaskID, deliveryRef),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 2, deliveryRef),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 3, deliveryRef, orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 4, deliveryRef),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:   orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(parent, child),
		EventReader: reader,
	}

	_, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: run, OccurredAt: "2026-05-17T14:21:00Z"},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if ok {
		t.Fatalf("no debe cerrar padre con child_task_refs abiertos")
	}
}

func TestOperationalClosureSourceV0CierraPadreCuandoHijosYaCerrados(t *testing.T) {
	runRef := "run-stack-operational-closure-source-parent-closed-child-001"
	parent := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-parent-closed", nil)
	child := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-child-closed", nil)
	parent.ChildTaskRefs = []string{child.TaskID}
	child.ParentTaskRef = parent.TaskID
	child.DelegationDepth = parent.DelegationDepth + 1
	run := stackOperationalClosureRunForTestV0(runRef, parent.TaskID, child.TaskID)
	deliveryRef := "delivery-ref-stack-operational-closure-parent-closed-child"
	run.Deliveries = []string{deliveryRef}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	run.ClosedTasks = []string{child.TaskID}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, parent.TaskID, deliveryRef),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 2, deliveryRef),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 3, deliveryRef, orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 4, deliveryRef),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:   orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(parent, child),
		EventReader: reader,
	}

	got, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: run, OccurredAt: "2026-05-17T14:22:00Z"},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if !ok || got.TaskID != parent.TaskID {
		t.Fatalf("debe cerrar padre cuando hijos ya estan cerrados: ok=%v request=%+v", ok, got)
	}
}

func TestOperationalClosureSourceV0NoCierraSinReviewRequestedCausal(t *testing.T) {
	runRef := "run-stack-operational-closure-source-no-review-requested-001"
	task := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-no-review-requested", nil)
	deliveryRef := "delivery-ref-stack-operational-closure-no-review-requested"
	run := stackOperationalClosureRunForTestV0(runRef, task.TaskID)
	run.Deliveries = []string{deliveryRef}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, task.TaskID, deliveryRef),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 2, deliveryRef, orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 3, deliveryRef),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:   orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		EventReader: reader,
	}

	_, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: run, OccurredAt: "2026-05-17T14:25:00Z"},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if ok {
		t.Fatalf("no debe cerrar sin ReviewRequested causal")
	}
}

func TestOperationalClosureSourceV0NoCierraSinReviewAceptada(t *testing.T) {
	runRef := "run-stack-operational-closure-source-no-review-001"
	task := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-no-review", nil)
	run := stackOperationalClosureRunForTestV0(runRef, task.TaskID)
	run.Deliveries = []string{"delivery-ref-stack-operational-closure-no-review"}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, task.TaskID, "delivery-ref-stack-operational-closure-no-review"),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 2, "delivery-ref-stack-operational-closure-no-review"),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 3, "delivery-ref-stack-operational-closure-no-review", orquestacoreworkflow.ReviewResultStatusChangesRequestedV0),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:   orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		EventReader: reader,
	}

	_, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:        run,
			OccurredAt: "2026-05-17T14:05:00Z",
		},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if ok {
		t.Fatalf("no debe producir cierre sin review aceptada")
	}
}

func TestOperationalClosureSourceV0NoCierraSinTestEvidenceDurable(t *testing.T) {
	runRef := "run-stack-operational-closure-source-no-tests-001"
	task := stackOperationalClosureTaskForTestV0(runRef, "task-stack-operational-closure-no-tests", []string{"go test ./..."})
	run := stackOperationalClosureRunForTestV0(runRef, task.TaskID)
	run.Deliveries = []string{"delivery-ref-stack-operational-closure-no-tests"}
	run.AcceptedReviews = []string{"accepted-review-ref-delivery-ref-stack-operational-closure-no-tests"}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, task.TaskID, "delivery-ref-stack-operational-closure-no-tests"),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 2, "delivery-ref-stack-operational-closure-no-tests"),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 3, "delivery-ref-stack-operational-closure-no-tests", orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 4, "delivery-ref-stack-operational-closure-no-tests"),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:   orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		EventReader: reader,
	}

	_, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:        run,
			OccurredAt: "2026-05-17T14:10:00Z",
		},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if ok {
		t.Fatalf("no debe producir cierre sin evidencia durable de tests")
	}
}

func stackOperationalClosureRunForTestV0(
	runRef string,
	taskRefs ...string,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-" + runRef,
		AppSpecRef:    "appspec-" + runRef,
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		}},
		Tasks: append([]string(nil), taskRefs...),
	}
}

func stackOperationalClosureTaskForTestV0(
	runRef string,
	taskRef string,
	requiredTests []string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Task cierre operativo stack",
		WriteSet:           []string{"internal/" + taskRef},
		AcceptanceCriteria: []string{"operational_director.plan_ref: plan-stack-operational-closure", "entrega revisada"},
		RequiredTests:      append([]string(nil), requiredTests...),
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:operational-director:v0",
			FunctionName: "OperationalDirectorCut",
		}},
	}
}

func stackOperationalClosureRequiredTestEvidenceForTestV0(
	runRef string,
	taskRef string,
	deliveryRef string,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       "test-evidence-ref-" + deliveryRef,
		RunRef:            runRef,
		TaskRef:           taskRef,
		TestCommand:       "go test ./...",
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       deliveryRef,
		ReviewRequestID:   "review-request-ref-" + deliveryRef,
		ReviewResultRef:   "review-result-ref-" + deliveryRef,
		AcceptedReviewRef: "accepted-review-ref-" + deliveryRef,
		OccurredAt:        "2026-05-17T14:00:00Z",
		EvidenceRefs:      []string{"test-output-ref-" + deliveryRef},
	}
}

func stackOperationalClosureDeliveryEventForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	taskRef string,
	deliveryRef string,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	return stackOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
		DeliveryRef:  deliveryRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       taskRef,
		AgentRef:     orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef),
		Summary:      "Entrega para cierre operativo stack.",
		EvidenceRefs: []string{"evidence-ref-" + deliveryRef},
	})
}

func stackOperationalClosureReviewRequestedEventForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	deliveryRef string,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	return stackOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
		ReviewRequestID: "review-request-ref-" + deliveryRef,
		PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		DeliveryRef:     deliveryRef,
		Summary:         "Review para cierre operativo stack.",
		EvidenceRefs:    []string{"evidence-ref-review-request-" + deliveryRef},
	})
}

func stackOperationalClosureReviewResultEventForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	deliveryRef string,
	status orquestacoreworkflow.ReviewResultStatusV0,
) orquestacoreworkflow.OrchestrationEventV0 {
	return stackOperationalClosureReviewResultEventWithEvidenceRefsForTestV0(
		t,
		runRef,
		sequence,
		deliveryRef,
		status,
		[]string{"test-evidence-ref-" + deliveryRef},
	)
}

func stackOperationalClosureReviewResultEventWithEvidenceRefsForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	deliveryRef string,
	status orquestacoreworkflow.ReviewResultStatusV0,
	evidenceRefs []string,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	return stackOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: "review-result-ref-" + deliveryRef,
		ReviewRequestID: "review-request-ref-" + deliveryRef,
		DeliveryRef:     deliveryRef,
		Status:          status,
		Summary:         "Resultado de review para cierre operativo stack.",
		EvidenceRefs:    append([]string(nil), evidenceRefs...),
	})
}

func stackOperationalClosureAcceptedReviewEventForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	deliveryRef string,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	return stackOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
		AcceptedReviewRef: "accepted-review-ref-" + deliveryRef,
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		ReviewRequestID:   "review-request-ref-" + deliveryRef,
		DeliveryRef:       deliveryRef,
		Summary:           "Review aceptada para cierre operativo stack.",
		EvidenceRefs:      []string{"evidence-ref-accepted-review-" + deliveryRef},
	})
}

func stackOperationalClosureEventForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	eventType string,
	payload any,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return orquestacoreworkflow.OrchestrationEventV0{
		EventID:        fmt.Sprintf("evt-stack-operational-closure-%s-%s-%03d", eventType, runRef, sequence),
		EventType:      eventType,
		RunID:          runRef,
		Sequence:       sequence,
		PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
		Payload:        raw,
	}
}
