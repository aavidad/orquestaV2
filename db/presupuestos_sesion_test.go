/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
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
