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
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/gitgobernanza"
	"orquesta/internal/controlruntime"
	"orquesta/notificaciones"
	"orquesta/planocontrol"
	"orquesta/reviewapp"
	"orquesta/runtimeagente"
	"orquesta/tareasapp"
)

type dbAutomationService struct{}

const autonomiaReplanTaskTitle = "Autonomía: replanificar backlog y abrir siguiente frente útil"

var runtimeBudgetObservationBackgroundGate struct {
	mu      sync.Mutex
	expires time.Time
}

var runtimeMailboxReevaluationGate struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func runtimeMailboxReevaluationInterval() time.Duration {
	return 2 * time.Minute
}

func runtimeMailboxShouldReevaluate(lane string, msgID, handleID int64) bool {
	if msgID <= 0 || handleID <= 0 {
		return true
	}
	scope := strings.TrimSpace(db.CurrentStorageDisplayTarget())
	if scope == "" {
		scope = "global"
	}
	key := scope + "|" + strings.TrimSpace(lane) + "|" + strconv.FormatInt(msgID, 10) + "|" + strconv.FormatInt(handleID, 10)
	now := time.Now().UTC()
	interval := runtimeMailboxReevaluationInterval()
	runtimeMailboxReevaluationGate.mu.Lock()
	defer runtimeMailboxReevaluationGate.mu.Unlock()
	if runtimeMailboxReevaluationGate.last == nil {
		runtimeMailboxReevaluationGate.last = map[string]time.Time{}
	}
	if last, ok := runtimeMailboxReevaluationGate.last[key]; ok && now.Sub(last) < interval {
		return false
	}
	runtimeMailboxReevaluationGate.last[key] = now
	return true
}

func resetRuntimeMailboxReevaluationGate() {
	runtimeMailboxReevaluationGate.mu.Lock()
	defer runtimeMailboxReevaluationGate.mu.Unlock()
	runtimeMailboxReevaluationGate.last = nil
}

func resetAutonomiaIdleAutoassignGate() {
	autonomiaIdleAutoassignGate.mu.Lock()
	defer autonomiaIdleAutoassignGate.mu.Unlock()
	autonomiaIdleAutoassignGate.last = nil
}

func resetAutonomiaDegradedTaskGate() {
	autonomiaDegradedTaskGate.mu.Lock()
	defer autonomiaDegradedTaskGate.mu.Unlock()
	autonomiaDegradedTaskGate.last = nil
}

func resetPresupuestoPrimerUsoSesionGate() {
	presupuestoPrimerUsoSesionGate.mu.Lock()
	defer presupuestoPrimerUsoSesionGate.mu.Unlock()
	presupuestoPrimerUsoSesionGate.seen = nil
}

var autonomiaActiveSessionsGate struct {
	mu      sync.Mutex
	expires time.Time
}

var autonomiaIdleAutoassignGate struct {
	mu   sync.Mutex
	last map[string]time.Time
}

var autonomiaDegradedTaskGate struct {
	mu   sync.Mutex
	last map[int64]time.Time
}

const (
	autonomiaDegradedTaskCooldown                = 12 * time.Minute
	autonomiaWorkerOpenTasksCeiling              = 2
	autonomiaBlockedTaskRecoveryOpenTasksCeiling = autonomiaWorkerOpenTasksCeiling
	autonomiaBlockedTaskRecoveryBurstCeiling     = autonomiaBlockedTaskRecoveryOpenTasksCeiling + 1
	autonomiaTMUXWorkerOpenTasksCeilingDefault   = 3
	autonomiaTMUXBlockedRecoveryOpenTasksDefault = autonomiaTMUXWorkerOpenTasksCeilingDefault + 1
	autonomiaStuckRestartEscalationWindowDefault = 6 * time.Hour
	autonomiaStuckRestartEscalationCountDefault  = 2
)

var presupuestoPrimerUsoSesionGate struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

func autonomiaIdleAutoassignInterval() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("autonomia_idle_autoassign_interval_seconds", 300)
	if seconds <= 0 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

func autonomiaCuentaCompartidaDisponible(agente string) (bool, string, error) {
	return db.CuentaCompartidaPermiteActivacionAgente(strings.TrimSpace(agente))
}

func autonomiaIdleAutoassignShouldAttempt(agente string, proyectoID int64) bool {
	agente = strings.ToLower(strings.TrimSpace(agente))
	if agente == "" || proyectoID <= 0 {
		return false
	}
	key := agente + "|" + strconv.FormatInt(proyectoID, 10)
	now := time.Now().UTC()
	interval := autonomiaIdleAutoassignInterval()
	autonomiaIdleAutoassignGate.mu.Lock()
	defer autonomiaIdleAutoassignGate.mu.Unlock()
	if autonomiaIdleAutoassignGate.last == nil {
		autonomiaIdleAutoassignGate.last = map[string]time.Time{}
	}
	if last, ok := autonomiaIdleAutoassignGate.last[key]; ok && now.Sub(last) < interval {
		return false
	}
	autonomiaIdleAutoassignGate.last[key] = now
	return true
}

func presupuestoPrimerUsoSesion(nombre string) bool {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	if nombre == "" {
		return false
	}
	presupuestoPrimerUsoSesionGate.mu.Lock()
	defer presupuestoPrimerUsoSesionGate.mu.Unlock()
	if presupuestoPrimerUsoSesionGate.seen == nil {
		presupuestoPrimerUsoSesionGate.seen = map[string]time.Time{}
	}
	if _, ok := presupuestoPrimerUsoSesionGate.seen[nombre]; ok {
		return false
	}
	presupuestoPrimerUsoSesionGate.seen[nombre] = time.Now().UTC()
	return true
}

func controlPlaneHasOperationalHandles() bool {
	handles, err := db.ListarRuntimeHandlesActivosOperativosRecientes()
	if err != nil {
		return true
	}
	return len(handles) > 0
}

func (dbAutomationService) CheckReanimaciones() ([]*db.Agente, error) {
	if _, err := revalidarPresupuestoBloqueadoBatch(); err != nil {
		return nil, err
	}
	return db.CheckReanimaciones()
}

func (dbAutomationService) ResetReanimacion(nombre string) error {
	nombre = strings.TrimSpace(nombre)
	if _, err := revalidarPresupuestoAgenteSiCorresponde(nombre, presupuestoPreflightRevalidationAge(), true); err != nil {
		return err
	}
	bloqueado, err := sostenerCooldownSiLaCuotaVisibleSigueBloqueada(nombre)
	if err != nil {
		return err
	}
	if bloqueado {
		return nil
	}
	if err := reactivarAgenteTrasReanimacion(nombre); err != nil {
		return err
	}
	if err := db.ResetReanimacion(nombre); err != nil {
		return err
	}
	resetStatusSnapshotCache()
	return nil
}

func (dbAutomationService) GarantizarSaludAgentes() error {
	return db.GarantizarSaludAgentes()
}

func (dbAutomationService) ProcesarAutonomiaAgentesBatch() (int, error) {
	if !allowAutonomiaActiveSessionsObservation(time.Now().UTC()) {
		return 0, nil
	}
	return procesarAutonomiaAgentesBatch()
}

func agenteIfExists(nombre string) (*db.Agente, error) {
	agente, err := db.GetAgente(strings.TrimSpace(nombre))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return agente, err
}

func proyectoIfExists(ref string) (*db.Proyecto, error) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return proyecto, err
}

func sostenerCooldownSiLaCuotaVisibleSigueBloqueada(nombre string) (bool, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return false, nil
	}
	agente, err := agenteIfExists(nombre)
	if err != nil {
		return false, err
	}
	if agente == nil || agente.CuotaRestantePct == nil || *agente.CuotaRestantePct > 0 {
		return false, nil
	}
	if strings.TrimSpace(agente.PresupuestoVentana) == "" {
		return false, nil
	}
	if agente.PresupuestoResetAt != nil && agente.PresupuestoResetAt.After(time.Now().UTC()) {
		motivo := strings.TrimSpace(agente.MotivoPausa)
		if motivo == "" {
			if dbWindow := strings.TrimSpace(agente.PresupuestoVentana); dbWindow != "" {
				motivo = "Auto-pausa sostenida por cuota visible: ventana " + dbWindow + " agotada"
			} else {
				motivo = "Auto-pausa sostenida por cuota visible agotada"
			}
		}
		return true, db.PausarAgenteHasta(nombre, agente.PresupuestoResetAt.UTC(), motivo)
	}
	return false, nil
}

func presupuestoBlockedRevalidationAge() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("pool_budget_blocked_revalidation_seconds", 3600)
	if seconds <= 0 {
		seconds = 3600
	}
	return time.Duration(seconds) * time.Second
}

func presupuestoPreflightRevalidationAge() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("pool_budget_preflight_revalidation_seconds", 60)
	if seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func runtimeBudgetBackgroundObservationInterval() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("runtime_budget_background_observation_interval_seconds", 120)
	if seconds <= 0 {
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}

func autonomiaActiveSessionsInterval() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("autonomia_active_sessions_interval_seconds", 60)
	if seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func allowRuntimeBudgetBackgroundObservation(now time.Time) bool {
	runtimeBudgetObservationBackgroundGate.mu.Lock()
	defer runtimeBudgetObservationBackgroundGate.mu.Unlock()
	if !runtimeBudgetObservationBackgroundGate.expires.IsZero() && now.Before(runtimeBudgetObservationBackgroundGate.expires) {
		return false
	}
	runtimeBudgetObservationBackgroundGate.expires = now.Add(runtimeBudgetBackgroundObservationInterval())
	return true
}

func allowAutonomiaActiveSessionsObservation(now time.Time) bool {
	autonomiaActiveSessionsGate.mu.Lock()
	defer autonomiaActiveSessionsGate.mu.Unlock()
	if !autonomiaActiveSessionsGate.expires.IsZero() && now.Before(autonomiaActiveSessionsGate.expires) {
		return false
	}
	autonomiaActiveSessionsGate.expires = now.Add(autonomiaActiveSessionsInterval())
	return true
}

func resetRuntimeBudgetObservationBackgroundGate() {
	runtimeBudgetObservationBackgroundGate.mu.Lock()
	defer runtimeBudgetObservationBackgroundGate.mu.Unlock()
	runtimeBudgetObservationBackgroundGate.expires = time.Time{}
}

func resetAutonomiaActiveSessionsObservationGate() {
	autonomiaActiveSessionsGate.mu.Lock()
	defer autonomiaActiveSessionsGate.mu.Unlock()
	autonomiaActiveSessionsGate.expires = time.Time{}
}

func presupuestoAgenteDebeRevalidarseAhora(a *db.Agente, minAge time.Duration, soloBloqueadosOStale bool) bool {
	if a == nil {
		return false
	}
	if soloBloqueadosOStale && !agenteBloqueadoPorCuotaVisible(a) && !a.PresupuestoStale {
		return false
	}
	if minAge <= 0 {
		minAge = time.Hour
	}
	if a.PresupuestoCheckedAt == nil || a.PresupuestoCheckedAt.IsZero() {
		return true
	}
	return time.Since(a.PresupuestoCheckedAt.UTC()) >= minAge
}

func revalidarPresupuestoAgenteSiCorresponde(nombre string, minAge time.Duration, force bool) (int, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return 0, nil
	}
	agente, err := db.GetAgente(nombre)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if agente == nil {
		return 0, nil
	}
	if force && presupuestoPrimerUsoSesion(nombre) {
		return refrescarPresupuestoSesionObservadoAgente(nombre)
	}
	if !force && !presupuestoAgenteDebeRevalidarseAhora(agente, minAge, true) {
		return 0, nil
	}
	if force && !presupuestoAgenteDebeRevalidarseAhora(agente, minAge, false) {
		return 0, nil
	}
	return refrescarPresupuestoSesionObservadoAgente(nombre)
}

func revalidarPresupuestoBloqueadoBatch() (int, error) {
	agentes, err := agentesService.ListAgents()
	if err != nil {
		return 0, err
	}
	total := 0
	minAge := presupuestoBlockedRevalidationAge()
	for _, agente := range agentes {
		if !presupuestoAgenteDebeRevalidarseAhora(agente, minAge, true) {
			continue
		}
		n, err := revalidarPresupuestoAgenteSiCorresponde(agente.Nombre, minAge, false)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func revalidarPresupuestoPlanificacionBatch() (int, error) {
	agentes, err := agentesService.ListAgents()
	if err != nil {
		return 0, err
	}
	total := 0
	minAge := presupuestoPreflightRevalidationAge()
	for _, agente := range agentes {
		if agente == nil || !agente.Habilitado {
			continue
		}
		if !presupuestoAgenteDebeRevalidarseAhora(agente, minAge, false) {
			continue
		}
		n, err := revalidarPresupuestoAgenteSiCorresponde(agente.Nombre, minAge, true)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func revalidarYVerificarAgenteDisponibleParaTrabajo(nombre string) error {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return fmt.Errorf("agente obligatorio")
	}
	agente, err := db.GetAgente(nombre)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("agente %s no encontrado", nombre)
		}
		return err
	}
	if agente == nil {
		return fmt.Errorf("agente %s no encontrado", nombre)
	}
	if err := validarAgenteDisponibleParaTrabajo(nombre, agente); err != nil {
		return err
	}
	if err := validarEstadoOperativoAgenteParaTrabajo(nombre); err != nil {
		return err
	}
	if _, err := revalidarPresupuestoAgenteSiCorresponde(nombre, presupuestoPreflightRevalidationAge(), true); err != nil && err != sql.ErrNoRows {
		return err
	}
	agente, err = db.GetAgente(nombre)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("agente %s no encontrado", nombre)
		}
		return err
	}
	if agente == nil {
		return fmt.Errorf("agente %s no encontrado", nombre)
	}
	if err := validarAgenteDisponibleParaTrabajo(nombre, agente); err != nil {
		return err
	}
	return validarEstadoOperativoAgenteParaTrabajo(nombre)
}

func validarAgenteDisponibleParaTrabajo(nombre string, agente *db.Agente) error {
	if !agenteBloqueadoPorCuotaVisible(agente) {
		return nil
	}
	motivo := strings.TrimSpace(agente.MotivoPausa)
	if motivo != "" {
		motivo = ": " + motivo
	}
	if bloqueoCuotaEstimadoVisible(agente) {
		return fmt.Errorf("agente %s bloqueado por cuota estimada%s", nombre, motivo)
	}
	return fmt.Errorf("agente %s bloqueado por cuota%s", nombre, motivo)
}

func validarEstadoOperativoAgenteParaTrabajo(nombre string) error {
	rows, err := agentesService.BuildPanelRows()
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Agente == nil || !strings.EqualFold(strings.TrimSpace(row.Agente.Nombre), strings.TrimSpace(nombre)) {
			continue
		}
		estado := strings.TrimSpace(row.EstadoOperativo)
		switch estado {
		case "", "disponible", "sin_tarea":
			return nil
		case "trabajando":
			return nil
		case "bloqueado_por_cuota", "bloqueado_por_runtime", "mailbox_atascada", "atascado", "saturado":
			detalle := strings.TrimSpace(row.DetalleOperativo)
			if detalle != "" {
				return fmt.Errorf("agente %s no disponible para trabajo: %s (%s)", nombre, estado, detalle)
			}
			return fmt.Errorf("agente %s no disponible para trabajo: %s", nombre, estado)
		default:
			return nil
		}
	}
	return nil
}

func (dbAutomationService) PlanificarTareasAutomaticamente() error {
	if _, err := revalidarPresupuestoPlanificacionBatch(); err != nil {
		return err
	}
	return db.PlanificarTareasAutomaticamente()
}

func (dbAutomationService) ProcesarRuntimeSupervisionBatch() (int, error) {
	return db.ProcesarRuntimeSupervisionBatch()
}

func (dbAutomationService) ReconciliarRuntimeHandlesStale() (int, error) {
	return db.ReconciliarRuntimeHandlesStale()
}

func (dbAutomationService) ReconciliarRuntimeOrdersStale() (int, error) {
	return db.ReconciliarRuntimeOrdersStale()
}

func (dbAutomationService) ProcesarRuntimeTranscriptBatch() (int, error) {
	return procesarRuntimeTranscriptBatch()
}

func (dbAutomationService) ProcesarPresupuestoSesionObservadoBatch() (int, error) {
	return procesarPresupuestoSesionObservadoBatch()
}

func (dbAutomationService) ProcesarRuntimeMailboxBatch() (int, error) {
	return procesarRuntimeMailboxBatch()
}

func (dbAutomationService) ProcesarRuntimeOrdersBatch() (int, error) {
	return db.ProcesarRuntimeOrdersBatch()
}

func (dbAutomationService) ProcesarRuntimeHygieneBatch() (int, error) {
	total := 0
	resultado, err := db.PurgarRuntimeHistorico()
	if err != nil {
		return 0, err
	}
	if resultado != nil {
		if resultado.Handles != nil {
			total += resultado.Handles.Deleted
		}
		if resultado.Orders != nil {
			total += resultado.Orders.Deleted
		}
	}
	n, err := db.PurgarDatosOperacionales()
	if err != nil {
		return total, err
	}
	total += n
	return total, nil
}

func (dbAutomationService) ProcesarRefineriaBatch() (int, error) {
	return db.ProcesarRefineriaBatch()
}

func (dbAutomationService) ProcesarHandoffsBatch() (int, error) {
	return db.ProcesarHandoffsBatch()
}

func (dbAutomationService) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(agente, accion, entidad, entidadID, detalle)
}

func newControlPlaneRunner(debugLogger *log.Logger, debugControlPlane bool) *planocontrol.Runner {
	runner := &planocontrol.Runner{
		Automation:        dbAutomationService{},
		NotificationFeed:  db.CanalNotificaciones,
		InitNotifications: notificaciones.InicializarDesdeConfig,
		Notifier: func() notificaciones.Notificador {
			return notificaciones.GlobalNotificador
		},
	}
	if debugControlPlane && debugLogger != nil {
		runner.Debugf = debugLogger.Printf
	}
	return runner
}

func procesarRuntimeTranscriptBatch() (int, error) {
	ingested, err := db.IngestarRuntimeTranscriptActivos()
	if err != nil {
		return ingested, err
	}
	if !controlPlaneConfigBoolOrDefault("runtime_transcript_auto_guidance_enabled", true) {
		return ingested, nil
	}
	signals, err := db.ListarRuntimeTranscript(db.FiltroRuntimeTranscript{
		SoloSenalesPend: true,
		Limit:           50,
	})
	if err != nil {
		return ingested, err
	}
	processedSignals := 0
	for i := len(signals) - 1; i >= 0; i-- {
		item := signals[i]
		if item == nil || strings.TrimSpace(item.Classification) == "" {
			continue
		}
		note, err := procesarSignalTranscript(item)
		if err != nil {
			return ingested + processedSignals, err
		}
		if strings.TrimSpace(note) != "" {
			if err := db.MarcarRuntimeTranscriptManejado(item.ID, note); err != nil {
				return ingested + processedSignals, err
			}
		}
		processedSignals++
	}
	return ingested + processedSignals, nil
}

func procesarPresupuestoSesionObservadoBatch() (int, error) {
	if !allowRuntimeBudgetBackgroundObservation(time.Now().UTC()) {
		return 0, nil
	}
	handles, err := db.ListarRuntimeHandlesParaPresupuesto()
	if err != nil {
		return 0, err
	}
	ordenarHandlesParaPresupuestoVivo(handles)
	procesados := 0
	vistos := map[string]struct{}{}
	for _, handle := range handles {
		ok, agente, err := refrescarPresupuestoSesionObservadoHandle(handle)
		if err != nil {
			return procesados, err
		}
		if !ok || agente == "" {
			continue
		}
		if _, dup := vistos[agente]; dup {
			continue
		}
		vistos[agente] = struct{}{}
		procesados++
	}
	return procesados, nil
}

func refrescarPresupuestoSesionObservadoAgente(nombre string) (int, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		procesados, err := procesarPresupuestoSesionObservadoBatch()
		if err != nil {
			return procesados, err
		}
		if procesados > 0 {
			resetStatusSnapshotCache()
		}
		return procesados, nil
	}
	handles, err := listarRuntimeHandlesPresupuestoAgente(nombre)
	if err != nil {
		return 0, err
	}
	ordenarHandlesParaPresupuestoVivo(handles)
	procesados := 0
	for _, handle := range handles {
		ok, _, err := refrescarPresupuestoSesionObservadoHandle(handle)
		if err != nil {
			return procesados, err
		}
		if ok {
			procesados++
			resetStatusSnapshotCache()
			break
		}
	}
	if procesados > 0 {
		return procesados, nil
	}
	sesion, err := db.ObtenerUltimaSesion(nombre, nil)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	ok, err := refrescarPresupuestoSesionObservadoDesdeSesion(sesion)
	if err != nil {
		return 0, err
	}
	if ok {
		resetStatusSnapshotCache()
		return 1, nil
	}
	return procesados, nil
}

func listarRuntimeHandlesPresupuestoAgente(nombre string) ([]*db.RuntimeHandle, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return nil, nil
	}
	handles, err := db.ListarRuntimeHandlesCanonicosRecientes(&nombre)
	if err != nil {
		return nil, err
	}
	if len(handles) > 0 {
		return handles, nil
	}
	return db.ListarRuntimeHandles(&nombre)
}

func refrescarPresupuestoSesionObservadoDesdeSesion(sesion *db.Sesion) (bool, error) {
	if sesion == nil || sesion.ID <= 0 {
		return false, nil
	}
	meta := metadataPresupuestoDesdeSesion(sesion)
	metaJSON, _ := json.Marshal(meta)
	handle := &db.RuntimeHandle{
		Agente:       strings.TrimSpace(sesion.Agente),
		ProyectoID:   sesion.ProyectoID,
		SesionID:     &sesion.ID,
		Estado:       "activo",
		MetadataJSON: string(metaJSON),
	}
	obj := controlruntime.ObjetivoProceso{
		MetadataJSON: handle.MetadataJSON,
	}
	if observed, err := controlruntime.ObserveCodexProfileStatus(obj); err != nil {
		return false, nil
	} else if observed != nil && !observed.ObservedAt.IsZero() {
		return true, persistirPresupuestoSesionObservado(handle, observed)
	}
	if observed, err := controlruntime.ObserveCodexArtifacts(obj); err != nil {
		return false, nil
	} else if observed != nil && !observed.ObservedAt.IsZero() {
		return true, persistirPresupuestoSesionObservado(handle, observed)
	}
	if observed, err := controlruntime.ObserveClaudeRustArtifacts(obj); err != nil {
		return false, nil
	} else if observed != nil && !observed.ObservedAt.IsZero() {
		return true, persistirPresupuestoSesionClaudeObservado(handle, observed)
	}
	return false, nil
}

func metadataPresupuestoDesdeSesion(sesion *db.Sesion) map[string]any {
	meta := map[string]any{}
	if sesion == nil {
		return meta
	}
	if handle := runtimeHandleCanonicoDesdeSesionPresupuesto(sesion); handle != nil {
		for key, value := range mapFromJSON(handle.MetadataJSON) {
			meta[key] = value
		}
	}
	workingDir := strings.TrimSpace(sesion.CWD)
	if workingDir == "" && sesion.ProyectoID != nil && *sesion.ProyectoID > 0 {
		if proyecto, err := db.GetProyecto(strconv.FormatInt(*sesion.ProyectoID, 10)); err == nil && proyecto != nil {
			workingDir = strings.TrimSpace(proyecto.RutaAbs)
		}
	}
	if workingDir != "" {
		if budgetMetadataShouldFill(meta["working_dir"]) {
			meta["working_dir"] = workingDir
		}
	}
	if herramienta := strings.TrimSpace(sesion.Herramienta); herramienta != "" {
		if budgetMetadataShouldFill(meta["herramienta"]) {
			meta["herramienta"] = herramienta
		}
	}
	if externalID := strings.TrimSpace(sesion.ExternalSessionID); externalID != "" {
		if budgetMetadataShouldFill(meta["external_session_id"]) {
			meta["external_session_id"] = externalID
		}
	}
	if resume := strings.TrimSpace(sesion.ResumePayloadJSON); resume != "" {
		if budgetMetadataShouldFill(meta["resume_payload_json"]) {
			meta["resume_payload_json"] = resume
		}
	}
	if cfgHome := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_HOME")); cfgHome != "" {
		if budgetMetadataShouldFill(meta["claude_config_home"]) {
			meta["claude_config_home"] = cfgHome
		}
	}
	if strings.TrimSpace(sesion.Herramienta) != "" {
		if budgetMetadataShouldFill(meta["rendered_command"]) {
			meta["rendered_command"] = strings.TrimSpace(sesion.Herramienta)
		}
	}
	if rendered := renderedCommandDesdeSesion(sesion); rendered != "" {
		meta["rendered_command"] = rendered
	}
	if sesion.HeartbeatAt != nil && !sesion.HeartbeatAt.IsZero() {
		if budgetMetadataShouldFill(meta["started_at"]) {
			meta["started_at"] = sesion.HeartbeatAt.UTC().Format(time.RFC3339Nano)
		}
	} else if !sesion.Inicio.IsZero() {
		if budgetMetadataShouldFill(meta["started_at"]) {
			meta["started_at"] = sesion.Inicio.UTC().Format(time.RFC3339Nano)
		}
	}
	return meta
}

