package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"orquesta/db"
	"orquesta/progresoapp"
)

type webProgresoData struct {
	Proyecto  string
	Proyectos []*db.Proyecto
	Resumen   *progresoapp.ResumenProgresoProyecto
	Fases     []*progresoapp.FaseProyecto
	Msg       string
	Err       string
}

func webRouterProgreso(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/progreso", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "fases" && r.Method == http.MethodPost:
		webHandlerProgresoFaseRegistrar(w, r)
	case len(parts) == 4 && parts[1] == "fases" && parts[3] == "actualizar" && r.Method == http.MethodPost:
		webHandlerProgresoFaseActualizar(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "tareas" && parts[3] == "registrar" && r.Method == http.MethodPost:
		webHandlerProgresoTareaRegistrar(w, r, parts[2])
	default:
		http.NotFound(w, r)
	}
}

func webHandlerProgreso(w http.ResponseWriter, r *http.Request) {
	proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	proyectos, err := webCargarProyectosPorAPI()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := webProgresoData{
		Proyecto:  proyecto,
		Proyectos: proyectos,
		Msg:       r.URL.Query().Get("ok"),
		Err:       r.URL.Query().Get("err"),
	}
	if proyecto != "" {
		resumen, fases, err := webCargarProgresoPorAPI(proyecto)
		if err != nil {
			data.Err = err.Error()
		} else {
			data.Resumen = resumen
			data.Fases = fases
		}
	}
	webRender(w, r, webTplLayout+webTplProgreso, data)
}

func webHandlerProgresoFaseRegistrar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	proyecto := strings.TrimSpace(r.FormValue("proyecto"))
	req := apiProgresoFaseRegistrarRequest{
		Proyecto:    proyecto,
		Nombre:      strings.TrimSpace(r.FormValue("nombre")),
		Descripcion: strings.TrimSpace(r.FormValue("descripcion")),
		Orden:       webParseInt64Default(r.FormValue("orden"), 100),
		Peso:        webParseFloat64Default(r.FormValue("peso"), 1),
		Estado:      strings.TrimSpace(r.FormValue("estado")),
	}
	if req.Estado == "" {
		req.Estado = "pendiente"
	}
	if err := webInvocarAPIJSON(http.MethodPost, "/api/progreso/fases", req, &apiProgresoFaseResponse{}); err != nil {
		webRedirectProgreso(w, r, proyecto, "", err)
		return
	}
	webRedirectProgreso(w, r, proyecto, webTranslateRequestf(r, "progress.flash.phase_saved", req.Nombre), nil)
}

