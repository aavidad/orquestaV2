/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	appi18n "orquesta/i18n"
	"orquesta/internal/localrpc"
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
	Proyecto     string
	ProyectoSlug string
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

type webHistorialProyectoRow struct {
	Codigo     string
	Titulo     string
	Tipo       string
	Estado     string
	Fecha      string
	Acuerdo    int
	Desacuerdo int
	Abstencion int
	Pendiente  int
	Votos      []*db.Voto
}

type webHistorialProyectoData struct {
	Proyecto string
	Items    []webHistorialProyectoRow
	Msg      string
	Err      string
}

type webMemoriaRow struct {
	Proyecto        string
	Resumen         string
	ActualizadoPor  string
	Actualizado     string
	Fuentes         int
	Hallazgos       int
	Derivas         int
	DerivasAbiertas int
}

type webMemoriaData struct {
	Items []webMemoriaRow
	Msg   string
	Err   string
}

type webMemoriaDetalleData struct {
	M         *db.MemoriaProyecto
	Proyecto  string
	Fuentes   []*db.FuenteMemoria
	Hallazgos []*db.HallazgoMemoria
	Derivas   []*db.DerivaMemoria
	Msg       string
	Err       string
}

type webProgresoRow struct {
	Proyecto          string
	ProgresoPct       float64
	TareasTotales     int
	TareasCompletadas int
}

type webProgresoData struct {
	Items []webProgresoRow
	Msg   string
	Err   string
}

type webProgresoDetalleData struct {
	R   *db.ResumenProgresoProyecto
	Msg string
	Err string
}

// ─── FuncMap ─────────────────────────────────────────────────────────────────

var webFuncMap = template.FuncMap{
	"trunc": func(s string, n int) string {
		runes := []rune(s)
		if len(runes) <= n {
			return s
		}
		return string(runes[:n-1]) + "…"
	},
	"eqStr":   func(a, b string) bool { return a == b },
	"pathEsc": func(s string) string { return url.PathEscape(s) },
}

var webI18n = appi18n.NewBundle(appi18n.ResolveDir(), appi18n.DefaultLang)

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
	case len(parts) == 3 && parts[1] == "historial" && r.Method == http.MethodGet:
		webHandlerHistorialProyecto(w, r, parts[2])
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerPropuestaDetalle(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "accion" && r.Method == http.MethodPost:
		webHandlerPropuestaAccion(w, r, parts[1])
	default:
		http.NotFound(w, r)
	}
}

func webRouterMemoria(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/memoria", http.StatusSeeOther)
		return
	}

	switch {
	case parts[1] == "nueva" && r.Method == http.MethodPost:
		webHandlerMemoriaNueva(w, r)
	case len(parts) == 2 && r.Method == http.MethodGet:
		webHandlerMemoriaDetalle(w, r, parts[1])
	case len(parts) == 3 && parts[2] == "accion" && r.Method == http.MethodPost:
		webHandlerMemoriaAccion(w, r, parts[1])
	default:
		http.NotFound(w, r)
	}
}

func webRouterProgreso(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Redirect(w, r, "/progreso", http.StatusSeeOther)
		return
	}
	if len(parts) == 2 && r.Method == http.MethodGet {
		webHandlerProgresoDetalle(w, r, parts[1])
		return
	}
	http.NotFound(w, r)
}

func webAPIProgreso(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "api" || parts[1] != "progreso" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	resumen, err := db.CalcularResumenProgresoProyecto(parts[2])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resumen)
}

// ─── Dashboard ────────────────────────────────────────────────────────────────

