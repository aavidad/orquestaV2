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
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/propuestasapp"
)

func TestAPIObservabilidadReadOnly(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar codex2: %v", err)
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
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex2",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-read-002",
		ResumenContinuidad: "lectura api 2",
		Branch:             "main",
		Host:               "host-b",
	}); err != nil {
		t.Fatalf("iniciar sesion codex2: %v", err)
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
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"continua"}`,
		Estado:      "pendiente",
	}); err != nil {
		t.Fatalf("crear mailbox pendiente: %v", err)
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
	if _, ok := operatorJSON["queue_summary"]; !ok {
		t.Fatalf("openclaw operator sin queue_summary: %s", recOperator.Body.String())
	}
	if queueSummary, ok := operatorJSON["queue_summary"].(map[string]any); !ok || queueSummary["safe_by_kind"] == nil {
		t.Fatalf("openclaw operator sin safe_by_kind en queue_summary: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["capacity_summary"]; !ok {
		t.Fatalf("openclaw operator sin capacity_summary: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["saturated_agents"]; !ok {
		t.Fatalf("openclaw operator sin saturated_agents: %s", recOperator.Body.String())
	}
	reviewMap, ok := operatorJSON["review"].(map[string]any)
	if !ok {
		t.Fatalf("openclaw operator sin review compacta: %s", recOperator.Body.String())
	}
	for _, forbidden := range []string{"action_queue", "next_action", "next_safe_action", "safe_action_queue", "thread_sessions", "pipeline_state", "queue_summary", "normalized_events", "mailbox_pending"} {
		if _, present := reviewMap[forbidden]; present {
			t.Fatalf("review compacta no deberia arrastrar %q: %s", forbidden, recOperator.Body.String())
		}
	}
	for _, token := range []string{"next_safe_action", "safe_action_queue"} {
		if !strings.Contains(bodyOperator, token) {
			t.Fatalf("openclaw operator sin cola segura, falta %q:\n%s", token, bodyOperator)
		}
	}
	for _, forbidden := range []string{"resume_payload_json", "worktreesActivas", "runtimesActivos"} {
		if strings.Contains(bodyOperator, forbidden) {
			t.Fatalf("openclaw operator demasiado verboso, contiene %q:\n%s", forbidden, bodyOperator)
		}
	}
	statusMap, ok := operatorJSON["status"].(map[string]any)
	if !ok {
		t.Fatalf("openclaw operator sin status estructurado: %s", recOperator.Body.String())
	}
	activos, ok := statusMap["agentesActivos"].([]any)
	if !ok {
		t.Fatalf("openclaw operator sin agentesActivos estructurados: %s", recOperator.Body.String())
	}
	if len(activos) != 2 {
		t.Fatalf("openclaw operator deberia reflejar 2 agentes activos, obtuvo %d: %s", len(activos), recOperator.Body.String())
	}
	mailboxPendiente, ok := statusMap["mailboxPendiente"].([]any)
	if !ok || len(mailboxPendiente) == 0 {
		t.Fatalf("openclaw operator sin mailboxPendiente: %s", recOperator.Body.String())
	}
	firstMailbox, ok := mailboxPendiente[0].(map[string]any)
	if !ok {
		t.Fatalf("openclaw operator sin mailboxPendiente estructurado: %s", recOperator.Body.String())
	}
	if firstMailbox["oldest_created_at"] == nil {
		t.Fatalf("openclaw operator sin oldest_created_at en mailboxPendiente: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["capacity_summary"].(map[string]any); !ok {
		t.Fatalf("openclaw operator sin capacity_summary estructurado: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["saturated_agents"].([]any); !ok {
		t.Fatalf("openclaw operator sin saturated_agents estructurado: %s", recOperator.Body.String())
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

func TestAPIOpenClawOperatorSeparaCargaActivaYReservada(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() { statusService = prev }()

	statusService = stubStatusService{response: apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		Agentes: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasActivas: []tareaLite{
			{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex3", Titulo: "activa"},
			{ID: 2, Estado: db.TareaAsignada, Agente: "Codex3", Titulo: "reservada"},
		},
	}}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/openclaw/operator", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("openclaw operator status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("decode operator: %v", err)
	}
	statusMap, _ := payload["status"].(map[string]any)
	activos, _ := statusMap["agentesActivos"].([]any)
	if len(activos) != 1 {
		t.Fatalf("agentesActivos inesperados: %#v", statusMap["agentesActivos"])
	}
	first, _ := activos[0].(map[string]any)
	if got := int(first["carga_activa"].(float64)); got != 1 {
		t.Fatalf("carga_activa inesperada: %#v", first)
	}
	if got := int(first["carga_reservada"].(float64)); got != 1 {
		t.Fatalf("carga_reservada inesperada: %#v", first)
	}
	capacityMap, ok := payload["capacity_summary"].(map[string]any)
	if !ok {
		t.Fatalf("capacity_summary ausente: %s", rec.Body.String())
	}
	if got := int(capacityMap["workers_saturados"].(float64)); got != 1 {
		t.Fatalf("workers_saturados inesperado: %#v", capacityMap)
	}
	saturated, ok := payload["saturated_agents"].([]any)
	if !ok || len(saturated) != 1 {
		t.Fatalf("saturated_agents inesperado: %#v", payload["saturated_agents"])
	}
	reservadas, _ := statusMap["tareasReservadas"].([]any)
	if len(reservadas) != 1 {
		t.Fatalf("tareasReservadas inesperadas: %#v", statusMap["tareasReservadas"])
	}
	activas, _ := statusMap["tareasActivas"].([]any)
	if len(activas) != 1 {
		t.Fatalf("tareasActivas inesperadas: %#v", statusMap["tareasActivas"])
	}
}

func TestAPIOpenClawOperatorExponeStatusLiteEnRaiz(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() { statusService = prev }()

	statusService = stubStatusService{response: apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		Agentes: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasActivas: []tareaLite{
			{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex3", Titulo: "activa"},
			{ID: 2, Estado: db.TareaAsignada, Agente: "Codex3", Titulo: "reservada"},
		},
		PropuestasResumen: []propuestaLite{
			{ID: 9, Codigo: "OP-999", Titulo: "demo", Estado: db.PropuestaAbierta},
		},
	}}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/openclaw/operator", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("openclaw operator status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("decode operator: %v", err)
	}
	if activos, ok := payload["agentesActivos"].([]any); !ok || len(activos) != 1 {
		t.Fatalf("agentesActivos raiz inesperados: %#v", payload["agentesActivos"])
	}
	if tareas, ok := payload["tareasActivas"].([]any); !ok || len(tareas) != 1 {
		t.Fatalf("tareasActivas raiz inesperadas: %#v", payload["tareasActivas"])
	}
	if reservas, ok := payload["tareasReservadas"].([]any); !ok || len(reservas) != 1 {
		t.Fatalf("tareasReservadas raiz inesperadas: %#v", payload["tareasReservadas"])
	}
	if propuestas, ok := payload["propuestasAbiertas"].([]any); !ok || len(propuestas) != 1 {
		t.Fatalf("propuestasAbiertas raiz inesperadas: %#v", payload["propuestasAbiertas"])
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

func TestAPIOpenClawOperatorAccionaPorLaViaCanonica(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() { statusService = prev }()

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar Codex4: %v", err)
	}
	propuestaID, _, err := propuestasService.Create(propuestasapp.CreateProposalInput{
		Codigo:       "OP-604",
		Titulo:       "Cerrar propuesta por API",
		Descripcion:  "Debe cerrarse por /api/openclaw/operator",
		Tipo:         "implementacion",
		PropuestoPor: "antigravity",
	})
	if err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	if _, err := db.Votar(propuestaID, "Codex1", db.VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("voto acuerdo: %v", err)
	}
	if _, err := db.Votar(propuestaID, "Codex2", db.VotoDesacuerdo, "no"); err != nil {
		t.Fatalf("voto desacuerdo: %v", err)
	}
	if _, err := db.Votar(propuestaID, "Codex3", db.VotoDesacuerdo, "no"); err != nil {
		t.Fatalf("segundo voto desacuerdo: %v", err)
	}
	agenteCodex3 := "Codex3"
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reserva por API",
		Descripcion: "Debe rebalancearse por API OpenClaw",
		Modulo:      "cmd",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "OpenClaw",
		Agente:      &agenteCodex3,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='asignada', agente=? WHERE id=?`, agenteCodex3, tareaID); err != nil {
		t.Fatalf("preparar tarea asignada: %v", err)
	}

	statusService = stubStatusService{response: apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		Agentes: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		PropuestasResumen: []propuestaLite{
			{Codigo: "OP-604", Titulo: "Cerrar propuesta por API", Estado: db.PropuestaAbierta, Acuerdo: 1, Desacuerdo: 2, Pendiente: 0},
		},
		TareasActivas: []tareaLite{
			{ID: tareaID, Estado: db.TareaAsignada, Titulo: "Reserva por API", Agente: "Codex3", Modulo: "cmd"},
			{ID: 3001, Estado: db.TareaEnProgreso, Titulo: "Carga en Codex3", Agente: "Codex3", Modulo: "db"},
			{ID: 3002, Estado: db.TareaEnProgreso, Titulo: "Carga en Codex3 2", Agente: "Codex3", Modulo: "api"},
			{ID: 3003, Estado: db.TareaEnProgreso, Titulo: "Carga en Codex4", Agente: "Codex4", Modulo: "planocontrol"},
		},
	}}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recBatch := httptest.NewRecorder()
	reqBatch := httptest.NewRequest(http.MethodPost, "/api/openclaw/operator", strings.NewReader(`{
		"mode":"batch",
		"batch_kind":"proposal",
		"max_items":1
	}`))
	reqBatch.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recBatch, reqBatch)
	if recBatch.Code != http.StatusOK {
		t.Fatalf("openclaw operator batch status=%d body=%s", recBatch.Code, recBatch.Body.String())
	}
	propuesta, err := db.GetPropuesta("OP-604")
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}
	if propuesta == nil || propuesta.Estado != db.PropuestaRechazada {
		t.Fatalf("propuesta no quedó rechazada: %+v", propuesta)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || *tarea.Agente != "Codex3" || tarea.Estado != db.TareaAsignada {
		t.Fatalf("dispatch mezclado en batch proposal por API: %+v", tarea)
	}

	recAction := httptest.NewRecorder()
	reqAction := httptest.NewRequest(http.MethodPost, "/api/openclaw/operator", strings.NewReader(fmt.Sprintf(`{
		"mode":"action",
		"action":"rebalancear_reserva",
		"target":"tarea:%d",
		"assignee":"Codex4"
	}`, tareaID)))
	reqAction.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recAction, reqAction)
	if recAction.Code != http.StatusOK {
		t.Fatalf("openclaw operator action status=%d body=%s", recAction.Code, recAction.Body.String())
	}
	tarea, err = db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea tras action: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || *tarea.Agente != "Codex4" || tarea.Estado != db.TareaAsignada {
		t.Fatalf("rebalanceo por API no aplicado: %+v", tarea)
	}

	recNext := httptest.NewRecorder()
	reqNext := httptest.NewRequest(http.MethodPost, "/api/openclaw/operator", strings.NewReader(`{
		"mode":"next"
	}`))
	reqNext.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recNext, reqNext)
	if recNext.Code != http.StatusOK {
		t.Fatalf("openclaw operator next status=%d body=%s", recNext.Code, recNext.Body.String())
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
