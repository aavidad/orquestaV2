package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func autoprogrammingScopeObjectiveLineV0(task orquestacoreworkflow.WorkflowTaskV0) string {
	worktreeRef := workflowTaskContextRefValueV0(task.ContextRefs, "worktree_ref:")
	branchRef := workflowTaskContextRefValueV0(task.ContextRefs, "branch_ref:")
	if worktreeRef == "" || branchRef == "" {
		return ""
	}
	return "Autoprogramacion: worktree aislada validada; conserva worktree_ref=" +
		worktreeRef + " y branch_ref=" + branchRef + " como refs opacas, sin convertirlas en rutas ni nombres Git."
}

func autoprogrammingScopeDoneCriteriaV0(task orquestacoreworkflow.WorkflowTaskV0) []string {
	if autoprogrammingScopeObjectiveLineV0(task) == "" {
		return nil
	}
	return []string{
		"Worktree aislada y branch_ref opaco preservados en la entrega.",
	}
}

func workflowTaskContextRefValueV0(refs []string, prefix string) string {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if strings.HasPrefix(ref, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(ref, prefix))
		}
	}
	return ""
}