func webHandlerDash(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	agentes, _ := db.ListarAgentes()
	counts, _ := db.ContarTareasPorEstado()
	total, completadas := 0, 0
	for est, n := range counts {
		total += n
		if est == "completada" {
			completadas = n
		}
	}
	pct := 0
	if total > 0 {
		pct = completadas * 100 / total
	}
	estadoAb := db.PropuestaAbierta
	abiertas, _ := db.ListarPropuestas(&estadoAb)
	var resAbiertas []webPropResumen
	for _, p := range abiertas {
		ac, des, _, pend, _ := db.ContarVotos(p.ID)
		resAbiertas = append(resAbiertas, webPropResumen{
			Codigo: p.Codigo, Titulo: p.Titulo,
			Acuerdo: ac, Desacuerdo: des, Pendiente: pend,
		})
	}
	estadoEP := db.EstadoEnProgreso
	tareas, _ := db.ListarTareas(db.FiltroTareas{Estado: &estadoEP})
	var ep []webTareaRow
	for _, t := range tareas {
		ep = append(ep, toWebTarea(t))
	}
	webRender(w, r, webTplLayout+webTplDash, webDashData{
		Agentes: agentes, Counts: counts, Total: total,
		Completadas: completadas, Pct: pct,
		Abiertas: resAbiertas, EnProgreso: ep,
		Generado: time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ─── Tareas: lista ────────────────────────────────────────────────────────────

func webHandlerTareas(w http.ResponseWriter, r *http.Request) {
	filtro := r.URL.Query().Get("estado")
	msg := r.URL.Query().Get("ok")
	errMsg := r.URL.Query().Get("err")

	var f db.FiltroTareas
	if filtro != "" {
		e := db.EstadoTarea(filtro)
		f.Estado = &e
	}
	tareas, _ := db.ListarTareas(f)
	var wt []webTareaRow
	for _, t := range tareas {
		wt = append(wt, toWebTarea(t))
	}
	agentes, _ := db.ListarAgentes()
	webRender(w, r, webTplLayout+webTplTareas, webTareasData{
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
	t := &db.Tarea{
		Titulo:      titulo,
		Descripcion: r.FormValue("descripcion"),
		Modulo:      r.FormValue("modulo"),
		Prioridad:   db.PrioridadTarea(r.FormValue("prioridad")),
		CreadoPor:   "alberto",
	}
	if p := strings.TrimSpace(r.FormValue("propuesta")); p != "" {
		prop, err := db.GetPropuesta(p)
		if err == nil {
			t.PropuestaID = &prop.ID
		}
	}
	id, err := db.CrearTarea(t)
	if err != nil {
		http.Redirect(w, r, "/tareas?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	if agente := strings.TrimSpace(r.FormValue("agente")); agente != "" {
		_ = db.TomarTarea(id, agente)
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
	t, err := db.GetTarea(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	agentes, _ := db.ListarAgentes()
	webRender(w, r, webTplLayout+webTplTareaDetalle, webTareaDetalleData{
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
		err = db.TomarTarea(id, agente)
	case "iniciar":
		err = db.IniciarTarea(id, agente)
	case "completar":
		err = db.CompletarTarea(id, agente, r.FormValue("commit"))
	case "bloquear":
		err = db.BloquearTarea(id, agente, r.FormValue("motivo"))
	case "desbloquear":
		err = db.DesbloquearTarea(id, agente, r.FormValue("resolucion"))
	case "nota":
		err = db.AnotarTarea(id, agente, r.FormValue("nota"))
	case "backlog":
		err = db.MoverTareaABacklog(id)
	case "reasignar":
		nuevoAgente := strings.TrimSpace(r.FormValue("nuevo_agente"))
		err = db.ReasignarTarea(id, nuevoAgente)
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
	proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	msg := r.URL.Query().Get("ok")
	errMsg := r.URL.Query().Get("err")

	var estadoPtr *db.EstadoPropuesta
	if filtro != "" {
		e := db.EstadoPropuesta(filtro)
		estadoPtr = &e
	}
	var (
		props []*db.Propuesta
		err   error
	)
	if proyecto != "" {
		props, err = db.ListarPropuestasProyecto(proyecto, estadoPtr)
	} else {
		props, err = db.ListarPropuestas(estadoPtr)
	}
	if err != nil {
		webRender(w, r, webTplLayout+webTplPropuestas, webPropData{Filtro: filtro, Msg: msg, Err: err.Error()})
		return
	}
	var detalles []webPropDetalle
	for _, p := range props {
		votos, _ := db.VotosDePropuesta(p.ID)
		cerrada := ""
		if p.CerradaAt != nil {
			cerrada = p.CerradaAt.Format("2006-01-02")
		}
		detalles = append(detalles, webPropDetalle{
			Codigo: p.Codigo, Titulo: p.Titulo, Descripcion: p.Descripcion,
			Tipo: p.Tipo, Estado: string(p.Estado), Proyecto: p.Proyecto, ProyectoSlug: p.ProyectoSlug, PropuestoPor: p.PropuestoPor,
			Fecha: p.CreatedAt.Format("2006-01-02"), CerradaFecha: cerrada,
			Votos: votos,
		})
	}
	webRender(w, r, webTplLayout+webTplPropuestas, webPropData{
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
	p := &db.Propuesta{
		Codigo:       strings.TrimSpace(r.FormValue("codigo")),
		Titulo:       titulo,
		Descripcion:  r.FormValue("descripcion"),
		Tipo:         r.FormValue("tipo"),
		ProyectoSlug: strings.TrimSpace(r.FormValue("proyecto")),
		PropuestoPor: "alberto",
		Distribuidor: "alberto",
	}
	_, err := db.CrearPropuesta(p)
	if err != nil {
		http.Redirect(w, r, "/propuestas?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/propuestas/"+p.Codigo+"?ok="+url.QueryEscape("Propuesta "+p.Codigo+" creada"), http.StatusSeeOther)
}

// ─── Propuestas: detalle ──────────────────────────────────────────────────────

func webHandlerPropuestaDetalle(w http.ResponseWriter, r *http.Request, codigo string) {
	p, err := db.GetPropuesta(codigo)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	votos, _ := db.VotosDePropuesta(p.ID)
	cerrada := ""
	if p.CerradaAt != nil {
		cerrada = p.CerradaAt.Format("2006-01-02")
	}
	agentes, _ := db.ListarAgentes()
	webRender(w, r, webTplLayout+webTplPropuestaDetalle, webPropDetalleData{
		P: webPropDetalle{
			Codigo: p.Codigo, Titulo: p.Titulo, Descripcion: p.Descripcion,
			Tipo: p.Tipo, Estado: string(p.Estado), Proyecto: p.Proyecto, ProyectoSlug: p.ProyectoSlug, PropuestoPor: p.PropuestoPor,
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

	p, err := db.GetPropuesta(codigo)
	if err != nil {
		http.Redirect(w, r, "/propuestas?err=Propuesta+no+encontrada", http.StatusSeeOther)
		return
	}

	switch accion {
	case "votar":
		agente := strings.TrimSpace(r.FormValue("agente"))
		posicion := db.PosicionVoto(r.FormValue("posicion"))
		comentario := r.FormValue("comentario")
		_, err = db.Votar(p.ID, agente, posicion, comentario)
	case "cerrar":
		nuevoEstado := r.FormValue("estado_cierre")
		err = db.CerrarPropuesta(codigo, nuevoEstado, "alberto")
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

func webHandlerHistorialProyecto(w http.ResponseWriter, r *http.Request, proyecto string) {
	historial, err := db.ListarHistorialVotacionesProyecto(proyecto)
	if err != nil {
		webRender(w, r, webTplLayout+webTplHistorialProyecto, webHistorialProyectoData{
			Proyecto: proyecto,
			Err:      err.Error(),
		})
		return
	}
	if len(historial) == 0 {
		webRender(w, r, webTplLayout+webTplHistorialProyecto, webHistorialProyectoData{
			Proyecto: proyecto,
			Err:      "No hay propuestas para ese proyecto.",
		})
		return
	}

	items := make([]webHistorialProyectoRow, 0, len(historial))
	nombreProyecto := historial[0].Propuesta.Proyecto
	if strings.TrimSpace(nombreProyecto) == "" {
		nombreProyecto = proyecto
	}
	for _, item := range historial {
		p := item.Propuesta
		items = append(items, webHistorialProyectoRow{
			Codigo:     p.Codigo,
			Titulo:     p.Titulo,
			Tipo:       p.Tipo,
			Estado:     string(p.Estado),
			Fecha:      p.CreatedAt.Format("2006-01-02"),
			Acuerdo:    item.Acuerdo,
			Desacuerdo: item.Desacuerdo,
			Abstencion: item.Abstencion,
			Pendiente:  item.Pendiente,
			Votos:      item.Votos,
		})
	}
	webRender(w, r, webTplLayout+webTplHistorialProyecto, webHistorialProyectoData{
		Proyecto: nombreProyecto,
		Items:    items,
	})
}

// ─── Memoria: lista ───────────────────────────────────────────────────────────

func webHandlerMemoria(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("ok")
	errMsg := r.URL.Query().Get("err")

	memorias, err := db.ListarMemoriaProyectos()
	if err != nil {
		webRender(w, r, webTplLayout+webTplMemoria, webMemoriaData{Msg: msg, Err: err.Error()})
		return
	}
	var items []webMemoriaRow
	for _, m := range memorias {
		fuentes, hallazgos, derivas, abiertas, _ := db.ContarMemoriaProyecto(m.Proyecto)
		items = append(items, webMemoriaRow{
			Proyecto:        m.Proyecto,
			Resumen:         m.Resumen,
			ActualizadoPor:  m.ActualizadoPor,
			Actualizado:     m.UpdatedAt.Format("2006-01-02 15:04"),
			Fuentes:         fuentes,
			Hallazgos:       hallazgos,
			Derivas:         derivas,
			DerivasAbiertas: abiertas,
		})
	}
	webRender(w, r, webTplLayout+webTplMemoria, webMemoriaData{Items: items, Msg: msg, Err: errMsg})
}

func webHandlerProgreso(w http.ResponseWriter, r *http.Request) {
	proyectos, err := db.ListarProyectosConProgreso()
	if err != nil {
		webRender(w, r, webTplLayout+webTplProgreso, webProgresoData{Err: err.Error()})
		return
	}
	var items []webProgresoRow
	for _, proyecto := range proyectos {
		resumen, err := db.CalcularResumenProgresoProyecto(proyecto)
		if err != nil {
			continue
		}
		items = append(items, webProgresoRow{
			Proyecto:          proyecto,
			ProgresoPct:       resumen.ProgresoPct,
			TareasTotales:     resumen.TareasTotales,
			TareasCompletadas: resumen.TareasCompletadas,
		})
	}
	webRender(w, r, webTplLayout+webTplProgreso, webProgresoData{
		Items: items,
		Msg:   r.URL.Query().Get("ok"),
		Err:   r.URL.Query().Get("err"),
	})
}

func webHandlerProgresoDetalle(w http.ResponseWriter, r *http.Request, proyecto string) {
	resumen, err := db.CalcularResumenProgresoProyecto(proyecto)
	if err != nil {
		webRender(w, r, webTplLayout+webTplProgresoDetalle, webProgresoDetalleData{Err: err.Error()})
		return
	}
	webRender(w, r, webTplLayout+webTplProgresoDetalle, webProgresoDetalleData{
		R:   resumen,
		Msg: r.URL.Query().Get("ok"),
		Err: r.URL.Query().Get("err"),
	})
}

func webHandlerMemoriaNueva(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	proyecto := strings.TrimSpace(r.FormValue("proyecto"))
	if proyecto == "" {
		http.Redirect(w, r, "/memoria?err="+url.QueryEscape("El proyecto es obligatorio"), http.StatusSeeOther)
		return
	}
	_, err := db.GuardarMemoriaProyecto(&db.MemoriaProyecto{
		Proyecto:          proyecto,
		Resumen:           strings.TrimSpace(r.FormValue("resumen")),
		Contexto:          strings.TrimSpace(r.FormValue("contexto")),
		PreguntasAbiertas: strings.TrimSpace(r.FormValue("preguntas")),
		ActualizadoPor:    strings.TrimSpace(defaultIfEmpty(r.FormValue("agente"), "alberto")),
	})
	if err != nil {
		http.Redirect(w, r, "/memoria?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/memoria/"+url.PathEscape(proyecto)+"?ok="+url.QueryEscape("Memoria de "+proyecto+" creada"), http.StatusSeeOther)
}

func webHandlerMemoriaDetalle(w http.ResponseWriter, r *http.Request, proyecto string) {
	memoria, err := db.GetMemoriaProyecto(proyecto)
	if err != nil && err != sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	fuentes, _ := db.ListarFuentesMemoria(proyecto)
	hallazgos, _ := db.ListarHallazgosMemoria(proyecto)
	derivas, _ := db.ListarDerivasMemoria(proyecto, false)

	webRender(w, r, webTplLayout+webTplMemoriaDetalle, webMemoriaDetalleData{
		M:         memoria,
		Proyecto:  proyecto,
		Fuentes:   fuentes,
		Hallazgos: hallazgos,
		Derivas:   derivas,
		Msg:       r.URL.Query().Get("ok"),
		Err:       r.URL.Query().Get("err"),
	})
}

func webHandlerMemoriaAccion(w http.ResponseWriter, r *http.Request, proyecto string) {
	_ = r.ParseForm()
	accion := r.FormValue("accion")
	back := "/memoria/" + url.PathEscape(proyecto)
	agente := strings.TrimSpace(defaultIfEmpty(r.FormValue("agente"), "alberto"))

	var err error
	switch accion {
	case "guardar":
		actual, getErr := db.GetMemoriaProyecto(proyecto)
		if getErr != nil && getErr != sql.ErrNoRows {
			http.Redirect(w, r, back+"?err="+url.QueryEscape(getErr.Error()), http.StatusSeeOther)
			return
		}
		resumen := strings.TrimSpace(r.FormValue("resumen"))
		contexto := strings.TrimSpace(r.FormValue("contexto"))
		preguntas := strings.TrimSpace(r.FormValue("preguntas"))
		if actual != nil {
			if resumen == "" {
				resumen = actual.Resumen
			}
			if contexto == "" {
				contexto = actual.Contexto
			}
			if preguntas == "" {
				preguntas = actual.PreguntasAbiertas
			}
		}
		_, err = db.GuardarMemoriaProyecto(&db.MemoriaProyecto{
			Proyecto:          proyecto,
			Resumen:           resumen,
			Contexto:          contexto,
			PreguntasAbiertas: preguntas,
			ActualizadoPor:    agente,
		})
	case "fuente":
		_, err = db.RegistrarFuenteMemoria(&db.FuenteMemoria{
			Proyecto:      proyecto,
			Tipo:          strings.TrimSpace(defaultIfEmpty(r.FormValue("tipo"), "documentacion")),
			Referencia:    strings.TrimSpace(r.FormValue("referencia")),
			Titulo:        strings.TrimSpace(r.FormValue("titulo")),
			URL:           strings.TrimSpace(r.FormValue("url")),
			Confianza:     strings.TrimSpace(defaultIfEmpty(r.FormValue("confianza"), "media")),
			Detalle:       strings.TrimSpace(r.FormValue("detalle")),
			RegistradoPor: agente,
		})
	case "hallazgo":
		fuenteID, idErr := parseIDForm(r.FormValue("fuente_id"))
		if idErr != nil {
			err = idErr
			break
		}
		_, err = db.RegistrarHallazgoMemoria(&db.HallazgoMemoria{
			Proyecto:      proyecto,
			FuenteID:      fuenteID,
			Tipo:          strings.TrimSpace(defaultIfEmpty(r.FormValue("tipo"), "hecho")),
			Titulo:        strings.TrimSpace(r.FormValue("titulo")),
			Descripcion:   strings.TrimSpace(r.FormValue("descripcion")),
			Impacto:       strings.TrimSpace(defaultIfEmpty(r.FormValue("impacto"), "medio")),
			Confianza:     strings.TrimSpace(defaultIfEmpty(r.FormValue("confianza"), "media")),
			RegistradoPor: agente,
		})
	case "deriva":
		hallazgoID, idErr := parseIDForm(r.FormValue("hallazgo_id"))
		if idErr != nil {
			err = idErr
			break
		}
		_, err = db.RegistrarDerivaMemoria(&db.DerivaMemoria{
			Proyecto:     proyecto,
			HallazgoID:   hallazgoID,
			Tipo:         strings.TrimSpace(defaultIfEmpty(r.FormValue("tipo"), "documental")),
			Severidad:    strings.TrimSpace(defaultIfEmpty(r.FormValue("severidad"), "media")),
			Estado:       db.EstadoDeriva(strings.TrimSpace(defaultIfEmpty(r.FormValue("estado"), string(db.DerivaAbierta)))),
			Descripcion:  strings.TrimSpace(r.FormValue("descripcion")),
			Evidencia:    strings.TrimSpace(r.FormValue("evidencia")),
			DetectadaPor: agente,
		})
	case "resolver-deriva":
		id, idErr := strconv.ParseInt(strings.TrimSpace(r.FormValue("deriva_id")), 10, 64)
		if idErr != nil {
			err = fmt.Errorf("deriva_id inválido")
			break
		}
		err = db.ResolverDerivaMemoria(id, agente, strings.TrimSpace(defaultIfEmpty(r.FormValue("estado"), string(db.DerivaResuelta))), strings.TrimSpace(r.FormValue("resolucion")))
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

func webResolveLang(r *http.Request) string {
	if r == nil {
		return webI18n.ResolveLang("")
	}
	if lang := strings.TrimSpace(r.URL.Query().Get("lang")); lang != "" && webI18n.HasLang(lang) {
		return webI18n.ResolveLang(lang)
	}
	if c, err := r.Cookie("orquesta_lang"); err == nil && webI18n.HasLang(c.Value) {
		return webI18n.ResolveLang(c.Value)
	}
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		if lang := strings.TrimSpace(strings.Split(part, ";")[0]); webI18n.HasLang(lang) {
			return webI18n.ResolveLang(lang)
		}
	}
	return webI18n.ResolveLang("")
}

func webLangURL(r *http.Request, lang string) string {
	if r == nil {
		return "/"
	}
	q := r.URL.Query()
	q.Set("lang", webI18n.ResolveLang(lang))
	if len(q) == 0 {
		return r.URL.Path
	}
	return r.URL.Path + "?" + q.Encode()
}

func webFuncMapForRequest(r *http.Request) template.FuncMap {
	lang := webResolveLang(r)
	funcs := template.FuncMap{}
	for k, v := range webFuncMap {
		funcs[k] = v
	}
	funcs["t"] = func(key string) string {
		return webI18n.T(lang, key)
	}
	funcs["lang"] = func() string {
		return lang
	}
	funcs["langs"] = func() []string {
		return webI18n.Languages()
	}
	funcs["langURL"] = func(code string) string {
		return webLangURL(r, code)
	}
	funcs["langName"] = func(code string) string {
		return webI18n.T(lang, "lang."+webI18n.ResolveLang(code))
	}
	return funcs
}

func webRender(w http.ResponseWriter, r *http.Request, tplStr string, data any) {
	if r != nil {
		if requested := strings.TrimSpace(r.URL.Query().Get("lang")); requested != "" && webI18n.HasLang(requested) {
			http.SetCookie(w, &http.Cookie{
				Name:     "orquesta_lang",
				Value:    webI18n.ResolveLang(requested),
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   86400 * 365,
			})
		}
	}
	tmpl, err := template.New("layout").Funcs(webFuncMapForRequest(r)).Parse(tplStr)
	if err != nil {
		http.Error(w, "Error de plantilla: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

func defaultIfEmpty(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func parseIDForm(v string) (*int64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("id relacionado inválido")
	}
	if id <= 0 {
		return nil, nil
	}
	return &id, nil
}

// ─── Comando ──────────────────────────────────────────────────────────────────

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Arranca el panel web (por defecto: http://localhost:8080)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := webI18n.Reload(); err != nil {
			return err
		}
		if policy, err := db.GetLanguagePolicy(); err == nil {
			webI18n.SetDefaultLang(policy.DefaultLanguage)
		}
		puerto, _ := cmd.Flags().GetInt("puerto")
		addr := fmt.Sprintf(":%d", puerto)
		info := localrpc.ServerInfo{
			Addr:      "127.0.0.1" + addr,
			PID:       os.Getpid(),
			Kind:      "web",
			ScopeID:   localrpc.CurrentScopeID(),
			DBPath:    db.CurrentDBPath(),
			StartedAt: time.Now().UTC(),
			Version:   "dev",
		}
		mux := http.NewServeMux()
		registerRPCHandlers(mux, info)
		mux.HandleFunc("/", webHandlerDash)
		mux.HandleFunc("/tareas", webHandlerTareas)
		mux.HandleFunc("/tareas/", webRouterTareas)
		mux.HandleFunc("/propuestas", webHandlerPropuestas)
		mux.HandleFunc("/propuestas/", webRouterPropuestas)
		mux.HandleFunc("/memoria", webHandlerMemoria)
		mux.HandleFunc("/memoria/", webRouterMemoria)
		mux.HandleFunc("/progreso", webHandlerProgreso)
		mux.HandleFunc("/progreso/", webRouterProgreso)
		mux.HandleFunc("/api/progreso/", webAPIProgreso)
		if err := localrpc.SaveServerInfo(info); err != nil {
			return err
		}
		defer localrpc.RemoveState("")
		fmt.Printf("✓ Panel web en http://localhost%s\n", addr)
		fmt.Println("  Ctrl+C para detener.")
		return http.ListenAndServe(addr, mux)
	},
}

func init() {
	serveCmd.Flags().Int("puerto", 8080, "Puerto HTTP")
	rootCmd.AddCommand(serveCmd)
}

// ─────────────────────────────────────────────────────────────────────────────
// PLANTILLAS HTML
// ─────────────────────────────────────────────────────────────────────────────

const webTplLayout = `<!doctype html>
<html lang="{{lang}}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{t "app.title"}}</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">
  <style>
    body{--pico-font-size:14px}
    header{background:#0f172a;padding:.7rem 0;margin-bottom:0}
    header nav{display:flex;align-items:center;justify-content:space-between}
    header a{color:#94a3b8;text-decoration:none;margin-right:1.2rem;font-size:.95rem}
    header a:hover,header a.sel{color:#fff}
    header strong{color:#fff}
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
    @media(max-width:700px){.grid2{grid-template-columns:1fr}}
  </style>
</head>
<body>
<header>
  <nav class="container">
    <div><a href="/"><strong>⚙ {{t "app.name"}}</strong></a></div>
    <div>
      <a href="/">{{t "Dashboard"}}</a>
      <a href="/tareas">{{t "Tareas"}}</a>
      <a href="/propuestas">{{t "Propuestas"}}</a>
      <a href="/memoria">{{t "Memoria"}}</a>
      <a href="/progreso">{{t "Progreso"}}</a>
      <span style="margin-left:.6rem;color:#cbd5e1;font-size:.82rem">{{t "Idioma"}}:</span>
      {{range langs}}<a href="{{langURL .}}"{{if eqStr . (lang)}} class="sel"{{end}} style="margin-right:.4rem">{{langName .}}</a>{{end}}
    </div>
  </nav>
</header>
<main class="container" style="padding-top:1.5rem;padding-bottom:2rem">
{{template "content" .}}
</main>
<footer>{{t "ContaGrx footer"}}</footer>
</body></html>
`

// ─── Dashboard ────────────────────────────────────────────────────────────────

const webTplDash = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1.2rem">
  <h2 style="margin:0">{{t "Estado del Proyecto"}}</h2>
  <small style="color:#94a3b8">{{.Generado}} · {{t "auto-refresh"}}</small>
</div>
<meta http-equiv="refresh" content="30">
<div class="stats">
  <div class="stat"><div class="n">{{.Total}}</div><div class="l">{{t "tareas totales"}}</div></div>
  <div class="stat"><div class="n" style="color:#16a34a">{{.Completadas}}</div><div class="l">{{t "completadas"}}</div></div>
  <div class="stat"><div class="n" style="color:#7c3aed">{{index .Counts "en_progreso"}}</div><div class="l">{{t "en_progreso"}}</div></div>
  <div class="stat"><div class="n" style="color:#2563eb">{{index .Counts "asignada"}}</div><div class="l">{{t "asignadas"}}</div></div>
  <div class="stat"><div class="n" style="color:#dc2626">{{index .Counts "bloqueada"}}</div><div class="l">{{t "bloqueadas"}}</div></div>
  <div class="stat"><div class="n">{{len .Abiertas}}</div><div class="l">{{t "props. abiertas"}}</div></div>
</div>
<div class="pgbar">
  <div class="pgfill" style="width:{{if gt .Pct 0}}{{.Pct}}%{{else}}2.5rem{{end}}">{{.Pct}}%</div>
</div>
<div style="display:grid;grid-template-columns:220px 1fr;gap:1.5rem;align-items:start">
  <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.5rem;padding:.8rem 1rem">
    <h4 style="margin:0 0 .7rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">{{t "Agentes"}}</h4>
    <table style="width:100%"><tbody>
    {{range .Agentes}}{{if .Habilitado}}
      <tr style="border-bottom:1px solid #f1f5f9">
        <td style="width:1.2rem;padding:.3rem 0">{{if .Activo}}<span class="dot-on">●</span>{{else}}<span class="dot-off">○</span>{{end}}</td>
        <td style="padding:.3rem .3rem">
          <div style="font-weight:600;font-size:.85rem">{{.Nombre}}</div>
          <div style="font-size:.72rem;color:#94a3b8">{{.Rol}}</div>
        </td>
        <td style="text-align:right;padding:.3rem 0">
          {{if .EstadoSesion}}<span class="es-{{.EstadoSesion}}">{{t .EstadoSesion}}</span>{{end}}
        </td>
      </tr>
    {{end}}{{end}}
    </tbody></table>
  </div>
  <div>
    {{if .EnProgreso}}
    <h4 style="margin:0 0 .5rem 0">{{t "En progreso"}}</h4>
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
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">{{t "Estado"}} <a href="/tareas" style="font-size:.9em;font-weight:normal;text-transform:none">{{t "Ver"}} →</a></h4>
        <table><tbody>
        {{range $est,$n := .Counts}}{{if gt $n 0}}
          <tr><td style="padding:.2rem .4rem"><span class="tag t-{{$est}}">{{t $est}}</span></td><td style="padding:.2rem .6rem"><strong>{{$n}}</strong></td></tr>
        {{end}}{{end}}
        </tbody></table>
      </div>
      {{if .Abiertas}}
      <div>
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">{{t "Propuestas abiertas"}}</h4>
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
  <h2 style="margin:0">{{t "Tareas"}} <small style="font-size:.5em;color:#94a3b8">{{len .Tareas}}</small></h2>
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
  <a href="/tareas"{{if eqStr .Filtro ""}} class="sel"{{end}}>{{t "Todas"}}</a>
  <a href="/tareas?estado=en_progreso"{{if eqStr .Filtro "en_progreso"}} class="sel"{{end}}>{{t "En progreso"}}</a>
  <a href="/tareas?estado=asignada"{{if eqStr .Filtro "asignada"}} class="sel"{{end}}>{{t "Asignadas"}}</a>
  <a href="/tareas?estado=libre"{{if eqStr .Filtro "libre"}} class="sel"{{end}}>{{t "Libres"}}</a>
  <a href="/tareas?estado=bloqueada"{{if eqStr .Filtro "bloqueada"}} class="sel"{{end}}>{{t "Bloqueadas"}}</a>
  <a href="/tareas?estado=backlog"{{if eqStr .Filtro "backlog"}} class="sel"{{end}}>Backlog</a>
  <a href="/tareas?estado=completada"{{if eqStr .Filtro "completada"}} class="sel"{{end}}>{{t "Completadas"}}</a>
</div>

{{if .Tareas}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>#</th><th>{{t "Estado"}}</th><th>{{t "Prioridad"}}</th><th>Modulo</th><th>{{t "Agentes"}}</th><th>Titulo</th><th></th></tr></thead>
  <tbody>
  {{range .Tareas}}
    <tr>
      <td style="color:#94a3b8">{{.ID}}</td>
      <td><span class="tag t-{{.Estado}}">{{t .Estado}}</span></td>
      <td><span class="tag t-{{.Prioridad}}">{{t .Prioridad}}</span></td>
      <td><code style="font-size:.8em">{{.Modulo}}</code></td>
      <td>{{.Agente}}</td>
      <td>{{.Titulo}}</td>
      <td><a href="/tareas/{{.ID}}" class="btn-sm">{{t "Ver"}} →</a></td>
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
<a href="/tareas" style="font-size:.85rem;color:#64748b">← {{t "Volver"}} {{t "Tareas"}}</a>
<h2 style="margin:.5rem 0">#{{.T.ID}} — {{.T.Titulo}}</h2>
<p>
  <span class="tag t-{{.T.Estado}}">{{t .T.Estado}}</span>
  <span class="tag t-{{.T.Prioridad}}">{{t .T.Prioridad}}</span>
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
  <h2 style="margin:0">{{t "Propuestas (OPs)"}} <small style="font-size:.5em;color:#94a3b8">{{len .Propuestas}}</small></h2>
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
    <div style="margin-top:.4rem"><label>Proyecto</label>
      <input type="text" name="proyecto" placeholder="orquestador">
    </div>
    <div style="margin-top:.4rem"><label>Descripción</label>
      <textarea name="descripcion" rows="2" placeholder="Descripción detallada de la propuesta" style="width:100%"></textarea>
    </div>
    <button type="submit" class="btn-sm" style="margin-top:.4rem">Crear propuesta</button>
  </form>
</details>

<div class="filtros">
  <a href="/propuestas"{{if eqStr .Filtro ""}} class="sel"{{end}}>{{t "Todas"}}</a>
  <a href="/propuestas?estado=abierta"{{if eqStr .Filtro "abierta"}} class="sel"{{end}}>{{t "abierta"}}</a>
  <a href="/propuestas?estado=consenso"{{if eqStr .Filtro "consenso"}} class="sel"{{end}}>{{t "consenso"}}</a>
  <a href="/propuestas?estado=backlog"{{if eqStr .Filtro "backlog"}} class="sel"{{end}}>Backlog</a>
  <a href="/propuestas?estado=rechazada"{{if eqStr .Filtro "rechazada"}} class="sel"{{end}}>{{t "rechazada"}}</a>
</div>

{{range .Propuestas}}
<details class="card">
  <summary>
    <span style="color:#94a3b8;margin-right:.4rem">{{.Codigo}}</span>
    <span class="tag t-{{.Estado}}">{{t .Estado}}</span>
    &nbsp;<strong>{{trunc .Titulo 65}}</strong>
    <span style="color:#94a3b8;font-weight:normal;font-size:.82em;margin-left:.5rem">· {{.PropuestoPor}} · {{.Fecha}}</span>
    {{if .ProyectoSlug}}<a href="/propuestas/historial/{{pathEsc .ProyectoSlug}}" style="margin-left:.5rem;font-size:.78em;color:#2563eb" onclick="event.stopPropagation()">historial {{.Proyecto}}</a>{{end}}
    <a href="/propuestas/{{.Codigo}}" style="float:right;font-size:.8em;font-weight:normal;color:#2563eb" onclick="event.stopPropagation()">Gestionar →</a>
  </summary>
  {{if .Descripcion}}<p style="color:#475569;font-size:.88em;margin:.6rem 0">{{.Descripcion}}</p>{{end}}
  {{if .Proyecto}}<p style="margin:.2rem 0 .6rem 0;font-size:.82em;color:#64748b">Proyecto: <strong>{{.Proyecto}}</strong></p>{{end}}
  {{if .Votos}}
  <table style="font-size:.82em"><thead><tr><th>Agente</th><th>Posición</th><th>Comentario</th></tr></thead><tbody>
  {{range .Votos}}
    <tr>
      <td><strong>{{.Agente}}</strong></td>
      <td><span class="tag t-{{.Posicion}}">{{t .Posicion}}</span></td>
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
<a href="/propuestas" style="font-size:.85rem;color:#64748b">← {{t "Volver"}} {{t "Propuestas"}}</a>
<h2 style="margin:.5rem 0">{{.P.Codigo}} — {{.P.Titulo}}</h2>
<p>
  <span class="tag t-{{.P.Estado}}">{{t .P.Estado}}</span>
  <span class="tag" style="background:#e2e8f0;color:#475569">{{t .P.Tipo}}</span>
  <span style="color:#64748b;font-size:.85rem;margin-left:.5rem">Propuesto por {{.P.PropuestoPor}} · {{.P.Fecha}}</span>
  {{if .P.CerradaFecha}}<span style="color:#94a3b8;font-size:.82rem;margin-left:.5rem">(cerrada {{.P.CerradaFecha}})</span>{{end}}
</p>
{{if .P.Proyecto}}<p style="margin-top:-.4rem"><a href="/propuestas/historial/{{pathEsc .P.ProyectoSlug}}" style="font-size:.85rem">Ver historial del proyecto {{.P.Proyecto}}</a></p>{{end}}
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
            <option value="acuerdo">{{t "acuerdo"}}</option>
            <option value="desacuerdo">{{t "desacuerdo"}}</option>
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
    <td><span class="tag t-{{.Posicion}}">{{t .Posicion}}</span></td>
    <td style="color:#64748b">{{.Comentario}}</td>
  </tr>
{{end}}
</tbody></table>
{{else}}
<p style="color:#94a3b8">No hay votos registrados aún.</p>
{{end}}
{{end}}
`

const webTplHistorialProyecto = `{{define "content"}}
<a href="/propuestas" style="font-size:.85rem;color:#64748b">← {{t "Volver"}} {{t "Propuestas"}}</a>
<div style="display:flex;justify-content:space-between;align-items:baseline;margin:.5rem 0 1rem 0">
  <h2 style="margin:0">Historial de votaciones <small style="font-size:.5em;color:#94a3b8">{{.Proyecto}}</small></h2>
</div>
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

{{range .Items}}
<details class="card">
  <summary>
    <span style="color:#94a3b8;margin-right:.4rem">{{.Codigo}}</span>
    <span class="tag t-{{.Estado}}">{{t .Estado}}</span>
    &nbsp;<strong>{{trunc .Titulo 65}}</strong>
    <span style="color:#94a3b8;font-weight:normal;font-size:.82em;margin-left:.5rem">· {{.Tipo}} · {{.Fecha}}</span>
    <span style="float:right;font-size:.82em;color:#475569">✓{{.Acuerdo}} · ✗{{.Desacuerdo}} · ～{{.Abstencion}} · ⏳{{.Pendiente}}</span>
  </summary>
  {{if .Votos}}
  <table style="font-size:.82em"><thead><tr><th>Agente</th><th>Posición</th><th>Comentario</th></tr></thead><tbody>
  {{range .Votos}}
    <tr>
      <td><strong>{{.Agente}}</strong></td>
      <td><span class="tag t-{{.Posicion}}">{{t .Posicion}}</span></td>
      <td style="color:#64748b">{{.Comentario}}</td>
    </tr>
  {{end}}
  </tbody></table>
  {{else}}
  <p style="color:#94a3b8">No hay votos registrados aún.</p>
  {{end}}
</details>
{{end}}
{{end}}
`

// ─── Memoria: lista y detalle ────────────────────────────────────────────────

const webTplMemoria = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1rem">
  <h2 style="margin:0">{{t "Memoria de proyecto"}} <small style="font-size:.5em;color:#94a3b8">{{len .Items}}</small></h2>
</div>
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

<details class="form-panel">
  <summary>＋ Nueva memoria de proyecto</summary>
  <form method="POST" action="/memoria/nueva" style="margin-top:.8rem">
    <div style="display:grid;grid-template-columns:1fr 1fr;gap:.5rem">
      <div><label>Proyecto *</label><input type="text" name="proyecto" required placeholder="orquestador"></div>
      <div><label>Agente</label><input type="text" name="agente" placeholder="Codex3"></div>
    </div>
    <div style="margin-top:.4rem"><label>Resumen</label><textarea name="resumen" rows="2" style="width:100%"></textarea></div>
    <div style="margin-top:.4rem"><label>Contexto</label><textarea name="contexto" rows="2" style="width:100%"></textarea></div>
    <div style="margin-top:.4rem"><label>Preguntas abiertas</label><textarea name="preguntas" rows="2" style="width:100%"></textarea></div>
    <button type="submit" class="btn-sm" style="margin-top:.4rem">Crear memoria</button>
  </form>
</details>

{{if .Items}}
<table>
  <thead><tr><th>Proyecto</th><th>Resumen</th><th>Fuentes</th><th>Hallazgos</th><th>Derivas</th><th>Abiertas</th><th>Actualizado</th><th></th></tr></thead>
  <tbody>
  {{range .Items}}
    <tr>
      <td><strong>{{.Proyecto}}</strong></td>
      <td style="font-size:.84rem;color:#475569">{{trunc .Resumen 72}}</td>
      <td>{{.Fuentes}}</td>
      <td>{{.Hallazgos}}</td>
      <td>{{.Derivas}}</td>
      <td>{{if gt .DerivasAbiertas 0}}<span class="tag t-media">{{.DerivasAbiertas}}</span>{{else}}<span class="tag t-consenso">0</span>{{end}}</td>
      <td style="font-size:.82rem;color:#64748b">{{.Actualizado}} · {{.ActualizadoPor}}</td>
      <td><a href="/memoria/{{pathEsc .Proyecto}}" class="btn-sm">Abrir →</a></td>
    </tr>
  {{end}}
  </tbody>
</table>
{{else}}
<p style="color:#94a3b8">Todavía no hay memorias registradas.</p>
{{end}}
{{end}}
`

const webTplMemoriaDetalle = `{{define "content"}}
<a href="/memoria" style="font-size:.85rem;color:#64748b">← {{t "Volver"}} {{t "Memoria"}}</a>
<h2 style="margin:.5rem 0">{{t "Memoria"}} — {{.Proyecto}}</h2>
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

<div class="action-box">
  <h4>Resumen vivo</h4>
  <form method="POST" action="/memoria/{{pathEsc .Proyecto}}/accion">
    <input type="hidden" name="accion" value="guardar">
    <div style="display:grid;grid-template-columns:1fr 220px;gap:.5rem">
      <div><label>Agente</label><input type="text" name="agente" value="{{if .M}}{{.M.ActualizadoPor}}{{end}}" placeholder="Codex3"></div>
      <div><label>Proyecto</label><input type="text" value="{{.Proyecto}}" disabled></div>
    </div>
    <div style="margin-top:.4rem"><label>Resumen</label><textarea name="resumen" rows="3" style="width:100%">{{if .M}}{{.M.Resumen}}{{end}}</textarea></div>
    <div style="margin-top:.4rem"><label>Contexto</label><textarea name="contexto" rows="3" style="width:100%">{{if .M}}{{.M.Contexto}}{{end}}</textarea></div>
    <div style="margin-top:.4rem"><label>Preguntas abiertas</label><textarea name="preguntas" rows="2" style="width:100%">{{if .M}}{{.M.PreguntasAbiertas}}{{end}}</textarea></div>
    <button type="submit" class="btn-sm" style="margin-top:.5rem">Guardar memoria</button>
  </form>
</div>

<div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem">
  <div class="action-box">
    <h4>Registrar fuente</h4>
    <form method="POST" action="/memoria/{{pathEsc .Proyecto}}/accion">
      <input type="hidden" name="accion" value="fuente">
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:.5rem">
        <div><label>Agente</label><input type="text" name="agente" placeholder="Codex3"></div>
        <div><label>Tipo</label><input type="text" name="tipo" value="documentacion"></div>
      </div>
      <div style="margin-top:.4rem"><label>Referencia *</label><input type="text" name="referencia" required style="width:100%"></div>
      <div style="margin-top:.4rem"><label>Título</label><input type="text" name="titulo" style="width:100%"></div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:.5rem;margin-top:.4rem">
        <div><label>URL</label><input type="text" name="url"></div>
        <div><label>Confianza</label><input type="text" name="confianza" value="media"></div>
      </div>
      <div style="margin-top:.4rem"><label>Detalle</label><textarea name="detalle" rows="2" style="width:100%"></textarea></div>
      <button type="submit" class="btn-sm" style="margin-top:.5rem">Registrar fuente</button>
    </form>
  </div>

  <div class="action-box">
    <h4>Registrar hallazgo</h4>
    <form method="POST" action="/memoria/{{pathEsc .Proyecto}}/accion">
      <input type="hidden" name="accion" value="hallazgo">
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:.5rem">
        <div><label>Agente</label><input type="text" name="agente" placeholder="Codex3"></div>
        <div><label>Fuente ID</label><input type="number" name="fuente_id" min="1" placeholder="opcional"></div>
      </div>
      <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:.5rem;margin-top:.4rem">
        <div><label>Tipo</label><input type="text" name="tipo" value="hecho"></div>
        <div><label>Impacto</label><input type="text" name="impacto" value="medio"></div>
        <div><label>Confianza</label><input type="text" name="confianza" value="media"></div>
      </div>
      <div style="margin-top:.4rem"><label>Título *</label><input type="text" name="titulo" required style="width:100%"></div>
      <div style="margin-top:.4rem"><label>Descripción</label><textarea name="descripcion" rows="3" style="width:100%"></textarea></div>
      <button type="submit" class="btn-sm" style="margin-top:.5rem">Registrar hallazgo</button>
    </form>
  </div>
</div>

<div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;margin-top:1rem">
  <div class="action-box">
    <h4>Registrar deriva</h4>
    <form method="POST" action="/memoria/{{pathEsc .Proyecto}}/accion">
      <input type="hidden" name="accion" value="deriva">
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:.5rem">
        <div><label>Agente</label><input type="text" name="agente" placeholder="Codex3"></div>
        <div><label>Hallazgo ID</label><input type="number" name="hallazgo_id" min="1" placeholder="opcional"></div>
      </div>
      <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:.5rem;margin-top:.4rem">
        <div><label>Tipo</label><input type="text" name="tipo" value="documental"></div>
        <div><label>Severidad</label><input type="text" name="severidad" value="media"></div>
        <div><label>Estado</label><input type="text" name="estado" value="abierta"></div>
      </div>
      <div style="margin-top:.4rem"><label>Descripción *</label><textarea name="descripcion" rows="3" required style="width:100%"></textarea></div>
      <div style="margin-top:.4rem"><label>Evidencia</label><textarea name="evidencia" rows="2" style="width:100%"></textarea></div>
      <button type="submit" class="btn-sm" style="margin-top:.5rem">Registrar deriva</button>
    </form>
  </div>

  <div class="action-box">
    <h4>Resolver deriva</h4>
    <form method="POST" action="/memoria/{{pathEsc .Proyecto}}/accion">
      <input type="hidden" name="accion" value="resolver-deriva">
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:.5rem">
        <div><label>Agente</label><input type="text" name="agente" placeholder="Codex3"></div>
        <div><label>Deriva ID *</label><input type="number" name="deriva_id" min="1" required></div>
      </div>
      <div style="margin-top:.4rem"><label>Estado</label><input type="text" name="estado" value="resuelta"></div>
      <div style="margin-top:.4rem"><label>Resolución *</label><textarea name="resolucion" rows="3" required style="width:100%"></textarea></div>
      <button type="submit" class="btn-sm" style="margin-top:.5rem">Actualizar deriva</button>
    </form>
  </div>
</div>

<div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:1rem;margin-top:1rem">
  <div>
    <h4>Fuentes</h4>
    {{if .Fuentes}}
    <table><thead><tr><th>ID</th><th>Referencia</th></tr></thead><tbody>
    {{range .Fuentes}}
      <tr><td>{{.ID}}</td><td><strong>{{.Referencia}}</strong><br><small>{{.Tipo}} / {{.Confianza}}{{if .Titulo}} · {{.Titulo}}{{end}}</small></td></tr>
    {{end}}
    </tbody></table>
    {{else}}<p style="color:#94a3b8">Sin fuentes.</p>{{end}}
  </div>
  <div>
    <h4>Hallazgos</h4>
    {{if .Hallazgos}}
    <table><thead><tr><th>ID</th><th>Título</th></tr></thead><tbody>
    {{range .Hallazgos}}
      <tr><td>{{.ID}}</td><td><strong>{{.Titulo}}</strong><br><small>{{.Tipo}} / {{.Impacto}} / {{.Confianza}}</small></td></tr>
    {{end}}
    </tbody></table>
    {{else}}<p style="color:#94a3b8">Sin hallazgos.</p>{{end}}
  </div>
  <div>
    <h4>Derivas</h4>
    {{if .Derivas}}
    <table><thead><tr><th>ID</th><th>Descripción</th></tr></thead><tbody>
    {{range .Derivas}}
      <tr><td>{{.ID}}</td><td><strong>{{.Descripcion}}</strong><br><small>{{.Tipo}} / {{.Severidad}} / {{.Estado}}</small></td></tr>
    {{end}}
    </tbody></table>
    {{else}}<p style="color:#94a3b8">Sin derivas.</p>{{end}}
  </div>
</div>
{{end}}
`

const webTplProgreso = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1rem">
  <h2 style="margin:0">{{t "Progreso real por proyecto"}} <small style="font-size:.5em;color:#94a3b8">{{len .Items}}</small></h2>
</div>
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}
{{if .Items}}
<table>
  <thead><tr><th>{{t "Proyecto"}}</th><th>{{t "Progreso"}}</th><th>{{t "Tareas"}}</th><th></th></tr></thead>
  <tbody>
  {{range .Items}}
    <tr>
      <td><strong>{{.Proyecto}}</strong></td>
      <td>{{printf "%.1f" .ProgresoPct}}%</td>
      <td>{{.TareasCompletadas}} / {{.TareasTotales}}</td>
      <td><a href="/progreso/{{pathEsc .Proyecto}}" class="btn-sm">{{t "Ver"}} →</a></td>
    </tr>
  {{end}}
  </tbody>
</table>
{{else}}
<p style="color:#94a3b8">Todavía no hay proyectos con fases o avance registrados.</p>
{{end}}
{{end}}
`

const webTplProgresoDetalle = `{{define "content"}}
{{if .R}}<a href="/progreso" style="font-size:.85rem;color:#64748b">← {{t "Volver"}} {{t "Progreso"}}</a>{{end}}
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}
{{if .R}}
<h2 style="margin:.5rem 0">{{.R.Proyecto}}</h2>
<div class="stats">
  <div class="stat"><div class="n">{{printf "%.1f" .R.ProgresoPct}}%</div><div class="l">{{t "Progreso real"}}</div></div>
  <div class="stat"><div class="n">{{.R.TareasTotales}}</div><div class="l">{{t "Tareas registradas"}}</div></div>
  <div class="stat"><div class="n">{{.R.TareasCompletadas}}</div><div class="l">{{t "Tareas completadas"}}</div></div>
</div>
<p style="font-size:.85rem;color:#64748b">API JSON: <code>/api/progreso/{{pathEsc .R.Proyecto}}</code></p>

{{if .R.Fases}}
<h4>{{t "Fase"}}s</h4>
<table>
  <thead><tr><th>Orden</th><th>{{t "Fase"}}</th><th>{{t "Estado"}}</th><th>{{t "Peso"}}</th><th>{{t "Progreso"}}</th><th>{{t "Tareas"}}</th></tr></thead>
  <tbody>
  {{range .R.Fases}}
    <tr>
      <td>{{.Fase.Orden}}</td>
      <td><strong>{{.Fase.Nombre}}</strong><br><small style="color:#64748b">{{.Fase.Descripcion}}</small></td>
      <td><span class="tag t-{{.Fase.Estado}}">{{t .Fase.Estado}}</span></td>
      <td>{{printf "%.1f" .Fase.Peso}}</td>
      <td>{{printf "%.1f" .ProgresoPct}}%</td>
      <td>{{.TareasCompletadas}} / {{.TareasTotales}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
{{end}}

{{if .R.TareasSinFase}}
<h4>{{t "Tareas"}} sin {{t "Fase"}}</h4>
<table>
  <thead><tr><th>ID</th><th>Titulo</th><th>{{t "Estado"}}</th><th>{{t "Progreso"}}</th></tr></thead>
  <tbody>
  {{range .R.TareasSinFase}}
    <tr>
      <td>{{.Tarea.ID}}</td>
      <td>{{.Tarea.Titulo}}</td>
      <td><span class="tag t-{{.Tarea.Estado}}">{{t .Tarea.Estado}}</span></td>
      <td>{{printf "%.1f" .ProgresoPct}}%</td>
    </tr>
  {{end}}
  </tbody>
</table>
{{end}}
{{end}}
{{end}}
`
