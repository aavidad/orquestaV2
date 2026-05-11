package orquestaruntimecodex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCodexAgentAckFileV0(t *testing.T) {
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"modulo","task_ref":"task-ref-001","status":"completed"}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	ack, err := ReadCodexAgentAckFileV0(path)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if !ack.ValidFor("req-codex-001") {
		t.Fatalf("ack invalido: %+v", ack)
	}
}

func TestReadCodexAgentAckFileV0NormalizaEvidenciaEstructurada(t *testing.T) {
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"modulo","task_ref":"task-ref-001","status":"completed","files":[{"path":"README.md","action":"created"}],"tests":[{"command":"go test ./...","status":"failed"}],"notes":[{"ref":"quality-finding","severity":"high"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	ack, err := ReadCodexAgentAckFileV0(path)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if len(ack.Files) != 1 || ack.Files[0] != "README.md" ||
		len(ack.Tests) != 1 || ack.Tests[0] != "go test ./..." ||
		len(ack.Notes) != 1 || ack.Notes[0] != "quality-finding" {
		t.Fatalf("evidencia no normalizada: %+v", ack)
	}
}
