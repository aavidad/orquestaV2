/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package gitgobernanza

import (
	"strings"

	"orquesta/coordinacion"
	"orquesta/db"
)

type Store interface {
	ListWorktrees(estado, agente string) ([]*db.Worktree, error)
	ListLocks(estado, agente string) ([]*db.Lock, error)
	GetWorktree(id int64) (*coordinacion.Worktree, error)
	GetLock(id int64) (*coordinacion.Lock, error)
	ResolveProyectoIDBySlug(slug string) (*int64, error)
	SaveGitMerge(m *db.GitMerge) (int64, error)
	ListGitMerges(proyectoID *int64, estado string) ([]*db.GitMerge, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type SaveMergeRequestInput struct {
	ID           int64
	ProyectoSlug string
	SourceBranch string
	TargetBranch string
	RequestedBy  string
	Estado       string
	CommitOrigen string
	CommitMerge  string
	Notas        string
	MetadataJSON string
}

func (s *Service) ListWorktrees(estado, agente string) ([]*db.Worktree, error) {
	return s.store.ListWorktrees(strings.TrimSpace(estado), strings.TrimSpace(agente))
}

func (s *Service) ListLocks(estado, agente string) ([]*db.Lock, error) {
	return s.store.ListLocks(strings.TrimSpace(estado), strings.TrimSpace(agente))
}

func (s *Service) GetWorktree(id int64) (*coordinacion.Worktree, error) {
	return s.store.GetWorktree(id)
}

func (s *Service) GetLock(id int64) (*coordinacion.Lock, error) {
	return s.store.GetLock(id)
}

func (s *Service) SaveRequest(input SaveMergeRequestInput) (int64, error) {
	proyectoID, err := s.store.ResolveProyectoIDBySlug(strings.TrimSpace(input.ProyectoSlug))
	if err != nil {
		return 0, err
	}
	return s.store.SaveGitMerge(&db.GitMerge{
		ID:           input.ID,
		ProyectoID:   derefProyectoID(proyectoID),
		SourceBranch: strings.TrimSpace(input.SourceBranch),
		TargetBranch: strings.TrimSpace(input.TargetBranch),
		RequestedBy:  strings.TrimSpace(input.RequestedBy),
		Estado:       strings.TrimSpace(input.Estado),
		CommitOrigen: strings.TrimSpace(input.CommitOrigen),
		CommitMerge:  strings.TrimSpace(input.CommitMerge),
		Notas:        strings.TrimSpace(input.Notas),
		MetadataJSON: strings.TrimSpace(input.MetadataJSON),
	})
}

func (s *Service) ListRequests(proyectoSlug, estado string) ([]*db.GitMerge, error) {
	var proyectoID *int64
	var err error
	if strings.TrimSpace(proyectoSlug) != "" {
		proyectoID, err = s.store.ResolveProyectoIDBySlug(strings.TrimSpace(proyectoSlug))
		if err != nil {
			return nil, err
		}
	}
	return s.store.ListGitMerges(proyectoID, strings.TrimSpace(estado))
}

func derefProyectoID(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

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

func (Repository) SaveGitMerge(m *db.GitMerge) (int64, error) {
	return db.GuardarGitMerge(m)
}

func (Repository) ListGitMerges(proyectoID *int64, estado string) ([]*db.GitMerge, error) {
	return db.ListarGitMerges(proyectoID, estado)
}
