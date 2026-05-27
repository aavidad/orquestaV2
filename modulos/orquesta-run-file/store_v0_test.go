package orquestarunfile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestRunFileStoreRunControlPreservaEvidenciasPreviasV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()

	if _, err := store.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:       "run-evidence-history",
		RequestedBy:  "operador",
		Reason:       "parada solicitada",
		EvidenceRefs: []string{"evidence-ref-stop-origin"},
	}); err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	state, err := store.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       "run-evidence-history",
		TargetStatus: orquestaruncontrol.RunControlStatusStoppedV0,
		RequestedBy:  "orquesta-run-control",
		Reason:       "agentes drenados",
		EvidenceRefs: []string{"evidence-ref-stop-complete"},
	})
	if err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}
	if !runFileStringInSetV0(state.EvidenceRefs, "evidence-ref-stop-origin") ||
		!runFileStringInSetV0(state.EvidenceRefs, "evidence-ref-stop-complete") {
		t.Fatalf("state evidence=%+v", state.EvidenceRefs)
	}
}

func TestRunFileStoreLimitaLecturaSnapshotV0(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, runFileControlNameV0)
	if err := os.WriteFile(path, []byte(strings.Repeat("x", int(runFileSnapshotMaxBytesV0)+1)), 0o600); err != nil {
		t.Fatalf("write oversized snapshot: %v", err)
	}
	_, err := NewRunFileStoreV0(dir)
	if err == nil || err.Error() != "orquesta_run_file: size_limit_exceeded" {
		t.Fatalf("err=%v", err)
	}
}

func TestRunFileStoreLimitaRecordsSnapshotV0(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, runFileControlNameV0)
	records := make([]string, 0, runFileSnapshotMaxRecordsV0+1)
	for index := 0; index <= runFileSnapshotMaxRecordsV0; index++ {
		records = append(records, `{}`)
	}
	body := `{"schema_version":"` + runFileControlSchemaVersionV0 +
		`","records":[` + strings.Join(records, ",") + `]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	_, err := NewRunFileStoreV0(dir)
	if err == nil || err.Error() != "orquesta_run_file: control_records_limit_exceeded" {
		t.Fatalf("err=%v", err)
	}
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

func runFileStringInSetV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
