/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"orquesta/db"
	"orquesta/operacionesapp"
)

var operacionesService = operacionesapp.NewService(operacionesapp.Repository{})

type webAsignacionesData struct {
	Asignaciones []*db.Asignacion
	Estado       string
	Agente       string
}

type webSesionesData struct {
	Sesiones []*db.Sesion
	Agente   string
	Proyecto string
	Estado   string
	Activa   string
}

type webSesionDetalleData struct {
	Sesion *db.Sesion
}

func webHandlerAgentes(w http.ResponseWriter, r *http.Request) {
	webHandlerAgentesPanel(w, r)
}

func webRouterSesiones(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 || r.Method != http.MethodGet {
		http.Redirect(w, r, "/sesiones", http.StatusSeeOther)
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	webHandlerSesionDetalle(w, r, id)
}

func webHandlerAsignaciones(w http.ResponseWriter, r *http.Request) {
	estado := strings.TrimSpace(r.URL.Query().Get("estado"))
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	items, err := webCargarAsignacionesOpsPorAPI(estado, agente)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	webRender(w, r, webTplLayout+webTplAsignaciones, webAsignacionesData{
		Asignaciones: items,
		Estado:       estado,
		Agente:       agente,
	})
}

func webHandlerSesiones(w http.ResponseWriter, r *http.Request) {
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	estado := strings.TrimSpace(r.URL.Query().Get("estado"))
	activaRaw := strings.TrimSpace(r.URL.Query().Get("activa"))

	var activa *bool
	if activaRaw != "" {
		value, err := parseBoolFiltro(activaRaw)
		if err != nil {
			http.Error(w, webTranslateRequestf(r, "ops.flash.invalid_boolean", activaRaw), http.StatusBadRequest)
			return
		}
		activa = &value
	}

	items, err := webCargarSesionesInspeccionPorAPI(operacionesapp.ListInspectionSessionsInput{
		Agente:      agente,
		ProyectoRef: proyectoRef,
		Estado:      estado,
		Activa:      activa,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	webRender(w, r, webTplLayout+webTplSesiones, webSesionesData{
		Sesiones: items,
		Agente:   agente,
		Proyecto: proyectoRef,
		Estado:   estado,
		Activa:   activaRaw,
	})
}

func webHandlerSesionDetalle(w http.ResponseWriter, r *http.Request, id int64) {
	sesion, err := webCargarSesionInspeccionPorAPI(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, r, webTplLayout+webTplSesionDetalle, webSesionDetalleData{Sesion: sesion})
}

func webCargarAsignacionesOpsPorAPI(estado, agente string) ([]*db.Asignacion, error) {
	query := url.Values{}
	if estado = strings.TrimSpace(estado); estado != "" {
		query.Set("estado", estado)
	}
	if agente = strings.TrimSpace(agente); agente != "" {
		query.Set("agente", agente)
	}
	path := "/api/asignaciones"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp apiAsignacionesResponse
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Asignaciones, nil
}

func webCargarSesionesInspeccionPorAPI(input operacionesapp.ListInspectionSessionsInput) ([]*db.Sesion, error) {
	query := url.Values{}
	if agente := strings.TrimSpace(input.Agente); agente != "" {
		query.Set("agente", agente)
	}
	if proyecto := strings.TrimSpace(input.ProyectoRef); proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	if estado := strings.TrimSpace(input.Estado); estado != "" {
		query.Set("estado", estado)
	}
	if input.Activa != nil {
		query.Set("activa", strconv.FormatBool(*input.Activa))
	}
	path := "/api/sesiones"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp apiSesionesInspeccionResponse
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Sesiones, nil
}

func webCargarSesionInspeccionPorAPI(id int64) (*db.Sesion, error) {
	var resp apiSesionResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/sesiones/"+strconv.FormatInt(id, 10), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Sesion, nil
}

func parseBoolFiltro(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "si", "sí":
		return true, nil
	case "0", "false", "no":
		return false, nil
	default:
		return false, fmt.Errorf("valor booleano inválido: %s", raw)
	}
}

const webTplAsignaciones = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "ops.assignments.title"}}</h2>
  <form method="get" action="/asignaciones">
    <div class="grid2">
      <label>{{tr "Estado"}} <input name="estado" value="{{.Estado}}"></label>
      <label>{{tr "Agente"}} <input name="agente" value="{{.Agente}}"></label>
    </div>
    <button type="submit" class="btn-sm">{{tr "common.filter"}}</button>
  </form>
  <table>
    <thead><tr><th>ID</th><th>{{tr "Agente"}}</th><th>{{tr "Proyecto"}}</th><th>{{tr "Estado"}}</th><th>{{tr "common.note"}}</th></tr></thead>
    <tbody>
      {{range .Asignaciones}}
      <tr>
        <td>{{.ID}}</td>
        <td>{{.Agente}}</td>
        <td>{{.ProyectoSlug}}</td>
        <td>{{.Estado}}</td>
        <td>{{.Nota}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</section>
{{end}}`

const webTplSesiones = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "ops.sessions.title"}}</h2>
  <form method="get" action="/sesiones">
    <div class="grid2">
      <label>{{tr "Agente"}} <input name="agente" value="{{.Agente}}"></label>
      <label>{{tr "Proyecto"}} <input name="proyecto" value="{{.Proyecto}}"></label>
    </div>
    <div class="grid2">
      <label>{{tr "Estado"}} <input name="estado" value="{{.Estado}}"></label>
      <label>{{tr "common.active"}} <input name="activa" value="{{.Activa}}" placeholder="{{tr "common.boolean_true"}}"></label>
    </div>
    <button type="submit" class="btn-sm">{{tr "common.filter"}}</button>
  </form>
  <table>
    <thead><tr><th>ID</th><th>{{tr "Agente"}}</th><th>{{tr "Proyecto"}}</th><th>{{tr "Estado"}}</th><th>{{tr "common.active"}}</th><th>{{tr "agentes.tool"}}</th><th>{{tr "ops.sessions.external_session"}}</th><th>{{tr "common.host"}}</th><th>{{tr "common.branch"}}</th><th>{{tr "common.started_at"}}</th></tr></thead>
    <tbody>
      {{range .Sesiones}}
      <tr>
        <td><a href="/sesiones/{{.ID}}">{{.ID}}</a></td>
        <td>{{.Agente}}</td>
        <td>{{.ProyectoSlug}}</td>
        <td>{{.Estado}}</td>
        <td>{{if .Activa}}{{tr "common.yes"}}{{else}}{{tr "common.no"}}{{end}}</td>
        <td>{{.Herramienta}}</td>
        <td><code>{{orDash .ExternalSessionID}}</code></td>
        <td>{{.Host}}</td>
        <td>{{.Branch}}</td>
        <td>{{.Inicio.Format "2006-01-02 15:04:05"}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</section>
{{end}}`

const webTplSesionDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/sesiones">← {{tr "ops.sessions.back"}}</a></p>
  <h2 style="margin:0">{{tr "ops.sessions.detail_title"}} #{{.Sesion.ID}}</h2>
  <p style="color:#64748b">{{tr "ops.sessions.detail_subtitle"}}</p>
  <table>
    <tbody>
      <tr><th>{{tr "Agente"}}</th><td>{{.Sesion.Agente}}</td></tr>
      <tr><th>{{tr "Proyecto"}}</th><td>{{.Sesion.ProyectoSlug}}</td></tr>
      <tr><th>{{tr "Estado"}}</th><td>{{.Sesion.Estado}}</td></tr>
      <tr><th>{{tr "common.active"}}</th><td>{{if .Sesion.Activa}}{{tr "common.yes"}}{{else}}{{tr "common.no"}}{{end}}</td></tr>
      <tr><th>{{tr "agentes.tool"}}</th><td>{{.Sesion.Herramienta}}</td></tr>
      <tr><th>{{tr "common.connector"}}</th><td>{{.Sesion.ConectorSlug}}</td></tr>
      <tr><th>{{tr "common.host"}}</th><td>{{.Sesion.Host}}</td></tr>
      <tr><th>{{tr "common.pid"}}</th><td>{{if .Sesion.PID}}{{.Sesion.PID}}{{end}}</td></tr>
      <tr><th>{{tr "common.branch"}}</th><td><code>{{.Sesion.Branch}}</code></td></tr>
      <tr><th>{{tr "common.cwd"}}</th><td><code>{{.Sesion.CWD}}</code></td></tr>
      <tr><th>{{tr "ops.sessions.external_session"}}</th><td><code>{{.Sesion.ExternalSessionID}}</code></td></tr>
      <tr><th>{{tr "common.continuity"}}</th><td>{{.Sesion.ResumenContinuidad}}</td></tr>
      <tr><th>{{tr "common.started_at"}}</th><td>{{.Sesion.Inicio.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>{{tr "common.heartbeat"}}</th><td>{{if .Sesion.HeartbeatAt}}{{.Sesion.HeartbeatAt.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
      <tr><th>{{tr "common.finished_at"}}</th><td>{{if .Sesion.Fin}}{{.Sesion.Fin.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
      <tr><th>{{tr "ops.sessions.resume_payload"}}</th><td><pre style="white-space:pre-wrap;margin:0">{{.Sesion.ResumePayloadJSON}}</pre></td></tr>
    </tbody>
  </table>
</section>
{{end}}`
