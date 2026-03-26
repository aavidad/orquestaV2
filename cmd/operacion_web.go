package cmd

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"orquesta/db"
)

type webAuditoriaData struct {
	Entries []db.AuditEntry
	Limit   int
	Agente  string
	Accion  string
	Entidad string
	Err     string
}

type webRespaldoData struct {
	Msg string
	Err string
}

type webDiagnosticoData struct {
	Snapshot *db.SnapshotDiagnostico
	Err      string
}

type apiRefineriaListResponse struct {
	Solicitudes []*db.RefineriaSolicitud `json:"solicitudes"`
}

type apiRefineriaDetailResponse struct {
	Solicitud *db.RefineriaSolicitud `json:"solicitud"`
}

type webRefineriaData struct {
	Solicitudes []*db.RefineriaSolicitud
	TaskFilter  string
	Err         string
}

type webRefineriaDetalleData struct {
	Solicitud *db.RefineriaSolicitud
	Historial []*db.RefineriaSolicitud
	Err       string
}

func webHandlerAuditoria(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	query := url.Values{"limit": []string{strconv.Itoa(limit)}}
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	accion := strings.TrimSpace(r.URL.Query().Get("accion"))
	entidad := strings.TrimSpace(r.URL.Query().Get("entidad"))
	if agente != "" {
		query.Set("agente", agente)
	}
	if accion != "" {
		query.Set("accion", accion)
	}
	if entidad != "" {
		query.Set("entidad", entidad)
	}
	var resp apiAuditResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/audit?"+query.Encode(), nil, &resp); err != nil {
		webRender(w, r, webTplLayout+webTplAuditoria, webAuditoriaData{
			Limit:   limit,
			Agente:  agente,
			Accion:  accion,
			Entidad: entidad,
			Err:     err.Error(),
		})
		return
	}
	webRender(w, r, webTplLayout+webTplAuditoria, webAuditoriaData{
		Entries: resp.Audit,
		Limit:   limit,
		Agente:  agente,
		Accion:  accion,
		Entidad: entidad,
	})
}

func webHandlerRespaldo(w http.ResponseWriter, r *http.Request) {
	webRender(w, r, webTplLayout+webTplRespaldo, webRespaldoData{
		Msg: r.URL.Query().Get("ok"),
		Err: r.URL.Query().Get("err"),
	})
}

func webHandlerRespaldoCrear(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	retener := 0
	if raw := strings.TrimSpace(r.FormValue("retener")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 0 {
			retener = v
		}
	}
	var resp apiRespaldoBDResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/respaldo/bd", apiRespaldoBDRequest{
		Destino:  strings.TrimSpace(r.FormValue("destino")),
		Etiqueta: strings.TrimSpace(r.FormValue("etiqueta")),
		Retener:  retener,
	}, &resp); err != nil {
		http.Redirect(w, r, "/respaldo?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/respaldo?ok="+url.QueryEscape(webTranslateRequestf(r, "ops.backup.saved", resp.Ruta)), http.StatusSeeOther)
}

func webHandlerDiagnostico(w http.ResponseWriter, r *http.Request) {
	query := url.Values{}
	if limit := strings.TrimSpace(r.URL.Query().Get("audit_limit")); limit != "" {
		query.Set("audit_limit", limit)
	}
	path := "/api/diagnostico"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp apiDiagnosticoResponse
	data := webDiagnosticoData{}
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		data.Err = err.Error()
	} else {
		data.Snapshot = &resp.Diagnostico
	}
	webRender(w, r, webTplLayout+webTplDiagnostico, data)
}

func webRouterRefineria(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/refineria", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerRefineriaDetalle(w, r, parts[1])
	case len(parts) == 3 && parts[1] == "tarea" && r.Method == http.MethodGet:
		webHandlerRefineriaPorTarea(w, r, parts[2])
	default:
		http.NotFound(w, r)
	}
}

