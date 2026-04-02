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
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestWebProyectosRespetaIdiomaDelRequest(t *testing.T) {
	prepararDBTemporalCmd(t)
	var proyectoID int64
	if err := db.DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	if _, err := db.GuardarDecisionProyecto(&db.DecisionProyecto{
		ProyectoID: proyectoID,
		Titulo:     "Arquitectura base",
		Categoria:  "general",
		Solucion:   "Puerto de almacenamiento",
	}); err != nil {
		t.Fatalf("GuardarDecisionProyecto: %v", err)
	}
	if _, err := db.GuardarDocumentoExterno(&db.DocumentoExterno{
		ProyectoID:    proyectoID,
		Titulo:        "ADR",
		TipoDocumento: "markdown",
		RutaRef:       "/tmp/adr.md",
		Resumen:       "Resumen",
	}); err != nil {
		t.Fatalf("GuardarDocumentoExterno: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proyectos?lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerProyectos(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proyectos status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Projects") || !strings.Contains(body, "Project memory and traceability view.") {
		t.Fatalf("listado de proyectos sin i18n: %s", body)
	}
	if !strings.Contains(body, `<html lang="en">`) {
		t.Fatalf("lang html inesperado: %s", body)
	}
	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language=%q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/proyectos/orquestador?lang=en", nil)
	rec = httptest.NewRecorder()
	webHandlerProyectoDetalle(rec, req, "orquestador")
	if rec.Code != http.StatusOK {
		t.Fatalf("proyecto detalle status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Back to projects") || !strings.Contains(body, "External documentation") || !strings.Contains(body, "Voting history") {
		t.Fatalf("detalle de proyectos sin i18n: %s", body)
	}
}

func TestWebProyectoFabricarAppGeneraBacklog(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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

	form := strings.NewReader("tipo=web_api&nombre=Orquestador&descripcion=Panel+de+control&frontend=1&api=1&docker=1&i18n=1&idiomas=es,en&por=web")
	req := httptest.NewRequest(http.MethodPost, "/proyectos/orquestador/fabricar-app", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webRouterProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) < 4 {
		t.Fatalf("se esperaban tareas generadas, got=%d", len(tareas))
	}
	backlog := 0
	for _, tarea := range tareas {
		if tarea != nil && tarea.Estado == db.EstadoBacklog {
			backlog++
		}
	}
	if backlog == 0 {
		t.Fatalf("se esperaba backlog dependiente, tareas=%+v", tareas)
	}
}

func TestWebProyectoOperacionRenderizaYGuardaPorAPI(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "flujo normal",
		ObjetivoPct:      100,
		Prioridad:        80,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proyectos/orquestador?lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerProyectoDetalle(rec, req, "orquestador")
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Operational policy") || !strings.Contains(body, "Waiting for human") || !strings.Contains(body, "Auto resume") {
		t.Fatalf("detalle de operacion sin i18n esperada: %s", body)
	}

	form := strings.NewReader("estado_operativo=esperando_humano&motivo=pendiente+de+validacion&objetivo_pct=60&prioridad=250&min_agentes=1&max_agentes=2&resume_automatico=1")
	req = httptest.NewRequest(http.MethodPost, "/proyectos/orquestador/operacion?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	webRouterProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("guardar operacion status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Operation+policy+saved") {
		t.Fatalf("redirect inesperado: %s", location)
	}

	operacion, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if operacion.EstadoOperativo != db.ProyectoOperativoEsperandoHumano ||
		operacion.Motivo != "pendiente de validacion" ||
		operacion.ObjetivoPct != 60 ||
		operacion.Prioridad != 250 ||
		operacion.MinAgentes != 1 ||
		operacion.MaxAgentes != 2 ||
		!operacion.ResumeAutomatico {
		t.Fatalf("operacion guardada inesperada: %+v", operacion)
	}
}

func TestWebProyectoAutonomiaYReviewGatesPorAPI(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "cerrar el proyecto sin intervención",
		DefinitionOfDoneJSON: `{"tests":"green"}`,
		MaxWorkers:           3,
		ReserveReviewer:      true,
		ReserveSupervisor:    true,
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if _, err := db.RegistrarAutonomiaCiclo(&db.AutonomiaCiclo{
		ProyectoID:   proyectoID,
		Kind:         "supervision",
		Agente:       "Codex3",
		InputJSON:    `{"reason":"periodic"}`,
		DecisionJSON: `{"decision":"seguir"}`,
		Resultado:    "ok",
	}); err != nil {
		t.Fatalf("registrar ciclo autonomia: %v", err)
	}
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReview",
		Estado:         db.ReviewGatePendiente,
		SeverityMax:    "medium",
		FindingsJSON:   `[{"kind":"coverage"}]`,
	}); err != nil {
		t.Fatalf("crear review gate: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proyectos/orquestador?lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerProyectoDetalle(rec, req, "orquestador")
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Project autonomy") || !strings.Contains(body, "Review gates") || !strings.Contains(body, "Recent cycles") {
		t.Fatalf("detalle sin secciones de autonomia/review: %s", body)
	}

	form := strings.NewReader("enabled=1&estado_autonomia=activo&objetivo_general=seguir+autonomamente&definition_of_done_json=%7B%22tests%22%3A%22green%22%7D&max_workers=4&supervisor_agente=CodexSupervisor&reviewer_agente=CodexReview&reserve_reviewer=1&reserve_supervisor=1&review_required=1&auto_create_tasks=1&auto_close_project=1")
	req = httptest.NewRequest(http.MethodPost, "/proyectos/orquestador/autonomia?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	webRouterProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("guardar autonomia status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Autonomy+policy+saved") {
		t.Fatalf("redirect autonomia inesperado: %s", location)
	}

	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.MaxWorkers != 4 || strings.TrimSpace(policy.ObjetivoGeneral) != "seguir autonomamente" {
		t.Fatalf("autonomia guardada inesperada: %+v", policy)
	}
	if policy.SupervisorAgente != "CodexSupervisor" || policy.ReviewerAgente != "CodexReview" {
		t.Fatalf("agentes preferidos en web inesperados: %+v", policy)
	}
}

func TestWebProyectoDetalleMuestraCockpitOperativo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Codex3", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Integrar cockpit",
		ProyectoID:  &proyectoID,
		Modulo:      "web",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "detalle",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex3"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex3"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    "Codex3",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{}`,
	}); err != nil {
		t.Fatalf("encolar mailbox: %v", err)
	}
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGatePendiente,
	}); err != nil {
		t.Fatalf("crear review gate: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proyectos/orquestador", nil)
	rec := httptest.NewRecorder()
	webHandlerProyectoDetalle(rec, req, "orquestador")
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Cockpit operativo") ||
		!strings.Contains(body, "agentes activos") ||
		!strings.Contains(body, "guidance durable pendiente") ||
		!strings.Contains(body, "Codex3") {
		t.Fatalf("detalle sin cockpit operativo: %s", body)
	}
}
