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
	"path/filepath"
	"strings"
	"time"

	"orquesta/coordinacion"
)

type CoordinationWorktreeSQLRepository struct{}

type CoordinationWorktreeDBRepository struct {
	CoordinationWorktreeSQLRepository
}

func ListarWorktreesCoord(filter coordinacion.WorktreeFilter) ([]*coordinacion.Worktree, error) {
	return (CoordinationWorktreeSQLRepository{}).List(filter)
}

func ListarWorktreesCoordRaw(filter coordinacion.WorktreeFilter) ([]*coordinacion.Worktree, error) {
	return (CoordinationWorktreeSQLRepository{}).ListRaw(filter)
}

func ListarWorktreesCoordPrepareLite(filter coordinacion.WorktreeFilter) ([]*coordinacion.Worktree, error) {
	if items, ok, err := listarWorktreesCoordPrepareLiteReadOnly(filter); ok {
		return items, err
	}
	return ListarWorktreesCoordRaw(filter)
}

func listarWorktreesCoordPrepareLiteReadOnly(filter coordinacion.WorktreeFilter) ([]*coordinacion.Worktree, bool, error) {
	backend, cfg, err := resolveOpenConfig()
	if err != nil {
		return nil, false, err
	}
	if !supportsPrepareLiteReadOnlyBackend(cfg.Driver) {
		return nil, false, nil
	}
	disabled := false
	skipPost := true
	cfg = applyOpenOptions(cfg, OpenOptions{
		BootstrapSchema:    &disabled,
		SkipPostMigrations: &skipPost,
		ReadOnly:           true,
	})
	cfg.MaxOpenConns = 1
	raw, err := backend.Open(cfg)
	if err != nil {
		return nil, true, err
	}
	defer raw.Close()

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
		agenteCanonico, err := CanonicalizeAgentName(*filter.Agent)
		if err != nil {
			return nil, true, err
		}
		q += ` AND agente = ?`
		args = append(args, agenteCanonico)
	}
	if filter.State != nil {
		q += ` AND estado = ?`
		args = append(args, string(*filter.State))
	}
	q += ` ORDER BY id DESC`
	rows, err := raw.Query(q, args...)
	if err != nil {
		return nil, true, err
	}
	defer rows.Close()

	var out []*coordinacion.Worktree
	for rows.Next() {
		worktree, err := scanCoordinationWorktree(rows)
		if err != nil {
			return nil, true, err
		}
		out = append(out, canonizarCoordinationWorktreePrepareLiteRaw(raw, worktree))
	}
	return out, true, rows.Err()
}

func (CoordinationWorktreeSQLRepository) Create(worktree *coordinacion.Worktree) (*coordinacion.Worktree, error) {
	if worktree == nil {
		return nil, fmt.Errorf("worktree nil")
	}
	agenteCanonico, err := CanonicalizeAgentName(worktree.Agent)
	if err != nil {
		return nil, err
	}
	worktree.Agent = agenteCanonico
	worktree.Path = rutaWorktreeCanonicaProyecto(worktree.ProjectID, worktree.Path)
	if created, ok, err := createCoordinationWorktreePrepareFast(worktree); ok {
		return created, err
	}
	id, err := insertReturningID(`
		INSERT INTO worktrees (proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		worktree.ProjectID, worktree.TaskID, worktree.LockID, worktree.Agent, worktree.Name, worktree.Path,
		worktree.Branch, worktree.BaseRef, string(worktree.State), worktree.Reason,
	)
	if err != nil {
		return nil, err
	}
	return (CoordinationWorktreeSQLRepository{}).GetByID(id)
}

func createCoordinationWorktreePrepareFast(worktree *coordinacion.Worktree) (*coordinacion.Worktree, bool, error) {
	backend, cfg, err := resolveOpenConfig()
	if err != nil {
		return nil, false, err
	}
	if !supportsPrepareLiteReadOnlyBackend(cfg.Driver) {
		return nil, false, nil
	}
	disabled := false
	skipPost := true
	cfg = applyOpenOptions(cfg, OpenOptions{
		BootstrapSchema:    &disabled,
		SkipPostMigrations: &skipPost,
	})
	cfg.MaxOpenConns = 1
	raw, err := backend.Open(cfg)
	if err != nil {
		return nil, true, err
	}
	defer raw.Close()

	id, err := insertReturningIDWith(raw, `
		INSERT INTO worktrees (proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		worktree.ProjectID, worktree.TaskID, worktree.LockID, worktree.Agent, worktree.Name, worktree.Path,
		worktree.Branch, worktree.BaseRef, string(worktree.State), worktree.Reason,
	)
	if err != nil {
		return nil, true, err
	}
	now := time.Now().UTC()
	copia := *worktree
	copia.ID = id
	copia.CreatedAt = now
	copia.UpdatedAt = now
	return &copia, true, nil
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
		agenteCanonico, err := CanonicalizeAgentName(*filter.Agent)
		if err != nil {
			return nil, err
		}
		q += ` AND agente = ?`
		args = append(args, agenteCanonico)
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

type CoordinationProjectDBRepository struct {
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

func (CoordinationProjectSQLRepository) GetByRefPrepareLite(ref string) (*coordinacion.Project, error) {
	project, err := GetProyectoPrepareLite(ref)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, sql.ErrNoRows
	}
	project = ProyectoPrepareLiteConRutaEfectiva(project, "")
	return &coordinacion.Project{
		ID:       project.ID,
		Slug:     project.Slug,
		Name:     project.Nombre,
		RootPath: project.RutaAbs,
	}, nil
}

func (CoordinationProjectSQLRepository) GetByRefPrepareLiteForAgent(ref, agent string) (*coordinacion.Project, error) {
	project, err := GetProyectoPrepareLite(ref)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, sql.ErrNoRows
	}
	project = ProyectoPrepareLiteConRutaEfectiva(project, agent)
	return &coordinacion.Project{
		ID:       project.ID,
		Slug:     project.Slug,
		Name:     project.Nombre,
		RootPath: project.RutaAbs,
	}, nil
}

type CoordinationSessionSQLRepository struct{}

type CoordinationSessionDBRepository struct {
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

type CoordinationConfigDBRepository struct {
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

func canonizarCoordinationWorktreePrepareLiteRaw(raw *sql.DB, worktree *coordinacion.Worktree) *coordinacion.Worktree {
	if worktree == nil {
		return nil
	}
	copia := *worktree
	ruta := strings.TrimSpace(copia.Path)
	if filepath.IsAbs(ruta) {
		copia.Path = filepath.Clean(ruta)
		return &copia
	}
	copia.Path = rutaProyectoEscopadaCanonica(rutaProyectoPrepareLiteReadOnly(raw, copia.ProjectID), ruta)
	return &copia
}

func rutaProyectoPrepareLiteReadOnly(raw *sql.DB, proyectoID int64) string {
	if raw == nil || proyectoID <= 0 {
		return ""
	}
	var ruta string
	if err := raw.QueryRow(`SELECT ruta_abs FROM proyectos WHERE id = ?`, proyectoID).Scan(&ruta); err != nil {
		return ""
	}
	return normalizarRutaProyecto(ruta)
}
