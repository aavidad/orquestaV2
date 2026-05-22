package orquestaruntimecodex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodexAgentAckReceiptV0AceptaACKValido(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := codexValidAckJSONV0()

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0AceptaACKMinimoHidratadoDesdeSpec(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.RequiredTests = nil
	ack := `{"schema_version":"codex_agent_ack.v0","status":"completed"}`

	got, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	if got.RequestID != spec.RequestID ||
		got.CorrelationID != spec.CorrelationID ||
		got.AckRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		got.TargetModule != spec.AgentPacket.TargetModule ||
		got.TaskRef != spec.AgentPacket.Task.TaskRef {
		t.Fatalf("ack no hidratado desde spec: %+v spec=%+v", got, spec)
	}
}

func TestCodexAgentAckReceiptV0RechazaACKMinimoConRequiredTests(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","status":"completed"}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
}

func TestCodexAgentAckReceiptV0RechazaCorruptoEIncompleto(t *testing.T) {
	spec := codexSpecForTestV0()
	cases := []struct {
		name string
		data string
		code CodexConnectorIssueCodeV0
	}{
		{
			name: "json corrupto",
			data: `{"schema_version":`,
			code: CodexConnectorAckInvalidV0,
		},
		{
			name: "correlacion contradictoria",
			data: `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"otra-corr","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."]}`,
			code: CodexConnectorAckCorrelationV0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(tc.data), spec)
			requireCodexIssueV0(t, issues, tc.code)
		})
	}
}

func TestCodexAgentAckReceiptV0LeeArchivoYRechazaACKInvalidoConErrorPublico(t *testing.T) {
	spec := codexSpecForTestV0()
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	_, issues := ReadAndValidateCodexAgentAckFileV0(path, spec)
	requireCodexIssueV0(t, issues, CodexConnectorAckInvalidV0)
	for _, issue := range issues {
		if issue.MessageKey == "" || issue.CorrelationID != spec.CorrelationID {
			t.Fatalf("error no publico/correlado: %+v", issue)
		}
	}
}

func TestCodexAgentAckReceiptV0RechazaArtifactsFaltantesOFueraDeWriteSet(t *testing.T) {
	spec := codexSpecForTestV0()
	cases := []struct {
		name string
		data string
	}{
		{
			name: "sin artifact requerido",
			data: `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":[],"tests":["go test ./..."]}`,
		},
		{
			name: "artifact fuera de write-set",
			data: `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md","docs/no-autorizado.md"],"tests":["go test ./..."]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(tc.data), spec)
			requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
		})
	}
}

func TestCodexAgentAckReceiptV0AceptaWriteSetConGlobCerrado(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{
		"go.mod",
		"cmd/server/main.go",
		"internal/bootstrap/*.go",
	}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["go.mod","cmd/server/main.go","internal/bootstrap/config.go"],"tests":["go test ./..."],"notes":["done"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0AceptaWriteSetConGlobstarRecursivo(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{
		"go.mod",
		"cmd/server/**",
		"internal/domain/**",
		"**/*_test.go",
	}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["go.mod","cmd/server/main.go","internal/domain/book.go","internal/domain/book_test.go"],"tests":["go test ./..."],"notes":["done"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0RechazaFilesConGlobAunqueWriteSetUseGlob(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{
		"go.mod",
		"cmd/server/**",
		"internal/**",
	}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["go.mod","cmd/server/**","internal/**"],"tests":["go test ./..."],"notes":["done"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
}

func TestCodexAgentAckReceiptV0AceptaWriteSetRaiz(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{"."}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["docs/extra.md"],"tests":["go test ./..."],"notes":["alcance raiz autorizado"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

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

func TestCodexAgentAckReceiptV0RechazaSecretoHOMEYTranscript(t *testing.T) {
	spec := codexSpecForTestV0()
	forbidden := []string{
		`"notes":["access_token=abc123"]`,
		`"notes":["HOME=/home/alberto/.codex"]`,
		`"notes":["full transcript: prompt=todo completion=ok"]`,
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

func codexValidAckJSONV0() string {
	return `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["done"]}`
}
