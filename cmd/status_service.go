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
	statusSnapshotTTL           = time.Minute
	statusFallbackTTL           = 5 * time.Second
	statusFreshTimeout          = 1500 * time.Millisecond
	statusFastTimeout           = 750 * time.Millisecond
	statusRowsTimeout           = 200 * time.Millisecond
	statusFreshFetcher          = fetchStatusFresh
	statusFastFetcher           = fetchStatusFastFallback
	statusRowsFetcher           = agentRowsForStatus
	statusRuntimeHandlesFetcher = db.ListarRuntimeHandles
	statusDispatchFailureWindow = 2 * time.Hour
	statusConfigGet             = db.ConfigGet
	statusNowFunc               = time.Now
	statusAsyncRefresh          = true
	statusCacheState            struct {
		mu         sync.Mutex
		value      apiStatusResponse
		expires    time.Time
		ok         bool
		refreshing bool
		waitCh     chan struct{}
	}
)

type dbStatusService struct{}

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
	out := deudaDispatchResumen{}
	var allHandles []*db.RuntimeHandle
	var handlesLoaded bool
	now := statusNowFunc().UTC()
	estados := []string{"pendiente", "ejecutando", "fallida"}
	for _, estado := range estados {
		estadoFiltro := estado
		orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Estado: &estadoFiltro, Limit: 500})
		if err != nil {
			return out, err
		}
		for _, order := range orders {
			if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" {
				continue
			}
			payload := mapFromJSON(order.PayloadJSON)
			if int64FromStatusAny(payload["mailbox_id"]) <= 0 {
				continue
			}
			if absorbed, _, err := db.RuntimeOrderSendInstructionAbsorbidaPorTrabajoVivo(order, now); err != nil {
				return out, err
			} else if absorbed {
				out.WorkConfirmed++
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
				out.WorkConfirmed++
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
					out.WorkConfirmed++
				} else {
					out.Pendientes++
				}
			case strings.TrimSpace(order.Estado) == "fallida" || dispatchState == "failed" || deliveryState == "failed":
				if dispatchOrderFailureCountsAsDebt(order, now) {
					out.Fallidas++
				}
			case dispatchState == "notified" || deliveryState == "notified":
				out.Notificadas++
			default:
				out.Pendientes++
			}
		}
	}
	return normalizeDispatchDebtTotal(out), nil
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
	supervisor := statusSupervisorAgentName()
	for _, row := range rows {
		if !rowCuentaComoAutonomiaActiva(row) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row.LastAutonomySource), "assignment_handoff") &&
			(row.OpenTasks > 0 || row.SupervisorRoleActive(now)) &&
			row.WorkerFresh(now) {
			out.Handoffs++
		}
		switch {
		case supervisor != "" && row.SupervisorRoleActive(now) && row.Agente != nil && strings.EqualFold(strings.TrimSpace(row.Agente.Nombre), supervisor):
			out.Supervisando++
		case strings.TrimSpace(row.LastAutonomyAction) == "continuar_trabajo":
			out.Continuando++
		}
		if strings.EqualFold(strings.TrimSpace(row.LastAutonomyState), "work_confirmed") {
			out.WorkConfirmed++
		}
		if row.EffectiveContinuityPending(now) {
			out.ContinuidadPendiente++
		}
	}
	return out
}

