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

	"orquesta/coordinacion"
	"orquesta/db"
)

type webGitGovData struct {
	Estado      string
	Agente      string
	Proyecto    string
	MergeEstado string
	Worktrees   []webGitWorktreeRow
	Locks       []webGitLockRow
	Merges      []*db.GitMerge
	Msg         string
	Err         string
}

type webGitWorktreeRow struct {
	ID           int64
	ProyectoSlug string
	Agente       string
	Nombre       string
	Branch       string
	Estado       string
}

type webGitLockRow struct {
	ID        int64
	Agente    string
	ScopeType string
	ScopeKey  string
	Branch    string
	Estado    string
}

type webGitWorktreeDetalleData struct {
	Worktree *coordinacion.Worktree
}

type webGitLockDetalleData struct {
	Lock *coordinacion.Lock
}

func webRouterGitGov(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/git", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "merges" && r.Method == http.MethodPost:
		webHandlerGitMergeNueva(w, r)
	case len(parts) == 3 && parts[1] == "worktrees" && r.Method == http.MethodGet:
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || id <= 0 {
			http.NotFound(w, r)
			return
		}
		webHandlerGitWorktreeDetalle(w, r, id)
	case len(parts) == 3 && parts[1] == "locks" && r.Method == http.MethodGet:
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || id <= 0 {
			http.NotFound(w, r)
			return
		}
		webHandlerGitLockDetalle(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func webHandlerGitGov(w http.ResponseWriter, r *http.Request) {
	estado := strings.TrimSpace(r.URL.Query().Get("estado"))
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	mergeEstado := strings.TrimSpace(r.URL.Query().Get("merge_estado"))

	worktrees, err := webCargarGitWorktreesPorAPI(estado, agente, proyecto)
	if err != nil {
		webRender(w, r, webTplLayout+webTplGitGov, webGitGovData{
			Estado:      estado,
			Agente:      agente,
			Proyecto:    proyecto,
			MergeEstado: mergeEstado,
			Err:         err.Error(),
		})
		return
	}
	locks, err := webCargarGitLocksPorAPI(estado, agente, proyecto)
	if err != nil {
		webRender(w, r, webTplLayout+webTplGitGov, webGitGovData{
			Estado:      estado,
			Agente:      agente,
			Proyecto:    proyecto,
			MergeEstado: mergeEstado,
			Worktrees:   worktrees,
			Err:         err.Error(),
		})
		return
	}

	merges, mergeErr := webCargarGitMergesPorAPI(proyecto, mergeEstado)
	data := webGitGovData{
		Estado:      estado,
		Agente:      agente,
		Proyecto:    proyecto,
		MergeEstado: mergeEstado,
		Worktrees:   worktrees,
		Locks:       locks,
		Merges:      merges,
		Msg:         r.URL.Query().Get("ok"),
		Err:         r.URL.Query().Get("err"),
	}
	if mergeErr != nil {
		data.Err = mergeErr.Error()
	}
	webRender(w, r, webTplLayout+webTplGitGov, data)
}

func webHandlerGitMergeNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, err := webCrearGitMergePorAPI(apiGitMergeSaveRequest{
		ProyectoSlug: strings.TrimSpace(r.FormValue("proyecto_slug")),
		SourceBranch: strings.TrimSpace(r.FormValue("source_branch")),
		TargetBranch: strings.TrimSpace(r.FormValue("target_branch")),
		RequestedBy:  strings.TrimSpace(r.FormValue("requested_by")),
		Estado:       strings.TrimSpace(r.FormValue("estado")),
		CommitOrigen: strings.TrimSpace(r.FormValue("commit_origen")),
		CommitMerge:  strings.TrimSpace(r.FormValue("commit_merge")),
		Notas:        r.FormValue("notas"),
		MetadataJSON: strings.TrimSpace(r.FormValue("metadata_json")),
	})
	if err != nil {
		http.Redirect(w, r, "/git?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/git?ok="+url.QueryEscape(webTranslateRequestf(r, "gitgov.flash.request_saved", id)), http.StatusSeeOther)
}

func webHandlerGitWorktreeDetalle(w http.ResponseWriter, r *http.Request, id int64) {
	worktree, err := webCargarGitWorktreeDetallePorAPI(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, r, webTplLayout+webTplGitWorktreeDetalle, webGitWorktreeDetalleData{Worktree: worktree})
}

func webHandlerGitLockDetalle(w http.ResponseWriter, r *http.Request, id int64) {
	lock, err := webCargarGitLockDetallePorAPI(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, r, webTplLayout+webTplGitLockDetalle, webGitLockDetalleData{Lock: lock})
}

func webCargarGitWorktreesPorAPI(estado, agente, proyecto string) ([]webGitWorktreeRow, error) {
	query := url.Values{}
	if estado = strings.TrimSpace(estado); estado != "" {
		query.Set("estado", estado)
	}
	if agente = strings.TrimSpace(agente); agente != "" {
		query.Set("agente", agente)
	}
	if proyecto = strings.TrimSpace(proyecto); proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	path := "/api/worktrees"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp apiWorktreesResponse
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	projectSlugByID, err := webProyectoSlugPorID()
	if err != nil {
		return nil, err
	}
	rows := make([]webGitWorktreeRow, 0, len(resp.Worktrees))
	for _, item := range resp.Worktrees {
		if item == nil {
			continue
		}
		rows = append(rows, webGitWorktreeRow{
			ID:           item.ID,
			ProyectoSlug: projectSlugByID[item.ProjectID],
			Agente:       item.Agent,
			Nombre:       item.Name,
			Branch:       item.Branch,
			Estado:       string(item.State),
		})
	}
	return rows, nil
}

func webCargarGitLocksPorAPI(estado, agente, proyecto string) ([]webGitLockRow, error) {
	query := url.Values{}
	if estado = strings.TrimSpace(estado); estado != "" {
		query.Set("estado", estado)
	}
	if agente = strings.TrimSpace(agente); agente != "" {
		query.Set("agente", agente)
	}
	if proyecto = strings.TrimSpace(proyecto); proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	path := "/api/locks"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp apiLocksResponse
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	rows := make([]webGitLockRow, 0, len(resp.Locks))
	for _, item := range resp.Locks {
		if item == nil {
			continue
		}
		rows = append(rows, webGitLockRow{
			ID:        item.ID,
			Agente:    item.Agent,
			ScopeType: item.ScopeType,
			ScopeKey:  item.ScopeKey,
			Branch:    item.Branch,
			Estado:    string(item.State),
		})
	}
	return rows, nil
}

func webCargarGitWorktreeDetallePorAPI(id int64) (*coordinacion.Worktree, error) {
	var resp apiWorktreeResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/worktrees/"+strconv.FormatInt(id, 10), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Worktree, nil
}

func webCargarGitLockDetallePorAPI(id int64) (*coordinacion.Lock, error) {
	var resp apiLockResponse
	if err := webInvocarAPIJSON(http.MethodGet, "/api/locks/"+strconv.FormatInt(id, 10), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Lock, nil
}

func webCargarGitMergesPorAPI(proyecto, estado string) ([]*db.GitMerge, error) {
	query := url.Values{}
	if proyecto = strings.TrimSpace(proyecto); proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	if estado = strings.TrimSpace(estado); estado != "" {
		query.Set("estado", estado)
	}
	path := "/api/git/merges"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp apiGitMergesResponse
	if err := webInvocarAPIJSON(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Merges, nil
}

func webCrearGitMergePorAPI(req apiGitMergeSaveRequest) (int64, error) {
	var resp apiGitMergeSaveResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/git/merges", req, &resp); err != nil {
		return 0, err
	}
	return resp.ID, nil
}

func webProyectoSlugPorID() (map[int64]string, error) {
	proyectos, err := webCargarProyectosPorAPI()
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(proyectos))
	for _, proyecto := range proyectos {
		if proyecto != nil {
			out[proyecto.ID] = proyecto.Slug
		}
	}
	return out, nil
}

const webTplGitGov = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "gitgov.title"}}</h2>
  <p style="color:#64748b">{{tr "gitgov.subtitle"}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  <form method="get" action="/git">
    <div class="grid2">
      <label>{{tr "gitgov.operational_state"}} <input name="estado" value="{{.Estado}}" placeholder="activa"></label>
      <label>{{tr "Agente"}} <input name="agente" value="{{.Agente}}" placeholder="codex2"></label>
    </div>
    <div class="grid2">
      <label>{{tr "gitgov.merge_project"}} <input name="proyecto" value="{{.Proyecto}}" placeholder="orquestador"></label>
      <label>{{tr "gitgov.merge_state"}} <input name="merge_estado" value="{{.MergeEstado}}" placeholder="pendiente"></label>
    </div>
    <button type="submit" class="btn-sm">{{tr "common.filter"}}</button>
  </form>
</section>

<section class="container grid">
  <article>
    <h3>{{tr "gitgov.worktrees_title"}} <small style="color:#94a3b8">{{len .Worktrees}}</small></h3>
    {{if .Worktrees}}
    <table>
      <thead><tr><th>ID</th><th>{{tr "Proyecto"}}</th><th>{{tr "Agente"}}</th><th>{{tr "common.name"}}</th><th>{{tr "common.branch"}}</th><th>{{tr "Estado"}}</th></tr></thead>
      <tbody>
        {{range .Worktrees}}
        <tr>
          <td><a href="/git/worktrees/{{.ID}}">{{.ID}}</a></td>
          <td>{{.ProyectoSlug}}</td>
          <td>{{.Agente}}</td>
          <td>{{.Nombre}}</td>
          <td><code>{{.Branch}}</code></td>
          <td>{{.Estado}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
    {{else}}
    <p>{{tr "gitgov.no_worktrees"}}</p>
    {{end}}
  </article>

  <article>
    <h3>{{tr "common.locks"}} <small style="color:#94a3b8">{{len .Locks}}</small></h3>
    {{if .Locks}}
    <table>
      <thead><tr><th>ID</th><th>{{tr "Agente"}}</th><th>{{tr "gitgov.scope"}}</th><th>{{tr "gitgov.key"}}</th><th>{{tr "common.branch"}}</th><th>{{tr "Estado"}}</th></tr></thead>
      <tbody>
        {{range .Locks}}
        <tr>
          <td><a href="/git/locks/{{.ID}}">{{.ID}}</a></td>
          <td>{{.Agente}}</td>
          <td>{{.ScopeType}}</td>
          <td>{{.ScopeKey}}</td>
          <td><code>{{.Branch}}</code></td>
          <td>{{.Estado}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
    {{else}}
    <p>{{tr "gitgov.no_locks"}}</p>
    {{end}}
  </article>
</section>

<section class="container">
  <article>
    <h3>{{tr "gitgov.requests_title"}}</h3>
      {{if .Merges}}
      <table>
        <thead><tr><th>ID</th><th>{{tr "Proyecto"}}</th><th>{{tr "gitgov.source"}}</th><th>{{tr "gitgov.target"}}</th><th>{{tr "Estado"}}</th><th>{{tr "gitgov.requested_by"}}</th><th>{{tr "gitgov.commit_source"}}</th><th>{{tr "gitgov.commit_merge"}}</th><th>{{tr "Nota"}}</th></tr></thead>
        <tbody>
          {{range .Merges}}
          <tr>
            <td>{{.ID}}</td>
            <td>{{.ProyectoSlug}}</td>
            <td><code>{{.SourceBranch}}</code></td>
            <td><code>{{.TargetBranch}}</code></td>
            <td>{{.Estado}}</td>
            <td>{{.RequestedBy}}</td>
            <td><code>{{orDash .CommitOrigen}}</code></td>
            <td><code>{{orDash .CommitMerge}}</code></td>
            <td>{{orDash .Notas}}</td>
          </tr>
          {{end}}
        </tbody>
      </table>
      {{else}}
      <p>{{tr "gitgov.no_merges"}}</p>
      {{end}}
      <form method="post" action="/git/merges">
        <div class="grid2">
          <label>{{tr "Proyecto"}} <input name="proyecto_slug" value="{{.Proyecto}}" required></label>
          <label>{{tr "gitgov.requested_by" }} <input name="requested_by" value="alberto" required></label>
        </div>
        <div class="grid2">
          <label>{{tr "gitgov.source_branch" }} <input name="source_branch" required></label>
          <label>{{tr "gitgov.target_branch" }} <input name="target_branch" value="main" required></label>
        </div>
        <div class="grid2">
          <label>{{tr "Estado"}}
            <select name="estado">
              <option value="pendiente">{{tr "pendiente"}}</option>
              <option value="validando">{{tr "validando"}}</option>
              <option value="aprobado">{{tr "aprobado"}}</option>
              <option value="rechazado">{{tr "rechazado"}}</option>
              <option value="ejecutando">{{tr "ejecutando"}}</option>
              <option value="fusionado">{{tr "fusionado"}}</option>
              <option value="fallido">{{tr "fallido"}}</option>
              <option value="cancelado">{{tr "cancelado"}}</option>
            </select>
          </label>
          <label>{{tr "gitgov.commit_source" }} <input name="commit_origen"></label>
        </div>
        <label>{{tr "gitgov.commit_merge" }} <input name="commit_merge"></label>
        <label>{{tr "gitgov.notes" }} <textarea name="notas"></textarea></label>
        <label>{{tr "gitgov.metadata_json" }} <textarea name="metadata_json" placeholder='{"riesgo":"medio"}'></textarea></label>
        <button type="submit">{{tr "gitgov.save_request"}}</button>
      </form>
  </article>
</section>
{{end}}`

const webTplGitWorktreeDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/git">← {{tr "gitgov.back"}}</a></p>
  <h2 style="margin:0">{{tr "gitgov.worktree_detail_title"}} #{{.Worktree.ID}}</h2>
  <table>
    <tbody>
      <tr><th>{{tr "Proyecto"}}</th><td>{{.Worktree.ProjectID}}</td></tr>
      <tr><th>{{tr "common.task"}}</th><td>{{pid .Worktree.TaskID}}</td></tr>
      <tr><th>{{tr "common.lock"}}</th><td>{{pid .Worktree.LockID}}</td></tr>
      <tr><th>{{tr "Agente"}}</th><td>{{.Worktree.Agent}}</td></tr>
      <tr><th>{{tr "common.name"}}</th><td>{{.Worktree.Name}}</td></tr>
      <tr><th>{{tr "gitgov.path"}}</th><td><code>{{.Worktree.Path}}</code></td></tr>
      <tr><th>{{tr "common.branch"}}</th><td><code>{{.Worktree.Branch}}</code></td></tr>
      <tr><th>{{tr "gitgov.base_ref"}}</th><td><code>{{.Worktree.BaseRef}}</code></td></tr>
      <tr><th>{{tr "Estado"}}</th><td>{{.Worktree.State}}</td></tr>
      <tr><th>{{tr "common.reason"}}</th><td>{{.Worktree.Reason}}</td></tr>
      <tr><th>{{tr "common.created_at"}}</th><td>{{.Worktree.CreatedAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>{{tr "common.updated_at"}}</th><td>{{.Worktree.UpdatedAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>{{tr "common.closed_at"}}</th><td>{{if .Worktree.ClosedAt}}{{.Worktree.ClosedAt.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
    </tbody>
  </table>
</section>
{{end}}`

const webTplGitLockDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/git">← {{tr "gitgov.back"}}</a></p>
  <h2 style="margin:0">{{tr "gitgov.lock_detail_title"}} #{{.Lock.ID}}</h2>
  <table>
    <tbody>
      <tr><th>{{tr "Proyecto"}}</th><td>{{pid .Lock.ProjectID}}</td></tr>
      <tr><th>{{tr "common.task"}}</th><td>{{pid .Lock.TaskID}}</td></tr>
      <tr><th>{{tr "Sesión"}}</th><td>{{pid .Lock.SessionID}}</td></tr>
      <tr><th>{{tr "Agente"}}</th><td>{{.Lock.Agent}}</td></tr>
      <tr><th>{{tr "gitgov.scope"}}</th><td>{{.Lock.ScopeType}}</td></tr>
      <tr><th>{{tr "gitgov.key"}}</th><td>{{.Lock.ScopeKey}}</td></tr>
      <tr><th>{{tr "gitgov.path"}}</th><td><code>{{.Lock.Path}}</code></td></tr>
      <tr><th>{{tr "common.branch"}}</th><td><code>{{.Lock.Branch}}</code></td></tr>
      <tr><th>{{tr "common.reason"}}</th><td>{{.Lock.Reason}}</td></tr>
      <tr><th>{{tr "gitgov.lease_token"}}</th><td><code>{{.Lock.LeaseToken}}</code></td></tr>
      <tr><th>{{tr "Estado"}}</th><td>{{.Lock.State}}</td></tr>
      <tr><th>{{tr "common.heartbeat"}}</th><td>{{if .Lock.HeartbeatAt}}{{.Lock.HeartbeatAt.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
      <tr><th>{{tr "gitgov.expires_at"}}</th><td>{{.Lock.ExpiresAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>{{tr "common.created_at"}}</th><td>{{.Lock.CreatedAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>{{tr "common.updated_at"}}</th><td>{{.Lock.UpdatedAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>{{tr "gitgov.released_at"}}</th><td>{{if .Lock.ReleasedAt}}{{.Lock.ReleasedAt.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
    </tbody>
  </table>
</section>
{{end}}`
