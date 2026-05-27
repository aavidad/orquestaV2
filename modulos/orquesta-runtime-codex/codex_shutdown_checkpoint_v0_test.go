package orquestaruntimecodex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexShutdownCheckpointV0EscribeRequestYValidaAck(t *testing.T) {
	dir := t.TempDir()
	requestPath := filepath.Join(dir, CodexShutdownRequestFileNameV0)
	ackPath := filepath.Join(dir, CodexShutdownCheckpointAckFileNameV0)
	request := codexShutdownRequestForTestV0()

	if issues := WriteCodexShutdownRequestFileV0(requestPath, request); len(issues) != 0 {
		t.Fatalf("WriteCodexShutdownRequestFileV0 issues=%+v", issues)
	}
	if _, err := os.Stat(requestPath); err != nil {
		t.Fatalf("request no escrito: %v", err)
	}
	writeCodexShutdownAckForTestV0(t, ackPath, CodexShutdownCheckpointAckV0{
		SchemaVersion:      CodexShutdownCheckpointAckSchemaVersionV0,
		RunRef:             request.RunRef,
		AgentRef:           request.AgentRef,
		ShutdownAttemptRef: request.ShutdownAttemptRef,
		CheckpointRef:      request.CheckpointRef,
		Status:             CodexShutdownCheckpointStatusReadyV0,
		Summary:            "checkpoint coherente",
		EvidenceRefs:       []string{"evidence-ref-checkpoint"},
	})

	ack, issues := ReadCodexShutdownCheckpointAckFileV0(ackPath, request)
	if len(issues) != 0 {
		t.Fatalf("ReadCodexShutdownCheckpointAckFileV0 issues=%+v", issues)
	}
	if ack.CheckpointRef != request.CheckpointRef || ack.AgentRef != request.AgentRef {
		t.Fatalf("ack=%+v request=%+v", ack, request)
	}
}

func TestCodexShutdownCheckpointV0AckPendienteEsRetryable(t *testing.T) {
	_, issues := ReadCodexShutdownCheckpointAckFileV0(
		filepath.Join(t.TempDir(), CodexShutdownCheckpointAckFileNameV0),
		codexShutdownRequestForTestV0(),
	)
	if len(issues) != 1 {
		t.Fatalf("issues=%+v", issues)
	}
	if !issues[0].Retryable ||
		issues[0].Code != orquestaruntime.ExternalAgentConnectorErrorCodeV0(CodexConnectorAckInvalidV0) ||
		issues[0].Field != CodexShutdownCheckpointAckFileNameV0 {
		t.Fatalf("issue=%+v", issues[0])
	}
}

func TestCodexShutdownCheckpointV0RechazaAckDeOtroAgente(t *testing.T) {
	dir := t.TempDir()
	request := codexShutdownRequestForTestV0()
	ackPath := filepath.Join(dir, CodexShutdownCheckpointAckFileNameV0)
	writeCodexShutdownAckForTestV0(t, ackPath, CodexShutdownCheckpointAckV0{
		SchemaVersion:      CodexShutdownCheckpointAckSchemaVersionV0,
		RunRef:             request.RunRef,
		AgentRef:           "agent-ref-otro",
		ShutdownAttemptRef: request.ShutdownAttemptRef,
		CheckpointRef:      request.CheckpointRef,
		Status:             CodexShutdownCheckpointStatusReadyV0,
	})

	_, issues := ReadCodexShutdownCheckpointAckFileV0(ackPath, request)
	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
}

func TestCodexShutdownCheckpointV0PermiteDetalleConRailsDesactivados(t *testing.T) {
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "off")

	dir := t.TempDir()
	request := codexShutdownRequestForTestV0()
	ackPath := filepath.Join(dir, CodexShutdownCheckpointAckFileNameV0)
	writeCodexShutdownAckForTestV0(t, ackPath, CodexShutdownCheckpointAckV0{
		SchemaVersion:      CodexShutdownCheckpointAckSchemaVersionV0,
		RunRef:             request.RunRef,
		AgentRef:           request.AgentRef,
		ShutdownAttemptRef: request.ShutdownAttemptRef,
		CheckpointRef:      request.CheckpointRef,
		Status:             CodexShutdownCheckpointStatusReadyV0,
		Summary:            "checkpoint con token policy y prompt policy como refs operativas",
		EvidenceRefs:       []string{"evidence-ref-token-policy"},
	})

	_, issues := ReadCodexShutdownCheckpointAckFileV0(ackPath, request)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestCodexShutdownCheckpointV0ConRailsDetalleOffRechazaSecretoEfectivo(t *testing.T) {
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "off")

	dir := t.TempDir()
	request := codexShutdownRequestForTestV0()
	ackPath := filepath.Join(dir, CodexShutdownCheckpointAckFileNameV0)
	writeCodexShutdownAckForTestV0(t, ackPath, CodexShutdownCheckpointAckV0{
		SchemaVersion:      CodexShutdownCheckpointAckSchemaVersionV0,
		RunRef:             request.RunRef,
		AgentRef:           request.AgentRef,
		ShutdownAttemptRef: request.ShutdownAttemptRef,
		CheckpointRef:      request.CheckpointRef,
		Status:             CodexShutdownCheckpointStatusReadyV0,
		Summary:            "access_token=abc123",
	})

	_, issues := ReadCodexShutdownCheckpointAckFileV0(ackPath, request)
	requireCodexIssueV0(t, issues, CodexConnectorAckForbiddenV0)
}

