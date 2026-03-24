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

	"orquesta/db"
	"orquesta/gitgov"
)

var gitService = gitgov.NewService(gitgov.Repository{})

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

func webRouterGitGov(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/git", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "merges" && r.Method == http.MethodPost:
		webHandlerGitMergeNueva(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webHandlerGitGov(w http.ResponseWriter, r *http.Request) {
	estado := strings.TrimSpace(r.URL.Query().Get("estado"))
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	mergeEstado := strings.TrimSpace(r.URL.Query().Get("merge_estado"))

	worktrees, err := gitService.ListWorktrees(estado, agente)
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
	locks, err := gitService.ListLocks(estado, agente)
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
	id, err := gitService.SaveRequest(gitgov.SaveMergeRequestInput{
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

func webHandlerAPIWorktrees(w http.ResponseWriter, r *http.Request) {
	items, err := gitService.ListWorktrees(
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
	items, err := gitService.ListLocks(
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
		id, err := gitService.SaveRequest(gitgov.SaveMergeRequestInput{
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
	return gitService.ListRequests(proyecto, estado)
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
          <td>{{.ID}}</td>
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
          <td>{{.ID}}</td>
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