func runtimeHandleCanonicoDesdeSesionPresupuesto(sesion *db.Sesion) *db.RuntimeHandle {
	if sesion == nil {
		return nil
	}
	if sesion.ID > 0 {
		if handle, err := db.GetRuntimeHandleBySesionID(sesion.ID); err == nil && handle != nil {
			if runtimeHandleSirveComoCanonicoPresupuesto(handle) {
				if canonico := runtimeHandleCanonicoParaPresupuesto(handle); canonico != nil {
					return canonico
				}
			}
		}
	}
	agente := strings.TrimSpace(sesion.Agente)
	if agente == "" {
		return nil
	}
	var (
		handle *db.RuntimeHandle
		err    error
	)
	if sesion.ProyectoID != nil && *sesion.ProyectoID > 0 {
		handle, err = db.GetRuntimeHandleCanonicoRecienteAgenteProyecto(agente, sesion.ProyectoID)
	} else {
		handle, err = db.GetRuntimeHandleCanonicoRecienteAgente(agente)
	}
	if err != nil || handle == nil {
		return nil
	}
	return handle
}

func runtimeHandleSirveComoCanonicoPresupuesto(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") {
		return true
	}
	meta := mapFromJSON(handle.MetadataJSON)
	for _, key := range []string{
		"driver",
		"profile_status_wrapper",
		"worker_manifest_path",
		"worker_status_path",
		"worker_heartbeat_path",
		"wrapped_command",
		"rendered_command",
		"profile_name",
	} {
		if strings.TrimSpace(mapStringValue(meta, key)) != "" {
			return true
		}
	}
	return false
}

func renderedCommandDesdeSesion(sesion *db.Sesion) string {
	if sesion == nil {
		return ""
	}
	var (
		conector *db.Conector
		err      error
	)
	if sesion.ConectorID != nil && *sesion.ConectorID > 0 {
		conector, err = db.GetConector(strconv.FormatInt(*sesion.ConectorID, 10))
		if err != nil && err != sql.ErrNoRows {
			return ""
		}
	}
	if conector == nil && strings.TrimSpace(sesion.ConectorSlug) != "" {
		conector, err = db.GetConector(strings.TrimSpace(sesion.ConectorSlug))
		if err != nil && err != sql.ErrNoRows {
			return ""
		}
	}
	if conector == nil {
		return ""
	}
	base := strings.TrimSpace(conector.Comando)
	if base == "" {
		return ""
	}
	args := make([]string, 0, 4)
	if raw := strings.TrimSpace(conector.ArgsJSON); raw != "" {
		var parsed []string
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			for _, arg := range parsed {
				arg = strings.TrimSpace(arg)
				arg = strings.ReplaceAll(arg, "{{agent}}", strings.TrimSpace(sesion.Agente))
				if arg != "" {
					args = append(args, arg)
				}
			}
		}
	}
	if len(args) == 0 {
		return base
	}
	return strings.TrimSpace(base + " " + strings.Join(args, " "))
}

func ordenarHandlesParaPresupuestoVivo(handles []*db.RuntimeHandle) {
	sort.SliceStable(handles, func(i, j int) bool {
		return scoreHandlePresupuestoVivo(handles[i]) > scoreHandlePresupuestoVivo(handles[j])
	})
}

func scoreHandlePresupuestoVivo(handle *db.RuntimeHandle) int64 {
	if handle == nil {
		return -1
	}
	var score int64
	if runtimeHandleEsLegacyControlPlane(handle) {
		score -= 80_000_000_000
	}
	if strings.EqualFold(strings.TrimSpace(handle.Estado), "activo") {
		score += 50_000_000_000
	} else if strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		score += 40_000_000_000
	}
	if db.RuntimeHandlePermiteSendInputInteractivo(handle) {
		score += 20_000_000_000
	}
	meta := mapFromJSON(handle.MetadataJSON)
	switch strings.ToLower(strings.TrimSpace(mapStringValue(meta, "driver"))) {
	case "tmux_cli_session":
		score += 70_000_000_000
	case "process_pty_cli":
		score -= 60_000_000_000
	}
	if strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") &&
		!strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		!strings.EqualFold(strings.TrimSpace(mapStringValue(meta, "driver")), "tmux_cli_session") {
		score += 10_000_000_000
	}
	if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() {
		score += handle.LastSeenAt.UTC().Unix()
	}
	return score
}

func mapStringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if raw, ok := m[key]; ok {
		if value, ok := raw.(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func refrescarPresupuestoSesionObservadoHandle(handle *db.RuntimeHandle) (bool, string, error) {
	if handle == nil {
		return false, "", nil
	}
	handle = runtimeHandleCanonicoParaPresupuesto(handle)
	if handle == nil {
		return false, "", nil
	}
	agente := strings.TrimSpace(handle.Agente)
	if agente == "" {
		return false, "", nil
	}
	estado := strings.TrimSpace(handle.Estado)
	if !strings.EqualFold(estado, "activo") && !strings.EqualFold(estado, "pausado") {
		return false, agente, nil
	}
	sesionCanonica := sesionPresupuestoDesdeHandle(handle)
	baseMetaJSON := metadataPresupuestoDesdeHandle(handle)
	if handle.ID > 0 && db.DB != nil {
		if syncedHandle, _, _, err := db.SincronizarRuntimeHandleSupervisado(handle, nil, "budget_refresh"); err == nil && syncedHandle != nil {
			handle = syncedHandle
		} else if err != nil {
			controlPlaneBudgetDebugf("agente=%s sync_handle error=%v handle_id=%d", agente, err, handle.ID)
		}
	}
	metaJSON := mergeBudgetMetadataJSON(metadataPresupuestoDesdeHandle(handle), baseMetaJSON)
	obj := objetivoProcesoPresupuestoDesdeHandle(handle, metaJSON)
	if ok, err := refrescarPresupuestoHandleDesdeObjetivo(handle, agente, obj); err != nil {
		return false, agente, err
	} else if ok {
		return true, agente, nil
	}
	if strings.TrimSpace(baseMetaJSON) != "" && strings.TrimSpace(baseMetaJSON) != strings.TrimSpace(metaJSON) {
		fallbackObj := controlruntime.ObjetivoProceso{MetadataJSON: baseMetaJSON}
		if ok, err := refrescarPresupuestoHandleDesdeObjetivo(handle, agente, fallbackObj); err != nil {
			return false, agente, err
		} else if ok {
			return true, agente, nil
		}
	}
	if sesionCanonica != nil {
		if ok, err := refrescarPresupuestoSesionObservadoDesdeSesion(sesionCanonica); err == nil && ok {
			return true, agente, nil
		} else if err != nil {
			return false, agente, err
		}
	}
	controlPlaneBudgetDebugf("agente=%s sin presupuesto observado vivo", agente)
	return false, agente, nil
}

func objetivoProcesoPresupuestoDesdeHandle(handle *db.RuntimeHandle, metadataJSON string) controlruntime.ObjetivoProceso {
	obj := controlruntime.ObjetivoProceso{
		MetadataJSON: strings.TrimSpace(metadataJSON),
	}
	if handle == nil {
		return obj
	}
	obj.HandleKind = strings.TrimSpace(handle.HandleKind)
	obj.HandleRef = strings.TrimSpace(handle.HandleRef)
	if runtimeHandleEsLegacyControlPlane(handle) {
		return obj
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") {
		return obj
	}
	obj.PID = int64PtrFromHandleRef(handle.HandleKind, handle.HandleRef)
	return obj
}

func refrescarPresupuestoHandleDesdeObjetivo(handle *db.RuntimeHandle, agente string, obj controlruntime.ObjetivoProceso) (bool, error) {
	observed, err := controlruntime.ObserveCodexProfileStatus(obj)
	if err != nil {
		controlPlaneBudgetDebugf("agente=%s codex_profile_status error=%v handle_kind=%s handle_ref=%s", agente, err, strings.TrimSpace(handle.HandleKind), strings.TrimSpace(handle.HandleRef))
		observed = nil
	}
	if observed == nil {
		observed, err = controlruntime.ObserveCodexArtifacts(obj)
		if err != nil {
			controlPlaneBudgetDebugf("agente=%s codex_artifacts error=%v", agente, err)
			return false, nil
		}
	}
	if observed != nil && !observed.ObservedAt.IsZero() {
		controlPlaneBudgetDebugf("agente=%s presupuesto fuente=%s checked_at=%s cuenta=%s", agente, strings.TrimSpace(observed.ObservationSource), observed.ObservedAt.Format(time.RFC3339Nano), strings.TrimSpace(observed.AccountEmail))
		if err := persistirPresupuestoSesionObservado(handle, observed); err != nil {
			return false, err
		}
		return true, nil
	}
	claudeObserved, err := controlruntime.ObserveClaudeRustArtifacts(obj)
	if err != nil {
		controlPlaneBudgetDebugf("agente=%s claude_artifacts error=%v", agente, err)
		return false, nil
	}
	if claudeObserved == nil || claudeObserved.ObservedAt.IsZero() {
		return false, nil
	}
	controlPlaneBudgetDebugf("agente=%s presupuesto fuente=%s checked_at=%s cuenta=%s", agente, strings.TrimSpace(claudeObserved.AccountSource), claudeObserved.ObservedAt.Format(time.RFC3339Nano), strings.TrimSpace(claudeObserved.AccountEmail))
	if err := persistirPresupuestoSesionClaudeObservado(handle, claudeObserved); err != nil {
		return false, err
	}
	return true, nil
}

func metadataPresupuestoDesdeHandle(handle *db.RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		meta = map[string]any{}
	}
	if sesion := sesionPresupuestoDesdeHandle(handle); sesion != nil {
		mergeBudgetMetadataPreferMissing(meta, metadataPresupuestoDesdeSesion(sesion))
	}
	if len(meta) == 0 {
		return strings.TrimSpace(handle.MetadataJSON)
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return strings.TrimSpace(handle.MetadataJSON)
	}
	return string(raw)
}

func mergeBudgetMetadataJSON(primaryJSON, fallbackJSON string) string {
	primary := mapFromJSON(primaryJSON)
	fallback := mapFromJSON(fallbackJSON)
	switch {
	case primary == nil && fallback == nil:
		if strings.TrimSpace(primaryJSON) != "" {
			return strings.TrimSpace(primaryJSON)
		}
		return strings.TrimSpace(fallbackJSON)
	case primary == nil:
		raw, err := json.Marshal(fallback)
		if err != nil {
			return strings.TrimSpace(fallbackJSON)
		}
		return string(raw)
	case fallback == nil:
		raw, err := json.Marshal(primary)
		if err != nil {
			return strings.TrimSpace(primaryJSON)
		}
		return string(raw)
	default:
		mergeBudgetMetadataPreferMissing(primary, fallback)
		raw, err := json.Marshal(primary)
		if err != nil {
			return strings.TrimSpace(primaryJSON)
		}
		return string(raw)
	}
}

func sesionPresupuestoDesdeHandle(handle *db.RuntimeHandle) *db.Sesion {
	if handle == nil {
		return nil
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		sesion, err := db.GetSesionByID(*handle.SesionID)
		if err == nil && sesion != nil {
			return sesion
		}
	}
	sesion, err := db.GetSesionActivaOperativa(strings.TrimSpace(handle.Agente), handle.ProyectoID)
	if err != nil {
		return nil
	}
	return sesion
}

func mergeBudgetMetadataPreferMissing(dst, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for key, value := range src {
		if !budgetMetadataShouldFill(dst[key]) {
			continue
		}
		dst[key] = value
	}
}

func budgetMetadataShouldFill(current any) bool {
	switch value := current.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(value) == ""
	default:
		return false
	}
}

func runtimeHandleCanonicoParaPresupuesto(handle *db.RuntimeHandle) *db.RuntimeHandle {
	if handle == nil || !runtimeHandleEsLegacyControlPlane(handle) {
		return handle
	}
	agente := strings.TrimSpace(handle.Agente)
	if agente == "" {
		return handle
	}
	var (
		canonico *db.RuntimeHandle
		err      error
	)
	if handle.ProyectoID != nil && *handle.ProyectoID > 0 {
		canonico, err = db.GetRuntimeHandleCanonicoRecienteAgenteProyecto(agente, handle.ProyectoID)
	} else {
		canonico, err = db.GetRuntimeHandleCanonicoRecienteAgente(agente)
	}
	if err != nil || canonico == nil || canonico.ID == handle.ID {
		return handle
	}
	if runtimeHandleEsLegacyControlPlane(canonico) {
		return handle
	}
	return canonico
}

func controlPlaneBudgetDebugf(format string, args ...any) {
	base := parseBoolDebug(strings.TrimSpace(os.Getenv("ORQUESTA_DEBUG")), false)
	if !parseBoolDebug(strings.TrimSpace(os.Getenv("ORQUESTA_DEBUG_RUNTIME_STATUS")), base) {
		return
	}
	log.Printf("orquesta[budget_status][debug] "+format, args...)
}

func stringValueFromJSON(raw, key string) string {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return ""
	}
	if value, ok := payload[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func persistirPresupuestoSesionObservado(handle *db.RuntimeHandle, observed *controlruntime.CodexObservedArtifacts) error {
	if handle == nil || observed == nil {
		return nil
	}
	sesionID := int64(0)
	if handle.SesionID != nil && *handle.SesionID > 0 {
		sesionID = *handle.SesionID
	}
	if sesionID <= 0 {
		sesion, err := db.GetSesionActivaOperativa(strings.TrimSpace(handle.Agente), handle.ProyectoID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		if sesion == nil || sesion.ID <= 0 {
			return nil
		}
		sesionID = sesion.ID
	}
	source := strings.TrimSpace(observed.ObservationSource)
	if source == "" {
		source = "codex_token_count_observed"
	}
	if ultimo, err := db.UltimoPresupuestoSesionPorFuente(sesionID, source); err == nil && ultimo != nil && !ultimo.CheckedAt.IsZero() && !observed.ObservedAt.After(ultimo.CheckedAt) {
		if presupuestoSnapshotObservadoUsable(ultimo.RawSnapshotJSON) {
			return nil
		}
	}
	rawSnapshot := observed.RawSnapshot
	if strings.TrimSpace(rawSnapshot) == "" {
		payload := map[string]any{}
		if observed.Primary.UsedPercent != nil || observed.Secondary.UsedPercent != nil {
			rateLimits := map[string]any{}
			if observed.Primary.UsedPercent != nil {
				rateLimits["primary"] = codexObservedRateLimitPayload(observed.Primary)
			}
			if observed.Secondary.UsedPercent != nil {
				rateLimits["secondary"] = codexObservedRateLimitPayload(observed.Secondary)
			}
			payload["rate_limits"] = rateLimits
		}
		if observed.AccountID != "" {
			payload["account_id"] = observed.AccountID
		}
		if observed.AccountEmail != "" {
			payload["account_email"] = observed.AccountEmail
		}
		if observed.AccountUser != "" {
			payload["account_user"] = observed.AccountUser
		}
		if observed.AccountSource != "" {
			payload["account_source"] = observed.AccountSource
		}
		if observed.PlanType != "" {
			payload["plan_type"] = observed.PlanType
		}
		if raw, err := json.Marshal(payload); err == nil {
			rawSnapshot = string(raw)
		}
	}
	windowKind, startedAt, resetAt := codexObservedWindowMeta(observed.Primary, observed.Secondary, observed.ObservedAt)
	presupuesto := &db.PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       windowKind,
		WindowStartedAt:  startedAt,
		ResetAt:          resetAt,
		RemainingCredits: observed.Credits,
		BudgetSource:     source,
		RawSnapshotJSON:  rawSnapshot,
		CheckedAt:        observed.ObservedAt,
	}
	if _, err := db.RegistrarPresupuestoSesion(presupuesto); err != nil {
		return err
	}
	db.Audit("orquesta", "registrar_presupuesto_codex_observado", "presupuesto_sesion", 0,
		fmt.Sprintf("agente=%s sesion=%d source=%s session_file=%s", strings.TrimSpace(handle.Agente), sesionID, source, strings.TrimSpace(observed.SessionPath)))
	return nil
}

func persistirPresupuestoSesionClaudeObservado(handle *db.RuntimeHandle, observed *controlruntime.ClaudeRustObservedArtifacts) error {
	if handle == nil || observed == nil {
		return nil
	}
	sesionID := int64(0)
	if handle.SesionID != nil && *handle.SesionID > 0 {
		sesionID = *handle.SesionID
	}
	if sesionID <= 0 {
		sesion, err := db.GetSesionActivaOperativa(strings.TrimSpace(handle.Agente), handle.ProyectoID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		if sesion == nil || sesion.ID <= 0 {
			return nil
		}
		sesionID = sesion.ID
	}
	source := strings.TrimSpace(observed.AccountSource)
	if strings.TrimSpace(observed.SessionPath) != "" ||
		observed.Usage.TotalTokens > 0 ||
		observed.Usage.MessageCount > 0 ||
		observed.Usage.Turns > 0 {
		source = "claude_rust_session_observed"
	}
	if source == "" {
		source = "claude_rust_session_observed"
	}
	if ultimo, err := db.UltimoPresupuestoSesionPorFuente(sesionID, source); err == nil && ultimo != nil && !ultimo.CheckedAt.IsZero() && !observed.ObservedAt.After(ultimo.CheckedAt) {
		if presupuestoSnapshotObservadoUsable(ultimo.RawSnapshotJSON) {
			return nil
		}
	}
	rawSnapshot := strings.TrimSpace(observed.RawSnapshot)
	if rawSnapshot == "" {
		payload := map[string]any{
			"observed_scope": "claude_rust_session",
		}
		if observed.AccountEmail != "" {
			payload["account_email"] = observed.AccountEmail
		}
		if observed.AccountUser != "" {
			payload["account_user"] = observed.AccountUser
		}
		if observed.AccountSource != "" {
			payload["account_source"] = observed.AccountSource
		}
		if observed.OAuthExpiresAt != nil && !observed.OAuthExpiresAt.IsZero() {
			payload["oauth"] = map[string]any{"expires_at": observed.OAuthExpiresAt.UTC().Format(time.RFC3339Nano)}
		}
		if observed.Usage.TotalTokens > 0 || observed.Usage.MessageCount > 0 || strings.TrimSpace(observed.SessionPath) != "" {
			sessionPayload := map[string]any{
				"session_path":                strings.TrimSpace(observed.SessionPath),
				"message_count":               observed.Usage.MessageCount,
				"turns":                       observed.Usage.Turns,
				"input_tokens":                observed.Usage.InputTokens,
				"output_tokens":               observed.Usage.OutputTokens,
				"cache_creation_input_tokens": observed.Usage.CacheCreationInputTokens,
				"cache_read_input_tokens":     observed.Usage.CacheReadInputTokens,
				"total_tokens":                observed.Usage.TotalTokens,
				"pricing_source":              "rust_default_sonnet",
			}
			if observed.Usage.EstimatedCostUSD != nil {
				sessionPayload["estimated_cost_usd"] = *observed.Usage.EstimatedCostUSD
			}
			if observed.Usage.UpdatedAt != nil && !observed.Usage.UpdatedAt.IsZero() {
				sessionPayload["updated_at"] = observed.Usage.UpdatedAt.UTC().Format(time.RFC3339Nano)
			}
			payload["session_usage"] = sessionPayload
		}
		if raw, err := json.Marshal(payload); err == nil {
			rawSnapshot = string(raw)
		}
	}
	presupuesto := &db.PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "unknown",
		BudgetSource:    source,
		RawSnapshotJSON: rawSnapshot,
		CheckedAt:       observed.ObservedAt,
	}
	if _, err := db.RegistrarPresupuestoSesion(presupuesto); err != nil {
		return err
	}
	db.Audit("orquesta", "registrar_presupuesto_claude_observado", "presupuesto_sesion", 0,
		fmt.Sprintf("agente=%s sesion=%d source=%s session_file=%s", strings.TrimSpace(handle.Agente), sesionID, source, strings.TrimSpace(observed.SessionPath)))
	return nil
}

func int64PtrFromHandleRef(kind, ref string) *int64 {
	if !strings.EqualFold(strings.TrimSpace(kind), "process") {
		return nil
	}
	n, err := strconv.ParseInt(strings.TrimSpace(ref), 10, 64)
	if err != nil || n <= 0 {
		return nil
	}
	return &n
}

func codexObservedRateLimitPayload(limit controlruntime.CodexObservedRateLimit) map[string]any {
	payload := map[string]any{
		"window_minutes": limit.WindowMinutes,
	}
	if limit.UsedPercent != nil {
		payload["used_percent"] = *limit.UsedPercent
	}
	if limit.ResetsAt != nil && !limit.ResetsAt.IsZero() {
		payload["resets_at"] = limit.ResetsAt.UTC().Unix()
	}
	return payload
}

func presupuestoSnapshotObservadoUsable(raw string) bool {
	meta := mapFromJSON(raw)
	if meta == nil {
		return false
	}
	rateLimits, _ := meta["rate_limits"].(map[string]any)
	if rateLimits == nil {
		return false
	}
	for _, key := range []string{"primary", "secondary"} {
		window, _ := rateLimits[key].(map[string]any)
		if window == nil {
			continue
		}
		if _, ok := float64FromAny(window["used_percent"]); ok {
			return true
		}
	}
	if usage, _ := meta["session_usage"].(map[string]any); usage != nil {
		if _, ok := float64FromAny(usage["total_tokens"]); ok {
			return true
		}
		if _, ok := float64FromAny(usage["estimated_cost_usd"]); ok {
			return true
		}
	}
	if email, _ := meta["account_email"].(string); strings.TrimSpace(email) != "" {
		return true
	}
	if accountID, _ := meta["account_id"].(string); strings.TrimSpace(accountID) != "" {
		return true
	}
	if user, _ := meta["account_user"].(string); strings.TrimSpace(user) != "" {
		return true
	}
	return false
}

func codexObservedWindowMeta(primary, secondary controlruntime.CodexObservedRateLimit, observedAt time.Time) (string, *time.Time, *time.Time) {
	chosen := primary
	kind := observedWindowKind(primary.WindowMinutes, "primary")
	if chosen.UsedPercent == nil && secondary.UsedPercent != nil {
		chosen = secondary
		kind = observedWindowKind(secondary.WindowMinutes, "secondary")
	}
	if chosen.ResetsAt == nil || chosen.ResetsAt.IsZero() {
		return kind, nil, nil
	}
	minutes := chosen.WindowMinutes
	if minutes <= 0 {
		return kind, nil, chosen.ResetsAt
	}
	startedAt := chosen.ResetsAt.Add(-time.Duration(minutes) * time.Minute).UTC()
	return kind, &startedAt, chosen.ResetsAt
}

func observedWindowKind(minutes int, fallback string) string {
	switch {
	case minutes == 300:
		return "5h"
	case minutes >= 7*24*60:
		return "weekly"
	case minutes > 0:
		return fmt.Sprintf("%dm", minutes)
	default:
		return fallback
	}
}

func mapFromJSON(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func float64FromAny(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		out, err := v.Float64()
		if err == nil {
			return out, true
		}
	case string:
		out, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return out, true
		}
	}
	return 0, false
}

type runtimeMailboxBatchSnapshot struct {
	hotHandles           map[string]*db.RuntimeHandle
	activeHandles        map[string]*db.RuntimeHandle
	activeHandleLoaded   map[string]struct{}
	projectsByID         map[int64]*db.Proyecto
	projectsLoaded       map[int64]struct{}
	quotaState           map[string]bool
	quotaStateLoaded     map[string]struct{}
	ordersByAgentProject map[string][]*db.RuntimeOrder
	ordersLoaded         map[string]struct{}
	supervisedByHandleID map[int64]runtimeMailboxSupervisedHandleSnapshot
	externalByHandleID   map[int64]runtimeMailboxExternalSessionSnapshot
}

type runtimeMailboxSupervisedHandleSnapshot struct {
	handle            *db.RuntimeHandle
	runtime           *db.RuntimeInstance
	externalSessionID string
	err               error
}

type runtimeMailboxExternalSessionSnapshot struct {
	handle            *db.RuntimeHandle
	externalSessionID string
	err               error
}

func newRuntimeMailboxBatchSnapshot() *runtimeMailboxBatchSnapshot {
	hotHandles, _ := db.ListarRuntimeHandlesActivosOperativosRecientes()
	return &runtimeMailboxBatchSnapshot{
		hotHandles:           hotHandles,
		activeHandles:        map[string]*db.RuntimeHandle{},
		activeHandleLoaded:   map[string]struct{}{},
		projectsByID:         map[int64]*db.Proyecto{},
		projectsLoaded:       map[int64]struct{}{},
		quotaState:           map[string]bool{},
		quotaStateLoaded:     map[string]struct{}{},
		ordersByAgentProject: map[string][]*db.RuntimeOrder{},
		ordersLoaded:         map[string]struct{}{},
		supervisedByHandleID: map[int64]runtimeMailboxSupervisedHandleSnapshot{},
		externalByHandleID:   map[int64]runtimeMailboxExternalSessionSnapshot{},
	}
}

func runtimeMailboxBatchKey(agente string, proyectoID *int64) string {
	agente = strings.TrimSpace(agente)
	if proyectoID == nil || *proyectoID <= 0 {
		return agente + "|-"
	}
	return agente + "|" + strconv.FormatInt(*proyectoID, 10)
}

func runtimeMailboxHotHandleLookup(snapshot map[string]*db.RuntimeHandle, agente string, proyectoID *int64) *db.RuntimeHandle {
	if len(snapshot) == 0 {
		return nil
	}
	agente = strings.ToLower(strings.TrimSpace(agente))
	if agente == "" {
		return nil
	}
	if proyectoID != nil && *proyectoID > 0 {
		key := agente + "#" + strconv.FormatInt(*proyectoID, 10)
		if handle := snapshot[key]; handle != nil {
			return handle
		}
		return nil
	}
	var best *db.RuntimeHandle
	for _, handle := range snapshot {
		if handle == nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(handle.Agente)) != agente {
			continue
		}
		if best == nil || db.RuntimeHandleSnapshotIsFresh(handle, 2*time.Minute) && !db.RuntimeHandleSnapshotIsFresh(best, 2*time.Minute) {
			best = handle
		}
	}
	return best
}

