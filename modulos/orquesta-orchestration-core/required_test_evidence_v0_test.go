package orquestacionnucleoapp

import (
	"context"
	"testing"
)

func TestRequiredTestEvidenceV0NormalizaYValida(t *testing.T) {
	evidence, err := NewRequiredTestEvidenceV0(requiredTestEvidenceForTestV0("run-test-evidence-001", "task-test-evidence-001", "go test ./..."))
	if err != nil {
		t.Fatalf("NewRequiredTestEvidenceV0: %v", err)
	}
	if evidence.EvidenceRef != "test-evidence-ref-task-test-evidence-001" ||
		evidence.Status != RequiredTestEvidenceStatusPassedV0 {
		t.Fatalf("evidence=%+v", evidence)
	}
}

func TestRequiredTestEvidenceV0RechazaStatusDesconocido(t *testing.T) {
	evidence := requiredTestEvidenceForTestV0("run-test-evidence-status-001", "task-test-evidence-status-001", "go test ./...")
	evidence.Status = "unknown"
	if _, err := NewRequiredTestEvidenceV0(evidence); err == nil {
		t.Fatal("err=nil, want status invalido")
	}
}

func TestRequiredTestEvidenceV0RechazaSinArtefactos(t *testing.T) {
	evidence := requiredTestEvidenceForTestV0("run-test-evidence-artifacts-001", "task-test-evidence-artifacts-001", "go test ./...")
	evidence.EvidenceRefs = nil
	if _, err := NewRequiredTestEvidenceV0(evidence); err == nil {
		t.Fatal("err=nil, want evidence_refs requerido")
	}
}

func TestInMemoryRequiredTestEvidenceStoreV0CargaRefs(t *testing.T) {
	item := requiredTestEvidenceForTestV0("run-test-evidence-store-001", "task-test-evidence-store-001", "go test ./...")
	store := NewInMemoryRequiredTestEvidenceStoreV0(item)
	got, err := store.LoadRequiredTestEvidenceV0(context.Background(), item.RunRef, []string{item.EvidenceRef})
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(got) != 1 || got[0].TestCommand != item.TestCommand {
		t.Fatalf("got=%+v", got)
	}
}

func TestInMemoryRequiredTestEvidenceStoreV0RechazaMismaRefConPayloadDistinto(t *testing.T) {
	item := requiredTestEvidenceForTestV0("run-test-evidence-conflict-001", "task-test-evidence-conflict-001", "go test ./...")
	store := NewInMemoryRequiredTestEvidenceStoreV0(item)
	if err := store.SaveRequiredTestEvidenceV0(context.Background(), item); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0 idempotente: %v", err)
	}
	changed := item
	changed.Status = RequiredTestEvidenceStatusFailedV0
	if err := store.SaveRequiredTestEvidenceV0(context.Background(), changed); err == nil {
		t.Fatal("err=nil, want conflicto durable")
	}
}

func requiredTestEvidenceForTestV0(
	runRef string,
	taskRef string,
	command string,
) RequiredTestEvidenceV0 {
	return RequiredTestEvidenceV0{
		SchemaVersion:     RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       "test-evidence-ref-" + taskRef,
		RunRef:            runRef,
		TaskRef:           taskRef,
		TestCommand:       command,
		Status:            RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       "delivery-ref-nucleo-review-001",
		ReviewRequestID:   "review-request-ref-nucleo-review-001",
		ReviewResultRef:   "review-result-ref-nucleo-review-001",
		AcceptedReviewRef: "accepted-review-ref-nucleo-review-001",
		OccurredAt:        "2026-05-17T15:30:00Z",
		EvidenceRefs:      []string{"evidence-ref-test-output-001"},
	}
}
