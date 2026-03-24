package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

type webAgenteRow struct {
	Agente         *db.Agente
	Asignacion     *db.Asignacion
	Sesion         *db.Sesion
	Runtime        *db.RuntimeInstance
	Handle         *db.RuntimeHandle
	OrdersOpen     int
	OrdersFailed   int
	MailboxPending int
	MailboxTotal   int
	Checkpoints    int
	LastCheckpoint *db.RuntimeCheckpoint
	OpenTasks      int
}

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
	Asignaciones []*db.Asignacion
	Sesiones     []*db.Sesion
	Runtimes     []*db.RuntimeInstance
	Handles      []*db.RuntimeHandle
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
		webRender(w, webTplLayout+webTplAgentesPanel, webAgentesPanelData{
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
	webRender(w, webTplLayout+webTplAgentesPanel, data)
}

func webHandlerAgenteNuevo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/agentes?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	nombre := strings.TrimSpace(r.FormValue("nombre"))
	rol := strings.TrimSpace(r.FormValue("rol"))
	if err := db.RegistrarAgente(nombre, rol); err != nil {
		http.Redirect(w, r, "/agentes?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/agentes?ok="+url.QueryEscape("Agente "+nombre+" registrado"), http.StatusSeeOther)
}

func webHandlerAgenteDetalle(w http.ResponseWriter, r *http.Request, nombre string) {
	data, err := construirWebAgenteDetalleData(strings.TrimSpace(nombre))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data.Msg = r.URL.Query().Get("ok")
	data.Err = r.URL.Query().Get("err")
	webRender(w, webTplLayout+webTplAgenteDetalle, data)
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
	orderID, accion, err := encolarControlAgenteLocal(req)
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
	var err error
	switch accion {
	case "retirar":
		err = db.RetirarAgente(nombre)
	case "rehabilitar":
		err = db.RehabilitarAgente(nombre)
	case "reset-reanimacion":
		err = db.ResetReanimacion(nombre)
	default:
		err = fmt.Errorf("acción de agente no soportada: %s", accion)
	}
	if err != nil {
		http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/agentes/"+url.PathEscape(nombre)+"?ok="+url.QueryEscape("Acción '"+accion+"' aplicada"), http.StatusSeeOther)
}

func construirWebAgenteRows() ([]webAgenteRow, error) {
	agentes, err := db.ListarAgentes()
	if err != nil {
		return nil, err
	}
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{})
	if err != nil {
		return nil, err
	}
	activa := true
	sesiones, err := db.ListarSesionesInspeccion(db.FiltroSesionesInspeccion{Activa: &activa})
	if err != nil {
		return nil, err
	}
	runtimes, err := db.ListarRuntimes(db.FiltroRuntimes{})
	if err != nil {
		return nil, err
	}
	handles, err := db.ListarRuntimeHandles(nil)
	if err != nil {
		return nil, err
	}
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{})
	if err != nil {
		return nil, err
	}
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{})
	if err != nil {
		return nil, err
	}
	checkpoints, err := db.ListarRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{})
	if err != nil {
		return nil, err
	}
	tareas, err := db.ListarTareas(db.FiltroTareas{})
	if err != nil {
		return nil, err
	}

	asignacionPorAgente := map[string]*db.Asignacion{}
	for _, asignacion := range asignaciones {
		if asignacion == nil || asignacion.Estado != db.AsignacionActiva {
			continue
		}
		if _, ok := asignacionPorAgente[asignacion.Agente]; !ok {
			asignacionPorAgente[asignacion.Agente] = asignacion
		}
	}

	sesionPorAgente := map[string]*db.Sesion{}
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		if _, ok := sesionPorAgente[sesion.Agente]; !ok {
			sesionPorAgente[sesion.Agente] = sesion
		}
	}

	runtimePorAgente := map[string]*db.RuntimeInstance{}
	for _, runtime := range runtimes {
		if runtime == nil {
			continue
		}
		actual := runtimePorAgente[runtime.Agente]
		if actual == nil || runtimeMomento(runtime).After(runtimeMomento(actual)) {
			runtimePorAgente[runtime.Agente] = runtime
		}
	}

	handlePorAgente := map[string]*db.RuntimeHandle{}
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		actual := handlePorAgente[handle.Agente]
		if actual == nil || runtimeHandleMomento(handle).After(runtimeHandleMomento(actual)) {
			handlePorAgente[handle.Agente] = handle
		}
	}

	type orderCounters struct {
		Open   int
		Failed int
	}
	ordersPorAgente := map[string]orderCounters{}
	for _, order := range orders {
		if order == nil {
			continue
		}
		stats := ordersPorAgente[order.Agente]
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			stats.Open++
		case "fallida":
			stats.Failed++
		}
		if strings.TrimSpace(order.ErrorText) != "" {
			stats.Failed++
		}
		ordersPorAgente[order.Agente] = stats
	}

	type mailboxCounters struct {
		Pending int
		Total   int
	}
	mailboxPorAgente := map[string]mailboxCounters{}
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		for _, agente := range []string{msg.ToAgente, msg.FromAgente} {
			if strings.TrimSpace(agente) == "" {
				continue
			}
			stats := mailboxPorAgente[agente]
			stats.Total++
			if strings.TrimSpace(msg.Estado) == "pendiente" {
				stats.Pending++
			}
			mailboxPorAgente[agente] = stats
		}
	}

	checkpointTotalPorAgente := map[string]int{}
	lastCheckpointPorAgente := map[string]*db.RuntimeCheckpoint{}
	for _, checkpoint := range checkpoints {
		if checkpoint == nil {
			continue
		}
		checkpointTotalPorAgente[checkpoint.Agente]++
		if _, ok := lastCheckpointPorAgente[checkpoint.Agente]; !ok {
			lastCheckpointPorAgente[checkpoint.Agente] = checkpoint
		}
	}

	openTasksPorAgente := map[string]int{}
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
		}
		switch tarea.Estado {
		case db.EstadoCompletada, db.EstadoCancelada:
			continue
		}
		openTasksPorAgente[*tarea.Agente]++
	}

	rows := make([]webAgenteRow, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		orderStats := ordersPorAgente[agente.Nombre]
		mailboxStats := mailboxPorAgente[agente.Nombre]
		rows = append(rows, webAgenteRow{
			Agente:         agente,
			Asignacion:     asignacionPorAgente[agente.Nombre],
			Sesion:         sesionPorAgente[agente.Nombre],
			Runtime:        runtimePorAgente[agente.Nombre],
			Handle:         handlePorAgente[agente.Nombre],
			OrdersOpen:     orderStats.Open,
			OrdersFailed:   orderStats.Failed,
			MailboxPending: mailboxStats.Pending,
			MailboxTotal:   mailboxStats.Total,
			Checkpoints:    checkpointTotalPorAgente[agente.Nombre],
			LastCheckpoint: lastCheckpointPorAgente[agente.Nombre],
			OpenTasks:      openTasksPorAgente[agente.Nombre],
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].Agente.Nombre) < strings.ToLower(rows[j].Agente.Nombre)
	})
	return rows, nil
}

