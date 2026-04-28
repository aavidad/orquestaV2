/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/fabricaapp"
	"orquesta/memoriaproyecto"
	"orquesta/reviewapp"
	"orquesta/supervisionapp"
)

var memoriaProyectoService = memoriaproyecto.NewService(db.ProjectMemoryRepository{})

type webProyectoResumen struct {
	Slug    string
	Nombre  string
	Tipo    string
	RutaAbs string
	Activo  bool
	Cockpit *apiProyectoCockpit
}

type webProyectosData struct {
	Proyectos []webProyectoResumen
	Msg       string
	Err       string
}

type webWorkspaceControlData struct {
	Control *workspaceControlReport
	Msg     string
	Err     string
}

type webProyectoDetalleData struct {
	Proyecto               *db.Proyecto
	Cockpit                *apiProyectoCockpit
	IntegrationRisk        string
	IntegrationRiskScore   int
	IntegrationHighlights  []string
	Operacion              *db.ProyectoOperacion
	Autonomia              *supervisionapp.Policy
	Ciclos                 []*supervisionapp.Cycle
	ReviewGates            []*reviewapp.Gate
	Votaciones             []*db.HistorialVotacionProyecto
	Decisiones             []*db.DecisionProyecto
	Documentos             []*db.DocumentoExterno
	SharedContext          []*db.SharedContextItem
	SharedContextSummary   string
	Idiomas                []webIdiomaOption
	DBRecommendation       webDBRecommendation
	FrontendRecommendation webFieldRecommendation
	AuthRecommendation     webFieldRecommendation
	DeployRecommendation   webFieldRecommendation
	ArtifactRecommendation webFieldRecommendation
	Msg                    string
	Err                    string
}