func webHandlerRefineria(w http.ResponseWriter, r *http.Request) {
	var resp apiRefineriaListResponse
	err := webInvocarAPIJSON(http.MethodGet, "/api/refineria", nil, &resp)
	data := webRefineriaData{
		Solicitudes: resp.Solicitudes,
		TaskFilter:  strings.TrimSpace(r.URL.Query().Get("tarea")),
	}
	if err != nil {
		data.Err = err.Error()
	}
	webRender(w, r, webTplLayout+webTplRefineria, data)
}

func webHandlerRefineriaDetalle(w http.ResponseWriter, r *http.Request, idRaw string) {
	var detalle apiRefineriaDetailResponse
	err := webInvocarAPIJSON(http.MethodGet, "/api/refineria/"+url.PathEscape(strings.TrimSpace(idRaw)), nil, &detalle)
	data := webRefineriaDetalleData{}
	if err != nil {
		data.Err = err.Error()
		webRender(w, r, webTplLayout+webTplRefineriaDetalle, data)
		return
	}
	data.Solicitud = detalle.Solicitud
	if detalle.Solicitud != nil {
		var historial apiRefineriaListResponse
		if err := webInvocarAPIJSON(http.MethodGet, "/api/refineria/tarea/"+strconv.FormatInt(detalle.Solicitud.TareaID, 10), nil, &historial); err == nil {
			data.Historial = historial.Solicitudes
		}
	}
	webRender(w, r, webTplLayout+webTplRefineriaDetalle, data)
}

func webHandlerRefineriaPorTarea(w http.ResponseWriter, r *http.Request, tareaRaw string) {
	var resp apiRefineriaListResponse
	err := webInvocarAPIJSON(http.MethodGet, "/api/refineria/tarea/"+url.PathEscape(strings.TrimSpace(tareaRaw)), nil, &resp)
	data := webRefineriaData{
		Solicitudes: resp.Solicitudes,
		TaskFilter:  strings.TrimSpace(tareaRaw),
	}
	if err != nil {
		data.Err = err.Error()
	}
	webRender(w, r, webTplLayout+webTplRefineria, data)
}

const webTplAuditoria = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "ops.audit.title"}}</h2>
  <p style="color:#64748b">{{tr "ops.audit.subtitle"}}</p>
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <form method="get" action="/auditoria" style="display:flex;gap:.7rem;flex-wrap:wrap;align-items:end;margin-bottom:1rem">
    <label>{{tr "agents.agent"}} <input name="agente" value="{{.Agente}}"></label>
    <label>{{tr "ops.audit.action"}} <input name="accion" value="{{.Accion}}"></label>
    <label>{{tr "ops.audit.entity"}} <input name="entidad" value="{{.Entidad}}"></label>
    <label>{{tr "ops.audit.limit"}} <input name="limit" type="number" min="1" value="{{.Limit}}"></label>
    <button type="submit">{{tr "common.view"}}</button>
  </form>
  <table>
    <thead><tr><th>{{tr "common.date"}}</th><th>{{tr "agents.agent"}}</th><th>{{tr "ops.audit.action"}}</th><th>{{tr "ops.audit.entity"}}</th><th>ID</th><th>{{tr "common.detail"}}</th></tr></thead>
    <tbody>
      {{range .Entries}}
      <tr>
        <td>{{ftime .CreatedAt}}</td>
        <td>{{.Agente}}</td>
        <td>{{.Accion}}</td>
        <td>{{.Entidad}}</td>
        <td>{{.EntidadID}}</td>
        <td>{{.Detalle}}</td>
      </tr>
      {{else}}
      <tr><td colspan="6">{{tr "ops.audit.none"}}</td></tr>
      {{end}}
    </tbody>
  </table>
