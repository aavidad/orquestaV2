/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/dashboardapp"
	"orquesta/db"
	"orquesta/proposalapp"
	"orquesta/runtimectl"
	"orquesta/sessionapp"
	"orquesta/taskapp"
)

// ─── Structs de datos ────────────────────────────────────────────────────────

type webDashData struct {
	Agentes     []*db.Agente
	Counts      map[string]int
	Total       int
	Completadas int
	Pct         int
	Abiertas    []webPropResumen
	EnProgreso  []webTareaRow
	Generado    string
	Msg         string
}

type webPropResumen struct {
	Codigo     string
	Titulo     string
	Acuerdo    int
	Desacuerdo int
	Pendiente  int
}

type webTareaRow struct {
	ID        int64
	Titulo    string
	Modulo    string
	Estado    string
	Prioridad string
	Agente    string
}

type webTareasData struct {
	Tareas  []webTareaRow
	Filtro  string
	Agentes []*db.Agente
	Msg     string
	Err     string
}

type webTareaDetalleData struct {
	T       webTareaRow
	Agentes []*db.Agente
	Msg     string
	Err     string
}

type webPropData struct {
	Propuestas []webPropDetalle
	Filtro     string
	Msg        string
	Err        string
}

type webPropDetalle struct {
	Codigo       string
	Titulo       string
	Descripcion  string
	Tipo         string
	Estado       string
	PropuestoPor string
	Fecha        string
	CerradaFecha string
	Votos        []*db.Voto
}

type webPropDetalleData struct {
	P       webPropDetalle
	Agentes []*db.Agente
	Msg     string
	Err     string
}

var dashboardService = dashboardapp.NewService(dashboardapp.Repository{})
var sessionAPIService = sessionapp.NewService(sessionapp.Repository{})

// ─── FuncMap ─────────────────────────────────────────────────────────────────

var webFuncMap = template.FuncMap{
	"trunc": func(s string, n int) string {
		runes := []rune(s)
		if len(runes) <= n {
			return s
		}
		return string(runes[:n-1]) + "…"
	},
	"eqStr": func(a, b string) bool { return a == b },
}

// ─── Router principal ─────────────────────────────────────────────────────────

