package orquestaruntimecodex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestReadCodexControlFileBytesV0RechazaSobrelimite(t *testing.T) {
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	data := strings.Repeat("x", int(CodexControlFileMaxBytesV0)+1)
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := ReadCodexControlFileBytesV0(path, CodexAgentAckFileNameV0)
	if err == nil {
		t.Fatalf("esperaba error de sobrelimite")
	}
	reason, retryable := CodexControlFileReadErrorReasonV0(err)
	if reason != "control_file_too_large" || retryable {
		t.Fatalf("reason=%q retryable=%v", reason, retryable)
	}
}

func TestReadCodexControlFileBytesV0RechazaNombreInesperado(t *testing.T) {
	path := filepath.Join(t.TempDir(), "otro_ack.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := ReadCodexControlFileBytesV0(path, CodexAgentAckFileNameV0)

	requireCodexControlFileReadReasonV0(t, err, "file_name_mismatch", false)
}

func TestReadCodexControlFileBytesV0RechazaSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	path := filepath.Join(dir, CodexAgentAckFileNameV0)
	if err := os.WriteFile(target, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink no disponible: %v", err)
	}

	_, err := ReadCodexControlFileBytesV0(path, CodexAgentAckFileNameV0)

	requireCodexControlFileReadReasonV0(t, err, "symlink", false)
}

func TestReadCodexControlFileBytesV0RechazaHardlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	path := filepath.Join(dir, CodexAgentAckFileNameV0)
	if err := os.WriteFile(target, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Link(target, path); err != nil {
		t.Skipf("hardlink no disponible: %v", err)
	}

	_, err := ReadCodexControlFileBytesV0(path, CodexAgentAckFileNameV0)

	requireCodexControlFileReadReasonV0(t, err, "hardlink", false)
}

func TestReadAndValidateCodexAgentAckFileV0SobrelimiteIssueCompacto(t *testing.T) {
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	data := strings.Repeat("x", int(CodexControlFileMaxBytesV0)+1)
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	spec := codexControlFileReadSpecV0()

	_, issues := ReadAndValidateCodexAgentAckFileV0(path, spec)
	if len(issues) != 1 {
		t.Fatalf("issues=%+v", issues)
	}
	issue := issues[0]
	if issue.Field != CodexAgentAckFileNameV0 || issue.Retryable {
		t.Fatalf("issue=%+v", issue)
	}
	if !codexAckIssueHasEvidenceV0(issue, "control_file_too_large") {
		t.Fatalf("issue sin reason compacto: %+v", issue)
	}
	if strings.Contains(strings.Join(issue.Evidence, " "), path) {
		t.Fatalf("issue filtra path local: %+v", issue)
	}
}

func TestReadCodexShutdownCheckpointAckFileV0SobrelimiteIssueCompacto(t *testing.T) {
	path := filepath.Join(t.TempDir(), CodexShutdownCheckpointAckFileNameV0)
	data := strings.Repeat("x", int(CodexControlFileMaxBytesV0)+1)
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	request := CodexShutdownRequestV0{
		SchemaVersion:      CodexShutdownRequestSchemaVersionV0,
		RunRef:             "run-ref-001",
		AgentRef:           "agent-ref-001",
		CorrelationID:      "corr-001",
		ShutdownAttemptRef: "shutdown-attempt-ref-001",
		CheckpointRef:      "checkpoint-ref-001",
	}

	_, issues := ReadCodexShutdownCheckpointAckFileV0(path, request)
	if len(issues) != 1 || !codexAckIssueHasEvidenceV0(issues[0], "control_file_too_large") {
		t.Fatalf("issues=%+v", issues)
	}
}

func requireCodexControlFileReadReasonV0(t *testing.T, err error, want string, wantRetryable bool) {
	t.Helper()
	if err == nil {
		t.Fatalf("esperaba error %q", want)
	}
	reason, retryable := CodexControlFileReadErrorReasonV0(err)
	if reason != want || retryable != wantRetryable {
		t.Fatalf("reason=%q retryable=%v want reason=%q retryable=%v", reason, retryable, want, wantRetryable)
	}
}

func codexControlFileReadSpecV0() orquestaruntime.ExternalAgentLaunchSpecV0 {
	packet := orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     "agent-ref-001",
		CorrelationID: "corr-001",
		TargetModule:  "modulo",
		Task: orquestaruntime.AgentStartTaskV0{
			TaskRef: "task-ref-001",
		},
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			AckRef: "ack-ref-001",
		},
	}
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		RequestID:     packet.RequestID,
		CorrelationID: packet.CorrelationID,
		AgentPacket:   packet,
	}
}
