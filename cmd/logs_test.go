/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestLogsUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/audit", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiAuditResponse{
			Audit: []db.AuditEntry{
				{
					Agente:    "Codex1",
					Accion:    "test",
					Entidad:   "runtime",
					EntidadID: 12,
					Detalle:   "detalle api",
					CreatedAt: time.Date(2026, 3, 22, 22, 0, 0, 0, time.UTC),
				},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := logsCmd.Flags().Set("limit", "10"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	if err := logsCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := logsCmd.Flags().Set("accion", ""); err != nil {
		t.Fatalf("set accion: %v", err)
	}
	if err := logsCmd.Flags().Set("entidad", ""); err != nil {
		t.Fatalf("set entidad: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := logsCmd.RunE(logsCmd, nil); err != nil {
			t.Fatalf("logs via API: %v", err)
		}
	})
	for _, token := range []string{"FECHA", "Codex1", "detalle api"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida logs sin %q:\n%s", token, out)
		}
	}
}

func TestLogsLocalFiltraAntesDeAplicarLimite(t *testing.T) {
	prepararDBTemporalCmd(t)

	db.Audit("Codex2", "accion-reciente", "runtime", 22, "detalle reciente")
	db.Audit("Codex1", "accion-filtrada", "runtime", 21, "detalle esperado")

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "1")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := logsCmd.Flags().Set("limit", "1"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	if err := logsCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := logsCmd.Flags().Set("accion", ""); err != nil {
		t.Fatalf("set accion: %v", err)
	}
	if err := logsCmd.Flags().Set("entidad", ""); err != nil {
		t.Fatalf("set entidad: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := logsCmd.RunE(logsCmd, nil); err != nil {
			t.Fatalf("logs local: %v", err)
		}
	})

	if !strings.Contains(out, "Codex1") || !strings.Contains(out, "detalle esperado") {
		t.Fatalf("logs local no aplico bien filtro+limite:\n%s", out)
	}
	if strings.Contains(out, "Codex2") {
		t.Fatalf("logs local devolvio una entrada fuera del filtro:\n%s", out)
	}
}
