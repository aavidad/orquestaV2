package cmd

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"orquesta/db"
	"orquesta/notificaciones"
)

const (
	supervisorReviewOpenGatesLimit     = 20
	supervisorReviewSignalsLimit       = 12
	supervisorReviewMergesLimit        = 12
	supervisorReviewOutboxLimit        = 20
	supervisorRevisionSnapshotTTL      = 5 * time.Second
	supervisorRevisionSnapshotStaleTTL = 30 * time.Second
)

var (
	supervisorReviewConflictsBuilder = listarSupervisorModuleConflictsSnapshotSafe
	supervisorReviewMailboxBuilder   = func(agentes []*db.Agente) ([]apiOpenClawMailboxLite, error) {
		return openClawPendingMailboxFetcher(agentes)
	}
	supervisorReviewOutboxBuilder = func(limit int) (notificaciones.OutboxSummary, error) {
		return notificaciones.DescribirOutbox(limit), nil
	}
	supervisorRevisionSnapshotBuilder = buildSupervisorRevisionSnapshot
)

type supervisorRevisionSnapshotEntry struct {
	Snapshot   map[string]any
	FreshUntil time.Time
	StaleUntil time.Time
}

var (
	supervisorRevisionSnapshotMu      sync.RWMutex
	supervisorRevisionSnapshotCache   = map[string]supervisorRevisionSnapshotEntry{}
	supervisorRevisionSnapshotRefresh sync.Map
)

type supervisorReviewAsyncResult[T any] struct {
	value T
	ok    bool
}

func supervisorReviewOptionalAsync[T any](timeout time.Duration, fn func() (T, error)) <-chan supervisorReviewAsyncResult[T] {
	ch := make(chan supervisorReviewAsyncResult[T], 1)
	go func() {
		var res supervisorReviewAsyncResult[T]
		if fn != nil {
			if value, err := runAPITimeboxed(timeout, fn, errStatusFetchTimeout); err == nil {
				res.value = value
				res.ok = true
			}
		}
		ch <- res
	}()
	return ch
}

func supervisorReviewAwait[T any](ch <-chan supervisorReviewAsyncResult[T]) (T, bool) {
	res := <-ch
	return res.value, res.ok
}

func supervisorReviewIntString[T ~int64 | ~int](v T) string {
	return strconv.FormatInt(int64(v), 10)
}

func listSupervisorReviewOverviewData() ([]*db.ReviewGate, []*supervisorReviewSignal, []*db.GitMerge, error) {
	gates, err := db.ListarReviewGates(db.FiltroReviewGates{Limit: supervisorReviewOpenGatesLimit})
	if err != nil {
		return nil, nil, nil, err
	}
	openGates := make([]*db.ReviewGate, 0, len(gates))
	for _, gate := range gates {
		if gate == nil || gate.Estado == db.ReviewGateAprobado {
			continue
		}
		openGates = append(openGates, gate)
	}
	signals, err := listarSignalsRevisionSupervisor(supervisorReviewSignalsLimit)
	if err != nil {
		return nil, nil, nil, err
	}
	merges, err := listarMergesRevisionSupervisor(supervisorReviewMergesLimit)
	if err != nil {
		return nil, nil, nil, err
	}
	return openGates, signals, merges, nil
}

func buildSupervisorReviewOverviewFromSnapshot(snapshot map[string]any) (string, error) {
	if len(snapshot) == 0 {
		return "", nil
	}
	openGates, _ := snapshot["review_gates"].([]*db.ReviewGate)
	signals, _ := snapshot["signals"].([]*supervisorReviewSignal)
	merges, _ := snapshot["merges"].([]*db.GitMerge)
	return buildSupervisorReviewOverviewFromData(openGates, signals, merges)
}