func webHandlerProgresoFaseActualizar(w http.ResponseWriter, r *http.Request, idRaw string) {
	_ = r.ParseForm()
	id, err := strconv.ParseInt(strings.TrimSpace(idRaw), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	proyecto := strings.TrimSpace(r.FormValue("proyecto"))
	nombre := strings.TrimSpace(r.FormValue("nombre"))
	descripcion := strings.TrimSpace(r.FormValue("descripcion"))
	estado := strings.TrimSpace(r.FormValue("estado"))
	orden := webParseInt64Default(r.FormValue("orden"), 0)
	peso := webParseFloat64Default(r.FormValue("peso"), 0)
	req := apiProgresoFaseActualizarRequest{
		Proyecto:    &proyecto,
		Nombre:      &nombre,
		Descripcion: &descripcion,
		Orden:       &orden,
		Peso:        &peso,
		Estado:      &estado,
	}
	if err := webInvocarAPIJSON(http.MethodPost, "/api/progreso/fases/"+strconv.FormatInt(id, 10), req, &apiProgresoFaseResponse{}); err != nil {
		webRedirectProgreso(w, r, proyecto, "", err)
		return
	}
	webRedirectProgreso(w, r, proyecto, webTranslateRequestf(r, "progress.flash.phase_updated", nombre), nil)
}

func webHandlerProgresoTareaRegistrar(w http.ResponseWriter, r *http.Request, idRaw string) {
	_ = r.ParseForm()
	tareaID, err := strconv.ParseInt(strings.TrimSpace(idRaw), 10, 64)
	if err != nil || tareaID <= 0 {
		http.NotFound(w, r)
		return
	}
	proyecto := strings.TrimSpace(r.FormValue("proyecto"))
	pct := webParseFloat64Default(r.FormValue("pct"), 0)
	actualizadoPor := strings.TrimSpace(r.FormValue("por"))
	faseID, err := webParseOptionalInt64(strings.TrimSpace(r.FormValue("fase")))
	if err != nil {
		webRedirectProgreso(w, r, proyecto, "", err)
		return
	}
	req := apiProgresoTareaRegistrarRequest{
		Proyecto:       proyecto,
		FaseID:         faseID,
		ProgresoPct:    pct,
		ActualizadoPor: actualizadoPor,
	}
	if err := webInvocarAPIJSON(http.MethodPost, "/api/progreso/tareas/"+strconv.FormatInt(tareaID, 10), req, &map[string]any{}); err != nil {
		webRedirectProgreso(w, r, proyecto, "", err)
		return
	}
	webRedirectProgreso(w, r, proyecto, webTranslateRequestf(r, "progress.flash.task_saved", strconv.FormatInt(tareaID, 10)), nil)
}

func webRedirectProgreso(w http.ResponseWriter, r *http.Request, proyecto, okMsg string, err error) {
	query := url.Values{}
	if proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	if okMsg != "" {
		query.Set("ok", okMsg)
	}
	if err != nil {
		query.Set("err", err.Error())
	}
	http.Redirect(w, r, "/progreso?"+query.Encode(), http.StatusSeeOther)
}

func webCargarProyectosPorAPI(activa ...*bool) ([]*db.Proyecto, error) {
	path := "/api/proyectos"
	if len(activa) > 0 && activa[0] != nil {
		path += "?activa=" + strconv.FormatBool(*activa[0])
	}
	var resp apiProyectosResponse
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Proyectos, nil
}

func webCargarProgresoPorAPI(proyecto string) (*progresoapp.ResumenProgresoProyecto, []*progresoapp.FaseProyecto, error) {
	var resumenResp apiProgresoResumenResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/progreso?proyecto="+url.QueryEscape(proyecto), nil, &resumenResp); err != nil {
		return nil, nil, err
	}
	var fasesResp apiProgresoFasesResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/progreso/fases?proyecto="+url.QueryEscape(proyecto), nil, &fasesResp); err != nil {
		return nil, nil, err
	}
	return resumenResp.Resumen, fasesResp.Fases, nil
}

func webParseInt64Default(raw string, defaultValue int64) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return defaultValue
	}
	return v
}

func webParseFloat64Default(raw string, defaultValue float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return defaultValue
	}
	return v
}

func webParseOptionalInt64(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return nil, fmt.Errorf("fase inválida")
	}
	return &v, nil
}

