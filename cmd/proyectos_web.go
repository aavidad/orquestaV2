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
	"orquesta/projectmem"
)

var projectMemoryService = projectmem.NewService(db.ProjectMemoryRepository{})

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
	Proyecto   *db.Proyecto
	Votaciones []*db.HistorialVotacionProyecto
	Decisiones []*db.DecisionProyecto
	Documentos []*db.DocumentoExterno
	Msg        string
	Err        string
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
	case len(parts) == 3 && parts[2] == "decisiones" && r.Method == http.MethodPost:
		webHandlerProyectoDecisionNueva(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "documentacion" && r.Method == http.MethodPost:
		webHandlerProyectoDocumentoNuevo(w, r, parts[1])
	default:
		http.NotFound(w, r)
	}
}

func webRouterAPIProyectos(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "api" || parts[1] != "proyectos" {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerAPIProyectos(w, r)
	case len(parts) == 3 && r.Method == http.MethodGet:
		webHandlerAPIProyectoDetalle(w, r, parts[2])
	case len(parts) == 4 && parts[3] == "votaciones" && r.Method == http.MethodGet:
		webHandlerAPIProyectoVotaciones(w, r, parts[2])
	case len(parts) == 4 && parts[3] == "decisiones" && r.Method == http.MethodGet:
		webHandlerAPIProyectoDecisiones(w, r, parts[2])
	case len(parts) == 4 && parts[3] == "decisiones" && r.Method == http.MethodPost:
		webHandlerAPIProyectoDecisionCrear(w, r, parts[2])
	case len(parts) == 4 && parts[3] == "documentacion" && r.Method == http.MethodGet:
		webHandlerAPIProyectoDocumentacion(w, r, parts[2])
	case len(parts) == 4 && parts[3] == "documentacion" && r.Method == http.MethodPost:
		webHandlerAPIProyectoDocumentoCrear(w, r, parts[2])
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
	items, err := projectMemoryService.ListProjects(activaPtr)
	if err != nil {
		webRender(w, webTplLayout+webTplProyectos, webProyectosData{
			Err: err.Error(),
		})
		return
	}
	var proyectos []webProyectoResumen
	for _, item := range items {
		proyectos = append(proyectos, webProyectoResumen{
			Slug:    item.Slug,
			Nombre:  item.Nombre,
			Tipo:    item.Tipo,
			RutaAbs: item.RutaAbs,
			Activo:  item.Activo,
		})
	}
	webRender(w, webTplLayout+webTplProyectos, webProyectosData{
		Proyectos: proyectos,
		Msg:       r.URL.Query().Get("ok"),
		Err:       r.URL.Query().Get("err"),
	})
}

func webHandlerProyectoDetalle(w http.ResponseWriter, r *http.Request, slug string) {
	overview, err := projectMemoryService.Overview(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, webTplLayout+webTplProyectoDetalle, webProyectoDetalleData{
		Proyecto:   overview.Proyecto,
		Votaciones: overview.Votaciones,
		Decisiones: overview.Decisiones,
		Documentos: overview.Documentos,
		Msg:        r.URL.Query().Get("ok"),
		Err:        r.URL.Query().Get("err"),
	})
}

func webHandlerProyectoDecisionNueva(w http.ResponseWriter, r *http.Request, slug string) {
	_ = r.ParseForm()
	_, err := projectMemoryService.CreateDecision(projectmem.CreateDecisionInput{
		ProyectoSlug: slug,
		Categoria:    r.FormValue("categoria"),
		Titulo:       r.FormValue("titulo"),
		Solucion:     r.FormValue("solucion"),
		Motivo:       r.FormValue("motivo"),
		Alternativas: r.FormValue("alternativas"),
		Impacto:      r.FormValue("impacto"),
		Estado:       r.FormValue("estado"),
		PropuestaID:  parseOptionalInt64(r.FormValue("propuesta_id")),
		TareaID:      parseOptionalInt64(r.FormValue("tarea_id")),
	})
	if err != nil {
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/proyectos/"+slug+"?ok="+url.QueryEscape("Decisión guardada"), http.StatusSeeOther)
}

func webHandlerProyectoDocumentoNuevo(w http.ResponseWriter, r *http.Request, slug string) {
	_ = r.ParseForm()
	_, err := projectMemoryService.CreateExternalDoc(projectmem.CreateExternalDocInput{
		ProyectoSlug:  slug,
		TipoDocumento: r.FormValue("tipo_documento"),
		Titulo:        r.FormValue("titulo"),
		RutaRef:       r.FormValue("ruta_ref"),
		Resumen:       r.FormValue("resumen"),
		Estado:        r.FormValue("estado"),
		Fuente:        r.FormValue("fuente"),
		PropuestaID:   parseOptionalInt64(r.FormValue("propuesta_id")),
		TareaID:       parseOptionalInt64(r.FormValue("tarea_id")),
	})
	if err != nil {
		http.Redirect(w, r, "/proyectos/"+slug+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/proyectos/"+slug+"?ok="+url.QueryEscape("Documento guardado"), http.StatusSeeOther)
}

func webHandlerAPIProyectos(w http.ResponseWriter, r *http.Request) {
	var activaPtr *bool
	if raw := strings.TrimSpace(r.URL.Query().Get("activa")); raw != "" {
		value := raw == "1" || strings.EqualFold(raw, "true")
		activaPtr = &value
	}
	items, err := projectMemoryService.ListProjects(activaPtr)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIProyectoDetalle(w http.ResponseWriter, r *http.Request, slug string) {
	overview, err := projectMemoryService.Overview(slug)
	if err != nil {
		webWriteJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, overview)
}

func webHandlerAPIProyectoVotaciones(w http.ResponseWriter, r *http.Request, slug string) {
	items, err := projectMemoryService.ListVoteHistory(slug)
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIProyectoDecisiones(w http.ResponseWriter, r *http.Request, slug string) {
	items, err := projectMemoryService.ListDecisions(slug)
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIProyectoDecisionCrear(w http.ResponseWriter, r *http.Request, slug string) {
	var payload struct {
		Categoria    string `json:"categoria"`
		Titulo       string `json:"titulo"`
		Solucion     string `json:"solucion"`
		Motivo       string `json:"motivo"`
		Alternativas string `json:"alternativas"`
		Impacto      string `json:"impacto"`
		Estado       string `json:"estado"`
		PropuestaID  *int64 `json:"propuesta_id"`
		TareaID      *int64 `json:"tarea_id"`
		MetadataJSON string `json:"metadata_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	id, err := projectMemoryService.CreateDecision(projectmem.CreateDecisionInput{
		ProyectoSlug: slug,
		Categoria:    payload.Categoria,
		Titulo:       payload.Titulo,
		Solucion:     payload.Solucion,
		Motivo:       payload.Motivo,
		Alternativas: payload.Alternativas,
		Impacto:      payload.Impacto,
		Estado:       payload.Estado,
		PropuestaID:  payload.PropuestaID,
		TareaID:      payload.TareaID,
		MetadataJSON: payload.MetadataJSON,
	})
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func webHandlerAPIProyectoDocumentacion(w http.ResponseWriter, r *http.Request, slug string) {
	items, err := projectMemoryService.ListExternalDocs(slug)
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIProyectoDocumentoCrear(w http.ResponseWriter, r *http.Request, slug string) {
	var payload struct {
		TipoDocumento string `json:"tipo_documento"`
		Titulo        string `json:"titulo"`
		RutaRef       string `json:"ruta_ref"`
		Resumen       string `json:"resumen"`
		Estado        string `json:"estado"`
		Fuente        string `json:"fuente"`
		PropuestaID   *int64 `json:"propuesta_id"`
		TareaID       *int64 `json:"tarea_id"`
		MetadataJSON  string `json:"metadata_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	id, err := projectMemoryService.CreateExternalDoc(projectmem.CreateExternalDocInput{
		ProyectoSlug:  slug,
		TipoDocumento: payload.TipoDocumento,
		Titulo:        payload.Titulo,
		RutaRef:       payload.RutaRef,
		Resumen:       payload.Resumen,
		Estado:        payload.Estado,
		Fuente:        payload.Fuente,
		PropuestaID:   payload.PropuestaID,
		TareaID:       payload.TareaID,
		MetadataJSON:  payload.MetadataJSON,
	})
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
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

const webTplProyectos = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">Proyectos <small style="font-size:.5em;color:#94a3b8">{{len .Proyectos}}</small></h2>
  <p style="color:#64748b">Vista de memoria y trazabilidad por proyecto.</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
  {{if .Proyectos}}
  <table>
    <thead><tr><th>Proyecto</th><th>Tipo</th><th>Ruta</th><th>Estado</th></tr></thead>
    <tbody>
      {{range .Proyectos}}
      <tr>
        <td><a href="/proyectos/{{.Slug}}"><strong>{{.Nombre}}</strong></a><br><small>{{.Slug}}</small></td>
        <td>{{.Tipo}}</td>
        <td><code>{{.RutaAbs}}</code></td>
        <td>{{if .Activo}}activo{{else}}inactivo{{end}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
  {{else}}
  <article>No hay proyectos registrados.</article>
  {{end}}
</section>
{{end}}`

const webTplProyectoDetalle = `{{define "content"}}
<section class="container">
  <p style="margin:0 0 .35rem 0"><a href="/proyectos">← Volver a proyectos</a></p>
  <h2 style="margin:0">{{.Proyecto.Nombre}} <small style="font-size:.5em;color:#94a3b8">{{.Proyecto.Slug}}</small></h2>
  <p style="color:#64748b"><code>{{.Proyecto.RutaAbs}}</code> · tipo {{.Proyecto.Tipo}}</p>
  {{if .Msg}}<article style="background:#ecfccb;border:1px solid #84cc16;padding:.75rem">{{.Msg}}</article>{{end}}
  {{if .Err}}<article style="background:#fee2e2;border:1px solid #ef4444;padding:.75rem">{{.Err}}</article>{{end}}
</section>

<section class="container grid">
  <article>
    <h3>Decisiones</h3>
    {{if .Decisiones}}
      {{range .Decisiones}}
      <details open style="margin-bottom:.75rem">
        <summary><strong>{{.Titulo}}</strong> · {{.Estado}}</summary>
        <p><strong>Categoría:</strong> {{.Categoria}}</p>
        <p><strong>Solución:</strong> {{.Solucion}}</p>
        {{if .Motivo}}<p><strong>Motivo:</strong> {{.Motivo}}</p>{{end}}
        {{if .Alternativas}}<p><strong>Alternativas:</strong> {{.Alternativas}}</p>{{end}}
        {{if .Impacto}}<p><strong>Impacto:</strong> {{.Impacto}}</p>{{end}}
      </details>
      {{end}}
    {{else}}
      <p>No hay decisiones registradas.</p>
    {{end}}
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/decisiones">
      <label>Título <input name="titulo" required></label>
      <label>Categoría <input name="categoria" value="general"></label>
      <label>Solución <textarea name="solucion" required></textarea></label>
      <label>Motivo <textarea name="motivo"></textarea></label>
      <label>Alternativas <textarea name="alternativas"></textarea></label>
      <label>Impacto <textarea name="impacto"></textarea></label>
      <label>Estado
        <select name="estado">
          <option value="vigente">vigente</option>
          <option value="experimental">experimental</option>
          <option value="reemplazada">reemplazada</option>
          <option value="descartada">descartada</option>
          <option value="archivada">archivada</option>
        </select>
      </label>
      <label>Propuesta relacionada <input name="propuesta_id" inputmode="numeric"></label>
      <label>Tarea relacionada <input name="tarea_id" inputmode="numeric"></label>
      <button type="submit">Guardar decisión</button>
    </form>
  </article>

  <article>
    <h3>Documentación externa</h3>
    {{if .Documentos}}
      {{range .Documentos}}
      <details open style="margin-bottom:.75rem">
        <summary><strong>{{.Titulo}}</strong> · {{.TipoDocumento}}</summary>
        <p><strong>Ruta:</strong> <code>{{.RutaRef}}</code></p>
        <p><strong>Resumen:</strong> {{.Resumen}}</p>
        <p><strong>Estado:</strong> {{.Estado}} · <strong>Fuente:</strong> {{.Fuente}}</p>
      </details>
      {{end}}
    {{else}}
      <p>No hay documentación externa registrada.</p>
    {{end}}
    <form method="post" action="/proyectos/{{.Proyecto.Slug}}/documentacion">
      <label>Título <input name="titulo" required></label>
      <label>Tipo de documento <input name="tipo_documento" value="markdown"></label>
      <label>Ruta o referencia <input name="ruta_ref" required></label>
      <label>Resumen <textarea name="resumen" required></textarea></label>
      <label>Estado
        <select name="estado">
          <option value="vigente">vigente</option>
          <option value="borrador">borrador</option>
          <option value="archivado">archivado</option>
        </select>
      </label>
      <label>Fuente
        <select name="fuente">
          <option value="manual">manual</option>
          <option value="propuesta">propuesta</option>
          <option value="tarea">tarea</option>
          <option value="externo">externo</option>
        </select>
      </label>
      <label>Propuesta relacionada <input name="propuesta_id" inputmode="numeric"></label>
      <label>Tarea relacionada <input name="tarea_id" inputmode="numeric"></label>
      <button type="submit">Guardar documento</button>
    </form>
  </article>
</section>

<section class="container">
  <article>
    <h3>Historial de votaciones</h3>
    {{if .Votaciones}}
      {{range .Votaciones}}
      <details style="margin-bottom:.75rem">
        <summary><strong>{{.Codigo}}</strong> · {{.Titulo}} · {{.Estado}}</summary>
        <p>{{.Tipo}} · propuesta por {{.PropuestoPor}}</p>
        <p>✓ {{.Acuerdo}} · ✗ {{.Desacuerdo}} · ～ {{.Abstencion}} · ⏳ {{.Pendiente}}</p>
        {{if .Votos}}
        <table>
          <thead><tr><th>Agente</th><th>Posición</th><th>Comentario</th></tr></thead>
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
      <p>No hay propuestas asociadas a este proyecto.</p>
    {{end}}
  </article>
</section>
{{end}}`