func webRouterTareas(w http.ResponseWriter, r *http.Request) {
	// /tareas/nueva (POST) | /tareas/{id} (GET) | /tareas/{id}/accion (POST)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// parts[0]="tareas", parts[1]=id or "nueva", parts[2]="accion" optional

	if len(parts) < 2 {
		http.Redirect(w, r, "/tareas", http.StatusSeeOther)
		return
	}

	switch {
	case parts[1] == "nueva" && r.Method == http.MethodPost:
		webHandlerTareaNueva(w, r)
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerTareaDetalle(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "accion" && r.Method == http.MethodPost:
		webHandlerTareaAccion(w, r, parts[1])
	default:
		http.NotFound(w, r)
	}
}

func webRouterPropuestas(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(parts) < 2 {
		http.Redirect(w, r, "/propuestas", http.StatusSeeOther)
		return
	}

	switch {
	case parts[1] == "nueva" && r.Method == http.MethodPost:
		webHandlerPropuestaNueva(w, r)
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerPropuestaDetalle(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "accion" && r.Method == http.MethodPost:
		webHandlerPropuestaAccion(w, r, parts[1])
	default:
		http.NotFound(w, r)
	}
}

func webRouterAPIPools(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerAPIPools(w, r)
	case len(parts) == 3 && r.Method == http.MethodGet:
		webHandlerAPIPoolDetalle(w, r, parts[2])
	case len(parts) == 4 && parts[3] == "modelos" && r.Method == http.MethodGet:
		webHandlerAPIPoolModelos(w, r, parts[2])
	default:
		http.NotFound(w, r)
	}
}

func webRouterAPITareas(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "tareas" {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 3 && r.Method == http.MethodGet:
		webHandlerAPITareaDetalle(w, r, parts[2])
	case len(parts) == 4 && r.Method == http.MethodPost:
		webHandlerAPITareaAccion(w, r, parts[2], parts[3])
	default:
		http.NotFound(w, r)
	}
}

func webRouterAPIPropuestasCLI(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "propuestas" {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 3 && r.Method == http.MethodGet:
		webHandlerAPIPropuestaDetalle(w, r, parts[2])
	case len(parts) == 4 && parts[3] == "cerrar" && r.Method == http.MethodPost:
		webHandlerAPIPropuestaCerrar(w, r, parts[2])
	default:
		http.NotFound(w, r)
	}
}

func webRouterAPIModelos(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/modelos/resolver" && r.Method == http.MethodGet:
		webHandlerAPIModelosResolver(w, r)
	case r.URL.Path == "/api/modelos/politicas" && r.Method == http.MethodGet:
		webHandlerAPIModelosPoliticas(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webRouterAPIExport(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/export/estado" && r.Method == http.MethodGet:
		webHandlerAPIExportEstado(w, r)
	case r.URL.Path == "/api/export/audit" && r.Method == http.MethodGet:
		webHandlerAPIExportAudit(w, r)
	default:
		http.NotFound(w, r)
	}
}

func webRouterAPIAgentes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[0] != "api" || parts[1] != "agentes" {
		http.NotFound(w, r)
		return
	}
	agente := parts[2]
	switch {
	case len(parts) == 4 && parts[3] == "retirar" && r.Method == http.MethodPost:
		webHandlerAPIAgenteRetirar(w, r, agente)
	case len(parts) == 4 && parts[3] == "rehabilitar" && r.Method == http.MethodPost:
		webHandlerAPIAgenteRehabilitar(w, r, agente)
	case len(parts) == 4 && parts[3] == "runtime-handles" && r.Method == http.MethodGet:
		webHandlerAPIAgenteRuntimeHandles(w, r, agente)
	case len(parts) == 4 && parts[3] == "runtime-handles" && r.Method == http.MethodPost:
		webHandlerAPIAgenteRuntimeHandleRegistrar(w, r, agente)
	case len(parts) == 4 && parts[3] == "runtime-orders" && r.Method == http.MethodGet:
		webHandlerAPIAgenteRuntimeOrders(w, r, agente)
	case len(parts) == 4 && parts[3] == "runtime-orders" && r.Method == http.MethodPost:
		webHandlerAPIAgenteRuntimeOrderCrear(w, r, agente)
	case len(parts) == 5 && parts[3] == "runtime-orders" && parts[4] == "ejecutar" && r.Method == http.MethodPost:
		webHandlerAPIAgenteRuntimeOrderEjecutar(w, r, agente)
	default:
		http.NotFound(w, r)
	}
}

// ─── Dashboard ────────────────────────────────────────────────────────────────

func webHandlerDash(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	summary, err := dashboardService.BuildSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resAbiertas []webPropResumen
	for _, p := range summary.OpenProps {
		resAbiertas = append(resAbiertas, webPropResumen{
			Codigo: p.Codigo, Titulo: p.Titulo,
			Acuerdo: p.Acuerdo, Desacuerdo: p.Desacuerdo, Pendiente: p.Pendiente,
		})
	}
	var ep []webTareaRow
	for _, t := range summary.ActiveTasks {
		ep = append(ep, toWebTarea(t))
	}
	webRender(w, webTplLayout+webTplDash, webDashData{
		Agentes: summary.Agents, Counts: summary.TaskCounts, Total: summary.TotalTasks,
		Completadas: summary.DoneTasks, Pct: summary.PercentDone,
		Abiertas: resAbiertas, EnProgreso: ep,
		Generado: summary.GeneratedAt.Format("2006-01-02 15:04:05"),
	})
}

// ─── Tareas: lista ────────────────────────────────────────────────────────────

func webHandlerTareas(w http.ResponseWriter, r *http.Request) {
	filtro := r.URL.Query().Get("estado")
	msg := r.URL.Query().Get("ok")
	errMsg := r.URL.Query().Get("err")

	f := taskFilterFromEstado(filtro)
	tareas, _ := taskService.List(f)
	var wt []webTareaRow
	for _, t := range tareas {
		wt = append(wt, toWebTarea(t))
	}
	agentes, _ := taskService.ListAgents()
	webRender(w, webTplLayout+webTplTareas, webTareasData{
		Tareas: wt, Filtro: filtro, Agentes: agentes,
		Msg: msg, Err: errMsg,
	})
}

// ─── Tareas: nueva (POST) ─────────────────────────────────────────────────────

func webHandlerTareaNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	titulo := strings.TrimSpace(r.FormValue("titulo"))
	if titulo == "" {
		http.Redirect(w, r, "/tareas?err="+url.QueryEscape("El título es obligatorio"), http.StatusSeeOther)
		return
	}
	id, err := taskService.Create(taskapp.CreateTaskInput{
		Titulo:          titulo,
		Descripcion:     r.FormValue("descripcion"),
		Modulo:          r.FormValue("modulo"),
		Prioridad:       taskPriority(r.FormValue("prioridad")),
		CreadoPor:       "alberto",
		Agente:          strings.TrimSpace(r.FormValue("agente")),
		PropuestaCodigo: strings.TrimSpace(r.FormValue("propuesta")),
	})
	if err != nil {
		http.Redirect(w, r, "/tareas?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/tareas?ok="+url.QueryEscape(fmt.Sprintf("Tarea #%d creada", id)), http.StatusSeeOther)
}

// ─── Tareas: detalle ──────────────────────────────────────────────────────────

func webHandlerTareaDetalle(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	t, err := taskService.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	agentes, _ := taskService.ListAgents()
	webRender(w, webTplLayout+webTplTareaDetalle, webTareaDetalleData{
		T:       toWebTarea(t),
		Agentes: agentes,
		Msg:     r.URL.Query().Get("ok"),
		Err:     r.URL.Query().Get("err"),
	})
}

// ─── Tareas: acción (POST) ────────────────────────────────────────────────────

func webHandlerTareaAccion(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/tareas?err=ID+inválido", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	accion := r.FormValue("accion")
	agente := strings.TrimSpace(r.FormValue("agente"))
	back := fmt.Sprintf("/tareas/%d", id)

	switch accion {
	case "tomar":
		err = taskService.Take(id, agente)
	case "iniciar":
		err = taskService.Start(id, agente)
	case "completar":
		err = taskService.Complete(id, agente, r.FormValue("commit"))
	case "bloquear":
		err = taskService.Block(id, agente, r.FormValue("motivo"))
	case "desbloquear":
		err = taskService.Unblock(id, agente, r.FormValue("resolucion"))
	case "nota":
		err = taskService.Note(id, agente, r.FormValue("nota"))
	case "backlog":
		err = taskService.MoveToBacklog(id)
	case "reasignar":
		nuevoAgente := strings.TrimSpace(r.FormValue("nuevo_agente"))
		err = taskService.Reassign(id, nuevoAgente)
	default:
		http.Redirect(w, r, back+"?err=Acción+desconocida", http.StatusSeeOther)
		return
	}

	if err != nil {
		http.Redirect(w, r, back+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, back+"?ok="+url.QueryEscape("Acción '"+accion+"' aplicada"), http.StatusSeeOther)
}

// ─── Propuestas: lista ────────────────────────────────────────────────────────

func webHandlerPropuestas(w http.ResponseWriter, r *http.Request) {
	filtro := r.URL.Query().Get("estado")
	msg := r.URL.Query().Get("ok")
	errMsg := r.URL.Query().Get("err")

	props, _ := proposalService.List(proposalState(filtro))
	var detalles []webPropDetalle
	for _, p := range props {
		detail, _ := proposalService.GetDetail(p.Codigo)
		var votos []*db.Voto
		if detail != nil {
			votos = detail.Votes
		}
		cerrada := ""
		if p.CerradaAt != nil {
			cerrada = p.CerradaAt.Format("2006-01-02")
		}
		detalles = append(detalles, webPropDetalle{
			Codigo: p.Codigo, Titulo: p.Titulo, Descripcion: p.Descripcion,
			Tipo: p.Tipo, Estado: string(p.Estado), PropuestoPor: p.PropuestoPor,
			Fecha: p.CreatedAt.Format("2006-01-02"), CerradaFecha: cerrada,
			Votos: votos,
		})
	}
	webRender(w, webTplLayout+webTplPropuestas, webPropData{
		Propuestas: detalles, Filtro: filtro, Msg: msg, Err: errMsg,
	})
}

// ─── Propuestas: nueva (POST) ─────────────────────────────────────────────────

func webHandlerPropuestaNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	titulo := strings.TrimSpace(r.FormValue("titulo"))
	if titulo == "" {
		http.Redirect(w, r, "/propuestas?err="+url.QueryEscape("El título es obligatorio"), http.StatusSeeOther)
		return
	}
	_, created, err := proposalService.Create(proposalapp.CreateProposalInput{
		Codigo:       strings.TrimSpace(r.FormValue("codigo")),
		Titulo:       titulo,
		Descripcion:  r.FormValue("descripcion"),
		Tipo:         r.FormValue("tipo"),
		PropuestoPor: "alberto",
		Distribuidor: "alberto",
	})
	if err != nil {
		http.Redirect(w, r, "/propuestas?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/propuestas/"+created.Codigo+"?ok="+url.QueryEscape("Propuesta "+created.Codigo+" creada"), http.StatusSeeOther)
}

// ─── Propuestas: detalle ──────────────────────────────────────────────────────

func webHandlerPropuestaDetalle(w http.ResponseWriter, r *http.Request, codigo string) {
	detail, err := proposalService.GetDetail(codigo)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p := detail.Proposal
	votos := detail.Votes
	cerrada := ""
	if p.CerradaAt != nil {
		cerrada = p.CerradaAt.Format("2006-01-02")
	}
	agentes, _ := proposalService.ListAgents()
	webRender(w, webTplLayout+webTplPropuestaDetalle, webPropDetalleData{
		P: webPropDetalle{
			Codigo: p.Codigo, Titulo: p.Titulo, Descripcion: p.Descripcion,
			Tipo: p.Tipo, Estado: string(p.Estado), PropuestoPor: p.PropuestoPor,
			Fecha: p.CreatedAt.Format("2006-01-02"), CerradaFecha: cerrada,
			Votos: votos,
		},
		Agentes: agentes,
		Msg:     r.URL.Query().Get("ok"),
		Err:     r.URL.Query().Get("err"),
	})
}

// ─── Propuestas: acción (POST) ────────────────────────────────────────────────

func webHandlerPropuestaAccion(w http.ResponseWriter, r *http.Request, codigo string) {
	_ = r.ParseForm()
	accion := r.FormValue("accion")
	back := "/propuestas/" + codigo

	detail, err := proposalService.GetDetail(codigo)
	if err != nil {
		http.Redirect(w, r, "/propuestas?err=Propuesta+no+encontrada", http.StatusSeeOther)
		return
	}
	p := detail.Proposal

	switch accion {
	case "votar":
		agente := strings.TrimSpace(r.FormValue("agente"))
		posicion, ok := votePosition(r.FormValue("posicion"))
		if !ok {
			http.Redirect(w, r, back+"?err=Posición+inválida", http.StatusSeeOther)
			return
		}
		comentario := r.FormValue("comentario")
		err = proposalService.Vote(p.Codigo, agente, posicion, comentario)
	case "cerrar":
		nuevoEstado := r.FormValue("estado_cierre")
		err = proposalService.Close(codigo, nuevoEstado, "alberto")
	default:
		http.Redirect(w, r, back+"?err=Acción+desconocida", http.StatusSeeOther)
		return
	}

	if err != nil {
		http.Redirect(w, r, back+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, back+"?ok="+url.QueryEscape("Acción '"+accion+"' aplicada"), http.StatusSeeOther)
}

// ─── API JSON: pools y modelos ──────────────────────────────────────────────

func webHandlerAPIPools(w http.ResponseWriter, r *http.Request) {
	pools, err := capacityService.ListPoolsSummary(nil)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": pools})
}

func webHandlerAPIPoolDetalle(w http.ResponseWriter, r *http.Request, slug string) {
	detail, err := capacityService.GetPoolDetail(slug)
	if err != nil {
		webWriteJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{
		"pool":                 detail.Pool,
		"sesiones_activas":     detail.SesionesActivas,
		"capacidad_disponible": detail.CapacidadDisponible,
		"modelos":              detail.Modelos,
	})
}

func webHandlerAPIPoolModelos(w http.ResponseWriter, r *http.Request, slug string) {
	modelos, err := capacityService.ListPoolModels(slug)
	if err != nil {
		webWriteJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": modelos})
}

func webHandlerAPIModelosPoliticas(w http.ResponseWriter, r *http.Request) {
	scopeTipo := strings.TrimSpace(r.URL.Query().Get("scope_tipo"))
	scopeRef := strings.TrimSpace(r.URL.Query().Get("scope_ref"))
	var activaPtr *bool
	if activaRaw := strings.TrimSpace(r.URL.Query().Get("activa")); activaRaw != "" {
		activa := activaRaw == "1" || strings.EqualFold(activaRaw, "true")
		activaPtr = &activa
	}
	items, err := capacityService.ListModelPolicies(scopeTipo, scopeRef, activaPtr)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIModelosResolver(w http.ResponseWriter, r *http.Request) {
	var tareaIDPtr *int64
	if tareaIDRaw := strings.TrimSpace(r.URL.Query().Get("tarea_id")); tareaIDRaw != "" {
		value, err := strconv.ParseInt(tareaIDRaw, 10, 64)
		if err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "tarea_id invalido"})
			return
		}
		tareaIDPtr = &value
	}
	res, err := capacityService.ResolveModelPolicy(db.ResolverPoliticaInput{
		TareaID:      tareaIDPtr,
		ProyectoSlug: strings.TrimSpace(r.URL.Query().Get("proyecto")),
		Fase:         strings.TrimSpace(r.URL.Query().Get("fase")),
		PerfilTarea:  strings.TrimSpace(r.URL.Query().Get("perfil")),
	})
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, res)
}

func webHandlerAPIAgenteRuntimeHandles(w http.ResponseWriter, r *http.Request, agente string) {
	estado := strings.TrimSpace(r.URL.Query().Get("estado"))
	items, err := runtimeService.ListHandles(agente, estado)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIAgenteRuntimeHandleRegistrar(w http.ResponseWriter, r *http.Request, agente string) {
	var payload struct {
		SesionID     *int64 `json:"sesion_id"`
		ProyectoSlug string `json:"proyecto_slug"`
		Transporte   string `json:"transporte"`
		HandleKind   string `json:"handle_kind"`
		HandleRef    string `json:"handle_ref"`
		Estado       string `json:"estado"`
		MetadataJSON string `json:"metadata_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	id, err := runtimeService.RegisterHandle(runtimectl.RegisterHandleInput{
		Agente:       agente,
		SesionID:     payload.SesionID,
		ProyectoSlug: payload.ProyectoSlug,
		Transporte:   payload.Transporte,
		HandleKind:   payload.HandleKind,
		HandleRef:    payload.HandleRef,
		Estado:       payload.Estado,
		MetadataJSON: payload.MetadataJSON,
	})
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func webHandlerAPIAgenteRuntimeOrders(w http.ResponseWriter, r *http.Request, agente string) {
	estado := strings.TrimSpace(r.URL.Query().Get("estado"))
	items, err := runtimeService.ListOrders(agente, estado)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIAgenteRuntimeOrderCrear(w http.ResponseWriter, r *http.Request, agente string) {
	var payload struct {
		Tipo         string `json:"tipo"`
		ProyectoSlug string `json:"proyecto_slug"`
		Mensaje      string `json:"mensaje"`
		Destino      string `json:"destino"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	var (
		id  int64
		err error
	)
	switch strings.TrimSpace(payload.Tipo) {
	case "enviar_instruccion":
		id, err = runtimeService.EnqueueInstruction(agente, payload.ProyectoSlug, payload.Mensaje)
	case "pausar":
		id, err = runtimeService.EnqueuePause(agente, payload.ProyectoSlug)
	case "continuar":
		id, err = runtimeService.EnqueueContinue(agente, payload.ProyectoSlug)
	case "handoff":
		id, err = runtimeService.EnqueueHandoff(agente, payload.Destino, payload.ProyectoSlug)
	default:
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "tipo de runtime order no soportado"})
		return
	}
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func webHandlerAPIAgenteRuntimeOrderEjecutar(w http.ResponseWriter, r *http.Request, agente string) {
	item, err := runtimeService.ExecuteNext(agente)
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if item == nil {
		webWriteJSON(w, http.StatusOK, map[string]any{"item": nil, "message": "sin ordenes pendientes"})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func toWebTarea(t *db.Tarea) webTareaRow {
	ag := "—"
	if t.Agente != nil {
		ag = *t.Agente
	}
	return webTareaRow{
		ID: t.ID, Titulo: t.Titulo, Modulo: t.Modulo,
		Estado: string(t.Estado), Prioridad: string(t.Prioridad), Agente: ag,
	}
}

func taskPriority(raw string) db.PrioridadTarea {
	return db.PrioridadTarea(strings.TrimSpace(raw))
}

func taskFilterFromEstado(raw string) db.FiltroTareas {
	var f db.FiltroTareas
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return f
	}
	e := db.EstadoTarea(raw)
	f.Estado = &e
	return f
}

func proposalState(raw string) *db.EstadoPropuesta {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	e := db.EstadoPropuesta(raw)
	return &e
}

func votePosition(raw string) (db.PosicionVoto, bool) {
	posicion := db.PosicionVoto(strings.TrimSpace(raw))
	switch posicion {
	case db.VotoAcuerdo, db.VotoDesacuerdo, db.VotoAbstencion:
		return posicion, true
	default:
		return "", false
	}
}

func webRender(w http.ResponseWriter, tplStr string, data any) {
	tmpl, err := template.New("layout").Funcs(webFuncMap).Parse(tplStr)
	if err != nil {
		http.Error(w, "Error de plantilla: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

func webWriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func webHandlerAPIServerInfo(w http.ResponseWriter, r *http.Request) {
	webWriteJSON(w, http.StatusOK, serverInfo{
		Name:        "orquesta",
		Version:     "v1",
		StorageMode: "single-process",
		Capabilities: []string{
			"status",
			"web",
			"api",
			"mcp",
		},
	})
}

func webHandlerAPIStatus(w http.ResponseWriter, r *http.Request) {
	resumen, err := buildEstadoResumen()
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, resumen)
}

func webHandlerAPIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var payload struct {
			Clave string `json:"clave"`
			Valor string `json:"valor"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
			return
		}
		if strings.TrimSpace(payload.Clave) == "" {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "clave obligatoria"})
			return
		}
		if err := configService.Set(payload.Clave, payload.Valor); err != nil {
			webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		webWriteJSON(w, http.StatusOK, map[string]any{"clave": payload.Clave, "valor": payload.Valor})
		return
	}
	clave := strings.TrimSpace(r.URL.Query().Get("clave"))
	if clave != "" {
		valor, err := configService.Get(clave)
		if err != nil {
			webWriteJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		webWriteJSON(w, http.StatusOK, map[string]any{"clave": clave, "valor": valor})
		return
	}
	items, err := configService.List()
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIVotar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var payload struct {
		Codigo     string `json:"codigo"`
		Agente     string `json:"agente"`
		Posicion   string `json:"posicion"`
		Comentario string `json:"comentario"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	posicion, ok := votePosition(payload.Posicion)
	if !ok {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "posicion invalida"})
		return
	}
	result, err := proposalService.VoteDetail(strings.TrimSpace(payload.Codigo), strings.TrimSpace(payload.Agente), posicion, strings.TrimSpace(payload.Comentario))
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{
		"codigo":            result.Proposal.Codigo,
		"agente":            strings.TrimSpace(payload.Agente),
		"posicion":          posicion,
		"comentario":        strings.TrimSpace(payload.Comentario),
		"consensoAlcanzado": result.Consenso,
		"conteo": map[string]int{
			"acuerdo":    result.Acuerdo,
			"desacuerdo": result.Desacuerdo,
			"abstencion": result.Abstencion,
			"pendiente":  result.Pendiente,
		},
	})
}

func webHandlerAPISesionInicio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var payload struct {
		Agente     string `json:"agente"`
		NuevoCodex bool   `json:"nuevo_codex"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	agente := strings.TrimSpace(payload.Agente)
	if agente == "" && !payload.NuevoCodex {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "agente obligatorio"})
		return
	}
	result, err := sessionAPIService.Start(agente, payload.NuevoCodex)
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, serverSessionStartResult{
		Agente:               result.Agente,
		SesionID:             result.SesionID,
		Rol:                  result.Rol,
		PropuestasPendientes: result.PropuestasPendientes,
		Reglas:               result.Reglas,
		Skills:               result.Skills,
		WorkflowPasos:        result.WorkflowPasos,
	})
}

func webHandlerAPISesionFin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var payload struct {
		Agente string `json:"agente"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	agente := strings.TrimSpace(payload.Agente)
	if agente == "" {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "agente obligatorio"})
		return
	}
	if _, err := sessionAPIService.Finish(agente); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "agente": agente})
}

