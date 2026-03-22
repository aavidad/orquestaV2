/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type RuntimeHandle struct {
	ID           int64
	Agente       string
	SesionID     *int64
	Transporte   string
	HandleKind   string
	HandleRef    string
	Estado       string
	MetadataJSON string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RuntimeOrder struct {
	ID          int64
	Agente      string
	SesionID    *int64
	Tipo        string
	PayloadJSON string
	Estado      string
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

type HandoffPayload struct {
	AgenteOrigen       string `json:"agente_origen"`
	AgenteDestino      string `json:"agente_destino"`
	TareaID            *int64 `json:"tarea_id,omitempty"`
	Motivo             string `json:"motivo,omitempty"`
	ResumenContinuidad string `json:"resumen_continuidad,omitempty"`
	ExternalSessionID  string `json:"external_session_id,omitempty"`
}

func RegistrarRuntimeHandle(h *RuntimeHandle) (int64, error) {
	if h == nil {
		return 0, fmt.Errorf("runtime handle nulo")
	}
	if h.Agente == "" || h.Transporte == "" || h.HandleKind == "" || h.HandleRef == "" {
		return 0, fmt.Errorf("agente, transporte, handle_kind y handle_ref son obligatorios")
	}
	if h.Estado == "" {
		h.Estado = "activo"
	}
	if h.MetadataJSON == "" {
		h.MetadataJSON = "{}"
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if h.Estado == "activo" {
		if _, err := tx.Exec(`UPDATE runtime_handles SET estado='reemplazado' WHERE agente=? AND estado='activo'`, h.Agente); err != nil {
			return 0, err
		}
	}
	res, err := tx.Exec(`
		INSERT INTO runtime_handles (agente, sesion_id, transporte, handle_kind, handle_ref, estado, metadata_json)
		VALUES (?,?,?,?,?,?,?)`,
		h.Agente, h.SesionID, h.Transporte, h.HandleKind, h.HandleRef, h.Estado, h.MetadataJSON,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	Audit(h.Agente, "registrar_runtime_handle", "runtime_handle", id, h.HandleRef)
	return id, nil
}

func RuntimeHandleActivo(agente string) (*RuntimeHandle, error) {
	row := DB.QueryRow(`
		SELECT id, agente, sesion_id, transporte, handle_kind, handle_ref, estado, metadata_json, created_at, updated_at
		FROM runtime_handles
		WHERE agente = ? AND estado = 'activo'
		ORDER BY id DESC
		LIMIT 1`, agente)
	return escanearRuntimeHandle(row)
}

func ListarRuntimeHandles() ([]*RuntimeHandle, error) {
	rows, err := DB.Query(`
		SELECT id, agente, sesion_id, transporte, handle_kind, handle_ref, estado, metadata_json, created_at, updated_at
		FROM runtime_handles
		ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*RuntimeHandle
	for rows.Next() {
		h, err := escanearRuntimeHandle(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

func CrearRuntimeOrder(o *RuntimeOrder) (int64, error) {
	if o == nil {
		return 0, fmt.Errorf("runtime order nula")
	}
	if o.Agente == "" || o.Tipo == "" {
		return 0, fmt.Errorf("agente y tipo son obligatorios")
	}
	if o.PayloadJSON == "" {
		o.PayloadJSON = "{}"
	}
	if o.Estado == "" {
		o.Estado = "pendiente"
	}

	res, err := DB.Exec(`
		INSERT INTO runtime_orders (agente, sesion_id, tipo, payload_json, estado, started_at, finished_at)
		VALUES (?,?,?,?,?,?,?)`,
		o.Agente, o.SesionID, o.Tipo, o.PayloadJSON, o.Estado, nullableTime(o.StartedAt), nullableTime(o.FinishedAt),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit(o.Agente, "crear_runtime_order", "runtime_order", id, o.Tipo)
	return id, nil
}

func ListarRuntimeOrders(agente, estado string) ([]*RuntimeOrder, error) {
	q := `
		SELECT id, agente, sesion_id, tipo, payload_json, estado, created_at, started_at, finished_at
		FROM runtime_orders
		WHERE 1=1`
	args := []any{}
	if agente != "" {
		q += ` AND agente = ?`
		args = append(args, agente)
	}
	if estado != "" {
		q += ` AND estado = ?`
		args = append(args, estado)
	}
	q += ` ORDER BY id DESC`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*RuntimeOrder
	for rows.Next() {
		o, err := escanearRuntimeOrder(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

func ActualizarRuntimeOrderEstado(id int64, estado string) error {
	var startedAt any
	var finishedAt any
	switch estado {
	case "en_progreso":
		startedAt = time.Now()
	case "completada", "fallida", "cancelada":
		finishedAt = time.Now()
	}
	res, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado = ?, started_at = COALESCE(?, started_at), finished_at = COALESCE(?, finished_at)
		WHERE id = ?`,
		estado, startedAt, finishedAt, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("runtime_order #%d no encontrada", id)
	}
	Audit("orquesta", "actualizar_runtime_order", "runtime_order", id, estado)
	return nil
}

func CrearHandoffAgenteVivo(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	if origen == "" || destino == "" {
		return 0, fmt.Errorf("agente origen y destino son obligatorios")
	}
	if origen == destino {
		return 0, fmt.Errorf("origen y destino deben ser distintos")
	}

	sesionOrigen, err := SesionActivaDeAgente(origen)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("el agente origen %s no tiene sesión activa", origen)
	}
	if err != nil {
		return 0, err
	}
	if _, err := RuntimeHandleActivo(origen); err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("el agente origen %s no tiene runtime handle activo", origen)
		}
		return 0, err
	}

	var sesionDestinoID *int64
	sesionDestino, err := SesionActivaDeAgente(destino)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if err == nil {
		sesionDestinoID = &sesionDestino.ID
	}

	if tareaID != nil {
		t, err := GetTarea(*tareaID)
		if err != nil {
			return 0, fmt.Errorf("tarea #%d no encontrada", *tareaID)
		}
		if t.Agente != nil && *t.Agente != origen {
			return 0, fmt.Errorf("la tarea #%d no pertenece a %s", *tareaID, origen)
		}
	}

	payload := HandoffPayload{
		AgenteOrigen:       origen,
		AgenteDestino:      destino,
		TareaID:            tareaID,
		Motivo:             motivo,
		ResumenContinuidad: resumenContinuidad,
		ExternalSessionID:  externalSessionID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if tareaID != nil {
		if _, err := tx.Exec(`UPDATE tareas SET agente=?, estado='asignada' WHERE id=?`, destino, *tareaID); err != nil {
			return 0, err
		}
		nota := fmt.Sprintf("handoff %s→%s", origen, destino)
		if motivo != "" {
			nota += ": " + motivo
		}
		if _, err := tx.Exec(
			`UPDATE tareas SET notas = notas || char(10) || ? || ' [' || datetime('now') || ' orquesta]' WHERE id=?`,
			nota, *tareaID,
		); err != nil {
			return 0, err
		}
	}

	res, err := tx.Exec(`
		INSERT INTO runtime_orders (agente, sesion_id, tipo, payload_json, estado)
		VALUES (?,?,?,?, 'pendiente')`,
		destino, nullableInt64(sesionDestinoID), "handoff", string(payloadJSON),
	)
	if err != nil {
		return 0, err
	}
	orderID, _ := res.LastInsertId()

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	detalle := fmt.Sprintf("%s→%s sesion_origen=%d", origen, destino, sesionOrigen.ID)
	if sesionDestinoID != nil {
		detalle += fmt.Sprintf(" sesion_destino=%d", *sesionDestinoID)
	}
	Audit(origen, "handoff_agente_vivo", "runtime_order", orderID, detalle)
	return orderID, nil
}

func escanearRuntimeHandle(s scanner) (*RuntimeHandle, error) {
	h := &RuntimeHandle{}
	var sesionID sql.NullInt64
	if err := s.Scan(&h.ID, &h.Agente, &sesionID, &h.Transporte, &h.HandleKind, &h.HandleRef, &h.Estado, &h.MetadataJSON, &h.CreatedAt, &h.UpdatedAt); err != nil {
		return nil, err
	}
	if sesionID.Valid {
		h.SesionID = &sesionID.Int64
	}
	return h, nil
}

func escanearRuntimeOrder(s scanner) (*RuntimeOrder, error) {
	o := &RuntimeOrder{}
	var sesionID sql.NullInt64
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	if err := s.Scan(&o.ID, &o.Agente, &sesionID, &o.Tipo, &o.PayloadJSON, &o.Estado, &o.CreatedAt, &startedAt, &finishedAt); err != nil {
		return nil, err
	}
	if sesionID.Valid {
		o.SesionID = &sesionID.Int64
	}
	if startedAt.Valid {
		o.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		o.FinishedAt = &finishedAt.Time
	}
	return o, nil
}
