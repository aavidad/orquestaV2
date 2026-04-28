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
	"strings"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/i18n"
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
	if _, err := db.CreateSharedContextItem(&db.SharedContextItem{
		ProyectoID: &proyectoID,
		Tipo:       "restriccion",
		Titulo:     "Preserve ports",
		Detalle:    "Do not move domain into adapters",
		Peso:       2,
		Origen:     "test",
	}); err != nil {
		t.Fatalf("CreateSharedContextItem: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proyectos?lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerProyectos(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proyectos status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Projects") || !strings.Contains(body, "Project memory and traceability view.") || !strings.Contains(body, "Materialize repository") {
		t.Fatalf("listado de proyectos sin i18n: %s", body)
	}
	if !strings.Contains(body, `action="/proyectos?lang=en"`) {
		t.Fatalf("formulario de materializacion sin lang: %s", body)
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
	if !strings.Contains(body, "Back to projects") || !strings.Contains(body, "External documentation") || !strings.Contains(body, "Voting history") || !strings.Contains(body, "Repository") {
		t.Fatalf("detalle de proyectos sin i18n: %s", body)
	}
	for _, token := range []string{
		`action="/proyectos/orquestador/repo-revisar?lang=en"`,
		`action="/proyectos/orquestador/idiomas?lang=en"`,
		`action="/proyectos/orquestador/contexto-compartido?lang=en"`,
		`name="frontend_stack"`,
		`name="db_engine"`,
		`name="auth_mode"`,
		`name="identity_provider"`,
		`name="deployment_target"`,
		`name="artifact_type"`,
		`name="theme_support"`,
		`name="branding_profiles"`,
		"Recommended stack",
		"Recommended engine",
		"Recommended mode",
		"Recommended deployment",
		"Recommended artifact",
		"If the project already started",
		"Expand project languages",
		"Shared context",
		"Preserve ports",
		`name="funcion_objetivo"`,
		`name="write_set"`,
		`name="modelos_candidatos"`,
		`name="preservar_arquitectura"`,
		`action="/proyectos/orquestador/repo-mejorar?lang=en"`,
		`action="/proyectos/orquestador/operacion?lang=en"`,
		`action="/proyectos/orquestador/autonomia?lang=en"`,
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("detalle de proyectos sin action con lang %q: %s", token, body)
		}
	}
}

