package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

const (
	autoprogrammingWorktreeIsolationEvidencePrefixV0 = "worktree_isolation_evidence_ref:"
	autoprogrammingWorktreeIsolationRefPrefixV0      = "worktree_isolation_ref:"
	autoprogrammingWorktreeBaselineRefPrefixV0       = "worktree_baseline_ref:"
)

func autoprogrammingPrepareWorktreeIsolationV0(
	ctx context.Context,
	projectWorkDir string,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	snapshotStore orquestaruntimeworktree.WorktreeSnapshotStorePortV0,
) (orquestaautoprogramming.AutoprogrammingProgrammableWorkV0, []orquestaautoprogramming.AutoprogrammingRequestIssueV0) {
	projectWorkDir = strings.TrimSpace(projectWorkDir)
	if projectWorkDir == "" {
		return work, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{
			Code:    "worktree_isolation_missing",
			Field:   "project_work_dir",
			Message: "project_work_dir requerido para materializar worktree antes de lanzar agentes",
		}}
	}
	for _, task := range autoprogrammingWorktreeIsolationTasksV0(work) {
		isolation, issues := autoprogrammingPrepareTaskWorktreeIsolationV0(ctx, projectWorkDir, work, task)
		if len(issues) > 0 {
			return work, autoprogrammingWorktreeIssuesV0(issues)
		}
		if snapshotStore != nil {
			if err := snapshotStore.RecordWorktreeSnapshotV0(ctx, isolation.Snapshot); err != nil {
				return work, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{
					Code:    "worktree_baseline_store_failed",
					Field:   "worktree_baseline",
					Message: "no se pudo persistir el baseline congelado antes de lanzar el goal",
				}}
			}
		}
		work = autoprogrammingApplyTaskWorktreeIsolationV0(work, task.TaskID, isolation)
	}
	return work, nil
}

func autoprogrammingWorktreeIsolationTasksV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) []orquestacoreworkflow.WorkflowTaskV0 {
	if len(work.Tasks) > 0 {
		return append([]orquestacoreworkflow.WorkflowTaskV0(nil), work.Tasks...)
	}
	tasks := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(work.GoalSpecs))
	for _, spec := range work.GoalSpecs {
		tasks = append(tasks, orquestacoreworkflow.WorkflowTaskV0{
			TaskID: strings.TrimPrefix(strings.TrimSpace(spec.GoalRef), "goal-ref-"),
		})
	}
	return tasks
}

func autoprogrammingPrepareTaskWorktreeIsolationV0(
	ctx context.Context,
	projectWorkDir string,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestaruntimeworktree.WorktreeIsolationV0, []orquestaruntimeworktree.WorktreeIssueV0) {
	worktreeRef := firstNonEmptyAutoprogrammingStackV0(
		workflowTaskContextRefValueV0(task.ContextRefs, "worktree_ref:"),
		work.WorktreeRef,
	)
	branchRef := firstNonEmptyAutoprogrammingStackV0(
		workflowTaskContextRefValueV0(task.ContextRefs, "branch_ref:"),
		work.BranchRef,
	)
	return orquestaruntimeworktree.PrepareIsolatedWorktreeV0(ctx, orquestaruntimeworktree.WorktreeIsolationRequestV0{
		IsolationRef:         "worktree-isolation-ref-" + strings.TrimSpace(task.TaskID),
		ProjectRef:           strings.TrimSpace(work.ProjectRef),
		WorktreeRef:          worktreeRef,
		BranchRef:            branchRef,
		ProjectWorkDir:       projectWorkDir,
		Isolated:             true,
		BaselineRef:          "worktree-baseline-ref-" + strings.TrimSpace(task.TaskID),
		IgnorePrefixes:       codexStackWorktreeIgnorePrefixesV0(),
		AllowPartialSnapshot: true,
	})
}

func autoprogrammingApplyTaskWorktreeIsolationV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	taskRef string,
	isolation orquestaruntimeworktree.WorktreeIsolationV0,
) orquestaautoprogramming.AutoprogrammingProgrammableWorkV0 {
	for index := range work.Tasks {
		if strings.TrimSpace(work.Tasks[index].TaskID) != strings.TrimSpace(taskRef) {
			continue
		}
		work.Tasks[index].ContextRefs = autoprogrammingTaskWorktreeIsolationContextRefsV0(
			work.Tasks[index].ContextRefs,
			isolation,
		)
	}
	for index := range work.Groups {
		if strings.TrimSpace(work.Groups[index].Task.TaskID) != strings.TrimSpace(taskRef) {
			continue
		}
		work.Groups[index].Task.ContextRefs = autoprogrammingTaskWorktreeIsolationContextRefsV0(
			work.Groups[index].Task.ContextRefs,
			isolation,
		)
	}
	goalRef := "goal-ref-" + strings.TrimSpace(taskRef)
	for index := range work.GoalSpecs {
		if strings.TrimSpace(work.GoalSpecs[index].GoalRef) != goalRef {
			continue
		}
		work.GoalSpecs[index].ContextRefs = append(
			work.GoalSpecs[index].ContextRefs,
			orquestagoal.GoalContextRefV0{
				Kind:     "worktree_baseline",
				Ref:      strings.TrimSpace(isolation.BaselineRef),
				Purpose:  "Baseline congelado del worktree antes de lanzar el Goal.",
				Required: true,
			},
		)
		work.GoalSpecs[index] = orquestagoal.NormalizeGoalWorkSpecV0(work.GoalSpecs[index])
	}
	return work
}

func autoprogrammingTaskWorktreeIsolationContextRefsV0(
	refs []string,
	isolation orquestaruntimeworktree.WorktreeIsolationV0,
) []string {
	out := append([]string(nil), refs...)
	out = append(out,
		autoprogrammingWorktreeIsolationRefPrefixV0+strings.TrimSpace(isolation.IsolationRef),
		autoprogrammingWorktreeBaselineRefPrefixV0+strings.TrimSpace(isolation.BaselineRef),
	)
	for _, ref := range isolation.EvidenceRefs {
		out = append(out, autoprogrammingWorktreeIsolationEvidencePrefixV0+strings.TrimSpace(ref))
	}
	return compactStringsV0(out)
}

func autoprogrammingWorktreeIssuesV0(
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) []orquestaautoprogramming.AutoprogrammingRequestIssueV0 {
	out := make([]orquestaautoprogramming.AutoprogrammingRequestIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestaautoprogramming.AutoprogrammingRequestIssueV0{
			Code:    "worktree_isolation_invalid",
			Field:   strings.TrimSpace(issue.Field),
			Message: string(issue.Code),
		})
	}
	return out
}
