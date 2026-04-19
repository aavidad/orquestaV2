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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestRuntimeControlPlaneUsaAPICuandoHayServidor(t *testing.T) {
	var transcriptQuery string
	var transcriptRuntimeID string
	var transcriptHandleID string
	var ordersProjectQuery string
	var ordersLimitQuery string
	var wakeReq map[string]any
	var processOrdersReq map[string]any
	var processMailboxReq map[string]any
	var processAutonomiaReq map[string]any
	var processTranscriptReq map[string]any
	var purgeTranscriptNoiseReq map[string]any
	var processDegradadosReq map[string]any
	var processHygieneReq map[string]any
	var processReanimationsReq map[string]any
	var clearMailboxReq map[string]any
	var clearTasksReq map[string]any
	var purgeOrdersReq map[string]any
	var purgeHandlesReq map[string]any
	var closeHandlesReq map[string]any
	var closeRuntimesReq map[string]any
	var agentControlReq map[string]any
	runtimeActive := true
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/api/runtimes/tree" && r.Method == http.MethodGet:
			runtimes := []map[string]any{}
			if runtimeActive {
				runtimes = append(runtimes, map[string]any{
					"runtime": map[string]any{
						"id":             7,
						"agente":         "Codex2",
						"proyecto_slug":  "orquestador",
						"provider_slug":  "openai",
						"connector_slug": "codex-cli",
						"logical_state":  "activo",
						"pid":            1234,
						"last_event_at":  "2026-03-23T10:00:00Z",
					},
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"runtimes": runtimes})
		case r.URL.Path == "/api/runtime-transcript" && r.Method == http.MethodGet:
			transcriptQuery = r.URL.Query().Get("q")
			transcriptRuntimeID = r.URL.Query().Get("runtime_id")
			transcriptHandleID = r.URL.Query().Get("handle_id")
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
			handles := []map[string]any{}
			if runtimeActive {
				handles = append(handles, map[string]any{
					"id":          7,
					"agente":      "Codex2",
					"transporte":  "cli",
					"handle_kind": "session",
					"handle_ref":  "sess-001",
					"estado":      "activo",
					"created_at":  "2026-03-23T10:00:00Z",
					"updated_at":  "2026-03-23T10:00:00Z",
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"handles": handles})
		case r.URL.Path == "/api/runtime-handles/purgar" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&purgeHandlesReq)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":          true,
				"deleted":     2,
				"deleted_ids": []int64{48, 39},
				"estados":     []string{"cerrado", "fallido"},
			})
		case r.URL.Path == "/api/runtime-handles/cerrar" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&closeHandlesReq)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":     true,
				"closed": true,
			})
		case r.URL.Path == "/api/runtimes/cerrar" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&closeRuntimesReq)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":         true,
				"closed":     1,
				"closed_ids": []int64{7},
			})
		case r.URL.Path == "/api/runtime-orders/purgar" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&purgeOrdersReq)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":          true,
				"deleted":     3,
				"deleted_ids": []int64{91, 88, 77},
				"estados":     []string{"completada", "fallida"},
				"tipos":       []string{"send_instruction"},
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
			ordersProjectQuery = r.URL.Query().Get("proyecto")
			ordersLimitQuery = r.URL.Query().Get("limit")
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
		case r.URL.Path == "/api/runtime-mailbox/limpiar" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&clearMailboxReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "cleared": 1, "cleared_ids": []int64{21}})
		case r.URL.Path == "/api/agente/control" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&agentControlReq)
			if accion, _ := agentControlReq["accion"].(string); strings.TrimSpace(accion) == "stop" {
				runtimeActive = false
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 99, "agente": "Codex2", "accion": "stop"})
		case r.URL.Path == "/api/tareas/limpiar-frente" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&clearTasksReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "resultado": map[string]any{"total": 2, "moved_ids": []int64{44, 45}}})
		case r.URL.Path == "/api/runtime/wake" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&wakeReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"orders": true, "mailbox": true, "warm": false})
		case r.URL.Path == "/api/runtime/process-orders" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&processOrdersReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": 2})
		case r.URL.Path == "/api/runtime/process-mailbox" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&processMailboxReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": 3})
		case r.URL.Path == "/api/runtime/process-autonomia" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&processAutonomiaReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": 0, "accepted": true, "running": true})
		case r.URL.Path == "/api/runtime/process-transcript" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&processTranscriptReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": 7, "accepted": false, "running": false})
		case r.URL.Path == "/api/runtime/purge-transcript-noise" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&purgeTranscriptNoiseReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": 4})
		case r.URL.Path == "/api/runtime/process-degradados" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&processDegradadosReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": 2, "accepted": true, "running": false})
		case r.URL.Path == "/api/runtime/process-hygiene" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&processHygieneReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": 5, "accepted": true, "running": false})
		case r.URL.Path == "/api/runtime/process-reanimations" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&processReanimationsReq)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": 1, "accepted": true, "running": false, "candidates": 2, "reactivated": 1, "cooldown_sustained": 1, "errors": 0})
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

	if err := runtimePurgarOrdenesCmd.Flags().Set("agente", "Codex5"); err != nil {
		t.Fatalf("set agente purga ordenes: %v", err)
	}
	if err := runtimePurgarOrdenesCmd.Flags().Set("tipo", "send_instruction"); err != nil {
		t.Fatalf("set tipo purga ordenes: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimePurgarOrdenesCmd.Flags().Set("agente", "")
		_ = runtimePurgarOrdenesCmd.Flags().Set("tipo", "")
	})
	outPurgeOrders := capturarStdout(t, func() {
		if err := runtimePurgarOrdenesCmd.RunE(runtimePurgarOrdenesCmd, nil); err != nil {
			t.Fatalf("runtime purgar-ordenes via api: %v", err)
		}
	})
	for _, token := range []string{"Purgadas 3", "completada,fallida", "tipos=send_instruction", "[91 88 77]"} {
		if !strings.Contains(outPurgeOrders, token) {
			t.Fatalf("salida purgar-ordenes sin %q:\n%s", token, outPurgeOrders)
		}
	}
	if got, ok := purgeOrdersReq["older_than_minutes"].(float64); !ok || int(got) != 60 {
		t.Fatalf("older_than_minutes inesperado en request: %+v", purgeOrdersReq)
	}

	if err := runtimeDespertarCmd.Flags().Set("orders", "true"); err != nil {
		t.Fatalf("set orders despertar: %v", err)
	}
	if err := runtimeDespertarCmd.Flags().Set("mailbox", "true"); err != nil {
		t.Fatalf("set mailbox despertar: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeDespertarCmd.Flags().Set("orders", "false")
		_ = runtimeDespertarCmd.Flags().Set("mailbox", "false")
		_ = runtimeDespertarCmd.Flags().Set("warm", "false")
	})
	outWake := capturarStdout(t, func() {
		if err := runtimeDespertarCmd.RunE(runtimeDespertarCmd, nil); err != nil {
			t.Fatalf("runtime despertar via api: %v", err)
		}
	})
	for _, token := range []string{"orders=true", "mailbox=true"} {
		if !strings.Contains(outWake, token) {
			t.Fatalf("salida runtime despertar sin %q:\n%s", token, outWake)
		}
	}
	if got, _ := wakeReq["orders"].(bool); !got {
		t.Fatalf("request wake sin orders=true: %+v", wakeReq)
	}
	if got, _ := wakeReq["mailbox"].(bool); !got {
		t.Fatalf("request wake sin mailbox=true: %+v", wakeReq)
	}
	if err := runtimeProcesarMailboxCmd.Flags().Set("agente", "Codex2"); err != nil {
		t.Fatalf("set agente procesar-mailbox: %v", err)
	}
	if err := runtimeProcesarMailboxCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto procesar-mailbox: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeProcesarMailboxCmd.Flags().Set("agente", "")
		_ = runtimeProcesarMailboxCmd.Flags().Set("proyecto", "")
	})
	outProcessMailbox := capturarStdout(t, func() {
		if err := runtimeProcesarMailboxCmd.RunE(runtimeProcesarMailboxCmd, nil); err != nil {
			t.Fatalf("runtime procesar-mailbox via api: %v", err)
		}
	})
	if !strings.Contains(outProcessMailbox, "count=3") {
		t.Fatalf("salida runtime procesar-mailbox inesperada:\n%s", outProcessMailbox)
	}
	if got, _ := processMailboxReq["to_agente"].(string); got != "Codex2" {
		t.Fatalf("request process mailbox sin to_agente esperado: %+v", processMailboxReq)
	}
	if got, _ := processMailboxReq["proyecto"].(string); got != "orquestador" {
		t.Fatalf("request process mailbox sin proyecto esperado: %+v", processMailboxReq)
	}

	outProcessOrders := capturarStdout(t, func() {
		if err := runtimeProcesarOrdenesCmd.RunE(runtimeProcesarOrdenesCmd, nil); err != nil {
			t.Fatalf("runtime procesar-ordenes via api: %v", err)
		}
	})
	if !strings.Contains(outProcessOrders, "count=2") {
		t.Fatalf("salida runtime procesar-ordenes inesperada:\n%s", outProcessOrders)
	}
	if len(processOrdersReq) != 0 {
		t.Fatalf("request process orders deberia ser vacia: %+v", processOrdersReq)
	}

	outProcessAutonomia := capturarStdout(t, func() {
		if err := runtimeProcesarAutonomiaCmd.RunE(runtimeProcesarAutonomiaCmd, nil); err != nil {
			t.Fatalf("runtime procesar-autonomia via api: %v", err)
		}
	})
	for _, token := range []string{"accepted=true", "running=true", "count=0"} {
		if !strings.Contains(outProcessAutonomia, token) {
			t.Fatalf("salida runtime procesar-autonomia sin %q:\n%s", token, outProcessAutonomia)
		}
	}
	if got, ok := processAutonomiaReq["wait"].(bool); !ok || got {
		t.Fatalf("request process autonomia deberia enviar wait=false: %+v", processAutonomiaReq)
	}

	outProcessTranscript := capturarStdout(t, func() {
		if err := runtimeProcesarTranscriptCmd.RunE(runtimeProcesarTranscriptCmd, nil); err != nil {
			t.Fatalf("runtime procesar-transcript via api: %v", err)
		}
	})
	for _, token := range []string{"accepted=false", "running=false", "count=7"} {
		if !strings.Contains(outProcessTranscript, token) {
			t.Fatalf("salida runtime procesar-transcript sin %q:\n%s", token, outProcessTranscript)
		}
	}
	if got, ok := processTranscriptReq["wait"].(bool); !ok || got {
		t.Fatalf("request process transcript deberia enviar wait=false: %+v", processTranscriptReq)
	}

	outPurgeTranscriptNoise := capturarStdout(t, func() {
		if err := runtimePurgarTranscriptRuidoCmd.RunE(runtimePurgarTranscriptRuidoCmd, nil); err != nil {
			t.Fatalf("runtime purgar-transcript-ruido via api: %v", err)
		}
	})
	if !strings.Contains(outPurgeTranscriptNoise, "count=4") {
		t.Fatalf("salida runtime purgar-transcript-ruido inesperada:\n%s", outPurgeTranscriptNoise)
	}
	if len(purgeTranscriptNoiseReq) != 0 {
		t.Fatalf("request purge transcript noise deberia llevar payload: %+v", purgeTranscriptNoiseReq)
	}
	if got, ok := purgeTranscriptNoiseReq["all"].(bool); !ok || !got {
		t.Fatalf("request purge transcript noise deberia enviar all=true: %+v", purgeTranscriptNoiseReq)
	}

	outProcessDegradados := capturarStdout(t, func() {
		if err := runtimeProcesarDegradadosCmd.RunE(runtimeProcesarDegradadosCmd, nil); err != nil {
			t.Fatalf("runtime procesar-degradados via api: %v", err)
		}
	})
	for _, token := range []string{"accepted=true", "running=false", "count=2"} {
		if !strings.Contains(outProcessDegradados, token) {
			t.Fatalf("salida runtime procesar-degradados sin %q:\n%s", token, outProcessDegradados)
		}
	}
	if got, ok := processDegradadosReq["wait"].(bool); !ok || got {
		t.Fatalf("request process degradados deberia enviar wait=false: %+v", processDegradadosReq)
	}

	outProcessHygiene := capturarStdout(t, func() {
		if err := runtimeProcesarHigieneCmd.RunE(runtimeProcesarHigieneCmd, nil); err != nil {
			t.Fatalf("runtime procesar-higiene via api: %v", err)
		}
	})
	for _, token := range []string{"accepted=true", "running=false", "count=5"} {
		if !strings.Contains(outProcessHygiene, token) {
			t.Fatalf("salida runtime procesar-higiene sin %q:\n%s", token, outProcessHygiene)
		}
	}
	if got, ok := processHygieneReq["wait"].(bool); !ok || got {
		t.Fatalf("request process hygiene deberia enviar wait=false: %+v", processHygieneReq)
	}

	outProcessReanimations := capturarStdout(t, func() {
		if err := runtimeProcesarReanimacionesCmd.RunE(runtimeProcesarReanimacionesCmd, nil); err != nil {
			t.Fatalf("runtime procesar-reanimaciones via api: %v", err)
		}
	})
	for _, token := range []string{"accepted=true", "running=false", "count=1", "candidates=2", "reactivated=1", "cooldown_sustained=1", "errors=0"} {
		if !strings.Contains(outProcessReanimations, token) {
			t.Fatalf("salida runtime procesar-reanimaciones sin %q:\n%s", token, outProcessReanimations)
		}
	}
	if len(processReanimationsReq) != 0 {
		t.Fatalf("request process reanimations deberia ser vacia: %+v", processReanimationsReq)
	}

	if err := runtimeMailboxLimpiarCmd.Flags().Set("to", "Codex2"); err != nil {
		t.Fatalf("set to mailbox limpiar: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeMailboxLimpiarCmd.Flags().Set("to", "")
	})
	outMailboxClear := capturarStdout(t, func() {
		if err := runtimeMailboxLimpiarCmd.RunE(runtimeMailboxLimpiarCmd, nil); err != nil {
			t.Fatalf("runtime mailbox-limpiar via api: %v", err)
		}
	})
	if !strings.Contains(outMailboxClear, "Mailbox limpiado: 1") {
		t.Fatalf("salida runtime mailbox-limpiar inesperada:\n%s", outMailboxClear)
	}
	if got, _ := clearMailboxReq["to_agente"].(string); got != "Codex2" {
		t.Fatalf("request mailbox limpiar sin to_agente esperado: %+v", clearMailboxReq)
	}

	if err := runtimeLimpiarPruebasCmd.Flags().Set("agente", "Codex2"); err != nil {
		t.Fatalf("set agente limpiar-pruebas: %v", err)
	}
	if err := runtimeLimpiarPruebasCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto limpiar-pruebas: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeLimpiarPruebasCmd.Flags().Set("agente", "")
		_ = runtimeLimpiarPruebasCmd.Flags().Set("proyecto", "")
		_ = runtimeLimpiarPruebasCmd.Flags().Set("mantener-tareas", "")
	})
	outClean := capturarStdout(t, func() {
		if err := runtimeLimpiarPruebasCmd.RunE(runtimeLimpiarPruebasCmd, nil); err != nil {
			t.Fatalf("runtime limpiar-pruebas via api: %v", err)
		}
	})
	if !strings.Contains(outClean, "Entorno de prueba limpiado proyecto=orquestador agente=Codex2") {
		t.Fatalf("salida runtime limpiar-pruebas inesperada:\n%s", outClean)
	}
	if got, _ := clearTasksReq["agente"].(string); got != "Codex2" {
		t.Fatalf("request limpiar-frente sin agente esperado: %+v", clearTasksReq)
	}
	if got, _ := clearTasksReq["proyecto"].(string); got != "orquestador" {
		t.Fatalf("request limpiar-frente sin proyecto esperado: %+v", clearTasksReq)
	}
	if got, _ := purgeHandlesReq["agente"].(string); got != "Codex2" {
		t.Fatalf("request purgar-handles sin agente esperado: %+v", purgeHandlesReq)
	}
	if got, _ := closeHandlesReq["agente"].(string); got != "Codex2" {
		t.Fatalf("request cerrar-handles sin agente esperado: %+v", closeHandlesReq)
	}
	if got, _ := closeHandlesReq["proyecto"].(string); got != "orquestador" {
		t.Fatalf("request cerrar-handles sin proyecto esperado: %+v", closeHandlesReq)
	}
	if got, _ := closeRuntimesReq["agente"].(string); got != "Codex2" {
		t.Fatalf("request cerrar-runtimes sin agente esperado: %+v", closeRuntimesReq)
	}
	if got, _ := closeRuntimesReq["proyecto"].(string); got != "orquestador" {
		t.Fatalf("request cerrar-runtimes sin proyecto esperado: %+v", closeRuntimesReq)
	}
	if got, _ := agentControlReq["agente"].(string); got != "Codex2" {
		t.Fatalf("request agente control sin agente esperado: %+v", agentControlReq)
	}
	if got, _ := agentControlReq["accion"].(string); got != "stop" {
		t.Fatalf("request agente control sin stop esperado: %+v", agentControlReq)
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
	if err := runtimeTranscriptCmd.Flags().Set("runtime-id", "77"); err != nil {
		t.Fatalf("set runtime-id transcript: %v", err)
	}
	if err := runtimeTranscriptCmd.Flags().Set("handle-id", "88"); err != nil {
		t.Fatalf("set handle-id transcript: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeTranscriptCmd.Flags().Set("agente", "")
		_ = runtimeTranscriptCmd.Flags().Set("q", "")
		_ = runtimeTranscriptCmd.Flags().Set("runtime-id", "0")
		_ = runtimeTranscriptCmd.Flags().Set("handle-id", "0")
	})
	outTranscript := capturarStdout(t, func() {
		if err := runtimeTranscriptCmd.RunE(runtimeTranscriptCmd, nil); err != nil {
			t.Fatalf("runtime transcript via api: %v", err)
		}
	})
	if transcriptQuery != "refactor" {
		t.Fatalf("query transcript inesperada: %q", transcriptQuery)
	}
	if transcriptRuntimeID != "77" {
		t.Fatalf("runtime_id transcript inesperado: %q", transcriptRuntimeID)
	}
	if transcriptHandleID != "88" {
		t.Fatalf("handle_id transcript inesperado: %q", transcriptHandleID)
	}
	if !strings.Contains(outTranscript, "refactor del router") {
		t.Fatalf("salida transcript sin datos del servidor:\n%s", outTranscript)
	}

	if err := runtimeOrdenesCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto ordenes: %v", err)
	}
	if err := runtimeOrdenesCmd.Flags().Set("limit", "5"); err != nil {
		t.Fatalf("set limit ordenes: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeOrdenesCmd.Flags().Set("proyecto", "")
		_ = runtimeOrdenesCmd.Flags().Set("limit", "0")
	})
	outOrders := capturarStdout(t, func() {
		if err := runtimeOrdenesCmd.RunE(runtimeOrdenesCmd, nil); err != nil {
			t.Fatalf("runtime ordenes via api: %v", err)
		}
	})
	if ordersProjectQuery != "orquestador" {
		t.Fatalf("query proyecto ordenes inesperada: %q", ordersProjectQuery)
	}
	if ordersLimitQuery != "5" {
		t.Fatalf("query limit ordenes inesperada: %q", ordersLimitQuery)
	}
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

