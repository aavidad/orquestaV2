/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func capturarStdout(t *testing.T, fn func()) string {
	t.Helper()

	anterior := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = anterior
	}()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copiando stdout: %v", err)
	}
	_ = r.Close()
	return buf.String()
}

func TestExportarDiagnosticoMarkdown(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-diag",
		ResumenContinuidad: "diagnostico",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if err := exportarDiagnosticoCmd.Flags().Set("json", "false"); err != nil {
		t.Fatalf("set flag json: %v", err)
	}
	if err := exportarDiagnosticoCmd.Flags().Set("audit-limit", "5"); err != nil {
		t.Fatalf("set flag audit-limit: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := exportarDiagnosticoCmd.RunE(exportarDiagnosticoCmd, nil); err != nil {
			t.Fatalf("run diagnostico markdown: %v", err)
		}
	})

	for _, token := range []string{"# Orquesta — Diagnóstico", "## Resumen", "## Sesiones activas", "## Configuración", "## Auditoría reciente"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida markdown sin %q:\n%s", token, out)
		}
	}
}

func TestExportarDiagnosticoJSON(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := exportarDiagnosticoCmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set flag json: %v", err)
	}
	if err := exportarDiagnosticoCmd.Flags().Set("audit-limit", "3"); err != nil {
		t.Fatalf("set flag audit-limit: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := exportarDiagnosticoCmd.RunE(exportarDiagnosticoCmd, nil); err != nil {
			t.Fatalf("run diagnostico json: %v", err)
		}
	})

	for _, token := range []string{`"GeneradoEn"`, `"Agentes"`, `"ConteoTareas"`, `"Config"`} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida json sin %q:\n%s", token, out)
		}
	}
}