</section>
{{end}}`

const webTplRespaldo = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "ops.backup.title"}}</h2>
  <p style="color:#64748b">{{tr "ops.backup.subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <form method="post" action="/respaldo" style="display:grid;gap:.7rem;max-width:48rem">
    <label>{{tr "ops.backup.destination"}} <input name="destino"></label>
    <div class="grid2">
      <label>{{tr "ops.backup.label"}} <input name="etiqueta"></label>
      <label>{{tr "ops.backup.retain"}} <input name="retener" type="number" min="0" value="0"></label>
    </div>
    <button type="submit">{{tr "ops.backup.create"}}</button>
  </form>
</section>
{{end}}`

const webTplRefineria = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "ops.refinery.title"}}</h2>
  <p style="color:#64748b">{{tr "ops.refinery.subtitle"}}</p>
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  {{if .TaskFilter}}<p><strong>{{tr "common.task"}}:</strong> #{{.TaskFilter}}</p>{{end}}
  <table>
    <thead><tr><th>ID</th><th>{{tr "common.task"}}</th><th>{{tr "Agente"}}</th><th>{{tr "common.branch"}}</th><th>{{tr "ops.refinery.command"}}</th><th>{{tr "common.status"}}</th><th>{{tr "common.view"}}</th></tr></thead>
    <tbody>
      {{range .Solicitudes}}
      <tr>
        <td>#{{.ID}}</td>
        <td><a href="/refineria/tarea/{{.TareaID}}">#{{.TareaID}}</a></td>
        <td>{{.Agente}}</td>
        <td><code>{{.Rama}}</code></td>
        <td><code>{{.CmdTest}}</code></td>
        <td>{{.Estado}}</td>
        <td><a href="/refineria/{{.ID}}">{{tr "common.view"}}</a></td>
      </tr>
      {{else}}
      <tr><td colspan="7">{{tr "ops.refinery.none"}}</td></tr>
      {{end}}
    </tbody>
  </table>
</section>
{{end}}`

const webTplRefineriaDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/refineria">← {{tr "ops.refinery.back"}}</a></p>
  {{if .Err}}
  <article class="alert-err">{{.Err}}</article>
  {{else if .Solicitud}}
  <h2 style="margin:0">{{tr "ops.refinery.detail_title"}} #{{.Solicitud.ID}}</h2>
  <table>
    <tbody>
      <tr><th>{{tr "common.task"}}</th><td>#{{.Solicitud.TareaID}}</td></tr>
      <tr><th>{{tr "Agente"}}</th><td>{{.Solicitud.Agente}}</td></tr>
      <tr><th>{{tr "common.branch"}}</th><td><code>{{.Solicitud.Rama}}</code></td></tr>
      <tr><th>{{tr "ops.refinery.workdir"}}</th><td><code>{{.Solicitud.DirTrabajo}}</code></td></tr>
      <tr><th>{{tr "ops.refinery.command"}}</th><td><code>{{.Solicitud.CmdTest}}</code></td></tr>
      <tr><th>{{tr "common.status"}}</th><td>{{.Solicitud.Estado}}</td></tr>
      <tr><th>{{tr "ops.refinery.result"}}</th><td><pre style="margin:0;white-space:pre-wrap">{{.Solicitud.Resultado}}</pre></td></tr>
      <tr><th>{{tr "ops.refinery.error"}}</th><td><pre style="margin:0;white-space:pre-wrap">{{.Solicitud.ErrorTexto}}</pre></td></tr>
      <tr><th>{{tr "ops.refinery.merge_commit"}}</th><td><code>{{.Solicitud.CommitMerge}}</code></td></tr>
    </tbody>
  </table>
  {{if .Historial}}
  <h3 style="margin:1rem 0 .5rem 0">{{tr "ops.refinery.history"}}</h3>
  <table>
    <thead><tr><th>ID</th><th>{{tr "common.status"}}</th><th>{{tr "common.branch"}}</th><th>{{tr "common.started_at"}}</th><th>{{tr "common.finished_at"}}</th></tr></thead>
    <tbody>
      {{range .Historial}}
      <tr>
        <td><a href="/refineria/{{.ID}}">#{{.ID}}</a></td>
        <td>{{.Estado}}</td>
        <td><code>{{.Rama}}</code></td>
        <td>{{.StartedAt}}</td>
        <td>{{.FinishedAt}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
  {{end}}
  {{end}}
</section>
{{end}}`

