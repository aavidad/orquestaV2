package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/internal/controlruntime"
)

// StatusService encapsulates the data needed by /api/status.
type StatusService interface {
	FetchStatus() (apiStatusResponse, error)
}

var statusService StatusService = dbStatusService{}

var statusWorkspaceRiskSummaryBypassDepth atomic.Int32

var (
	statusSnapshotTTL                   = time.Minute
	statusFallbackTTL                   = 5 * time.Second
	statusSoftInvalidateTTL             = 5 * time.Second
	statusFailureBackoffTTL             = 2 * time.Second
	statusFreshTimeout                  = 1500 * time.Millisecond
	statusFastTimeout                   = 750 * time.Millisecond
	statusRefreshWaitTimeout            = 150 * time.Millisecond
	statusRowsTimeout                   = 400 * time.Millisecond
	statusOptionalSectionTimeout        = 300 * time.Millisecond
	statusFreshFetcher                  = fetchStatusFresh
	statusFastFetcher                   = fetchStatusFastFallback
	statusRowsFetcher                   = agentRowsForStatus
	statusRuntimeHandlesFetcher         = db.ListarRuntimeHandles
	statusListAgentsFetcher             = db.ListarAgentesEstadoLigero
	statusListAgentsWithSessionsFetcher = db.ListarAgentesEstadoLigeroConSesionesActivas
	statusVisibleSessionsFetcher        = sesionesVisiblesParaEstado
	statusListAutonomyFetcher           = func() ([]*db.ProyectoAutonomia, error) {
		enabled := true
		return db.ListarProyectosAutonomia(&enabled)
	}
	statusCountTasksFetcher   = db.ContarTareasPorEstado
	statusListProjectsFetcher = func() ([]*db.Proyecto, error) {
		return db.ListarProyectosConRutaEfectiva(db.FiltroProyectos{}, "")
	}
	statusCountAssignmentsFetcher  = db.ContarAsignacionesActivasPorProyecto
	statusListOpenProposalsFetcher = func() ([]*db.Propuesta, error) {
		estadoAbierta := db.PropuestaAbierta
		return db.ListarPropuestas(&estadoAbierta, nil)
	}
	statusVotesSummaryFetcher       = db.ResumenVotosPorPropuestas
	statusPoolsFetcher              = listarPoolsLocalesCompartidosEstado
	statusDispatchDebtFetcher       = listarDeudaDispatchEstado
	statusHandoffsFetcher           = listarHandoffsAutonomiaEstado
	statusDispatchSummaryFetcher    = listarResumenDispatchYHandoffsEstado
	statusIncludeProposalSections   = true
	statusDispatchFailureWindow     = 2 * time.Hour
	statusAutonomyEventsWindow      = 24 * time.Hour
	statusAutonomyEventsFetchLimit  = 50
	statusAutonomyEventsRecentLimit = 8
	statusAutonomySurfaceFetcher    = buildStatusAutonomySurfaceLocal
	statusAutonomySurfaceCacheTTL   = 10 * time.Second
	statusAutonomyEventsFetcher     = func(since time.Time, limit int) ([]autonomyEventSummary, error) {
		items, err := db.ListarAutonomyEvents(db.FiltroAutonomyEvents{
			Desde:  &since,
			Limite: limit,
		})
		if err != nil {
			return nil, err
		}
		return projectAutonomyEventSummariesFromEvents(items), nil
	}
	statusConfigGet    = db.ConfigGet
	statusNowFunc      = time.Now
	statusAsyncRefresh = true
	statusCacheState   struct {
		mu         sync.Mutex
		value      apiStatusResponse
		expires    time.Time
		ok         bool
		hardStale  bool
		retryAfter time.Time
		refreshing bool
		waitCh     chan struct{}
	}
	statusRowsFlightState struct {
		mu      sync.Mutex
		running bool
		waitCh  chan struct{}
		rows    []agentesapp.Row
		err     error
	}
	statusAutonomySurfaceState struct {
		mu         sync.Mutex
		value      *autonomySurface
		expires    time.Time
		refreshing bool
		waitCh     chan struct{}
	}
)

func statusRowsForSnapshot(timeout time.Duration) ([]agentesapp.Row, error) {
	if rows, ok := readAgentPanelSnapshotFresh(); ok {
		return rows, nil
	}
	if status, ok := readStatusSnapshotAny(); ok && statusSnapshotCanStayLight(status) {
		return buildAgentPanelRowsFromStatusSnapshot(status), nil
	}
	rows, err := agentRowsForStatusWithinTimeout(timeout)
	if err != nil {
		return nil, err
	}
	storeAgentPanelSnapshot(rows, statusNowFunc().UTC())
	return rows, nil
}

type dbStatusService struct{}

func statusVisibleWorkerCounters(agentesActivos, agentesTrabajando []*db.Agente, autonomia autonomiaResumen) (int, int, int) {
	workersActivos := visibleWorkerCount(len(agentesActivos), autonomia.Supervisando)
	workersTrabajando := visibleWorkerCount(len(agentesTrabajando), autonomia.Supervisando)
	supervisorNames := autonomia.supervisorNames
	if len(supervisorNames) == 0 {
		names := statusSupervisorAgentNames()
		if len(names) > 0 {
			supervisorNames = make(map[string]struct{}, len(names))
			for _, nombre := range names {
				canon := nombreAgenteCanonico(nombre)
				if canon == "" {
					continue
				}
				supervisorNames[canon] = struct{}{}
			}
		}
	}
	if len(supervisorNames) > 0 {
		autonomia.supervisorNames = supervisorNames
		workersActivos = len(filterVisibleWorkersByAutonomy(agentesActivos, autonomia))
		workersTrabajando = len(filterVisibleWorkersByAutonomy(agentesTrabajando, autonomia))
	}
	supervisoresActivos := autonomia.Supervisando
	if supervisoresActivos < 0 {
		supervisoresActivos = 0
	}
	return workersActivos, workersTrabajando, supervisoresActivos
}

func reconciledVisibleWorkerCounters(workersConectados, workersTrabajando, supervisoresActivos int, agentesActivos, agentesTrabajando []*db.Agente, autonomia autonomiaResumen) (int, int, int) {
	recomputedConectados, recomputedTrabajando, recomputedSupervisores := statusVisibleWorkerCounters(agentesActivos, agentesTrabajando, autonomia)
	if workersConectados > 0 || workersTrabajando > 0 {
		if supervisoresActivos == 0 && recomputedSupervisores > 0 {
			supervisoresActivos = recomputedSupervisores
		}
		return workersConectados, workersTrabajando, supervisoresActivos
	}
	if recomputedConectados > 0 || recomputedTrabajando > 0 || supervisoresActivos == 0 {
		return recomputedConectados, recomputedTrabajando, recomputedSupervisores
	}
	return workersConectados, workersTrabajando, supervisoresActivos
}

func filterVisibleWorkersByAutonomy(items []*db.Agente, autonomia autonomiaResumen) []*db.Agente {
	if len(items) == 0 || len(autonomia.supervisorNames) == 0 {
		return items
	}
	out := make([]*db.Agente, 0, len(items))
	for _, agente := range items {
		if agente == nil {
			continue
		}
		if _, ok := autonomia.supervisorNames[nombreAgenteCanonico(agente.Nombre)]; ok {
			continue
		}
		out = append(out, agente)
	}
	return out
}

var errStatusFetchTimeout = errors.New("status fetch timeout")

type deudaDispatchResumen struct {
	Total         int `json:"total"`
	Pendientes    int `json:"pending"`
	Notificadas   int `json:"notified"`
	Fallidas      int `json:"failed"`
	WorkConfirmed int `json:"work_confirmed"`
}

type autonomiaResumen struct {
	Supervisando         int                    `json:"supervising"`
	Continuando          int                    `json:"continuing"`
	ContinuidadPendiente int                    `json:"continuity_pending"`
	WorkConfirmed        int                    `json:"work_confirmed"`
	Handoffs             int                    `json:"handoffing"`
	Count                int                    `json:"count"`
	ByKind               map[string]int         `json:"by_kind"`
	LastAt               *time.Time             `json:"last_at,omitempty"`
	Recent               []autonomyEventSummary `json:"recent,omitempty"`
	supervisorNames      map[string]struct{}
}

type statusWorkspaceRiskSummary struct {
	CriticalProjectRisk *workspaceAutonomyProjectSummary
	Highlights          []string
}

func (a *autonomiaResumen) addSupervisorName(nombre string) {
	if a == nil {
		return
	}
	nombre = nombreAgenteCanonico(nombre)
	if nombre == "" {
		return
	}
	if a.supervisorNames == nil {
		a.supervisorNames = make(map[string]struct{}, 2)
	}
	a.supervisorNames[nombre] = struct{}{}
}

type statusDispatchSummary struct {
	Deuda    deudaDispatchResumen
	Handoffs int
}

func int64FromStatusAny(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		if err == nil {
			return n
		}
	}
	return 0
}

func normalizeDispatchDebtTotal(out deudaDispatchResumen) deudaDispatchResumen {
	out.Total = out.Pendientes + out.Notificadas + out.Fallidas + out.WorkConfirmed
	return out
}

func listarDeudaDispatchEstado() (deudaDispatchResumen, error) {
	out, err := listarResumenDispatchYHandoffsEstado()
	if err != nil {
		return deudaDispatchResumen{}, err
	}
	return out.Deuda, nil
}

func listarResumenDispatchYHandoffsEstado() (statusDispatchSummary, error) {
	out := statusDispatchSummary{}
	var allHandles []*db.RuntimeHandle
	var handlesLoaded bool
	now := statusNowFunc().UTC()
	seenHandoffs := map[int64]struct{}{}
	estados := []string{"pendiente", "ejecutando", "fallida"}
	for _, estado := range estados {
		estadoFiltro := estado
		orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{
			Estado: &estadoFiltro,
			Tipos:  []string{"send_instruction", "handoff"},
			Limit:  500,
		})
		if err != nil {
			return out, err
		}
		for _, order := range orders {
			if order == nil {
				continue
			}
			switch strings.TrimSpace(order.Tipo) {
			case "handoff":
				if estadoFiltro == "fallida" {
					continue
				}
				if _, ok := seenHandoffs[order.ID]; ok {
					continue
				}
				seenHandoffs[order.ID] = struct{}{}
				out.Handoffs++
				continue
			case "send_instruction":
			default:
				continue
			}
			payload := mapFromJSON(order.PayloadJSON)
			if int64FromStatusAny(payload["mailbox_id"]) <= 0 {
				continue
			}
			if absorbed, _, err := db.RuntimeOrderSendInstructionAbsorbidaPorTrabajoVivo(order, now); err != nil {
				return out, err
			} else if absorbed {
				out.Deuda.WorkConfirmed++
				continue
			}
			result := mapFromJSON(order.ResultadoJSON)
			dispatchState := ""
			deliveryState := ""
			if raw, ok := result["dispatch_state"].(string); ok {
				dispatchState = strings.ToLower(strings.TrimSpace(raw))
			}
			if raw, ok := result["delivery_state"].(string); ok {
				deliveryState = strings.ToLower(strings.TrimSpace(raw))
			}
			receiptSource := ""
			if raw, ok := result["receipt_source"].(string); ok {
				receiptSource = strings.ToLower(strings.TrimSpace(raw))
			}
			switch {
			case dispatchState == "delivered" && deliveryState == "delivered" && receiptSourceConfirmsWorkStatus(receiptSource):
				out.Deuda.WorkConfirmed++
			case dispatchState == "delivered" && deliveryState == "delivered":
				if !handlesLoaded {
					handlesLoaded = true
					handles, err := statusRuntimeHandlesFetcher(nil)
					if err != nil {
						return out, err
					}
					allHandles = handles
				}
				if dispatchOrderWorkConfirmedFromHandles(order, allHandles) {
					out.Deuda.WorkConfirmed++
				} else {
					out.Deuda.Pendientes++
				}
			case strings.TrimSpace(order.Estado) == "fallida" || dispatchState == "failed" || deliveryState == "failed":
				if dispatchOrderFailureCountsAsDebt(order, now) {
					out.Deuda.Fallidas++
				}
			case dispatchState == "notified" || deliveryState == "notified":
				out.Deuda.Notificadas++
			default:
				out.Deuda.Pendientes++
			}
		}
	}
	out.Deuda = normalizeDispatchDebtTotal(out.Deuda)
	return out, nil
}

