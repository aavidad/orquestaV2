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

func TestRuntimeListarYVer(t *testing.T) {
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
		ExternalSessionID:  "sess-runtime-cli",
		ResumenContinuidad: "runtime cli",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if err := runtimeListarCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeListarCmd.Flags().Set("proyecto", "")
	})

	outListar := capturarStdout(t, func() {
		if err := runtimeListarCmd.RunE(runtimeListarCmd, nil); err != nil {
			t.Fatalf("run listar: %v", err)
		}
	})
	for _, token := range []string{"ID", "Codex1", "orquestador", "codex-cli"} {
		if !strings.Contains(outListar, token) {
			t.Fatalf("salida listar sin %q:\n%s", token, outListar)
		}
	}

	outVer := capturarStdout(t, func() {
		if err := runtimeVerCmd.RunE(runtimeVerCmd, []string{itoa(sesion.ID)}); err != nil {
			t.Fatalf("run ver: %v", err)
		}
	})
	for _, token := range []string{"Runtime #", "Codex1", "disponible", "codex-cli", "Muestras recientes"} {
		if !strings.Contains(outVer, token) {
			t.Fatalf("salida ver sin %q:\n%s", token, outVer)
		}
	}
}
