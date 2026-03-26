/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

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
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/diagnostico", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiDiagnosticoResponse{
			Diagnostico: db.SnapshotDiagnostico{
				GeneradoEn:         time.Date(2026, 3, 22, 21, 0, 0, 0, time.UTC),
				Agentes:            []*db.Agente{{Nombre: "Codex1", Activo: true}},
				ConteoTareas:       map[string]int{"en_progreso": 1},
				SesionesActivas:    []*db.Sesion{{ID: 1, Agente: "Codex1", Estado: "activa"}},
				PropuestasAbiertas: []db.PropuestaDiagnostico{},
				Config:             map[string]string{"agent_tick_seconds": "30"},
				AuditoriaReciente:  []db.AuditEntry{{Agente: "Codex1", Accion: "test", Entidad: "runtime", EntidadID: 1, CreatedAt: time.Date(2026, 3, 22, 21, 0, 0, 0, time.UTC)}},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(exportarDiagnosticoCmd)
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
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/diagnostico", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiDiagnosticoResponse{
			Diagnostico: db.SnapshotDiagnostico{
				GeneradoEn:   time.Date(2026, 3, 22, 21, 0, 0, 0, time.UTC),
				Agentes:      []*db.Agente{{Nombre: "Codex1", Activo: true}},
				ConteoTareas: map[string]int{"en_progreso": 1},
				Config:       map[string]string{"agent_tick_seconds": "30"},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(exportarDiagnosticoCmd)
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

func TestExportarDiagnosticoUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/diagnostico", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiDiagnosticoResponse{
			Diagnostico: db.SnapshotDiagnostico{
				GeneradoEn:         time.Date(2026, 3, 22, 21, 0, 0, 0, time.UTC),
				Agentes:            []*db.Agente{{Nombre: "Codex1", Activo: true}},
				ConteoTareas:       map[string]int{"en_progreso": 1},
				SesionesActivas:    []*db.Sesion{{ID: 1, Agente: "Codex1", Estado: "activa"}},
				PropuestasAbiertas: []db.PropuestaDiagnostico{},
				Config:             map[string]string{"agent_tick_seconds": "30"},
				AuditoriaReciente:  []db.AuditEntry{{Agente: "Codex1", Accion: "test", Entidad: "runtime", EntidadID: 1, CreatedAt: time.Date(2026, 3, 22, 21, 0, 0, 0, time.UTC)}},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(exportarDiagnosticoCmd)
	if err := exportarDiagnosticoCmd.Flags().Set("json", "false"); err != nil {
		t.Fatalf("set flag json: %v", err)
	}
	if err := exportarDiagnosticoCmd.Flags().Set("audit-limit", "5"); err != nil {
		t.Fatalf("set flag audit-limit: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := exportarDiagnosticoCmd.RunE(exportarDiagnosticoCmd, nil); err != nil {
			t.Fatalf("run diagnostico via API: %v", err)
		}
	})

	for _, token := range []string{"# Orquesta — Diagnóstico", "Agentes activos: 1", "## Configuración", "## Auditoría reciente"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida markdown via API sin %q:\n%s", token, out)
		}
	}
}