func runtimeMailboxQuotaStateKey(agente, estado string) string {
	return strings.TrimSpace(agente) + "|" + strings.TrimSpace(estado)
}

func (s *runtimeMailboxBatchSnapshot) activeHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if s == nil {
		handle, err := runtimesService.GetOperationalRuntimeHandleAgentProject(strings.TrimSpace(agente), proyectoID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
		return runtimesService.GetActiveRuntimeHandleAgentProject(strings.TrimSpace(agente), proyectoID)
	}
	key := runtimeMailboxBatchKey(agente, proyectoID)
	if _, ok := s.activeHandleLoaded[key]; ok {
		return s.activeHandles[key], nil
	}
	resolveFreshByID := func(handle *db.RuntimeHandle, err error) (*db.RuntimeHandle, error) {
		if err != nil || handle == nil {
			return nil, err
		}
		fresh, err := db.GetRuntimeHandle(handle.ID)
		if err != nil {
			return nil, err
		}
		return fresh, nil
	}
	if handle := runtimeMailboxHotHandleLookup(s.hotHandles, agente, proyectoID); handle != nil && db.RuntimeHandleSnapshotIsFresh(handle, 2*time.Minute) {
		fresh, err := resolveFreshByID(handle, nil)
		if err != nil {
			return nil, err
		}
		if fresh != nil {
			s.activeHandleLoaded[key] = struct{}{}
			s.activeHandles[key] = fresh
			return fresh, nil
		}
	}
	var (
		candidate *db.RuntimeHandle
		err       error
	)
	if proyectoID != nil && *proyectoID > 0 {
		candidate, err = db.GetRuntimeHandleCanonicoRecienteAgenteProyecto(strings.TrimSpace(agente), proyectoID)
	} else {
		candidate, err = db.GetRuntimeHandleCanonicoRecienteAgente(strings.TrimSpace(agente))
	}
	if err != nil {
		return nil, err
	}
	if fresh, err := resolveFreshByID(candidate, nil); err != nil {
		return nil, err
	} else if fresh != nil {
		s.activeHandleLoaded[key] = struct{}{}
		s.activeHandles[key] = fresh
		return fresh, nil
	}
	handle, err := runtimesService.GetOperationalRuntimeHandleAgentProject(strings.TrimSpace(agente), proyectoID)
	if err != nil {
		return nil, err
	}
	if handle == nil {
		handle, err = runtimesService.GetActiveRuntimeHandleAgentProject(strings.TrimSpace(agente), proyectoID)
		if err != nil {
			return nil, err
		}
	}
	s.activeHandleLoaded[key] = struct{}{}
	s.activeHandles[key] = handle
	return handle, nil
}

func (s *runtimeMailboxBatchSnapshot) ordersForAgentProject(agente string, proyectoID *int64) ([]*db.RuntimeOrder, error) {
	if s == nil {
		return runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: stringPtr(strings.TrimSpace(agente)), ProyectoID: proyectoID})
	}
	key := runtimeMailboxBatchKey(agente, proyectoID)
	if _, ok := s.ordersLoaded[key]; ok {
		return s.ordersByAgentProject[key], nil
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: stringPtr(strings.TrimSpace(agente)), ProyectoID: proyectoID})
	if err != nil {
		return nil, err
	}
	s.ordersLoaded[key] = struct{}{}
	s.ordersByAgentProject[key] = orders
	return orders, nil
}

func (s *runtimeMailboxBatchSnapshot) project(id int64) (*db.Proyecto, error) {
	if id <= 0 {
		return nil, nil
	}
	if s == nil {
		return runtimesService.GetProject(strconv.FormatInt(id, 10))
	}
	if _, ok := s.projectsLoaded[id]; ok {
		return s.projectsByID[id], nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(id, 10))
	if err != nil {
		return nil, err
	}
	s.projectsLoaded[id] = struct{}{}
	s.projectsByID[id] = proyecto
	return proyecto, nil
}

func (s *runtimeMailboxBatchSnapshot) quotaMatches(agente, estado string) (bool, error) {
	if s == nil {
		return agenteEstadoCuotaPersistido(agente, estado)
	}
	key := runtimeMailboxQuotaStateKey(agente, estado)
	if _, ok := s.quotaStateLoaded[key]; ok {
		return s.quotaState[key], nil
	}
	match, err := agenteEstadoCuotaPersistido(agente, estado)
	if err != nil {
		return false, err
	}
	s.quotaStateLoaded[key] = struct{}{}
	s.quotaState[key] = match
	return match, nil
}

func (s *runtimeMailboxBatchSnapshot) supervisedHandle(handle *db.RuntimeHandle) (*db.RuntimeHandle, *db.RuntimeInstance, string, error) {
	if handle == nil {
		return nil, nil, "", nil
	}
	if s == nil {
		return db.SincronizarRuntimeHandleSupervisado(handle, nil, "runtime_mailbox_supervisor_local")
	}
	if cached, ok := s.supervisedByHandleID[handle.ID]; ok {
		return cached.handle, cached.runtime, cached.externalSessionID, cached.err
	}
	resultHandle, runtime, externalSessionID, err := db.SincronizarRuntimeHandleSupervisado(handle, nil, "runtime_mailbox_supervisor_local")
	s.supervisedByHandleID[handle.ID] = runtimeMailboxSupervisedHandleSnapshot{
		handle:            resultHandle,
		runtime:           runtime,
		externalSessionID: externalSessionID,
		err:               err,
	}
	return resultHandle, runtime, externalSessionID, err
}

func (s *runtimeMailboxBatchSnapshot) externalSessionHandle(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) (*db.RuntimeHandle, string, error) {
	if handle == nil {
		return nil, "", nil
	}
	if s == nil {
		return db.SincronizarRuntimeHandleExternalSessionID(handle, runtime)
	}
	if cached, ok := s.externalByHandleID[handle.ID]; ok {
		return cached.handle, cached.externalSessionID, cached.err
	}
	resultHandle, externalSessionID, err := db.SincronizarRuntimeHandleExternalSessionID(handle, runtime)
	s.externalByHandleID[handle.ID] = runtimeMailboxExternalSessionSnapshot{
		handle:            resultHandle,
		externalSessionID: externalSessionID,
		err:               err,
	}
	return resultHandle, externalSessionID, err
}

func procesarRuntimeMailboxBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	consumed := map[int64]struct{}{}
	snapshot := newRuntimeMailboxBatchSnapshot()
	reconciled, err := reconciliarRuntimeMailboxAgenteSinVidaBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return reconciled, err
	}
	refreshCooldownReconciled, err := reconciliarRuntimeMailboxRefreshEnfriamientoBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return reconciled + refreshCooldownReconciled, err
	}
	reconciled += refreshCooldownReconciled
	watchdogReconciled, err := reconciliarRuntimeMailboxWatchdogSinHandleBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return reconciled + watchdogReconciled, err
	}
	reconciled += watchdogReconciled
	interactive, err := procesarRuntimeMailboxInteractivoBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return reconciled + interactive, err
	}
	sessionResume, err := procesarRuntimeMailboxSessionResumeBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return reconciled + interactive + sessionResume, err
	}
	bootstrapTMUX, err := procesarRuntimeMailboxBootstrapTMUXBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return reconciled + interactive + sessionResume + bootstrapTMUX, err
	}
	restarts, err := procesarRuntimeMailboxCoordinatedRestartBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return reconciled + interactive + sessionResume + bootstrapTMUX + restarts, err
	}
	return reconciled + interactive + sessionResume + bootstrapTMUX + restarts, nil
}

func reconciliarRuntimeMailboxRefreshEnfriamientoBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	return reconciliarRuntimeMailboxRefreshEnfriamientoBatchConMailbox(mailbox, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
}

func reconciliarRuntimeMailboxRefreshEnfriamientoBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	for _, msg := range mailbox {
		if msg == nil || !runtimeMailboxRefreshPuedeConsumirsePorEnfriamiento(msg, snapshot) {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
			return total, err
		} else if !canConsume {
			continue
		}
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return total, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return total, err
		}
		db.Audit("orquesta", "runtime_mailbox_refresh_enfriamiento", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s kind=%s refresh consumido por agente en enfriamiento",
				strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
		consumed[msg.ID] = struct{}{}
		total++
	}
	return total, nil
}

func reconciliarRuntimeMailboxAgenteSinVidaBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	return reconciliarRuntimeMailboxAgenteSinVidaBatchConMailbox(mailbox, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
}

func reconciliarRuntimeMailboxAgenteSinVidaBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		debeConsumirse, detalle, err := runtimeMailboxDebeConsumirsePorAgenteSinVida(msg, snapshot)
		if err != nil {
			return total, err
		}
		if !debeConsumirse {
			continue
		}
		if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
			return total, err
		} else if !canConsume {
			continue
		}
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return total, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return total, err
		}
		db.Audit("orquesta", "runtime_mailbox_agente_sin_vida", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s kind=%s %s", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), detalle))
		consumed[msg.ID] = struct{}{}
		total++
	}
	return total, nil
}

func reconciliarRuntimeMailboxWatchdogSinHandleBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	return reconciliarRuntimeMailboxWatchdogSinHandleBatchConMailbox(mailbox, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
}

func reconciliarRuntimeMailboxWatchdogSinHandleBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	for _, msg := range mailbox {
		if msg == nil || strings.TrimSpace(msg.Kind) != "watchdog" {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		handle, err := snapshot.activeHandle(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		if watchdogPuedeConsumirsePorEnfriamiento(msg, handle, snapshot) {
			if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
				return total, err
			} else if !canConsume {
				continue
			}
			if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
				return total, err
			}
			if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
				return total, err
			}
			db.Audit("orquesta", "runtime_mailbox_watchdog_enfriamiento", "runtime_mailbox", msg.ID,
				fmt.Sprintf("agente=%s proyecto_id=%s watchdog consumido por agente en enfriamiento",
					strings.TrimSpace(msg.ToAgente), runtimeMailboxProyectoDetalle(msg.ProyectoID)))
			consumed[msg.ID] = struct{}{}
			total++
			continue
		}
		if handle != nil {
			continue
		}
		if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
			return total, err
		} else if !canConsume {
			continue
		}
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return total, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return total, err
		}
		db.Audit("orquesta", "runtime_mailbox_watchdog_sin_handle", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s proyecto_id=%s watchdog consumido por ausencia de handle activo",
				strings.TrimSpace(msg.ToAgente), runtimeMailboxProyectoDetalle(msg.ProyectoID)))
		consumed[msg.ID] = struct{}{}
		total++
	}
	return total, nil
}

func watchdogPuedeConsumirsePorEnfriamiento(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, snapshot *runtimeMailboxBatchSnapshot) bool {
	if msg == nil || handle == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		return false
	}
	ok, err := snapshot.quotaMatches(strings.TrimSpace(msg.ToAgente), "enfriamiento")
	if err != nil {
		return false
	}
	return ok
}

func runtimeMailboxRefreshPuedeConsumirsePorEnfriamiento(msg *db.RuntimeMailboxMessage, snapshot *runtimeMailboxBatchSnapshot) bool {
	if msg == nil {
		return false
	}
	switch strings.TrimSpace(msg.Kind) {
	case db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
	default:
		return false
	}
	ok, err := snapshot.quotaMatches(strings.TrimSpace(msg.ToAgente), "enfriamiento")
	if err != nil {
		return false
	}
	return ok
}

func agenteEstadoCuotaPersistido(nombre, estado string) (bool, error) {
	nombre = strings.TrimSpace(nombre)
	estado = strings.TrimSpace(estado)
	if nombre == "" || estado == "" {
		return false, nil
	}
	actual, err := db.GetPersistedAgentQuotaState(nombre)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(actual), estado), nil
}

func runtimeMailboxProyectoDetalle(proyectoID *int64) string {
	if proyectoID == nil || *proyectoID <= 0 {
		return "-"
	}
	return strconv.FormatInt(*proyectoID, 10)
}

func runtimeMailboxDebeConsumirsePorAgenteSinVida(msg *db.RuntimeMailboxMessage, snapshot *runtimeMailboxBatchSnapshot) (bool, string, error) {
	if msg == nil {
		return false, "", nil
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if agente == "" {
		return false, "", nil
	}
	if strings.EqualFold(strings.TrimSpace(msg.Kind), "watchdog") {
		return false, "", nil
	}
	if presente, err := runtimeMailboxTienePresenciaRuntimeNoCanonica(agente, msg.ProyectoID); err != nil {
		return false, "", err
	} else if presente {
		return false, "", nil
	}
	handle, err := snapshot.activeHandle(agente, msg.ProyectoID)
	if err != nil {
		return false, "", err
	}
	if handle != nil {
		return false, "", nil
	}
	activa := true
	sesiones, err := sesionesAPIService.ListInspectionSessions(db.FiltroSesionesInspeccion{Activa: &activa})
	if err != nil {
		return false, "", err
	}
	for _, sesion := range sesiones {
		if sesion == nil || strings.TrimSpace(sesion.Agente) != agente {
			continue
		}
		if msg.ProyectoID == nil || sesion.ProyectoID == nil || *sesion.ProyectoID == *msg.ProyectoID {
			return false, "", nil
		}
	}
	proyectoActivoID, err := db.ObtenerProyectoActivoAgente(agente)
	if err != nil {
		return false, "", err
	}
	if proyectoActivoID > 0 && agentePerteneceAFlotaAutobootstrap(agente) {
		return false, "", nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{Agente: &agente})
	if err != nil {
		return false, "", err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada:
			return false, "", nil
		}
	}
	if abierta, err := existeRuntimeOrderAbiertaAgente(agente, msg.ProyectoID); err != nil {
		return false, "", err
	} else if abierta {
		return false, "", nil
	}
	return true, fmt.Sprintf("proyecto_id=%s sin runtime, sesion, asignacion, tareas ni ordenes abiertas", runtimeMailboxProyectoDetalle(msg.ProyectoID)), nil
}

func runtimeMailboxTienePresenciaRuntimeNoCanonica(agente string, proyectoID *int64) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false, nil
	}
	handles, err := db.ListarRuntimeHandlesPasivos(stringPtr(agente))
	if err != nil {
		return false, err
	}
	for _, handle := range handles {
		if handle == nil || !strings.EqualFold(strings.TrimSpace(handle.Agente), agente) {
			continue
		}
		if proyectoID != nil && *proyectoID > 0 {
			if handle.ProyectoID == nil || *handle.ProyectoID != *proyectoID {
				continue
			}
		}
		switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
		case "activo", "pausado":
			return true, nil
		}
	}
	return false, nil
}

func agentePerteneceAFlotaAutobootstrap(agente string) bool {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false
	}
	cfg := loadServerAutobootstrapConfig()
	if strings.EqualFold(agente, strings.TrimSpace(cfg.SupervisorAgent)) {
		return true
	}
	for _, worker := range cfg.WorkerAgents {
		if strings.EqualFold(agente, strings.TrimSpace(worker)) {
			return true
		}
	}
	return false
}

func enfriarAgentePorRuntimePanic(agente string, item *db.RuntimeTranscriptEntry) (string, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return "", nil
	}
	notes := make([]string, 0, 2)
	if item != nil && item.HandleID != nil && *item.HandleID > 0 {
		handle, err := db.GetRuntimeHandle(*item.HandleID)
		if err != nil {
			return "", err
		}
		if handle == nil {
			notes = append(notes, "runtime_panic_handle_missing")
		}
	}
	infoAgente, err := agenteIfExists(agente)
	if err != nil {
		return "", err
	}
	if infoAgente == nil {
		notes = append(notes, "runtime_panic_agent_missing")
		return strings.Join(notes, ";"), nil
	}
	cooldownSeconds := controlPlaneConfigIntOrDefault("runtime_panic_cooldown_seconds", 180)
	if cooldownSeconds <= 0 {
		cooldownSeconds = 180
	}
	reanimarAt := time.Now().UTC().Add(time.Duration(cooldownSeconds) * time.Second)
	if infoAgente != nil &&
		strings.EqualFold(strings.TrimSpace(infoAgente.EstadoCuota), "enfriamiento") &&
		infoAgente.ReanimarAt != nil &&
		infoAgente.ReanimarAt.After(reanimarAt) {
		notes = append(notes, "runtime_panic_cooldown_exists")
		return strings.Join(notes, ";"), nil
	}
	motivo := "Auto-pausa por runtime_panic"
	if item != nil && item.ID > 0 {
		motivo = fmt.Sprintf("Auto-pausa por runtime_panic: transcript=%d", item.ID)
	}
	if err := db.PausarAgenteHasta(agente, reanimarAt, motivo); err != nil {
		return "", err
	}
	db.Audit("orquesta", "runtime_panic_cooldown", "agente", 0,
		fmt.Sprintf("agente=%s cooldown_until=%s motivo=%s", agente, reanimarAt.Format(time.RFC3339), motivo))
	notes = append(notes, "runtime_panic_cooldown")
	return strings.Join(notes, ";"), nil
}

func procesarSignalTranscript(item *db.RuntimeTranscriptEntry) (string, error) {
	if item == nil {
		return "", nil
	}
	agente := strings.TrimSpace(item.Agente)
	if agente == "" {
		return "signal_sin_agente", nil
	}
	handle, err := resolverHandleEntregaTranscript(item)
	if err != nil {
		return "", err
	}
	if handle == nil {
		return "sin_runtime_activo", nil
	}
	if note, handled, err := resolverReviewGateDesdeSignalTranscript(item); err != nil {
		return "", err
	} else if handled {
		return note, nil
	}
	if esSignalFalloRuntime(item) {
		notes := make([]string, 0, 3)
		if cooldownNote, err := enfriarAgentePorRuntimePanic(agente, item); err != nil {
			return "", err
		} else if strings.TrimSpace(cooldownNote) != "" {
			notes = append(notes, strings.TrimSpace(cooldownNote))
		}
		if supervisorNote, err := notificarSupervisorSignalTranscript(item); err != nil {
			return "", err
		} else if strings.TrimSpace(supervisorNote) != "" {
			notes = append(notes, strings.TrimSpace(supervisorNote))
		}
		notes = append(notes, "runtime_failure_signal")
		return strings.Join(notes, ";"), nil
	}
	payload := map[string]any{
		"to_agente":      agente,
		"from_agente":    "orquesta",
		"texto":          construirRespuestaSignalTranscript(item),
		"classification": strings.TrimSpace(item.Classification),
		"transcript_id":  item.ID,
		"kind":           "instruction",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	order := &db.RuntimeOrder{
		Agente:      agente,
		ProyectoID:  item.ProyectoID,
		RuntimeID:   &item.RuntimeID,
		Tipo:        "send_instruction",
		PayloadJSON: string(raw),
	}
	if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		order.RuntimeID = handle.RuntimeID
	}
	if handle.ID > 0 {
		order.HandleID = &handle.ID
	} else if item.HandleID != nil && *item.HandleID > 0 {
		order.HandleID = item.HandleID
	}
	orderID, err := db.EncolarRuntimeOrder(order)
	if err != nil {
		return "", err
	}
	detail := fmt.Sprintf("transcript_id=%d agente=%s signal=%s", item.ID, agente, strings.TrimSpace(item.Classification))
	db.Audit("orquesta", "runtime_transcript_signal", "runtime_order", orderID, detail)
	payloadEvent, _ := json.Marshal(construirPayloadEventoAutoGuidance(item, orderID, order.Tipo, payload))
	eventPayload := construirPayloadEventoAutoGuidance(item, orderID, order.Tipo, payload)
	_, _ = db.RegistrarRuntimeEvent(&db.RuntimeEvent{
		RuntimeID:   item.RuntimeID,
		Kind:        "auto_guidance_sent",
		Level:       "info",
		Message:     fmt.Sprintf("Orquesta envió guía automática a %s tras %s", agente, strings.TrimSpace(item.Classification)),
		PayloadJSON: string(payloadEvent),
	})
	select {
	case db.CanalNotificaciones <- db.EventoNotificacion{
		Tipo:       "runtime_auto_guidance",
		ID:         orderID,
		Agente:     agente,
		Texto:      construirMensajeNotificacionAutoGuidance(item),
		ProyectoID: int64ProyectoRuntimeTranscript(item),
		Payload:    eventPayload,
	}:
	default:
	}
	notes := []string{fmt.Sprintf("auto_guidance_order:%d", orderID)}
	if supervisorNote, err := notificarSupervisorSignalTranscript(item); err != nil {
		return "", err
	} else if strings.TrimSpace(supervisorNote) != "" {
		notes = append(notes, strings.TrimSpace(supervisorNote))
	}
	return strings.Join(notes, ";"), nil
}

func construirPayloadEventoAutoGuidance(item *db.RuntimeTranscriptEntry, orderID int64, orderType string, instruction map[string]any) map[string]any {
	if item == nil {
		payload := map[string]any{
			"runtime_order_id":   orderID,
			"runtime_order_type": strings.TrimSpace(orderType),
		}
		if len(instruction) > 0 {
			payload["instruction"] = instruction
		}
		return payload
	}
	payload := map[string]any{
		"agente":             strings.TrimSpace(item.Agente),
		"classification":     strings.TrimSpace(item.Classification),
		"runtime_id":         item.RuntimeID,
		"runtime_order_id":   orderID,
		"runtime_order_type": strings.TrimSpace(orderType),
		"signal_text":        strings.TrimSpace(item.Text),
		"transcript_id":      item.ID,
	}
	if item.ProyectoID != nil && *item.ProyectoID > 0 {
		payload["proyecto_id"] = *item.ProyectoID
	}
	if item.HandleID != nil && *item.HandleID > 0 {
		payload["handle_id"] = *item.HandleID
	}
	if len(instruction) > 0 {
		payload["instruction"] = instruction
	}
	return payload
}

func construirMensajeNotificacionAutoGuidance(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return "Orquesta envió guía automática a un agente"
	}
	agente := strings.TrimSpace(item.Agente)
	classification := strings.TrimSpace(item.Classification)
	if agente == "" {
		agente = "agente-desconocido"
	}
	texto := strings.TrimSpace(item.Text)
	if texto == "" {
		return fmt.Sprintf("Orquesta envió guía automática a %s tras %s", agente, classification)
	}
	return fmt.Sprintf("Orquesta envió guía automática a %s tras %s: %s", agente, classification, texto)
}

func int64ProyectoRuntimeTranscript(item *db.RuntimeTranscriptEntry) int64 {
	if item == nil || item.ProyectoID == nil {
		return 0
	}
	return *item.ProyectoID
}

func resolverReviewGateDesdeSignalTranscript(item *db.RuntimeTranscriptEntry) (string, bool, error) {
	if item == nil || item.ProyectoID == nil || *item.ProyectoID <= 0 {
		return "", false, nil
	}
	var estado string
	switch strings.TrimSpace(item.Classification) {
	case "review_approved":
		estado = reviewapp.GateStateApproved
	case "review_changes_requested":
		estado = reviewapp.GateStateChangesAsked
	case "review_blocked":
		estado = reviewapp.GateStateBlocked
	default:
		return "", false, nil
	}
	proyecto, err := proyectoIfExists(strconv.FormatInt(*item.ProyectoID, 10))
	if err != nil || proyecto == nil {
		return "", false, err
	}
	gates, err := reviewService.List(reviewapp.ListInput{
		ProyectoRef:    proyecto.Slug,
		ReviewerAgente: strings.TrimSpace(item.Agente),
		Limit:          20,
	})
	if err != nil {
		return "", false, err
	}
	gate := firstOpenGate(gates)
	if gate == nil {
		return "", false, nil
	}
	findingsJSON, err := construirFindingsReviewTranscript(item)
	if err != nil {
		return "", false, err
	}
	resolved, err := reviewService.Resolve(reviewapp.ResolveGateInput{
		ID:             gate.ID,
		Estado:         estado,
		ReviewerAgente: strings.TrimSpace(item.Agente),
		FindingsJSON:   findingsJSON,
	})
	if err != nil {
		return "", false, err
	}
	detail := fmt.Sprintf("gate=%d transcript=%d estado=%s", resolved.ID, item.ID, strings.TrimSpace(estado))
	db.Audit("orquesta", "runtime_transcript_review_resolution", "review_gate", resolved.ID, detail)
	return "review_gate_resolved:" + detail, true, nil
}

func construirFindingsReviewTranscript(item *db.RuntimeTranscriptEntry) (string, error) {
	payload := []map[string]any{{
		"source":         "runtime_transcript",
		"transcript_id":  item.ID,
		"classification": strings.TrimSpace(item.Classification),
		"agente":         strings.TrimSpace(item.Agente),
		"text":           strings.TrimSpace(item.Text),
		"runtime_id":     item.RuntimeID,
		"handle_id":      item.HandleID,
	}}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func esSignalFalloRuntime(item *db.RuntimeTranscriptEntry) bool {
	if item == nil {
		return false
	}
	switch strings.TrimSpace(item.Classification) {
	case "runtime_panic", "runtime_crash":
		return true
	default:
		return false
	}
}

func construirRespuestaSignalTranscript(item *db.RuntimeTranscriptEntry) string {
	base := "Orquesta: continua de forma autonoma dentro de la gobernanza efectiva del proyecto. Usa API/servicios para el flujo normal, no BD local salvo diagnostico o recuperacion, y mantén tests coherentes y eficientes. Si el cambio es delicado, crea checkpoint y sigue."
	normalized := normalizarSignalTranscriptTexto(item)
	politicaPermisos := construirPoliticaPermisosSignalTranscript(strings.TrimSpace(item.Classification), normalized)
	switch strings.TrimSpace(item.Classification) {
	case "approval_request":
		return base + " " + politicaPermisos + " Si dudas entre varias opciones seguras, elige la mas alineada con el proyecto y continua sin detenerte."
	case "waiting_human":
		return base + " " + politicaPermisos + " No te quedes esperando respuesta: formula el siguiente paso razonable, ejecuta y documenta los supuestos."
	case "ready_for_review":
		return base + " Deja un resumen breve, asegurate de que el frente queda verificable y sigue disponible para que Orquesta relance review si procede."
	case "needs_replan":
		return base + " Si no ves el siguiente paso, revisa backlog, tareas, propuestas y ultimo checkpoint, y continua por el frente mas util disponible."
	case "blocked":
		return base + " Si el bloqueo es de contexto, reevalua tareas, propuestas y estado del proyecto y avanza por el mejor siguiente paso disponible."
	default:
		return base
	}
}

func normalizarSignalTranscriptTexto(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return ""
	}
	texto := strings.TrimSpace(item.NormalizedText)
	if texto == "" {
		texto = strings.TrimSpace(strings.ToLower(item.Text))
	}
	texto = strings.ReplaceAll(texto, "\n", " ")
	texto = strings.ReplaceAll(texto, "\r", " ")
	return strings.TrimSpace(texto)
}

