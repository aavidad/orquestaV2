/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package coordinacion

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
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

type projectPrepareLiteGetter interface {
	GetByRefPrepareLite(ref string) (*Project, error)
}

type projectPrepareLiteAgentGetter interface {
	GetByRefPrepareLiteForAgent(ref, agent string) (*Project, error)
}

const defaultWorktreeRootName = ".orquesta-worktrees"

var prepareWorktreeStoreTimeout = 750 * time.Millisecond

func (s *Service) ResolveProjectID(ref string) (*int64, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, nil
	}
	project, err := s.Projects.GetByRef(ref)
	if err != nil {
		return nil, err
	}
	return &project.ID, nil
}

func (s *Service) ResolveActiveSessionID(agent string, projectID *int64) (*int64, error) {
	if s.Sessions == nil || strings.TrimSpace(agent) == "" {
		return nil, nil
	}
	session, err := s.Sessions.GetActive(strings.TrimSpace(agent), projectID)
	if err != nil || session == nil {
		return nil, err
	}
	return &session.ID, nil
}

func (s *Service) ListLocks(filter LockFilter) ([]*Lock, error) {
	return s.Locks.List(filter)
}

func (s *Service) GetLock(id int64) (*Lock, error) {
	return s.Locks.GetByID(id)
}

func (s *Service) ListWorktrees(filter WorktreeFilter) ([]*Worktree, error) {
	return s.Worktrees.List(filter)
}

func (s *Service) GetWorktree(id int64) (*Worktree, error) {
	return s.Worktrees.GetByID(id)
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
	if strings.TrimSpace(in.Agent) == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	start := time.Now()
	worktreeDebugf("PrepareWorktree start project_ref=%s agent=%s task_id=%v", strings.TrimSpace(in.ProjectRef), strings.TrimSpace(in.Agent), in.TaskID)
	var (
		project *Project
		err     error
	)
	if lite, ok := s.Projects.(projectPrepareLiteAgentGetter); ok {
		project, err = lite.GetByRefPrepareLiteForAgent(in.ProjectRef, in.Agent)
	} else if lite, ok := s.Projects.(projectPrepareLiteGetter); ok {
		project, err = lite.GetByRefPrepareLite(in.ProjectRef)
	} else {
		project, err = s.Projects.GetByRef(in.ProjectRef)
	}
	if err != nil {
		return nil, err
	}
	worktreeDebugf("PrepareWorktree step=get_project duration=%s project_id=%d root=%s", time.Since(start).Round(time.Millisecond), project.ID, strings.TrimSpace(project.RootPath))
	rootName := defaultWorktreeRootName
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
	stepStart := time.Now()
	if err := s.Workspace.CreateWorktree(project.RootPath, worktreePath, branch, baseRef); err != nil {
		if shouldReusePhysicalWorktree(err, worktreePath) {
			worktreeDebugf("PrepareWorktree step=reuse_existing_physical duration=%s path=%s branch=%s", time.Since(stepStart).Round(time.Millisecond), worktreePath, branch)
		} else {
			return nil, err
		}
	}
	worktreeDebugf("PrepareWorktree step=create_worktree duration=%s path=%s branch=%s", time.Since(stepStart).Round(time.Millisecond), worktreePath, branch)
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
	stepStart = time.Now()
	created, err := s.createWorktreeRecordWithTimeout(worktree)
	worktreeDebugf("PrepareWorktree step=store_create duration=%s err=%v synthetic=%t total=%s", time.Since(stepStart).Round(time.Millisecond), err, created != nil && created.ID <= 0, time.Since(start).Round(time.Millisecond))
	return created, err
}

func shouldReusePhysicalWorktree(err error, worktreePath string) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if !strings.Contains(msg, "already used by worktree") {
		return false
	}
	info, statErr := os.Stat(strings.TrimSpace(worktreePath))
	return statErr == nil && info.IsDir()
}

func (s *Service) createWorktreeRecordWithTimeout(worktree *Worktree) (*Worktree, error) {
	if worktree == nil {
		return nil, fmt.Errorf("worktree nil")
	}
	type result struct {
		worktree *Worktree
		err      error
	}
	done := make(chan result, 1)
	go func() {
		created, err := s.Worktrees.Create(worktree)
		done <- result{worktree: created, err: err}
	}()
	select {
	case res := <-done:
		return res.worktree, res.err
	case <-time.After(prepareWorktreeStoreTimeout):
		if !WorktreePathUsable(worktree.Path) {
			return nil, fmt.Errorf("timeout persistiendo worktree tras %s", prepareWorktreeStoreTimeout)
		}
		now := time.Now().UTC()
		copia := *worktree
		copia.CreatedAt = now
		copia.UpdatedAt = now
		worktreeDebugf("PrepareWorktree step=store_create_timeout path=%s timeout=%s", strings.TrimSpace(worktree.Path), prepareWorktreeStoreTimeout)
		return &copia, nil
	}
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
		return defaultWorktreeRootName
	}
	raw, err := s.Config.Get("worktree_root_name")
	if err != nil || strings.TrimSpace(raw) == "" {
		return defaultWorktreeRootName
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

func buildBranchBase(projectSlug, agent string) string {
	return sanitizeName("orq/" + projectSlug + "/" + agent)
}

func buildBranchName(projectSlug, agent string, taskID *int64) string {
	base := buildBranchBase(projectSlug, agent)
	if taskID == nil {
		return base
	}
	return fmt.Sprintf("%s-t%d", base, *taskID)
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

func worktreeDebugf(format string, args ...any) {
	if !worktreeDebugEnabled() {
		return
	}
	log.Printf("orquesta[prepare-worktree] "+format, args...)
}

func worktreeDebugEnabled() bool {
	for _, key := range []string{"ORQUESTA_DEBUG_PREPARE", "ORQUESTA_DEBUG"} {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
		switch value {
		case "1", "true", "yes", "on", "si", "sí":
			return true
		}
	}
	return false
}
