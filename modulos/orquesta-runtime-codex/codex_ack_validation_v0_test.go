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

func TestCodexAgentAckReceiptV0NormalizaRefsRedundantesSiLaIdentidadCuadra(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-erronea","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-00l","status":"completed","files":["README.md"],"tests":["go test ./..."]}`

	got, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	if got.TaskRef != spec.AgentPacket.Task.TaskRef {
		t.Fatalf("task_ref=%q want %q", got.TaskRef, spec.AgentPacket.Task.TaskRef)
	}
	if got.CorrelationID != spec.CorrelationID {
		t.Fatalf("correlation_id=%q want %q", got.CorrelationID, spec.CorrelationID)
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
			name: "target module contradictorio",
			data: `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"otro-modulo","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."]}`,
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

func TestCodexAgentAckReceiptV0AceptaRetrySoloConTestsSinArtifacts(t *testing.T) {
	spec := codexSpecForTestV0()
	data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":[],"tests":["go test ./..."],"notes":["retry de tests sin ediciones de producto"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(data), spec)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v, want none", issues)
	}
}

func TestCodexAgentAckReceiptV0RechazaArtifactsFaltantesSinEvidencia(t *testing.T) {
	spec := codexSpecForTestV0()
	data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":[]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(data), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
}

func TestCodexAgentAckReceiptV0AceptaArtifactFueraDeWriteSetComoRailBlando(t *testing.T) {
	spec := codexSpecForTestV0()
	data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md","docs/no-autorizado.md"],"tests":["go test ./..."]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(data), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0AceptaExpansionWriteSetJustificada(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md","internal/adapters/memory/store.go"],"tests":["go test ./..."],"notes":["Se amplio alcance con internal/adapters/memory porque la app necesita un conector ejecutable."]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0AceptaEntregaParcialDentroDelWriteSet(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{
		"go.mod",
		"cmd/server/main.go",
		"internal/domain",
		"internal/application",
		"internal/ports",
		"internal/http",
		"web",
		"README.md",
		"docs",
		"tests",
	}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["web/index.html","web/app.js","web/styles.css"],"tests":["go test ./...","node --check web/app.js"],"notes":["Correccion parcial dentro del write-set; la revision/director conserva la comprobacion de completitud."]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0RechazaArchivosDeControlAunqueExpansionEsteJustificada(t *testing.T) {
	spec := codexSpecForTestV0()
	cases := []string{
		".orquesta-runtime/run/agent_ack.json",
		"director_decisions.json",
		"agent_packet.json",
		"codex_stderr.log",
	}
	for _, file := range cases {
		t.Run(file, func(t *testing.T) {
			ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md","` + file + `"],"tests":["go test ./..."],"notes":["Se amplio alcance porque era imprescindible."]}`

			_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

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

func codexValidAckJSONV0() string {
	return `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["done"]}`
}