const webTplDiagnostico = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "ops.diagnosis.title"}}</h2>
  <p style="color:#64748b">{{tr "ops.diagnosis.subtitle"}}</p>
  {{if .Err}}
  <article class="alert-err">{{.Err}}</article>
  {{else if .Snapshot}}
  <div class="stats">
    <div class="stat"><div class="n">{{len .Snapshot.Agentes}}</div><div class="l">{{tr "Agentes"}}</div></div>
    <div class="stat"><div class="n">{{len .Snapshot.SesionesActivas}}</div><div class="l">{{tr "ops.diagnosis.active_sessions"}}</div></div>
    <div class="stat"><div class="n">{{len .Snapshot.TareasEnProgreso}}</div><div class="l">{{tr "En progreso"}}</div></div>
    <div class="stat"><div class="n">{{len .Snapshot.TareasBloqueadas}}</div><div class="l">{{tr "Bloqueadas"}}</div></div>
    <div class="stat"><div class="n">{{len .Snapshot.WorktreesActivos}}</div><div class="l">{{tr "ops.diagnosis.worktrees"}}</div></div>
    <div class="stat"><div class="n">{{len .Snapshot.LocksActivos}}</div><div class="l">{{tr "ops.diagnosis.locks"}}</div></div>
  </div>

  <p><strong>{{tr "ops.diagnosis.generated_at"}}:</strong> {{ftime .Snapshot.GeneradoEn}}</p>

  <section class="container">
    <h3 style="margin:0 0 .5rem 0">{{tr "ops.diagnosis.task_counts"}}</h3>
    <table>
      <thead><tr><th>{{tr "Estado"}}</th><th>{{tr "ops.diagnosis.count"}}</th></tr></thead>
      <tbody>
        {{range $estado, $count := .Snapshot.ConteoTareas}}
        <tr><td>{{$estado}}</td><td>{{$count}}</td></tr>
        {{end}}
      </tbody>
    </table>
  </section>

  <section class="container">
    <h3 style="margin:0 0 .5rem 0">{{tr "ops.diagnosis.open_proposals"}}</h3>
    <table>
      <thead><tr><th>{{tr "dashboard.code"}}</th><th>{{tr "Propuestas"}}</th><th>{{tr "acuerdo"}}</th><th>{{tr "desacuerdo"}}</th><th>{{tr "pendiente"}}</th></tr></thead>
      <tbody>
        {{range .Snapshot.PropuestasAbiertas}}
        <tr>
          <td>{{.Propuesta.Codigo}}</td>
          <td>{{.Propuesta.Titulo}}</td>
          <td>{{.Acuerdo}}</td>
          <td>{{.Desacuerdo}}</td>
          <td>{{.Pendiente}}</td>
        </tr>
        {{else}}
        <tr><td colspan="5">{{tr "ops.diagnosis.none"}}</td></tr>
        {{end}}
      </tbody>
    </table>
  </section>

  <section class="container">
    <h3 style="margin:0 0 .5rem 0">{{tr "ops.audit.title"}}</h3>
    <table>
      <thead><tr><th>{{tr "common.started_at"}}</th><th>{{tr "Agente"}}</th><th>{{tr "Acción"}}</th><th>{{tr "Entidad"}}</th></tr></thead>
      <tbody>
        {{range .Snapshot.AuditoriaReciente}}
        <tr><td>{{ftime .CreatedAt}}</td><td>{{.Agente}}</td><td>{{.Accion}}</td><td>{{.Entidad}}</td></tr>
        {{else}}
        <tr><td colspan="4">{{tr "ops.diagnosis.none"}}</td></tr>
        {{end}}
      </tbody>
    </table>
  </section>
  {{end}}
</section>
{{end}}`
