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
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type RuntimeRepository struct{}

type RuntimeHandle struct {
	ID          int64
	Agente      string
	SesionID    *int64
	ProyectoID  *int64
	Transporte  string
	HandleKind  string
	HandleRef   string
	Estado      string
	MetadataJSON string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RuntimeOrder struct {
	ID         int64
	Agente     string
	ProyectoID *int64
	Tipo       string
	PayloadJSON string
	Estado     string
	ErrorText  string
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

func GuardarRuntimeHandle(h *RuntimeHandle) (int64, error) {
	if h == nil {
		return 0, fmt.Errorf("runtime handle obligatorio")
	}
	h.Agente = strings.TrimSpace(h.Agente)
	h.Transporte = strings.TrimSpace(h.Transporte)
	h.HandleKind = strings.TrimSpace(h.HandleKind)
	h.HandleRef = strings.TrimSpace(h.HandleRef)
	if h.Estado == "" {
		h.Estado = "activo"
	}
	if h.MetadataJSON == "" {
		h.MetadataJSON = "{}"
	}
	if h.Agente == "" || h.Transporte == "" || h.HandleKind == "" || h.HandleRef == "" {
		return 0, fmt.Errorf("agente, transporte, handle_kind y handle_ref son obligatorios")
	}
	if !runtimeEstadoValido(h.Estado) {
		return 0, fmt.Errorf("estado de runtime_handle invalido: %s", h.Estado)
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if h.Estado == "activo" || h.Estado == "pausado" {
		if _, err := tx.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=? AND estado IN ('activo','pausado')`, h.Agente); err != nil {
			return 0, err
		}
	}

	res, err := tx.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json
		) VALUES (?,?,?,?,?,?,?,?)`,
		h.Agente, h.SesionID, h.ProyectoID, h.Transporte, h.HandleKind, h.HandleRef, h.Estado, h.MetadataJSON,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	Audit(h.Agente, "guardar_runtime_handle", "runtime_handle", id, h.HandleKind+" "+h.HandleRef)
	return id, nil
}

func GetRuntimeHandleActivo(agente string) (*RuntimeHandle, error) {
	row := DB.QueryRow(`
		SELECT id, agente, sesion_id, proyecto_id, transporte, handle_kind, handle_ref,
		       estado, metadata_json, created_at, updated_at
		FROM runtime_handles
		WHERE agente = ? AND estado IN ('activo','pausado')
		ORDER BY id DESC
		LIMIT 1`, strings.TrimSpace(agente))
	return scanRuntimeHandle(row)
}

func ListarRuntimeHandles(agente, estado string) ([]*RuntimeHandle, error) {
	q := `
		SELECT id, agente, sesion_id, proyecto_id, transporte, handle_kind, handle_ref,
		       estado, metadata_json, created_at, updated_at
		FROM runtime_handles
		WHERE 1=1`
	args := []any{}
	if agente = strings.TrimSpace(agente); agente != "" {
		q += ` AND agente = ?`
		args = append(args, agente)
	}
	if estado = strings.TrimSpace(estado); estado != "" {
		q += ` AND estado = ?`
		args = append(args, estado)
	}
	q += ` ORDER BY id DESC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*RuntimeHandle
	for rows.Next() {
		item, err := scanRuntimeHandle(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func CrearRuntimeOrder(o *RuntimeOrder) (int64, error) {
	if o == nil {
		return 0, fmt.Errorf("runtime order obligatoria")
	}
	o.Agente = strings.TrimSpace(o.Agente)
	o.Tipo = strings.TrimSpace(o.Tipo)
	if o.PayloadJSON == "" {
		o.PayloadJSON = "{}"
	}
	if o.Estado == "" {
		o.Estado = "pendiente"
	}
	if o.Agente == "" || o.Tipo == "" {
		return 0, fmt.Errorf("agente y tipo son obligatorios")
	}
	if !runtimeOrderTipoValido(o.Tipo) {
		return 0, fmt.Errorf("tipo de runtime_order invalido: %s", o.Tipo)
	}
	if !runtimeOrderEstadoValido(o.Estado) {
		return 0, fmt.Errorf("estado de runtime_order invalido: %s", o.Estado)
	}
	res, err := DB.Exec(`
		INSERT INTO runtime_orders (agente, proyecto_id, tipo, payload_json, estado, error_text)
		VALUES (?,?,?,?,?,?)`,
		o.Agente, o.ProyectoID, o.Tipo, o.PayloadJSON, o.Estado, o.ErrorText,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	Audit(o.Agente, "crear_runtime_order", "runtime_order", id, o.Tipo)
	return id, nil
}

func ListarRuntimeOrders(agente, estado string) ([]*RuntimeOrder, error) {
	q := `
		SELECT id, agente, proyecto_id, tipo, payload_json, estado, error_text,
		       created_at, started_at, finished_at
		FROM runtime_orders
		WHERE 1=1`
	args := []any{}
	if agente = strings.TrimSpace(agente); agente != "" {
		q += ` AND agente = ?`
		args = append(args, agente)
	}
	if estado = strings.TrimSpace(estado); estado != "" {
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
		item, err := scanRuntimeOrder(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func EjecutarSiguienteRuntimeOrder(agente string) (*RuntimeOrder, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRow(`
		SELECT id, agente, proyecto_id, tipo, payload_json, estado, error_text,
		       created_at, started_at, finished_at
		FROM runtime_orders
		WHERE agente = ? AND estado = 'pendiente'
		ORDER BY id
		LIMIT 1`, strings.TrimSpace(agente))
	order, err := scanRuntimeOrder(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`
		UPDATE runtime_orders
		SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, error_text=''
		WHERE id = ?`, order.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	handle, err := GetRuntimeHandleActivo(order.Agente)
	if err != nil {
		err = fmt.Errorf("sin runtime_handle activo para %s: %w", order.Agente, err)
		_ = marcarRuntimeOrderFallida(order.ID, err.Error())
		return nil, err
	}
	if err := ejecutarRuntimeOrder(order, handle); err != nil {
		_ = marcarRuntimeOrderFallida(order.ID, err.Error())
		return nil, err
	}
	if err := marcarRuntimeOrderCompletada(order.ID); err != nil {
		return nil, err
	}
	order.Estado = "completada"
	now := time.Now()
	order.FinishedAt = &now
	return order, nil
}

func marcarRuntimeOrderCompletada(id int64) error {
	_, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado='completada', finished_at=CURRENT_TIMESTAMP, error_text=''
		WHERE id = ?`, id)
	return err
}

func marcarRuntimeOrderFallida(id int64, errText string) error {
	_, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado='fallida', finished_at=CURRENT_TIMESTAMP, error_text=?
		WHERE id = ?`, errText, id)
	return err
}

func ejecutarRuntimeOrder(order *RuntimeOrder, handle *RuntimeHandle) error {
	switch order.Tipo {
	case "enviar_instruccion":
		var payload struct {
			Mensaje string `json:"mensaje"`
		}
		if err := json.Unmarshal([]byte(order.PayloadJSON), &payload); err != nil {
			return fmt.Errorf("payload_json invalido: %w", err)
		}
		if strings.TrimSpace(payload.Mensaje) == "" {
			return fmt.Errorf("mensaje vacio")
		}
		writePath := runtimeWritePath(handle)
		if writePath == "" {
			return fmt.Errorf("runtime_handle sin write_path resoluble")
		}
		f, err := os.OpenFile(writePath, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			return fmt.Errorf("abriendo write_path %s: %w", writePath, err)
		}
		defer f.Close()
		if _, err := f.WriteString(payload.Mensaje + "\n"); err != nil {
			return fmt.Errorf("escribiendo instruccion: %w", err)
		}
		return nil
	case "pausar":
		pid, err := runtimePID(handle)
		if err != nil {
			return err
		}
		if err := syscall.Kill(pid, syscall.SIGSTOP); err != nil {
			return fmt.Errorf("pausando pid %d: %w", pid, err)
		}
		_, err = DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id = ?`, handle.ID)
		return err
	case "continuar":
		pid, err := runtimePID(handle)
		if err != nil {
			return err
		}
		if err := syscall.Kill(pid, syscall.SIGCONT); err != nil {
			return fmt.Errorf("continuando pid %d: %w", pid, err)
		}
		_, err = DB.Exec(`UPDATE runtime_handles SET estado='activo' WHERE id = ?`, handle.ID)
		return err
	case "handoff":
		return fmt.Errorf("handoff aun no implementado en runtime executor")
	default:
		return fmt.Errorf("tipo de order no soportado: %s", order.Tipo)
	}
}

func runtimeWritePath(handle *RuntimeHandle) string {
	meta := runtimeHandleMetadata(handle)
	if v, ok := meta["write_path"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if handle.HandleKind == "pty" {
		return handle.HandleRef
	}
	if strings.Contains(handle.HandleRef, "/") {
		return handle.HandleRef
	}
	return ""
}

func runtimePID(handle *RuntimeHandle) (int, error) {
	meta := runtimeHandleMetadata(handle)
	switch v := meta["pid"].(type) {
	case float64:
		if int(v) > 0 {
			return int(v), nil
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			return n, nil
		}
	}
	if n, err := strconv.Atoi(strings.TrimSpace(handle.HandleRef)); err == nil && n > 0 {
		return n, nil
	}
	return 0, fmt.Errorf("runtime_handle sin pid resoluble")
}

func runtimeHandleMetadata(handle *RuntimeHandle) map[string]any {
	if handle == nil || strings.TrimSpace(handle.MetadataJSON) == "" {
		return map[string]any{}
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(handle.MetadataJSON), &payload); err != nil {
		return map[string]any{}
	}
	return payload
}

func runtimeEstadoValido(v string) bool {
	switch v {
	case "activo", "pausado", "cerrado", "fallido":
		return true
	default:
		return false
	}
}

func runtimeOrderTipoValido(v string) bool {
	switch v {
	case "enviar_instruccion", "pausar", "continuar", "handoff":
		return true
	default:
		return false
	}
}

func runtimeOrderEstadoValido(v string) bool {
	switch v {
	case "pendiente", "ejecutando", "completada", "fallida":
		return true
	default:
		return false
	}
}

func ResolveProyectoIDBySlug(slug string) (*int64, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, nil
	}
	var id int64
	if err := DB.QueryRow(`SELECT id FROM proyectos WHERE slug = ?`, slug).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("proyecto '%s' no encontrado", slug)
		}
		return nil, err
	}
	return &id, nil
}