func TestWakeControlPlaneRuntimeMailboxReseteaThrottleDeReevaluacion(t *testing.T) {
	resetRuntimeMailboxReevaluationGate()

	if !runtimeMailboxShouldReevaluate("session_resume", 41, 33) {
		t.Fatalf("primer intento deberia permitirse")
	}
	if runtimeMailboxShouldReevaluate("session_resume", 41, 33) {
		t.Fatalf("segundo intento inmediato deberia quedar throttled")
	}
	if !wakeControlPlaneRuntimeMailbox() {
		t.Fatalf("el wake del carril mailbox deberia aceptarse")
	}
	if !runtimeMailboxShouldReevaluate("session_resume", 41, 33) {
		t.Fatalf("el wake mailbox deberia limpiar el throttle para reevaluar de inmediato")
	}
}

func TestRuntimeTranscriptValidaIDs(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(runtimeTranscriptCmd)
	if err := runtimeTranscriptCmd.Flags().Set("runtime-id", "-1"); err != nil {
		t.Fatalf("set runtime-id invalido: %v", err)
	}
	err := runtimeTranscriptCmd.RunE(runtimeTranscriptCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--runtime-id inválido") {
		t.Fatalf("error inesperado runtime-id inválido: %v", err)
	}

	resetCommandFlags(runtimeTranscriptCmd)
	if err := runtimeTranscriptCmd.Flags().Set("handle-id", "-2"); err != nil {
		t.Fatalf("set handle-id invalido: %v", err)
	}
	err = runtimeTranscriptCmd.RunE(runtimeTranscriptCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--handle-id inválido") {
		t.Fatalf("error inesperado handle-id inválido: %v", err)
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

func TestRuntimeOrdenesPermiteRecuperacionConForceLocal(t *testing.T) {
	prepararDBTemporalCmd(t)
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL", "1")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex1: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		Tipo:        "nudge",
		PayloadJSON: `{"texto":"seguir"}`,
	}); err != nil {
		t.Fatalf("EncolarRuntimeOrder: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		Tipo:        "checkpoint",
		PayloadJSON: `{}`,
	}); err != nil {
		t.Fatalf("EncolarRuntimeOrder segunda: %v", err)
	}

	resetCommandFlags(runtimeOrdenesCmd)
	if err := runtimeOrdenesCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente runtime ordenes: %v", err)
	}
	if err := runtimeOrdenesCmd.Flags().Set("limit", "1"); err != nil {
		t.Fatalf("set limit runtime ordenes: %v", err)
	}
	out := capturarStdout(t, func() {
		if err := runtimeOrdenesCmd.RunE(runtimeOrdenesCmd, nil); err != nil {
			t.Fatalf("runtime ordenes en recuperacion local: %v", err)
		}
	})
	if !strings.Contains(out, "checkpoint") || strings.Contains(out, "nudge") {
		t.Fatalf("salida runtime ordenes no respeta limit local:\n%s", out)
	}
}