func dispatchOrderFailureCountsAsDebt(order *db.RuntimeOrder, now time.Time) bool {
	if order == nil {
		return false
	}
	window := statusDispatchFailureWindow
	if window <= 0 {
		return true
	}
	last := order.UpdatedAt.UTC()
	if order.FinishedAt != nil && !order.FinishedAt.IsZero() && order.FinishedAt.UTC().After(last) {
		last = order.FinishedAt.UTC()
	}
	if last.IsZero() {
		last = order.CreatedAt.UTC()
	}
	if last.IsZero() {
		return true
	}
	return !last.Before(now.Add(-window))
}

func dispatchOrderWorkConfirmedFromHandles(order *db.RuntimeOrder, handles []*db.RuntimeHandle) bool {
	if order == nil {
		return false
	}
	payload := mapFromJSON(order.PayloadJSON)
	result := mapFromJSON(order.ResultadoJSON)
	verificationKey := strings.TrimSpace(stringMapValue(result, "verification_key"))
	if verificationKey == "" {
		verificationKey = strings.TrimSpace(stringMapValue(payload, "verification_key"))
	}
	if verificationKey == "" {
		return false
	}
	match := controlruntime.WorkQueueMatch{
		Kind:            "autonomia",
		Action:          "continuar_trabajo",
		VerificationKey: verificationKey,
		Within:          2 * time.Hour,
	}
	for _, handle := range handles {
		if handle == nil || handle.ProyectoID == nil || order.ProyectoID == nil || *handle.ProyectoID != *order.ProyectoID {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(handle.Agente), strings.TrimSpace(order.Agente)) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(handle.Estado), "activo") {
			continue
		}
		if !db.RuntimeHandleSnapshotIsFresh(handle, 2*time.Minute) {
			continue
		}
		started, err := controlruntime.HasStartedWorkQueueEntryFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON), match)
		if err == nil && started {
			return true
		}
	}
	return false
}

func receiptSourceConfirmsWorkStatus(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "last_progress",
		"worker_activity",
		"tmux_transcript_activity",
		"transcript_patch",
		"tmux_pane_patch",
		"git_worktree":
		return true
	default:
		return false
	}
}

func resumirAutonomiaRows(rows []agentesapp.Row, now time.Time) autonomiaResumen {
	out := autonomiaResumen{}
	for _, row := range rows {
		if !rowCuentaComoAutonomiaActiva(row) && !row.SupervisorRoleActive(now) {
			continue
		}
		workConfirmed := strings.EqualFold(strings.TrimSpace(row.LastAutonomyState), "work_confirmed")
		if strings.EqualFold(strings.TrimSpace(row.LastAutonomySource), "assignment_handoff") &&
			!workConfirmed &&
			(row.OpenTasks > 0 || row.SupervisorRoleActive(now)) &&
			row.WorkerFresh(now) {
			out.Handoffs++
		}
		switch {
		case row.SupervisorRoleActive(now):
			out.Supervisando++
			if row.Agente != nil {
				out.addSupervisorName(row.Agente.Nombre)
			}
		case !workConfirmed && strings.TrimSpace(row.LastAutonomyAction) == "continuar_trabajo" && row.OpenTasks > 0:
			out.Continuando++
		}
		if workConfirmed {
			out.WorkConfirmed++
		}
		if row.EffectiveContinuityPending(now) {
			out.ContinuidadPendiente++
		}
	}
	return out
}

func resumirAutonomiaLigera(agentesActivos, agentesTrabajando []*db.Agente, tareasEnProgreso []tareaLite, now time.Time) autonomiaResumen {
	return resumirAutonomiaLigeraConSupervisor(agentesActivos, agentesTrabajando, tareasEnProgreso, now, true)
}

func resumirAutonomiaLigeraConSupervisor(agentesActivos, agentesTrabajando []*db.Agente, tareasEnProgreso []tareaLite, now time.Time, includeSupervisorConfig bool) autonomiaResumen {
	out := autonomiaResumen{}
	supervisores := map[string]struct{}{}
	tareasPorAgente := map[string]struct{}{}
	if includeSupervisorConfig {
		for _, nombre := range statusSupervisorAgentNames() {
			if nombre = nombreAgenteCanonico(nombre); nombre != "" {
				supervisores[nombre] = struct{}{}
			}
		}
	}
	activos := map[string]*db.Agente{}
	trabajando := map[string]*db.Agente{}
	for _, agente := range agentesActivos {
		if agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" {
			continue
		}
		activos[nombre] = agente
	}
	for _, agente := range agentesTrabajando {
		if agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" {
			continue
		}
		if activo := activos[nombre]; activo != nil {
			trabajando[nombre] = activo
		}
	}
	for _, tarea := range tareasEnProgreso {
		nombre := nombreAgenteCanonico(tarea.Agente)
		if nombre == "" {
			continue
		}
		tareasPorAgente[nombre] = struct{}{}
		if activo := activos[nombre]; activo != nil {
			trabajando[nombre] = activo
		}
	}
	for supervisor := range supervisores {
		if _, ok := activos[supervisor]; ok {
			if _, ocupadoComoWorker := tareasPorAgente[supervisor]; ocupadoComoWorker {
				continue
			}
			out.Supervisando++
			out.addSupervisorName(supervisor)
		}
	}
	for nombre := range trabajando {
		if _, ok := supervisores[nombre]; ok {
			if _, ocupadoComoWorker := tareasPorAgente[nombre]; !ocupadoComoWorker {
				continue
			}
		}
		out.WorkConfirmed++
	}
	for supervisor := range supervisores {
		if _, ocupadoComoWorker := tareasPorAgente[supervisor]; ocupadoComoWorker {
			continue
		}
		if _, ok := trabajando[supervisor]; ok {
			out.WorkConfirmed++
		}
	}
	return out
}

func agentesVisiblesLigero(agentes []*db.Agente, tareasEnProgreso []tareaLite) ([]*db.Agente, []*db.Agente, []*db.Agente) {
	snapshotConHabilitado := snapshotExponeHabilitado(agentes)
	agentesPorNombre := make(map[string]*db.Agente, len(agentes))
	activos := make([]*db.Agente, 0, len(agentes))
	trabajando := make([]*db.Agente, 0, len(agentes))
	trabajandoSet := make(map[string]struct{}, len(tareasEnProgreso))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" || !agenteVisibleEnStatusFleet(nombre) {
			continue
		}
		agentesPorNombre[nombre] = agente
		if !agenteCuentaComoHabilitadoEnSnapshot(agente, snapshotConHabilitado) || agenteBloqueadoPorCuotaVisible(agente) {
			continue
		}
		if agenteCuentaComoConectado(agente) {
			activos = append(activos, agente)
		}
	}
	for _, tarea := range tareasEnProgreso {
		nombre := nombreAgenteCanonico(tarea.Agente)
		if nombre == "" {
			continue
		}
		if _, ok := trabajandoSet[nombre]; ok {
			continue
		}
		agente := agentesPorNombre[nombre]
		if agente == nil || !agenteCuentaComoConectado(agente) || agenteBloqueadoPorCuotaVisible(agente) {
			continue
		}
		trabajando = append(trabajando, agente)
		trabajandoSet[nombre] = struct{}{}
	}
	return activos, trabajando, agentesNoActivosConCuotaConResumen(agentes, nil)
}

func normalizarAgentesVisiblesStatus(activos, trabajando, saturados []*db.Agente) ([]*db.Agente, []*db.Agente, []*db.Agente) {
	activosCanonicos := make([]*db.Agente, 0, len(activos))
	activosPorNombre := make(map[string]*db.Agente, len(activos))
	for _, agente := range activos {
		if agente == nil || !agente.Activo {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" {
			continue
		}
		if actual := activosPorNombre[nombre]; actual != nil {
			activosPorNombre[nombre] = preferAgenteStatusCanonico(actual, agente)
			continue
		}
		activosPorNombre[nombre] = agente
		activosCanonicos = append(activosCanonicos, agente)
	}
	trabajandoCanonicos := make([]*db.Agente, 0, len(trabajando))
	seenWorking := make(map[string]struct{}, len(trabajando))
	for _, agente := range trabajando {
		if agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" {
			continue
		}
		activo := activosPorNombre[nombre]
		if activo == nil {
			continue
		}
		if _, ok := seenWorking[nombre]; ok {
			continue
		}
		seenWorking[nombre] = struct{}{}
		trabajandoCanonicos = append(trabajandoCanonicos, activo)
	}
	saturadosCanonicos := make([]*db.Agente, 0, len(saturados))
	seenSaturated := make(map[string]struct{}, len(saturados))
	for _, agente := range saturados {
		if agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" {
			continue
		}
		activo := activosPorNombre[nombre]
		if activo == nil {
			continue
		}
		if _, ok := seenSaturated[nombre]; ok {
			continue
		}
		seenSaturated[nombre] = struct{}{}
		saturadosCanonicos = append(saturadosCanonicos, activo)
	}
	return activosCanonicos, trabajandoCanonicos, saturadosCanonicos
}

func rowCuentaComoAutonomiaActiva(row agentesapp.Row) bool {
	if row.Agente != nil && agenteBloqueadoPorCuotaVisible(row.Agente) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(row.EstadoOperativo)) {
	case "bloqueado_por_cuota",
		"bloqueado_por_runtime",
		"atascado",
		"mailbox_atascada",
		"retirado",
		"desconocido":
		return false
	default:
		return true
	}
}

