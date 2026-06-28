package orquestaruntimecodex

import (
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
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

func TestStrictCompletedCodexAgentAckV0ToleraRefsConEnvoltorioCosmeticoV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":" ` + "`" + `req-codex-001` + "`" + ` ","correlation_id":"\"corr-codex-001\"","ack_ref":"'ack-ref-001'","target_module":" orquesta-generated-app ","task_ref":" ` + "`" + `task-ref-001` + "`" + ` ","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]}`

	validated, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("refs cosmeticas no deben bloquear strict: %+v", issues)
	}
	if validated.RequestID != spec.RequestID ||
		validated.CorrelationID != spec.CorrelationID ||
		validated.AckRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		validated.TaskRef != spec.AgentPacket.Task.TaskRef {
		t.Fatalf("ACK strict no normalizado contra spec: %+v", validated)
	}
}

func TestStrictCompletedCodexAgentAckV0ClasificaColisionPadreSubrolV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001-subrol-redaccion","status":"completed","files":["README.md"],"tests":["go test ./..."]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
	requireCodexIssueEvidenceV0(t, issues, CodexAgentAckInvalidParentSubroleCollisionEvidenceV0)
}

func TestStrictCompletedCodexAgentAckV0ClasificaColisionPadreSubrolConEnvoltorioCosmeticoV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":" ` + "`" + `req-codex-001` + "`" + ` ","correlation_id":"corr-codex-001","ack_ref":"'ack-ref-001'","target_module":" orquesta-generated-app ","task_ref":" ` + "`" + `task-ref-001-subrol-redaccion` + "`" + ` ","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
	requireCodexIssueEvidenceV0(t, issues, CodexAgentAckInvalidParentSubroleCollisionEvidenceV0)
}

func TestStrictCompletedCodexAgentAckV0ClasificaColisionPadreChildTaskRefV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.ChildTaskRefs = []string{"task-ref-child-redaccion-001"}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-child-redaccion-001","status":"completed","files":["README.md"],"tests":["go test ./..."]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
	requireCodexIssueEvidenceV0(t, issues, CodexAgentAckInvalidParentChildTaskCollisionEvidenceV0)
}

func TestStrictCompletedCodexAgentAckV0ClasificaColisionPadreChildTaskRefConEnvoltorioCosmeticoV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.ChildTaskRefs = []string{"task-ref-child-redaccion-001"}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"` + "`" + `ack-ref-001` + "`" + `","target_module":"orquesta-generated-app","task_ref":" ` + "`" + `task-ref-child-redaccion-001` + "`" + ` ","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
	requireCodexIssueEvidenceV0(t, issues, CodexAgentAckInvalidParentChildTaskCollisionEvidenceV0)
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

func TestStrictCompletedCodexAgentAckV0AceptaContextoRefOnlyResueltoSinReciboShellV0(t *testing.T) {
	spec := codexSpecForTestV0()
	refOnlyCommand := "validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita"
	spec.AgentPacket.Task.RequiredTests = []string{"go test ./...", refOnlyCommand}
	spec.AgentPacket.Context = orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     "bundle-ref-strict-ref-only",
		WorkOrderRef:  spec.AgentPacket.WorkOrderRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		Entries: []orquestacontext.ContextMaterializedEntryV0{{
			EntryRef:          "entry-ref-strict-ref-only",
			Layer:             orquestacontext.ContextLayerTaskContextV0,
			Kind:              orquestacontext.ContextEntryDocRefV0,
			SourceRef:         "source-ref-strict-ref-only",
			Mode:              orquestacontext.ContextMaterializationModeRefOnlyV0,
			Required:          true,
			RefOnlyReason:     orquestacontext.ContextRefOnlyReasonMaterializationMissingV0,
			RequiredRefAction: orquestacontext.ContextRequiredRefActionAckEvidenceV0,
		}},
	}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./...","` + refOnlyCommand + `"],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}],"notes":["contexto_ref_only_resuelto: contexto requerido validado por evidencia explicita"]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("contexto ref_only resuelto no debe exigir recibo shell: %+v", issues)
	}
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

