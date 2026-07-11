package orquestaappcodexstack

import (
	"context"
	"path/filepath"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestAutoprogrammingGoalProjectWorkDirV0ResolvesPhysicalWorkspaceV0(t *testing.T) {
	root := t.TempDir()
	provisioner := &fakeGoalWorkspaceProvisionerForStackTestV0{root: root}
	stack := StackV0{
		Codex: CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()},
		AutoprogrammingPromotion: AutoprogrammingPromotionConfigV0{
			GoalWorkspaceProvisioner: provisioner,
			GoalWorkspaceRoot:        t.TempDir(),
		},
	}
	state := orquestagoal.GoalWorkStateV0{
		GoalRef: "goal-ref-physical-001",
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef: "goal-ref-physical-001", RequestRef: "request-ref-physical-001", ProjectRef: "project-ref-physical-001",
			ContextRefs: []orquestagoal.GoalContextRefV0{
				{Kind: "worktree", Ref: "worktree-ref-physical-001"},
				{Kind: "goal_workspace", Ref: "workspace-goal-ref-physical-001"},
			},
		},
	}
	got, err := stack.autoprogrammingGoalProjectWorkDirV0(context.Background(), state)
	if err != nil || got != filepath.Join(root, state.GoalRef) {
		t.Fatalf("got=%q err=%v", got, err)
	}
	state.Spec.ContextRefs[1].Ref = "workspace-ref-wrong"
	if _, err := stack.autoprogrammingGoalProjectWorkDirV0(context.Background(), state); err == nil {
		t.Fatal("workspace identity mismatch must fail closed")
	}
}