func listarHandoffsAutonomiaEstado() (int, error) {
	out, err := listarResumenDispatchYHandoffsEstado()
	if err != nil {
		return 0, err
	}
	return out.Handoffs, nil
}

func buildStatusWorkspaceRiskSummary(surface *autonomySurface) statusWorkspaceRiskSummary {
	if surface == nil {
		return statusWorkspaceRiskSummary{}
	}
	out := statusWorkspaceRiskSummary{
		Highlights: append([]string(nil), surface.Highlights...),
	}
	if top := statusCriticalProjectRiskFromSurface(surface); top != nil {
		out.CriticalProjectRisk = top
	}
	return out
}

func fetchStatusWorkspaceRiskSummary() (statusWorkspaceRiskSummary, error) {
	if statusWorkspaceRiskSummaryBypassDepth.Load() > 0 {
		return statusWorkspaceRiskSummary{}, nil
	}
	if workspaceControlListProjects == nil || workspaceControlCockpitBuilder == nil {
		return statusWorkspaceRiskSummary{}, nil
	}
	projectItems, err := workspaceControlListProjects()
	if err != nil {
		return statusWorkspaceRiskSummary{}, err
	}
	cockpits := make([]*apiProyectoCockpit, 0, len(projectItems))
	for _, item := range projectItems {
		project := strings.TrimSpace(fmt.Sprint(item["slug"]))
		if project == "" {
			continue
		}
		cockpit, err := workspaceControlCockpitBuilder(project)
		if err != nil || cockpit == nil {
			continue
		}
		cockpits = append(cockpits, cockpit)
	}
	surface := buildAutonomySurfaceFromCockpits(cockpits, workspaceAutonomyRecentLimit)
	report := buildStatusWorkspaceRiskSummary(surface)
	projects := buildWorkspaceAutonomyProjects(cockpits, surface, workspaceAutonomyProjectLimit)
	if blocking := workspaceBlockingProjectsCount(projects); blocking > 0 {
		report.Highlights = appendWorkspaceHighlight(report.Highlights, fmt.Sprintf("frentes_bloqueantes=%d", blocking))
	}
	if topRisk, ok := workspaceTopRiskProject(projects); ok {
		topRiskCopy := topRisk
		report.CriticalProjectRisk = &topRiskCopy
		report.Highlights = appendWorkspaceHighlight(report.Highlights, fmt.Sprintf("integracion_bloqueada=%d", topRisk.Blocking))
		report.Highlights = appendWorkspaceHighlight(report.Highlights, fmt.Sprintf("riesgo_top=%s(%d)", topRisk.Project, topRisk.Blocking))
	}
	return report, nil
}

func runWithoutStatusWorkspaceRiskSummary[T any](fn func() (T, error)) (T, error) {
	var zero T
	if fn == nil {
		return zero, errors.New("status workspace risk fn nil")
	}
	statusWorkspaceRiskSummaryBypassDepth.Add(1)
	defer statusWorkspaceRiskSummaryBypassDepth.Add(-1)
	return fn()
}

func statusCriticalProjectRiskFromSurface(surface *autonomySurface) *workspaceAutonomyProjectSummary {
	if surface == nil || len(surface.Projects) == 0 {
		return nil
	}
	bestIndex := -1
	for i := range surface.Projects {
		project := strings.TrimSpace(surface.Projects[i].Project)
		if project == "" {
			continue
		}
		if bestIndex < 0 {
			bestIndex = i
			continue
		}
		current := surface.Projects[i]
		best := surface.Projects[bestIndex]
		switch {
		case current.LastAt != nil && best.LastAt == nil:
			bestIndex = i
		case current.LastAt != nil && best.LastAt != nil && current.LastAt.After(*best.LastAt):
			bestIndex = i
		case current.LastAt != nil && best.LastAt != nil && current.LastAt.Equal(*best.LastAt) && current.Events > best.Events:
			bestIndex = i
		case current.LastAt == nil && best.LastAt == nil && current.Events > best.Events:
			bestIndex = i
		case current.LastAt == nil && best.LastAt == nil && current.Events == best.Events &&
			strings.TrimSpace(current.Project) < strings.TrimSpace(best.Project):
			bestIndex = i
		}
	}
	if bestIndex < 0 {
		return nil
	}
	best := surface.Projects[bestIndex]
	project := strings.TrimSpace(best.Project)
	if project == "" {
		return nil
	}
	item := &workspaceAutonomyProjectSummary{
		Project:    project,
		Events:     best.Events,
		LastAt:     best.LastAt,
		Blocking:   statusBlockingFromHighlights(best.Highlights),
		Highlights: append([]string(nil), best.Highlights...),
	}
	return item
}

