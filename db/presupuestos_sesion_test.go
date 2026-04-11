/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRegistrarYLeerPresupuestoSesion(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}

	inicio := time.Now().Add(-4 * time.Hour).UTC()
	reset := inicio.Add(5 * time.Hour)
	remaining := int64(1200)
	tokens := int64(40000)

	id, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		ModelSlug:        "gpt-5",
		WindowKind:       "5h",
		WindowStartedAt:  &inicio,
		ResetAt:          &reset,
		RemainingSeconds: &remaining,
		RemainingTokens:  &tokens,
		BudgetSource:     "manual",
	})
	if err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}
	if id == 0 {
		t.Fatalf("id inesperado: %d", id)
	}

	p, err := UltimoPresupuestoSesion(sesionID)
	if err != nil {
		t.Fatalf("UltimoPresupuestoSesion: %v", err)
	}
	if p.SesionID != sesionID {
		t.Fatalf("sesion inesperada: %d", p.SesionID)
	}
	if p.RemainingSeconds == nil || *p.RemainingSeconds != remaining {
		t.Fatalf("remaining_seconds inesperado: %+v", p.RemainingSeconds)
	}
}

func TestEvaluarPresupuestoSesionHandoffPreventivo(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}

	inicio := time.Now().Add(-4*time.Hour - 30*time.Minute).UTC()
	reset := inicio.Add(5 * time.Hour)
	remaining := int64(900)

	_, err = RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       "5h",
		WindowStartedAt:  &inicio,
		ResetAt:          &reset,
		RemainingSeconds: &remaining,
		BudgetSource:     "manual",
	})
	if err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	p, sesion, err := UltimoPresupuestoAgente("codex1")
	if err != nil {
		t.Fatalf("UltimoPresupuestoAgente: %v", err)
	}
	if sesion == nil || sesion.ID != sesionID {
		t.Fatalf("sesion activa inesperada: %+v", sesion)
	}

	ev, err := EvaluarPresupuestoSesion(p)
	if err != nil {
		t.Fatalf("EvaluarPresupuestoSesion: %v", err)
	}
	if !ev.DebeHandoff || ev.Estado != "handoff_preventivo" {
		t.Fatalf("evaluacion inesperada: %+v", ev)
	}
	if ev.RemainingRatio == nil {
		t.Fatalf("esperaba ratio restante")
	}
}

func TestEvaluarPresupuestoSesionAgotadoPorCreditsFuerzaRatioCero(t *testing.T) {
	abrirDBTemporalMemoria(t)

	credits := 0.0
	inicio := time.Now().Add(-10 * time.Minute).UTC()
	reset := inicio.Add(5 * time.Hour)
	p := &PresupuestoSesion{
		WindowKind:       "5h",
		WindowStartedAt:  &inicio,
		ResetAt:          &reset,
		RemainingCredits: &credits,
		BudgetSource:     "codex_token_count_observed",
	}
	ev, err := EvaluarPresupuestoSesion(p)
	if err != nil {
		t.Fatalf("EvaluarPresupuestoSesion: %v", err)
	}
	if ev.Estado != "agotado" {
		t.Fatalf("estado inesperado: %+v", ev)
	}
	if ev.RemainingRatio == nil || *ev.RemainingRatio != 0 {
		t.Fatalf("ratio deberia ser cero cuando el credito esta agotado: %+v", ev)
	}
}

