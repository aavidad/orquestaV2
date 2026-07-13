package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func (stack StackV0) autoprogrammingGoalProjectWorkDirV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (string, error) {
	provisioner := stack.AutoprogrammingPromotion.GoalWorkspaceProvisioner
	if provisioner == nil {
		projectDir := stack.autoprogrammingCanonicalWorkDirV0()
		if projectDir == "" {
			return "", fmt.Errorf("autoprogramming_goal_project_work_dir_missing")
		}
		return projectDir, nil
	}
	state = orquestagoal.NormalizeGoalWorkStateV0(state)
	workspaceRef := autoprogrammingPromotionGoalContextRefV0(state.Spec.ContextRefs, "goal_workspace", "")
	if workspaceRef == "" {
		return "", fmt.Errorf("autoprogramming_goal_workspace_ref_missing")
	}
	workspace, issues := provisioner.ResolveGoalWorkspaceV0(ctx, orquestaruntimeworktree.GoalWorkspaceRequestV0{
		RunRef:        strings.TrimSpace(state.Spec.RequestRef),
		GoalRef:       strings.TrimSpace(state.GoalRef),
		ProjectRef:    strings.TrimSpace(state.Spec.ProjectRef),
		WorktreeRef:   autoprogrammingPromotionGoalContextRefV0(state.Spec.ContextRefs, "worktree", "worktree_ref:"),
		SourceWorkDir: stack.autoprogrammingCanonicalWorkDirV0(),
		WorkspaceRoot: strings.TrimSpace(stack.AutoprogrammingPromotion.GoalWorkspaceRoot),
	})
	if len(issues) > 0 || strings.TrimSpace(workspace.ProjectWorkDir) == "" || strings.TrimSpace(workspace.WorkspaceID) != workspaceRef {
		return "", fmt.Errorf("autoprogramming_goal_workspace_unavailable")
	}
	return strings.TrimSpace(workspace.ProjectWorkDir), nil
}

func (stack StackV0) autoprogrammingCanonicalWorkDirV0() string {
	if configured := strings.TrimSpace(stack.AutoprogrammingPromotion.CanonicalWorkDir); configured != "" {
		return configured
	}
	return strings.TrimSpace(stack.Codex.ProjectWorkDir)
}
