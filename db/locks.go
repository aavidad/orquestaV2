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

type CoordinationLockSQLRepository struct{}

type SQLiteLockRepository struct {
	CoordinationLockSQLRepository
}

func ListarLocksCoord(filter coordinacion.LockFilter) ([]*coordinacion.Lock, error) {
	return (CoordinationLockSQLRepository{}).List(filter)
}

func (CoordinationLockSQLRepository) ExpireActiveBefore(now time.Time) error {
	_, err := DB.Exec(`
		UPDATE locks
		SET estado='expirada'
		WHERE estado='activa' AND expires_at <= ?`, now)
	return err
}

func (CoordinationLockSQLRepository) FindActive(scopeType, scopeKey string) (*coordinacion.Lock, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto_id, tarea_id, sesion_id, agente, scope_type, scope_key, ruta_abs, branch,
		       motivo, token_lease, estado, heartbeat_at, expires_at, created_at, updated_at, liberada_at
		FROM locks
		WHERE scope_type = ? AND scope_key = ? AND estado = 'activa'
		ORDER BY id DESC LIMIT 1`, scopeType, scopeKey)
	lock, err := scanCoordinationLock(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return lock, err
}

func (CoordinationLockSQLRepository) Create(lock *coordinacion.Lock) (*coordinacion.Lock, error) {
	if lock == nil {
		return nil, fmt.Errorf("lock nil")
	}
	id, err := insertReturningID(`
		INSERT INTO locks (
			proyecto_id, tarea_id, sesion_id, agente, scope_type, scope_key, ruta_abs, branch,
			motivo, token_lease, estado, heartbeat_at, expires_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		lock.ProjectID, lock.TaskID, lock.SessionID, lock.Agent, lock.ScopeType, lock.ScopeKey, lock.Path,
		lock.Branch, lock.Reason, lock.LeaseToken, string(lock.State), lock.HeartbeatAt, lock.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return (CoordinationLockSQLRepository{}).GetByID(id)
}

func (CoordinationLockSQLRepository) GetByID(id int64) (*coordinacion.Lock, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto_id, tarea_id, sesion_id, agente, scope_type, scope_key, ruta_abs, branch,
		       motivo, token_lease, estado, heartbeat_at, expires_at, created_at, updated_at, liberada_at
		FROM locks
		WHERE id = ?`, id)
	return scanCoordinationLock(row)
}

func (CoordinationLockSQLRepository) Renew(id int64, expiresAt time.Time, agent, leaseToken string) (*coordinacion.Lock, error) {
	res, err := DB.Exec(`
		UPDATE locks
		SET heartbeat_at = CURRENT_TIMESTAMP, expires_at = ?
		WHERE id = ? AND agente = ? AND token_lease = ? AND estado = 'activa'`,
		expiresAt, id, agent, leaseToken,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("lock no renovable o sin permiso")
	}
	return (CoordinationLockSQLRepository{}).GetByID(id)
}

func (CoordinationLockSQLRepository) Release(id int64, releasedAt time.Time, agent, leaseToken string) (*coordinacion.Lock, error) {
	res, err := DB.Exec(`
		UPDATE locks
		SET estado = 'liberada', liberada_at = ?, heartbeat_at = CURRENT_TIMESTAMP
		WHERE id = ? AND agente = ? AND token_lease = ? AND estado = 'activa'`,
		releasedAt, id, agent, leaseToken,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("lock no liberable o sin permiso")
	}
	return (CoordinationLockSQLRepository{}).GetByID(id)
}

func (CoordinationLockSQLRepository) List(filter coordinacion.LockFilter) ([]*coordinacion.Lock, error) {
	q := `
		SELECT id, proyecto_id, tarea_id, sesion_id, agente, scope_type, scope_key, ruta_abs, branch,
		       motivo, token_lease, estado, heartbeat_at, expires_at, created_at, updated_at, liberada_at
		FROM locks
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
	var out []*coordinacion.Lock
	for rows.Next() {
		lock, err := scanCoordinationLock(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, lock)
	}
	return out, rows.Err()
}

func scanCoordinationLock(s scanner) (*coordinacion.Lock, error) {
	var lock coordinacion.Lock
	var projectID sql.NullInt64
	var taskID sql.NullInt64
	var sessionID sql.NullInt64
	var releasedAt sql.NullTime
	var state string
	err := s.Scan(
		&lock.ID, &projectID, &taskID, &sessionID, &lock.Agent, &lock.ScopeType, &lock.ScopeKey, &lock.Path,
		&lock.Branch, &lock.Reason, &lock.LeaseToken, &state, &lock.HeartbeatAt, &lock.ExpiresAt,
		&lock.CreatedAt, &lock.UpdatedAt, &releasedAt,
	)
	if err != nil {
		return nil, err
	}
	if projectID.Valid {
		lock.ProjectID = &projectID.Int64
	}
	if taskID.Valid {
		lock.TaskID = &taskID.Int64
	}
	if sessionID.Valid {
		lock.SessionID = &sessionID.Int64
	}
	if releasedAt.Valid {
		lock.ReleasedAt = &releasedAt.Time
	}
	lock.State = coordinacion.LockState(strings.TrimSpace(state))
	return &lock, nil
}
