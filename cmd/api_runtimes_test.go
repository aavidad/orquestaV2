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
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"orquesta/db"
)

func TestAPIRuntimesReadOnly(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente destino: %v", err)
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
		ExternalSessionID:  "sess-runtime-api",
		ResumenContinuidad: "api runtime",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get runtime: %v", err)
	}
	if runtime == nil {
		t.Fatalf("runtime nil")
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	assertRuntimeOK := func(path string, key string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status inesperado %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode json %s: %v", path, err)
		}
		if _, ok := body[key]; !ok {
			t.Fatalf("respuesta %s sin clave %q: %s", path, key, rec.Body.String())
		}
	}

	assertRuntimeOK("/api/runtimes", "runtimes")
	assertRuntimeOK("/api/runtimes?proyecto=orquestador&activos=true", "runtimes")
	assertRuntimeOK("/api/runtimes/tree?proyecto=orquestador", "runtimes")
	assertRuntimeOK("/api/runtimes/"+itoa(runtime.ID), "runtime")
}

func TestAPIRuntimeControlPlaneEndpoints(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente destino: %v", err)
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
		ExternalSessionID:  "sess-runtime-api-controlplane",
		ResumenContinuidad: "api runtime controlplane",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"motivo":"test"}`,
	}); err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	mailID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "Codex1",
		ToAgente:    "Codex2",
		ProyectoID:  &proyectoID,
		Kind:        "handoff",
		PayloadJSON: `{"ok":true}`,
	})
	if err != nil {
		t.Fatalf("crear runtime mailbox: %v", err)
	}
	if _, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		SesionID:       &sesion.ID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint api",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"ok":true}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "test",
	}); err != nil {
		t.Fatalf("crear runtime checkpoint: %v", err)
	}
	if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
		RuntimeID:   runtime.ID,
		Kind:        "auto_guidance_sent",
		Level:       "info",
		Message:     "Orquesta respondió a approval_request",
		PayloadJSON: `{"runtime_order_id":7,"classification":"approval_request"}`,
	}); err != nil {
		t.Fatalf("crear runtime event: %v", err)
	}
	if _, err := db.RegistrarRuntimeTranscript(&db.RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "¿me dejas seguir con el refactor?",
		NormalizedText: "me dejas seguir con el refactor",
		Classification: "approval_request",
	}); err != nil {
		t.Fatalf("crear runtime transcript: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get runtime handle: %+v err=%v", handle, err)
	}
	traceDir := filepath.Join(tmp, "orquestador", ".orquesta-runtime", "codex1", "20260329-120000-000000001")
	if err := os.MkdirAll(traceDir, 0o700); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	logPath := filepath.Join(traceDir, "pty.log")
	manifestPath := filepath.Join(traceDir, "runtime.json")
	if err := os.WriteFile(logPath, []byte("inicio\navance\nfin\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"trace_dir":      traceDir,
		"trace_manifest": manifestPath,
		"log_path":       logPath,
		"working_dir":    filepath.Join(tmp, "orquestador"),
		"driver":         "process_pty_cli",
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json = ? WHERE id = ?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update runtime handle metadata: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	assertKey := func(method, path string, body []byte, key string, wantCode int) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(string(body)))
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		mux.ServeHTTP(rec, req)
		if rec.Code != wantCode {
			t.Fatalf("status inesperado %s %s: %d body=%s", method, path, rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode json %s %s: %v", method, path, err)
		}
		if _, ok := payload[key]; !ok {
			t.Fatalf("respuesta %s %s sin clave %q: %s", method, path, key, rec.Body.String())
		}
	}

	assertKey(http.MethodGet, "/api/runtime-handles?agente=Codex1", nil, "handles", http.StatusOK)
	assertKey(http.MethodGet, "/api/runtime-events?agente=Codex1&proyecto=orquestador&kind=auto_guidance_sent", nil, "events", http.StatusOK)
	res, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado
	) VALUES (?,?,?,?,?,?,?)`,
		"Codex1", proyectoID, runtime.ID, "cli", "process", "stale-api", "cerrado",
	)
	if err != nil {
		t.Fatalf("insert runtime handle cerrado: %v", err)
	}
	if _, err := res.LastInsertId(); err != nil {
		t.Fatalf("last insert id runtime handle cerrado: %v", err)
	}
	assertKey(http.MethodPost, "/api/runtime-handles/purgar", []byte(`{"agente":"Codex1","actor":"Codex1"}`), "deleted", http.StatusOK)
	assertKey(http.MethodGet, "/api/runtime-trace?agente=Codex1&proyecto=orquestador&bytes=8", nil, "trace", http.StatusOK)
	assertKey(http.MethodGet, "/api/runtime-transcript?agente=Codex1&proyecto=orquestador", nil, "transcript", http.StatusOK)
	recTranscript := httptest.NewRecorder()
	reqTranscript := httptest.NewRequest(http.MethodGet, "/api/runtime-transcript?agente=Codex1&proyecto=orquestador&q=refactor", nil)
	mux.ServeHTTP(recTranscript, reqTranscript)
	if recTranscript.Code != http.StatusOK {
		t.Fatalf("status inesperado transcript search: %d body=%s", recTranscript.Code, recTranscript.Body.String())
	}
	var payloadTranscript map[string]any
	if err := json.Unmarshal(recTranscript.Body.Bytes(), &payloadTranscript); err != nil {
		t.Fatalf("decode transcript search: %v", err)
	}
	items, _ := payloadTranscript["transcript"].([]any)
	if len(items) != 1 {
		t.Fatalf("transcript search inesperado: %s", recTranscript.Body.String())
	}
	recTrace := httptest.NewRecorder()
	reqTrace := httptest.NewRequest(http.MethodGet, "/api/runtime-trace?agente=Codex1&proyecto=orquestador&bytes=8", nil)
	mux.ServeHTTP(recTrace, reqTrace)
	if recTrace.Code != http.StatusOK {
		t.Fatalf("status inesperado runtime trace: %d body=%s", recTrace.Code, recTrace.Body.String())
	}
	var payloadTrace map[string]any
	if err := json.Unmarshal(recTrace.Body.Bytes(), &payloadTrace); err != nil {
		t.Fatalf("decode runtime trace: %v", err)
	}
	trace, _ := payloadTrace["trace"].(map[string]any)
	if rawTail, _ := trace["raw_tail"].(string); !strings.Contains(rawTail, "ce\nfin\n") {
		t.Fatalf("raw_tail inesperado: %s", recTrace.Body.String())
	}
	if available, _ := trace["available"].(bool); !available {
		t.Fatalf("trace deberia estar disponible: %s", recTrace.Body.String())
	}
	assertKey(http.MethodGet, "/api/runtime-orders?agente=Codex1", nil, "orders", http.StatusOK)
	assertKey(http.MethodPost, "/api/runtime-orders/purgar", []byte(`{"agente":"Codex1","estados":["completada","fallida"],"tipos":["send_instruction"],"actor":"Codex1"}`), "deleted", http.StatusOK)
	assertKey(http.MethodPost, "/api/runtimes/cerrar", []byte(`{"agente":"Codex1","proyecto":"orquestador","actor":"Codex1"}`), "closed", http.StatusOK)
	assertKey(http.MethodGet, "/api/runtime-mailbox?to_agente=Codex2&proyecto=orquestador", nil, "mailbox", http.StatusOK)
	assertKey(http.MethodGet, "/api/runtime-checkpoints/latest?agente=Codex1&proyecto=orquestador", nil, "checkpoint", http.StatusOK)
	assertKey(http.MethodPost, "/api/runtime-orders", []byte(`{"agente":"Codex1","tipo":"checkpoint","proyecto":"orquestador","payload":"{}"}`), "id", http.StatusCreated)
	recBadOrder := httptest.NewRecorder()
	reqBadOrder := httptest.NewRequest(http.MethodPost, "/api/runtime-orders", strings.NewReader(`{"agente":"Codex1","tipo":"send_instruction","proyecto":"orquestador","payload":"{\"texto\":\"rota\""}`))
	reqBadOrder.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recBadOrder, reqBadOrder)
	if recBadOrder.Code != http.StatusBadRequest {
		t.Fatalf("status inesperado runtime order invalida: %d body=%s", recBadOrder.Code, recBadOrder.Body.String())
	}
	if !strings.Contains(recBadOrder.Body.String(), "payload_json inválido") {
		t.Fatalf("mensaje inesperado runtime order invalida: %s", recBadOrder.Body.String())
	}
	assertKey(http.MethodPost, "/api/runtime-checkpoints", []byte(`{"agente":"Codex1","proyecto":"orquestador","checkpoint_kind":"manual","resumen":"checkpoint manual","branch":"main","cwd":"/tmp/orquestador","payload":"{}","resume_strategy":"resumen_y_payload","source":"api-test"}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/runtime-mailbox", []byte(`{"from_agente":"Codex1","to_agente":"Codex2","kind":"handoff","proyecto":"orquestador","payload":"{}"}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/runtime-mailbox/"+itoa(mailID)+"/entregar", []byte(`{}`), "id", http.StatusOK)
	assertKey(http.MethodPost, "/api/runtime-mailbox/"+itoa(mailID)+"/consumir", []byte(`{}`), "id", http.StatusOK)
}

func TestAPIRuntimeOrdersAplicaLimitPorDefecto(t *testing.T) {
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
	for i := 0; i < 60; i++ {
		if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
			Agente:      "Codex1",
			ProyectoID:  &proyectoID,
			Tipo:        "checkpoint",
			PayloadJSON: `{"n":1}`,
		}); err != nil {
			t.Fatalf("encolar runtime order %d: %v", i, err)
		}
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/runtime-orders?agente=Codex1&proyecto=orquestador", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	orders, _ := payload["orders"].([]any)
	if len(orders) != 50 {
		t.Fatalf("orders esperadas=50 obtenidas=%d", len(orders))
	}
}

