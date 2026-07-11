package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestStackGoalMaterializedRefsSourceV0DetectaWorkDeliveryEnWriteSet(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_006")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "work_delivery.json"), []byte(`{"status":"partial"}`), 0o600); err != nil {
		t.Fatalf("write delivery: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-001", "temas/tema_006")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		len(result.DomainReceiptRefs) != 1 ||
		!strings.Contains(result.DomainReceiptRefs[0], "work-delivery") ||
		strings.Contains(result.DomainReceiptRefs[0], projectDir) ||
		strings.Contains(result.DomainReceiptRefs[0], string(filepath.Separator)+filepath.Base(projectDir)) ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-work-delivery-detected") {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaCheckpointEnWriteSet(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_001", "coordinacion_wave10_phase1")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "checkpoint_started.txt"), []byte("started\n"), 0o600); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-checkpoint-001", "temas/tema_001/coordinacion_wave10_phase1")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		len(result.ArtifactRefs) != 1 ||
		!strings.Contains(result.ArtifactRefs[0], "artifact-ref-checkpoint") ||
		strings.Contains(result.ArtifactRefs[0], projectDir) ||
		containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized:") ||
		containsStringV0(result.IssueCodes, "partial_artifacts_written") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-checkpoint-detected") ||
		!containsStringV0(result.IssueCodes, "goal_first_materialized_checkpoint_detected") {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaArtefactosTxtBUG088V0(t *testing.T) {
	projectDir := t.TempDir()
	generatedDir := filepath.Join(projectDir, "generated-apps")
	if err := os.MkdirAll(generatedDir, 0o700); err != nil {
		t.Fatalf("mkdir generated: %v", err)
	}
	if err := os.WriteFile(filepath.Join(generatedDir, "checkpoint_started_bug088.txt"), []byte("started\n"), 0o600); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(generatedDir, "bug088_second_artifact.txt"), []byte("second artifact\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-bug088-001", "generated-apps")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-checkpoint:") ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized:") ||
		!containsStringV0(result.IssueCodes, "partial_artifacts_written") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-partial-artifacts-written") {
		t.Fatalf("artefactos BUG-088 no proyectados: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaArtefactosParcialesSinReceiptTerminal(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_036")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "borrador_reutilizable.md"), []byte("# Borrador\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-partial-artifacts-001", "temas/tema_036")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "partial_artifacts_written") ||
		containsStringV0(result.IssueCodes, "missing_terminal_receipt_after_artifacts_pass") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-partial-artifacts-written") ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized:") {
		t.Fatalf("artefactos parciales no proyectados: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaPhase0CheckpointDeliveryNoPublicable(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_039")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(topicDir, "orquesta_phase0_checkpoint_delivery.json"),
		[]byte(`{"status":"phase0_complete","publishable":false}`),
		0o600,
	); err != nil {
		t.Fatalf("write phase0 delivery: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-phase0-001", "temas/tema_039")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "phase0_complete_non_publishable") ||
		!containsStringV0(result.IssueCodes, "partial_artifacts_written") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-phase0-complete-non-publishable") ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized:") {
		t.Fatalf("phase0 delivery no publicable no detectada: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0PublicaChecklistEsperadoDesdeSpec(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_038")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "borrador_reutilizable.md"), []byte("# Borrador\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-checklist-001", "temas/tema_038")
	state.Spec.ArtifactContracts = []orquestagoal.GoalArtifactContractV0{{
		ArtifactRef:  "artifact-ref-tema-038-ampliado",
		ArtifactType: "topic_text",
		Required:     true,
	}}
	state.Spec.RequiredTests = []orquestagoal.GoalRequiredTestV0{{
		TestRef: "test-ref-tema-038-qa",
	}}
	state.Spec.ClosurePolicy.RequiredEvidenceRefs = []string{"evidence-ref-tema-038-qa"}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.ExpectedReceiptRefs, "expected-artifact-ref:artifact-ref-tema-038-ampliado") ||
		!containsStringV0(result.ExpectedReceiptRefs, "expected-required-test-ref:test-ref-tema-038-qa") ||
		!containsStringV0(result.ExpectedReceiptRefs, "expected-evidence-ref:evidence-ref-tema-038-qa") {
		t.Fatalf("checklist esperado no proyectado: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0NoSaleDelProjectWorkDir(t *testing.T) {
	projectDir := t.TempDir()
	state := orquestagoal.GoalWorkStateV0{
		RunRef: "run-goal-materialized-traversal-001",
		Spec: orquestagoal.GoalWorkSpecV0{
			WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "../fuera"}},
		},
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if ok || len(result.DomainReceiptRefs) != 0 {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaReceiptTerminalAusenteTrasQAPass(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_029")
	validationDir := filepath.Join(topicDir, "09_validacion")
	if err := os.MkdirAll(validationDir, 0o700); err != nil {
		t.Fatalf("mkdir validation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(validationDir, "informe_qa.json"),
		[]byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":true,"strict_editorial_qa_pass":true,
				"question_bank_publicable":true,
				"tutor_assets_publicable":true}}`),
		0o600,
	); err != nil {
		t.Fatalf("write qa: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-missing-receipt-001", "temas/tema_029")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "missing_terminal_receipt_after_artifacts_pass") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-missing-terminal-receipt-after-artifacts-pass") ||
		len(result.ExpectedReceiptRefs) != 3 ||
		!containsStringV0(result.ExpectedReceiptRefs, "expected-terminal-receipt:orquesta-goal-result-goal-ref-run-goal-materialized-missing-receipt-001-json") {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
	if !containsStringPrefixForTestV0(result.EvidenceRefs, "evidence-ref-goal-materialized-qa-pass:") {
		t.Fatalf("evidence=%+v", result.EvidenceRefs)
	}
}

func TestGoalMaterializedFileLooksLikeArtifactV0NoCuentaResultadoGoalSidecarV0(t *testing.T) {
	projectDir := t.TempDir()
	base := "orquesta_goal_result_goal-ref-autoprogramming-backlog-srv-task-022-a54b0a70.json"
	if goalMaterializedFileLooksLikeArtifactV0(projectDir, filepath.Join(projectDir, "docs", base), base) {
		t.Fatalf("goal result sidecar contado como artefacto")
	}
	if !goalMaterializedFileIsGoalResultV0(base) {
		t.Fatalf("goal result sidecar no reconocido")
	}
}

func TestStackGoalMaterializedRefsSourceV0ReparaReceiptTerminalConPuertosV0(t *testing.T) {
	ctx := context.Background()
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_029")
	validationDir := filepath.Join(topicDir, "09_validacion")
	if err := os.MkdirAll(validationDir, 0o700); err != nil {
		t.Fatalf("mkdir validation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(validationDir, "informe_qa.json"),
		[]byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":true,"strict_editorial_qa_pass":true,
				"question_bank_publicable":true,
				"tutor_assets_publicable":true}}`),
		0o600,
	); err != nil {
		t.Fatalf("write qa: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-repair-receipt-001", "temas/tema_029")
	store := newGoalFirstQueueStateStoreForTestV0()
	if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config:                       ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
		GoalStateStore:               store,
		GoalClosureValidator:         orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		RepairMissingTerminalReceipt: true,
	}).ResolveDirectorGoalMaterializedRefsV0(ctx, state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		containsStringV0(result.IssueCodes, "missing_terminal_receipt_after_artifacts_pass") ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized:") {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
	persisted, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusCompleteV0 ||
		persisted.LastResult == nil ||
		persisted.LastClosure == nil ||
		!persisted.LastClosure.Accepted ||
		!containsStringV0(persisted.LastResult.EvidenceRefs, goalFirstRepairReceiptAttemptedEvidenceRefV0) {
		t.Fatalf("persisted=%+v", persisted)
	}
}

