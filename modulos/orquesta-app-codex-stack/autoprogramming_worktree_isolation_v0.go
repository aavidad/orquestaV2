package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
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
) (orquestaautoprogramming.AutoprogrammingProgrammableWorkV0, []orquestaautoprogramming.AutoprogrammingRequestIssueV0) {
	projectWorkDir = strings.TrimSpace(projectWorkDir)
	if projectWorkDir == "" {
		return work, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{
			Code:    "worktree_isolation_missing",
			Field:   "project_work_dir",
			Message: "project_work_dir requerido para materializar worktree antes de lanzar agentes",
		}}
	}
	for _, task := range work.Tasks {
		isolation, issues := autoprogrammingPrepareTaskWorktreeIsolationV0(ctx, projectWorkDir, work, task)
		if len(issues) > 0 {
			return work, autoprogrammingWorktreeIssuesV0(issues)
		}
		work = autoprogrammingApplyTaskWorktreeIsolationV0(work, task.TaskID, isolation)
	}
	return work, nil
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