func TestAPIRuntimeHandlesCerrarResiduals(t *testing.T) {
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
		ExternalSessionID:  "sess-close-runtime-handle",
		ResumenContinuidad: "test cierre handle residual",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get runtime handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("activar handle: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/runtime-handles/cerrar", strings.NewReader(`{"agente":"Codex1","proyecto":"orquestador","actor":"Codex1"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado runtime-handles/cerrar: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime-handles/cerrar: %v", err)
	}
	if closed, _ := payload["closed"].(bool); !closed {
		t.Fatalf("respuesta sin closed=true: %s", rec.Body.String())
	}
	handles, err := db.ListarRuntimeHandles(nil)
	if err != nil {
		t.Fatalf("listar handles: %v", err)
	}
	if len(handles) != 1 || handles[0] == nil || handles[0].Estado != "cerrado" {
		t.Fatalf("handles cerrados inesperados: %+v", handles)
	}
}

func TestAPIRuntimeDetailIncluyeWorkerEstructurado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
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
		Agente:             "Codex7",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-runtime-worker-api",
		ResumenContinuidad: "api worker",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get runtime handle: %+v err=%v", handle, err)
	}

	runDir := filepath.Join(tmp, "orquestador", ".orquesta-runtime", "codex7", "20260405-120000-000000001")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC()
	writeJSON := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"version":      1,
		"agent":        "Codex7",
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex7-120000",
		"tmux_pane_id": "%4",
		"child_pid":    4321,
	})
	writeJSON(statusPath, map[string]any{
		"state":          "running",
		"updated_at":     now.Format(time.RFC3339Nano),
		"alive":          true,
		"last_output_at": now.Format(time.RFC3339Nano),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update metadata: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/runtimes/"+itoa(runtime.ID), nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado runtime detail: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime detail: %v", err)
	}
	worker, _ := payload["worker"].(map[string]any)
	if worker == nil {
		t.Fatalf("runtime detail sin worker: %s", rec.Body.String())
	}
	if got, _ := worker["driver"].(string); got != "tmux_cli_session" {
		t.Fatalf("driver worker inesperado: %+v", worker)
	}
	if got, _ := worker["runtime_ref"].(string); got != "orq-codex7-120000/%4" {
		t.Fatalf("runtime_ref worker inesperado: %+v", worker)
	}
	if alive, _ := worker["alive"].(bool); !alive {
		t.Fatalf("worker deberia salir vivo: %+v", worker)
	}
	if _, ok := payload["handle"]; !ok {
		t.Fatalf("runtime detail sin handle: %s", rec.Body.String())
	}
}

func TestAPIRuntimeEventsCompactaPayloadGigante(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	gigante := strings.Repeat("x", 5000)
	if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
		RuntimeID:   runtime.ID,
		Kind:        "runtime_panic",
		Level:       "critical",
		Message:     "panic " + gigante,
		PayloadJSON: `{"text":"` + gigante + `","command":"` + gigante + `"}`,
	}); err != nil {
		t.Fatalf("registrar runtime event gigante: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/runtime-events?agente=Codex1&proyecto=orquestador&limit=5", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado runtime events: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime events: %v", err)
	}
	items, _ := payload["events"].([]any)
	if len(items) == 0 {
		t.Fatalf("runtime events vacíos: %s", rec.Body.String())
	}
	first, _ := items[0].(map[string]any)
	msg, _ := first["message"].(string)
	if utf8.RuneCountInString(msg) > 512 {
		t.Fatalf("message debería salir compactado, runes=%d", utf8.RuneCountInString(msg))
	}
	rawPayload, _ := first["payload_json"].(string)
	if utf8.RuneCountInString(rawPayload) > 1536 {
		t.Fatalf("payload_json debería salir compactado, runes=%d", utf8.RuneCountInString(rawPayload))
	}
	payloadMap, _ := first["payload"].(map[string]any)
	texto, _ := payloadMap["text"].(string)
	if utf8.RuneCountInString(texto) > 512 {
		t.Fatalf("payload.text debería salir compactado, runes=%d", utf8.RuneCountInString(texto))
	}
}

func TestAPIRuntimeEventsSaneaCSIHistorico(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	raw := ".\x1b[<1uThe application panicked (crashed)."
	if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
		RuntimeID:   runtime.ID,
		Kind:        "runtime_panic",
		Level:       "critical",
		Message:     raw,
		PayloadJSON: `{"text":"` + raw + `"}`,
	}); err != nil {
		t.Fatalf("registrar runtime event CSI: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/runtime-events?agente=Codex1&proyecto=orquestador&limit=5", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado runtime events: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime events: %v", err)
	}
	items, _ := payload["events"].([]any)
	if len(items) == 0 {
		t.Fatalf("runtime events vacíos: %s", rec.Body.String())
	}
	first, _ := items[0].(map[string]any)
	msg, _ := first["message"].(string)
	if strings.Contains(msg, "[<1u") || strings.Contains(msg, "\u001b") {
		t.Fatalf("message debería salir saneado: %q", msg)
	}
	payloadMap, _ := first["payload"].(map[string]any)
	texto, _ := payloadMap["text"].(string)
	if strings.Contains(texto, "[<1u") || strings.Contains(texto, "\u001b") {
		t.Fatalf("payload.text debería salir saneado: %q", texto)
	}
}
