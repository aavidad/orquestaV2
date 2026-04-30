package cmd

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

var (
	agentPanelSnapshotTTL        = 5 * time.Second
	agentPanelSnapshotStaleTTL   = 20 * time.Second
	agentPanelSnapshotBackoffTTL = 2 * time.Second
	agentPanelSnapshotWarmTicker = 20 * time.Second
	agentPanelSnapshotWarmDelay  = 3 * time.Second
	agentPanelSnapshotWarmTimeo  = 1500 * time.Millisecond
	agentPanelSnapshotState      struct {
		mu         sync.Mutex
		rows       []agentesapp.Row
		expires    time.Time
		staleUntil time.Time
		ok         bool
		refreshing bool
		retryAfter time.Time
	}
)

func cloneAgentPanelRows(rows []agentesapp.Row) []agentesapp.Row {
	if len(rows) == 0 {
		return nil
	}
	out := make([]agentesapp.Row, len(rows))
	copy(out, rows)
	return out
}

func readAgentPanelSnapshotAny() ([]agentesapp.Row, bool) {
	agentPanelSnapshotState.mu.Lock()
	defer agentPanelSnapshotState.mu.Unlock()
	if !agentPanelSnapshotState.ok {
		return nil, false
	}
	if !agentPanelSnapshotState.staleUntil.IsZero() && !statusNowFunc().UTC().Before(agentPanelSnapshotState.staleUntil) {
		return nil, false
	}
	return cloneAgentPanelRows(agentPanelSnapshotState.rows), true
}

func readAgentPanelSnapshotFresh() ([]agentesapp.Row, bool) {
	agentPanelSnapshotState.mu.Lock()
	defer agentPanelSnapshotState.mu.Unlock()
	if !agentPanelSnapshotState.ok {
		return nil, false
	}
	if agentPanelSnapshotState.expires.IsZero() || !statusNowFunc().UTC().Before(agentPanelSnapshotState.expires) {
		return nil, false
	}
	return cloneAgentPanelRows(agentPanelSnapshotState.rows), true
}

func storeAgentPanelSnapshot(rows []agentesapp.Row, now time.Time) {
	storeAgentPanelSnapshotWithTTL(rows, now, agentPanelSnapshotTTL, agentPanelSnapshotStaleTTL)
}

func storeAgentPanelSnapshotWithTTL(rows []agentesapp.Row, now time.Time, freshTTL, staleTTL time.Duration) {
	agentPanelSnapshotState.mu.Lock()
	defer agentPanelSnapshotState.mu.Unlock()
	agentPanelSnapshotState.rows = cloneAgentPanelRows(rows)
	if freshTTL > 0 {
		agentPanelSnapshotState.expires = now.Add(freshTTL)
	} else {
		agentPanelSnapshotState.expires = now
	}
	if staleTTL > 0 {
		agentPanelSnapshotState.staleUntil = now.Add(staleTTL)
	} else {
		agentPanelSnapshotState.staleUntil = now
	}
	agentPanelSnapshotState.ok = true
	agentPanelSnapshotState.refreshing = false
	agentPanelSnapshotState.retryAfter = time.Time{}
}

func markAgentPanelSnapshotRefreshFailure(now time.Time) {
	agentPanelSnapshotState.mu.Lock()
	defer agentPanelSnapshotState.mu.Unlock()
	agentPanelSnapshotState.refreshing = false
	agentPanelSnapshotState.retryAfter = now.Add(agentPanelSnapshotBackoffTTL)
}

func resetAgentPanelSnapshotCache() {
	agentPanelSnapshotState.mu.Lock()
	defer agentPanelSnapshotState.mu.Unlock()
	agentPanelSnapshotState.rows = nil
	agentPanelSnapshotState.expires = time.Time{}
	agentPanelSnapshotState.staleUntil = time.Time{}
	agentPanelSnapshotState.ok = false
	agentPanelSnapshotState.refreshing = false
	agentPanelSnapshotState.retryAfter = time.Time{}
}

func startAgentPanelSnapshotRefreshLocked(now time.Time) bool {
	if agentPanelSnapshotState.refreshing {
		return false
	}
	if agentPanelSnapshotState.retryAfter.After(now) {
		return false
	}
	agentPanelSnapshotState.refreshing = true
	return true
}

func maybeRefreshAgentPanelSnapshotAsync(timeout time.Duration) {
	now := time.Now().UTC()
	agentPanelSnapshotState.mu.Lock()
	start := startAgentPanelSnapshotRefreshLocked(now)
	agentPanelSnapshotState.mu.Unlock()
	if !start {
		return
	}
	go func() {
		rows, err := runAPITimeboxed(timeout, apiAgentPanelRowsBuilder, errStatusFetchTimeout)
		if err != nil {
			markAgentPanelSnapshotRefreshFailure(time.Now().UTC())
			return
		}
		storeAgentPanelSnapshot(rows, time.Now().UTC())
	}()
}

