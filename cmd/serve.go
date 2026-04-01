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
	"net"
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
	"orquesta/notificaciones"
	"orquesta/propuestasapp"
	"orquesta/tareasapp"
)

// ─── Structs de datos ────────────────────────────────────────────────────────

type webDashData struct {
	Agentes        []*db.Agente
	Counts         map[string]int
	Total          int
	Completadas    int
	Pct            int
	Abiertas       []webPropResumen
	EnProgreso     []webTareaRow
	Runtimes       []webRuntimeRow
	Checkpoints    []webTimeTravelCheckpointRow
	Notificaciones notificaciones.EstadoNotificaciones
	NotifOutbox    notificaciones.OutboxSummary
	Generado       string
	Msg            string
}

type webOpenClawData struct {
	Status           apiOpenClawStatusLite
	EnCuota          []*db.Agente
	MailboxPendiente []apiOpenClawMailboxLite
	Retenidas        []tareaLite
	TareasReservadas []tareaLite
	ReviewGates      []*db.ReviewGate
	Signals          []*supervisorReviewSignal
	Merges           []*db.GitMerge
	NormalizedEvents []openClawNormalizedEvent
	ThreadSessions   []*db.SupervisorThreadSessionSummary
	PipelineStates   []*db.SupervisorPipelineState
	Recommended      []supervisorRecommendedAction
	SafeRecommended  []supervisorRecommendedAction
	NextAction       *supervisorRecommendedAction
	NextSafeAction   *supervisorRecommendedAction
	Notificaciones   notificaciones.EstadoNotificaciones
	NotifOutbox      notificaciones.OutboxSummary
	Integration      webOpenClawIntegrationInfo
	Generado         string
	Msg              string
	Err              string
}

type webOpenClawIntegrationInfo struct {
	MCPEndpoint       string
	Configured        bool
	Transport         string
	Endpoint          string
	WorkspaceMode     string
	AgentPrefix       string
	RequireIdentity   string
	PluginPath        string
	RunbookPath       string
	SmokeCommand      string
	GatewayConfigured bool
	TelegramConfigured bool
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
	"sub": func(a, b int) int {
		return a - b
	},
	"ftime": func(t *time.Time) string {
		if t == nil || t.IsZero() {
			return "—"
		}
		return t.Format("2006-01-02 15:04:05")
	},
	"ftimev": func(t time.Time) string {
		if t.IsZero() {
			return "—"
		}
		return t.Format("2006-01-02 15:04:05")
	},
	"orDash": func(s string) string {
		if strings.TrimSpace(s) == "" {
			return "—"
		}
		return s
	},
	"pid": func(v *int64) string {
		if v == nil || *v <= 0 {
			return "—"
		}
		return strconv.FormatInt(*v, 10)
	},
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
	"jsonLines": textoPlanoDesdeListaJSON,
}

var (
	webI18nBundle     = i18n.NewBundle(resolveWebI18nDir(), i18n.DefaultLang)
	webI18nBundleOnce sync.Once
)

const webLangCookieName = "orquesta_lang"

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
	return webTranslateForLang(i18n.DefaultLang, key)
}

func webTranslateForLang(lang, key string) string {
	webI18nBundleOnce.Do(func() {
		_ = webI18nBundle.Reload()
	})
	return webI18nBundle.T(lang, key)
}