func TestImprimirRuntimeOrdersMuestraDetalleDeferido(t *testing.T) {
	out := capturarStdout(t, func() {
		err := imprimirRuntimeOrders([]*db.RuntimeOrder{{
			ID:          77,
			Agente:      "Codex1",
			Tipo:        "send_instruction",
			Estado:      "pendiente",
			ErrorText:   "primer intento\nruntime no disponible",
			AvailableAt: time.Date(2026, 3, 30, 20, 30, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, 3, 30, 20, 15, 0, 0, time.UTC),
			ResultadoJSON: `{
				"deferred": true,
				"deferred_reason": "sin runtime activo entregable",
				"retry_after": "2026-03-30T20:30:00Z"
			}`,
		}})
		if err != nil {
			t.Fatalf("imprimirRuntimeOrders: %v", err)
		}
	})
	for _, token := range []string{"DISPONIBLE", "DETALLE", "sin runtime activo entregable", "2026-03-30 20:30:00"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida runtime ordenes sin %q:\n%s", token, out)
		}
	}
}

func TestImprimirRuntimeTranscriptOcultaRuidoOperativoPorDefecto(t *testing.T) {
	items := []*db.RuntimeTranscriptEntry{
		{ID: 1, Agente: "Codex2", Stream: "pty_out", Text: "Perfil activo: Codex2", CreatedAt: time.Date(2026, 3, 31, 19, 0, 0, 0, time.UTC)},
		{ID: 2, Agente: "Codex2", Stream: "pty_out", Text: "He terminado el refactor del router", CreatedAt: time.Date(2026, 3, 31, 19, 1, 0, 0, time.UTC)},
	}
	out := capturarStdout(t, func() {
		if err := imprimirRuntimeTranscript(items, false); err != nil {
			t.Fatalf("imprimirRuntimeTranscript: %v", err)
		}
	})
	if strings.Contains(out, "Perfil activo") || !strings.Contains(out, "He terminado el refactor del router") {
		t.Fatalf("salida transcript filtrada inesperada:\n%s", out)
	}
}

