package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"orquesta/internal/wizard/gaps"
)

func TestWizardGapsApplicationBindsExactResultSnapshot(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	request := wizardGapsRequest(t, system, "request:wizard-gaps-result-snapshot", 1)
	first, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	record := mustWizardGapsInputRecord(t, system, request.RequestRef)
	assertWizardGapsResultSnapshot(t, record.Receipt, first)

	replayed, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Changed || replayed.EvaluationReplayExact {
		t.Fatalf("replayed changed=%t exact=%t", replayed.Changed, replayed.EvaluationReplayExact)
	}
	assertWizardGapsResultSnapshot(t, record.Receipt, replayed)
	t.Run("noncanonical bytes with matching digest", func(t *testing.T) {
		mutated := cloneWizardGapsResultSnapshot(record.Receipt.ResultSnapshot)
		var indented bytes.Buffer
		if err := json.Indent(&indented, mutated.Bytes, "", "  "); err != nil {
			t.Fatal(err)
		}
		mutated.Bytes = indented.Bytes()
		digest := sha256.Sum256(mutated.Bytes)
		mutated.Digest = hex.EncodeToString(digest[:])
		mutated.Ref = wizardGapsResultSnapshotRef(mutated.InputReceiptRef, mutated.Digest)
		if _, err := validateWizardGapsResultSnapshot(record.Receipt.Ref, mutated, first.Evaluation); err == nil {
			t.Fatal("noncanonical snapshot accepted")
		}
	})
	t.Run("foreign input receipt", func(t *testing.T) {
		foreign := cloneWizardGapsResultSnapshot(record.Receipt.ResultSnapshot)
		foreign.InputReceiptRef = "wizard-gaps-input:" + string(bytes.Repeat([]byte{'a'}, 64))
		foreign.Ref = wizardGapsResultSnapshotRef(foreign.InputReceiptRef, foreign.Digest)
		if _, err := validateWizardGapsResultSnapshot(record.Receipt.Ref, foreign, first.Evaluation); err == nil {
			t.Fatal("snapshot from another input receipt accepted")
		}
	})
	t.Run("foreign evaluation", func(t *testing.T) {
		foreignResult, err := gaps.Evaluate(gaps.Input{
			Facts: gaps.Facts{Surface: gaps.SurfaceServerService},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := validateWizardGapsResultSnapshot(record.Receipt.Ref, record.Receipt.ResultSnapshot, foreignResult); err == nil {
			t.Fatal("snapshot from another evaluation accepted")
		}
	})
}

func assertWizardGapsResultSnapshot(t *testing.T, receipt WizardGapsInputReceipt, result ApplyWizardGapsResult) {
	t.Helper()
	snapshot := receipt.ResultSnapshot
	if wizardGapsResultSnapshotEmpty(snapshot) ||
		snapshot.InputReceiptRef != receipt.Ref ||
		result.EvaluationReplayExact ||
		result.EvaluationSnapshot.Ref != snapshot.Ref ||
		result.EvaluationSnapshot.Digest != snapshot.Digest ||
		!bytes.Equal(result.EvaluationSnapshot.Bytes, snapshot.Bytes) {
		t.Fatalf("receipt_snapshot=%+v result_snapshot=%+v exact=%t", snapshot, result.EvaluationSnapshot, result.EvaluationReplayExact)
	}
	canonical, digest, err := gaps.MarshalResultSnapshot(result.Evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if digest != snapshot.Digest || !bytes.Equal(canonical, snapshot.Bytes) {
		t.Fatal("result snapshot does not bind canonical evaluation")
	}
	if _, err := validateWizardGapsResultSnapshot(receipt.Ref, snapshot, result.Evaluation); err != nil {
		t.Fatalf("validate result snapshot: %v", err)
	}
}