func agentPanelRowsNeedImmediateRefresh(rows []agentesapp.Row) bool {
	for _, row := range rows {
		estado := strings.ToLower(strings.TrimSpace(row.EstadoOperativo))
		needsLiveEvidence := row.OpenTasks > 0
		switch estado {
		case "trabajando", "saturado", "atascado", "mailbox_atascada", "bloqueado_por_runtime":
			needsLiveEvidence = true
		}
		if !needsLiveEvidence {
			continue
		}
		if row.Runtime != nil || row.Handle != nil || strings.TrimSpace(row.WorkerState) != "" || row.WorkerAlive ||
			row.WorkerHeartbeat != nil || row.WorkerUpdatedAt != nil || strings.TrimSpace(row.WorkerTMUXSession) != "" {
			continue
		}
		if strings.TrimSpace(row.DetalleOperativo) != "" {
			continue
		}
		return true
	}
	return false
}

func tryBuildAgentPanelRowsNow(timeout time.Duration) ([]agentesapp.Row, bool) {
	if apiAgentPanelRowsBuilder == nil {
		return nil, false
	}
	rows, err := runAPITimeboxed(timeout, apiAgentPanelRowsBuilder, errStatusFetchTimeout)
	if err != nil {
		return nil, false
	}
	storeAgentPanelSnapshot(rows, time.Now().UTC())
	return rows, true
}

func fetchAgentPanelRowsCached(timeout time.Duration) ([]agentesapp.Row, error) {
	if rows, ok := readAgentPanelSnapshotFresh(); ok {
		if agentPanelRowsNeedImmediateRefresh(rows) {
			if refreshed, ok := tryBuildAgentPanelRowsNow(timeout); ok {
				return refreshed, nil
			}
		}
		return rows, nil
	}
	if rows, ok := readAgentPanelSnapshotAny(); ok {
		if agentPanelRowsNeedImmediateRefresh(rows) {
			if refreshed, ok := tryBuildAgentPanelRowsNow(timeout); ok {
				return refreshed, nil
			}
		}
		maybeRefreshAgentPanelSnapshotAsync(timeout)
		return rows, nil
	}
	if status, ok := readStatusSnapshotAny(); ok {
		rows := buildAgentPanelRowsFromStatusSnapshot(status)
		if agentPanelRowsNeedImmediateRefresh(rows) {
			if refreshed, ok := tryBuildAgentPanelRowsNow(timeout); ok {
				return refreshed, nil
			}
		}
		storeAgentPanelSnapshotWithTTL(rows, time.Now().UTC(), 0, agentPanelSnapshotStaleTTL)
		return rows, nil
	}
	if apiStatusUltraLiteFetcher != nil {
		if status, ok := apiStatusUltraLiteFetcher(statusFastTimeout); ok {
			now := time.Now().UTC()
			storeStatusSnapshotWithTTL(status, now, statusFallbackTTL)
			rows := buildAgentPanelRowsFromStatusSnapshot(status)
			if agentPanelRowsNeedImmediateRefresh(rows) {
				if refreshed, ok := tryBuildAgentPanelRowsNow(timeout); ok {
					return refreshed, nil
				}
			}
			storeAgentPanelSnapshotWithTTL(rows, now, 0, agentPanelSnapshotStaleTTL)
			return rows, nil
		}
	}
	now := time.Now().UTC()
	rows := buildAgentPanelRowsFromStatusSnapshot(degradedAPIStatusResponse())
	storeAgentPanelSnapshotWithTTL(rows, now, 0, agentPanelSnapshotStaleTTL)
	markAgentPanelSnapshotRefreshFailure(now)
	return rows, nil
}

func fetchAgentPanelRowsReadOnlyCached(timeout time.Duration) ([]agentesapp.Row, error) {
	if rows, ok := readAgentPanelSnapshotFresh(); ok {
		if agentPanelRowsNeedImmediateRefresh(rows) {
			maybeRefreshAgentPanelSnapshotAsync(timeout)
		}
		return rows, nil
	}
	if rows, ok := readAgentPanelSnapshotAny(); ok {
		maybeRefreshAgentPanelSnapshotAsync(timeout)
		return rows, nil
	}
	if status, ok := readStatusSnapshotAny(); ok {
		rows := buildAgentPanelRowsFromStatusSnapshot(status)
		storeAgentPanelSnapshotWithTTL(rows, time.Now().UTC(), 0, agentPanelSnapshotStaleTTL)
		if agentPanelRowsNeedImmediateRefresh(rows) {
			maybeRefreshAgentPanelSnapshotAsync(timeout)
		}
		return rows, nil
	}
	if apiStatusUltraLiteFetcher != nil {
		if status, ok := apiStatusUltraLiteFetcher(statusFastTimeout); ok {
			now := time.Now().UTC()
			storeStatusSnapshotWithTTL(status, now, statusFallbackTTL)
			rows := buildAgentPanelRowsFromStatusSnapshot(status)
			storeAgentPanelSnapshotWithTTL(rows, now, 0, agentPanelSnapshotStaleTTL)
			if agentPanelRowsNeedImmediateRefresh(rows) {
				maybeRefreshAgentPanelSnapshotAsync(timeout)
			}
			return rows, nil
		}
	}
	now := time.Now().UTC()
	rows := buildAgentPanelRowsFromStatusSnapshot(degradedAPIStatusResponse())
	storeAgentPanelSnapshotWithTTL(rows, now, 0, agentPanelSnapshotStaleTTL)
	markAgentPanelSnapshotRefreshFailure(now)
	return rows, nil
}