func statusBlockingFromHighlights(highlights []string) int {
	for _, item := range highlights {
		item = strings.TrimSpace(item)
		if !strings.HasPrefix(item, "integracion_bloqueada=") {
			continue
		}
		value := strings.TrimPrefix(item, "integracion_bloqueada=")
		if n, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

func enrichStatusAutonomySurfaceWithRisk(surface *autonomySurface, risk statusWorkspaceRiskSummary) *autonomySurface {
	if surface == nil && len(risk.Highlights) == 0 && risk.CriticalProjectRisk == nil {
		return nil
	}
	if surface == nil {
		surface = &autonomySurface{}
	}
	surface.Highlights = mergeStatusRiskHighlights(surface.Highlights, risk.Highlights)
	if risk.CriticalProjectRisk == nil {
		return surface
	}
	surface.Highlights = mergeStatusRiskHighlights(surface.Highlights, []string{
		fmt.Sprintf("integracion_bloqueada=%d", risk.CriticalProjectRisk.Blocking),
		fmt.Sprintf("riesgo_top=%s(%d)", strings.TrimSpace(risk.CriticalProjectRisk.Project), risk.CriticalProjectRisk.Blocking),
	})
	project := strings.TrimSpace(risk.CriticalProjectRisk.Project)
	if project == "" {
		return surface
	}
	for i := range surface.Projects {
		if !strings.EqualFold(strings.TrimSpace(surface.Projects[i].Project), project) {
			continue
		}
		surface.Projects[i].Highlights = mergeStatusRiskHighlights(surface.Projects[i].Highlights, risk.CriticalProjectRisk.Highlights)
		return surface
	}
	surface.Projects = append(surface.Projects, autonomyProjectSurface{
		Project:    project,
		Events:     risk.CriticalProjectRisk.Events,
		LastAt:     risk.CriticalProjectRisk.LastAt,
		Highlights: append([]string(nil), risk.CriticalProjectRisk.Highlights...),
	})
	sort.SliceStable(surface.Projects, func(i, j int) bool {
		if strings.EqualFold(strings.TrimSpace(surface.Projects[i].Project), project) {
			return true
		}
		if strings.EqualFold(strings.TrimSpace(surface.Projects[j].Project), project) {
			return false
		}
		return strings.TrimSpace(surface.Projects[i].Project) < strings.TrimSpace(surface.Projects[j].Project)
	})
	return surface
}

func statusAutonomyHighlights(surface *autonomySurface, fallback []string) []string {
	if surface == nil {
		if len(fallback) == 0 {
			return nil
		}
		return append([]string(nil), fallback...)
	}
	return mergeStatusRiskHighlights(surface.Highlights, fallback)
}

func cloneStatusCriticalProjectRisk(item *workspaceAutonomyProjectSummary) *workspaceAutonomyProjectSummary {
	if item == nil {
		return nil
	}
	out := *item
	out.Highlights = append([]string(nil), item.Highlights...)
	return &out
}

func mergeStatusWorkspaceRiskSummary(base, extra statusWorkspaceRiskSummary) statusWorkspaceRiskSummary {
	out := statusWorkspaceRiskSummary{
		CriticalProjectRisk: cloneStatusCriticalProjectRisk(base.CriticalProjectRisk),
		Highlights:          append([]string(nil), base.Highlights...),
	}
	if extra.CriticalProjectRisk != nil {
		out.CriticalProjectRisk = cloneStatusCriticalProjectRisk(extra.CriticalProjectRisk)
	}
	out.Highlights = mergeStatusRiskHighlights(out.Highlights, extra.Highlights)
	return out
}

func canonicalizeStatusAutonomyRisk(surface *autonomySurface, risk statusWorkspaceRiskSummary) (*autonomySurface, statusWorkspaceRiskSummary) {
	risk = mergeStatusWorkspaceRiskSummary(buildStatusWorkspaceRiskSummary(surface), risk)
	surface = enrichStatusAutonomySurfaceWithRisk(surface, risk)
	if risk.CriticalProjectRisk == nil {
		risk.CriticalProjectRisk = statusCriticalProjectRiskFromSurface(surface)
	}
	risk.Highlights = statusAutonomyHighlights(surface, risk.Highlights)
	return surface, risk
}

func cloneStatusAutonomySurface(surface *autonomySurface) *autonomySurface {
	if surface == nil {
		return nil
	}
	out := &autonomySurface{
		Events:     surface.Events,
		ByKind:     make(map[string]int, len(surface.ByKind)),
		Highlights: append([]string(nil), surface.Highlights...),
		Recent:     make([]autonomySurfaceRecentItem, 0, len(surface.Recent)),
		Projects:   make([]autonomyProjectSurface, 0, len(surface.Projects)),
	}
	if surface.LastAt != nil {
		last := *surface.LastAt
		out.LastAt = &last
	}
	for key, value := range surface.ByKind {
		out.ByKind[key] = value
	}
	for _, item := range surface.Recent {
		out.Recent = append(out.Recent, item)
	}
	for _, item := range surface.Projects {
		project := autonomyProjectSurface{
			Project:    item.Project,
			Events:     item.Events,
			ByKind:     make(map[string]int, len(item.ByKind)),
			Highlights: append([]string(nil), item.Highlights...),
			Recent:     append([]autonomyEventSummary(nil), item.Recent...),
		}
		if item.LastAt != nil {
			last := *item.LastAt
			project.LastAt = &last
		}
		for key, value := range item.ByKind {
			project.ByKind[key] = value
		}
		out.Projects = append(out.Projects, project)
	}
	return out
}

func fetchStatusAutonomySurfaceCached() (*autonomySurface, error) {
	if statusAutonomySurfaceFetcher == nil {
		return nil, nil
	}
	if statusAutonomySurfaceCacheTTL <= 0 {
		return statusAutonomySurfaceFetcher()
	}
	now := statusNowFunc().UTC()
	statusAutonomySurfaceState.mu.Lock()
	if cached := statusAutonomySurfaceState.value; cached != nil && now.Before(statusAutonomySurfaceState.expires) {
		out := cloneStatusAutonomySurface(cached)
		statusAutonomySurfaceState.mu.Unlock()
		return out, nil
	}
	if statusAutonomySurfaceState.refreshing && statusAutonomySurfaceState.waitCh != nil {
		waitCh := statusAutonomySurfaceState.waitCh
		cached := cloneStatusAutonomySurface(statusAutonomySurfaceState.value)
		statusAutonomySurfaceState.mu.Unlock()
		if cached != nil {
			return cached, nil
		}
		<-waitCh
		statusAutonomySurfaceState.mu.Lock()
		out := cloneStatusAutonomySurface(statusAutonomySurfaceState.value)
		statusAutonomySurfaceState.mu.Unlock()
		return out, nil
	}
	waitCh := make(chan struct{})
	statusAutonomySurfaceState.refreshing = true
	statusAutonomySurfaceState.waitCh = waitCh
	statusAutonomySurfaceState.mu.Unlock()

	surface, err := statusAutonomySurfaceFetcher()

	statusAutonomySurfaceState.mu.Lock()
	if err == nil {
		statusAutonomySurfaceState.value = cloneStatusAutonomySurface(surface)
		statusAutonomySurfaceState.expires = now.Add(statusAutonomySurfaceCacheTTL)
	}
	statusAutonomySurfaceState.refreshing = false
	if statusAutonomySurfaceState.waitCh == waitCh {
		statusAutonomySurfaceState.waitCh = nil
	}
	close(waitCh)
	out := cloneStatusAutonomySurface(statusAutonomySurfaceState.value)
	statusAutonomySurfaceState.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func loadStatusAutonomySurfaceAndRisk() (*autonomySurface, statusWorkspaceRiskSummary) {
	var surface *autonomySurface
	if out, ok := runStatusOptional(statusOptionalSectionTimeout, func() (*autonomySurface, error) {
		return fetchStatusAutonomySurfaceCached()
	}); ok {
		surface = out
	}
	riskSummary := buildStatusWorkspaceRiskSummary(surface)
	if out, ok := runStatusOptional(statusOptionalSectionTimeout, fetchStatusWorkspaceRiskSummary); ok {
		riskSummary = mergeStatusWorkspaceRiskSummary(riskSummary, out)
	}
	return canonicalizeStatusAutonomyRisk(surface, riskSummary)
}

func normalizeStatusAutonomyPayload(surface *autonomySurface, highlights []string, criticalProjectRisk *workspaceAutonomyProjectSummary) (*autonomySurface, []string, *workspaceAutonomyProjectSummary) {
	riskSummary := statusWorkspaceRiskSummary{
		CriticalProjectRisk: cloneStatusCriticalProjectRisk(criticalProjectRisk),
		Highlights:          append([]string(nil), highlights...),
	}
	surface, riskSummary = canonicalizeStatusAutonomyRisk(surface, riskSummary)
	return surface, riskSummary.Highlights, riskSummary.CriticalProjectRisk
}

func resumirAutonomiaEventosRecientes(base autonomiaResumen, now time.Time) (autonomiaResumen, error) {
	base.ByKind = map[string]int{}
	if statusAutonomyEventsFetcher == nil {
		return base, nil
	}
	if statusAutonomyEventsWindow <= 0 {
		return base, nil
	}
	limit := statusAutonomyEventsFetchLimit
	if limit <= 0 {
		limit = 50
	}
	items, err := statusAutonomyEventsFetcher(now.UTC().Add(-statusAutonomyEventsWindow), limit)
	if err != nil {
		return base, err
	}
	base.Recent, base.ByKind, base.LastAt, base.Count = compactAutonomyEventSummaries(items, statusAutonomyEventsRecentLimit)
	if base.ByKind == nil {
		base.ByKind = map[string]int{}
	}
	return base, nil
}

func buildStatusAutonomySurfaceLocal() (*autonomySurface, error) {
	if statusListProjectsFetcher == nil || statusAutonomyEventsWindow <= 0 {
		return nil, nil
	}
	proyectos, err := statusListProjectsFetcher()
	if err != nil {
		return nil, err
	}
	since := statusNowFunc().UTC().Add(-statusAutonomyEventsWindow)
	out := &autonomySurface{
		ByKind:   map[string]int{},
		Projects: make([]autonomyProjectSurface, 0, len(proyectos)),
	}
	recent := make([]autonomySurfaceRecentItem, 0)
	for _, proyecto := range proyectos {
		if proyecto == nil || !proyecto.Activo {
			continue
		}
		project := strings.TrimSpace(proyecto.Slug)
		if project == "" {
			continue
		}
		items, err := buildProjectAutonomyEventSummaries(proyecto.ID, since, statusAutonomyEventsFetchLimit)
		if err != nil || len(items) == 0 {
			continue
		}
		projectRecent, projectByKind, projectLastAt, total := compactAutonomyEventSummaries(items, statusAutonomyEventsRecentLimit)
		if total <= 0 {
			continue
		}
		for kind, count := range projectByKind {
			out.ByKind[kind] += count
		}
		out.Events += total
		out.LastAt = maxTimePtr(out.LastAt, projectLastAt)
		out.Projects = append(out.Projects, autonomyProjectSurface{
			Project:    project,
			Events:     total,
			ByKind:     projectByKind,
			LastAt:     projectLastAt,
			Recent:     projectRecent,
			Highlights: buildAutonomyHighlights(projectByKind, projectRecent, projectLastAt, 3),
		})
		for _, item := range projectRecent {
			recent = append(recent, autonomySurfaceRecentItem{
				Project:              project,
				autonomyEventSummary: item,
			})
		}
	}
	if out.Events == 0 {
		return nil, nil
	}
	sort.SliceStable(out.Projects, func(i, j int) bool {
		left := out.Projects[i]
		right := out.Projects[j]
		switch {
		case left.LastAt == nil && right.LastAt == nil:
		case left.LastAt == nil:
			return false
		case right.LastAt == nil:
			return true
		case !left.LastAt.Equal(*right.LastAt):
			return left.LastAt.After(*right.LastAt)
		}
		if left.Events != right.Events {
			return left.Events > right.Events
		}
		return left.Project < right.Project
	})
	sort.SliceStable(recent, func(i, j int) bool {
		if recent[i].CreatedAt.Equal(recent[j].CreatedAt) {
			if recent[i].Project == recent[j].Project {
				return recent[i].Kind < recent[j].Kind
			}
			return recent[i].Project < recent[j].Project
		}
		return recent[i].CreatedAt.After(recent[j].CreatedAt)
	})
	if statusAutonomyEventsRecentLimit > 0 && len(recent) > statusAutonomyEventsRecentLimit {
		recent = recent[:statusAutonomyEventsRecentLimit]
	}
	out.Recent = recent
	out.Highlights = buildAutonomyHighlights(out.ByKind, projectRecentFromSurfaceRecent(recent), out.LastAt, 4)
	return out, nil
}

func projectRecentFromSurfaceRecent(items []autonomySurfaceRecentItem) []autonomyEventSummary {
	if len(items) == 0 {
		return nil
	}
	out := make([]autonomyEventSummary, 0, len(items))
	for _, item := range items {
		out = append(out, item.autonomyEventSummary)
	}
	return out
}

func statusSupervisorAgentName() string {
	if statusConfigGet == nil {
		return "Codex1"
	}
	value, err := statusConfigGet("server_autobootstrap_supervisor_agent")
	if err != nil {
		return "Codex1"
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "Codex1"
	}
	return value
}

func statusSupervisorAgentNames() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	add := func(nombre string) {
		nombre = nombreAgenteCanonico(nombre)
		if nombre == "" {
			return
		}
		key := strings.ToLower(nombre)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, nombre)
	}
	add(statusSupervisorAgentName())
	if statusListAutonomyFetcher == nil {
		return out
	}
	policies, err := statusListAutonomyFetcher()
	if err != nil {
		return out
	}
	for _, policy := range policies {
		if policy == nil || !policy.Enabled || !policy.ReserveSupervisor {
			continue
		}
		add(strings.TrimSpace(policy.SupervisorAgente))
	}
	return out
}

func agenteTieneActividadRecienteVisible(agente *db.Agente) bool {
	if agente == nil || agente.UltimaSesion == nil || agente.UltimaSesion.IsZero() {
		return false
	}
	return agente.UltimaSesion.After(time.Now().UTC().Add(-5 * time.Minute))
}

func sesionesVisiblesParaEstado() ([]*db.Sesion, error) {
	return db.ListarSesionesActivasOperativas()
}

func nombresSesionesVisibles() (map[string]bool, error) {
	sesiones, err := sesionesVisiblesParaEstado()
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(sesiones))
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		nombre := strings.ToLower(strings.TrimSpace(sesion.Agente))
		if nombre != "" {
			out[nombre] = true
		}
	}
	return out, nil
}

func agenteCuentaComoConectado(agente *db.Agente) bool {
	if agente == nil {
		return false
	}
	if agenteBloqueadoPorCuotaVisible(agente) {
		return false
	}
	if agente.Activo {
		return true
	}
	estadoSesion := strings.TrimSpace(agente.EstadoSesion)
	if estadoSesion != "" && !strings.EqualFold(estadoSesion, "cerrada") && !strings.EqualFold(estadoSesion, "pausada") {
		return true
	}
	return false
}

func agentRowsForStatus() ([]agentesapp.Row, error) {
	return agentesService.BuildPanelRows()
}

func runStatusOptional[T any](timeout time.Duration, fn func() (T, error)) (T, bool) {
	var zero T
	if fn == nil {
		return zero, false
	}
	if timeout <= 0 {
		out, err := fn()
		return out, err == nil
	}
	type result struct {
		value T
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		value, err := fn()
		ch <- result{value: value, err: err}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			return zero, false
		}
		return res.value, true
	case <-time.After(timeout):
		return zero, false
	}
}

func agentRowsForStatusWithinTimeout(timeout time.Duration) ([]agentesapp.Row, error) {
	if statusRowsFetcher == nil {
		return nil, errors.New("status rows fetcher nil")
	}
	if timeout <= 0 {
		return statusRowsFetcher()
	}
	statusRowsFlightState.mu.Lock()
	ch := statusRowsFlightState.waitCh
	if !statusRowsFlightState.running {
		ch = make(chan struct{})
		statusRowsFlightState.running = true
		statusRowsFlightState.waitCh = ch
		go func(waitCh chan struct{}) {
			rows, err := statusRowsFetcher()
			statusRowsFlightState.mu.Lock()
			statusRowsFlightState.rows = cloneAgentPanelRows(rows)
			statusRowsFlightState.err = err
			statusRowsFlightState.running = false
			close(waitCh)
			statusRowsFlightState.mu.Unlock()
		}(ch)
	}
	statusRowsFlightState.mu.Unlock()
	select {
	case <-ch:
		statusRowsFlightState.mu.Lock()
		rows := cloneAgentPanelRows(statusRowsFlightState.rows)
		err := statusRowsFlightState.err
		statusRowsFlightState.mu.Unlock()
		return rows, err
	case <-time.After(timeout):
		return nil, errStatusFetchTimeout
	}
}