func buildSupervisorReviewOverviewFromData(openGates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge) (string, error) {
	if len(openGates) == 0 && len(signals) == 0 && len(merges) == 0 {
		return "", nil
	}
	taskTitles, projectSlugs := buildSupervisorReviewGateOverviewContext(openGates)

	var b strings.Builder
	if len(openGates) > 0 {
		b.WriteString("## Review gates abiertos\n")
		for _, gate := range openGates {
			linea := formatSupervisorReviewGateOverviewLine(gate, taskTitles, projectSlugs)
			if strings.TrimSpace(linea) == "" {
				continue
			}
			b.WriteString(linea + "\n")
		}
		b.WriteString("\n")
	}
	if len(signals) > 0 {
		b.WriteString("## Señales recientes de revisión e integración\n")
		for _, item := range signals {
			if item == nil || item.Event == nil {
				continue
			}
			linea := formatSupervisorReviewSignalOverviewLine(item)
			if strings.TrimSpace(linea) == "" {
				continue
			}
			b.WriteString(linea + "\n")
		}
		b.WriteString("\n")
	}
	if len(merges) > 0 {
		b.WriteString("## Solicitudes de merge vivas\n")
		for _, merge := range merges {
			if merge == nil {
				continue
			}
			linea := formatSupervisorReviewMergeOverviewLine(merge)
			if strings.TrimSpace(linea) == "" {
				continue
			}
			b.WriteString(linea + "\n")
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func buildSupervisorRevisionSnapshot(supervisor string) (map[string]any, error) {
	status := degradedAPIStatusResponse()
	switch {
	case func() bool {
		snapshot, ok := readStatusSnapshotFresh()
		if ok {
			status = snapshot
		}
		return ok
	}():
	case func() bool {
		snapshot, ok := readStatusSnapshotAny()
		if ok {
			status = snapshot
			ensureStatusRefreshAsync()
		}
		return ok
	}():
	case func() bool {
		snapshot, ok := fetchStatusReadOnlyLiteDirect(statusFastTimeout)
		if ok {
			status = snapshot
			storeStatusSnapshot(snapshot, statusNowFunc().UTC())
		}
		return ok
	}():
	default:
		ensureStatusRefreshAsync()
	}
	return buildSupervisorRevisionSnapshotWithStatus(supervisor, status)
}

func buildSupervisorRevisionReadSnapshot(supervisor string) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	start := time.Now()
	logSlow := func(source string) {
		if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
			log.Printf("orquesta[supervision-revision-slow] source=%s supervisor=%s elapsed=%s", source, supervisor, elapsed)
		}
	}
	if snapshot, ok := readSupervisorRevisionSnapshotFresh(supervisor); ok {
		logSlow("cache_fresh")
		return snapshot, nil
	}
	if snapshot, ok := readSupervisorRevisionSnapshotAny(supervisor); ok {
		ensureSupervisorRevisionSnapshotAsync(supervisor)
		logSlow("cache_any")
		return snapshot, nil
	}
	if status, ok := readStatusSnapshotFresh(); ok {
		snapshot := buildSupervisorRevisionFastSnapshotFromStatus(supervisor, status)
		storeSupervisorRevisionSnapshot(supervisor, snapshot, time.Now().UTC())
		ensureSupervisorRevisionSnapshotAsync(supervisor)
		logSlow("status_fresh")
		return snapshot, nil
	}
	if status, ok := readStatusSnapshotAny(); ok {
		snapshot := buildSupervisorRevisionFastSnapshotFromStatus(supervisor, status)
		storeSupervisorRevisionSnapshot(supervisor, snapshot, time.Now().UTC())
		ensureSupervisorRevisionSnapshotAsync(supervisor)
		logSlow("status_any")
		return snapshot, nil
	}
	if status, ok := fetchStatusReadOnlyLiteDirect(statusFastTimeout); ok {
		storeStatusSnapshot(status, statusNowFunc().UTC())
		snapshot := buildSupervisorRevisionFastSnapshotFromStatus(supervisor, status)
		storeSupervisorRevisionSnapshot(supervisor, snapshot, time.Now().UTC())
		ensureSupervisorRevisionSnapshotAsync(supervisor)
		logSlow("status_direct")
		return snapshot, nil
	}
	snapshot := buildSupervisorRevisionFastSnapshotFromStatus(supervisor, degradedAPIStatusResponse())
	storeSupervisorRevisionSnapshot(supervisor, snapshot, time.Now().UTC())
	ensureSupervisorRevisionSnapshotAsync(supervisor)
	logSlow("degraded")
	return snapshot, nil
}

func storeSupervisorRevisionSnapshot(supervisor string, snapshot map[string]any, now time.Time) {
	storeSupervisorRevisionSnapshotWithTTL(supervisor, snapshot, now, supervisorRevisionSnapshotTTL, supervisorRevisionSnapshotStaleTTL)
}

func storeSupervisorRevisionSnapshotWithTTL(supervisor string, snapshot map[string]any, now time.Time, freshTTL, staleTTL time.Duration) {
	supervisor = resolveSupervisorName(supervisor)
	if snapshot == nil {
		return
	}
	entry := supervisorRevisionSnapshotEntry{
		Snapshot: snapshot,
	}
	if freshTTL > 0 {
		entry.FreshUntil = now.Add(freshTTL)
	}
	if staleTTL > 0 {
		entry.StaleUntil = now.Add(staleTTL)
	}
	supervisorRevisionSnapshotMu.Lock()
	supervisorRevisionSnapshotCache[supervisor] = entry
	supervisorRevisionSnapshotMu.Unlock()
}

func readSupervisorRevisionSnapshotFresh(supervisor string) (map[string]any, bool) {
	supervisor = resolveSupervisorName(supervisor)
	supervisorRevisionSnapshotMu.RLock()
	entry, ok := supervisorRevisionSnapshotCache[supervisor]
	supervisorRevisionSnapshotMu.RUnlock()
	if !ok || entry.Snapshot == nil {
		return nil, false
	}
	if entry.FreshUntil.IsZero() || time.Now().UTC().After(entry.FreshUntil) {
		return nil, false
	}
	return entry.Snapshot, true
}

func readSupervisorRevisionSnapshotAny(supervisor string) (map[string]any, bool) {
	supervisor = resolveSupervisorName(supervisor)
	supervisorRevisionSnapshotMu.RLock()
	entry, ok := supervisorRevisionSnapshotCache[supervisor]
	supervisorRevisionSnapshotMu.RUnlock()
	if !ok || entry.Snapshot == nil {
		return nil, false
	}
	if !entry.StaleUntil.IsZero() && time.Now().UTC().After(entry.StaleUntil) {
		return nil, false
	}
	return entry.Snapshot, true
}

func ensureSupervisorRevisionSnapshotAsync(supervisor string) {
	supervisor = resolveSupervisorName(supervisor)
	if _, loaded := supervisorRevisionSnapshotRefresh.LoadOrStore(supervisor, struct{}{}); loaded {
		return
	}
	go func() {
		defer supervisorRevisionSnapshotRefresh.Delete(supervisor)
		snapshot, err := supervisorRevisionSnapshotBuilder(supervisor)
		if err != nil || snapshot == nil {
			return
		}
		storeSupervisorRevisionSnapshot(supervisor, snapshot, time.Now().UTC())
	}()
}

func resetSupervisorRevisionSnapshotCache() {
	supervisorRevisionSnapshotMu.Lock()
	supervisorRevisionSnapshotCache = map[string]supervisorRevisionSnapshotEntry{}
	supervisorRevisionSnapshotMu.Unlock()
	supervisorRevisionSnapshotRefresh = sync.Map{}
}

func buildSupervisorRevisionFastSnapshotFromStatus(supervisor string, status apiStatusResponse) map[string]any {
	supervisor = resolveSupervisorName(supervisor)
	mailboxPendiente := []apiOpenClawMailboxLite{}
	recommended := buildSupervisorOperationalActionsFast(status, mailboxPendiente)
	recommended = append(recommended, buildSupervisorProposalActions(status.PropuestasResumen)...)
	sortSupervisorRecommendedActionsFast(recommended)
	markSupervisorRecommendedActions(recommended)
	actionQueue := cloneSupervisorRecommendedActions(recommended)
	safeQueue := buildSupervisorSafeActionQueueFast(actionQueue)
	_, _, criticalProjectRisk := normalizeStatusAutonomyPayload(status.AutonomySurface, status.AutonomyHighlights, status.CriticalProjectRisk)
	var nextAction any
	if len(actionQueue) > 0 {
		nextAction = actionQueue[0]
	}
	var nextSafeAction any
	if len(safeQueue) > 0 {
		nextSafeAction = safeQueue[0]
	}
	capacitySummary := buildOpenClawCapacitySummary(status.AgentesActivos, status.AgentesTrabajando, status.TareasActivas, status.TareasPorEstado)
	saturatedAgents := buildOpenClawSaturatedAgents(status.AgentesActivos, status.TareasActivas, nil)
	queueSummary := buildOpenClawQueueSummaryFromActions(actionQueue, safeQueue)

	return map[string]any{
		"supervisor":            supervisor,
		"review_gates":          []*db.ReviewGate{},
		"signals":               []*supervisorReviewSignal{},
		"merges":                []*db.GitMerge{},
		"module_conflicts":      []supervisorModuleConflict{},
		"mailbox_pending":       mailboxPendiente,
		"normalized_events":     []openClawNormalizedEvent{},
		"recommended_actions":   recommended,
		"action_queue":          actionQueue,
		"next_action":           nextAction,
		"queue_kind":            "safe",
		"safe_action_queue":     safeQueue,
		"next_safe_action":      nextSafeAction,
		"critical_project_risk": criticalProjectRisk,
		"capacity_summary":      capacitySummary,
		"saturated_agents":      saturatedAgents,
		"queue_summary":         queueSummary,
	}
}

func buildSupervisorRevisionSnapshotWithStatus(supervisor string, status apiStatusResponse) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	gates, err := db.ListarReviewGates(db.FiltroReviewGates{Limit: supervisorReviewOpenGatesLimit})
	if err != nil {
		return nil, err
	}
	openGates := make([]*db.ReviewGate, 0, len(gates))
	for _, gate := range gates {
		if gate == nil || gate.Estado == db.ReviewGateAprobado {
			continue
		}
		openGates = append(openGates, gate)
	}

	signalsCh := supervisorReviewOptionalAsync(150*time.Millisecond, func() ([]*supervisorReviewSignal, error) {
		return supervisorReviewSignalsBuilder(supervisorReviewSignalsLimit)
	})
	mergesCh := supervisorReviewOptionalAsync(150*time.Millisecond, func() ([]*db.GitMerge, error) {
		return supervisorReviewMergesBuilder(supervisorReviewMergesLimit)
	})
	conflictsCh := supervisorReviewOptionalAsync(100*time.Millisecond, func() ([]supervisorModuleConflict, error) {
		return supervisorReviewConflictsBuilder(status.TareasActivas)
	})
	mailboxCh := supervisorReviewOptionalAsync(100*time.Millisecond, func() ([]apiOpenClawMailboxLite, error) {
		return supervisorReviewMailboxBuilder(status.Agentes)
	})
	outboxCh := supervisorReviewOptionalAsync(100*time.Millisecond, func() (notificaciones.OutboxSummary, error) {
		return supervisorReviewOutboxBuilder(supervisorReviewOutboxLimit)
	})

	signals := []*supervisorReviewSignal{}
	if snapshot, ok := supervisorReviewAwait(signalsCh); ok && snapshot != nil {
		signals = snapshot
	}
	merges := []*db.GitMerge{}
	if snapshot, ok := supervisorReviewAwait(mergesCh); ok && snapshot != nil {
		merges = snapshot
	}
	conflicts := []supervisorModuleConflict{}
	if snapshot, ok := supervisorReviewAwait(conflictsCh); ok && snapshot != nil {
		conflicts = snapshot
	}
	mailboxPendiente := []apiOpenClawMailboxLite{}
	if snapshot, ok := supervisorReviewAwait(mailboxCh); ok && snapshot != nil {
		mailboxPendiente = snapshot
	}
	outbox := notificaciones.OutboxSummary{}
	if snapshot, ok := supervisorReviewAwait(outboxCh); ok {
		outbox = snapshot
	}

	normalizedEvents := buildOpenClawNormalizedEventsFromData(openGates, signals, merges, outbox.Recientes, supervisorReviewOutboxLimit)
	recommended := buildSupervisorRecommendedActions(openGates, signals, merges, conflicts)
	recommended = append(recommended, buildSupervisorOperationalActionsFast(status, mailboxPendiente)...)
	recommended = append(recommended, buildSupervisorProposalActions(status.PropuestasResumen)...)
	sortSupervisorRecommendedActions(recommended)
	markSupervisorRecommendedActions(recommended)

	actionQueue := cloneSupervisorRecommendedActions(recommended)
	safeQueue := buildSupervisorSafeActionQueueFast(actionQueue)
	_, _, criticalProjectRisk := normalizeStatusAutonomyPayload(status.AutonomySurface, status.AutonomyHighlights, status.CriticalProjectRisk)
	var nextAction any
	if len(actionQueue) > 0 {
		nextAction = actionQueue[0]
	}
	var nextSafeAction any
	if len(safeQueue) > 0 {
		nextSafeAction = safeQueue[0]
	}
	capacitySummary := buildOpenClawCapacitySummary(status.AgentesActivos, status.AgentesTrabajando, status.TareasActivas, status.TareasPorEstado)
	saturatedAgents := buildOpenClawSaturatedAgents(status.AgentesActivos, status.TareasActivas, nil)
	queueSummary := buildOpenClawQueueSummaryFromActions(actionQueue, safeQueue)

	return map[string]any{
		"supervisor":            supervisor,
		"review_gates":          openGates,
		"signals":               signals,
		"merges":                merges,
		"module_conflicts":      conflicts,
		"mailbox_pending":       mailboxPendiente,
		"normalized_events":     normalizedEvents,
		"recommended_actions":   recommended,
		"action_queue":          actionQueue,
		"next_action":           nextAction,
		"queue_kind":            "safe",
		"safe_action_queue":     safeQueue,
		"next_safe_action":      nextSafeAction,
		"critical_project_risk": criticalProjectRisk,
		"capacity_summary":      capacitySummary,
		"saturated_agents":      saturatedAgents,
		"queue_summary":         queueSummary,
	}, nil
}

