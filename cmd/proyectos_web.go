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

	"orquesta/db"
	"orquesta/memoriaproyecto"
	"orquesta/reviewapp"
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

type webProyectoDetalleData struct {
	Proyecto    *db.Proyecto
	Cockpit     *apiProyectoCockpit
	Operacion   *db.ProyectoOperacion
	Autonomia   *db.ProyectoAutonomia
	Ciclos      []*db.AutonomiaCiclo
	ReviewGates []*reviewapp.Gate
	Votaciones  []*db.HistorialVotacionProyecto
	Decisiones  []*db.DecisionProyecto
	Documentos  []*db.DocumentoExterno
	Msg         string
	Err         string
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

func webHandlerProyectoDetalle(w http.ResponseWriter, r *http.Request, slug string) {
	overview, err := webCargarProyectoOverviewPorAPI(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cockpit, errCockpit := webCargarProyectoCockpitPorAPI(slug)
	operacion, err := webCargarProyectoOperacionPorAPI(slug)
	autonomia, errAutonomia := webCargarProyectoAutonomiaPorAPI(slug)
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
	var cicloItems []*db.AutonomiaCiclo
	var autonomiaItem *db.ProyectoAutonomia
	if errAutonomia == nil && autonomia != nil {
		autonomiaItem = autonomia.Policy
		cicloItems = autonomia.Cycles
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
	webRender(w, r, webTplLayout+webTplProyectoDetalle, webProyectoDetalleData{
		Proyecto:    overview.Proyecto,
		Cockpit:     cockpit,
		Operacion:   operacion,
		Autonomia:   autonomiaItem,
		Ciclos:      cicloItems,
		ReviewGates: reviewGates,
		Votaciones:  overview.Votaciones,
		Decisiones:  overview.Decisiones,
		Documentos:  overview.Documentos,
		Msg:         r.URL.Query().Get("ok"),
		Err:         strings.TrimSpace(strings.Join(append([]string{r.URL.Query().Get("err")}, errParts...), " · ")),
	})
}

func webHandlerProyectoOperacionGuardar(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
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
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/proyectos/"+slug+"?ok="+url.QueryEscape(webTranslateRequestf(r, "projects.flash.operation_saved")), http.StatusSeeOther)
}

func webHandlerProyectoAutonomiaGuardar(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
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
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/proyectos/"+slug+"?ok="+url.QueryEscape(webTranslateRequestf(r, "projects.flash.autonomy_saved")), http.StatusSeeOther)
}

func webHandlerProyectoReviewGateResolver(w http.ResponseWriter, r *http.Request, slug string, gateID string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(gateID), 10, 64)
	if err != nil || id <= 0 {
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape("gate inválida"), http.StatusSeeOther)
		return
	}
	req := apiReviewGateResolveRequest{
		Estado:         strings.TrimSpace(r.FormValue("estado")),
		ReviewerAgente: strings.TrimSpace(r.FormValue("reviewer_agente")),
		FindingsJSON:   strings.TrimSpace(r.FormValue("findings_json")),
	}
	if _, err := webResolverReviewGatePorAPI(id, req); err != nil {
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/proyectos/"+slug+"?ok="+url.QueryEscape(webTranslateRequestf(r, "projects.flash.review_gate_saved")), http.StatusSeeOther)
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
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/proyectos/"+slug+"?ok="+url.QueryEscape(webTranslateRequestf(r, "projects.flash.decision_saved")), http.StatusSeeOther)
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
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/proyectos/"+slug+"?ok="+url.QueryEscape(webTranslateRequestf(r, "projects.flash.document_saved")), http.StatusSeeOther)
}

func webHandlerProyectoFabricarApp(w http.ResponseWriter, r *http.Request, slug string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	actor := strings.TrimSpace(r.FormValue("por"))
	if actor == "" {
		actor = "web"
	}
	if _, err := webFabricarAppProyectoPorAPI(slug, apiProyectoFabricarAppRequest{
		Nombre:      strings.TrimSpace(r.FormValue("nombre")),
		Descripcion: strings.TrimSpace(r.FormValue("descripcion")),
		Tipo:        strings.TrimSpace(r.FormValue("tipo")),
		Frontend:    webFormBool(r, "frontend"),
		API:         webFormBool(r, "api"),
		Auth:        webFormBool(r, "auth"),
		Database:    webFormBool(r, "db"),
		Docker:      webFormBool(r, "docker"),
		I18n:        webFormBool(r, "i18n"),
		Idiomas:     splitCSV(strings.TrimSpace(r.FormValue("idiomas"))),
		Por:         actor,

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
	}); err != nil {
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/proyectos/"+slug+"?ok="+url.QueryEscape(webTranslateRequestf(r, "projects.flash.factory_created")), http.StatusSeeOther)
}

func webFabricarAppProyectoPorAPI(ref string, req apiProyectoFabricarAppRequest) (*apiProyectoFabricarAppResponse, error) {
	var resp apiProyectoFabricarAppResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref)) + "/fabricar-app"
	if err := webInvocarAPIJSON(http.MethodPost, path, req, &resp); err != nil {
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

func webGuardarProyectoAutonomiaPorAPI(ref string, req apiProyectoAutonomiaSaveRequest) (*db.ProyectoAutonomia, error) {
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

const webTplProyectos = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "projects.title"}} <small style="font-size:.5em;color:#94a3b8">{{len .Proyectos}}</small></h2>
  <p style="color:#64748b">{{tr "projects.subtitle"}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
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
            proposals {{.Cockpit.PropuestasAbiertas}} · mailbox {{.Cockpit.RuntimeMailboxPendiente}} · orders {{.Cockpit.RuntimeOrdersAbiertas}}
          </small>
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
    {{else}}
      <p>Sin resumen operativo.</p>
    {{end}}
  </article>

  <article>
    <h3>{{tr "projects.operation.title"}}</h3>
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/operacion">
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
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/autonomia">
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

    <form id="factory-form" method="post" action="/proyectos/{{.Proyecto.Slug}}/fabricar-app">
      <label>{{tr "projects.factory.name"}} <input name="nombre" value="{{.Proyecto.Nombre}}" required></label>
      <label>{{tr "projects.factory.description"}} <textarea name="descripcion"></textarea></label>

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

      <fieldset>
        <legend>{{tr "projects.factory.section.components"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="frontend" value="1" checked> {{tr "projects.factory.frontend"}}</label>
          <label><input type="checkbox" name="api" value="1" checked> {{tr "projects.factory.api"}}</label>
          <label><input type="checkbox" name="auth" value="1"> {{tr "projects.factory.auth"}}</label>
          <label><input type="checkbox" name="db" value="1"> {{tr "projects.factory.db"}}</label>
          <label><input type="checkbox" name="docker" id="cb-docker" value="1" checked> {{tr "projects.factory.docker"}}</label>
          <label><input type="checkbox" name="i18n" value="1" checked> {{tr "projects.factory.i18n"}}</label>
        </div>
        <label style="margin-top:.5rem">{{tr "projects.factory.languages"}} <input name="idiomas" value="es,en"></label>
      </fieldset>

      <fieldset>
        <legend>{{tr "projects.factory.section.platforms"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="plat_web" id="cb-plat-web" value="1"> {{tr "projects.factory.plat_web"}}</label>
          <label><input type="checkbox" name="plat_desktop" id="cb-plat-desktop" value="1"> {{tr "projects.factory.plat_desktop"}}</label>
          <label><input type="checkbox" name="plat_mobile" id="cb-plat-mobile" value="1"> {{tr "projects.factory.plat_mobile"}}</label>
          <label><input type="checkbox" name="plat_cli" id="cb-plat-cli" value="1"> {{tr "projects.factory.plat_cli"}}</label>
          <label><input type="checkbox" name="plat_embedded" id="cb-plat-embedded" value="1"> {{tr "projects.factory.plat_embedded"}}</label>
        </div>
      </fieldset>

      <fieldset>
        <legend>{{tr "projects.factory.section.os"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="so_linux" id="cb-so-linux" value="1"> Linux</label>
          <label><input type="checkbox" name="so_windows" id="cb-so-windows" value="1"> Windows</label>
          <label><input type="checkbox" name="so_macos" id="cb-so-macos" value="1"> macOS</label>
          <label><input type="checkbox" name="so_android" id="cb-so-android" value="1"> Android</label>
          <label><input type="checkbox" name="so_ios" id="cb-so-ios" value="1"> iOS</label>
        </div>
      </fieldset>

      <fieldset>
        <legend>{{tr "projects.factory.section.compliance"}}</legend>
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
        <legend>{{tr "projects.factory.section.infra"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="ci" value="1" checked> CI/CD — {{tr "projects.factory.ci"}}</label>
          <label><input type="checkbox" name="kubernetes" id="cb-kubernetes" value="1"> Kubernetes — {{tr "projects.factory.kubernetes"}}</label>
          <label><input type="checkbox" name="terraform" value="1"> Terraform — {{tr "projects.factory.terraform"}}</label>
          <label><input type="checkbox" name="monitoring" value="1"> {{tr "projects.factory.monitoring"}}</label>
        </div>
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

  <script>
  (function(){
    // ── Dependencias automáticas entre checkboxes ────────────────────────
    function cb(id){ return document.getElementById(id); }

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
      var form = document.getElementById('factory-form');
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
      // reset preset selector back to blank so user can re-apply
      document.getElementById('factory-preset').value = '';
    };

    // ── Preview ───────────────────────────────────────────────────────────
    window.factoryPreview = function(btn){
      var form = document.getElementById('factory-form');
      var data = new FormData(form);
      var obj = {};
      data.forEach(function(v,k){ obj[k] = v; });
      // convert checkbox booleans
      var bools = ['frontend','api','auth','db','docker','i18n',
                   'plat_web','plat_desktop','plat_mobile','plat_cli','plat_embedded',
                   'so_linux','so_windows','so_macos','so_android','so_ios',
                   'compliance_rgpd','compliance_ens','compliance_lssi','compliance_wcag',
                   'compliance_factura_elec','compliance_reutilizacion',
                   'ci','kubernetes','terraform','monitoring'];
      var payload = {
        nombre: obj['nombre']||'',
        descripcion: obj['descripcion']||'',
        tipo: obj['tipo']||'',
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
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/decisiones">
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
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/documentacion">
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
        <form method="post" action="/proyectos/{{$.Proyecto.Slug}}/review-gates/{{.ID}}/resolver">
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
	Proyectos []webProyectoResumen
	Preview   []previewTask
	Msg       string
	Err       string
}

type previewTask struct {
	Key    string
	Titulo string
	Fase   string
	Modulo string
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
		Proyectos: resumen,
		Msg:       r.URL.Query().Get("ok"),
		Err:       r.URL.Query().Get("err"),
	})
}

func webHandlerNuevaAppPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/nueva-app?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	slug := strings.TrimSpace(r.FormValue("proyecto_slug"))
	if slug == "" {
		http.Redirect(w, r, "/nueva-app?err="+url.QueryEscape("selecciona un proyecto"), http.StatusSeeOther)
		return
	}

	actor := strings.TrimSpace(r.FormValue("por"))
	if actor == "" {
		actor = "web"
	}

	req := apiProyectoFabricarAppRequest{
		Nombre:      strings.TrimSpace(r.FormValue("nombre")),
		Descripcion: strings.TrimSpace(r.FormValue("descripcion")),
		Tipo:        strings.TrimSpace(r.FormValue("tipo")),
		Frontend:    webFormBool(r, "frontend"),
		API:         webFormBool(r, "api"),
		Auth:        webFormBool(r, "auth"),
		Database:    webFormBool(r, "db"),
		Docker:      webFormBool(r, "docker"),
		I18n:        webFormBool(r, "i18n"),
		Idiomas:     splitCSV(strings.TrimSpace(r.FormValue("idiomas"))),
		Por:         actor,

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
	}

	// Preview: muestra tareas sin persistir
	if r.URL.Query().Get("preview") == "1" {
		var previewResp apiProyectoFabricarAppPreviewResponse
		path := "/api/proyectos/" + url.PathEscape(slug) + "/fabricar-app/preview"
		if err := webInvocarAPIJSON(http.MethodPost, path, req, &previewResp); err != nil {
			http.Redirect(w, r, "/nueva-app?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
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
		webRender(w, r, webTplLayout+webTplNuevaApp, webNuevaAppData{Proyectos: resumen, Preview: tasks})
		return
	}

	if _, err := webFabricarAppProyectoPorAPI(slug, req); err != nil {
		http.Redirect(w, r, "/nueva-app?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/tareas?proyecto="+url.QueryEscape(slug)+"&ok="+url.QueryEscape(webTranslateRequestf(r, "projects.flash.factory_created")), http.StatusSeeOther)
}

const webTplNuevaApp = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "nueva_app.title"}}</h2>
  <p style="color:#64748b">{{tr "projects.factory.fixed_note"}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}

  <article>
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

    <form id="nueva-app-form" method="post" action="/nueva-app">
      <input type="hidden" name="proyecto_slug" id="na-slug-hidden">
      <label>{{tr "projects.factory.name"}} <input name="nombre" required></label>
      <label>{{tr "projects.factory.description"}} <textarea name="descripcion"></textarea></label>

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

      <fieldset>
        <legend>{{tr "projects.factory.section.components"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="frontend" value="1" checked> {{tr "projects.factory.frontend"}}</label>
          <label><input type="checkbox" name="api" value="1" checked> {{tr "projects.factory.api"}}</label>
          <label><input type="checkbox" name="auth" value="1"> {{tr "projects.factory.auth"}}</label>
          <label><input type="checkbox" name="db" value="1"> {{tr "projects.factory.db"}}</label>
          <label><input type="checkbox" name="docker" id="cb-docker" value="1" checked> {{tr "projects.factory.docker"}}</label>
          <label><input type="checkbox" name="i18n" value="1" checked> {{tr "projects.factory.i18n"}}</label>
        </div>
        <label style="margin-top:.5rem">{{tr "projects.factory.languages"}} <input name="idiomas" value="es,en"></label>
      </fieldset>

      <fieldset>
        <legend>{{tr "projects.factory.section.platforms"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="plat_web" id="cb-plat-web" value="1"> {{tr "projects.factory.plat_web"}}</label>
          <label><input type="checkbox" name="plat_desktop" id="cb-plat-desktop" value="1"> {{tr "projects.factory.plat_desktop"}}</label>
          <label><input type="checkbox" name="plat_mobile" id="cb-plat-mobile" value="1"> {{tr "projects.factory.plat_mobile"}}</label>
          <label><input type="checkbox" name="plat_cli" id="cb-plat-cli" value="1"> {{tr "projects.factory.plat_cli"}}</label>
          <label><input type="checkbox" name="plat_embedded" id="cb-plat-embedded" value="1"> {{tr "projects.factory.plat_embedded"}}</label>
        </div>
      </fieldset>

      <fieldset>
        <legend>{{tr "projects.factory.section.os"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="so_linux" id="cb-so-linux" value="1"> Linux</label>
          <label><input type="checkbox" name="so_windows" id="cb-so-windows" value="1"> Windows</label>
          <label><input type="checkbox" name="so_macos" id="cb-so-macos" value="1"> macOS</label>
          <label><input type="checkbox" name="so_android" id="cb-so-android" value="1"> Android</label>
          <label><input type="checkbox" name="so_ios" id="cb-so-ios" value="1"> iOS</label>
        </div>
      </fieldset>

      <fieldset>
        <legend>{{tr "projects.factory.section.compliance"}}</legend>
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
        <legend>{{tr "projects.factory.section.infra"}}</legend>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.5rem">
          <label><input type="checkbox" name="ci" value="1" checked> CI/CD — {{tr "projects.factory.ci"}}</label>
          <label><input type="checkbox" name="kubernetes" id="cb-kubernetes" value="1"> Kubernetes — {{tr "projects.factory.kubernetes"}}</label>
          <label><input type="checkbox" name="terraform" value="1"> Terraform — {{tr "projects.factory.terraform"}}</label>
          <label><input type="checkbox" name="monitoring" value="1"> {{tr "projects.factory.monitoring"}}</label>
        </div>
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
        <thead><tr><th>Key</th><th>{{tr "projects.factory.preview_col_title"}}</th><th>{{tr "projects.factory.preview_col_phase"}}</th><th>{{tr "projects.factory.preview_col_module"}}</th></tr></thead>
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
  var depRules = [
    {src:'cb-plat-mobile',   targets:['cb-so-android','cb-so-ios']},
    {src:'cb-plat-desktop',  targets:['cb-so-linux','cb-so-windows','cb-so-macos']},
    {src:'cb-plat-embedded', targets:['cb-so-linux']},
    {src:'cb-plat-cli',      targets:['cb-so-linux']},
    {src:'cb-kubernetes',    targets:['cb-docker']},
  ];
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
    var form=document.getElementById('nueva-app-form');
    form.querySelectorAll('input[type=checkbox]').forEach(function(el){ el.checked=false; });
    var t=document.getElementById('factory-tipo'); if(t) t.value=p.tipo;
    Object.keys(p.checks).forEach(function(n){ var el=form.querySelector('input[name="'+n+'"]'); if(el) el.checked=true; });
    document.getElementById('na-preset').value='';
  };
  window.naPreview=function(){
    var slug=document.querySelector('select[name=proyecto_slug]');
    if(!slug||!slug.value){ alert('Selecciona un proyecto'); return; }
    var form=document.getElementById('nueva-app-form');
    var data=new FormData(form);
    var obj={}; data.forEach(function(v,k){ obj[k]=v; });
    var bools=['frontend','api','auth','db','docker','i18n','plat_web','plat_desktop','plat_mobile','plat_cli','plat_embedded','so_linux','so_windows','so_macos','so_android','so_ios','compliance_rgpd','compliance_ens','compliance_lssi','compliance_wcag','compliance_factura_elec','compliance_reutilizacion','ci','kubernetes','terraform','monitoring'];
    var payload={nombre:obj['nombre']||'',descripcion:obj['descripcion']||'',tipo:obj['tipo']||'',idiomas:(obj['idiomas']||'es,en').split(','),por:obj['por']||'web'};
    bools.forEach(function(f){ payload[f]=obj[f]==='1'; });
    fetch('/api/proyectos/'+encodeURIComponent(slug.value)+'/fabricar-app/preview',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)})
    .then(function(r){ return r.json(); })
    .then(function(d){
      if(!d.ok||!d.tasks) return;
      var form2=document.createElement('form');
      form2.method='post'; form2.action='/nueva-app?preview=1';
      Object.keys(obj).forEach(function(k){ var i=document.createElement('input'); i.type='hidden'; i.name=k; i.value=obj[k]||''; form2.appendChild(i); });
      var si=document.createElement('input'); si.type='hidden'; si.name='proyecto_slug'; si.value=slug.value; form2.appendChild(si);
      document.body.appendChild(form2); form2.submit();
    });
  };
})();
</script>
{{end}}`
