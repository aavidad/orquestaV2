/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"orquesta/db"
	"orquesta/gobernanzaapp"
	"orquesta/skillsapp"
)

var gobernanzaService = gobernanzaapp.NewService(db.GovernanceRepository{})

type webGobernanzaData struct {
	TipoAgente string
	Reglas     []*db.Regla
	Skills     []*db.Skill
	Workflows  []webWorkflowView
	Msg        string
	Err        string
}

type webWorkflowView struct {
	ID          int64
	TipoAgente  string
	Nombre      string
	Descripcion string
	Pasos       []string
	Activo      bool
}

func webHandlerGobernanza(w http.ResponseWriter, r *http.Request) {
	tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
	if tipoAgente == "" {
		tipoAgente = "programador"
	}
	itemsReglas, err := gobernanzaService.ListRules(tipoAgente, nil)
	if err != nil {
		webRender(w, webTplLayout+webTplGobernanza, webGobernanzaData{Err: err.Error(), TipoAgente: tipoAgente})
		return
	}
	itemsSkills, err := gobernanzaService.ListSkills(tipoAgente, nil)
	if err != nil {
		webRender(w, webTplLayout+webTplGobernanza, webGobernanzaData{Err: err.Error(), TipoAgente: tipoAgente})
		return
	}
	itemsWorkflows, err := gobernanzaService.ListWorkflows(tipoAgente, nil)
	if err != nil {
		webRender(w, webTplLayout+webTplGobernanza, webGobernanzaData{Err: err.Error(), TipoAgente: tipoAgente})
		return
	}
	webRender(w, webTplLayout+webTplGobernanza, webGobernanzaData{
		TipoAgente: tipoAgente,
		Reglas:     itemsReglas,
		Skills:     itemsSkills,
		Workflows:  workflowsToView(itemsWorkflows),
		Msg:        r.URL.Query().Get("ok"),
		Err:        r.URL.Query().Get("err"),
	})
}