func buildSupervisorReviewGateOverviewContext(gates []*db.ReviewGate) (map[int64]string, map[int64]string) {
	taskTitles := make(map[int64]string, len(gates))
	projectSlugs := make(map[int64]string, len(gates))
	for _, gate := range gates {
		if gate == nil {
			continue
		}
		if gate.TareaID != nil && *gate.TareaID > 0 {
			taskID := *gate.TareaID
			if _, ok := taskTitles[taskID]; !ok {
				taskTitles[taskID] = ""
				if tarea, err := tareasService.Get(taskID); err == nil && tarea != nil {
					taskTitles[taskID] = strings.TrimSpace(tarea.Titulo)
				}
			}
		}
		if gate.ProyectoID != nil && *gate.ProyectoID > 0 {
			projectID := *gate.ProyectoID
			if _, ok := projectSlugs[projectID]; !ok {
				projectSlugs[projectID] = ""
				if proyecto, err := db.GetProyecto(strconv.FormatInt(projectID, 10)); err == nil && proyecto != nil {
					projectSlugs[projectID] = strings.TrimSpace(proyecto.Slug)
				}
			}
		}
	}
	return taskTitles, projectSlugs
}

func formatSupervisorReviewGateOverviewLine(gate *db.ReviewGate, taskTitles, projectSlugs map[int64]string) string {
	if gate == nil {
		return ""
	}
	linea := "- #" + supervisorReviewIntString(gate.ID) + " estado=" + strings.TrimSpace(string(gate.Estado))
	if gate.TareaID != nil && *gate.TareaID > 0 {
		linea += " tarea=" + supervisorReviewIntString(*gate.TareaID)
		if titulo := compactMCPLine(strings.TrimSpace(taskTitles[*gate.TareaID]), 80); titulo != "" {
			linea += " \"" + titulo + "\""
		}
	}
	if gate.ProyectoID != nil && *gate.ProyectoID > 0 {
		if slug := strings.TrimSpace(projectSlugs[*gate.ProyectoID]); slug != "" {
			linea += " proyecto=" + slug
		}
	}
	if reviewer := strings.TrimSpace(gate.ReviewerAgente); reviewer != "" {
		linea += " reviewer=" + reviewer
	}
	if severity := strings.TrimSpace(gate.SeverityMax); severity != "" {
		linea += " severity=" + severity
	}
	return linea
}

