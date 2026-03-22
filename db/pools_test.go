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

func TestSeleccionModeloSegunPresupuesto(t *testing.T) {
	abrirDBTemporalMemoria(t)

	poolID, err := CrearPoolCapacidad(&PoolCapacidad{
		Slug:                "codex-main",
		Proveedor:           "openai",
		Runtime:             "codex",
		CapacidadTotal:      2,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        "{}",
		PermiteHijos:        true,
		PermiteModelosMulti: true,
	})
	if err != nil {
		t.Fatalf("CrearPoolCapacidad: %v", err)
	}

	_, err = RegistrarPoolModelo(&PoolModelo{
		PoolID:        poolID,
		ModelSlug:     "gpt-5.4",
		Prioridad:     10,
		CosteRelativo: 2.0,
	})
	if err != nil {
		t.Fatalf("RegistrarPoolModelo premium: %v", err)
	}
	_, err = RegistrarPoolModelo(&PoolModelo{
		PoolID:        poolID,
		ModelSlug:     "gpt-5.4-mini",
		Prioridad:     20,
		CosteRelativo: 1.0,
	})
	if err != nil {
		t.Fatalf("RegistrarPoolModelo barato: %v", err)
	}

	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}

	startedAt := time.Now().Add(-2 * time.Hour)
	resetAt := time.Now().Add(3 * time.Hour)
	remainingOK := int64(7200)
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		PoolID:           &poolID,
		ModelSlug:        "gpt-5.4",
		WindowKind:       "session_5h",
		WindowStartedAt:  &startedAt,
		ResetAt:          &resetAt,
		RemainingSeconds: &remainingOK,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion ok: %v", err)
	}

	sel, err := SeleccionarModeloParaAgente("codex1", nil, "")
	if err != nil {
		t.Fatalf("SeleccionarModeloParaAgente continuidad: %v", err)
	}
	if sel.Modelo == nil || sel.Modelo.ModelSlug != "gpt-5.4" {
		t.Fatalf("modelo esperado gpt-5.4, obtenido: %+v", sel.Modelo)
	}

	remainingLow := int64(120)
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		PoolID:           &poolID,
		ModelSlug:        "gpt-5.4",
		WindowKind:       "session_5h",
		WindowStartedAt:  &startedAt,
		ResetAt:          &resetAt,
		RemainingSeconds: &remainingLow,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion low: %v", err)
	}

	sel, err = SeleccionarModeloParaAgente("codex1", nil, "")
	if err != nil {
		t.Fatalf("SeleccionarModeloParaAgente ahorro: %v", err)
	}
	if sel.Modelo == nil || sel.Modelo.ModelSlug != "gpt-5.4-mini" {
		t.Fatalf("modelo esperado gpt-5.4-mini, obtenido: %+v", sel.Modelo)
	}
}

func TestRegistrarPresupuestoSesionValidaPoolYModelo(t *testing.T) {
	abrirDBTemporalMemoria(t)

	poolID, err := CrearPoolCapacidad(&PoolCapacidad{
		Slug:                "claude-main",
		Proveedor:           "anthropic",
		Runtime:             "claude",
		CapacidadTotal:      1,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        "{}",
		PermiteHijos:        true,
		PermiteModelosMulti: true,
	})
	if err != nil {
		t.Fatalf("CrearPoolCapacidad: %v", err)
	}

	if _, err := RegistrarPoolModelo(&PoolModelo{
		PoolID:        poolID,
		ModelSlug:     "claude-sonnet",
		Prioridad:     10,
		CosteRelativo: 1.0,
	}); err != nil {
		t.Fatalf("RegistrarPoolModelo: %v", err)
	}

	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}

	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:  sesionID,
		PoolID:    &poolID,
		ModelSlug: "modelo-inexistente",
	}); err == nil {
		t.Fatalf("se esperaba error por modelo no perteneciente al pool")
	}
}
