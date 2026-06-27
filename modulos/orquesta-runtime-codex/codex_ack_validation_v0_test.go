package orquestaruntimecodex

import (
	"os"
	"path/filepath"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestCodexAgentAckReceiptV0AceptaACKValido(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := codexValidAckJSONV0()

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0AceptaACKProgramacionResiGRXConContextoRefOnly(t *testing.T) {
	spec := codexSpecForTestV0()
	requestID := "agent-ref-task-resigrx-rx000-bootstrap-vertical-mvp"
	correlationID := "corr-run-spec-resigrx-req-resigrx-887f4566d73cdaa5c23e03d06a14da8a-burst-002"
	targetModule := "orquesta-app-stack-programacion"
	taskRef := "task-resigrx-rx000-bootstrap-vertical-mvp"
	ackRef := "ack-ref-app-stack-agent-ref-task-resigrx-rx000-bootstrap-vertical-mvp"
	requiredTests := []string{
		"go test ./...",
		"git diff --check",
		"test de arquitectura contra imports prohibidos en dominio",
		"test de i18n para errores publicos",
		"OpenAPI parseable",
		"validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita",
	}
	spec.RequestID = requestID
	spec.CorrelationID = correlationID
	spec.AgentPacket.RequestID = requestID
	spec.AgentPacket.CorrelationID = correlationID
	spec.AgentPacket.WorkOrderRef = taskRef
	spec.AgentPacket.TargetModule = targetModule
	spec.AgentPacket.Task.TaskRef = taskRef
	spec.AgentPacket.Task.WriteSet = []string{
		"go.mod",
		"cmd/resigrx-api/main.go",
		"cmd/resigrx-worker/main.go",
		"internal/platform/**",
		"internal/modules/identity/**",
		"internal/modules/residents/**",
		"internal/modules/care/**",
		"internal/modules/medication/**",
		"internal/modules/audit/**",
		"i18n/**",
		"docs/openapi.yaml",
		"README.md",
		".env.example",
	}
	spec.AgentPacket.Task.RequiredTests = requiredTests
	spec.AgentPacket.Context = orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     "bundle-ref-app-stack-task-resigrx-rx000-bootstrap-vertical-mvp",
		WorkOrderRef:  taskRef,
		TargetModule:  targetModule,
		Entries: []orquestacontext.ContextMaterializedEntryV0{{
			EntryRef:          "entry-ref-app-stack-task-resigrx-rx000-bootstrap-vertical-mvp",
			Layer:             orquestacontext.ContextLayerTaskContextV0,
			Kind:              orquestacontext.ContextEntryDocRefV0,
			SourceRef:         "source-ref-app-stack-task-resigrx-rx000-bootstrap-vertical-mvp",
			Mode:              orquestacontext.ContextMaterializationModeRefOnlyV0,
			Required:          true,
			RefOnlyReason:     orquestacontext.ContextRefOnlyReasonMaterializationMissingV0,
			RequiredRefAction: orquestacontext.ContextRequiredRefActionAckEvidenceV0,
		}},
	}
	spec.AgentPacket.DeliveryRefs.AckRef = ackRef
	spec.AgentPacket.Policies = []string{
		"write_set_closed",
		"ack_required",
		"context_small_by_refs",
		"required_ref_only_context_guard",
	}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"` + requestID + `","correlation_id":"` + correlationID + `","ack_ref":"` + ackRef + `","target_module":"` + targetModule + `","task_ref":"` + taskRef + `","status":"completed","files":["go.mod","cmd/resigrx-api/main.go","cmd/resigrx-worker/main.go","internal/platform/api/server.go","internal/modules/residents/application/service.go","i18n/es/errors.json","docs/openapi.yaml","README.md",".env.example"],"tests":["go test ./...","git diff --check","test de arquitectura contra imports prohibidos en dominio","test de i18n para errores publicos","OpenAPI parseable","validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita"],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-go-test-all-20260527T1724Z"],"occurred_at":"2026-05-27T17:24:40Z","sequence":1,"output_redacted":true},{"schema_version":"codex_required_test_receipt.v0","command":"git diff --check","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-git-diff-check-20260527T1724Z"],"occurred_at":"2026-05-27T17:24:40Z","sequence":2,"output_redacted":true},{"schema_version":"codex_required_test_receipt.v0","command":"test de arquitectura contra imports prohibidos en dominio","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-arch-domain-imports-20260527T1724Z"],"occurred_at":"2026-05-27T17:24:40Z","sequence":3,"output_redacted":true},{"schema_version":"codex_required_test_receipt.v0","command":"test de i18n para errores publicos","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-i18n-errors-20260527T1724Z"],"occurred_at":"2026-05-27T17:24:40Z","sequence":4,"output_redacted":true},{"schema_version":"codex_required_test_receipt.v0","command":"OpenAPI parseable","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-openapi-parseable-20260527T1724Z"],"occurred_at":"2026-05-27T17:24:40Z","sequence":5,"output_redacted":true},{"schema_version":"codex_required_test_receipt.v0","command":"validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-ref-only-context-20260527T1724Z"],"occurred_at":"2026-05-27T17:24:40Z","sequence":6,"output_redacted":true}],"notes":["contexto_ref_only_resuelto: agent_packet.json leido; entrada required=true mode=ref_only required_ref_action=ack_evidence_required resuelta por evidencia explicita en ACK","documentos obligatorios leidos antes de programar; sin datos reales de residentes en tests"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0BloqueaSecretoEfectivoConRailOffPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "")
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["diagnostico conservado access_token=abc123 para auditoria local"]}`
	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)
	if !codexAckIssuesContainEvidenceV0(issues, "forbidden_sensitive_detail") {
		t.Fatalf("secreto efectivo debe bloquear con rail off por defecto: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0RailEstrictoAceptaEvidenciaRefOnlyOPES(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "on")
	spec := codexSpecForTestV0()
	spec.AgentPacket.Context = orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		Entries: []orquestacontext.ContextMaterializedEntryV0{{
			Required:          true,
			Mode:              orquestacontext.ContextMaterializationModeRefOnlyV0,
			RequiredRefAction: orquestacontext.ContextRequiredRefActionAckEvidenceV0,
		}},
	}
	spec.AgentPacket.Task.RequiredTests = append(
		spec.AgentPacket.Task.RequiredTests,
		"validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita",
	)
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./...","validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita"],"notes":["contexto_ref_only_resuelto: entrada required ref_only con required_ref_action ack_evidence_required resuelta mediante lectura de agent_packet.json y evidencia explicita en test_receipts"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("evidencia ref_only OPES no debe bloquear con rail estricto: %+v", issues)
	}
}

func TestCodexAgentAckReceiptV0RailEstrictoRefOnlyCortaSecretoEfectivo(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "on")
	spec := codexSpecForTestV0()
	spec.AgentPacket.Context = orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		Entries: []orquestacontext.ContextMaterializedEntryV0{{
			Required:          true,
			Mode:              orquestacontext.ContextMaterializationModeRefOnlyV0,
			RequiredRefAction: orquestacontext.ContextRequiredRefActionAckEvidenceV0,
		}},
	}
	spec.AgentPacket.Task.RequiredTests = append(
		spec.AgentPacket.Task.RequiredTests,
		"validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita",
	)
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./...","validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita"],"notes":["contexto_ref_only_resuelto: required_ref_action ack_evidence_required con access_token=abc123"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if !codexAckIssuesContainEvidenceV0(issues, "forbidden_sensitive_detail") {
		t.Fatalf("secreto efectivo debe bloquear aunque rails blandos esten quitados: %+v", issues)
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

func TestCodexAgentAckReceiptV0DetectaColisionACKPadreConTaskRefSubrolV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001-subrole-s3","status":"completed","files":["README.md"],"tests":["go test ./..."]}`

	got, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
	requireCodexIssueEvidenceV0(t, issues, CodexAgentAckInvalidParentSubroleCollisionEvidenceV0)
	if got.TaskRef != "task-ref-001-subrole-s3" {
		t.Fatalf("task_ref de subrol no debe normalizarse como typo recuperable: %+v", got)
	}
}