func TestImprimirRuntimeTranscriptRawMantieneRuidoOperativo(t *testing.T) {
	items := []*db.RuntimeTranscriptEntry{
		{ID: 1, Agente: "Codex2", Stream: "pty_out", Text: "Perfil activo: Codex2", CreatedAt: time.Date(2026, 3, 31, 19, 0, 0, 0, time.UTC)},
	}
	out := capturarStdout(t, func() {
		if err := imprimirRuntimeTranscript(items, true); err != nil {
			t.Fatalf("imprimirRuntimeTranscript raw: %v", err)
		}
	})
	if !strings.Contains(out, "Perfil activo") {
		t.Fatalf("salida transcript raw sin banner:\n%s", out)
	}
}

func TestListarRuntimeHandlesSaneaPremiumContaminadoPorOllamaPool(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
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
	pid := int64(os.Getpid())
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "Claude1",
		ProyectoID:   &proyectoID,
		LogicalState: "pausado",
		ProcessState: "idle",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-claude1-legacy",
		"tmux_pane_id":          "%1",
		"rendered_command":      "claude-code",
		"external_session_id":   "ollama-pool-claude1-legacy-1",
		"mailbox_delivery_mode": "session_resume",
	})
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'pausado', ?, ?, CURRENT_TIMESTAMP)`,
		"Claude1", proyectoID, runtimeID, "tmux", "session", "orq-claude1-legacy/%1", string(metaJSON), `{"mailbox_delivery_mode":"session_resume"}`); err != nil {
		t.Fatalf("insert runtime handle: %v", err)
	}

	agente := "Claude1"
	handles, err := db.ListarRuntimeHandles(&agente)
	if err != nil {
		t.Fatalf("listar runtime handles: %v", err)
	}
	if len(handles) != 1 {
		t.Fatalf("handles inesperados: %+v", handles)
	}
	if got := strings.TrimSpace(handles[0].Estado); got != "fallido" {
		t.Fatalf("el handle contaminado deberia quedar fallido, got=%q handle=%+v", got, handles[0])
	}
	runtime, err := db.GetRuntime(runtimeID)
	if err != nil {
		t.Fatalf("get runtime: %v", err)
	}
	if runtime == nil || strings.TrimSpace(runtime.LogicalState) != "degradado" || strings.TrimSpace(runtime.ProcessState) != "missing" {
		t.Fatalf("runtime no degradado tras sanear handle contaminado: %+v", runtime)
	}
}

func TestSincronizarRuntimeHandleSupervisadoNoReanimaPremiumConCanalRoto(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
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
	fakeTmux := filepath.Join(tmp, "tmux")
	if err := os.WriteFile(fakeTmux, []byte("#!/usr/bin/env bash\nset -euo pipefail\nif [[ \"$1\" == \"has-session\" ]]; then exit 0; fi\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	pid := int64(os.Getpid())
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "Claude1",
		ProyectoID:   &proyectoID,
		LogicalState: "pausado",
		ProcessState: "idle",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                     "tmux_cli_session",
		"transport":                  "tmux",
		"tmux_command":               fakeTmux,
		"tmux_session":               "orq-claude1-live",
		"tmux_pane_id":               "%1",
		"working_dir":                workingDir,
		"rendered_command":           "claude-code",
		"wrapped_command":            exe,
		"external_session_id":        "ollama-pool-claude1-legacy-1",
		"mailbox_delivery_mode":      "session_resume",
		"pty_last_broken_pipe_error": "external_session_id incompatible with tmux premium runtime",
		"pty_last_broken_pipe_at":    time.Now().UTC().Format(time.RFC3339Nano),
	})
	res, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?, CURRENT_TIMESTAMP)`,
		"Claude1", proyectoID, runtimeID, "tmux", "session", "orq-claude1-live/%1", string(metaJSON), `{"mailbox_delivery_mode":"session_resume"}`)
	if err != nil {
		t.Fatalf("insert runtime handle: %v", err)
	}
	handleID, _ := res.LastInsertId()
	handle, err := db.GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get runtime handle: %v", err)
	}

	refreshed, runtime, _, err := db.SincronizarRuntimeHandleSupervisado(handle, nil, "test_broken_channel")
	if err != nil {
		t.Fatalf("sincronizar runtime handle: %v", err)
	}
	if refreshed == nil || strings.TrimSpace(refreshed.Estado) != "fallido" {
		t.Fatalf("el handle con canal roto no deberia reanimarse: %+v", refreshed)
	}
	if runtime == nil {
		runtime, err = db.GetRuntime(runtimeID)
		if err != nil {
			t.Fatalf("get runtime: %v", err)
		}
	}
	if runtime == nil || strings.TrimSpace(runtime.LogicalState) != "degradado" || strings.TrimSpace(runtime.ProcessState) != "missing" {
		t.Fatalf("runtime no degradado tras canal roto: %+v", runtime)
	}
}
