package orquestaruntimecodex

import (
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestCodexAgentAckReceiptV0AceptaCompletedConTestsFallidosComoRevisionV0(t *testing.T) {
	spec := codexSpecForTestV0()
	cases := []struct {
		name string
		data string
	}{
		{
			name: "tests declara status fallido",
			data: `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":[{"command":"go test ./...","status":"failed"}]}`,
		},
		{
			name: "tests evidencia salida fallida",
			data: `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./... exit status 1"]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(tc.data), spec)
			if len(issues) != 0 {
				t.Fatalf("test fallido estructurado debe llegar a review, no bloquear ACK: %+v", issues)
			}
		})
	}
}

func TestCodexAgentAckReceiptV0NotasLibresNoCreanFailedTestEvidenceV0(t *testing.T) {
	spec := codexSpecForTestV0()
	data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["go test failed earlier but now passed"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(data), spec)

	if len(issues) != 0 {
		t.Fatalf("nota libre no debe bloquear failed_test_evidence: %+v", issues)
	}
}

func TestCodexAgentAckDeclaresIncompleteRequiredEvidenceV0(t *testing.T) {
	cases := []CodexAgentAckV0{
		{
			SchemaVersion: CodexAgentAckSchemaVersionV0,
			Status:        "completed",
			Tests:         EvidenceListV0{"smoke OPES real no ejecutado por faltar entorno temporal"},
			Notes:         EvidenceListV0{"falta confirmacion explicita de instancia temporal"},
		},
		{
			SchemaVersion: CodexAgentAckSchemaVersionV0,
			Status:        "completed",
			Tests:         EvidenceListV0{"go test ./..."},
			Notes: EvidenceListV0{
				"T12 reconciliada como bloqueado verificable; faltan OPES temporal, Orquesta temporal, confirmacion de efectos y cuota/modelo",
			},
		},
	}
	for _, ack := range cases {
		if !CodexAgentAckDeclaresIncompleteRequiredEvidenceV0(ack) {
			t.Fatalf("debe detectar evidencia obligatoria incompleta: %+v", ack)
		}
	}
	ack := CodexAgentAckV0{
		SchemaVersion: CodexAgentAckSchemaVersionV0,
		Status:        "completed",
		Tests:         EvidenceListV0{"go test ./..."},
		Notes:         EvidenceListV0{"no tests failed; contexto_ref_only_resuelto; no faltan evidencias obligatorias"},
	}
	if CodexAgentAckDeclaresIncompleteRequiredEvidenceV0(ack) {
		t.Fatalf("no debe marcar completitud incompleta para nota inocua")
	}
	resolved := CodexAgentAckV0{
		SchemaVersion: CodexAgentAckSchemaVersionV0,
		Status:        "completed",
		Tests:         EvidenceListV0{"go test ./..."},
		Notes: EvidenceListV0{
			"contexto_ref_only_resuelto: agent_packet traia materialization_missing y ack_evidence_required resuelto mediante lectura local",
		},
	}
	if CodexAgentAckDeclaresIncompleteRequiredEvidenceV0(resolved) {
		t.Fatalf("no debe marcar incompleta una evidencia missing ya resuelta")
	}
}

func TestCodexAgentAckReceiptV0AceptaCompletedConContextoRequeridoTruncadoSinJustificar(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Context.Entries[0].Truncated = true

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(codexValidAckJSONV0()), spec)

	if len(issues) != 0 {
		t.Fatalf("contexto truncado debe quedar para review, no bloqueo ACK: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0AceptaCompletedConContextoTruncadoJustificado(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Context.Entries[0].Truncated = true
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["contexto_truncado_resuelto: source_refs y paquete externo suficientes"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0AceptaCompletedConRequiredRefOnlySinEvidencia(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Context.Entries[0].Mode = orquestacontext.ContextMaterializationModeRefOnlyV0
	spec.AgentPacket.Context.Entries[0].Content = ""
	spec.AgentPacket.Context.Entries[0].Bytes = 0
	spec.AgentPacket.Context.Entries[0].RefOnlyReason = orquestacontext.ContextRefOnlyReasonMaterializationMissingV0
	spec.AgentPacket.Context.Entries[0].RequiredRefAction = orquestacontext.ContextRequiredRefActionReadLocalV0

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(codexValidAckJSONV0()), spec)

	if len(issues) != 0 {
		t.Fatalf("contexto ref_only debe quedar para review, no bloqueo ACK: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0AceptaCompletedConRequiredRefOnlyResuelto(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Context.Entries[0].Mode = orquestacontext.ContextMaterializationModeRefOnlyV0
	spec.AgentPacket.Context.Entries[0].Content = ""
	spec.AgentPacket.Context.Entries[0].Bytes = 0
	spec.AgentPacket.Context.Entries[0].RefOnlyReason = orquestacontext.ContextRefOnlyReasonMaterializationMissingV0
	spec.AgentPacket.Context.Entries[0].RequiredRefAction = orquestacontext.ContextRequiredRefActionReadLocalV0
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["contexto_ref_only_resuelto: docs locales leidos desde workdir"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0AceptaMarcadoresDudososComoRailPendiente(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	spec := codexSpecForTestV0()
	cases := []string{
		`"notes":["rail pendiente: access_token policy sin valor"]`,
		`"notes":["rail pendiente: access_token=redacted sin valor real"]`,
		`"notes":["rail pendiente: client_secret: redacted"]`,
		`"notes":["rail pendiente: authorization: bearer redacted"]`,
		`"notes":["rail pendiente: secret: policy sin valor"]`,
		`"notes":["rail pendiente: home=redacted"]`,
		`"notes":["rail pendiente: prompt=redacted sin transcript"]`,
	}
	for _, fragment := range cases {
		data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],` + fragment + `}`
		_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(data), spec)
		if len(issues) != 0 {
			t.Fatalf("issues=%+v para %s", issues, fragment)
		}
	}
}

func TestCodexAgentAckReceiptV0RechazaValoresSensiblesEfectivos(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "off")
	spec := codexSpecForTestV0()
	forbidden := []string{
		`"notes":["access_token=abc123"]`,
		`"notes":["client_secret: valor"]`,
		`"notes":["authorization: bearer valor"]`,
		`"notes":["-----BEGIN PRIVATE KEY-----"]`,
	}
	for _, fragment := range forbidden {
		data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],` + fragment + `}`
		_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(data), spec)
		if !codexAckIssuesContainEvidenceV0(issues, "forbidden_sensitive_detail") {
			t.Fatalf("secreto efectivo debe bloquear fragment=%s issues=%+v", fragment, issues)
		}
	}
}

func TestCodexAgentAckReceiptV0RechazaPromptBrutoExtenso(t *testing.T) {
	spec := codexSpecForTestV0()
	rawPrompt := "prompt=" + strings.Repeat("instruccion interna completa ", 8)
	data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["` + rawPrompt + `"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(data), spec)

	if !codexAckIssuesContainEvidenceV0(issues, "forbidden_sensitive_detail") {
		t.Fatalf("prompt bruto extenso debe bloquear issues=%+v", issues)
	}
}

func TestCodexAgentAckReceiptV0NoBloqueaDetalleOperativoComoEvidenciaProducto(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "on")
	spec := codexSpecForTestV0()
	cases := []string{
		`"notes":["diagnostico publico en /home/alberto/proyecto"]`,
		`"tests":["go test ./...","cat .orquesta-runtime/agent_packet.json"]`,
		`"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["agent_ack.json"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]`,
	}
	for _, fragment := range cases {
		data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],` + fragment + `}`
		_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(data), spec)
		if len(issues) != 0 {
			t.Fatalf("detalle operativo no debe bloquear produccion issues=%+v fragment=%s", issues, fragment)
		}
	}
}

func TestCodexAgentAckReceiptV0PermiteVariablesYRefsOpacasDeEstado(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["estado via ${ORQUESTA_SERVER_STATE_DIR} y state-dir-ref-demo sin valor local"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0RechazaSinCorrelacionOutboxAgent(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := CodexAgentAckV0{
		SchemaVersion: CodexAgentAckSchemaVersionV0,
		RequestID:     "otro-request",
		CorrelationID: spec.CorrelationID,
		AckRef:        spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		TaskRef:       spec.AgentPacket.Task.TaskRef,
		Status:        codexAgentAckStatusCompletedV0,
		Files:         EvidenceListV0{"README.md"},
		Tests:         EvidenceListV0{"go test ./..."},
	}

	issues := ValidateCodexAgentAckForSpecV0(ack, spec)
	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
}