func construirWebAgenteDetalleData(nombre string) (*webAgenteDetalleData, error) {
	agente, err := db.GetAgente(nombre)
	if err != nil {
		return nil, err
	}
	rows, err := construirWebAgenteRows()
	if err != nil {
		return nil, err
	}
	row := webAgenteRow{Agente: agente}
	for _, item := range rows {
		if item.Agente != nil && item.Agente.Nombre == nombre {
			row = item
			break
		}
	}

	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	sesiones, err := db.ListarSesionesInspeccion(db.FiltroSesionesInspeccion{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	runtimes, err := db.ListarRuntimes(db.FiltroRuntimes{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	handles, err := db.ListarRuntimeHandles(&nombre)
	if err != nil {
		return nil, err
	}
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	mailbox, err := listarMailboxAgente(nombre)
	if err != nil {
		return nil, err
	}
	checkpoints, err := db.ListarRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Agente: &nombre, Limit: 20})
	if err != nil {
		return nil, err
	}

	return &webAgenteDetalleData{
		Row:          row,
		Asignaciones: asignaciones,
		Sesiones:     limitarSesiones(sesiones, 12),
		Runtimes:     limitarRuntimes(runtimes, 12),
		Handles:      limitarHandles(handles, 12),
		Orders:       limitarOrders(orders, 20),
		Mailbox:      limitarMailbox(mailbox, 20),
		Checkpoints:  checkpoints,
	}, nil
}

func listarMailboxAgente(nombre string) ([]*db.RuntimeMailboxMessage, error) {
	toAgente := nombre
	inbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &toAgente})
	if err != nil {
		return nil, err
	}
	fromAgente := nombre
	outbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{FromAgente: &fromAgente})
	if err != nil {
		return nil, err
	}
	merged := make([]*db.RuntimeMailboxMessage, 0, len(inbox)+len(outbox))
	seen := map[int64]struct{}{}
	for _, set := range [][]*db.RuntimeMailboxMessage{inbox, outbox} {
		for _, msg := range set {
			if msg == nil {
				continue
			}
			if _, ok := seen[msg.ID]; ok {
				continue
			}
			seen[msg.ID] = struct{}{}
			merged = append(merged, msg)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].ID > merged[j].ID
	})
	return merged, nil
}

