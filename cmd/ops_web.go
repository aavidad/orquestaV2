/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"orquesta/db"
	"orquesta/operacionesapp"
)

var operacionesService = operacionesapp.NewService(db.OpsViewRepository{})

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
	items, err := operacionesService.ListAssignments(estado, agente)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	webRender(w, webTplLayout+webTplAsignaciones, webAsignacionesData{
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

	var filtro db.FiltroSesionesInspeccion
	if agente != "" {
		filtro.Agente = &agente
	}
	if proyectoRef != "" {
		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		filtro.ProyectoID = &proyecto.ID
	}
	if estado != "" {
		filtro.Estado = &estado
	}
	if activaRaw != "" {
		activa, err := parseBoolFiltro(activaRaw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		filtro.Activa = &activa
	}

	items, err := db.ListarSesionesInspeccion(filtro)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	webRender(w, webTplLayout+webTplSesiones, webSesionesData{
		Sesiones: items,
		Agente:   agente,
		Proyecto: proyectoRef,
		Estado:   estado,
		Activa:   activaRaw,
	})
}

func webHandlerSesionDetalle(w http.ResponseWriter, r *http.Request, id int64) {
	sesion, err := db.GetSesionInspeccionByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	webRender(w, webTplLayout+webTplSesionDetalle, webSesionDetalleData{Sesion: sesion})
}

func webHandlerAPIAgentesLista(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var payload struct {
			Nombre string `json:"nombre"`
			Rol    string `json:"rol"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
			return
		}
		if err := operacionesService.RegisterAgent(strings.TrimSpace(payload.Nombre), strings.TrimSpace(payload.Rol)); err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		webWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "nombre": strings.TrimSpace(payload.Nombre), "rol": strings.TrimSpace(payload.Rol)})
		return
	}
	items, err := operacionesService.ListAgents()
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIAgenteRetirar(w http.ResponseWriter, r *http.Request, agente string) {
	nombre := strings.TrimSpace(agente)
	if nombre == "" {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "agente obligatorio"})
		return
	}
	if nombre == "alberto" {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "no puedes retirar al administrador"})
		return
	}
	if err := operacionesService.RetireAgent(nombre); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "nombre": nombre})
}

func webHandlerAPIAgenteRehabilitar(w http.ResponseWriter, r *http.Request, agente string) {
	nombre := strings.TrimSpace(agente)
	if nombre == "" {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "agente obligatorio"})
		return
	}
	if err := operacionesService.RehabilitateAgent(nombre); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "nombre": nombre})
}

func webHandlerAPIAsignaciones(w http.ResponseWriter, r *http.Request) {
	items, err := operacionesService.ListAssignments(
		strings.TrimSpace(r.URL.Query().Get("estado")),
		strings.TrimSpace(r.URL.Query().Get("agente")),
	)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPISesiones(w http.ResponseWriter, r *http.Request) {
	items, err := operacionesService.ListActiveSessions()
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIExportEstado(w http.ResponseWriter, r *http.Request) {
	body, err := buildExportStateMarkdown()
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
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

func webHandlerAPIExportAudit(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "limit invalido"})
			return
		}
		limit = value
	}
	body, err := buildExportAuditMarkdown(limit)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

const webTplAsignaciones = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">Asignaciones</h2>
  <form method="get" action="/asignaciones">
    <div class="grid2">
      <label>Estado <input name="estado" value="{{.Estado}}"></label>
      <label>Agente <input name="agente" value="{{.Agente}}"></label>
    </div>
    <button type="submit" class="btn-sm">Filtrar</button>
  </form>
  <table>
    <thead><tr><th>ID</th><th>Agente</th><th>Proyecto</th><th>Estado</th><th>Nota</th></tr></thead>
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
  <h2 style="margin:0">Sesiones</h2>
  <form method="get" action="/sesiones">
    <div class="grid2">
      <label>Agente <input name="agente" value="{{.Agente}}"></label>
      <label>Proyecto <input name="proyecto" value="{{.Proyecto}}"></label>
    </div>
    <div class="grid2">
      <label>Estado <input name="estado" value="{{.Estado}}"></label>
      <label>Activa <input name="activa" value="{{.Activa}}" placeholder="true"></label>
    </div>
    <button type="submit" class="btn-sm">Filtrar</button>
  </form>
  <table>
    <thead><tr><th>ID</th><th>Agente</th><th>Proyecto</th><th>Estado</th><th>Activa</th><th>Herramienta</th><th>External session</th><th>Host</th><th>Branch</th><th>Inicio</th></tr></thead>
    <tbody>
      {{range .Sesiones}}
      <tr>
        <td><a href="/sesiones/{{.ID}}">{{.ID}}</a></td>
        <td>{{.Agente}}</td>
        <td>{{.ProyectoSlug}}</td>
        <td>{{.Estado}}</td>
        <td>{{if .Activa}}sí{{else}}no{{end}}</td>
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
  <p style="margin:0 0 .35rem 0"><a href="/sesiones">← Volver a sesiones</a></p>
  <h2 style="margin:0">Sesión #{{.Sesion.ID}}</h2>
  <p style="color:#64748b">Detalle read-only de inspección operativa.</p>
  <table>
    <tbody>
      <tr><th>Agente</th><td>{{.Sesion.Agente}}</td></tr>
      <tr><th>Proyecto</th><td>{{.Sesion.ProyectoSlug}}</td></tr>
      <tr><th>Estado</th><td>{{.Sesion.Estado}}</td></tr>
      <tr><th>Activa</th><td>{{if .Sesion.Activa}}sí{{else}}no{{end}}</td></tr>
      <tr><th>Herramienta</th><td>{{.Sesion.Herramienta}}</td></tr>
      <tr><th>Conector</th><td>{{.Sesion.ConectorSlug}}</td></tr>
      <tr><th>Host</th><td>{{.Sesion.Host}}</td></tr>
      <tr><th>PID</th><td>{{if .Sesion.PID}}{{.Sesion.PID}}{{end}}</td></tr>
      <tr><th>Branch</th><td><code>{{.Sesion.Branch}}</code></td></tr>
      <tr><th>CWD</th><td><code>{{.Sesion.CWD}}</code></td></tr>
      <tr><th>External session</th><td><code>{{.Sesion.ExternalSessionID}}</code></td></tr>
      <tr><th>Continuidad</th><td>{{.Sesion.ResumenContinuidad}}</td></tr>
      <tr><th>Inicio</th><td>{{.Sesion.Inicio.Format "2006-01-02 15:04:05"}}</td></tr>
      <tr><th>Heartbeat</th><td>{{if .Sesion.HeartbeatAt}}{{.Sesion.HeartbeatAt.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
      <tr><th>Fin</th><td>{{if .Sesion.Fin}}{{.Sesion.Fin.Format "2006-01-02 15:04:05"}}{{end}}</td></tr>
      <tr><th>Resume payload</th><td><pre style="white-space:pre-wrap;margin:0">{{.Sesion.ResumePayloadJSON}}</pre></td></tr>
    </tbody>
  </table>
</section>
{{end}}`
