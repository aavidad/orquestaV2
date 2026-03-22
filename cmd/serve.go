/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
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

type webRuntimeRow struct {
	ID           int64
	Nivel        int
	Agente       string
	Proyecto     string
	Provider     string
	Connector    string
	Estado       string
	PID          string
	Hijos        int
	Branch       string
	Modelo       string
	Razonamiento string
	UltimaSenal  string
}

type webRuntimesData struct {
	Runtimes []webRuntimeRow
	Filtro   string
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
	"eqStr": func(a, b string) bool { return a == b },
	"neStr": func(a, b string) bool { return a != b },
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

func webHandlerRuntimes(w http.ResponseWriter, r *http.Request) {
	filtro := strings.TrimSpace(r.URL.Query().Get("activos"))
	agenteFiltro := strings.TrimSpace(r.URL.Query().Get("agente"))
	proyectoFiltro := strings.TrimSpace(r.URL.Query().Get("proyecto"))

	filter := db.FiltroRuntimes{}
	if agenteFiltro != "" {
		filter.Agente = &agenteFiltro
	}
	if proyectoFiltro != "" {
		proyecto, err := db.GetProyecto(proyectoFiltro)
		if err == nil {
			filter.ProyectoID = &proyecto.ID
		}
	}
	if filtro != "" {
		switch filtro {
		case "true":
			v := true
			filter.Activos = &v
		case "false":
			v := false
			filter.Activos = &v
		}
	}

	tree, _ := db.ConstruirArbolRuntimes(filter)
	rows := make([]webRuntimeRow, 0)
	aplanarRuntimes(tree, 0, &rows)
	webRender(w, webTplLayout+webTplRuntimes, webRuntimesData{
		Runtimes: rows,
		Filtro:   filtro,
	})
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
	abiertas, _ := db.ListarPropuestas(&estadoAb, nil)
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
	webRender(w, webTplLayout+webTplDash, webDashData{
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
		_, err = db.DB.Exec(`UPDATE tareas SET estado='backlog', agente=NULL WHERE id=?`, id)
		if err == nil {
			db.Audit("alberto", "backlog_tarea", "tarea", id, "")
		}
	case "reasignar":
		nuevoAgente := strings.TrimSpace(r.FormValue("nuevo_agente"))
		_, err = db.DB.Exec(`UPDATE tareas SET agente=?, estado='asignada' WHERE id=?`, nuevoAgente, id)
		if err == nil {
			db.Audit("alberto", "reasignar_tarea", "tarea", id, nuevoAgente)
		}
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

	var estadoPtr *db.EstadoPropuesta
	if filtro != "" {
		e := db.EstadoPropuesta(filtro)
		estadoPtr = &e
	}
	props, _ := db.ListarPropuestas(estadoPtr, nil)
	var detalles []webPropDetalle
	for _, p := range props {
		votos, _ := db.VotosDePropuesta(p.ID)
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
	p := &db.Propuesta{
		Codigo:       strings.TrimSpace(r.FormValue("codigo")),
		Titulo:       titulo,
		Descripcion:  r.FormValue("descripcion"),
		Tipo:         r.FormValue("tipo"),
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
	case "reabrir":
		_, err = db.ReabrirPropuesta(codigo, "alberto")
	case "reparar-votos":
		_, err = db.RepararVotosPendientesPropuesta(codigo, "alberto")
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

func aplanarRuntimes(nodes []*db.RuntimeTreeNode, nivel int, out *[]webRuntimeRow) {
	for _, node := range nodes {
		if node == nil || node.Runtime == nil {
			continue
		}
		r := node.Runtime
		pid := "—"
		if r.PID != nil {
			pid = strconv.FormatInt(*r.PID, 10)
		}
		proyecto := r.ProyectoSlug
		if proyecto == "" {
			proyecto = "—"
		}
		ultima := "—"
		if r.LastHeartbeatAt != nil {
			ultima = r.LastHeartbeatAt.Format("2006-01-02 15:04:05")
		} else if r.LastEventAt != nil {
			ultima = r.LastEventAt.Format("2006-01-02 15:04:05")
		}
		*out = append(*out, webRuntimeRow{
			ID:           r.ID,
			Nivel:        nivel,
			Agente:       r.Agente,
			Proyecto:     proyecto,
			Provider:     valorVacio(r.Provider),
			Connector:    valorVacio(r.Connector),
			Estado:       valorVacio(r.LogicalState),
			PID:          pid,
			Hijos:        len(node.Hijos),
			Branch:       valorVacio(r.Branch),
			Modelo:       valorVacio(r.Model),
			Razonamiento: valorVacio(r.Reasoning),
			UltimaSenal:  ultima,
		})
		aplanarRuntimes(node.Hijos, nivel+1, out)
	}
}

func valorVacio(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	return v
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

// ─── Comando ──────────────────────────────────────────────────────────────────

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Arranca el panel web (por defecto: http://localhost:8080)",
	RunE: func(cmd *cobra.Command, args []string) error {
		puerto, _ := cmd.Flags().GetInt("puerto")
		addr := fmt.Sprintf(":%d", puerto)
		mux := http.NewServeMux()
		mux.HandleFunc("/", webHandlerDash)
		mux.HandleFunc("/tareas", webHandlerTareas)
		mux.HandleFunc("/tareas/", webRouterTareas)
		mux.HandleFunc("/propuestas", webHandlerPropuestas)
		mux.HandleFunc("/propuestas/", webRouterPropuestas)
		mux.HandleFunc("/runtimes", webHandlerRuntimes)
		registerAPIRoutes(mux)
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
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Orquesta — ContaGrx</title>
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
    .rt-arrancando,.rt-disponible{background:#dcfce7;color:#166534}
    .rt-pensando,.rt-ejecutando_herramienta{background:#ede9fe;color:#5b21b6}
    .rt-esperando_io,.rt-handoff{background:#dbeafe;color:#1e40af}
    .rt-pausado{background:#fef3c7;color:#92400e}
    .rt-bloqueado,.rt-cerrado{background:#fee2e2;color:#991b1b}
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
    <div><a href="/"><strong>⚙ Orquesta</strong></a></div>
    <div>
      <a href="/">Dashboard</a>
      <a href="/tareas">Tareas</a>
      <a href="/propuestas">Propuestas</a>
      <a href="/runtimes">Runtimes</a>
    </div>
  </nav>
</header>
<main class="container" style="padding-top:1.5rem;padding-bottom:2rem">
{{template "content" .}}
</main>
<footer>ContaGrx · OSL Diputación de Granada · GPLv3</footer>
</body></html>
`

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

// ─── Runtimes: lista ─────────────────────────────────────────────────────────

const webTplRuntimes = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1rem">
  <h2 style="margin:0">Runtimes <small style="font-size:.5em;color:#94a3b8">{{len .Runtimes}}</small></h2>
</div>

<div class="filtros">
  <a href="/runtimes"{{if eqStr .Filtro ""}} class="sel"{{end}}>Todos</a>
  <a href="/runtimes?activos=true"{{if eqStr .Filtro "true"}} class="sel"{{end}}>Activos</a>
  <a href="/runtimes?activos=false"{{if eqStr .Filtro "false"}} class="sel"{{end}}>Cerrados</a>
</div>

{{if .Runtimes}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>ID</th><th>Estado</th><th>Agente</th><th>Proyecto</th><th>Provider</th><th>Connector</th><th>PID</th><th>Hijos</th><th>Branch</th><th>Última señal</th></tr></thead>
  <tbody>
  {{range .Runtimes}}
    <tr>
      <td style="padding-left:calc(.5rem + {{.Nivel}}rem)">#{{.ID}}</td>
      <td><span class="tag rt-{{.Estado}}">{{.Estado}}</span></td>
      <td>{{.Agente}}</td>
      <td>{{.Proyecto}}</td>
      <td>{{.Provider}}</td>
      <td>{{.Connector}}</td>
      <td>{{.PID}}</td>
      <td>{{.Hijos}}</td>
      <td><code style="font-size:.8em">{{.Branch}}</code></td>
      <td style="font-size:.82em;color:#64748b">{{.UltimaSenal}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>
{{else}}
<p style="color:#94a3b8">No hay runtimes visibles con este filtro.</p>
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

<div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;margin-bottom:1rem">
  {{if neStr $estado "abierta"}}
  <div class="action-box">
    <h4>Reabrir propuesta (Alberto)</h4>
    <form method="POST" action="/propuestas/{{$codigo}}/accion">
      <input type="hidden" name="accion" value="reabrir">
      <p style="margin:.2rem 0 .6rem 0;color:#64748b;font-size:.9rem">
        Vuelve a estado abierto y reconstruye votos pendientes faltantes.
      </p>
      <button type="submit" class="btn-sm">Reabrir propuesta</button>
    </form>
  </div>
  {{end}}
  <div class="action-box">
    <h4>Reparar votos pendientes (Alberto)</h4>
    <form method="POST" action="/propuestas/{{$codigo}}/accion">
      <input type="hidden" name="accion" value="reparar-votos">
      <p style="margin:.2rem 0 .6rem 0;color:#64748b;font-size:.9rem">
        Reconstruye filas de voto pendiente faltantes para agentes habilitados.
      </p>
      <button type="submit" class="btn-sm">Reparar votos</button>
    </form>
  </div>
</div>

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
