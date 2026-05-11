package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileCodexReceiptDescriptorStoreV0RecuperaTrasRecrearInstancia(t *testing.T) {
	dir := t.TempDir()
	spec := codexDeliverySpecForTestV0()
	store := newFileCodexReceiptDescriptorStoreForTestV0(t, dir)
	descriptor := CodexReceiptDescriptorV0{
		DescriptorRef:       "receipt-ref-001",
		RunID:               "run-ref-001",
		AgentRef:            spec.RequestID,
		Spec:                spec,
		AckPath:             filepath.Join(t.TempDir(), "agent_ack.json"),
		ProjectWorkDir:      t.TempDir(),
		WorktreeBaselineRef: "worktree-snapshot-ref-001",
	}
	if err := store.RecordCodexReceiptDescriptorV0(context.Background(), descriptor); err != nil {
		t.Fatalf("RecordCodexReceiptDescriptorV0: %v", err)
	}

	reopened := newFileCodexReceiptDescriptorStoreForTestV0(t, dir)
	got, err := reopened.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID:         "run-ref-001",
		StartedAgents: []string{spec.RequestID},
	})
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(got) != 1 ||
		got[0].AckPath != descriptor.AckPath ||
		got[0].WorktreeBaselineRef != descriptor.WorktreeBaselineRef {
		t.Fatalf("descriptors=%+v", got)
	}
}

func TestFileCodexReceiptDescriptorStoreV0ReemplazaDescriptorPorRef(t *testing.T) {
	dir := t.TempDir()
	spec := codexDeliverySpecForTestV0()
	store := newFileCodexReceiptDescriptorStoreForTestV0(t, dir)
	first := CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       filepath.Join(t.TempDir(), "agent_ack_1.json"),
	}
	second := first
	second.AckPath = filepath.Join(t.TempDir(), "agent_ack_2.json")

	if err := store.RecordCodexReceiptDescriptorV0(context.Background(), first); err != nil {
		t.Fatalf("record first: %v", err)
	}
	if err := store.RecordCodexReceiptDescriptorV0(context.Background(), second); err != nil {
		t.Fatalf("record second: %v", err)
	}
	got, err := store.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID:         "run-ref-001",
		StartedAgents: []string{spec.RequestID},
	})
	if err != nil || len(got) != 1 || got[0].AckPath != second.AckPath {
		t.Fatalf("descriptors=%+v err=%v", got, err)
	}
}

func TestFileCodexReceiptDescriptorStoreV0NoFiltraPathEnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileCodexReceiptDescriptorStoreNameV0)
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}

	_, err := NewFileCodexReceiptDescriptorStoreV0(dir)
	if err == nil {
		t.Fatalf("esperaba error")
	}
	if strings.Contains(err.Error(), dir) || strings.Contains(err.Error(), path) {
		t.Fatalf("error filtra path: %v", err)
	}
}

func newFileCodexReceiptDescriptorStoreForTestV0(
	t *testing.T,
	dir string,
) *FileCodexReceiptDescriptorStoreV0 {
	t.Helper()
	store, err := NewFileCodexReceiptDescriptorStoreV0(dir)
	if err != nil {
		t.Fatalf("NewFileCodexReceiptDescriptorStoreV0: %v", err)
	}
	return store
}