func TestListarAgentesEnriquecePresupuestoVisible(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	remaining := int64(900)
	credits := 12.5
	inicio := time.Now().Add(-4*time.Hour - 30*time.Minute).UTC()
	reset := inicio.Add(5 * time.Hour)
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       "5h",
		WindowStartedAt:  &inicio,
		ResetAt:          &reset,
		RemainingSeconds: &remaining,
		RemainingCredits: &credits,
		BudgetSource:     "runtime",
		RawSnapshotJSON:  `{"account":{"email":"codex1@example.com","username":"codex1_user"}}`,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agentes, err := ListarAgentes()
	if err != nil {
		t.Fatalf("ListarAgentes: %v", err)
	}
	var agente *Agente
	for _, item := range agentes {
		if item != nil && item.Nombre == "codex1" {
			agente = item
			break
		}
	}
	if agente == nil {
		t.Fatalf("codex1 no aparece en agentes: %+v", agentes)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct <= 0 {
		t.Fatalf("cuota restante no enriquecida: %+v", agente)
	}
	if agente.PresupuestoSesionPct == nil || *agente.PresupuestoSesionPct <= 0 {
		t.Fatalf("porcentaje de sesión no enriquecido: %+v", agente)
	}
	if agente.PresupuestoDiarioPct == nil || *agente.PresupuestoDiarioPct <= 0 {
		t.Fatalf("porcentaje diario no enriquecido: %+v", agente)
	}
	if agente.PresupuestoSemanalPct == nil || *agente.PresupuestoSemanalPct <= 0 {
		t.Fatalf("porcentaje semanal no enriquecido: %+v", agente)
	}
	if agente.PresupuestoVentana != "5h" {
		t.Fatalf("ventana efectiva inesperada: %+v", agente)
	}
	if agente.PresupuestoResetAt == nil || agente.PresupuestoResetAt.IsZero() {
		t.Fatalf("reset_at debería exponerse: %+v", agente)
	}
	if agente.RemainingCredits == nil || *agente.RemainingCredits != credits {
		t.Fatalf("remaining credits inesperado: %+v", agente)
	}
	if agente.PresupuestoEstado != "handoff_preventivo" {
		t.Fatalf("presupuesto estado inesperado: %+v", agente)
	}
	if agente.CuentaEmail != "codex1@example.com" {
		t.Fatalf("email de cuenta inesperado: %+v", agente)
	}
	if agente.CuentaUsuario != "codex1_user" {
		t.Fatalf("usuario de cuenta inesperado: %+v", agente)
	}
}

func TestListarAgentesUsaPresupuestoSemanalSiEsMasRestrictivo(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET consumo_dia_segundos=100, limite_dia_segundos=1000, consumo_semanal_segundos=950, limite_semanal_segundos=1000 WHERE nombre='codex1'`); err != nil {
		t.Fatalf("update agente: %v", err)
	}
	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 5 {
		t.Fatalf("debería usar semanal como restricción efectiva: %+v", agente)
	}
	if agente.PresupuestoDiarioPct == nil || *agente.PresupuestoDiarioPct != 90 {
		t.Fatalf("porcentaje diario inesperado: %+v", agente)
	}
	if agente.PresupuestoSemanalPct == nil || *agente.PresupuestoSemanalPct != 5 {
		t.Fatalf("porcentaje semanal inesperado: %+v", agente)
	}
	if agente.PresupuestoVentana != "weekly" {
		t.Fatalf("ventana efectiva debería ser weekly: %+v", agente)
	}
}

func TestListarAgentesUsaVentanasObservadasDesdeTokenCount(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(4 * time.Hour)
	resetSecondary := now.Add(4 * 24 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":43,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}},"account_email":"codex1@example.com","account_user":"codex1_user"}`
	checkedAt := now
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       checkedAt,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.PresupuestoSesionPct == nil || *agente.PresupuestoSesionPct != 88 {
		t.Fatalf("sesion pct inesperado: %+v", agente)
	}
	if agente.PresupuestoSemanalPct == nil || *agente.PresupuestoSemanalPct != 57 {
		t.Fatalf("semanal pct inesperado: %+v", agente)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 88 {
		t.Fatalf("cuota efectiva inesperada: %+v", agente)
	}
	if agente.PresupuestoVentana != "5h" {
		t.Fatalf("ventana efectiva inesperada: %+v", agente)
	}
	if agente.PresupuestoResetAt == nil || agente.PresupuestoResetAt.UTC().Unix() != resetPrimary.UTC().Unix() {
		t.Fatalf("reset efectivo inesperado: %+v", agente)
	}
	if agente.CuentaEmail != "codex1@example.com" || agente.CuentaUsuario != "codex1_user" {
		t.Fatalf("identidad inesperada: %+v", agente)
	}
}

func TestListarAgentesAgotaSiLaSemanalLlegaACeroAunqueLaVentanaCortaSigaViva(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(4 * time.Hour)
	resetSecondary := now.Add(4 * 24 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":18,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}}}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 0 {
		t.Fatalf("la cuota efectiva deberia agotarse por semanal: %+v", agente)
	}
	if agente.PresupuestoVentana != "weekly" {
		t.Fatalf("la ventana efectiva deberia ser weekly cuando la semanal es 0: %+v", agente)
	}
}

func TestGetAgenteNoDejaQueSnapshotObservadoStaleMandeSobreLaCuotaEfectiva(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET consumo_dia_segundos=4200, limite_dia_segundos=18000, consumo_semanal_segundos=4560, limite_semanal_segundos=126000 WHERE nombre='codex1'`); err != nil {
		t.Fatalf("update agente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_max_age_seconds", "60"); err != nil {
		t.Fatalf("config snapshot max age: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_observed_max_age_seconds", "60"); err != nil {
		t.Fatalf("config observed snapshot max age: %v", err)
	}
	resetPrimary := time.Now().UTC().Add(-4 * time.Hour)
	resetSecondary := time.Now().UTC().Add(3 * 24 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":97,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}},"account_email":"codex1@example.com"}`
	checkedAt := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       checkedAt,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 96 {
		t.Fatalf("la cuota efectiva deberia volver al derivado semanal, got=%+v", agente)
	}
	if !agente.PresupuestoStale {
		t.Fatalf("deberia marcar el snapshot observado como stale: %+v", agente)
	}
	if agente.PresupuestoEstado != "observado_stale" {
		t.Fatalf("estado de presupuesto inesperado: %+v", agente)
	}
	if agente.PresupuestoSesionPct == nil || *agente.PresupuestoSesionPct != 88 {
		t.Fatalf("deberia seguir mostrando la sesion observada para inspeccion: %+v", agente)
	}
	if agente.PresupuestoSemanalPct == nil || *agente.PresupuestoSemanalPct != 3 {
		t.Fatalf("deberia seguir mostrando la semanal observada para inspeccion: %+v", agente)
	}
	if agente.PresupuestoVentana != "weekly" {
		t.Fatalf("la ventana efectiva deberia seguir siendo weekly: %+v", agente)
	}
}

func TestPresupuestoSesionFrescoUsaTTLObservadoEspecifico(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := ConfigSet("pool_budget_snapshot_max_age_seconds", "60"); err != nil {
		t.Fatalf("config snapshot max age: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_observed_max_age_seconds", "3600"); err != nil {
		t.Fatalf("config observed snapshot max age: %v", err)
	}
	p := &PresupuestoSesion{
		BudgetSource: "codex_token_count_observed",
		CheckedAt:    time.Now().UTC().Add(-30 * time.Minute),
	}
	if !PresupuestoSesionFresco(p) {
		t.Fatalf("la cuota observada de Codex deberia seguir fresca con TTL especifico")
	}
}

func TestUltimoPresupuestoAgenteRecuperaSesionPausadaReciente(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesionID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}
	credits := 0.0
	reset := time.Now().UTC().Add(90 * time.Minute)
	started := time.Now().UTC().Add(-10 * time.Minute)
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       "5h",
		WindowStartedAt:  &started,
		ResetAt:          &reset,
		RemainingCredits: &credits,
		BudgetSource:     "codex_token_count_observed",
		CheckedAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	p, sesion, err := UltimoPresupuestoAgente("codex1")
	if err != nil {
		t.Fatalf("UltimoPresupuestoAgente: %v", err)
	}
	if sesion == nil || sesion.ID != sesionID {
		t.Fatalf("deberia reutilizar la sesion pausada reciente: %+v", sesion)
	}
	if p == nil || p.SesionID != sesionID {
		t.Fatalf("presupuesto inesperado: %+v", p)
	}
}

func TestGetAgenteProyectaEstadoCuotaDesdePresupuestoObservadoFresco(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_observed_max_age_seconds", "3600"); err != nil {
		t.Fatalf("config observed snapshot max age: %v", err)
	}
	credits := 0.0
	reset := time.Now().UTC().Add(90 * time.Minute)
	checkedAt := time.Now().UTC().Add(-10 * time.Minute)
	raw := `{"account_email":"codex1@example.com","account_user":"codex1_user"}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       "5h",
		ResetAt:          &reset,
		RemainingCredits: &credits,
		BudgetSource:     "codex_token_count_observed",
		RawSnapshotJSON:  raw,
		CheckedAt:        checkedAt,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.EstadoCuota != "enfriamiento" {
		t.Fatalf("estado_cuota proyectado inesperado: %+v", agente)
	}
	if agente.ReanimarAt == nil || agente.ReanimarAt.UTC().Unix() != reset.UTC().Unix() {
		t.Fatalf("reanimar_at proyectado inesperado: %+v", agente)
	}
	if agente.PresupuestoStale {
		t.Fatalf("el presupuesto observado no deberia estar stale: %+v", agente)
	}
}

func TestGetAgenteMarcaAgotadoSiLaSemanalEsCeroAunqueLaVentanaCortaSigaViva(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_observed_max_age_seconds", "3600"); err != nil {
		t.Fatalf("config observed snapshot max age: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(4 * time.Hour)
	resetSecondary := now.Add(4 * 24 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":18,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}}}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 0 {
		t.Fatalf("la cuota efectiva deberia ser cero: %+v", agente)
	}
	if agente.PresupuestoVentana != "weekly" {
		t.Fatalf("la ventana efectiva deberia ser weekly: %+v", agente)
	}
	if agente.EstadoCuota != "agotado" {
		t.Fatalf("deberia proyectar agotado desde la semanal visible: %+v", agente)
	}
	if agente.ReanimarAt == nil || agente.ReanimarAt.UTC().Unix() != resetSecondary.UTC().Unix() {
		t.Fatalf("reanimar_at semanal inesperado: %+v", agente)
	}
}

func TestGetAgenteMarcaEnfriamientoSiSoloSeAgotaLaVentanaCorta(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_observed_max_age_seconds", "3600"); err != nil {
		t.Fatalf("config observed snapshot max age: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(4 * time.Hour)
	resetSecondary := now.Add(4 * 24 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":100,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":12,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}}}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 0 {
		t.Fatalf("la cuota efectiva deberia ser cero por 5h: %+v", agente)
	}
	if agente.PresupuestoVentana != "5h" {
		t.Fatalf("la ventana efectiva deberia ser 5h: %+v", agente)
	}
	if agente.EstadoCuota != "enfriamiento" {
		t.Fatalf("deberia proyectar enfriamiento desde la ventana corta: %+v", agente)
	}
	if agente.ReanimarAt == nil || agente.ReanimarAt.UTC().Unix() != resetPrimary.UTC().Unix() {
		t.Fatalf("reanimar_at 5h inesperado: %+v", agente)
	}
}

func TestGetAgenteUsaSoloLaSemanalCuandoNoExisteVentanaTemporal(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_observed_max_age_seconds", "3600"); err != nil {
		t.Fatalf("config observed snapshot max age: %v", err)
	}
	now := time.Now().UTC()
	resetWeekly := now.Add(5 * 24 * time.Hour)
	raw := `{"rate_limits":{"secondary":{"used_percent":27,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeekly.Unix(), 10) + `}}}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "weekly",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.PresupuestoSesionPct != nil {
		t.Fatalf("no deberia inventar ventana temporal: %+v", agente)
	}
	if agente.PresupuestoSemanalPct == nil || *agente.PresupuestoSemanalPct != 73 {
		t.Fatalf("semanal pct inesperado: %+v", agente)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 73 {
		t.Fatalf("la cuota efectiva deberia salir de la semanal: %+v", agente)
	}
	if agente.PresupuestoVentana != "weekly" {
		t.Fatalf("la ventana efectiva deberia ser weekly: %+v", agente)
	}
}

func TestGetAgenteNoBloqueaEstadoVisibleCuandoSoloHaySemanalPositiva(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='agotado', consumo_dia_segundos=18000, limite_dia_segundos=18000 WHERE nombre='codex1'`); err != nil {
		t.Fatalf("update agente: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_observed_max_age_seconds", "3600"); err != nil {
		t.Fatalf("config observed snapshot max age: %v", err)
	}
	now := time.Now().UTC()
	resetWeekly := now.Add(5 * 24 * time.Hour)
	raw := `{"rate_limits":{"secondary":{"used_percent":27,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeekly.Unix(), 10) + `}}}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "weekly",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if !agente.Activo {
		t.Fatalf("el estado visible no deberia bloquear una cuenta solo semanal con saldo: %+v", agente)
	}
	if strings.TrimSpace(agente.EstadoCuota) == "agotado" || strings.TrimSpace(agente.EstadoCuota) == "enfriamiento" {
		t.Fatalf("estado_cuota visible no deberia seguir secuestrado por el derivado corto: %+v", agente)
	}
}

