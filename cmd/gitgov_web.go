/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitgobernanza"
)

var gitGobernanzaService = gitgobernanza.NewService(gitgobernanza.Repository{})

type webGitGovData struct {
	Estado      string
	Agente      string
	Proyecto    string
	MergeEstado string
	Worktrees   []*db.Worktree
	Locks       []*db.Lock
	Merges      []*db.GitMerge
	Msg         string
	Err         string
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

	worktrees, err := gitGobernanzaService.ListWorktrees(estado, agente)
	if err != nil {
		webRender(w, webTplLayout+webTplGitGov, webGitGovData{
			Estado:      estado,
			Agente:      agente,
			Proyecto:    proyecto,
			MergeEstado: mergeEstado,
			Err:         err.Error(),
		})
		return
	}
	locks, err := gitGobernanzaService.ListLocks(estado, agente)
	if err != nil {
		webRender(w, webTplLayout+webTplGitGov, webGitGovData{
			Estado:      estado,
			Agente:      agente,
			Proyecto:    proyecto,
			MergeEstado: mergeEstado,
			Worktrees:   worktrees,
			Err:         err.Error(),
		})
		return
	}

	merges, mergeErr := listGitMerges(proyecto, mergeEstado)
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
	webRender(w, webTplLayout+webTplGitGov, data)
}

func webHandlerGitMergeNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, err := gitGobernanzaService.SaveRequest(gitgobernanza.SaveMergeRequestInput{
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
	http.Redirect(w, r, "/git?ok="+url.QueryEscape("Solicitud de merge guardada #"+strconv.FormatInt(id, 10)), http.StatusSeeOther)
}

func webHandlerGitWorktreeDetalle(w http.ResponseWriter, r *http.Request, id int64) {
	worktree, err := (db.SQLiteWorktreeRepository{}).GetByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, webTplLayout+webTplGitWorktreeDetalle, webGitWorktreeDetalleData{Worktree: worktree})
}

func webHandlerGitLockDetalle(w http.ResponseWriter, r *http.Request, id int64) {
	lock, err := (db.SQLiteLockRepository{}).GetByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, webTplLayout+webTplGitLockDetalle, webGitLockDetalleData{Lock: lock})
}

