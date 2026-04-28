/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/propuestasapp"
)

func TestWebDashMuestraEstadoOpenClawNotificaciones(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.ConfigSet("openclaw_gateway_url", "https://openclaw.local/gateway"); err != nil {
		t.Fatalf("config openclaw url: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_operator", "alberto"); err != nil {
		t.Fatalf("config openclaw operator: %v", err)
	}
	if err := db.ConfigSet("telegram_token", "tg-token"); err != nil {
		t.Fatalf("config telegram token: %v", err)
	}
	if err := db.ConfigSet("telegram_chat_id", "12345"); err != nil {
		t.Fatalf("config telegram chat id: %v", err)
	}
	entregaID, err := db.CrearEntregaNotificacion("openclaw_gateway", "https://openclaw.local/gateway", db.EventoNotificacion{
		Tipo:  "mensaje",
		Texto: "hola openclaw",
	})
	if err != nil {
		t.Fatalf("crear entrega notificacion: %v", err)
	}
	if err := db.MarcarEntregaNotificacionFallida(entregaID, "gateway down", time.Now().UTC().Add(time.Minute)); err != nil {
		t.Fatalf("marcar entrega fallida: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("dash status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"OpenClaw / notificaciones", "OpenClaw Gateway", "https://openclaw.local/gateway", "Telegram", "12345", "Entregas recientes", "fallida", "gateway down"} {
		if !strings.Contains(body, token) {
			t.Fatalf("dashboard sin %q:\n%s", token, body)
		}
	}
}

func TestWebOpenClawMuestraOperatorReviewYEntregas(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{Slug: "orquestador", Nombre: "Orquestador", RutaAbs: "/tmp/orquestador", Tipo: db.ProyectoRepo, Activo: true})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "Codex3",
		ProyectoID:        &proyectoID,
		CWD:               "/tmp/orquestador",
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-openclaw-web-agent",
		Host:              "host-openclaw",
	}); err != nil {
		t.Fatalf("iniciar sesion agente: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_url", "https://openclaw.local/gateway"); err != nil {
		t.Fatalf("config openclaw url: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_operator", "OpenClaw"); err != nil {
		t.Fatalf("config openclaw operator: %v", err)
	}
	entregaID, err := db.CrearEntregaNotificacion("openclaw_gateway", "https://openclaw.local/gateway", db.EventoNotificacion{
		Tipo:  "mensaje",
		Texto: "hola openclaw",
	})
	if err != nil {
		t.Fatalf("crear entrega notificacion: %v", err)
	}
	if err := db.MarcarEntregaNotificacionFallida(entregaID, "gateway down", time.Now().UTC().Add(time.Minute)); err != nil {
		t.Fatalf("marcar entrega fallida: %v", err)
	}
	taskID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Operador OpenClaw",
		Modulo:    "openclaw",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(taskID, "Codex3"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(taskID, "Codex3"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	gateID, err := db.CrearReviewGate(&db.ReviewGate{
		TareaID:        &taskID,
		Estado:         db.ReviewGatePendiente,
		ReviewerAgente: "OpenClaw",
		SeverityMax:    "media",
		RequestedBy:    "Codex3",
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}
	if gateID <= 0 {
		t.Fatalf("review gate invalido: %d", gateID)
	}
	if _, err := db.RecordSupervisorThreadTurn(db.RecordSupervisorThreadInput{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
		SessionID:    "sess-openclaw-web",
		ThreadID:     "leader-1",
		Kind:         "leader",
		Mode:         "review",
		Source:       "test",
	}); err != nil {
		t.Fatalf("record supervisor thread: %v", err)
	}
	if _, err := db.UpsertSupervisorPipelineState(db.UpsertSupervisorPipelineStateInput{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
		PipelineName: "autopilot",
		CurrentPhase: "review",
		Status:       "active",
	}); err != nil {
		t.Fatalf("upsert supervisor pipeline: %v", err)
	}
	if _, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
		SessionID:    "sess-openclaw-web",
		ThreadID:     "slice-openclaw-web-1",
		SubagentName: "OpenClaw-orquestador-implementacion-slice-web",
		SubagentType: "general-purpose",
		Status:       "completed",
		MetadataJSON: `{"source":"pipeline_local_parallel","task_id":530,"slice_index":1,"slice_total":2,"pipeline_parent_followup_dispatched":true,"pipeline_parent_followup_phase":"revision","pipeline_parent_followup_action":"avanzar_fase","pipeline_parent_followup_git_merge_id":91}`,
	}); err != nil {
		t.Fatalf("upsert supervisor subagent: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", webHandlerOpenClaw)
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openclaw", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"OpenClaw Operator",
		"Acción siguiente",
		"capacidad libre",
		"cola completa",
		"cola segura",
		"saturados",
		"Integración server-first",
		"__orquesta/healthz",
		"/api/mcp",
		"openclaw-orquesta-api",
		"RUNBOOK_TELEGRAM.md",
		"Reservas preparadas",
		"Review e integración",
		"Worktrees desfasadas",
		"Sin worktrees desfasadas detectadas.",
		"Eventos normalizados del supervisor",
		"Threads y subagentes",
		"Sesiones observadas de agentes",
		"Candidatas para reuse/spawn",
		"Subagentes explícitos",
		"Pipeline del supervisor",
		"sidecar paralelo",
		"follow-up revision",
		"pipeline_local_parallel",
		"slice 1/2",
		"Guidance durable pendiente",
		"autopilot",
		"sess-openclaw-web",
		"sess-openclaw-web-agent",
		"OpenClaw Gateway y notificaciones",
		"gateway down",
		"OpenClaw",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("pagina openclaw sin %q:\n%s", token, body)
		}
	}
	if strings.Contains(body, "Aplicar siguiente acción") {
		t.Fatalf("la página no debería ofrecer aplicar automáticamente una acción no segura:\n%s", body)
	}
	if !strings.Contains(body, "Requiere revisión manual") {
		t.Fatalf("la página debería advertir revisión manual para la siguiente acción:\n%s", body)
	}
	if !strings.Contains(body, "</html>") {
		t.Fatalf("la página openclaw quedó truncada:\n%s", body)
	}
}

func TestWebOpenClawAccionResuelveReviewGate(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	taskID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Review OpenClaw",
		Modulo:    "openclaw",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	gateID, err := db.CrearReviewGate(&db.ReviewGate{
		TareaID:        &taskID,
		RequestedBy:    "Codex3",
		ReviewerAgente: "OpenClaw",
		Estado:         db.ReviewGatePendiente,
		SeverityMax:    "media",
		FindingsJSON:   `[{"severity":"media","title":"falta prueba"}]`,
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind":            {"review_gate"},
		"gate_id":         {strconv.FormatInt(gateID, 10)},
		"estado":          {"aprobado"},
		"reviewer_agente": {"OpenClaw"},
		"findings_json":   {`[{"severity":"baja","title":"ok"}]`},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Review+gate") {
		t.Fatalf("redirect inesperado: %s", location)
	}
	gate, err := db.GetReviewGate(gateID)
	if err != nil {
		t.Fatalf("get review gate: %v", err)
	}
	if gate == nil || gate.Estado != db.ReviewGateAprobado {
		t.Fatalf("gate no resuelto: %#v", gate)
	}
	if gate.ResolvedAt == nil {
		t.Fatalf("gate aprobado sin resolved_at: %#v", gate)
	}
}

func TestWebOpenClawAccionReseteaReanimacionDeAgente(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	reanimarAt := time.Now().UTC().Add(45 * time.Minute)
	if _, err := db.DB.Exec(`
		UPDATE agentes
		SET activo = 0,
		    estado_cuota = 'enfriamiento',
		    motivo_pausa = 'usage limit',
		    reanimar_at = ?,
		    limite_semanal_segundos = 604800,
		    consumo_semanal_segundos = 3600,
		    limite_dia_segundos = 18000,
		    consumo_dia_segundos = 1200
		WHERE nombre = ?`, reanimarAt, "Codex5"); err != nil {
		t.Fatalf("preparar agente: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind":   {"agente"},
		"agente": {"Codex5"},
		"accion": {"reset-reanimacion"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Acci%C3%B3n+reset-reanimacion") {
		t.Fatalf("redirect inesperado: %s", location)
	}
	agente, err := db.GetAgente("Codex5")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente == nil {
		t.Fatalf("agente nil")
	}
	if strings.TrimSpace(agente.EstadoCuota) != "activo" {
		t.Fatalf("estado_cuota no reseteado: %#v", agente)
	}
	if agente.ReanimarAt != nil {
		t.Fatalf("reanmiar_at no limpiado: %#v", agente)
	}
}

func TestWebOpenClawAccionAplicaReplanificacionSupervisor(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() { statusService = prev }()

	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Replanificacion web OpenClaw",
		Descripcion: "Debe reasignarse por la web del supervisor",
		Modulo:      "openclaw",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex2' WHERE id=?`, tareaID); err != nil {
		t.Fatalf("preparar tarea en progreso: %v", err)
	}
	statusService = stubStatusService{response: apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		Agentes: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
	}}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind":     {"supervision_action"},
		"action":   {"replanificar_por_cuota"},
		"target":   {"tarea:" + strconv.FormatInt(tareaID, 10)},
		"assignee": {"Codex3"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Acci%C3%B3n+replanificar_por_cuota") {
		t.Fatalf("redirect inesperado: %s", location)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || *tarea.Agente != "Codex3" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("replanificacion web no aplicada: %+v", tarea)
	}
}

func TestWebOpenClawAccionAplicaRebalanceoReserva(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() { statusService = prev }()

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar Codex4: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Rebalanceo web OpenClaw",
		Descripcion: "Debe moverse una reserva por la web del supervisor",
		Modulo:      "openclaw",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='asignada', agente='Codex4' WHERE id=?`, tareaID); err != nil {
		t.Fatalf("preparar agente de reserva: %v", err)
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
	}}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind":     {"supervision_action"},
		"action":   {"rebalancear_reserva"},
		"target":   {"tarea:" + strconv.FormatInt(tareaID, 10)},
		"assignee": {"Codex3"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "rebalancear_reserva") {
		t.Fatalf("redirect inesperado: %s", location)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || *tarea.Agente != "Codex3" || tarea.Estado != db.TareaAsignada {
		t.Fatalf("rebalanceo web no aplicado: %+v", tarea)
	}
}

func TestWebOpenClawAccionCierraPropuestaRechazada(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	propuestaID, _, err := propuestasService.Create(propuestasapp.CreateProposalInput{
		Codigo:       "OP-601",
		Titulo:       "Cerrar propuesta en web",
		Descripcion:  "Debe cerrarse como rechazada por /openclaw",
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

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind":     {"supervision_action"},
		"action":   {"cerrar_propuesta_rechazada"},
		"target":   {"propuesta:OP-601"},
		"assignee": {"OpenClaw"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "cerrar_propuesta_rechazada") {
		t.Fatalf("redirect inesperado: %s", location)
	}
	propuesta, err := db.GetPropuesta("OP-601")
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}
	if propuesta == nil || propuesta.Estado != db.PropuestaRechazada {
		t.Fatalf("propuesta no quedó rechazada: %+v", propuesta)
	}
}

func TestWebOpenClawAccionAplicaLoteProposal(t *testing.T) {
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
		Codigo:       "OP-603",
		Titulo:       "Cerrar propuesta por lote web",
		Descripcion:  "Debe cerrarse en batch filtrado",
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
		Titulo:      "Reserva web que no debe moverse",
		Descripcion: "No debe entrar en el batch de propuestas",
		Modulo:      "web",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "OpenClaw",
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
			{Codigo: "OP-603", Titulo: "Cerrar propuesta por lote web", Estado: db.PropuestaAbierta, Acuerdo: 1, Desacuerdo: 2, Pendiente: 0},
		},
		TareasActivas: []tareaLite{
			{ID: tareaID, Estado: db.TareaAsignada, Titulo: "Reserva web que no debe moverse", Agente: "Codex3", Modulo: "web"},
			{ID: 2001, Estado: db.TareaEnProgreso, Titulo: "Carga en Codex3", Agente: "Codex3", Modulo: "db"},
			{ID: 2002, Estado: db.TareaEnProgreso, Titulo: "Carga en Codex3 2", Agente: "Codex3", Modulo: "api"},
			{ID: 2003, Estado: db.TareaEnProgreso, Titulo: "Carga en Codex4", Agente: "Codex4", Modulo: "planocontrol"},
		},
	}}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind":       {"supervision_batch"},
		"batch_kind": {"proposal"},
		"max_items":  {"1"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion lote proposal openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); !strings.Contains(location, "proposal") {
		t.Fatalf("redirect inesperado: %s", location)
	}
	propuesta, err := db.GetPropuesta("OP-603")
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
		t.Fatalf("dispatch mezclado en lote web de proposal: %+v", tarea)
	}
}

func TestWebOpenClawAccionAplicaLoteSupervisor(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() { statusService = prev }()

	tareaLibreID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Dispatch web batch",
		Descripcion: "Debe asignarse por lote web",
		Modulo:      "web",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}
	tareaRetenidaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retenida web batch",
		Descripcion: "Debe replanificarse por lote web",
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea retenida: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex2' WHERE id=?`, tareaRetenidaID); err != nil {
		t.Fatalf("preparar retenida: %v", err)
	}

	statusService = stubStatusService{response: apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{},
		Agentes: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaLibre): 1,
		},
		TareasActivas: []tareaLite{
			{ID: tareaRetenidaID, Estado: db.TareaEnProgreso, Titulo: "Retenida web batch", Agente: "Codex2"},
		},
	}}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind":      {"supervision_batch"},
		"max_items": {"2"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion lote openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); !strings.Contains(location, "Aplicadas+2+acciones+seguras+del+supervisor") {
		t.Fatalf("redirect inesperado: %s", location)
	}

	tareaLibre, err := db.GetTarea(tareaLibreID)
	if err != nil {
		t.Fatalf("get tarea libre: %v", err)
	}
	if tareaLibre.Estado != db.TareaEnProgreso || tareaLibre.Agente == nil || *tareaLibre.Agente != "Codex3" {
		t.Fatalf("tarea libre no aplicada por lote web: %+v", tareaLibre)
	}
	tareaRetenida, err := db.GetTarea(tareaRetenidaID)
	if err != nil {
		t.Fatalf("get tarea retenida: %v", err)
	}
	if tareaRetenida.Estado != db.TareaEnProgreso || tareaRetenida.Agente == nil || *tareaRetenida.Agente != "Codex3" {
		t.Fatalf("tarea retenida no aplicada por lote web: %+v", tareaRetenida)
	}
}