func TestGetAgenteRecuperaReanimarAtVisibleDesdeResetSemanalSiYaEstaEnfriado(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', reanimar_at=? WHERE nombre='codex1'`, time.Now().UTC().Add(-5*time.Minute)); err != nil {
		t.Fatalf("update agente: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(2 * time.Hour)
	resetWeekly := now.Add(4 * 24 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":5,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeekly.Unix(), 10) + `}}}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now.Add(-6 * time.Hour),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.ReanimarAt == nil || agente.PresupuestoResetAt == nil || agente.ReanimarAt.UTC().Unix() != agente.PresupuestoResetAt.UTC().Unix() {
		t.Fatalf("reanimar_at visible inesperado: %+v", agente)
	}
}

func TestGetAgenteDesbloqueaCuotaViejaCuandoElPresupuestoFrescoYaTieneSaldo(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex12", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex12")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='Presupuesto agotado observado', reanimar_at=? WHERE nombre='codex12'`, time.Now().UTC().Add(24*time.Hour)); err != nil {
		t.Fatalf("update agente: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(4 * time.Hour)
	resetWeekly := now.Add(6 * 24 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":6,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":32,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeekly.Unix(), 10) + `}},"account_email":"carlos@avidad.com","account_user":"Carlos Avidad"}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_profile_status",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex12")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.EstadoCuota != "activo" {
		t.Fatalf("deberia limpiar el bloqueo de cuota viejo cuando ya hay saldo fresco: %+v", agente)
	}
	if agente.ReanimarAt != nil {
		t.Fatalf("reanimar_at no deberia persistir tras limpiar bloqueo falso: %+v", agente)
	}
	if strings.TrimSpace(agente.MotivoPausa) != "" {
		t.Fatalf("motivo_pausa no deberia conservar el bloqueo de cuota viejo: %+v", agente)
	}
}

