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
	"path/filepath"
	"strings"
	"testing"

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
	assertKey(http.MethodGet, "/api/runtime-transcript?agente=Codex1&proyecto=orquestador", nil, "transcript", http.StatusOK)
	assertKey(http.MethodGet, "/api/runtime-orders?agente=Codex1", nil, "orders", http.StatusOK)
	assertKey(http.MethodGet, "/api/runtime-mailbox?to_agente=Codex2&proyecto=orquestador", nil, "mailbox", http.StatusOK)
	assertKey(http.MethodGet, "/api/runtime-checkpoints/latest?agente=Codex1&proyecto=orquestador", nil, "checkpoint", http.StatusOK)
	assertKey(http.MethodPost, "/api/runtime-orders", []byte(`{"agente":"Codex1","tipo":"checkpoint","proyecto":"orquestador","payload":"{}"}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/runtime-checkpoints", []byte(`{"agente":"Codex1","proyecto":"orquestador","checkpoint_kind":"manual","resumen":"checkpoint manual","branch":"main","cwd":"/tmp/orquestador","payload":"{}","resume_strategy":"resumen_y_payload","source":"api-test"}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/runtime-mailbox", []byte(`{"from_agente":"Codex1","to_agente":"Codex2","kind":"handoff","proyecto":"orquestador","payload":"{}"}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/runtime-mailbox/"+itoa(mailID)+"/entregar", []byte(`{}`), "id", http.StatusOK)
	assertKey(http.MethodPost, "/api/runtime-mailbox/"+itoa(mailID)+"/consumir", []byte(`{}`), "id", http.StatusOK)
}
