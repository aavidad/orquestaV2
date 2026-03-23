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
	"strconv"
	"time"
)

type Sesion struct {
	ID     int64
	Agente string
	Inicio time.Time
	Fin    *time.Time
	Activa bool
}

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

func GetSesion(id int64) (*Sesion, error) {
	row := DB.QueryRow(`SELECT id, agente, inicio, fin, activa FROM sesiones WHERE id = ?`, id)
	return escanearSesion(row)
}

func SesionActivaDeAgente(agente string) (*Sesion, error) {
	row := DB.QueryRow(`
		SELECT id, agente, inicio, fin, activa
		FROM sesiones
		WHERE agente = ? AND activa = 1
		ORDER BY id DESC
		LIMIT 1`, agente)
	return escanearSesion(row)
}

func RegistrarPresupuestoSesion(p *PresupuestoSesion) (int64, error) {
	if p == nil {
		return 0, fmt.Errorf("presupuesto nulo")
	}
	if p.SesionID <= 0 {
		return 0, fmt.Errorf("sesion_id obligatorio")
	}
	if _, err := GetSesion(p.SesionID); err != nil {
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
		p.SesionID, p.PoolID, p.ModelSlug, p.WindowKind, nullableTime(p.WindowStartedAt), nullableTime(p.ResetAt),
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

func UltimoPresupuestoAgente(agente string) (*PresupuestoSesion, *Sesion, error) {
	sesion, err := SesionActivaDeAgente(agente)
	if err == sql.ErrNoRows {
		return nil, nil, err
	}
	if err != nil {
		return nil, nil, err
	}
	p, err := UltimoPresupuestoSesion(sesion.ID)
	return p, sesion, err
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

func escanearSesion(s scanner) (*Sesion, error) {
	var sesion Sesion
	var fin sql.NullTime
	if err := s.Scan(&sesion.ID, &sesion.Agente, &sesion.Inicio, &fin, &sesion.Activa); err != nil {
		return nil, err
	}
	if fin.Valid {
		sesion.Fin = &fin.Time
	}
	return &sesion, nil
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
		float64PtrLTEZero(p.RemainingCredits)
}

func int64PtrLTEZero(v *int64) bool {
	return v != nil && *v <= 0
}

func float64PtrLTEZero(v *float64) bool {
	return v != nil && *v <= 0
}

func remainingRatio(p *PresupuestoSesion) *float64 {
	if p == nil || p.RemainingSeconds == nil || p.WindowStartedAt == nil || p.ResetAt == nil {
		return nil
	}
	total := p.ResetAt.Sub(*p.WindowStartedAt).Seconds()
	if total <= 0 {
		return nil
	}
	ratio := float64(*p.RemainingSeconds) / total
	if ratio < 0 {
		ratio = 0
	}
	return &ratio
}
