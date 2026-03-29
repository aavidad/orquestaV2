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
)

func TestRuntimeControlPlaneUsaAPICuandoHayServidor(t *testing.T) {
	var transcriptQuery string
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/api/runtime-transcript" && r.Method == http.MethodGet:
			transcriptQuery = r.URL.Query().Get("q")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"transcript": []map[string]any{{
					"id":              19,
					"runtime_id":      7,
					"agente":          "Codex2",
					"proyecto_slug":   "orquestador",
					"stream":          "pty_out",
					"text":            "He preparado el refactor del router",
					"normalized_text": "he preparado el refactor del router",
					"classification":  "ready_for_review",
					"created_at":      "2026-03-23T10:00:00Z",
				}},
			})
		case r.URL.Path == "/api/runtime-handles":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"handles": []map[string]any{{
					"id":          7,
					"agente":      "Codex1",
					"transporte":  "cli",
					"handle_kind": "session",
					"handle_ref":  "sess-001",
					"estado":      "activo",
					"created_at":  "2026-03-23T10:00:00Z",
					"updated_at":  "2026-03-23T10:00:00Z",
				}},
			})
		case r.URL.Path == "/api/runtime-handles/purgar" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":          true,
				"deleted":     2,
				"deleted_ids": []int64{48, 39},
				"estados":     []string{"cerrado", "fallido"},
			})
		case r.URL.Path == "/api/runtime-trace" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"trace": map[string]any{
					"handle_id":      7,
					"agente":         "Codex1",
					"proyecto":       "orquestador",
					"trace_dir":      "/repo/.orquesta-runtime/codex1/20260323-100000-000000001",
					"trace_manifest": "/repo/.orquesta-runtime/codex1/20260323-100000-000000001/runtime.json",
					"log_path":       "/repo/.orquesta-runtime/codex1/20260323-100000-000000001/pty.log",
					"working_dir":    "/repo",
					"driver":         "process_pty_cli",
					"available":      true,
					"total_bytes":    42,
					"bytes_read":     24,
					"truncated":      true,
					"raw_tail":       "linea uno\nlinea dos\n",
				},
			})
		case r.URL.Path == "/api/runtime-orders" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"orders": []map[string]any{{
					"id":             9,
					"agente":         "Codex1",
					"tipo":           "checkpoint",
					"payload_json":   "{}",
					"resultado_json": "{}",
					"error_text":     "",
					"estado":         "pendiente",
					"available_at":   "2026-03-23T10:00:00Z",
					"created_at":     "2026-03-23T10:00:00Z",
					"updated_at":     "2026-03-23T10:00:00Z",
				}},
			})
		case r.URL.Path == "/api/runtime-orders" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 15})
		case r.URL.Path == "/api/runtime-checkpoints" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"checkpoints": []map[string]any{
					{
						"id":              31,
						"agente":          "Codex1",
						"checkpoint_kind": "manual",
						"resumen":         "checkpoint remoto 1",
						"branch":          "main",
						"cwd":             "/tmp/orquestador",
						"payload_json":    "{}",
						"resume_strategy": "resumen_y_payload",
						"source":          "api-listado-1",
						"created_at":      "2026-03-23T10:00:00Z",
					},
					{
						"id":              30,
						"agente":          "Codex1",
						"checkpoint_kind": "manual",
						"resumen":         "checkpoint remoto 2",
						"branch":          "main",
						"cwd":             "/tmp/orquestador",
						"payload_json":    "{}",
						"resume_strategy": "resumen_y_payload",
						"source":          "api-listado-2",
						"created_at":      "2026-03-23T09:00:00Z",
					},
				},
			})
		case r.URL.Path == "/api/runtime-checkpoints" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 16})
		case r.URL.Path == "/api/runtime-checkpoints/21":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"checkpoint": map[string]any{
					"id":              21,
					"agente":          "Codex1",
					"checkpoint_kind": "manual",
					"resumen":         "checkpoint remoto",
					"branch":          "main",
					"cwd":             "/tmp/orquestador",
					"payload_json":    "{}",
					"resume_strategy": "resumen_y_payload",
					"source":          "api-checkpoint",
					"created_at":      "2026-03-23T10:00:00Z",
				},
			})
		case r.URL.Path == "/api/runtime-mailbox" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"mailbox": []map[string]any{{
					"id":           21,
					"from_agente":  "Codex1",
					"to_agente":    "Codex2",
					"kind":         "handoff",
					"payload_json": `{"ok":true}`,
					"estado":       "pendiente",
					"created_at":   "2026-03-23T10:00:00Z",
				}},
			})
		case r.URL.Path == "/api/runtime-mailbox" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 22})
		case r.URL.Path == "/api/runtime-mailbox/21/entregar" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 21})
		case r.URL.Path == "/api/runtime-mailbox/21/consumir" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 21})
		case r.URL.Path == "/api/runtime-checkpoints/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"checkpoint": map[string]any{
					"id":              3,
					"agente":          "Codex1",
					"checkpoint_kind": "handoff_prepare",
					"resumen":         "seguir desde API",
					"branch":          "main",
					"cwd":             "/tmp/orquestador",
					"payload_json":    "{}",
					"resume_strategy": "resumen_y_payload",
					"source":          "api",
					"created_at":      "2026-03-23T10:00:00Z",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	outHandles := capturarStdout(t, func() {
		if err := runtimeHandlesCmd.RunE(runtimeHandlesCmd, nil); err != nil {
			t.Fatalf("runtime handles via api: %v", err)
		}
	})
	if !strings.Contains(outHandles, "sess-001") {
		t.Fatalf("salida handles sin datos del servidor:\n%s", outHandles)
	}

	if err := runtimePurgarHandlesCmd.Flags().Set("agente", "Codex5"); err != nil {
		t.Fatalf("set agente purga handles: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimePurgarHandlesCmd.Flags().Set("agente", "")
	})
	outPurge := capturarStdout(t, func() {
		if err := runtimePurgarHandlesCmd.RunE(runtimePurgarHandlesCmd, nil); err != nil {
			t.Fatalf("runtime purgar-handles via api: %v", err)
		}
	})
	for _, token := range []string{"Purgados 2", "cerrado,fallido", "[48 39]"} {
		if !strings.Contains(outPurge, token) {
			t.Fatalf("salida purgar-handles sin %q:\n%s", token, outPurge)
		}
	}

	if err := runtimeTrazaCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente traza: %v", err)
	}
	if err := runtimeTrazaCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto traza: %v", err)
	}
	if err := runtimeTrazaCmd.Flags().Set("bytes", "32"); err != nil {
		t.Fatalf("set bytes traza: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeTrazaCmd.Flags().Set("agente", "")
		_ = runtimeTrazaCmd.Flags().Set("proyecto", "")
		_ = runtimeTrazaCmd.Flags().Set("bytes", "8192")
	})
	outTraza := capturarStdout(t, func() {
		if err := runtimeTrazaCmd.RunE(runtimeTrazaCmd, nil); err != nil {
			t.Fatalf("runtime traza via api: %v", err)
		}
	})
	for _, token := range []string{"Codex1", "process_pty_cli", "pty.log", "linea dos"} {
		if !strings.Contains(outTraza, token) {
			t.Fatalf("salida traza sin %q:\n%s", token, outTraza)
		}
	}

	if err := runtimeTranscriptCmd.Flags().Set("agente", "Codex2"); err != nil {
		t.Fatalf("set agente transcript: %v", err)
	}
	if err := runtimeTranscriptCmd.Flags().Set("q", "refactor"); err != nil {
		t.Fatalf("set q transcript: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeTranscriptCmd.Flags().Set("agente", "")
		_ = runtimeTranscriptCmd.Flags().Set("q", "")
	})
	outTranscript := capturarStdout(t, func() {
		if err := runtimeTranscriptCmd.RunE(runtimeTranscriptCmd, nil); err != nil {
			t.Fatalf("runtime transcript via api: %v", err)
		}
	})
	if transcriptQuery != "refactor" {
		t.Fatalf("query transcript inesperada: %q", transcriptQuery)
	}
	if !strings.Contains(outTranscript, "refactor del router") {
		t.Fatalf("salida transcript sin datos del servidor:\n%s", outTranscript)
	}

	outOrders := capturarStdout(t, func() {
		if err := runtimeOrdenesCmd.RunE(runtimeOrdenesCmd, nil); err != nil {
			t.Fatalf("runtime ordenes via api: %v", err)
		}
	})
	if !strings.Contains(outOrders, "checkpoint") {
		t.Fatalf("salida ordenes sin datos del servidor:\n%s", outOrders)
	}

	outCreate := capturarStdout(t, func() {
		if err := runtimeOrdenNuevaCmd.RunE(runtimeOrdenNuevaCmd, []string{"Codex1", "checkpoint"}); err != nil {
			t.Fatalf("runtime orden-nueva via api: %v", err)
		}
	})
	if !strings.Contains(outCreate, "#15") {
		t.Fatalf("salida orden-nueva sin id remoto:\n%s", outCreate)
	}

	outNudge := capturarStdout(t, func() {
		if err := runtimeNudgeCmd.RunE(runtimeNudgeCmd, []string{"Codex2", "revisa", "el", "bloqueo"}); err != nil {
			t.Fatalf("runtime nudge via api: %v", err)
		}
	})
	if !strings.Contains(outNudge, "#15") {
		t.Fatalf("salida nudge sin id remoto:\n%s", outNudge)
	}

	outDiscordia := capturarStdout(t, func() {
		if err := runtimeDiscordiaCmd.RunE(runtimeDiscordiaCmd, []string{"alberto", "Codex2", "desacuerdo", "tecnico"}); err != nil {
			t.Fatalf("runtime discordia via api: %v", err)
		}
	})
	if !strings.Contains(outDiscordia, "#15") {
		t.Fatalf("salida discordia sin id remoto:\n%s", outDiscordia)
	}

	if err := runtimeCheckpointsCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente checkpoint: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeCheckpointsCmd.Flags().Set("agente", "")
	})
	outCheckpoint := capturarStdout(t, func() {
		if err := runtimeCheckpointsCmd.RunE(runtimeCheckpointsCmd, nil); err != nil {
			t.Fatalf("runtime checkpoints via api: %v", err)
		}
	})
	if !strings.Contains(outCheckpoint, "seguir desde API") {
		t.Fatalf("salida checkpoints sin datos del servidor:\n%s", outCheckpoint)
	}

	outCheckpointCrear := capturarStdout(t, func() {
		if err := runtimeCheckpointNuevoCmd.RunE(runtimeCheckpointNuevoCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("runtime checkpoint-nuevo via api: %v", err)
		}
	})
	if !strings.Contains(outCheckpointCrear, "#16") {
		t.Fatalf("salida checkpoint-nuevo sin id remoto:\n%s", outCheckpointCrear)
	}

	if err := runtimeCheckpointsCmd.Flags().Set("kind", "manual"); err != nil {
		t.Fatalf("set kind checkpoints: %v", err)
	}
	if err := runtimeCheckpointsCmd.Flags().Set("limit", "2"); err != nil {
		t.Fatalf("set limit checkpoints: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeCheckpointsCmd.Flags().Set("kind", "")
		_ = runtimeCheckpointsCmd.Flags().Set("limit", "1")
	})
	outCheckpointHistorial := capturarStdout(t, func() {
		if err := runtimeCheckpointsCmd.RunE(runtimeCheckpointsCmd, nil); err != nil {
			t.Fatalf("runtime checkpoints historial via api: %v", err)
		}
	})
	for _, token := range []string{"api-listado-1", "api-listado-2", "manual"} {
		if !strings.Contains(outCheckpointHistorial, token) {
			t.Fatalf("salida checkpoints historial sin %q:\n%s", token, outCheckpointHistorial)
		}
	}

	outCheckpointVer := capturarStdout(t, func() {
		if err := runtimeCheckpointVerCmd.RunE(runtimeCheckpointVerCmd, []string{"21"}); err != nil {
			t.Fatalf("runtime checkpoint-ver via api: %v", err)
		}
	})
	for _, token := range []string{"Checkpoint #21", "Codex1", "checkpoint remoto", "api-checkpoint"} {
		if !strings.Contains(outCheckpointVer, token) {
			t.Fatalf("salida checkpoint-ver sin %q:\n%s", token, outCheckpointVer)
		}
	}

	if err := runtimeMailboxCmd.Flags().Set("to", "Codex2"); err != nil {
		t.Fatalf("set to mailbox: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeMailboxCmd.Flags().Set("to", "")
	})
	outMailbox := capturarStdout(t, func() {
		if err := runtimeMailboxCmd.RunE(runtimeMailboxCmd, nil); err != nil {
			t.Fatalf("runtime mailbox via api: %v", err)
		}
	})
	if !strings.Contains(outMailbox, "handoff") {
		t.Fatalf("salida mailbox sin datos del servidor:\n%s", outMailbox)
	}

	outMailboxCrear := capturarStdout(t, func() {
		if err := runtimeMailboxEnviarCmd.RunE(runtimeMailboxEnviarCmd, []string{"Codex1", "Codex2", "handoff"}); err != nil {
			t.Fatalf("runtime mailbox-enviar via api: %v", err)
		}
	})
	if !strings.Contains(outMailboxCrear, "#22") {
		t.Fatalf("salida mailbox-enviar sin id remoto:\n%s", outMailboxCrear)
	}

	outMailboxEntregar := capturarStdout(t, func() {
		if err := runtimeMailboxEntregarCmd.RunE(runtimeMailboxEntregarCmd, []string{"21"}); err != nil {
			t.Fatalf("runtime mailbox-entregar via api: %v", err)
		}
	})
	if !strings.Contains(outMailboxEntregar, "entregado") {
		t.Fatalf("salida mailbox-entregar inesperada:\n%s", outMailboxEntregar)
	}

	outMailboxConsumir := capturarStdout(t, func() {
		if err := runtimeMailboxConsumirCmd.RunE(runtimeMailboxConsumirCmd, []string{"21"}); err != nil {
			t.Fatalf("runtime mailbox-consumir via api: %v", err)
		}
	})
	if !strings.Contains(outMailboxConsumir, "consumido") {
		t.Fatalf("salida mailbox-consumir inesperada:\n%s", outMailboxConsumir)
	}
}

func TestRuntimeRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(runtimeListarCmd)
	err := runtimeListarCmd.RunE(runtimeListarCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}

	resetCommandFlags(runtimeTrazaCmd)
	_ = runtimeTrazaCmd.Flags().Set("agente", "Codex1")
	err = runtimeTrazaCmd.RunE(runtimeTrazaCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor en runtime traza")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado runtime traza: %v", err)
	}
}