func TestStrictCompletedCodexAgentAckV0AceptaSchemaCompatibleYComandoCanonicoV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.RequiredTests = []string{"go test -count=1 ./..."}
	ack := `{"schema_version":"codex_agent_ack.v0.1","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go   test   ./...   -count=1"],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0.1","command":"go test ./... -count=1","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("schema compatible y comando canonico no deben bloquear: %+v", issues)
	}
}

func TestStrictCompletedCodexAgentAckV0AceptaIdentidadParcialDerivableConEvidenciaV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]}`

	validated, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("identidad derivable con evidencia valida no debe bloquear: %+v", issues)
	}
	if validated.RequestID != spec.RequestID ||
		validated.CorrelationID != spec.CorrelationID ||
		validated.AckRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		validated.TaskRef != spec.AgentPacket.Task.TaskRef {
		t.Fatalf("ACK no hidratado desde spec: %+v", validated)
	}
}

func TestStrictCompletedCodexAgentAckV0RechazaSchemaMayorNoSoportadoV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v1","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckInvalidV0)
}

func TestStrictCompletedCodexAgentAckV0RechazaSchemaCompatibleMalFormadoV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0.","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckInvalidV0)
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
	enableCodexRailsModeEnforcedForTestV0(t)
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}],"notes":["rail pendiente: token provider home prompt sin valor operativo"]}`

	validated, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("rail pendiente generico no debe bloquear ACK estricto: %+v", issues)
	}
	if CodexAgentAckHasPendingRailV0(validated) {
		t.Fatalf("rail pendiente generico no debe conservarse como evidencia")
	}
}

func TestCodexDeliveryObservationV0RechazaACKStrictSinRecibosV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Policies = append(spec.AgentPacket.Policies, "ack_terminal_strict")
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

func TestStrictCompletedCodexAgentAckV0AceptaTestsExtraConRecibosValidosV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./...","node --check web/app.js"],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true},{"schema_version":"codex_required_test_receipt.v0","command":"node --check web/app.js","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-002"],"occurred_at":"2026-05-24T10:01:00Z","sequence":2,"output_redacted":true}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	if len(issues) != 0 {
		t.Fatalf("tests extra validos deben llegar a review como evidencia: %+v", issues)
	}
}

func TestStrictCompletedCodexAgentAckV0RechazaReceiptFaltanteParaCadaRequiredTestV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.RequiredTests = []string{"go test ./...", "git diff --check"}
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./...","git diff --check"],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"go test ./...","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-001"],"occurred_at":"2026-05-24T10:00:00Z","sequence":1,"output_redacted":true}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueEvidenceV0(t, issues, "missing_required_test_receipt")
}

func TestStrictCompletedCodexAgentAckV0RechazaSiFaltaTestObligatorioAunqueHayaExtrasV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["node --check web/app.js"],"test_receipts":[{"schema_version":"codex_required_test_receipt.v0","command":"node --check web/app.js","status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-002"],"occurred_at":"2026-05-24T10:01:00Z","sequence":2,"output_redacted":true}]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueEvidenceV0(t, issues, "required_tests_mismatch")
}

func TestCodexAgentPacketRequiresStrictTerminalAckV0SoloPorPolicyExplicitaV0(t *testing.T) {
	spec := codexSpecForTestV0()

	if CodexAgentPacketRequiresStrictTerminalAckV0(spec.AgentPacket) {
		t.Fatalf("write_set_closed no debe activar strict: %+v", spec.AgentPacket.Policies)
	}

	spec.AgentPacket.Policies = []string{"ack_terminal_strict"}
	if !CodexAgentPacketRequiresStrictTerminalAckV0(spec.AgentPacket) {
		t.Fatalf("policy strict explicita no detectada: %+v", spec.AgentPacket.Policies)
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
