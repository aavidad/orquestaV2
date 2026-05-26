package orquestaruntimecodex

import (
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestStrictCompletedCodexAgentAckV0RechazaACKMinimoHidratableV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.RequiredTests = nil
	ack := `{"schema_version":"codex_agent_ack.v0","status":"completed"}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckInvalidV0)
}

func TestStrictCompletedCodexAgentAckV0RechazaCorrelacionNormalizableV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-erronea","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
}

func TestStrictCompletedCodexAgentAckV0RechazaEvidenciaAusenteV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed"}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
}

func TestStrictCompletedCodexAgentAckV0BloqueaTestFallidoEstructuradoV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":[{"command":"go test ./...","status":"failed"}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
}

func TestStrictCompletedCodexAgentAckV0RequiereReciboDeTestV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueEvidenceV0(t, issues, "missing_required_test_receipt")
}

func TestStrictCompletedCodexAgentAckV0RechazaReciboConSalidaCrudaV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true,"stdout":"ok"}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckForbiddenV0)
}

func TestStrictCompletedCodexAgentAckV0AceptaACKCompletoSinIssuesV0(t *testing.T) {
	spec := codexSpecForTestV0()

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(codexValidAckJSONV0()), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestStrictCompletedCodexAgentAckV0AceptaFileFueraDeWriteSetSeguroV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["docs/no-autorizado.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("file fuera de write-set seguro debe llegar a review como rail blando: %+v", issues)
	}
}

func TestStrictCompletedCodexAgentAckV0AceptaSinFilesConRecibosYNotasV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":[],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}],"notes":["sin cambios de producto; validacion completada con recibos"]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("ACK sin files pero con recibos/notas debe llegar a review: %+v", issues)
	}
}

func TestStrictCompletedCodexAgentAckV0NoBloqueaRailsPendientesGenericosV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}],"notes":["rail pendiente: token provider home prompt sin valor operativo"]}`

	validated, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("rail pendiente generico no debe bloquear ACK estricto: %+v", issues)
	}
	if !CodexAgentAckHasPendingRailV0(validated) {
		t.Fatalf("rail pendiente generico debe conservarse como evidencia")
	}
}

func TestCodexDeliveryObservationV0RechazaACKStrictSinRecibosV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["contexto_ref_only_resuelto: fixture local sin contexto externo"]}`

	_, regularIssues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)
	_, strictIssues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)
	_, issues := validateCodexDeliveryAckBytesForSpecV0([]byte(ack), spec)

	if len(regularIssues) != 0 {
		t.Fatalf("lectura diagnostica legacy debe seguir tolerante: %+v", regularIssues)
	}
	if !codexDeliveryObservationIssuesOnlyMissingTestReceiptV0(strictIssues) {
		t.Fatalf("strict debe identificar solo recibo faltante: %+v", strictIssues)
	}
	if !codexDeliveryObservationIssuesOnlyMissingTestReceiptV0(issues) {
		t.Fatalf("delivery terminal strict no debe degradar a legacy: strict=%+v delivery=%+v", strictIssues, issues)
	}
}

func TestStrictCompletedCodexAgentAckV0RechazaTestsExtraV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./...","node --check web/app.js"]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
}

func TestCodexAgentPacketRequiresStrictTerminalAckV0PorPolicyV0(t *testing.T) {
	spec := codexSpecForTestV0()

	if !CodexAgentPacketRequiresStrictTerminalAckV0(spec.AgentPacket) {
		t.Fatalf("policy strict no detectada: %+v", spec.AgentPacket.Policies)
	}

	spec.AgentPacket.Policies = nil
	if CodexAgentPacketRequiresStrictTerminalAckV0(spec.AgentPacket) {
		t.Fatalf("packet legacy no debe activar strict")
	}
}

func requireCodexIssueEvidenceV0(
	t *testing.T,
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
	evidence string,
) {
	t.Helper()
	if codexAckIssuesContainEvidenceV0(issues, evidence) {
		return
	}
	t.Fatalf("issues sin evidence %q: %+v", evidence, issues)
}