func construirPoliticaPermisosSignalTranscript(classification, normalized string) string {
	if classification != "approval_request" && classification != "waiting_human" {
		return ""
	}
	if textoPareceAccionDestructiva(normalized) {
		return "No autorizado automaticamente para acciones destructivas o irreversibles. No uses rm -rf, git reset --hard, borrados masivos, drops de base de datos ni cambios fuera del workspace. Replantea una alternativa segura."
	}
	if textoPareceDependenciaExterna(normalized) {
		return "No inventes credenciales, secretos ni permisos externos. Si de verdad faltan, deja trazabilidad del bloqueo y continua por otro frente util del proyecto."
	}
	if textoPareceAccionNormalAutonoma(normalized) {
		return "Aprobado automaticamente para acciones normales dentro del workspace: editar codigo, refactorizar, crear o ajustar archivos, ejecutar build/test/lint, preparar worktrees y continuar el siguiente paso util. No necesitas confirmacion humana para seguir. " + detalleAccionNormalAutonoma(normalized)
	}
	return "Aprobado automaticamente para trabajo normal dentro del proyecto. No necesitas confirmacion humana para seguir por una opcion segura."
}

func textoPareceAccionNormalAutonoma(normalized string) bool {
	for _, token := range []string{
		"refactor",
		"seguir",
		"continuar",
		"go test",
		"npm test",
		"pnpm test",
		"yarn test",
		"pytest",
		"cargo test",
		"build",
		"lint",
		"gofmt",
		"formate",
		"crear archivo",
		"editar",
		"ajustar",
		"implementar",
		"worktree",
		"rama",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func detalleAccionNormalAutonoma(normalized string) string {
	switch {
	case strings.Contains(normalized, "go test"):
		return "Puedes ejecutar go test y seguir con el ciclo normal."
	case strings.Contains(normalized, "pytest"):
		return "Puedes ejecutar pytest y seguir con el ciclo normal."
	case strings.Contains(normalized, "npm test"), strings.Contains(normalized, "pnpm test"), strings.Contains(normalized, "yarn test"):
		return "Puedes ejecutar los tests del frontend y seguir con el ciclo normal."
	case strings.Contains(normalized, "build"):
		return "Puedes ejecutar el build y seguir con el ciclo normal."
	case strings.Contains(normalized, "lint"):
		return "Puedes ejecutar lint y seguir con el ciclo normal."
	case strings.Contains(normalized, "refactor"):
		return "Puedes continuar con el refactor sin esperar confirmacion humana."
	default:
		return "Sigue por la opcion mas segura y util."
	}
}

func textoPareceAccionDestructiva(normalized string) bool {
	for _, token := range []string{
		"rm -rf",
		"rm -f",
		"git reset --hard",
		"git checkout --",
		"drop table",
		"drop database",
		"truncate table",
		"borrar la base de datos",
		"borrar la bd",
		"delete database",
		"wipe",
		"formatear disco",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func textoPareceDependenciaExterna(normalized string) bool {
	for _, token := range []string{
		"credencial",
		"credenciales",
		"secret",
		"secreto",
		"api key",
		"token",
		"ssh key",
		"oauth",
		"permiso externo",
		"acceso externo",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func notificarSupervisorSignalTranscript(item *db.RuntimeTranscriptEntry) (string, error) {
	if item == nil || item.ProyectoID == nil || *item.ProyectoID <= 0 {
		return "", nil
	}
	policy, err := db.GetProyectoAutonomia(*item.ProyectoID)
	if err != nil || policy == nil || !policy.Enabled {
		return "", err
	}
	proyecto, err := proyectoIfExists(strconv.FormatInt(*item.ProyectoID, 10))
	if err != nil || proyecto == nil {
		return "", err
	}
	if strings.TrimSpace(item.Classification) == "ready_for_review" {
		if note, err := notificarReviewerSignalTranscript(item, policy, proyecto); err != nil {
			return "", err
		} else if strings.TrimSpace(note) != "" {
			return note, nil
		}
	}
	notes := make([]string, 0, 3)
	supervisor, activado, err := resolverSupervisorAutonomiaOperativo(proyecto.ID, policy, "supervision_transcript_signal", strings.TrimSpace(item.Agente))
	if err != nil {
		return "", err
	}
	if activado {
		notes = append(notes, "supervisor_start")
	}
	if note, err := asegurarTareaReplanAutonomiaSignal(item, policy, proyecto, supervisor); err != nil {
		return "", err
	} else if strings.TrimSpace(note) != "" {
		notes = append(notes, note)
	}
	if supervisor == nil || strings.EqualFold(strings.TrimSpace(supervisor.Nombre), strings.TrimSpace(item.Agente)) {
		return strings.Join(notes, ";"), nil
	}
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(supervisor.Nombre, &proyecto.ID, "nudge", "inspeccionar_transcript_signal"); err != nil {
		return "", err
	} else if pendiente {
		notes = append(notes, "supervisor_nudge_pendiente")
		return strings.Join(notes, ";"), nil
	}
	resumen := fmt.Sprintf("agente=%s signal=%s transcript=%d", strings.TrimSpace(item.Agente), strings.TrimSpace(item.Classification), item.ID)
	if encolada, err := encolarNudgeAutonomiaDetallado(supervisor.Nombre, proyecto, "inspeccionar_transcript_signal", resumen, construirInstruccionSupervisionSignalTranscript(policy, proyecto, supervisor.Nombre, item), map[string]any{
		"signal_classification": strings.TrimSpace(item.Classification),
		"signal_transcript_id":  item.ID,
		"signal_agente":         strings.TrimSpace(item.Agente),
		"signal_text":           strings.TrimSpace(item.Text),
	}); err != nil {
		return "", err
	} else if !encolada {
		notes = append(notes, "supervisor_nudge_omitido")
		return strings.Join(notes, ";"), nil
	}
	notes = append(notes, "supervisor_nudged")
	return strings.Join(notes, ";"), nil
}

func asegurarTareaReplanAutonomiaSignal(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto, supervisor *db.Agente) (string, error) {
	if item == nil || policy == nil || proyecto == nil || !policy.Enabled {
		return "", nil
	}
	if strings.TrimSpace(item.Classification) != "needs_replan" {
		return "", nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyecto.ID})
	if err != nil {
		return "", err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaCompletada, db.TareaCancelada:
			continue
		}
		if strings.EqualFold(strings.TrimSpace(tarea.Titulo), autonomiaReplanTaskTitle) || strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:needs_replan") {
			return "replan_task_exists", nil
		}
	}
	target := ""
	if supervisor != nil && !strings.EqualFold(strings.TrimSpace(supervisor.Nombre), strings.TrimSpace(item.Agente)) {
		target = strings.TrimSpace(supervisor.Nombre)
	} else if preferred := strings.TrimSpace(policy.SupervisorAgente); preferred != "" && !strings.EqualFold(preferred, strings.TrimSpace(item.Agente)) {
		target = preferred
	}
	id, err := tareasService.Create(tareasapp.CreateTaskInput{
		Titulo:      autonomiaReplanTaskTitle,
		Descripcion: construirDescripcionTareaReplanAutonomia(policy, proyecto, item),
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Agente:      target,
		Proyecto:    proyecto.Slug,
		Notas:       fmt.Sprintf("autonomia:needs_replan;transcript:%d;agente_origen:%s", item.ID, strings.TrimSpace(item.Agente)),
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(target) != "" {
		if err := tareasService.Start(id, target); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("replan_task_created:%d", id), nil
}

func construirDescripcionTareaReplanAutonomia(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, item *db.RuntimeTranscriptEntry) string {
	partes := []string{
		"Orquesta ha detectado en el transcript una señal de needs_replan: el agente no tiene claro el siguiente paso útil.",
		"Replanifica backlog, prioridades, propuestas, worktrees y checkpoints para abrir o reforzar el siguiente frente útil sin intervención humana.",
	}
	if proyecto != nil && strings.TrimSpace(proyecto.Slug) != "" {
		partes = append(partes, "Proyecto: "+strings.TrimSpace(proyecto.Slug)+".")
	}
	if item != nil && strings.TrimSpace(item.Agente) != "" {
		partes = append(partes, "Agente origen: "+strings.TrimSpace(item.Agente)+".")
	}
	if item != nil && strings.TrimSpace(item.Text) != "" {
		partes = append(partes, "Señal transcript: "+strings.TrimSpace(item.Text)+".")
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		partes = append(partes, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	if policy != nil && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "" && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "{}" {
		partes = append(partes, "Definition of done: "+strings.TrimSpace(policy.DefinitionOfDoneJSON)+".")
	}
	return strings.Join(partes, " ")
}

func notificarReviewerSignalTranscript(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto) (string, error) {
	if item == nil || proyecto == nil || policy == nil || !policy.Enabled {
		return "", nil
	}
	reviewer, activado, err := prepararAgentePreferidoAutonomia(proyecto.ID, strings.TrimSpace(policy.ReviewerAgente), "review_transcript_signal")
	if err != nil {
		return "", err
	}
	if activado {
		return "reviewer_start", nil
	}
	if reviewer == nil {
		reviewer, err = seleccionarAgenteActivoProyectoPreferido(proyecto.ID, strings.TrimSpace(policy.ReviewerAgente), []string{"revisor", "reviewer", "supervisor", "orquestador", "admin", "programador"}, strings.TrimSpace(item.Agente))
		if err != nil {
			return "", err
		}
	}
	if reviewer == nil || strings.EqualFold(strings.TrimSpace(reviewer.Nombre), strings.TrimSpace(item.Agente)) {
		return "", nil
	}
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(reviewer.Nombre, &proyecto.ID, "nudge", "inspeccionar_ready_for_review"); err != nil {
		return "", err
	} else if pendiente {
		return "reviewer_nudge_pendiente", nil
	}
	resumen := fmt.Sprintf("agente=%s signal=%s transcript=%d", strings.TrimSpace(item.Agente), strings.TrimSpace(item.Classification), item.ID)
	if encolada, err := encolarNudgeAutonomiaDetallado(reviewer.Nombre, proyecto, "inspeccionar_ready_for_review", resumen, construirInstruccionReviewSignalTranscript(policy, proyecto, reviewer.Nombre, item), map[string]any{
		"signal_classification": strings.TrimSpace(item.Classification),
		"signal_transcript_id":  item.ID,
		"signal_agente":         strings.TrimSpace(item.Agente),
		"signal_text":           strings.TrimSpace(item.Text),
	}); err != nil {
		return "", err
	} else if !encolada {
		return "reviewer_nudge_omitido", nil
	}
	return "reviewer_nudged", nil
}

func construirInstruccionSupervisionSignalTranscript(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, supervisor string, item *db.RuntimeTranscriptEntry) string {
	parts := []string{
		"Orquesta: un agente del proyecto ha emitido una señal de duda, espera o bloqueo en su transcript.",
		"Supervisa el frente ahora mismo: revisa el transcript, el contexto del proyecto, las tareas y propuestas activas, y decide el siguiente paso sin escalar a humano salvo que falten credenciales, secretos o un recurso externo real.",
	}
	if proyecto != nil {
		parts = append(parts, fmt.Sprintf("Proyecto: %s.", strings.TrimSpace(proyecto.Slug)))
	}
	if strings.TrimSpace(supervisor) != "" {
		parts = append(parts, "Supervisor responsable: "+strings.TrimSpace(supervisor)+".")
	}
	if item != nil {
		if strings.TrimSpace(item.Agente) != "" {
			parts = append(parts, "Agente origen: "+strings.TrimSpace(item.Agente)+".")
		}
		if strings.TrimSpace(item.Classification) != "" {
			parts = append(parts, "Clasificación: "+strings.TrimSpace(item.Classification)+".")
		}
		if strings.TrimSpace(item.Text) != "" {
			parts = append(parts, "Fragmento detectado: "+strings.TrimSpace(item.Text)+".")
		}
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		parts = append(parts, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	if policy != nil && policy.AutoCreateTasks {
		parts = append(parts, "Si basta con una instrucción, emítela; si hace falta, crea o reajusta tareas, propuestas o handoff para que el proyecto no se quede parado.")
	} else {
		parts = append(parts, "Si basta con una instrucción, emítela; si hacen falta cambios estructurales, deja el siguiente frente claro sin crear tareas nuevas automáticamente.")
	}
	return strings.Join(parts, " ")
}

func construirInstruccionReviewSignalTranscript(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, reviewer string, item *db.RuntimeTranscriptEntry) string {
	parts := []string{
		"Orquesta: un agente del proyecto ha indicado en su transcript que el frente está listo para revisión.",
		"Valida el estado real del código, la definición de terminado, tests, arquitectura e i18n; si el frente está maduro, impulsa o ejecuta la revisión, y si no, pide cambios concretos sin bloquear el proyecto más de lo necesario.",
	}
	if proyecto != nil {
		parts = append(parts, fmt.Sprintf("Proyecto: %s.", strings.TrimSpace(proyecto.Slug)))
	}
	if strings.TrimSpace(reviewer) != "" {
		parts = append(parts, "Reviewer responsable: "+strings.TrimSpace(reviewer)+".")
	}
	if item != nil {
		if strings.TrimSpace(item.Agente) != "" {
			parts = append(parts, "Agente origen: "+strings.TrimSpace(item.Agente)+".")
		}
		if strings.TrimSpace(item.Text) != "" {
			parts = append(parts, "Fragmento detectado: "+strings.TrimSpace(item.Text)+".")
		}
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		parts = append(parts, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	parts = append(parts, "Si aún falta trabajo, conviértelo en feedback accionable; si está listo, deja trazabilidad de revisión sin pedir intervención humana por defecto.")
	return strings.Join(parts, " ")
}

func controlPlaneConfigBoolOrDefault(clave string, fallback bool) bool {
	v, err := controlPlaneConfigGetCached(clave)
	if err != nil {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "si", "sí", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func procesarRuntimeMailboxInteractivoBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	return procesarRuntimeMailboxInteractivoBatchConMailbox(mailbox, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
}

func procesarRuntimeMailboxInteractivoBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		texto, ok := construirInstruccionMailboxInteractivo(msg)
		if !ok {
			continue
		}
		if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
			return total, err
		}
		if !runtimeMailboxSiguePendiente(msg.ID) {
			continue
		}
		handle, err := snapshot.activeHandle(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		if obsoleta, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg, handle, "interactive", time.Now().UTC()); err != nil {
			return total, err
		} else if obsoleta {
			consumed[msg.ID] = struct{}{}
			total++
			continue
		}
		if db.RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliveryInteractive {
			continue
		}
		if !runtimeHandleListaParaDispatchInteractivo(handle) {
			continue
		}
		if !runtimeMailboxShouldReevaluate("interactive", msg.ID, handle.ID) {
			continue
		}
		if abierta, err := existeRuntimeOrderAbiertaPorHandleEnSnapshot(snapshot, msg, handle.ID, "send_instruction"); err != nil {
			return total, err
		} else if abierta {
			continue
		}
		if dedupe, err := existeIntentoSendInstructionMailboxParaHandleEnSnapshot(snapshot, msg, handle, ""); err != nil {
			return total, err
		} else if dedupe {
			continue
		}
		orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, texto, "")
		if err != nil {
			return total, err
		}
		if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
			superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
			if err != nil {
				return total, err
			}
			if superseded > 0 {
				db.Audit("orquesta", "runtime_mailbox_interactivo_supersede", "runtime_mailbox", msg.ID,
					fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
			}
		}
		db.Audit("orquesta", "runtime_mailbox_interactivo", "runtime_order", orderID,
			fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
		consumed[msg.ID] = struct{}{}
		total++
	}
	return total, nil
}

func procesarRuntimeMailboxSupervisorLocalBatch() (int, error) {
	return 0, nil
}

func procesarRuntimeMailboxSupervisorLocalBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	return 0, nil
}

func procesarRuntimeMailboxCoordinatedRestartBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	return procesarRuntimeMailboxCoordinatedRestartBatchConMailbox(mailbox, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
}

func procesarRuntimeMailboxCoordinatedRestartBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	type recycleKey struct {
		agente     string
		proyectoID int64
	}
	latest := make(map[recycleKey]*db.RuntimeMailboxMessage)
	for _, msg := range mailbox {
		if msg == nil || msg.ProyectoID == nil || *msg.ProyectoID <= 0 {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
			return 0, err
		}
		if !runtimeMailboxSiguePendiente(msg.ID) {
			continue
		}
		handle, err := snapshot.activeHandle(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return 0, err
		}
		if !runtimeMailboxDebeCoordinarReinicio(strings.TrimSpace(msg.Kind), handle) {
			continue
		}
		if pendiente, err := existeRuntimeOrderAbiertaAutonomia(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, "stop", "start", "restart", "resume", "handoff"); err != nil {
			return 0, err
		} else if pendiente {
			continue
		}
		key := recycleKey{
			agente:     strings.TrimSpace(msg.ToAgente),
			proyectoID: *msg.ProyectoID,
		}
		if prev := latest[key]; prev == nil || msg.ID > prev.ID {
			latest[key] = msg
		}
	}
	keys := make([]recycleKey, 0, len(latest))
	for key := range latest {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].proyectoID == keys[j].proyectoID {
			return keys[i].agente < keys[j].agente
		}
		return keys[i].proyectoID < keys[j].proyectoID
	})
	total := 0
	for _, key := range keys {
		msg := latest[key]
		if msg == nil {
			continue
		}
		proyecto, err := snapshot.project(key.proyectoID)
		if err != nil || proyecto == nil {
			return total, err
		}
		handle, err := snapshot.activeHandle(key.agente, &key.proyectoID)
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		if obsoleta, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg, handle, "coordinated_restart", time.Now().UTC()); err != nil {
			return total, err
		} else if obsoleta {
			total++
			continue
		}
		motivo := fmt.Sprintf("runtime_mailbox_coordinated_restart:%s:%d", strings.TrimSpace(msg.Kind), msg.ID)
		stopOrderID, startOrderID, err := encolarReinicioCoordinadoMailbox(handle, proyecto, msg, motivo)
		if err != nil {
			return total, err
		}
		if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
			superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
			if err != nil {
				return total, err
			}
			if superseded > 0 {
				db.Audit("orquesta", "runtime_mailbox_coordinated_restart_supersede", "runtime_mailbox", msg.ID,
					fmt.Sprintf("agente=%s kind=%s superseded=%d", key.agente, strings.TrimSpace(msg.Kind), superseded))
			}
		}
		db.Audit("orquesta", "runtime_mailbox_coordinated_restart", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s proyecto=%s kind=%s stop_order_id=%d start_order_id=%d", key.agente, proyecto.Slug, strings.TrimSpace(msg.Kind), stopOrderID, startOrderID))
		total++
	}
	return total, nil
}

func procesarRuntimeMailboxSessionResumeBatch() (int, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	return procesarRuntimeMailboxSessionResumeBatchConMailbox(mailbox, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
}

func procesarRuntimeMailboxSessionResumeBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		texto, ok := construirInstruccionMailboxInteractivo(msg)
		if !ok {
			continue
		}
		if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
			return total, err
		}
		if !runtimeMailboxSiguePendiente(msg.ID) {
			continue
		}
		handle, err := snapshot.activeHandle(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		if !runtimeMailboxShouldReevaluate("session_resume", msg.ID, handle.ID) {
			continue
		}
		runtime, err := runtimeCanonicoDesdeHandleAgenteProyecto(handle, strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		handle, externalSessionID, err := snapshot.externalSessionHandle(handle, runtime)
		if err != nil {
			return total, err
		}
		if handle == nil || strings.TrimSpace(externalSessionID) == "" {
			continue
		}
		if obsoleta, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg, handle, "session_resume", time.Now().UTC()); err != nil {
			return total, err
		} else if obsoleta {
			consumed[msg.ID] = struct{}{}
			total++
			continue
		}
		if covered, _, _, err := db.RuntimeMailboxCubiertoPorBootstrapPendiente(msg.ID, handle, runtime); err != nil {
			return total, err
		} else if covered {
			continue
		}
		if db.RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliverySessionResume {
			continue
		}
		if abierta, err := existeRuntimeOrderAbiertaPorHandleEnSnapshot(snapshot, msg, handle.ID, "send_instruction"); err != nil {
			return total, err
		} else if abierta {
			continue
		}
		if dedupe, err := existeIntentoSendInstructionMailboxParaHandleEnSnapshot(snapshot, msg, handle, externalSessionID); err != nil {
			return total, err
		} else if dedupe {
			continue
		}
		orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, texto, externalSessionID)
		if err != nil {
			return total, err
		}
		if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
			superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
			if err != nil {
				return total, err
			}
			if superseded > 0 {
				db.Audit("orquesta", "runtime_mailbox_session_resume_supersede", "runtime_mailbox", msg.ID,
					fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
			}
		}
		db.Audit("orquesta", "runtime_mailbox_session_resume", "runtime_order", orderID,
			fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
		consumed[msg.ID] = struct{}{}
		total++
	}
	return total, nil
}

func procesarRuntimeMailboxBootstrapTMUXBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		texto, ok := construirInstruccionMailboxInteractivo(msg)
		if !ok {
			continue
		}
		if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
			return total, err
		}
		if !runtimeMailboxSiguePendiente(msg.ID) {
			continue
		}
		handle, err := snapshot.activeHandle(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return total, err
		}
		if handle == nil {
			continue
		}
		if !runtimeMailboxShouldReevaluate("bootstrap_tmux", msg.ID, handle.ID) {
			continue
		}
		if obsoleta, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg, handle, "bootstrap_tmux", time.Now().UTC()); err != nil {
			return total, err
		} else if obsoleta {
			consumed[msg.ID] = struct{}{}
			total++
			continue
		}
		if db.RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliveryBootstrapOnly {
			continue
		}
		if !runtimeHandleListaParaDispatchBootstrapTMUX(handle) {
			continue
		}
		if abierta, err := existeRuntimeOrderAbiertaPorHandleEnSnapshot(snapshot, msg, handle.ID, "send_instruction"); err != nil {
			return total, err
		} else if abierta {
			continue
		}
		if dedupe, err := existeIntentoSendInstructionMailboxParaHandleEnSnapshot(snapshot, msg, handle, ""); err != nil {
			return total, err
		} else if dedupe {
			continue
		}
		orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, texto, "")
		if err != nil {
			return total, err
		}
		if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
			superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
			if err != nil {
				return total, err
			}
			if superseded > 0 {
				db.Audit("orquesta", "runtime_mailbox_bootstrap_tmux_supersede", "runtime_mailbox", msg.ID,
					fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
			}
		}
		db.Audit("orquesta", "runtime_mailbox_bootstrap_tmux", "runtime_order", orderID,
			fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
		consumed[msg.ID] = struct{}{}
		total++
	}
	return total, nil
}

func runtimeHandleListaParaDispatchBootstrapTMUX(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON))
	if err != nil || snap == nil {
		return false
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil || view.HeartbeatStale || !view.Alive {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(view.Driver), "tmux_cli_session") {
		return false
	}
	ready, _ := snap.ReadyForTextDispatch(time.Now().UTC(), time.Minute)
	return ready
}

func runtimeHandleListaParaDispatchInteractivo(handle *db.RuntimeHandle) bool {
	if handle == nil || !db.RuntimeHandlePermiteSendInputInteractivo(handle) {
		return false
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON))
	if err != nil || snap == nil {
		return true
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil {
		return true
	}
	if !strings.EqualFold(strings.TrimSpace(view.Driver), "tmux_cli_session") &&
		!strings.EqualFold(strings.TrimSpace(view.Transport), "tmux") {
		return true
	}
	ready, _ := snap.ReadyForTextDispatch(time.Now().UTC(), time.Minute)
	return ready
}