func launchAgentPanelSnapshotWarmLoop(ctx context.Context, debugLogger *log.Logger) {
	if ctx == nil || apiAgentPanelRowsBuilder == nil {
		return
	}
	go func() {
		timer := time.NewTimer(agentPanelSnapshotWarmDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		ticker := time.NewTicker(agentPanelSnapshotWarmTicker)
		defer ticker.Stop()
		refresh := func() {
			if status, ok := readStatusSnapshotAny(); ok && statusSnapshotCanStayLight(status) {
				return
			}
			if _, ok := readAgentPanelSnapshotFresh(); ok {
				return
			}
			if _, ok := readAgentPanelSnapshotAny(); !ok {
				return
			}
			start := time.Now()
			rows, err := runAPITimeboxed(agentPanelSnapshotWarmTimeo, apiAgentPanelRowsBuilder, errStatusFetchTimeout)
			if err != nil {
				markAgentPanelSnapshotRefreshFailure(time.Now().UTC())
				if debugLogger != nil {
					debugLogger.Printf("panel_warm err=%v duration=%s", err, time.Since(start).Round(time.Millisecond))
				}
				return
			}
			storeAgentPanelSnapshot(rows, time.Now().UTC())
			if debugLogger != nil {
				debugLogger.Printf("panel_warm ok rows=%d duration=%s", len(rows), time.Since(start).Round(time.Millisecond))
			}
		}
		refresh()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh()
			}
		}
	}()
}

func buildAgentPanelRowsFromStatusSnapshot(status apiStatusResponse) []agentesapp.Row {
	byName := map[string]*agentesapp.Row{}
	get := func(agente *db.Agente) *agentesapp.Row {
		if agente == nil {
			return nil
		}
		nombre := strings.TrimSpace(agente.Nombre)
		key := strings.ToLower(nombre)
		if key == "" {
			return nil
		}
		if existing, ok := byName[key]; ok {
			if existing.Agente == nil {
				existing.Agente = agente
			}
			return existing
		}
		row := &agentesapp.Row{Agente: agente, EstadoOperativo: "desconocido"}
		byName[key] = row
		return row
	}
	setState := func(items []*db.Agente, state string) {
		for _, agente := range items {
			row := get(agente)
			if row == nil {
				continue
			}
			row.EstadoOperativo = state
		}
	}
	for _, agente := range status.Agentes {
		_ = get(agente)
	}
	setState(status.AgentesActivos, "disponible")
	setState(status.AgentesTrabajando, "trabajando")
	setState(status.AgentesSaturados, "saturado")
	setState(status.AgentesAtascados, "atascado")
	setState(status.AgentesAuthManual, "bloqueado_por_runtime")
	setState(status.AgentesQuotaBlocked, "bloqueado_por_cuota")
	for _, tarea := range status.TareasEnProgreso {
		key := strings.ToLower(strings.TrimSpace(tarea.Agente))
		if key == "" {
			continue
		}
		row := byName[key]
		if row == nil {
			row = &agentesapp.Row{Agente: &db.Agente{Nombre: strings.TrimSpace(tarea.Agente)}, EstadoOperativo: "trabajando"}
			byName[key] = row
		}
		row.OpenTasks++
		row.CurrentTask = &agentesapp.TaskFocus{
			TaskID: tarea.ID,
			Title:  strings.TrimSpace(tarea.Titulo),
			State:  tarea.Estado,
			Module: strings.TrimSpace(tarea.Modulo),
		}
		if row.EstadoOperativo == "desconocido" || row.EstadoOperativo == "disponible" {
			row.EstadoOperativo = "trabajando"
		}
	}
	for _, tarea := range status.TareasReservadas {
		key := strings.ToLower(strings.TrimSpace(tarea.Agente))
		if key == "" {
			continue
		}
		row := byName[key]
		if row == nil {
			row = &agentesapp.Row{Agente: &db.Agente{Nombre: strings.TrimSpace(tarea.Agente)}, EstadoOperativo: "disponible"}
			byName[key] = row
		}
		row.OpenTasks++
		if row.CurrentTask == nil {
			row.CurrentTask = &agentesapp.TaskFocus{
				TaskID: tarea.ID,
				Title:  strings.TrimSpace(tarea.Titulo),
				State:  tarea.Estado,
				Module: strings.TrimSpace(tarea.Modulo),
			}
		}
	}
	keys := make([]string, 0, len(byName))
	for key := range byName {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]agentesapp.Row, 0, len(keys))
	for _, key := range keys {
		row := byName[key]
		if row == nil {
			continue
		}
		if row.Agente == nil {
			row.Agente = &db.Agente{Nombre: key}
		}
		out = append(out, *row)
	}
	return out
}