func TestWebProyectoSharedContextCreaEntrada(t *testing.T) {
	prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	form := strings.NewReader("tipo=restriccion&titulo=Preservar+hexagonalidad&detalle=No+mover+dominio+a+adaptadores&peso=3&origen=web&agente=Gemma1")
	req := httptest.NewRequest(http.MethodPost, "/proyectos/orquestador/contexto-compartido?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webRouterProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "lang=en") || !strings.Contains(location, "Shared+context+saved") {
		t.Fatalf("redirect shared context inesperado: %s", location)
	}

	items, err := db.ListSharedContextItems(db.FiltroSharedContextItems{
		ProyectoID: &proyectoID,
		Activos:    true,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("listar shared context: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("shared context items=%d", len(items))
	}
	if items[0].Titulo != "Preservar hexagonalidad" || items[0].Agente != "Gemma1" {
		t.Fatalf("shared context inesperado: %+v", items[0])
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

func TestWebProyectoIdiomasAmpliaBacklogIncremental(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	rutaProyecto := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(rutaProyecto, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	if _, err := i18n.MaterializeProjectSkeleton(i18n.ProjectSkeletonSpec{
		RootDir:         rutaProyecto,
		DefaultLanguage: "es",
		Languages:       []string{"es"},
		Domains:         []string{"common", "errors"},
	}); err != nil {
		t.Fatalf("materialize i18n: %v", err)
	}

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaProyecto,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	form := strings.NewReader("idioma=en&idioma=fr&idiomas_extra=de&por=web")
	req := httptest.NewRequest(http.MethodPost, "/proyectos/orquestador/idiomas?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webRouterProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "lang=en") || !strings.Contains(location, "Languages+added+to+backlog") {
		t.Fatalf("redirect idiomas inesperado: %s", location)
	}

	for _, key := range []string{"i18n_expand_en", "documentacion_expand_en", "qa_i18n_expand_en", "i18n_expand_fr", "i18n_expand_de"} {
		if id := db.GetTareaIDBlueprintKey(proyectoID, key); id <= 0 {
			t.Fatalf("faltaba tarea incremental %s", key)
		}
	}
	for _, rel := range []string{"i18n/en/common.json", "i18n/fr/common.json", "i18n/de/common.json"} {
		if _, err := os.Stat(filepath.Join(rutaProyecto, rel)); err != nil {
			t.Fatalf("falta idioma materializado %s: %v", rel, err)
		}
	}
}

func TestWebProyectoDetalleWizardEmbebidoAlineado(t *testing.T) {
	prepararDBTemporalCmd(t)
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proyectos/orquestador?lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerProyectoDetalle(rec, req, "orquestador")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"1. Briefing and brainstorming",
		"2. Core architecture and runtime model",
		"3. Interfaces and access",
		"4. Data and persistence",
		"5. Target platforms",
		"6. Operating systems",
		"7. Regulatory compliance",
		"8. Delivery, operations and quality",
		`name="objetivo_negocio"`,
		`name="usuarios_objetivo"`,
		`name="restricciones"`,
		`name="arquitectura"`,
		`name="api_style"`,
		`name="background_jobs"`,
		`name="queue"`,
		`name="scheduler"`,
		`name="multi_tenant"`,
		`name="feature_flags"`,
		`name="notifications"`,
		`name="integraciones"`,
		`name="idiomas_extra"`,
		"Detailed help",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("wizard embebido sin token %q: %s", token, body)
		}
	}
}

func TestWebNuevaAppRespetaIdiomaYRenderizaWizardGuiado(t *testing.T) {
	prepararDBTemporalCmd(t)
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/nueva-app?lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerNuevaApp(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		`action="/nueva-app?lang=en"`,
		"Core structure",
		"Web stack / UI library",
		"RBAC authorization",
		"Decoupled theme support",
		"Multi-theme / white-label branding",
		"Recommended stack",
		"Active Directory",
		"Offline mode and sync",
		"Recommended engine",
		"Recommended mode",
		"Recommended deployment",
		"Recommended artifact",
		"The filesystem is auxiliary storage",
		"If the project already started",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("wizard sin token %q: %s", token, body)
		}
	}
}

func TestWebNuevaAppPreservaLangEnErrorYExito(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/nueva-app?lang=en", strings.NewReader("nombre=SinProyecto&tipo=web_api"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webHandlerNuevaApp(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status error=%d body=%s", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); !strings.Contains(location, "/nueva-app?") || !strings.Contains(location, "lang=en") || !strings.Contains(location, "Select+a+target+project") {
		t.Fatalf("redirect error inesperado: %s", location)
	}

	form := url.Values{
		"proyecto_slug":     {"orquestador"},
		"nombre":            {"Portal interno"},
		"descripcion":       {"Portal con SSO y reporting"},
		"tipo":              {"web_api"},
		"frontend":          {"1"},
		"api":               {"1"},
		"auth":              {"1"},
		"db":                {"1"},
		"idioma":            {"es", "en"},
		"arquitectura":      {"clean"},
		"auth_mode":         {"sso"},
		"identity_provider": {"active_directory"},
		"rbac":              {"1"},
		"db_engine":         {"postgres"},
		"background_jobs":   {"1"},
		"queue":             {"1"},
		"reporting":         {"1"},
		"rate_limiting":     {"1"},
		"por":               {"web"},
	}
	req = httptest.NewRequest(http.MethodPost, "/nueva-app?lang=en", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	webHandlerNuevaApp(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status exito=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/tareas?") || !strings.Contains(location, "lang=en") || !strings.Contains(location, "ok=") {
		t.Fatalf("redirect exito inesperado: %s", location)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) == 0 {
		t.Fatalf("se esperaban tareas generadas")
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
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "task_reassigned",
		Actor:     "orquesta",
		ProjectID: &proyectoID,
		TaskID:    &tareaID,
		Source:    "control_plane",
		Reason:    "worker_degradado",
		StateDelta: map[string]any{
			"agente":         "Codex3",
			"agente_destino": "Codex4",
		},
		ArtifactsRef: []string{"runtime_checkpoint:41", "runtime_checkpoint:42", "runtime_checkpoint:43", "runtime_checkpoint:44"},
	}); err != nil {
		t.Fatalf("registrar autonomy event: %v", err)
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
		!strings.Contains(body, "Riesgo integración:") ||
		!strings.Contains(body, "alto") ||
		!strings.Contains(body, "integracion_bloqueada=6") ||
		!strings.Contains(body, "causas review_gates=1 | mailbox_rt=1") ||
		!strings.Contains(body, "task_reassigned") ||
		!strings.Contains(body, "worker_degradado") ||
		!strings.Contains(body, "Codex3") ||
		!strings.Contains(body, "runtime_checkpoint:41") ||
		!strings.Contains(body, "(+1)") {
		t.Fatalf("detalle sin cockpit operativo: %s", body)
	}
}

func TestWebProyectosMuestraCockpitOperativoLigero(t *testing.T) {
	prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaActivaID, err := db.CrearTarea(&db.Tarea{
		ProyectoID: &proyectoID,
		Titulo:     "Cerrar cockpit",
		Modulo:     "web",
		Prioridad:  db.PrioridadAlta,
	})
	if err != nil {
		t.Fatalf("crear tarea activa: %v", err)
	}
	tareaReservadaID, err := db.CrearTarea(&db.Tarea{
		ProyectoID: &proyectoID,
		Titulo:     "Reservar backlog",
		Modulo:     "api",
		Prioridad:  db.PrioridadMedia,
	})
	if err != nil {
		t.Fatalf("crear tarea reservada: %v", err)
	}
	tareaBloqueadaID, err := db.CrearTarea(&db.Tarea{
		ProyectoID: &proyectoID,
		Titulo:     "Bloqueo integración",
		Modulo:     "api",
		Prioridad:  db.PrioridadAlta,
	})
	if err != nil {
		t.Fatalf("crear tarea bloqueada: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex3' WHERE id=?`, tareaActivaID); err != nil {
		t.Fatalf("activar tarea en progreso: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='asignada', agente='Codex4' WHERE id=?`, tareaReservadaID); err != nil {
		t.Fatalf("activar tarea asignada: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='bloqueada', agente='Codex3' WHERE id=?`, tareaBloqueadaID); err != nil {
		t.Fatalf("activar tarea bloqueada: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "worker_recovery_requested",
		Actor:     "orquesta",
		ProjectID: &proyectoID,
		Source:    "control_plane",
		Reason:    "runtime_degradado",
		StateDelta: map[string]any{
			"agente": "Codex3",
		},
	}); err != nil {
		t.Fatalf("registrar autonomy event listado: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:     "Codex3",
		ProyectoID: &proyectoID,
	}); err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proyectos", nil)
	rec := httptest.NewRecorder()
	webHandlerProyectos(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proyectos status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Cockpit",
		"agentes",
		"activas",
		"reservadas",
		"proposals 0",
		"mailbox 0",
		"orders 0",
		"autonomy 1",
		"riesgo alto",
		"integracion_bloqueada=5",
		"causas bloqueadas=1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("listado sin resumen operativo %q: %s", want, body)
		}
	}
}

func TestWebWorkspaceControlMuestraResumenGlobal(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := db.CrearTarea(&db.Tarea{
		ProyectoID: &proyectoID,
		Titulo:     "Workspace control",
		Estado:     db.TareaEnProgreso,
		Agente:     stringPtr("Codex1"),
		Modulo:     "orquestador",
		Prioridad:  db.PrioridadAlta,
	}); err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "task_reassigned",
		Actor:     "orquesta",
		ProjectID: &proyectoID,
		Source:    "control_plane",
		Reason:    "worker_degradado",
		StateDelta: map[string]any{
			"agente_destino": "Codex1",
		},
	}); err != nil {
		t.Fatalf("RegistrarAutonomyEvent: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/workspace/control", nil)
	rec := httptest.NewRecorder()
	webHandlerWorkspaceControl(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("workspace control status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"Control global del workspace", "Proyectos activos", "Autonomía reciente", "Timeline", "Resumen por proyecto", "orquestador", "task_reassigned", "task_reassigned=1"} {
		if !strings.Contains(body, token) {
			t.Fatalf("falta %q en workspace control: %s", token, body)
		}
	}
}

func TestWebWorkspaceControlRenderizaErrorConDesdeInvalido(t *testing.T) {
	prepararDBTemporalCmd(t)

	req := httptest.NewRequest(http.MethodGet, "/workspace/control?desde=xxx", nil)
	rec := httptest.NewRecorder()
	webHandlerWorkspaceControl(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("workspace control status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"Control global del workspace", "valor --desde inválido", "workspace"} {
		if !strings.Contains(body, token) {
			t.Fatalf("falta %q en workspace control con desde invalido: %s", token, body)
		}
	}
}

func TestWebWorkspaceControlNoConfundeRiesgoConAutonomiaVisible(t *testing.T) {
	prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "infra",
		Nombre:  "Infra",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		ProyectoID:  &proyectoID,
		Titulo:      "Merge bloqueado",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "test",
		Descripcion: "solo riesgo operativo, sin eventos de autonomia",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado=? WHERE id=?`, string(db.TareaBloqueada), tareaID); err != nil {
		t.Fatalf("forzar tarea bloqueada: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/workspace/control", nil)
	rec := httptest.NewRecorder()
	webHandlerWorkspaceControl(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("workspace control status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "1 proyecto(s) con riesgo/autonomía visible") {
		t.Fatalf("texto de resumen workspace inesperado: %s", body)
	}
	if !strings.Contains(body, "Riesgo y autonomía recientes") {
		t.Fatalf("la web no destaca el bloque mixto de riesgo/autonomia: %s", body)
	}
	if strings.Contains(body, "Autonomía reciente</h3>") {
		t.Fatalf("la web sigue rotulando como autonomia reciente un bloque solo de riesgo: %s", body)
	}
	if strings.Contains(body, "1 proyecto(s) con autonomía visible") {
		t.Fatalf("la web sigue confundiendo riesgo con autonomia visible: %s", body)
	}
}

func TestWebCargarWorkspaceControlPorAPIPropagaDesde(t *testing.T) {
	prepararDBTemporalCmd(t)
	report, err := webCargarWorkspaceControlPorAPI(time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("webCargarWorkspaceControlPorAPI: %v", err)
	}
	if report == nil || !report.Since.Equal(time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("workspace control inesperado: %+v", report)
	}
}

func TestWebProyectosMaterializaRepoPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
	repo := filepath.Join(workspace, "repo-web")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola web\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")

	form := strings.NewReader("path=" + url.QueryEscape(repo))
	req := httptest.NewRequest(http.MethodPost, "/proyectos?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webHandlerProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/proyectos/repo-web?") || !strings.Contains(location, "ok=") {
		t.Fatalf("redirect materializar inesperado: %s", location)
	}
	if !strings.Contains(location, "lang=en") || !strings.Contains(location, "Repo+materialized") {
		t.Fatalf("redirect materializar sin lang/i18n: %s", location)
	}
	proyecto, err := db.GetProyecto("repo-web")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if proyecto == nil || proyecto.OrigenRepo != "local" || filepath.Clean(proyecto.RutaAbs) != filepath.Clean(repo) {
		t.Fatalf("proyecto materializado inesperado: %+v", proyecto)
	}
}

func TestWebProyectoRepoRevisarPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
	repo := filepath.Join(workspace, "repo-revisar-web")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola review\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")
	resultado, err := repoAddServer(repo, "", "", "")
	if err != nil {
		t.Fatalf("repoAddServer: %v", err)
	}
	if resultado == nil || resultado.Proyecto == nil {
		t.Fatalf("repoAddServer sin proyecto")
	}

	form := strings.NewReader("plan=1")
	req := httptest.NewRequest(http.MethodPost, "/proyectos/"+resultado.Proyecto.Slug+"/repo-revisar?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webRouterProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status revisar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/proyectos/"+resultado.Proyecto.Slug+"?") || !strings.Contains(location, "ok=") || !strings.Contains(location, "lang=en") || !strings.Contains(location, "Review+planned") {
		t.Fatalf("redirect revisar inesperado: %s", location)
	}
}

func TestWebProyectoRepoMejorarPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
	repo := filepath.Join(workspace, "repo-mejorar-web")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola improve\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")
	resultado, err := repoAddServer(repo, "", "", "")
	if err != nil {
		t.Fatalf("repoAddServer: %v", err)
	}
	if resultado == nil || resultado.Proyecto == nil {
		t.Fatalf("repoAddServer sin proyecto")
	}

	form := strings.NewReader("titulo=Mejora+web&descripcion=desde+detalle&modulo=web&prioridad=alta&creado_por=web&funcion_objetivo=web.Render&write_set=cmd/proyectos_web.go,cmd/proyectos_web_test.go&modelos_candidatos=qwen,gemma&preservar_arquitectura=1&despachar=1")
	req := httptest.NewRequest(http.MethodPost, "/proyectos/"+resultado.Proyecto.Slug+"/repo-mejorar?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webRouterProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status mejorar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/proyectos/"+resultado.Proyecto.Slug+"?") || !strings.Contains(location, "ok=") || !strings.Contains(location, "lang=en") || !strings.Contains(location, "Improvement+created+and+dispatched") {
		t.Fatalf("redirect mejorar inesperado: %s", location)
	}
	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &resultado.Proyecto.ID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	found := false
	for _, tarea := range tareas {
		if tarea != nil && tarea.Titulo == "Mejora web" {
			for _, token := range []string{"fork_funcion_v1:", "funcion_objetivo: web.Render", "write_set: cmd/proyectos_web.go, cmd/proyectos_web_test.go", "modelos_candidatos: qwen, gemma", "preservar_arquitectura: true"} {
				if !strings.Contains(tarea.Notas, token) {
					t.Fatalf("tarea mejorar web sin token %q en notas:\n%s", token, tarea.Notas)
				}
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no se encontró tarea sembrada en el proyecto")
	}
}

func TestWebProyectosMaterializaRepoRemotoPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
	remoteParent := t.TempDir()
	remoteRepo := filepath.Join(remoteParent, "repo-remoto-web")
	if err := os.MkdirAll(remoteRepo, 0o755); err != nil {
		t.Fatalf("mkdir repo remoto: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "init")
	runGitCmdAPITest(t, remoteRepo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, remoteRepo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(remoteRepo, "README.md"), []byte("hola remoto web\n"), 0o644); err != nil {
		t.Fatalf("write readme remoto: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "add", "README.md")
	runGitCmdAPITest(t, remoteRepo, "commit", "-m", "init")

	form := strings.NewReader("git=" + url.QueryEscape(remoteRepo))
	req := httptest.NewRequest(http.MethodPost, "/proyectos?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webHandlerProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/proyectos/repo-remoto-web?") || !strings.Contains(location, "ok=") {
		t.Fatalf("redirect materializar remoto inesperado: %s", location)
	}
	if !strings.Contains(location, "lang=en") || !strings.Contains(location, "Repo+materialized") {
		t.Fatalf("redirect materializar remoto sin lang/i18n: %s", location)
	}
	proyecto, err := db.GetProyecto("repo-remoto-web")
	if err != nil {
		t.Fatalf("get proyecto remoto: %v", err)
	}
	if proyecto == nil || proyecto.OrigenRepo != "git" || proyecto.RemoteURL != remoteRepo {
		t.Fatalf("proyecto remoto materializado inesperado: %+v", proyecto)
	}
}

func TestWebProyectosMaterializaRepoErrorPreservaLangYErr(t *testing.T) {
	prepararDBTemporalCmd(t)
	form := strings.NewReader("path=%2Ftmp%2Fno-git")
	req := httptest.NewRequest(http.MethodPost, "/proyectos?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webHandlerProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/proyectos?") || !strings.Contains(location, "lang=en") || !strings.Contains(location, "err=") {
		t.Fatalf("redirect de error sin lang/err: %s", location)
	}
}

func TestWebProyectoRepoMejorarErrorPreservaLangYErr(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
	repo := filepath.Join(workspace, "repo-mejorar-error-web")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola improve error\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")
	resultado, err := repoAddServer(repo, "", "", "")
	if err != nil {
		t.Fatalf("repoAddServer: %v", err)
	}
	if resultado == nil || resultado.Proyecto == nil {
		t.Fatalf("repoAddServer sin proyecto")
	}

	form := strings.NewReader("descripcion=sin+titulo&modulo=web&prioridad=alta&creado_por=web&despachar=1")
	req := httptest.NewRequest(http.MethodPost, "/proyectos/"+resultado.Proyecto.Slug+"/repo-mejorar?lang=en", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webRouterProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/proyectos/"+resultado.Proyecto.Slug+"?") || !strings.Contains(location, "lang=en") || !strings.Contains(location, "err=") {
		t.Fatalf("redirect de error mejorar sin lang/err: %s", location)
	}
}