func resumirAutonomiaLigera(agentesActivos, agentesTrabajando []*db.Agente, tareasEnProgreso []tareaLite, now time.Time) autonomiaResumen {
	out := autonomiaResumen{}
	supervisor := strings.ToLower(strings.TrimSpace(statusSupervisorAgentName()))
	activos := map[string]*db.Agente{}
	trabajando := map[string]*db.Agente{}
	for _, agente := range agentesActivos {
		if agente == nil {
			continue
		}
		nombre := strings.ToLower(strings.TrimSpace(agente.Nombre))
		if nombre == "" {
			continue
		}
		activos[nombre] = agente
	}
	for _, agente := range agentesTrabajando {
		if agente == nil {
			continue
		}
		nombre := strings.ToLower(strings.TrimSpace(agente.Nombre))
		if nombre == "" {
			continue
		}
		if activo := activos[nombre]; activo != nil {
			trabajando[nombre] = activo
		}
	}
	for _, tarea := range tareasEnProgreso {
		nombre := strings.ToLower(strings.TrimSpace(tarea.Agente))
		if nombre == "" {
			continue
		}
		if activo := activos[nombre]; activo != nil {
			trabajando[nombre] = activo
		}
	}
	if supervisor != "" {
		if _, ok := activos[supervisor]; ok {
			out.Supervisando = 1
		}
	}
	for nombre := range trabajando {
		if supervisor != "" && nombre == supervisor {
			continue
		}
		out.Continuando++
		out.WorkConfirmed++
	}
	if supervisor != "" {
		if _, ok := trabajando[supervisor]; ok {
			out.WorkConfirmed++
		}
	}
	return out
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
	total := 0
	seen := map[int64]struct{}{}
	for _, estado := range []string{"pendiente", "ejecutando"} {
		estadoFiltro := estado
		orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Estado: &estadoFiltro, Limit: 500})
		if err != nil {
			return 0, err
		}
		for _, order := range orders {
			if order == nil || strings.TrimSpace(order.Tipo) != "handoff" {
				continue
			}
			if _, ok := seen[order.ID]; ok {
				continue
			}
			seen[order.ID] = struct{}{}
			total++
		}
	}
	return total, nil
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
	return agenteTieneActividadRecienteVisible(agente)
}

