/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"strings"
	"time"
)

type Worktree struct {
	ID           int64
	ProyectoID   int64
	ProyectoSlug string
	TareaID      *int64
	LockID       *int64
	Agente       string
	Nombre       string
	RutaAbs      string
	Branch       string
	BaseRef      string
	Estado       string
	Motivo       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CerradaAt    *time.Time
}

type Lock struct {
	ID          int64
	ProyectoID  *int64
	TareaID     *int64
	SesionID    *int64
	Agente      string
	ScopeType   string
	ScopeKey    string
	RutaAbs     string
	Branch      string
	Motivo      string
	TokenLease  string
	Estado      string
	HeartbeatAt *time.Time
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LiberadaAt  *time.Time
}

func ListarWorktrees(estado, agente string) ([]*Worktree, error) {
	q := `SELECT w.id, w.proyecto_id, COALESCE(p.slug,''), w.tarea_id, w.lock_id, w.agente, w.nombre, w.ruta_abs, w.branch, w.base_ref, w.estado, w.motivo, w.created_at, w.updated_at, w.cerrada_at
		FROM worktrees w
		LEFT JOIN proyectos p ON p.id = w.proyecto_id
		WHERE 1=1`
	var args []any
	if estado = strings.TrimSpace(estado); estado != "" {
		q += ` AND w.estado = ?`
		args = append(args, estado)
	}
	if agente = strings.TrimSpace(agente); agente != "" {
		q += ` AND w.agente = ?`
		args = append(args, agente)
	}
	q += ` ORDER BY w.id`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Worktree
	for rows.Next() {
		item, err := scanWorktree(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func ListarLocks(estado, agente string) ([]*Lock, error) {
	q := `SELECT id, proyecto_id, tarea_id, sesion_id, agente, scope_type, scope_key, ruta_abs, branch, motivo, token_lease, estado, heartbeat_at, expires_at, created_at, updated_at, liberada_at
		FROM locks WHERE 1=1`
	var args []any
	if estado = strings.TrimSpace(estado); estado != "" {
		q += ` AND estado = ?`
		args = append(args, estado)
	}
	if agente = strings.TrimSpace(agente); agente != "" {
		q += ` AND agente = ?`
		args = append(args, agente)
	}
	q += ` ORDER BY id`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Lock
	for rows.Next() {
		item, err := scanLock(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func scanWorktree(s scanner) (*Worktree, error) {
	var item Worktree
	var tareaID, lockID sql.NullInt64
	var cerradaAt sql.NullTime
	if err := s.Scan(&item.ID, &item.ProyectoID, &item.ProyectoSlug, &tareaID, &lockID, &item.Agente, &item.Nombre, &item.RutaAbs, &item.Branch, &item.BaseRef, &item.Estado, &item.Motivo, &item.CreatedAt, &item.UpdatedAt, &cerradaAt); err != nil {
		return nil, err
	}
	if tareaID.Valid {
		item.TareaID = &tareaID.Int64
	}
	if lockID.Valid {
		item.LockID = &lockID.Int64
	}
	if cerradaAt.Valid {
		item.CerradaAt = &cerradaAt.Time
	}
	return &item, nil
}

func scanLock(s scanner) (*Lock, error) {
	var item Lock
	var proyectoID, tareaID, sesionID sql.NullInt64
	var heartbeatAt, liberadaAt sql.NullTime
	if err := s.Scan(&item.ID, &proyectoID, &tareaID, &sesionID, &item.Agente, &item.ScopeType, &item.ScopeKey, &item.RutaAbs, &item.Branch, &item.Motivo, &item.TokenLease, &item.Estado, &heartbeatAt, &item.ExpiresAt, &item.CreatedAt, &item.UpdatedAt, &liberadaAt); err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		item.ProyectoID = &proyectoID.Int64
	}
	if tareaID.Valid {
		item.TareaID = &tareaID.Int64
	}
	if sesionID.Valid {
		item.SesionID = &sesionID.Int64
	}
	if heartbeatAt.Valid {
		item.HeartbeatAt = &heartbeatAt.Time
	}
	if liberadaAt.Valid {
		item.LiberadaAt = &liberadaAt.Time
	}
	return &item, nil
}
