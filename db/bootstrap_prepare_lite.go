package db

import (
	"database/sql"
	"strings"
)

func PeekNextBootstrapRuntimeOrderPrepareLite(agente string, proyectoID *int64) (*RuntimeOrder, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	if order, ok, err := peekNextBootstrapRuntimeOrderPrepareLiteReadOnly(agente, proyectoID); ok {
		return order, err
	}
	return PeekNextBootstrapRuntimeOrder(agente, proyectoID)
}

func peekNextBootstrapRuntimeOrderPrepareLiteReadOnly(agente string, proyectoID *int64) (*RuntimeOrder, bool, error) {
	raw, ok, err := openPrepareLiteReadOnlyLocal()
	if !ok || err != nil {
		return nil, ok, err
	}
	defer raw.Close()

	query := runtimeOrderSelectBase() + `
		WHERE agente = ? AND estado = 'pendiente'
		  AND tipo IN ('handoff','resume','start')
		  AND available_at <= CURRENT_TIMESTAMP`
	args := []any{agente}
	if proyectoID != nil && *proyectoID > 0 {
		query += ` AND (proyecto_id = ? OR proyecto_id IS NULL)
			ORDER BY CASE WHEN proyecto_id = ? THEN 0 ELSE 1 END,
			         CASE tipo WHEN 'handoff' THEN 0 WHEN 'resume' THEN 1 WHEN 'start' THEN 2 ELSE 9 END,
			         id
			LIMIT 1`
		args = append(args, *proyectoID, *proyectoID)
	} else {
		query += ` ORDER BY CASE WHEN proyecto_id IS NULL THEN 0 ELSE 1 END,
		                  CASE tipo WHEN 'handoff' THEN 0 WHEN 'resume' THEN 1 WHEN 'start' THEN 2 ELSE 9 END,
		                  id
		           LIMIT 1`
	}
	order, err := scanRuntimeOrder(raw.QueryRow(query, args...))
	if err == sql.ErrNoRows {
		return nil, true, nil
	}
	return order, true, err
}

func ListarRuntimeMailboxPrepareLite(filter FiltroRuntimeMailbox) ([]*RuntimeMailboxMessage, error) {
	if mailbox, ok, err := listarRuntimeMailboxPrepareLiteReadOnly(filter); ok {
		return mailbox, err
	}
	return ListarRuntimeMailbox(filter)
}

func listarRuntimeMailboxPrepareLiteReadOnly(filter FiltroRuntimeMailbox) ([]*RuntimeMailboxMessage, bool, error) {
	raw, ok, err := openPrepareLiteReadOnlyLocal()
	if !ok || err != nil {
		return nil, ok, err
	}
	defer raw.Close()
	q := runtimeMailboxSelectBase() + ` WHERE 1=1`
	var args []any
	if filter.ToAgente != nil {
		agenteCanonico, err := CanonicalizeAgentName(*filter.ToAgente)
		if err != nil {
			return nil, true, err
		}
		q += ` AND to_agente = ?`
		args = append(args, agenteCanonico)
	}
	if filter.FromAgente != nil {
		agenteCanonico, err := CanonicalizeAgentName(*filter.FromAgente)
		if err != nil {
			return nil, true, err
		}
		q += ` AND from_agente = ?`
		args = append(args, agenteCanonico)
	}
	if filter.ProyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *filter.ProyectoID)
	}
	if filter.Estado != nil {
		q += ` AND estado = ?`
		args = append(args, strings.TrimSpace(*filter.Estado))
	}
	q += ` ORDER BY id DESC`
	if filter.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	rows, err := raw.Query(q, args...)
	if err != nil {
		return nil, true, err
	}
	defer rows.Close()
	var out []*RuntimeMailboxMessage
	for rows.Next() {
		msg, err := scanRuntimeMailbox(rows)
		if err != nil {
			return nil, true, err
		}
		out = append(out, msg)
	}
	return out, true, rows.Err()
}

func UltimoRuntimeCheckpointPrepareLite(agente string, proyectoID *int64) (*RuntimeCheckpoint, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	if cp, ok, err := ultimoRuntimeCheckpointPrepareLiteReadOnly(agente, proyectoID); ok {
		return cp, err
	}
	return UltimoRuntimeCheckpoint(agente, proyectoID)
}

func ultimoRuntimeCheckpointPrepareLiteReadOnly(agente string, proyectoID *int64) (*RuntimeCheckpoint, bool, error) {
	raw, ok, err := openPrepareLiteReadOnlyLocal()
	if !ok || err != nil {
		return nil, ok, err
	}
	defer raw.Close()
	q := runtimeCheckpointSelectBase() + ` WHERE agente = ?`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY id DESC LIMIT 1`
	cp, err := scanRuntimeCheckpoint(raw.QueryRow(q, args...))
	if err == sql.ErrNoRows {
		return nil, true, nil
	}
	return cp, true, err
}

func ListarBootstrapRuntimeOrdersMailboxPrepareLite(agente string, proyectoID *int64) ([]*RuntimeOrder, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	if orders, ok, err := listarBootstrapRuntimeOrdersMailboxPrepareLiteReadOnly(agente, proyectoID); ok {
		return orders, err
	}
	estado := "pendiente"
	return ListarRuntimeOrders(FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
		Tipos:      []string{"nudge", "discordia"},
	})
}

func listarBootstrapRuntimeOrdersMailboxPrepareLiteReadOnly(agente string, proyectoID *int64) ([]*RuntimeOrder, bool, error) {
	raw, ok, err := openPrepareLiteReadOnlyLocal()
	if !ok || err != nil {
		return nil, ok, err
	}
	defer raw.Close()
	q := runtimeOrderSelectBase() + `
		WHERE agente = ? AND estado = 'pendiente'
		  AND tipo IN ('nudge','discordia')
		  AND available_at <= CURRENT_TIMESTAMP`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY id DESC`
	rows, err := raw.Query(q, args...)
	if err != nil {
		return nil, true, err
	}
	defer rows.Close()
	var out []*RuntimeOrder
	for rows.Next() {
		order, err := scanRuntimeOrder(rows)
		if err != nil {
			return nil, true, err
		}
		out = append(out, order)
	}
	return out, true, rows.Err()
}

func GetRuntimeMailboxByRuntimeOrderIDPrepareLite(runtimeOrderID int64) (*RuntimeMailboxMessage, error) {
	if runtimeOrderID <= 0 {
		return nil, nil
	}
	if msg, ok, err := getRuntimeMailboxByRuntimeOrderIDPrepareLiteReadOnly(runtimeOrderID); ok {
		return msg, err
	}
	return GetRuntimeMailboxByRuntimeOrderID(runtimeOrderID)
}

func getRuntimeMailboxByRuntimeOrderIDPrepareLiteReadOnly(runtimeOrderID int64) (*RuntimeMailboxMessage, bool, error) {
	raw, ok, err := openPrepareLiteReadOnlyLocal()
	if !ok || err != nil {
		return nil, ok, err
	}
	defer raw.Close()
	msg, err := scanRuntimeMailbox(raw.QueryRow(runtimeMailboxSelectBase()+`
		WHERE runtime_order_id = ?
		ORDER BY id DESC
		LIMIT 1`, runtimeOrderID))
	if err == sql.ErrNoRows {
		return nil, true, nil
	}
	return msg, true, err
}

func openPrepareLiteReadOnlyLocal() (*sql.DB, bool, error) {
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
	return raw, true, nil
}

func supportsPrepareLiteReadOnlyBackend(driver string) bool {
	return usesLegacyLocalDriver(driver)
}
