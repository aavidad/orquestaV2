/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package gitgobernanza

import (
	"strings"
	"time"

	"orquesta/coordinacion"
	"orquesta/db"
)

type GitMerge struct {
	ID            int64     `json:"id"`
	ProyectoID    int64     `json:"proyecto_id"`
	ProyectoSlug  string    `json:"proyecto_slug"`
	SourceBranch  string    `json:"source_branch"`
	TargetBranch  string    `json:"target_branch"`
	RequestedBy   string    `json:"requested_by"`
	SolicitadoPor string    `json:"solicitado_por"`
	Estado        string    `json:"estado"`
	SourceCommit  string    `json:"source_commit"`
	MergeCommit   string    `json:"merge_commit"`
	CommitOrigen  string    `json:"commit_origen"`
	CommitMerge   string    `json:"commit_merge"`
	Notas         string    `json:"notas"`
	MetadataJSON  string    `json:"metadata_json"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Store interface {
	ListWorktrees(estado, agente string) ([]*db.Worktree, error)
	ListLocks(estado, agente string) ([]*db.Lock, error)
	GetWorktree(id int64) (*coordinacion.Worktree, error)
	GetLock(id int64) (*coordinacion.Lock, error)
	ResolveProyectoIDBySlug(slug string) (*int64, error)
	GetGitMerge(id int64) (*GitMerge, error)
	SaveGitMerge(m *GitMerge) (int64, error)
	ListGitMerges(proyectoID *int64, estado string) ([]*GitMerge, error)
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

func (s *Service) GetRequest(id int64) (*GitMerge, error) {
	return s.store.GetGitMerge(id)
}

func (s *Service) SaveRequest(input SaveMergeRequestInput) (int64, error) {
	proyectoID, err := s.store.ResolveProyectoIDBySlug(strings.TrimSpace(input.ProyectoSlug))
	if err != nil {
		return 0, err
	}
	return s.store.SaveGitMerge(&GitMerge{
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

func (s *Service) ListRequests(proyectoSlug, estado string) ([]*GitMerge, error) {
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

func mapGitMerge(in *db.GitMerge) *GitMerge {
	if in == nil {
		return nil
	}
	return &GitMerge{
		ID:            in.ID,
		ProyectoID:    in.ProyectoID,
		ProyectoSlug:  in.ProyectoSlug,
		SourceBranch:  in.SourceBranch,
		TargetBranch:  in.TargetBranch,
		RequestedBy:   in.RequestedBy,
		SolicitadoPor: in.SolicitadoPor,
		Estado:        in.Estado,
		SourceCommit:  in.SourceCommit,
		MergeCommit:   in.MergeCommit,
		CommitOrigen:  in.CommitOrigen,
		CommitMerge:   in.CommitMerge,
		Notas:         in.Notas,
		MetadataJSON:  in.MetadataJSON,
		CreatedAt:     in.CreatedAt,
		UpdatedAt:     in.UpdatedAt,
	}
}

func mapDBGitMerge(in *GitMerge) *db.GitMerge {
	if in == nil {
		return nil
	}
	return &db.GitMerge{
		ID:            in.ID,
		ProyectoID:    in.ProyectoID,
		ProyectoSlug:  in.ProyectoSlug,
		SourceBranch:  in.SourceBranch,
		TargetBranch:  in.TargetBranch,
		RequestedBy:   in.RequestedBy,
		SolicitadoPor: in.SolicitadoPor,
		Estado:        in.Estado,
		SourceCommit:  in.SourceCommit,
		MergeCommit:   in.MergeCommit,
		CommitOrigen:  in.CommitOrigen,
		CommitMerge:   in.CommitMerge,
		Notas:         in.Notas,
		MetadataJSON:  in.MetadataJSON,
		CreatedAt:     in.CreatedAt,
		UpdatedAt:     in.UpdatedAt,
	}
}
