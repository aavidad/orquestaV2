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
	for _, token := range []string{"Runtime #", "Codex1", "codex-cli", "Muestras recientes"} {
		if !strings.Contains(outVer, token) {
			t.Fatalf("salida ver sin %q:\n%s", token, outVer)
		}
	}
	if !strings.Contains(outVer, "Estado:") {
		t.Fatalf("salida ver sin bloque de estado:\n%s", outVer)
	}
}

func TestRuntimeNudgeEncolaOrdenLocal(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := runtimeNudgeCmd.RunE(runtimeNudgeCmd, []string{"Codex2", "retoma", "el", "bloqueo"}); err != nil {
			t.Fatalf("runtime nudge local: %v", err)
		}
	})
	if !strings.Contains(out, "Nudge runtime #") {
		t.Fatalf("salida runtime nudge inesperada:\n%s", out)
	}

	agente := "Codex2"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" {
		t.Fatalf("orden nudge inesperada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, "\"texto\":\"retoma el bloqueo\"") {
		t.Fatalf("payload nudge inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestRuntimeCheckpointsHistorialLocal(t *testing.T) {
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
	if _, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		CheckpointKind: "manual",
		Resumen:        "checkpoint uno",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"n":1}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "manual-1",
	}); err != nil {
		t.Fatalf("crear checkpoint 1: %v", err)
	}
	if _, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		CheckpointKind: "manual",
		Resumen:        "checkpoint dos",
		Branch:         "feature/controlplane",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"n":2}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "manual-2",
	}); err != nil {
		t.Fatalf("crear checkpoint 2: %v", err)
	}

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "1")()
	if err := runtimeCheckpointsCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := runtimeCheckpointsCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto: %v", err)
	}
	if err := runtimeCheckpointsCmd.Flags().Set("limit", "2"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeCheckpointsCmd.Flags().Set("agente", "")
		_ = runtimeCheckpointsCmd.Flags().Set("proyecto", "")
		_ = runtimeCheckpointsCmd.Flags().Set("limit", "1")
	})

	out := capturarStdout(t, func() {
		if err := runtimeCheckpointsCmd.RunE(runtimeCheckpointsCmd, nil); err != nil {
			t.Fatalf("runtime checkpoints historial local: %v", err)
		}
	})
	for _, token := range []string{"AGENTE", "manual-1", "manual-2", "feature/control"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida checkpoints historial sin %q:\n%s", token, out)
		}
	}
}

func TestRuntimeCheckpointVerLocal(t *testing.T) {
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
	checkpointID, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		CheckpointKind: "manual",
		Resumen:        "checkpoint inspeccionable",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"paso":"revisar"}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "manual-ver",
	})
	if err != nil {
		t.Fatalf("crear checkpoint: %v", err)
	}

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "1")()

	out := capturarStdout(t, func() {
		if err := runtimeCheckpointVerCmd.RunE(runtimeCheckpointVerCmd, []string{itoa(checkpointID)}); err != nil {
			t.Fatalf("runtime checkpoint-ver local: %v", err)
		}
	})
	for _, token := range []string{"Checkpoint #", "Codex1", "checkpoint inspeccionable", "manual-ver"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida checkpoint-ver sin %q:\n%s", token, out)
		}
	}
}

func TestRuntimeControlPlaneUsaAPICuandoHayServidor(t *testing.T) {
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
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
