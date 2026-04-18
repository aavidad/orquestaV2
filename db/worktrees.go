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

	"orquesta/coordinacion"
)

type CoordinationWorktreeSQLRepository struct{}

type SQLiteWorktreeRepository struct {
	CoordinationWorktreeSQLRepository
}

func ListarWorktreesCoord(filter coordinacion.WorktreeFilter) ([]*coordinacion.Worktree, error) {
	return (CoordinationWorktreeSQLRepository{}).List(filter)
}

func ListarWorktreesCoordRaw(filter coordinacion.WorktreeFilter) ([]*coordinacion.Worktree, error) {
	return (CoordinationWorktreeSQLRepository{}).ListRaw(filter)
}

func (CoordinationWorktreeSQLRepository) Create(worktree *coordinacion.Worktree) (*coordinacion.Worktree, error) {
	if worktree == nil {
		return nil, fmt.Errorf("worktree nil")
	}
	worktree.Path = rutaWorktreeCanonicaProyecto(worktree.ProjectID, worktree.Path)
	res, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		worktree.ProjectID, worktree.TaskID, worktree.LockID, worktree.Agent, worktree.Name, worktree.Path,
		worktree.Branch, worktree.BaseRef, string(worktree.State), worktree.Reason,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return (CoordinationWorktreeSQLRepository{}).GetByID(id)
}

func (CoordinationWorktreeSQLRepository) GetByID(id int64) (*coordinacion.Worktree, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado,
		       motivo, created_at, updated_at, cerrada_at
		FROM worktrees
		WHERE id = ?`, id)
	worktree, err := scanCoordinationWorktree(row)
	if err != nil {
		return nil, err
	}
	return canonizarCoordinationWorktree(worktree), nil
}

func (CoordinationWorktreeSQLRepository) GetActiveByPath(path string) (*coordinacion.Worktree, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado,
		       motivo, created_at, updated_at, cerrada_at
		FROM worktrees
		WHERE ruta_abs = ? AND estado = 'activa'
		ORDER BY id DESC LIMIT 1`, path)
	worktree, err := scanCoordinationWorktree(row)
	if err == sql.ErrNoRows {
		estado := coordinacion.WorktreeActive
		items, listErr := (CoordinationWorktreeSQLRepository{}).ListRaw(coordinacion.WorktreeFilter{State: &estado})
		if listErr != nil {
			return nil, listErr
		}
		target := normalizarRutaProyecto(path)
		for _, item := range items {
			if item == nil {
				continue
			}
			if normalizarRutaProyecto(item.Path) == target {
				return item, nil
			}
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return canonizarCoordinationWorktree(worktree), nil
}

func getCoordinationWorktreeByPath(path string) (*coordinacion.Worktree, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado,
		       motivo, created_at, updated_at, cerrada_at
		FROM worktrees
		WHERE ruta_abs = ?
		ORDER BY id DESC LIMIT 1`, strings.TrimSpace(path))
	worktree, err := scanCoordinationWorktree(row)
	if err == sql.ErrNoRows {
		items, listErr := (CoordinationWorktreeSQLRepository{}).ListRaw(coordinacion.WorktreeFilter{})
		if listErr != nil {
			return nil, listErr
		}
		target := normalizarRutaProyecto(path)
		for _, item := range items {
			if item == nil {
				continue
			}
			if normalizarRutaProyecto(item.Path) == target {
				return item, nil
			}
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return canonizarCoordinationWorktree(worktree), nil
}

func (CoordinationWorktreeSQLRepository) List(filter coordinacion.WorktreeFilter) ([]*coordinacion.Worktree, error) {
	out, err := (CoordinationWorktreeSQLRepository{}).ListRaw(filter)
	if err != nil {
		return nil, err
	}
	filtradas := make([]*coordinacion.Worktree, 0, len(out))
	for _, worktree := range out {
		if worktree == nil {
			filtradas = append(filtradas, worktree)
			continue
		}
		if worktree.State == coordinacion.WorktreeActive && !coordinacion.WorktreePathUsable(worktree.Path) {
			continue
		}
		proyecto, getErr := GetProyectoConRutaEfectiva(jsonNumber(worktree.ProjectID), "")
		if getErr != nil || proyecto == nil {
			filtradas = append(filtradas, worktree)
			continue
		}
		if coordinacion.ShouldIncludeWorktreeForListing(worktree.State, worktree.Path, proyecto.RutaAbs) {
			filtradas = append(filtradas, worktree)
		}
	}
	return filtradas, nil
}

func (CoordinationWorktreeSQLRepository) ListRaw(filter coordinacion.WorktreeFilter) ([]*coordinacion.Worktree, error) {
	q := `
		SELECT id, proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado,
		       motivo, created_at, updated_at, cerrada_at
		FROM worktrees
		WHERE 1=1`
	var args []any
	if filter.ProjectID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *filter.ProjectID)
	}
	if filter.Agent != nil {
		q += ` AND agente = ?`
		args = append(args, *filter.Agent)
	}
	if filter.State != nil {
		q += ` AND estado = ?`
		args = append(args, string(*filter.State))
	}
	q += ` ORDER BY id DESC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	var out []*coordinacion.Worktree
	for rows.Next() {
		worktree, err := scanCoordinationWorktree(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, worktree)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return nil, rowsErr
	}
	for i, w := range out {
		out[i] = canonizarCoordinationWorktree(w)
	}
	return out, nil
}

func (CoordinationWorktreeSQLRepository) Close(id int64, closedAt time.Time, reason string) (*coordinacion.Worktree, error) {
	res, err := DB.Exec(`
		UPDATE worktrees
		SET estado='cerrada', motivo=?, cerrada_at=?
		WHERE id = ? AND estado = 'activa'`,
		reason, closedAt, id,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("worktree no cerrable")
	}
	return (CoordinationWorktreeSQLRepository{}).GetByID(id)
}

type CoordinationProjectSQLRepository struct{}

type SQLiteProjectRepository struct {
	CoordinationProjectSQLRepository
}

func (CoordinationProjectSQLRepository) GetByRef(ref string) (*coordinacion.Project, error) {
	project, err := GetProyecto(ref)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, sql.ErrNoRows
	}
	project = ProyectoConRutaEfectiva(project, "")
	return &coordinacion.Project{
		ID:       project.ID,
		Slug:     project.Slug,
		Name:     project.Nombre,
		RootPath: project.RutaAbs,
	}, nil
}

