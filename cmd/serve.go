/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/i18n"
	"orquesta/internal/a2ui"
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
	Runtimes    []webRuntimeRow
	Checkpoints []webTimeTravelCheckpointRow
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
	Msg      string
	Err      string
}

type webRuntimeSampleRow struct {
	Creada      string
	CPUPct      string
	MemBytes    int64
	RSSBytes    int64
	OpenFDs     int64
	ChildCount  int
	ThreadCount int
	Estado      string
	Fuente      string
}

type webRuntimeTimelineRow struct {
	Creado  string
	Canal   string
	Tipo    string
	Estado  string
	Detalle string
	Link    string
}

type webRuntimeA2UIRow struct {
	Creado     string
	FromAgente string
	Estado     string
	Component  string
	Error      string
	DataTable  *webRuntimeA2UITable
	Chart      *webRuntimeA2UIChart
	Approval   *a2ui.ApprovalFormProps
	Markdown   *a2ui.MarkdownBlockProps
}

type webRuntimeA2UITable struct {
	Title   string
	Columns []string
	Rows    [][]string
}

type webRuntimeA2UIChart struct {
	Title     string
	ChartType string
	Series    []string
	Rows      []webRuntimeA2UIChartRow
}

type webRuntimeA2UIChartRow struct {
	Label  string
	Values []string
}

type webRuntimeDetalleData struct {
	Runtime  webRuntimeRow
	Muestras []webRuntimeSampleRow
	A2UI     []webRuntimeA2UIRow
	Timeline []webRuntimeTimelineRow
}

type webTimeTravelCheckpointRow struct {
	ID             int64
	Creado         string
	Agente         string
	Proyecto       string
	CheckpointKind string
	Resumen        string
	Branch         string
	CWD            string
	ResumeStrategy string
	Source         string
}

type webTimeTravelData struct {
	Checkpoints    []webTimeTravelCheckpointRow
	FiltroAgente   string
	FiltroProyecto string
	FiltroKind     string
}

type webTimeTravelDetalleData struct {
	Checkpoint webTimeTravelCheckpointRow
	Payload    string
	Memoria    []*db.EntidadMemoria
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
	"tr":    webTranslate,
	"reanimacionEn": func(t *time.Time) string {
		if t == nil || t.IsZero() {
			return ""
		}
		rem := time.Until(*t)
		if rem <= 0 {
			return "reanimando..."
		}
		if rem.Minutes() < 1 {
			return "reanima en segundos"
		}
		return fmt.Sprintf("reanima en %d min", int(rem.Minutes()))
	},
}

var (
	webI18nBundle     = i18n.NewBundle(resolveWebI18nDir(), i18n.DefaultLang)
	webI18nBundleOnce sync.Once
)

func resolveWebI18nDir() string {
	if dir := strings.TrimSpace(i18n.ResolveDir()); dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		dir := filepath.Join(filepath.Dir(file), "..", "i18n")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return i18n.ResolveDir()
}

func webTranslate(key string) string {
	webI18nBundleOnce.Do(func() {
		_ = webI18nBundle.Reload()
	})
	return webI18nBundle.T(i18n.DefaultLang, key)
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

func webRouterRuntimes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "runtimes" {
		http.NotFound(w, r)
		return
	}
	switch {
	case parts[1] == "control" && r.Method == http.MethodPost:
		webHandlerRuntimeControl(w, r)
		return
	case r.Method == http.MethodGet:
		webHandlerRuntimeDetalle(w, r, parts[1])
		return
	default:
		http.NotFound(w, r)
		return
	}
}

func webRouterTimeTravel(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "time-travel" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	webHandlerTimeTravelDetalle(w, r, parts[1])
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
		Msg:      r.URL.Query().Get("ok"),
		Err:      r.URL.Query().Get("err"),
	})
}