func resetStatusRowsFlightState() {
	statusRowsFlightState.mu.Lock()
	defer statusRowsFlightState.mu.Unlock()
	statusRowsFlightState.running = false
	statusRowsFlightState.waitCh = nil
	statusRowsFlightState.rows = nil
	statusRowsFlightState.err = nil
}

func agentesVisiblesPorEstadoOperativoRows(agentes []*db.Agente, rows []agentesapp.Row) ([]*db.Agente, []*db.Agente, []*db.Agente, []*db.Agente, []*db.Agente, []*db.Agente, bool) {
	now := statusNowFunc().UTC()
	rowPorNombre := make(map[string]agentesapp.Row, len(rows))
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(row.Agente.Nombre)
		if nombre == "" {
			continue
		}
		rowPorNombre[nombre] = row
	}
	agenteCanonico := make(map[string]*db.Agente, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" {
			continue
		}
		agenteCanonico[nombre] = preferAgenteStatusCanonico(agenteCanonico[nombre], agente)
	}
	agentesActivos := make([]*db.Agente, 0, len(agentes))
	agentesTrabajando := make([]*db.Agente, 0, len(agentes))
	agentesSaturados := make([]*db.Agente, 0, len(agentes))
	agentesAtascados := make([]*db.Agente, 0, len(agentes))
	agentesAuthManual := make([]*db.Agente, 0, len(agentes))
	agentesQuotaBlocked := make([]*db.Agente, 0, len(agentes))
	for nombre, agente := range agenteCanonico {
		if agente == nil {
			continue
		}
		if !agenteVisibleEnStatusFleet(nombre) {
			continue
		}
		row, ok := rowPorNombre[nombre]
		if !ok {
			continue
		}
		if rowEsResiduoPausadoSinTrabajo(row, now) {
			continue
		}
		if agenteBloqueadoPorCuotaVisible(agente) {
			agentesQuotaBlocked = append(agentesQuotaBlocked, agente)
			continue
		}
		switch strings.TrimSpace(row.EstadoOperativo) {
		case "arrancando":
			agentesActivos = append(agentesActivos, agente)
		case "trabajando":
			agentesActivos = append(agentesActivos, agente)
			if rowCountsAsWorkingVisible(row) {
				agentesTrabajando = append(agentesTrabajando, agente)
			}
		case "saturado":
			agentesActivos = append(agentesActivos, agente)
			if rowCountsAsWorkingVisible(row) {
				agentesTrabajando = append(agentesTrabajando, agente)
				agentesSaturados = append(agentesSaturados, agente)
			}
		case "atascado", "mailbox_atascada":
			agentesAtascados = append(agentesAtascados, agente)
			if rowCountsAsConnected(row) {
				agentesActivos = append(agentesActivos, agente)
			}
		case "disponible":
			agentesActivos = append(agentesActivos, agente)
		case "bloqueado_por_runtime":
			if strings.EqualFold(strings.TrimSpace(row.WorkerState), "blocked_auth") ||
				strings.Contains(strings.ToLower(strings.TrimSpace(row.DetalleOperativo)), "autenticacion manual") {
				agentesAuthManual = append(agentesAuthManual, agente)
			}
		case "bloqueado_por_cuota":
			agentesQuotaBlocked = append(agentesQuotaBlocked, agente)
		}
	}
	return agentesActivos, agentesTrabajando, agentesSaturados, agentesAtascados, agentesAuthManual, agentesQuotaBlocked, true
}

func rowCountsAsWorkingVisible(row agentesapp.Row) bool {
	if row.OpenTasks > 0 || row.BlockedTasks > 0 {
		return true
	}
	if row.CurrentTask != nil &&
		row.CurrentTask.TaskID != 0 &&
		(row.CurrentTask.State == db.TareaEnProgreso || row.CurrentTask.State == db.TareaBloqueada) {
		return true
	}
	return false
}

func rowEsResiduoPausadoSinTrabajo(row agentesapp.Row, now time.Time) bool {
	if row.Asignacion == nil || !strings.EqualFold(strings.TrimSpace(string(row.Asignacion.Estado)), string(db.AsignacionPausada)) {
		return false
	}
	if row.OpenTasks > 0 || row.BlockedTasks > 0 {
		return false
	}
	if row.SupervisorRoleActive(now) {
		return false
	}
	return true
}

func aplicarVisibilidadOperativaAgentes(agentes []*db.Agente, rows []agentesapp.Row) {
	rowPorNombre := make(map[string]agentesapp.Row, len(rows))
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(row.Agente.Nombre)
		if nombre == "" {
			continue
		}
		rowPorNombre[nombre] = row
	}
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		row, ok := rowPorNombre[nombreAgenteCanonico(agente.Nombre)]
		if !ok {
			continue
		}
		switch strings.TrimSpace(row.EstadoOperativo) {
		case "bloqueado_por_cuota":
			agente.Activo = false
			if !agenteBloqueadoPorCuotaVisible(agente) {
				agente.EstadoCuota = "enfriamiento"
				if strings.TrimSpace(agente.MotivoPausa) == "" {
					agente.MotivoPausa = strings.TrimSpace(row.DetalleOperativo)
				}
			}
		case "arrancando", "trabajando", "disponible", "saturado", "atascado", "mailbox_atascada":
			agente.Activo = rowCountsAsConnected(row)
		default:
			agente.Activo = false
		}
	}
}

func rowCountsAsConnected(row agentesapp.Row) bool {
	switch strings.TrimSpace(row.EstadoOperativo) {
	case "atascado", "mailbox_atascada":
		if row.Handle != nil {
			switch strings.ToLower(strings.TrimSpace(row.Handle.Estado)) {
			case "activo", "active", "pausado", "paused":
				return true
			}
		}
		if row.Runtime != nil {
			switch strings.ToLower(strings.TrimSpace(firstNonEmpty(row.Runtime.LogicalState, row.Runtime.ProcessState))) {
			case "activo", "active", "pausado", "paused", "esperando_io", "running":
				return true
			}
		}
		if row.WorkerAlive || strings.TrimSpace(row.WorkerState) != "" || row.WorkerHeartbeat != nil || row.WorkerUpdatedAt != nil || strings.TrimSpace(row.WorkerTMUXSession) != "" {
			return true
		}
		return false
	default:
		return true
	}
}

func listarPoolsLocalesCompartidosEstado() ([]*capacidadapp.PoolLocalCompartido, error) {
	resumen, err := capacidadService.ListPoolsSummary(nil)
	if err != nil {
		return nil, err
	}
	out := make([]*capacidadapp.PoolLocalCompartido, 0, len(resumen))
	for _, item := range resumen {
		if item == nil || item.Pool == nil {
			continue
		}
		pool := item.Pool
		proveedor := strings.ToLower(strings.TrimSpace(pool.Proveedor))
		runtime := strings.ToLower(strings.TrimSpace(pool.Runtime))
		meta := strings.ToLower(strings.TrimSpace(pool.MetadataJSON))
		if proveedor != "local" {
			continue
		}
		if !strings.Contains(runtime, "ollama") && !strings.Contains(meta, "ollama_pool_local") {
			continue
		}
		detalle, err := capacidadService.DescribirPoolLocalCompartido(strings.TrimSpace(pool.Slug))
		if err != nil {
			out = append(out, &capacidadapp.PoolLocalCompartido{
				PoolSlug:         strings.TrimSpace(pool.Slug),
				Proveedor:        strings.TrimSpace(pool.Proveedor),
				Runtime:          strings.TrimSpace(pool.Runtime),
				SlotsMaximos:     pool.CapacidadTotal,
				ConectorCanonico: "ollama_pool_local",
				Pool:             pool,
			})
			continue
		}
		out = append(out, detalle)
	}
	return out, nil
}

func agentesVisiblesPorEstadoOperativo(agentes []*db.Agente) ([]*db.Agente, []*db.Agente, bool) {
	rows, err := agentRowsForStatus()
	if err != nil {
		return nil, nil, false
	}
	activos, trabajando, _, _, _, _, ok := agentesVisiblesPorEstadoOperativoRows(agentes, rows)
	return activos, trabajando, ok
}

func (dbStatusService) FetchStatus() (apiStatusResponse, error) {
	for {
		now := statusNowFunc().UTC()
		statusCacheState.mu.Lock()
		ok := statusCacheState.ok
		value := statusCacheState.value
		expires := statusCacheState.expires
		refreshing := statusCacheState.refreshing
		retryAfter := statusCacheState.retryAfter
		waitCh := statusCacheState.waitCh
		needsRefresh := statusSnapshotNeedsImmediateRefresh(value)
		if ok && now.Before(expires) && !needsRefresh {
			statusCacheState.mu.Unlock()
			return value, nil
		}
		if retryAfter.After(now) {
			statusCacheState.mu.Unlock()
			if ok {
				return value, nil
			}
			if status, fallbackErr := runStatusFetcherWithTimeout(statusFastFetcher, statusFastTimeout); fallbackErr == nil {
				storeStatusSnapshot(status, now)
				markStatusRefreshBackoff(now)
				return status, nil
			}
			return apiStatusResponse{}, errStatusFetchTimeout
		}
		if ok && !needsRefresh {
			if statusAsyncRefresh && !refreshing {
				waitCh = startStatusRefreshLocked()
				statusCacheState.mu.Unlock()
				go refreshStatusSnapshot(waitCh)
				return value, nil
			}
			statusCacheState.mu.Unlock()
			return value, nil
		}
		if refreshing && waitCh != nil {
			statusCacheState.mu.Unlock()
			if ok {
				return value, nil
			}
			now = statusNowFunc().UTC()
			if status, fallbackErr := runStatusFetcherWithTimeout(statusFastFetcher, statusFastTimeout); fallbackErr == nil {
				storeStatusSnapshot(status, now)
				markStatusRefreshBackoff(now)
				return status, nil
			}
			if waitStatusRefresh(waitCh, statusRefreshWaitTimeout) {
				if cached, ok := readStatusSnapshotAny(); ok {
					return cached, nil
				}
			}
			return apiStatusResponse{}, errStatusFetchTimeout
		}
		waitCh = startStatusRefreshLocked()
		statusCacheState.mu.Unlock()
		go refreshStatusSnapshot(waitCh)
		if status, fallbackErr := runStatusFetcherWithTimeout(statusFastFetcher, statusFastTimeout); fallbackErr == nil {
			storeStatusSnapshot(status, now)
			markStatusRefreshBackoff(now)
			return status, nil
		}
		if waitStatusRefresh(waitCh, statusRefreshWaitTimeout) {
			if cached, ok := readStatusSnapshotAny(); ok {
				return cached, nil
			}
		}
		return apiStatusResponse{}, errStatusFetchTimeout
	}
}

func startStatusRefreshLocked() chan struct{} {
	if statusCacheState.refreshing && statusCacheState.waitCh != nil {
		return statusCacheState.waitCh
	}
	statusCacheState.refreshing = true
	statusCacheState.waitCh = make(chan struct{})
	return statusCacheState.waitCh
}

