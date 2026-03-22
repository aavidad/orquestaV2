/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package coordination

import "time"

type LockRepository interface {
	ExpireActiveBefore(now time.Time) error
	FindActive(scopeType, scopeKey string) (*Lock, error)
	Create(lock *Lock) (*Lock, error)
	GetByID(id int64) (*Lock, error)
	Renew(id int64, expiresAt time.Time, agent, leaseToken string) (*Lock, error)
	Release(id int64, releasedAt time.Time, agent, leaseToken string) (*Lock, error)
	List(filter LockFilter) ([]*Lock, error)
}

type WorktreeRepository interface {
	Create(worktree *Worktree) (*Worktree, error)
	GetByID(id int64) (*Worktree, error)
	GetActiveByPath(path string) (*Worktree, error)
	List(filter WorktreeFilter) ([]*Worktree, error)
	Close(id int64, closedAt time.Time, reason string) (*Worktree, error)
}

type ProjectRepository interface {
	GetByRef(ref string) (*Project, error)
}

type SessionRepository interface {
	GetActive(agent string, projectID *int64) (*Session, error)
}

type ConfigRepository interface {
	Get(key string) (string, error)
}

type WorkspaceManager interface {
	CreateWorktree(repoPath, worktreePath, branch, baseRef string) error
	RemoveWorktree(repoPath, worktreePath string) error
}
