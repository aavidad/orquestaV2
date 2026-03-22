/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestSesionHistorialYVer(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-cli-001",
		ResumenContinuidad: "historial cli",
		Branch:             "main",
		Host:               "host-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if err := sesionHistorialCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := sesionHistorialCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto: %v", err)
	}
	if err := sesionHistorialCmd.Flags().Set("activa", "true"); err != nil {
		t.Fatalf("set activa: %v", err)
	}

	outHistorial := capturarStdout(t, func() {
		if err := sesionHistorialCmd.RunE(sesionHistorialCmd, nil); err != nil {
			t.Fatalf("run historial: %v", err)
		}
	})
	for _, token := range []string{"ID", "Codex1", "orquestador", "sess-cli-001"} {
		if !strings.Contains(outHistorial, token) {
			t.Fatalf("salida historial sin %q:\n%s", token, outHistorial)
		}
	}

	outVer := capturarStdout(t, func() {
		if err := sesionVerCmd.RunE(sesionVerCmd, []string{itoa(sesion.ID)}); err != nil {
			t.Fatalf("run ver: %v", err)
		}
	})
	for _, token := range []string{"Sesión", "Codex1", "sess-cli-001", "historial cli"} {
		if !strings.Contains(outVer, token) {
			t.Fatalf("salida ver sin %q:\n%s", token, outVer)
		}
	}
}