func webHandlerAPITareas(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var payload struct {
			Titulo      string `json:"titulo"`
			Descripcion string `json:"descripcion"`
			Modulo      string `json:"modulo"`
			Prioridad   string `json:"prioridad"`
			CreadoPor   string `json:"creado_por"`
			Agente      string `json:"agente"`
			Propuesta   string `json:"propuesta"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
			return
		}
		id, err := taskService.Create(taskapp.CreateTaskInput{
			Titulo:          strings.TrimSpace(payload.Titulo),
			Descripcion:     strings.TrimSpace(payload.Descripcion),
			Modulo:          strings.TrimSpace(payload.Modulo),
			Prioridad:       taskPriority(payload.Prioridad),
			CreadoPor:       strings.TrimSpace(payload.CreadoPor),
			Agente:          strings.TrimSpace(payload.Agente),
			PropuestaCodigo: strings.TrimSpace(payload.Propuesta),
		})
		if err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		webWriteJSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	estadoStr := strings.TrimSpace(r.URL.Query().Get("estado"))
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	modulo := strings.TrimSpace(r.URL.Query().Get("modulo"))
	propuestaCodigo := strings.TrimSpace(r.URL.Query().Get("propuesta"))

	f := taskFilterFromEstado(estadoStr)
	if agente != "" {
		f.Agente = &agente
	}
	if modulo != "" {
		f.Modulo = &modulo
	}
	if propuestaID, err := taskService.ResolveProposalID(propuestaCodigo); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	} else if propuestaID != nil {
		f.PropuestaID = propuestaID
	}

	items, err := taskService.List(f)
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPITareaDetalle(w http.ResponseWriter, r *http.Request, idRaw string) {
	id, err := strconv.ParseInt(strings.TrimSpace(idRaw), 10, 64)
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "id invalido"})
		return
	}
	item, err := taskService.Get(id)
	if err != nil {
		webWriteJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func webHandlerAPITareaAccion(w http.ResponseWriter, r *http.Request, idRaw, accion string) {
	id, err := strconv.ParseInt(strings.TrimSpace(idRaw), 10, 64)
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "id invalido"})
		return
	}
	var payload struct {
		Agente     string `json:"agente"`
		Commit     string `json:"commit"`
		Motivo     string `json:"motivo"`
		Resolucion string `json:"resolucion"`
		Nota       string `json:"nota"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	agente := strings.TrimSpace(payload.Agente)
	switch strings.TrimSpace(accion) {
	case "tomar":
		err = taskService.Take(id, agente)
	case "iniciar":
		err = taskService.Start(id, agente)
	case "completar":
		err = taskService.Complete(id, agente, strings.TrimSpace(payload.Commit))
	case "bloquear":
		err = taskService.Block(id, agente, strings.TrimSpace(payload.Motivo))
	case "desbloquear":
		err = taskService.Unblock(id, agente, strings.TrimSpace(payload.Resolucion))
	case "nota":
		err = taskService.Note(id, agente, strings.TrimSpace(payload.Nota))
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "accion": accion})
}

