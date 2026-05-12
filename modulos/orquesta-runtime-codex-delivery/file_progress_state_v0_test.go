package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestFileCodexProgressStateStoreV0RecuperaHeartbeatTrasRecrearInstancia(t *testing.T) {
	dir := t.TempDir()
	store := newFileCodexProgressStateStoreForTestV0(t, dir)
	sample := codexProgressSampleForFileStoreTestV0("firma-uno")
	first, err := store.ObserveCodexProgressV0(context.Background(), sample)
	if err != nil {
		t.Fatalf("observe first: %v", err)
	}
	if first.Current.TickCounter != 1 || first.Previous != nil {
		t.Fatalf("first=%+v", first)
	}
	if !first.SampleAccepted || first.ObservedAt.IsZero() || first.FirstObservedAt.IsZero() {
		t.Fatalf("first state incompleto=%+v", first)
	}

	reopened := newFileCodexProgressStateStoreForTestV0(t, dir)
	second, err := reopened.ObserveCodexProgressV0(context.Background(), sample)
	if err != nil {
		t.Fatalf("observe second: %v", err)
	}
	if second.Current.TickCounter != 2 ||
		second.Current.RepeatedActionCount != 0 ||
		second.Previous == nil {
		t.Fatalf("second=%+v", second)
	}
	if !second.SampleAccepted || second.ObservedAt.IsZero() {
		t.Fatalf("second state incompleto=%+v", second)
	}
}

func TestFileCodexProgressStateStoreV0RespetaIntervaloDeMuestra(t *testing.T) {
	dir := t.TempDir()
	store := newFileCodexProgressStateStoreForTestV0(t, dir)
	observedAt := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	sample := codexProgressSampleForFileStoreTestV0("firma-intervalo")
	sample.ObservedAt = observedAt
	sample.MinUnchangedInterval = time.Minute

	first, err := store.ObserveCodexProgressV0(context.Background(), sample)
	if err != nil {
		t.Fatalf("observe first: %v", err)
	}
	sample.ObservedAt = observedAt.Add(30 * time.Second)
	second, err := store.ObserveCodexProgressV0(context.Background(), sample)
	if err != nil {
		t.Fatalf("observe second: %v", err)
	}

	if !first.SampleAccepted || second.SampleAccepted {
		t.Fatalf("accepted first=%v second=%v", first.SampleAccepted, second.SampleAccepted)
	}
	if second.Current.TickCounter != first.Current.TickCounter ||
		!second.ObservedAt.Equal(first.ObservedAt) {
		t.Fatalf("second debe devolver estado previo sin avanzar: first=%+v second=%+v", first, second)
	}
}

func TestFileCodexProgressStateStoreV0RecuperaReporteMarcado(t *testing.T) {
	dir := t.TempDir()
	store := newFileCodexProgressStateStoreForTestV0(t, dir)
	sample := codexProgressSampleForFileStoreTestV0("firma-dos")
	if _, err := store.ObserveCodexProgressV0(context.Background(), sample); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := store.MarkCodexProgressReportedV0(context.Background(), CodexProgressReportMarkV0{
		RunID:          sample.RunID,
		AgentRequestID: sample.AgentRequestID,
		ProcessRef:     sample.ProcessRef,
		Signature:      sample.Signature,
		Status:         orquestaruntime.AgentStalledV0,
	}); err != nil {
		t.Fatalf("mark: %v", err)
	}

	reopened := newFileCodexProgressStateStoreForTestV0(t, dir)
	got, err := reopened.ObserveCodexProgressV0(context.Background(), sample)
	if err != nil {
		t.Fatalf("observe reopened: %v", err)
	}
	if got.ReportedSignature != sample.Signature ||
		got.ReportedStatus != orquestaruntime.AgentStalledV0 {
		t.Fatalf("reported=%+v", got)
	}
}

func TestFileCodexProgressStateStoreV0NoFiltraPathEnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileCodexProgressStateNameV0)
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}

	_, err := NewFileCodexProgressStateStoreV0(dir)
	if err == nil {
		t.Fatalf("esperaba error")
	}
	if strings.Contains(err.Error(), dir) || strings.Contains(err.Error(), path) {
		t.Fatalf("error filtra path: %v", err)
	}
}

func codexProgressSampleForFileStoreTestV0(signature string) CodexProgressSampleV0 {
	return CodexProgressSampleV0{
		RunID:          "run-ref-file-progress-001",
		AgentRequestID: "agent-ref-file-progress-001",
		ProcessRef:     "process-ref-file-progress-001",
		Signature:      signature,
		EvidenceRefs:   []string{"evidence-ref-file-progress-001"},
	}
}

func newFileCodexProgressStateStoreForTestV0(
	t *testing.T,
	dir string,
) *FileCodexProgressStateStoreV0 {
	t.Helper()
	store, err := NewFileCodexProgressStateStoreV0(dir)
	if err != nil {
		t.Fatalf("NewFileCodexProgressStateStoreV0: %v", err)
	}
	return store
}
