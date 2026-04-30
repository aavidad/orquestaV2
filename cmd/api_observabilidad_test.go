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
	"os"
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
		PayloadJSON: `{"texto":"continua","carril":"premium_worktree","tarea_objetivo_id":530,"worktree_id":77,"write_set":["cmd/api.go","cmd/serve.go"]}`,
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

	lock, err := (db.CoordinationLockDBRepository{}).Create(&coordinacion.Lock{
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

	worktree, err := (db.CoordinationWorktreeDBRepository{}).Create(&coordinacion.Worktree{
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
	if _, ok := operatorJSON["subagentes"]; !ok {
		t.Fatalf("operator sin subagentes: %s", recOperator.Body.String())
	}
	if candidates, ok := operatorJSON["session_candidates"].([]any); !ok || len(candidates) == 0 {
		t.Fatalf("operator sin session_candidates: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["pipeline_state"]; !ok {
		t.Fatalf("operator sin pipeline_state: %s", recOperator.Body.String())
	}
	bodyOperator := recOperator.Body.String()
	for _, token := range []string{"status", "review", "notificaciones", "entregas", "eventos_normalizados", "thread_sessions", "subagentes", "pipeline_state"} {
		if !strings.Contains(bodyOperator, token) {
			t.Fatalf("openclaw operator incompleto, falta %q:\n%s", token, bodyOperator)
		}
	}
	if _, ok := operatorJSON["queue_summary"]; !ok {
		t.Fatalf("openclaw operator sin queue_summary: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["server_operational"]; !ok {
		t.Fatalf("openclaw operator sin server_operational: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["runtime_health"]; !ok {
		t.Fatalf("openclaw operator sin runtime_health: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["hot_paths"]; !ok {
		t.Fatalf("openclaw operator sin hot_paths: %s", recOperator.Body.String())
	}
	if summary, ok := operatorJSON["server_operational_summary"].(string); !ok || strings.TrimSpace(summary) == "" {
		t.Fatalf("openclaw operator sin server_operational_summary: %s", recOperator.Body.String())
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
	operationalMap, ok := operatorJSON["server_operational"].(map[string]any)
	if !ok {
		t.Fatalf("openclaw operator sin server_operational estructurado: %s", recOperator.Body.String())
	}
	if _, ok := operationalMap["state"].(string); !ok {
		t.Fatalf("openclaw operator sin state en server_operational: %#v", operationalMap)
	}
	if _, ok := operationalMap["operational"].(bool); !ok {
		t.Fatalf("openclaw operator sin operational bool en server_operational: %#v", operationalMap)
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
	if got, _ := firstMailbox["contexts_csv"].(string); !strings.Contains(got, "premium_worktree") || !strings.Contains(got, "tarea#530") || !strings.Contains(got, "wt#77") || !strings.Contains(got, "cmd/api.go (+1)") {
		t.Fatalf("openclaw operator sin contexto operativo en mailboxPendiente: %#v", firstMailbox)
	}
	if _, ok := operatorJSON["capacity_summary"].(map[string]any); !ok {
		t.Fatalf("openclaw operator sin capacity_summary estructurado: %s", recOperator.Body.String())
	}
	if _, ok := operatorJSON["saturated_agents"].([]any); !ok {
		t.Fatalf("openclaw operator sin saturated_agents estructurado: %s", recOperator.Body.String())
	}

	recMailbox := httptest.NewRecorder()
	reqMailbox := httptest.NewRequest(http.MethodGet, "/api/runtime-mailbox?estado=pendiente", nil)
	mux.ServeHTTP(recMailbox, reqMailbox)
	if recMailbox.Code != http.StatusOK {
		t.Fatalf("runtime mailbox status=%d body=%s", recMailbox.Code, recMailbox.Body.String())
	}
	var mailboxJSON map[string]any
	if err := json.NewDecoder(bytes.NewReader(recMailbox.Body.Bytes())).Decode(&mailboxJSON); err != nil {
		t.Fatalf("decode mailbox: %v", err)
	}
	mailboxList, ok := mailboxJSON["mailbox"].([]any)
	if !ok {
		t.Fatalf("runtime mailbox sin lista estructurada: %s", recMailbox.Body.String())
	}
	if len(mailboxList) != len(mailboxPendiente) {
		t.Fatalf("openclaw operator y runtime mailbox discrepan (%d vs %d): operator=%s mailbox=%s", len(mailboxPendiente), len(mailboxList), recOperator.Body.String(), recMailbox.Body.String())
	}
	firstMailboxAPI, ok := mailboxList[0].(map[string]any)
	if !ok {
		t.Fatalf("runtime mailbox sin primer elemento estructurado: %s", recMailbox.Body.String())
	}
	if firstMailboxAPI["to_agente"] != firstMailbox["agente"] {
		t.Fatalf("openclaw operator y runtime mailbox discrepan en agente destino: operator=%v mailbox=%v", firstMailbox["agente"], firstMailboxAPI["to_agente"])
	}
}

func TestAPIOpenClawThreadsOperaPorLaViaCanonica(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
		t.Fatalf("registrar codex6: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{Slug: "orquestador", Nombre: "Orquestador", RutaAbs: "/tmp/orquestador", Tipo: db.ProyectoRepo, Activo: true})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "Codex6",
		ProyectoID:        &proyectoID,
		CWD:               "/tmp/orquestador",
		Herramienta:       "claude-rust",
		ExternalSessionID: "claude-sess-1",
		Host:              "host-claude",
	}); err != nil {
		t.Fatalf("iniciar sesion codex6: %v", err)
	}
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
	if !strings.Contains(recGet.Body.String(), "observed_agent_sessions") || !strings.Contains(recGet.Body.String(), "claude-sess-1") {
		t.Fatalf("openclaw threads sin sesiones observadas de agentes: %s", recGet.Body.String())
	}
}

func TestBuildOpenClawPendingMailboxNoOcultaDeudaPorProyeccionVisible(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar codex3: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex3",
		Kind:        "nudge",
		PayloadJSON: `{"texto":"continua"}`,
		Estado:      "pendiente",
	}); err != nil {
		t.Fatalf("crear mailbox pendiente: %v", err)
	}
	items, err := buildOpenClawPendingMailbox([]*db.Agente{{Nombre: "Codex4"}})
	if err != nil {
		t.Fatalf("buildOpenClawPendingMailbox: %v", err)
	}
	if len(items) != 1 || items[0].Agente != "Codex3" || items[0].Count != 1 {
		t.Fatalf("mailbox pendiente inesperada: %#v", items)
	}
}

func TestBuildOpenClawPendingMailboxExponeSupervisorAction(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar codex3: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex3",
		Kind:        "autonomia",
		PayloadJSON: `{"supervisor_action":"revisar_worktree_desfasada","texto":"haz checkpoint"}`,
		Estado:      "pendiente",
	}); err != nil {
		t.Fatalf("crear mailbox pendiente: %v", err)
	}
	items, err := buildOpenClawPendingMailbox([]*db.Agente{{Nombre: "Codex3"}})
	if err != nil {
		t.Fatalf("buildOpenClawPendingMailbox: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("mailbox pendiente inesperada: %#v", items)
	}
	if items[0].SupervisorActionsCSV != "revisar_worktree_desfasada" {
		t.Fatalf("supervisor action inesperada: %#v", items[0])
	}
}

func TestBuildOpenClawPendingMailboxExponeContextoOperativo(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar codex3: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex3",
		Kind:        "autonomia",
		PayloadJSON: `{"carril":"premium_worktree","tarea_objetivo_id":604,"worktree_id":88,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"]}`,
		Estado:      "pendiente",
	}); err != nil {
		t.Fatalf("crear mailbox pendiente: %v", err)
	}
	items, err := buildOpenClawPendingMailbox([]*db.Agente{{Nombre: "Codex3"}})
	if err != nil {
		t.Fatalf("buildOpenClawPendingMailbox: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("mailbox pendiente inesperada: %#v", items)
	}
	if got := items[0].ContextsCSV; got != "premium_worktree · tarea#604 · wt#88 · cmd/controlplane_support.go (+1)" {
		t.Fatalf("contexto operativo inesperado: %q", got)
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
	defer func() {
		statusService = prev
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()

	snapshot := apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesAuthManual: []*db.Agente{
			{Nombre: "CodexLogin", Rol: "programador", EstadoCuota: "activo"},
		},
		Agentes: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			{Nombre: "CodexLogin", Rol: "programador", EstadoCuota: "activo"},
		},
		TareasActivas: []tareaLite{
			{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex3", Titulo: "activa"},
			{ID: 2, Estado: db.TareaAsignada, Agente: "Codex3", Titulo: "reservada"},
		},
		PropuestasResumen: []propuestaLite{
			{ID: 9, Codigo: "OP-999", Titulo: "demo", Estado: db.PropuestaAbierta},
		},
	}
	statusService = stubStatusService{response: snapshot}
	storeStatusSnapshot(snapshot, time.Now().UTC())

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
	if auth, ok := payload["agentesAuthManual"].([]any); !ok || len(auth) != 1 {
		t.Fatalf("agentesAuthManual raiz inesperados: %#v", payload["agentesAuthManual"])
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
	operational, ok := payload["server_operational"].(map[string]any)
	if !ok {
		t.Fatalf("server_operational raiz ausente: %#v", payload["server_operational"])
	}
	if _, ok := operational["state"].(string); !ok {
		t.Fatalf("server_operational sin state: %#v", operational)
	}
	if _, ok := operational["operational"].(bool); !ok {
		t.Fatalf("server_operational sin operational bool: %#v", operational)
	}
	if summary, ok := payload["server_operational_summary"].(string); !ok || strings.TrimSpace(summary) == "" {
		t.Fatalf("server_operational_summary inesperado: %#v", payload["server_operational_summary"])
	}
}

func TestAPIOpenClawOperatorUsaSupervisorCanonicoAunqueGatewaySeaHumano(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() { statusService = prev }()

	if err := db.ConfigSet("openclaw_gateway_operator", "alberto"); err != nil {
		t.Fatalf("config openclaw gateway operator: %v", err)
	}
	statusService = stubStatusService{response: apiStatusResponse{}}

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
	review, ok := payload["review"].(map[string]any)
	if !ok {
		t.Fatalf("review ausente: %s", rec.Body.String())
	}
	if got, _ := review["supervisor"].(string); got != "OpenClaw" {
		t.Fatalf("supervisor openclaw inesperado: got=%q body=%s", got, rec.Body.String())
	}
}

func TestAPIOpenClawOperatorExponeFollowupSidecarCompacto(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() {
		statusService = prev
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()

	snapshot := apiStatusResponse{
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
			{ID: 410, Estado: db.TareaEnProgreso, Agente: "Codex3", Titulo: "activa"},
		},
	}
	statusService = stubStatusService{response: snapshot}
	storeStatusSnapshot(snapshot, time.Now().UTC())

	if _, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
		Supervisor:   resolveSupervisorName(""),
		ProyectoSlug: "orquestador",
		SessionID:    "sess-api-followup",
		ThreadID:     "slice-api-followup-1",
		SubagentName: "OpenClaw-orquestador-implementacion-slice-api",
		SubagentType: "general-purpose",
		Status:       "completed",
		MetadataJSON: `{"source":"pipeline_local_parallel","task_id":530,"task_title":"Frente amplio del control plane","slice_index":1,"slice_total":2,"pipeline_parent_followup_dispatched":true,"pipeline_parent_followup_phase":"revision","pipeline_parent_followup_action":"avanzar_fase","pipeline_parent_followup_git_merge_id":91}`,
	}); err != nil {
		t.Fatalf("upsert supervisor subagent: %v", err)
	}

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
	followup, ok := payload["pipeline_followup"].(map[string]any)
	if !ok || followup["phase"] != "revision" {
		t.Fatalf("pipeline_followup inesperado: %#v", payload["pipeline_followup"])
	}
	if got := int(followup["task_id"].(float64)); got != 530 {
		t.Fatalf("pipeline_followup task_id inesperado: %#v", followup)
	}
	subagentsFollowup, ok := payload["subagentes_followup"].([]any)
	if !ok || len(subagentsFollowup) != 1 {
		t.Fatalf("subagentes_followup inesperado: %#v", payload["subagentes_followup"])
	}
	first, _ := subagentsFollowup[0].(map[string]any)
	if first["source"] != "pipeline_local_parallel" || first["followup_phase"] != "revision" {
		t.Fatalf("subagentes_followup sin fase/source esperadas: %#v", first)
	}
}

func TestAPIOpenClawOperatorConservaGeneradoDelSnapshotReutilizado(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() {
		statusService = prev
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()

	snapshot := apiStatusResponse{
		Generado: "2026-04-20T09:45:00Z",
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		Agentes: []*db.Agente{
			{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasActivas: []tareaLite{
			{ID: 41, Estado: db.TareaEnProgreso, Agente: "Codex4", Titulo: "runtime mailbox/session_resume", Modulo: "cmd"},
		},
	}
	storeStatusSnapshot(snapshot, time.Now().UTC())
	statusService = stubStatusService{err: errStatusFetchTimeout}

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
	statusMap, ok := payload["status"].(map[string]any)
	if !ok {
		t.Fatalf("status openclaw ausente: %s", rec.Body.String())
	}
	if got, _ := statusMap["generado"].(string); got != snapshot.Generado {
		t.Fatalf("generado operator inesperado: got=%q want=%q body=%s", got, snapshot.Generado, rec.Body.String())
	}
	if activos, ok := payload["agentesActivos"].([]any); !ok || len(activos) != 1 {
		t.Fatalf("agentesActivos raiz inesperados: %#v", payload["agentesActivos"])
	}
}

func TestAPIOpenClawOperatorToleraFalloMailboxYMantieneStatusBase(t *testing.T) {
	prepararDBTemporalCmd(t)

	prevStatus := statusService
	prevMailbox := openClawPendingMailboxFetcher
	defer func() {
		statusService = prevStatus
		openClawPendingMailboxFetcher = prevMailbox
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()

	snapshot := apiStatusResponse{
		Generado: "2026-04-29T10:15:00Z",
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex7", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex7", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		Agentes: []*db.Agente{
			{Nombre: "Codex7", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasActivas: []tareaLite{
			{ID: 77, Estado: db.TareaEnProgreso, Agente: "Codex7", Titulo: "slice activa", Modulo: "cmd"},
		},
	}
	statusService = stubStatusService{response: snapshot}
	storeStatusSnapshot(snapshot, time.Now().UTC())
	openClawPendingMailboxFetcher = func(agentes []*db.Agente) ([]apiOpenClawMailboxLite, error) {
		return nil, fmt.Errorf("mailbox temporalmente no disponible")
	}

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
	statusMap, ok := payload["status"].(map[string]any)
	if !ok {
		t.Fatalf("status openclaw ausente: %s", rec.Body.String())
	}
	if got, _ := statusMap["generado"].(string); got != snapshot.Generado {
		t.Fatalf("generado operator inesperado: got=%q want=%q body=%s", got, snapshot.Generado, rec.Body.String())
	}
	if activos, ok := payload["agentesActivos"].([]any); !ok || len(activos) != 1 {
		t.Fatalf("agentesActivos raiz inesperados: %#v body=%s", payload["agentesActivos"], rec.Body.String())
	}
	if tareas, ok := payload["tareasActivas"].([]any); !ok || len(tareas) != 1 {
		t.Fatalf("tareasActivas raiz inesperadas: %#v body=%s", payload["tareasActivas"], rec.Body.String())
	}
	mailboxRoot, ok := payload["mailboxPendiente"].([]any)
	if !ok || len(mailboxRoot) != 0 {
		t.Fatalf("mailboxPendiente raiz deberia degradar a lista vacia: %#v body=%s", payload["mailboxPendiente"], rec.Body.String())
	}
	mailboxStatus, ok := statusMap["mailboxPendiente"].([]any)
	if !ok || len(mailboxStatus) != 0 {
		t.Fatalf("mailboxPendiente status deberia degradar a lista vacia: %#v body=%s", statusMap["mailboxPendiente"], rec.Body.String())
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

func TestAPIOpenClawPipelineExponeFollowupSidecarCompacto(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() {
		statusService = prev
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()

	snapshot := apiStatusResponse{
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
			{ID: 410, Estado: db.TareaEnProgreso, Agente: "Codex3", Titulo: "activa"},
		},
	}
	statusService = stubStatusService{response: snapshot}
	storeStatusSnapshot(snapshot, time.Now().UTC())

	if _, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
		SessionID:    "sess-pipeline-api-followup",
		ThreadID:     "slice-pipeline-api-followup-1",
		SubagentName: "OpenClaw-orquestador-implementacion-slice-pipeline-api",
		SubagentType: "general-purpose",
		Status:       "completed",
		MetadataJSON: `{"source":"pipeline_local_parallel","task_id":530,"task_title":"Frente amplio del control plane","slice_index":1,"slice_total":2,"pipeline_parent_followup_dispatched":true,"pipeline_parent_followup_phase":"revision","pipeline_parent_followup_action":"avanzar_fase","pipeline_parent_followup_git_merge_id":91}`,
	}); err != nil {
		t.Fatalf("upsert supervisor subagent: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/openclaw/pipeline?supervisor=OpenClaw&proyecto=orquestador", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("openclaw pipeline status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("decode pipeline: %v", err)
	}
	followup, ok := payload["followup"].(map[string]any)
	if !ok || followup["phase"] != "revision" {
		t.Fatalf("followup inesperado: %#v", payload["followup"])
	}
	if got := int(followup["task_id"].(float64)); got != 530 {
		t.Fatalf("followup task_id inesperado: %#v", followup)
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

func TestAPIOpenClawOperatorAccionaAgenteYReviewGate(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	taskID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Gate por API OpenClaw",
		Modulo:    "openclaw",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	gateID, err := db.CrearReviewGate(&db.ReviewGate{
		TareaID:        &taskID,
		Estado:         db.ReviewGatePendiente,
		ReviewerAgente: "OpenClaw",
		RequestedBy:    "Codex3",
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recAgent := httptest.NewRecorder()
	reqAgent := httptest.NewRequest(http.MethodPost, "/api/openclaw/operator", strings.NewReader(`{
		"mode":"agent_action",
		"agente":"Codex3",
		"action":"retirar"
	}`))
	reqAgent.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recAgent, reqAgent)
	if recAgent.Code != http.StatusOK {
		t.Fatalf("openclaw operator agent action status=%d body=%s", recAgent.Code, recAgent.Body.String())
	}
	agente, err := db.GetAgente("Codex3")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente == nil || agente.Habilitado {
		t.Fatalf("el agente no quedó retirado: %+v", agente)
	}

	recGate := httptest.NewRecorder()
	reqGate := httptest.NewRequest(http.MethodPost, "/api/openclaw/operator", strings.NewReader(fmt.Sprintf(`{
		"mode":"review_gate_resolve",
		"gate_id":%d,
		"estado":"aprobado",
		"reviewer_agente":"OpenClaw",
		"findings_json":"[{\"severity\":\"baja\",\"title\":\"ok\"}]"
	}`, gateID)))
	reqGate.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recGate, reqGate)
	if recGate.Code != http.StatusOK {
		t.Fatalf("openclaw operator review gate status=%d body=%s", recGate.Code, recGate.Body.String())
	}
	gate, err := db.GetReviewGate(gateID)
	if err != nil {
		t.Fatalf("get review gate: %v", err)
	}
	if gate == nil || gate.Estado != db.ReviewGateAprobado || gate.ResolvedAt == nil {
		t.Fatalf("review gate no quedó resuelta: %+v", gate)
	}
}

func TestAPIOpenClawOperatorBatchGuidanceAplicaMailboxPending(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() { statusService = prev }()

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex3",
		Kind:        "nudge",
		PayloadJSON: `{"texto":"continua"}`,
		Estado:      "pendiente",
	})
	if err != nil {
		t.Fatalf("crear mailbox pendiente: %v", err)
	}

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
	}}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recBatch := httptest.NewRecorder()
	reqBatch := httptest.NewRequest(http.MethodPost, "/api/openclaw/operator", strings.NewReader(`{
		"mode":"batch",
		"batch_kind":"guidance",
		"max_items":1
	}`))
	reqBatch.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recBatch, reqBatch)
	if recBatch.Code != http.StatusOK {
		t.Fatalf("openclaw operator batch guidance status=%d body=%s", recBatch.Code, recBatch.Body.String())
	}

	pendiente := "pendiente"
	items, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar runtime mailbox: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("batch guidance no dreno mailbox pendiente: %+v", items)
	}
	consumido := "consumido"
	items, err = db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &consumido})
	if err != nil {
		t.Fatalf("listar runtime mailbox consumido: %v", err)
	}
	if len(items) != 1 || items[0].ID != msgID {
		t.Fatalf("batch guidance no consumio la mailbox esperada: %+v", items)
	}
}

func TestAPIOpenClawOperatorOperaSubagentes(t *testing.T) {
	prepararDBTemporalCmd(t)

	storeDir := t.TempDir()
	t.Setenv("CLAWD_AGENT_STORE", storeDir)
	launcherPath := filepath.Join(t.TempDir(), "launcher.sh")
	script := `#!/usr/bin/env bash
set -euo pipefail
id="agent-test-api-001"
manifest="${ORQUESTA_SUBAGENT_STORE}/${id}.json"
output="${ORQUESTA_SUBAGENT_STORE}/${id}.md"
mkdir -p "${ORQUESTA_SUBAGENT_STORE}"
printf '# result\n' > "${output}"
cat > "${manifest}" <<JSON
{"agentId":"${id}","name":"${ORQUESTA_SUBAGENT_NAME}","description":"${ORQUESTA_SUBAGENT_DESCRIPTION}","subagentType":"${ORQUESTA_SUBAGENT_TYPE}","model":"${ORQUESTA_SUBAGENT_MODEL}","status":"running","outputFile":"${output}","manifestFile":"${manifest}","createdAt":"2026-04-02T12:00:00Z","startedAt":"2026-04-02T12:00:00Z"}
JSON
printf 'ok\n'
`
	if err := os.WriteFile(launcherPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write launcher: %v", err)
	}
	t.Setenv("ORQUESTA_CLAUDE_SUBAGENT_LAUNCHER", launcherPath)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recLaunch := httptest.NewRecorder()
	reqLaunch := httptest.NewRequest(http.MethodPost, "/api/openclaw/operator", strings.NewReader(`{
		"mode":"subagent_launch",
		"supervisor":"OpenClaw",
		"proyecto":"orquestador",
		"name":"Claude Explore API",
		"description":"explora modulo openclaw",
		"prompt":"resume riesgos y plan",
		"subagent_type":"explore",
		"model":"claude-opus-4-6"
	}`))
	reqLaunch.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recLaunch, reqLaunch)
	if recLaunch.Code != http.StatusOK {
		t.Fatalf("launch subagent API status=%d body=%s", recLaunch.Code, recLaunch.Body.String())
	}

	items, err := db.ListarSupervisorSubagents(db.FiltroSupervisorSubagents{Supervisor: "OpenClaw", ProyectoSlug: "orquestador", Limit: 10})
	if err != nil {
		t.Fatalf("listar subagentes: %v", err)
	}
	if len(items) != 1 || items[0].ThreadID != "agent-test-api-001" {
		t.Fatalf("subagentes inesperados tras launch API: %+v", items)
	}

	recRefresh := httptest.NewRecorder()
	reqRefresh := httptest.NewRequest(http.MethodPost, "/api/openclaw/operator", strings.NewReader(`{
		"mode":"subagent_store_refresh",
		"supervisor":"OpenClaw",
		"proyecto":"orquestador"
	}`))
	reqRefresh.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recRefresh, reqRefresh)
	if recRefresh.Code != http.StatusOK {
		t.Fatalf("refresh store API status=%d body=%s", recRefresh.Code, recRefresh.Body.String())
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
