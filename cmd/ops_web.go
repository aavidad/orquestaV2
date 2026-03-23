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
	"strconv"
	"strings"

	"orquesta/db"
	"orquesta/opsview"
)

var opsViewService = opsview.NewService(db.OpsViewRepository{})

type webAgentesData struct {
	Agentes []*db.Agente
}

type webAsignacionesData struct {
	Asignaciones []*db.Asignacion
	Estado       string
	Agente       string
}

type webSesionesData struct {
	Sesiones []*db.SesionActiva
}

func webHandlerAgentes(w http.ResponseWriter, r *http.Request) {
	items, err := opsViewService.ListAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	webRender(w, webTplLayout+webTplAgentes, webAgentesData{Agentes: items})
}

func webHandlerAsignaciones(w http.ResponseWriter, r *http.Request) {
	estado := strings.TrimSpace(r.URL.Query().Get("estado"))
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	items, err := opsViewService.ListAssignments(estado, agente)
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
	items, err := opsViewService.ListActiveSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	webRender(w, webTplLayout+webTplSesiones, webSesionesData{Sesiones: items})
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
		if err := opsViewService.RegisterAgent(strings.TrimSpace(payload.Nombre), strings.TrimSpace(payload.Rol)); err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		webWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "nombre": strings.TrimSpace(payload.Nombre), "rol": strings.TrimSpace(payload.Rol)})
		return
	}
	items, err := opsViewService.ListAgents()
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
	if err := opsViewService.RetireAgent(nombre); err != nil {
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
	if err := opsViewService.RehabilitateAgent(nombre); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "nombre": nombre})
}

func webHandlerAPIAsignaciones(w http.ResponseWriter, r *http.Request) {
	items, err := opsViewService.ListAssignments(
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
	items, err := opsViewService.ListActiveSessions()
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

const webTplAgentes = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">Agentes</h2>
  <table>
    <thead><tr><th>Agente</th><th>Rol</th><th>Activo</th><th>Estado</th><th>Habilitado</th><th>Última sesión</th></tr></thead>
    <tbody>
      {{range .Agentes}}
      <tr>
        <td>{{.Nombre}}</td>
        <td>{{.Rol}}</td>
        <td>{{if .Activo}}sí{{else}}no{{end}}</td>
        <td>{{.EstadoSesion}}</td>
        <td>{{if .Habilitado}}sí{{else}}no{{end}}</td>
        <td>{{if .UltimaSesion}}{{.UltimaSesion.Format "2006-01-02 15:04:05"}}{{else}}—{{end}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</section>
{{end}}`

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
  <h2 style="margin:0">Sesiones activas</h2>
  <table>
    <thead><tr><th>ID</th><th>Agente</th><th>Proyecto</th><th>Estado</th><th>Herramienta</th><th>Host</th><th>Branch</th><th>Inicio</th></tr></thead>
    <tbody>
      {{range .Sesiones}}
      <tr>
        <td>{{.ID}}</td>
        <td>{{.Agente}}</td>
        <td>{{.ProyectoSlug}}</td>
        <td>{{.Estado}}</td>
        <td>{{.Herramienta}}</td>
        <td>{{.Host}}</td>
        <td>{{.Branch}}</td>
        <td>{{.Inicio.Format "2006-01-02 15:04:05"}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</section>
{{end}}`