func TestStackGoalMaterializedRefsSourceV0ReintentaReceiptTerminalCorregidoTrasReworkV0(t *testing.T) {
	ctx := context.Background()
	projectDir := t.TempDir()
	appDir := filepath.Join(projectDir, "generated-apps", "smoke-goal-first")
	docsDir := filepath.Join(appDir, "docs")
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		t.Fatalf("mkdir app docs: %v", err)
	}
	for _, item := range map[string]string{
		"README.md":         "# App\n",
		"handoff_report.md": "# Handoff\n",
		"source_tree.md":    "# Tree\n",
	} {
		if err := os.WriteFile(filepath.Join(appDir, item), []byte(item), 0o600); err != nil {
			t.Fatalf("write %s: %v", item, err)
		}
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-retry-terminal-001", "generated-apps/smoke-goal-first")
	sourceArtifactRef := "artifact-ref-" + state.RunRef + "-source"
	handoffArtifactRef := "artifact-ref-" + state.RunRef + "-handoff"
	testRef := "test-ref-" + state.RunRef + "-generated-app"
	evidenceRef := "evidence-ref-app-director-goal-first-v0"
	state.Spec.ArtifactContracts = []orquestagoal.GoalArtifactContractV0{
		{ArtifactRef: sourceArtifactRef, ArtifactType: "source_tree", Required: true},
		{ArtifactRef: handoffArtifactRef, ArtifactType: "handoff_report", Required: true},
	}
	state.Spec.RequiredTests = []orquestagoal.GoalRequiredTestV0{{TestRef: testRef}}
	state.Spec.ClosurePolicy = orquestagoal.GoalClosurePolicyV0{
		RequireArtifacts:     true,
		RequireArtifactPaths: true,
		RequireRequiredTests: true,
		RequiredEvidenceRefs: []string{evidenceRef},
	}
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       state.GoalRef,
		EvidenceRefs:  []string{goalFirstRepairReceiptAttemptedEvidenceRefV0},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  orquestagoal.ErrGoalClosureInvalidV0,
			Field: "artifact_refs",
		}},
	}
	state, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		t.Fatalf("NewGoalWorkStateV0 retry state: %v", err)
	}
	receipt := `{
  "schema_version":"orquesta_goal_result.v0",
  "goal_ref":"` + state.GoalRef + `",
  "status":"complete",
  "summary":"app generada y verificada",
  "artifact_refs":["` + sourceArtifactRef + `","` + handoffArtifactRef + `"],
  "artifact_paths":["generated-apps/smoke-goal-first/README.md","generated-apps/smoke-goal-first/handoff_report.md"],
  "required_test_results":[{"test_ref":"` + testRef + `","status":"passed","evidence_refs":["` + evidenceRef + `"]}],
  "evidence_refs":["` + evidenceRef + `"]
}`
	if err := os.WriteFile(filepath.Join(docsDir, "orquesta_goal_result_v0.json"), []byte(receipt), 0o600); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	store := newGoalFirstQueueStateStoreForTestV0()
	if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config:                       ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
		GoalStateStore:               store,
		GoalClosureValidator:         orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		RepairMissingTerminalReceipt: true,
	}).ResolveDirectorGoalMaterializedRefsV0(ctx, state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok || containsStringV0(result.IssueCodes, goalFirstRepairReceiptRequiresReworkIssueV0) {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
	persisted, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.LastClosure == nil ||
		!persisted.LastClosure.Accepted ||
		persisted.LastResult == nil ||
		!containsStringV0(persisted.LastResult.ArtifactRefs, sourceArtifactRef) ||
		!containsStringV0(persisted.LastResult.ArtifactRefs, handoffArtifactRef) {
		t.Fatalf("persisted=%+v", persisted)
	}
}

func TestStackGoalMaterializedRefsSourceV0NoAceptaQAGenericaEnContextoOPES(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_032")
	validationDir := filepath.Join(topicDir, "09_validacion")
	if err := os.MkdirAll(validationDir, 0o700); err != nil {
		t.Fatalf("mkdir validation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(validationDir, "informe_qa.json"),
		[]byte(`{"passed":true,"ok":true,"status":"ok"}`),
		0o600,
	); err != nil {
		t.Fatalf("write qa: %v", err)
	}
	state := goalMaterializedRefsOPESStateForTestV0(t, "run-goal-materialized-opes-generic-qa-001", "temas/tema_032")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if containsStringV0(result.IssueCodes, "missing_terminal_receipt_after_artifacts_pass") ||
		containsStringPrefixForTestV0(result.EvidenceRefs, "evidence-ref-goal-materialized-qa-pass:") {
		t.Fatalf("QA generica no debe bastar en OPES: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0AceptaQAGenericaSinContextoOPES(t *testing.T) {
	projectDir := t.TempDir()
	appDir := filepath.Join(projectDir, "apps", "app_001")
	validationDir := filepath.Join(appDir, "validation")
	if err := os.MkdirAll(validationDir, 0o700); err != nil {
		t.Fatalf("mkdir validation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "resultado.md"), []byte("# Resultado\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(validationDir, "qa_report.json"),
		[]byte(`{"passed":true}`),
		0o600,
	); err != nil {
		t.Fatalf("write qa: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-generic-qa-001", "apps/app_001")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "missing_terminal_receipt_after_artifacts_pass") ||
		!containsStringPrefixForTestV0(result.EvidenceRefs, "evidence-ref-goal-materialized-qa-pass:") {
		t.Fatalf("QA generica debe seguir valiendo sin OPES: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0AceptaTernaQAEnContextoOPES(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_033")
	validationDir := filepath.Join(topicDir, "09_validacion")
	if err := os.MkdirAll(validationDir, 0o700); err != nil {
		t.Fatalf("mkdir validation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(validationDir, "informe_qa.json"),
		[]byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":true,"strict_editorial_qa_pass":true,
				"question_bank_publicable":true,
				"tutor_assets_publicable":true}}`),
		0o600,
	); err != nil {
		t.Fatalf("write qa: %v", err)
	}
	state := goalMaterializedRefsOPESStateForTestV0(t, "run-goal-materialized-opes-terna-qa-001", "temas/tema_033")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "missing_terminal_receipt_after_artifacts_pass") ||
		!containsStringPrefixForTestV0(result.EvidenceRefs, "evidence-ref-goal-materialized-qa-pass:") {
		t.Fatalf("QA OPES canonica debe activar receipt ausente: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaQAFailedPublicTextEnContextoOPES(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_034")
	validationDir := filepath.Join(topicDir, "09_validacion")
	if err := os.MkdirAll(validationDir, 0o700); err != nil {
		t.Fatalf("mkdir validation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(validationDir, "informe_qa.json"),
		[]byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":false,"strict_editorial_qa_pass":false},"valid_artifact_paths":["temas/tema_034/fuentes.md"],"invalid_artifact_paths":["temas/tema_034/tema_ampliado.md"],"findings":["texto publico no apto"]}`),
		0o600,
	); err != nil {
		t.Fatalf("write qa: %v", err)
	}
	state := goalMaterializedRefsOPESStateForTestV0(t, "run-goal-materialized-opes-qa-fail-001", "temas/tema_034")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "qa_failed_public_text") ||
		containsStringV0(result.IssueCodes, "missing_terminal_receipt_after_artifacts_pass") ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized:") ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized-valid:") ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized-invalid:") ||
		!containsStringPrefixForTestV0(result.EvidenceRefs, "evidence-ref-goal-materialized-qa-failed-public-text:") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-valid-artifact-list") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-invalid-artifact-list") {
		t.Fatalf("QA fail OPES debe conservar artefacto y pedir rework: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaArtifactPathsOmitidosEnReceiptTerminal(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_035")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "plan_rework_por_fases.md"), []byte("# Plan\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "matriz_fuentes_reutilizacion.md"), []byte("# Matriz\n"), 0o600); err != nil {
		t.Fatalf("write omitted artifact: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "orquesta_goal_result_v0.json"), []byte(`{"status":"complete"}`), 0o600); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-omitted-paths-001", "temas/tema_035")
	state.Spec.ClosurePolicy.RequireArtifactPaths = true
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       state.GoalRef,
		ArtifactPaths: []string{"temas/tema_035/plan_rework_por_fases.md"},
		ArtifactRefs:  []string{"artifact-ref-plan-rework"},
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "artifact_paths_omitted_materialized") ||
		!containsStringPrefixForTestV0(result.EvidenceRefs, "evidence-ref-goal-materialized-artifact-paths-omitted:") ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized:") {
		t.Fatalf("artifact_paths omitidos no detectados: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaArtifactPathsOmitidosDesdeReceiptDurable(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_036")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "plan_rework_por_fases.md"), []byte("# Plan\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "matriz_fuentes_reutilizacion.md"), []byte("# Matriz\n"), 0o600); err != nil {
		t.Fatalf("write omitted artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(topicDir, "orquesta_goal_result_v0.json"),
		[]byte(`{"status":"complete","artifact_paths":["temas/tema_036/plan_rework_por_fases.md"]}`),
		0o600,
	); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-durable-omitted-paths-001", "temas/tema_036")
	state.Spec.ClosurePolicy.RequireArtifactPaths = true
	state.LastResult = nil

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "artifact_paths_omitted_materialized") ||
		!containsStringPrefixForTestV0(result.EvidenceRefs, "evidence-ref-goal-materialized-artifact-paths-omitted:") {
		t.Fatalf("artifact_paths omitidos desde receipt durable no detectados: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0NormalizaMissingRefsRaizEnReceiptDurable(t *testing.T) {
	projectDir := t.TempDir()
	appDir := filepath.Join(projectDir, "generated-apps", "missing-refs")
	docsDir := filepath.Join(appDir, "docs")
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-missing-refs-001", "generated-apps/missing-refs")
	raw := `{
		"schema_version":"orquesta_goal_result.v0",
		"status":"invalid",
		"goal_ref":"` + state.GoalRef + `",
		"summary":"resultado durable inicial; pendiente de implementacion, artefactos y pruebas",
		"artifact_paths":["generated-apps/missing-refs/checkpoint_started.txt"],
		"missing_refs":["source_tree","handoff_report","technical_stack_manifest","go_app","tests"],
		"evidence_refs":["evidence-ref-app-director-goal-first-v0"]
	}`
	if err := os.WriteFile(filepath.Join(docsDir, "orquesta_goal_result_v0.json"), []byte(raw), 0o600); err != nil {
		t.Fatalf("write receipt: %v", err)
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).LoadTerminalGoalMaterializedResultV0(context.Background(), state)
	if err != nil {
		t.Fatalf("LoadTerminalGoalMaterializedResultV0: %v", err)
	}
	if !ok ||
		result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!containsStringV0(result.Checklist.MissingRefs, "source_tree") ||
		!containsStringV0(result.Checklist.MissingRefs, "tests") {
		t.Fatalf("missing_refs raiz no normalizado: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0EncuentraReceiptTerminalTrasCacheVoluminosaV0(t *testing.T) {
	projectDir := t.TempDir()
	appDir := filepath.Join(projectDir, "generated-apps", "cache-heavy")
	cacheDir := filepath.Join(appDir, ".gocache", "00")
	localCacheDir := filepath.Join(appDir, ".gocache-local", "00")
	docsDir := filepath.Join(appDir, "docs")
	for _, dir := range []string{cacheDir, localCacheDir, docsDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for i := 0; i < goalMaterializedQAScanMaxFilesV0+40; i++ {
		for _, dir := range []string{cacheDir, localCacheDir} {
			if err := os.WriteFile(
				filepath.Join(dir, "cache-"+strconv.Itoa(i)+".txt"),
				[]byte("cache\n"),
				0o600,
			); err != nil {
				t.Fatalf("write cache %d: %v", i, err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(appDir, "source_tree.md"), []byte("# Source\n"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(
		t,
		"run-goal-materialized-cache-heavy-001",
		"generated-apps/cache-heavy",
	)
	artifactRef := "artifact-ref-" + state.RunRef + "-source"
	testRef := "test-ref-" + state.RunRef + "-generated-app"
	evidenceRef := "evidence-ref-app-director-goal-first-v0"
	receipt := `{
  "schema_version":"orquesta_goal_result.v0",
  "goal_ref":"` + state.GoalRef + `",
  "status":"complete",
  "summary":"receipt terminal tras cache",
  "artifact_refs":["` + artifactRef + `"],
  "artifact_paths":["generated-apps/cache-heavy/source_tree.md"],
  "required_test_results":[{"test_ref":"` + testRef + `","status":"passed","evidence_refs":["` + evidenceRef + `"]}],
  "evidence_refs":["` + evidenceRef + `"]
}`
	if err := os.WriteFile(filepath.Join(docsDir, goalMaterializedGoalResultFileV0), []byte(receipt), 0o600); err != nil {
		t.Fatalf("write receipt: %v", err)
	}

	terminal, terminalOK, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).LoadTerminalGoalMaterializedResultV0(context.Background(), state)
	if err != nil {
		t.Fatalf("LoadTerminalGoalMaterializedResultV0: %v", err)
	}
	if !terminalOK ||
		terminal.Status != orquestagoal.GoalStatusCompleteV0 ||
		!containsStringV0(terminal.ArtifactRefs, artifactRef) {
		t.Fatalf("terminalOK=%v terminal=%+v", terminalOK, terminal)
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		containsStringV0(result.IssueCodes, "partial_artifacts_written") ||
		!containsStringV0(result.ArtifactRefs, artifactRef) ||
		!containsStringV0(result.EvidenceRefs, evidenceRef) {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0ConservaReasonCodeCheckpointStartedV0(t *testing.T) {
	projectDir := t.TempDir()
	docsDir := filepath.Join(projectDir, "docs", "runbooks", "docs")
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(
		t,
		"run-goal-materialized-checkpoint-started-001",
		"docs/runbooks",
	)
	receipt := `{
  "schema_version":"orquesta_goal_result.v0",
  "goal_ref":"` + state.GoalRef + `",
  "status":"blocked",
  "reason_code":"checkpoint_started",
  "summary":"checkpoint temprano; faltan implementacion y pruebas",
  "missing_refs":["implementation","required_tests"],
  "evidence_refs":["evidence-ref-checkpoint-started"]
}`
	if err := os.WriteFile(filepath.Join(docsDir, goalMaterializedGoalResultFileV0), []byte(receipt), 0o600); err != nil {
		t.Fatalf("write receipt: %v", err)
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).LoadTerminalGoalMaterializedResultV0(context.Background(), state)
	if err != nil {
		t.Fatalf("LoadTerminalGoalMaterializedResultV0: %v", err)
	}
	if !ok || result.Status != orquestagoal.GoalStatusBlockedV0 ||
		!containsStringV0(result.Checklist.MissingRefs, "implementation") ||
		!goalFirstResultHasIssueCodeForTestV0(result, goalFirstResidentReworkReasonCheckpointStartedV0) {
		t.Fatalf("reason_code causal perdido: ok=%v result=%+v", ok, result)
	}
}

func goalFirstResultHasIssueCodeForTestV0(result orquestagoal.GoalWorkResultV0, code string) bool {
	for _, issue := range result.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestStackGoalMaterializedRefsSourceV0IgnoraReceiptTerminalDeOtroGoalV0(t *testing.T) {
	projectDir := t.TempDir()
	appDir := filepath.Join(projectDir, "generated-apps", "stale-receipt")
	docsDir := filepath.Join(appDir, "docs")
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "source_tree.md"), []byte("# Source\n"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(
		t,
		"run-goal-materialized-stale-receipt-001",
		"generated-apps/stale-receipt",
	)
	staleArtifactRef := "artifact-ref-run-goal-materialized-old-source"
	receipt := `{
  "schema_version":"orquesta_goal_result.v0",
  "goal_ref":"goal-ref-run-goal-materialized-old-001",
  "status":"complete",
  "summary":"receipt terminal antiguo",
  "artifact_refs":["` + staleArtifactRef + `"],
  "artifact_paths":["generated-apps/stale-receipt/source_tree.md"],
  "required_test_results":[{"test_ref":"test-ref-old","status":"passed","evidence_refs":["evidence-ref-old"]}],
  "evidence_refs":["evidence-ref-old"]
}`
	if err := os.WriteFile(filepath.Join(docsDir, goalMaterializedGoalResultFileV0), []byte(receipt), 0o600); err != nil {
		t.Fatalf("write stale receipt: %v", err)
	}

	terminal, terminalOK, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).LoadTerminalGoalMaterializedResultV0(context.Background(), state)
	if err != nil {
		t.Fatalf("LoadTerminalGoalMaterializedResultV0: %v", err)
	}
	if terminalOK || terminal.Status != "" {
		t.Fatalf("receipt antiguo aceptado: terminalOK=%v terminal=%+v", terminalOK, terminal)
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		containsStringV0(result.ArtifactRefs, staleArtifactRef) ||
		containsStringV0(result.EvidenceRefs, "evidence-ref-old") ||
		!containsStringV0(result.IssueCodes, "partial_artifacts_written") {
		t.Fatalf("receipt antiguo no ignorado: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0ReparaReceiptCanonicoFueraDeWriteSetBUG208AFV0(t *testing.T) {
	ctx := context.Background()
	projectDir := t.TempDir()
	writeSet := "generated-apps/receipt-canonical"
	artifactPath := filepath.Join(projectDir, writeSet, "README.md")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o700); err != nil {
		t.Fatalf("mkdir artifact: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("# Generated\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-canonical-repair-001", writeSet)
	artifactRef := "artifact-ref-" + state.RunRef + "-readme"
	state.Spec.ArtifactContracts = []orquestagoal.GoalArtifactContractV0{{
		ArtifactRef:  artifactRef,
		ArtifactType: "source_tree",
		Required:     true,
	}}
	state.Spec.ClosurePolicy = orquestagoal.GoalClosurePolicyV0{
		RequireArtifacts:     true,
		RequireArtifactPaths: true,
	}
	var err error
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		t.Fatalf("NewGoalWorkStateV0: %v", err)
	}
	goalMaterializedWriteCanonicalReceiptForTestV0(t, projectDir, state, `{
  "schema_version":"orquesta_goal_result.v0",
  "goal_ref":"`+state.GoalRef+`",
  "status":"complete",
  "summary":"receipt canonico",
  "artifact_refs":["`+artifactRef+`"],
  "artifact_paths":["`+writeSet+`/README.md"]
}`)
	if err := os.WriteFile(filepath.Join(projectDir, writeSet, "orquesta_goal_result_v0.json"), []byte(`{
  "schema_version":"orquesta_goal_result.v0",
  "goal_ref":"`+state.GoalRef+`",
  "status":"complete",
  "summary":"receipt legacy no debe ganar"
}`), 0o600); err != nil {
		t.Fatalf("write legacy receipt: %v", err)
	}
	store := newGoalFirstQueueStateStoreForTestV0()
	if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	source := stackGoalMaterializedRefsSourceV0{
		Config:                       ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
		GoalStateStore:               store,
		GoalClosureValidator:         orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		RepairMissingTerminalReceipt: true,
	}
	loadedResult, ok, err := source.LoadTerminalGoalMaterializedResultV0(ctx, state)
	if err != nil || !ok || loadedResult.GoalRef != state.GoalRef {
		t.Fatalf("LoadTerminalGoalMaterializedResultV0: ok=%v result=%+v err=%v", ok, loadedResult, err)
	}
	if _, ok, err := source.ResolveDirectorGoalMaterializedRefsV0(ctx, state); err != nil || !ok {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: ok=%v err=%v", ok, err)
	}
	persisted, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusCompleteV0 || persisted.LastClosure == nil || !persisted.LastClosure.Accepted {
		t.Fatalf("receipt canonico no reparo state=%+v", persisted)
	}
	if persisted.LastResult == nil || persisted.LastResult.Summary != "receipt canonico" {
		t.Fatalf("receipt legacy gano precedencia: result=%+v", persisted.LastResult)
	}
	if _, ok, err := source.ResolveDirectorGoalMaterializedRefsV0(ctx, persisted); err != nil || !ok {
		t.Fatalf("replay ResolveDirectorGoalMaterializedRefsV0: ok=%v err=%v", ok, err)
	}
	replayed, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 replay: %v", err)
	}
	if !reflect.DeepEqual(persisted, replayed) {
		t.Fatalf("replay reescribio state: before=%+v after=%+v", persisted, replayed)
	}
}

func TestStackGoalMaterializedRefsSourceV0ReceiptCanonicoInvalidoNoActivaRepairSinteticoBUG208AFV0(t *testing.T) {
	ctx := context.Background()
	projectDir := t.TempDir()
	writeSet := "generated-apps/canonical-invalid-with-qa"
	if err := os.MkdirAll(filepath.Join(projectDir, writeSet), 0o700); err != nil {
		t.Fatalf("mkdir write set: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, writeSet, "artifact.md"), []byte("# valid\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, writeSet, "informe_qa.json"), []byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":true,"strict_editorial_qa_pass":true,"question_bank_publicable":true,"tutor_assets_publicable":true}}`), 0o600); err != nil {
		t.Fatalf("write qa: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-canonical-invalid-qa", writeSet)
	goalMaterializedWriteCanonicalReceiptForTestV0(t, projectDir, state, `{`)
	if err := os.WriteFile(filepath.Join(projectDir, writeSet, "orquesta_goal_result_v0.json"), []byte(`{
  "schema_version":"orquesta_goal_result.v0",
  "goal_ref":"`+state.GoalRef+`",
  "status":"complete",
  "summary":"legacy valido no debe saltar canonical invalido"
}`), 0o600); err != nil {
		t.Fatalf("write legacy receipt: %v", err)
	}
	store := newGoalFirstQueueStateStoreForTestV0()
	if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config:                       ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
		GoalStateStore:               store,
		GoalClosureValidator:         orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		RepairMissingTerminalReceipt: true,
	}).ResolveDirectorGoalMaterializedRefsV0(ctx, state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok || containsStringV0(result.IssueCodes, orquestamcp.MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0) {
		t.Fatalf("receipt canonico invalido tratado como ausente: ok=%v result=%+v", ok, result)
	}
	persisted, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if !reflect.DeepEqual(state, persisted) {
		t.Fatalf("receipt canonico invalido activo repair sintetico: before=%+v after=%+v", state, persisted)
	}
}

func TestStackGoalMaterializedRefsSourceV0NoReparaReceiptCanonicoInvalidoBUG208AFV0(t *testing.T) {
	for name, receipt := range map[string]string{
		"goal_ref_vacio": `{"schema_version":"orquesta_goal_result.v0","goal_ref":"","status":"complete","summary":"sin goal"}`,
		"goal_ref_ajeno": `{"schema_version":"orquesta_goal_result.v0","goal_ref":"goal-ref-ajeno","status":"complete","summary":"otro goal"}`,
		"malformed":      `{`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			projectDir := t.TempDir()
			state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-canonical-invalid-"+name, "generated-apps/empty")
			goalMaterializedWriteCanonicalReceiptForTestV0(t, projectDir, state, receipt)
			store := newGoalFirstQueueStateStoreForTestV0()
			if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
				t.Fatalf("SaveGoalWorkStateV0: %v", err)
			}
			result, ok, err := (stackGoalMaterializedRefsSourceV0{
				Config:                       ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
				GoalStateStore:               store,
				GoalClosureValidator:         orquestagoal.DefaultGoalWorkClosureValidatorV0{},
				RepairMissingTerminalReceipt: true,
			}).ResolveDirectorGoalMaterializedRefsV0(ctx, state)
			if err != nil || ok || len(result.EvidenceRefs) != 0 {
				t.Fatalf("receipt invalido aceptado: ok=%v result=%+v err=%v", ok, result, err)
			}
			persisted, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
			if err != nil {
				t.Fatalf("LoadGoalWorkStateV0: %v", err)
			}
			if !reflect.DeepEqual(state, persisted) {
				t.Fatalf("receipt invalido reescribio state: before=%+v after=%+v", state, persisted)
			}
		})
	}
}

func TestStackGoalMaterializedRefsSourceV0RechazaExternalGoalRefCanonicoContradictorioBUG208AFV0(t *testing.T) {
	ctx := context.Background()
	projectDir := t.TempDir()
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-canonical-external-mismatch", "generated-apps/empty")
	state.ExternalGoalRef = "external-goal-real"
	state.LaunchReceipt.ExternalGoalRef = state.ExternalGoalRef
	var err error
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		t.Fatalf("NewGoalWorkStateV0: %v", err)
	}
	goalMaterializedWriteCanonicalReceiptForTestV0(t, projectDir, state, `{
  "schema_version":"orquesta_goal_result.v0",
  "goal_ref":"`+state.GoalRef+`",
  "external_goal_ref":"external-goal-ajeno",
  "status":"complete",
  "summary":"external contradictorio"
}`)
	store := newGoalFirstQueueStateStoreForTestV0()
	if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	_, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config:                       ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
		GoalStateStore:               store,
		RepairMissingTerminalReceipt: true,
	}).ResolveDirectorGoalMaterializedRefsV0(ctx, state)
	if err != nil || ok {
		t.Fatalf("external_goal_ref contradictorio aceptado: ok=%v err=%v", ok, err)
	}
	persisted, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil || !reflect.DeepEqual(state, persisted) {
		t.Fatalf("estado modificado: state=%+v persisted=%+v err=%v", state, persisted, err)
	}
}

func TestStackGoalMaterializedRefsSourceV0RechazaReceiptCanonicoSymlinkFueraDelProyectoBUG208AFV0(t *testing.T) {
	projectDir := t.TempDir()
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-canonical-symlink", "generated-apps/empty")
	outside := filepath.Join(t.TempDir(), "receipt.json")
	if err := os.WriteFile(outside, []byte(`{"schema_version":"orquesta_goal_result.v0","goal_ref":"`+state.GoalRef+`","status":"complete"}`), 0o600); err != nil {
		t.Fatalf("write outside receipt: %v", err)
	}
	canonicalPath := goalMaterializedCanonicalReceiptPathForTestV0(projectDir, state)
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0o700); err != nil {
		t.Fatalf("mkdir canonical receipt: %v", err)
	}
	if err := os.Symlink(outside, canonicalPath); err != nil {
		t.Fatalf("symlink canonical receipt: %v", err)
	}
	scan, err := (stackGoalMaterializedRefsSourceV0{}).scanCanonicalGoalMaterializedReceiptV0(projectDir, state)
	if err != nil || !scan.HasInvalidCanonicalReceipt || scan.TerminalResult != nil {
		t.Fatalf("symlink canonico no rechazado: scan=%+v err=%v", scan, err)
	}
}

func TestStackGoalMaterializedRefsSourceV0MantieneBlockedAjenoSinReceiptBUG208AFV0(t *testing.T) {
	ctx := context.Background()
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-blocked-intact-001", "generated-apps/empty")
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusBlockedV0,
		GoalRef:       state.GoalRef,
		Summary:       "blocked ajeno",
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{Status: orquestagoal.GoalStatusBlockedV0, NeedsRework: true}
	var err error
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		t.Fatalf("NewGoalWorkStateV0: %v", err)
	}
	store := newGoalFirstQueueStateStoreForTestV0()
	if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config:                       ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()}},
		GoalStateStore:               store,
		RepairMissingTerminalReceipt: true,
	}).ResolveDirectorGoalMaterializedRefsV0(ctx, state)
	if err != nil || ok || len(result.EvidenceRefs) != 0 {
		t.Fatalf("blocked ajeno modificado: ok=%v result=%+v err=%v", ok, result, err)
	}
	persisted, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if !reflect.DeepEqual(state, persisted) {
		t.Fatalf("blocked ajeno reescrito: before=%+v after=%+v", state, persisted)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaArtefactosOPESFueraDeWriteSet(t *testing.T) {
	projectDir := t.TempDir()
	phaseDir := filepath.Join(projectDir, "temas", "tema_003", "coordinacion_wave14")
	docsDir := filepath.Join(projectDir, "temas", "tema_003", "trabajo", "docs")
	htmlDir := filepath.Join(projectDir, "temas", "tema_003", "html")
	testsDir := filepath.Join(projectDir, "temas", "tema_003", "tests")
	for _, dir := range []string{phaseDir, docsDir, htmlDir, testsDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(phaseDir, "checkpoint_started.txt"), []byte("started\n"), 0o600); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "orquesta_goal_result_v0.json"), []byte(`{"status":"complete"}`), 0o600); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(htmlDir, "index.html"), []byte("<main>tema fuera de scope</main>\n"), 0o600); err != nil {
		t.Fatalf("write html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "preguntas.json"), []byte(`{"questions":[]}`), 0o600); err != nil {
		t.Fatalf("write tests: %v", err)
	}
	state := goalMaterializedRefsOPESStateForTestV0(t, "run-goal-materialized-opes-out-scope-001", "temas/tema_003/coordinacion_wave14")
	state.Spec.WriteSet = append(state.Spec.WriteSet, orquestagoal.GoalWriteScopeV0{
		Path:    "temas/tema_003/trabajo/docs",
		Purpose: "receipt",
	})
	state.Spec.ClosurePolicy.RequireArtifactPaths = true
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       state.GoalRef,
		ArtifactPaths: []string{
			"temas/tema_003/coordinacion_wave14/checkpoint_started.txt",
			"temas/tema_003/trabajo/docs/orquesta_goal_result_v0.json",
		},
		ArtifactRefs: []string{"artifact-ref-phase0"},
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "out_of_scope_materialized_artifacts") ||
		!containsStringPrefixForTestV0(result.EvidenceRefs, "evidence-ref-goal-materialized-out-of-scope-artifacts:") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-out-of-scope-artifacts") {
		t.Fatalf("out-of-scope OPES no detectado: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaArtifactPathDeclaradoPeroBorradoV0(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_041")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-missing-artifact-001", "temas/tema_041")
	state.Status = orquestagoal.GoalStatusCompleteV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       state.GoalRef,
		ArtifactPaths: []string{"temas/tema_041/tema_ampliado.md"},
		ArtifactRefs:  []string{"artifact-ref-tema-041-ampliado"},
		EvidenceRefs:  []string{"evidence-ref-goal-result-durable-041"},
	}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "terminal_artifact_missing_after_goal_complete") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-terminal-artifact-missing-after-complete") ||
		!containsStringPrefixForTestV0(result.EvidenceRefs, "evidence-ref-goal-materialized-terminal-artifact-missing-after-complete:") {
		t.Fatalf("artefacto terminal borrado no detectado: ok=%v result=%+v", ok, result)
	}
	if containsStringV0(result.IssueCodes, "artifact_paths_omitted_materialized") {
		t.Fatalf("no debe tratar path declarado y ausente como path omitido: %+v", result.IssueCodes)
	}
}