func webTranslateRequestf(r *http.Request, key string, args ...any) string {
	msg := webTranslateForLang(resolveWebRequestLang(r), key)
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

func buildWebFuncMap(lang string) template.FuncMap {
	funcs := make(template.FuncMap, len(webFuncMap)+2)
	for key, value := range webFuncMap {
		funcs[key] = value
	}
	funcs["tr"] = func(key string) string {
		return webTranslateForLang(lang, key)
	}
	funcs["lang"] = func() string {
		return lang
	}
	return funcs
}

func resolveWebRequestLang(r *http.Request) string {
	webI18nBundleOnce.Do(func() {
		_ = webI18nBundle.Reload()
	})
	if r != nil {
		if lang, ok := resolveExplicitWebLang(r.URL.Query().Get("lang")); ok {
			return lang
		}
		if cookie, err := r.Cookie(webLangCookieName); err == nil {
			if lang, ok := resolveExplicitWebLang(cookie.Value); ok {
				return lang
			}
		}
		for _, cand := range parseAcceptLanguageHeader(r.Header.Get("Accept-Language")) {
			if lang, ok := resolveExplicitWebLang(cand); ok {
				return lang
			}
		}
	}
	return webI18nBundle.ResolveLang(i18n.DefaultLang)
}

func resolveExplicitWebLang(raw string) (string, bool) {
	lang := i18n.NormalizeLang(raw)
	if lang == "" || !webI18nBundle.HasLang(lang) {
		return "", false
	}
	return webI18nBundle.ResolveLang(lang), true
}

func parseAcceptLanguageHeader(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.IndexByte(part, ';'); idx >= 0 {
			part = strings.TrimSpace(part[:idx])
		}
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func persistWebLangCookie(w http.ResponseWriter, r *http.Request, resolved string) {
	if r == nil {
		return
	}
	explicit := strings.TrimSpace(r.URL.Query().Get("lang"))
	if explicit == "" {
		return
	}
	if lang, ok := resolveExplicitWebLang(explicit); ok {
		http.SetCookie(w, &http.Cookie{
			Name:     webLangCookieName,
			Value:    lang,
			Path:     "/",
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})
		return
	}
	if resolved != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     webLangCookieName,
			Value:    resolved,
			Path:     "/",
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})
	}
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
		proyecto, err := runtimesService.GetProject(proyectoFiltro)
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

	tree, _ := runtimesService.BuildRuntimeTree(filter)
	rows := make([]webRuntimeRow, 0)
	aplanarRuntimes(tree, 0, &rows)
	webRender(w, r, webTplLayout+webTplRuntimes, webRuntimesData{
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
	runtime, err := runtimesService.GetRuntime(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	samples, _ := runtimesService.ListRuntimeSamples(id, 20)

	row := runtimeRowToWeb(runtimeRowDesdeModelo(runtime, 0, runtime.ChildCount))
	data := webRuntimeDetalleData{
		Runtime:  row,
		Muestras: runtimeSamplesToWeb(samples),
		A2UI:     runtimeA2UIToWeb(runtime, 8),
		Timeline: runtimeTimelineToWeb(runtime, 24),
	}
	webRender(w, r, webTplLayout+webTplRuntimeDetalle, data)
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
		if proyecto, err := runtimesService.GetProject(proyectoFiltro); err == nil {
			filter.ProyectoID = &proyecto.ID
		}
	}
	if kindFiltro != "" {
		filter.CheckpointKind = &kindFiltro
	}

	checkpoints, _ := runtimesService.ListRuntimeCheckpoints(filter)
	rows := make([]webTimeTravelCheckpointRow, 0, len(checkpoints))
	for _, cp := range checkpoints {
		if cp == nil {
			continue
		}
		rows = append(rows, runtimeCheckpointToWeb(cp))
	}

	webRender(w, r, webTplLayout+webTplTimeTravel, webTimeTravelData{
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
	checkpoint, err := runtimesService.GetRuntimeCheckpoint(id)
	if err != nil || checkpoint == nil {
		http.NotFound(w, r)
		return
	}

	var memoria []*db.EntidadMemoria
	if checkpoint.ProyectoID != nil {
		memoria, _ = runtimesService.ListMemoryEntities(db.FiltroEntidadesMemoria{ProyectoID: checkpoint.ProyectoID})
	}
	payload := strings.TrimSpace(checkpoint.PayloadJSON)
	if payload == "" {
		payload = "{}"
	}

	webRender(w, r, webTplLayout+webTplTimeTravelDetalle, webTimeTravelDetalleData{
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
	summary, _ := panelService.BuildSummary()
	var (
		agentes     []*db.Agente
		counts      map[string]int
		total       int
		completadas int
		pct         int
		resAbiertas []webPropResumen
		ep          []webTareaRow
	)
	if summary != nil {
		agentes = summary.Agents
		counts = summary.TaskCounts
		total = summary.TotalTasks
		completadas = summary.DoneTasks
		pct = summary.PercentDone
		resAbiertas = make([]webPropResumen, 0, len(summary.OpenProps))
		for _, item := range summary.OpenProps {
			resAbiertas = append(resAbiertas, webPropResumen{
				Codigo:     item.Codigo,
				Titulo:     item.Titulo,
				Acuerdo:    item.Acuerdo,
				Desacuerdo: item.Desacuerdo,
				Pendiente:  item.Pendiente,
			})
		}
		ep = make([]webTareaRow, 0, len(summary.ActiveTasks))
		for _, t := range summary.ActiveTasks {
			ep = append(ep, toWebTarea(t))
		}
	}
	activos := true
	runtimeTree, _ := runtimesService.BuildRuntimeTree(db.FiltroRuntimes{Activos: &activos})
	var runtimes []webRuntimeRow
	aplanarRuntimes(runtimeTree, 0, &runtimes)
	checkpoints, _ := runtimesService.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Limit: 5})
	var recentCheckpoints []webTimeTravelCheckpointRow
	for _, cp := range checkpoints {
		if cp == nil {
			continue
		}
		recentCheckpoints = append(recentCheckpoints, runtimeCheckpointToWeb(cp))
	}
	notifs := notificaciones.DescribirConfiguracion()
	notifOutbox := notificaciones.DescribirOutbox(5)
	webRender(w, r, webTplLayout+webTplDash, webDashData{
		Agentes: agentes, Counts: counts, Total: total,
		Completadas: completadas, Pct: pct,
		Abiertas: resAbiertas, EnProgreso: ep, Runtimes: runtimes, Checkpoints: recentCheckpoints,
		Notificaciones: notifs,
		NotifOutbox:    notifOutbox,
		Generado:       time.Now().Format("2006-01-02 15:04:05"),
	})
}

func webHandlerOpenClaw(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/openclaw" {
		http.NotFound(w, r)
		return
	}
	status, err := buildEstadoResumen()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	operatorStatus, err := buildOpenClawOperatorStatus(status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	review, err := buildSupervisorReviewSnapshot("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reviewGates, _ := review["review_gates"].([]*db.ReviewGate)
	signals, _ := review["signals"].([]*supervisorReviewSignal)
	merges, _ := review["merges"].([]*db.GitMerge)
	normalizedEvents, _ := review["normalized_events"].([]openClawNormalizedEvent)
	var threadSessions []*db.SupervisorThreadSessionSummary
	if snapshot, ok := review["thread_sessions"].(map[string]any); ok {
		threadSessions, _ = snapshot["sessions"].([]*db.SupervisorThreadSessionSummary)
	}
	var pipelineStates []*db.SupervisorPipelineState
	if snapshot, ok := review["pipeline_state"].(map[string]any); ok {
		pipelineStates, _ = snapshot["pipelines"].([]*db.SupervisorPipelineState)
	}
	recommended, _ := review["recommended_actions"].([]supervisorRecommendedAction)
	safeRecommended, _ := review["safe_action_queue"].([]supervisorRecommendedAction)
	var nextAction *supervisorRecommendedAction
	switch item := review["next_action"].(type) {
	case supervisorRecommendedAction:
		nextAction = &item
	case *supervisorRecommendedAction:
		nextAction = item
	}
	var nextSafeAction *supervisorRecommendedAction
	switch item := review["next_safe_action"].(type) {
	case supervisorRecommendedAction:
		nextSafeAction = &item
	case *supervisorRecommendedAction:
		nextSafeAction = item
	}
	if nextSafeAction == nil {
		for i := range safeRecommended {
			nextSafeAction = &safeRecommended[i]
			break
		}
	}
	webRender(w, r, webTplLayout+webTplOpenClaw, webOpenClawData{
		Status:           operatorStatus,
		EnCuota:          agentesNoActivosConCuota(status.Agentes),
		MailboxPendiente: mustOpenClawPendingMailbox(status.Agentes),
		Retenidas:        tareasRetenidasPorCuota(status.TareasActivas, status.Agentes),
		TareasReservadas: filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaAsignada),
		ReviewGates:      reviewGates,
		Signals:          signals,
		Merges:           merges,
		NormalizedEvents: normalizedEvents,
		ThreadSessions:   threadSessions,
		PipelineStates:   pipelineStates,
		Recommended:      recommended,
		SafeRecommended:  safeRecommended,
		NextAction:       nextAction,
		NextSafeAction:   nextSafeAction,
		Notificaciones:   notificaciones.DescribirConfiguracion(),
		NotifOutbox:      notificaciones.DescribirOutbox(10),
		Integration:      buildWebOpenClawIntegrationInfo(),
		Generado:         time.Now().Format("2006-01-02 15:04:05"),
		Msg:              r.URL.Query().Get("ok"),
		Err:              r.URL.Query().Get("err"),
	})
}

func mustOpenClawPendingMailbox(agentes []*db.Agente) []apiOpenClawMailboxLite {
	items, err := buildOpenClawPendingMailbox(agentes)
	if err != nil {
		return nil
	}
	return items
}

func buildWebOpenClawIntegrationInfo() webOpenClawIntegrationInfo {
	info := webOpenClawIntegrationInfo{
		MCPEndpoint:      webOpenClawDefaultEndpoint(),
		PluginPath:       "./plugins/openclaw-orquesta-api",
		RunbookPath:      "./plugins/openclaw-orquesta-api/RUNBOOK_TELEGRAM.md",
		SmokeCommand:     "./plugins/openclaw-orquesta-api/scripts/smoke_openclaw_orquesta.sh",
		Transport:        strings.TrimSpace(configGetDefault("integration_openclaw_transport", "")),
		Endpoint:         strings.TrimSpace(configGetDefault("integration_openclaw_endpoint", "")),
		WorkspaceMode:    strings.TrimSpace(configGetDefault("integration_openclaw_workspace_mode", "")),
		AgentPrefix:      strings.TrimSpace(configGetDefault("integration_openclaw_agent_prefix", "")),
		RequireIdentity:  strings.TrimSpace(configGetDefault("integration_openclaw_require_identity", "")),
		GatewayConfigured: strings.TrimSpace(configGetDefault("openclaw_gateway_url", "")) != "",
		TelegramConfigured: strings.TrimSpace(configGetDefault("telegram_token", "")) != "" &&
			strings.TrimSpace(configGetDefault("telegram_chat_id", "")) != "",
	}
	info.Configured = info.Transport != "" || info.Endpoint != "" || info.WorkspaceMode != ""
	if info.Transport == "" {
		info.Transport = "mcp_http"
	}
	if info.Endpoint == "" {
		info.Endpoint = info.MCPEndpoint
	}
	if info.WorkspaceMode == "" {
		info.WorkspaceMode = "worktree"
	}
	if info.AgentPrefix == "" {
		info.AgentPrefix = "OpenClaw-"
	}
	if info.RequireIdentity == "" {
		info.RequireIdentity = "true"
	}
	return info
}

func configGetDefault(key, fallback string) string {
	value, err := db.ConfigGet(key)
	if err != nil {
		return fallback
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func webHandlerOpenClawAccion(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/openclaw" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/openclaw?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	kind := strings.TrimSpace(r.FormValue("kind"))
	switch kind {
	case "agente":
		nombre := strings.TrimSpace(r.FormValue("agente"))
		accion := strings.TrimSpace(r.FormValue("accion"))
		if nombre == "" || !webAccionEstadoAgenteValida(accion) {
			http.Redirect(w, r, "/openclaw?err="+url.QueryEscape("accion de agente invalida"), http.StatusSeeOther)
			return
		}
		if err := webAplicarAccionEstadoAgentePorAPI(nombre, accion); err != nil {
			http.Redirect(w, r, "/openclaw?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/openclaw?ok="+url.QueryEscape(fmt.Sprintf("Acción %s aplicada a %s", accion, nombre)), http.StatusSeeOther)
		return
	case "review_gate":
		gateID := strings.TrimSpace(r.FormValue("gate_id"))
		estado := strings.TrimSpace(r.FormValue("estado"))
		reviewer := strings.TrimSpace(r.FormValue("reviewer_agente"))
		findings := strings.TrimSpace(r.FormValue("findings_json"))
		if gateID == "" || estado == "" {
			http.Redirect(w, r, "/openclaw?err="+url.QueryEscape("review gate invalido"), http.StatusSeeOther)
			return
		}
		var resp map[string]any
		path := "/api/review-gates/" + url.PathEscape(gateID) + "/resolver"
		req := apiReviewGateResolveRequest{
			Estado:         estado,
			ReviewerAgente: reviewer,
			FindingsJSON:   findings,
		}
		if err := webInvocarAPIJSON(http.MethodPost, path, req, &resp); err != nil {
			http.Redirect(w, r, "/openclaw?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/openclaw?ok="+url.QueryEscape(fmt.Sprintf("Review gate %s actualizada a %s", gateID, estado)), http.StatusSeeOther)
		return
	case "supervision_action":
		action := strings.TrimSpace(r.FormValue("action"))
		target := strings.TrimSpace(r.FormValue("target"))
		assignee := strings.TrimSpace(r.FormValue("assignee"))
		if action == "" || target == "" {
			http.Redirect(w, r, "/openclaw?err="+url.QueryEscape("accion de supervision invalida"), http.StatusSeeOther)
			return
		}
		if _, err := applySupervisorRecommendedAction("OpenClaw", action, target, assignee); err != nil {
			http.Redirect(w, r, "/openclaw?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}
		msg := fmt.Sprintf("Acción %s aplicada sobre %s", action, target)
		if assignee != "" {
			msg += " con " + assignee
		}
		http.Redirect(w, r, "/openclaw?ok="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	case "supervision_batch":
		maxItems := 0
		if raw := strings.TrimSpace(r.FormValue("max_items")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				maxItems = parsed
			}
		}
		result, err := applySupervisorRecommendedActionsBatch("OpenClaw", maxItems)
		if err != nil {
			http.Redirect(w, r, "/openclaw?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}
		count, _ := result["count"].(int)
		http.Redirect(w, r, "/openclaw?ok="+url.QueryEscape(fmt.Sprintf("Aplicadas %d acciones seguras del supervisor", count)), http.StatusSeeOther)
		return
	case "supervision_next":
		result, err := applySupervisorNextAction("OpenClaw")
		if err != nil {
			http.Redirect(w, r, "/openclaw?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}
		action, _ := result["action"].(supervisorRecommendedAction)
		msg := "Aplicada la siguiente acción segura del supervisor"
		if strings.TrimSpace(action.Action) != "" {
			msg = "Aplicada " + strings.TrimSpace(action.Action)
			if strings.TrimSpace(action.Target) != "" {
				msg += " sobre " + strings.TrimSpace(action.Target)
			}
		}
		http.Redirect(w, r, "/openclaw?ok="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	default:
		http.Redirect(w, r, "/openclaw?err="+url.QueryEscape("accion openclaw desconocida"), http.StatusSeeOther)
		return
	}
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
	tareas, _ := tareasService.List(f)
	var wt []webTareaRow
	for _, t := range tareas {
		wt = append(wt, toWebTarea(t))
	}
	agentes, _ := tareasService.ListAgents()
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
		http.Redirect(w, r, "/tareas?err="+url.QueryEscape(webTranslateRequestf(r, "tasks.flash.title_required")), http.StatusSeeOther)
		return
	}
	id, err := tareasService.Create(tareasapp.CreateTaskInput{
		Titulo:          titulo,
		Descripcion:     r.FormValue("descripcion"),
		Modulo:          r.FormValue("modulo"),
		Prioridad:       db.PrioridadTarea(r.FormValue("prioridad")),
		CreadoPor:       "alberto",
		Agente:          strings.TrimSpace(r.FormValue("agente")),
		PropuestaCodigo: strings.TrimSpace(r.FormValue("propuesta")),
	})
	if err != nil {
		http.Redirect(w, r, "/tareas?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/tareas?ok="+url.QueryEscape(webTranslateRequestf(r, "tasks.flash.created", id)), http.StatusSeeOther)
}

// ─── Tareas: detalle ──────────────────────────────────────────────────────────

func webHandlerTareaDetalle(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	t, err := tareasService.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	agentes, _ := tareasService.ListAgents()
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
		http.Redirect(w, r, "/tareas?err="+url.QueryEscape(webTranslateRequestf(r, "tasks.flash.invalid_id")), http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	accion := r.FormValue("accion")
	agente := strings.TrimSpace(r.FormValue("agente"))
	back := fmt.Sprintf("/tareas/%d", id)

	switch accion {
	case "tomar":
		err = tareasService.Take(id, agente)
	case "iniciar":
		err = tareasService.Start(id, agente)
	case "completar":
		err = tareasService.Complete(id, agente, r.FormValue("commit"))
	case "bloquear":
		err = tareasService.Block(id, agente, r.FormValue("motivo"))
	case "desbloquear":
		err = tareasService.Unblock(id, agente, r.FormValue("resolucion"))
	case "nota":
		err = tareasService.Note(id, agente, r.FormValue("nota"))
	case "backlog":
		err = tareasService.MoveToBacklog(id)
	case "reasignar":
		nuevoAgente := strings.TrimSpace(r.FormValue("nuevo_agente"))
		err = tareasService.Reassign(id, nuevoAgente)
	default:
		http.Redirect(w, r, back+"?err="+url.QueryEscape(webTranslateRequestf(r, "tasks.flash.unknown_action")), http.StatusSeeOther)
		return
	}

	if err != nil {
		http.Redirect(w, r, back+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, back+"?ok="+url.QueryEscape(webTranslateRequestf(r, "tasks.flash.action_applied", accion)), http.StatusSeeOther)
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
	props, _ := propuestasService.List(estadoPtr)
	var detalles []webPropDetalle
	for _, p := range props {
		detail, err := propuestasService.GetDetail(p.Codigo)
		if err != nil {
			continue
		}
		cerrada := ""
		if p.CerradaAt != nil {
			cerrada = p.CerradaAt.Format("2006-01-02")
		}
		detalles = append(detalles, webPropDetalle{
			Codigo: p.Codigo, Titulo: p.Titulo, Descripcion: p.Descripcion,
			Tipo: p.Tipo, Estado: string(p.Estado), PropuestoPor: p.PropuestoPor,
			Fecha: p.CreatedAt.Format("2006-01-02"), CerradaFecha: cerrada,
			Votos: detail.Votes,
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
		http.Redirect(w, r, "/propuestas?err="+url.QueryEscape(webTranslateRequestf(r, "proposals.flash.title_required")), http.StatusSeeOther)
		return
	}
	_, p, err := propuestasService.Create(propuestasapp.CreateProposalInput{
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
	http.Redirect(w, r, "/propuestas/"+p.Codigo+"?ok="+url.QueryEscape(webTranslateRequestf(r, "proposals.flash.created", p.Codigo)), http.StatusSeeOther)
}

// ─── Propuestas: detalle ──────────────────────────────────────────────────────

func webHandlerPropuestaDetalle(w http.ResponseWriter, r *http.Request, codigo string) {
	detail, err := propuestasService.GetDetail(codigo)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p := detail.Proposal
	cerrada := ""
	if p.CerradaAt != nil {
		cerrada = p.CerradaAt.Format("2006-01-02")
	}
	agentes, _ := propuestasService.ListAgents()
	webRender(w, r, webTplLayout+webTplPropuestaDetalle, webPropDetalleData{
		P: webPropDetalle{
			Codigo: p.Codigo, Titulo: p.Titulo, Descripcion: p.Descripcion,
			Tipo: p.Tipo, Estado: string(p.Estado), PropuestoPor: p.PropuestoPor,
			Fecha: p.CreatedAt.Format("2006-01-02"), CerradaFecha: cerrada,
			Votos: detail.Votes,
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

	if _, err := propuestasService.GetDetail(codigo); err != nil {
		http.Redirect(w, r, "/propuestas?err="+url.QueryEscape(webTranslateRequestf(r, "proposals.flash.not_found")), http.StatusSeeOther)
		return
	}
	var err error
	switch accion {
	case "votar":
		agente := strings.TrimSpace(r.FormValue("agente"))
		posicion := db.PosicionVoto(r.FormValue("posicion"))
		comentario := r.FormValue("comentario")
		err = propuestasService.Vote(codigo, agente, posicion, comentario)
	case "cerrar":
		nuevoEstado := r.FormValue("estado_cierre")
		err = propuestasService.Close(codigo, nuevoEstado, "alberto")
	case "reabrir":
		_, err = propuestasService.Reopen(codigo, "alberto")
	case "reparar-votos":
		_, err = propuestasService.RepairPendingVotes(codigo, "alberto")
	default:
		http.Redirect(w, r, back+"?err="+url.QueryEscape(webTranslateRequestf(r, "proposals.flash.unknown_action")), http.StatusSeeOther)
		return
	}

	if err != nil {
		http.Redirect(w, r, back+"?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, back+"?ok="+url.QueryEscape(webTranslateRequestf(r, "proposals.flash.action_applied", accion)), http.StatusSeeOther)
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
		if p, err := runtimesService.GetProject(strconv.FormatInt(*cp.ProyectoID, 10)); err == nil && p != nil {
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
	messages, _ := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
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
	orders, _ := runtimesService.ListRuntimeOrders(orderFilter)

	toAgente := agente
	mailboxTo, _ := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: runtime.ProyectoID,
	})
	fromAgente := agente
	mailboxFrom, _ := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		FromAgente: &fromAgente,
		ProyectoID: runtime.ProyectoID,
	})
	checkpoints, _ := runtimesService.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{
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

func webRender(w http.ResponseWriter, args ...any) {
	var (
		r      *http.Request
		tplStr string
		data   any
		ok     bool
	)
	switch len(args) {
	case 2:
		tplStr, ok = args[0].(string)
		if !ok {
			http.Error(w, "Error de plantilla: argumento inválido", 500)
			return
		}
		data = args[1]
	case 3:
		r, _ = args[0].(*http.Request)
		tplStr, ok = args[1].(string)
		if !ok {
			http.Error(w, "Error de plantilla: argumento inválido", 500)
			return
		}
		data = args[2]
	default:
		http.Error(w, "Error de plantilla: argumentos inválidos", 500)
		return
	}
	lang := resolveWebRequestLang(r)
	persistWebLangCookie(w, r, lang)
	tmpl, err := template.New("layout").Funcs(buildWebFuncMap(lang)).Parse(tplStr)
	if err != nil {
		http.Error(w, "Error de plantilla: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", lang)
	_ = tmpl.Execute(w, data)
}

// ─── Comando ──────────────────────────────────────────────────────────────────

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Arranca el panel web (por defecto: http://localhost:16543)",
	RunE: func(cmd *cobra.Command, args []string) error {
		host, _ := cmd.Flags().GetString("host")
		puerto, _ := cmd.Flags().GetInt("puerto")
		security, err := resolveServerSecurityOptions(cmd)
		if err != nil {
			return err
		}
		addr := net.JoinHostPort(host, fmt.Sprintf("%d", puerto))
		return arrancarServidorUnificado(addr, "serve", true, resolveServerDebugOptions(cmd), security)
	},
}

func init() {
	serveCmd.Flags().String("host", "127.0.0.1", "Host de escucha HTTP/HTTPS")
	serveCmd.Flags().Int("puerto", defaultServePort, "Puerto HTTP")
	serveCmd.Flags().String("tls-cert", "", "Certificado PEM del servidor")
	serveCmd.Flags().String("tls-key", "", "Clave privada PEM del servidor")
	serveCmd.Flags().String("tls-client-ca", "", "CA PEM para exigir certificados cliente (mTLS)")
	serveCmd.Flags().Bool("debug", false, "Activa logging de depuración del servidor")
	serveCmd.Flags().Bool("debug-http", false, "Log HTTP detallado por request")
	serveCmd.Flags().Bool("debug-control-plane", false, "Log detallado del ciclo del control plane")
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
  <title>{{tr "app.title"}}</title>
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
    <div><a href="/"><strong>⚙ {{tr "app.name"}}</strong></a></div>
    <div>
      <a href="/">{{tr "Dashboard"}}</a>
      <a href="/tareas">{{tr "Tareas"}}</a>
      <a href="/propuestas">{{tr "Propuestas"}}</a>
      <a href="/agentes">{{tr "Agentes"}}</a>
      <a href="/nueva-app">{{tr "nueva_app.nav"}}</a>
      <a href="/progreso">{{tr "progress.title"}}</a>
      <a href="/pools">{{tr "pools.title"}}</a>
      <a href="/modelo">{{tr "model.title"}}</a>
      <a href="/memoria">{{tr "memory.title"}}</a>
      <a href="/deploy">{{tr "deploy.title"}}</a>
      <a href="/config">{{tr "config.title"}}</a>
      <a href="/openclaw">OpenClaw</a>
      <a href="/conectores">{{tr "connectors.title"}}</a>
      <a href="/diagnostico">{{tr "ops.diagnosis.title"}}</a>
      <a href="/auditoria">{{tr "ops.audit.title"}}</a>
      <a href="/refineria">{{tr "ops.refinery.title"}}</a>
      <a href="/respaldo">{{tr "ops.backup.title"}}</a>
      <a href="/gobernanza">{{tr "governance.title"}}</a>
      <a href="/runtimes">{{tr "Runtimes"}}</a>
      <a href="/time-travel">{{tr "Time Travel"}}</a>
    </div>
  </nav>
</header>
<main class="container" style="padding-top:1.5rem;padding-bottom:2rem">
{{template "content" .}}
</main>
<footer>{{tr "app.footer"}}</footer>
</body></html>
`

// ─── Dashboard ────────────────────────────────────────────────────────────────

const webTplDash = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1.2rem">
  <h2 style="margin:0">{{tr "dashboard.title"}}</h2>
  <small style="color:#94a3b8">{{.Generado}} · {{tr "dashboard.auto_refresh"}}</small>
</div>
<meta http-equiv="refresh" content="10">
<div class="stats">
  <div class="stat"><a href="/tareas"><div class="n">{{.Total}}</div><div class="l">{{tr "tareas totales"}}</div></a></div>
  <div class="stat"><a href="/tareas?estado=completada"><div class="n" style="color:#16a34a">{{.Completadas}}</div><div class="l">{{tr "completadas"}}</div></a></div>
  <div class="stat"><a href="/tareas?estado=en_progreso"><div class="n" style="color:#7c3aed">{{index .Counts "en_progreso"}}</div><div class="l">{{tr "en_progreso"}}</div></a></div>
  <div class="stat"><a href="/tareas?estado=asignada"><div class="n" style="color:#2563eb">{{index .Counts "asignada"}}</div><div class="l">{{tr "asignadas"}}</div></a></div>
  <div class="stat"><a href="/tareas?estado=bloqueada"><div class="n" style="color:#dc2626">{{index .Counts "bloqueada"}}</div><div class="l">{{tr "bloqueadas"}}</div></a></div>
  <div class="stat"><a href="/propuestas"><div class="n">{{len .Abiertas}}</div><div class="l">{{tr "props. abiertas"}}</div></a></div>
</div>
<div class="pgbar">
  <div class="pgfill" style="width:{{if gt .Pct 0}}{{.Pct}}%{{else}}2.5rem{{end}}">{{.Pct}}%</div>
</div>
<div style="display:grid;grid-template-columns:220px 1fr;gap:1.5rem;align-items:start">
  <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.5rem;padding:.8rem 1rem">
    <h4 style="margin:0 0 .7rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">{{tr "Agentes"}}</h4>
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
          {{if .CuentaEmail}}
            <div style="font-size:.68rem;color:#0f172a">cuenta: {{.CuentaEmail}}</div>
          {{end}}
          {{if .CuentaUsuario}}
            <div style="font-size:.66rem;color:#64748b">usuario: {{.CuentaUsuario}}</div>
          {{end}}
          {{if eq .EstadoCuota "enfriamiento"}}
            <div style="font-size:.7rem;color:#f59e0b;font-style:italic">{{tr "dashboard.sleeping"}}: {{.MotivoPausa}}</div>
            <div style="font-size:.65rem;color:#94a3b8">{{reanimacionEn .ReanimarAt}}</div>
          {{else if eq .EstadoCuota "agotado"}}
            <div style="font-size:.7rem;color:#dc2626">{{tr "dashboard.weekly_exhausted"}}</div>
          {{else}}
            <div style="font-size:.72rem;color:#94a3b8">{{tr .Rol}}</div>
          {{end}}
          {{if .CuotaRestantePct}}
            <div style="font-size:.66rem;color:#0f766e">
              efectivo {{.CuotaRestantePct}}%{{if .PresupuestoVentana}} · {{.PresupuestoVentana}}{{end}}{{if .PresupuestoResetAt}} · reset {{.PresupuestoResetAt.Local.Format "2006-01-02 15:04"}}{{end}}
            </div>
          {{end}}
          {{if or .PresupuestoSesionPct .PresupuestoDiarioPct .PresupuestoSemanalPct}}
            <div style="font-size:.64rem;color:#64748b">
              {{if .PresupuestoSesionPct}}sesión {{.PresupuestoSesionPct}}%{{if .PresupuestoSesionResetAt}} reset {{.PresupuestoSesionResetAt.Local.Format "2006-01-02 15:04"}}{{end}}{{end}}
              {{if and .PresupuestoSesionPct (or .PresupuestoDiarioPct .PresupuestoSemanalPct)}} · {{end}}
              {{if .PresupuestoDiarioPct}}diario {{.PresupuestoDiarioPct}}%{{if .PresupuestoDiarioResetAt}} reset {{.PresupuestoDiarioResetAt.Local.Format "2006-01-02 15:04"}}{{end}}{{end}}
              {{if and .PresupuestoDiarioPct .PresupuestoSemanalPct}} · {{end}}
              {{if .PresupuestoSemanalPct}}semanal {{.PresupuestoSemanalPct}}%{{if .PresupuestoSemanalResetAt}} reset {{.PresupuestoSemanalResetAt.Local.Format "2006-01-02 15:04"}}{{end}}{{end}}
            </div>
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
    <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.5rem;padding:.8rem 1rem;margin-bottom:1rem">
      <h4 style="margin:0 0 .7rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">OpenClaw / notificaciones</h4>
      <table style="width:100%"><tbody>
      {{range .Notificaciones.Canales}}
        <tr style="border-bottom:1px solid #f1f5f9">
          <td style="padding:.3rem 0;font-weight:600;font-size:.84rem">{{.Nombre}}</td>
          <td style="padding:.3rem .4rem;text-align:right"><span class="tag {{if .Activo}}rt-disponible{{else}}rt-cerrado{{end}}">{{if .Activo}}activo{{else}}inactivo{{end}}</span></td>
        </tr>
        <tr>
          <td colspan="2" style="padding:0 0 .45rem 0;font-size:.72rem;color:#64748b">{{.Detalle}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{if .NotifOutbox.Recientes}}
      <div style="margin-top:.8rem;border-top:1px solid #e2e8f0;padding-top:.7rem">
        <div style="font-size:.75rem;color:#64748b;margin-bottom:.4rem">Entregas recientes</div>
        <table style="width:100%"><tbody>
        {{range .NotifOutbox.Recientes}}
          <tr style="border-bottom:1px solid #f1f5f9">
            <td style="padding:.3rem 0;font-size:.74rem"><strong>{{.Canal}}</strong> · {{.TipoEvento}}</td>
            <td style="padding:.3rem 0;text-align:right"><span class="tag {{if eq .Estado "entregada"}}rt-disponible{{else if eq .Estado "fallida"}}rt-cerrado{{else}}t-media{{end}}">{{.Estado}}</span></td>
          </tr>
          <tr>
            <td colspan="2" style="padding:0 0 .45rem 0;font-size:.7rem;color:#64748b">
              {{if .Destino}}{{.Destino}}{{if .UltimoError}} · {{end}}{{end}}{{if .UltimoError}}err={{.UltimoError}}{{end}}
            </td>
          </tr>
        {{end}}
        </tbody></table>
      </div>
      {{end}}
    </div>
    {{if .EnProgreso}}
    <h4 style="margin:0 0 .5rem 0">{{tr "En progreso"}}</h4>
    <table style="width:100%;margin-bottom:1.2rem"><thead><tr><th>#</th><th>{{tr "dashboard.module"}}</th><th>{{tr "Agente"}}</th><th>{{tr "projects.title_label"}}</th></tr></thead><tbody>
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
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">{{tr "dashboard.by_state"}} <a href="/tareas" style="font-size:.9em;font-weight:normal;text-transform:none">{{tr "dashboard.view_all"}} →</a></h4>
        <table><tbody>
        {{range $est,$n := .Counts}}{{if gt $n 0}}
          <tr><td style="padding:.2rem .4rem"><a href="/tareas?estado={{$est}}"><span class="tag t-{{$est}}">{{$est}}</span></a></td><td style="padding:.2rem .6rem"><a href="/tareas?estado={{$est}}"><strong>{{$n}}</strong></a></td></tr>
        {{end}}{{end}}
        </tbody></table>
      </div>
      {{if .Abiertas}}
      <div>
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">{{tr "Propuestas abiertas"}}</h4>
        <table><thead><tr><th>{{tr "dashboard.code"}}</th><th>✓</th><th>✗</th><th>⏳</th></tr></thead><tbody>
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
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">{{tr "dashboard.active_runtimes"}} <a href="/runtimes" style="font-size:.9em;font-weight:normal;text-transform:none">{{tr "dashboard.view_all"}} →</a></h4>
        <table><thead><tr><th>{{tr "Agente"}}</th><th>{{tr "Estado"}}</th><th>{{tr "common.pid"}}</th><th>{{tr "runtime.children"}}</th></tr></thead><tbody>
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
        <h4 style="margin:0 0 .5rem 0;font-size:.85rem;color:#64748b;text-transform:uppercase;letter-spacing:.05em">{{tr "dashboard.recent_checkpoints"}} <a href="/time-travel" style="font-size:.9em;font-weight:normal;text-transform:none">{{tr "dashboard.view_all"}} →</a></h4>
        <table><thead><tr><th>{{tr "Agente"}}</th><th>{{tr "Tipo"}}</th><th>{{tr "time_travel.summary"}}</th></tr></thead><tbody>
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

const webTplOpenClaw = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1.2rem">
  <div>
    <h2 style="margin:0">OpenClaw Operator</h2>
    <small style="color:#64748b">Vista operativa server-first del supervisor</small>
  </div>
  <small style="color:#94a3b8">{{.Generado}}</small>
</div>
<meta http-equiv="refresh" content="10">
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

<div class="stats">
  <div class="stat"><div class="n">{{len .Status.AgentesActivos}}</div><div class="l">conectados</div></div>
  <div class="stat"><div class="n">{{len .Status.AgentesTrabajando}}</div><div class="l">trabajando</div></div>
  <div class="stat"><div class="n">{{len .EnCuota}}</div><div class="l">en cuota/cooldown</div></div>
  <div class="stat"><div class="n">{{.Status.CapacitySummary.CapacidadLibre}}</div><div class="l">capacidad libre</div></div>
  <div class="stat"><div class="n">{{.Status.CapacitySummary.BacklogLibre}}</div><div class="l">backlog libre</div></div>
  <div class="stat"><div class="n">{{len .Retenidas}}</div><div class="l">retenidas por cuota</div></div>
  <div class="stat"><div class="n">{{len .Recommended}}</div><div class="l">cola completa</div></div>
  <div class="stat"><div class="n">{{len .SafeRecommended}}</div><div class="l">cola segura</div></div>
  <div class="stat"><div class="n">{{sub (len .Recommended) (len .SafeRecommended)}}</div><div class="l">requieren arbitraje</div></div>
  <div class="stat"><div class="n">{{len .ReviewGates}}</div><div class="l">review gates</div></div>
  <div class="stat"><div class="n">{{len .Merges}}</div><div class="l">merges vivos</div></div>
</div>

{{if and .NextAction (or (not .NextSafeAction) (ne .NextAction.Action .NextSafeAction.Action) (ne .NextAction.Target .NextSafeAction.Target) (ne .NextAction.Assignee .NextSafeAction.Assignee))}}
<section style="background:linear-gradient(135deg,#0f172a,#1e293b);color:#fff;border-radius:.8rem;padding:1rem 1.2rem;margin-bottom:1.2rem">
  <div style="font-size:.74rem;letter-spacing:.08em;text-transform:uppercase;color:#93c5fd;margin-bottom:.35rem">Acción siguiente</div>
  <div style="font-size:1.05rem;font-weight:700">{{.NextAction.Action}}</div>
  <div style="margin-top:.25rem;color:#cbd5e1">{{.NextAction.Reason}}</div>
  <div style="margin-top:.45rem;font-size:.8rem;color:#93c5fd">objetivo={{.NextAction.Target}} · prioridad={{.NextAction.Priority}} · tipo={{.NextAction.Kind}}{{if .NextAction.Assignee}} · sugerido={{.NextAction.Assignee}}{{end}}</div>
  {{if .NextAction.AutoAplicable}}
  <div style="margin-top:.7rem;font-size:.82rem;color:#cbd5e1">Ya existe una acción segura equivalente más abajo.</div>
  {{else}}
  <div style="margin-top:.7rem;font-size:.82rem;color:#cbd5e1">Requiere revisión manual; usa la cola o el formulario específico.</div>
  {{end}}
</section>
{{end}}

{{if .NextSafeAction}}
<section style="background:linear-gradient(135deg,#0f766e,#115e59);color:#fff;border-radius:.8rem;padding:1rem 1.2rem;margin-bottom:1.2rem">
  <div style="font-size:.74rem;letter-spacing:.08em;text-transform:uppercase;color:#99f6e4;margin-bottom:.35rem">Siguiente acción segura</div>
  <div style="font-size:1.05rem;font-weight:700">{{.NextSafeAction.Action}}</div>
  <div style="margin-top:.25rem;color:#ccfbf1">{{.NextSafeAction.Reason}}</div>
  <div style="margin-top:.45rem;font-size:.8rem;color:#99f6e4">objetivo={{.NextSafeAction.Target}} · prioridad={{.NextSafeAction.Priority}} · tipo={{.NextSafeAction.Kind}}{{if .NextSafeAction.Assignee}} · sugerido={{.NextSafeAction.Assignee}}{{end}}</div>
  <form method="post" action="/openclaw?lang={{lang}}" style="display:flex;gap:.55rem;align-items:end;flex-wrap:wrap;margin-top:.8rem">
    <input type="hidden" name="kind" value="supervision_next">
    <button type="submit" class="btn-sm" style="background:#99f6e4;border-color:#99f6e4;color:#134e4a">Aplicar siguiente acción segura</button>
  </form>
</section>
{{end}}

<div style="display:grid;grid-template-columns:1.15fr .85fr;gap:1rem;align-items:start">
  <div style="display:grid;gap:1rem">
    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Flota disponible</h3>
      <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(120px,1fr));gap:.6rem;margin-bottom:.8rem">
        <div class="stat" style="padding:.7rem"><div class="n" style="font-size:1.35rem">{{.Status.CapacitySummary.WorkersDisponibles}}</div><div class="l">disponibles</div></div>
        <div class="stat" style="padding:.7rem"><div class="n" style="font-size:1.35rem">{{.Status.CapacitySummary.WorkersOciosos}}</div><div class="l">ociosos</div></div>
        <div class="stat" style="padding:.7rem"><div class="n" style="font-size:1.35rem">{{.Status.CapacitySummary.WorkersSaturados}}</div><div class="l">saturados</div></div>
      </div>
      {{if .Status.AgentesActivos}}
      <table style="width:100%"><thead><tr><th>Agente</th><th>Rol</th><th>Activa</th><th>Reservada</th><th>Cuota visible</th><th>Cuenta</th></tr></thead><tbody>
      {{range .Status.AgentesActivos}}
        <tr>
          <td><strong>{{.Nombre}}</strong></td>
          <td>{{orDash .Rol}}</td>
          <td>{{.CargaActiva}}</td>
          <td>{{.CargaReservada}}</td>
          <td>{{if .CuotaRestantePct}}{{.CuotaRestantePct}}%{{if .PresupuestoVentana}} · {{.PresupuestoVentana}}{{end}}{{else}}—{{end}}</td>
          <td>{{if .CuentaEmail}}{{.CuentaEmail}}{{else}}—{{end}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{else}}
      <p style="margin:0;color:#64748b">Sin workers disponibles ahora mismo.</p>
      {{end}}
      {{if .Status.AgentesSaturados}}
      <div style="margin-top:.8rem">
        <h4 style="margin:0 0 .45rem 0;font-size:.9rem">Agentes saturados</h4>
        <table style="width:100%"><thead><tr><th>Agente</th><th>Activa</th><th>Reservada</th><th>Cuenta</th></tr></thead><tbody>
        {{range .Status.AgentesSaturados}}
          <tr>
            <td><strong>{{.Nombre}}</strong></td>
            <td>{{.CargaActiva}}</td>
            <td>{{.CargaReservada}}</td>
            <td>{{if .CuentaEmail}}{{.CuentaEmail}}{{else}}—{{end}}</td>
          </tr>
        {{end}}
        </tbody></table>
      </div>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Frentes activos</h3>
      {{if .Status.TareasActivas}}
      <table style="width:100%"><thead><tr><th>#</th><th>Estado</th><th>Módulo</th><th>Agente</th><th>Título</th></tr></thead><tbody>
      {{range .Status.TareasActivas}}
        <tr>
          <td>{{.ID}}</td>
          <td><span class="tag t-{{.Estado}}">{{.Estado}}</span></td>
          <td><code>{{orDash .Modulo}}</code></td>
          <td>{{orDash .Agente}}</td>
          <td>{{.Titulo}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{else}}
      <p style="margin:0;color:#64748b">Sin tareas activas.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Reservas preparadas</h3>
      {{if .TareasReservadas}}
      <table style="width:100%"><thead><tr><th>#</th><th>Estado</th><th>Módulo</th><th>Agente</th><th>Título</th></tr></thead><tbody>
      {{range .TareasReservadas}}
        <tr>
          <td>{{.ID}}</td>
          <td><span class="tag t-{{.Estado}}">{{.Estado}}</span></td>
          <td><code>{{orDash .Modulo}}</code></td>
          <td>{{orDash .Agente}}</td>
          <td>{{.Titulo}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{else}}
      <p style="margin:0;color:#64748b">Sin reservas preparadas.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Cola completa del supervisor</h3>
      {{if .Recommended}}
      <ol style="margin:0;padding-left:1.2rem">
      {{range .Recommended}}
        <li style="margin-bottom:.45rem">
          <strong>{{.Action}}</strong>
          <div style="font-size:.8rem;color:#64748b">{{.Reason}}</div>
          <div style="font-size:.74rem;color:#94a3b8">objetivo={{.Target}} · prioridad={{.Priority}} · tipo={{.Kind}}{{if .Assignee}} · sugerido={{.Assignee}}{{end}}{{if .AutoAplicable}} · segura{{else}} · manual{{end}}</div>
        </li>
      {{end}}
      </ol>
      {{else}}
      <p style="margin:0;color:#64748b">Sin acciones recomendadas ahora mismo.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Cola segura</h3>
      {{if .SafeRecommended}}
      <form method="post" action="/openclaw?lang={{lang}}" style="display:flex;gap:.45rem;align-items:end;flex-wrap:wrap;margin:0 0 .75rem 0">
        <input type="hidden" name="kind" value="supervision_batch">
        <input type="hidden" name="max_items" value="{{len .SafeRecommended}}">
        <button type="submit" class="btn-sm" style="background:#0f766e;border-color:#0f766e;color:#ecfeff">Aplicar cola segura</button>
      </form>
      <ol style="margin:0;padding-left:1.2rem">
      {{range .SafeRecommended}}
        <li style="margin-bottom:.45rem">
          <strong>{{.Action}}</strong>
          <div style="font-size:.8rem;color:#64748b">{{.Reason}}</div>
          <div style="font-size:.74rem;color:#94a3b8">objetivo={{.Target}} · prioridad={{.Priority}} · tipo={{.Kind}}{{if .Assignee}} · sugerido={{.Assignee}}{{end}}</div>
          <form method="post" action="/openclaw?lang={{lang}}" style="display:flex;gap:.45rem;align-items:end;flex-wrap:wrap;margin-top:.35rem">
            <input type="hidden" name="kind" value="supervision_action">
            <input type="hidden" name="action" value="{{.Action}}">
            <input type="hidden" name="target" value="{{.Target}}">
            <label style="margin:0;font-size:.75rem;color:#475569">Assignee
              <input type="text" name="assignee" value="{{.Assignee}}" placeholder="Codex3" style="min-width:8rem">
            </label>
            <button type="submit" class="btn-sm">Aplicar</button>
          </form>
        </li>
      {{end}}
      </ol>
      {{else}}
      <p style="margin:0;color:#64748b">Sin acciones seguras ahora mismo.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Review e integración</h3>
      {{if .ReviewGates}}
      <table style="width:100%;margin-bottom:1rem"><thead><tr><th>Gate</th><th>Estado</th><th>Reviewer</th><th>Severidad</th></tr></thead><tbody>
      {{range .ReviewGates}}
        <tr>
          <td>#{{.ID}}{{if .TareaID}} · tarea {{.TareaID}}{{end}}</td>
          <td><span class="tag t-media">{{.Estado}}</span></td>
          <td>{{orDash .ReviewerAgente}}</td>
          <td>{{orDash .SeverityMax}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{range .ReviewGates}}
      <form method="post" action="/openclaw?lang={{lang}}" style="display:grid;grid-template-columns:1fr 1fr 1fr auto;gap:.45rem;align-items:end;margin:.55rem 0;padding:.65rem;border:1px solid #e2e8f0;border-radius:.45rem;background:white">
        <input type="hidden" name="kind" value="review_gate">
        <input type="hidden" name="gate_id" value="{{.ID}}">
        <label style="margin:0">Estado
          <select name="estado">
            <option value="pendiente" {{if eq .Estado "pendiente"}}selected{{end}}>pendiente</option>
            <option value="en_revision" {{if eq .Estado "en_revision"}}selected{{end}}>en_revision</option>
            <option value="cambios_pedidos" {{if eq .Estado "cambios_pedidos"}}selected{{end}}>cambios_pedidos</option>
            <option value="aprobado" {{if eq .Estado "aprobado"}}selected{{end}}>aprobado</option>
            <option value="bloqueado" {{if eq .Estado "bloqueado"}}selected{{end}}>bloqueado</option>
          </select>
        </label>
        <label style="margin:0">Reviewer
          <input type="text" name="reviewer_agente" value="{{.ReviewerAgente}}">
        </label>
        <label style="margin:0">Findings JSON
          <input type="text" name="findings_json" value="{{.FindingsJSON}}">
        </label>
        <button type="submit" class="btn-sm">Guardar gate</button>
      </form>
      {{end}}
      {{end}}
      {{if .Signals}}
      <div style="font-size:.8rem;color:#334155;margin-bottom:.45rem;font-weight:600">Señales recientes</div>
      <table style="width:100%;margin-bottom:1rem"><thead><tr><th>Tipo</th><th>Agente</th><th>Proyecto</th><th>Mensaje</th></tr></thead><tbody>
      {{range .Signals}}
        <tr>
          <td>{{.Event.Kind}}</td>
          <td>{{orDash .Agent}}</td>
          <td>{{orDash .Project}}</td>
          <td>{{orDash .Event.Message}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{end}}
      {{if .Merges}}
      <div style="font-size:.8rem;color:#334155;margin-bottom:.45rem;font-weight:600">Solicitudes de merge</div>
      <table style="width:100%"><thead><tr><th>#</th><th>Estado</th><th>Proyecto</th><th>Ramas</th></tr></thead><tbody>
      {{range .Merges}}
        <tr>
          <td>#{{.ID}}</td>
          <td><span class="tag t-media">{{.Estado}}</span></td>
          <td>{{orDash .ProyectoSlug}}</td>
          <td>{{orDash .SourceBranch}} → {{orDash .TargetBranch}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{end}}
      {{if .NormalizedEvents}}
      <div style="font-size:.8rem;color:#334155;margin:.75rem 0 .45rem;font-weight:600">Eventos normalizados del supervisor</div>
      <table style="width:100%"><thead><tr><th>Evento</th><th>Origen</th><th>Proyecto</th><th>Acción sugerida</th></tr></thead><tbody>
      {{range .NormalizedEvents}}
        <tr>
          <td><code>{{.NormalizedEvent}}</code></td>
          <td>{{.Source}}</td>
          <td>{{orDash .Project}}</td>
          <td>{{orDash .SuggestedAction}}</td>
        </tr>
        <tr><td colspan="4" style="font-size:.74rem;color:#64748b;padding-bottom:.4rem">{{orDash .Message}}</td></tr>
      {{end}}
      </tbody></table>
      {{end}}
      {{if and (not .ReviewGates) (not .Signals) (not .Merges) (not .NormalizedEvents)}}
      <p style="margin:0;color:#64748b">Sin review gates, señales ni merges vivos.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Propuestas abiertas</h3>
      {{if .Status.PropuestasAbiertas}}
      <table style="width:100%"><thead><tr><th>Código</th><th>Título</th><th>Acuerdo</th><th>Desacuerdo</th><th>Pendiente</th></tr></thead><tbody>
      {{range .Status.PropuestasAbiertas}}
        <tr>
          <td><code>{{.Codigo}}</code></td>
          <td>{{.Titulo}}</td>
          <td>{{.Acuerdo}}</td>
          <td>{{.Desacuerdo}}</td>
          <td>{{.Pendiente}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{else}}
      <p style="margin:0;color:#64748b">Sin propuestas abiertas.</p>
      {{end}}
    </section>
  </div>

  <div style="display:grid;gap:1rem">
    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Integración server-first</h3>
      <table style="width:100%;margin-bottom:.8rem"><tbody>
        <tr><td style="font-weight:600">MCP HTTP</td><td style="text-align:right"><code>{{.Integration.MCPEndpoint}}</code></td></tr>
        <tr><td style="font-weight:600">Transporte</td><td style="text-align:right"><code>{{.Integration.Transport}}</code></td></tr>
        <tr><td style="font-weight:600">Endpoint configurado</td><td style="text-align:right"><code>{{.Integration.Endpoint}}</code></td></tr>
        <tr><td style="font-weight:600">Workspace</td><td style="text-align:right"><code>{{.Integration.WorkspaceMode}}</code></td></tr>
        <tr><td style="font-weight:600">Prefijo agente</td><td style="text-align:right"><code>{{.Integration.AgentPrefix}}</code></td></tr>
        <tr><td style="font-weight:600">Require identity</td><td style="text-align:right"><code>{{.Integration.RequireIdentity}}</code></td></tr>
        <tr><td style="font-weight:600">Plugin local</td><td style="text-align:right"><code>{{.Integration.PluginPath}}</code></td></tr>
        <tr><td style="font-weight:600">Runbook</td><td style="text-align:right"><code>{{.Integration.RunbookPath}}</code></td></tr>
      </tbody></table>
      <div style="font-size:.78rem;color:#475569;display:grid;gap:.2rem">
        <div>Smoke: <code>{{.Integration.SmokeCommand}}</code></div>
        <div>MCP canónico: <code>/api/mcp</code>{{if not .Integration.Configured}} · preset aún no guardado en config{{end}}</div>
        <div>Gateway OpenClaw: {{if .Integration.GatewayConfigured}}configurado{{else}}sin configurar{{end}} · Telegram: {{if .Integration.TelegramConfigured}}configurado{{else}}sin configurar{{end}}</div>
      </div>
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Workers fuera del pool</h3>
      {{if .EnCuota}}
      <table style="width:100%"><thead><tr><th>Agente</th><th>Estado</th><th>Reanima</th></tr></thead><tbody>
      {{range .EnCuota}}
        <tr>
          <td>{{.Nombre}}</td>
          <td>{{orDash .EstadoCuota}}</td>
          <td>{{if .ReanimarAt}}{{.ReanimarAt.Local.Format "2006-01-02 15:04"}}{{else}}—{{end}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{range .EnCuota}}
      <form method="post" action="/openclaw?lang={{lang}}" style="display:flex;gap:.45rem;align-items:center;flex-wrap:wrap;margin-top:.5rem">
        <input type="hidden" name="kind" value="agente">
        <input type="hidden" name="agente" value="{{.Nombre}}">
        <span style="font-size:.82rem;color:#475569;min-width:8rem">{{.Nombre}}</span>
        <button type="submit" class="btn-sm" name="accion" value="reset-reanimacion">Reset reanimación</button>
        {{if not .Habilitado}}<button type="submit" class="btn-sm" name="accion" value="rehabilitar">Rehabilitar</button>{{end}}
      </form>
      {{end}}
      {{else}}
      <p style="margin:0;color:#64748b">Sin agentes retenidos por cuota.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Tareas retenidas por cuota</h3>
      {{if .Retenidas}}
      <table style="width:100%"><thead><tr><th>#</th><th>Agente</th><th>Título</th></tr></thead><tbody>
      {{range .Retenidas}}
        <tr>
          <td>{{.ID}}</td>
          <td>{{orDash .Agente}}</td>
          <td>{{.Titulo}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{else}}
      <p style="margin:0;color:#64748b">Sin tareas retenidas.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Guidance durable pendiente</h3>
      {{if .MailboxPendiente}}
      <table style="width:100%"><thead><tr><th>Agente</th><th>Mensajes</th><th>Kinds</th><th>Antigüedad</th></tr></thead><tbody>
      {{range .MailboxPendiente}}
        <tr>
          <td>{{.Agente}}</td>
          <td>{{.Count}}</td>
          <td>{{orDash .KindsCSV}}</td>
          <td>{{if .OldestAgeMin}}{{.OldestAgeMin}} min{{else}}0 min{{end}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{else}}
      <p style="margin:0;color:#64748b">Sin guidance durable pendiente.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Pipeline del supervisor</h3>
      {{if .PipelineStates}}
      <table style="width:100%;margin-bottom:.8rem"><thead><tr><th>Pipeline</th><th>Proyecto</th><th>Fase</th><th>Estado</th></tr></thead><tbody>
      {{range .PipelineStates}}
        <tr>
          <td>{{orDash .PipelineName}}</td>
          <td>{{orDash .ProyectoSlug}}</td>
          <td><code>{{orDash .CurrentPhase}}</code></td>
          <td><span class="tag t-media">{{.Status}}</span></td>
        </tr>
      {{end}}
      </tbody></table>
      {{else}}
      <p style="margin:0;color:#64748b">Sin pipeline explícita registrada.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">Threads y subagentes</h3>
      {{if .ThreadSessions}}
      <table style="width:100%"><thead><tr><th>Sesión</th><th>Líder</th><th>Subagentes</th><th>Activos</th></tr></thead><tbody>
      {{range .ThreadSessions}}
        <tr>
          <td>{{if .SessionID}}{{.SessionID}}{{else}}&lt;default&gt;{{end}}</td>
          <td>{{orDash .LeaderThreadID}}</td>
          <td>{{len .AllSubagentThreadIDs}}</td>
          <td>{{len .ActiveSubagentThreadIDs}}</td>
        </tr>
      {{end}}
      </tbody></table>
      {{else}}
      <p style="margin:0;color:#64748b">Sin threads registradas todavía.</p>
      {{end}}
    </section>

    <section style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.6rem;padding:1rem">
      <h3 style="margin:0 0 .7rem 0">OpenClaw Gateway y notificaciones</h3>
      <table style="width:100%;margin-bottom:.8rem"><tbody>
      {{range .Notificaciones.Canales}}
        <tr>
          <td style="font-weight:600">{{.Nombre}}</td>
          <td style="text-align:right"><span class="tag {{if .Activo}}rt-disponible{{else}}rt-cerrado{{end}}">{{if .Activo}}activo{{else}}inactivo{{end}}</span></td>
        </tr>
        <tr><td colspan="2" style="font-size:.75rem;color:#64748b;padding-bottom:.4rem">{{.Detalle}}</td></tr>
      {{end}}
      </tbody></table>
      {{if .NotifOutbox.Recientes}}
      <table style="width:100%"><thead><tr><th>Canal</th><th>Evento</th><th>Estado</th></tr></thead><tbody>
      {{range .NotifOutbox.Recientes}}
        <tr>
          <td>{{.Canal}}</td>
          <td>{{.TipoEvento}}</td>
          <td><span class="tag {{if eq .Estado "entregada"}}rt-disponible{{else if eq .Estado "fallida"}}rt-cerrado{{else}}t-media{{end}}">{{.Estado}}</span></td>
        </tr>
        <tr><td colspan="3" style="font-size:.74rem;color:#64748b;padding-bottom:.4rem">{{orDash .Destino}}{{if .UltimoError}} · err={{.UltimoError}}{{end}}</td></tr>
      {{end}}
      </tbody></table>
      {{else}}
      <p style="margin:0;color:#64748b">Sin entregas recientes.</p>
      {{end}}
    </section>

  </div>
</div>
{{end}}
`

// ─── Tareas: lista ────────────────────────────────────────────────────────────

const webTplTareas = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1rem">
  <h2 style="margin:0">{{tr "Tareas"}} <small style="font-size:.5em;color:#94a3b8">{{len .Tareas}}</small></h2>
</div>
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

<details class="form-panel">
  <summary>＋ {{tr "tasks.new"}}</summary>
  <form method="POST" action="/tareas/nueva" style="margin-top:.8rem">
    <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:.5rem">
      <div><label>{{tr "projects.title_label"}} *</label><input type="text" name="titulo" required placeholder="{{tr "tasks.title_placeholder"}}"></div>
      <div><label>{{tr "tasks.module"}}</label><input type="text" name="modulo" placeholder="{{tr "tasks.module_placeholder"}}"></div>
      <div><label>{{tr "tasks.linked_proposal"}}</label><input type="text" name="propuesta" placeholder="OP-030"></div>
    </div>
    <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:.5rem;margin-top:.4rem">
      <div><label>{{tr "Prioridad"}}</label>
        <select name="prioridad">
          <option value="alta">{{tr "alta"}}</option>
          <option value="media" selected>{{tr "media"}}</option>
          <option value="baja">{{tr "baja"}}</option>
        </select>
      </div>
      <div><label>{{tr "tasks.assign_agent"}}</label>
        <select name="agente">
          <option value="">{{tr "tasks.unassigned"}}</option>
          {{range .Agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
        </select>
      </div>
      <div><label>{{tr "common.description"}}</label><input type="text" name="descripcion" placeholder="{{tr "tasks.description_placeholder"}}"></div>
    </div>
    <button type="submit" class="btn-sm" style="margin-top:.6rem">{{tr "tasks.create"}}</button>
  </form>
</details>

<div class="filtros">
  <a href="/tareas"{{if eqStr .Filtro ""}} class="sel"{{end}}>{{tr "Todas"}}</a>
  <a href="/tareas?estado=en_progreso"{{if eqStr .Filtro "en_progreso"}} class="sel"{{end}}>{{tr "En progreso"}}</a>
  <a href="/tareas?estado=asignada"{{if eqStr .Filtro "asignada"}} class="sel"{{end}}>{{tr "Asignadas"}}</a>
  <a href="/tareas?estado=libre"{{if eqStr .Filtro "libre"}} class="sel"{{end}}>{{tr "Libres"}}</a>
  <a href="/tareas?estado=bloqueada"{{if eqStr .Filtro "bloqueada"}} class="sel"{{end}}>{{tr "Bloqueadas"}}</a>
  <a href="/tareas?estado=backlog"{{if eqStr .Filtro "backlog"}} class="sel"{{end}}>{{tr "backlog"}}</a>
  <a href="/tareas?estado=completada"{{if eqStr .Filtro "completada"}} class="sel"{{end}}>{{tr "Completadas"}}</a>
</div>

{{if .Tareas}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>#</th><th>{{tr "Estado"}}</th><th>{{tr "Prioridad"}}</th><th>{{tr "tasks.module"}}</th><th>{{tr "Agente"}}</th><th>{{tr "projects.title_label"}}</th><th></th></tr></thead>
  <tbody>
  {{range .Tareas}}
    <tr>
      <td style="color:#94a3b8">{{.ID}}</td>
      <td><span class="tag t-{{.Estado}}">{{tr .Estado}}</span></td>
      <td><span class="tag t-{{.Prioridad}}">{{tr .Prioridad}}</span></td>
      <td><code style="font-size:.8em">{{.Modulo}}</code></td>
      <td>{{.Agente}}</td>
      <td>{{.Titulo}}</td>
      <td><a href="/tareas/{{.ID}}" class="btn-sm">{{tr "tasks.manage"}} →</a></td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>
{{else}}
<p style="color:#94a3b8">{{tr "tasks.none_visible"}}</p>
{{end}}
{{end}}
`

// ─── Tarea: detalle + acciones ────────────────────────────────────────────────

const webTplTareaDetalle = `{{define "content"}}
<a href="/tareas" style="font-size:.85rem;color:#64748b">← {{tr "tasks.back"}}</a>
<h2 style="margin:.5rem 0">#{{.T.ID}} — {{.T.Titulo}}</h2>
<p>
  <span class="tag t-{{.T.Estado}}">{{tr .T.Estado}}</span>
  <span class="tag t-{{.T.Prioridad}}">{{tr .T.Prioridad}}</span>
  <span style="color:#64748b;font-size:.85rem;margin-left:.5rem">{{tr "tasks.module"}}: {{.T.Modulo}} · {{tr "Agente"}}: {{.T.Agente}}</span>
</p>

{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

{{$id := .T.ID}}
{{$estado := .T.Estado}}
{{$agentes := .Agentes}}

{{if or (eqStr $estado "libre") (eqStr $estado "backlog")}}
<div class="action-box">
  <h4>{{tr "tasks.assign_title"}}</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="tomar">
    <div><label>{{tr "Agente"}}</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <button type="submit" class="btn-sm">{{tr "tasks.assign_submit"}}</button>
  </form>
</div>
{{end}}

{{if eqStr $estado "asignada"}}
<div class="action-box">
  <h4>{{tr "tasks.start_title"}}</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="iniciar">
    <div><label>{{tr "Agente"}}</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}"{{if eqStr .Nombre $.T.Agente}} selected{{end}}>{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <button type="submit" class="btn-sm">{{tr "tasks.start_submit"}}</button>
  </form>
</div>
{{end}}

{{if or (eqStr $estado "asignada") (eqStr $estado "en_progreso")}}
<div class="action-box">
  <h4>{{tr "tasks.complete_title"}}</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="completar">
    <div><label>{{tr "Agente"}}</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}"{{if eqStr .Nombre $.T.Agente}} selected{{end}}>{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <div><label>{{tr "tasks.commit_optional"}}</label><input type="text" name="commit" placeholder="abc1234" style="width:140px"></div>
    <button type="submit" class="btn-sm" style="background:#16a34a;border-color:#16a34a;color:#fff">{{tr "tasks.complete_submit"}}</button>
  </form>
</div>
{{end}}

{{if eqStr $estado "en_progreso"}}
<div class="action-box">
  <h4>{{tr "tasks.block_title"}}</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="bloquear">
    <div><label>{{tr "Agente"}}</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}"{{if eqStr .Nombre $.T.Agente}} selected{{end}}>{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <div style="flex:1"><label>{{tr "tasks.block_reason"}}</label><input type="text" name="motivo" required placeholder="{{tr "tasks.block_reason_placeholder"}}" style="width:100%"></div>
    <button type="submit" class="btn-sm" style="background:#dc2626;border-color:#dc2626;color:#fff">{{tr "tasks.block_submit"}}</button>
  </form>
</div>
{{end}}

{{if eqStr $estado "bloqueada"}}
<div class="action-box">
  <h4>{{tr "tasks.unblock_title"}}</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="desbloquear">
    <div><label>{{tr "Agente"}}</label>
      <select name="agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <div style="flex:1"><label>{{tr "tasks.resolution"}}</label><input type="text" name="resolucion" required placeholder="{{tr "tasks.resolution_placeholder"}}" style="width:100%"></div>
    <button type="submit" class="btn-sm">{{tr "tasks.unblock_submit"}}</button>
  </form>
</div>
{{end}}

<div class="action-box">
  <h4>{{tr "tasks.reassign_title"}}</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="reasignar">
    <div><label>{{tr "tasks.new_agent"}}</label>
      <select name="nuevo_agente">
        {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
      </select>
    </div>
    <button type="submit" class="btn-sm">{{tr "tasks.reassign_submit"}}</button>
  </form>
</div>

<div class="action-box">
  <h4>{{tr "tasks.add_note_title"}}</h4>
  <form method="POST" action="/tareas/{{$id}}/accion" style="display:flex;gap:.5rem;align-items:flex-end">
    <input type="hidden" name="accion" value="nota">
    <div><label>{{tr "Agente"}}</label>
      <select name="agente">
        {{range $agentes}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}
      </select>
    </div>
    <div style="flex:1"><label>{{tr "common.note"}}</label><input type="text" name="nota" required placeholder="{{tr "tasks.note_placeholder"}}" style="width:100%"></div>
    <button type="submit" class="btn-sm">{{tr "tasks.add_note_submit"}}</button>
  </form>
</div>

{{if not (eqStr $estado "backlog")}}
<div class="action-box">
  <h4>{{tr "tasks.move_backlog_title"}}</h4>
  <form method="POST" action="/tareas/{{$id}}/accion">
    <input type="hidden" name="accion" value="backlog">
    <button type="submit" class="btn-sm">{{tr "tasks.move_backlog_submit"}}</button>
  </form>
</div>
{{end}}
{{end}}
`

// ─── Propuestas: lista ────────────────────────────────────────────────────────

const webTplPropuestas = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1rem">
  <h2 style="margin:0">{{tr "Propuestas (OPs)"}} <small style="font-size:.5em;color:#94a3b8">{{len .Propuestas}}</small></h2>
</div>
{{if .Msg}}<div class="alert-ok">✓ {{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">✗ {{.Err}}</div>{{end}}

<details class="form-panel">
  <summary>＋ {{tr "proposals.new"}}</summary>
  <form method="POST" action="/propuestas/nueva" style="margin-top:.8rem">
    <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:.5rem">
      <div><label>{{tr "projects.title_label"}} *</label><input type="text" name="titulo" required placeholder="{{tr "proposals.title_placeholder"}}"></div>
      <div><label>{{tr "Tipo"}}</label>
        <select name="tipo">
          <option value="implementacion">{{tr "implementacion"}}</option>
          <option value="arquitectura">{{tr "arquitectura"}}</option>
          <option value="seguridad">{{tr "seguridad"}}</option>
          <option value="backlog">{{tr "backlog"}}</option>
          <option value="otro">{{tr "otro"}}</option>
        </select>
      </div>
      <div><label>{{tr "proposals.code_optional"}}</label><input type="text" name="codigo" placeholder="{{tr "proposals.code_placeholder"}}"></div>
    </div>
    <div style="margin-top:.4rem"><label>{{tr "common.description"}}</label>
      <textarea name="descripcion" rows="2" placeholder="{{tr "proposals.description_placeholder"}}" style="width:100%"></textarea>
    </div>
    <button type="submit" class="btn-sm" style="margin-top:.4rem">{{tr "proposals.create"}}</button>
  </form>
</details>

<div class="filtros">
  <a href="/propuestas"{{if eqStr .Filtro ""}} class="sel"{{end}}>{{tr "Todas"}}</a>
  <a href="/propuestas?estado=abierta"{{if eqStr .Filtro "abierta"}} class="sel"{{end}}>{{tr "proposals.open"}}</a>
  <a href="/propuestas?estado=consenso"{{if eqStr .Filtro "consenso"}} class="sel"{{end}}>{{tr "proposals.consensus"}}</a>
  <a href="/propuestas?estado=backlog"{{if eqStr .Filtro "backlog"}} class="sel"{{end}}>{{tr "backlog"}}</a>
  <a href="/propuestas?estado=rechazada"{{if eqStr .Filtro "rechazada"}} class="sel"{{end}}>{{tr "proposals.rejected"}}</a>
</div>

{{range .Propuestas}}
<details class="card">
  <summary>
    <span style="color:#94a3b8;margin-right:.4rem">{{.Codigo}}</span>
    <span class="tag t-{{.Estado}}">{{tr .Estado}}</span>
    &nbsp;<strong>{{trunc .Titulo 65}}</strong>
    <span style="color:#94a3b8;font-weight:normal;font-size:.82em;margin-left:.5rem">· {{.PropuestoPor}} · {{.Fecha}}</span>
    <a href="/propuestas/{{.Codigo}}" style="float:right;font-size:.8em;font-weight:normal;color:#2563eb" onclick="event.stopPropagation()">{{tr "proposals.manage"}} →</a>
  </summary>
  {{if .Descripcion}}<p style="color:#475569;font-size:.88em;margin:.6rem 0">{{.Descripcion}}</p>{{end}}
  {{if .Votos}}
  <table style="font-size:.82em"><thead><tr><th>{{tr "Agente"}}</th><th>{{tr "projects.position"}}</th><th>{{tr "projects.comment"}}</th></tr></thead><tbody>
  {{range .Votos}}
    <tr>
      <td><strong>{{.Agente}}</strong></td>
      <td><span class="tag t-{{.Posicion}}">{{tr .Posicion}}</span></td>
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
  <h2 style="margin:0">{{tr "Runtimes"}} <small style="font-size:.5em;color:#94a3b8">{{len .Runtimes}}</small></h2>
</div>

{{if .Msg}}<div class="alert-ok">{{.Msg}}</div>{{end}}
{{if .Err}}<div class="alert-err">{{.Err}}</div>{{end}}

<div class="filtros">
  <a href="/runtimes"{{if eqStr .Filtro ""}} class="sel"{{end}}>{{tr "Todas"}}</a>
  <a href="/runtimes?activos=true"{{if eqStr .Filtro "true"}} class="sel"{{end}}>{{tr "runtime.filter.active"}}</a>
  <a href="/runtimes?activos=false"{{if eqStr .Filtro "false"}} class="sel"{{end}}>{{tr "runtime.filter.closed"}}</a>
</div>

<div class="action-box">
  <h4>{{tr "runtime.control.title"}}</h4>
  <form method="POST" action="/runtimes/control" style="display:grid;grid-template-columns:1fr 1fr 1fr 1fr;gap:.6rem">
    <div><label>{{tr "Agente"}}</label><input type="text" name="agente" placeholder="Codex1" required></div>
    <div><label>{{tr "Proyecto"}}</label><input type="text" name="proyecto" placeholder="orquestador"></div>
    <div>
      <label>{{tr "Acción"}}</label>
      <select name="accion">
        <option value="arrancar">{{tr "Arrancar"}}</option>
        <option value="pausar">{{tr "Pausar"}}</option>
        <option value="continuar">{{tr "Continuar"}}</option>
        <option value="detener">{{tr "Detener"}}</option>
      </select>
    </div>
    <div><label>{{tr "Motivo"}}</label><input type="text" name="motivo" placeholder="{{tr "runtime.control.reason_placeholder"}}"></div>
    <div><label>{{tr "Conector"}}</label><input type="text" name="conector" placeholder="codex-cli"></div>
    <div><label>{{tr "Modelo"}}</label><input type="text" name="modelo" placeholder="gpt-5.4"></div>
    <div><label>{{tr "Reasoning"}}</label><input type="text" name="razonamiento" placeholder="high"></div>
    <div><label>{{tr "Perfil"}}</label><input type="text" name="perfil" placeholder="implementacion"></div>
    <div style="grid-column:1/-1"><button type="submit" class="btn-sm">{{tr "runtime.control.enqueue"}}</button></div>
  </form>
</div>

{{if .Runtimes}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>ID</th><th>{{tr "Estado"}}</th><th>{{tr "Agente"}}</th><th>{{tr "Proyecto"}}</th><th>{{tr "Provider"}}</th><th>{{tr "Connector"}}</th><th>{{tr "common.pid"}}</th><th>{{tr "runtime.children"}}</th><th>{{tr "branch"}}</th><th>{{tr "runtime.last_signal"}}</th></tr></thead>
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
<p style="color:#94a3b8">{{tr "runtime.none_visible"}}</p>
{{end}}
{{end}}
`

const webTplRuntimeDetalle = `{{define "content"}}
<a href="/runtimes" style="font-size:.85rem;color:#64748b">← {{tr "runtime.back_to_list"}}</a>
<h2 style="margin:.5rem 0">{{tr "runtime.detail.title"}} #{{.Runtime.ID}} — {{.Runtime.Agente}}</h2>
<p>
  <span class="tag rt-{{.Runtime.Estado}}">{{.Runtime.Estado}}</span>
  <span style="color:#64748b;font-size:.85rem;margin-left:.5rem">{{tr "Proyecto"}}: {{.Runtime.Proyecto}} · {{tr "Provider"}}: {{.Runtime.Provider}} · {{tr "Connector"}}: {{.Runtime.Connector}}</span>
</p>
<p style="margin-top:-.2rem">
  <a href="/time-travel?agente={{.Runtime.Agente}}{{if neStr .Runtime.Proyecto "—"}}&proyecto={{.Runtime.Proyecto}}{{end}}" style="font-size:.84rem">{{tr "runtime.view_checkpoints"}} →</a>
</p>

<div class="stats" style="grid-template-columns:repeat(auto-fit,minmax(110px,1fr));margin-bottom:1rem">
  <div class="stat"><div class="n" style="font-size:1.2rem">{{.Runtime.PID}}</div><div class="l">{{tr "common.pid"}}</div></div>
  <div class="stat"><div class="n" style="font-size:1.2rem">{{.Runtime.Hijos}}</div><div class="l">{{tr "runtime.children"}}</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Runtime.Branch}}</div><div class="l">{{tr "branch"}}</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Runtime.Modelo}}</div><div class="l">{{tr "runtime.model"}}</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Runtime.Razonamiento}}</div><div class="l">{{tr "Reasoning"}}</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Runtime.UltimaSenal}}</div><div class="l">{{tr "runtime.last_signal"}}</div></div>
</div>

<h4 style="margin:0 0 .6rem 0">{{tr "runtime.samples.title"}}</h4>
{{if .Muestras}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>{{tr "runtime.created_feminine"}}</th><th>{{tr "Estado"}}</th><th>{{tr "runtime.cpu_pct"}}</th><th>{{tr "runtime.mem"}}</th><th>{{tr "runtime.rss"}}</th><th>{{tr "runtime.fds"}}</th><th>{{tr "runtime.children"}}</th><th>{{tr "runtime.threads"}}</th><th>{{tr "runtime.source"}}</th></tr></thead>
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
<p style="color:#94a3b8">{{tr "runtime.samples.none"}}</p>
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

<h4 style="margin:1.1rem 0 .6rem 0">{{tr "runtime.timeline.title"}}</h4>
{{if .Timeline}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>{{tr "runtime.when"}}</th><th>{{tr "runtime.channel"}}</th><th>{{tr "runtime.event"}}</th><th>{{tr "Estado"}}</th><th>{{tr "runtime.detail"}}</th></tr></thead>
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
<p style="color:#94a3b8">{{tr "runtime.timeline.none"}}</p>
{{end}}
{{end}}
`

const webTplTimeTravel = `{{define "content"}}
<div style="display:flex;justify-content:space-between;align-items:baseline;margin-bottom:1rem">
  <h2 style="margin:0">{{tr "Time Travel"}} <small style="font-size:.5em;color:#94a3b8">{{len .Checkpoints}}</small></h2>
</div>

<div class="filtros">
  <a href="/time-travel"{{if and (eqStr .FiltroAgente "") (eqStr .FiltroProyecto "") (eqStr .FiltroKind "")}} class="sel"{{end}}>{{tr "Todas"}}</a>
  <a href="/time-travel?kind=checkpoint"{{if eqStr .FiltroKind "checkpoint"}} class="sel"{{end}}>{{tr "time_travel.filter.checkpoints"}}</a>
  <a href="/time-travel?kind=handoff"{{if eqStr .FiltroKind "handoff"}} class="sel"{{end}}>{{tr "time_travel.filter.handoffs"}}</a>
  <a href="/time-travel?kind=sync_status"{{if eqStr .FiltroKind "sync_status"}} class="sel"{{end}}>{{tr "time_travel.filter.sync_status"}}</a>
</div>

<form method="GET" action="/time-travel" style="display:grid;grid-template-columns:1fr 1fr 1fr auto;gap:.5rem;align-items:end;margin-bottom:1rem">
  <div><label>{{tr "Agente"}}</label><input type="text" name="agente" value="{{.FiltroAgente}}" placeholder="Codex1"></div>
  <div><label>{{tr "Proyecto"}}</label><input type="text" name="proyecto" value="{{.FiltroProyecto}}" placeholder="orquestador"></div>
  <div><label>{{tr "Tipo"}}</label><input type="text" name="kind" value="{{.FiltroKind}}" placeholder="checkpoint"></div>
  <div><button type="submit" class="btn-sm">{{tr "time_travel.filter.submit"}}</button></div>
</form>

{{if .Checkpoints}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>ID</th><th>{{tr "time_travel.created"}}</th><th>{{tr "Agente"}}</th><th>{{tr "Proyecto"}}</th><th>{{tr "Tipo"}}</th><th>{{tr "time_travel.summary"}}</th><th>{{tr "branch"}}</th><th>{{tr "time_travel.resume"}}</th><th></th></tr></thead>
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
      <td><a href="/time-travel/{{.ID}}" class="btn-sm">{{tr "time_travel.inspect"}} →</a></td>
    </tr>
  {{end}}
  </tbody>
</table>
</div>
{{else}}
<p style="color:#94a3b8">{{tr "time_travel.none_visible"}}</p>
{{end}}
{{end}}
`

const webTplTimeTravelDetalle = `{{define "content"}}
<a href="/time-travel" style="font-size:.85rem;color:#64748b">← {{tr "time_travel.back"}}</a>
<h2 style="margin:.5rem 0">{{tr "time_travel.checkpoint"}} #{{.Checkpoint.ID}} — {{.Checkpoint.Agente}}</h2>
<p>
  <span class="tag t-media">{{.Checkpoint.CheckpointKind}}</span>
  <span style="color:#64748b;font-size:.85rem;margin-left:.5rem">{{tr "Proyecto"}}: {{.Checkpoint.Proyecto}} · {{tr "branch"}}: {{.Checkpoint.Branch}} · {{.Checkpoint.Creado}}</span>
</p>

<div class="stats" style="grid-template-columns:repeat(auto-fit,minmax(150px,1fr));margin-bottom:1rem">
  <div class="stat"><div class="n" style="font-size:1rem">{{.Checkpoint.ResumeStrategy}}</div><div class="l">{{tr "time_travel.resume"}}</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Checkpoint.Source}}</div><div class="l">{{tr "time_travel.source"}}</div></div>
  <div class="stat"><div class="n" style="font-size:1rem">{{.Checkpoint.Branch}}</div><div class="l">{{tr "branch"}}</div></div>
  <div class="stat"><div class="n" style="font-size:.92rem">{{trunc .Checkpoint.CWD 28}}</div><div class="l">{{tr "time_travel.cwd"}}</div></div>
</div>

<h4 style="margin:0 0 .5rem 0">{{tr "time_travel.summary"}}</h4>
<p style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:.4rem;padding:.8rem 1rem">{{.Checkpoint.Resumen}}</p>

<h4 style="margin:1rem 0 .5rem 0">{{tr "time_travel.payload"}}</h4>
<pre style="background:#0f172a;color:#e2e8f0;padding:1rem;border-radius:.5rem;overflow:auto;font-size:.8rem">{{.Payload}}</pre>

<h4 style="margin:1rem 0 .5rem 0">{{tr "Memoria de proyecto"}}</h4>
{{if .Memoria}}
<div style="overflow-x:auto">
<table>
  <thead><tr><th>{{tr "time_travel.entity"}}</th><th>{{tr "Tipo"}}</th><th>{{tr "time_travel.verified_by"}}</th><th>{{tr "time_travel.value"}}</th></tr></thead>
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
<p style="color:#94a3b8">{{tr "time_travel.memory.none"}}</p>
{{end}}
{{end}}
`

// ─── Propuesta: detalle + acciones ───────────────────────────────────────────

const webTplPropuestaDetalle = `{{define "content"}}
<a href="/propuestas" style="font-size:.85rem;color:#64748b">← {{tr "proposals.back"}}</a>
<h2 style="margin:.5rem 0">{{.P.Codigo}} — {{.P.Titulo}}</h2>
<p>
  <span class="tag t-{{.P.Estado}}">{{tr .P.Estado}}</span>
  <span class="tag" style="background:#e2e8f0;color:#475569">{{tr .P.Tipo}}</span>
  <span style="color:#64748b;font-size:.85rem;margin-left:.5rem">{{tr "proposals.proposed_by"}} {{.P.PropuestoPor}} · {{.P.Fecha}}</span>
  {{if .P.CerradaFecha}}<span style="color:#94a3b8;font-size:.82rem;margin-left:.5rem">({{tr "proposals.closed_on"}} {{.P.CerradaFecha}})</span>{{end}}
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
    <h4>{{tr "proposals.vote_title"}}</h4>
    <form method="POST" action="/propuestas/{{$codigo}}/accion">
      <input type="hidden" name="accion" value="votar">
      <div style="display:flex;gap:.5rem;align-items:flex-end;flex-wrap:wrap">
        <div><label>{{tr "Agente"}}</label>
          <select name="agente">
            {{range $agentes}}{{if ne .Rol "admin"}}<option value="{{.Nombre}}">{{.Nombre}}</option>{{end}}{{end}}
          </select>
        </div>
        <div><label>{{tr "projects.position"}}</label>
          <select name="posicion">
            <option value="acuerdo">{{tr "acuerdo"}}</option>
            <option value="desacuerdo">{{tr "desacuerdo"}}</option>
            <option value="abstencion">{{tr "abstencion"}}</option>
          </select>
        </div>
      </div>
      <div style="margin-top:.4rem"><label>{{tr "proposals.comment_optional"}}</label>
        <input type="text" name="comentario" placeholder="{{tr "proposals.vote_placeholder"}}" style="width:100%">
      </div>
      <button type="submit" class="btn-sm" style="margin-top:.5rem">{{tr "proposals.vote_submit"}}</button>
    </form>
  </div>
  <div class="action-box">
    <h4>{{tr "proposals.close_title"}}</h4>
    <form method="POST" action="/propuestas/{{$codigo}}/accion">
      <input type="hidden" name="accion" value="cerrar">
      <div><label>{{tr "proposals.close_state"}}</label>
        <select name="estado_cierre">
          <option value="consenso">{{tr "proposals.close_consensus"}}</option>
          <option value="rechazada">{{tr "proposals.close_rejected"}}</option>
          <option value="backlog">{{tr "proposals.close_backlog"}}</option>
        </select>
      </div>
      <button type="submit" class="btn-sm" style="margin-top:.5rem">{{tr "proposals.close_submit"}}</button>
    </form>
  </div>
</div>
{{end}}

<div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;margin-bottom:1rem">
  {{if neStr $estado "abierta"}}
  <div class="action-box">
    <h4>{{tr "proposals.reopen_title"}}</h4>
    <form method="POST" action="/propuestas/{{$codigo}}/accion">
      <input type="hidden" name="accion" value="reabrir">
      <p style="margin:.2rem 0 .6rem 0;color:#64748b;font-size:.9rem">
        {{tr "proposals.reopen_help"}}
      </p>
      <button type="submit" class="btn-sm">{{tr "proposals.reopen_submit"}}</button>
    </form>
  </div>
  {{end}}
  <div class="action-box">
    <h4>{{tr "proposals.repair_title"}}</h4>
    <form method="POST" action="/propuestas/{{$codigo}}/accion">
      <input type="hidden" name="accion" value="reparar-votos">
      <p style="margin:.2rem 0 .6rem 0;color:#64748b;font-size:.9rem">
        {{tr "proposals.repair_help"}}
      </p>
      <button type="submit" class="btn-sm">{{tr "proposals.repair_submit"}}</button>
    </form>
  </div>
</div>

<h4>{{tr "proposals.recorded_votes"}}</h4>
{{if .P.Votos}}
<table><thead><tr><th>{{tr "Agente"}}</th><th>{{tr "projects.position"}}</th><th>{{tr "projects.comment"}}</th></tr></thead><tbody>
{{range .P.Votos}}
  <tr>
    <td><strong>{{.Agente}}</strong></td>
    <td><span class="tag t-{{.Posicion}}">{{tr .Posicion}}</span></td>
    <td style="color:#64748b">{{.Comentario}}</td>
  </tr>
{{end}}
</tbody></table>
{{else}}
<p style="color:#94a3b8">{{tr "proposals.no_votes"}}</p>
{{end}}
{{end}}
`