type CoordinationSessionSQLRepository struct{}

type SQLiteSessionRepository struct {
	CoordinationSessionSQLRepository
}

func (CoordinationSessionSQLRepository) GetActive(agent string, projectID *int64) (*coordinacion.Session, error) {
	session, err := GetSesionActiva(agent, projectID)
	if err != nil {
		return nil, err
	}
	return &coordinacion.Session{
		ID:        session.ID,
		Agent:     session.Agente,
		ProjectID: session.ProyectoID,
		Branch:    session.Branch,
		CWD:       session.CWD,
	}, nil
}

type CoordinationConfigSQLRepository struct{}

type SQLiteConfigRepository struct {
	CoordinationConfigSQLRepository
}

func (CoordinationConfigSQLRepository) Get(key string) (string, error) {
	return ConfigGet(key)
}

func scanCoordinationWorktree(s scanner) (*coordinacion.Worktree, error) {
	var worktree coordinacion.Worktree
	var taskID sql.NullInt64
	var lockID sql.NullInt64
	var state string
	var closedAt sql.NullTime
	err := s.Scan(
		&worktree.ID, &worktree.ProjectID, &taskID, &lockID, &worktree.Agent, &worktree.Name,
		&worktree.Path, &worktree.Branch, &worktree.BaseRef, &state, &worktree.Reason,
		&worktree.CreatedAt, &worktree.UpdatedAt, &closedAt,
	)
	if err != nil {
		return nil, err
	}
	if taskID.Valid {
		worktree.TaskID = &taskID.Int64
	}
	if lockID.Valid {
		worktree.LockID = &lockID.Int64
	}
	if closedAt.Valid {
		worktree.ClosedAt = &closedAt.Time
	}
	worktree.State = coordinacion.WorktreeState(strings.TrimSpace(state))
	return &worktree, nil
}

func canonizarCoordinationWorktree(worktree *coordinacion.Worktree) *coordinacion.Worktree {
	if worktree == nil {
		return nil
	}
	copia := *worktree
	copia.Path = rutaWorktreeCanonicaProyecto(copia.ProjectID, copia.Path)
	return &copia
}
