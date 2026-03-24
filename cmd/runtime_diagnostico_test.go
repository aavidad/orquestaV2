package cmd

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestRuntimeDiagnosticoLocal(t *testing.T) {
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
		ExternalSessionID:  "sess-runtime-diag",
		ResumenContinuidad: "runtime diagnostico",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert runtime handle: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"motivo":"diag"}`,
	}); err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "Supervisor",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"sigue"}`,
	}); err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if _, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		SesionID:       &sesion.ID,
		CheckpointKind: "manual",
		Resumen:        "checkpoint local",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"ok":true}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "runtime-diag-test",
	}); err != nil {
		t.Fatalf("crear checkpoint: %v", err)
	}

	if err := runtimeDiagnosticoCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := runtimeDiagnosticoCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto: %v", err)
	}
	if err := runtimeDiagnosticoCmd.Flags().Set("limit", "3"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeDiagnosticoCmd.Flags().Set("agente", "")
		_ = runtimeDiagnosticoCmd.Flags().Set("proyecto", "")
		_ = runtimeDiagnosticoCmd.Flags().Set("limit", "5")
	})

	out := capturarStdout(t, func() {
		if err := runtimeDiagnosticoCmd.RunE(runtimeDiagnosticoCmd, nil); err != nil {
			t.Fatalf("run runtime diagnostico local: %v", err)
		}
	})

	for _, token := range []string{
		"DIAGNÓSTICO RUNTIME",
		"Agente:     Codex1",
		"Fuente:     local",
		"Runtimes",
		"Handles",
		"Órdenes relevantes",
		"Mailbox pendiente",
		"Checkpoints recientes",
		"checkpoint local",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida diagnostico local sin %q:\n%s", token, out)
		}
	}
}

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
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
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