func TestWebOpenClawAccionAplicaSiguienteSupervisor(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := statusService
	defer func() { statusService = prev }()

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Siguiente acción web",
		Descripcion: "Debe aplicarse desde la tarjeta principal de OpenClaw",
		Modulo:      "openclaw",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	statusService = stubStatusService{response: apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		Agentes: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
	}}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind": {"supervision_next"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion siguiente openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); !strings.Contains(location, "Aplicada+") {
		t.Fatalf("redirect inesperado: %s", location)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Estado != db.TareaEnProgreso || tarea.Agente == nil || *tarea.Agente != "Codex3" {
		t.Fatalf("siguiente acción web no aplicada: %+v", tarea)
	}
}

func TestWebOpenClawOperaSubagentes(t *testing.T) {
	prepararDBTemporalCmd(t)

	storeDir := t.TempDir()
	t.Setenv("CLAWD_AGENT_STORE", storeDir)
	launcherPath := filepath.Join(t.TempDir(), "launcher.sh")
	script := `#!/usr/bin/env bash
set -euo pipefail
id="agent-test-web-001"
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
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	formLaunch := url.Values{
		"kind":          {"subagent_launch"},
		"proyecto":      {"orquestador"},
		"name":          {"Claude Explore Web"},
		"description":   {"explora web openclaw"},
		"prompt":        {"resume riesgos"},
		"subagent_type": {"explore"},
		"model":         {"claude-opus-4-6"},
	}
	recLaunch := httptest.NewRecorder()
	reqLaunch := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(formLaunch.Encode()))
	reqLaunch.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(recLaunch, reqLaunch)
	if recLaunch.Code != http.StatusSeeOther {
		t.Fatalf("launch subagent web status=%d body=%s", recLaunch.Code, recLaunch.Body.String())
	}
	if location := recLaunch.Header().Get("Location"); !strings.Contains(location, "Subagente+Claude+lanzado") {
		t.Fatalf("redirect launch inesperado: %s", location)
	}

	formRefresh := url.Values{
		"kind":     {"subagent_store_refresh"},
		"proyecto": {"orquestador"},
	}
	recRefresh := httptest.NewRecorder()
	reqRefresh := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(formRefresh.Encode()))
	reqRefresh.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(recRefresh, reqRefresh)
	if recRefresh.Code != http.StatusSeeOther {
		t.Fatalf("refresh subagent web status=%d body=%s", recRefresh.Code, recRefresh.Body.String())
	}
	if location := recRefresh.Header().Get("Location"); !strings.Contains(location, "Store+de+subagentes+refrescado") {
		t.Fatalf("redirect refresh inesperado: %s", location)
	}

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/openclaw", nil)
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("openclaw get status=%d body=%s", recGet.Code, recGet.Body.String())
	}
	body := recGet.Body.String()
	for _, token := range []string{"Store Claude observada", "Lanzar subagente Claude", "Subagentes explícitos", "agent-test-web-001"} {
		if !strings.Contains(body, token) {
			t.Fatalf("openclaw sin %q:\n%s", token, body)
		}
	}
}

func TestWebOpenClawAccionaSubagenteTerminal(t *testing.T) {
	prepararDBTemporalCmd(t)

	subagente, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
		ThreadID:     "sub-done-web-1",
		SubagentName: "Claude Verify Web",
		SubagentType: "verification",
		Status:       "completed",
		OutputPath:   "/tmp/sub-done-web-1/output.md",
	})
	if err != nil {
		t.Fatalf("crear subagente completado: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/openclaw", nil)
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("openclaw get status=%d body=%s", recGet.Code, recGet.Body.String())
	}
	body := recGet.Body.String()
	for _, token := range []string{"Claude Verify Web", "Recoger resultado", "/tmp/sub-done-web-1/output.md"} {
		if !strings.Contains(body, token) {
			t.Fatalf("openclaw sin %q:\n%s", token, body)
		}
	}

	form := url.Values{
		"kind":     {"supervision_action"},
		"action":   {"recoger_resultado_subagente"},
		"target":   {"subagente:" + strconv.FormatInt(subagente.ID, 10)},
		"assignee": {"OpenClaw"},
	}
	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusSeeOther {
		t.Fatalf("accion subagente web status=%d body=%s", recPost.Code, recPost.Body.String())
	}
	if location := recPost.Header().Get("Location"); !strings.Contains(location, "Acci%C3%B3n+recoger_resultado_subagente+aplicada") && !strings.Contains(location, "Acción+recoger_resultado_subagente+aplicada") {
		t.Fatalf("redirect inesperado: %s", location)
	}
}

func TestWebDashMuestraCuentaYVentanasDeCuotaAgente(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := db.IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	remaining := int64(900)
	resetSesion := time.Date(2026, 3, 31, 21, 0, 0, 0, time.UTC)
	inicioSesion := resetSesion.Add(-5 * time.Hour)
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       "5h",
		WindowStartedAt:  &inicioSesion,
		ResetAt:          &resetSesion,
		RemainingSeconds: &remaining,
		BudgetSource:     "runtime",
		RawSnapshotJSON:  `{"account":{"email":"codex1@example.com","username":"codex1_user"}}`,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("dash status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"cuenta: codex1@example.com",
		"usuario: codex1_user",
		"efectivo",
		"sesión",
		"diario",
		"semanal",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("dashboard sin %q:\n%s", token, body)
		}
	}
}