const webTplProgreso = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "progress.title"}}</h2>
  <p style="color:#64748b">{{tr "progress.subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <form method="get" action="/progreso" style="display:flex;gap:.7rem;flex-wrap:wrap;align-items:end">
    <label>{{tr "projects.project"}}
      <select name="proyecto">
        <option value="">{{tr "progress.select_project"}}</option>
        {{range .Proyectos}}
        <option value="{{.Slug}}"{{if eqStr $.Proyecto .Slug}} selected{{end}}>{{.Slug}}</option>
        {{end}}
      </select>
    </label>
    <button type="submit">{{tr "common.view"}}</button>
  </form>
</section>

{{if .Resumen}}
<section class="container">
  <h3 style="margin:0 0 .5rem 0">{{tr "progress.summary_title"}}</h3>
  <div class="stats">
    <div class="stat"><div class="n">{{printf "%.1f%%" .Resumen.ProgresoPct}}</div><div class="l">{{tr "progress.progress_pct"}}</div></div>
    <div class="stat"><div class="n">{{.Resumen.TareasTotales}}</div><div class="l">{{tr "progress.tasks_total"}}</div></div>
    <div class="stat"><div class="n">{{.Resumen.TareasCompletadas}}</div><div class="l">{{tr "progress.tasks_done"}}</div></div>
  </div>
</section>

<section class="container">
  <h3 style="margin:0 0 .5rem 0">{{tr "progress.new_phase_title"}}</h3>
  <form method="post" action="/progreso/fases" style="display:grid;gap:.7rem">
    <input type="hidden" name="proyecto" value="{{.Proyecto}}">
    <div class="grid2">
      <label>{{tr "common.name"}} <input name="nombre" required></label>
      <label>{{tr "common.status"}} <input name="estado" value="pendiente"></label>
    </div>
    <label>{{tr "common.description"}} <textarea name="descripcion" rows="3"></textarea></label>
    <div class="grid2">
      <label>{{tr "progress.order"}} <input name="orden" type="number" value="100"></label>
      <label>{{tr "progress.weight"}} <input name="peso" type="number" step="0.1" value="1"></label>
    </div>
    <button type="submit">{{tr "common.save"}}</button>
  </form>
</section>

{{range .Resumen.Fases}}
<section class="container">
  {{$faseID := .Fase.ID}}
  <h3 style="margin:0">{{.Fase.Nombre}}</h3>
  <p style="color:#64748b">{{printf "%.1f%%" .ProgresoPct}} · {{.TareasCompletadas}}/{{.TareasTotales}} {{tr "progress.tasks_label"}}</p>
  <form method="post" action="/progreso/fases/{{.Fase.ID}}/actualizar" style="display:grid;gap:.7rem;margin-bottom:1rem">
    <input type="hidden" name="proyecto" value="{{$.Proyecto}}">
    <div class="grid2">
      <label>{{tr "common.name"}} <input name="nombre" value="{{.Fase.Nombre}}"></label>
      <label>{{tr "common.status"}} <input name="estado" value="{{.Fase.Estado}}"></label>
    </div>
    <label>{{tr "common.description"}} <textarea name="descripcion" rows="2">{{.Fase.Descripcion}}</textarea></label>
    <div class="grid2">
      <label>{{tr "progress.order"}} <input name="orden" type="number" value="{{.Fase.Orden}}"></label>
      <label>{{tr "progress.weight"}} <input name="peso" type="number" step="0.1" value="{{printf "%.1f" .Fase.Peso}}"></label>
    </div>
    <button type="submit">{{tr "progress.update_phase"}}</button>
  </form>
  <table>
    <thead><tr><th>ID</th><th>{{tr "Tareas"}}</th><th>{{tr "progress.progress_pct"}}</th><th>{{tr "progress.updated_by"}}</th><th>{{tr "common.save"}}</th></tr></thead>
    <tbody>
      {{range .Tareas}}
      <tr>
        <td>#{{.TareaID}}</td>
        <td>{{.TareaTitulo}}</td>
        <td>
          <form method="post" action="/progreso/tareas/{{.TareaID}}/registrar" style="display:flex;gap:.5rem;align-items:center;flex-wrap:wrap">
            <input type="hidden" name="proyecto" value="{{$.Proyecto}}">
            <input type="hidden" name="fase" value="{{$faseID}}">
            <input name="pct" type="number" min="0" max="100" step="0.1" value="{{printf "%.1f" .ProgresoPct}}" style="width:7rem">
        </td>
        <td><input name="por" style="width:10rem"></td>
        <td><button type="submit">{{tr "common.save"}}</button></form></td>
      </tr>
      {{else}}
      <tr><td colspan="5">{{tr "progress.no_tasks"}}</td></tr>
      {{end}}
    </tbody>
  </table>
</section>
{{end}}

{{if .Resumen.TareasSinFase}}
<section class="container">
  <h3 style="margin:0 0 .5rem 0">{{tr "progress.tasks_without_phase"}}</h3>
  <table>
    <thead><tr><th>ID</th><th>{{tr "Tareas"}}</th><th>{{tr "progress.phase"}}</th><th>{{tr "progress.progress_pct"}}</th><th>{{tr "progress.updated_by"}}</th><th>{{tr "common.save"}}</th></tr></thead>
    <tbody>
      {{range .Resumen.TareasSinFase}}
      <tr>
        <td>#{{.TareaID}}</td>
        <td>{{.TareaTitulo}}</td>
        <td>
          <form method="post" action="/progreso/tareas/{{.TareaID}}/registrar" style="display:flex;gap:.5rem;align-items:center;flex-wrap:wrap">
            <input type="hidden" name="proyecto" value="{{$.Proyecto}}">
            <select name="fase">
              <option value="">{{tr "progress.no_phase"}}</option>
              {{range $.Fases}}
              <option value="{{.ID}}">{{.Nombre}}</option>
              {{end}}
            </select>
        </td>
        <td><input name="pct" type="number" min="0" max="100" step="0.1" value="{{printf "%.1f" .ProgresoPct}}" style="width:7rem"></td>
        <td><input name="por" style="width:10rem"></td>
        <td><button type="submit">{{tr "common.save"}}</button></form></td>
      </tr>
      {{end}}
    </tbody>
  </table>
</section>
{{end}}
{{end}}
{{end}}`