func finishStatusRefresh(ch chan struct{}, status apiStatusResponse, now time.Time, ttl time.Duration, ok bool) {
	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	if ok {
		statusCacheState.value = status
		statusCacheState.expires = now.Add(ttl)
		statusCacheState.ok = true
		statusCacheState.retryAfter = time.Time{}
	}
	if statusCacheState.waitCh == ch {
		statusCacheState.waitCh = nil
	}
	statusCacheState.refreshing = false
	if ch != nil {
		close(ch)
	}
}

func storeStatusSnapshotAndFinish(ch chan struct{}, status apiStatusResponse, now time.Time, ttl time.Duration) {
	finishStatusRefresh(ch, status, now, ttl, true)
}

func refreshStatusSnapshot(ch chan struct{}) {
	defer func() {
		_ = recover()
	}()
	now := statusNowFunc().UTC()
	status, err := runStatusFetcherWithTimeout(statusFreshFetcher, statusFreshTimeout)
	if err != nil {
		markStatusRefreshBackoff(now)
		finishStatusRefresh(ch, apiStatusResponse{}, time.Time{}, 0, false)
		return
	}
	storeStatusSnapshotAndFinish(ch, status, now, statusSnapshotTTL)
}

func runStatusFetcherWithTimeout(fetcher func() (apiStatusResponse, error), timeout time.Duration) (apiStatusResponse, error) {
	if fetcher == nil {
		return apiStatusResponse{}, errors.New("status fetcher nil")
	}
	if timeout <= 0 {
		return fetcher()
	}
	type result struct {
		status apiStatusResponse
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		status, err := fetcher()
		ch <- result{status: status, err: err}
	}()
	select {
	case res := <-ch:
		return res.status, res.err
	case <-time.After(timeout):
		return apiStatusResponse{}, errStatusFetchTimeout
	}
}

func waitStatusRefresh(ch chan struct{}, timeout time.Duration) bool {
	if ch == nil {
		return true
	}
	if timeout <= 0 {
		<-ch
		return true
	}
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

func storeStatusSnapshot(status apiStatusResponse, now time.Time) {
	if statusSnapshotIsDegenerate(status) {
		return
	}
	ttl := statusSnapshotTTL
	if statusSnapshotNeedsImmediateRefresh(status) {
		ttl = statusFallbackTTL
	}
	storeStatusSnapshotWithTTL(status, now, ttl)
}

func storeStatusSnapshotWithTTL(status apiStatusResponse, now time.Time, ttl time.Duration) {
	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	statusCacheState.value = status
	statusCacheState.expires = now.Add(ttl)
	statusCacheState.ok = true
	statusCacheState.hardStale = false
	statusCacheState.retryAfter = time.Time{}
	statusCacheState.refreshing = false
	statusCacheState.waitCh = nil
}

func markStatusRefreshBackoff(now time.Time) {
	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	statusCacheState.retryAfter = now.Add(statusFailureBackoffTTL)
}

func readStatusSnapshotAny() (apiStatusResponse, bool) {
	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	if !statusCacheState.ok || statusCacheState.hardStale {
		return apiStatusResponse{}, false
	}
	if statusSnapshotIsDegenerate(statusCacheState.value) {
		return apiStatusResponse{}, false
	}
	return statusCacheState.value, true
}

func readStatusSnapshotFresh() (apiStatusResponse, bool) {
	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	if !statusCacheState.ok || statusCacheState.hardStale {
		return apiStatusResponse{}, false
	}
	if statusSnapshotIsDegenerate(statusCacheState.value) {
		return apiStatusResponse{}, false
	}
	if statusCacheState.expires.IsZero() || !statusNowFunc().UTC().Before(statusCacheState.expires) {
		return apiStatusResponse{}, false
	}
	return statusCacheState.value, true
}

func readStatusSnapshotFreshUsable() (apiStatusResponse, bool) {
	status, ok := readStatusSnapshotFresh()
	if !ok {
		return apiStatusResponse{}, false
	}
	if statusSnapshotNeedsImmediateRefresh(status) {
		return apiStatusResponse{}, false
	}
	return status, true
}

func ensureStatusRefreshAsync() {
	now := statusNowFunc().UTC()
	statusCacheState.mu.Lock()
	if statusCacheState.refreshing || statusCacheState.retryAfter.After(now) {
		statusCacheState.mu.Unlock()
		return
	}
	waitCh := startStatusRefreshLocked()
	statusCacheState.mu.Unlock()
	go refreshStatusSnapshot(waitCh)
}

func statusSnapshotIsDegenerate(status apiStatusResponse) bool {
	if len(status.Agentes) > 0 ||
		len(status.AgentesActivos) > 0 ||
		len(status.AgentesTrabajando) > 0 ||
		len(status.AgentesSaturados) > 0 ||
		len(status.AgentesAtascados) > 0 ||
		len(status.AgentesAuthManual) > 0 ||
		len(status.AgentesQuotaBlocked) > 0 ||
		len(status.TareasActivas) > 0 ||
		len(status.TareasEnProgreso) > 0 ||
		len(status.TareasReservadas) > 0 ||
		len(status.Proyectos) > 0 ||
		len(status.PropuestasAbiertas) > 0 ||
		len(status.PropuestasResumen) > 0 ||
		len(status.PoolsLocales) > 0 ||
		len(status.ConteoTareas) > 0 ||
		len(status.TareasPorEstado) > 0 ||
		len(status.ResumenTareas) > 0 ||
		len(status.AsignacionesActivas) > 0 ||
		len(status.SesionesActivas) > 0 {
		return false
	}
	if status.DeudaDispatch.Total > 0 || status.DeudaDispatch.Pendientes > 0 || status.DeudaDispatch.Notificadas > 0 || status.DeudaDispatch.Fallidas > 0 || status.DeudaDispatch.WorkConfirmed > 0 {
		return false
	}
	if status.Autonomia.Supervisando > 0 || status.Autonomia.Continuando > 0 || status.Autonomia.ContinuidadPendiente > 0 || status.Autonomia.WorkConfirmed > 0 || status.Autonomia.Handoffs > 0 {
		return false
	}
	if status.Autonomia.Count > 0 || len(status.Autonomia.ByKind) > 0 || status.Autonomia.LastAt != nil || len(status.Autonomia.Recent) > 0 {
		return false
	}
	return true
}

func statusSnapshotNeedsImmediateRefresh(status apiStatusResponse) bool {
	if statusSnapshotCanStayLight(status) {
		return false
	}
	if statusSnapshotIsDegenerate(status) {
		return true
	}
	for _, agente := range status.AgentesActivos {
		if agente != nil && !agente.Activo {
			return true
		}
	}
	if len(status.AgentesAuthManual) > 0 {
		return true
	}
	if len(status.AgentesAtascados) > 0 {
		return true
	}
	if status.Autonomia.ContinuidadPendiente > 0 || status.Autonomia.Handoffs > 0 {
		return true
	}
	tasksInProgress := len(status.TareasEnProgreso)
	if tasksInProgress == 0 && status.TareasPorEstado != nil {
		tasksInProgress = status.TareasPorEstado[string(db.TareaEnProgreso)]
	}
	if tasksInProgress == 0 && status.ConteoTareas != nil {
		tasksInProgress = status.ConteoTareas[string(db.TareaEnProgreso)]
	}
	if tasksInProgress == 0 {
		for _, tarea := range status.TareasActivas {
			if tarea.Estado == db.TareaEnProgreso {
				tasksInProgress++
			}
		}
	}
	if len(status.AgentesActivos) > 0 {
		if tasksInProgress > 0 && len(status.AgentesTrabajando) == 0 && status.Autonomia.WorkConfirmed == 0 {
			return true
		}
		return false
	}
	if len(status.AgentesQuotaBlocked) > 0 && tasksInProgress > 0 {
		return false
	}
	if tasksInProgress > 0 {
		return true
	}
	return false
}

func statusSnapshotCanStayLight(status apiStatusResponse) bool {
	info := buildServerOperationalInfo(status)
	if info.State != "idle" || info.Reason != "workers_quota_blocked" {
		return false
	}
	if info.ConnectedWorkers > 0 || info.WorkingWorkers > 0 {
		return false
	}
	if info.AuthAgents > 0 || info.StuckAgents > 0 {
		return false
	}
	if info.ReservedTasks > 0 || info.BlockedTasks > 0 {
		return false
	}
	if info.DispatchPending > 0 || info.DispatchNotified > 0 || info.DispatchFailed > 0 {
		return false
	}
	if status.Autonomia.ContinuidadPendiente > 0 || status.Autonomia.Handoffs > 0 {
		return false
	}
	return true
}

func fetchStatusFastFallback() (apiStatusResponse, error) {
	type agentesResult struct {
		sesiones []*db.Sesion
		agentes  []*db.Agente
		err      error
	}
	type cuentasResult struct {
		cuentas map[string]int
		err     error
	}
	type tareasResult struct {
		tareas []tareaLite
		err    error
	}

	agentesCh := make(chan agentesResult, 1)
	cuentasCh := make(chan cuentasResult, 1)
	tareasCh := make(chan tareasResult, 1)

	go func() {
		sesiones, err := sesionesVisiblesParaEstado()
		if err != nil {
			agentesCh <- agentesResult{err: err}
			return
		}
		agentes, err := db.ListarAgentesEstadoLigeroConSesionesActivas(sesiones)
		agentesCh <- agentesResult{sesiones: sesiones, agentes: agentes, err: err}
	}()
	go func() {
		cuentas, err := statusCountTasksFetcher()
		cuentasCh <- cuentasResult{cuentas: cuentas, err: err}
	}()
	go func() {
		tareas, err := listarTareasActivasRapido()
		tareasCh <- tareasResult{tareas: tareas, err: err}
	}()

	agentesRes := <-agentesCh
	if agentesRes.err != nil {
		return apiStatusResponse{}, agentesRes.err
	}
	agentes := agentesRes.agentes
	cuentasRes := <-cuentasCh
	if cuentasRes.err != nil {
		return apiStatusResponse{}, cuentasRes.err
	}
	cuentas := cuentasRes.cuentas
	tareasRes := <-tareasCh
	if tareasRes.err != nil {
		return apiStatusResponse{}, tareasRes.err
	}
	tareasActivas := tareasRes.tareas

	tareasActivas = filtrarTareasActivasVisibles(tareasActivas, agentes)
	cuentas = reconciliarConteoTareasActivasVisible(cuentas, tareasActivas)
	tareasEnProgreso := filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaEnProgreso)
	tareasReservadas := filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaAsignada)
	var agentesActivos []*db.Agente
	var agentesTrabajando []*db.Agente
	var agentesSaturados []*db.Agente
	var agentesAtascados []*db.Agente
	var agentesAuthManual []*db.Agente
	var agentesQuotaBlocked []*db.Agente
	autonomia := autonomiaResumen{}
	rowsResolved := false
	if rows, err := statusRowsForSnapshot(statusRowsTimeout); err == nil {
		rowsResolved = true
		tareasActivas = reconciliarTareasActivasConPanelRows(tareasActivas, rows)
		cuentas = reconciliarConteoTareasActivasVisible(cuentas, tareasActivas)
		tareasEnProgreso = filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaEnProgreso)
		tareasReservadas = filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaAsignada)
		aplicarVisibilidadOperativaAgentes(agentes, rows)
		agentesActivos, agentesTrabajando, agentesSaturados, agentesAtascados, agentesAuthManual, agentesQuotaBlocked, _ = agentesVisiblesPorEstadoOperativoRows(agentes, rows)
		agentesActivos, agentesTrabajando, agentesSaturados = normalizarAgentesVisiblesStatus(agentesActivos, agentesTrabajando, agentesSaturados)
		autonomia = resumirAutonomiaRows(rows, statusNowFunc().UTC())
	} else {
		agentesActivos, agentesTrabajando, agentesQuotaBlocked = agentesVisiblesLigero(agentes, tareasEnProgreso)
		agentesActivos, agentesTrabajando, _ = normalizarAgentesVisiblesStatus(agentesActivos, agentesTrabajando, nil)
		autonomia = resumirAutonomiaLigera(agentesActivos, agentesTrabajando, tareasEnProgreso, statusNowFunc().UTC())
	}
	if autonomia.Handoffs == 0 && !rowsResolved {
		if handoffs, err := statusHandoffsFetcher(); err == nil && handoffs > autonomia.Handoffs {
			autonomia.Handoffs = handoffs
		}
	}
	if out, ok := runStatusOptional(statusOptionalSectionTimeout, func() (autonomiaResumen, error) {
		return resumirAutonomiaEventosRecientes(autonomia, statusNowFunc().UTC())
	}); ok {
		autonomia = out
	} else if autonomia.ByKind == nil {
		autonomia.ByKind = map[string]int{}
	}
	surface, riskSummary := loadStatusAutonomySurfaceAndRisk()
	workersConectados, workersTrabajando, supervisoresActivos := statusVisibleWorkerCounters(agentesActivos, agentesTrabajando, autonomia)
	return apiStatusResponse{
		Agentes:             agentes,
		ConteoTareas:        cuentas,
		ResumenTareas:       cuentas,
		Generado:            statusNowFunc().UTC().Format(time.RFC3339),
		TareasPorEstado:     cuentas,
		AgentesActivos:      agentesActivos,
		AgentesTrabajando:   agentesTrabajando,
		AgentesSaturados:    agentesSaturados,
		AgentesAtascados:    agentesAtascados,
		AgentesAuthManual:   agentesAuthManual,
		AgentesQuotaBlocked: agentesQuotaBlocked,
		TareasActivas:       tareasActivas,
		TareasEnProgreso:    tareasEnProgreso,
		TareasReservadas:    tareasReservadas,
		Autonomia:           autonomia,
		AutonomySurface:     surface,
		AutonomyHighlights:  riskSummary.Highlights,
		CriticalProjectRisk: riskSummary.CriticalProjectRisk,
		WorkersConectados:   workersConectados,
		WorkersTrabajando:   workersTrabajando,
		SupervisoresActivos: supervisoresActivos,
	}, nil
}

