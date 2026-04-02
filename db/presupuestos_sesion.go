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
	"strconv"
	"strings"
	"time"
)

type PresupuestoSesion struct {
	ID                int64
	SesionID          int64
	PoolID            *int64
	ModelSlug         string
	WindowKind        string
	WindowStartedAt   *time.Time
	ResetAt           *time.Time
	RemainingSeconds  *int64
	RemainingMessages *int64
	RemainingTokens   *int64
	RemainingCredits  *float64
	BudgetSource      string
	RawSnapshotJSON   string
	CheckedAt         time.Time
	CreatedAt         time.Time
}

type EvaluacionPresupuesto struct {
	Estado           string
	DebeHandoff      bool
	Motivo           string
	ThresholdSeconds int64
	ThresholdRatio   float64
	RemainingRatio   *float64
}

func RegistrarPresupuestoSesion(p *PresupuestoSesion) (int64, error) {
	if p == nil {
		return 0, fmt.Errorf("presupuesto nulo")
	}
	if p.SesionID <= 0 {
		return 0, fmt.Errorf("sesion_id obligatorio")
	}
	if _, err := GetSesionByID(p.SesionID); err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("sesión #%d no encontrada", p.SesionID)
		}
		return 0, err
	}
	if p.WindowKind == "" {
		p.WindowKind = "unknown"
	}
	if p.PoolID != nil {
		if _, err := GetPoolCapacidad(*p.PoolID); err != nil {
			if err == sql.ErrNoRows {
				return 0, fmt.Errorf("pool #%d no encontrado", *p.PoolID)
			}
			return 0, err
		}
		if p.ModelSlug != "" {
			if _, err := GetPoolModeloPorSlug(*p.PoolID, p.ModelSlug); err != nil {
				if err == sql.ErrNoRows {
					return 0, fmt.Errorf("el modelo %s no pertenece al pool #%d", p.ModelSlug, *p.PoolID)
				}
				return 0, err
			}
		}
	}
	if p.BudgetSource == "" {
		p.BudgetSource = defaultBudgetSource()
	}
	if p.RawSnapshotJSON == "" {
		p.RawSnapshotJSON = "{}"
	}

	res, err := DB.Exec(`
		INSERT INTO presupuestos_sesion (
			sesion_id, pool_id, model_slug, window_kind, window_started_at, reset_at,
			remaining_seconds, remaining_messages, remaining_tokens, remaining_credits,
			budget_source, raw_snapshot_json, checked_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.SesionID, p.PoolID, p.ModelSlug, p.WindowKind, nullableTimePresupuesto(p.WindowStartedAt), nullableTimePresupuesto(p.ResetAt),
		nullableInt64(p.RemainingSeconds), nullableInt64(p.RemainingMessages), nullableInt64(p.RemainingTokens),
		nullableFloat64(p.RemainingCredits), p.BudgetSource, p.RawSnapshotJSON, checkedAtOrNow(p.CheckedAt),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("orquesta", "registrar_presupuesto_sesion", "presupuesto_sesion", id, fmt.Sprintf("sesion=%d", p.SesionID))
	return id, nil
}

func UltimoPresupuestoSesion(sesionID int64) (*PresupuestoSesion, error) {
	row := DB.QueryRow(`
		SELECT id, sesion_id, pool_id, model_slug, window_kind, window_started_at, reset_at,
		       remaining_seconds, remaining_messages, remaining_tokens, remaining_credits,
		       budget_source, raw_snapshot_json, checked_at, created_at
		FROM presupuestos_sesion
		WHERE sesion_id = ?
		ORDER BY checked_at DESC, id DESC
		LIMIT 1`, sesionID)
	return escanearPresupuestoSesion(row)
}

func UltimoPresupuestoSesionPorFuente(sesionID int64, budgetSource string) (*PresupuestoSesion, error) {
	row := DB.QueryRow(`
		SELECT id, sesion_id, pool_id, model_slug, window_kind, window_started_at, reset_at,
		       remaining_seconds, remaining_messages, remaining_tokens, remaining_credits,
		       budget_source, raw_snapshot_json, checked_at, created_at
		FROM presupuestos_sesion
		WHERE sesion_id = ? AND budget_source = ?
		ORDER BY checked_at DESC, id DESC
		LIMIT 1`, sesionID, strings.TrimSpace(budgetSource))
	return escanearPresupuestoSesion(row)
}

func UltimoPresupuestoAgente(agente string) (*PresupuestoSesion, *Sesion, error) {
	sesion, err := GetSesionActiva(agente, nil)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, nil, err
		}
		sesion = nil
	}
	if sesion == nil {
		sesion, err = GetSesionAbierta(agente, nil)
		if err != nil && err != sql.ErrNoRows {
			return nil, nil, err
		}
	}
	if sesion == nil {
		sesion, err = ObtenerUltimaSesion(agente, nil)
		if err != nil {
			return nil, nil, err
		}
	}
	if sesion == nil {
		return nil, nil, sql.ErrNoRows
	}
	p, err := UltimoPresupuestoSesion(sesion.ID)
	return p, sesion, err
}

func PresupuestoSesionFresco(p *PresupuestoSesion) bool {
	if p == nil {
		return false
	}
	maxAge := time.Duration(presupuestoSnapshotMaxAgeSeconds(p)) * time.Second
	if maxAge <= 0 {
		maxAge = 5 * time.Minute
	}
	if p.CheckedAt.IsZero() {
		return false
	}
	return time.Since(p.CheckedAt) <= maxAge
}

func presupuestoSnapshotMaxAgeSeconds(p *PresupuestoSesion) int64 {
	if p != nil && (strings.EqualFold(strings.TrimSpace(p.BudgetSource), "codex_token_count_observed") ||
		strings.EqualFold(strings.TrimSpace(p.BudgetSource), "claude_rust_session_observed")) {
		if observed := configInt64Fallback("pool_budget_snapshot_observed_max_age_seconds", 3600); observed > 0 {
			return observed
		}
	}
	return configInt64Fallback("pool_budget_snapshot_max_age_seconds", 300)
}

func PresupuestoSesionAportaCuota(p *PresupuestoSesion) bool {
	if p == nil {
		return false
	}
	if p.RemainingSeconds != nil || p.RemainingMessages != nil || p.RemainingTokens != nil || p.RemainingCredits != nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(p.BudgetSource), "provider_backoff") {
		return true
	}
	raw := mapFromJSON(p.RawSnapshotJSON)
	if raw == nil {
		return false
	}
	rateLimits, _ := raw["rate_limits"].(map[string]any)
	if rateLimits == nil {
		return false
	}
	for _, key := range []string{"primary", "secondary"} {
		window, _ := rateLimits[key].(map[string]any)
		if window == nil {
			continue
		}
		if _, ok := snapshotFloat64(window["used_percent"]); ok {
			return true
		}
	}
	return false
}

func UltimoPresupuestoAgenteConCuota(agente string) (*PresupuestoSesion, *Sesion, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil, sql.ErrNoRows
	}
	rows, err := DB.Query(`
		SELECT p.id, p.sesion_id, p.pool_id, p.model_slug, p.window_kind, p.window_started_at, p.reset_at,
		       p.remaining_seconds, p.remaining_messages, p.remaining_tokens, p.remaining_credits,
		       p.budget_source, p.raw_snapshot_json, p.checked_at, p.created_at,
		       s.id, s.agente, s.conector_id, s.proyecto_id, s.inicio, s.fin, s.activa, s.estado,
		       s.cwd, s.herramienta, s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM presupuestos_sesion p
		JOIN sesiones s ON s.id = p.sesion_id
		WHERE s.agente = ?
		ORDER BY p.checked_at DESC, p.id DESC
		LIMIT 25`, agente)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		p, sesion, err := scanPresupuestoAgenteConSesion(rows)
		if err != nil {
			return nil, nil, err
		}
		if PresupuestoSesionAportaCuota(p) {
			return p, sesion, nil
		}
	}
	return nil, nil, sql.ErrNoRows
}

func scanPresupuestoAgenteConSesion(s scanner) (*PresupuestoSesion, *Sesion, error) {
	p := &PresupuestoSesion{}
	sesion := &Sesion{}
	var poolID sql.NullInt64
	var startedAt sql.NullTime
	var resetAt sql.NullTime
	var remainingSeconds sql.NullInt64
	var remainingMessages sql.NullInt64
	var remainingTokens sql.NullInt64
	var remainingCredits sql.NullFloat64
	var conectorID sql.NullInt64
	var proyectoID sql.NullInt64
	var fin sql.NullTime
	var heartbeat sql.NullTime
	var pid sql.NullInt64
	if err := s.Scan(
		&p.ID, &p.SesionID, &poolID, &p.ModelSlug, &p.WindowKind, &startedAt, &resetAt,
		&remainingSeconds, &remainingMessages, &remainingTokens, &remainingCredits,
		&p.BudgetSource, &p.RawSnapshotJSON, &p.CheckedAt, &p.CreatedAt,
		&sesion.ID, &sesion.Agente, &conectorID, &proyectoID, &sesion.Inicio, &fin, &sesion.Activa, &sesion.Estado,
		&sesion.CWD, &sesion.Herramienta, &sesion.ExternalSessionID, &sesion.ResumePayloadJSON, &sesion.ResumenContinuidad,
		&sesion.Branch, &heartbeat, &sesion.Host, &pid,
	); err != nil {
		return nil, nil, err
	}
	if poolID.Valid {
		p.PoolID = &poolID.Int64
	}
	if startedAt.Valid {
		p.WindowStartedAt = &startedAt.Time
	}
	if resetAt.Valid {
		p.ResetAt = &resetAt.Time
	}
	if remainingSeconds.Valid {
		p.RemainingSeconds = &remainingSeconds.Int64
	}
	if remainingMessages.Valid {
		p.RemainingMessages = &remainingMessages.Int64
	}
	if remainingTokens.Valid {
		p.RemainingTokens = &remainingTokens.Int64
	}
	if remainingCredits.Valid {
		p.RemainingCredits = &remainingCredits.Float64
	}
	if conectorID.Valid {
		sesion.ConectorID = &conectorID.Int64
	}
	if proyectoID.Valid {
		sesion.ProyectoID = &proyectoID.Int64
	}
	if fin.Valid {
		sesion.Fin = &fin.Time
	}
	if heartbeat.Valid {
		sesion.HeartbeatAt = &heartbeat.Time
	}
	if pid.Valid {
		sesion.PID = &pid.Int64
	}
	return p, sesion, nil
}

func EvaluarPresupuestoSesion(p *PresupuestoSesion) (*EvaluacionPresupuesto, error) {
	if p == nil {
		return &EvaluacionPresupuesto{Estado: "sin_datos", Motivo: "sin presupuesto registrado"}, nil
	}
	thresholdSeconds := configInt64Fallback("pool_handoff_threshold_seconds", 1800)
	thresholdRatio := configFloat64Fallback("pool_handoff_threshold_ratio", 0.10)

	ev := &EvaluacionPresupuesto{
		Estado:           "ok",
		ThresholdSeconds: thresholdSeconds,
		ThresholdRatio:   thresholdRatio,
	}
	ev.RemainingRatio = remainingRatio(p)

	if agotadoPorConteo(p) {
		ev.Estado = "agotado"
		ev.DebeHandoff = true
		ev.Motivo = "presupuesto agotado"
		return ev, nil
	}

	if p.RemainingSeconds != nil && *p.RemainingSeconds <= thresholdSeconds {
		ev.Estado = "handoff_preventivo"
		ev.DebeHandoff = true
		ev.Motivo = fmt.Sprintf("quedan %d s, umbral=%d s", *p.RemainingSeconds, thresholdSeconds)
		return ev, nil
	}

	if ev.RemainingRatio != nil {
		if *ev.RemainingRatio <= thresholdRatio {
			ev.Estado = "handoff_preventivo"
			ev.DebeHandoff = true
			ev.Motivo = fmt.Sprintf("ratio restante %.2f <= %.2f", *ev.RemainingRatio, thresholdRatio)
			return ev, nil
		}
		ev.Motivo = fmt.Sprintf("ratio restante %.2f", *ev.RemainingRatio)
		return ev, nil
	}

	if p.RemainingSeconds != nil {
		ev.Motivo = fmt.Sprintf("quedan %d s", *p.RemainingSeconds)
		return ev, nil
	}

	ev.Estado = "sin_datos"
	ev.Motivo = "sin telemetría suficiente para decidir handoff"
	return ev, nil
}

func escanearPresupuestoSesion(s scanner) (*PresupuestoSesion, error) {
	p := &PresupuestoSesion{}
	var poolID sql.NullInt64
	var startedAt sql.NullTime
	var resetAt sql.NullTime
	var remainingSeconds sql.NullInt64
	var remainingMessages sql.NullInt64
	var remainingTokens sql.NullInt64
	var remainingCredits sql.NullFloat64
	if err := s.Scan(
		&p.ID, &p.SesionID, &poolID, &p.ModelSlug, &p.WindowKind, &startedAt, &resetAt,
		&remainingSeconds, &remainingMessages, &remainingTokens, &remainingCredits,
		&p.BudgetSource, &p.RawSnapshotJSON, &p.CheckedAt, &p.CreatedAt,
	); err != nil {
		return nil, err
	}
	if poolID.Valid {
		p.PoolID = &poolID.Int64
	}
	if startedAt.Valid {
		p.WindowStartedAt = &startedAt.Time
	}
	if resetAt.Valid {
		p.ResetAt = &resetAt.Time
	}
	if remainingSeconds.Valid {
		p.RemainingSeconds = &remainingSeconds.Int64
	}
	if remainingMessages.Valid {
		p.RemainingMessages = &remainingMessages.Int64
	}
	if remainingTokens.Valid {
		p.RemainingTokens = &remainingTokens.Int64
	}
	if remainingCredits.Valid {
		p.RemainingCredits = &remainingCredits.Float64
	}
	return p, nil
}

func defaultBudgetSource() string {
	v, err := ConfigGet("pool_default_budget_source")
	if err != nil || v == "" {
		return "manual"
	}
	return v
}

func configInt64Fallback(clave string, fallback int64) int64 {
	v, err := ConfigGet(clave)
	if err != nil {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func configFloat64Fallback(clave string, fallback float64) float64 {
	v, err := ConfigGet(clave)
	if err != nil {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}

func checkedAtOrNow(v time.Time) time.Time {
	if v.IsZero() {
		return time.Now()
	}
	return v
}

func nullableTimePresupuesto(v *time.Time) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableFloat64(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func agotadoPorConteo(p *PresupuestoSesion) bool {
	return int64PtrLTEZero(p.RemainingSeconds) ||
		int64PtrLTEZero(p.RemainingMessages) ||
		int64PtrLTEZero(p.RemainingTokens) ||
		float64PtrLTEZero(p.RemainingCredits) ||
		snapshotRateLimitAgotado(p)
}

func int64PtrLTEZero(v *int64) bool {
	return v != nil && *v <= 0
}

func float64PtrLTEZero(v *float64) bool {
	return v != nil && *v <= 0
}

func remainingRatio(p *PresupuestoSesion) *float64 {
	if agotadoPorConteo(p) {
		ratio := 0.0
		return &ratio
	}
	if p != nil && p.RemainingSeconds != nil && p.WindowStartedAt != nil && p.ResetAt != nil {
		total := p.ResetAt.Sub(*p.WindowStartedAt).Seconds()
		if total > 0 {
			ratio := float64(*p.RemainingSeconds) / total
			if ratio < 0 {
				ratio = 0
			}
			return &ratio
		}
	}
	if ratio := snapshotRemainingRatio(p); ratio != nil {
		return ratio
	}
	return nil
}

func snapshotRemainingRatio(p *PresupuestoSesion) *float64 {
	if p == nil {
		return nil
	}
	raw := mapFromJSON(p.RawSnapshotJSON)
	if raw == nil {
		return nil
	}
	rateLimits, _ := raw["rate_limits"].(map[string]any)
	if rateLimits == nil {
		return nil
	}
	primary, _ := rateLimits["primary"].(map[string]any)
	if primary == nil {
		return nil
	}
	used, ok := snapshotFloat64(primary["used_percent"])
	if !ok {
		return nil
	}
	ratio := 1 - (used / 100)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return &ratio
}

func snapshotRateLimitAgotado(p *PresupuestoSesion) bool {
	ratio := snapshotRemainingRatio(p)
	return ratio != nil && *ratio <= 0
}

func snapshotFloat64(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		out, err := v.Float64()
		if err == nil {
			return out, true
		}
	case string:
		out, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return out, true
		}
	}
	return 0, false
}