func webHandlerAPIPropuestas(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var payload struct {
			Codigo       string `json:"codigo"`
			Titulo       string `json:"titulo"`
			Descripcion  string `json:"descripcion"`
			Tipo         string `json:"tipo"`
			PropuestoPor string `json:"propuesto_por"`
			Distribuidor string `json:"distribuidor"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
			return
		}
		id, p, err := proposalService.Create(proposalapp.CreateProposalInput{
			Codigo:       strings.TrimSpace(payload.Codigo),
			Titulo:       strings.TrimSpace(payload.Titulo),
			Descripcion:  strings.TrimSpace(payload.Descripcion),
			Tipo:         strings.TrimSpace(payload.Tipo),
			PropuestoPor: strings.TrimSpace(payload.PropuestoPor),
			Distribuidor: strings.TrimSpace(payload.Distribuidor),
		})
		if err != nil {
			webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		webWriteJSON(w, http.StatusCreated, map[string]any{"id": id, "codigo": p.Codigo})
		return
	}
	estadoStr := strings.TrimSpace(r.URL.Query().Get("estado"))
	items, err := proposalService.List(proposalState(estadoStr))
	if err != nil {
		webWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func webHandlerAPIPropuestaDetalle(w http.ResponseWriter, r *http.Request, codigo string) {
	detail, err := proposalService.GetDetail(codigo)
	if err != nil {
		webWriteJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, detail)
}

func webHandlerAPIPropuestaCerrar(w http.ResponseWriter, r *http.Request, codigo string) {
	var payload struct {
		Estado string `json:"estado"`
		Agente string `json:"agente"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "json invalido"})
		return
	}
	if err := proposalService.Close(codigo, strings.TrimSpace(payload.Estado), strings.TrimSpace(payload.Agente)); err != nil {
		webWriteJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	webWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "codigo": codigo, "estado": strings.TrimSpace(payload.Estado)})
}

