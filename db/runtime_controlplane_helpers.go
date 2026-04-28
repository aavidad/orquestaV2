package db

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func runtimeOrderPlaceholders(tipos []string) (string, []any) {
	placeholders := make([]string, 0, len(tipos))
	args := make([]any, 0, len(tipos))
	for _, tipo := range tipos {
		tipo = strings.TrimSpace(tipo)
		if tipo == "" {
			continue
		}
		placeholders = append(placeholders, "?")
		args = append(args, tipo)
	}
	if len(placeholders) == 0 {
		return "''", args
	}
	return strings.Join(placeholders, ","), args
}

func scanRuntimeHandle(s scanner) (*RuntimeHandle, error) {
	var h RuntimeHandle
	var proyectoID sql.NullInt64
	var sesionID sql.NullInt64
	var runtimeID sql.NullInt64
	var lastSeen sql.NullTime
	err := s.Scan(
		&h.ID, &h.Agente, &proyectoID, &sesionID, &runtimeID, &h.Transporte, &h.HandleKind, &h.HandleRef,
		&h.Estado, &h.LeaseToken, &h.CapabilitiesJSON, &h.MetadataJSON, &lastSeen, &h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		h.ProyectoID = &proyectoID.Int64
	}
	if sesionID.Valid {
		h.SesionID = &sesionID.Int64
	}
	if runtimeID.Valid {
		h.RuntimeID = &runtimeID.Int64
	}
	if lastSeen.Valid {
		h.LastSeenAt = &lastSeen.Time
	}
	return &h, nil
}

func scanRuntimeOrder(s scanner) (*RuntimeOrder, error) {
	var o RuntimeOrder
	var proyectoID sql.NullInt64
	var runtimeID sql.NullInt64
	var handleID sql.NullInt64
	var leaseExpires sql.NullTime
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	err := s.Scan(
		&o.ID, &o.Agente, &proyectoID, &runtimeID, &handleID, &o.Tipo, &o.PayloadJSON, &o.ResultadoJSON,
		&o.ErrorText, &o.Estado, &o.ClaimedBy, &o.LeaseToken, &o.AttemptCount, &leaseExpires,
		&o.AvailableAt, &o.CreatedAt, &startedAt, &finishedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		o.ProyectoID = &proyectoID.Int64
	}
	if runtimeID.Valid {
		o.RuntimeID = &runtimeID.Int64
	}
	if handleID.Valid {
		o.HandleID = &handleID.Int64
	}
	if leaseExpires.Valid {
		o.LeaseExpiresAt = &leaseExpires.Time
	}
	if startedAt.Valid {
		o.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		o.FinishedAt = &finishedAt.Time
	}
	return &o, nil
}

func scanRuntimeMailbox(s scanner) (*RuntimeMailboxMessage, error) {
	var msg RuntimeMailboxMessage
	var proyectoID sql.NullInt64
	var orderID sql.NullInt64
	var deliveredAt sql.NullTime
	var consumedAt sql.NullTime
	err := s.Scan(
		&msg.ID, &msg.FromAgente, &msg.ToAgente, &proyectoID, &orderID, &msg.Kind, &msg.PayloadJSON,
		&msg.Estado, &msg.CreatedAt, &deliveredAt, &consumedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		msg.ProyectoID = &proyectoID.Int64
	}
	if orderID.Valid {
		msg.RuntimeOrderID = &orderID.Int64
	}
	if deliveredAt.Valid {
		msg.DeliveredAt = &deliveredAt.Time
	}
	if consumedAt.Valid {
		msg.ConsumedAt = &consumedAt.Time
	}
	return &msg, nil
}

func scanRuntimeCheckpoint(s scanner) (*RuntimeCheckpoint, error) {
	var cp RuntimeCheckpoint
	var proyectoID sql.NullInt64
	var sesionID sql.NullInt64
	var runtimeID sql.NullInt64
	err := s.Scan(
		&cp.ID, &cp.Agente, &proyectoID, &sesionID, &runtimeID, &cp.CheckpointKind, &cp.Resumen,
		&cp.Branch, &cp.CWD, &cp.PayloadJSON, &cp.ResumeStrategy, &cp.Source, &cp.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		cp.ProyectoID = &proyectoID.Int64
	}
	if sesionID.Valid {
		cp.SesionID = &sesionID.Int64
	}
	if runtimeID.Valid {
		cp.RuntimeID = &runtimeID.Int64
	}
	return &cp, nil
}

func scanRuntimeCheckpointWithLeadingTotal(s scanner, total *int) (*RuntimeCheckpoint, error) {
	var cp RuntimeCheckpoint
	var proyectoID sql.NullInt64
	var sesionID sql.NullInt64
	var runtimeID sql.NullInt64
	err := s.Scan(
		total,
		&cp.ID, &cp.Agente, &proyectoID, &sesionID, &runtimeID, &cp.CheckpointKind, &cp.Resumen,
		&cp.Branch, &cp.CWD, &cp.PayloadJSON, &cp.ResumeStrategy, &cp.Source, &cp.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		cp.ProyectoID = &proyectoID.Int64
	}
	if sesionID.Valid {
		cp.SesionID = &sesionID.Int64
	}
	if runtimeID.Valid {
		cp.RuntimeID = &runtimeID.Int64
	}
	return &cp, nil
}

func defaultRuntimeOrderEstado(v string) string {
	switch strings.TrimSpace(v) {
	case "tomada", "ejecutando", "completada", "fallida", "expirada", "cancelada":
		return strings.TrimSpace(v)
	default:
		return "pendiente"
	}
}

func defaultMailboxEstado(v string) string {
	switch strings.TrimSpace(v) {
	case "entregado", "consumido", "expirado", "cancelado":
		return strings.TrimSpace(v)
	default:
		return "pendiente"
	}
}

func mapFromJSON(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func boolFromMap(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "si", "sí", "on":
			return true
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	}
	return false
}

func intFromMap(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return 0
		}
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}

func int64PtrFromMap(m map[string]any, key string) *int64 {
	if m == nil {
		return nil
	}
	switch v := m[key].(type) {
	case int:
		val := int64(v)
		return &val
	case int64:
		return &v
	case float64:
		val := int64(v)
		return &val
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return &n
		}
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return &n
		}
	}
	return nil
}

func stringFromMap(m map[string]any, key, fallback string) string {
	if m == nil {
		return fallback
	}
	raw, ok := m[key]
	if !ok {
		return fallback
	}
	v, ok := raw.(string)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func preferTime(values ...*time.Time) *time.Time {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func timePtr(t time.Time) *time.Time { return &t }

func inferirLastSeenSesion(s *Sesion) *time.Time {
	if s == nil {
		return nil
	}
	now := time.Now().UTC()
	if estado := inferirEstadoHandleSesion(s); estado == "activo" || estado == "pausado" {
		return &now
	}
	return preferTime(s.HeartbeatAt, timePtr(s.Inicio), &now)
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func jsonNumber[T ~int64](v T) string {
	data, _ := json.Marshal(v)
	return string(data)
}