func TestGetAgenteOllamaIgnoraCuotaLegacyYPresupuestoProveedor(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	conector, err := GetConector("ollama-cli")
	if err != nil {
		t.Fatalf("GetConector ollama-cli: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Gemma1",
		ConectorID:  &conector.ID,
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if _, err := DB.Exec(
		`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='Ventana corta agotada observada', reanimar_at=?, consumo_dia_segundos=1200, consumo_semanal_segundos=1200 WHERE nombre='Gemma1'`,
		time.Now().UTC().Add(2*time.Hour),
	); err != nil {
		t.Fatalf("update agente: %v", err)
	}
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesion.ID,
		WindowKind:      "weekly",
		BudgetSource:    "provider_backoff",
		RawSnapshotJSON: `{"note":"no deberia aplicar a ollama"}`,
		CheckedAt:       time.Now().UTC(),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("Gemma1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if !agente.SinCuotaProveedor {
		t.Fatalf("Gemma1 deberia quedar marcado sin cuota de proveedor: %+v", agente)
	}
	if strings.TrimSpace(agente.EstadoCuota) != "activo" {
		t.Fatalf("Gemma1 no deberia quedar en cuota: %+v", agente)
	}
	if agente.ReanimarAt != nil {
		t.Fatalf("Gemma1 no deberia conservar cooldown: %+v", agente)
	}
	if agente.CuotaRestantePct != nil || agente.PresupuestoSesionPct != nil || agente.PresupuestoSemanalPct != nil {
		t.Fatalf("Gemma1 no deberia proyectar cuota visible: %+v", agente)
	}
	if strings.TrimSpace(agente.PresupuestoVentana) != "indefinida" || strings.TrimSpace(agente.PresupuestoFuente) != "local" {
		t.Fatalf("Gemma1 deberia indicar cuota local indefinida: %+v", agente)
	}
}

func TestGetAgenteExtraeCuentaDesdeRuntimeHandle(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	sesion, err := GetSesionByID(sesionID)
	if err != nil {
		t.Fatalf("GetSesionByID: %v", err)
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("UpsertRuntimeHandleDesdeSesion: %v", err)
	}
	if _, err := DB.Exec(
		`UPDATE runtime_handles SET metadata_json=? WHERE sesion_id=?`,
		`{"auth":{"email":"codex1-handle@example.com"},"username":"codex1_handle"}`,
		sesionID,
	); err != nil {
		t.Fatalf("update runtime_handle metadata: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuentaEmail != "codex1-handle@example.com" {
		t.Fatalf("email desde handle inesperado: %+v", agente)
	}
	if agente.CuentaUsuario != "codex1_handle" {
		t.Fatalf("usuario desde handle inesperado: %+v", agente)
	}
}

func TestGetAgenteRecuperaCuentaDesdePresupuestoPrevioSiBackoffNoTraeCorreo(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	checkedObserved := time.Now().UTC().Add(-20 * time.Minute)
	rawObserved := `{"account_email":"codex1@example.com","account_user":"codex1_user"}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: rawObserved,
		CheckedAt:       checkedObserved,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion observed: %v", err)
	}
	reset := time.Now().UTC().Add(50 * time.Minute)
	zeroMessages := int64(0)
	rawBackoff := `{"source":"runtime_order_send_instruction","account_user":"Codex1"}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:          sesionID,
		WindowKind:        "provider",
		ResetAt:           &reset,
		RemainingMessages: &zeroMessages,
		BudgetSource:      "provider_backoff",
		RawSnapshotJSON:   rawBackoff,
		CheckedAt:         time.Now().UTC(),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion backoff: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuentaEmail != "codex1@example.com" {
		t.Fatalf("deberia recuperar email observado previo: %+v", agente)
	}
	if agente.CuentaUsuario != "Codex1" {
		t.Fatalf("deberia preservar el usuario mas reciente del backoff: %+v", agente)
	}
}

func TestGetAgenteOcultaDerivadosCuandoProviderBackoffMarcaAgotado(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	// El consumo local por si solo derivaria porcentajes positivos.
	if _, err := DB.Exec(`UPDATE agentes SET consumo_semanal_segundos=5880, limite_semanal_segundos=126000, consumo_dia_segundos=0, limite_dia_segundos=18000 WHERE nombre=?`, "codex1"); err != nil {
		t.Fatalf("update agentes: %v", err)
	}
	reset := time.Now().UTC().Add(50 * time.Minute)
	zeroMessages := int64(0)
	rawBackoff := `{"source":"runtime_order_send_instruction","account_user":"Codex1"}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:          sesionID,
		WindowKind:        "provider",
		ResetAt:           &reset,
		RemainingMessages: &zeroMessages,
		BudgetSource:      "provider_backoff",
		RawSnapshotJSON:   rawBackoff,
		CheckedAt:         time.Now().UTC(),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion backoff: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.PresupuestoEstado != "agotado" || agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 0 {
		t.Fatalf("bloqueo visible inesperado: %+v", agente)
	}
	if agente.PresupuestoSesionPct != nil || agente.PresupuestoDiarioPct != nil || agente.PresupuestoSemanalPct != nil {
		t.Fatalf("no deberia conservar porcentajes derivados tras provider_backoff agotado: %+v", agente)
	}
}

func TestGetAgenteConservaDerivadosSiProviderBackoffYaEstaStale(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET consumo_semanal_segundos=31500, limite_semanal_segundos=126000, consumo_dia_segundos=0, limite_dia_segundos=18000 WHERE nombre=?`, "codex1"); err != nil {
		t.Fatalf("update agentes: %v", err)
	}
	reset := time.Now().UTC().Add(6 * 24 * time.Hour)
	zeroMessages := int64(0)
	rawBackoff := `{"source":"runtime_order_send_instruction","account_user":"Codex1","account_email":"maritere@avidad.com"}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:          sesionID,
		WindowKind:        "provider",
		ResetAt:           &reset,
		RemainingMessages: &zeroMessages,
		BudgetSource:      "provider_backoff",
		RawSnapshotJSON:   rawBackoff,
		CheckedAt:         time.Now().UTC().Add(-25 * time.Hour),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion backoff stale: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if !agente.PresupuestoStale {
		t.Fatalf("deberia marcarse stale: %+v", agente)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct <= 0 {
		t.Fatalf("no deberia bloquear con provider_backoff stale: %+v", agente)
	}
	if agente.PresupuestoSemanalPct == nil || *agente.PresupuestoSemanalPct <= 0 {
		t.Fatalf("deberia conservar semanal derivada: %+v", agente)
	}
	if agente.EstadoCuota != "activo" {
		t.Fatalf("no deberia quedar bloqueado con provider_backoff stale: %+v", agente)
	}
	if agente.PresupuestoEstado != "observado_stale" {
		t.Fatalf("deberia degradar el estado a observado_stale: %+v", agente)
	}
}

func TestGetAgenteConservaCuotaObservadaYUsoClaudeMasReciente(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("claude1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("claude1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	now := time.Now().UTC()
	resetWeekly := now.Add(5 * 24 * time.Hour)
	rawObserved := `{"rate_limits":{"secondary":{"used_percent":40,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeekly.Unix(), 10) + `}},"account_email":"carlos@avidad.com"}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "weekly",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: rawObserved,
		CheckedAt:       now.Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion observed: %v", err)
	}
	rawClaude := `{"account_email":"carlos@avidad.com","account_user":"Carlos Claude","session_usage":{"session_path":"/tmp/.claude/sessions/session-1.json","message_count":3,"total_tokens":1570,"estimated_cost_usd":0.042,"turns":1,"updated_at":"` + now.Format(time.RFC3339Nano) + `"}}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "unknown",
		BudgetSource:    "claude_rust_session_observed",
		RawSnapshotJSON: rawClaude,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion claude observed: %v", err)
	}

	agente, err := GetAgente("claude1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 60 {
		t.Fatalf("deberia conservar la cuota observada previa: %+v", agente)
	}
	if agente.CuentaEmail != "carlos@avidad.com" || agente.CuentaUsuario != "Carlos Claude" {
		t.Fatalf("deberia usar la identidad mas reciente de Claude: %+v", agente)
	}
	if agente.ObservedUsageTokens == nil || *agente.ObservedUsageTokens != 1570 {
		t.Fatalf("uso observado inesperado: %+v", agente)
	}
	if agente.ObservedUsageCostUSD == nil || *agente.ObservedUsageCostUSD <= 0 {
		t.Fatalf("coste observado inesperado: %+v", agente)
	}
	if agente.ObservedUsageMessages == nil || *agente.ObservedUsageMessages != 3 {
		t.Fatalf("message_count observado inesperado: %+v", agente)
	}
	if agente.ObservedSessionPath != "/tmp/.claude/sessions/session-1.json" {
		t.Fatalf("session_path observada inesperada: %+v", agente)
	}
	if agente.PresupuestoFuente != "codex_token_count_observed" {
		t.Fatalf("la fuente de cuota deberia seguir siendo la que aporta cuota: %+v", agente)
	}
}

func TestGetAgenteConsolidaCuotaCanonicaPorCuentaCompartida(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex5: %v", err)
	}
	sesion1, err := IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion Codex1: %v", err)
	}
	sesion5, err := IniciarSesion("Codex5")
	if err != nil {
		t.Fatalf("IniciarSesion Codex5: %v", err)
	}
	now := time.Now().UTC()
	resetWeekly := now.Add(5 * 24 * time.Hour)
	rawObserved := `{"rate_limits":{"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeekly.Unix(), 10) + `}},"account_email":"maritere@avidad.com","account_user":"maritere"}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesion5,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: rawObserved,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion observed: %v", err)
	}
	zeroMessages := int64(0)
	rawBackoff := `{"source":"runtime_order_send_instruction","account_user":"Codex1","account_email":"maritere@avidad.com"}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:          sesion1,
		WindowKind:        "provider",
		ResetAt:           &resetWeekly,
		RemainingMessages: &zeroMessages,
		BudgetSource:      "provider_backoff",
		RawSnapshotJSON:   rawBackoff,
		CheckedAt:         now.Add(time.Second),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion backoff: %v", err)
	}

	agente1, err := GetAgente("Codex1")
	if err != nil {
		t.Fatalf("GetAgente Codex1: %v", err)
	}
	agente5, err := GetAgente("Codex5")
	if err != nil {
		t.Fatalf("GetAgente Codex5: %v", err)
	}
	if agente1.PresupuestoFuente != "codex_token_count_observed" || agente5.PresupuestoFuente != "codex_token_count_observed" {
		t.Fatalf("la cuota canonica por cuenta deberia preferir la observacion real: a1=%s a5=%s", agente1.PresupuestoFuente, agente5.PresupuestoFuente)
	}
	if agente1.CuotaRestantePct == nil || agente5.CuotaRestantePct == nil || *agente1.CuotaRestantePct != *agente5.CuotaRestantePct {
		t.Fatalf("los agentes con la misma cuenta no deberian divergir: a1=%+v a5=%+v", agente1, agente5)
	}
}

func TestGetAgentePrefiereCodexProfileStatusComoCuotaCanonicaPorCuenta(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex7: %v", err)
	}
	sesionID, err := IniciarSesion("Codex7")
	if err != nil {
		t.Fatalf("IniciarSesion Codex7: %v", err)
	}

	now := time.Now().UTC()
	resetPrimary := now.Add(4 * time.Hour)
	resetWeekly := now.Add(5 * 24 * time.Hour)
	rawObserved := `{"rate_limits":{"primary":{"used_percent":5,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":62,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeekly.Unix(), 10) + `}},"account_email":"alberto@avidad.com","account_user":"Alberto Qvidad","plan_type":"team"}`

	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: rawObserved,
		CheckedAt:       now.Add(-30 * time.Minute),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion observed: %v", err)
	}
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_profile_status",
		RawSnapshotJSON: rawObserved,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion profile: %v", err)
	}
	if err := UpsertAgenteIdentidadObservada("Codex7", "alberto@avidad.com", "Alberto Qvidad", "codex_profile_status", &now); err != nil {
		t.Fatalf("UpsertAgenteIdentidadObservada: %v", err)
	}

	agente, err := GetAgente("Codex7")
	if err != nil {
		t.Fatalf("GetAgente Codex7: %v", err)
	}
	if agente.PresupuestoFuente != "codex_profile_status" {
		t.Fatalf("la cuota canonica deberia preferir codex_profile_status fresco: %+v", agente)
	}
	if agente.CuentaFuente != "codex_profile_status" {
		t.Fatalf("la identidad canónica debería mantenerse en codex_profile_status: %+v", agente)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 95 {
		t.Fatalf("cuota restante inesperada: %+v", agente)
	}
	if agente.PresupuestoSemanalPct == nil || *agente.PresupuestoSemanalPct != 38 {
		t.Fatalf("cuota semanal inesperada: %+v", agente)
	}
}

func TestGetAgenteNoMezclaPresupuestoCanonicoSiSoloCoincideEmailPeroDifiereAccountID(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex2: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex3: %v", err)
	}
	sesion2, err := IniciarSesion("Codex2")
	if err != nil {
		t.Fatalf("IniciarSesion Codex2: %v", err)
	}
	sesion3, err := IniciarSesion("Codex3")
	if err != nil {
		t.Fatalf("IniciarSesion Codex3: %v", err)
	}

	now := time.Now().UTC()
	resetShort := now.Add(4 * time.Hour)
	resetWeek := now.Add(5 * 24 * time.Hour)
	rawCodex2 := `{"account_id":"acct-codex2","account_email":"shared@example.com","account_user":"Codex2","rate_limits":{"primary":{"used_percent":10,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetShort.Unix(), 10) + `},"secondary":{"used_percent":60,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeek.Unix(), 10) + `}}}`
	rawCodex3 := `{"account_id":"acct-codex3","account_email":"shared@example.com","account_user":"Codex3","rate_limits":{"primary":{"used_percent":80,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetShort.Unix(), 10) + `},"secondary":{"used_percent":90,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeek.Unix(), 10) + `}}}`

	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesion2,
		WindowKind:      "5h",
		BudgetSource:    "codex_profile_status",
		RawSnapshotJSON: rawCodex2,
		CheckedAt:       now.Add(-1 * time.Minute),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion Codex2: %v", err)
	}
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesion3,
		WindowKind:      "5h",
		BudgetSource:    "codex_profile_status",
		RawSnapshotJSON: rawCodex3,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion Codex3: %v", err)
	}

	agente2, err := GetAgente("Codex2")
	if err != nil {
		t.Fatalf("GetAgente Codex2: %v", err)
	}
	if agente2.CuentaID != "acct-codex2" {
		t.Fatalf("cuenta canónica inesperada: %+v", agente2)
	}
	if agente2.CuotaRestantePct == nil || *agente2.CuotaRestantePct != 90 {
		t.Fatalf("Codex2 no deberia heredar la cuota de otra cuenta con mismo email: %+v", agente2)
	}
}