func agentRowsForStatus() ([]agentesapp.Row, error) {
	return agentesService.BuildPanelRows()
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
	rowPorNombre := make(map[string]agentesapp.Row, len(rows))
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		rowPorNombre[strings.ToLower(strings.TrimSpace(row.Agente.Nombre))] = row
	}
	agentesActivos := make([]*db.Agente, 0, len(agentes))
	agentesTrabajando := make([]*db.Agente, 0, len(agentes))
	agentesSaturados := make([]*db.Agente, 0, len(agentes))
	agentesAtascados := make([]*db.Agente, 0, len(agentes))
	agentesAuthManual := make([]*db.Agente, 0, len(agentes))
	agentesQuotaBlocked := make([]*db.Agente, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		if !agenteVisibleEnStatusFleet(agente.Nombre) {
			continue
		}
		row, ok := rowPorNombre[strings.ToLower(strings.TrimSpace(agente.Nombre))]
		if !ok {
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

func aplicarVisibilidadOperativaAgentes(agentes []*db.Agente, rows []agentesapp.Row) {
	rowPorNombre := make(map[string]agentesapp.Row, len(rows))
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		rowPorNombre[strings.ToLower(strings.TrimSpace(row.Agente.Nombre))] = row
	}
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		row, ok := rowPorNombre[strings.ToLower(strings.TrimSpace(agente.Nombre))]
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
		waitCh := statusCacheState.waitCh
		needsRefresh := statusSnapshotNeedsImmediateRefresh(value)
		if ok && now.Before(expires) && !needsRefresh {
			statusCacheState.mu.Unlock()
			return value, nil
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
			if waitStatusRefresh(waitCh, statusFreshTimeout) {
				continue
			}
			if ok {
				return value, nil
			}
			now = statusNowFunc().UTC()
			if status, fallbackErr := runStatusFetcherWithTimeout(statusFastFetcher, statusFastTimeout); fallbackErr == nil {
				storeStatusSnapshotWithTTL(status, now, statusFallbackTTL)
				return status, nil
			}
			return apiStatusResponse{}, errStatusFetchTimeout
		}
		waitCh = startStatusRefreshLocked()
		statusCacheState.mu.Unlock()

		status, err := runStatusFetcherWithTimeout(statusFreshFetcher, statusFreshTimeout)
		if err == nil {
			storeStatusSnapshotAndFinish(waitCh, status, now, statusSnapshotTTL)
			return status, nil
		}
		if status, fallbackErr := runStatusFetcherWithTimeout(statusFastFetcher, statusFastTimeout); fallbackErr == nil {
			storeStatusSnapshotAndFinish(waitCh, status, now, statusFallbackTTL)
			return status, nil
		}
		finishStatusRefresh(waitCh, apiStatusResponse{}, time.Time{}, 0, false)
		return apiStatusResponse{}, err
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
	statusCacheState.refreshing = false
	statusCacheState.waitCh = nil
}

func statusSnapshotNeedsImmediateRefresh(status apiStatusResponse) bool {
	if len(status.AgentesAuthManual) > 0 {
		return true
	}
	if status.Autonomia.ContinuidadPendiente > 0 || status.Autonomia.Handoffs > 0 {
		return true
	}
	if len(status.AgentesActivos) > 0 {
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

func fetchStatusFastFallback() (apiStatusResponse, error) {
	agentes, err := db.ListarAgentesEstadoLigero()
	if err != nil {
		return apiStatusResponse{}, err
	}
	cuentas, err := db.ContarTareasPorEstado()
	if err != nil {
		return apiStatusResponse{}, err
	}
	openState := db.PropuestaAbierta
	abiertas, err := db.ListarPropuestas(&openState, nil)
	if err != nil {
		return apiStatusResponse{}, err
	}
	propuestaIDs := make([]int64, 0, len(abiertas))
	for _, propuesta := range abiertas {
		if propuesta != nil {
			propuestaIDs = append(propuestaIDs, propuesta.ID)
		}
	}
	votosPorPropuesta, err := db.ResumenVotosPorPropuestas(propuestaIDs)
	if err != nil {
		return apiStatusResponse{}, err
	}
	propuestasResumen := make([]propuestaLite, 0, len(abiertas))
	for _, propuesta := range abiertas {
		if propuesta == nil {
			continue
		}
		votos := votosPorPropuesta[propuesta.ID]
		propuesta.Votos = votos
		acuerdo, desacuerdo, abstencion, pendiente := 0, 0, 0, 0
		for _, voto := range votos {
			if voto == nil {
				continue
			}
			switch voto.Posicion {
			case db.VotoAcuerdo:
				acuerdo++
			case db.VotoDesacuerdo:
				desacuerdo++
			case db.VotoAbstencion:
				abstencion++
			default:
				pendiente++
			}
		}
		propuestasResumen = append(propuestasResumen, propuestaLite{
			ID:           propuesta.ID,
			Codigo:       propuesta.Codigo,
			Titulo:       propuesta.Titulo,
			Estado:       propuesta.Estado,
			PropuestoPor: propuesta.PropuestoPor,
			Acuerdo:      acuerdo,
			Desacuerdo:   desacuerdo,
			Abstencion:   abstencion,
			Pendiente:    pendiente,
		})
	}
	tareasActivas, err := listarTareasActivasRapido()
	if err != nil {
		return apiStatusResponse{}, err
	}
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
	poolsLocales, _ := listarPoolsLocalesCompartidosEstado()
	deudaDispatch, _ := listarDeudaDispatchEstado()
	autonomia := autonomiaResumen{}
	if rows, err := agentRowsForStatusWithinTimeout(statusRowsTimeout); err == nil {
		aplicarVisibilidadOperativaAgentes(agentes, rows)
		agentesActivos, agentesTrabajando, agentesSaturados, agentesAtascados, agentesAuthManual, agentesQuotaBlocked, _ = agentesVisiblesPorEstadoOperativoRows(agentes, rows)
		autonomia = resumirAutonomiaRows(rows, statusNowFunc().UTC())
	} else {
		agentesQuotaBlocked = agentesNoActivosConCuotaConResumen(agentes, nil)
		autonomia = resumirAutonomiaLigera(agentesActivos, agentesTrabajando, tareasEnProgreso, statusNowFunc().UTC())
	}
	if handoffs, err := listarHandoffsAutonomiaEstado(); err == nil && handoffs > autonomia.Handoffs {
		autonomia.Handoffs = handoffs
	}
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
		PropuestasResumen:   propuestasResumen,
		TareasActivas:       tareasActivas,
		TareasEnProgreso:    tareasEnProgreso,
		TareasReservadas:    tareasReservadas,
		PropuestasAbiertas:  abiertas,
		PoolsLocales:        poolsLocales,
		DeudaDispatch:       deudaDispatch,
		Autonomia:           autonomia,
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
			if tarea == nil || tarea.ProyectoID == nil {
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
	return out, nil
}

func fetchStatusFresh() (apiStatusResponse, error) {
	sesiones, err := sesionesVisiblesParaEstado()
	if err != nil {
		return apiStatusResponse{}, err
	}
	agentes, err := db.ListarAgentesEstadoLigeroConSesionesActivas(sesiones)
	if err != nil {
		return apiStatusResponse{}, err
	}
	cuentas, err := db.ContarTareasPorEstado()
	if err != nil {
		return apiStatusResponse{}, err
	}
	proyectos, err := db.ListarProyectosConRutaEfectiva(db.FiltroProyectos{}, "")
	if err != nil {
		return apiStatusResponse{}, err
	}
	asignaciones, err := db.ContarAsignacionesActivasPorProyecto()
	if err != nil {
		return apiStatusResponse{}, err
	}
	sesionesPorProyecto := map[int64]int{}
	for _, sesion := range sesiones {
		if sesion.ProyectoID != nil {
			sesionesPorProyecto[*sesion.ProyectoID]++
		}
	}
	estadoAbierta := db.PropuestaAbierta
	abiertas, err := db.ListarPropuestas(&estadoAbierta, nil)
	if err != nil {
		return apiStatusResponse{}, err
	}
	propuestaIDs := make([]int64, 0, len(abiertas))
	for _, propuesta := range abiertas {
		if propuesta == nil {
			continue
		}
		propuestaIDs = append(propuestaIDs, propuesta.ID)
	}
	votosPorPropuesta, err := db.ResumenVotosPorPropuestas(propuestaIDs)
	if err != nil {
		return apiStatusResponse{}, err
	}
	propuestasResumen := make([]propuestaLite, 0, len(abiertas))
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
			trabajandoNombres[strings.TrimSpace(tarea.Agente)] = true
		}
	}
	agentesActivos := make([]*db.Agente, 0, len(agentes))
	agentesPorNombre := make(map[string]*db.Agente, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		agentesPorNombre[agente.Nombre] = agente
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
		trabajandoPorNombre[agente.Nombre] = agente
	}
	agentesTrabajando := make([]*db.Agente, 0, len(trabajandoPorNombre))
	agentesSaturados := make([]*db.Agente, 0)
	agentesAtascados := make([]*db.Agente, 0)
	agentesAuthManual := make([]*db.Agente, 0)
	agentesQuotaBlocked := make([]*db.Agente, 0)
	poolsLocales, _ := listarPoolsLocalesCompartidosEstado()
	deudaDispatch, _ := listarDeudaDispatchEstado()
	autonomia := autonomiaResumen{}
	for _, agente := range agentesActivos {
		if agente == nil {
			continue
		}
		if trabajandoPorNombre[agente.Nombre] != nil {
			agentesTrabajando = append(agentesTrabajando, agente)
		}
	}
	if rows, rowsErr := agentRowsForStatusWithinTimeout(statusRowsTimeout); rowsErr == nil {
		aplicarVisibilidadOperativaAgentes(agentes, rows)
		if activosOperativos, trabajandoOperativos, saturadosOperativos, atascadosOperativos, authManualOperativos, quotaBlockedOperativos, ok := agentesVisiblesPorEstadoOperativoRows(agentes, rows); ok {
			agentesActivos = activosOperativos
			agentesTrabajando = trabajandoOperativos
			agentesSaturados = saturadosOperativos
			agentesAtascados = atascadosOperativos
			agentesAuthManual = authManualOperativos
			agentesQuotaBlocked = quotaBlockedOperativos
		}
		autonomia = resumirAutonomiaRows(rows, statusNowFunc().UTC())
	} else if activosOperativos, trabajandoOperativos, ok := agentesVisiblesPorEstadoOperativo(agentes); ok {
		agentesActivos = activosOperativos
		agentesTrabajando = trabajandoOperativos
		agentesQuotaBlocked = agentesNoActivosConCuotaConResumen(agentes, nil)
		autonomia = resumirAutonomiaLigera(agentesActivos, agentesTrabajando, tareasEnProgreso, statusNowFunc().UTC())
	}
	if handoffs, err := listarHandoffsAutonomiaEstado(); err == nil && handoffs > autonomia.Handoffs {
		autonomia.Handoffs = handoffs
	}
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
		PoolsLocales:        poolsLocales,
		DeudaDispatch:       deudaDispatch,
		Autonomia:           autonomia,
	}, nil
}

func resetStatusSnapshotCache() {
	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	statusCacheState.value = apiStatusResponse{}
	statusCacheState.expires = time.Time{}
	statusCacheState.ok = false
	statusCacheState.refreshing = false
	statusCacheState.waitCh = nil
	if agentesService != nil {
		agentesService.InvalidateCompactDetailCache()
	}
}

func reconciliarConteoTareasActivasVisible(cuentas map[string]int, tareasActivas []tareaLite) map[string]int {
	if cuentas == nil {
		cuentas = map[string]int{}
	}
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

func nombreAgenteCanonico(nombre string) string {
	return strings.ToLower(strings.TrimSpace(nombre))
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
