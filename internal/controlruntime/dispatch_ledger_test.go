package controlruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordDispatchLedgerFromMetadataJSONAndFindLatest(t *testing.T) {
	tmp := t.TempDir()
	metaRaw, err := json.Marshal(map[string]any{
		"trace_dir": tmp,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	if err := RecordDispatchLedgerFromMetadataJSON(string(metaRaw), DispatchLedgerRecordInput{
		RuntimeOrderID:           11,
		MailboxID:                91,
		HandleID:                 7,
		DeliveryAttemptSignature: "session_resume|handle:7|session:sess-a",
		ExternalSessionID:        "sess-a",
		DispatchState:            "notified",
		DeliveryState:            "notified",
		Reason:                   "mailbox pending receipt",
	}); err != nil {
		t.Fatalf("record notified: %v", err)
	}
	if err := RecordDispatchLedgerFromMetadataJSON(string(metaRaw), DispatchLedgerRecordInput{
		RuntimeOrderID:           11,
		MailboxID:                91,
		HandleID:                 7,
		DeliveryAttemptSignature: "session_resume|handle:7|session:sess-a",
		ExternalSessionID:        "sess-a",
		DispatchState:            "delivered",
		DeliveryState:            "delivered",
		ReceiptSource:            "last_progress",
	}); err != nil {
		t.Fatalf("record delivered: %v", err)
	}

	entry, err := FindDispatchLedgerEntryFromMetadataJSON(string(metaRaw), 91, "session_resume|handle:7|session:sess-a", "")
	if err != nil {
		t.Fatalf("find latest: %v", err)
	}
	if entry == nil {
		t.Fatal("faltaba entry")
	}
	if entry.DeliveryState != "delivered" || entry.ReceiptSource != "last_progress" {
		t.Fatalf("entry inesperada: %+v", entry)
	}
}

func TestDispatchLedgerPathFromMetadataJSONFallsBackToWorkerArtifacts(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	metaRaw, err := json.Marshal(map[string]any{
		"worker_manifest_path": manifestPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	path := DispatchLedgerPathFromMetadataJSON(string(metaRaw))
	want := filepath.Join(tmp, "dispatch-ledger.json")
	if path != want {
		t.Fatalf("path=%q want=%q", path, want)
	}
}
