/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package coordinacion

import "time"

type LockState string

const (
	LockActive   LockState = "activa"
	LockReleased LockState = "liberada"
	LockExpired  LockState = "expirada"
	LockFailed   LockState = "fallida"
)

type Lock struct {
	ID          int64
	ProjectID   *int64
	TaskID      *int64
	SessionID   *int64
	Agent       string
	ScopeType   string
	ScopeKey    string
	Path        string
	Branch      string
	Reason      string
	LeaseToken  string
	State       LockState
	HeartbeatAt time.Time
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ReleasedAt  *time.Time
}

type WorktreeState string

const (
	WorktreeActive WorktreeState = "activa"
	WorktreeClosed WorktreeState = "cerrada"
	WorktreeFailed WorktreeState = "fallida"
)

type Worktree struct {
	ID        int64
	ProjectID int64
	TaskID    *int64
	LockID    *int64
	Agent     string
	Name      string
	Path      string
	Branch    string
	BaseRef   string
	State     WorktreeState
	Reason    string
	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  *time.Time
}

type Project struct {
	ID       int64
	Slug     string
	Name     string
	RootPath string
}

type Session struct {
	ID        int64
	Agent     string
	ProjectID *int64
	Branch    string
	CWD       string
}

type LockFilter struct {
	ProjectID *int64
	Agent     *string
	State     *LockState
}

type WorktreeFilter struct {
	ProjectID *int64
	Agent     *string
	State     *WorktreeState
}

type AcquireLockInput struct {
	ProjectID    *int64
	TaskID       *int64
	SessionID    *int64
	Agent        string
	ScopeType    string
	ScopeKey     string
	Path         string
	Branch       string
	Reason       string
	LeaseSeconds int
}

type RenewLockInput struct {
	ID           int64
	Agent        string
	LeaseToken   string
	LeaseSeconds int
}

type ReleaseLockInput struct {
	ID         int64
	Agent      string
	LeaseToken string
	Reason     string
}

type PrepareWorktreeInput struct {
	ProjectRef string
	Agent      string
	TaskID     *int64
	LockID     *int64
	Name       string
	Branch     string
	BaseRef    string
	Reason     string
}
