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
	"strings"

	"orquesta/agentesapp"
	"orquesta/db"
)

type webAgenteRow = agentesapp.Row

type webAgentesPanelData struct {
	Rows        []webAgenteRow
	Total       int
	Activos     int
	Habilitados int
	AutoRefresh bool
	Msg         string
	Err         string
}

type webAgenteDetalleData struct {
	Row          webAgenteRow
	Proyectos    []*db.Proyecto
	Asignaciones []*db.Asignacion
	Sesiones     []*db.Sesion
	Runtimes     []*db.RuntimeInstance
	Handles      []*db.RuntimeHandle
	Transcript   []*db.RuntimeTranscriptEntry
	Orders       []*db.RuntimeOrder
	Mailbox      []*db.RuntimeMailboxMessage
	Checkpoints  []*db.RuntimeCheckpoint
	Msg          string
	Err          string
}

func webRouterAgentes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/agentes", http.StatusSeeOther)
		return
	}

	switch {
	case parts[1] == "nuevo" && r.Method == http.MethodPost:
		webHandlerAgenteNuevo(w, r)
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerAgenteDetalle(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "asignacion" && r.Method == http.MethodPost:
		webHandlerAgenteAsignacion(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "control" && r.Method == http.MethodPost:
		webHandlerAgenteControl(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "estado" && r.Method == http.MethodPost:
		webHandlerAgenteEstado(w, r, parts[1])
	default:
		http.NotFound(w, r)
	}
}

func webHandlerAgentesPanel(w http.ResponseWriter, r *http.Request) {
	rows, err := construirWebAgenteRows()
	if err != nil {
		webRender(w, r, webTplLayout+webTplAgentesPanel, webAgentesPanelData{
			Err: err.Error(),
		})
		return
	}
	data := webAgentesPanelData{
		Rows:        rows,
		Total:       len(rows),
		AutoRefresh: strings.TrimSpace(r.URL.Query().Get("auto")) != "off",
		Msg:         r.URL.Query().Get("ok"),
		Err:         r.URL.Query().Get("err"),
	}
	for _, row := range rows {
		if row.Agente != nil && row.Agente.Activo {
			data.Activos++
		}
		if row.Agente != nil && row.Agente.Habilitado {
			data.Habilitados++
		}
	}
	webRender(w, r, webTplLayout+webTplAgentesPanel, data)
}

func webHandlerAgenteNuevo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/agentes?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	nombre := strings.TrimSpace(r.FormValue("nombre"))
	rol := strings.TrimSpace(r.FormValue("rol"))
	if rol == "" {
		rol = "programador"
	}
	creado, err := webCrearAgentePorAPI(apiAgenteRequest{
		Nombre:    nombre,
		Proveedor: strings.TrimSpace(r.FormValue("proveedor")),
		Rol:       rol,
	})
	if err != nil {
		http.Redirect(w, r, "/agentes?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	nombre = creado
	http.Redirect(w, r, "/agentes?ok="+url.QueryEscape(webTranslateRequestf(r, "agentes.flash.created", nombre)), http.StatusSeeOther)
}

func webHandlerAgenteAsignacion(w http.ResponseWriter, r *http.Request, nombre string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	proyectoRef := strings.TrimSpace(r.FormValue("proyecto"))
	if proyectoRef == "" {
		http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?err="+url.QueryEscape(webTranslateRequestf(r, "agentes.flash.project_required")), http.StatusSeeOther)
		return
	}
	proyecto, err := webCargarProyectoPorAPI(proyectoRef)
	if err != nil {
		http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	if err := webActivarAsignacionPorAPI(apiAsignacionActivarRequest{
		Agente:   strings.TrimSpace(nombre),
		Proyecto: proyectoRef,
		Nota:     strings.TrimSpace(r.FormValue("nota")),
	}); err != nil {
		http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?ok="+url.QueryEscape(webTranslateRequestf(r, "agentes.flash.assignment_saved", nombre, proyecto.Slug)), http.StatusSeeOther)
}

func webHandlerAgenteDetalle(w http.ResponseWriter, r *http.Request, nombre string) {
	data, err := construirWebAgenteDetalleData(strings.TrimSpace(nombre))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data.Msg = r.URL.Query().Get("ok")
	data.Err = r.URL.Query().Get("err")
	webRender(w, r, webTplLayout+webTplAgenteDetalle, data)
}

func webHandlerAgenteControl(w http.ResponseWriter, r *http.Request, nombre string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	req := apiAgenteControlRequest{
		Agente:       strings.TrimSpace(nombre),
		Proyecto:     strings.TrimSpace(r.FormValue("proyecto")),
		Accion:       strings.TrimSpace(r.FormValue("accion")),
		Conector:     strings.TrimSpace(r.FormValue("conector")),
		Modelo:       strings.TrimSpace(r.FormValue("modelo")),
		Razonamiento: strings.TrimSpace(r.FormValue("razonamiento")),
		Perfil:       strings.TrimSpace(r.FormValue("perfil")),
		Motivo:       strings.TrimSpace(r.FormValue("motivo")),
		Por:          "web",
	}
	orderID, accion, err := webEncolarControlAgentePorAPI(req)
	if err != nil {
		http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	msg := fmt.Sprintf("Orden %s #%d encolada para %s", accion, orderID, nombre)
	http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?ok="+url.QueryEscape(msg), http.StatusSeeOther)
}

func webHandlerAgenteEstado(w http.ResponseWriter, r *http.Request, nombre string) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	accion := strings.TrimSpace(r.FormValue("accion"))
	err := webAplicarAccionEstadoAgentePorAPI(nombre, accion)
	if err != nil {
		http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?ok="+url.QueryEscape("Acción '"+accion+"' aplicada"), http.StatusSeeOther)
}

func construirWebAgenteRows() ([]webAgenteRow, error) {
	var resp apiAgentesPanelResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/agentes?vista=panel", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Rows, nil
}

func construirWebAgenteDetalleData(nombre string) (*webAgenteDetalleData, error) {
	var resp apiAgenteOverviewResponse
	path := "/api/agentes/" + url.PathEscape(strings.TrimSpace(nombre)) + "/overview"
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	proyectos, err := webCargarProyectosActivosPorAPI()
	if err != nil {
		return nil, err
	}
	detail := resp.Detail
	if detail == nil {
		return nil, fmt.Errorf("detalle de agente vacío")
	}
	return &webAgenteDetalleData{
		Row:          detail.Row,
		Proyectos:    proyectos,
		Asignaciones: detail.Asignaciones,
		Sesiones:     limitarSesiones(detail.Sesiones, 12),
		Runtimes:     limitarRuntimes(detail.Runtimes, 12),
		Handles:      limitarHandles(detail.Handles, 12),
		Transcript:   limitarTranscript(detail.Transcript, 40),
		Orders:       limitarOrders(detail.Orders, 20),
		Mailbox:      limitarMailbox(detail.Mailbox, 20),
		Checkpoints:  detail.Checkpoints,
	}, nil
}

type apiAgenteCrearResponse struct {
	OK     bool   `json:"ok"`
	Nombre string `json:"nombre"`
	Rol    string `json:"rol"`
}

func webCrearAgentePorAPI(req apiAgenteRequest) (string, error) {
	var resp apiAgenteCrearResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/agentes", req, &resp); err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Nombre), nil
}

func webCargarProyectoPorAPI(ref string) (*db.Proyecto, error) {
	var resp apiProyectoResponse
	path := "/api/proyectos/" + url.PathEscape(strings.TrimSpace(ref))
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Proyecto, nil
}

func webCargarProyectosActivosPorAPI() ([]*db.Proyecto, error) {
	var resp apiProyectosResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/proyectos", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]*db.Proyecto, 0, len(resp.Proyectos))
	for _, proyecto := range resp.Proyectos {
		if proyecto != nil && proyecto.Activo {
			out = append(out, proyecto)
		}
	}
	return out, nil
}

func webActivarAsignacionPorAPI(req apiAsignacionActivarRequest) error {
	return webInvocarAPIJSON(http.MethodPost, "/api/asignaciones/activar", req, nil)
}

func webEncolarControlAgentePorAPI(req apiAgenteControlRequest) (int64, string, error) {
	var resp apiAgenteControlResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/agente/control", req, &resp); err != nil {
		return 0, "", err
	}
	return resp.ID, strings.TrimSpace(resp.Accion), nil
}

func webAplicarAccionEstadoAgentePorAPI(nombre, accion string) error {
	accion = strings.TrimSpace(accion)
	nombre = strings.TrimSpace(nombre)
	if nombre == "" || accion == "" {
		return fmt.Errorf("agente y acción obligatorios")
	}
	path := "/api/agentes/" + url.PathEscape(nombre) + "/" + url.PathEscape(accion)
	return webInvocarAPIJSON(http.MethodPost, path, map[string]any{}, nil)
}

func limitarSesiones(items []*db.Sesion, max int) []*db.Sesion {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func limitarRuntimes(items []*db.RuntimeInstance, max int) []*db.RuntimeInstance {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func limitarHandles(items []*db.RuntimeHandle, max int) []*db.RuntimeHandle {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func limitarOrders(items []*db.RuntimeOrder, max int) []*db.RuntimeOrder {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func limitarMailbox(items []*db.RuntimeMailboxMessage, max int) []*db.RuntimeMailboxMessage {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func limitarTranscript(items []*db.RuntimeTranscriptEntry, max int) []*db.RuntimeTranscriptEntry {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

const webTplAgentesPanel = `{{define "content"}}
<section>
  <h2>{{tr "Agentes"}} · {{tr "agentes.live.title"}}</h2>
  <p style="color:#64748b">{{tr "agentes.live.subtitle"}}</p>
  {{if .Msg}}<div class="alert-ok">{{.Msg}}</div>{{end}}
  {{if .Err}}<div class="alert-err">{{.Err}}</div>{{end}}
  <div class="stats">
    <div class="stat"><div class="n">{{.Total}}</div><div class="l">{{tr "Agentes"}}</div></div>
    <div class="stat"><div class="n">{{.Habilitados}}</div><div class="l">{{tr "agentes.enabled"}}</div></div>
    <div class="stat"><div class="n">{{.Activos}}</div><div class="l">{{tr "agentes.active_now"}}</div></div>
  </div>
  <div style="display:flex;justify-content:space-between;align-items:center;gap:1rem;flex-wrap:wrap;margin-bottom:1rem">
    <div style="font-size:.85rem;color:#64748b">
      {{if .AutoRefresh}}
        {{tr "agentes.auto_refresh"}}
        · <a href="/agentes?auto=off">{{tr "agentes.auto_refresh_off"}}</a>
      {{else}}
        <a href="/agentes">{{tr "agentes.auto_refresh_on"}}</a>
      {{end}}
    </div>
    <details class="form-panel" style="margin:0;min-width:min(100%,30rem)">
      <summary>{{tr "agentes.create"}}</summary>
      <form method="POST" action="/agentes/nuevo" style="display:grid;grid-template-columns:1.1fr 1.6fr 1fr auto;gap:.6rem;align-items:end;margin-top:.8rem">
        <div><label>{{tr "agentes.create.provider"}}</label><select name="proveedor"><option value="codex">Codex</option><option value="claude">Claude</option><option value="gemini">Gemini</option></select></div>
        <div><label>{{tr "agentes.create.name_optional"}}</label><input type="text" name="nombre" placeholder="{{tr "agentes.create.name_auto"}}"></div>
        <div><label>{{tr "Rol"}}</label><input type="text" name="rol" value="programador" required></div>
        <div><button type="submit" class="btn-sm">{{tr "agentes.create_button"}}</button></div>
      </form>
    </details>
  </div>
  {{if .Rows}}
  <div style="overflow:auto">
    <table id="tabla-agentes">
      <thead>
        <tr>
          <th>{{tr "Agente"}}</th>
          <th>{{tr "Rol"}}</th>
          <th>{{tr "Estado"}}</th>
          <th>{{tr "agentes.assignment"}}</th>
          <th>{{tr "Sesión"}}</th>
          <th>{{tr "Runtime"}}</th>
          <th>{{tr "Handle"}}</th>
          <th>{{tr "agentes.controlplane"}}</th>
          <th>{{tr "agentes.last_checkpoint"}}</th>
          <th>{{tr "Acciones"}}</th>
        </tr>
      </thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td>
            <strong>{{.Agente.Nombre}}</strong><br>
            <small>{{if .Agente.Habilitado}}{{tr "agentes.enabled"}}{{else}}{{tr "agentes.retired"}}{{end}} · {{if .Agente.Activo}}{{tr "agentes.active_now"}}{{else}}{{tr "agentes.inactive_now"}}{{end}}</small>
          </td>
          <td>{{orDash .Agente.Rol}}</td>
          <td>
            <span class="es-{{orDash .Agente.EstadoSesion}}">{{orDash .Agente.EstadoSesion}}</span><br>
            <small>{{tr "Cuota"}}: {{orDash .Agente.EstadoCuota}}</small><br>
            <small>{{tr "Última sesión"}}: {{ftime .Agente.UltimaSesion}}</small>
            {{if .Agente.ReanimarAt}}<br><small>{{tr "Reanimación"}}: {{reanimacionEn .Agente.ReanimarAt}}</small>{{end}}
          </td>
          <td>
            {{if .Asignacion}}
              <strong>{{.Asignacion.ProyectoSlug}}</strong><br>
              <small>{{orDash .Asignacion.Nota}}</small>
            {{else}}—{{end}}
          </td>
          <td>
            {{if .Sesion}}
              <strong>#{{.Sesion.ID}}</strong> · {{orDash .Sesion.Estado}}<br>
              <small>{{orDash .Sesion.Herramienta}} · {{orDash .Sesion.Branch}} · {{orDash .Sesion.Host}}</small><br>
              <small>heartbeat {{ftime .Sesion.HeartbeatAt}}</small>
            {{else}}—{{end}}
          </td>
          <td>
            {{if .Runtime}}
              <a href="/runtimes/{{.Runtime.ID}}">#{{.Runtime.ID}}</a> · {{orDash .Runtime.LogicalState}}<br>
              <small>{{orDash .Runtime.Connector}} · {{orDash .Runtime.Model}} · pid {{pid .Runtime.PID}}</small><br>
              <small>{{ftime .Runtime.UltimaActividadAt}}</small>
            {{else}}—{{end}}
          </td>
          <td>
            {{if .Handle}}
              {{orDash .Handle.Estado}}<br>
              <small>{{orDash .Handle.Transporte}} / {{orDash .Handle.HandleKind}}</small><br>
              <small>{{orDash .Handle.HandleRef}}</small>
            {{else}}—{{end}}
          </td>
          <td>
            <small>{{tr "agentes.open_tasks"}}: {{.OpenTasks}}</small><br>
            <small>{{tr "agentes.orders_open"}}: {{.OrdersOpen}}</small><br>
            <small>{{tr "agentes.mailbox_pending"}}: {{.MailboxPending}} / {{.MailboxTotal}}</small>
            {{if gt .OrdersFailed 0}}<br><small>{{tr "agentes.orders_failed"}}: {{.OrdersFailed}}</small>{{end}}
          </td>
          <td>
            {{if .LastCheckpoint}}
              <a href="/time-travel/{{.LastCheckpoint.ID}}">#{{.LastCheckpoint.ID}}</a> · {{orDash .LastCheckpoint.CheckpointKind}}<br>
              <small>{{ftimev .LastCheckpoint.CreatedAt}}</small><br>
              <small>{{tr "agentes.checkpoint_total"}}: {{.Checkpoints}}</small>
            {{else}}—{{end}}
          </td>
          <td><a href="/agentes/{{.Agente.Nombre}}">{{tr "Ver"}}</a></td>
        </tr>
        {{end}}
      </tbody>
    </table>
  </div>
  {{else}}
    <p>{{tr "agentes.none_visible"}}</p>
  {{end}}
</section>
{{if .AutoRefresh}}
<script>
window.setTimeout(function(){ window.location.reload(); }, 5000);
</script>
{{end}}
{{end}}`

const webTplAgenteDetalle = `{{define "content"}}
<section>
  <p><a href="/agentes">{{tr "agentes.detail.back"}}</a></p>
  <h2>{{tr "Agente"}} {{.Row.Agente.Nombre}}</h2>
  {{if .Msg}}<div class="alert-ok">{{.Msg}}</div>{{end}}
  {{if .Err}}<div class="alert-err">{{.Err}}</div>{{end}}
  <div class="stats">
    <div class="stat"><div class="n">{{orDash .Row.Agente.EstadoSesion}}</div><div class="l">{{tr "Sesión"}}</div></div>
    <div class="stat"><div class="n">{{orDash .Row.Agente.EstadoCuota}}</div><div class="l">{{tr "Cuota"}}</div></div>
    <div class="stat"><div class="n">{{.Row.OrdersOpen}}</div><div class="l">{{tr "agentes.orders_open"}}</div></div>
    <div class="stat"><div class="n">{{.Row.MailboxPending}}</div><div class="l">{{tr "agentes.mailbox_pending"}}</div></div>
    <div class="stat"><div class="n">{{.Row.Checkpoints}}</div><div class="l">{{tr "agentes.checkpoint_total"}}</div></div>
  </div>
  <div class="grid2">
    <aside>
      <article>
        <header><strong>{{tr "agentes.detail.summary"}}</strong></header>
        <p><strong>{{tr "Rol"}}:</strong> {{orDash .Row.Agente.Rol}}</p>
        <p><strong>{{tr "agentes.enabled"}}:</strong> {{if .Row.Agente.Habilitado}}{{tr "common.yes"}}{{else}}{{tr "common.no"}}{{end}}</p>
        <p><strong>{{tr "agentes.active_now"}}:</strong> {{if .Row.Agente.Activo}}{{tr "common.yes"}}{{else}}{{tr "common.no"}}{{end}}</p>
        <p><strong>{{tr "Última sesión"}}:</strong> {{ftime .Row.Agente.UltimaSesion}}</p>
        <p><strong>{{tr "Reanimación"}}:</strong> {{if .Row.Agente.ReanimarAt}}{{reanimacionEn .Row.Agente.ReanimarAt}}{{else}}—{{end}}</p>
        {{if .Row.Agente.MotivoPausa}}<p><strong>{{tr "Motivo"}}:</strong> {{.Row.Agente.MotivoPausa}}</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.control"}}</strong></header>
        <form method="POST" action="/agentes/{{.Row.Agente.Nombre}}/control" style="display:grid;gap:.6rem">
          <div><label>{{tr "Acción"}}</label>
            <select name="accion">
              <option value="arrancar">{{tr "Arrancar"}}</option>
              <option value="pausar">{{tr "Pausar"}}</option>
              <option value="continuar">{{tr "Continuar"}}</option>
              <option value="detener">{{tr "Detener"}}</option>
            </select>
          </div>
          <div><label>{{tr "Proyecto"}}</label><input type="text" name="proyecto" value="{{if .Row.Asignacion}}{{.Row.Asignacion.ProyectoSlug}}{{end}}"></div>
          <div><label>{{tr "Conector"}}</label><input type="text" name="conector" value="{{if .Row.Runtime}}{{.Row.Runtime.Connector}}{{end}}"></div>
          <div><label>{{tr "Modelo"}}</label><input type="text" name="modelo" value="{{if .Row.Runtime}}{{.Row.Runtime.Model}}{{end}}"></div>
          <div><label>{{tr "Reasoning"}}</label><input type="text" name="razonamiento" value="{{if .Row.Runtime}}{{.Row.Runtime.Reasoning}}{{end}}"></div>
          <div><label>{{tr "Perfil"}}</label><input type="text" name="perfil" value="{{if .Row.Runtime}}{{.Row.Runtime.TaskProfile}}{{end}}"></div>
          <div><label>{{tr "Motivo"}}</label><input type="text" name="motivo" placeholder="{{tr "runtime.control.reason_placeholder"}}"></div>
          <div><button type="submit" class="btn-sm">{{tr "runtime.control.enqueue"}}</button></div>
        </form>
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.admin"}}</strong></header>
        <form method="POST" action="/agentes/{{.Row.Agente.Nombre}}/estado" style="display:grid;gap:.4rem">
          <button type="submit" class="btn-sm" name="accion" value="retirar">{{tr "Retirar"}}</button>
          <button type="submit" class="btn-sm" name="accion" value="rehabilitar">{{tr "Rehabilitar"}}</button>
          <button type="submit" class="btn-sm" name="accion" value="reset-reanimacion">{{tr "agentes.reset_reanimation"}}</button>
        </form>
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.assign_project"}}</strong></header>
        <form method="POST" action="/agentes/{{.Row.Agente.Nombre}}/asignacion" style="display:grid;gap:.6rem">
          <div>
            <label>{{tr "agentes.assign.choose_project"}}</label>
            <select name="proyecto" required>
              <option value="">{{tr "agentes.assign.choose_project"}}</option>
              {{range .Proyectos}}<option value="{{.Slug}}">{{.Nombre}} · {{.Slug}}</option>{{end}}
            </select>
          </div>
          <div><label>{{tr "Nota"}}</label><input type="text" name="nota" placeholder="{{tr "agentes.assign.note_placeholder"}}"></div>
          <div><button type="submit" class="btn-sm">{{tr "agentes.assign.submit"}}</button></div>
        </form>
      </article>
    </aside>
    <div>
      <article>
        <header><strong>{{tr "agentes.detail.assignments"}}</strong></header>
        {{if .Asignaciones}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "Proyecto"}}</th><th>{{tr "Estado"}}</th><th>{{tr "Motivo"}}</th></tr></thead>
          <tbody>{{range .Asignaciones}}<tr><td>{{.ID}}</td><td>{{.ProyectoSlug}}</td><td>{{.Estado}}</td><td>{{orDash .Nota}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.sessions"}}</strong></header>
        {{if .Sesiones}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "Proyecto"}}</th><th>{{tr "Estado"}}</th><th>{{tr "agentes.tool"}}</th><th>{{tr "common.branch"}}</th><th>{{tr "common.host"}}</th><th>{{tr "common.heartbeat"}}</th></tr></thead>
          <tbody>{{range .Sesiones}}<tr><td>{{.ID}}</td><td>{{orDash .ProyectoSlug}}</td><td>{{orDash .Estado}}</td><td>{{orDash .Herramienta}}</td><td>{{orDash .Branch}}</td><td>{{orDash .Host}}</td><td>{{ftime .HeartbeatAt}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.runtimes"}}</strong></header>
        {{if .Runtimes}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "Proyecto"}}</th><th>{{tr "Estado"}}</th><th>{{tr "Connector"}}</th><th>{{tr "Modelo"}}</th><th>{{tr "common.pid"}}</th><th>{{tr "runtime.last_signal"}}</th></tr></thead>
          <tbody>{{range .Runtimes}}<tr><td><a href="/runtimes/{{.ID}}">{{.ID}}</a></td><td>{{orDash .ProyectoSlug}}</td><td>{{orDash .LogicalState}}</td><td>{{orDash .Connector}}</td><td>{{orDash .Model}}</td><td>{{pid .PID}}</td><td>{{ftime .UltimaActividadAt}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.handles"}}</strong></header>
        {{if .Handles}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "Estado"}}</th><th>{{tr "agentes.transport"}}</th><th>{{tr "common.kind"}}</th><th>{{tr "common.ref"}}</th><th>{{tr "common.last_seen"}}</th></tr></thead>
          <tbody>{{range .Handles}}<tr><td>{{.ID}}</td><td>{{orDash .Estado}}</td><td>{{orDash .Transporte}}</td><td>{{orDash .HandleKind}}</td><td>{{orDash .HandleRef}}</td><td>{{ftime .LastSeenAt}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.orders"}}</strong></header>
        {{if .Orders}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "Tipo"}}</th><th>{{tr "Estado"}}</th><th>{{tr "Creado"}}</th><th>{{tr "Detalle"}}</th></tr></thead>
          <tbody>{{range .Orders}}<tr><td>{{.ID}}</td><td>{{orDash .Tipo}}</td><td>{{orDash .Estado}}</td><td>{{ftimev .CreatedAt}}</td><td>{{trunc .PayloadJSON 80}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.mailbox"}}</strong></header>
        {{if .Mailbox}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "common.from"}}</th><th>{{tr "common.to"}}</th><th>{{tr "common.kind"}}</th><th>{{tr "Estado"}}</th><th>{{tr "Creado"}}</th></tr></thead>
          <tbody>{{range .Mailbox}}<tr><td>{{.ID}}</td><td>{{orDash .FromAgente}}</td><td>{{orDash .ToAgente}}</td><td>{{orDash .Kind}}</td><td>{{orDash .Estado}}</td><td>{{ftimev .CreatedAt}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.transcript"}}</strong></header>
        {{if .Transcript}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "runtime.when"}}</th><th>{{tr "agentes.transcript.stream"}}</th><th>{{tr "agentes.transcript.signal"}}</th><th>{{tr "runtime.detail"}}</th><th>{{tr "agentes.transcript.handled"}}</th></tr></thead>
          <tbody>{{range .Transcript}}<tr><td>{{.ID}}</td><td>{{ftimev .CreatedAt}}</td><td>{{orDash .Stream}}</td><td>{{orDash .Classification}}</td><td>{{trunc .Text 140}}</td><td>{{if .HandledAt}}{{ftimev .HandledAt}}{{else}}{{orDash .HandlingNote}}{{end}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>{{tr "agentes.transcript.none"}}</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.checkpoints"}}</strong></header>
        {{if .Checkpoints}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "Tipo"}}</th><th>{{tr "common.branch"}}</th><th>{{tr "time_travel.source"}}</th><th>{{tr "Creado"}}</th></tr></thead>
          <tbody>{{range .Checkpoints}}<tr><td><a href="/time-travel/{{.ID}}">{{.ID}}</a></td><td>{{orDash .CheckpointKind}}</td><td>{{orDash .Branch}}</td><td>{{orDash .Source}}</td><td>{{ftimev .CreatedAt}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
    </div>
  </div>
</section>
{{end}}`