func TestCodexAgentAckReceiptV0DetectaColisionACKPadreConChildTaskRefV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.ChildTaskRefs = []string{"task-ref-child-redaccion-001"}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-child-redaccion-001","status":"completed","files":["README.md"],"tests":["go test ./..."]}`

	got, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
	requireCodexIssueEvidenceV0(t, issues, CodexAgentAckInvalidParentChildTaskCollisionEvidenceV0)
	if got.TaskRef != "task-ref-child-redaccion-001" {
		t.Fatalf("child task_ref no debe normalizarse como typo recuperable: %+v", got)
	}
}

func TestCodexAgentAckReceiptV0AceptaACKMinimoHidratableConRequiredTests(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","status":"completed"}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("ACK minimo hidratable debe llegar a review, no bloquear: %+v", issues)
	}
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

func TestCodexAgentAckReceiptV0AceptaArtifactsFaltantesComoReview(t *testing.T) {
	spec := codexSpecForTestV0()
	data := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":[]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(data), spec)

	if len(issues) != 0 {
		t.Fatalf("artefactos faltantes deben llegar a review, no bloquear ACK: %+v", issues)
	}
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
		".orquesta-codex-runtime/run/agent_prompt.txt",
		"director_decisions.json",
		"agent_packet.json",
		"agent_shutdown_checkpoint_ack.json",
		"codex_stderr.log",
		"orquesta_shutdown_request.json",
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

func TestCodexAgentAckReceiptV0RechazaControlFilesConWriteSetRaiz(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{"."}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":[".orquesta-runtime/run/agent_ack.json"],"tests":["go test ./..."],"notes":["alcance raiz no exporta control"]}`

	_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
}

func codexValidAckJSONV0() string {
	return `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}],"notes":["done"]}`
}
