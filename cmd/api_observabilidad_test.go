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
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/coordinacion"
	"orquesta/db"
)

func TestAPIObservabilidadReadOnly(t *testing.T) {
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
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-read-001",
		ResumenContinuidad: "lectura api",
		Branch:             "main",
		Host:               "host-a",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	sesion, err := db.GetSesionActiva("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get sesion activa: %v", err)
	}

	if err := db.ConfigSet("clave_test_observabilidad", "valor-test"); err != nil {
		t.Fatalf("config set: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_url", "https://openclaw.local/gateway"); err != nil {
		t.Fatalf("config openclaw url: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_token", "secreto"); err != nil {
		t.Fatalf("config openclaw token: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_operator", "alberto"); err != nil {
		t.Fatalf("config openclaw operator: %v", err)
	}
	entregaID, err := db.CrearEntregaNotificacion("openclaw_gateway", "https://openclaw.local/gateway", db.EventoNotificacion{
		Tipo:  "mensaje",
		Texto: "hola openclaw",
	})
	if err != nil {
		t.Fatalf("crear entrega notificacion: %v", err)
	}
	if err := db.MarcarEntregaNotificacionFallida(entregaID, "boom", time.Now().UTC().Add(time.Minute)); err != nil {
		t.Fatalf("marcar entrega fallida: %v", err)
	}
	if err := db.ConfigSet("telegram_token", "tg-token"); err != nil {
		t.Fatalf("config telegram token: %v", err)
	}
	if err := db.ConfigSet("telegram_chat_id", "12345"); err != nil {
		t.Fatalf("config telegram chat id: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Tarea API read-only",
		Descripcion: "Detalle de tarea",
		Modulo:      "orquestador",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "Codex1",
		ProyectoID:  &proyectoID,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	propuesta := &db.Propuesta{
		Titulo:       "Propuesta API read-only",
		Descripcion:  "Detalle de propuesta",
		Tipo:         "implementacion",
		PropuestoPor: "Codex1",
		Distribuidor: "Codex1",
		ProyectoID:   &proyectoID,
	}
	if _, err := db.CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	propuestaDB, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}

	lock, err := (db.SQLiteLockRepository{}).Create(&coordinacion.Lock{
		ProjectID:   &proyectoID,
		TaskID:      &tareaID,
		SessionID:   &sesion.ID,
		Agent:       "Codex1",
		ScopeType:   "task",
		ScopeKey:    "tarea-api-read-only",
		Path:        filepath.Join(tmp, "orquestador"),
		Branch:      "main",
		Reason:      "prueba api",
		LeaseToken:  "lease-001",
		State:       coordinacion.LockState("activa"),
		HeartbeatAt: time.Now(),
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("crear lock: %v", err)
	}

	worktree, err := (db.SQLiteWorktreeRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		LockID:    &lock.ID,
		Agent:     "Codex1",
		Name:      "wt-api",
		Path:      filepath.Join(tmp, ".orquesta-worktrees", "wt-api"),
		Branch:    "feat/api-read-only",
		BaseRef:   "main",
		State:     coordinacion.WorktreeState("activa"),
		Reason:    "prueba api",
	})
	if err != nil {
		t.Fatalf("crear worktree: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	assertOK := func(path string, key string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status inesperado %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&body); err != nil {
			t.Fatalf("decode json %s: %v", path, err)
		}
		if _, ok := body[key]; !ok {
			t.Fatalf("respuesta %s sin clave %q: %s", path, key, rec.Body.String())
		}
	}

	assertOK("/api/config", "config")
	assertOK("/api/config?clave=clave_test_observabilidad", "valor")
	assertOK("/api/notificaciones", "notificaciones")
	assertOK("/api/openclaw/operator", "status")
	assertOK("/api/audit?limit=10&agente=Codex1", "audit")
	assertOK("/api/diagnostico?audit_limit=5", "diagnostico")
	assertOK("/api/reglas?tipo_agente=programador", "reglas")
	assertOK("/api/skills?agente=Codex1", "skills")
	assertOK("/api/workflows?tipo_agente=programador", "workflows")
	assertOK("/api/sesiones", "sesiones")
	assertOK("/api/sesiones?agente=Codex1&proyecto=orquestador&activa=true&estado=activa&limit=1", "sesiones")
	assertOK("/api/sesiones/"+itoa(sesion.ID), "sesion")
	assertOK("/api/tareas/"+itoa(tareaID), "tarea")
	assertOK("/api/propuestas/"+propuestaDB.Codigo, "propuesta")
	assertOK("/api/locks/"+itoa(lock.ID), "lock")
	assertOK("/api/worktrees/"+itoa(worktree.ID), "worktree")

	recNotif := httptest.NewRecorder()
	reqNotif := httptest.NewRequest(http.MethodGet, "/api/notificaciones", nil)
	mux.ServeHTTP(recNotif, reqNotif)
	if recNotif.Code != http.StatusOK {
		t.Fatalf("notificaciones status=%d body=%s", recNotif.Code, recNotif.Body.String())
	}
	var notifJSON map[string]any
	if err := json.NewDecoder(bytes.NewReader(recNotif.Body.Bytes())).Decode(&notifJSON); err != nil {
		t.Fatalf("decode notificaciones: %v", err)
	}
	if _, ok := notifJSON["eventos_normalizados"]; !ok {
		t.Fatalf("notificaciones sin eventos_normalizados: %s", recNotif.Body.String())
	}
	bodyNotif := recNotif.Body.String()
	for _, token := range []string{"OpenClaw Gateway", "openclaw.local/gateway", "Telegram", "12345", "entregas", "fallida"} {
		if !strings.Contains(bodyNotif, token) {
			t.Fatalf("notificaciones incompleta, falta %q:\n%s", token, bodyNotif)
		}
	}

	recOperator := httptest.NewRecorder()
	reqOperator := httptest.NewRequest(http.MethodGet, "/api/openclaw/operator", nil)
	mux.ServeHTTP(recOperator, reqOperator)
	if recOperator.Code != http.StatusOK {
		t.Fatalf("openclaw operator status=%d body=%s", recOperator.Code, recOperator.Body.String())
	}
	var operatorJSON map[string]any
	if err := json.NewDecoder(bytes.NewReader(recOperator.Body.Bytes())).Decode(&operatorJSON); err != nil {
		t.Fatalf("decode operator: %v", err)
	}
	if _, ok := operatorJSON["eventos_normalizados"]; !ok {
		t.Fatalf("operator sin eventos_normalizados: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["thread_sessions"]; !ok {
		t.Fatalf("operator sin thread_sessions: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["pipeline_state"]; !ok {
		t.Fatalf("operator sin pipeline_state: %s", recOperator.Body.String())
	}
	bodyOperator := recOperator.Body.String()
	for _, token := range []string{"status", "review", "notificaciones", "entregas", "eventos_normalizados", "thread_sessions", "pipeline_state"} {
		if !strings.Contains(bodyOperator, token) {
			t.Fatalf("openclaw operator incompleto, falta %q:\n%s", token, bodyOperator)
		}
	}
	for _, forbidden := range []string{"resume_payload_json", "worktreesActivas", "runtimesActivos"} {
		if strings.Contains(bodyOperator, forbidden) {
			t.Fatalf("openclaw operator demasiado verboso, contiene %q:\n%s", forbidden, bodyOperator)
		}
	}
}

func TestAPIOpenClawThreadsOperaPorLaViaCanonica(t *testing.T) {
	prepararDBTemporalCmd(t)
	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/api/openclaw/threads", strings.NewReader(`{
		"supervisor":"OpenClaw",
		"proyecto":"orquestador",
		"session_id":"sess-api-1",
		"thread_id":"leader-1",
		"kind":"leader",
		"mode":"review",
		"source":"api_test",
		"turn_id":"turn-1"
	}`))
	reqPost.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Fatalf("openclaw threads post status=%d body=%s", recPost.Code, recPost.Body.String())
	}

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/openclaw/threads?supervisor=OpenClaw&session_id=sess-api-1", nil)
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("openclaw threads get status=%d body=%s", recGet.Code, recGet.Body.String())
	}
	if !strings.Contains(recGet.Body.String(), "sess-api-1") {
		t.Fatalf("openclaw threads sin sesion esperada: %s", recGet.Body.String())
	}
}

func TestAPIOpenClawPipelineOperaPorLaViaCanonica(t *testing.T) {
	prepararDBTemporalCmd(t)
	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/api/openclaw/pipeline", strings.NewReader(`{
		"supervisor":"OpenClaw",
		"proyecto":"orquestador",
		"pipeline_name":"autopilot",
		"current_phase":"review",
		"status":"active",
		"metadata_json":"{\"mode\":\"review\"}"
	}`))
	reqPost.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Fatalf("openclaw pipeline post status=%d body=%s", recPost.Code, recPost.Body.String())
	}

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/openclaw/pipeline?supervisor=OpenClaw&proyecto=orquestador", nil)
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("openclaw pipeline get status=%d body=%s", recGet.Code, recGet.Body.String())
	}
	if !strings.Contains(recGet.Body.String(), "autopilot") || !strings.Contains(recGet.Body.String(), "review") {
		t.Fatalf("openclaw pipeline sin contenido esperado: %s", recGet.Body.String())
	}
}

func TestAPIWorkflowsDetalleReadOnly(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workflows?tipo_agente=programador&nombre=inicio-sesion", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&body); err != nil {
		t.Fatalf("decode workflow detalle: %v", err)
	}
	if _, ok := body["workflow"]; !ok {
		t.Fatalf("respuesta sin workflow: %s", rec.Body.String())
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