func webHandlerRuntimeControl(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/runtimes?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	req := apiAgenteControlRequest{
		Agente:       strings.TrimSpace(r.FormValue("agente")),
		Proyecto:     strings.TrimSpace(r.FormValue("proyecto")),
		Accion:       strings.TrimSpace(r.FormValue("accion")),
		Conector:     strings.TrimSpace(r.FormValue("conector")),
		Modelo:       strings.TrimSpace(r.FormValue("modelo")),
		Razonamiento: strings.TrimSpace(r.FormValue("razonamiento")),
		Perfil:       strings.TrimSpace(r.FormValue("perfil")),
		Motivo:       strings.TrimSpace(r.FormValue("motivo")),
		Por:          "web",
	}
	orderID, accion, err := encolarControlAgenteLocal(req)
	if err != nil {
		http.Redirect(w, r, "/runtimes?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	msg := fmt.Sprintf("Orden %s #%d encolada para %s", accion, orderID, req.Agente)
	http.Redirect(w, r, "/runtimes?ok="+url.QueryEscape(msg), http.StatusSeeOther)
}

func webHandlerRuntimeDetalle(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	runtime, err := db.GetRuntime(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	samples, _ := db.ListarMuestrasRuntime(id, 20)

	row := runtimeRowToWeb(runtimeRowDesdeModelo(runtime, 0, runtime.ChildCount))
	data := webRuntimeDetalleData{
		Runtime:  row,
		Muestras: runtimeSamplesToWeb(samples),
		A2UI:     runtimeA2UIToWeb(runtime, 8),
		Timeline: runtimeTimelineToWeb(runtime, 24),
	}
	webRender(w, webTplLayout+webTplRuntimeDetalle, data)
}

func webHandlerTimeTravel(w http.ResponseWriter, r *http.Request) {
	agenteFiltro := strings.TrimSpace(r.URL.Query().Get("agente"))
	proyectoFiltro := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	kindFiltro := strings.TrimSpace(r.URL.Query().Get("kind"))

	filter := db.FiltroRuntimeCheckpoints{Limit: 100}
	if agenteFiltro != "" {
		filter.Agente = &agenteFiltro
	}
	if proyectoFiltro != "" {
		if proyecto, err := db.GetProyecto(proyectoFiltro); err == nil {
			filter.ProyectoID = &proyecto.ID
		}
	}
	if kindFiltro != "" {
		filter.CheckpointKind = &kindFiltro
	}

	checkpoints, _ := db.ListarRuntimeCheckpoints(filter)
	rows := make([]webTimeTravelCheckpointRow, 0, len(checkpoints))
	for _, cp := range checkpoints {
		if cp == nil {
			continue
		}
		rows = append(rows, runtimeCheckpointToWeb(cp))
	}

	webRender(w, webTplLayout+webTplTimeTravel, webTimeTravelData{
		Checkpoints:    rows,
		FiltroAgente:   agenteFiltro,
		FiltroProyecto: proyectoFiltro,
		FiltroKind:     kindFiltro,
	})
}

func webHandlerTimeTravelDetalle(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	checkpoint, err := db.GetRuntimeCheckpoint(id)
	if err != nil || checkpoint == nil {
		http.NotFound(w, r)
		return
	}

	var memoria []*db.EntidadMemoria
	if checkpoint.ProyectoID != nil {
		memoria, _ = db.ListarEntidadesMemoria(db.FiltroEntidadesMemoria{ProyectoID: checkpoint.ProyectoID})
	}
	payload := strings.TrimSpace(checkpoint.PayloadJSON)
	if payload == "" {
		payload = "{}"
	}

	webRender(w, webTplLayout+webTplTimeTravelDetalle, webTimeTravelDetalleData{
		Checkpoint: runtimeCheckpointToWeb(checkpoint),
		Payload:    payload,
		Memoria:    memoria,
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
	activos := true
	runtimeTree, _ := db.ConstruirArbolRuntimes(db.FiltroRuntimes{Activos: &activos})
	var runtimes []webRuntimeRow
	aplanarRuntimes(runtimeTree, 0, &runtimes)
	checkpoints, _ := db.ListarRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Limit: 5})
	var recentCheckpoints []webTimeTravelCheckpointRow
	for _, cp := range checkpoints {
		if cp == nil {
			continue
		}
		recentCheckpoints = append(recentCheckpoints, runtimeCheckpointToWeb(cp))
	}
	webRender(w, webTplLayout+webTplDash, webDashData{
		Agentes: agentes, Counts: counts, Total: total,
		Completadas: completadas, Pct: pct,
		Abiertas: resAbiertas, EnProgreso: ep, Runtimes: runtimes, Checkpoints: recentCheckpoints,
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
		err = db.MoverTareaABacklog(id)
		if err == nil {
			db.Audit("alberto", "backlog_tarea", "tarea", id, "")
		}
	case "reasignar":
		nuevoAgente := strings.TrimSpace(r.FormValue("nuevo_agente"))
		err = db.ReasignarTarea(id, nuevoAgente)
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

func runtimeSamplesToWeb(samples []*db.RuntimeTelemetrySample) []webRuntimeSampleRow {
	out := make([]webRuntimeSampleRow, 0, len(samples))
	for _, s := range samples {
		if s == nil {
			continue
		}
		out = append(out, webRuntimeSampleRow{
			Creada:      s.CreatedAt.Format("2006-01-02 15:04:05"),
			CPUPct:      fmt.Sprintf("%.1f", s.CPUPct),
			MemBytes:    s.MemBytes,
			RSSBytes:    s.RSSBytes,
			OpenFDs:     s.OpenFDs,
			ChildCount:  s.ChildCount,
			ThreadCount: s.ThreadCount,
			Estado:      valorVacio(s.LogicalState),
			Fuente:      valorVacio(s.Source),
		})
	}
	return out
}

func runtimeRowToWeb(row runtimeRow) webRuntimeRow {
	return webRuntimeRow{
		ID:           row.ID,
		Nivel:        row.Nivel,
		Agente:       row.Agente,
		Proyecto:     row.Proyecto,
		Provider:     row.Provider,
		Connector:    row.Connector,
		Estado:       row.Estado,
		PID:          row.PID,
		Hijos:        row.Hijos,
		Branch:       row.Branch,
		Modelo:       row.Modelo,
		Razonamiento: row.Razonamiento,
		UltimaSenal:  row.Ultima,
	}
}

func runtimeCheckpointToWeb(cp *db.RuntimeCheckpoint) webTimeTravelCheckpointRow {
	proyecto := "—"
	if cp.ProyectoID != nil {
		if p, err := db.GetProyecto(strconv.FormatInt(*cp.ProyectoID, 10)); err == nil && p != nil {
			proyecto = p.Slug
		} else {
			proyecto = strconv.FormatInt(*cp.ProyectoID, 10)
		}
	}
	return webTimeTravelCheckpointRow{
		ID:             cp.ID,
		Creado:         cp.CreatedAt.Format("2006-01-02 15:04:05"),
		Agente:         cp.Agente,
		Proyecto:       proyecto,
		CheckpointKind: valorVacio(cp.CheckpointKind),
		Resumen:        valorVacio(cp.Resumen),
		Branch:         valorVacio(cp.Branch),
		CWD:            valorVacio(cp.CWD),
		ResumeStrategy: valorVacio(cp.ResumeStrategy),
		Source:         valorVacio(cp.Source),
	}
}

func runtimeA2UIToWeb(runtime *db.RuntimeInstance, limit int) []webRuntimeA2UIRow {
	if runtime == nil {
		return nil
	}
	agente := strings.TrimSpace(runtime.Agente)
	if agente == "" {
		return nil
	}
	if limit <= 0 {
		limit = 8
	}
	messages, _ := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: runtime.ProyectoID,
	})
	out := make([]webRuntimeA2UIRow, 0, min(limit, len(messages)))
	for _, msg := range messages {
		if msg == nil || strings.TrimSpace(msg.Kind) != a2ui.MailboxKindRender {
			continue
		}
		row := webRuntimeA2UIRow{
			Creado:     msg.CreatedAt.Format("2006-01-02 15:04:05"),
			FromAgente: valorVacio(msg.FromAgente),
			Estado:     valorVacio(msg.Estado),
			Component:  valorVacio(msg.Kind),
		}
		envelope, props, err := a2ui.DecodeMailboxPayload(msg.Kind, msg.PayloadJSON)
		if err != nil {
			row.Error = err.Error()
			out = append(out, row)
			if len(out) >= limit {
				break
			}
			continue
		}
		row.Component = valorVacio(string(envelope.Request.Component))
		switch v := props.(type) {
		case a2ui.DataTableProps:
			rows := make([][]string, 0, len(v.Data))
			for _, src := range v.Data {
				rowVals := make([]string, 0, len(src))
				for _, cell := range src {
					rowVals = append(rowVals, fmt.Sprintf("%v", cell))
				}
				rows = append(rows, rowVals)
			}
			row.DataTable = &webRuntimeA2UITable{
				Title:   valorVacio(v.Title),
				Columns: append([]string(nil), v.Columns...),
				Rows:    rows,
			}
		case a2ui.ChartProps:
			chartRows := make([]webRuntimeA2UIChartRow, 0, len(v.Labels))
			series := make([]string, 0, len(v.Series))
			for _, s := range v.Series {
				series = append(series, s.Name)
			}
			for idx, label := range v.Labels {
				values := make([]string, 0, len(v.Series))
				for _, s := range v.Series {
					values = append(values, fmt.Sprintf("%.2f", s.Values[idx]))
				}
				chartRows = append(chartRows, webRuntimeA2UIChartRow{
					Label:  label,
					Values: values,
				})
			}
			row.Chart = &webRuntimeA2UIChart{
				Title:     valorVacio(v.Title),
				ChartType: valorVacio(v.ChartType),
				Series:    series,
				Rows:      chartRows,
			}
		case a2ui.ApprovalFormProps:
			copy := v
			row.Approval = &copy
		case a2ui.MarkdownBlockProps:
			copy := v
			row.Markdown = &copy
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func runtimeTimelineToWeb(runtime *db.RuntimeInstance, limit int) []webRuntimeTimelineRow {
	if runtime == nil {
		return nil
	}
	agente := strings.TrimSpace(runtime.Agente)
	if agente == "" {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}

	orderFilter := db.FiltroRuntimeOrders{Agente: &agente}
	if runtime.ProyectoID != nil {
		orderFilter.ProyectoID = runtime.ProyectoID
	}
	orders, _ := db.ListarRuntimeOrders(orderFilter)

	toAgente := agente
	mailboxTo, _ := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: runtime.ProyectoID,
	})
	fromAgente := agente
	mailboxFrom, _ := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		FromAgente: &fromAgente,
		ProyectoID: runtime.ProyectoID,
	})
	checkpoints, _ := db.ListarRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{
		Agente:     &agente,
		ProyectoID: runtime.ProyectoID,
		Limit:      limit,
	})

	type timelineEvent struct {
		When time.Time
		Row  webRuntimeTimelineRow
	}
	events := make([]timelineEvent, 0, len(orders)+len(mailboxTo)+len(mailboxFrom)+len(checkpoints))
	for _, order := range orders {
		if order == nil {
			continue
		}
		when := order.UpdatedAt
		if order.FinishedAt != nil {
			when = *order.FinishedAt
		} else if order.StartedAt != nil {
			when = *order.StartedAt
		}
		detalle := truncar(strings.TrimSpace(order.PayloadJSON), 120)
		if strings.TrimSpace(order.ErrorText) != "" {
			detalle = truncar(strings.TrimSpace(detalle+" · "+order.ErrorText), 120)
		}
		events = append(events, timelineEvent{
			When: when,
			Row: webRuntimeTimelineRow{
				Creado:  when.Format("2006-01-02 15:04:05"),
				Canal:   "runtime_order",
				Tipo:    fmt.Sprintf("#%d %s", order.ID, valorVacio(order.Tipo)),
				Estado:  valorVacio(order.Estado),
				Detalle: valorVacio(detalle),
			},
		})
	}

	mailboxByID := make(map[int64]struct{}, len(mailboxTo)+len(mailboxFrom))
	appendMailbox := func(msg *db.RuntimeMailboxMessage) {
		if msg == nil {
			return
		}
		if _, seen := mailboxByID[msg.ID]; seen {
			return
		}
		mailboxByID[msg.ID] = struct{}{}
		when := msg.CreatedAt
		if msg.ConsumedAt != nil {
			when = *msg.ConsumedAt
		} else if msg.DeliveredAt != nil {
			when = *msg.DeliveredAt
		}
		detalle := fmt.Sprintf("%s → %s", valorVacio(msg.FromAgente), valorVacio(msg.ToAgente))
		payload := strings.TrimSpace(msg.PayloadJSON)
		if payload != "" && payload != "{}" {
			detalle += " · " + truncar(payload, 96)
		}
		events = append(events, timelineEvent{
			When: when,
			Row: webRuntimeTimelineRow{
				Creado:  when.Format("2006-01-02 15:04:05"),
				Canal:   "mailbox",
				Tipo:    fmt.Sprintf("#%d %s", msg.ID, valorVacio(msg.Kind)),
				Estado:  valorVacio(msg.Estado),
				Detalle: detalle,
			},
		})
	}
	for _, msg := range mailboxTo {
		appendMailbox(msg)
	}
	for _, msg := range mailboxFrom {
		appendMailbox(msg)
	}

	for _, cp := range checkpoints {
		if cp == nil {
			continue
		}
		detalle := strings.TrimSpace(cp.Resumen)
		if strings.TrimSpace(cp.Source) != "" {
			if detalle != "" {
				detalle += " · "
			}
			detalle += "source=" + strings.TrimSpace(cp.Source)
		}
		if strings.TrimSpace(cp.ResumeStrategy) != "" {
			if detalle != "" {
				detalle += " · "
			}
			detalle += "resume=" + strings.TrimSpace(cp.ResumeStrategy)
		}
		events = append(events, timelineEvent{
			When: cp.CreatedAt,
			Row: webRuntimeTimelineRow{
				Creado:  cp.CreatedAt.Format("2006-01-02 15:04:05"),
				Canal:   "checkpoint",
				Tipo:    fmt.Sprintf("#%d %s", cp.ID, valorVacio(cp.CheckpointKind)),
				Estado:  valorVacio(cp.Branch),
				Detalle: valorVacio(detalle),
				Link:    "/time-travel/" + strconv.FormatInt(cp.ID, 10),
			},
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].When.After(events[j].When)
	})
	if len(events) > limit {
		events = events[:limit]
	}
	rows := make([]webRuntimeTimelineRow, 0, len(events))
	for _, ev := range events {
		rows = append(rows, ev.Row)
	}
	return rows
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
	Short: "Arranca el panel web (por defecto: http://localhost:16543)",
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
		mux.HandleFunc("/runtimes/", webRouterRuntimes)
		mux.HandleFunc("/time-travel", webHandlerTimeTravel)
		mux.HandleFunc("/time-travel/", webRouterTimeTravel)
		registerAPIRoutes(mux)
		fmt.Printf("✓ Panel web en http://localhost%s\n", addr)
		fmt.Println("  Ctrl+C para detener.")

		controlCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		newControlPlaneRunner().Start(controlCtx)

		return http.ListenAndServe(addr, mux)
	},
}

func init() {
	serveCmd.Flags().Int("puerto", 16543, "Puerto HTTP")
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
      <a href="/time-travel">Time Travel</a>
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
  <small style="color:#94a3b8">{{.Generado}} · auto-refresca cada 10 s</small>
</div>
<meta http-equiv="refresh" content="10">
<div class="stats">
  <div class="stat"><a href="/tareas"><div class="n">{{.Total}}</div><div class="l">tareas totales</div></a></div>
  <div class="stat"><a href="/tareas?estado=completada"><div class="n" style="color:#16a34a">{{.Completadas}}</div><div class="l">completadas</div></a></div>
  <div class="stat"><a href="/tareas?estado=en_progreso"><div class="n" style="color:#7c3aed">{{index .Counts "en_progreso"}}</div><div class="l">en progreso</div></a></div>
  <div class="stat"><a href="/tareas?estado=asignada"><div class="n" style="color:#2563eb">{{index .Counts "asignada"}}</div><div class="l">asignadas</div></a></div>
  <div class="stat"><a href="/tareas?estado=bloqueada"><div class="n" style="color:#dc2626">{{index .Counts "bloqueada"}}</div><div class="l">bloqueadas</div></a></div>
  <div class="stat"><a href="/propuestas"><div class="n">{{len .Abiertas}}</div><div class="l">props. abiertas</div></a></div>
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
        <td style="width:1.2rem;padding:.3rem 0">
          {{if eq .EstadoCuota "enfriamiento"}}
            <span class="dot-off" style="color:#f59e0b">💤</span>
          {{else if .Activo}}
            <span class="dot-on">●</span>
          {{else}}
            <span class="dot-off">○</span>
          {{end}}
        </td>
        <td style="padding:.3rem .3rem">
          <div style="font-weight:600;font-size:.85rem">{{.Nombre}}</div>
          {{if eq .EstadoCuota "enfriamiento"}}
            <div style="font-size:.7rem;color:#f59e0b;font-style:italic">Dormido: {{.MotivoPausa}}</div>
            <div style="font-size:.65rem;color:#94a3b8">{{reanimacionEn .ReanimarAt}}</div>
          {{else if eq .EstadoCuota "agotado"}}
            <div style="font-size:.7rem;color:#dc2626">Agotado semanal</div>
          {{else}}
            <div style="font-size:.72rem;color:#94a3b8">{{.Rol}}</div>
          {{end}}
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
          <tr><td style="padding:.2rem .4rem"><a href="/tareas?estado={{$est}}"><span class="tag t-{{$est}}">{{$est}}</span></a></td><td style="padding:.2rem .6rem"><a href="/tareas?estado={{$est}}"><strong>{{$n}}</strong></a></td></tr>
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
      {{if .Runtimes}}
      <div>
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">Runtimes activos <a href="/runtimes" style="font-size:.9em;font-weight:normal;text-transform:none">ver todos →</a></h4>
        <table><thead><tr><th>Agente</th><th>Estado</th><th>PID</th><th>Hijos</th></tr></thead><tbody>
        {{range .Runtimes}}
          <tr>
            <td style="padding-left:{{.Nivel}}rem"><a href="/runtimes/{{.ID}}">{{if gt .Nivel 0}}↳ {{end}}{{.Agente}}</a></td>
            <td><span class="tag rt-{{.Estado}}">{{.Estado}}</span></td>
            <td>{{.PID}}</td>
            <td>{{.Hijos}}</td>
          </tr>
        {{end}}
        </tbody></table>
      </div>
      {{end}}
      {{if .Checkpoints}}
      <div>
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">Checkpoints recientes <a href="/time-travel" style="font-size:.9em;font-weight:normal;text-transform:none">ver todos →</a></h4>
        <table><thead><tr><th>Agente</th><th>Tipo</th><th>Resumen</th></tr></thead><tbody>
        {{range .Checkpoints}}
          <tr>
            <td><a href="/time-travel/{{.ID}}">{{.Agente}}</a></td>
            <td><span class="tag t-media">{{.CheckpointKind}}</span></td>
            <td style="font-size:.82rem">{{trunc .Resumen 32}}</td>
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

{{if .Msg}}<div class="alert-ok">{{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">{{.Err}}</div>{{end}}

<div class="filtros">
  <a href="/runtimes"{{if eqStr .Filtro ""}} class="sel"{{end}}>Todos</a>
  <a href="/runtimes?activos=true"{{if eqStr .Filtro "true"}} class="sel"{{end}}>Activos</a>
  <a href="/runtimes?activos=false"{{if eqStr .Filtro "false"}} class="sel"{{end}}>Cerrados</a>
</div>

<div class="action-box">
  <h4>Control de agentes</h4>
  <form method="POST" action="/runtimes/control" style="display:grid;grid-template-columns:1fr 1fr 1fr 1fr;gap:.6rem">
    <div><label>Agente</label><input type="text" name="agente" placeholder="Codex1" required></div>
    <div><label>Proyecto</label><input type="text" name="proyecto" placeholder="orquestador"></div>
    <div>
      <label>Acción</label>
      <select name="accion">
        <option value="arrancar">Arrancar</option>
        <option value="pausar">Pausar</option>
        <option value="continuar">Continuar</option>
        <option value="detener">Detener</option>
      </select>
    </div>
    <div><label>Motivo</label><input type="text" name="motivo" placeholder="motivo operativo"></div>
    <div><label>Conector</label><input type="text" name="conector" placeholder="codex-cli"></div>
    <div><label>Modelo</label><input type="text" name="modelo" placeholder="gpt-5.4"></div>
    <div><label>Reasoning</label><input type="text" name="razonamiento" placeholder="high"></div>
    <div><label>Perfil</label><input type="text" name="perfil" placeholder="implementacion"></div>
    <div style="grid-column:1/-1"><button type="submit" class="btn-sm">Encolar orden</button></div>
  </form>
</div>

{{if .Runtimes}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>ID</th><th>Estado</th><th>Agente</th><th>Proyecto</th><th>Provider</th><th>Connector</th><th>PID</th><th>Hijos</th><th>Branch</th><th>Última señal</th></tr></thead>
  <tbody>
  {{range .Runtimes}}
    <tr>
      <td style="padding-left:calc(.5rem + {{.Nivel}}rem)"><a href="/runtimes/{{.ID}}">#{{.ID}}</a></td>
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

const webTplRuntimeDetalle = `{{define "content"}}
<a href="/runtimes" style="font-size:.85rem;color:#64748b">← volver a runtimes</a>
<h2 style="margin:.5rem 0">Runtime #{{.Runtime.ID}} — {{.Runtime.Agente}}</h2>
<p>
  <span class="tag rt-{{.Runtime.Estado}}">{{.Runtime.Estado}}</span>
  <span style="color:#64748b;font-size:.85rem;margin-left:.5rem">Proyecto: {{.Runtime.Proyecto}} · Provider: {{.Runtime.Provider}} · Connector: {{.Runtime.Connector}}</span>
</p>
<p style="margin-top:-.2rem">
  <a href="/time-travel?agente={{.Runtime.Agente}}{{if neStr .Runtime.Proyecto "—"}}&proyecto={{.Runtime.Proyecto}}{{end}}" style="font-size:.84rem">Ver checkpoints del agente →</a>
</p>

<div class="stats" style="grid-template-columns:repeat(auto-fit,minmax(110px,1fr));margin-bottom:1rem">
  <div class="stat"><div class="n" style="font-size:1.2rem">{{.Runtime.PID}}</div><div class="l">pid</div></div>
  <div class="stat"><div class="n" style="font-size:1.2rem">{{.Runtime.Hijos}}</div><div class="l">hijos</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Runtime.Branch}}</div><div class="l">branch</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Runtime.Modelo}}</div><div class="l">modelo</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Runtime.Razonamiento}}</div><div class="l">reasoning</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Runtime.UltimaSenal}}</div><div class="l">última señal</div></div>
</div>

<h4 style="margin:0 0 .6rem 0">Muestras recientes</h4>
{{if .Muestras}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>Creada</th><th>Estado</th><th>CPU%</th><th>MEM</th><th>RSS</th><th>FDs</th><th>Hijos</th><th>Hilos</th><th>Fuente</th></tr></thead>
  <tbody>
  {{range .Muestras}}
    <tr>
      <td>{{.Creada}}</td>
      <td><span class="tag rt-{{.Estado}}">{{.Estado}}</span></td>
      <td>{{.CPUPct}}</td>
      <td>{{.MemBytes}}</td>
      <td>{{.RSSBytes}}</td>
      <td>{{.OpenFDs}}</td>
      <td>{{.ChildCount}}</td>
      <td>{{.ThreadCount}}</td>
      <td>{{.Fuente}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>
{{else}}
<p style="color:#94a3b8">No hay muestras registradas para este runtime.</p>
{{end}}

<h4 style="margin:1.1rem 0 .6rem 0">{{tr "runtime.a2ui.title"}}</h4>
{{if .A2UI}}
  {{range .A2UI}}
  <article style="padding:1rem;border:1px solid #e2e8f0;border-radius:.5rem;margin-bottom:.8rem">
    <div style="display:flex;justify-content:space-between;gap:.6rem;align-items:baseline;flex-wrap:wrap">
      <strong>{{.Component}}</strong>
      <small style="color:#64748b">{{tr "runtime.a2ui.from"}} {{.FromAgente}} · {{.Creado}} · {{.Estado}}</small>
    </div>
    {{if .Error}}
      <p class="alert-err" style="margin:.75rem 0 0 0">{{tr "runtime.a2ui.invalid"}}: {{.Error}}</p>
    {{end}}
    {{if .DataTable}}
      <h5 style="margin:.8rem 0 .5rem 0">{{.DataTable.Title}}</h5>
      <div style="overflow-x:auto">
        <table>
          <thead>
            <tr>{{range .DataTable.Columns}}<th>{{.}}</th>{{end}}</tr>
          </thead>
          <tbody>
          {{range .DataTable.Rows}}
            <tr>{{range .}}<td>{{.}}</td>{{end}}</tr>
          {{end}}
          </tbody>
        </table>
      </div>
    {{end}}
    {{if .Chart}}
      <h5 style="margin:.8rem 0 .2rem 0">{{.Chart.Title}}</h5>
      <p style="margin:0 0 .5rem 0;color:#64748b;font-size:.84rem">{{tr "runtime.a2ui.chart_type"}}: {{.Chart.ChartType}}</p>
      <div style="overflow-x:auto">
        <table>
          <thead>
            <tr>
              <th>{{tr "runtime.a2ui.label"}}</th>
              {{range .Chart.Series}}<th>{{.}}</th>{{end}}
            </tr>
          </thead>
          <tbody>
          {{range .Chart.Rows}}
            <tr>
              <td>{{.Label}}</td>
              {{range .Values}}<td>{{.}}</td>{{end}}
            </tr>
          {{end}}
          </tbody>
        </table>
      </div>
    {{end}}
    {{if .Approval}}
      <h5 style="margin:.8rem 0 .2rem 0">{{.Approval.Title}}</h5>
      <p style="margin:0 0 .5rem 0">{{.Approval.Message}}</p>
      <div style="display:flex;gap:.5rem;flex-wrap:wrap;align-items:center">
        <button type="button" class="btn-sm" disabled>{{if .Approval.ConfirmLabel}}{{.Approval.ConfirmLabel}}{{else}}{{tr "runtime.a2ui.confirm"}}{{end}}</button>
        <button type="button" class="btn-sm" disabled>{{if .Approval.CancelLabel}}{{.Approval.CancelLabel}}{{else}}{{tr "runtime.a2ui.cancel"}}{{end}}</button>
        <small style="color:#64748b">{{tr "runtime.a2ui.action_id"}}: {{.Approval.ActionID}}</small>
      </div>
    {{end}}
    {{if .Markdown}}
      <h5 style="margin:.8rem 0 .2rem 0">{{.Markdown.Title}}</h5>
      <pre style="white-space:pre-wrap;background:#f8fafc;border:1px solid #e2e8f0;border-radius:.4rem;padding:.8rem;margin:0">{{.Markdown.Markdown}}</pre>
    {{end}}
  </article>
  {{end}}
{{else}}
<p style="color:#94a3b8">{{tr "runtime.a2ui.none"}}</p>
{{end}}

<h4 style="margin:1.1rem 0 .6rem 0">Timeline operativa</h4>
{{if .Timeline}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>Cuándo</th><th>Canal</th><th>Evento</th><th>Estado</th><th>Detalle</th></tr></thead>
  <tbody>
  {{range .Timeline}}
    <tr>
      <td>{{.Creado}}</td>
      <td><span class="tag">{{.Canal}}</span></td>
      <td>{{if .Link}}<a href="{{.Link}}">{{.Tipo}}</a>{{else}}{{.Tipo}}{{end}}</td>
      <td><span class="tag">{{.Estado}}</span></td>
      <td style="color:#64748b">{{.Detalle}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>
{{else}}
<p style="color:#94a3b8">Sin órdenes, mailbox ni checkpoints asociados.</p>
{{end}}
{{end}}
`

const webTplTimeTravel = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1rem">
  <h2 style="margin:0">Time Travel <small style="font-size:.5em;color:#94a3b8">{{len .Checkpoints}}</small></h2>
</div>

<div class="filtros">
  <a href="/time-travel"{{if and (eqStr .FiltroAgente "") (eqStr .FiltroProyecto "") (eqStr .FiltroKind "")}} class="sel"{{end}}>Todos</a>
  <a href="/time-travel?kind=checkpoint"{{if eqStr .FiltroKind "checkpoint"}} class="sel"{{end}}>Checkpoints</a>
  <a href="/time-travel?kind=handoff"{{if eqStr .FiltroKind "handoff"}} class="sel"{{end}}>Handoffs</a>
  <a href="/time-travel?kind=sync_status"{{if eqStr .FiltroKind "sync_status"}} class="sel"{{end}}>Sync status</a>
</div>

<form method="GET" action="/time-travel" style="display:grid;grid-template-columns:1fr 1fr 1fr auto;gap:.5rem;align-items:end;margin-bottom:1rem">
  <div><label>Agente</label><input type="text" name="agente" value="{{.FiltroAgente}}" placeholder="Codex1"></div>
  <div><label>Proyecto</label><input type="text" name="proyecto" value="{{.FiltroProyecto}}" placeholder="orquestador"></div>
  <div><label>Tipo</label><input type="text" name="kind" value="{{.FiltroKind}}" placeholder="checkpoint"></div>
  <div><button type="submit" class="btn-sm">Filtrar</button></div>
</form>

{{if .Checkpoints}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>ID</th><th>Creado</th><th>Agente</th><th>Proyecto</th><th>Tipo</th><th>Resumen</th><th>Branch</th><th>Resume</th><th></th></tr></thead>
  <tbody>
  {{range .Checkpoints}}
    <tr>
      <td>#{{.ID}}</td>
      <td style="font-size:.82em;color:#64748b">{{.Creado}}</td>
      <td>{{.Agente}}</td>
      <td>{{.Proyecto}}</td>
      <td><span class="tag t-media">{{.CheckpointKind}}</span></td>
      <td>{{trunc .Resumen 64}}</td>
      <td><code style="font-size:.8em">{{.Branch}}</code></td>
      <td>{{.ResumeStrategy}}</td>
      <td><a href="/time-travel/{{.ID}}" class="btn-sm">Inspeccionar →</a></td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>
{{else}}
<p style="color:#94a3b8">No hay checkpoints visibles con este filtro.</p>
{{end}}
{{end}}
`

const webTplTimeTravelDetalle = `{{define "content"}}
<a href="/time-travel" style="font-size:.85rem;color:#64748b">← volver a Time Travel</a>
<h2 style="margin:.5rem 0">Checkpoint #{{.Checkpoint.ID}} — {{.Checkpoint.Agente}}</h2>
<p>
  <span class="tag t-media">{{.Checkpoint.CheckpointKind}}</span>
  <span style="color:#64748b;font-size:.85rem;margin-left:.5rem">Proyecto: {{.Checkpoint.Proyecto}} · Branch: {{.Checkpoint.Branch}} · {{.Checkpoint.Creado}}</span>
</p>

<div class="stats" style="grid-template-columns:repeat(auto-fit,minmax(150px,1fr));margin-bottom:1rem">
  <div class="stat"><div class="n" style="font-size:1rem">{{.Checkpoint.ResumeStrategy}}</div><div class="l">resume</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Checkpoint.Source}}</div><div class="l">source</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Checkpoint.Branch}}</div><div class="l">branch</div></div>
  <div class="stat"><div class="n" style="font-size:.92rem">{{trunc .Checkpoint.CWD 28}}</div><div class="l">cwd</div></div>
</div>

<h4 style="margin:0 0 .5rem 0">Resumen</h4>
<p style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.4rem;padding:.8rem 1rem">{{.Checkpoint.Resumen}}</p>

<h4 style="margin:1rem 0 .5rem 0">Payload</h4>
<pre style="background:#0f172a;color:#e2e8f0;padding:1rem;border-radius:.5rem;overflow:auto;font-size:.8rem">{{.Payload}}</pre>

<h4 style="margin:1rem 0 .5rem 0">Memoria de proyecto</h4>
{{if .Memoria}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>Entidad</th><th>Tipo</th><th>Verificado por</th><th>Valor</th></tr></thead>
  <tbody>
  {{range .Memoria}}
    <tr>
      <td><strong>{{.Nombre}}</strong></td>
      <td><span class="tag t-media">{{.Tipo}}</span></td>
      <td>{{.VerificadoPor}}</td>
      <td><code style="font-size:.78em">{{trunc .ValorJSON 96}}</code></td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>
{{else}}
<p style="color:#94a3b8">No hay memoria asociada a este proyecto.</p>
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