func TestStackGoalMaterializedRefsSourceV0DetectaRequiredTestEvidenceAusenteEnReceiptTerminal(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_037")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(topicDir, "orquesta_goal_result_v0.json"),
		[]byte(`{"status":"complete","artifact_paths":["temas/tema_037/tema_ampliado.md"],"required_test_results":[{"test_ref":"test-ref-qa","status":"passed","evidence_refs":[]}]}`),
		0o600,
	); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-required-test-evidence-001", "temas/tema_037")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		!containsStringV0(result.IssueCodes, "required_test_evidence_missing") ||
		containsStringV0(result.IssueCodes, "missing_terminal_receipt_after_artifacts_pass") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-required-test-evidence-missing") ||
		!containsStringPrefixForTestV0(result.ArtifactRefs, "artifact-ref-materialized:") {
		t.Fatalf("required-test evidence ausente no detectada: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0UsaRequiredTestEvidenceMaterializadaFueraDeReceipt(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_040")
	if err := os.MkdirAll(topicDir, 0o700); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(topicDir, "orquesta_goal_result_v0.json"),
		[]byte(`{"status":"complete","artifact_paths":["temas/tema_040/tema_ampliado.md"],"required_test_results":[{"test_ref":"test-ref-tema-040-qa","status":"passed","evidence_refs":[]}]}`),
		0o600,
	); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(topicDir, "required_test_evidence_qa.json"),
		[]byte(`{
			"schema_version":"required_test_evidence.v0",
			"evidence_ref":"evidence-ref-required-test-file-040",
			"run_ref":"run-goal-materialized-required-test-external-001",
			"task_ref":"task-ref-tema-040",
			"test_command":"validar tema 040",
			"status":"passed",
			"delivery_ref":"delivery-ref-tema-040",
			"review_request_id":"review-request-tema-040",
			"review_result_ref":"review-result-tema-040",
			"accepted_review_ref":"accepted-review-tema-040",
			"occurred_at":"2026-07-02T00:00:00Z",
			"evidence_refs":["evidence-ref-required-test-log-040"]
		}`),
		0o600,
	); err != nil {
		t.Fatalf("write required test evidence: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-required-test-external-001", "temas/tema_040")
	state.Spec.RequiredTests = []orquestagoal.GoalRequiredTestV0{{
		TestRef: "test-ref-tema-040-qa",
		Command: "validar tema 040",
	}}

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if !ok ||
		containsStringV0(result.IssueCodes, "required_test_evidence_missing") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-required-test-file-040") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-required-test-log-040") ||
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-required-test-evidence-detected") {
		t.Fatalf("required test evidence materializada no usada: ok=%v result=%+v", ok, result)
	}
}

