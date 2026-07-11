package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexStackMaterialProgressEvidenceV0ClasificaDiffVerificadoV0(t *testing.T) {
	projectDir, store, state := materialProgressEvidenceFixtureV0(t)
	if err := os.WriteFile(filepath.Join(projectDir, "feature.md"), []byte("after\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := CodexStackMaterialProgressEvidenceV0{Stack: &StackV0{
		Codex:                    CodexRuntimeConfigV0{ProjectWorkDir: projectDir},
		AutoprogrammingPromotion: AutoprogrammingPromotionConfigV0{GoalFirstSnapshotStore: store},
	}}
	result, err := source.ClassifyMaterialProgressV0(context.Background(), orquestaautoprogramming.MaterialProgressEvidenceRequestV0{
		State: state, Result: materialProgressRunningResultV0(state),
	})
	if err != nil || !result.Verified || result.MaterialClass != orquestaautoprogramming.MaterialProgressClassDiffV0 ||
		len(result.EvidenceRefs) != 1 || result.ContextRevisionRef == "" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexStackMaterialProgressEvidenceV0SinCambioEsNoneV0(t *testing.T) {
	projectDir, store, state := materialProgressEvidenceFixtureV0(t)
	source := CodexStackMaterialProgressEvidenceV0{Stack: &StackV0{
		Codex:                    CodexRuntimeConfigV0{ProjectWorkDir: projectDir},
		AutoprogrammingPromotion: AutoprogrammingPromotionConfigV0{GoalFirstSnapshotStore: store},
	}}
	result, err := source.ClassifyMaterialProgressV0(context.Background(), orquestaautoprogramming.MaterialProgressEvidenceRequestV0{
		State: state, Result: materialProgressRunningResultV0(state),
	})
	if err != nil || !result.Verified || result.MaterialClass != orquestaautoprogramming.MaterialProgressClassNoneV0 ||
		len(result.EvidenceRefs) != 1 || result.EvidenceRefs[0] != "evidence-ref-material-progress-no-diff" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexStackMaterialProgressEvidenceV0ReceiptPrecedeDiffV0(t *testing.T) {
	_, _, state := materialProgressEvidenceFixtureV0(t)
	source := CodexStackMaterialProgressEvidenceV0{Stack: &StackV0{}}
	result := materialProgressRunningResultV0(state)
	result.DomainReceiptRefs = []string{"receipt-ref-material-progress"}
	classified, err := source.ClassifyMaterialProgressV0(context.Background(), orquestaautoprogramming.MaterialProgressEvidenceRequestV0{
		State: state, Result: result,
	})
	if err != nil || !classified.Verified || classified.MaterialClass != orquestaautoprogramming.MaterialProgressClassReceiptV0 {
		t.Fatalf("result=%+v err=%v", classified, err)
	}
}

func TestCodexStackMaterialProgressEvidenceV0CambioFueraDeWriteSetNoRenuevaV0(t *testing.T) {
	projectDir, store, state := materialProgressEvidenceFixtureV0(t)
	if err := os.WriteFile(filepath.Join(projectDir, "outside.md"), []byte("outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := CodexStackMaterialProgressEvidenceV0{Stack: &StackV0{
		Codex:                    CodexRuntimeConfigV0{ProjectWorkDir: projectDir},
		AutoprogrammingPromotion: AutoprogrammingPromotionConfigV0{GoalFirstSnapshotStore: store},
	}}
	result, err := source.ClassifyMaterialProgressV0(context.Background(), orquestaautoprogramming.MaterialProgressEvidenceRequestV0{
		State: state, Result: materialProgressRunningResultV0(state),
	})
	if err != nil || !result.Verified || result.MaterialClass != orquestaautoprogramming.MaterialProgressClassNoneV0 ||
		len(result.EvidenceRefs) < 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func materialProgressEvidenceFixtureV0(t *testing.T) (string, orquestaruntimeworktree.WorktreeSnapshotStorePortV0, orquestagoal.GoalWorkStateV0) {
	t.Helper()
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "feature.md"), []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseline, issues := orquestaruntimeworktree.CaptureWorktreeSnapshotV0(context.Background(), orquestaruntimeworktree.WorktreeSnapshotRequestV0{
		SnapshotRef: "baseline-ref-material-progress-stack", ProjectWorkDir: projectDir,
	})
	if len(issues) > 0 {
		t.Fatalf("capture issues=%+v", issues)
	}
	store := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	if err := store.RecordWorktreeSnapshotV0(context.Background(), baseline); err != nil {
		t.Fatal(err)
	}
	writeSet := []orquestagoal.GoalWriteScopeV0{{Path: "feature.md"}}
	state := orquestagoal.GoalWorkStateV0{
		RunRef: "run-ref-material-progress-stack", GoalRef: "goal-ref-material-progress-stack",
		Spec: orquestagoal.GoalWorkSpecV0{
			WriteSet: writeSet, WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0(writeSet),
			ContextRefs: []orquestagoal.GoalContextRefV0{{Kind: "worktree_baseline", Ref: baseline.SnapshotRef}},
		},
	}
	return projectDir, store, state
}

func materialProgressRunningResultV0(state orquestagoal.GoalWorkStateV0) orquestagoal.GoalWorkResultV0 {
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0, Status: orquestagoal.GoalStatusRunningV0,
		GoalRef: state.GoalRef,
	}
}
