package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (resolver CodexLaunchSpecResolverV0) ensureLaunchWorktreeIsolationV0(
	ctx context.Context,
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) error {
	if !isProgrammingPhaseV0(payload.PhaseID) || resolver.TaskStore == nil {
		return nil
	}
	tasks, err := resolver.TaskStore.LoadWorkflowTasksV0(
		ctx,
		strings.TrimSpace(payload.RunID),
		[]string{strings.TrimSpace(payload.TaskRef)},
	)
	if err != nil || len(tasks) != 1 {
		return err
	}
	task := tasks[0]
	worktreeRef := workflowTaskContextRefValueV0(task.ContextRefs, "worktree_ref:")
	branchRef := workflowTaskContextRefValueV0(task.ContextRefs, "branch_ref:")
	if strings.TrimSpace(worktreeRef) == "" && strings.TrimSpace(branchRef) == "" {
		return nil
	}
	if strings.TrimSpace(resolver.Config.ProjectWorkDir) == "" {
		return fmt.Errorf("worktree_isolation_missing: project_work_dir")
	}
	work := orquestaautoprogramming.AutoprogrammingProgrammableWorkV0{
		ProjectRef:  workflowTaskContextRefValueV0(task.ContextRefs, "project_ref:"),
		WorktreeRef: worktreeRef,
		BranchRef:   branchRef,
	}
	_, issues := autoprogrammingPrepareTaskWorktreeIsolationV0(
		ctx,
		resolver.Config.ProjectWorkDir,
		work,
		task,
	)
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("worktree_isolation_invalid: %s:%s", issues[0].Field, issues[0].Code)
}