func encolarSendInstructionDesdeRuntimeMailbox(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, texto, externalSessionID string) (int64, error) {
	if msg == nil || handle == nil {
		return 0, nil
	}
	texto = compactarInstruccionMailboxEntrega(texto)
	texto = controlruntime.NormalizarInstruccionProceso(controlruntime.ObjetivoProceso{
		HandleKind:   handle.HandleKind,
		HandleRef:    handle.HandleRef,
		MetadataJSON: handle.MetadataJSON,
	}, texto)
	payload := map[string]any{
		"to_agente":                  strings.TrimSpace(msg.ToAgente),
		"from_agente":                strings.TrimSpace(msg.FromAgente),
		"texto":                      strings.TrimSpace(texto),
		"mailbox_id":                 msg.ID,
		"mailbox_kind":               strings.TrimSpace(msg.Kind),
		"delivery_attempt_signature": runtimeMailboxDeliveryAttemptSignature(handle, externalSessionID),
	}
	if strings.TrimSpace(externalSessionID) != "" {
		payload["external_session_id"] = strings.TrimSpace(externalSessionID)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	order := &db.RuntimeOrder{
		Agente:      strings.TrimSpace(msg.ToAgente),
		ProyectoID:  msg.ProyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: string(raw),
	}
	runtimeID, err := runtimeIDCanonicoDesdeHandleAgenteProyecto(handle, strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil {
		return 0, err
	}
	if runtimeID != nil {
		order.RuntimeID = runtimeID
	} else if handle.RuntimeID != nil {
		order.RuntimeID = handle.RuntimeID
	}
	order.HandleID = &handle.ID
	return db.EncolarRuntimeOrder(order)
}

func compactarInstruccionMailboxEntrega(texto string) string {
	lower := strings.ToLower(strings.TrimSpace(texto))
	switch {
	case strings.Contains(lower, "se te ha asignado automaticamente la tarea"),
		strings.Contains(lower, "se te ha asignado la tarea"):
		return "toma tarea asignada y sigue"
	case strings.Contains(lower, "has sido arrancado como programador"),
		strings.Contains(lower, "has sido arrancado como agente"),
		strings.Contains(lower, "has sido arrancado como"):
		return "continua trabajo actual"
	case strings.Contains(lower, "supervisor autonomo"),
		strings.Contains(lower, "orquestador autonomo"),
		strings.Contains(lower, "arrancado como orquestador"),
		strings.Contains(lower, "supervisar_proyecto"),
		strings.Contains(lower, "supervision_transcript_signal"):
		return "supervisa proyecto actual y sigue"
	default:
		return strings.TrimSpace(texto)
	}
}

func existeIntentoSendInstructionMailboxParaHandle(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, externalSessionID string) (bool, error) {
	if msg == nil || handle == nil || msg.ID <= 0 {
		return false, nil
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if agente == "" {
		return false, nil
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: msg.ProyectoID})
	if err != nil {
		return false, err
	}
	currentSessionID := strings.TrimSpace(externalSessionID)
	currentSignature := runtimeMailboxDeliveryAttemptSignature(handle, externalSessionID)
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" {
			continue
		}
		if runtimeOrderMailboxIDFromJSON(order.PayloadJSON) != msg.ID {
			continue
		}
		if order.HandleID != nil && *order.HandleID != handle.ID {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			return true, nil
		case "completada":
			if !runtimeOrderMailboxOnlyResult(order.ResultadoJSON) {
				continue
			}
			if runtimeOrderMailboxOnlyExpired(order) {
				continue
			}
			orderSignature := runtimeOrderDeliveryAttemptSignature(order.PayloadJSON)
			if currentSignature != "" {
				if orderSignature == "" {
					continue
				}
				if currentSignature != orderSignature {
					continue
				}
				return true, nil
			}
			orderSessionID := strings.TrimSpace(runtimeOrderExternalSessionIDFromJSON(order.PayloadJSON))
			if currentSessionID != "" && orderSessionID != "" && currentSessionID != orderSessionID {
				continue
			}
			return true, nil
		}
	}
	return false, nil
}

func existeIntentoSendInstructionMailboxParaHandleEnSnapshot(snapshot *runtimeMailboxBatchSnapshot, msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, externalSessionID string) (bool, error) {
	if snapshot == nil {
		return existeIntentoSendInstructionMailboxParaHandle(msg, handle, externalSessionID)
	}
	if msg == nil || handle == nil || msg.ID <= 0 {
		return false, nil
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if agente == "" {
		return false, nil
	}
	orders, err := snapshot.ordersForAgentProject(agente, msg.ProyectoID)
	if err != nil {
		return false, err
	}
	currentSessionID := strings.TrimSpace(externalSessionID)
	currentSignature := runtimeMailboxDeliveryAttemptSignature(handle, externalSessionID)
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" {
			continue
		}
		if runtimeOrderMailboxIDFromJSON(order.PayloadJSON) != msg.ID {
			continue
		}
		if order.HandleID != nil && *order.HandleID != handle.ID {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			return true, nil
		case "completada":
			if !runtimeOrderMailboxOnlyResult(order.ResultadoJSON) {
				continue
			}
			if runtimeOrderMailboxOnlyExpired(order) {
				continue
			}
			orderSignature := runtimeOrderDeliveryAttemptSignature(order.PayloadJSON)
			if currentSignature != "" {
				if orderSignature == "" {
					continue
				}
				if currentSignature != orderSignature {
					continue
				}
				return true, nil
			}
			orderSessionID := strings.TrimSpace(runtimeOrderExternalSessionIDFromJSON(order.PayloadJSON))
			if currentSessionID != "" && orderSessionID != "" && currentSessionID != orderSessionID {
				continue
			}
			return true, nil
		}
	}
	return false, nil
}

func runtimeOrderMailboxOnlyExpired(order *db.RuntimeOrder) bool {
	if order == nil {
		return false
	}
	if runtimeOrderMailboxOnlySticky(order) {
		return false
	}
	reference := order.UpdatedAt
	if order.FinishedAt != nil && !order.FinishedAt.IsZero() {
		reference = order.FinishedAt.UTC()
	}
	if reference.IsZero() {
		reference = order.CreatedAt
	}
	if reference.IsZero() {
		return false
	}
	return time.Since(reference.UTC()) > runtimeMailboxOnlyDedupeTTL()
}

func runtimeOrderMailboxOnlySticky(order *db.RuntimeOrder) bool {
	if order == nil || !runtimeOrderMailboxOnlyResult(order.ResultadoJSON) {
		return false
	}
	switch strings.TrimSpace(runtimeOrderMailboxKindFromJSON(order.PayloadJSON)) {
	case "autonomia", "nudge", "watchdog", db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func runtimeMailboxOnlyDedupeTTL() time.Duration {
	seconds := configIntOrDefault("runtime_send_instruction_retry_seconds", 15)
	if seconds <= 0 {
		seconds = 15
	}
	return time.Duration(seconds) * time.Second
}

func runtimeOrderMailboxIDFromJSON(raw string) int64 {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return 0
	}
	value, _ := payload["mailbox_id"]
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}

func runtimeOrderExternalSessionIDFromJSON(raw string) string {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return ""
	}
	value, _ := payload["external_session_id"]
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func runtimeOrderDeliveryAttemptSignature(raw string) string {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return ""
	}
	if text, ok := payload["delivery_attempt_signature"].(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func runtimeOrderMailboxKindFromJSON(raw string) string {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return ""
	}
	if text, ok := payload["mailbox_kind"].(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func runtimeMailboxDeliveryAttemptSignature(handle *db.RuntimeHandle, externalSessionID string) string {
	if handle == nil || handle.ID <= 0 {
		return ""
	}
	mode := strings.TrimSpace(string(db.RuntimeHandleMailboxDeliveryMode(handle)))
	if mode == "" {
		return ""
	}
	return fmt.Sprintf("%s|handle:%d|session:%s", mode, handle.ID, strings.TrimSpace(externalSessionID))
}

func runtimeOrderMailboxOnlyResult(raw string) bool {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return false
	}
	value, _ := payload["mailbox_only"]
	flag, ok := value.(bool)
	return ok && flag
}

func runtimeOrderDeferredReason(raw string) string {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return ""
	}
	if text, ok := payload["deferred_reason"].(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func encolarReinicioCoordinadoMailbox(handle *db.RuntimeHandle, proyecto *db.Proyecto, msg *db.RuntimeMailboxMessage, motivo string) (int64, int64, error) {
	if handle == nil || msg == nil {
		return 0, 0, nil
	}
	return db.EncolarReinicioCoordinadoRuntimeHandle(handle, proyectoIDPtr(proyecto), motivo, "orquesta")
}

func proyectoIDPtr(proyecto *db.Proyecto) *int64 {
	if proyecto == nil || proyecto.ID <= 0 {
		return nil
	}
	return &proyecto.ID
}

func runtimeMailboxKindCoalescible(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "instruction", "autonomia", "nudge", "watchdog", db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func runtimeMailboxKindEphemeral(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "autonomia", "nudge", "watchdog", db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func coalescerRuntimeMailboxPendiente(msg *db.RuntimeMailboxMessage) error {
	if msg == nil || !runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
		return nil
	}
	superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
	if err != nil {
		return err
	}
	if superseded > 0 {
		db.Audit("orquesta", "runtime_mailbox_supersede", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
	}
	return nil
}

func runtimeMailboxSiguePendiente(id int64) bool {
	if id <= 0 {
		return false
	}
	msg, err := db.GetRuntimeMailbox(id)
	if err != nil || msg == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(msg.Estado), "pendiente")
}

func consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, lane string, now time.Time) (bool, error) {
	if msg == nil || handle == nil || !runtimeMailboxKindEphemeral(strings.TrimSpace(msg.Kind)) {
		return false, nil
	}
	if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
		return false, err
	} else if !canConsume {
		return false, nil
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON))
	if err != nil || snap == nil {
		return false, err
	}
	view := snap.View(now.UTC(), 90*time.Second)
	if view == nil || !view.Alive || view.HeartbeatStale {
		return false, nil
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "running", "idle", "waiting_input":
	default:
		return false, nil
	}
	if view.StartedAt == nil || view.StartedAt.IsZero() {
		return false, nil
	}
	startedAt := view.StartedAt.UTC()
	if !msg.CreatedAt.UTC().Before(startedAt) {
		return false, nil
	}
	if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_obsoleta_worker", "runtime_mailbox", msg.ID,
		fmt.Sprintf("lane=%s agente=%s kind=%s created_at=%s worker_started_at=%s handle_id=%d", strings.TrimSpace(lane), strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), msg.CreatedAt.UTC().Format(time.RFC3339Nano), startedAt.Format(time.RFC3339Nano), handle.ID))
	return true, nil
}

func runtimeMailboxPuedeConsumirseFueraDeOrden(msg *db.RuntimeMailboxMessage) (bool, error) {
	if msg == nil || msg.RuntimeOrderID == nil || *msg.RuntimeOrderID <= 0 {
		return true, nil
	}
	order, err := db.GetRuntimeOrder(*msg.RuntimeOrderID)
	if err != nil {
		return false, err
	}
	if order == nil {
		return true, nil
	}
	switch strings.ToLower(strings.TrimSpace(order.Estado)) {
	case "pendiente", "tomada", "ejecutando":
		return false, nil
	default:
		return true, nil
	}
}

func runtimeHandleRequiereCoordinatedRestartMailbox(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(mapStringValue(meta, "driver"))
	tmuxLike := strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") ||
		strings.EqualFold(driver, "tmux_cli_session")
	switch db.RuntimeHandleMailboxDeliveryMode(handle) {
	case runtimeagente.MailboxDeliveryCoordinatedRestart:
		if tmuxLike {
			return true
		}
		if strings.TrimSpace(handle.Transporte) != "cli" || strings.TrimSpace(handle.HandleKind) != "process" {
			return false
		}
		return !runtimeHandleEsCandidatoLegacyATMUX(handle)
	case runtimeagente.MailboxDeliveryBootstrapOnly:
		return tmuxLike
	default:
		return false
	}
}

func runtimeMailboxDebeCoordinarReinicio(kind string, handle *db.RuntimeHandle) bool {
	if !runtimeHandleRequiereCoordinatedRestartMailbox(handle) {
		return false
	}
	kind = strings.TrimSpace(kind)
	switch db.RuntimeHandleMailboxDeliveryMode(handle) {
	case runtimeagente.MailboxDeliveryBootstrapOnly:
		switch kind {
		case "instruction", "autonomia", "nudge", "watchdog", db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
			return true
		default:
			return false
		}
	default:
		switch kind {
		case "instruction", "watchdog", db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
			return true
		default:
			return false
		}
	}
}

func existeRuntimeOrderAbiertaPorHandle(handleID int64, tipos ...string) (bool, error) {
	if handleID <= 0 || len(tipos) == 0 {
		return false, nil
	}
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Estado: &estado})
		if err != nil {
			return false, err
		}
		for _, order := range orders {
			if order == nil || order.HandleID == nil || *order.HandleID != handleID {
				continue
			}
			for _, tipo := range tipos {
				if strings.TrimSpace(order.Tipo) == strings.TrimSpace(tipo) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func existeRuntimeOrderAbiertaPorHandleEnSnapshot(snapshot *runtimeMailboxBatchSnapshot, msg *db.RuntimeMailboxMessage, handleID int64, tipos ...string) (bool, error) {
	if snapshot == nil {
		return existeRuntimeOrderAbiertaPorHandle(handleID, tipos...)
	}
	if msg == nil || handleID <= 0 || len(tipos) == 0 {
		return false, nil
	}
	orders, err := snapshot.ordersForAgentProject(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil {
		return false, err
	}
	tiposWanted := map[string]struct{}{}
	for _, tipo := range tipos {
		tiposWanted[strings.TrimSpace(tipo)] = struct{}{}
	}
	for _, order := range orders {
		if order == nil || order.HandleID == nil || *order.HandleID != handleID {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
		default:
			continue
		}
		if _, ok := tiposWanted[strings.TrimSpace(order.Tipo)]; ok {
			return true, nil
		}
	}
	return false, nil
}

func construirInstruccionMailboxInteractivo(msg *db.RuntimeMailboxMessage) (string, bool) {
	if msg == nil {
		return "", false
	}
	payload := map[string]any{}
	if strings.TrimSpace(msg.PayloadJSON) != "" {
		_ = json.Unmarshal([]byte(msg.PayloadJSON), &payload)
	}
	switch strings.TrimSpace(msg.Kind) {
	case "instruction":
		texto := stringMapValue(payload, "texto")
		if texto == "" {
			texto = stringMapValue(payload, "instruction")
		}
		return texto, strings.TrimSpace(texto) != ""
	case "nudge", "watchdog":
		texto := stringMapValue(payload, "instruction")
		if texto == "" {
			texto = stringMapValue(payload, "texto")
		}
		return texto, strings.TrimSpace(texto) != ""
	case "autonomia":
		if instruction := stringMapValue(payload, "instruction"); strings.TrimSpace(instruction) != "" {
			return strings.TrimSpace(instruction), true
		}
		accion := stringMapValue(payload, "accion")
		motivo := stringMapValue(payload, "motivo")
		base := "Orquesta: continúa de forma autónoma dentro de la gobernanza efectiva del proyecto. No necesitas aprobación humana salvo que falten credenciales, secretos o un recurso externo real."
		switch strings.TrimSpace(accion) {
		case "supervisar_proyecto":
			base += " Actúa como supervisor del proyecto: revisa el estado real, comprueba cumplimiento de reglas, detecta flecos, crea o ajusta tareas si falta trabajo y deja el siguiente frente útil encaminado."
		case "ejecutar_review_gate":
			base += " Actúa como revisor del proyecto: inspecciona el código, valida arquitectura, tests y definición de terminado, documenta findings y aprueba o pide cambios sin detener el proyecto."
		case "continuar_trabajo":
			base += " Sigue con el trabajo en curso y cierra el siguiente frente útil."
		case "esperar_o_pedir_tarea":
			base += " No te quedes esperando: revisa backlog, asignación y siguiente paso útil, y continúa."
		case "votar_propuestas_pendientes":
			base += " Revisa y resuelve las propuestas pendientes del proyecto antes de continuar."
		case "pedir_intervencion":
			base += " No escales a humano por defecto: decide el mejor siguiente paso permitido y ejecuta."
		default:
			base += " Evalúa el mejor siguiente paso y ejecútalo."
		}
		if strings.TrimSpace(motivo) != "" {
			base += " Contexto: " + strings.TrimSpace(motivo) + "."
		}
		base += " Si el cambio es delicado, crea checkpoint y sigue."
		return base, true
	case db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
		return construirInstruccionRefreshRuntime(msg)
	default:
		return "", false
	}
}

func procesarAutonomiaAgentesBatch() (int, error) {
	start := time.Now()
	activa := true
	sessionsStart := time.Now()
	sesiones, err := sesionesAPIService.ListInspectionSessions(db.FiltroSesionesInspeccion{Activa: &activa})
	if err != nil {
		return 0, err
	}
	autonomiaTickDebugf("listar_sesiones sesiones=%d duration=%s", len(sesiones), time.Since(sessionsStart).Round(time.Millisecond))
	snapshotStart := time.Now()
	snapshot, err := newAutonomiaBatchSnapshot(sesiones)
	if err != nil {
		return 0, err
	}
	autonomiaTickDebugf("snapshot agentes=%d duration=%s", len(snapshot.agentesByName), time.Since(snapshotStart).Round(time.Millisecond))
	procesadas := 0
	vistas := map[string]struct{}{}
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		agente := strings.TrimSpace(sesion.Agente)
		if agente == "" {
			continue
		}
		if _, ok := vistas[agente]; ok {
			continue
		}
		vistas[agente] = struct{}{}
		sesionStart := time.Now()
		n, err := procesarAutonomiaSesionActivaConSnapshot(sesion, snapshot)
		if err != nil {
			db.Audit("server", "autonomia_agente_error", "agente", 0, fmt.Sprintf("agente=%s error=%s", agente, err.Error()))
			continue
		}
		procesadas += n
		autonomiaTickDebugf("agente=%s proyecto_id=%d duration=%s procesadas=%d", agente, valorProyectoID(sesion.ProyectoID), time.Since(sesionStart).Round(time.Millisecond), n)
	}
	degradadosStart := time.Now()
	n, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		return procesadas, err
	}
	procesadas += n
	autonomiaTickDebugf("agentes_degradados duration=%s procesadas=%d", time.Since(degradadosStart).Round(time.Millisecond), n)
	autonomiaTickDebugf("batch sesiones=%d agentes=%d duration=%s procesadas=%d", len(sesiones), len(vistas), time.Since(start).Round(time.Millisecond), procesadas)
	if procesadas > 0 {
		resetStatusSnapshotCache()
	}
	return procesadas, nil
}

func procesarAgentesDegradadosAutonomiaBatch() (int, error) {
	db.ResetRuntimeHandlesHotCache()
	rows, err := agentesService.BuildPanelRows()
	if err != nil {
		return 0, err
	}
	rowsPorAgente := map[string]agentesapp.Row{}
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		nombre := strings.ToLower(strings.TrimSpace(row.Agente.Nombre))
		if nombre == "" {
			continue
		}
		rowsPorAgente[nombre] = row
	}
	tareas, err := db.ListarTareas(db.FiltroTareas{})
	if err != nil {
		return 0, err
	}
	tareasActivasPorAgente := map[string][]*db.Tarea{}
	tareasBloqueadasPorAgente := map[string][]*db.Tarea{}
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
		}
		switch tarea.Estado {
		case db.EstadoAsignada, db.EstadoEnProgreso:
			agente := strings.TrimSpace(*tarea.Agente)
			if agente != "" {
				tareasActivasPorAgente[agente] = append(tareasActivasPorAgente[agente], tarea)
			}
		case db.EstadoBloqueada:
			agente := strings.TrimSpace(*tarea.Agente)
			if agente != "" {
				tareasBloqueadasPorAgente[agente] = append(tareasBloqueadasPorAgente[agente], tarea)
			}
		}
	}
	resumenBloqueos, err := db.ListarResumenBloqueos()
	if err != nil {
		return 0, err
	}
	bloqueosPorTarea := map[int64]db.ResumenBloqueo{}
	for _, bloqueo := range resumenBloqueos {
		if bloqueo.ID <= 0 {
			continue
		}
		if _, ok := bloqueosPorTarea[bloqueo.ID]; !ok {
			bloqueosPorTarea[bloqueo.ID] = bloqueo
		}
	}
	openTasksProjected := map[string]int{}
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		openTasksProjected[row.Agente.Nombre] = row.OpenTasks
	}

	procesadas := 0
	now := time.Now().UTC()
	migradas, err := procesarMigracionRuntimeLegacyTMUXBatch(rows, now)
	if err != nil {
		return 0, err
	}
	procesadas += migradas
	if migradas > 0 {
		rows, err = agentesService.BuildPanelRows()
		if err != nil {
			return procesadas, err
		}
		openTasksProjected = openTasksProjectedFromRows(rows)
	}
	huerfanasTMUX, err := procesarSesionesTMUXHuerfanasAutonomiaBatch(now)
	if err != nil {
		return procesadas, err
	}
	procesadas += huerfanasTMUX
	reiniciosAtascados, err := procesarWorkersAtascadosAutonomiaBatch(rows, now)
	if err != nil {
		return procesadas, err
	}
	procesadas += reiniciosAtascados
	if reiniciosAtascados > 0 {
		rows, err = agentesService.BuildPanelRows()
		if err != nil {
			return procesadas, err
		}
		openTasksProjected = openTasksProjectedFromRows(rows)
	}
	limpiadasFuera, err := procesarTareasActivasFueraDeOrquestacionBatch(tareasActivasPorAgente, rowsPorAgente)
	if err != nil {
		return procesadas, err
	}
	procesadas += limpiadasFuera
	redistribuidas, err := procesarSobrecargaAgentesAutonomiaBatch(rows, tareasActivasPorAgente, openTasksProjected, now)
	if err != nil {
		return procesadas, err
	}
	procesadas += redistribuidas
	for _, row := range rows {
		if row.Agente == nil || !rowPermiteAutoRecuperacion(row, now) {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" {
			continue
		}
		workerCeiling := autonomiaOpenTasksCeilingForRow(row, now)
		for _, tarea := range tareasBloqueadasPorAgente[agente] {
			if tarea == nil {
				continue
			}
			actual, err := db.GetTarea(tarea.ID)
			if err != nil {
				return procesadas, err
			}
			if actual == nil || actual.Agente == nil || actual.Estado != db.EstadoBloqueada {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(*actual.Agente), agente) {
				continue
			}
			bloqueo, ok := bloqueosPorTarea[actual.ID]
			if !ok || !esBloqueoAutonomiaAgenteRecuperable(agente, strings.TrimSpace(bloqueo.Agente), strings.TrimSpace(bloqueo.Motivo)) {
				continue
			}
			if esBloqueoSobrecargaOperativa(strings.TrimSpace(bloqueo.Motivo)) && openTasksProjected[agente] >= workerCeiling {
				relevo := seleccionarRelevoAutonomiaBloqueada(rows, openTasksProjected, actual, agente)
				if relevo == "" {
					continue
				}
				resolucion := fmt.Sprintf("reasignación automática por sobrecarga desde %s", agente)
				if err := tareasService.Unblock(actual.ID, "orquesta", resolucion); err != nil {
					return procesadas, err
				}
				if err := tareasService.Reassign(actual.ID, relevo); err != nil {
					return procesadas, err
				}
				if err := tareasService.Start(actual.ID, relevo); err != nil {
					return procesadas, err
				}
				_ = tareasService.Note(actual.ID, "orquesta", fmt.Sprintf("Reasignada automáticamente desde %s a %s tras bloqueo por sobrecarga", agente, relevo))
				openTasksProjected[relevo]++
				procesadas++
				continue
			}
			resolucion := fmt.Sprintf("recuperación automática tras %s (%s)", row.EstadoOperativo, firstNonEmpty(strings.TrimSpace(row.DetalleOperativo), "worker recuperado"))
			if err := tareasService.Unblock(actual.ID, "orquesta", resolucion); err != nil {
				return procesadas, err
			}
			if err := tareasService.Take(actual.ID, agente); err != nil {
				return procesadas, err
			}
			if err := tareasService.Start(actual.ID, agente); err != nil {
				return procesadas, err
			}
			_ = tareasService.Note(actual.ID, "orquesta", fmt.Sprintf("Reactivada automáticamente en %s tras recuperar estado operativo", agente))
			openTasksProjected[agente]++
			procesadas++
		}
	}
	for _, row := range rows {
		if row.Agente == nil || !estadoOperativoAutoIntervencion(row.EstadoOperativo) {
			continue
		}
		if rowControlOrderPending(row, now) {
			continue
		}
		if strings.TrimSpace(row.EstadoOperativo) == "atascado" && !rowAtascadoShouldEscalate(row, now) {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" {
			continue
		}
		for _, tarea := range tareasActivasPorAgente[agente] {
			if tarea == nil {
				continue
			}
			actual, err := db.GetTarea(tarea.ID)
			if err != nil {
				return procesadas, err
			}
			if actual == nil || actual.Agente == nil {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(*actual.Agente), agente) {
				continue
			}
			if actual.Estado != db.EstadoAsignada && actual.Estado != db.EstadoEnProgreso {
				continue
			}
			if degradedTaskRecentlyAutoReassigned(actual, now) {
				continue
			}
			if !degradedTaskShouldIntervene(actual, now) && !degradedTaskRequiresImmediateIntervention(row, now) {
				continue
			}

			motivo := construirMotivoAutonomiaAgenteDegradado(row)
			relevo := seleccionarRelevoAutonomia(rows, openTasksProjected, actual, agente)
			if relevo == "" {
				if err := tareasService.Block(actual.ID, agente, motivo); err != nil {
					return procesadas, err
				}
				procesadas++
				if openTasksProjected[agente] > 0 {
					openTasksProjected[agente]--
				}
				continue
			}

			if err := tareasService.Reassign(actual.ID, relevo); err != nil {
				return procesadas, err
			}
			if err := tareasService.Start(actual.ID, relevo); err != nil {
				return procesadas, err
			}
			_ = tareasService.Note(actual.ID, "orquesta", fmt.Sprintf("Reasignada automáticamente desde %s a %s: %s", agente, relevo, motivo))
			if openTasksProjected[agente] > 0 {
				openTasksProjected[agente]--
			}
			openTasksProjected[relevo]++
			procesadas++

			if actual.ProyectoID != nil {
				proyecto, err := runtimesService.GetProject(strconv.FormatInt(*actual.ProyectoID, 10))
				if err == nil && proyecto != nil {
					if pendiente, err := existeRuntimeOrderAutonomiaPendiente(relevo, &proyecto.ID, "nudge", "continuar_trabajo"); err == nil && !pendiente {
						_, _ = encolarNudgeAutonomiaDetallado(
							relevo,
							proyecto,
							"continuar_trabajo",
							fmt.Sprintf("Tarea #%d reasignada automáticamente", actual.ID),
							"continúa con la tarea reasignada y deja evidencia de avance",
							map[string]any{
								"tarea_id":         actual.ID,
								"reasignada_desde": agente,
								"motivo":           row.EstadoOperativo,
							},
						)
					}
				}
			}
		}
	}
	for _, row := range priorizarRowsAutonomiaBloqueadas(rows, tareasBloqueadasPorAgente) {
		if row.Agente == nil || !estadoOperativoAutoIntervencion(row.EstadoOperativo) {
			continue
		}
		if rowControlOrderPending(row, now) {
			continue
		}
		if strings.TrimSpace(row.EstadoOperativo) == "atascado" && !rowAtascadoShouldEscalate(row, now) {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" {
			continue
		}
		for _, tarea := range tareasBloqueadasPorAgente[agente] {
			if tarea == nil {
				continue
			}
			actual, err := db.GetTarea(tarea.ID)
			if err != nil {
				return procesadas, err
			}
			if actual == nil || actual.Agente == nil {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(*actual.Agente), agente) {
				continue
			}
			if actual.Estado != db.EstadoBloqueada {
				continue
			}
			bloqueo, ok := bloqueosPorTarea[actual.ID]
			if !ok || !esBloqueoAutonomiaAgenteRecuperable(agente, strings.TrimSpace(bloqueo.Agente), strings.TrimSpace(bloqueo.Motivo)) {
				continue
			}
			if degradedTaskRecentlyAutoReassigned(actual, now) {
				continue
			}
			if !degradedTaskShouldIntervene(actual, now) && !degradedTaskRequiresImmediateIntervention(row, now) {
				continue
			}

			relevo := seleccionarRelevoAutonomiaBloqueada(rows, openTasksProjected, actual, agente)
			if relevo == "" {
				continue
			}

			motivo := construirMotivoAutonomiaAgenteDegradado(row)
			resolucion := fmt.Sprintf("reasignación automática desde %s tras %s (%s)", agente, row.EstadoOperativo, firstNonEmpty(strings.TrimSpace(row.DetalleOperativo), "worker degradado"))
			if err := tareasService.Unblock(actual.ID, "orquesta", resolucion); err != nil {
				return procesadas, err
			}
			if err := tareasService.Reassign(actual.ID, relevo); err != nil {
				return procesadas, err
			}
			if err := tareasService.Start(actual.ID, relevo); err != nil {
				return procesadas, err
			}
			_ = tareasService.Note(actual.ID, "orquesta", fmt.Sprintf("Reasignada automáticamente desde %s a %s: %s", agente, relevo, motivo))
			openTasksProjected[relevo]++
			procesadas++

			if actual.ProyectoID != nil {
				proyecto, err := runtimesService.GetProject(strconv.FormatInt(*actual.ProyectoID, 10))
				if err == nil && proyecto != nil {
					if pendiente, err := existeRuntimeOrderAutonomiaPendiente(relevo, &proyecto.ID, "nudge", "continuar_trabajo"); err == nil && !pendiente {
						_, _ = encolarNudgeAutonomiaDetallado(
							relevo,
							proyecto,
							"continuar_trabajo",
							fmt.Sprintf("Tarea #%d reasignada automáticamente", actual.ID),
							"continúa con la tarea reasignada y deja evidencia de avance",
							map[string]any{
								"tarea_id":         actual.ID,
								"reasignada_desde": agente,
								"motivo":           row.EstadoOperativo,
							},
						)
					}
				}
			}
		}
	}
	if procesadas > 0 {
		resetStatusSnapshotCache()
	}
	return procesadas, nil
}