func (RuntimeRepository) ResolveProyectoIDBySlug(slug string) (*int64, error) {
	return ResolveProyectoIDBySlug(slug)
}

func (RuntimeRepository) SaveRuntimeHandle(h *RuntimeHandle) (int64, error) {
	return GuardarRuntimeHandle(h)
}

func (RuntimeRepository) ListRuntimeHandles(agente, estado string) ([]*RuntimeHandle, error) {
	return ListarRuntimeHandles(agente, estado)
}

func (RuntimeRepository) CreateRuntimeOrder(o *RuntimeOrder) (int64, error) {
	return CrearRuntimeOrder(o)
}

func (RuntimeRepository) ListRuntimeOrders(agente, estado string) ([]*RuntimeOrder, error) {
	return ListarRuntimeOrders(agente, estado)
}

func (RuntimeRepository) ExecuteNextRuntimeOrder(agente string) (*RuntimeOrder, error) {
	return EjecutarSiguienteRuntimeOrder(agente)
}

func scanRuntimeHandle(scanner interface{ Scan(...any) error }) (*RuntimeHandle, error) {
	item := &RuntimeHandle{}
	var sesionID, proyectoID sql.NullInt64
	err := scanner.Scan(
		&item.ID,
		&item.Agente,
		&sesionID,
		&proyectoID,
		&item.Transporte,
		&item.HandleKind,
		&item.HandleRef,
		&item.Estado,
		&item.MetadataJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if sesionID.Valid {
		item.SesionID = &sesionID.Int64
	}
	if proyectoID.Valid {
		item.ProyectoID = &proyectoID.Int64
	}
	return item, nil
}

func scanRuntimeOrder(scanner interface{ Scan(...any) error }) (*RuntimeOrder, error) {
	item := &RuntimeOrder{}
	var proyectoID sql.NullInt64
	var startedAt, finishedAt sql.NullTime
	err := scanner.Scan(
		&item.ID,
		&item.Agente,
		&proyectoID,
		&item.Tipo,
		&item.PayloadJSON,
		&item.Estado,
		&item.ErrorText,
		&item.CreatedAt,
		&startedAt,
		&finishedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		item.ProyectoID = &proyectoID.Int64
	}
	if startedAt.Valid {
		item.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		item.FinishedAt = &finishedAt.Time
	}
	return item, nil
}
