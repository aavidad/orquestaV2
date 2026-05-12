package orquestarunfile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestRunFileStoreSatisfiesPortsV0(t *testing.T) {
	store := mustNewRunFileStoreV0(t, t.TempDir())

	var _ orquestaruncontrol.RunControlPortV0 = store
	var _ orquestarunqueue.RunQueueReaderPortV0 = store
	var _ orquestarunqueue.RunQueuePriorityWriterPortV0 = store
	var _ orquestarunqueue.RunQueuePortV0 = store
	var _ orquestaappchange.AppChangeRecordStorePortV0 = store
}

func TestRunFileStoreWritesStructuredJSONSnapshotsV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	now := time.Date(2026, 5, 12, 8, 0, 0, 0, time.UTC)

	if _, err := store.PauseRunV0(context.Background(), orquestaruncontrol.PauseRunCommandV0{
		RunRef: "run-json",
	}); err != nil {
		t.Fatalf("PauseRunV0: %v", err)
	}
	if _, err := store.SetRunPriorityV0(context.Background(), orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-json",
		QueueRef:      "global",
		AppRef:        "app-json",
		PriorityScore: 10,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	if err := store.SaveAppChangeRequestV0(
		context.Background(),
		appChangeRecordForRunFileTestV0("run-json", "change-json"),
	); err != nil {
		t.Fatalf("SaveAppChangeRequestV0: %v", err)
	}

	assertStructuredSnapshotV0(t, filepath.Join(dir, runFileControlNameV0))
	assertStructuredSnapshotV0(t, filepath.Join(dir, runFileQueueNameV0))
	assertStructuredSnapshotV0(t, filepath.Join(dir, runFileAppChangeNameV0))
}

func mustNewRunFileStoreV0(t *testing.T, dir string) *RunFileStoreV0 {
	t.Helper()
	store, err := NewRunFileStoreV0(dir)
	if err != nil {
		t.Fatalf("NewRunFileStoreV0: %v", err)
	}
	return store
}

func assertStructuredSnapshotV0(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json %s: %v", path, err)
	}
	if len(raw["schema_version"]) == 0 {
		t.Fatalf("%s missing schema_version", path)
	}
	var records []json.RawMessage
	if err := json.Unmarshal(raw["records"], &records); err != nil {
		t.Fatalf("%s records invalid: %v", path, err)
	}
	if len(records) == 0 {
		t.Fatalf("%s records empty", path)
	}
}