func TestGetAgenteNoMezclaCuotaCanonicaSiComparteEmailPeroTieneAccountIDDistinto(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex2: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex3: %v", err)
	}
	sesion2, err := IniciarSesion("Codex2")
	if err != nil {
		t.Fatalf("IniciarSesion Codex2: %v", err)
	}
	sesion3, err := IniciarSesion("Codex3")
	if err != nil {
		t.Fatalf("IniciarSesion Codex3: %v", err)
	}

	now := time.Now().UTC()
	reset2 := now.Add(4 * time.Hour)
	reset3 := now.Add(2 * time.Hour)
	raw2 := `{"account_id":"acc-codex2","account_email":"shared@example.com","account_user":"Codex2","rate_limits":{"primary":{"used_percent":15,"window_minutes":300,"resets_at":` + strconv.FormatInt(reset2.Unix(), 10) + `}}}`
	raw3 := `{"account_id":"acc-codex3","account_email":"shared@example.com","account_user":"Codex3","rate_limits":{"primary":{"used_percent":90,"window_minutes":300,"resets_at":` + strconv.FormatInt(reset3.Unix(), 10) + `}}}`

	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesion2,
		WindowKind:      "5h",
		BudgetSource:    "codex_profile_status",
		RawSnapshotJSON: raw2,
		CheckedAt:       now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion Codex2: %v", err)
	}
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesion3,
		WindowKind:      "5h",
		BudgetSource:    "codex_profile_status",
		RawSnapshotJSON: raw3,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion Codex3: %v", err)
	}

	agente2, err := GetAgente("Codex2")
	if err != nil {
		t.Fatalf("GetAgente Codex2: %v", err)
	}
	agente3, err := GetAgente("Codex3")
	if err != nil {
		t.Fatalf("GetAgente Codex3: %v", err)
	}
	if agente2.CuentaID != "acc-codex2" || agente3.CuentaID != "acc-codex3" {
		t.Fatalf("account_id inesperado: a2=%+v a3=%+v", agente2, agente3)
	}
	if agente2.CuotaRestantePct == nil || *agente2.CuotaRestantePct != 85 {
		t.Fatalf("Codex2 no deberia heredar la cuota de otra cuenta con mismo email: %+v", agente2)
	}
	if agente3.CuotaRestantePct == nil || *agente3.CuotaRestantePct != 10 {
		t.Fatalf("Codex3 cuota inesperada: %+v", agente3)
	}
}

