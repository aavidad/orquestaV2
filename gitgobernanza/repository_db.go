package gitgobernanza

import (
	"orquesta/coordinacion"
	"orquesta/db"
)

type Repository struct{}

func (Repository) ListWorktrees(estado, agente string) ([]*db.Worktree, error) {
	return db.ListarWorktrees(estado, agente)
}

func (Repository) ListLocks(estado, agente string) ([]*db.Lock, error) {
	return db.ListarLocks(estado, agente)
}

func (Repository) GetWorktree(id int64) (*coordinacion.Worktree, error) {
	return db.CoordinationWorktreeRepository().GetByID(id)
}

func (Repository) GetLock(id int64) (*coordinacion.Lock, error) {
	return db.CoordinationLockRepository().GetByID(id)
}

func (Repository) ResolveProyectoIDBySlug(slug string) (*int64, error) {
	return db.ResolveProyectoIDBySlug(slug)
}

func (Repository) GetGitMerge(id int64) (*GitMerge, error) {
	item, err := db.GetGitMerge(id)
	if err != nil || item == nil {
		return nil, err
	}
	return mapGitMerge(item), nil
}

func (Repository) SaveGitMerge(m *GitMerge) (int64, error) {
	return db.GuardarGitMerge(mapDBGitMerge(m))
}

func (Repository) ListGitMerges(proyectoID *int64, estado string) ([]*GitMerge, error) {
	items, err := db.ListarGitMerges(proyectoID, estado)
	if err != nil {
		return nil, err
	}
	out := make([]*GitMerge, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapGitMerge(item))
	}
	return out, nil
}

func (Repository) ListGitMergesLimitado(proyectoID *int64, estado string, limit int) ([]*GitMerge, error) {
	items, err := db.ListarGitMergesLimitado(proyectoID, estado, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*GitMerge, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapGitMerge(item))
	}
	return out, nil
}
