package cmd

import (
	"strings"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitoperaciones"
)

type worktreeRuntimeService struct{}

func (worktreeRuntimeService) ResolveActiveWorktree(proyectoSlug, agente string) (*db.Worktree, error) {
	return gitService.ResolveActiveWorktree(strings.TrimSpace(proyectoSlug), strings.TrimSpace(agente))
}

func (worktreeRuntimeService) EnsureActiveWorktree(proyectoSlug, agente string) (*db.Worktree, error) {
	proyecto, err := runtimesService.GetProject(strings.TrimSpace(proyectoSlug))
	if err != nil || proyecto == nil || proyecto.ID <= 0 {
		return nil, err
	}
	if item, err := gitService.ResolveActiveWorktree(strings.TrimSpace(proyecto.Slug), strings.TrimSpace(agente)); err == nil && item != nil {
		return item, nil
	}
	svc := &coordinacion.Service{
		Locks:     db.CoordinationLockRepository(),
		Worktrees: db.CoordinationWorktreeRepository(),
		Projects:  db.CoordinationProjectRepository(),
		Sessions:  db.CoordinationSessionRepository(),
		Config:    db.CoordinationConfigRepository(),
		Workspace: gitoperaciones.WorktreeManager{},
	}
	worktree, err := svc.PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: strings.TrimSpace(proyecto.Slug),
		Agent:      strings.TrimSpace(agente),
		Reason:     "microprogramacion_git",
		BaseRef:    "HEAD",
	})
	if err != nil || worktree == nil {
		return nil, err
	}
	return &db.Worktree{
		ID:           worktree.ID,
		ProyectoID:   worktree.ProjectID,
		ProyectoSlug: strings.TrimSpace(proyecto.Slug),
		TareaID:      worktree.TaskID,
		LockID:       worktree.LockID,
		Agente:       strings.TrimSpace(worktree.Agent),
		Nombre:       strings.TrimSpace(worktree.Name),
		RutaAbs:      strings.TrimSpace(worktree.Path),
		Branch:       strings.TrimSpace(worktree.Branch),
		BaseRef:      strings.TrimSpace(worktree.BaseRef),
		Estado:       string(worktree.State),
		Motivo:       strings.TrimSpace(worktree.Reason),
		CreatedAt:    worktree.CreatedAt,
		UpdatedAt:    worktree.UpdatedAt,
		CerradaAt:    worktree.ClosedAt,
	}, nil
}