func formatSupervisorReviewSignalOverviewLine(item *supervisorReviewSignal) string {
	if item == nil || item.Event == nil {
		return ""
	}
	linea := "- " + strings.TrimSpace(item.Event.Kind)
	if agent := strings.TrimSpace(item.Agent); agent != "" {
		linea += " agente=" + agent
	}
	if project := strings.TrimSpace(item.Project); project != "" {
		linea += " proyecto=" + project
	}
	if msg := compactMCPLine(strings.TrimSpace(item.Event.Message), 180); msg != "" {
		linea += " · " + msg
	}
	return linea
}

func formatSupervisorReviewMergeOverviewLine(merge *db.GitMerge) string {
	if merge == nil {
		return ""
	}
	linea := "- #" + supervisorReviewIntString(merge.ID) + " estado=" + strings.TrimSpace(merge.Estado)
	if proyecto := strings.TrimSpace(merge.ProyectoSlug); proyecto != "" {
		linea += " proyecto=" + proyecto
	}
	if strings.TrimSpace(merge.SourceBranch) != "" || strings.TrimSpace(merge.TargetBranch) != "" {
		linea += " " + strings.TrimSpace(merge.SourceBranch) + "->" + strings.TrimSpace(merge.TargetBranch)
	}
	if requestedBy := strings.TrimSpace(merge.RequestedBy); requestedBy != "" {
		linea += " por=" + requestedBy
	}
	if note := compactMCPLine(strings.TrimSpace(merge.Notas), 180); note != "" {
		linea += " · " + note
	}
	return linea
}