func webRouterProyectos(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/proyectos", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerProyectoDetalle(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "operacion" && r.Method == http.MethodPost:
		webHandlerProyectoOperacionGuardar(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "autonomia" && r.Method == http.MethodPost:
		webHandlerProyectoAutonomiaGuardar(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "fabricar-app" && r.Method == http.MethodPost:
		webHandlerProyectoFabricarApp(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "idiomas" && r.Method == http.MethodPost:
		webHandlerProyectoIdiomas(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "contexto-compartido" && r.Method == http.MethodPost:
		webHandlerProyectoSharedContext(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "repo-revisar" && r.Method == http.MethodPost:
		webHandlerProyectoRepoRevisar(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "repo-mejorar" && r.Method == http.MethodPost:
		webHandlerProyectoRepoMejorar(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "decisiones" && r.Method == http.MethodPost:
		webHandlerProyectoDecisionNueva(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "documentacion" && r.Method == http.MethodPost:
		webHandlerProyectoDocumentoNuevo(w, r, parts[1])
	case len(parts) == 5 && parts[2] == "review-gates" && parts[4] == "resolver" && r.Method == http.MethodPost:
		webHandlerProyectoReviewGateResolver(w, r, parts[1], parts[3])
	default:
		http.NotFound(w, r)
	}
}

func webHandlerProyectos(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		webHandlerProyectoRepoMaterializar(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	var activaPtr *bool
	if raw := strings.TrimSpace(r.URL.Query().Get("activa")); raw != "" {
		value := raw == "1" || strings.EqualFold(raw, "true")
		activaPtr = &value
	}
	items, err := webCargarProyectosPorAPI(activaPtr)
	if err != nil {
		webRender(w, r, webTplLayout+webTplProyectos, webProyectosData{
			Err: err.Error(),
		})
		return
	}
	var proyectos []webProyectoResumen
	for _, item := range items {
		cockpit, errCockpit := webCargarProyectoCockpitPorAPI(item.Slug)
		if errCockpit != nil {
			cockpit = nil
		}
		proyectos = append(proyectos, webProyectoResumen{
			Slug:    item.Slug,
			Nombre:  item.Nombre,
			Tipo:    string(item.Tipo),
			RutaAbs: item.RutaAbs,
			Activo:  item.Activo,
			Cockpit: cockpit,
		})
	}
	webRender(w, r, webTplLayout+webTplProyectos, webProyectosData{
		Proyectos: proyectos,
		Msg:       r.URL.Query().Get("ok"),
		Err:       r.URL.Query().Get("err"),
	})
}

func webHandlerWorkspaceControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	since, err := parseStatsSince(strings.TrimSpace(r.URL.Query().Get("desde")))
	if err != nil {
		webRender(w, r, webTplLayout+webTplWorkspaceControl, webWorkspaceControlData{
			Err: err.Error(),
		})
		return
	}
	control, err := webCargarWorkspaceControlPorAPI(since)
	if err != nil {
		webRender(w, r, webTplLayout+webTplWorkspaceControl, webWorkspaceControlData{
			Err: err.Error(),
		})
		return
	}
	webRender(w, r, webTplLayout+webTplWorkspaceControl, webWorkspaceControlData{
		Control: control,
		Msg:     r.URL.Query().Get("ok"),
		Err:     r.URL.Query().Get("err"),
	})
}

func webHandlerProyectoRepoMaterializar(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webProyectosRedirectURL(r, "err", err.Error()), http.StatusSeeOther)
		return
	}
	resp, err := webMaterializarRepoPorAPI(apiRepoMaterializarRequest{
		Path:    strings.TrimSpace(r.FormValue("path")),
		Git:     strings.TrimSpace(r.FormValue("git")),
		Branch:  strings.TrimSpace(r.FormValue("branch")),
		Destino: strings.TrimSpace(r.FormValue("destino")),
	})
	if err != nil {
		http.Redirect(w, r, webProyectosRedirectURL(r, "err", err.Error()), http.StatusSeeOther)
		return
	}
	if resp == nil || resp.Proyecto == nil || strings.TrimSpace(resp.Proyecto.Slug) == "" {
		http.Redirect(w, r, webProyectosRedirectURL(r, "err", "materialización sin proyecto"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, resp.Proyecto.Slug, "ok", webTranslateRequestf(r, "projects.flash.repo_materialized", resp.Proyecto.Slug)), http.StatusSeeOther)
}

func webHandlerProyectoDetalle(w http.ResponseWriter, r *http.Request, slug string) {
	overview, err := webCargarProyectoOverviewPorAPI(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cockpit, errCockpit := webCargarProyectoCockpitPorAPI(slug)
	operacion, err := webCargarProyectoOperacionPorAPI(slug)
	autonomia, errAutonomia := webCargarProyectoAutonomiaPorAPI(slug)
	sharedContext, errSharedContext := webListarContextoCompartidoProyectoPorAPI(slug, "", "", 12)
	reviewGates, errReviewGates := webListarReviewGatesPorAPI(slug, "", 20)
	if err != nil {
		webRender(w, r, webTplLayout+webTplProyectoDetalle, webProyectoDetalleData{
			Proyecto:   overview.Proyecto,
			Votaciones: overview.Votaciones,
			Decisiones: overview.Decisiones,
			Documentos: overview.Documentos,
			Err:        err.Error(),
		})
		return
	}
	var cicloItems []*supervisionapp.Cycle
	var autonomiaItem *supervisionapp.Policy
	var sharedContextItems []*db.SharedContextItem
	var sharedContextSummary string
	if errAutonomia == nil && autonomia != nil {
		autonomiaItem = autonomia.Policy
		cicloItems = autonomia.Cycles
	}
	if errSharedContext == nil && sharedContext != nil {
		sharedContextItems = sharedContext.Items
		sharedContextSummary = sharedContext.Summary
	}
	integrationRiskScore := 0
	integrationRisk := ""
	var integrationHighlights []string
	if cockpit != nil && (strings.TrimSpace(cockpit.IntegrationRisk) != "" || cockpit.IntegrationRiskScore > 0 || len(cockpit.IntegrationHighlights) > 0) {
		integrationRiskScore = cockpit.IntegrationRiskScore
		integrationRisk = strings.TrimSpace(cockpit.IntegrationRisk)
		integrationHighlights = compactProjectControlIntegrationHighlights(cockpit.IntegrationHighlights)
	} else {
		integrationRiskScore, integrationRisk, integrationHighlights = projectControlIntegrationRisk(cockpit, nil)
		integrationHighlights = compactProjectControlIntegrationHighlights(integrationHighlights)
	}
	var errParts []string
	if errAutonomia != nil {
		errParts = append(errParts, errAutonomia.Error())
	}
	if errCockpit != nil {
		errParts = append(errParts, errCockpit.Error())
	}
	if errReviewGates != nil {
		errParts = append(errParts, errReviewGates.Error())
	}
	if errSharedContext != nil {
		errParts = append(errParts, errSharedContext.Error())
	}
	webRender(w, r, webTplLayout+webTplProyectoDetalle, webProyectoDetalleData{
		Proyecto:               overview.Proyecto,
		Cockpit:                cockpit,
		IntegrationRisk:        integrationRisk,
		IntegrationRiskScore:   integrationRiskScore,
		IntegrationHighlights:  integrationHighlights,
		Operacion:              operacion,
		Autonomia:              autonomiaItem,
		Ciclos:                 cicloItems,
		ReviewGates:            reviewGates,
		Votaciones:             overview.Votaciones,
		Decisiones:             overview.Decisiones,
		Documentos:             overview.Documentos,
		SharedContext:          sharedContextItems,
		SharedContextSummary:   sharedContextSummary,
		Idiomas:                webNuevaAppIdiomaOptions(),
		DBRecommendation:       webNuevaAppDefaultDBRecommendation(r),
		FrontendRecommendation: webNuevaAppDefaultFrontendRecommendation(r),
		AuthRecommendation:     webNuevaAppDefaultAuthRecommendation(r),
		DeployRecommendation:   webNuevaAppDefaultDeploymentRecommendation(r),
		ArtifactRecommendation: webNuevaAppDefaultArtifactRecommendation(r),
		Msg:                    r.URL.Query().Get("ok"),
		Err:                    strings.TrimSpace(strings.Join(append([]string{r.URL.Query().Get("err")}, errParts...), " · ")),
	})
}

func webHandlerProyectoOperacionGuardar(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	req := apiProyectoOperacionSetRequest{
		EstadoOperativo:  strings.TrimSpace(r.FormValue("estado_operativo")),
		Motivo:           strings.TrimSpace(r.FormValue("motivo")),
		ObjetivoPct:      parseIntForm(r.FormValue("objetivo_pct"), 100),
		MinAgentes:       parseIntForm(r.FormValue("min_agentes"), 0),
		MaxAgentes:       parseIntForm(r.FormValue("max_agentes"), 0),
		Prioridad:        parseIntForm(r.FormValue("prioridad"), 100),
		ResumeAutomatico: webFormBool(r, "resume_automatico"),
	}
	if _, err := webGuardarProyectoOperacionPorAPI(slug, req); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", webTranslateRequestf(r, "projects.flash.operation_saved")), http.StatusSeeOther)
}

func webHandlerProyectoAutonomiaGuardar(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	req := apiProyectoAutonomiaSaveRequest{
		Enabled:              webFormBool(r, "enabled"),
		ObjetivoGeneral:      strings.TrimSpace(r.FormValue("objetivo_general")),
		DefinitionOfDoneJSON: strings.TrimSpace(r.FormValue("definition_of_done_json")),
		MaxWorkers:           parseIntForm(r.FormValue("max_workers"), 0),
		SupervisorAgente:     strings.TrimSpace(r.FormValue("supervisor_agente")),
		ReviewerAgente:       strings.TrimSpace(r.FormValue("reviewer_agente")),
		ReserveReviewer:      webFormBool(r, "reserve_reviewer"),
		ReserveSupervisor:    webFormBool(r, "reserve_supervisor"),
		ReviewRequired:       webFormBool(r, "review_required"),
		AutoCreateTasks:      webFormBool(r, "auto_create_tasks"),
		AutoCloseProject:     webFormBool(r, "auto_close_project"),
		EstadoAutonomia:      strings.TrimSpace(r.FormValue("estado_autonomia")),
	}
	if _, err := webGuardarProyectoAutonomiaPorAPI(slug, req); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", webTranslateRequestf(r, "projects.flash.autonomy_saved")), http.StatusSeeOther)
}

func webHandlerProyectoReviewGateResolver(w http.ResponseWriter, r *http.Request, slug string, gateID string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(gateID), 10, 64)
	if err != nil || id <= 0 {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", "gate inválida"), http.StatusSeeOther)
		return
	}
	req := apiReviewGateResolveRequest{
		Estado:         strings.TrimSpace(r.FormValue("estado")),
		ReviewerAgente: strings.TrimSpace(r.FormValue("reviewer_agente")),
		FindingsJSON:   strings.TrimSpace(r.FormValue("findings_json")),
	}
	if _, err := webResolverReviewGatePorAPI(id, req); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", webTranslateRequestf(r, "projects.flash.review_gate_saved")), http.StatusSeeOther)
}

func webHandlerProyectoDecisionNueva(w http.ResponseWriter, r *http.Request, slug string) {
	_ = r.ParseForm()
	if _, err := webCrearDecisionProyectoPorAPI(slug, apiProyectoDecisionCreateRequest{
		Categoria:    r.FormValue("categoria"),
		Titulo:       r.FormValue("titulo"),
		Solucion:     r.FormValue("solucion"),
		Motivo:       r.FormValue("motivo"),
		Alternativas: r.FormValue("alternativas"),
		Impacto:      r.FormValue("impacto"),
		Estado:       r.FormValue("estado"),
		PropuestaID:  parseOptionalInt64(r.FormValue("propuesta_id")),
		TareaID:      parseOptionalInt64(r.FormValue("tarea_id")),
	}); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", webTranslateRequestf(r, "projects.flash.decision_saved")), http.StatusSeeOther)
}

func webHandlerProyectoDocumentoNuevo(w http.ResponseWriter, r *http.Request, slug string) {
	_ = r.ParseForm()
	if _, err := webCrearDocumentoProyectoPorAPI(slug, apiProyectoDocumentoCreateRequest{
		TipoDocumento: r.FormValue("tipo_documento"),
		Titulo:        r.FormValue("titulo"),
		RutaRef:       r.FormValue("ruta_ref"),
		Resumen:       r.FormValue("resumen"),
		Estado:        r.FormValue("estado"),
		Fuente:        r.FormValue("fuente"),
		PropuestaID:   parseOptionalInt64(r.FormValue("propuesta_id")),
		TareaID:       parseOptionalInt64(r.FormValue("tarea_id")),
	}); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", webTranslateRequestf(r, "projects.flash.document_saved")), http.StatusSeeOther)
}

func webHandlerProyectoFabricarApp(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	actor := strings.TrimSpace(r.FormValue("por"))
	if actor == "" {
		actor = "web"
	}
	if _, err := webFabricarAppProyectoPorAPI(slug, apiProyectoFabricarAppRequest{
		Nombre:           strings.TrimSpace(r.FormValue("nombre")),
		Descripcion:      strings.TrimSpace(r.FormValue("descripcion")),
		ObjetivoNegocio:  strings.TrimSpace(r.FormValue("objetivo_negocio")),
		UsuariosObjetivo: strings.TrimSpace(r.FormValue("usuarios_objetivo")),
		Restricciones:    strings.TrimSpace(r.FormValue("restricciones")),
		Tipo:             strings.TrimSpace(r.FormValue("tipo")),
		Frontend:         webFormBool(r, "frontend"),
		API:              webFormBool(r, "api"),
		Auth:             webFormBool(r, "auth"),
		Database:         webFormBool(r, "db"),
		Docker:           webFormBool(r, "docker"),
		I18n:             webFormBool(r, "i18n"),
		Idiomas:          collectProyectoFactoryIdiomas(r),
		Por:              actor,

		PlatWeb:      webFormBool(r, "plat_web"),
		PlatDesktop:  webFormBool(r, "plat_desktop"),
		PlatMobile:   webFormBool(r, "plat_mobile"),
		PlatCLI:      webFormBool(r, "plat_cli"),
		PlatEmbedded: webFormBool(r, "plat_embedded"),

		SOLinux:   webFormBool(r, "so_linux"),
		SOWindows: webFormBool(r, "so_windows"),
		SOmacOS:   webFormBool(r, "so_macos"),
		SOAndroid: webFormBool(r, "so_android"),
		SOiOS:     webFormBool(r, "so_ios"),

		ComplianceRGPD:          webFormBool(r, "compliance_rgpd"),
		ComplianceENS:           webFormBool(r, "compliance_ens"),
		ComplianceLSSI:          webFormBool(r, "compliance_lssi"),
		ComplianceWCAG:          webFormBool(r, "compliance_wcag"),
		ComplianceFacturaElec:   webFormBool(r, "compliance_factura_elec"),
		ComplianceReutilizacion: webFormBool(r, "compliance_reutilizacion"),

		CI:                 webFormBool(r, "ci"),
		Kubernetes:         webFormBool(r, "kubernetes"),
		Terraform:          webFormBool(r, "terraform"),
		Monitoring:         webFormBool(r, "monitoring"),
		Arquitectura:       strings.TrimSpace(r.FormValue("arquitectura")),
		APIStyle:           strings.TrimSpace(r.FormValue("api_style")),
		FrontendStack:      strings.TrimSpace(r.FormValue("frontend_stack")),
		AuthMode:           strings.TrimSpace(r.FormValue("auth_mode")),
		IdentityProvider:   strings.TrimSpace(r.FormValue("identity_provider")),
		TestingLevel:       strings.TrimSpace(r.FormValue("testing_level")),
		ObservabilityLevel: strings.TrimSpace(r.FormValue("observability_level")),
		DeploymentTarget:   strings.TrimSpace(r.FormValue("deployment_target")),
		ArtifactType:       strings.TrimSpace(r.FormValue("artifact_type")),
		DatabaseEngine:     strings.TrimSpace(r.FormValue("db_engine")),
		BackgroundJobs:     webFormBool(r, "background_jobs"),
		Notifications:      webFormBool(r, "notifications"),
		MultiTenant:        webFormBool(r, "multi_tenant"),
		RBAC:               webFormBool(r, "rbac"),
		ThemeSupport:       webFormBool(r, "theme_support"),
		BrandingProfiles:   webFormBool(r, "branding_profiles"),
		OfflineMode:        webFormBool(r, "offline_mode"),
		ImportExport:       webFormBool(r, "import_export"),
		Webhooks:           webFormBool(r, "webhooks"),
		FileUploads:        webFormBool(r, "file_uploads"),
		Reporting:          webFormBool(r, "reporting"),
		ServicioResidente:  webFormBool(r, "servicio_residente"),
		Cache:              webFormBool(r, "cache"),
		Queue:              webFormBool(r, "queue"),
		Scheduler:          webFormBool(r, "scheduler"),
		ObjectStorage:      webFormBool(r, "object_storage"),
		Search:             webFormBool(r, "search"),
		RateLimiting:       webFormBool(r, "rate_limiting"),
		FeatureFlags:       webFormBool(r, "feature_flags"),
		AuditTrail:         webFormBool(r, "audit_trail"),
		Backups:            webFormBool(r, "backups"),
		DisasterRecovery:   webFormBool(r, "disaster_recovery"),
		Integraciones:      splitCSV(strings.TrimSpace(r.FormValue("integraciones"))),
	}); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", webTranslateRequestf(r, "projects.flash.factory_created")), http.StatusSeeOther)
}

func collectProyectoFactoryIdiomas(r *http.Request) []string {
	idiomas := splitCSV(strings.TrimSpace(r.FormValue("idiomas")))
	if extra := strings.TrimSpace(r.FormValue("idiomas_extra")); extra != "" {
		idiomas = append(idiomas, splitCSV(extra)...)
	}
	return splitCSV(strings.Join(idiomas, ","))
}

func collectProyectoIdiomaExpansion(r *http.Request) []string {
	idiomas := append([]string(nil), r.Form["idioma"]...)
	if extra := strings.TrimSpace(r.FormValue("idiomas_extra")); extra != "" {
		idiomas = append(idiomas, splitCSV(extra)...)
	}
	return splitCSV(strings.Join(idiomas, ","))
}

func webHandlerProyectoIdiomas(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	resp, err := webAgregarIdiomasProyectoPorAPI(slug, apiProyectoIdiomasRequest{
		Idiomas: collectProyectoIdiomaExpansion(r),
		Por:     valorConFallback(strings.TrimSpace(r.FormValue("por")), "web"),
	})
	if err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	if resp == nil || !resp.OK {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", "ampliacion de idiomas sin respuesta util"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", webTranslateRequestf(r, "projects.flash.languages_expanded", strings.Join(resp.Idiomas, ", "))), http.StatusSeeOther)
}

func webHandlerProyectoSharedContext(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	resp, err := webAnotarContextoCompartidoProyectoPorAPI(slug, apiProyectoSharedContextCreateRequest{
		Agente:      strings.TrimSpace(r.FormValue("agente")),
		Tipo:        valorConFallback(strings.TrimSpace(r.FormValue("tipo")), "decision"),
		Titulo:      strings.TrimSpace(r.FormValue("titulo")),
		Detalle:     strings.TrimSpace(r.FormValue("detalle")),
		PayloadJSON: strings.TrimSpace(r.FormValue("payload_json")),
		Peso:        parseFloat64Form(r.FormValue("peso"), 1),
		Origen:      valorConFallback(strings.TrimSpace(r.FormValue("origen")), "web"),
		ExpiresAt:   strings.TrimSpace(r.FormValue("expires_at")),
	})
	if err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	if resp == nil || !resp.OK || resp.ID <= 0 {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", "contexto compartido sin respuesta util"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", webTranslateRequestf(r, "projects.flash.shared_context_saved", resp.ID)), http.StatusSeeOther)
}

func webHandlerProyectoRepoRevisar(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	planOnly := webFormBool(r, "plan")
	resp, err := webRevisarRepoPorAPI(apiRepoRevisarRequest{
		Proyecto: slug,
		Plan:     planOnly,
	})
	if err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	if resp == nil || resp.Proyecto == nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", "revisión sin proyecto"), http.StatusSeeOther)
		return
	}
	msg := webTranslateRequestf(r, "projects.flash.repo_review_planned", slug)
	if !planOnly {
		msg = webTranslateRequestf(r, "projects.flash.repo_review_dispatched", slug)
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", msg), http.StatusSeeOther)
}

func webHandlerProyectoRepoMejorar(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	despachar := webFormBool(r, "despachar")
	resp, err := webMejorarRepoPorAPI(apiRepoMejorarRequest{
		Proyecto:              slug,
		Titulo:                strings.TrimSpace(r.FormValue("titulo")),
		Descripcion:           strings.TrimSpace(r.FormValue("descripcion")),
		Modulo:                strings.TrimSpace(r.FormValue("modulo")),
		Prioridad:             strings.TrimSpace(r.FormValue("prioridad")),
		CreadoPor:             valorConFallback(strings.TrimSpace(r.FormValue("creado_por")), "web"),
		Notas:                 strings.TrimSpace(r.FormValue("notas")),
		FuncionObjetivo:       strings.TrimSpace(r.FormValue("funcion_objetivo")),
		WriteSet:              splitCSV(strings.TrimSpace(r.FormValue("write_set"))),
		ModelosCandidatos:     splitCSV(strings.TrimSpace(r.FormValue("modelos_candidatos"))),
		PreservarArquitectura: r.FormValue("preservar_arquitectura") != "0",
		Despachar:             repoBoolPtr(despachar),
	})
	if err != nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", err.Error()), http.StatusSeeOther)
		return
	}
	if resp == nil || resp.Proyecto == nil || resp.Tarea == nil {
		http.Redirect(w, r, webProyectoRedirectURL(r, slug, "err", "mejora sin tarea"), http.StatusSeeOther)
		return
	}
	msg := webTranslateRequestf(r, "projects.flash.repo_improve_created", resp.Tarea.ID)
	if despachar {
		msg = webTranslateRequestf(r, "projects.flash.repo_improve_dispatched", resp.Tarea.ID)
	}
	http.Redirect(w, r, webProyectoRedirectURL(r, slug, "ok", msg), http.StatusSeeOther)
}

func webFabricarAppProyectoPorAPI(ref string, req apiProyectoFabricarAppRequest) (*apiProyectoFabricarAppResponse, error) {
	var resp apiProyectoFabricarAppResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/fabricar-app"
	if err := webInvocarAPIJSON(http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func webAgregarIdiomasProyectoPorAPI(ref string, req apiProyectoIdiomasRequest) (*apiProyectoIdiomasResponse, error) {
	var resp apiProyectoIdiomasResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/idiomas"
	if err := webInvocarAPIJSON(http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func webListarContextoCompartidoProyectoPorAPI(ref, agente, tipo string, limit int) (*apiProyectoSharedContextResponse, error) {
	resp, ok, err := listarContextoCompartidoProyectoPorAPI(ref, agente, tipo, limit)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, serverFirstCommandError("proyecto contexto-compartido listar")
	}
	if resp == nil {
		return &apiProyectoSharedContextResponse{}, nil
	}
	return resp, nil
}

func webAnotarContextoCompartidoProyectoPorAPI(ref string, req apiProyectoSharedContextCreateRequest) (*apiProyectoSharedContextMutationResponse, error) {
	resp, ok, err := anotarContextoCompartidoProyectoPorAPI(ref, req)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, serverFirstCommandError("proyecto contexto-compartido anotar")
	}
	return resp, nil
}

func webMaterializarRepoPorAPI(req apiRepoMaterializarRequest) (*apiRepoMaterializarResponse, error) {
	var resp apiRepoMaterializarResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/repos/materializar", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func webRevisarRepoPorAPI(req apiRepoRevisarRequest) (*apiRepoRevisarResponse, error) {
	var resp apiRepoRevisarResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/repos/revisar", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func webMejorarRepoPorAPI(req apiRepoMejorarRequest) (*apiRepoMejorarResponse, error) {
	var resp apiRepoMejorarResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/repos/mejorar", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func webCargarProyectoOverviewPorAPI(ref string) (*memoriaproyecto.ProjectOverview, error) {
	var resp apiProyectoOverviewResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/overview"
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Overview, nil
}

func webCargarProyectoCockpitPorAPI(ref string) (*apiProyectoCockpit, error) {
	var resp apiProyectoCockpitResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/cockpit"
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Cockpit, nil
}

func webCargarWorkspaceControlPorAPI(since time.Time) (*workspaceControlReport, error) {
	var resp apiWorkspaceControlResponse
	path := "/api/workspace/control"
	if !since.IsZero() {
		query := url.Values{}
		query.Set("desde", since.UTC().Format(time.RFC3339))
		path += "?" + query.Encode()
	}
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Control, nil
}

func webCrearDecisionProyectoPorAPI(ref string, req apiProyectoDecisionCreateRequest) (int64, error) {
	var resp apiCatalogoMutationResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/decisiones"
	if err := webInvocarAPIJSON(http.MethodPost, path, req, &resp); err != nil {
		return 0, err
	}
	return resp.ID, nil
}

func webCrearDocumentoProyectoPorAPI(ref string, req apiProyectoDocumentoCreateRequest) (int64, error) {
	var resp apiCatalogoMutationResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/documentacion"
	if err := webInvocarAPIJSON(http.MethodPost, path, req, &resp); err != nil {
		return 0, err
	}
	return resp.ID, nil
}

func webCargarProyectoOperacionPorAPI(ref string) (*db.ProyectoOperacion, error) {
	var resp apiProyectoOperacionResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/operacion"
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Operacion, nil
}

func webGuardarProyectoOperacionPorAPI(ref string, req apiProyectoOperacionSetRequest) (*db.ProyectoOperacion, error) {
	var resp apiProyectoOperacionResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/operacion"
	if err := webInvocarAPIJSON(http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return resp.Operacion, nil
}

func webCargarProyectoAutonomiaPorAPI(ref string) (*apiProyectoAutonomiaResponse, error) {
	var resp apiProyectoAutonomiaResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/autonomia"
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func webGuardarProyectoAutonomiaPorAPI(ref string, req apiProyectoAutonomiaSaveRequest) (*supervisionapp.Policy, error) {
	var resp apiProyectoAutonomiaResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/autonomia"
	if err := webInvocarAPIJSON(http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return resp.Policy, nil
}

func webListarReviewGatesPorAPI(proyecto, estado string, limit int) ([]*reviewapp.Gate, error) {
	values := url.Values{}
	if strings.TrimSpace(proyecto) != "" {
		values.Set("proyecto", strings.TrimSpace(proyecto))
	}
	if strings.TrimSpace(estado) != "" {
		values.Set("estado", strings.TrimSpace(estado))
	}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	path := "/api/review-gates"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp struct {
		Gates []*reviewapp.Gate `json:"gates"`
	}
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Gates, nil
}

func webResolverReviewGatePorAPI(id int64, req apiReviewGateResolveRequest) (*reviewapp.Gate, error) {
	var resp struct {
		Gate *reviewapp.Gate `json:"gate"`
	}
	path := "/api/review-gates/" + strconv.FormatInt(id, 10) + "/resolver"
	if err := webInvocarAPIJSON(http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return resp.Gate, nil
}

func webFormBool(r *http.Request, key string) bool {
	raw := strings.TrimSpace(r.FormValue(key))
	return raw == "1" || raw == "on" || strings.EqualFold(raw, "true")
}

func webProyectosRedirectURL(r *http.Request, flashKey, flashMsg string) string {
	params := url.Values{}
	params.Set(flashKey, flashMsg)
	params.Set("lang", resolveWebRequestLang(r))
	return "/proyectos?" + params.Encode()
}

func webProyectoRedirectURL(r *http.Request, slug, flashKey, flashMsg string) string {
	params := url.Values{}
	params.Set(flashKey, flashMsg)
	params.Set("lang", resolveWebRequestLang(r))
	return "/proyectos/" + url.PathEscape(strings.TrimSpace(slug)) + "?" + params.Encode()
}

func parseOptionalInt64(raw string) *int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &value
}

func parseIntForm(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parseFloat64Form(raw string, fallback float64) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}

const webTplProyectos = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "projects.title"}} <small style="font-size:.5em;color:#94a3b8">{{len .Proyectos}}</small></h2>
  <p style="color:#64748b">{{tr "projects.subtitle"}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  <p style="margin:0 0 1rem 0"><a href="/workspace/control">Control global del workspace</a></p>
  <article>
    <h3>{{tr "projects.repo.materialize_title"}}</h3>
    <p style="color:#64748b">{{tr "projects.repo.materialize_help"}}</p>
    <form method="post" action="/proyectos?lang={{lang}}">
      <label>{{tr "projects.repo.local_path"}} <input name="path" placeholder="/ruta/al/repo"></label>
      <label>{{tr "projects.repo.remote_git"}} <input name="git" placeholder="git@github.com:org/repo.git"></label>
      <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem">
        <label>{{tr "projects.repo.branch"}} <input name="branch" placeholder="main"></label>
        <label>{{tr "projects.repo.destination"}} <input name="destino" placeholder="/ruta/worktree"></label>
      </div>
      <button type="submit">{{tr "projects.repo.materialize_submit"}}</button>
    </form>
  </article>
  {{if .Proyectos}}
  <table>
    <thead><tr><th>{{tr "Proyecto"}}</th><th>{{tr "Tipo"}}</th><th>{{tr "projects.path"}}</th><th>{{tr "Estado"}}</th><th>Cockpit</th></tr></thead>
    <tbody>
      {{range .Proyectos}}
      <tr>
        <td><a href="/proyectos/{{.Slug}}"><strong>{{.Nombre}}</strong></a><br><small>{{.Slug}}</small></td>
        <td>{{.Tipo}}</td>
        <td><code>{{.RutaAbs}}</code></td>
        <td>{{if .Activo}}{{tr "projects.active"}}{{else}}{{tr "projects.inactive"}}{{end}}</td>
        <td>
          {{if .Cockpit}}
          <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.35rem;min-width:18rem">
            <div style="border:1px solid var(--pico-muted-border-color);border-radius:.4rem;padding:.45rem"><strong>{{len .Cockpit.AgentesActivos}}</strong><br><small>agentes</small></div>
            <div style="border:1px solid var(--pico-muted-border-color);border-radius:.4rem;padding:.45rem"><strong>{{len .Cockpit.TareasActivas}}</strong><br><small>activas</small></div>
            <div style="border:1px solid var(--pico-muted-border-color);border-radius:.4rem;padding:.45rem"><strong>{{len .Cockpit.TareasReservadas}}</strong><br><small>reservadas</small></div>
            <div style="border:1px solid var(--pico-muted-border-color);border-radius:.4rem;padding:.45rem"><strong>{{.Cockpit.ReviewGatesAbiertas}}</strong><br><small>review</small></div>
          </div>
          <small style="display:block;color:#64748b;margin-top:.35rem">
            proposals {{.Cockpit.PropuestasAbiertas}} · mailbox {{.Cockpit.RuntimeMailboxPendiente}} · orders {{.Cockpit.RuntimeOrdersAbiertas}}{{if gt .Cockpit.AutonomyEvents 0}} · autonomy {{.Cockpit.AutonomyEvents}}{{end}}
          </small>
          {{if .Cockpit.IntegrationRisk}}
          <small style="display:block;color:#64748b;margin-top:.25rem">
            riesgo {{.Cockpit.IntegrationRisk}} · integracion_bloqueada={{.Cockpit.IntegrationRiskScore}}{{if .Cockpit.IntegrationHighlights}} · causas {{range $i, $item := .Cockpit.IntegrationHighlights}}{{if $i}} | {{end}}{{$item}}{{end}}{{end}}
          </small>
          {{end}}
          {{else}}
          <small style="color:#64748b">Sin resumen operativo</small>
          {{end}}
        </td>
      </tr>
      {{end}}
    </tbody>
  </table>
  {{else}}
  <article>{{tr "projects.none_visible"}}</article>
  {{end}}
</section>
{{end}}`

const webTplWorkspaceControl = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/proyectos">← {{tr "projects.back"}}</a></p>
  <h2 style="margin:0">Control global del workspace</h2>
  <p style="color:#64748b">Vista compacta del workspace reutilizando la API canónica.</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  {{if .Control}}
  <section style="display:grid;gap:1rem">
    <article>
      <h3>Resumen</h3>
      <p>Proyectos activos: <strong>{{.Control.ActiveProjects}}</strong></p>
      <p>Workers: conectados={{.Control.WorkersConectados}} · trabajando={{.Control.WorkersTrabajando}} · supervisores={{.Control.SupervisoresActivos}}</p>
      <p>Dispatch: total={{.Control.DeudaDispatch.Total}} · pending={{.Control.DeudaDispatch.Pendientes}} · notified={{.Control.DeudaDispatch.Notificadas}} · failed={{.Control.DeudaDispatch.Fallidas}} · confirmed={{.Control.DeudaDispatch.WorkConfirmed}}</p>
      <p>Autonomía: supervising={{.Control.Autonomia.Supervisando}} · continuing={{.Control.Autonomia.Continuando}} · pending={{.Control.Autonomia.ContinuidadPendiente}} · confirmed={{.Control.Autonomia.WorkConfirmed}} · events={{.Control.Autonomia.Count}}</p>
    </article>
    <article>
      <h3>Proyectos</h3>
      {{if .Control.Projects}}
      <table>
        <thead><tr><th>Proyecto</th><th>Tareas</th><th>Autonomía</th><th>Runtime</th></tr></thead>
        <tbody>
          {{range .Control.Projects}}
          <tr>
            <td>{{if .Proyecto}}<a href="/proyectos/{{.Proyecto.Slug}}">{{.Proyecto.Slug}}</a>{{else}}—{{end}}</td>
            <td>{{range $k, $v := .TareasPorEstado}}<span>{{$k}}={{$v}}</span> {{end}}</td>
            <td>{{.AutonomyEvents}}</td>
            <td>mailbox={{.RuntimeMailboxPendiente}} · orders={{.RuntimeOrdersAbiertas}}</td>
          </tr>
          {{end}}
        </tbody>
      </table>
      {{else}}
      <p style="color:#64748b">Sin proyectos activos visibles.</p>
      {{end}}
    </article>
    {{if or .Control.AutonomySurface .Control.AutonomyHighlights .Control.AutonomyRecent .Control.AutonomyProjects}}
    <article>
      {{if or .Control.AutonomySurface .Control.AutonomyRecent}}
      <h3>Autonomía reciente</h3>
      {{else}}
      <h3>Riesgo y autonomía recientes</h3>
      {{end}}
      {{if .Control.AutonomySurface}}
      <p>{{.Control.AutonomySurface.Events}} evento(s){{if .Control.AutonomySurface.LastAt}} · último {{.Control.AutonomySurface.LastAt.Format "2006-01-02 15:04:05"}}{{end}}</p>
      {{else if .Control.AutonomyRecent}}
      <p>{{len .Control.AutonomyRecent}} evento(s) recientes visibles</p>
      {{else if .Control.AutonomyProjects}}
      <p>{{len .Control.AutonomyProjects}} proyecto(s) con riesgo/autonomía visible</p>
      {{end}}
      {{if .Control.AutonomyHighlights}}
      <p>{{range $i, $item := .Control.AutonomyHighlights}}{{if $i}} · {{end}}{{$item}}{{end}}</p>
      {{end}}
    {{if .Control.AutonomyRecent}}
      <h4>Timeline</h4>
      <ul>
      {{range .Control.AutonomyRecent}}
        <li>{{.Project}} · {{.Kind}}{{if .TargetAgent}} · destino={{.TargetAgent}}{{else if .Agent}} · agente={{.Agent}}{{end}} · {{.CreatedAt.Format "2006-01-02 15:04:05"}}</li>
      {{end}}
      </ul>
      {{end}}
      {{if .Control.AutonomyProjects}}
      <h4>Resumen por proyecto</h4>
      <ul>
      {{range .Control.AutonomyProjects}}
        <li>{{.Project}} · {{.Events}} evento(s){{if gt .Blocking 0}} · integracion_bloqueada={{.Blocking}}{{end}}{{if .LastAt}} · {{.LastAt.Format "2006-01-02 15:04:05"}}{{end}}{{if .Highlights}} · {{range $i, $item := .Highlights}}{{if $i}} | {{end}}{{$item}}{{end}}{{end}}</li>
      {{end}}
      </ul>
      {{end}}
    </article>
    {{end}}
  </section>
  {{end}}
</section>
{{end}}`

const webTplProyectoDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/proyectos">← {{tr "projects.back"}}</a></p>
  <h2 style="margin:0">{{.Proyecto.Nombre}} <small style="font-size:.5em;color:#94a3b8">{{.Proyecto.Slug}}</small></h2>
  <p style="color:#64748b"><code>{{.Proyecto.RutaAbs}}</code> · {{tr "projects.type_label"}} {{.Proyecto.Tipo}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
</section>

<section class="container grid">
  <article>
    <h3>Cockpit operativo</h3>
    {{if .Cockpit}}
      <div style="display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:.5rem">
        <div style="border:1px solid var(--pico-muted-border-color);border-radius:.5rem;padding:.75rem"><strong>{{len .Cockpit.AgentesActivos}}</strong><br><small>agentes activos</small></div>
        <div style="border:1px solid var(--pico-muted-border-color);border-radius:.5rem;padding:.75rem"><strong>{{len .Cockpit.TareasActivas}}</strong><br><small>tareas en progreso</small></div>
        <div style="border:1px solid var(--pico-muted-border-color);border-radius:.5rem;padding:.75rem"><strong>{{len .Cockpit.TareasReservadas}}</strong><br><small>tareas reservadas</small></div>
        <div style="border:1px solid var(--pico-muted-border-color);border-radius:.5rem;padding:.75rem"><strong>{{.Cockpit.PropuestasAbiertas}}</strong><br><small>propuestas abiertas</small></div>
      </div>
      <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem;margin-top:.5rem">
        <div style="border:1px solid var(--pico-muted-border-color);border-radius:.5rem;padding:.75rem"><strong>{{.Cockpit.ReviewGatesAbiertas}}</strong><br><small>review gates abiertas</small></div>
        <div style="border:1px solid var(--pico-muted-border-color);border-radius:.5rem;padding:.75rem"><strong>{{.Cockpit.RuntimeMailboxPendiente}}</strong><br><small>guidance durable pendiente</small></div>
        <div style="border:1px solid var(--pico-muted-border-color);border-radius:.5rem;padding:.75rem"><strong>{{.Cockpit.RuntimeOrdersAbiertas}}</strong><br><small>runtime orders abiertas</small></div>
      </div>
      <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem;margin-top:.5rem">
        <div style="border:1px solid var(--pico-muted-border-color);border-radius:.5rem;padding:.75rem"><strong>{{len .Cockpit.MailboxPendiente}}</strong><br><small>agentes con mailbox pendiente</small></div>
        <div style="border:1px solid var(--pico-muted-border-color);border-radius:.5rem;padding:.75rem"><strong>{{len .Cockpit.WorktreeDrift}}</strong><br><small>worktrees desfasadas</small></div>
      </div>
      {{if .IntegrationRisk}}
      <p style="margin:.75rem 0 0 0"><strong>Riesgo integración:</strong> {{.IntegrationRisk}} · integracion_bloqueada={{.IntegrationRiskScore}}{{if .IntegrationHighlights}} · causas {{range $i, $item := .IntegrationHighlights}}{{if $i}} | {{end}}{{$item}}{{end}}{{end}}</p>
      {{end}}
      {{if .Cockpit.AgentesActivos}}
      <p style="margin:.75rem 0 0 0"><strong>Agentes activos:</strong>
        {{range $i, $item := .Cockpit.AgentesActivos}}{{if $i}}, {{end}}{{$item.Nombre}}{{if $item.Rol}} <small style="color:#64748b">({{$item.Rol}})</small>{{end}}{{end}}
      </p>
      {{end}}
      {{if .Cockpit.MailboxPendiente}}
      <div style="margin-top:.75rem">
        <strong>Guidance durable por agente</strong>
        <ul style="margin:.35rem 0 0 1rem">
          {{range .Cockpit.MailboxPendiente}}
          <li><strong>{{.Agente}}</strong> — {{.Count}} pendiente(s) · {{.KindsCSV}}{{if .SupervisorActionsCSV}} · acciones {{.SupervisorActionsCSV}}{{end}}{{if gt .OldestAgeMin 0}} · {{.OldestAgeMin}} min{{end}}</li>
          {{end}}
        </ul>
      </div>
      {{end}}
      {{if .Cockpit.WorktreeDrift}}
      <div style="margin-top:.75rem">
        <strong>Worktrees desfasadas</strong>
        <ul style="margin:.35rem 0 0 1rem">
          {{range .Cockpit.WorktreeDrift}}
          <li><strong>{{.Agente}}</strong> — ahead {{.CommitsAhead}} / behind {{.CommitsBehind}}{{if .Dirty}} · {{.DirtySummary}}{{end}}</li>
          {{end}}
        </ul>
      </div>
      {{end}}
      {{if .Cockpit.Autonomy}}
      <div style="margin-top:.75rem">
        <strong>Autonomía reciente</strong>
        <p style="margin:.2rem 0;color:#64748b">{{.Cockpit.AutonomyEvents}} evento(s) · {{if .Cockpit.AutonomyLastAt}}{{.Cockpit.AutonomyLastAt.Format "2006-01-02 15:04:05"}}{{else}}sin actividad{{end}}</p>
        <ul style="margin:.35rem 0 0 1rem">
          {{range .Cockpit.Autonomy}}
          <li><strong>{{.Kind}}</strong>{{if .Agent}} · agente {{.Agent}}{{end}}{{if .TargetAgent}} · destino {{.TargetAgent}}{{end}}{{if .Supervisor}} · supervisor {{.Supervisor}}{{end}}{{if .Reason}} · {{.Reason}}{{end}}{{if .Artifacts}} · artifacts {{range $i, $item := .Artifacts}}{{if $i}}, {{end}}<code>{{$item}}</code>{{end}}{{if gt .ArtifactsMore 0}} (+{{.ArtifactsMore}}){{end}}{{end}}</li>
          {{end}}
        </ul>
      </div>
      {{end}}
    {{else}}
      <p>Sin resumen operativo.</p>
    {{end}}
  </article>

  <article>
    <h3>{{tr "projects.repo.title"}}</h3>
    <p style="color:#64748b">
      {{tr "projects.repo.origin"}} <strong>{{if .Proyecto.OrigenRepo}}{{.Proyecto.OrigenRepo}}{{else}}-{{end}}</strong>
      · {{tr "projects.repo.remote"}} <code>{{if .Proyecto.RemoteURL}}{{.Proyecto.RemoteURL}}{{else}}-{{end}}</code>
      · {{tr "projects.repo.base"}} <strong>{{if .Proyecto.BranchBase}}{{.Proyecto.BranchBase}}{{else}}-{{end}}</strong>
    </p>
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/repo-revisar?lang={{lang}}">
      <label><input type="checkbox" name="plan" value="1" checked> {{tr "projects.repo.review_plan_only"}}</label>
      <button type="submit">{{tr "projects.repo.review_submit"}}</button>
    </form>
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/repo-mejorar?lang={{lang}}" style="margin-top:1rem">
      <label>{{tr "projects.title_label"}} <input name="titulo" required></label>
      <label>{{tr "projects.factory.description"}} <textarea name="descripcion"></textarea></label>
      <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem">
        <label>{{tr "tasks.module"}} <input name="modulo" placeholder="backend"></label>
        <label>{{tr "Prioridad"}}
          <select name="prioridad">
            <option value="media" selected>{{tr "media"}}</option>
            <option value="alta">{{tr "alta"}}</option>
            <option value="baja">{{tr "baja"}}</option>
          </select>
        </label>
      </div>
      <label>{{tr "projects.repo.function_target"}} <input name="funcion_objetivo" placeholder="app.CalcularTotales"></label>
      <label>{{tr "projects.repo.write_set"}} <input name="write_set" placeholder="app/totales.go,app/totales_test.go"></label>
      <label>{{tr "projects.repo.candidate_models"}} <input name="modelos_candidatos" placeholder="qwen, gemma, claude"></label>
      <label><input type="checkbox" name="preservar_arquitectura" value="1" checked> {{tr "projects.repo.preserve_architecture"}}</label>
      <label>{{tr "projects.repo.created_by"}} <input name="creado_por" value="web"></label>
      <label>{{tr "projects.repo.notes"}} <textarea name="notas"></textarea></label>
      <label><input type="checkbox" name="despachar" value="1" checked> {{tr "projects.repo.dispatch_now"}}</label>
      <button type="submit">{{tr "projects.repo.improve_submit"}}</button>
    </form>
  </article>

  <article>
    <h3>{{tr "projects.operation.title"}}</h3>
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/operacion?lang={{lang}}">
      <label>{{tr "projects.operation.state"}}
        <select name="estado_operativo">
          <option value="activo" {{if or (not .Operacion) (eq .Operacion.EstadoOperativo "activo")}}selected{{end}}>{{tr "projects.operation.active"}}</option>
          <option value="esperando_humano" {{if and .Operacion (eq .Operacion.EstadoOperativo "esperando_humano")}}selected{{end}}>{{tr "projects.operation.waiting_human"}}</option>
          <option value="bloqueado_externo" {{if and .Operacion (eq .Operacion.EstadoOperativo "bloqueado_externo")}}selected{{end}}>{{tr "projects.operation.blocked_external"}}</option>
          <option value="cerrado" {{if and .Operacion (eq .Operacion.EstadoOperativo "cerrado")}}selected{{end}}>{{tr "projects.operation.closed"}}</option>
        </select>
      </label>
      <label>{{tr "Motivo"}} <textarea name="motivo">{{if .Operacion}}{{.Operacion.Motivo}}{{end}}</textarea></label>
      <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem">
        <label>{{tr "projects.operation.target_pct"}} <input name="objetivo_pct" inputmode="numeric" value="{{if .Operacion}}{{.Operacion.ObjetivoPct}}{{else}}100{{end}}"></label>
        <label>{{tr "projects.operation.priority"}} <input name="prioridad" inputmode="numeric" value="{{if .Operacion}}{{.Operacion.Prioridad}}{{else}}100{{end}}"></label>
        <label>{{tr "projects.operation.min_agents"}} <input name="min_agentes" inputmode="numeric" value="{{if .Operacion}}{{.Operacion.MinAgentes}}{{else}}0{{end}}"></label>
        <label>{{tr "projects.operation.max_agents"}} <input name="max_agentes" inputmode="numeric" value="{{if .Operacion}}{{.Operacion.MaxAgentes}}{{else}}0{{end}}"></label>
      </div>
      <label><input type="checkbox" name="resume_automatico" value="1" {{if or (not .Operacion) .Operacion.ResumeAutomatico}}checked{{end}}> {{tr "projects.operation.auto_resume"}}</label>
      <button type="submit">{{tr "projects.operation.save"}}</button>
    </form>
  </article>

  <article>
    <h3>{{tr "projects.autonomy.title"}}</h3>
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/autonomia?lang={{lang}}">
      <label><input type="checkbox" name="enabled" value="1" {{if and .Autonomia .Autonomia.Enabled}}checked{{end}}> {{tr "projects.autonomy.enabled"}}</label>
      <label>{{tr "projects.autonomy.state"}}
        <select name="estado_autonomia">
          <option value="activo" {{if or (not .Autonomia) (eq .Autonomia.EstadoAutonomia "activo")}}selected{{end}}>{{tr "projects.autonomy.active"}}</option>
          <option value="esperando_review" {{if and .Autonomia (eq .Autonomia.EstadoAutonomia "esperando_review")}}selected{{end}}>{{tr "projects.autonomy.waiting_review"}}</option>
          <option value="esperando_humano" {{if and .Autonomia (eq .Autonomia.EstadoAutonomia "esperando_humano")}}selected{{end}}>{{tr "projects.autonomy.waiting_human"}}</option>
          <option value="cerrando" {{if and .Autonomia (eq .Autonomia.EstadoAutonomia "cerrando")}}selected{{end}}>{{tr "projects.autonomy.closing"}}</option>
          <option value="cerrado" {{if and .Autonomia (eq .Autonomia.EstadoAutonomia "cerrado")}}selected{{end}}>{{tr "projects.autonomy.closed"}}</option>
        </select>
      </label>
      <label>{{tr "projects.autonomy.goal"}} <textarea name="objetivo_general">{{if .Autonomia}}{{.Autonomia.ObjetivoGeneral}}{{end}}</textarea></label>
      <label>{{tr "projects.autonomy.dod"}} <textarea name="definition_of_done_json">{{if .Autonomia}}{{.Autonomia.DefinitionOfDoneJSON}}{{else}}{}{{end}}</textarea></label>
      <label>{{tr "projects.autonomy.max_workers"}} <input name="max_workers" inputmode="numeric" value="{{if .Autonomia}}{{.Autonomia.MaxWorkers}}{{else}}0{{end}}"></label>
      <label>{{tr "projects.autonomy.supervisor_agent"}} <input name="supervisor_agente" value="{{if .Autonomia}}{{.Autonomia.SupervisorAgente}}{{end}}"></label>
      <label>{{tr "projects.autonomy.reviewer_agent"}} <input name="reviewer_agente" value="{{if .Autonomia}}{{.Autonomia.ReviewerAgente}}{{end}}"></label>
      <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem">
        <label><input type="checkbox" name="reserve_reviewer" value="1" {{if and .Autonomia .Autonomia.ReserveReviewer}}checked{{end}}> {{tr "projects.autonomy.reserve_reviewer"}}</label>
        <label><input type="checkbox" name="reserve_supervisor" value="1" {{if and .Autonomia .Autonomia.ReserveSupervisor}}checked{{end}}> {{tr "projects.autonomy.reserve_supervisor"}}</label>
        <label><input type="checkbox" name="review_required" value="1" {{if or (not .Autonomia) .Autonomia.ReviewRequired}}checked{{end}}> {{tr "projects.autonomy.review_required"}}</label>
        <label><input type="checkbox" name="auto_create_tasks" value="1" {{if or (not .Autonomia) .Autonomia.AutoCreateTasks}}checked{{end}}> {{tr "projects.autonomy.auto_create_tasks"}}</label>
        <label><input type="checkbox" name="auto_close_project" value="1" {{if or (not .Autonomia) .Autonomia.AutoCloseProject}}checked{{end}}> {{tr "projects.autonomy.auto_close_project"}}</label>
      </div>
      <button type="submit">{{tr "projects.autonomy.save"}}</button>
    </form>
    <div style="margin-top:1rem">
      <h4 style="margin:.2rem 0">{{tr "projects.autonomy.cycles"}}</h4>
      {{if .Ciclos}}
        <table>
          <thead><tr><th>{{tr "projects.autonomy.kind"}}</th><th>{{tr "Agente"}}</th><th>{{tr "projects.autonomy.result"}}</th><th>{{tr "projects.autonomy.when"}}</th></tr></thead>
          <tbody>
            {{range .Ciclos}}
            <tr><td>{{.Kind}}</td><td>{{.Agente}}</td><td>{{.Resultado}}</td><td>{{.CreatedAt}}</td></tr>
            {{end}}
          </tbody>
        </table>
      {{else}}
        <p>{{tr "projects.autonomy.cycles_none"}}</p>
      {{end}}
    </div>
  </article>

  <article id="factory-wizard">
    <h3>{{tr "projects.factory.title"}}</h3>
    <p style="font-size:.85rem;color:var(--pico-muted-color)">{{tr "projects.factory.fixed_note"}}</p>

    <label>{{tr "projects.factory.preset"}}
      <select id="factory-preset" onchange="factoryApplyPreset(this.value)">
        <option value="">{{tr "projects.factory.preset.none"}}</option>
        <option value="web-publica">{{tr "projects.factory.preset.web_publica"}}</option>
        <option value="api-interna">{{tr "projects.factory.preset.api_interna"}}</option>
        <option value="cli-devops">{{tr "projects.factory.preset.cli_devops"}}</option>
        <option value="app-movil">{{tr "projects.factory.preset.app_movil"}}</option>
        <option value="embedded">{{tr "projects.factory.preset.embedded"}}</option>
      </select>
    </label>

    <form id="factory-form" method="post" action="/proyectos/{{.Proyecto.Slug}}/fabricar-app?lang={{lang}}">
      <fieldset>
        <legend>1. {{tr "projects.factory.section.brainstorming"}}</legend>
        <label>{{tr "projects.factory.name"}} <input name="nombre" value="{{.Proyecto.Nombre}}" required></label>
        <label>{{tr "projects.factory.description"}} <textarea name="descripcion"></textarea></label>
        <label>{{tr "projects.factory.business_goal"}} <input name="objetivo_negocio"></label>
        <label>{{tr "projects.factory.target_users"}} <input name="usuarios_objetivo"></label>
        <label>{{tr "projects.factory.constraints"}} <textarea name="restricciones"></textarea></label>
        <label>{{tr "projects.factory.type"}}
          <select name="tipo" id="factory-tipo" required>
            <option value="web_api">web_api — {{tr "projects.factory.type.web_api"}}</option>
            <option value="web">web — {{tr "projects.factory.type.web"}}</option>
            <option value="api">api — {{tr "projects.factory.type.api"}}</option>
            <option value="cli">cli — {{tr "projects.factory.type.cli"}}</option>
            <option value="desktop">desktop — {{tr "projects.factory.type.desktop"}}</option>
            <option value="mobile">mobile — {{tr "projects.factory.type.mobile"}}</option>
            <option value="embedded">embedded — {{tr "projects.factory.type.embedded"}}</option>
          </select>
        </label>
      </fieldset>

      <fieldset>
        <legend>2. {{tr "projects.factory.section.architecture"}}</legend>
        <label>{{tr "projects.factory.architecture"}}
          <select name="arquitectura">
            <option value="hexagonal">{{tr "projects.factory.architecture.hexagonal"}}</option>
            <option value="clean">{{tr "projects.factory.architecture.clean"}}</option>
            <option value="layered">{{tr "projects.factory.architecture.layered"}}</option>
            <option value="event_driven">{{tr "projects.factory.architecture.event_driven"}}</option>
          </select>
        </label>
        {{with wizardHelp "arquitectura"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="servicio_residente" value="1"> {{tr "projects.factory.servicio_residente"}}</label>
          <label><input type="checkbox" name="background_jobs" value="1"> {{tr "projects.factory.background_jobs"}}</label>
          <label><input type="checkbox" name="queue" value="1"> {{tr "projects.factory.queue"}}</label>
          <label><input type="checkbox" name="scheduler" value="1"> {{tr "projects.factory.scheduler"}}</label>
          <label><input type="checkbox" name="multi_tenant" value="1"> {{tr "projects.factory.multi_tenant"}}</label>
          <label><input type="checkbox" name="feature_flags" value="1"> {{tr "projects.factory.feature_flags"}}</label>
          <label><input type="checkbox" name="offline_mode" value="1"> {{tr "projects.factory.offline_mode"}}</label>
          <label><input type="checkbox" name="import_export" value="1"> {{tr "projects.factory.import_export"}}</label>
          <label><input type="checkbox" name="reporting" value="1"> {{tr "projects.factory.reporting"}}</label>
        </div>
        <label style="margin-top:.5rem">{{tr "projects.factory.integrations"}} <input name="integraciones" placeholder="payments, crm, email, sso"></label>
        {{with wizardHelp "nucleo_caps"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
      </fieldset>

      <fieldset>
        <legend>3. {{tr "projects.factory.section.interfaces"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="frontend" value="1" checked> {{tr "projects.factory.frontend"}}</label>
          <label><input type="checkbox" name="api" value="1" checked> {{tr "projects.factory.api"}}</label>
          <label><input type="checkbox" name="auth" value="1"> {{tr "projects.factory.auth"}}</label>
          <label><input type="checkbox" name="i18n" value="1" checked> {{tr "projects.factory.i18n"}}</label>
          <label><input type="checkbox" name="notifications" value="1"> {{tr "projects.factory.notifications"}}</label>
          <label><input type="checkbox" name="webhooks" value="1"> {{tr "projects.factory.webhooks"}}</label>
          <label><input type="checkbox" name="file_uploads" value="1"> {{tr "projects.factory.file_uploads"}}</label>
        </div>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem;margin-top:.5rem">
          <label>{{tr "projects.factory.frontend_stack"}}
            <select name="frontend_stack">
              <option value="react">{{tr "projects.factory.frontend_stack.react"}}</option>
              <option value="vue">{{tr "projects.factory.frontend_stack.vue"}}</option>
              <option value="sveltekit">{{tr "projects.factory.frontend_stack.sveltekit"}}</option>
              <option value="nextjs">{{tr "projects.factory.frontend_stack.nextjs"}}</option>
              <option value="astro">{{tr "projects.factory.frontend_stack.astro"}}</option>
              <option value="htmx">{{tr "projects.factory.frontend_stack.htmx"}}</option>
              <option value="server_rendered">{{tr "projects.factory.frontend_stack.server_rendered"}}</option>
              <option value="angular">{{tr "projects.factory.frontend_stack.angular"}}</option>
              <option value="other">{{tr "projects.factory.frontend_stack.other"}}</option>
            </select>
          </label>
          <label>{{tr "projects.factory.api_style"}}
            <select name="api_style">
              <option value="rest">{{tr "projects.factory.api_style.rest"}}</option>
              <option value="graphql">{{tr "projects.factory.api_style.graphql"}}</option>
              <option value="grpc">{{tr "projects.factory.api_style.grpc"}}</option>
              <option value="async">{{tr "projects.factory.api_style.async"}}</option>
              <option value="mixed">{{tr "projects.factory.api_style.mixed"}}</option>
            </select>
          </label>
          {{with wizardHelp "api_style"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          {{with wizardHelp "frontend_stack"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.auth_mode"}}
            <select name="auth_mode">
              <option value="none">{{tr "projects.factory.auth_mode.none"}}</option>
              <option value="session">{{tr "projects.factory.auth_mode.session"}}</option>
              <option value="jwt">{{tr "projects.factory.auth_mode.jwt"}}</option>
              <option value="oauth">{{tr "projects.factory.auth_mode.oauth"}}</option>
              <option value="sso">{{tr "projects.factory.auth_mode.sso"}}</option>
              <option value="api_key">{{tr "projects.factory.auth_mode.api_key"}}</option>
            </select>
          </label>
          {{with wizardHelp "auth_mode"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.identity_provider"}}
            <select name="identity_provider">
              <option value="none">{{tr "projects.factory.identity_provider.none"}}</option>
              <option value="active_directory">{{tr "projects.factory.identity_provider.active_directory"}}</option>
              <option value="ldap">{{tr "projects.factory.identity_provider.ldap"}}</option>
              <option value="entra_id">{{tr "projects.factory.identity_provider.entra_id"}}</option>
              <option value="keycloak">{{tr "projects.factory.identity_provider.keycloak"}}</option>
              <option value="auth0">{{tr "projects.factory.identity_provider.auth0"}}</option>
              <option value="google">{{tr "projects.factory.identity_provider.google"}}</option>
              <option value="other">{{tr "projects.factory.identity_provider.other"}}</option>
            </select>
          </label>
          {{with wizardHelp "identity_provider"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.db_engine"}}
            <select name="db_engine">
              <option value="none">{{tr "projects.factory.db_engine.none"}}</option>
              <option value="postgres">{{tr "projects.factory.db_engine.postgres"}}</option>
              <option value="mysql">{{tr "projects.factory.db_engine.mysql"}}</option>
              <option value="sqlite">{{tr "projects.factory.db_engine.sqlite"}}</option>
              <option value="mongodb">{{tr "projects.factory.db_engine.mongodb"}}</option>
              <option value="redis">{{tr "projects.factory.db_engine.redis"}}</option>
              <option value="other">{{tr "projects.factory.db_engine.other"}}</option>
            </select>
          </label>
          {{with wizardHelp "db_engine"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
        </div>
        <article id="factory-auth-recommendation" style="margin-top:.75rem;padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
          <strong>{{tr "projects.factory.auth_recommendation.title"}}: <span id="factory-auth-recommendation-label">{{.AuthRecommendation.Label}}</span></strong>
          <p id="factory-auth-recommendation-reason" style="margin:.35rem 0 0">{{.AuthRecommendation.Reason}}</p>
          <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.recommendation.manual"}}</small>
        </article>
        <article id="factory-frontend-recommendation" style="margin-top:.75rem;padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
          <strong>{{tr "projects.factory.frontend_recommendation.title"}}: <span id="factory-frontend-recommendation-label">{{.FrontendRecommendation.Label}}</span></strong>
          <p id="factory-frontend-recommendation-reason" style="margin:.35rem 0 0">{{.FrontendRecommendation.Reason}}</p>
          <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.recommendation.manual"}}</small>
        </article>
        <article id="factory-db-recommendation" style="margin-top:.75rem;padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
          <strong>{{tr "projects.factory.db_recommendation.title"}}: <span id="factory-db-recommendation-label">{{.DBRecommendation.Label}}</span></strong>
          <p id="factory-db-recommendation-reason" style="margin:.35rem 0 0">{{.DBRecommendation.Reason}}</p>
          <small style="display:block;margin-top:.35rem;color:#64748b">{{tr "projects.factory.db_recommendation.filesystem_note"}}</small>
          <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.db_recommendation.manual"}}</small>
        </article>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem;margin-top:.5rem">
          <label><input type="checkbox" name="theme_support" value="1" checked> {{tr "projects.factory.theme_support"}}</label>
          <label><input type="checkbox" name="branding_profiles" value="1"> {{tr "projects.factory.branding_profiles"}}</label>
          <label><input type="checkbox" name="rbac" value="1"> {{tr "projects.factory.rbac"}}</label>
          <label><input type="checkbox" name="rate_limiting" value="1"> {{tr "projects.factory.rate_limiting"}}</label>
        </div>
        <label style="margin-top:.5rem">{{tr "projects.factory.languages"}} <input name="idiomas" value="es,en"></label>
        <label style="margin-top:.5rem">{{tr "projects.factory.languages_extra"}} <input name="idiomas_extra" placeholder="nl, pl, ar"></label>
        <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.languages_extra_help"}}</small>
        {{with wizardHelp "interface_caps"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
      </fieldset>

      <fieldset>
        <legend>4. {{tr "projects.factory.section.data"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="db" value="1"> {{tr "projects.factory.db"}}</label>
          <label><input type="checkbox" name="cache" value="1"> {{tr "projects.factory.cache"}}</label>
          <label><input type="checkbox" name="object_storage" value="1"> {{tr "projects.factory.object_storage"}}</label>
          <label><input type="checkbox" name="search" value="1"> {{tr "projects.factory.search"}}</label>
          <label><input type="checkbox" name="audit_trail" value="1"> {{tr "projects.factory.audit_trail"}}</label>
        </div>
        {{with wizardHelp "data_caps"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
      </fieldset>

      <fieldset>
        <legend>5. {{tr "projects.factory.section.platforms"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="plat_web" id="cb-plat-web" value="1"> {{tr "projects.factory.plat_web"}}</label>
          <label><input type="checkbox" name="plat_desktop" id="cb-plat-desktop" value="1"> {{tr "projects.factory.plat_desktop"}}</label>
          <label><input type="checkbox" name="plat_mobile" id="cb-plat-mobile" value="1"> {{tr "projects.factory.plat_mobile"}}</label>
          <label><input type="checkbox" name="plat_cli" id="cb-plat-cli" value="1"> {{tr "projects.factory.plat_cli"}}</label>
          <label><input type="checkbox" name="plat_embedded" id="cb-plat-embedded" value="1"> {{tr "projects.factory.plat_embedded"}}</label>
        </div>
      </fieldset>

      <fieldset>
        <legend>6. {{tr "projects.factory.section.os"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="so_linux" id="cb-so-linux" value="1"> Linux</label>
          <label><input type="checkbox" name="so_windows" id="cb-so-windows" value="1"> Windows</label>
          <label><input type="checkbox" name="so_macos" id="cb-so-macos" value="1"> macOS</label>
          <label><input type="checkbox" name="so_android" id="cb-so-android" value="1"> Android</label>
          <label><input type="checkbox" name="so_ios" id="cb-so-ios" value="1"> iOS</label>
        </div>
      </fieldset>

      <fieldset>
        <legend>7. {{tr "projects.factory.section.compliance"}}</legend>
        <p style="font-size:.8rem;color:var(--pico-muted-color)">{{tr "projects.factory.compliance_note"}}</p>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.75rem">
          <div>
            <label><input type="checkbox" name="compliance_rgpd" value="1"> RGPD</label>
            <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_rgpd_help"}}</small>
          </div>
          <div>
            <label><input type="checkbox" name="compliance_ens" value="1"> ENS</label>
            <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_ens_help"}}</small>
          </div>
          <div>
            <label><input type="checkbox" name="compliance_lssi" value="1"> LSSI</label>
            <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_lssi_help"}}</small>
          </div>
          <div>
            <label><input type="checkbox" name="compliance_wcag" value="1"> WCAG 2.1 AA</label>
            <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_wcag_help"}}</small>
          </div>
          <div>
            <label><input type="checkbox" name="compliance_factura_elec" value="1"> {{tr "projects.factory.compliance_factura_elec"}}</label>
            <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_factura_elec_help"}}</small>
          </div>
          <div>
            <label><input type="checkbox" name="compliance_reutilizacion" value="1"> {{tr "projects.factory.compliance_reutilizacion"}}</label>
            <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_reutilizacion_help"}}</small>
          </div>
        </div>
      </fieldset>

      <fieldset>
        <legend>8. {{tr "projects.factory.section.delivery_quality"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="docker" id="cb-docker" value="1" checked> {{tr "projects.factory.docker"}}</label>
          <label><input type="checkbox" name="ci" value="1" checked> CI/CD — {{tr "projects.factory.ci"}}</label>
          <label><input type="checkbox" name="kubernetes" id="cb-kubernetes" value="1"> Kubernetes — {{tr "projects.factory.kubernetes"}}</label>
          <label><input type="checkbox" name="terraform" value="1"> Terraform — {{tr "projects.factory.terraform"}}</label>
          <label><input type="checkbox" name="monitoring" value="1"> {{tr "projects.factory.monitoring"}}</label>
          <label><input type="checkbox" name="backups" value="1"> {{tr "projects.factory.backups"}}</label>
          <label><input type="checkbox" name="disaster_recovery" value="1"> {{tr "projects.factory.disaster_recovery"}}</label>
        </div>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem;margin-top:.5rem">
          <label>{{tr "projects.factory.deployment_target"}}
            <select name="deployment_target">
              <option value="docker">{{tr "projects.factory.deployment_target.docker"}}</option>
              <option value="kubernetes">{{tr "projects.factory.deployment_target.kubernetes"}}</option>
              <option value="serverless">{{tr "projects.factory.deployment_target.serverless"}}</option>
              <option value="vm">{{tr "projects.factory.deployment_target.vm"}}</option>
              <option value="edge">{{tr "projects.factory.deployment_target.edge"}}</option>
            </select>
          </label>
          {{with wizardHelp "deployment_target"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.artifact_type"}}
            <select name="artifact_type">
              <option value="executable">{{tr "projects.factory.artifact_type.executable"}}</option>
              <option value="docker_image">{{tr "projects.factory.artifact_type.docker_image"}}</option>
              <option value="library">{{tr "projects.factory.artifact_type.library"}}</option>
              <option value="desktop_installer">{{tr "projects.factory.artifact_type.desktop_installer"}}</option>
              <option value="mobile_bundle">{{tr "projects.factory.artifact_type.mobile_bundle"}}</option>
              <option value="firmware">{{tr "projects.factory.artifact_type.firmware"}}</option>
              <option value="static_site">{{tr "projects.factory.artifact_type.static_site"}}</option>
            </select>
          </label>
          {{with wizardHelp "artifact_type"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
        </div>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.75rem;margin-top:.75rem">
          <article id="factory-deployment-recommendation" style="padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
            <strong>{{tr "projects.factory.deployment_recommendation.title"}}: <span id="factory-deployment-recommendation-label">{{.DeployRecommendation.Label}}</span></strong>
            <p id="factory-deployment-recommendation-reason" style="margin:.35rem 0 0">{{.DeployRecommendation.Reason}}</p>
            <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.recommendation.manual"}}</small>
          </article>
          <article id="factory-artifact-recommendation" style="padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
            <strong>{{tr "projects.factory.artifact_recommendation.title"}}: <span id="factory-artifact-recommendation-label">{{.ArtifactRecommendation.Label}}</span></strong>
            <p id="factory-artifact-recommendation-reason" style="margin:.35rem 0 0">{{.ArtifactRecommendation.Reason}}</p>
            <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.recommendation.manual"}}</small>
          </article>
        </div>
        {{with wizardHelp "delivery_caps"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
      </fieldset>

      <label>{{tr "projects.factory.actor"}} <input name="por" value="web"></label>
      <div style="display:flex;gap:.5rem;flex-wrap:wrap">
        <button type="button" class="secondary" onclick="factoryPreview(this)">{{tr "projects.factory.preview_btn"}}</button>
        <button type="submit">{{tr "projects.factory.submit"}}</button>
      </div>
    </form>

    <div id="factory-preview-result" style="display:none;margin-top:1rem">
      <h4>{{tr "projects.factory.preview_title"}} (<span id="factory-preview-count">0</span>)</h4>
      <table style="font-size:.8rem">
        <thead><tr><th>Key</th><th>{{tr "projects.factory.preview_col_title"}}</th><th>{{tr "projects.factory.preview_col_phase"}}</th><th>{{tr "projects.factory.preview_col_module"}}</th></tr></thead>
        <tbody id="factory-preview-tbody"></tbody>
      </table>
    </div>
  </article>

  <article id="project-language-expansion">
    <h3>{{tr "projects.factory.languages_expand_title"}}</h3>
    <p style="font-size:.85rem;color:var(--pico-muted-color)">{{tr "projects.factory.languages_expand_note"}}</p>
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/idiomas?lang={{lang}}">
      <div style="display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:.35rem;margin-top:.35rem">
        {{range .Idiomas}}
        <label><input type="checkbox" name="idioma" value="{{.Code}}"> {{.Label}}</label>
        {{end}}
      </div>
      <label style="margin-top:.5rem">{{tr "projects.factory.languages_extra"}} <input name="idiomas_extra" placeholder="nl, pl, ar"></label>
      <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.languages_expand_help"}}</small>
      <label style="margin-top:.5rem">{{tr "projects.factory.actor"}} <input name="por" value="web"></label>
      <button type="submit">{{tr "projects.factory.languages_expand_submit"}}</button>
    </form>
  </article>

  <article id="project-shared-context">
    <h3>{{tr "projects.shared_context.title"}}</h3>
    <p style="font-size:.85rem;color:var(--pico-muted-color)">{{tr "projects.shared_context.subtitle"}}</p>
    {{if .SharedContextSummary}}
    <p><strong>{{tr "projects.shared_context.summary"}}:</strong> {{.SharedContextSummary}}</p>
    {{end}}
    {{if .SharedContext}}
    <table style="font-size:.8rem">
      <thead>
        <tr>
          <th>ID</th>
          <th>{{tr "projects.shared_context.type"}}</th>
          <th>{{tr "projects.shared_context.agent"}}</th>
          <th>{{tr "projects.shared_context.weight"}}</th>
          <th>{{tr "projects.shared_context.title_col"}}</th>
        </tr>
      </thead>
      <tbody>
        {{range .SharedContext}}
        <tr>
          <td>{{.ID}}</td>
          <td>{{.Tipo}}</td>
          <td>{{if .Agente}}{{.Agente}}{{else}}—{{end}}</td>
          <td>{{printf "%.1f" .Peso}}</td>
          <td>{{.Titulo}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
    {{else}}
    <p>{{tr "projects.shared_context.empty"}}</p>
    {{end}}
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/contexto-compartido?lang={{lang}}">
      <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.75rem">
        <label>{{tr "projects.shared_context.type"}} <input name="tipo" value="decision"></label>
        <label>{{tr "projects.shared_context.agent"}} <input name="agente" placeholder="Gemma1"></label>
      </div>
      <label style="margin-top:.5rem">{{tr "projects.shared_context.title_col"}} <input name="titulo" required></label>
      <label style="margin-top:.5rem">{{tr "projects.shared_context.detail"}} <textarea name="detalle" rows="3"></textarea></label>
      <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.75rem">
        <label>{{tr "projects.shared_context.weight"}} <input name="peso" value="1"></label>
        <label>{{tr "projects.shared_context.origin"}} <input name="origen" value="web"></label>
      </div>
      <button type="submit">{{tr "projects.shared_context.submit"}}</button>
    </form>
  </article>

  <script>
  (function(){
    // ── Dependencias automáticas entre checkboxes ────────────────────────
    function cb(id){ return document.getElementById(id); }
    var form = document.getElementById('factory-form');
  var dbEngineLabels = {
    none: {{printf "%q" (tr "projects.factory.db_engine.none")}},
    sqlite: {{printf "%q" (tr "projects.factory.db_engine.sqlite")}},
    postgres: {{printf "%q" (tr "projects.factory.db_engine.postgres")}}
  };
  var frontendStackLabels = {
    react: {{printf "%q" (tr "projects.factory.frontend_stack.react")}},
    server_rendered: {{printf "%q" (tr "projects.factory.frontend_stack.server_rendered")}}
  };
  var frontendStackReasons = {
    react: {{printf "%q" (tr "projects.factory.frontend_recommendation.reason.react")}},
    server_rendered: {{printf "%q" (tr "projects.factory.frontend_recommendation.reason.server_rendered")}}
  };
  var dbEngineReasons = {
      none: {{printf "%q" (tr "projects.factory.db_recommendation.reason.none")}},
      sqlite: {{printf "%q" (tr "projects.factory.db_recommendation.reason.sqlite")}},
      postgres: {{printf "%q" (tr "projects.factory.db_recommendation.reason.postgres")}}
    };
    var authModeLabels = {
      none: {{printf "%q" (tr "projects.factory.auth_mode.none")}},
      session: {{printf "%q" (tr "projects.factory.auth_mode.session")}},
      jwt: {{printf "%q" (tr "projects.factory.auth_mode.jwt")}},
      oauth: {{printf "%q" (tr "projects.factory.auth_mode.oauth")}},
      sso: {{printf "%q" (tr "projects.factory.auth_mode.sso")}},
      api_key: {{printf "%q" (tr "projects.factory.auth_mode.api_key")}}
    };
    var authModeReasons = {
      none: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.none")}},
      session: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.session")}},
      jwt: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.jwt")}},
      oauth: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.oauth")}},
      sso: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.sso")}},
      api_key: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.api_key")}}
    };
    var deploymentLabels = {
      docker: {{printf "%q" (tr "projects.factory.deployment_target.docker")}},
      kubernetes: {{printf "%q" (tr "projects.factory.deployment_target.kubernetes")}},
      serverless: {{printf "%q" (tr "projects.factory.deployment_target.serverless")}},
      vm: {{printf "%q" (tr "projects.factory.deployment_target.vm")}},
      edge: {{printf "%q" (tr "projects.factory.deployment_target.edge")}}
    };
    var deploymentReasons = {
      docker: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.docker")}},
      kubernetes: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.kubernetes")}},
      serverless: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.serverless")}},
      vm: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.vm")}},
      edge: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.edge")}}
    };
    var artifactLabels = {
      executable: {{printf "%q" (tr "projects.factory.artifact_type.executable")}},
      docker_image: {{printf "%q" (tr "projects.factory.artifact_type.docker_image")}},
      library: {{printf "%q" (tr "projects.factory.artifact_type.library")}},
      desktop_installer: {{printf "%q" (tr "projects.factory.artifact_type.desktop_installer")}},
      mobile_bundle: {{printf "%q" (tr "projects.factory.artifact_type.mobile_bundle")}},
      firmware: {{printf "%q" (tr "projects.factory.artifact_type.firmware")}},
      static_site: {{printf "%q" (tr "projects.factory.artifact_type.static_site")}}
    };
    var artifactReasons = {
      executable: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.executable")}},
      docker_image: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.docker_image")}},
      library: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.library")}},
      desktop_installer: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.desktop_installer")}},
      mobile_bundle: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.mobile_bundle")}},
      firmware: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.firmware")}},
      static_site: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.static_site")}}
    };

    function syncDeps(src, targets, checked){
      targets.forEach(function(id){
        var el = cb(id);
        if(el && checked) el.checked = true;
        // solo desmarcar si ninguna otra plataforma los requiere
        if(el && !checked) {
          var stillNeeded = false;
          depRules.forEach(function(r){
            if(r.targets.indexOf(id) !== -1 && r.src !== src){
              var srcEl = cb(r.src);
              if(srcEl && srcEl.checked) stillNeeded = true;
            }
          });
          if(!stillNeeded) el.checked = false;
        }
      });
    }

    var depRules = [
      {src:'cb-plat-mobile',   targets:['cb-so-android','cb-so-ios']},
      {src:'cb-plat-desktop',  targets:['cb-so-linux','cb-so-windows','cb-so-macos']},
      {src:'cb-plat-embedded', targets:['cb-so-linux']},
      {src:'cb-plat-cli',      targets:['cb-so-linux']},
      {src:'cb-kubernetes',    targets:['cb-docker']},
    ];

    depRules.forEach(function(rule){
      var el = cb(rule.src);
      if(!el) return;
      el.addEventListener('change', function(){
        syncDeps(rule.src, rule.targets, this.checked);
      });
    });

    function fieldValue(name, fallback){
      var el = form ? form.querySelector('[name="'+name+'"]') : null;
      if(!el){ return (fallback||'').toLowerCase(); }
      return String(el.value || fallback || '').toLowerCase().trim();
    }
    function fieldChecked(name){
      return !!(form && form.querySelector('input[name="'+name+'"]:checked'));
    }
    function recommendDBEngine(){
      var type = fieldValue('tipo', 'web_api');
      var authMode = fieldValue('auth_mode', 'none');
      var identityProvider = fieldValue('identity_provider', 'none');
      var deploymentTarget = fieldValue('deployment_target', 'docker');
      var artifactType = fieldValue('artifact_type', 'executable');
      var durable = fieldChecked('db') || fieldChecked('auth') || type === 'api' || type === 'web_api' ||
        authMode !== 'none' || identityProvider !== 'none' || deploymentTarget === 'kubernetes' || artifactType === 'docker_image';
      var serverGrade = type === 'api' || type === 'web_api' || fieldChecked('auth') ||
        authMode === 'oauth' || authMode === 'sso' || authMode === 'api_key' || identityProvider !== 'none' ||
        deploymentTarget === 'kubernetes' || fieldChecked('kubernetes') || artifactType === 'docker_image';
      var embeddedPreferred = (type === 'desktop' || type === 'mobile' || type === 'cli' || type === 'embedded' ||
        fieldChecked('plat_desktop') || fieldChecked('plat_mobile') || fieldChecked('plat_cli') || fieldChecked('plat_embedded')) && !serverGrade;
      if(!durable && !serverGrade && !embeddedPreferred) return 'none';
      if(embeddedPreferred) return 'sqlite';
      if(serverGrade) return 'postgres';
      return 'none';
    }
    function recommendFrontendStack(){
      var type = fieldValue('tipo', 'web_api');
      if(!fieldChecked('frontend') && type !== 'web' && type !== 'web_api') return 'server_rendered';
      if(fieldChecked('offline_mode') || fieldChecked('reporting') || fieldChecked('feature_flags') ||
         fieldChecked('multi_tenant') || fieldChecked('file_uploads') || fieldChecked('branding_profiles')) return 'react';
      if(type === 'web' && !fieldChecked('api') && !fieldChecked('servicio_residente') && !fieldChecked('background_jobs')) return 'server_rendered';
      return 'react';
    }
    function recommendAuthMode(){
      var type = fieldValue('tipo', 'web_api');
      var identityProvider = fieldValue('identity_provider', 'none');
      if(!fieldChecked('auth')) return 'none';
      if(identityProvider !== 'none') return 'sso';
      if(type === 'mobile') return 'oauth';
      if(type === 'cli') return 'api_key';
      if(type === 'api' && !fieldChecked('frontend')) return 'jwt';
      return 'session';
    }
    function recommendDeploymentTarget(){
      var type = fieldValue('tipo', 'web_api');
      if(fieldChecked('kubernetes')) return 'kubernetes';
      if(type === 'embedded' || fieldChecked('plat_embedded')) return 'edge';
      if(type === 'desktop' || type === 'mobile' || type === 'cli' || fieldChecked('plat_desktop') || fieldChecked('plat_mobile') || fieldChecked('plat_cli')) return 'vm';
      if(type === 'web' && !fieldChecked('api')) return 'serverless';
      return 'docker';
    }
    function recommendArtifactType(){
      var type = fieldValue('tipo', 'web_api');
      var deploymentTarget = fieldValue('deployment_target', 'docker');
      if(type === 'embedded' || fieldChecked('plat_embedded')) return 'firmware';
      if(type === 'mobile' || fieldChecked('plat_mobile')) return 'mobile_bundle';
      if(type === 'desktop' || fieldChecked('plat_desktop')) return 'desktop_installer';
      if(type === 'web' && !fieldChecked('api')) return 'static_site';
      if(deploymentTarget === 'kubernetes' || deploymentTarget === 'docker' || fieldChecked('kubernetes')) return 'docker_image';
      return 'executable';
    }
    var dbSelect = form ? form.querySelector('select[name="db_engine"]') : null;
    var dbToggle = form ? form.querySelector('input[name="db"]') : null;
    var dbRecommendationLabel = document.getElementById('factory-db-recommendation-label');
    var dbRecommendationReason = document.getElementById('factory-db-recommendation-reason');
    var frontendSelect = form ? form.querySelector('select[name="frontend_stack"]') : null;
    var frontendRecommendationLabel = document.getElementById('factory-frontend-recommendation-label');
    var frontendRecommendationReason = document.getElementById('factory-frontend-recommendation-reason');
    var authSelect = form ? form.querySelector('select[name="auth_mode"]') : null;
    var authToggle = form ? form.querySelector('input[name="auth"]') : null;
    var authRecommendationLabel = document.getElementById('factory-auth-recommendation-label');
    var authRecommendationReason = document.getElementById('factory-auth-recommendation-reason');
    var deploymentSelect = form ? form.querySelector('select[name="deployment_target"]') : null;
    var deploymentRecommendationLabel = document.getElementById('factory-deployment-recommendation-label');
    var deploymentRecommendationReason = document.getElementById('factory-deployment-recommendation-reason');
    var artifactSelect = form ? form.querySelector('select[name="artifact_type"]') : null;
    var artifactRecommendationLabel = document.getElementById('factory-artifact-recommendation-label');
    var artifactRecommendationReason = document.getElementById('factory-artifact-recommendation-reason');
    var dbEngineTouched = false;
    var dbToggleTouched = false;
    var frontendTouched = false;
    var authModeTouched = false;
    var authToggleTouched = false;
    var deploymentTouched = false;
    var artifactTouched = false;
    if(dbSelect){
      dbSelect.addEventListener('change', function(){
        dbEngineTouched = true;
        this.dataset.auto = '0';
      });
    }
    if(dbToggle){
      dbToggle.addEventListener('change', function(){
        dbToggleTouched = true;
        this.dataset.auto = '0';
      });
    }
    if(authSelect){
      authSelect.addEventListener('change', function(){
        authModeTouched = true;
        this.dataset.auto = '0';
      });
    }
    if(frontendSelect){
      frontendSelect.addEventListener('change', function(){
        frontendTouched = true;
        this.dataset.auto = '0';
      });
    }
    if(authToggle){
      authToggle.addEventListener('change', function(){
        authToggleTouched = true;
        this.dataset.auto = '0';
      });
    }
    if(deploymentSelect){
      deploymentSelect.addEventListener('change', function(){
        deploymentTouched = true;
        this.dataset.auto = '0';
      });
    }
    if(artifactSelect){
      artifactSelect.addEventListener('change', function(){
        artifactTouched = true;
        this.dataset.auto = '0';
      });
    }
    function syncDBRecommendation(){
      var engine = recommendDBEngine();
      if(dbRecommendationLabel) dbRecommendationLabel.textContent = dbEngineLabels[engine] || engine;
      if(dbRecommendationReason) dbRecommendationReason.textContent = dbEngineReasons[engine] || '';
      if(dbSelect && (!dbEngineTouched || dbSelect.dataset.auto === '1' || dbSelect.value === '' || dbSelect.value === 'none')){
        dbSelect.value = engine;
        dbSelect.dataset.auto = '1';
      }
      if(dbToggle && (!dbToggleTouched || dbToggle.dataset.auto === '1')){
        dbToggle.checked = engine !== 'none';
        dbToggle.dataset.auto = '1';
      }
    }
    function syncFrontendRecommendation(){
      var value = recommendFrontendStack();
      if(frontendRecommendationLabel) frontendRecommendationLabel.textContent = frontendStackLabels[value] || value;
      if(frontendRecommendationReason) frontendRecommendationReason.textContent = frontendStackReasons[value] || '';
      if(frontendSelect && (!frontendTouched || frontendSelect.dataset.auto === '1' || frontendSelect.value === '')){
        frontendSelect.value = value;
        frontendSelect.dataset.auto = '1';
      }
    }
    function syncAuthRecommendation(){
      var value = recommendAuthMode();
      if(authRecommendationLabel) authRecommendationLabel.textContent = authModeLabels[value] || value;
      if(authRecommendationReason) authRecommendationReason.textContent = authModeReasons[value] || '';
      if(authSelect && (!authModeTouched || authSelect.dataset.auto === '1' || authSelect.value === '' || authSelect.value === 'none')){
        authSelect.value = value;
        authSelect.dataset.auto = '1';
      }
      if(authToggle && (!authToggleTouched || authToggle.dataset.auto === '1')){
        authToggle.checked = value !== 'none';
        authToggle.dataset.auto = '1';
      }
    }
    function syncDeploymentRecommendation(){
      var value = recommendDeploymentTarget();
      if(deploymentRecommendationLabel) deploymentRecommendationLabel.textContent = deploymentLabels[value] || value;
      if(deploymentRecommendationReason) deploymentRecommendationReason.textContent = deploymentReasons[value] || '';
      if(deploymentSelect && (!deploymentTouched || deploymentSelect.dataset.auto === '1' || deploymentSelect.value === '')){
        deploymentSelect.value = value;
        deploymentSelect.dataset.auto = '1';
      }
    }
    function syncArtifactRecommendation(){
      var value = recommendArtifactType();
      if(artifactRecommendationLabel) artifactRecommendationLabel.textContent = artifactLabels[value] || value;
      if(artifactRecommendationReason) artifactRecommendationReason.textContent = artifactReasons[value] || '';
      if(artifactSelect && (!artifactTouched || artifactSelect.dataset.auto === '1' || artifactSelect.value === '')){
        artifactSelect.value = value;
        artifactSelect.dataset.auto = '1';
      }
    }

    // ── Presets ───────────────────────────────────────────────────────────
    var PRESETS = {
      'web-publica': {
        tipo:'web_api',
        checks:{frontend:1,api:1,auth:1,db:1,docker:1,i18n:1,ci:1,
                'plat_web':1,'compliance_rgpd':1,'compliance_lssi':1,'compliance_wcag':1}
      },
      'api-interna': {
        tipo:'api',
        checks:{api:1,auth:1,db:1,docker:1,ci:1,'plat_web':1}
      },
      'cli-devops': {
        tipo:'cli',
        checks:{docker:1,ci:1,terraform:1,'plat_cli':1,'so_linux':1}
      },
      'app-movil': {
        tipo:'mobile',
        checks:{auth:1,db:1,'plat_mobile':1,'so_android':1,'so_ios':1,'compliance_rgpd':1}
      },
      'embedded': {
        tipo:'embedded',
        checks:{docker:1,'plat_embedded':1,'so_linux':1}
      }
    };

    window.factoryApplyPreset = function(key){
      if(!key) return;
      var preset = PRESETS[key];
      if(!preset) return;
      // reset all checkboxes in the form
      form.querySelectorAll('input[type=checkbox]').forEach(function(el){
        el.checked = false;
      });
      // set tipo
      var tipoEl = document.getElementById('factory-tipo');
      if(tipoEl) tipoEl.value = preset.tipo;
      // set checkboxes
      Object.keys(preset.checks).forEach(function(name){
        var el = form.querySelector('input[name="'+name+'"]');
        if(el) el.checked = true;
      });
      dbEngineTouched = false;
      dbToggleTouched = false;
      frontendTouched = false;
      authModeTouched = false;
      authToggleTouched = false;
      deploymentTouched = false;
      artifactTouched = false;
      // reset preset selector back to blank so user can re-apply
      document.getElementById('factory-preset').value = '';
      syncDBRecommendation();
      syncFrontendRecommendation();
      syncAuthRecommendation();
      syncDeploymentRecommendation();
      syncArtifactRecommendation();
    };
    if(form){
      form.querySelectorAll('input,select,textarea').forEach(function(el){
        if(el.name === 'db_engine' || el.name === 'db' || el.name === 'frontend_stack' || el.name === 'auth_mode' || el.name === 'auth' || el.name === 'deployment_target' || el.name === 'artifact_type') return;
        el.addEventListener('change', function(){
          syncDBRecommendation();
          syncFrontendRecommendation();
          syncAuthRecommendation();
          syncDeploymentRecommendation();
          syncArtifactRecommendation();
        });
      });
    }
    syncDBRecommendation();
    syncFrontendRecommendation();
    syncAuthRecommendation();
    syncDeploymentRecommendation();
    syncArtifactRecommendation();

    // ── Preview ───────────────────────────────────────────────────────────
    window.factoryPreview = function(btn){
      var data = new FormData(form);
      var obj = {};
      data.forEach(function(v,k){ obj[k] = v; });
      // convert checkbox booleans
      var bools = ['frontend','api','auth','db','docker','i18n','theme_support','branding_profiles',
                   'plat_web','plat_desktop','plat_mobile','plat_cli','plat_embedded',
                   'so_linux','so_windows','so_macos','so_android','so_ios',
                   'compliance_rgpd','compliance_ens','compliance_lssi','compliance_wcag',
                   'compliance_factura_elec','compliance_reutilizacion',
                   'ci','kubernetes','terraform','monitoring'];
      var payload = {
        nombre: obj['nombre']||'',
        descripcion: obj['descripcion']||'',
        tipo: obj['tipo']||'',
        frontend_stack: obj['frontend_stack']||'react',
        auth_mode: obj['auth_mode']||'none',
        identity_provider: obj['identity_provider']||'none',
        db_engine: obj['db_engine']||'none',
        deployment_target: obj['deployment_target']||'docker',
        artifact_type: obj['artifact_type']||'executable',
        idiomas: (obj['idiomas']||'es,en').split(','),
        por: obj['por']||'web'
      };
      bools.forEach(function(f){ payload[f] = obj[f]==='1'; });

      var slug = form.action.split('/proyectos/')[1].split('/')[0];
      btn.setAttribute('aria-busy','true');
      btn.disabled = true;
      fetch('/api/proyectos/'+encodeURIComponent(slug)+'/fabricar-app/preview',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body: JSON.stringify(payload)
      })
      .then(function(r){ return r.json(); })
      .then(function(data){
        btn.removeAttribute('aria-busy');
        btn.disabled = false;
        if(!data.ok || !data.tasks) return;
        var tbody = document.getElementById('factory-preview-tbody');
        tbody.innerHTML = '';
        data.tasks.forEach(function(t){
          var tr = document.createElement('tr');
          tr.innerHTML = '<td><code>'+t.Key+'</code></td><td>'+t.Titulo+'</td><td>'+t.Fase+'</td><td>'+t.Modulo+'</td>';
          tbody.appendChild(tr);
        });
        document.getElementById('factory-preview-count').textContent = data.tasks.length;
        document.getElementById('factory-preview-result').style.display = '';
      })
      .catch(function(e){
        btn.removeAttribute('aria-busy');
        btn.disabled = false;
        alert('Error al obtener preview: '+e);
      });
    };
  })();
  </script>

  <article>
    <h3>{{tr "projects.decisions.title"}}</h3>
    {{if .Decisiones}}
      {{range .Decisiones}}
      <details open style="margin-bottom:.75rem">
        <summary><strong>{{.Titulo}}</strong> · {{.Estado}}</summary>
        <p><strong>{{tr "projects.category"}}:</strong> {{.Categoria}}</p>
        <p><strong>{{tr "projects.solution"}}:</strong> {{.Solucion}}</p>
        {{if .Motivo}}<p><strong>{{tr "Motivo"}}:</strong> {{.Motivo}}</p>{{end}}
        {{if .Alternativas}}<p><strong>{{tr "projects.alternatives"}}:</strong> {{.Alternativas}}</p>{{end}}
        {{if .Impacto}}<p><strong>{{tr "projects.impact"}}:</strong> {{.Impacto}}</p>{{end}}
      </details>
      {{end}}
    {{else}}
      <p>{{tr "projects.decisions.none"}}</p>
    {{end}}
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/decisiones?lang={{lang}}">
      <label>{{tr "projects.title_label"}} <input name="titulo" required></label>
      <label>{{tr "projects.category"}} <input name="categoria" value="general"></label>
      <label>{{tr "projects.solution"}} <textarea name="solucion" required></textarea></label>
      <label>{{tr "Motivo"}} <textarea name="motivo"></textarea></label>
      <label>{{tr "projects.alternatives"}} <textarea name="alternativas"></textarea></label>
      <label>{{tr "projects.impact"}} <textarea name="impacto"></textarea></label>
      <label>{{tr "Estado"}}
        <select name="estado">
          <option value="vigente">{{tr "vigente"}}</option>
          <option value="experimental">{{tr "experimental"}}</option>
          <option value="reemplazada">{{tr "reemplazada"}}</option>
          <option value="descartada">{{tr "descartada"}}</option>
          <option value="archivada">{{tr "archivada"}}</option>
        </select>
      </label>
      <label>{{tr "projects.related_proposal"}} <input name="propuesta_id" inputmode="numeric"></label>
      <label>{{tr "projects.related_task"}} <input name="tarea_id" inputmode="numeric"></label>
      <button type="submit">{{tr "projects.save_decision"}}</button>
    </form>
  </article>

  <article>
    <h3>{{tr "projects.docs.title"}}</h3>
    {{if .Documentos}}
      {{range .Documentos}}
      <details open style="margin-bottom:.75rem">
        <summary><strong>{{.Titulo}}</strong> · {{.TipoDocumento}}</summary>
        <p><strong>{{tr "projects.path"}}:</strong> <code>{{.RutaRef}}</code></p>
        <p><strong>{{tr "projects.summary"}}:</strong> {{.Resumen}}</p>
        <p><strong>{{tr "Estado"}}:</strong> {{.Estado}} · <strong>{{tr "projects.source"}}:</strong> {{.Fuente}}</p>
      </details>
      {{end}}
    {{else}}
      <p>{{tr "projects.docs.none"}}</p>
    {{end}}
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/documentacion?lang={{lang}}">
      <label>{{tr "projects.title_label"}} <input name="titulo" required></label>
      <label>{{tr "projects.doc_type"}} <input name="tipo_documento" value="markdown"></label>
      <label>{{tr "projects.path_or_ref"}} <input name="ruta_ref" required></label>
      <label>{{tr "projects.summary"}} <textarea name="resumen" required></textarea></label>
      <label>{{tr "Estado"}}
        <select name="estado">
          <option value="vigente">{{tr "vigente"}}</option>
          <option value="borrador">{{tr "borrador"}}</option>
          <option value="archivado">{{tr "archivado"}}</option>
        </select>
      </label>
      <label>{{tr "projects.source"}}
        <select name="fuente">
          <option value="manual">{{tr "manual"}}</option>
          <option value="propuesta">{{tr "propuesta"}}</option>
          <option value="tarea">{{tr "tarea"}}</option>
          <option value="externo">{{tr "externo"}}</option>
        </select>
      </label>
      <label>{{tr "projects.related_proposal"}} <input name="propuesta_id" inputmode="numeric"></label>
      <label>{{tr "projects.related_task"}} <input name="tarea_id" inputmode="numeric"></label>
      <button type="submit">{{tr "projects.save_document"}}</button>
    </form>
  </article>
</section>

<section class="container">
  <article>
    <h3>{{tr "projects.review_gates.title"}}</h3>
    {{if .ReviewGates}}
      {{range .ReviewGates}}
      <details open style="margin-bottom:.75rem">
        <summary><strong>#{{.ID}}</strong> · {{.Estado}} · {{if .ReviewerAgente}}{{.ReviewerAgente}}{{else}}{{tr "projects.review_gates.unassigned"}}{{end}}</summary>
        {{if .SeverityMax}}<p><strong>{{tr "projects.review_gates.severity"}}:</strong> {{.SeverityMax}}</p>{{end}}
        {{if .FindingsJSON}}<p><strong>{{tr "projects.review_gates.findings"}}:</strong> <code>{{.FindingsJSON}}</code></p>{{end}}
        <form method="post" action="/proyectos/{{$.Proyecto.Slug}}/review-gates/{{.ID}}/resolver?lang={{lang}}">
          <label>{{tr "projects.review_gates.state"}}
            <select name="estado">
              <option value="pendiente" {{if eq .Estado "pendiente"}}selected{{end}}>{{tr "projects.review_gates.pending"}}</option>
              <option value="en_revision" {{if eq .Estado "en_revision"}}selected{{end}}>{{tr "projects.review_gates.in_review"}}</option>
              <option value="cambios_pedidos" {{if eq .Estado "cambios_pedidos"}}selected{{end}}>{{tr "projects.review_gates.changes_requested"}}</option>
              <option value="aprobado" {{if eq .Estado "aprobado"}}selected{{end}}>{{tr "projects.review_gates.approved"}}</option>
              <option value="bloqueado" {{if eq .Estado "bloqueado"}}selected{{end}}>{{tr "projects.review_gates.blocked"}}</option>
            </select>
          </label>
          <label>{{tr "Agente"}} <input name="reviewer_agente" value="{{.ReviewerAgente}}"></label>
          <label>{{tr "projects.review_gates.findings"}} <textarea name="findings_json">{{.FindingsJSON}}</textarea></label>
          <button type="submit">{{tr "projects.review_gates.save"}}</button>
        </form>
      </details>
      {{end}}
    {{else}}
      <p>{{tr "projects.review_gates.none"}}</p>
    {{end}}
  </article>
</section>

<section class="container">
  <article>
    <h3>{{tr "projects.vote_history.title"}}</h3>
    {{if .Votaciones}}
      {{range .Votaciones}}
      <details style="margin-bottom:.75rem">
        <summary><strong>{{.Codigo}}</strong> · {{.Titulo}} · {{.Estado}}</summary>
        <p>{{.Tipo}} · {{tr "projects.proposed_by"}} {{.PropuestoPor}}</p>
        <p>{{tr "projects.votes_summary"}}: ✓ {{.Acuerdo}} · ✗ {{.Desacuerdo}} · ～ {{.Abstencion}} · ⏳ {{.Pendiente}}</p>
        {{if .Votos}}
        <table>
          <thead><tr><th>{{tr "Agente"}}</th><th>{{tr "projects.position"}}</th><th>{{tr "projects.comment"}}</th></tr></thead>
          <tbody>
            {{range .Votos}}
            <tr><td>{{.Agente}}</td><td>{{.Posicion}}</td><td>{{.Comentario}}</td></tr>
            {{end}}
          </tbody>
        </table>
        {{end}}
      </details>
      {{end}}
    {{else}}
      <p>{{tr "projects.vote_history.none"}}</p>
    {{end}}
  </article>
</section>
{{end}}`

// ── /nueva-app: página de wizard independiente ───────────────────────────────

type webNuevaAppData struct {
	Proyectos              []webProyectoResumen
	Idiomas                []webIdiomaOption
	Preview                []previewTask
	DBRecommendation       webDBRecommendation
	FrontendRecommendation webFieldRecommendation
	AuthRecommendation     webFieldRecommendation
	DeployRecommendation   webFieldRecommendation
	ArtifactRecommendation webFieldRecommendation
	Msg                    string
	Err                    string
}

type webIdiomaOption struct {
	Code    string
	Label   string
	Default bool
}

type previewTask struct {
	Key    string
	Titulo string
	Fase   string
	Modulo string
}

type webDBRecommendation struct {
	Engine string
	Label  string
	Reason string
}

type webFieldRecommendation struct {
	Value  string
	Label  string
	Reason string
}

func webNuevaAppIdiomaOptions() []webIdiomaOption {
	return []webIdiomaOption{
		{Code: "es", Label: "Español", Default: true},
		{Code: "en", Label: "English", Default: true},
		{Code: "fr", Label: "Français"},
		{Code: "de", Label: "Deutsch"},
		{Code: "it", Label: "Italiano"},
		{Code: "pt", Label: "Português"},
		{Code: "ca", Label: "Català"},
		{Code: "eu", Label: "Euskara"},
		{Code: "gl", Label: "Galego"},
		{Code: "val", Label: "Valencià"},
	}
}

func collectNuevaAppIdiomas(r *http.Request) []string {
	idiomas := append([]string(nil), r.Form["idioma"]...)
	if extra := strings.TrimSpace(r.FormValue("idiomas_extra")); extra != "" {
		idiomas = append(idiomas, splitCSV(extra)...)
	}
	return splitCSV(strings.Join(idiomas, ","))
}

func webNuevaAppRedirectURL(r *http.Request, flashKey, flashMsg string) string {
	params := url.Values{}
	if flashKey != "" && flashMsg != "" {
		params.Set(flashKey, flashMsg)
	}
	if lang := strings.TrimSpace(resolveWebRequestLang(r)); lang != "" {
		params.Set("lang", lang)
	}
	if encoded := params.Encode(); encoded != "" {
		return "/nueva-app?" + encoded
	}
	return "/nueva-app"
}

func webTareasProyectoRedirectURL(r *http.Request, slug, flashKey, flashMsg string) string {
	params := url.Values{}
	params.Set("proyecto", slug)
	if flashKey != "" && flashMsg != "" {
		params.Set(flashKey, flashMsg)
	}
	if lang := strings.TrimSpace(resolveWebRequestLang(r)); lang != "" {
		params.Set("lang", lang)
	}
	return "/tareas?" + params.Encode()
}

func webHandlerNuevaApp(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		webHandlerNuevaAppGET(w, r)
	case http.MethodPost:
		webHandlerNuevaAppPOST(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func webHandlerNuevaAppGET(w http.ResponseWriter, r *http.Request) {
	proyectos, _ := webCargarProyectosPorAPI()
	var resumen []webProyectoResumen
	for _, p := range proyectos {
		resumen = append(resumen, webProyectoResumen{
			Slug:   p.Slug,
			Nombre: p.Nombre,
			Tipo:   string(p.Tipo),
			Activo: p.Activo,
		})
	}
	webRender(w, r, webTplLayout+webTplNuevaApp, webNuevaAppData{
		Proyectos:              resumen,
		Idiomas:                webNuevaAppIdiomaOptions(),
		DBRecommendation:       webNuevaAppDefaultDBRecommendation(r),
		FrontendRecommendation: webNuevaAppDefaultFrontendRecommendation(r),
		AuthRecommendation:     webNuevaAppDefaultAuthRecommendation(r),
		DeployRecommendation:   webNuevaAppDefaultDeploymentRecommendation(r),
		ArtifactRecommendation: webNuevaAppDefaultArtifactRecommendation(r),
		Msg:                    r.URL.Query().Get("ok"),
		Err:                    r.URL.Query().Get("err"),
	})
}

func webHandlerNuevaAppPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, webNuevaAppRedirectURL(r, "err", err.Error()), http.StatusSeeOther)
		return
	}
	slug := strings.TrimSpace(r.FormValue("proyecto_slug"))
	if slug == "" {
		http.Redirect(w, r, webNuevaAppRedirectURL(r, "err", webTranslateRequestf(r, "projects.factory.select_project_error")), http.StatusSeeOther)
		return
	}

	actor := strings.TrimSpace(r.FormValue("por"))
	if actor == "" {
		actor = "web"
	}

	req := apiProyectoFabricarAppRequest{
		Nombre:           strings.TrimSpace(r.FormValue("nombre")),
		Descripcion:      strings.TrimSpace(r.FormValue("descripcion")),
		ObjetivoNegocio:  strings.TrimSpace(r.FormValue("objetivo_negocio")),
		UsuariosObjetivo: strings.TrimSpace(r.FormValue("usuarios_objetivo")),
		Restricciones:    strings.TrimSpace(r.FormValue("restricciones")),
		Tipo:             strings.TrimSpace(r.FormValue("tipo")),
		Frontend:         webFormBool(r, "frontend"),
		API:              webFormBool(r, "api"),
		Auth:             webFormBool(r, "auth"),
		Database:         webFormBool(r, "db"),
		Docker:           webFormBool(r, "docker"),
		I18n:             webFormBool(r, "i18n"),
		Idiomas:          collectNuevaAppIdiomas(r),
		Por:              actor,

		PlatWeb:      webFormBool(r, "plat_web"),
		PlatDesktop:  webFormBool(r, "plat_desktop"),
		PlatMobile:   webFormBool(r, "plat_mobile"),
		PlatCLI:      webFormBool(r, "plat_cli"),
		PlatEmbedded: webFormBool(r, "plat_embedded"),

		SOLinux:   webFormBool(r, "so_linux"),
		SOWindows: webFormBool(r, "so_windows"),
		SOmacOS:   webFormBool(r, "so_macos"),
		SOAndroid: webFormBool(r, "so_android"),
		SOiOS:     webFormBool(r, "so_ios"),

		ComplianceRGPD:          webFormBool(r, "compliance_rgpd"),
		ComplianceENS:           webFormBool(r, "compliance_ens"),
		ComplianceLSSI:          webFormBool(r, "compliance_lssi"),
		ComplianceWCAG:          webFormBool(r, "compliance_wcag"),
		ComplianceFacturaElec:   webFormBool(r, "compliance_factura_elec"),
		ComplianceReutilizacion: webFormBool(r, "compliance_reutilizacion"),

		CI:         webFormBool(r, "ci"),
		Kubernetes: webFormBool(r, "kubernetes"),
		Terraform:  webFormBool(r, "terraform"),
		Monitoring: webFormBool(r, "monitoring"),

		Arquitectura:       strings.TrimSpace(r.FormValue("arquitectura")),
		APIStyle:           strings.TrimSpace(r.FormValue("api_style")),
		FrontendStack:      strings.TrimSpace(r.FormValue("frontend_stack")),
		DatabaseEngine:     strings.TrimSpace(r.FormValue("db_engine")),
		AuthMode:           strings.TrimSpace(r.FormValue("auth_mode")),
		IdentityProvider:   strings.TrimSpace(r.FormValue("identity_provider")),
		TestingLevel:       strings.TrimSpace(r.FormValue("testing_level")),
		ObservabilityLevel: strings.TrimSpace(r.FormValue("observability_level")),
		DeploymentTarget:   strings.TrimSpace(r.FormValue("deployment_target")),
		ArtifactType:       strings.TrimSpace(r.FormValue("artifact_type")),
		BackgroundJobs:     webFormBool(r, "background_jobs"),
		Notifications:      webFormBool(r, "notifications"),
		MultiTenant:        webFormBool(r, "multi_tenant"),
		RBAC:               webFormBool(r, "rbac"),
		ThemeSupport:       webFormBool(r, "theme_support"),
		BrandingProfiles:   webFormBool(r, "branding_profiles"),
		OfflineMode:        webFormBool(r, "offline_mode"),
		ImportExport:       webFormBool(r, "import_export"),
		Webhooks:           webFormBool(r, "webhooks"),
		FileUploads:        webFormBool(r, "file_uploads"),
		Reporting:          webFormBool(r, "reporting"),
		ServicioResidente:  webFormBool(r, "servicio_residente"),
		Cache:              webFormBool(r, "cache"),
		Queue:              webFormBool(r, "queue"),
		Scheduler:          webFormBool(r, "scheduler"),
		ObjectStorage:      webFormBool(r, "object_storage"),
		Search:             webFormBool(r, "search"),
		RateLimiting:       webFormBool(r, "rate_limiting"),
		FeatureFlags:       webFormBool(r, "feature_flags"),
		AuditTrail:         webFormBool(r, "audit_trail"),
		Backups:            webFormBool(r, "backups"),
		DisasterRecovery:   webFormBool(r, "disaster_recovery"),
		Integraciones:      splitCSV(strings.TrimSpace(r.FormValue("integraciones"))),
	}

	// Preview: muestra tareas sin persistir
	if r.URL.Query().Get("preview") == "1" {
		var previewResp apiProyectoFabricarAppPreviewResponse
		path := "/api/proyectos/" + url.PathEscape(slug) + "/fabricar-app/preview"
		if err := webInvocarAPIJSON(http.MethodPost, path, req, &previewResp); err != nil {
			http.Redirect(w, r, webNuevaAppRedirectURL(r, "err", err.Error()), http.StatusSeeOther)
			return
		}
		proyectos, _ := webCargarProyectosPorAPI()
		var resumen []webProyectoResumen
		for _, p := range proyectos {
			resumen = append(resumen, webProyectoResumen{Slug: p.Slug, Nombre: p.Nombre, Tipo: string(p.Tipo), Activo: p.Activo})
		}
		tasks := make([]previewTask, 0, len(previewResp.Tasks))
		for _, t := range previewResp.Tasks {
			tasks = append(tasks, previewTask{Key: t.Key, Titulo: t.Titulo, Fase: t.Fase, Modulo: t.Modulo})
		}
		webRender(w, r, webTplLayout+webTplNuevaApp, webNuevaAppData{
			Proyectos:              resumen,
			Idiomas:                webNuevaAppIdiomaOptions(),
			Preview:                tasks,
			DBRecommendation:       webNuevaAppDBRecommendation(r, req),
			FrontendRecommendation: webNuevaAppFrontendRecommendation(r, req),
			AuthRecommendation:     webNuevaAppAuthRecommendation(r, req),
			DeployRecommendation:   webNuevaAppDeploymentRecommendation(r, req),
			ArtifactRecommendation: webNuevaAppArtifactRecommendation(r, req),
		})
		return
	}

	if _, err := webFabricarAppProyectoPorAPI(slug, req); err != nil {
		http.Redirect(w, r, webNuevaAppRedirectURL(r, "err", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, webTareasProyectoRedirectURL(r, slug, "ok", webTranslateRequestf(r, "projects.flash.factory_created")), http.StatusSeeOther)
}

func webNuevaAppDefaultDBRecommendation(r *http.Request) webDBRecommendation {
	req := webNuevaAppDefaultRecommendationRequest()
	return webNuevaAppDBRecommendation(r, req)
}

func webNuevaAppDefaultFrontendRecommendation(r *http.Request) webFieldRecommendation {
	req := webNuevaAppDefaultRecommendationRequest()
	return webNuevaAppFrontendRecommendation(r, req)
}

func webNuevaAppDefaultAuthRecommendation(r *http.Request) webFieldRecommendation {
	req := webNuevaAppDefaultRecommendationRequest()
	return webNuevaAppAuthRecommendation(r, req)
}

func webNuevaAppDefaultDeploymentRecommendation(r *http.Request) webFieldRecommendation {
	req := webNuevaAppDefaultRecommendationRequest()
	return webNuevaAppDeploymentRecommendation(r, req)
}

func webNuevaAppDefaultArtifactRecommendation(r *http.Request) webFieldRecommendation {
	req := webNuevaAppDefaultRecommendationRequest()
	return webNuevaAppArtifactRecommendation(r, req)
}

func webNuevaAppDefaultRecommendationRequest() apiProyectoFabricarAppRequest {
	return apiProyectoFabricarAppRequest{
		Tipo:             "web_api",
		Frontend:         true,
		API:              true,
		I18n:             true,
		Arquitectura:     "hexagonal",
		APIStyle:         "rest",
		FrontendStack:    "react",
		ThemeSupport:     true,
		AuthMode:         "none",
		IdentityProvider: "none",
		TestingLevel:     "base",
		DeploymentTarget: "docker",
		ArtifactType:     "executable",
	}
}

func webAppSpecFromFactoryRequest(req apiProyectoFabricarAppRequest) fabricaapp.AppSpec {
	return fabricaapp.AppSpec{
		Tipo:              req.Tipo,
		Frontend:          req.Frontend,
		API:               req.API,
		Auth:              req.Auth,
		Database:          req.Database,
		PlatWeb:           req.PlatWeb,
		PlatDesktop:       req.PlatDesktop,
		PlatMobile:        req.PlatMobile,
		PlatCLI:           req.PlatCLI,
		PlatEmbedded:      req.PlatEmbedded,
		Kubernetes:        req.Kubernetes,
		Arquitectura:      req.Arquitectura,
		APIStyle:          req.APIStyle,
		FrontendStack:     req.FrontendStack,
		AuthMode:          req.AuthMode,
		IdentityProvider:  req.IdentityProvider,
		DeploymentTarget:  req.DeploymentTarget,
		ArtifactType:      req.ArtifactType,
		BackgroundJobs:    req.BackgroundJobs,
		Notifications:     req.Notifications,
		MultiTenant:       req.MultiTenant,
		RBAC:              req.RBAC,
		ThemeSupport:      req.ThemeSupport,
		BrandingProfiles:  req.BrandingProfiles,
		OfflineMode:       req.OfflineMode,
		ImportExport:      req.ImportExport,
		Webhooks:          req.Webhooks,
		FileUploads:       req.FileUploads,
		Reporting:         req.Reporting,
		ServicioResidente: req.ServicioResidente,
		Queue:             req.Queue,
		Scheduler:         req.Scheduler,
		ObjectStorage:     req.ObjectStorage,
		Search:            req.Search,
		AuditTrail:        req.AuditTrail,
		Backups:           req.Backups,
		DisasterRecovery:  req.DisasterRecovery,
	}
}

func webNuevaAppDBRecommendation(r *http.Request, req apiProyectoFabricarAppRequest) webDBRecommendation {
	engine := fabricaapp.RecommendDatabaseEngine(webAppSpecFromFactoryRequest(req))
	return webDBRecommendation{
		Engine: engine,
		Label:  webTranslateRequestf(r, "projects.factory.db_engine."+engine),
		Reason: webTranslateRequestf(r, "projects.factory.db_recommendation.reason."+engine),
	}
}

func webNuevaAppFrontendRecommendation(r *http.Request, req apiProyectoFabricarAppRequest) webFieldRecommendation {
	value := fabricaapp.RecommendFrontendStack(webAppSpecFromFactoryRequest(req))
	return webFieldRecommendation{
		Value:  value,
		Label:  webTranslateRequestf(r, "projects.factory.frontend_stack."+value),
		Reason: webTranslateRequestf(r, "projects.factory.frontend_recommendation.reason."+value),
	}
}

func webNuevaAppAuthRecommendation(r *http.Request, req apiProyectoFabricarAppRequest) webFieldRecommendation {
	value := fabricaapp.RecommendAuthMode(webAppSpecFromFactoryRequest(req))
	return webFieldRecommendation{
		Value:  value,
		Label:  webTranslateRequestf(r, "projects.factory.auth_mode."+value),
		Reason: webTranslateRequestf(r, "projects.factory.auth_recommendation.reason."+value),
	}
}

func webNuevaAppDeploymentRecommendation(r *http.Request, req apiProyectoFabricarAppRequest) webFieldRecommendation {
	value := fabricaapp.RecommendDeploymentTarget(webAppSpecFromFactoryRequest(req))
	return webFieldRecommendation{
		Value:  value,
		Label:  webTranslateRequestf(r, "projects.factory.deployment_target."+value),
		Reason: webTranslateRequestf(r, "projects.factory.deployment_recommendation.reason."+value),
	}
}

func webNuevaAppArtifactRecommendation(r *http.Request, req apiProyectoFabricarAppRequest) webFieldRecommendation {
	value := fabricaapp.RecommendArtifactType(webAppSpecFromFactoryRequest(req))
	return webFieldRecommendation{
		Value:  value,
		Label:  webTranslateRequestf(r, "projects.factory.artifact_type."+value),
		Reason: webTranslateRequestf(r, "projects.factory.artifact_recommendation.reason."+value),
	}
}

const webTplNuevaApp = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "nueva_app.title"}}</h2>
  <p style="color:#64748b">{{tr "projects.factory.fixed_note"}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}

  <article>
    <form id="nueva-app-form" method="post" action="/nueva-app?lang={{lang}}">
      <fieldset>
        <legend>1. {{tr "projects.factory.section.brainstorming"}}</legend>
        <label>{{tr "nueva_app.project_select"}}
          <select name="proyecto_slug" form="nueva-app-form" required>
            {{range .Proyectos}}
            <option value="{{.Slug}}">{{.Nombre}} ({{.Slug}})</option>
            {{end}}
          </select>
        </label>

        <label>{{tr "projects.factory.preset"}}
          <select id="na-preset" onchange="factoryApplyPreset(this.value)">
            <option value="">{{tr "projects.factory.preset.none"}}</option>
            <option value="web-publica">{{tr "projects.factory.preset.web_publica"}}</option>
            <option value="api-interna">{{tr "projects.factory.preset.api_interna"}}</option>
            <option value="cli-devops">{{tr "projects.factory.preset.cli_devops"}}</option>
            <option value="app-movil">{{tr "projects.factory.preset.app_movil"}}</option>
            <option value="embedded">{{tr "projects.factory.preset.embedded"}}</option>
          </select>
        </label>

      <input type="hidden" name="proyecto_slug" id="na-slug-hidden">
      <label>{{tr "projects.factory.name"}} <input name="nombre" required></label>
      <label>{{tr "projects.factory.description"}} <textarea name="descripcion"></textarea></label>
      <label>{{tr "projects.factory.business_goal"}} <input name="objetivo_negocio"></label>
      <label>{{tr "projects.factory.target_users"}} <input name="usuarios_objetivo"></label>
      <label>{{tr "projects.factory.constraints"}} <textarea name="restricciones"></textarea></label>

      <label>{{tr "projects.factory.type"}}
        <select name="tipo" id="factory-tipo" required>
          <option value="web_api">web_api — {{tr "projects.factory.type.web_api"}}</option>
          <option value="web">web — {{tr "projects.factory.type.web"}}</option>
          <option value="api">api — {{tr "projects.factory.type.api"}}</option>
          <option value="cli">cli — {{tr "projects.factory.type.cli"}}</option>
          <option value="desktop">desktop — {{tr "projects.factory.type.desktop"}}</option>
          <option value="mobile">mobile — {{tr "projects.factory.type.mobile"}}</option>
          <option value="embedded">embedded — {{tr "projects.factory.type.embedded"}}</option>
        </select>
      </label>
      </fieldset>

      <fieldset>
        <legend>2. {{tr "projects.factory.section.architecture"}}</legend>
        <label>{{tr "projects.factory.architecture"}}
          <select name="arquitectura">
            <option value="hexagonal">{{tr "projects.factory.architecture.hexagonal"}}</option>
            <option value="clean">{{tr "projects.factory.architecture.clean"}}</option>
            <option value="layered">{{tr "projects.factory.architecture.layered"}}</option>
            <option value="event_driven">{{tr "projects.factory.architecture.event_driven"}}</option>
          </select>
        </label>
        {{with wizardHelp "arquitectura"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="servicio_residente" value="1"> {{tr "projects.factory.servicio_residente"}}</label>
          <label><input type="checkbox" name="background_jobs" value="1"> {{tr "projects.factory.background_jobs"}}</label>
          <label><input type="checkbox" name="queue" value="1"> {{tr "projects.factory.queue"}}</label>
          <label><input type="checkbox" name="scheduler" value="1"> {{tr "projects.factory.scheduler"}}</label>
          <label><input type="checkbox" name="multi_tenant" value="1"> {{tr "projects.factory.multi_tenant"}}</label>
          <label><input type="checkbox" name="feature_flags" value="1"> {{tr "projects.factory.feature_flags"}}</label>
          <label><input type="checkbox" name="offline_mode" value="1"> {{tr "projects.factory.offline_mode"}}</label>
          <label><input type="checkbox" name="import_export" value="1"> {{tr "projects.factory.import_export"}}</label>
          <label><input type="checkbox" name="reporting" value="1"> {{tr "projects.factory.reporting"}}</label>
        </div>
        <label style="margin-top:.5rem">{{tr "projects.factory.integrations"}} <input name="integraciones" placeholder="payments, crm, email, sso"></label>
        {{with wizardHelp "nucleo_caps"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
      </fieldset>

      <fieldset>
        <legend>3. {{tr "projects.factory.section.interfaces"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="frontend" value="1" checked> {{tr "projects.factory.frontend"}}</label>
          <label><input type="checkbox" name="api" value="1" checked> {{tr "projects.factory.api"}}</label>
          <label><input type="checkbox" name="auth" value="1"> {{tr "projects.factory.auth"}}</label>
          <label><input type="checkbox" name="i18n" value="1" checked> {{tr "projects.factory.i18n"}}</label>
          <label><input type="checkbox" name="notifications" value="1"> {{tr "projects.factory.notifications"}}</label>
          <label><input type="checkbox" name="webhooks" value="1"> {{tr "projects.factory.webhooks"}}</label>
          <label><input type="checkbox" name="file_uploads" value="1"> {{tr "projects.factory.file_uploads"}}</label>
        </div>
        <div style="margin-top:.5rem">
          <strong>{{tr "projects.factory.languages"}}</strong>
          <div style="display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:.35rem;margin-top:.35rem">
            {{range .Idiomas}}
            <label><input type="checkbox" name="idioma" value="{{.Code}}" {{if .Default}}checked{{end}}> {{.Label}}</label>
            {{end}}
          </div>
        </div>
        <label style="margin-top:.5rem">{{tr "projects.factory.languages_extra"}} <input name="idiomas_extra" placeholder="nl, pl, ar"></label>
        <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.languages_extra_help"}}</small>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem;margin-top:.5rem">
          <label>{{tr "projects.factory.frontend_stack"}}
            <select name="frontend_stack">
              <option value="react">{{tr "projects.factory.frontend_stack.react"}}</option>
              <option value="vue">{{tr "projects.factory.frontend_stack.vue"}}</option>
              <option value="sveltekit">{{tr "projects.factory.frontend_stack.sveltekit"}}</option>
              <option value="nextjs">{{tr "projects.factory.frontend_stack.nextjs"}}</option>
              <option value="astro">{{tr "projects.factory.frontend_stack.astro"}}</option>
              <option value="htmx">{{tr "projects.factory.frontend_stack.htmx"}}</option>
              <option value="server_rendered">{{tr "projects.factory.frontend_stack.server_rendered"}}</option>
              <option value="angular">{{tr "projects.factory.frontend_stack.angular"}}</option>
              <option value="other">{{tr "projects.factory.frontend_stack.other"}}</option>
            </select>
          </label>
          {{with wizardHelp "frontend_stack"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.api_style"}}
            <select name="api_style">
              <option value="rest">{{tr "projects.factory.api_style.rest"}}</option>
              <option value="graphql">{{tr "projects.factory.api_style.graphql"}}</option>
              <option value="grpc">{{tr "projects.factory.api_style.grpc"}}</option>
              <option value="async">{{tr "projects.factory.api_style.async"}}</option>
              <option value="mixed">{{tr "projects.factory.api_style.mixed"}}</option>
            </select>
          </label>
          {{with wizardHelp "api_style"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.auth_mode"}}
            <select name="auth_mode">
              <option value="none">{{tr "projects.factory.auth_mode.none"}}</option>
              <option value="session">{{tr "projects.factory.auth_mode.session"}}</option>
              <option value="jwt">{{tr "projects.factory.auth_mode.jwt"}}</option>
              <option value="oauth">{{tr "projects.factory.auth_mode.oauth"}}</option>
              <option value="sso">{{tr "projects.factory.auth_mode.sso"}}</option>
              <option value="api_key">{{tr "projects.factory.auth_mode.api_key"}}</option>
            </select>
          </label>
          {{with wizardHelp "auth_mode"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.identity_provider"}}
            <select name="identity_provider">
              <option value="none">{{tr "projects.factory.identity_provider.none"}}</option>
              <option value="active_directory">{{tr "projects.factory.identity_provider.active_directory"}}</option>
              <option value="ldap">{{tr "projects.factory.identity_provider.ldap"}}</option>
              <option value="entra_id">{{tr "projects.factory.identity_provider.entra_id"}}</option>
              <option value="keycloak">{{tr "projects.factory.identity_provider.keycloak"}}</option>
              <option value="auth0">{{tr "projects.factory.identity_provider.auth0"}}</option>
              <option value="google">{{tr "projects.factory.identity_provider.google"}}</option>
              <option value="other">{{tr "projects.factory.identity_provider.other"}}</option>
            </select>
          </label>
          {{with wizardHelp "identity_provider"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
        </div>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem;margin-top:.5rem">
          <label><input type="checkbox" name="rbac" value="1"> {{tr "projects.factory.rbac"}}</label>
          <label><input type="checkbox" name="rate_limiting" value="1"> {{tr "projects.factory.rate_limiting"}}</label>
          <label><input type="checkbox" name="theme_support" value="1" checked> {{tr "projects.factory.theme_support"}}</label>
          <label><input type="checkbox" name="branding_profiles" value="1"> {{tr "projects.factory.branding_profiles"}}</label>
        </div>
        <article id="na-frontend-recommendation" style="margin-top:.75rem;padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
          <strong>{{tr "projects.factory.frontend_recommendation.title"}}: <span id="na-frontend-recommendation-label">{{.FrontendRecommendation.Label}}</span></strong>
          <p id="na-frontend-recommendation-reason" style="margin:.35rem 0 0">{{.FrontendRecommendation.Reason}}</p>
          <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.recommendation.manual"}}</small>
        </article>
        <article id="na-auth-recommendation" style="margin-top:.75rem;padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
          <strong>{{tr "projects.factory.auth_recommendation.title"}}: <span id="na-auth-recommendation-label">{{.AuthRecommendation.Label}}</span></strong>
          <p id="na-auth-recommendation-reason" style="margin:.35rem 0 0">{{.AuthRecommendation.Reason}}</p>
          <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.recommendation.manual"}}</small>
        </article>
        {{with wizardHelp "interface_caps"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
      </fieldset>

      <fieldset>
        <legend>4. {{tr "projects.factory.section.data"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="db" value="1"> {{tr "projects.factory.db"}}</label>
          <label><input type="checkbox" name="cache" value="1"> {{tr "projects.factory.cache"}}</label>
          <label><input type="checkbox" name="object_storage" value="1"> {{tr "projects.factory.object_storage"}}</label>
          <label><input type="checkbox" name="search" value="1"> {{tr "projects.factory.search"}}</label>
          <label><input type="checkbox" name="import_export" value="1"> {{tr "projects.factory.import_export"}}</label>
          <label><input type="checkbox" name="audit_trail" value="1"> {{tr "projects.factory.audit_trail"}}</label>
        </div>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem;margin-top:.5rem">
          <label>{{tr "projects.factory.db_engine"}}
            <select name="db_engine">
              <option value="none">{{tr "projects.factory.db_engine.none"}}</option>
              <option value="postgres">{{tr "projects.factory.db_engine.postgres"}}</option>
              <option value="mysql">{{tr "projects.factory.db_engine.mysql"}}</option>
              <option value="sqlite">{{tr "projects.factory.db_engine.sqlite"}}</option>
              <option value="mongodb">{{tr "projects.factory.db_engine.mongodb"}}</option>
              <option value="redis">{{tr "projects.factory.db_engine.redis"}}</option>
              <option value="other">{{tr "projects.factory.db_engine.other"}}</option>
            </select>
          </label>
        </div>
        {{with wizardHelp "db_engine"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
        <article id="na-db-recommendation" style="margin-top:.75rem;padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
          <strong>{{tr "projects.factory.db_recommendation.title"}}: <span id="na-db-recommendation-label">{{.DBRecommendation.Label}}</span></strong>
          <p id="na-db-recommendation-reason" style="margin:.35rem 0 0">{{.DBRecommendation.Reason}}</p>
          <small style="display:block;margin-top:.35rem;color:#64748b">{{tr "projects.factory.db_recommendation.filesystem_note"}}</small>
          <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.db_recommendation.manual"}}</small>
        </article>
        {{with wizardHelp "data_caps"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
      </fieldset>

      <fieldset>
        <legend>5. {{tr "projects.factory.section.platforms"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="plat_web" id="cb-plat-web" value="1"> {{tr "projects.factory.plat_web"}}</label>
          <label><input type="checkbox" name="plat_desktop" id="cb-plat-desktop" value="1"> {{tr "projects.factory.plat_desktop"}}</label>
          <label><input type="checkbox" name="plat_mobile" id="cb-plat-mobile" value="1"> {{tr "projects.factory.plat_mobile"}}</label>
          <label><input type="checkbox" name="plat_cli" id="cb-plat-cli" value="1"> {{tr "projects.factory.plat_cli"}}</label>
          <label><input type="checkbox" name="plat_embedded" id="cb-plat-embedded" value="1"> {{tr "projects.factory.plat_embedded"}}</label>
        </div>
      </fieldset>

      <fieldset>
        <legend>6. {{tr "projects.factory.section.os"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="so_linux" id="cb-so-linux" value="1"> Linux</label>
          <label><input type="checkbox" name="so_windows" id="cb-so-windows" value="1"> Windows</label>
          <label><input type="checkbox" name="so_macos" id="cb-so-macos" value="1"> macOS</label>
          <label><input type="checkbox" name="so_android" id="cb-so-android" value="1"> Android</label>
          <label><input type="checkbox" name="so_ios" id="cb-so-ios" value="1"> iOS</label>
        </div>
      </fieldset>

      <fieldset>
        <legend>7. {{tr "projects.factory.section.compliance"}}</legend>
        <p style="font-size:.8rem;color:var(--pico-muted-color)">{{tr "projects.factory.compliance_note"}}</p>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.75rem">
          <div><label><input type="checkbox" name="compliance_rgpd" value="1"> RGPD</label>
          <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_rgpd_help"}}</small></div>
          <div><label><input type="checkbox" name="compliance_ens" value="1"> ENS</label>
          <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_ens_help"}}</small></div>
          <div><label><input type="checkbox" name="compliance_lssi" value="1"> LSSI</label>
          <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_lssi_help"}}</small></div>
          <div><label><input type="checkbox" name="compliance_wcag" value="1"> WCAG 2.1 AA</label>
          <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_wcag_help"}}</small></div>
          <div><label><input type="checkbox" name="compliance_factura_elec" value="1"> {{tr "projects.factory.compliance_factura_elec"}}</label>
          <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_factura_elec_help"}}</small></div>
          <div><label><input type="checkbox" name="compliance_reutilizacion" value="1"> {{tr "projects.factory.compliance_reutilizacion"}}</label>
          <small style="display:block;color:var(--pico-muted-color);font-size:.75rem;margin-left:1.4rem">{{tr "projects.factory.compliance_reutilizacion_help"}}</small></div>
        </div>
      </fieldset>

      <fieldset>
        <legend>8. {{tr "projects.factory.section.delivery_quality"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="docker" id="cb-docker" value="1" checked> {{tr "projects.factory.docker"}}</label>
          <label><input type="checkbox" name="ci" value="1" checked> CI/CD — {{tr "projects.factory.ci"}}</label>
          <label><input type="checkbox" name="kubernetes" id="cb-kubernetes" value="1"> Kubernetes — {{tr "projects.factory.kubernetes"}}</label>
          <label><input type="checkbox" name="terraform" value="1"> Terraform — {{tr "projects.factory.terraform"}}</label>
          <label><input type="checkbox" name="monitoring" value="1"> {{tr "projects.factory.monitoring"}}</label>
          <label><input type="checkbox" name="backups" value="1"> {{tr "projects.factory.backups"}}</label>
          <label><input type="checkbox" name="disaster_recovery" value="1"> {{tr "projects.factory.disaster_recovery"}}</label>
        </div>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem;margin-top:.5rem">
          <label>{{tr "projects.factory.testing_level"}}
            <select name="testing_level">
              <option value="base">{{tr "projects.factory.testing_level.base"}}</option>
              <option value="strict">{{tr "projects.factory.testing_level.strict"}}</option>
              <option value="tdd">{{tr "projects.factory.testing_level.tdd"}}</option>
            </select>
          </label>
          {{with wizardHelp "testing_level"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.observability_level"}}
            <select name="observability_level">
              <option value="basic">{{tr "projects.factory.observability_level.basic"}}</option>
              <option value="standard">{{tr "projects.factory.observability_level.standard"}}</option>
              <option value="strict">{{tr "projects.factory.observability_level.strict"}}</option>
            </select>
          </label>
          {{with wizardHelp "observability_level"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.deployment_target"}}
            <select name="deployment_target">
              <option value="docker">{{tr "projects.factory.deployment_target.docker"}}</option>
              <option value="kubernetes">{{tr "projects.factory.deployment_target.kubernetes"}}</option>
              <option value="serverless">{{tr "projects.factory.deployment_target.serverless"}}</option>
              <option value="vm">{{tr "projects.factory.deployment_target.vm"}}</option>
              <option value="edge">{{tr "projects.factory.deployment_target.edge"}}</option>
            </select>
          </label>
          {{with wizardHelp "deployment_target"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
          <label>{{tr "projects.factory.artifact_type"}}
            <select name="artifact_type">
              <option value="executable">{{tr "projects.factory.artifact_type.executable"}}</option>
              <option value="docker_image">{{tr "projects.factory.artifact_type.docker_image"}}</option>
              <option value="library">{{tr "projects.factory.artifact_type.library"}}</option>
              <option value="desktop_installer">{{tr "projects.factory.artifact_type.desktop_installer"}}</option>
              <option value="mobile_bundle">{{tr "projects.factory.artifact_type.mobile_bundle"}}</option>
              <option value="firmware">{{tr "projects.factory.artifact_type.firmware"}}</option>
              <option value="static_site">{{tr "projects.factory.artifact_type.static_site"}}</option>
            </select>
          </label>
          {{with wizardHelp "artifact_type"}}
          <details style="grid-column:1 / -1;margin:0">
            <summary>{{tr "projects.factory.help_details"}}</summary>
            <ul style="margin:.5rem 0 0 1rem">
              {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
            </ul>
          </details>
          {{end}}
        </div>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.75rem;margin-top:.75rem">
          <article id="na-deployment-recommendation" style="padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
            <strong>{{tr "projects.factory.deployment_recommendation.title"}}: <span id="na-deployment-recommendation-label">{{.DeployRecommendation.Label}}</span></strong>
            <p id="na-deployment-recommendation-reason" style="margin:.35rem 0 0">{{.DeployRecommendation.Reason}}</p>
            <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.recommendation.manual"}}</small>
          </article>
          <article id="na-artifact-recommendation" style="padding:.75rem;border:1px solid #cbd5e1;background:#f8fafc">
            <strong>{{tr "projects.factory.artifact_recommendation.title"}}: <span id="na-artifact-recommendation-label">{{.ArtifactRecommendation.Label}}</span></strong>
            <p id="na-artifact-recommendation-reason" style="margin:.35rem 0 0">{{.ArtifactRecommendation.Reason}}</p>
            <small style="display:block;margin-top:.2rem;color:#64748b">{{tr "projects.factory.recommendation.manual"}}</small>
          </article>
        </div>
        {{with wizardHelp "delivery_caps"}}
        <details style="margin-top:.5rem">
          <summary>{{tr "projects.factory.help_details"}}</summary>
          <ul style="margin:.5rem 0 0 1rem">
            {{range .}}<li><strong>{{.Option}}:</strong> {{.Description}}</li>{{end}}
          </ul>
        </details>
        {{end}}
      </fieldset>

      <label>{{tr "projects.factory.actor"}} <input name="por" value="web"></label>
      <div style="display:flex;gap:.5rem;flex-wrap:wrap">
        <button type="button" class="secondary" onclick="naPreview()">{{tr "projects.factory.preview_btn"}}</button>
        <button type="submit">{{tr "projects.factory.submit"}}</button>
      </div>
    </form>

    {{if .Preview}}
    <div style="margin-top:1.5rem">
      <h4>{{tr "projects.factory.preview_title"}} ({{len .Preview}})</h4>
      <table style="font-size:.8rem">
        <thead><tr><th>{{tr "projects.factory.preview_col_key"}}</th><th>{{tr "projects.factory.preview_col_title"}}</th><th>{{tr "projects.factory.preview_col_phase"}}</th><th>{{tr "projects.factory.preview_col_module"}}</th></tr></thead>
        <tbody>
          {{range .Preview}}
          <tr><td><code>{{.Key}}</code></td><td>{{.Titulo}}</td><td>{{.Fase}}</td><td>{{.Modulo}}</td></tr>
          {{end}}
        </tbody>
      </table>
    </div>
    {{end}}
  </article>
</section>
<script>
(function(){
  // sync slug hidden field from select
  var sel = document.querySelector('select[name=proyecto_slug]');
  var hid = document.getElementById('na-slug-hidden');
  if(sel && hid){ sel.addEventListener('change',function(){ hid.value=this.value; }); sel.dispatchEvent(new Event('change')); }

  function cb(id){ return document.getElementById(id); }
  var form=document.getElementById('nueva-app-form');
  var depRules = [
    {src:'cb-plat-mobile',   targets:['cb-so-android','cb-so-ios']},
    {src:'cb-plat-desktop',  targets:['cb-so-linux','cb-so-windows','cb-so-macos']},
    {src:'cb-plat-embedded', targets:['cb-so-linux']},
    {src:'cb-plat-cli',      targets:['cb-so-linux']},
    {src:'cb-kubernetes',    targets:['cb-docker']},
  ];
  var dbEngineLabels = {
    none: {{printf "%q" (tr "projects.factory.db_engine.none")}},
    sqlite: {{printf "%q" (tr "projects.factory.db_engine.sqlite")}},
    postgres: {{printf "%q" (tr "projects.factory.db_engine.postgres")}}
  };
  var frontendStackLabels = {
    react: {{printf "%q" (tr "projects.factory.frontend_stack.react")}},
    server_rendered: {{printf "%q" (tr "projects.factory.frontend_stack.server_rendered")}}
  };
  var frontendStackReasons = {
    react: {{printf "%q" (tr "projects.factory.frontend_recommendation.reason.react")}},
    server_rendered: {{printf "%q" (tr "projects.factory.frontend_recommendation.reason.server_rendered")}}
  };
  var dbEngineReasons = {
    none: {{printf "%q" (tr "projects.factory.db_recommendation.reason.none")}},
    sqlite: {{printf "%q" (tr "projects.factory.db_recommendation.reason.sqlite")}},
    postgres: {{printf "%q" (tr "projects.factory.db_recommendation.reason.postgres")}}
  };
  var authModeLabels = {
    none: {{printf "%q" (tr "projects.factory.auth_mode.none")}},
    session: {{printf "%q" (tr "projects.factory.auth_mode.session")}},
    jwt: {{printf "%q" (tr "projects.factory.auth_mode.jwt")}},
    oauth: {{printf "%q" (tr "projects.factory.auth_mode.oauth")}},
    sso: {{printf "%q" (tr "projects.factory.auth_mode.sso")}},
    api_key: {{printf "%q" (tr "projects.factory.auth_mode.api_key")}}
  };
  var authModeReasons = {
    none: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.none")}},
    session: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.session")}},
    jwt: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.jwt")}},
    oauth: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.oauth")}},
    sso: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.sso")}},
    api_key: {{printf "%q" (tr "projects.factory.auth_recommendation.reason.api_key")}}
  };
  var deploymentLabels = {
    docker: {{printf "%q" (tr "projects.factory.deployment_target.docker")}},
    kubernetes: {{printf "%q" (tr "projects.factory.deployment_target.kubernetes")}},
    serverless: {{printf "%q" (tr "projects.factory.deployment_target.serverless")}},
    vm: {{printf "%q" (tr "projects.factory.deployment_target.vm")}},
    edge: {{printf "%q" (tr "projects.factory.deployment_target.edge")}}
  };
  var deploymentReasons = {
    docker: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.docker")}},
    kubernetes: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.kubernetes")}},
    serverless: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.serverless")}},
    vm: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.vm")}},
    edge: {{printf "%q" (tr "projects.factory.deployment_recommendation.reason.edge")}}
  };
  var artifactLabels = {
    executable: {{printf "%q" (tr "projects.factory.artifact_type.executable")}},
    docker_image: {{printf "%q" (tr "projects.factory.artifact_type.docker_image")}},
    library: {{printf "%q" (tr "projects.factory.artifact_type.library")}},
    desktop_installer: {{printf "%q" (tr "projects.factory.artifact_type.desktop_installer")}},
    mobile_bundle: {{printf "%q" (tr "projects.factory.artifact_type.mobile_bundle")}},
    firmware: {{printf "%q" (tr "projects.factory.artifact_type.firmware")}},
    static_site: {{printf "%q" (tr "projects.factory.artifact_type.static_site")}}
  };
  var artifactReasons = {
    executable: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.executable")}},
    docker_image: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.docker_image")}},
    library: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.library")}},
    desktop_installer: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.desktop_installer")}},
    mobile_bundle: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.mobile_bundle")}},
    firmware: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.firmware")}},
    static_site: {{printf "%q" (tr "projects.factory.artifact_recommendation.reason.static_site")}}
  };
  function syncDeps(src,targets,checked){
    targets.forEach(function(id){
      var el=cb(id); if(!el) return;
      if(checked){ el.checked=true; return; }
      var still=false;
      depRules.forEach(function(r){ if(r.targets.indexOf(id)!==-1&&r.src!==src){ var s=cb(r.src); if(s&&s.checked) still=true; } });
      if(!still) el.checked=false;
    });
  }
  depRules.forEach(function(r){ var el=cb(r.src); if(!el) return; el.addEventListener('change',function(){ syncDeps(r.src,r.targets,this.checked); }); });

  function fieldValue(name, fallback){
    var el = form ? form.querySelector('[name="'+name+'"]') : null;
    if(!el){ return (fallback||'').toLowerCase(); }
    return String(el.value || fallback || '').toLowerCase().trim();
  }
  function fieldChecked(name){
    return !!(form && form.querySelector('input[name="'+name+'"]:checked'));
  }
  function recommendDBEngine(){
    var type = fieldValue('tipo', 'web_api');
    var authMode = fieldValue('auth_mode', 'none');
    var identityProvider = fieldValue('identity_provider', 'none');
    var deploymentTarget = fieldValue('deployment_target', 'docker');
    var artifactType = fieldValue('artifact_type', 'executable');
    var durable = fieldChecked('db') || fieldChecked('auth') || fieldChecked('file_uploads') ||
      fieldChecked('import_export') || fieldChecked('reporting') || fieldChecked('search') ||
      fieldChecked('audit_trail') || fieldChecked('multi_tenant') || fieldChecked('background_jobs') ||
      fieldChecked('queue') || fieldChecked('scheduler') || fieldChecked('servicio_residente') ||
      fieldChecked('notifications') || fieldChecked('webhooks') || fieldChecked('object_storage') ||
      fieldChecked('backups') || fieldChecked('disaster_recovery') || authMode !== 'none' || identityProvider !== 'none';
    var serverGrade = type === 'api' || type === 'web_api' || fieldChecked('api') ||
      fieldChecked('multi_tenant') || fieldChecked('rbac') || fieldChecked('reporting') ||
      fieldChecked('background_jobs') || fieldChecked('queue') || fieldChecked('scheduler') ||
      fieldChecked('servicio_residente') || fieldChecked('notifications') || fieldChecked('webhooks') ||
      fieldChecked('audit_trail') || fieldChecked('backups') || fieldChecked('disaster_recovery') ||
      authMode === 'oauth' || authMode === 'sso' || authMode === 'api_key' || identityProvider !== 'none' ||
      deploymentTarget === 'kubernetes' || fieldChecked('kubernetes') || artifactType === 'docker_image';
    var embeddedPreferred = (fieldChecked('offline_mode') || type === 'desktop' || type === 'mobile' || type === 'cli' ||
      type === 'embedded' || fieldChecked('plat_desktop') || fieldChecked('plat_mobile') ||
      fieldChecked('plat_cli') || fieldChecked('plat_embedded')) && !serverGrade;
    if(!durable && !serverGrade && !embeddedPreferred){ return 'none'; }
    if(embeddedPreferred){ return 'sqlite'; }
    if(serverGrade){ return 'postgres'; }
    if(durable){
      if(fieldChecked('offline_mode') || fieldChecked('plat_desktop') || fieldChecked('plat_mobile') ||
         fieldChecked('plat_cli') || fieldChecked('plat_embedded')) {
        return 'sqlite';
      }
      return 'postgres';
    }
    return 'none';
  }
  function recommendFrontendStack(){
    var type = fieldValue('tipo', 'web_api');
    if(!fieldChecked('frontend') && type !== 'web' && type !== 'web_api'){ return 'server_rendered'; }
    if(fieldChecked('offline_mode') || fieldChecked('reporting') || fieldChecked('feature_flags') ||
       fieldChecked('multi_tenant') || fieldChecked('file_uploads') || fieldChecked('branding_profiles')) {
      return 'react';
    }
    if(type === 'web' && !fieldChecked('api') && !fieldChecked('servicio_residente') && !fieldChecked('background_jobs')){ return 'server_rendered'; }
    return 'react';
  }
  function recommendAuthMode(){
    var type = fieldValue('tipo', 'web_api');
    var identityProvider = fieldValue('identity_provider', 'none');
    if(!fieldChecked('auth')){ return 'none'; }
    if(identityProvider !== 'none'){ return 'sso'; }
    if(type === 'mobile'){ return 'oauth'; }
    if(type === 'cli'){ return 'api_key'; }
    if(type === 'api' && !fieldChecked('frontend')){ return 'jwt'; }
    return 'session';
  }
  function recommendDeploymentTarget(){
    var type = fieldValue('tipo', 'web_api');
    if(fieldChecked('kubernetes')){ return 'kubernetes'; }
    if(type === 'embedded' || fieldChecked('plat_embedded')){ return 'edge'; }
    if(type === 'desktop' || type === 'mobile' || type === 'cli' || fieldChecked('plat_desktop') || fieldChecked('plat_mobile') || fieldChecked('plat_cli')){ return 'vm'; }
    if(type === 'web' && !fieldChecked('api') && !fieldChecked('servicio_residente') && !fieldChecked('background_jobs') && !fieldChecked('queue') && !fieldChecked('scheduler')){ return 'serverless'; }
    return 'docker';
  }
  function recommendArtifactType(){
    var type = fieldValue('tipo', 'web_api');
    var deploymentTarget = fieldValue('deployment_target', 'docker');
    if(type === 'embedded' || fieldChecked('plat_embedded')){ return 'firmware'; }
    if(type === 'mobile' || fieldChecked('plat_mobile')){ return 'mobile_bundle'; }
    if(type === 'desktop' || fieldChecked('plat_desktop')){ return 'desktop_installer'; }
    if(type === 'web' && !fieldChecked('api') && !fieldChecked('servicio_residente') && !fieldChecked('background_jobs') && !fieldChecked('queue') && !fieldChecked('scheduler')){ return 'static_site'; }
    if(deploymentTarget === 'kubernetes' || deploymentTarget === 'docker' || fieldChecked('kubernetes')){ return 'docker_image'; }
    return 'executable';
  }
  var dbSelect = form ? form.querySelector('select[name="db_engine"]') : null;
  var dbToggle = form ? form.querySelector('input[name="db"]') : null;
  var dbRecommendationLabel = document.getElementById('na-db-recommendation-label');
  var dbRecommendationReason = document.getElementById('na-db-recommendation-reason');
  var frontendSelect = form ? form.querySelector('select[name="frontend_stack"]') : null;
  var frontendRecommendationLabel = document.getElementById('na-frontend-recommendation-label');
  var frontendRecommendationReason = document.getElementById('na-frontend-recommendation-reason');
  var authSelect = form ? form.querySelector('select[name="auth_mode"]') : null;
  var authToggle = form ? form.querySelector('input[name="auth"]') : null;
  var authRecommendationLabel = document.getElementById('na-auth-recommendation-label');
  var authRecommendationReason = document.getElementById('na-auth-recommendation-reason');
  var deploymentSelect = form ? form.querySelector('select[name="deployment_target"]') : null;
  var deploymentRecommendationLabel = document.getElementById('na-deployment-recommendation-label');
  var deploymentRecommendationReason = document.getElementById('na-deployment-recommendation-reason');
  var artifactSelect = form ? form.querySelector('select[name="artifact_type"]') : null;
  var artifactRecommendationLabel = document.getElementById('na-artifact-recommendation-label');
  var artifactRecommendationReason = document.getElementById('na-artifact-recommendation-reason');
  var dbEngineTouched = false;
  var dbToggleTouched = false;
  var frontendTouched = false;
  var authModeTouched = false;
  var authToggleTouched = false;
  var deploymentTouched = false;
  var artifactTouched = false;
  if(dbSelect){
    dbSelect.addEventListener('change', function(){
      dbEngineTouched = true;
      this.dataset.auto = '0';
    });
  }
  if(dbToggle){
    dbToggle.addEventListener('change', function(){
      dbToggleTouched = true;
      this.dataset.auto = '0';
    });
  }
  if(authSelect){
    authSelect.addEventListener('change', function(){
      authModeTouched = true;
      this.dataset.auto = '0';
    });
  }
  if(frontendSelect){
    frontendSelect.addEventListener('change', function(){
      frontendTouched = true;
      this.dataset.auto = '0';
    });
  }
  if(authToggle){
    authToggle.addEventListener('change', function(){
      authToggleTouched = true;
      this.dataset.auto = '0';
    });
  }
  if(deploymentSelect){
    deploymentSelect.addEventListener('change', function(){
      deploymentTouched = true;
      this.dataset.auto = '0';
    });
  }
  if(artifactSelect){
    artifactSelect.addEventListener('change', function(){
      artifactTouched = true;
      this.dataset.auto = '0';
    });
  }
  function syncDBRecommendation(){
    var engine = recommendDBEngine();
    if(dbRecommendationLabel){ dbRecommendationLabel.textContent = dbEngineLabels[engine] || engine; }
    if(dbRecommendationReason){ dbRecommendationReason.textContent = dbEngineReasons[engine] || ''; }
    if(dbSelect && (!dbEngineTouched || dbSelect.dataset.auto === '1' || dbSelect.value === '' || dbSelect.value === 'none')){
      dbSelect.value = engine;
      dbSelect.dataset.auto = '1';
    }
    if(dbToggle && (!dbToggleTouched || dbToggle.dataset.auto === '1')){
      dbToggle.checked = engine !== 'none';
      dbToggle.dataset.auto = '1';
    }
  }
  function syncFrontendRecommendation(){
    var value = recommendFrontendStack();
    if(frontendRecommendationLabel){ frontendRecommendationLabel.textContent = frontendStackLabels[value] || value; }
    if(frontendRecommendationReason){ frontendRecommendationReason.textContent = frontendStackReasons[value] || ''; }
    if(frontendSelect && (!frontendTouched || frontendSelect.dataset.auto === '1' || frontendSelect.value === '')){
      frontendSelect.value = value;
      frontendSelect.dataset.auto = '1';
    }
  }
  function syncAuthRecommendation(){
    var value = recommendAuthMode();
    if(authRecommendationLabel){ authRecommendationLabel.textContent = authModeLabels[value] || value; }
    if(authRecommendationReason){ authRecommendationReason.textContent = authModeReasons[value] || ''; }
    if(authSelect && (!authModeTouched || authSelect.dataset.auto === '1' || authSelect.value === '' || authSelect.value === 'none')){
      authSelect.value = value;
      authSelect.dataset.auto = '1';
    }
    if(authToggle && (!authToggleTouched || authToggle.dataset.auto === '1')){
      authToggle.checked = value !== 'none';
      authToggle.dataset.auto = '1';
    }
  }
  function syncDeploymentRecommendation(){
    var value = recommendDeploymentTarget();
    if(deploymentRecommendationLabel){ deploymentRecommendationLabel.textContent = deploymentLabels[value] || value; }
    if(deploymentRecommendationReason){ deploymentRecommendationReason.textContent = deploymentReasons[value] || ''; }
    if(deploymentSelect && (!deploymentTouched || deploymentSelect.dataset.auto === '1' || deploymentSelect.value === '')){
      deploymentSelect.value = value;
      deploymentSelect.dataset.auto = '1';
    }
  }
  function syncArtifactRecommendation(){
    var value = recommendArtifactType();
    if(artifactRecommendationLabel){ artifactRecommendationLabel.textContent = artifactLabels[value] || value; }
    if(artifactRecommendationReason){ artifactRecommendationReason.textContent = artifactReasons[value] || ''; }
    if(artifactSelect && (!artifactTouched || artifactSelect.dataset.auto === '1' || artifactSelect.value === '')){
      artifactSelect.value = value;
      artifactSelect.dataset.auto = '1';
    }
  }

  var PRESETS = {
    'web-publica':{tipo:'web_api',checks:{frontend:1,api:1,auth:1,db:1,docker:1,i18n:1,ci:1,plat_web:1,compliance_rgpd:1,compliance_lssi:1,compliance_wcag:1}},
    'api-interna':{tipo:'api',checks:{api:1,auth:1,db:1,docker:1,ci:1,plat_web:1}},
    'cli-devops':{tipo:'cli',checks:{docker:1,ci:1,terraform:1,plat_cli:1,so_linux:1}},
    'app-movil':{tipo:'mobile',checks:{auth:1,db:1,plat_mobile:1,so_android:1,so_ios:1,compliance_rgpd:1}},
    'embedded':{tipo:'embedded',checks:{docker:1,plat_embedded:1,so_linux:1}}
  };
  window.factoryApplyPreset=function(key){
    if(!key) return;
    var p=PRESETS[key]; if(!p) return;
    form.querySelectorAll('input[type=checkbox]').forEach(function(el){ el.checked=false; });
    var t=document.getElementById('factory-tipo'); if(t) t.value=p.tipo;
    Object.keys(p.checks).forEach(function(n){ var el=form.querySelector('input[name="'+n+'"]'); if(el) el.checked=true; });
    dbEngineTouched = false;
    dbToggleTouched = false;
    frontendTouched = false;
    authModeTouched = false;
    authToggleTouched = false;
    deploymentTouched = false;
    artifactTouched = false;
    document.getElementById('na-preset').value='';
    syncDBRecommendation();
    syncFrontendRecommendation();
    syncAuthRecommendation();
    syncDeploymentRecommendation();
    syncArtifactRecommendation();
  };
  if(form){
    form.querySelectorAll('input,select,textarea').forEach(function(el){
      if(el.name === 'db_engine' || el.name === 'db' || el.name === 'frontend_stack' || el.name === 'auth_mode' || el.name === 'auth' || el.name === 'deployment_target' || el.name === 'artifact_type'){ return; }
      el.addEventListener('change', function(){
        syncDBRecommendation();
        syncFrontendRecommendation();
        syncAuthRecommendation();
        syncDeploymentRecommendation();
        syncArtifactRecommendation();
      });
    });
  }
  syncDBRecommendation();
  syncFrontendRecommendation();
  syncAuthRecommendation();
  syncDeploymentRecommendation();
  syncArtifactRecommendation();
  window.naPreview=function(){
    var slug=document.querySelector('select[name=proyecto_slug]');
    if(!slug||!slug.value){ alert({{printf "%q" (tr "projects.factory.select_project_error")}}); return; }
    var data=new FormData(form);
    var obj={}; data.forEach(function(v,k){ obj[k]=v; });
    var bools=['frontend','api','auth','db','docker','i18n','plat_web','plat_desktop','plat_mobile','plat_cli','plat_embedded','so_linux','so_windows','so_macos','so_android','so_ios','compliance_rgpd','compliance_ens','compliance_lssi','compliance_wcag','compliance_factura_elec','compliance_reutilizacion','ci','kubernetes','terraform','monitoring','background_jobs','notifications','multi_tenant','rbac','theme_support','branding_profiles','offline_mode','import_export','webhooks','file_uploads','reporting','servicio_residente','cache','queue','scheduler','object_storage','search','rate_limiting','feature_flags','audit_trail','backups','disaster_recovery'];
    var payload={
      nombre:obj['nombre']||'',
      descripcion:obj['descripcion']||'',
      objetivo_negocio:obj['objetivo_negocio']||'',
      usuarios_objetivo:obj['usuarios_objetivo']||'',
      restricciones:obj['restricciones']||'',
      tipo:obj['tipo']||'',
      idiomas:Array.from(form.querySelectorAll('input[name="idioma"]:checked')).map(function(el){ return el.value; }),
      por:obj['por']||'web',
      arquitectura:obj['arquitectura']||'hexagonal',
      api_style:obj['api_style']||'rest',
      frontend_stack:obj['frontend_stack']||'react',
      db_engine:obj['db_engine']||'none',
      auth_mode:obj['auth_mode']||'none',
      identity_provider:obj['identity_provider']||'none',
      testing_level:obj['testing_level']||'base',
      observability_level:obj['observability_level']||'standard',
      deployment_target:obj['deployment_target']||'docker',
      artifact_type:obj['artifact_type']||'executable',
      integraciones:(obj['integraciones']||'').split(',')
    };
    if(obj['idiomas_extra']){ payload.idiomas = payload.idiomas.concat(obj['idiomas_extra'].split(',')); }
    bools.forEach(function(f){ payload[f]=obj[f]==='1'; });
    fetch('/api/proyectos/'+encodeURIComponent(slug.value)+'/fabricar-app/preview',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)})
    .then(function(r){ return r.json(); })
    .then(function(d){
      if(!d.ok||!d.tasks) return;
      var form2=document.createElement('form');
      form2.method='post'; form2.action='/nueva-app?preview=1&lang={{lang}}';
      Object.keys(obj).forEach(function(k){ var i=document.createElement('input'); i.type='hidden'; i.name=k; i.value=obj[k]||''; form2.appendChild(i); });
      var si=document.createElement('input'); si.type='hidden'; si.name='proyecto_slug'; si.value=slug.value; form2.appendChild(si);
      document.body.appendChild(form2); form2.submit();
    })
    .catch(function(e){
      alert({{printf "%q" (tr "projects.factory.preview_error")}} + ': ' + e);
    });
  };
})();
</script>
{{end}}`
