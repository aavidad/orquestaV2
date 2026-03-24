/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"testing"

	"orquesta/db"
)

func TestValorCampoSesion(t *testing.T) {
	pid := int64(4321)
	s := &db.Sesion{
		ID:                 12,
		Agente:             "Codex1",
		ProyectoSlug:       "orquestador",
		CWD:                "/tmp/work",
		Herramienta:        "codex",
		ExternalSessionID:  "sess-123",
		ResumenContinuidad: "seguir por runtime",
		ResumePayloadJSON:  `{"k":"v"}`,
		Branch:             "feat/op-086",
		Estado:             "pausada",
		Host:               "host1",
		PID:                &pid,
	}

	casos := map[string]string{
		"id":                  "12",
		"agente":              "Codex1",
		"proyecto":            "orquestador",
		"cwd":                 "/tmp/work",
		"herramienta":         "codex",
		"external-session-id": "sess-123",
		"branch":              "feat/op-086",
		"resumen":             "seguir por runtime",
		"resume-payload":      `{"k":"v"}`,
		"estado":              "pausada",
		"host":                "host1",
		"pid":                 "4321",
	}

	for campo, want := range casos {
		got, err := valorCampoSesion(s, campo)
		if err != nil {
			t.Fatalf("valorCampoSesion(%q): %v", campo, err)
		}
		if got != want {
			t.Fatalf("valorCampoSesion(%q) = %q, want %q", campo, got, want)
		}
	}
}

func TestValorCampoSesionCampoDesconocido(t *testing.T) {
	if _, err := valorCampoSesion(&db.Sesion{}, "foo"); err == nil {
		t.Fatalf("se esperaba error para campo no soportado")
	}
}
