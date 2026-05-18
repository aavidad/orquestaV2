package orquestastatefile

import (
	"context"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestStoreV0RecuperaRequiredTestEvidenceV0(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	evidence := stateFileRequiredTestEvidenceV0()
	if err := store.SaveRequiredTestEvidenceV0(context.Background(), evidence); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}
	recovered := mustStoreV0(t, rootDir)
	got, err := recovered.LoadRequiredTestEvidenceV0(context.Background(), evidence.RunRef, []string{evidence.EvidenceRef})
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(got) != 1 || got[0].EvidenceRef != evidence.EvidenceRef || got[0].Status != evidence.Status {
		t.Fatalf("evidence=%+v", got)
	}
}

func TestStoreV0RequiredTestEvidenceSaveEsIdempotenteYRechazaConflicto(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	evidence := stateFileRequiredTestEvidenceV0()
	if err := store.SaveRequiredTestEvidenceV0(context.Background(), evidence); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}
	if err := store.SaveRequiredTestEvidenceV0(context.Background(), evidence); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0 idempotente: %v", err)
	}
	changed := evidence
	changed.Status = orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0
	if err := store.SaveRequiredTestEvidenceV0(context.Background(), changed); err == nil {
		t.Fatal("err=nil, want conflicto durable")
	}
}

func TestStoreV0RechazaRequiredTestEvidenceConRefInternaInconsistente(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	evidence := stateFileRequiredTestEvidenceV0()
	document := requiredTestEvidenceDocumentV0{
		SchemaVersion: requiredTestEvidenceDocumentSchemaV0,
		RunRef:        evidence.RunRef,
		EvidenceRef:   evidence.EvidenceRef,
		Evidence:      evidence,
	}
	document.Evidence.RunRef = "run-state-file-distinto"
	if err := writeJSONAtomicV0(store.requiredTestEvidencePathV0(evidence.RunRef, evidence.EvidenceRef), document); err != nil {
		t.Fatalf("writeJSONAtomicV0: %v", err)
	}
	if _, err := store.LoadRequiredTestEvidenceV0(context.Background(), evidence.RunRef, []string{evidence.EvidenceRef}); err == nil {
		t.Fatal("err=nil, want ref interna inconsistente")
	}
}

func stateFileRequiredTestEvidenceV0() orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       "test-evidence-ref-state-file-001",
		RunRef:            "run-state-file-required-test-001",
		TaskRef:           "task-ref-state-file-required-test-001",
		TestCommand:       "go test ./...",
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       "delivery-ref-state-file-required-test-001",
		ReviewRequestID:   "review-request-ref-state-file-required-test-001",
		ReviewResultRef:   "review-result-ref-state-file-required-test-001",
		AcceptedReviewRef: "accepted-review-ref-state-file-required-test-001",
		OccurredAt:        "2026-05-17T15:40:00Z",
		EvidenceRefs:      []string{"evidence-ref-state-file-required-test-001"},
	}
}