// ─── Comando ──────────────────────────────────────────────────────────────────

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Arranca el panel web (por defecto: http://localhost:8080)",
	RunE: func(cmd *cobra.Command, args []string) error {
		puerto, _ := cmd.Flags().GetInt("puerto")
		host, _ := cmd.Flags().GetString("host")
		tlsCertFile, _ := cmd.Flags().GetString("tls-cert")
		tlsKeyFile, _ := cmd.Flags().GetString("tls-key")
		tlsClientCAFile, _ := cmd.Flags().GetString("tls-client-ca")
		host = strings.TrimSpace(host)
		if host == "" {
			host = "127.0.0.1"
		}
		addr := net.JoinHostPort(host, strconv.Itoa(puerto))
		if err := validateServeSecurity(host, tlsCertFile, tlsKeyFile, tlsClientCAFile); err != nil {
			return err
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/", webHandlerDash)
		mux.HandleFunc("/agentes", webHandlerAgentes)
		mux.HandleFunc("/asignaciones", webHandlerAsignaciones)
		mux.HandleFunc("/git", webHandlerGitGov)
		mux.HandleFunc("/git/", webRouterGitGov)
		mux.HandleFunc("/sesiones", webHandlerSesiones)
		mux.HandleFunc("/tareas", webHandlerTareas)
		mux.HandleFunc("/tareas/", webRouterTareas)
		mux.HandleFunc("/propuestas", webHandlerPropuestas)
		mux.HandleFunc("/propuestas/", webRouterPropuestas)
		mux.HandleFunc("/gobernanza", webHandlerGobernanza)
		mux.HandleFunc("/gobernanza/", webRouterGobernanza)
		mux.HandleFunc("/proyectos", webHandlerProyectos)
		mux.HandleFunc("/proyectos/", webRouterProyectos)
		mux.HandleFunc("/api/agentes", webHandlerAPIAgentesLista)
		mux.HandleFunc("/api/asignaciones", webHandlerAPIAsignaciones)
		mux.HandleFunc("/api/config", webHandlerAPIConfig)
		mux.HandleFunc("/api/propuestas", webHandlerAPIPropuestas)
		mux.HandleFunc("/api/propuestas/", webRouterAPIPropuestasCLI)
		mux.HandleFunc("/api/server", webHandlerAPIServerInfo)
		mux.HandleFunc("/api/sesiones/inicio", webHandlerAPISesionInicio)
		mux.HandleFunc("/api/sesiones/fin", webHandlerAPISesionFin)
		mux.HandleFunc("/api/status", webHandlerAPIStatus)
		mux.HandleFunc("/api/tareas", webHandlerAPITareas)
		mux.HandleFunc("/api/tareas/", webRouterAPITareas)
		mux.HandleFunc("/api/votar", webHandlerAPIVotar)
		mux.HandleFunc("/api/git/worktrees", webHandlerAPIWorktrees)
		mux.HandleFunc("/api/git/locks", webHandlerAPILocks)
		mux.HandleFunc("/api/git/merges", webHandlerAPIMerges)
		mux.HandleFunc("/api/gobernanza/", webRouterAPIGobernanza)
		mux.HandleFunc("/api/locks", webHandlerAPILocks)
		mux.HandleFunc("/api/merges", webHandlerAPIMerges)
		mux.HandleFunc("/api/proyectos", webRouterAPIProyectos)
		mux.HandleFunc("/api/proyectos/", webRouterAPIProyectos)
		mux.HandleFunc("/api/sesiones", webHandlerAPISesiones)
		mux.HandleFunc("/api/worktrees", webHandlerAPIWorktrees)
		mux.HandleFunc("/api/pools", webHandlerAPIPools)
		mux.HandleFunc("/api/pools/", webRouterAPIPools)
		mux.HandleFunc("/api/modelos/resolver", webRouterAPIModelos)
		mux.HandleFunc("/api/modelos/politicas", webRouterAPIModelos)
		mux.HandleFunc("/api/agentes/", webRouterAPIAgentes)
		mux.HandleFunc("/api/export/", webRouterAPIExport)
		server := &http.Server{Addr: addr, Handler: mux}
		scheme := "http"
		if strings.TrimSpace(tlsCertFile) != "" {
			tlsConfig, err := buildServeTLSConfig(tlsClientCAFile)
			if err != nil {
				return err
			}
			server.TLSConfig = tlsConfig
			scheme = "https"
		}
		for _, target := range describeServeTargets(host, puerto, scheme) {
			fmt.Printf("✓ Panel web en %s\n", target)
		}
		if scheme == "https" && strings.TrimSpace(tlsClientCAFile) != "" {
			fmt.Println("  mTLS activo: el cliente debe presentar un certificado válido.")
		}
		if !isLocalServeHost(host) {
			fmt.Println("  Exposición remota permitida solo por HTTPS con certificado de cliente.")
		}
		fmt.Println("  Ctrl+C para detener.")
		if scheme == "https" {
			return server.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
		}
		return server.ListenAndServe()
	},
}

func init() {
	serveCmd.Flags().String("host", "127.0.0.1", "Host de escucha (127.0.0.1 para solo local)")
	serveCmd.Flags().Int("puerto", 8080, "Puerto HTTP")
	serveCmd.Flags().String("tls-cert", "", "Certificado TLS del servidor (PEM)")
	serveCmd.Flags().String("tls-key", "", "Clave privada TLS del servidor (PEM)")
	serveCmd.Flags().String("tls-client-ca", "", "CA PEM para exigir certificado de cliente (mTLS)")
	rootCmd.AddCommand(serveCmd)
}

// ─────────────────────────────────────────────────────────────────────────────
// PLANTILLAS HTML
// ─────────────────────────────────────────────────────────────────────────────

const webTplLayout = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Orquesta — ContaGrx</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">
  <style>
    body{--pico-font-size:14px}
    header{background:#0f172a;padding:.7rem 0;margin-bottom:0}
    header nav{display:flex;align-items:center;justify-content:space-between;gap:.8rem;flex-wrap:wrap}
    header a{color:#94a3b8;text-decoration:none;margin-right:1.2rem;font-size:.95rem}
    header a:hover,header a.sel{color:#fff}
    header strong{color:#fff}
    .toplinks{display:flex;flex-wrap:wrap;gap:.2rem .8rem;align-items:center}
    .stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(130px,1fr));gap:.8rem;margin-bottom:1.2rem}
    .stat{background:#f8fafc;border:1px solid #e2e8f0;border-radius:.5rem;padding:.9rem;text-align:center}
    .stat .n{font-size:1.9rem;font-weight:700;color:#1e293b;line-height:1.1}
    .stat .l{font-size:.75rem;color:#64748b}
    .pgbar{height:1.5rem;background:#e2e8f0;border-radius:1rem;overflow:hidden;margin-bottom:1.5rem}
    .pgfill{height:100%;background:linear-gradient(90deg,#7c3aed,#2563eb);display:flex;align-items:center;justify-content:center;color:#fff;font-size:.78rem;font-weight:700;min-width:2.5rem}
    .tag{display:inline-block;padding:.15em .55em;border-radius:.3em;font-size:.75em;font-weight:600;vertical-align:middle}
    .t-alta,.t-bloqueada,.t-rechazada{background:#dc2626;color:#fff}
    .t-media,.t-abierta{background:#d97706;color:#fff}
    .t-baja,.t-completada,.t-consenso{background:#16a34a;color:#fff}
    .t-libre,.t-backlog,.t-cancelada{background:#6b7280;color:#fff}
    .t-asignada{background:#2563eb;color:#fff}
    .t-en_progreso{background:#7c3aed;color:#fff}
    .t-acuerdo{background:#16a34a;color:#fff}
    .t-desacuerdo{background:#dc2626;color:#fff}
    .t-pendiente{background:#d97706;color:#fff}
    .t-abstencion{background:#6b7280;color:#fff}
    .dot-on{color:#16a34a}
    .dot-off{color:#cbd5e1}
    .es-disponible{background:#dcfce7;color:#166534;border-radius:.3em;padding:.1em .45em;font-size:.72em;font-weight:600}
    .es-programando{background:#ede9fe;color:#5b21b6;border-radius:.3em;padding:.1em .45em;font-size:.72em;font-weight:600}
    .es-esperando{background:#fef3c7;color:#92400e;border-radius:.3em;padding:.1em .45em;font-size:.72em;font-weight:600}
    .es-votando{background:#dbeafe;color:#1e40af;border-radius:.3em;padding:.1em .45em;font-size:.72em;font-weight:600}
    .grid2{display:grid;grid-template-columns:260px 1fr;gap:1.5rem}
    .filtros{margin-bottom:.8rem;display:flex;flex-wrap:wrap;gap:.3rem}
    .filtros a{font-size:.8rem;padding:.25em .7em;border:1px solid #e2e8f0;border-radius:.3em;text-decoration:none;color:#475569}
    .filtros a:hover,.filtros a.sel{background:#1e293b;color:#fff;border-color:#1e293b}
    .alert-ok{background:#dcfce7;border:1px solid #86efac;border-radius:.4rem;padding:.6rem 1rem;margin-bottom:1rem;color:#166534;font-size:.88rem}
    .alert-err{background:#fee2e2;border:1px solid #fca5a5;border-radius:.4rem;padding:.6rem 1rem;margin-bottom:1rem;color:#991b1b;font-size:.88rem}
    details.form-panel{border:1px solid #e2e8f0;border-radius:.5rem;padding:.7rem 1rem;margin-bottom:1rem;background:#f8fafc}
    details.form-panel summary{cursor:pointer;font-weight:600;font-size:.9rem;color:#1e293b}
    details.card{border:1px solid #e2e8f0;border-radius:.4rem;padding:.5rem 1rem;margin-bottom:.5rem}
    details.card summary{cursor:pointer}
    .action-box{background:#f1f5f9;border:1px solid #e2e8f0;border-radius:.4rem;padding:.8rem 1rem;margin-bottom:.7rem}
    .action-box h4{margin:0 0 .5rem 0;font-size:.9rem;color:#334155}
    table{font-size:.85rem}
    th{white-space:nowrap}
    input[type=text],input[type=number],select,textarea{font-size:.85rem;padding:.35rem .6rem}
    .btn-sm{font-size:.8rem;padding:.3rem .8rem}
    footer{text-align:center;padding:1.5rem 0;color:#94a3b8;font-size:.78rem;margin-top:2rem}
    @media(max-width:700px){
      .grid2{grid-template-columns:1fr}
      header nav{align-items:flex-start}
      header a{margin-right:.6rem;font-size:.88rem}
      .toplinks{width:100%}
    }
  </style>
</head>
<body>
<header>
  <nav class="container">
    <div><a href="/"><strong>⚙ Orquesta</strong></a></div>
    <div class="toplinks">
      <a href="/">Dashboard</a>
      <a href="/git">Git</a>
      <a href="/agentes">Agentes</a>
      <a href="/sesiones">Sesiones</a>
      <a href="/asignaciones">Asignaciones</a>
      <a href="/git">Git</a>
      <a href="/tareas">Tareas</a>
      <a href="/propuestas">Propuestas</a>
      <a href="/gobernanza">Gobernanza</a>
      <a href="/proyectos">Proyectos</a>
    </div>
  </nav>
</header>
<main class="container" style="padding-top:1.5rem;padding-bottom:2rem">
{{template "content" .}}
</main>
<footer>ContaGrx · OSL Diputación de Granada · GPLv3</footer>
</body></html>
`

func validateServeSecurity(host, tlsCertFile, tlsKeyFile, tlsClientCAFile string) error {
	host = strings.TrimSpace(host)
	tlsCertFile = strings.TrimSpace(tlsCertFile)
	tlsKeyFile = strings.TrimSpace(tlsKeyFile)
	tlsClientCAFile = strings.TrimSpace(tlsClientCAFile)

	if (tlsCertFile == "") != (tlsKeyFile == "") {
		return fmt.Errorf("tls-cert y tls-key deben proporcionarse juntos")
	}
	if tlsClientCAFile != "" && tlsCertFile == "" {
		return fmt.Errorf("tls-client-ca requiere también tls-cert y tls-key")
	}
	if !isLocalServeHost(host) && tlsClientCAFile == "" {
		return fmt.Errorf("para exponer el panel fuera de localhost debes usar HTTPS con tls-client-ca")
	}
	return nil
}

func buildServeTLSConfig(clientCAFile string) (*tls.Config, error) {
	clientCAFile = strings.TrimSpace(clientCAFile)
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}
	if clientCAFile == "" {
		return cfg, nil
	}
	pemData, err := os.ReadFile(clientCAFile)
	if err != nil {
		return nil, fmt.Errorf("leyendo tls-client-ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemData) {
		return nil, fmt.Errorf("tls-client-ca no contiene certificados PEM válidos")
	}
	cfg.ClientAuth = tls.RequireAndVerifyClientCert
	cfg.ClientCAs = pool
	return cfg, nil
}

func isLocalServeHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	return host == "" || host == "127.0.0.1" || host == "localhost" || host == "::1"
}

func describeServeTargets(host string, port int, scheme string) []string {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	if host != "0.0.0.0" && host != "::" {
		return []string{fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(host, strconv.Itoa(port)))}
	}
	targets := []string{fmt.Sprintf("%s://%s", scheme, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))}
	seen := map[string]bool{targets[0]: true}
	ifaces, err := net.Interfaces()
	if err != nil {
		return targets
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, ok := extractServeIP(addr)
			if !ok || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				url := fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(ip4.String(), strconv.Itoa(port)))
				if !seen[url] {
					seen[url] = true
					targets = append(targets, url)
				}
			}
		}
	}
	return targets
}

func extractServeIP(addr net.Addr) (net.IP, bool) {
	switch value := addr.(type) {
	case *net.IPNet:
		return value.IP, true
	case *net.IPAddr:
		return value.IP, true
	default:
		return nil, false
	}
}

// ─── Dashboard ────────────────────────────────────────────────────────────────

const webTplDash = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1.2rem">
  <h2 style="margin:0">Estado del Proyecto</h2>
  <small style="color:#94a3b8">{{.Generado}} · auto-refresca cada 30 s</small>
</div>
<meta http-equiv="refresh" content="30">
<div class="stats">
  <div class="stat"><div class="n">{{.Total}}</div><div class="l">tareas totales</div></div>
  <div class="stat"><div class="n" style="color:#16a34a">{{.Completadas}}</div><div class="l">completadas</div></div>
  <div class="stat"><div class="n" style="color:#7c3aed">{{index .Counts "en_progreso"}}</div><div class="l">en progreso</div></div>
  <div class="stat"><div class="n" style="color:#2563eb">{{index .Counts "asignada"}}</div><div class="l">asignadas</div></div>
  <div class="stat"><div class="n" style="color:#dc2626">{{index .Counts "bloqueada"}}</div><div class="l">bloqueadas</div></div>
  <div class="stat"><div class="n">{{len .Abiertas}}</div><div class="l">props. abiertas</div></div>
</div>
<div class="pgbar">
  <div class="pgfill" style="width:{{if gt .Pct 0}}{{.Pct}}%{{else}}2.5rem{{end}}">{{.Pct}}%</div>
</div>
<div style="display:grid;grid-template-columns:220px 1fr;gap:1.5rem;align-items:start">
  <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.5rem;padding:.8rem 1rem">
    <h4 style="margin:0 0 .7rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">Agentes</h4>
    <table style="width:100%"><tbody>
    {{range .Agentes}}{{if .Habilitado}}
      <tr style="border-bottom:1px solid #f1f5f9">
        <td style="width:1.2rem;padding:.3rem 0">{{if .Activo}}<span class="dot-on">●</span>{{else}}<span class="dot-off">○</span>{{end}}</td>
        <td style="padding:.3rem .3rem">
          <div style="font-weight:600;font-size:.85rem">{{.Nombre}}</div>
          <div style="font-size:.72rem;color:#94a3b8">{{.Rol}}</div>
        </td>
        <td style="text-align:right;padding:.3rem 0">
          {{if .EstadoSesion}}<span class="es-{{.EstadoSesion}}">{{.EstadoSesion}}</span>{{end}}
        </td>
      </tr>
    {{end}}{{end}}
    </tbody></table>
  </div>
  <div>
    {{if .EnProgreso}}
    <h4 style="margin:0 0 .5rem 0">En progreso</h4>
    <table style="width:100%;margin-bottom:1.2rem"><thead><tr><th>#</th><th>Módulo</th><th>Agente</th><th>Título</th></tr></thead><tbody>
    {{range .EnProgreso}}
      <tr>
        <td><a href="/tareas/{{.ID}}">{{.ID}}</a></td>
        <td><span class="tag t-media">{{.Modulo}}</span></td>
        <td>{{if .Agente}}<span style="font-size:.82rem">*{{.Agente}}</span>{{end}}</td>
        <td style="font-size:.82rem">{{trunc .Titulo 55}}</td>
      </tr>
    {{end}}
    </tbody></table>
    {{end}}
    <div style="display:flex;gap:1rem;flex-wrap:wrap;align-items:start">
      <div>
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">Por estado <a href="/tareas" style="font-size:.9em;font-weight:normal;text-transform:none">ver todas →</a></h4>
        <table><tbody>
        {{range $est,$n := .Counts}}{{if gt $n 0}}
          <tr><td style="padding:.2rem .4rem"><span class="tag t-{{$est}}">{{$est}}</span></td><td style="padding:.2rem .6rem"><strong>{{$n}}</strong></td></tr>
        {{end}}{{end}}
        </tbody></table>
      </div>
      {{if .Abiertas}}
      <div>
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">Propuestas abiertas</h4>
        <table><thead><tr><th>Código</th><th>✓</th><th>✗</th><th>⏳</th></tr></thead><tbody>
        {{range .Abiertas}}
          <tr>
            <td><a href="/propuestas/{{.Codigo}}"><strong>{{.Codigo}}</strong></a><br><small>{{trunc .Titulo 28}}</small></td>
            <td style="color:#16a34a;font-weight:700;text-align:center">{{.Acuerdo}}</td>
            <td style="color:#dc2626;font-weight:700;text-align:center">{{.Desacuerdo}}</td>
            <td style="color:#d97706;font-weight:700;text-align:center">{{.Pendiente}}</td>
          </tr>
        {{end}}
        </tbody></table>
      </div>
      {{end}}
    </div>
  </div>
</div>
{{end}}
`

// ─── Tareas: lista ────────────────────────────────────────────────────────────

const webTplTareas = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1rem">
  <h2 style="margin:0">Tareas <small style="font-size:.5em;color:#94a3b8">{{len .Tareas}}</small></h2>
</div>
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

<details class="form-panel">
  <summary>＋ Nueva tarea</summary>
  <form method="POST" action="/tareas/nueva" style="margin-top:.8rem">
    <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:.5rem">
      <div><label>Título *</label><input type="text" name="titulo" required placeholder="Título de la tarea"></div>
      <div><label>Módulo</label><input type="text" name="modulo" placeholder="M25, transversal…"></div>
      <div><label>Propuesta vinculada</label><input type="text" name="propuesta" placeholder="OP-030"></div>
    </div>
    <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:.5rem;margin-top:.4rem">
      <div><label>Prioridad</label>
        <select name="prioridad">
          <option value="alta">Alta</option>
          <option value="media" selected>Media</option>
          <option value="baja">Baja</option>
        </select>
      </div>
      <div><label>Asignar a agente</label>
        <select name="agente">
          <option value="">— sin asignar —</option>
          {{range .Agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
        </select>
      </div>
      <div><label>Descripción</label><input type="text" name="descripcion" placeholder="Descripción breve"></div>
    </div>
    <button type="submit" class="btn-sm" style="margin-top:.6rem">Crear tarea</button>
  </form>
</details>

<div class="filtros">
  <a href="/tareas"{{if eqStr .Filtro ""}} class="sel"{{end}}>Todas</a>
  <a href="/tareas?estado=en_progreso"{{if eqStr .Filtro "en_progreso"}} class="sel"{{end}}>En progreso</a>
  <a href="/tareas?estado=asignada"{{if eqStr .Filtro "asignada"}} class="sel"{{end}}>Asignadas</a>
  <a href="/tareas?estado=libre"{{if eqStr .Filtro "libre"}} class="sel"{{end}}>Libres</a>
  <a href="/tareas?estado=bloqueada"{{if eqStr .Filtro "bloqueada"}} class="sel"{{end}}>Bloqueadas</a>
  <a href="/tareas?estado=backlog"{{if eqStr .Filtro "backlog"}} class="sel"{{end}}>Backlog</a>
  <a href="/tareas?estado=completada"{{if eqStr .Filtro "completada"}} class="sel"{{end}}>Completadas</a>
</div>

{{if .Tareas}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>#</th><th>Estado</th><th>Prioridad</th><th>Módulo</th><th>Agente</th><th>Título</th><th></th></tr></thead>
  <tbody>
  {{range .Tareas}}
    <tr>
      <td style="color:#94a3b8">{{.ID}}</td>
      <td><span class="tag t-{{.Estado}}">{{.Estado}}</span></td>
      <td><span class="tag t-{{.Prioridad}}">{{.Prioridad}}</span></td>
      <td><code style="font-size:.8em">{{.Modulo}}</code></td>
      <td>{{.Agente}}</td>
      <td>{{.Titulo}}</td>
      <td><a href="/tareas/{{.ID}}" class="btn-sm">Gestionar →</a></td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>
{{else}}
<p style="color:#94a3b8">No hay tareas con este filtro.</p>
{{end}}
{{end}}
`

// ─── Tarea: detalle + acciones ────────────────────────────────────────────────

const webTplTareaDetalle = `{{define "content"}}
<a href="/tareas" style="font-size:.85rem;color:#64748b">← volver a tareas</a>
<h2 style="margin:.5rem 0">#{{.T.ID}} — {{.T.Titulo}}</h2>
<p>
  <span class="tag t-{{.T.Estado}}">{{.T.Estado}}</span>
  <span class="tag t-{{.T.Prioridad}}">{{.T.Prioridad}}</span>
  <span style="color:#64748b;font-size:.85rem;margin-left:.5rem">Módulo: {{.T.Modulo}} · Agente: {{.T.Agente}}</span>
</p>

{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

{{$id := .T.ID}}
{{$estado := .T.Estado}}
{{$agentes := .Agentes}}

{{if or (eqStr $estado "libre") (eqStr $estado "backlog")}}
<div class="action-box">
  <h4>Asignar a agente</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="tomar">
    <div><label>Agente</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <button type="submit" class="btn-sm">Asignar</button>
  </form>
</div>
{{end}}

{{if eqStr $estado "asignada"}}
<div class="action-box">
  <h4>Iniciar trabajo</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="iniciar">
    <div><label>Agente</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}"{{if eqStr .Nombre $.T.Agente}} selected{{end}}>{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <button type="submit" class="btn-sm">Iniciar</button>
  </form>
</div>
{{end}}

{{if or (eqStr $estado "asignada") (eqStr $estado "en_progreso")}}
<div class="action-box">
  <h4>Completar tarea</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="completar">
    <div><label>Agente</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}"{{if eqStr .Nombre $.T.Agente}} selected{{end}}>{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <div><label>Commit (opcional)</label><input type="text" name="commit" placeholder="abc1234" style="width:140px"></div>
    <button type="submit" class="btn-sm" style="background:#16a34a;border-color:#16a34a;color:#fff">Completar</button>
  </form>
</div>
{{end}}

{{if eqStr $estado "en_progreso"}}
<div class="action-box">
  <h4>Bloquear tarea</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="bloquear">
    <div><label>Agente</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}"{{if eqStr .Nombre $.T.Agente}} selected{{end}}>{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <div style="flex:1"><label>Motivo del bloqueo</label><input type="text" name="motivo" required placeholder="Describe el bloqueo" style="width:100%"></div>
    <button type="submit" class="btn-sm" style="background:#dc2626;border-color:#dc2626;color:#fff">Bloquear</button>
  </form>
</div>
{{end}}

{{if eqStr $estado "bloqueada"}}
<div class="action-box">
  <h4>Desbloquear tarea</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="desbloquear">
    <div><label>Agente</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <div style="flex:1"><label>Resolución</label><input type="text" name="resolucion" required placeholder="Cómo se resolvió" style="width:100%"></div>
    <button type="submit" class="btn-sm">Desbloquear</button>
  </form>
</div>
{{end}}

<div class="action-box">
  <h4>Reasignar a otro agente</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="reasignar">
    <div><label>Nuevo agente</label>
      <select name="nuevo_agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <button type="submit" class="btn-sm">Reasignar</button>
  </form>
</div>

<div class="action-box">
  <h4>Añadir nota</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="nota">
    <div><label>Agente</label>
      <select name="agente">
        {{range $agentes}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}
      </select>
    </div>
    <div style="flex:1"><label>Nota</label><input type="text" name="nota" required placeholder="Texto de la nota" style="width:100%"></div>
    <button type="submit" class="btn-sm">Añadir</button>
  </form>
</div>

{{if not (eqStr $estado "backlog")}}
<div class="action-box">
  <h4>Mover a backlog</h4>
  <form method="POST" action="/tareas/{{$id}}/accion">
    <input type="hidden" name="accion" value="backlog">
    <button type="submit" class="btn-sm">Mover a backlog</button>
  </form>
</div>
{{end}}
{{end}}
`

// ─── Propuestas: lista ────────────────────────────────────────────────────────

const webTplPropuestas = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1rem">
  <h2 style="margin:0">Propuestas (OPs) <small style="font-size:.5em;color:#94a3b8">{{len .Propuestas}}</small></h2>
</div>
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

<details class="form-panel">
  <summary>＋ Nueva propuesta</summary>
  <form method="POST" action="/propuestas/nueva" style="margin-top:.8rem">
    <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:.5rem">
      <div><label>Título *</label><input type="text" name="titulo" required placeholder="Título de la propuesta"></div>
      <div><label>Tipo</label>
        <select name="tipo">
          <option value="implementacion">Implementación</option>
          <option value="arquitectura">Arquitectura</option>
          <option value="seguridad">Seguridad</option>
          <option value="backlog">Backlog</option>
          <option value="otro">Otro</option>
        </select>
      </div>
      <div><label>Código (opcional)</label><input type="text" name="codigo" placeholder="OP-030 (auto si vacío)"></div>
    </div>
    <div style="margin-top:.4rem"><label>Descripción</label>
      <textarea name="descripcion" rows="2" placeholder="Descripción detallada de la propuesta" style="width:100%"></textarea>
    </div>
    <button type="submit" class="btn-sm" style="margin-top:.4rem">Crear propuesta</button>
  </form>
</details>

<div class="filtros">
  <a href="/propuestas"{{if eqStr .Filtro ""}} class="sel"{{end}}>Todas</a>
  <a href="/propuestas?estado=abierta"{{if eqStr .Filtro "abierta"}} class="sel"{{end}}>Abiertas</a>
  <a href="/propuestas?estado=consenso"{{if eqStr .Filtro "consenso"}} class="sel"{{end}}>Consenso</a>
  <a href="/propuestas?estado=backlog"{{if eqStr .Filtro "backlog"}} class="sel"{{end}}>Backlog</a>
  <a href="/propuestas?estado=rechazada"{{if eqStr .Filtro "rechazada"}} class="sel"{{end}}>Rechazadas</a>
</div>

{{range .Propuestas}}
<details class="card">
  <summary>
    <span style="color:#94a3b8;margin-right:.4rem">{{.Codigo}}</span>
    <span class="tag t-{{.Estado}}">{{.Estado}}</span>
    &nbsp;<strong>{{trunc .Titulo 65}}</strong>
    <span style="color:#94a3b8;font-weight:normal;font-size:.82em;margin-left:.5rem">· {{.PropuestoPor}} · {{.Fecha}}</span>
    <a href="/propuestas/{{.Codigo}}" style="float:right;font-size:.8em;font-weight:normal;color:#2563eb" onclick="event.stopPropagation()">Gestionar →</a>
  </summary>
  {{if .Descripcion}}<p style="color:#475569;font-size:.88em;margin:.6rem 0">{{.Descripcion}}</p>{{end}}
  {{if .Votos}}
  <table style="font-size:.82em"><thead><tr><th>Agente</th><th>Posición</th><th>Comentario</th></tr></thead><tbody>
  {{range .Votos}}
    <tr>
      <td><strong>{{.Agente}}</strong></td>
      <td><span class="tag t-{{.Posicion}}">{{.Posicion}}</span></td>
      <td style="color:#64748b">{{.Comentario}}</td>
    </tr>
  {{end}}
  </tbody></table>
  {{end}}
</details>
{{end}}
{{end}}
`

// ─── Propuesta: detalle + acciones ───────────────────────────────────────────

const webTplPropuestaDetalle = `{{define "content"}}
<a href="/propuestas" style="font-size:.85rem;color:#64748b">← volver a propuestas</a>
<h2 style="margin:.5rem 0">{{.P.Codigo}} — {{.P.Titulo}}</h2>
<p>
  <span class="tag t-{{.P.Estado}}">{{.P.Estado}}</span>
  <span class="tag" style="background:#e2e8f0;color:#475569">{{.P.Tipo}}</span>
  <span style="color:#64748b;font-size:.85rem;margin-left:.5rem">Propuesto por {{.P.PropuestoPor}} · {{.P.Fecha}}</span>
  {{if .P.CerradaFecha}}<span style="color:#94a3b8;font-size:.82rem;margin-left:.5rem">(cerrada {{.P.CerradaFecha}})</span>{{end}}
</p>
{{if .P.Descripcion}}<p style="color:#475569;background:#f8fafc;border:1px solid #e2e8f0;border-radius:.4rem;padding:.8rem 1rem;font-size:.9rem">{{.P.Descripcion}}</p>{{end}}

{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

{{$codigo := .P.Codigo}}
{{$estado := .P.Estado}}
{{$agentes := .Agentes}}

{{if eqStr $estado "abierta"}}
<div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;margin-bottom:1rem">
  <div class="action-box">
    <h4>Votar</h4>
    <form method="POST" action="/propuestas/{{$codigo}}/accion">
      <input type="hidden" name="accion" value="votar">
      <div style="display:flex;gap:.5rem;align-items:flex-end;flex-wrap:wrap">
        <div><label>Agente</label>
          <select name="agente">
            {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
          </select>
        </div>
        <div><label>Posición</label>
          <select name="posicion">
            <option value="acuerdo">Acuerdo</option>
            <option value="desacuerdo">Desacuerdo</option>
            <option value="abstencion">Abstención</option>
          </select>
        </div>
      </div>
      <div style="margin-top:.4rem"><label>Comentario (opcional)</label>
        <input type="text" name="comentario" placeholder="Justificación del voto" style="width:100%">
      </div>
      <button type="submit" class="btn-sm" style="margin-top:.5rem">Registrar voto</button>
    </form>
  </div>
  <div class="action-box">
    <h4>Cerrar propuesta (Alberto)</h4>
    <form method="POST" action="/propuestas/{{$codigo}}/accion">
      <input type="hidden" name="accion" value="cerrar">
      <div><label>Estado de cierre</label>
        <select name="estado_cierre">
          <option value="consenso">Consenso ✓</option>
          <option value="rechazada">Rechazada ✗</option>
          <option value="backlog">Backlog (aplazar)</option>
        </select>
      </div>
      <button type="submit" class="btn-sm" style="margin-top:.5rem">Cerrar propuesta</button>
    </form>
  </div>
</div>
{{end}}

<h4>Votos registrados</h4>
{{if .P.Votos}}
<table><thead><tr><th>Agente</th><th>Posición</th><th>Comentario</th></tr></thead><tbody>
{{range .P.Votos}}
  <tr>
    <td><strong>{{.Agente}}</strong></td>
    <td><span class="tag t-{{.Posicion}}">{{.Posicion}}</span></td>
    <td style="color:#64748b">{{.Comentario}}</td>
  </tr>
{{end}}
</tbody></table>
{{else}}
<p style="color:#94a3b8">No hay votos registrados aún.</p>
{{end}}
{{end}}
`