func webRouterGobernanza(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/gobernanza", http.StatusSeeOther)
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "reglas" && r.Method == http.MethodPost:
		webHandlerGobernanzaReglaNueva(w, r)
	case len(parts) == 2 && parts[1] == "skills" && r.Method == http.MethodPost:
		webHandlerGobernanzaSkillNueva(w, r)
	case len(parts) == 3 && parts[1] == "skills" && parts[2] == "importar" && r.Method == http.MethodPost:
		webHandlerGobernanzaSkillImportar(w, r)
	case len(parts) == 2 && parts[1] == "workflows" && r.Method == http.MethodPost:
		webHandlerGobernanzaWorkflowNuevo(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webRouterAPIGobernanza(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "gobernanza" {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 3 && parts[2] == "reglas" && r.Method == http.MethodGet:
		webHandlerAPIGobernanzaReglas(w, r)
	case len(parts) == 3 && parts[2] == "reglas" && r.Method == http.MethodPost:
		webHandlerAPIGobernanzaReglaCrear(w, r)
	case len(parts) == 3 && parts[2] == "skills" && r.Method == http.MethodGet:
		webHandlerAPIGobernanzaSkills(w, r)
	case len(parts) == 3 && parts[2] == "skills" && r.Method == http.MethodPost:
		webHandlerAPIGobernanzaSkillCrear(w, r)
	case len(parts) == 3 && parts[2] == "workflows" && r.Method == http.MethodGet:
		webHandlerAPIGobernanzaWorkflows(w, r)
	case len(parts) == 3 && parts[2] == "workflows" && r.Method == http.MethodPost:
		webHandlerAPIGobernanzaWorkflowCrear(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webHandlerGobernanzaReglaNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if _, err := gobernanzaService.SaveRule(gobernanzaapp.SaveRuleInput{
		TipoAgente:  tipoAgente,
		Categoria:   r.FormValue("categoria"),
		Titulo:      r.FormValue("titulo"),
		Descripcion: r.FormValue("descripcion"),
		Activa:      parseBoolFormDefaultOn(r.FormValue("activa")),
	}); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape("Regla guardada"), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if _, err := gobernanzaService.SaveSkill(gobernanzaapp.SaveSkillInput{
		TipoAgente:  tipoAgente,
		Nombre:      r.FormValue("nombre"),
		Descripcion: r.FormValue("descripcion"),
		CuandoUsar:  r.FormValue("cuando_usar"),
		Activa:      parseBoolFormDefaultOn(r.FormValue("activa")),
	}); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape("Skill guardada"), http.StatusSeeOther)
}

func webHandlerGobernanzaSkillImportar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	sourceURL := strings.TrimSpace(r.FormValue("url"))
	repo := strings.TrimSpace(r.FormValue("repo"))
	skill := strings.TrimSpace(r.FormValue("skill"))
	result, err := skillsImportService.ImportFromWeb(context.Background(), skillsapp.ImportInput{
		Actor:      "alberto",
		TipoAgente: tipoAgente,
		URL:        sourceURL,
		Repo:       repo,
		Skill:      skill,
	})
	if err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	msg := "Skill importada"
	if result.Existente {
		msg = "Skill ya existía en el catálogo"
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape(msg), http.StatusSeeOther)
}

func webHandlerGobernanzaWorkflowNuevo(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tipoAgente := strings.TrimSpace(r.FormValue("tipo_agente"))
	if _, err := gobernanzaService.SaveWorkflow(gobernanzaapp.SaveWorkflowInput{
		TipoAgente:  tipoAgente,
		Nombre:      r.FormValue("nombre"),
		Descripcion: r.FormValue("descripcion"),
		Pasos:       splitPasos(r.FormValue("pasos")),
		Activo:      parseBoolFormDefaultOn(r.FormValue("activo")),
	}); err != nil {
		http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/gobernanza?tipo_agente="+url.QueryEscape(tipoAgente)+"&ok="+url.QueryEscape("Workflow guardado"), http.StatusSeeOther)
}

func webHandlerAPIGobernanzaReglas(w http.ResponseWriter, r *http.Request) {
	tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
	items, err := gobernanzaService.ListRules(tipoAgente, parseOptionalBool(r.URL.Query().Get("activa")))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIGobernanzaReglaCrear(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		TipoAgente  string `json:"tipo_agente"`
		Categoria   string `json:"categoria"`
		Titulo      string `json:"titulo"`
		Descripcion string `json:"descripcion"`
		Activa      bool   `json:"activa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	id, err := gobernanzaService.SaveRule(gobernanzaapp.SaveRuleInput(payload))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func webHandlerAPIGobernanzaSkills(w http.ResponseWriter, r *http.Request) {
	tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
	items, err := gobernanzaService.ListSkills(tipoAgente, parseOptionalBool(r.URL.Query().Get("activa")))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIGobernanzaSkillCrear(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		TipoAgente  string `json:"tipo_agente"`
		Nombre      string `json:"nombre"`
		Descripcion string `json:"descripcion"`
		CuandoUsar  string `json:"cuando_usar"`
		Activa      bool   `json:"activa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	id, err := gobernanzaService.SaveSkill(gobernanzaapp.SaveSkillInput(payload))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func webHandlerAPIGobernanzaWorkflows(w http.ResponseWriter, r *http.Request) {
	tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
	items, err := gobernanzaService.ListWorkflows(tipoAgente, parseOptionalBool(r.URL.Query().Get("activo")))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIGobernanzaWorkflowCrear(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		TipoAgente  string   `json:"tipo_agente"`
		Nombre      string   `json:"nombre"`
		Descripcion string   `json:"descripcion"`
		Pasos       []string `json:"pasos"`
		Activo      bool     `json:"activo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	id, err := gobernanzaService.SaveWorkflow(gobernanzaapp.SaveWorkflowInput(payload))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func parseBoolFormDefaultOn(raw string) bool {
	raw = strings.TrimSpace(raw)
	return raw == "" || raw == "on" || raw == "1" || strings.EqualFold(raw, "true")
}

func parseOptionalBool(raw string) *bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	value := raw == "1" || strings.EqualFold(raw, "true")
	return &value
}

func splitPasos(raw string) []string {
	lines := strings.Split(raw, "\n")
	var pasos []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pasos = append(pasos, line)
	}
	return pasos
}

func workflowsToView(items []*db.Workflow) []webWorkflowView {
	var out []webWorkflowView
	for _, item := range items {
		var pasos []string
		_ = json.Unmarshal([]byte(item.Pasos), &pasos)
		out = append(out, webWorkflowView{
			ID:          item.ID,
			TipoAgente:  item.TipoAgente,
			Nombre:      item.Nombre,
			Descripcion: item.Descripcion,
			Pasos:       pasos,
			Activo:      item.Activo,
		})
	}
	return out
}

const webTplGobernanza = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">Gobernanza operativa</h2>
  <p style="color:#64748b">Gestión de reglas, skills y workflows desde la app.</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  <form method="get" action="/gobernanza">
    <label>Tipo de agente
      <select name="tipo_agente">
        <option value="programador" {{if eqStr .TipoAgente "programador"}}selected{{end}}>programador</option>
        <option value="documentador" {{if eqStr .TipoAgente "documentador"}}selected{{end}}>documentador</option>
        <option value="admin" {{if eqStr .TipoAgente "admin"}}selected{{end}}>admin</option>
      </select>
    </label>
    <button type="submit" class="btn-sm">Filtrar</button>
  </form>
</section>

<section class="container grid">
  <article>
    <h3>Reglas</h3>
    {{range .Reglas}}
      <details style="margin-bottom:.6rem">
        <summary><strong>{{.Titulo}}</strong> · {{.Categoria}}</summary>
        <p>{{.Descripcion}}</p>
      </details>
    {{else}}
      <p>No hay reglas.</p>
    {{end}}
    <form method="post" action="/gobernanza/reglas">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <label>Categoría <input name="categoria" required></label>
      <label>Título <input name="titulo" required></label>
      <label>Descripción <textarea name="descripcion" required></textarea></label>
      <label><input type="checkbox" name="activa" checked> Activa</label>
      <button type="submit">Guardar regla</button>
    </form>
  </article>

  <article>
    <h3>Skills</h3>
    {{range .Skills}}
      <details style="margin-bottom:.6rem">
        <summary><strong>{{.Nombre}}</strong></summary>
        <p>{{.Descripcion}}</p>
        {{if .CuandoUsar}}<p><strong>Cuándo usar:</strong> {{.CuandoUsar}}</p>{{end}}
        <p style="display:flex;gap:.5rem;flex-wrap:wrap">
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #cbd5e1;border-radius:999px">{{tr "governance.skills.source"}}: {{.Origen}}</span>
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #cbd5e1;border-radius:999px">{{tr "governance.skills.risk"}}: {{.NivelRiesgo}}</span>
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #cbd5e1;border-radius:999px">
            {{tr "governance.skills.state"}}:
            {{if .Activa}}{{tr "common.active"}}{{else}}{{tr "governance.skills.inactive"}}{{end}}
          </span>
          {{if .RequiereAprobacion}}
          <span style="display:inline-block;padding:.2rem .45rem;border:1px solid #f59e0b;border-radius:999px;background:#fffbeb;color:#92400e">{{tr "governance.skills.pending_approval"}}</span>
          {{end}}
        </p>
      </details>
    {{else}}
      <p>No hay skills.</p>
    {{end}}
    <form method="post" action="/gobernanza/skills">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <label>Nombre <input name="nombre" required></label>
      <label>Descripción <textarea name="descripcion" required></textarea></label>
      <label>Cuándo usar <textarea name="cuando_usar"></textarea></label>
      <label><input type="checkbox" name="activa" checked> Activa</label>
      <button type="submit">Guardar skill</button>
    </form>
    <form method="post" action="/gobernanza/skills/importar" style="margin-top:1rem;border-top:1px solid #e2e8f0;padding-top:1rem">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <h4 style="margin:.2rem 0">{{tr "governance.skills.import.title"}}</h4>
      <p style="color:#64748b">{{tr "governance.skills.import.help"}}</p>
      <label>{{tr "governance.skills.import.url"}} <input name="url" placeholder="https://skills.sh/openai/skills/openai-docs"></label>
      <label>{{tr "governance.skills.import.repo"}} <input name="repo" placeholder="openai/skills"></label>
      <label>{{tr "governance.skills.import.skill"}} <input name="skill" placeholder="openai-docs"></label>
      <button type="submit">{{tr "governance.skills.import.button"}}</button>
    </form>
  </article>
</section>

<section class="container">
  <article>
    <h3>Workflows</h3>
    {{range .Workflows}}
      <details style="margin-bottom:.6rem">
        <summary><strong>{{.Nombre}}</strong></summary>
        <p>{{.Descripcion}}</p>
        {{if .Pasos}}
        <ol>
          {{range .Pasos}}<li>{{.}}</li>{{end}}
        </ol>
        {{end}}
      </details>
    {{else}}
      <p>No hay workflows.</p>
    {{end}}
    <form method="post" action="/gobernanza/workflows">
      <input type="hidden" name="tipo_agente" value="{{.TipoAgente}}">
      <label>Nombre <input name="nombre" required></label>
      <label>Descripción <textarea name="descripcion" required></textarea></label>
      <label>Pasos (uno por línea) <textarea name="pasos" required></textarea></label>
      <label><input type="checkbox" name="activo" checked> Activo</label>
      <button type="submit">Guardar workflow</button>
    </form>
  </article>
</section>
{{end}}`
