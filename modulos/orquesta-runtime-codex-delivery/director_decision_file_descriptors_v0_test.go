package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
)

func TestCodexReceiptDirectorDecisionFileDescriptorProviderV0MapeaDecisionJuntoAlACK(t *testing.T) {
	baseDir := t.TempDir()
	ackPath := filepath.Join(baseDir, "run-ref-001", "agent-ref-001", "agent_ack.json")
	writeCodexReceiptDecisionFilesForTestV0(t, ackPath, DefaultDirectorAgentDecisionFileNameV0)
	store := &directorDecisionReceiptStoreForTestV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "codex-receipt-ref-run-ref-001-agent-ref-001",
			RunID:         "run-ref-001",
			AgentRef:      "agent-ref-001",
			AckPath:       ackPath,
		}},
	}
	provider := CodexReceiptDirectorDecisionFileDescriptorProviderV0{Store: store}

	got, err := provider.ListDirectorAgentDecisionFilesV0(
		context.Background(),
		orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0{
			RunID:         " run-ref-001 ",
			CorrelationID: " corr-ref-001 ",
		},
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionFilesV0: %v", err)
	}
	if store.LastRequest.RunID != "run-ref-001" || store.LastRequest.CorrelationID != "corr-ref-001" {
		t.Fatalf("request=%+v", store.LastRequest)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
	wantPath := filepath.Join(baseDir, "run-ref-001", "agent-ref-001", DefaultDirectorAgentDecisionFileNameV0)
	if got[0].DescriptorRef != "codex-receipt-ref-run-ref-001-agent-ref-001-director-decisions" {
		t.Fatalf("descriptor_ref=%q", got[0].DescriptorRef)
	}
	if got[0].RunID != "run-ref-001" || got[0].Path != wantPath {
		t.Fatalf("descriptor=%+v want_path=%q", got[0], wantPath)
	}
}

func TestCodexReceiptDirectorDecisionFileDescriptorProviderV0FiltraPorRun(t *testing.T) {
	baseDir := t.TempDir()
	keepAckPath := filepath.Join(baseDir, "keep", "agent_ack.json")
	writeCodexReceiptDecisionFilesForTestV0(t, keepAckPath, "custom_decisions.json")
	store := &directorDecisionReceiptStoreForTestV0{
		Descriptors: []CodexReceiptDescriptorV0{
			{
				DescriptorRef: "receipt-keep",
				RunID:         "run-keep",
				AgentRef:      "agent-001",
				AckPath:       keepAckPath,
			},
			{
				DescriptorRef: "receipt-skip",
				RunID:         "run-skip",
				AgentRef:      "agent-002",
				AckPath:       filepath.Join(baseDir, "skip", "agent_ack.json"),
			},
		},
	}
	provider := CodexReceiptDirectorDecisionFileDescriptorProviderV0{
		Store:    store,
		FileName: "custom_decisions.json",
	}

	got, err := provider.ListDirectorAgentDecisionFilesV0(
		context.Background(),
		orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0{RunID: "run-keep"},
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionFilesV0: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
	if got[0].DescriptorRef != "receipt-keep-director-decisions" {
		t.Fatalf("descriptor_ref=%q", got[0].DescriptorRef)
	}
	if !strings.HasSuffix(got[0].Path, filepath.Join("keep", "custom_decisions.json")) {
		t.Fatalf("path=%q", got[0].Path)
	}
}

func TestCodexReceiptDirectorDecisionFileDescriptorProviderV0OmiteDecisionAusenteConACKExistente(t *testing.T) {
	baseDir := t.TempDir()
	ackPath := filepath.Join(baseDir, "run-ref-001", "agent-ref-001", "agent_ack.json")
	writeCodexReceiptFileForTestV0(t, ackPath, []byte(`{"status":"completed"}`))
	store := &directorDecisionReceiptStoreForTestV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "codex-receipt-ref-run-ref-001-agent-ref-001",
			RunID:         "run-ref-001",
			AgentRef:      "agent-ref-001",
			AckPath:       ackPath,
		}},
	}
	provider := CodexReceiptDirectorDecisionFileDescriptorProviderV0{Store: store}

	got, err := provider.ListDirectorAgentDecisionFilesV0(
		context.Background(),
		orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0{RunID: "run-ref-001"},
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionFilesV0: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len=%d want 0: %+v", len(got), got)
	}
}

func TestCodexReceiptDirectorDecisionFileDescriptorProviderV0RechazaFileNameInvalido(t *testing.T) {
	store := &directorDecisionReceiptStoreForTestV0{}
	provider := CodexReceiptDirectorDecisionFileDescriptorProviderV0{
		Store:    store,
		FileName: "../director_decisions.json",
	}

	_, err := provider.ListDirectorAgentDecisionFilesV0(
		context.Background(),
		orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0{RunID: "run-ref-001"},
	)
	if err == nil || !strings.Contains(err.Error(), "file_name") {
		t.Fatalf("err=%v want file_name", err)
	}
	if store.Calls != 0 {
		t.Fatalf("store calls=%d want 0", store.Calls)
	}
}

type directorDecisionReceiptStoreForTestV0 struct {
	Descriptors []CodexReceiptDescriptorV0
	LastRequest CodexReceiptDescriptorRequestV0
	Calls       int
}

func (store *directorDecisionReceiptStoreForTestV0) ListCodexReceiptDescriptorsV0(
	_ context.Context,
	request CodexReceiptDescriptorRequestV0,
) ([]CodexReceiptDescriptorV0, error) {
	store.Calls++
	store.LastRequest = request
	return append([]CodexReceiptDescriptorV0(nil), store.Descriptors...), nil
}

func writeCodexReceiptDecisionFilesForTestV0(t *testing.T, ackPath string, fileName string) {
	t.Helper()
	writeCodexReceiptFileForTestV0(t, ackPath, []byte(`{"status":"completed"}`))
	writeCodexReceiptFileForTestV0(t, filepath.Join(filepath.Dir(ackPath), fileName), []byte(`[]`))
}

func writeCodexReceiptFileForTestV0(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