func TestStackGoalMaterializedRefsSourceV0NoAceptaQAPassPorTextoLibre(t *testing.T) {
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_030")
	validationDir := filepath.Join(topicDir, "09_validacion")
	if err := os.MkdirAll(validationDir, 0o700); err != nil {
		t.Fatalf("mkdir validation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validationDir, "informe_qa.md"), []byte("QA pass\n"), 0o600); err != nil {
		t.Fatalf("write qa markdown: %v", err)
	}
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-materialized-text-qa-001", "temas/tema_030")

	result, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir}},
	}).ResolveDirectorGoalMaterializedRefsV0(context.Background(), state)
	if err != nil {
		t.Fatalf("ResolveDirectorGoalMaterializedRefsV0: %v", err)
	}
	if ok && containsStringV0(result.IssueCodes, "missing_terminal_receipt_after_artifacts_pass") {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
}

func goalMaterializedRefsStateForTestV0(
	t *testing.T,
	runRef string,
	writeSet string,
) orquestagoal.GoalWorkStateV0 {
	t.Helper()
	goalRef := "goal-ref-" + runRef
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:   goalRef,
			RunRef:    runRef,
			Objective: "detectar work_delivery materializado",
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path:    writeSet,
				Purpose: "delivery",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusRunningV0,
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	return state
}