func listarTareasActivasRapido() ([]tareaLite, error) {
	estados := []db.EstadoTarea{db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada}
	out := make([]tareaLite, 0, 16)
	for _, estado := range estados {
		items, err := db.ListarTareas(db.FiltroTareas{Estado: &estado})
		if err != nil {
			return nil, err
		}
		for _, tarea := range items {
			if tarea == nil {
				continue
			}
			lite := tareaLite{
				ID:        tarea.ID,
				Titulo:    tarea.Titulo,
				Estado:    tarea.Estado,
				Modulo:    tarea.Modulo,
				Prioridad: tarea.Prioridad,
			}
			if tarea.Agente != nil {
				lite.Agente = *tarea.Agente
			}
			out = append(out, lite)
		}
	}
	return normalizarTareasLiteVisibles(out), nil
}

func fetchStatusFresh() (apiStatusResponse, error) {
	sesiones, err := sesionesVisiblesParaEstado()
	if err != nil {
		return apiStatusResponse{}, err
	}
	agentes, err := statusListAgentsWithSessionsFetcher(sesiones)
	if err != nil {
		return apiStatusResponse{}, err
	}
	cuentas, err := statusCountTasksFetcher()
	if err != nil {
		return apiStatusResponse{}, err
	}
	var proyectos []*db.Proyecto
	if out, ok := runStatusOptional(statusOptionalSectionTimeout, statusListProjectsFetcher); ok {
		proyectos = out
	}
	asignaciones := map[int64]int{}
	if out, ok := runStatusOptional(statusOptionalSectionTimeout, statusCountAssignmentsFetcher); ok {
		asignaciones = out
	}
	sesionesPorProyecto := map[int64]int{}
	for _, sesion := range sesiones {
		if sesion.ProyectoID != nil {
			sesionesPorProyecto[*sesion.ProyectoID]++
		}
	}
	abiertas := []*db.Propuesta{}
	propuestasResumen := []propuestaLite{}
	if statusIncludeProposalSections {
		if out, ok := runStatusOptional(statusOptionalSectionTimeout, statusListOpenProposalsFetcher); ok {
			abiertas = out
		}
		propuestaIDs := make([]int64, 0, len(abiertas))
		for _, propuesta := range abiertas {
			if propuesta == nil {
				continue
			}
			propuestaIDs = append(propuestaIDs, propuesta.ID)
		}
		votosPorPropuesta := map[int64][]*db.Voto{}
		if len(propuestaIDs) > 0 {
			if out, ok := runStatusOptional(statusOptionalSectionTimeout, func() (map[int64][]*db.Voto, error) {
				return statusVotesSummaryFetcher(propuestaIDs)
			}); ok {
				votosPorPropuesta = out
			}
		}
		propuestasResumen = make([]propuestaLite, 0, len(abiertas))
		for _, propuesta := range abiertas {
			if propuesta == nil {
				continue
			}
			votos := votosPorPropuesta[propuesta.ID]
			propuesta.Votos = votos
			lite := propuestaLite{
				ID:           propuesta.ID,
				Codigo:       propuesta.Codigo,
				Titulo:       propuesta.Titulo,
				Estado:       propuesta.Estado,
				PropuestoPor: propuesta.PropuestoPor,
			}
			for _, voto := range votos {
				if voto == nil {
					continue
				}
				switch voto.Posicion {
				case db.VotoAcuerdo:
					lite.Acuerdo++
				case db.VotoDesacuerdo:
					lite.Desacuerdo++
				case db.VotoAbstencion:
					lite.Abstencion++
				default:
					lite.Pendiente++
				}
			}
			propuestasResumen = append(propuestasResumen, lite)
		}
	}
	tareasActivas, err := listarTareasActivasRapido()
	if err != nil {
		return apiStatusResponse{}, err
	}
	tareasActivas = filtrarTareasActivasVisibles(tareasActivas, agentes)
	tareasEnProgreso := filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaEnProgreso)
	cuentas = reconciliarConteoTareasActivasVisible(cuentas, tareasActivas)
	trabajandoNombres := make(map[string]bool)
	for _, tarea := range tareasActivas {
		if strings.TrimSpace(tarea.Agente) != "" && tarea.Estado == db.TareaEnProgreso {
			trabajandoNombres[nombreAgenteCanonico(tarea.Agente)] = true
		}
	}
	agentesActivos := make([]*db.Agente, 0, len(agentes))
	agentesPorNombre := make(map[string]*db.Agente, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" {
			continue
		}
		agente = preferAgenteStatusCanonico(agentesPorNombre[nombre], agente)
		agentesPorNombre[nombre] = agente
	}
	for _, agente := range agentesPorNombre {
		if agenteCuentaComoConectado(agente) {
			agentesActivos = append(agentesActivos, agente)
		}
	}
	trabajandoPorNombre := make(map[string]*db.Agente)
	for nombre := range trabajandoNombres {
		agente := agentesPorNombre[nombre]
		if !agenteCuentaComoConectado(agente) {
			continue
		}
		trabajandoPorNombre[nombre] = agente
	}
	agentesTrabajando := make([]*db.Agente, 0, len(trabajandoPorNombre))
	agentesSaturados := make([]*db.Agente, 0)
	agentesAtascados := make([]*db.Agente, 0)
	agentesAuthManual := make([]*db.Agente, 0)
	agentesQuotaBlocked := make([]*db.Agente, 0)
	poolsLocales, _ := runStatusOptional(statusOptionalSectionTimeout, statusPoolsFetcher)
	deudaDispatch := deudaDispatchResumen{}
	dispatchSummaryLoaded := false
	dispatchSummary, ok := runStatusOptional(statusOptionalSectionTimeout, statusDispatchSummaryFetcher)
	if ok {
		deudaDispatch = dispatchSummary.Deuda
		dispatchSummaryLoaded = true
	} else {
		deudaDispatch, _ = runStatusOptional(statusOptionalSectionTimeout, statusDispatchDebtFetcher)
	}
	autonomia := autonomiaResumen{}
	for _, agente := range agentesActivos {
		if agente == nil {
			continue
		}
		if trabajandoPorNombre[nombreAgenteCanonico(agente.Nombre)] != nil {
			agentesTrabajando = append(agentesTrabajando, agente)
		}
	}
	if rows, rowsErr := statusRowsForSnapshot(statusRowsTimeout); rowsErr == nil {
		tareasActivas = reconciliarTareasActivasConPanelRows(tareasActivas, rows)
		tareasEnProgreso = filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaEnProgreso)
		cuentas = reconciliarConteoTareasActivasVisible(cuentas, tareasActivas)
		aplicarVisibilidadOperativaAgentes(agentes, rows)
		if activosOperativos, trabajandoOperativos, saturadosOperativos, atascadosOperativos, authManualOperativos, quotaBlockedOperativos, ok := agentesVisiblesPorEstadoOperativoRows(agentes, rows); ok {
			agentesActivos = activosOperativos
			agentesTrabajando = trabajandoOperativos
			agentesSaturados = saturadosOperativos
			agentesAtascados = atascadosOperativos
			agentesAuthManual = authManualOperativos
			agentesQuotaBlocked = quotaBlockedOperativos
		}
		agentesActivos, agentesTrabajando, agentesSaturados = normalizarAgentesVisiblesStatus(agentesActivos, agentesTrabajando, agentesSaturados)
		autonomia = resumirAutonomiaRows(rows, statusNowFunc().UTC())
	} else {
		agentesActivos, agentesTrabajando, agentesQuotaBlocked = agentesVisiblesLigero(agentes, tareasEnProgreso)
		agentesActivos, agentesTrabajando, _ = normalizarAgentesVisiblesStatus(agentesActivos, agentesTrabajando, nil)
		autonomia = resumirAutonomiaLigera(agentesActivos, agentesTrabajando, tareasEnProgreso, statusNowFunc().UTC())
	}
	if autonomia.Handoffs == 0 && dispatchSummaryLoaded {
		if dispatchSummary.Handoffs > autonomia.Handoffs {
			autonomia.Handoffs = dispatchSummary.Handoffs
		}
	} else if autonomia.Handoffs == 0 {
		if handoffs, err := statusHandoffsFetcher(); err == nil && handoffs > autonomia.Handoffs {
			autonomia.Handoffs = handoffs
		}
	}
	if out, ok := runStatusOptional(statusOptionalSectionTimeout, func() (autonomiaResumen, error) {
		return resumirAutonomiaEventosRecientes(autonomia, statusNowFunc().UTC())
	}); ok {
		autonomia = out
	} else if autonomia.ByKind == nil {
		autonomia.ByKind = map[string]int{}
	}
	surface, riskSummary := loadStatusAutonomySurfaceAndRisk()
	workersConectados, workersTrabajando, supervisoresActivos := statusVisibleWorkerCounters(agentesActivos, agentesTrabajando, autonomia)
	return apiStatusResponse{
		Agentes:             agentes,
		ConteoTareas:        cuentas,
		ResumenTareas:       cuentas,
		Proyectos:           proyectos,
		AsignacionesActivas: asignaciones,
		SesionesActivas:     sesionesPorProyecto,
		PropuestasAbiertas:  abiertas,
		Generado:            time.Now().UTC().Format(time.RFC3339),
		TareasPorEstado:     cuentas,
		AgentesActivos:      agentesActivos,
		AgentesTrabajando:   agentesTrabajando,
		AgentesSaturados:    agentesSaturados,
		AgentesAtascados:    agentesAtascados,
		AgentesAuthManual:   agentesAuthManual,
		AgentesQuotaBlocked: agentesQuotaBlocked,
		PropuestasResumen:   propuestasResumen,
		TareasActivas:       tareasActivas,
		TareasEnProgreso:    tareasEnProgreso,
		TareasReservadas:    filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaAsignada),
		PoolsLocales:        poolsLocales,
		DeudaDispatch:       deudaDispatch,
		Autonomia:           autonomia,
		AutonomySurface:     surface,
		AutonomyHighlights:  riskSummary.Highlights,
		CriticalProjectRisk: riskSummary.CriticalProjectRisk,
		WorkersConectados:   workersConectados,
		WorkersTrabajando:   workersTrabajando,
		SupervisoresActivos: supervisoresActivos,
	}, nil
}