func procesarTareasActivasFueraDeOrquestacionBatch(tareasActivasPorAgente map[string][]*db.Tarea, rowsPorAgente map[string]agentesapp.Row) (int, error) {
	procesadas := 0
	for agente, tareas := range tareasActivasPorAgente {
		row, ok := rowsPorAgente[strings.ToLower(strings.TrimSpace(agente))]
		if ok && row.Agente != nil && strings.TrimSpace(row.EstadoOperativo) != "retirado" {
			continue
		}
		if !ok && agentePareceOperadorManualFueraDeFlota(agente) {
			continue
		}
		motivo := fmt.Sprintf("Agente %s fuera de orquestación: retirado o ausente en el control plane", strings.TrimSpace(agente))
		for _, tarea := range tareas {
			if tarea == nil {
				continue
			}
			actual, err := db.GetTarea(tarea.ID)
			if err != nil {
				return procesadas, err
			}
			if actual == nil || actual.Agente == nil {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(*actual.Agente), strings.TrimSpace(agente)) {
				continue
			}
			if actual.Estado != db.EstadoAsignada && actual.Estado != db.EstadoEnProgreso {
				continue
			}
			if err := tareasService.Block(actual.ID, strings.TrimSpace(agente), motivo); err != nil {
				return procesadas, err
			}
			_ = tareasService.Note(actual.ID, "orquesta", fmt.Sprintf("Bloqueada automáticamente: %s", motivo))
			procesadas++
		}
	}
	return procesadas, nil
}

func agentePareceOperadorManualFueraDeFlota(nombre string) bool {
	lower := strings.ToLower(strings.TrimSpace(nombre))
	if lower == "" {
		return false
	}
	for _, prefijo := range []string{
		"codex",
		"claude",
		"gemini",
		"ollama",
		"antigravity",
	} {
		if strings.HasPrefix(lower, prefijo) {
			return false
		}
	}
	return true
}

func procesarMigracionRuntimeLegacyTMUXBatch(rows []agentesapp.Row, now time.Time) (int, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return 0, nil
	}
	total := 0
	for _, row := range rows {
		if !rowDebeMigrarRuntimeLegacyATMUX(row, now) {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		proyectoID := rowProyectoIDPreferido(row)
		if pendiente, err := existeRuntimeOrderAbiertaAutonomia(agente, proyectoID, "stop", "start", "restart", "resume"); err != nil {
			return total, err
		} else if pendiente {
			continue
		}
		stopReciente, err := existeRuntimeOrderAutonomiaReciente(agente, proyectoID, "stop", "stop", 30*time.Minute)
		if err != nil {
			return total, err
		}
		startReciente, err := existeRuntimeOrderAutonomiaReciente(agente, proyectoID, "start", "start", 30*time.Minute)
		if err != nil {
			return total, err
		}
		if stopReciente || startReciente {
			continue
		}
		handle := row.Handle
		if handle == nil {
			continue
		}
		stopOrderID, startOrderID, err := db.EncolarReinicioCoordinadoRuntimeHandle(handle, proyectoID, "migrar a tmux", "orquesta")
		if err != nil {
			return total, err
		}
		detalleProyecto := "-"
		if proyectoID != nil && *proyectoID > 0 {
			detalleProyecto = strconv.FormatInt(*proyectoID, 10)
		}
		db.Audit("orquesta", "runtime_migrate_tmux", "runtime_handle", handle.ID,
			fmt.Sprintf("agente=%s proyecto_id=%s stop_order_id=%d start_order_id=%d", agente, detalleProyecto, stopOrderID, startOrderID))
		total++
	}
	return total, nil
}

func procesarWorkersAtascadosAutonomiaBatch(rows []agentesapp.Row, now time.Time) (int, error) {
	cooldown := time.Duration(controlPlaneConfigIntOrDefault("autonomia_stuck_restart_cooldown_seconds", 1800)) * time.Second
	if cooldown <= 0 {
		cooldown = 30 * time.Minute
	}
	escalationWindow := time.Duration(controlPlaneConfigIntOrDefault("autonomia_stuck_restart_escalation_window_seconds", int(autonomiaStuckRestartEscalationWindowDefault.Seconds()))) * time.Second
	if escalationWindow <= 0 {
		escalationWindow = autonomiaStuckRestartEscalationWindowDefault
	}
	escalationCount := controlPlaneConfigIntOrDefault("autonomia_stuck_restart_escalation_count", autonomiaStuckRestartEscalationCountDefault)
	if escalationCount <= 0 {
		escalationCount = autonomiaStuckRestartEscalationCountDefault
	}
	total := 0
	for _, row := range rows {
		if row.Agente == nil || strings.TrimSpace(row.EstadoOperativo) != "atascado" {
			continue
		}
		if !row.WorkerFresh(now) {
			continue
		}
		if row.OpenTasks <= 0 && row.MailboxPending <= 0 {
			continue
		}
		handle := row.Handle
		if handle == nil {
			continue
		}
		if !rowHasFreshTMUXWorkerForRecovery(row, now) {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" {
			continue
		}
		if disponible, _, err := autonomiaCuentaCompartidaDisponible(agente); err != nil {
			return total, err
		} else if !disponible {
			continue
		}
		proyectoID := rowProyectoIDPreferido(row)
		stopCount, err := contarRuntimeOrdersRecientes(agente, proyectoID, "stop", "stop", escalationWindow)
		if err != nil {
			return total, err
		}
		if stopCount >= escalationCount {
			escaladas, err := escalarTareasWorkerAtascado(row, rows, now)
			if err != nil {
				return total, err
			}
			total += escaladas
			continue
		}
		if pendiente, err := existeRuntimeOrderAbiertaAutonomia(agente, proyectoID, "stop", "start", "restart", "resume", "handoff"); err != nil {
			return total, err
		} else if pendiente {
			continue
		}
		stopReciente, err := existeRuntimeOrderAutonomiaReciente(agente, proyectoID, "stop", "stop", cooldown)
		if err != nil {
			return total, err
		}
		startReciente, err := existeRuntimeOrderAutonomiaReciente(agente, proyectoID, "start", "start", cooldown)
		if err != nil {
			return total, err
		}
		if stopReciente || startReciente {
			continue
		}
		stopOrderID, startOrderID, err := db.EncolarReinicioCoordinadoRuntimeHandle(handle, proyectoID, "worker_atascado", "orquesta")
		if err != nil {
			return total, err
		}
		detalleProyecto := "-"
		if proyectoID != nil && *proyectoID > 0 {
			detalleProyecto = strconv.FormatInt(*proyectoID, 10)
		}
		db.Audit("orquesta", "runtime_restart_stuck_worker", "runtime_handle", handle.ID,
			fmt.Sprintf("agente=%s proyecto_id=%s stop_order_id=%d start_order_id=%d detalle=%s", agente, detalleProyecto, stopOrderID, startOrderID, firstNonEmpty(strings.TrimSpace(row.DetalleOperativo), "worker atascado")))
		total++
	}
	return total, nil
}

func procesarSesionesTMUXHuerfanasAutonomiaBatch(now time.Time) (int, error) {
	handles, err := db.ListarRuntimeHandles(nil)
	if err != nil {
		return 0, err
	}
	aliveSessions := map[string]bool{}
	for _, handle := range handles {
		if sessionName, alive := runtimeHandleTMUXAliveSession(handle, now); sessionName != "" && alive {
			aliveSessions[sessionName] = true
		}
	}
	procesadas := 0
	seenSessions := map[string]bool{}
	for _, handle := range handles {
		sessionName, orphan := runtimeHandleTMUXOrphanSession(handle, now, aliveSessions)
		if sessionName == "" || !orphan {
			continue
		}
		if seenSessions[sessionName] {
			continue
		}
		seenSessions[sessionName] = true
		aplicado, err := controlruntime.DetenerSesionTMUXMetadata(handle.MetadataJSON)
		if err != nil {
			return procesadas, err
		}
		if !aplicado {
			continue
		}
		db.Audit("orquesta", "runtime_tmux_orphan_cleanup", "runtime_handle", handle.ID,
			fmt.Sprintf("agente=%s session=%s estado=%s", strings.TrimSpace(handle.Agente), sessionName, strings.TrimSpace(handle.Estado)))
		procesadas++
	}
	return procesadas, nil
}

func runtimeHandleTMUXAliveSession(handle *db.RuntimeHandle, now time.Time) (string, bool) {
	sessionName, view := runtimeHandleTMUXWorkerView(handle, now)
	if sessionName == "" || view == nil {
		return "", false
	}
	if view.HeartbeatStale || !view.Alive {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "", "failed", "stopped", "exited", "closed", "stale":
		return "", false
	}
	return sessionName, true
}

func runtimeHandleTMUXOrphanSession(handle *db.RuntimeHandle, now time.Time, aliveSessions map[string]bool) (string, bool) {
	sessionName, view := runtimeHandleTMUXWorkerView(handle, now)
	if sessionName == "" {
		return "", false
	}
	if aliveSessions[sessionName] {
		return "", false
	}
	estado := strings.ToLower(strings.TrimSpace(handle.Estado))
	switch estado {
	case "fallido", "cerrado":
		return sessionName, true
	}
	if view == nil {
		return "", false
	}
	if !view.Alive || view.HeartbeatStale {
		return sessionName, true
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "failed", "stopped", "exited", "closed", "stale":
		return sessionName, true
	default:
		return "", false
	}
}

func runtimeHandleTMUXWorkerView(handle *db.RuntimeHandle, now time.Time) (string, *runtimeagente.WorkerStatusView) {
	if handle == nil {
		return "", nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if !strings.EqualFold(strings.TrimSpace(mapStringValue(meta, "driver")), "tmux_cli_session") {
		return "", nil
	}
	sessionName := strings.TrimSpace(mapStringValue(meta, "tmux_session"))
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return sessionName, nil
	}
	if snap.Manifest != nil && sessionName == "" {
		sessionName = strings.TrimSpace(snap.Manifest.TmuxSession)
	}
	return sessionName, snap.View(now, 2*time.Minute)
}

func escalarTareasWorkerAtascado(row agentesapp.Row, rows []agentesapp.Row, now time.Time) (int, error) {
	if row.Agente == nil {
		return 0, nil
	}
	agente := strings.TrimSpace(row.Agente.Nombre)
	if agente == "" {
		return 0, nil
	}
	proyectoID := rowProyectoIDPreferido(row)
	filtro := db.FiltroTareas{Agente: &agente}
	if proyectoID != nil && *proyectoID > 0 {
		filtro.ProyectoID = proyectoID
	}
	tareas, err := tareasService.List(filtro)
	if err != nil {
		return 0, err
	}
	openTasksProjected := openTasksProjectedFromRows(rows)
	motivo := fmt.Sprintf("Agente %s atascado: %s", agente, firstNonEmpty(strings.TrimSpace(row.DetalleOperativo), "sin progreso reciente tras reinicios"))
	procesadas := 0
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		actual, err := db.GetTarea(tarea.ID)
		if err != nil {
			return procesadas, err
		}
		if actual == nil || actual.Agente == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(*actual.Agente), agente) {
			continue
		}
		if actual.Estado != db.EstadoAsignada && actual.Estado != db.EstadoEnProgreso {
			continue
		}
		if taskManualTakeover(actual) {
			continue
		}
		relevo := seleccionarRelevoAutonomia(rows, openTasksProjected, actual, agente)
		if relevo == "" {
			if err := tareasService.Block(actual.ID, agente, motivo); err != nil {
				return procesadas, err
			}
			_ = tareasService.Note(actual.ID, "orquesta", fmt.Sprintf("Bloqueada automáticamente en %s por atasco persistente tras reinicios recientes", agente))
			if openTasksProjected[agente] > 0 {
				openTasksProjected[agente]--
			}
			procesadas++
			continue
		}
		if err := tareasService.Reassign(actual.ID, relevo); err != nil {
			return procesadas, err
		}
		if err := tareasService.Start(actual.ID, relevo); err != nil {
			return procesadas, err
		}
		_ = tareasService.Note(actual.ID, "orquesta", fmt.Sprintf("Reasignada automáticamente desde %s a %s por atasco persistente tras reinicios recientes", agente, relevo))
		if openTasksProjected[agente] > 0 {
			openTasksProjected[agente]--
		}
		openTasksProjected[relevo]++
		procesadas++
	}
	if procesadas > 0 {
		resetStatusSnapshotCache()
	}
	return procesadas, nil
}

func rowAtascadoShouldEscalate(row agentesapp.Row, now time.Time) bool {
	if row.Agente == nil || strings.TrimSpace(row.EstadoOperativo) != "atascado" {
		return false
	}
	if !row.WorkerFresh(now) {
		return false
	}
	agente := strings.TrimSpace(row.Agente.Nombre)
	if agente == "" {
		return false
	}
	proyectoID := rowProyectoIDPreferido(row)
	window := time.Duration(controlPlaneConfigIntOrDefault("autonomia_stuck_restart_escalation_window_seconds", int(autonomiaStuckRestartEscalationWindowDefault.Seconds()))) * time.Second
	if window <= 0 {
		window = autonomiaStuckRestartEscalationWindowDefault
	}
	threshold := controlPlaneConfigIntOrDefault("autonomia_stuck_restart_escalation_count", autonomiaStuckRestartEscalationCountDefault)
	if threshold <= 0 {
		threshold = autonomiaStuckRestartEscalationCountDefault
	}
	stopCount, err := contarRuntimeOrdersRecientes(agente, proyectoID, "stop", "stop", window)
	if err != nil {
		return false
	}
	return stopCount >= threshold
}

func rowControlOrderPending(row agentesapp.Row, now time.Time) bool {
	if row.ControlOrdersOpen <= 0 {
		return false
	}
	tipo := strings.ToLower(strings.TrimSpace(row.LastControlOrderType))
	switch tipo {
	case "stop", "start", "pause", "resume":
	default:
		return false
	}
	if row.LastControlOrderMoment == nil || row.LastControlOrderMoment.IsZero() {
		return true
	}
	moment := row.LastControlOrderMoment.UTC()
	if moment.Before(now.Add(-20 * time.Minute)) {
		return false
	}
	if row.WorkerHeartbeat != nil && !row.WorkerHeartbeat.IsZero() && row.WorkerHeartbeat.UTC().After(moment) {
		return false
	}
	if row.WorkerUpdatedAt != nil && !row.WorkerUpdatedAt.IsZero() && row.WorkerUpdatedAt.UTC().After(moment) {
		return false
	}
	return true
}

func procesarSobrecargaAgentesAutonomiaBatch(rows []agentesapp.Row, tareasActivasPorAgente map[string][]*db.Tarea, openTasksProjected map[string]int, now time.Time) (int, error) {
	procesadas := 0
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" {
			continue
		}
		workerCeiling := autonomiaOpenTasksCeilingForRow(row, now)
		if openTasksProjected[agente] <= workerCeiling {
			continue
		}
		if !estadoOperativoAutoRecuperacion(row.EstadoOperativo) {
			continue
		}
		tareas := append([]*db.Tarea(nil), tareasActivasPorAgente[agente]...)
		if len(tareas) == 0 {
			continue
		}
		sort.SliceStable(tareas, func(i, j int) bool {
			pi := prioridadRedistribucionSobrecarga(tareas[i])
			pj := prioridadRedistribucionSobrecarga(tareas[j])
			if pi != pj {
				return pi < pj
			}
			if !tareas[i].UpdatedAt.Equal(tareas[j].UpdatedAt) {
				return tareas[i].UpdatedAt.Before(tareas[j].UpdatedAt)
			}
			return tareas[i].ID < tareas[j].ID
		})
		for _, tarea := range tareas {
			if openTasksProjected[agente] <= workerCeiling {
				break
			}
			if tarea == nil {
				continue
			}
			actual, err := db.GetTarea(tarea.ID)
			if err != nil {
				return procesadas, err
			}
			if actual == nil || actual.Agente == nil {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(*actual.Agente), agente) {
				continue
			}
			if actual.Estado != db.EstadoAsignada && actual.Estado != db.EstadoEnProgreso {
				continue
			}
			relevo := seleccionarRelevoAutonomiaConLimite(rows, openTasksProjected, actual, agente, autonomiaWorkerOpenTasksCeiling)
			if relevo == "" {
				if err := tareasService.Block(actual.ID, agente, "Sobrecarga operativa: sin relevo sano disponible"); err != nil {
					return procesadas, err
				}
				_ = tareasService.Note(actual.ID, "orquesta", fmt.Sprintf("Bloqueada automáticamente en %s por sobrecarga operativa sin relevo sano", agente))
				if openTasksProjected[agente] > 0 {
					openTasksProjected[agente]--
				}
				procesadas++
				continue
			}
			if err := tareasService.Reassign(actual.ID, relevo); err != nil {
				return procesadas, err
			}
			if err := tareasService.Start(actual.ID, relevo); err != nil {
				return procesadas, err
			}
			_ = tareasService.Note(actual.ID, "orquesta", fmt.Sprintf("Redistribuida automáticamente desde %s a %s por sobrecarga operativa", agente, relevo))
			if openTasksProjected[agente] > 0 {
				openTasksProjected[agente]--
			}
			openTasksProjected[relevo]++
			procesadas++

			if actual.ProyectoID != nil {
				proyecto, err := runtimesService.GetProject(strconv.FormatInt(*actual.ProyectoID, 10))
				if err == nil && proyecto != nil {
					if pendiente, err := existeRuntimeOrderAutonomiaPendiente(relevo, &proyecto.ID, "nudge", "continuar_trabajo"); err == nil && !pendiente {
						_, _ = encolarNudgeAutonomiaDetallado(
							relevo,
							proyecto,
							"continuar_trabajo",
							fmt.Sprintf("Tarea #%d redistribuida automáticamente", actual.ID),
							"continúa con la tarea redistribuida y deja evidencia de avance",
							map[string]any{
								"tarea_id":            actual.ID,
								"redistribuida_desde": agente,
								"motivo":              "sobrecarga_operativa",
							},
						)
					}
				}
			}
		}
	}
	return procesadas, nil
}

func prioridadRedistribucionSobrecarga(tarea *db.Tarea) int {
	if tarea == nil {
		return 99
	}
	switch tarea.Estado {
	case db.EstadoAsignada:
		return 0
	case db.EstadoEnProgreso:
		return 1
	default:
		return 2
	}
}

func rowDebeMigrarRuntimeLegacyATMUX(row agentesapp.Row, now time.Time) bool {
	if row.Agente == nil || row.Handle == nil {
		return false
	}
	if agenteBloqueadoPorCuotaVisible(row.Agente) {
		return false
	}
	if !row.WorkerFresh(now) {
		return false
	}
	if !runtimeHandleEsCandidatoLegacyATMUX(row.Handle) {
		return false
	}
	return true
}

func runtimeHandleEsCandidatoLegacyATMUX(handle *db.RuntimeHandle) bool {
	if !runtimeHandleEsLegacyControlPlane(handle) {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	for _, candidate := range []string{
		mapStringValue(meta, "rendered_command"),
		mapStringValue(meta, "wrapped_command"),
		mapStringValue(meta, "herramienta"),
		mapStringValue(meta, "conector"),
		mapStringValue(meta, "profile_status_wrapper"),
	} {
		if controlruntime.RenderedCommandLooksLikeTMUXPreferredCLI(candidate) {
			return true
		}
	}
	return false
}

func runtimeHandleEsLegacyControlPlane(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if strings.TrimSpace(handle.Transporte) != "cli" || strings.TrimSpace(handle.HandleKind) != "process" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
	case "activo", "pausado":
	default:
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if !strings.EqualFold(strings.TrimSpace(mapStringValue(meta, "driver")), "process_pty_cli") {
		return false
	}
	return true
}

func rowProyectoIDPreferido(row agentesapp.Row) *int64 {
	for _, id := range []*int64{
		rowHandleProyectoID(row.Handle),
		rowRuntimeProyectoID(row.Runtime),
		rowSesionProyectoID(row.Sesion),
		rowAsignacionProyectoID(row.Asignacion),
	} {
		if id != nil && *id > 0 {
			return id
		}
	}
	return nil
}

func rowHandleProyectoID(handle *db.RuntimeHandle) *int64 {
	if handle == nil || handle.ProyectoID == nil || *handle.ProyectoID <= 0 {
		return nil
	}
	id := *handle.ProyectoID
	return &id
}

func rowRuntimeProyectoID(runtime *db.RuntimeInstance) *int64 {
	if runtime == nil || runtime.ProyectoID == nil || *runtime.ProyectoID <= 0 {
		return nil
	}
	id := *runtime.ProyectoID
	return &id
}

func rowSesionProyectoID(sesion *db.Sesion) *int64 {
	if sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID <= 0 {
		return nil
	}
	id := *sesion.ProyectoID
	return &id
}

func rowAsignacionProyectoID(asignacion *db.Asignacion) *int64 {
	if asignacion == nil || asignacion.ProyectoID <= 0 {
		return nil
	}
	id := asignacion.ProyectoID
	return &id
}

func estadoOperativoAutoIntervencion(estado string) bool {
	switch strings.TrimSpace(estado) {
	case "bloqueado_por_runtime", "mailbox_atascada", "bloqueado_por_cuota", "caido", "atascado":
		return true
	default:
		return false
	}
}

func priorizarRowsAutonomiaBloqueadas(rows []agentesapp.Row, tareasBloqueadasPorAgente map[string][]*db.Tarea) []agentesapp.Row {
	prioritized := append([]agentesapp.Row(nil), rows...)
	sort.SliceStable(prioritized, func(i, j int) bool {
		nombreI := ""
		nombreJ := ""
		if prioritized[i].Agente != nil {
			nombreI = strings.TrimSpace(prioritized[i].Agente.Nombre)
		}
		if prioritized[j].Agente != nil {
			nombreJ = strings.TrimSpace(prioritized[j].Agente.Nombre)
		}
		bloqueadasI := len(tareasBloqueadasPorAgente[nombreI])
		bloqueadasJ := len(tareasBloqueadasPorAgente[nombreJ])
		if bloqueadasI != bloqueadasJ {
			return bloqueadasI > bloqueadasJ
		}
		return strings.ToLower(nombreI) < strings.ToLower(nombreJ)
	})
	return prioritized
}

func estadoOperativoAutoRecuperacion(estado string) bool {
	switch strings.TrimSpace(estado) {
	case "disponible", "trabajando", "saturado":
		return true
	default:
		return false
	}
}

func rowPermiteAutoRecuperacion(row agentesapp.Row, now time.Time) bool {
	if estadoOperativoAutoRecuperacion(row.EstadoOperativo) {
		return true
	}
	if row.OpenTasks == 0 && row.BlockedTasks > 0 && row.WorkerSupportsContinuityRecovery(now) {
		return true
	}
	if strings.TrimSpace(row.EstadoOperativo) != "bloqueado_por_runtime" {
		return false
	}
	if !row.WorkerFresh(now) {
		return false
	}
	if row.Handle != nil {
		switch strings.ToLower(strings.TrimSpace(row.Handle.Estado)) {
		case "pausado", "paused":
			return true
		}
	}
	if row.Runtime != nil {
		switch strings.ToLower(strings.TrimSpace(row.Runtime.LogicalState)) {
		case "pausado", "paused":
			return true
		}
	}
	return false
}

func openTasksProjectedFromRows(rows []agentesapp.Row) map[string]int {
	out := map[string]int{}
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		out[row.Agente.Nombre] = row.OpenTasks
	}
	return out
}