func goalMaterializedRefsOPESStateForTestV0(
	t *testing.T,
	runRef string,
	writeSet string,
) orquestagoal.GoalWorkStateV0 {
	t.Helper()
	state := goalMaterializedRefsStateForTestV0(t, runRef, writeSet)
	state.Spec.DomainRef = "opes"
	state.Spec.ContextRefs = []orquestagoal.GoalContextRefV0{{
		Kind:    "domain",
		Ref:     "opes://temario/tcae",
		Purpose: "qa-materializada",
	}}
	return state
}

func goalMaterializedWriteCanonicalReceiptForTestV0(
	t *testing.T,
	projectDir string,
	state orquestagoal.GoalWorkStateV0,
	receipt string,
) {
	t.Helper()
	path := goalMaterializedCanonicalReceiptPathForTestV0(projectDir, state)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir canonical receipt: %v", err)
	}
	if err := os.WriteFile(path, []byte(receipt), 0o600); err != nil {
		t.Fatalf("write canonical receipt: %v", err)
	}
}

func goalMaterializedCanonicalReceiptPathForTestV0(projectDir string, state orquestagoal.GoalWorkStateV0) string {
	goalRef := goalMaterializedStateGoalRefV0(state)
	return filepath.Join(
		projectDir,
		filepath.FromSlash(orquestaruntimecodexgoal.CodexGoalRuntimeReceiptRelativeDirV0(goalRef)),
		orquestaruntimecodexgoal.CodexGoalResultFileNameForGoalRefV0(goalRef),
	)
}

func containsStringPrefixForTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
