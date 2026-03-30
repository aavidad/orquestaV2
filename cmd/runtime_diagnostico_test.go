package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestRuntimeDiagnosticoUsaAPI(t *testing.T) {
	now := time.Date(2026, 3, 24, 20, 0, 0, 0, time.UTC)
	runtime := &db.RuntimeInstance{
		ID:              7,
		Agente:          "Codex1",
		ProyectoSlug:    "orquestador",
		Provider:        "openai",
		Connector:       "codex-cli",
		LogicalState:    "vivo",
		Branch:          "main",
		LastHeartbeatAt: &now,
	}
	handle := &db.RuntimeHandle{
		ID:         9,
		Agente:     "Codex1",
		Transporte: "stdio",
		HandleKind: "session",
		HandleRef:  "sess-runtime-api",
		Estado:     "vivo",
		LastSeenAt: &now,
	}
	order := &db.RuntimeOrder{
		ID:        11,
		Agente:    "Codex1",
		Tipo:      "checkpoint",
		Estado:    "pendiente",
		CreatedAt: now,
	}
	msg := &db.RuntimeMailboxMessage{
		ID:         13,
		FromAgente: "Supervisor",
		ToAgente:   "Codex1",
		Kind:       "nudge",
		Estado:     "pendiente",
		CreatedAt:  now,
	}
	cp := &db.RuntimeCheckpoint{
		ID:             17,
		Agente:         "Codex1",
		CheckpointKind: "manual",
		Branch:         "main",
		Source:         "api-test",
		CreatedAt:      now,
		Resumen:        "checkpoint api",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/runtimes/tree", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeTreeResponse{
			Runtimes: []*apiRuntimeTreeNode{{Runtime: runtime}},
		})
	})
	mux.HandleFunc("/api/runtime-handles", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeHandlesResponse{Handles: []*db.RuntimeHandle{handle}})
	})
	mux.HandleFunc("/api/runtime-orders", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeOrdersResponse{Orders: []*db.RuntimeOrder{order}})
	})
	mux.HandleFunc("/api/runtime-mailbox", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeMailboxResponse{Mailbox: []*db.RuntimeMailboxMessage{msg}})
	})
	mux.HandleFunc("/api/runtime-checkpoints", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeCheckpointsResponse{Checkpoints: []*db.RuntimeCheckpoint{cp}})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := runtimeDiagnosticoCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := runtimeDiagnosticoCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto: %v", err)
	}
	if err := runtimeDiagnosticoCmd.Flags().Set("limit", "2"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeDiagnosticoCmd.Flags().Set("agente", "")
		_ = runtimeDiagnosticoCmd.Flags().Set("proyecto", "")
		_ = runtimeDiagnosticoCmd.Flags().Set("limit", "5")
	})

	out := capturarStdout(t, func() {
		if err := runtimeDiagnosticoCmd.RunE(runtimeDiagnosticoCmd, nil); err != nil {
			t.Fatalf("run runtime diagnostico api: %v", err)
		}
	})

	for _, token := range []string{
		"Agente:     Codex1",
		"Fuente:     api",
		"openai",
		"sess-runtime-api",
		"checkpoint api",
		"Mailbox pendiente",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida diagnostico api sin %q:\n%s", token, out)
		}
	}
}

func TestRuntimeDiagnosticoRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(runtimeDiagnosticoCmd)
	_ = runtimeDiagnosticoCmd.Flags().Set("agente", "Codex1")
	err := runtimeDiagnosticoCmd.RunE(runtimeDiagnosticoCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}
