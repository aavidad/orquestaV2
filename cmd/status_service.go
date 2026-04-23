package cmd

import (
	"errors"
	"strconv"
	"strings"
	"sync"
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

var (
	statusSnapshotTTL                   = time.Minute
	statusFallbackTTL                   = 5 * time.Second
	statusSoftInvalidateTTL             = 5 * time.Second
	statusFailureBackoffTTL             = 2 * time.Second
	statusFreshTimeout                  = 1500 * time.Millisecond
	statusFastTimeout                   = 750 * time.Millisecond
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
	statusVotesSummaryFetcher     = db.ResumenVotosPorPropuestas
	statusPoolsFetcher            = listarPoolsLocalesCompartidosEstado
	statusDispatchDebtFetcher     = listarDeudaDispatchEstado
	statusHandoffsFetcher         = listarHandoffsAutonomiaEstado
	statusDispatchSummaryFetcher  = listarResumenDispatchYHandoffsEstado
	statusIncludeProposalSections = true
	statusDispatchFailureWindow   = 2 * time.Hour
	statusConfigGet               = db.ConfigGet
	statusNowFunc                 = time.Now
	statusAsyncRefresh            = true
	statusCacheState              struct {
		mu         sync.Mutex
		value      apiStatusResponse
		expires    time.Time
		ok         bool
		retryAfter time.Time
		refreshing bool
		waitCh     chan struct{}
	}
)

func statusRowsForSnapshot(timeout time.Duration) ([]agentesapp.Row, error) {
	if status, ok := readStatusSnapshotAny(); ok && statusSnapshotCanStayLight(status) {
		return buildAgentPanelRowsFromStatusSnapshot(status), nil
	}
	if rows, ok := readAgentPanelSnapshotFresh(); ok {
		return rows, nil
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
	if len(autonomia.supervisorNames) > 0 {
		workersActivos = len(filterVisibleWorkersByAutonomy(agentesActivos, autonomia))
		workersTrabajando = len(filterVisibleWorkersByAutonomy(agentesTrabajando, autonomia))
	}
	supervisoresActivos := autonomia.Supervisando
	if supervisoresActivos < 0 {
		supervisoresActivos = 0
	}
	return workersActivos, workersTrabajando, supervisoresActivos
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
	Supervisando         int `json:"supervising"`
	Continuando          int `json:"continuing"`
	ContinuidadPendiente int `json:"continuity_pending"`
	WorkConfirmed        int `json:"work_confirmed"`
	Handoffs             int `json:"handoffing"`
	supervisorNames      map[string]struct{}
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
	type result struct {
		rows []agentesapp.Row
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		rows, err := statusRowsFetcher()
		ch <- result{rows: rows, err: err}
	}()
	select {
	case res := <-ch:
		return res.rows, res.err
	case <-time.After(timeout):
		return nil, errStatusFetchTimeout
	}
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
			agentesTrabajando = append(agentesTrabajando, agente)
		case "saturado":
			agentesActivos = append(agentesActivos, agente)
			agentesTrabajando = append(agentesTrabajando, agente)
			agentesSaturados = append(agentesSaturados, agente)
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
			if waitStatusRefresh(waitCh, 0) {
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
		if waitStatusRefresh(waitCh, 0) {
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
	if !statusCacheState.ok {
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
	if !statusCacheState.ok {
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
	if status.Autonomia.ContinuidadPendiente > 0 || status.Autonomia.Handoffs > 0 {
		return true
	}
	if len(status.AgentesActivos) > 0 {
		return false
	}
	if len(status.AgentesQuotaBlocked) > 0 && len(status.TareasEnProgreso) > 0 {
		return false
	}
	if len(status.TareasEnProgreso) > 0 {
		return true
	}
	for _, tarea := range status.TareasActivas {
		if tarea.Estado == db.TareaEnProgreso {
			return true
		}
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
	statusCacheState.retryAfter = time.Time{}
	statusCacheState.refreshing = false
	statusCacheState.waitCh = nil
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
