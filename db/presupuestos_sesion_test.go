/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"strconv"
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