func construirMotivoAutonomiaAgenteDegradado(row agentesapp.Row) string {
	agente := ""
	if row.Agente != nil {
		agente = strings.TrimSpace(row.Agente.Nombre)
	}
	base := fmt.Sprintf("Agente %s en estado %s", agente, strings.TrimSpace(row.EstadoOperativo))
	detalle := strings.TrimSpace(row.DetalleOperativo)
	if detalle != "" {
		base += ": " + detalle
	}
	return base
}

func esBloqueoAutonomiaAgenteRecuperable(agente, bloqueadoPor, motivo string) bool {
	agente = strings.TrimSpace(agente)
	bloqueadoPor = strings.TrimSpace(bloqueadoPor)
	motivo = strings.TrimSpace(motivo)
	if agente == "" || motivo == "" {
		return false
	}
	if strings.HasPrefix(motivo, "Agente "+agente+" en estado ") {
		return true
	}
	if strings.EqualFold(bloqueadoPor, agente) && strings.HasPrefix(motivo, "Agente degradado:") {
		return true
	}
	if strings.EqualFold(bloqueadoPor, agente) && esBloqueoSobrecargaOperativa(motivo) {
		return true
	}
	return false
}

func esBloqueoSobrecargaOperativa(motivo string) bool {
	motivo = strings.TrimSpace(strings.ToLower(motivo))
	return strings.HasPrefix(motivo, strings.ToLower("Sobrecarga operativa:"))
}

func seleccionarRelevoAutonomia(rows []agentesapp.Row, openTasksProjected map[string]int, tarea *db.Tarea, agenteBloqueado string) string {
	return seleccionarRelevoAutonomiaConLimite(rows, openTasksProjected, tarea, agenteBloqueado, autonomiaWorkerOpenTasksCeiling)
}

func seleccionarRelevoAutonomiaBloqueada(rows []agentesapp.Row, openTasksProjected map[string]int, tarea *db.Tarea, agenteBloqueado string) string {
	return seleccionarRelevoAutonomiaConTecho(rows, openTasksProjected, tarea, agenteBloqueado, func(row agentesapp.Row, now time.Time) int {
		ceiling := autonomiaBlockedTaskRecoveryBurstCeiling
		if dynamic := autonomiaOpenTasksCeilingForRow(row, now); dynamic > ceiling {
			ceiling = dynamic
		}
		if rowHasFreshTMUXWorkerForRecovery(row, now) {
			burst := controlPlaneConfigIntOrDefault("autonomia_tmux_blocked_recovery_open_tasks_ceiling", autonomiaTMUXBlockedRecoveryOpenTasksDefault)
			if burst > ceiling {
				ceiling = burst
			}
		}
		return ceiling
	})
}

func seleccionarRelevoAutonomiaConLimite(rows []agentesapp.Row, openTasksProjected map[string]int, tarea *db.Tarea, agenteBloqueado string, maxOpenTasksPerRecoveryWorker int) string {
	return seleccionarRelevoAutonomiaConTecho(rows, openTasksProjected, tarea, agenteBloqueado, func(row agentesapp.Row, now time.Time) int {
		candidateCeiling := maxOpenTasksPerRecoveryWorker
		if dynamicCeiling := autonomiaOpenTasksCeilingForRow(row, now); dynamicCeiling > candidateCeiling {
			candidateCeiling = dynamicCeiling
		}
		return candidateCeiling
	})
}

func seleccionarRelevoAutonomiaConTecho(rows []agentesapp.Row, openTasksProjected map[string]int, tarea *db.Tarea, agenteBloqueado string, ceilingFn func(agentesapp.Row, time.Time) int) string {
	now := time.Now().UTC()
	lastReassign, hasLastReassign := latestAutoReassignmentInfo(taskNotes(tarea))
	type candidate struct {
		agente        string
		mismoProyecto bool
		estadoRank    int
		openTasks     int
	}
	candidates := make([]candidate, 0, len(rows))
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" || strings.EqualFold(agente, agenteBloqueado) {
			continue
		}
		estado := strings.TrimSpace(row.EstadoOperativo)
		candidateCeiling := 0
		if ceilingFn != nil {
			candidateCeiling = ceilingFn(row, now)
		}
		if estado != "disponible" && !(candidateCeiling > 0 && estado == "trabajando") {
			continue
		}
		if row.OrdersOpen > 0 {
			continue
		}
		if disponible, _, err := autonomiaCuentaCompartidaDisponible(agente); err == nil && !disponible {
			continue
		}
		if strings.TrimSpace(row.WorkerState) != "" && !row.WorkerFresh(now) {
			continue
		}
		if hasLastReassign && now.Sub(lastReassign.At) < 30*time.Minute && strings.EqualFold(agente, lastReassign.From) {
			continue
		}
		mismoProyecto := false
		if tarea != nil && tarea.ProyectoID != nil && row.Asignacion != nil && row.Asignacion.ProyectoID == *tarea.ProyectoID {
			mismoProyecto = true
		}
		openTasks := openTasksProjected[agente]
		if candidateCeiling > 0 && openTasks >= candidateCeiling {
			continue
		}
		candidates = append(candidates, candidate{
			agente:        agente,
			mismoProyecto: mismoProyecto,
			estadoRank:    0,
			openTasks:     openTasks,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].mismoProyecto != candidates[j].mismoProyecto {
			return candidates[i].mismoProyecto
		}
		if candidates[i].estadoRank != candidates[j].estadoRank {
			return candidates[i].estadoRank < candidates[j].estadoRank
		}
		if candidates[i].openTasks != candidates[j].openTasks {
			return candidates[i].openTasks < candidates[j].openTasks
		}
		return strings.ToLower(candidates[i].agente) < strings.ToLower(candidates[j].agente)
	})
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].agente
}

func degradedTaskShouldIntervene(tarea *db.Tarea, now time.Time) bool {
	if tarea == nil || tarea.ID <= 0 {
		return false
	}
	if taskManualTakeover(tarea) {
		return false
	}
	if info, ok := latestAutoReassignmentInfo(taskNotes(tarea)); ok && now.Sub(info.At) < autonomiaDegradedTaskCooldown {
		return false
	}
	autonomiaDegradedTaskGate.mu.Lock()
	defer autonomiaDegradedTaskGate.mu.Unlock()
	if autonomiaDegradedTaskGate.last == nil {
		autonomiaDegradedTaskGate.last = map[int64]time.Time{}
	}
	if last, ok := autonomiaDegradedTaskGate.last[tarea.ID]; ok && now.Sub(last) < autonomiaDegradedTaskCooldown {
		return false
	}
	autonomiaDegradedTaskGate.last[tarea.ID] = now
	return true
}

func degradedTaskRecentlyAutoReassigned(tarea *db.Tarea, now time.Time) bool {
	if tarea == nil {
		return false
	}
	info, ok := latestAutoReassignmentInfo(taskNotes(tarea))
	if !ok {
		return false
	}
	return now.Sub(info.At) < autonomiaDegradedTaskCooldown
}

func degradedTaskRequiresImmediateIntervention(row agentesapp.Row, now time.Time) bool {
	switch strings.TrimSpace(row.EstadoOperativo) {
	case "bloqueado_por_cuota", "caido":
		return true
	case "bloqueado_por_runtime":
		return !rowPermiteAutoRecuperacion(row, now)
	default:
		return false
	}
}

type autoReassignmentInfo struct {
	At   time.Time
	From string
	To   string
}

func latestAutoReassignmentInfo(notas string) (autoReassignmentInfo, bool) {
	for _, line := range reverseNonEmptyLines(notas) {
		if !strings.Contains(line, "Reasignada automáticamente") {
			continue
		}
		open := strings.LastIndex(line, "[")
		close := strings.LastIndex(line, "]")
		if open < 0 || close <= open {
			continue
		}
		meta := strings.TrimSpace(line[open+1 : close])
		fields := strings.Fields(meta)
		if len(fields) < 2 {
			continue
		}
		ts, err := time.ParseInLocation("2006-01-02 15:04:05", fields[0]+" "+fields[1], time.UTC)
		if err != nil {
			continue
		}
		info := autoReassignmentInfo{At: ts.UTC()}
		prefix := strings.TrimSpace(line[:open])
		if strings.HasPrefix(prefix, "Reasignada automáticamente desde ") {
			body := strings.TrimPrefix(prefix, "Reasignada automáticamente desde ")
			if idx := strings.Index(body, ":"); idx >= 0 {
				body = strings.TrimSpace(body[:idx])
			}
			parts := strings.SplitN(body, " a ", 2)
			if len(parts) == 2 {
				info.From = strings.TrimSpace(parts[0])
				info.To = strings.TrimSpace(parts[1])
			}
		}
		return info, true
	}
	return autoReassignmentInfo{}, false
}

func reverseNonEmptyLines(raw string) []string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func taskNotes(tarea *db.Tarea) string {
	if tarea == nil {
		return ""
	}
	return strings.TrimSpace(tarea.Notas)
}

func taskManualTakeover(tarea *db.Tarea) bool {
	return taskNotesMarkedManualTakeover(taskNotes(tarea))
}

func taskNotesMarkedManualTakeover(notas string) bool {
	notas = strings.ToLower(strings.TrimSpace(notas))
	if notas == "" {
		return false
	}
	return strings.Contains(notas, "asumida manualmente fuera de la flota automat")
}

func autonomiaOpenTasksCeilingForRow(row agentesapp.Row, now time.Time) int {
	ceiling := autonomiaWorkerOpenTasksCeiling
	tmuxCeiling := controlPlaneConfigIntOrDefault("autonomia_tmux_open_tasks_ceiling", autonomiaTMUXWorkerOpenTasksCeilingDefault)
	if tmuxCeiling < ceiling {
		tmuxCeiling = ceiling
	}
	if !rowHasFreshTMUXWorkerForRecovery(row, now) {
		return ceiling
	}
	return tmuxCeiling
}

func structuredWorkerSnapshotForRecovery(row agentesapp.Row) *runtimeagente.WorkerSnapshot {
	for _, raw := range []string{metadataJSONFromHandle(row.Handle)} {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(raw)
		if err == nil && snap != nil {
			return snap
		}
	}
	return nil
}

func metadataJSONFromHandle(handle *db.RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	return strings.TrimSpace(handle.MetadataJSON)
}

func rowWorkerDriverForRecovery(row agentesapp.Row) string {
	if driver := strings.TrimSpace(row.WorkerDriver); driver != "" {
		return driver
	}
	snap := structuredWorkerSnapshotForRecovery(row)
	if snap == nil {
		return ""
	}
	return strings.TrimSpace(snap.Driver())
}

func rowHasFreshTMUXWorkerForRecovery(row agentesapp.Row, now time.Time) bool {
	if row.WorkerTMUXFresh(now) {
		return true
	}
	if !row.WorkerFresh(now) {
		return false
	}
	return strings.EqualFold(rowWorkerDriverForRecovery(row), "tmux_cli_session")
}

func autonomiaTickDebugf(format string, args ...any) {
	if autonomiaTickDebugEnabled() {
		log.Printf("orquesta[autonomia-batch] "+format, args...)
	}
}

func valorProyectoID(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}

func procesarAutonomiaSesionActiva(sesion *db.Sesion) (int, error) {
	return procesarAutonomiaSesionActivaConSnapshot(sesion, nil)
}

func procesarAutonomiaSesionActivaConSnapshot(sesion *db.Sesion, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	if n, err := procesarCierreProyectoSesion(sesion); err != nil || n > 0 {
		if n > 0 && snapshot != nil {
			snapshot.invalidateAgent(sesion.Agente)
			snapshot.invalidateProject(*sesion.ProyectoID)
		}
		return n, err
	}
	if n, err := procesarReanudacionAutonomaSesion(sesion, snapshot); err != nil || n > 0 {
		if n > 0 && snapshot != nil {
			snapshot.invalidateAgent(sesion.Agente)
			snapshot.invalidateProject(*sesion.ProyectoID)
		}
		return n, err
	}
	if n, err := procesarAparcadoAutonomoSesion(sesion, snapshot); err != nil || n > 0 {
		if n > 0 && snapshot != nil {
			snapshot.invalidateAgent(sesion.Agente)
			snapshot.invalidateProject(*sesion.ProyectoID)
		}
		return n, err
	}
	if n, err := procesarRecuperacionRuntimeDegradadoSesion(sesion); err != nil || n > 0 {
		if n > 0 && snapshot != nil {
			snapshot.invalidateAgent(sesion.Agente)
			snapshot.invalidateProject(*sesion.ProyectoID)
		}
		return n, err
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	out, err := construirAgenteTickOutputConSnapshot(sesion.Agente, proyecto, sesion, 0, snapshot)
	if err != nil {
		return 0, err
	}
	switch strings.TrimSpace(out.AccionRecomendada) {
	case "pausar_por_cuota", "pausar_y_reasignar":
		if satisfecha, err := pausaAutonomiaYaSatisfecha(sesion.Agente, &proyecto.ID, sesion); err != nil {
			return 0, err
		} else if satisfecha {
			return 0, nil
		}
		if strings.TrimSpace(out.AccionRecomendada) == "pausar_por_cuota" {
			if err := persistirPausaPorCuotaAutonomia(sesion.Agente, out.Motivo); err != nil {
				return 0, err
			}
		}
		if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
			Agente:   sesion.Agente,
			Proyecto: proyecto.Slug,
			Accion:   agenteControlAccionPause,
			Motivo:   out.Motivo,
			Por:      "orquesta",
		}); err != nil {
			return 0, err
		}
		if snapshot != nil {
			snapshot.invalidateAgent(sesion.Agente)
			snapshot.invalidateProject(proyecto.ID)
		}
		return 1, nil
	case "supervisar_proyecto":
		// La supervisión rica del proyecto ya tiene su propio batch y señales
		// dedicadas. Repetir un nudge genérico por cada tick de una sesión viva
		// solo reinyecta guidance redundante sobre un runtime ya activo.
		return 0, nil
	case "votar_propuestas_pendientes", "pedir_intervencion":
		if pendiente, err := existeRuntimeOrderAutonomiaPendiente(sesion.Agente, &proyecto.ID, "nudge", strings.TrimSpace(out.AccionRecomendada)); err != nil {
			return 0, err
		} else if pendiente {
			return 0, nil
		}
		if encolada, err := encolarNudgeAutonomia(sesion.Agente, proyecto, out.AccionRecomendada, out.Motivo); err != nil {
			return 0, err
		} else if !encolada {
			return 0, nil
		}
		if snapshot != nil {
			snapshot.invalidateAgent(sesion.Agente)
			snapshot.invalidateProject(proyecto.ID)
		}
		return 1, nil
	case "continuar_trabajo", "esperar_o_pedir_tarea":
		// Una sesión ya activa no debe recibir recordatorios periódicos vacíos.
		// Pero si el agente está idle y acaba de autoasignarse una tarea real,
		// sí hay que empujarle ese nuevo frente sin esperar a reinicios o handoff.
		if strings.TrimSpace(out.AccionRecomendada) != "esperar_o_pedir_tarea" {
			return 0, nil
		}
		if !autonomiaIdleAutoassignShouldAttempt(sesion.Agente, proyecto.ID) {
			return 0, nil
		}
		if err := revalidarYVerificarAgenteDisponibleParaTrabajo(sesion.Agente); err != nil {
			return 0, nil
		}
		tarea, err := db.IntentarAutoasignarTareaAgente(sesion.Agente, proyecto.ID)
		if err != nil || tarea == nil {
			return 0, err
		}
		motivo := fmt.Sprintf("Tarea #%d asignada automáticamente", tarea.ID)
		if pendiente, err := existeRuntimeOrderAutonomiaPendiente(sesion.Agente, &proyecto.ID, "nudge", "continuar_trabajo"); err != nil {
			return 0, err
		} else if pendiente {
			return 0, nil
		}
		if encolada, err := encolarNudgeAutonomiaDetallado(
			sesion.Agente,
			proyecto,
			"continuar_trabajo",
			motivo,
			"toma tarea asignada y sigue",
			map[string]any{"tarea_id": tarea.ID, "motivo_autoasignacion": "sesion_activa_idle"},
		); err != nil {
			return 0, err
		} else if encolada {
			if snapshot != nil {
				snapshot.invalidateAgent(sesion.Agente)
				snapshot.invalidateProject(proyecto.ID)
			}
			return 1, nil
		}
		return 0, nil
	default:
		return 0, nil
	}
}

func persistirPausaPorCuotaAutonomia(agente, motivo string) error {
	agente = strings.TrimSpace(agente)
	motivo = strings.TrimSpace(motivo)
	if agente == "" {
		return nil
	}
	infoAgente, err := agenteIfExists(agente)
	if err != nil {
		return err
	}
	if infoAgente == nil {
		return nil
	}
	if infoAgente != nil && strings.EqualFold(strings.TrimSpace(infoAgente.EstadoCuota), "enfriamiento") && infoAgente.ReanimarAt != nil && infoAgente.ReanimarAt.After(time.Now().UTC()) {
		return nil
	}
	p, _, err := db.UltimoPresupuestoAgente(agente)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil && p != nil && db.PresupuestoSesionFresco(p) && p.ResetAt != nil && p.ResetAt.After(time.Now().UTC()) {
		return db.PausarAgenteHasta(agente, p.ResetAt.UTC(), motivo)
	}
	return db.PausarAgente(agente, 60, motivo)
}

func procesarCierreProyectoSesion(sesion *db.Sesion) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	if policy, err := supervisionService.GetProjectPolicy(proyecto.Slug); err != nil {
		return 0, err
	} else if policy == nil || !policy.Enabled || !policy.AutoCloseProject {
		return 0, nil
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil || !terminado {
		return 0, err
	}
	if err := db.MarcarProyectoCerrado(proyecto.ID, motivo); err != nil {
		return 0, err
	}
	estadoSesion := strings.ToLower(strings.TrimSpace(sesion.Estado))
	if estadoSesion == "pausada" {
		if err := db.AparcarSesionActiva(sesion.Agente, sesion.ProyectoID); err != nil {
			return 0, err
		}
		if err := db.PausarAsignacion(sesion.Agente, proyecto.ID, "proyecto_terminado:"+motivo); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if satisfecha, err := pausaAutonomiaYaSatisfecha(sesion.Agente, &proyecto.ID, sesion); err != nil {
		return 0, err
	} else if satisfecha {
		return 0, nil
	}
	if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   sesion.Agente,
		Proyecto: proyecto.Slug,
		Accion:   agenteControlAccionPause,
		Motivo:   "proyecto_terminado:" + motivo,
		Por:      "orquesta",
	}); err != nil {
		return 0, err
	}
	return 1, nil
}

func procesarReanudacionAutonomaSesion(sesion *db.Sesion, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	if !strings.EqualFold(strings.TrimSpace(sesion.Estado), "pausada") {
		return 0, nil
	}
	tieneTrabajoActivo, err := sesionTieneTrabajoArrancableAutonomia(sesion, snapshot)
	if err != nil || !tieneTrabajoActivo {
		return 0, err
	}
	disponible, err := db.ProyectoDisponibleParaAutonomia(*sesion.ProyectoID)
	if err != nil || !disponible {
		return 0, err
	}
	if snapshot != nil {
		if shouldPause, _, err := snapshot.budgetPause(strings.TrimSpace(sesion.Agente)); err != nil {
			return 0, err
		} else if shouldPause {
			return 0, nil
		}
	}
	return reactivarSesionAutonomiaPorTrabajo(sesion)
}

func proyectoTerminadoAutonomamente(proyecto *db.Proyecto) (bool, string, error) {
	if proyecto == nil {
		return false, "", nil
	}
	policy, err := supervisionService.GetProjectPolicy(proyecto.Slug)
	if err != nil {
		return false, "", err
	}
	if policy == nil || !policy.Enabled || !policy.AutoCloseProject {
		return false, "", nil
	}
	listo, motivo, err := proyectoSinTrabajoPendiente(proyecto)
	if err != nil || !listo {
		return listo, motivo, err
	}
	reviewOK, reviewMotivo, err := proyectoReviewAutonomoCompletado(proyecto)
	if err != nil || !reviewOK {
		return false, reviewMotivo, err
	}
	if strings.TrimSpace(reviewMotivo) != "" {
		motivo += "; " + strings.TrimSpace(reviewMotivo)
	}
	integrado, integracionMotivo, err := proyectoIntegracionAutonomaCompletada(proyecto)
	if err != nil || !integrado {
		return false, integracionMotivo, err
	}
	if strings.TrimSpace(integracionMotivo) != "" {
		motivo += "; " + strings.TrimSpace(integracionMotivo)
	}
	return true, motivo, nil
}

func proyectoIntegracionAutonomaCompletada(proyecto *db.Proyecto) (bool, string, error) {
	if proyecto == nil {
		return false, "", nil
	}
	merges, err := gitgobernanza.NewService(gitgobernanza.Repository{}).ListRequests(strings.TrimSpace(proyecto.Slug), "")
	if err != nil {
		return false, "", err
	}
	if len(merges) == 0 {
		return true, "", nil
	}
	var latestMerged *db.GitMerge
	for _, merge := range merges {
		if merge == nil {
			continue
		}
		switch strings.TrimSpace(merge.Estado) {
		case "pendiente", "validando", "aprobado", "ejecutando":
			return false, fmt.Sprintf("esperando integración merge #%d (%s)", merge.ID, strings.TrimSpace(merge.Estado)), nil
		case "fallido":
			return false, fmt.Sprintf("integración merge #%d fallida", merge.ID), nil
		case "fusionado":
			if latestMerged == nil || merge.ID > latestMerged.ID {
				latestMerged = merge
			}
		}
	}
	if latestMerged != nil {
		return true, fmt.Sprintf("merge #%d fusionado", latestMerged.ID), nil
	}
	return false, "sin integración fusionada", nil
}

func procesarAparcadoAutonomoSesion(sesion *db.Sesion, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	tieneTrabajoActivo, err := sesionTieneTrabajoArrancableAutonomia(sesion, snapshot)
	if err != nil {
		return 0, err
	}
	op, err := db.GetProyectoOperacion(*sesion.ProyectoID)
	if err != nil {
		return 0, err
	}
	motivo := strings.TrimSpace(op.Motivo)
	switch op.EstadoOperativo {
	case db.ProyectoOperativoActivo:
		bloqueado, motivoDetectado, err := db.ResolverBloqueoProyecto(*sesion.ProyectoID)
		if err != nil || !bloqueado {
			return 0, err
		}
		motivo = strings.TrimSpace(motivoDetectado)
		if err := db.MarcarProyectoEsperandoHumano(*sesion.ProyectoID, motivo); err != nil {
			return 0, err
		}
	case db.ProyectoOperativoEsperandoHumano, db.ProyectoOperativoBloqueadoExterno:
		if tieneTrabajoActivo {
			disponible, err := db.ProyectoDisponibleParaAutonomia(*sesion.ProyectoID)
			if err != nil {
				return 0, err
			}
			if disponible {
				if strings.EqualFold(strings.TrimSpace(sesion.Estado), "pausada") {
					return reactivarSesionAutonomiaPorTrabajo(sesion)
				}
				return 0, nil
			}
		}
		if strings.TrimSpace(motivo) == "" {
			motivo = "esperando_desbloqueo_humano"
		}
	default:
		return 0, nil
	}
	if tieneTrabajoActivo {
		autonomiaTickDebugf("agente=%s proyecto_id=%d skip_aparcado trabajo_activo=true motivo=%s", strings.TrimSpace(sesion.Agente), valorProyectoID(sesion.ProyectoID), motivo)
		return 0, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	estadoSesion := strings.ToLower(strings.TrimSpace(sesion.Estado))
	if estadoSesion == "pausada" {
		if err := db.AparcarSesionActiva(sesion.Agente, sesion.ProyectoID); err != nil {
			return 0, err
		}
		if err := db.PausarAsignacion(sesion.Agente, *sesion.ProyectoID, "bloqueo_humano:"+motivo); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if satisfecha, err := pausaAutonomiaYaSatisfecha(sesion.Agente, sesion.ProyectoID, sesion); err != nil {
		return 0, err
	} else if satisfecha {
		return 0, nil
	}
	if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   sesion.Agente,
		Proyecto: proyecto.Slug,
		Accion:   agenteControlAccionPause,
		Motivo:   "bloqueo_humano:" + motivo,
		Por:      "orquesta",
	}); err != nil {
		return 0, err
	}
	return 1, nil
}