func runtimeMomento(runtime *db.RuntimeInstance) time.Time {
	if runtime == nil {
		return time.Time{}
	}
	for _, value := range []*time.Time{runtime.UltimaActividadAt, runtime.LastHeartbeatAt, runtime.LastEventAt} {
		if value != nil && !value.IsZero() {
			return *value
		}
	}
	if !runtime.UpdatedAt.IsZero() {
		return runtime.UpdatedAt
	}
	return runtime.CreatedAt
}

func runtimeHandleMomento(handle *db.RuntimeHandle) time.Time {
	if handle == nil {
		return time.Time{}
	}
	if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() {
		return *handle.LastSeenAt
	}
	if !handle.UpdatedAt.IsZero() {
		return handle.UpdatedAt
	}
	return handle.CreatedAt
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
      <form method="POST" action="/agentes/nuevo" style="display:grid;grid-template-columns:2fr 1fr auto;gap:.6rem;align-items:end;margin-top:.8rem">
        <div><label>{{tr "Agente"}}</label><input type="text" name="nombre" required></div>
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
        <p><strong>{{tr "agentes.enabled"}}:</strong> {{if .Row.Agente.Habilitado}}sí{{else}}no{{end}}</p>
        <p><strong>{{tr "agentes.active_now"}}:</strong> {{if .Row.Agente.Activo}}sí{{else}}no{{end}}</p>
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
          <thead><tr><th>ID</th><th>{{tr "Proyecto"}}</th><th>{{tr "Estado"}}</th><th>{{tr "agentes.tool"}}</th><th>branch</th><th>host</th><th>heartbeat</th></tr></thead>
          <tbody>{{range .Sesiones}}<tr><td>{{.ID}}</td><td>{{orDash .ProyectoSlug}}</td><td>{{orDash .Estado}}</td><td>{{orDash .Herramienta}}</td><td>{{orDash .Branch}}</td><td>{{orDash .Host}}</td><td>{{ftime .HeartbeatAt}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.runtimes"}}</strong></header>
        {{if .Runtimes}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "Proyecto"}}</th><th>{{tr "Estado"}}</th><th>{{tr "Connector"}}</th><th>{{tr "Modelo"}}</th><th>pid</th><th>{{tr "runtime.last_signal"}}</th></tr></thead>
          <tbody>{{range .Runtimes}}<tr><td><a href="/runtimes/{{.ID}}">{{.ID}}</a></td><td>{{orDash .ProyectoSlug}}</td><td>{{orDash .LogicalState}}</td><td>{{orDash .Connector}}</td><td>{{orDash .Model}}</td><td>{{pid .PID}}</td><td>{{ftime .UltimaActividadAt}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.handles"}}</strong></header>
        {{if .Handles}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "Estado"}}</th><th>{{tr "agentes.transport"}}</th><th>kind</th><th>ref</th><th>last_seen</th></tr></thead>
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
          <thead><tr><th>ID</th><th>from</th><th>to</th><th>kind</th><th>{{tr "Estado"}}</th><th>{{tr "Creado"}}</th></tr></thead>
          <tbody>{{range .Mailbox}}<tr><td>{{.ID}}</td><td>{{orDash .FromAgente}}</td><td>{{orDash .ToAgente}}</td><td>{{orDash .Kind}}</td><td>{{orDash .Estado}}</td><td>{{ftimev .CreatedAt}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
      <article>
        <header><strong>{{tr "agentes.detail.checkpoints"}}</strong></header>
        {{if .Checkpoints}}
        <table>
          <thead><tr><th>ID</th><th>{{tr "Tipo"}}</th><th>branch</th><th>{{tr "time_travel.source"}}</th><th>{{tr "Creado"}}</th></tr></thead>
          <tbody>{{range .Checkpoints}}<tr><td><a href="/time-travel/{{.ID}}">{{.ID}}</a></td><td>{{orDash .CheckpointKind}}</td><td>{{orDash .Branch}}</td><td>{{orDash .Source}}</td><td>{{ftimev .CreatedAt}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<p>—</p>{{end}}
      </article>
    </div>
  </div>
</section>
{{end}}`
