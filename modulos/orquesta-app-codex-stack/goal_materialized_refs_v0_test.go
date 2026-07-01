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