func webHandlerAPIWorktrees(w http.ResponseWriter, r *http.Request) {
	items, err := gitGobernanzaService.ListWorktrees(
		strings.TrimSpace(r.URL.Query().Get("estado")),
		strings.TrimSpace(r.URL.Query().Get("agente")),
	)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPILocks(w http.ResponseWriter, r *http.Request) {
	items, err := gitGobernanzaService.ListLocks(
		strings.TrimSpace(r.URL.Query().Get("estado")),
		strings.TrimSpace(r.URL.Query().Get("agente")),
	)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIMerges(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := listGitMerges(
			strings.TrimSpace(r.URL.Query().Get("proyecto")),
			strings.TrimSpace(r.URL.Query().Get("estado")),
		)
		if err != nil {
			webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var payload struct {
			ProyectoSlug string `json:"proyecto_slug"`
			SourceBranch string `json:"source_branch"`
			TargetBranch string `json:"target_branch"`
			RequestedBy  string `json:"requested_by"`
			Estado       string `json:"estado"`
			CommitOrigen string `json:"commit_origen"`
			CommitMerge  string `json:"commit_merge"`
			Notas        string `json:"notas"`
			MetadataJSON string `json:"metadata_json"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
			return
		}
		id, err := gitGobernanzaService.SaveRequest(gitgobernanza.SaveMergeRequestInput{
			ProyectoSlug: payload.ProyectoSlug,
			SourceBranch: payload.SourceBranch,
			TargetBranch: payload.TargetBranch,
			RequestedBy:  payload.RequestedBy,
			Estado:       payload.Estado,
			CommitOrigen: payload.CommitOrigen,
			CommitMerge:  payload.CommitMerge,
			Notas:        payload.Notas,
			MetadataJSON: payload.MetadataJSON,
		})
		if err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		http.NotFound(w, r)
	}
}

func listGitMerges(proyecto, estado string) ([]*db.GitMerge, error) {
	return gitGobernanzaService.ListRequests(proyecto, estado)
}

func webWriteJSON(w http.ResponseWriter, status int, data any) {
	apiWriteJSON(w, status, data)
}

const webTplGitGov = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">Gobierno Git</h2>
  <p style="color:#64748b">Visión operativa de worktrees, locks y solicitudes de merge orquestado.</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  <form method="get" action="/git">
    <div class="grid2">
      <label>Estado operativo <input name="estado" value="{{.Estado}}" placeholder="activa"></label>
      <label>Agente <input name="agente" value="{{.Agente}}" placeholder="codex2"></label>
    </div>
    <div class="grid2">
      <label>Proyecto merge <input name="proyecto" value="{{.Proyecto}}" placeholder="orquestador"></label>
      <label>Estado merge <input name="merge_estado" value="{{.MergeEstado}}" placeholder="pendiente"></label>
    </div>
    <button type="submit" class="btn-sm">Filtrar</button>
  </form>
</section>

<section class="container grid">
  <article>
    <h3>Worktrees <small style="color:#94a3b8">{{len .Worktrees}}</small></h3>
    {{if .Worktrees}}
    <table>
      <thead><tr><th>ID</th><th>Proyecto</th><th>Agente</th><th>Nombre</th><th>Branch</th><th>Estado</th></tr></thead>
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
    <p>No hay worktrees para el filtro actual.</p>
    {{end}}
  </article>

  <article>
    <h3>Locks <small style="color:#94a3b8">{{len .Locks}}</small></h3>
    {{if .Locks}}
    <table>
      <thead><tr><th>ID</th><th>Agente</th><th>Scope</th><th>Key</th><th>Branch</th><th>Estado</th></tr></thead>
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
    <p>No hay locks para el filtro actual.</p>
    {{end}}
  </article>
</section>

<section class="container">
  <article>
    <h3>Solicitudes de merge orquestado</h3>
      {{if .Merges}}
      <table>
        <thead><tr><th>ID</th><th>Proyecto</th><th>Origen</th><th>Destino</th><th>Estado</th><th>Solicitado por</th></tr></thead>
        <tbody>
          {{range .Merges}}
          <tr>
            <td>{{.ID}}</td>
            <td>{{.ProyectoSlug}}</td>
            <td><code>{{.SourceBranch}}</code></td>
            <td><code>{{.TargetBranch}}</code></td>
            <td>{{.Estado}}</td>
            <td>{{.RequestedBy}}</td>
          </tr>
          {{end}}
        </tbody>
      </table>
      {{else}}
      <p>No hay solicitudes de merge para el filtro actual.</p>
      {{end}}
      <form method="post" action="/git/merges">
        <div class="grid2">
          <label>Proyecto <input name="proyecto_slug" value="{{.Proyecto}}" required></label>
          <label>Solicitado por <input name="requested_by" value="alberto" required></label>
        </div>
        <div class="grid2">
          <label>Branch origen <input name="source_branch" required></label>
          <label>Branch destino <input name="target_branch" value="main" required></label>
        </div>
        <div class="grid2">
          <label>Estado
            <select name="estado">
              <option value="pendiente">pendiente</option>
              <option value="validando">validando</option>
              <option value="aprobado">aprobado</option>
              <option value="rechazado">rechazado</option>
              <option value="ejecutando">ejecutando</option>
              <option value="fusionado">fusionado</option>
              <option value="fallido">fallido</option>
              <option value="cancelado">cancelado</option>
            </select>
          </label>
          <label>Commit origen <input name="commit_origen"></label>
        </div>
        <label>Commit merge <input name="commit_merge"></label>
        <label>Notas <textarea name="notas"></textarea></label>
        <label>Metadata JSON <textarea name="metadata_json" placeholder='{"riesgo":"medio"}'></textarea></label>
        <button type="submit">Guardar solicitud</button>
      </form>
  </article>
</section>
{{end}}`

const webTplGitWorktreeDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/git">← Volver a GitGov</a></p>
  <h2 style="margin:0">Worktree #{{.Worktree.ID}}</h2>
  <table>
    <tbody>
      <tr><th>Proyecto</th><td>{{.Worktree.ProjectID}}</td></tr>
      <tr><th>Tarea</th><td>{{pid .Worktree.TaskID}}</td></tr>
      <tr><th>Lock</th><td>{{pid .Worktree.LockID}}</td></tr>
      <tr><th>Agente</th><td>{{.Worktree.Agent}}</td></tr>
      <tr><th>Nombre</th><td>{{.Worktree.Name}}</td></tr>
      <tr><th>Ruta</th><td><code>{{.Worktree.Path}}</code></td></tr>
      <tr><th>Branch</th><td><code>{{.Worktree.Branch}}</code></td></tr>
      <tr><th>Base ref</th><td><code>{{.Worktree.BaseRef}}</code></td></tr>
      <tr><th>Estado</th><td>{{.Worktree.State}}</td></tr>
      <tr><th>Motivo</th><td>{{.Worktree.Reason}}</td></tr>
      <tr><th>Creado</th><td>{{.Worktree.CreatedAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>Actualizado</th><td>{{.Worktree.UpdatedAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>Cerrado</th><td>{{if .Worktree.ClosedAt}}{{.Worktree.ClosedAt.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
    </tbody>
  </table>
</section>
{{end}}`

const webTplGitLockDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/git">← Volver a GitGov</a></p>
  <h2 style="margin:0">Lock #{{.Lock.ID}}</h2>
  <table>
    <tbody>
      <tr><th>Proyecto</th><td>{{pid .Lock.ProjectID}}</td></tr>
      <tr><th>Tarea</th><td>{{pid .Lock.TaskID}}</td></tr>
      <tr><th>Sesión</th><td>{{pid .Lock.SessionID}}</td></tr>
      <tr><th>Agente</th><td>{{.Lock.Agent}}</td></tr>
      <tr><th>Scope</th><td>{{.Lock.ScopeType}}</td></tr>
      <tr><th>Key</th><td>{{.Lock.ScopeKey}}</td></tr>
      <tr><th>Ruta</th><td><code>{{.Lock.Path}}</code></td></tr>
      <tr><th>Branch</th><td><code>{{.Lock.Branch}}</code></td></tr>
      <tr><th>Motivo</th><td>{{.Lock.Reason}}</td></tr>
      <tr><th>Lease token</th><td><code>{{.Lock.LeaseToken}}</code></td></tr>
      <tr><th>Estado</th><td>{{.Lock.State}}</td></tr>
      <tr><th>Heartbeat</th><td>{{if .Lock.HeartbeatAt}}{{.Lock.HeartbeatAt.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
      <tr><th>Expira</th><td>{{.Lock.ExpiresAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>Creado</th><td>{{.Lock.CreatedAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>Actualizado</th><td>{{.Lock.UpdatedAt.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>Liberado</th><td>{{if .Lock.ReleasedAt}}{{.Lock.ReleasedAt.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
    </tbody>
  </table>
</section>
{{end}}`
