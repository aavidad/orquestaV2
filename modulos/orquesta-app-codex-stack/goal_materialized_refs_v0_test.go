package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
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
		!containsStringV0(result.EvidenceRefs, "evidence-ref-goal-materialized-checkpoint-detected") ||
		!containsStringV0(result.IssueCodes, "goal_first_materialized_checkpoint_detected") {
		t.Fatalf("ok=%v result=%+v", ok, result)
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
		[]byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":true,"strict_editorial_qa_pass":true}}`),
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
		len(result.ExpectedReceiptRefs) != 2 {
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
		[]byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":true,"strict_editorial_qa_pass":true}}`),
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
		[]byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":true,"strict_editorial_qa_pass":true}}`),
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

func containsStringPrefixForTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
