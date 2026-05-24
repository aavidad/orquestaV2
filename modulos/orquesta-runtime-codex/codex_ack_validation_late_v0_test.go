package orquestaruntimecodex

import "testing"

func TestCodexAgentAckReceiptV0RechazaCompletedConTestsFallidos(t *testing.T) {
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
			name: "notes evidencia fallo de test",
			data: `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["go test ./... failed"]}`,
		},
		{
			name: "tests evidencia salida fallida",
			data: `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./... exit status 1"]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(tc.data), spec)
			requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
		})
	}
}

func TestCodexAgentAckReceiptV0RechazaCompletedConContextoRequeridoTruncadoSinJustificar(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Context.Entries[0].Truncated = true

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(codexValidAckJSONV0()), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
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

func TestCodexAgentAckReceiptV0AceptaMarcadoresDudososComoRailPendiente(t *testing.T) {
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
		if !codexAckBytesContainForbiddenDetailV0([]byte(data)) {
			t.Fatalf("rail pendiente no detecto %s", fragment)
		}
	}
}

func TestCodexAgentAckReceiptV0RechazaValoresSensiblesEfectivos(t *testing.T) {
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
		requireCodexIssueV0(t, issues, CodexConnectorAckForbiddenV0)
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