func sesionTieneTrabajoArrancableAutonomia(sesion *db.Sesion, snapshot *autonomiaBatchSnapshot) (bool, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return false, nil
	}
	agente := strings.TrimSpace(sesion.Agente)
	if agente == "" {
		return false, nil
	}
	if snapshot != nil {
		tareas, err := snapshot.tasks(agente, *sesion.ProyectoID)
		if err != nil {
			return false, err
		}
		for _, tarea := range tareas {
			if tarea == nil {
				continue
			}
			switch tarea.Estado {
			case db.TareaAsignada, db.TareaEnProgreso:
				return true, nil
			}
		}
		return false, nil
	}
	return dbAgenteTieneTrabajoArrancable(agente, *sesion.ProyectoID)
}

func reactivarSesionAutonomiaPorTrabajo(sesion *db.Sesion) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	if disponible, _, err := autonomiaCuentaCompartidaDisponible(strings.TrimSpace(sesion.Agente)); err != nil {
		return 0, err
	} else if !disponible {
		return 0, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	if err := db.ActivarAsignacion(strings.TrimSpace(sesion.Agente), *sesion.ProyectoID, "reactivacion_automatica_trabajo_activo"); err != nil {
		return 0, err
	}
	estadoActiva := "activa"
	if err := db.GuardarSesionActiva(strings.TrimSpace(sesion.Agente), sesion.ProyectoID, db.SesionUpdate{
		Estado:    &estadoActiva,
		Heartbeat: true,
	}); err != nil {
		return 0, err
	}
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(strings.TrimSpace(sesion.Agente), sesion.ProyectoID, "resume", "start", "pause", "handoff", "checkpoint"); err != nil {
		return 0, err
	} else if pendiente {
		return 0, nil
	}
	handle, err := resolverHandleReactivacionAgente(strings.TrimSpace(sesion.Agente), sesion.ProyectoID)
	if err != nil {
		return 0, err
	}
	accion := agenteControlAccionStart
	if handle != nil {
		switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
		case "activo":
			accion = agenteControlAccionResume
		case "pausado":
			if !db.RuntimeHandlePauseRequiresFreshStart(handle) {
				accion = agenteControlAccionResume
			}
		}
	}
	if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   strings.TrimSpace(sesion.Agente),
		Proyecto: proyecto.Slug,
		Accion:   accion,
		Motivo:   "desbloqueo_humano_auto",
		Por:      "orquesta",
	}); err != nil {
		return 0, err
	}
	return 1, nil
}

func resolverHandleReactivacionAgente(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	handles, err := db.ListarRuntimeHandles(&agente)
	if err != nil {
		return nil, err
	}
	var fallback *db.RuntimeHandle
	for _, handle := range handles {
		if handle == nil || runtimeHandleEsCandidatoLegacyATMUX(handle) {
			continue
		}
		estado := strings.ToLower(strings.TrimSpace(handle.Estado))
		if estado != "activo" && estado != "pausado" && estado != "fallido" {
			continue
		}
		if proyectoID != nil && *proyectoID > 0 && handle.ProyectoID != nil && *handle.ProyectoID == *proyectoID {
			return handle, nil
		}
		if fallback == nil {
			fallback = handle
		}
	}
	return fallback, nil
}

func procesarRecuperacionRuntimeDegradadoSesion(sesion *db.Sesion) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	if disponible, _, err := autonomiaCuentaCompartidaDisponible(strings.TrimSpace(sesion.Agente)); err != nil {
		return 0, err
	} else if !disponible {
		return 0, nil
	}
	handle, runtime, err := runtimeRecuperacionSesionObjetivo(sesion)
	if err != nil {
		return 0, err
	}
	if handle == nil {
		return 0, nil
	}
	if !esTransporteRemotoAutonomia(handle.Transporte) {
		handle, runtime, _, err = db.SincronizarRuntimeHandleSupervisado(handle, runtime, "autonomia_runtime_recovery")
		if err != nil {
			return 0, err
		}
		if !runtimeLocalFallido(handle, runtime) {
			return 0, nil
		}
		if pendiente, err := existeRuntimeOrderAbiertaAutonomia(sesion.Agente, sesion.ProyectoID, "start", "resume", "handoff"); err != nil {
			return 0, err
		} else if pendiente {
			return 0, nil
		}
		proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
		if err != nil {
			return 0, err
		}
		tieneTrabajo, err := dbAgenteTieneTrabajoArrancable(strings.TrimSpace(sesion.Agente), proyecto.ID)
		if err != nil {
			return 0, err
		}
		if !tieneTrabajo {
			return 0, nil
		}
		perfilPersistido, modeloPersistido, razonamientoPersistido := db.ResumePayloadPerfilEjecucion(sesion.ResumePayloadJSON)

		// Sanear modelo si es legacy/placeholder para forzar re-resolución hexagonal en la recuperación
		if strings.Contains(modeloPersistido, "gpt-5") || modeloPersistido == "" {
			modeloPersistido = ""
			razonamientoPersistido = ""
		}

		if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
			Agente:       strings.TrimSpace(sesion.Agente),
			Proyecto:     proyecto.Slug,
			Accion:       agenteControlAccionStart,
			Modelo:       modeloPersistido,
			Razonamiento: razonamientoPersistido,
			Perfil:       perfilPersistido,
			Motivo:       "local_runtime_failed",
			Por:          "orquesta",
		}); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if !runtimeRemotoDegradado(handle, runtime) {
		return 0, nil
	}
	conector, err := resolverConectorSesionAutonomia(sesion, runtime, handle)
	if err != nil {
		return 0, err
	}
	if conector != nil {
		disponible, _, err := db.ConectorDisponibleParaArranque(conector.ID)
		if err != nil {
			return 0, err
		}
		if !disponible {
			motivo := "conector:" + strings.TrimSpace(conector.Slug) + ":circuito_abierto"
			if err := db.MarcarProyectoBloqueadoExterno(*sesion.ProyectoID, motivo); err != nil {
				return 0, err
			}
			proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
			if err != nil {
				return 0, err
			}
			if satisfecha, err := pausaAutonomiaYaSatisfecha(sesion.Agente, sesion.ProyectoID, sesion); err != nil {
				return 0, err
			} else if satisfecha {
				return 1, nil
			}
			if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
				Agente:   strings.TrimSpace(sesion.Agente),
				Proyecto: proyecto.Slug,
				Accion:   agenteControlAccionPause,
				Motivo:   motivo,
				Por:      "orquesta",
			}); err != nil {
				return 0, err
			}
			return 1, nil
		}
	}
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(sesion.Agente, sesion.ProyectoID, "checkpoint", "start", "resume", "handoff"); err != nil {
		return 0, err
	} else if pendiente {
		return 0, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	checkpointPayload, err := json.Marshal(map[string]any{
		"checkpoint_kind": "remote_recovery",
		"resumen":         "Checkpoint automático antes de recuperación de runtime remoto degradado",
		"motivo":          "remote_runtime_degraded",
	})
	if err != nil {
		return 0, err
	}
	if _, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:     strings.TrimSpace(sesion.Agente),
		ProyectoID: sesion.ProyectoID,
		RuntimeID: func() *int64 {
			if runtime != nil && runtime.ID > 0 {
				return &runtime.ID
			}
			return handle.RuntimeID
		}(),
		HandleID:    &handle.ID,
		Tipo:        "checkpoint",
		PayloadJSON: string(checkpointPayload),
	}); err != nil {
		return 0, err
	}
	accion := agenteControlAccionStart
	if runtimeRemotoReanudable(sesion, handle) {
		accion = agenteControlAccionResume
	}
	if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   strings.TrimSpace(sesion.Agente),
		Proyecto: proyecto.Slug,
		Accion:   accion,
		Motivo:   "remote_runtime_degraded",
		Por:      "orquesta",
	}); err != nil {
		return 0, err
	}
	return 2, nil
}

func runtimeRecuperacionSesionObjetivo(sesion *db.Sesion) (*db.RuntimeHandle, *db.RuntimeInstance, error) {
	if sesion == nil {
		return nil, nil, nil
	}
	agente := strings.TrimSpace(sesion.Agente)
	proyectoID := sesion.ProyectoID
	var (
		handle       *db.RuntimeHandle
		sesionHandle *db.RuntimeHandle
		runtime      *db.RuntimeInstance
		err          error
	)
	if sesion.ID > 0 {
		sesionHandle, err = db.GetRuntimeHandleBySesionID(sesion.ID)
		if err != nil {
			return nil, nil, err
		}
		handle = sesionHandle
	}
	if agente != "" {
		var candidate *db.RuntimeHandle
		if proyectoID != nil && *proyectoID > 0 {
			candidate, err = db.GetRuntimeHandleOperativoRecienteAgenteProyecto(agente, proyectoID)
		} else {
			candidate, err = db.GetRuntimeHandleOperativoRecienteAgente(agente)
		}
		if err != nil {
			return nil, nil, err
		}
		if candidate != nil && (sesionHandle == nil || candidate.ID != sesionHandle.ID) {
			handle = candidate
		}
	}
	runtime, err = runtimeCanonicoDesdeHandleAgenteProyecto(handle, agente, proyectoID)
	if err != nil {
		return nil, nil, err
	}
	if runtime == nil && agente != "" {
		runtime, err = db.GetRuntimePrincipalAgenteProyecto(agente, proyectoID)
		if err != nil {
			return nil, nil, err
		}
	}
	if runtime == nil && sesion.ID > 0 {
		runtime, err = db.GetRuntimeBySesionID(sesion.ID)
		if err != nil {
			return nil, nil, err
		}
	}
	return handle, runtime, nil
}

func runtimeCanonicoDesdeHandleAgenteProyecto(handle *db.RuntimeHandle, agente string, proyectoID *int64) (*db.RuntimeInstance, error) {
	if handle != nil {
		if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
			runtime, err := db.GetRuntime(*handle.RuntimeID)
			if err == nil && runtime != nil {
				return runtime, nil
			}
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}
		}
		if handle.SesionID != nil && *handle.SesionID > 0 {
			runtime, err := db.GetRuntimeBySesionID(*handle.SesionID)
			if err == nil && runtime != nil {
				return runtime, nil
			}
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}
		}
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	runtime, err := db.GetRuntimePrincipalAgenteProyecto(agente, proyectoID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return runtime, nil
}

func runtimeIDCanonicoDesdeHandleAgenteProyecto(handle *db.RuntimeHandle, agente string, proyectoID *int64) (*int64, error) {
	runtime, err := runtimeCanonicoDesdeHandleAgenteProyecto(handle, agente, proyectoID)
	if err != nil || runtime == nil || runtime.ID <= 0 {
		return nil, err
	}
	return &runtime.ID, nil
}

func runtimeLocalFallido(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) bool {
	if handle == nil || esTransporteRemotoAutonomia(handle.Transporte) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Estado), "fallido") {
		return true
	}
	if runtime == nil {
		return false
	}
	logical := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
	process := strings.ToLower(strings.TrimSpace(runtime.ProcessState))
	return logical == "fallido" || process == "fallido" || process == "crashed" || process == "exited"
}

func resolverConectorSesionAutonomia(sesion *db.Sesion, runtime *db.RuntimeInstance, handle *db.RuntimeHandle) (*db.Conector, error) {
	if sesion != nil && strings.TrimSpace(sesion.ConectorSlug) != "" {
		return db.GetConector(strings.TrimSpace(sesion.ConectorSlug))
	}
	if runtime != nil && strings.TrimSpace(runtime.Connector) != "" {
		return db.GetConector(strings.TrimSpace(runtime.Connector))
	}
	if handle != nil {
		slug := strings.TrimSpace(stringFromMetadataJSON(handle.MetadataJSON, "conector"))
		if slug != "" {
			return db.GetConector(slug)
		}
	}
	return nil, nil
}

func stringFromMetadataJSON(raw string, key string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil || parsed == nil {
		return ""
	}
	value, ok := parsed[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func esTransporteRemotoAutonomia(transport string) bool {
	switch strings.TrimSpace(transport) {
	case "api", "mcp_http", "otro":
		return true
	default:
		return false
	}
}

func runtimeRemotoDegradado(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) bool {
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "fallido") {
		return true
	}
	if runtime == nil {
		return false
	}
	logical := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
	process := strings.ToLower(strings.TrimSpace(runtime.ProcessState))
	return logical == "degradado" || process == "remote_status_error"
}

func runtimeRemotoReanudable(sesion *db.Sesion, handle *db.RuntimeHandle) bool {
	if sesion == nil || handle == nil {
		return false
	}
	if !esTransporteRemotoAutonomia(handle.Transporte) {
		return false
	}
	if strings.TrimSpace(handle.HandleKind) == "process" {
		return false
	}
	if ext := strings.TrimSpace(sesion.ExternalSessionID); ext != "" {
		return true
	}
	if ext := strings.TrimSpace(stringFromMetadataJSON(handle.MetadataJSON, "external_session_id")); ext != "" {
		return true
	}
	ref := strings.TrimSpace(handle.HandleRef)
	if ref == "" {
		return false
	}
	return ref != strconv.FormatInt(sesion.ID, 10)
}

func reactivarAgenteTrasReanimacion(agente string) error {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil
	}
	if disponible, _, err := autonomiaCuentaCompartidaDisponible(agente); err != nil {
		return err
	} else if !disponible {
		return nil
	}
	proyecto, err := resolverProyectoReactivacionAgente(agente)
	if err != nil {
		return err
	}
	if proyecto == nil {
		return nil
	}
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(agente, &proyecto.ID, "resume", "start", "handoff"); err != nil {
		return err
	} else if pendiente {
		return nil
	}
	handle, err := resolverHandleReactivacionAgente(agente, &proyecto.ID)
	if err != nil {
		return err
	}
	if handle != nil && strings.TrimSpace(handle.Estado) == "pausado" {
		accion := agenteControlAccionResume
		if db.RuntimeHandlePauseRequiresFreshStart(handle) {
			accion = agenteControlAccionStart
		}
		_, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
			Agente:   agente,
			Proyecto: proyecto.Slug,
			Accion:   accion,
			Motivo:   "reanimacion_automatica",
			Por:      "orquesta",
		})
		return err
	}
	tieneTrabajo, err := dbAgenteTieneTrabajoArrancable(agente, proyecto.ID)
	if err != nil {
		return err
	}
	if !tieneTrabajo {
		tieneBacklog, err := dbProyectoTieneBacklogReactivable(proyecto.ID)
		if err != nil {
			return err
		}
		if !tieneBacklog {
			return nil
		}
	}
	_, _, err = encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   agente,
		Proyecto: proyecto.Slug,
		Accion:   agenteControlAccionStart,
		Motivo:   "reanimacion_automatica",
		Por:      "orquesta",
	})
	return err
}

func resolverProyectoReactivacionAgente(agente string) (*db.Proyecto, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	if proyectoID, err := db.ObtenerProyectoActivoAgente(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return runtimesService.GetProject(strconv.FormatInt(proyectoID, 10))
	}
	if proyectoID, err := proyectoReactivacionDesdeTareas(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return runtimesService.GetProject(strconv.FormatInt(proyectoID, 10))
	}
	if sesion, err := db.GetSesionAbierta(agente, nil); err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if sesion != nil && sesion.ProyectoID != nil && *sesion.ProyectoID > 0 {
		return runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	}
	if sesion, err := db.ObtenerUltimaSesion(agente, nil); err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if sesion != nil && sesion.ProyectoID != nil && *sesion.ProyectoID > 0 {
		return runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	}
	if handle, err := resolverHandleReactivacionAgente(agente, nil); err != nil {
		return nil, err
	} else if handle != nil && handle.ProyectoID != nil && *handle.ProyectoID > 0 {
		return runtimesService.GetProject(strconv.FormatInt(*handle.ProyectoID, 10))
	}
	return nil, nil
}

func proyectoReactivacionDesdeTareas(agente string) (int64, error) {
	filtro := db.FiltroTareas{Agente: &agente}
	tareas, err := tareasService.List(filtro)
	if err != nil {
		return 0, err
	}
	bestProjectID := int64(0)
	bestRank := 99
	bestID := int64(0)
	rankForState := func(estado db.EstadoTarea) int {
		switch estado {
		case db.TareaEnProgreso:
			return 0
		case db.TareaAsignada:
			return 1
		case db.TareaBloqueada:
			return 2
		default:
			return 99
		}
	}
	for _, tarea := range tareas {
		if tarea == nil || tarea.ProyectoID == nil || *tarea.ProyectoID <= 0 {
			continue
		}
		rank := rankForState(tarea.Estado)
		if rank > 2 {
			continue
		}
		if bestProjectID == 0 || rank < bestRank || (rank == bestRank && tarea.ID > bestID) {
			bestProjectID = *tarea.ProyectoID
			bestRank = rank
			bestID = tarea.ID
		}
	}
	return bestProjectID, nil
}

func dbProyectoTieneBacklogReactivable(proyectoID int64) (bool, error) {
	filtro := db.FiltroTareas{ProyectoID: &proyectoID}
	tareas, err := tareasService.List(filtro)
	if err != nil {
		return false, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaLibre:
			return true, nil
		case db.TareaBloqueada:
			recuperable, err := tareaBloqueadaRecuperableParaReanimacion(tarea.ID)
			if err != nil {
				return false, err
			}
			if recuperable {
				return true, nil
			}
		}
	}
	return false, nil
}

func tareaBloqueadaRecuperableParaReanimacion(tareaID int64) (bool, error) {
	motivo, err := db.MotivoBloqueoActivoTarea(tareaID)
	if err != nil {
		return false, err
	}
	texto := strings.TrimSpace(motivo)
	if texto == "" {
		return false, nil
	}
	if esBloqueoSobrecargaOperativa(texto) {
		return true, nil
	}
	return strings.HasPrefix(texto, "Agente ") || strings.HasPrefix(texto, "Agente degradado:"), nil
}

func existeRuntimeOrderAutonomiaPendiente(agente string, proyectoID *int64, tipo string, accion string) (bool, error) {
	estado := "pendiente"
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != strings.TrimSpace(tipo) {
			continue
		}
		if accion == "" {
			return true, nil
		}
		if strings.Contains(strings.ToLower(order.PayloadJSON), `"kind":"autonomia"`) &&
			strings.Contains(strings.ToLower(order.PayloadJSON), fmt.Sprintf(`"accion":"%s"`, strings.ToLower(strings.TrimSpace(accion)))) {
			return true, nil
		}
	}
	return false, nil
}

func pausaAutonomiaYaSatisfecha(agente string, proyectoID *int64, sesion *db.Sesion) (bool, error) {
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(agente, proyectoID, "pause", ""); err != nil {
		return false, err
	} else if pendiente {
		return true, nil
	}
	if sesion != nil && strings.EqualFold(strings.TrimSpace(sesion.Estado), "pausada") {
		return true, nil
	}
	handleSesion, err := runtimeHandleSesionPersistenteAutonomia(sesion)
	if err != nil {
		return false, err
	}
	estadoCuotaPersistido, err := db.GetPersistedAgentQuotaState(strings.TrimSpace(agente))
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(estadoCuotaPersistido)) {
	case "enfriamiento", "agotado":
		if runtimeHandleEstadoEsPausado(handleSesion) || handleSesion == nil {
			return true, nil
		}
		handle, err := resolverHandleControlAgente(strings.TrimSpace(agente), proyectoID)
		if err != nil {
			return false, err
		}
		if handle == nil || strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
			return true, nil
		}
	}
	if runtimeHandleEstadoEsPausado(handleSesion) {
		return true, nil
	}
	handle, err := resolverHandleControlAgente(strings.TrimSpace(agente), proyectoID)
	if err != nil {
		return false, err
	}
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		return true, nil
	}
	return false, nil
}

func runtimeHandleSesionPersistenteAutonomia(sesion *db.Sesion) (*db.RuntimeHandle, error) {
	if sesion == nil || sesion.ID <= 0 {
		return nil, nil
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil || handle.ID <= 0 {
		return handle, err
	}
	return db.GetRuntimeHandle(handle.ID)
}

func runtimeHandleEstadoEsPausado(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado")
}

func existeRuntimeOrderAutonomiaReciente(agente string, proyectoID *int64, tipo string, accion string, within time.Duration) (bool, error) {
	n, err := contarRuntimeOrdersRecientes(agente, proyectoID, tipo, accion, within)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func contarRuntimeOrdersRecientes(agente string, proyectoID *int64, tipo string, accion string, within time.Duration) (int, error) {
	if within <= 0 {
		return 0, nil
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
	})
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().UTC().Add(-within)
	total := 0
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != strings.TrimSpace(tipo) {
			continue
		}
		if accion != "" && !runtimeOrderTieneAccion(order, accion) {
			continue
		}
		if runtimeOrderTimestamp(order).Before(cutoff) {
			continue
		}
		total++
	}
	return total, nil
}

func runtimeOrderTieneAccion(order *db.RuntimeOrder, accion string) bool {
	if order == nil {
		return false
	}
	accion = strings.TrimSpace(strings.ToLower(accion))
	if accion == "" {
		return false
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(order.PayloadJSON)), &payload); err == nil {
		if text, ok := payload["accion"].(string); ok && strings.EqualFold(strings.TrimSpace(text), accion) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(order.PayloadJSON), fmt.Sprintf(`"accion":"%s"`, accion))
}

func runtimeOrderTimestamp(order *db.RuntimeOrder) time.Time {
	if order == nil {
		return time.Time{}
	}
	if order.FinishedAt != nil && !order.FinishedAt.IsZero() {
		return order.FinishedAt.UTC()
	}
	if order.StartedAt != nil && !order.StartedAt.IsZero() {
		return order.StartedAt.UTC()
	}
	if !order.UpdatedAt.IsZero() {
		return order.UpdatedAt.UTC()
	}
	return order.CreatedAt.UTC()
}

func existeRuntimeOrderAbiertaAutonomia(agente string, proyectoID *int64, tipos ...string) (bool, error) {
	if len(tipos) == 0 {
		return false, nil
	}
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return false, err
		}
		for _, order := range orders {
			if order == nil {
				continue
			}
			for _, tipo := range tipos {
				if strings.TrimSpace(order.Tipo) == strings.TrimSpace(tipo) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func existeRuntimeOrderAbiertaAgente(agente string, proyectoID *int64) (bool, error) {
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return false, err
		}
		if len(orders) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func dbAgenteTieneTrabajoArrancable(agente string, proyectoID int64) (bool, error) {
	filtro := db.FiltroTareas{
		Agente:     &agente,
		ProyectoID: &proyectoID,
	}
	tareas, err := tareasService.List(filtro)
	if err != nil {
		return false, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaAsignada, db.TareaEnProgreso:
			return true, nil
		}
	}
	return false, nil
}

func encolarNudgeAutonomia(agente string, proyecto *db.Proyecto, accion, motivo string) (bool, error) {
	return encolarNudgeAutonomiaDetallado(agente, proyecto, accion, motivo, "", nil)
}

func encolarNudgeAutonomiaDetallado(agente string, proyecto *db.Proyecto, accion, motivo, instruction string, extras map[string]any) (bool, error) {
	if proyecto == nil {
		return false, nil
	}
	agente = strings.TrimSpace(agente)
	accion = strings.TrimSpace(accion)
	if abierta, err := existeRuntimeOrderAbiertaAutonomia(strings.TrimSpace(agente), &proyecto.ID, "send_instruction"); err != nil {
		return false, err
	} else if abierta {
		return false, nil
	}
	cooldown := time.Duration(controlPlaneConfigIntOrDefault("autonomia_nudge_cooldown_seconds", 60)) * time.Second
	if reciente, err := existeRuntimeOrderAutonomiaReciente(strings.TrimSpace(agente), &proyecto.ID, "nudge", "", cooldown); err != nil {
		return false, err
	} else if reciente {
		return false, nil
	}
	var runtimeID *int64
	var handleID *int64
	handle, err := resolverHandleEntregaAgente(strings.TrimSpace(agente), &proyecto.ID)
	if err != nil {
		return false, err
	}
	if handle != nil {
		handleID = &handle.ID
		runtimeID, err = runtimeIDCanonicoDesdeHandleAgenteProyecto(handle, strings.TrimSpace(agente), &proyecto.ID)
		if err != nil {
			return false, err
		}
		if runtimeID == nil && handle.RuntimeID != nil {
			runtimeID = handle.RuntimeID
		}
		if !db.RuntimeHandlePermiteSendInputInteractivo(handle) {
			if pendiente, err := existeRuntimeMailboxAutonomiaPendiente(agente, &proyecto.ID, accion); err != nil {
				return false, err
			} else if pendiente {
				return false, nil
			}
		}
	}
	payload := map[string]any{
		"from_agente": "server",
		"to_agente":   agente,
		"kind":        "autonomia",
		"accion":      accion,
		"texto":       strings.TrimSpace(motivo),
	}
	if strings.TrimSpace(instruction) != "" {
		payload["instruction"] = strings.TrimSpace(instruction)
	}
	for key, value := range extras {
		payload[key] = value
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	orderID, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      strings.TrimSpace(agente),
		ProyectoID:  &proyecto.ID,
		RuntimeID:   runtimeID,
		HandleID:    handleID,
		Tipo:        "nudge",
		PayloadJSON: string(payloadJSON),
	})
	if err != nil {
		return false, err
	}
	db.Audit("orquesta", "autonomia_nudge", "runtime_order", orderID,
		fmt.Sprintf("agente=%s proyecto=%s accion=%s", agente, proyecto.Slug, accion))
	return true, nil
}

func existeRuntimeMailboxAutonomiaPendiente(agente string, proyectoID *int64, accion string) (bool, error) {
	estado := "pendiente"
	agente = strings.TrimSpace(agente)
	accion = strings.TrimSpace(accion)
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return false, err
	}
	for _, msg := range mailbox {
		if msg == nil || strings.TrimSpace(msg.Kind) != "autonomia" {
			continue
		}
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(msg.PayloadJSON), &payload)
		if strings.TrimSpace(stringMapValue(payload, "accion")) == accion {
			return true, nil
		}
	}
	return false, nil
}
