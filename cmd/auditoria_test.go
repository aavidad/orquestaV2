package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestAuditoriaListarUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/audit", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiAuditResponse{
			Audit: []db.AuditEntry{
				{
					Agente:    "Codex1",
					Accion:    "refineria",
					Entidad:   "tarea",
					Detalle:   "detalle api auditoria",
					CreatedAt: time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC),
				},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := auditListarCmd.Flags().Set("limite", "10"); err != nil {
		t.Fatalf("set limite: %v", err)
	}
	if err := auditListarCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := auditListarCmd.Flags().Set("accion", ""); err != nil {
		t.Fatalf("set accion: %v", err)
	}
	if err := auditListarCmd.Flags().Set("entidad", ""); err != nil {
		t.Fatalf("set entidad: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := auditListarCmd.RunE(auditListarCmd, nil); err != nil {
			t.Fatalf("auditoria listar via API: %v", err)
		}
	})
	for _, token := range []string{"ID", "Codex1", "detalle api auditoria"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida auditoria sin %q:\n%s", token, out)
		}
	}
}