func TestGetAgenteExtraePerfilDesdeRenderedCommandHandle(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	sesion, err := GetSesionByID(sesionID)
	if err != nil {
		t.Fatalf("GetSesionByID: %v", err)
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("UpsertRuntimeHandleDesdeSesion: %v", err)
	}
	if _, err := DB.Exec(
		`UPDATE runtime_handles SET metadata_json=? WHERE sesion_id=?`,
		`{"rendered_command":"'/tmp/codex-perfiles/bin/codex-perfil' 'CuentaReal-01'","driver":"process_pty_cli"}`,
		sesionID,
	); err != nil {
		t.Fatalf("update runtime_handle metadata: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuentaUsuario != "CuentaReal-01" {
		t.Fatalf("usuario desde rendered_command inesperado: %+v", agente)
	}
	if agente.CuentaFuente != "runtime_handle_profile" {
		t.Fatalf("fuente inesperada: %+v", agente)
	}
}

func TestUltimoPresupuestoSesionPorFuenteIgnoraFuentesPosteriores(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	observedAt := time.Now().UTC().Add(-2 * time.Minute)
	backoffAt := observedAt.Add(1 * time.Minute)
	rawObserved := `{"account_email":"codex1@example.com","rate_limits":{"primary":{"used_percent":12}}}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: rawObserved,
		CheckedAt:       observedAt,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion observado: %v", err)
	}
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "provider",
		BudgetSource:    "provider_backoff",
		RawSnapshotJSON: `{"error":"usage_limit"}`,
		CheckedAt:       backoffAt,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion provider_backoff: %v", err)
	}

	ultimo, err := UltimoPresupuestoSesionPorFuente(sesionID, "codex_token_count_observed")
	if err != nil {
		t.Fatalf("UltimoPresupuestoSesionPorFuente: %v", err)
	}
	if ultimo == nil {
		t.Fatalf("deberia devolver el snapshot observado")
	}
	if ultimo.BudgetSource != "codex_token_count_observed" {
		t.Fatalf("fuente inesperada: %+v", ultimo)
	}
	if ultimo.CheckedAt.UTC().Unix() != observedAt.UTC().Unix() {
		t.Fatalf("checked_at inesperado: %+v", ultimo)
	}
	if strings.TrimSpace(ultimo.RawSnapshotJSON) != rawObserved {
		t.Fatalf("raw snapshot inesperado: %+v", ultimo)
	}
}

func TestPresupuestoSesionFrescoAceptaCodexStatusLive(t *testing.T) {
	abrirDBTemporalMemoria(t)
	p := &PresupuestoSesion{
		BudgetSource: "codex_status_live",
		CheckedAt:    time.Now().UTC().Add(-10 * time.Minute),
	}
	if !PresupuestoSesionFresco(p) {
		t.Fatalf("codex_status_live deberia considerarse snapshot observado fresco")
	}
}

func TestPresupuestoSesionFrescoAceptaCodexProfileStatus(t *testing.T) {
	abrirDBTemporalMemoria(t)
	p := &PresupuestoSesion{
		BudgetSource: "codex_profile_status",
		CheckedAt:    time.Now().UTC().Add(-30 * time.Minute),
	}
	if !PresupuestoSesionFresco(p) {
		t.Fatalf("codex_profile_status deberia considerarse snapshot observado fresco")
	}
}