func resetStatusSnapshotCache() {
	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	statusCacheState.value = apiStatusResponse{}
	statusCacheState.expires = time.Time{}
	statusCacheState.ok = false
	statusCacheState.hardStale = false
	statusCacheState.retryAfter = time.Time{}
	statusCacheState.refreshing = false
	statusCacheState.waitCh = nil
	statusAutonomySurfaceState.mu.Lock()
	statusAutonomySurfaceState.value = nil
	statusAutonomySurfaceState.expires = time.Time{}
	statusAutonomySurfaceState.refreshing = false
	statusAutonomySurfaceState.waitCh = nil
	statusAutonomySurfaceState.mu.Unlock()
	if agentesService != nil {
		agentesService.InvalidateCompactDetailCache()
	}
}

func invalidateStatusSnapshotCache() {
	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	if statusCacheState.ok {
		deadline := statusNowFunc().UTC().Add(statusSoftInvalidateTTL)
		if statusCacheState.expires.IsZero() || statusCacheState.expires.After(deadline) {
			statusCacheState.expires = deadline
		}
	}
	statusCacheState.retryAfter = time.Time{}
	if agentesService != nil {
		agentesService.InvalidateCompactDetailCache()
	}
}

func invalidateStatusSnapshotCacheHard() {
	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	if statusCacheState.ok {
		statusCacheState.hardStale = true
	}
	statusCacheState.retryAfter = time.Time{}
	if agentesService != nil {
		agentesService.InvalidateCompactDetailCache()
	}
}

func reconciliarConteoTareasActivasVisible(cuentas map[string]int, tareasActivas []tareaLite) map[string]int {
	if cuentas == nil {
		cuentas = map[string]int{}
	}
	tareasActivas = normalizarTareasLiteVisibles(tareasActivas)
	out := make(map[string]int, len(cuentas)+3)
	for estado, n := range cuentas {
		out[estado] = n
	}
	for _, estado := range []db.EstadoTarea{db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada} {
		out[string(estado)] = 0
	}
	for _, tarea := range tareasActivas {
		switch tarea.Estado {
		case db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada:
			out[string(tarea.Estado)]++
		}
	}
	return out
}

func reconciliarTareasActivasConPanelRows(tareas []tareaLite, rows []agentesapp.Row) []tareaLite {
	tareas = normalizarTareasLiteVisibles(tareas)
	if len(tareas) == 0 || len(rows) == 0 {
		return tareas
	}
	type agentConstraint struct {
		currentTaskID int64
		openTasks     int
	}
	constraints := make(map[string]agentConstraint, len(rows))
	taskStates := make(map[int64]db.EstadoTarea, len(rows))
	withoutVisibleTask := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(row.Agente.Nombre)
		if nombre == "" {
			continue
		}
		if row.CurrentTask == nil &&
			row.OpenTasks == 0 &&
			row.BlockedTasks == 0 &&
			row.MailboxPending == 0 &&
			row.MailboxActionablePending == 0 &&
			row.MailboxContinuityPending == 0 &&
			row.OrdersOpen == 0 &&
			row.ControlOrdersOpen == 0 {
			withoutVisibleTask[nombre] = struct{}{}
			continue
		}
		delete(withoutVisibleTask, nombre)
		if row.CurrentTask == nil {
			continue
		}
		if row.OpenTasks == 1 {
			constraints[nombre] = agentConstraint{
				currentTaskID: row.CurrentTask.TaskID,
				openTasks:     row.OpenTasks,
			}
		}
		switch row.CurrentTask.State {
		case db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada:
			taskStates[row.CurrentTask.TaskID] = row.CurrentTask.State
		}
	}
	if len(constraints) == 0 && len(taskStates) == 0 {
		return tareas
	}
	out := make([]tareaLite, 0, len(tareas))
	for _, tarea := range tareas {
		if !tareaLiteValida(tarea) {
			continue
		}
		if state, ok := taskStates[tarea.ID]; ok && tarea.Estado != state {
			tarea.Estado = state
		}
		nombre := nombreAgenteCanonico(tarea.Agente)
		if _, ok := withoutVisibleTask[nombre]; ok {
			continue
		}
		if constraint, ok := constraints[nombre]; ok {
			switch tarea.Estado {
			case db.TareaAsignada, db.TareaEnProgreso:
				if constraint.openTasks == 1 && tarea.ID != constraint.currentTaskID {
					continue
				}
			}
		}
		out = append(out, tarea)
	}
	return normalizarTareasLiteVisibles(out)
}

func normalizarTareasLiteVisibles(items []tareaLite) []tareaLite {
	if len(items) == 0 {
		return nil
	}
	out := make([]tareaLite, 0, len(items))
	for _, item := range items {
		if !tareaLiteValida(item) {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func nombreAgenteCanonico(nombre string) string {
	return strings.ToLower(strings.TrimSpace(nombre))
}

func preferAgenteStatusCanonico(actual, candidato *db.Agente) *db.Agente {
	if actual == nil {
		return candidato
	}
	if candidato == nil {
		return actual
	}
	actualCanon, _ := db.CanonicalizeAgentName(actual.Nombre)
	candidatoCanon, _ := db.CanonicalizeAgentName(candidato.Nombre)
	actualEsCanonico := strings.TrimSpace(actualCanon) != "" && strings.TrimSpace(actualCanon) == strings.TrimSpace(actual.Nombre)
	candidatoEsCanonico := strings.TrimSpace(candidatoCanon) != "" && strings.TrimSpace(candidatoCanon) == strings.TrimSpace(candidato.Nombre)
	switch {
	case candidatoEsCanonico && !actualEsCanonico:
		return candidato
	case actualEsCanonico && !candidatoEsCanonico:
		return actual
	case candidato.Activo && !actual.Activo:
		return candidato
	case candidato.Habilitado && !actual.Habilitado:
		return candidato
	case strings.TrimSpace(candidato.EstadoSesion) != "" && strings.TrimSpace(actual.EstadoSesion) == "":
		return candidato
	case candidato.UltimaSesion != nil && (actual.UltimaSesion == nil || candidato.UltimaSesion.After(*actual.UltimaSesion)):
		return candidato
	default:
		return actual
	}
}

func snapshotExponeHabilitado(agentes []*db.Agente) bool {
	for _, agente := range agentes {
		if agente != nil && agente.Habilitado {
			return true
		}
	}
	return false
}

func agenteCuentaComoHabilitadoEnSnapshot(agente *db.Agente, snapshotConHabilitado bool) bool {
	if agente == nil {
		return false
	}
	if !snapshotConHabilitado {
		return true
	}
	return agente.Habilitado
}

func tareaAgenteInternoSiempreVisible(nombre string) bool {
	return nombreAgenteCanonico(nombre) == "orquesta"
}

func filtrarTareasActivasVisibles(tareas []tareaLite, agentes []*db.Agente) []tareaLite {
	tareas = normalizarTareasLiteVisibles(tareas)
	if len(tareas) == 0 {
		return nil
	}
	agentesPorNombre := make(map[string]*db.Agente, len(agentes))
	snapshotConHabilitado := snapshotExponeHabilitado(agentes)
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" {
			continue
		}
		agentesPorNombre[nombre] = agente
	}
	out := make([]tareaLite, 0, len(tareas))
	for _, tarea := range tareas {
		if !tareaLiteValida(tarea) {
			continue
		}
		nombre := nombreAgenteCanonico(tarea.Agente)
		if nombre == "" || tareaAgenteInternoSiempreVisible(nombre) {
			out = append(out, tarea)
			continue
		}
		if !agenteVisibleEnStatusFleet(nombre) {
			continue
		}
		agente := agentesPorNombre[nombre]
		if agente != nil && !agenteCuentaComoHabilitadoEnSnapshot(agente, snapshotConHabilitado) {
			continue
		}
		out = append(out, tarea)
	}
	return out
}

func tareaLiteValida(tarea tareaLite) bool {
	if tarea.ID <= 0 {
		return false
	}
	switch tarea.Estado {
	case db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada, db.TareaBacklog, db.TareaCompletada, db.TareaCancelada, db.TareaLibre:
		return true
	default:
		return false
	}
}

func agenteVisibleEnStatusFleet(nombre string) bool {
	nombre = nombreAgenteCanonico(nombre)
	if nombre == "" {
		return false
	}
	if tareaAgenteInternoSiempreVisible(nombre) {
		return true
	}
	return perteneceAFlotaOficialAutobootstrap(nombre)
}
