/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type GitMerge struct {
	ID            int64
	ProyectoID    int64
	ProyectoSlug  string
	SourceBranch  string
	TargetBranch  string
	RequestedBy   string
	SolicitadoPor string
	Estado        string
	SourceCommit  string
	MergeCommit   string
	CommitOrigen  string
	CommitMerge   string
	Notas         string
	MetadataJSON  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type GitMergeRequest = GitMerge

func GuardarGitMerge(m *GitMerge) (int64, error) {
	if m == nil {
		return 0, fmt.Errorf("merge orquestado obligatorio")
	}
	syncGitMergeAliases(m)
	m.SourceBranch = strings.TrimSpace(m.SourceBranch)
	m.TargetBranch = strings.TrimSpace(m.TargetBranch)
	m.RequestedBy = strings.TrimSpace(m.RequestedBy)
	m.Estado = normalizarEstadoGitMerge(m.Estado)
	m.CommitOrigen = strings.TrimSpace(m.CommitOrigen)
	m.CommitMerge = strings.TrimSpace(m.CommitMerge)
	m.Notas = strings.TrimSpace(m.Notas)
	if m.Estado == "" {
		m.Estado = "pendiente"
	}
	if m.MetadataJSON == "" {
		m.MetadataJSON = "{}"
	}
	if m.ProyectoID == 0 || m.SourceBranch == "" || m.TargetBranch == "" || m.RequestedBy == "" {
		return 0, fmt.Errorf("proyecto, ramas y requested_by son obligatorios")
	}
	if !gitMergeEstadoValido(m.Estado) {
		return 0, fmt.Errorf("estado de git_merge invalido: %s", m.Estado)
	}

	if m.ID == 0 {
		res, err := DB.Exec(`
			INSERT INTO git_merges (
				proyecto_id, source_branch, target_branch, requested_by, estado,
				commit_origen, commit_merge, notas, metadata_json
			) VALUES (?,?,?,?,?,?,?,?,?)`,
			m.ProyectoID, m.SourceBranch, m.TargetBranch, m.RequestedBy, m.Estado,
			m.CommitOrigen, m.CommitMerge, m.Notas, m.MetadataJSON,
		)
		if err != nil {
			return 0, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}
		Audit(m.RequestedBy, "guardar_git_merge", "git_merge", id, m.SourceBranch+"->"+m.TargetBranch)
		return id, nil
	}

	res, err := DB.Exec(`
		UPDATE git_merges
		SET proyecto_id = ?, source_branch = ?, target_branch = ?, requested_by = ?, estado = ?,
		    commit_origen = ?, commit_merge = ?, notas = ?, metadata_json = ?
		WHERE id = ?`,
		m.ProyectoID, m.SourceBranch, m.TargetBranch, m.RequestedBy, m.Estado,
		m.CommitOrigen, m.CommitMerge, m.Notas, m.MetadataJSON, m.ID,
	)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows == 0 {
		return 0, sql.ErrNoRows
	}
	Audit(m.RequestedBy, "guardar_git_merge", "git_merge", m.ID, m.SourceBranch+"->"+m.TargetBranch)
	return m.ID, nil
}

func ListarGitMerges(proyectoID *int64, estado string) ([]*GitMerge, error) {
	q := `
		SELECT gm.id, gm.proyecto_id, p.slug, gm.source_branch, gm.target_branch, gm.requested_by, gm.estado,
		       gm.commit_origen, gm.commit_merge, gm.notas, gm.metadata_json, gm.created_at, gm.updated_at
		FROM git_merges gm
		JOIN proyectos p ON p.id = gm.proyecto_id
		WHERE 1=1`
	args := []any{}
	if proyectoID != nil {
		q += ` AND gm.proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	if estado = normalizarEstadoGitMerge(estado); estado != "" {
		q += ` AND gm.estado = ?`
		args = append(args, estado)
	}
	q += ` ORDER BY gm.created_at DESC, gm.id DESC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*GitMerge
	for rows.Next() {
		item, err := scanGitMerge(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func GetGitMerge(id int64) (*GitMerge, error) {
	row := DB.QueryRow(`
		SELECT gm.id, gm.proyecto_id, p.slug, gm.source_branch, gm.target_branch, gm.requested_by, gm.estado,
		       gm.commit_origen, gm.commit_merge, gm.notas, gm.metadata_json, gm.created_at, gm.updated_at
		FROM git_merges gm
		JOIN proyectos p ON p.id = gm.proyecto_id
		WHERE gm.id = ?`, id)
	return scanGitMerge(row)
}

func gitMergeEstadoValido(v string) bool {
	switch v {
	case "pendiente", "validando", "aprobado", "rechazado", "ejecutando", "fusionado", "fallido", "cancelado":
		return true
	default:
		return false
	}
}

func normalizarEstadoGitMerge(v string) string {
	switch strings.TrimSpace(v) {
	case "", "pendiente", "validando", "aprobado", "rechazado", "ejecutando", "fusionado", "fallido", "cancelado":
		return strings.TrimSpace(v)
	case "propuesta":
		return "pendiente"
	case "aprobada":
		return "aprobado"
	case "ejecutada":
		return "fusionado"
	default:
		return strings.TrimSpace(v)
	}
}

func syncGitMergeAliases(m *GitMerge) {
	m.ProyectoSlug = strings.TrimSpace(m.ProyectoSlug)
	m.SolicitadoPor = strings.TrimSpace(m.SolicitadoPor)
	m.SourceCommit = strings.TrimSpace(m.SourceCommit)
	m.MergeCommit = strings.TrimSpace(m.MergeCommit)
	if m.RequestedBy == "" && m.SolicitadoPor != "" {
		m.RequestedBy = m.SolicitadoPor
	}
	if m.SolicitadoPor == "" && m.RequestedBy != "" {
		m.SolicitadoPor = m.RequestedBy
	}
	if m.CommitOrigen == "" && m.SourceCommit != "" {
		m.CommitOrigen = m.SourceCommit
	}
	if m.SourceCommit == "" && m.CommitOrigen != "" {
		m.SourceCommit = m.CommitOrigen
	}
	if m.CommitMerge == "" && m.MergeCommit != "" {
		m.CommitMerge = m.MergeCommit
	}
	if m.MergeCommit == "" && m.CommitMerge != "" {
		m.MergeCommit = m.CommitMerge
	}
}

func scanGitMerge(scanner interface{ Scan(...any) error }) (*GitMerge, error) {
	item := &GitMerge{}
	err := scanner.Scan(
		&item.ID,
		&item.ProyectoID,
		&item.ProyectoSlug,
		&item.SourceBranch,
		&item.TargetBranch,
		&item.RequestedBy,
		&item.Estado,
		&item.CommitOrigen,
		&item.CommitMerge,
		&item.Notas,
		&item.MetadataJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	syncGitMergeAliases(item)
	return item, nil
}

type GitGovRepository struct{}

func (GitGovRepository) ListWorktrees(estado, agente string) ([]*Worktree, error) {
	return ListarWorktrees(estado, agente)
}

func (GitGovRepository) ListLocks(estado, agente string) ([]*Lock, error) {
	return ListarLocks(estado, agente)
}

func (GitGovRepository) ListMerges(proyectoSlug, estado string) ([]*GitMergeRequest, error) {
	proyectoID, err := ResolveProyectoIDBySlug(proyectoSlug)
	if err != nil {
		return nil, err
	}
	items, err := ListarGitMerges(proyectoID, estado)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (GitGovRepository) SaveMerge(item *GitMergeRequest) (int64, error) {
	return GuardarGitMerge(item)
}

func (GitGovRepository) ResolveProjectID(slug string) (*int64, error) {
	return ResolveProyectoIDBySlug(slug)
}
