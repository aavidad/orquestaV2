/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package coordination

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	Locks     LockRepository
	Worktrees WorktreeRepository
	Projects  ProjectRepository
	Sessions  SessionRepository
	Config    ConfigRepository
	Workspace WorkspaceManager
}

func (s *Service) AcquireLock(in AcquireLockInput) (*Lock, error) {
	if strings.TrimSpace(in.Agent) == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	if strings.TrimSpace(in.ScopeType) == "" || strings.TrimSpace(in.ScopeKey) == "" {
		return nil, fmt.Errorf("scope_type y scope_key obligatorios")
	}
	leaseSeconds := in.LeaseSeconds
	if leaseSeconds <= 0 {
		leaseSeconds = s.defaultLeaseSeconds()
	}
	now := time.Now().UTC()
	if err := s.Locks.ExpireActiveBefore(now); err != nil {
		return nil, err
	}
	activa, err := s.Locks.FindActive(in.ScopeType, in.ScopeKey)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if activa != nil {
		return nil, fmt.Errorf("el recurso ya está bloqueado por %s", activa.Agent)
	}
	token, err := nuevoLeaseToken()
	if err != nil {
		return nil, err
	}
	lock := &Lock{
		ProjectID:   in.ProjectID,
		TaskID:      in.TaskID,
		SessionID:   in.SessionID,
		Agent:       strings.TrimSpace(in.Agent),
		ScopeType:   strings.TrimSpace(in.ScopeType),
		ScopeKey:    strings.TrimSpace(in.ScopeKey),
		Path:        strings.TrimSpace(in.Path),
		Branch:      strings.TrimSpace(in.Branch),
		Reason:      strings.TrimSpace(in.Reason),
		LeaseToken:  token,
		State:       LockActive,
		HeartbeatAt: now,
		ExpiresAt:   now.Add(time.Duration(leaseSeconds) * time.Second),
	}
	return s.Locks.Create(lock)
}

func (s *Service) RenewLock(in RenewLockInput) (*Lock, error) {
	leaseSeconds := in.LeaseSeconds
	if leaseSeconds <= 0 {
		leaseSeconds = s.defaultLeaseSeconds()
	}
	expiresAt := time.Now().UTC().Add(time.Duration(leaseSeconds) * time.Second)
	return s.Locks.Renew(in.ID, expiresAt, strings.TrimSpace(in.Agent), strings.TrimSpace(in.LeaseToken))
}

func (s *Service) ReleaseLock(in ReleaseLockInput) (*Lock, error) {
	return s.Locks.Release(in.ID, time.Now().UTC(), strings.TrimSpace(in.Agent), strings.TrimSpace(in.LeaseToken))
}

func (s *Service) PrepareWorktree(in PrepareWorktreeInput) (*Worktree, error) {
	if s.Workspace == nil {
		return nil, fmt.Errorf("workspace manager no configurado")
	}
	project, err := s.Projects.GetByRef(in.ProjectRef)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Agent) == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	rootName := s.worktreeRootName()
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = buildWorktreeName(project.Slug, in.Agent, in.TaskID)
	}
	branch := strings.TrimSpace(in.Branch)
	if branch == "" {
		branch = buildBranchName(project.Slug, in.Agent, in.TaskID)
	}
	baseRef := strings.TrimSpace(in.BaseRef)
	if baseRef == "" {
		baseRef = "HEAD"
	}
	worktreeRoot := filepath.Join(project.RootPath, rootName)
	worktreePath := filepath.Join(worktreeRoot, name)
	if err := s.Workspace.CreateWorktree(project.RootPath, worktreePath, branch, baseRef); err != nil {
		return nil, err
	}
	worktree := &Worktree{
		ProjectID: project.ID,
		TaskID:    in.TaskID,
		LockID:    in.LockID,
		Agent:     strings.TrimSpace(in.Agent),
		Name:      name,
		Path:      worktreePath,
		Branch:    branch,
		BaseRef:   baseRef,
		State:     WorktreeActive,
		Reason:    strings.TrimSpace(in.Reason),
	}
	return s.Worktrees.Create(worktree)
}

func (s *Service) CloseWorktree(id int64, remove bool, reason string) (*Worktree, error) {
	worktree, err := s.Worktrees.GetByID(id)
	if err != nil {
		return nil, err
	}
	if remove {
		project, err := s.Projects.GetByRef(strconv.FormatInt(worktree.ProjectID, 10))
		if err != nil {
			return nil, err
		}
		if err := s.Workspace.RemoveWorktree(project.RootPath, worktree.Path); err != nil {
			return nil, err
		}
	}
	return s.Worktrees.Close(id, time.Now().UTC(), strings.TrimSpace(reason))
}

func (s *Service) defaultLeaseSeconds() int {
	if s.Config == nil {
		return 90
	}
	raw, err := s.Config.Get("lock_lease_seconds")
	if err != nil {
		return 90
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return 90
	}
	return n
}

func (s *Service) worktreeRootName() string {
	if s.Config == nil {
		return ".orquesta-worktrees"
	}
	raw, err := s.Config.Get("worktree_root_name")
	if err != nil || strings.TrimSpace(raw) == "" {
		return ".orquesta-worktrees"
	}
	return strings.TrimSpace(raw)
}

func nuevoLeaseToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func buildWorktreeName(projectSlug, agent string, taskID *int64) string {
	base := sanitizeName(projectSlug + "-" + agent)
	if taskID == nil {
		return base
	}
	return fmt.Sprintf("%s-t%d", base, *taskID)
}

func buildBranchName(projectSlug, agent string, taskID *int64) string {
	base := sanitizeName("orq/" + projectSlug + "/" + agent)
	if taskID == nil {
		return base
	}
	return fmt.Sprintf("%s/t%d", base, *taskID)
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-", ":", "-", "@", "-", "..", "-")
	s = replacer.Replace(s)
	s = strings.Trim(s, "-")
	if s == "" {
		return "work"
	}
	return s
}
