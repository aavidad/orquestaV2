package cmd

import (
	"fmt"
	"os"
	"strings"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitoperaciones"
)

type worktreeRuntimeService struct{}

func (worktreeRuntimeService) ResolveActiveWorktree(proyectoSlug, agente string) (*db.Worktree, error) {
	proyecto, err := runtimesService.GetProject(strings.TrimSpace(proyectoSlug))
	if err != nil || proyecto == nil {
		return nil, err
	}
	item, err := gitService.ResolveActiveWorktree(strings.TrimSpace(proyectoSlug), strings.TrimSpace(agente))
	if err != nil || item == nil {
		return item, err
	}
	if err := sincronizarWorkspaceProyectoEnWorktree(proyecto, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (worktreeRuntimeService) EnsureActiveWorktree(proyectoSlug, agente string) (*db.Worktree, error) {
	proyecto, err := runtimesService.GetProject(strings.TrimSpace(proyectoSlug))
	if err != nil || proyecto == nil || proyecto.ID <= 0 {
		return nil, err
	}
	if item, err := gitService.ResolveActiveWorktree(strings.TrimSpace(proyecto.Slug), strings.TrimSpace(agente)); err == nil && item != nil {
		reutilizable, err := worktreeActivaReutilizableParaPremium(proyecto, item)
		if err != nil {
			return nil, err
		}
		if !reutilizable {
			if err := descartarWorktreeActivaSucia(proyecto, item); err != nil {
				return nil, err
			}
		} else {
			if err := sincronizarWorkspaceProyectoEnWorktree(proyecto, item); err != nil {
				return nil, err
			}
			return item, nil
		}
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
	item := &db.Worktree{
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
	}
	if err := sincronizarWorkspaceProyectoEnWorktree(proyecto, item); err != nil {
		return nil, err
	}
	return item, nil
}

func worktreeActivaReutilizableParaPremium(proyecto *db.Proyecto, item *db.Worktree) (bool, error) {
	if proyecto == nil || item == nil {
		return false, fmt.Errorf("proyecto y worktree obligatorios")
	}
	if worktreeSyncDirtyWorkspaceEnabled() || !proyectoPareceRepoGit(strings.TrimSpace(proyecto.RutaAbs)) {
		return true, nil
	}
	return gitService.IsWorktreeClean(strings.TrimSpace(item.RutaAbs))
}

func descartarWorktreeActivaSucia(proyecto *db.Proyecto, item *db.Worktree) error {
	if proyecto == nil || item == nil {
		return fmt.Errorf("proyecto y worktree obligatorios")
	}
	svc := &coordinacion.Service{
		Locks:     db.CoordinationLockRepository(),
		Worktrees: db.CoordinationWorktreeRepository(),
		Projects:  db.CoordinationProjectRepository(),
		Sessions:  db.CoordinationSessionRepository(),
		Config:    db.CoordinationConfigRepository(),
		Workspace: gitoperaciones.WorktreeManager{},
	}
	if item.ID > 0 {
		if _, err := svc.CloseWorktree(item.ID, true, "worktree_sucia_recreada"); err != nil {
			if _, fallbackErr := svc.CloseWorktree(item.ID, false, "worktree_sucia_recreada"); fallbackErr != nil {
				return err
			}
		}
	}
	ruta := strings.TrimSpace(item.RutaAbs)
	if ruta != "" {
		if _, err := os.Stat(ruta); err == nil {
			if removeErr := os.RemoveAll(ruta); removeErr != nil {
				return removeErr
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if proyectoPareceRepoGit(strings.TrimSpace(proyecto.RutaAbs)) {
		if err := (gitoperaciones.WorktreeManager{}).PruneWorktrees(strings.TrimSpace(proyecto.RutaAbs)); err != nil {
			return err
		}
	}
	return nil
}

func sincronizarWorkspaceProyectoEnWorktree(proyecto *db.Proyecto, item *db.Worktree) error {
	if proyecto == nil || item == nil {
		return nil
	}
	// Las worktrees de agentes premium deben arrancar limpias y aisladas. La
	// sincronizacion del workspace sucio del repo principal solo se permite de
	// forma explicita para casos legacy o diagnostico.
	if !worktreeSyncDirtyWorkspaceEnabled() {
		return nil
	}
	repoPath := strings.TrimSpace(proyecto.RutaAbs)
	worktreePath := strings.TrimSpace(item.RutaAbs)
	if repoPath == "" || worktreePath == "" || repoPath == worktreePath || !proyectoPareceRepoGit(repoPath) {
		return nil
	}
	_, err := gitService.SyncDirtyWorkspaceToWorktree(repoPath, worktreePath)
	return err
}

func worktreeSyncDirtyWorkspaceEnabled() bool {
	v := strings.TrimSpace(os.Getenv("ORQUESTA_SYNC_DIRTY_WORKSPACE_TO_WORKTREE"))
	return strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}
