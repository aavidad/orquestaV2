package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"orquesta/coordination"
)

type SQLiteWorktreeRepository struct{}

func (SQLiteWorktreeRepository) Create(worktree *coordination.Worktree) (*coordination.Worktree, error) {
	if worktree == nil {
		return nil, fmt.Errorf("worktree nil")
	}
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
	return (SQLiteWorktreeRepository{}).GetByID(id)
}

func (SQLiteWorktreeRepository) GetByID(id int64) (*coordination.Worktree, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado,
		       motivo, created_at, updated_at, cerrada_at
		FROM worktrees
		WHERE id = ?`, id)
	return scanCoordinationWorktree(row)
}

func (SQLiteWorktreeRepository) GetActiveByPath(path string) (*coordination.Worktree, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado,
		       motivo, created_at, updated_at, cerrada_at
		FROM worktrees
		WHERE ruta_abs = ? AND estado = 'activa'
		ORDER BY id DESC LIMIT 1`, path)
	worktree, err := scanCoordinationWorktree(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return worktree, err
}

func (SQLiteWorktreeRepository) List(filter coordination.WorktreeFilter) ([]*coordination.Worktree, error) {
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
	defer rows.Close()
	var out []*coordination.Worktree
	for rows.Next() {
		worktree, err := scanCoordinationWorktree(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, worktree)
	}
	return out, rows.Err()
}

func (SQLiteWorktreeRepository) Close(id int64, closedAt time.Time, reason string) (*coordination.Worktree, error) {
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
	return (SQLiteWorktreeRepository{}).GetByID(id)
}

type SQLiteProjectRepository struct{}

func (SQLiteProjectRepository) GetByRef(ref string) (*coordination.Project, error) {
	project, err := GetProyecto(ref)
	if err != nil {
		return nil, err
	}
	return &coordination.Project{
		ID:       project.ID,
		Slug:     project.Slug,
		Name:     project.Nombre,
		RootPath: project.RutaAbs,
	}, nil
}

type SQLiteSessionRepository struct{}

func (SQLiteSessionRepository) GetActive(agent string, projectID *int64) (*coordination.Session, error) {
	session, err := GetSesionActiva(agent, projectID)
	if err != nil {
		return nil, err
	}
	return &coordination.Session{
		ID:        session.ID,
		Agent:     session.Agente,
		ProjectID: session.ProyectoID,
		Branch:    session.Branch,
		CWD:       session.CWD,
	}, nil
}

type SQLiteConfigRepository struct{}

func (SQLiteConfigRepository) Get(key string) (string, error) {
	return ConfigGet(key)
}

func scanCoordinationWorktree(s scanner) (*coordination.Worktree, error) {
	var worktree coordination.Worktree
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
	worktree.State = coordination.WorktreeState(strings.TrimSpace(state))
	return &worktree, nil
}
