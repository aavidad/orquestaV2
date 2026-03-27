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
	"orquesta/fabricaapp"
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
}

type webProyectosData struct {
	Proyectos []webProyectoResumen
	Msg       string
	Err       string
}

type webProyectoDetalleData struct {
	Proyecto    *db.Proyecto
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
		proyectos = append(proyectos, webProyectoResumen{
			Slug:    item.Slug,
			Nombre:  item.Nombre,
			Tipo:    string(item.Tipo),
			RutaAbs: item.RutaAbs,
			Activo:  item.Activo,
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
	if errReviewGates != nil {
		errParts = append(errParts, errReviewGates.Error())
	}
	webRender(w, r, webTplLayout+webTplProyectoDetalle, webProyectoDetalleData{
		Proyecto:    overview.Proyecto,
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
	spec := fabricaapp.AppSpec{
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
	}
	actor := strings.TrimSpace(r.FormValue("por"))
	if actor == "" {
		actor = "web"
	}
	if _, err := webFabricarAppProyectoPorAPI(slug, apiProyectoFabricarAppRequest{
		Nombre:      spec.Nombre,
		Descripcion: spec.Descripcion,
		Tipo:        spec.Tipo,
		Frontend:    spec.Frontend,
		API:         spec.API,
		Auth:        spec.Auth,
		Database:    spec.Database,
		Docker:      spec.Docker,
		I18n:        spec.I18n,
		Idiomas:     spec.Idiomas,
		Por:         actor,
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
    <thead><tr><th>{{tr "Proyecto"}}</th><th>{{tr "Tipo"}}</th><th>{{tr "projects.path"}}</th><th>{{tr "Estado"}}</th></tr></thead>
    <tbody>
      {{range .Proyectos}}
      <tr>
        <td><a href="/proyectos/{{.Slug}}"><strong>{{.Nombre}}</strong></a><br><small>{{.Slug}}</small></td>
        <td>{{.Tipo}}</td>
        <td><code>{{.RutaAbs}}</code></td>
        <td>{{if .Activo}}{{tr "projects.active"}}{{else}}{{tr "projects.inactive"}}{{end}}</td>
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

  <article>
    <h3>{{tr "projects.factory.title"}}</h3>
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/fabricar-app">
      <label>{{tr "projects.factory.type"}}
        <select name="tipo" required>
          <option value="web_api">web_api</option>
          <option value="web">web</option>
          <option value="api">api</option>
          <option value="cli">cli</option>
        </select>
      </label>
      <label>{{tr "projects.factory.name"}} <input name="nombre" value="{{.Proyecto.Nombre}}" required></label>
      <label>{{tr "projects.factory.description"}} <textarea name="descripcion"></textarea></label>
      <div style="display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem">
        <label><input type="checkbox" name="frontend" value="1" checked> {{tr "projects.factory.frontend"}}</label>
        <label><input type="checkbox" name="api" value="1" checked> {{tr "projects.factory.api"}}</label>
        <label><input type="checkbox" name="auth" value="1"> {{tr "projects.factory.auth"}}</label>
        <label><input type="checkbox" name="db" value="1"> {{tr "projects.factory.db"}}</label>
        <label><input type="checkbox" name="docker" value="1" checked> {{tr "projects.factory.docker"}}</label>
        <label><input type="checkbox" name="i18n" value="1" checked> {{tr "projects.factory.i18n"}}</label>
      </div>
      <label>{{tr "projects.factory.languages"}} <input name="idiomas" value="es,en"></label>
      <label>{{tr "projects.factory.actor"}} <input name="por" value="web"></label>
      <button type="submit">{{tr "projects.factory.submit"}}</button>
    </form>
  </article>

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