func TestCodexShutdownCheckpointV0NoCortaPorRailsGenericos(t *testing.T) {
	dir := t.TempDir()
	request := codexShutdownRequestForTestV0()
	request.Reason = "shutdown por provider policy y prompt policy"
	request.EvidenceRefs = []string{"evidence-ref-token-policy"}
	ackPath := filepath.Join(dir, CodexShutdownCheckpointAckFileNameV0)
	writeCodexShutdownAckForTestV0(t, ackPath, CodexShutdownCheckpointAckV0{
		SchemaVersion:      CodexShutdownCheckpointAckSchemaVersionV0,
		RunRef:             request.RunRef,
		AgentRef:           request.AgentRef,
		ShutdownAttemptRef: request.ShutdownAttemptRef,
		CheckpointRef:      request.CheckpointRef,
		Status:             CodexShutdownCheckpointStatusReadyV0,
		Summary:            "checkpoint con token provider home prompt como refs blandas",
		EvidenceRefs:       []string{"evidence-ref-provider-policy"},
	})

	if issues := ValidateCodexShutdownRequestV0(request); len(issues) != 0 {
		t.Fatalf("request issues=%+v", issues)
	}
	_, issues := ReadCodexShutdownCheckpointAckFileV0(ackPath, request)
	if len(issues) != 0 {
		t.Fatalf("ack issues=%+v", issues)
	}
}

func TestCodexShutdownCheckpointV0RechazaDetalleSensibleEfectivo(t *testing.T) {
	dir := t.TempDir()
	request := codexShutdownRequestForTestV0()
	ackPath := filepath.Join(dir, CodexShutdownCheckpointAckFileNameV0)
	writeCodexShutdownAckForTestV0(t, ackPath, CodexShutdownCheckpointAckV0{
		SchemaVersion:      CodexShutdownCheckpointAckSchemaVersionV0,
		RunRef:             request.RunRef,
		AgentRef:           request.AgentRef,
		ShutdownAttemptRef: request.ShutdownAttemptRef,
		CheckpointRef:      request.CheckpointRef,
		Status:             CodexShutdownCheckpointStatusReadyV0,
		Summary:            "access_token=abc123",
	})

	_, issues := ReadCodexShutdownCheckpointAckFileV0(ackPath, request)
	requireCodexIssueV0(t, issues, CodexConnectorAckForbiddenV0)
}

func TestCodexShutdownCheckpointV0RechazaAckDeIntentoAnterior(t *testing.T) {
	dir := t.TempDir()
	request := codexShutdownRequestForTestV0()
	ackPath := filepath.Join(dir, CodexShutdownCheckpointAckFileNameV0)
	writeCodexShutdownAckForTestV0(t, ackPath, CodexShutdownCheckpointAckV0{
		SchemaVersion:      CodexShutdownCheckpointAckSchemaVersionV0,
		RunRef:             request.RunRef,
		AgentRef:           request.AgentRef,
		ShutdownAttemptRef: "shutdown-attempt-ref-anterior",
		CheckpointRef:      request.CheckpointRef,
		Status:             CodexShutdownCheckpointStatusReadyV0,
	})

	_, issues := ReadCodexShutdownCheckpointAckFileV0(ackPath, request)
	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
	if !codexAckIssueHasEvidenceV0(issues[0], "shutdown_checkpoint_attempt_mismatch") {
		t.Fatalf("issues sin mismatch de intento: %+v", issues)
	}
}

func codexShutdownRequestForTestV0() CodexShutdownRequestV0 {
	return CodexShutdownRequestV0{
		SchemaVersion:      CodexShutdownRequestSchemaVersionV0,
		RunRef:             "run-ref-shutdown-001",
		AgentRef:           "agent-ref-shutdown-001",
		CorrelationID:      "corr-shutdown-001",
		RequestedBy:        "test",
		Reason:             "shutdown controlado",
		ShutdownAttemptRef: "shutdown-attempt-ref-001",
		CheckpointRef:      "checkpoint-ref-shutdown-001",
		EvidenceRefs:       []string{"evidence-ref-shutdown"},
	}
}

func writeCodexShutdownAckForTestV0(
	t *testing.T,
	path string,
	ack CodexShutdownCheckpointAckV0,
) {
	t.Helper()
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
}
