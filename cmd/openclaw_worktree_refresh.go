package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"orquesta/coordinacion"
	"orquesta/db"
)

func refreshSupervisorCleanWorktreeDrift(item apiOpenClawWorktreeDrift) (map[string]any, error) {
	path := strings.TrimSpace(item.Path)
	if path == "" {
		return nil, fmt.Errorf("worktree drift sin path")
	}
	worktree, err := db.CoordinationWorktreeRepository().GetActiveByPath(path)
	if err != nil {
		return nil, err
	}
	if worktree == nil {
		return nil, fmt.Errorf("no existe worktree activa para %s", path)
	}
	if item.Dirty {
		return nil, fmt.Errorf("la worktree %s no está limpia", path)
	}
	svc := newCoordinationService()
	closed, err := svc.CloseWorktree(worktree.ID, true, "refresco_worktree_desfasada_limpia")
	if err != nil {
		return nil, err
	}
	reopened, err := svc.PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: strconv.FormatInt(worktree.ProjectID, 10),
		TaskID:     worktree.TaskID,
		LockID:     worktree.LockID,
		Agent:      strings.TrimSpace(worktree.Agent),
		Name:       strings.TrimSpace(worktree.Name),
		Branch:     strings.TrimSpace(worktree.Branch),
		BaseRef:    "HEAD",
		Reason:     "refresco_worktree_desfasada_limpia",
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"closed_worktree_id":   closed.ID,
		"reopened_worktree_id": reopened.ID,
		"path":                 reopened.Path,
		"branch":               reopened.Branch,
		"base_ref":             reopened.BaseRef,
	}, nil
}
