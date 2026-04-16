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
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitgobernanza"
	"orquesta/internal/controlruntime"
	"orquesta/notificaciones"
	"orquesta/planocontrol"
	"orquesta/reviewapp"
	"orquesta/runtimeagente"
	"orquesta/runtimesapp"
	"orquesta/tareasapp"
)

type dbAutomationService struct{}

const autonomiaReplanTaskTitle = "Autonomía: replanificar backlog y abrir siguiente frente útil"

var runtimeBudgetObservationBackgroundGate = planocontrol.NewGate()

var wakeRuntimeOrdersAfterTranscript = wakeControlPlaneRuntimeOrders
var wakeRuntimeMailboxAfterTranscript = wakeControlPlaneRuntimeMailbox
var wakeRuntimeOrdersAfterMailbox = wakeControlPlaneRuntimeOrders

var runtimeMailboxReevaluationGate = planocontrol.NewThrottler()

func runtimeMailboxReevaluationInterval() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("runtime_mailbox_reevaluation_interval_seconds", 10)
	if seconds <= 0 {
		seconds = 10
	}
	return time.Duration(seconds) * time.Second
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
	return runtimeMailboxReevaluationGate.Allow(key, runtimeMailboxReevaluationInterval())
}

func resetRuntimeMailboxReevaluationGate() {
	runtimeMailboxReevaluationGate.Reset()
}

func resetAutonomiaIdleAutoassignGate() {
	autonomiaIdleAutoassignGate.Reset()
}

func resetAutonomiaDegradedTaskGate() {
	autonomiaDegradedTaskGate.Reset()
}

func resetPresupuestoPrimerUsoSesionGate() {
	presupuestoPrimerUsoSesionGate.Reset()
}

func resetPipelineLocalDispatchGate() {
	pipelineLocalDispatchGate.Reset()
}

var autonomiaActiveSessionsGate = planocontrol.NewGate()
var procesarAutonomiaSesionActivaBatchFn = procesarAutonomiaSesionActivaConSnapshot
var procesarAgentesDegradadosAutonomiaBatchFn = procesarAgentesDegradadosAutonomiaBatch
var autonomiaSessionStepTimeoutOverride time.Duration
var autonomiaDegradedBatchTimeoutOverride time.Duration

var autonomiaIdleAutoassignGate = planocontrol.NewThrottler()

var autonomiaDegradedTaskGate = planocontrol.NewThrottler()

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

var presupuestoPrimerUsoSesionGate = planocontrol.NewThrottler()
var autonomiaContinueNudgeGate = planocontrol.NewThrottler()

func autonomiaIdleAutoassignInterval() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("autonomia_idle_autoassign_interval_seconds", 300)
	if seconds <= 0 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

func autonomiaContinueNudgeInterval() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("autonomia_continue_nudge_interval_seconds", 60)
	if seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func autonomiaSessionStepTimeout() time.Duration {
	if autonomiaSessionStepTimeoutOverride > 0 {
		return autonomiaSessionStepTimeoutOverride
	}
	seconds := controlPlaneConfigIntOrDefault("autonomia_session_step_timeout_seconds", 20)
	if seconds <= 0 {
		seconds = 20
	}
	return time.Duration(seconds) * time.Second
}

func autonomiaDegradedBatchTimeout() time.Duration {
	if autonomiaDegradedBatchTimeoutOverride > 0 {
		return autonomiaDegradedBatchTimeoutOverride
	}
	seconds := controlPlaneConfigIntOrDefault("autonomia_degraded_batch_timeout_seconds", 20)
	if seconds <= 0 {
		seconds = 20
	}
	return time.Duration(seconds) * time.Second
}

func pipelineLocalDispatchInterval() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("pipeline_local_dispatch_interval_seconds", 180)
	if seconds <= 0 {
		seconds = 180
	}
	return time.Duration(seconds) * time.Second
}

var pipelineLocalDispatchGate = planocontrol.NewThrottler()

func pipelineLocalDispatchShouldAttempt(key string) bool {
	return pipelineLocalDispatchGate.Allow(key, pipelineLocalDispatchInterval())
}

func pipelineLocalDispatchKey(proyecto string, despacho *capacidadapp.DespachoPipelineLocal) string {
	if despacho == nil {
		return strings.TrimSpace(proyecto)
	}
	partes := []string{
		strings.TrimSpace(proyecto),
		strings.ToLower(strings.TrimSpace(despacho.Fase)),
		strings.ToLower(strings.TrimSpace(despacho.Carril)),
		strings.ToLower(strings.TrimSpace(despacho.AccionTarea)),
		strconv.FormatInt(despacho.TareaObjetivoID, 10),
		strings.ToLower(strings.TrimSpace(despacho.AgenteSugerido)),
	}
	return strings.Join(partes, "|")
}

func despachoPipelineRequiereRuntime(despacho *capacidadapp.DespachoPipelineLocal) bool {
	if despacho == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(despacho.Carril)) {
	case "premium_worktree", "revision_diff", "microprogramacion_local":
		return true
	default:
		return false
	}
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
	return autonomiaIdleAutoassignGate.Allow(key, autonomiaIdleAutoassignInterval())
}

func autonomiaContinueNudgeShouldAttempt(agente string, proyectoID, tareaID int64) bool {
	key := autonomiaContinueNudgeThrottleKey(agente, proyectoID, tareaID)
	if key == "" {
		return false
	}
	return autonomiaContinueNudgeGate.Allow(key, autonomiaContinueNudgeInterval())
}

func autonomiaContinueNudgeThrottleKey(agente string, proyectoID, tareaID int64) string {
	agente = strings.ToLower(strings.TrimSpace(agente))
	if agente == "" || proyectoID <= 0 || tareaID <= 0 {
		return ""
	}
	return agente + "|" + strconv.FormatInt(proyectoID, 10) + "|" + strconv.FormatInt(tareaID, 10)
}

func presupuestoPrimerUsoSesion(nombre string) bool {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	if nombre == "" {
		return false
	}
	return presupuestoPrimerUsoSesionGate.Allow(nombre, 365*24*time.Hour)
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
	candidatos, err := db.CheckReanimaciones()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	porNombre := map[string]*db.Agente{}
	for _, agente := range candidatos {
		if agente == nil {
			continue
		}
		porNombre[strings.ToLower(strings.TrimSpace(agente.Nombre))] = agente
	}
	tempranos, err := listarReanimacionesTempranasPorCuotaVisible(now)
	if err != nil {
		return nil, err
	}
	for _, agente := range tempranos {
		if agente == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(agente.Nombre))
		if key == "" {
			continue
		}
		if _, ok := porNombre[key]; !ok {
			porNombre[key] = agente
		}
	}
	filtrados := make([]*db.Agente, 0, len(porNombre))
	for _, agente := range porNombre {
		if !agenteDebeEntrarEnReanimacionAutomatica(agente, now) {
			continue
		}
		filtrados = append(filtrados, agente)
	}
	sort.SliceStable(filtrados, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(filtrados[i].Nombre)) < strings.ToLower(strings.TrimSpace(filtrados[j].Nombre))
	})
	return filtrados, nil
}

func listarReanimacionesTempranasPorCuotaVisible(now time.Time) ([]*db.Agente, error) {
	rows, err := agentesService.BuildReanimationSchedule(true, true)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	agentes, err := agentesService.ListAgents()
	if err != nil {
		return nil, err
	}
	porNombre := make(map[string]*db.Agente, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(agente.Nombre))
		if key == "" {
			continue
		}
		porNombre[key] = agente
	}
	candidatos := make([]*db.Agente, 0, len(rows))
	for _, row := range rows {
		if row.Due {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(row.Name))
		agente := porNombre[key]
		if agente == nil {
			continue
		}
		if agente.ReanimarAt == nil || agente.ReanimarAt.IsZero() || !agente.ReanimarAt.After(now) {
			continue
		}
		if !agentePuedeReanimarseAntesPorCuotaVisible(agente) {
			continue
		}
		candidatos = append(candidatos, agente)
	}
	return candidatos, nil
}

type resetReanimacionResultado string

const (
	resetReanimacionResultadoReactivado        resetReanimacionResultado = "reactivado"
	resetReanimacionResultadoCooldownSostenido resetReanimacionResultado = "cooldown_sostenido"
)

func agenteDebeEntrarEnReanimacionAutomatica(agente *db.Agente, now time.Time) bool {
	if agente == nil || strings.TrimSpace(agente.Nombre) == "" || !agente.Habilitado {
		return false
	}
	reanimacionVencida := agente.ReanimarAt != nil && !agente.ReanimarAt.IsZero() && !agente.ReanimarAt.After(now)
	reanimacionRecuperadaPorCuota := agentePuedeReanimarseAntesPorCuotaVisible(agente)
	if !reanimacionVencida && !reanimacionRecuperadaPorCuota {
		return false
	}
	if agenteMotivoPausaOperativa(agente.MotivoPausa) {
		return true
	}
	estadoCuota := strings.ToLower(strings.TrimSpace(agente.EstadoCuota))
	return estadoCuota == "enfriamiento" || estadoCuota == "agotado"
}

func agentePuedeReanimarseAntesPorCuotaVisible(agente *db.Agente) bool {
	if agente == nil || db.AgenteSinCuotaProveedorEfectivo(agente) {
		return false
	}
	estadoCuota := strings.ToLower(strings.TrimSpace(agente.EstadoCuota))
	if estadoCuota != "enfriamiento" && estadoCuota != "agotado" {
		return false
	}
	if agente.PresupuestoStale {
		return false
	}
	shouldPause, _ := agenteDebePausarPorPresupuestoVisible(agente)
	if shouldPause {
		return false
	}
	if agente.CuotaRestantePct != nil && *agente.CuotaRestantePct > 0 {
		return true
	}
	return strings.TrimSpace(agente.PresupuestoEstado) == "ok"
}

func (dbAutomationService) ResetReanimacion(nombre string) error {
	_, err := (dbAutomationService{}).ResetReanimacionResultado(nombre)
	return err
}

func (dbAutomationService) ResetReanimacionOutcome(nombre string) (string, error) {
	resultado, err := (dbAutomationService{}).ResetReanimacionResultado(nombre)
	return string(resultado), err
}

func (dbAutomationService) ResetReanimacionResultado(nombre string) (resetReanimacionResultado, error) {
	nombre = strings.TrimSpace(nombre)
	if _, err := revalidarPresupuestoAgenteSiCorresponde(nombre, presupuestoPreflightRevalidationAge(), true); err != nil {
		return "", err
	}
	bloqueado, err := sostenerCooldownSiLaCuotaVisibleSigueBloqueada(nombre)
	if err != nil {
		return "", err
	}
	if bloqueado {
		return resetReanimacionResultadoCooldownSostenido, nil
	}
	if err := cancelarBootstrapObsoletoReanimacion(nombre); err != nil {
		return "", err
	}
	if err := registrarCheckpointRehabilitacionManual(nombre); err != nil {
		return "", err
	}
	if err := reactivarAgenteTrasReanimacionConMotivo(nombre, "manual_rehabilitation"); err != nil {
		return "", err
	}
	if err := db.ResetReanimacion(nombre); err != nil {
		return "", err
	}
	resetStatusSnapshotCache()
	return resetReanimacionResultadoReactivado, nil
}

func cancelarBootstrapObsoletoReanimacion(agente string) error {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil
	}
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
			Agente: &agente,
			Estado: &estado,
		})
		if err != nil {
			return err
		}
		for _, order := range orders {
			if order == nil {
				continue
			}
			tipo := strings.ToLower(strings.TrimSpace(order.Tipo))
			if tipo != "handoff" && tipo != "resume" {
				continue
			}
			resultado := `{"ok":false,"dispatch_state":"cancelled","bootstrap_stale":true,"superseded_reason":"manual_rehabilitation"}`
			detalle := "rehabilitacion manual del agente"
			if err := db.MarcarRuntimeOrderEstado(order.ID, "cancelada", resultado, detalle); err != nil {
				return err
			}
			db.Audit("orquesta", "rehabilitar_agente_cancela_bootstrap_obsoleto", "runtime_order", order.ID,
				fmt.Sprintf("agente=%s tipo=%s estado_previo=%s", agente, strings.TrimSpace(order.Tipo), estado))
		}
	}
	if err := db.LimpiarContinuidadAgente(agente); err != nil {
		return err
	}
	return nil
}

func registrarCheckpointRehabilitacionManual(agente string) error {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil
	}
	proyecto, err := resolverProyectoReactivacionAgente(agente)
	if err != nil {
		return err
	}
	if proyecto == nil {
		return nil
	}
	if _, err := runtimesService.CreateRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         agente,
		ProyectoID:     &proyecto.ID,
		CheckpointKind: "manual_rehabilitation",
		Resumen:        "Corte de continuidad tras rehabilitación manual",
		PayloadJSON:    "{}",
		ResumeStrategy: "resumen_y_payload",
		Source:         "manual_rehabilitation",
	}); err != nil {
		return err
	}
	db.Audit("orquesta", "checkpoint_rehabilitacion_manual", "agente", 0,
		fmt.Sprintf("agente=%s proyecto=%s", agente, strings.TrimSpace(proyecto.Slug)))
	return nil
}

func (dbAutomationService) GarantizarSaludAgentes() error {
	return db.GarantizarSaludAgentes()
}

func (dbAutomationService) TieneTrabajoOrquestablePendiente() (bool, string, error) {
	return tieneTrabajoOrquestablePendiente()
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
	return runtimeBudgetObservationBackgroundGate.AllowAt(runtimeBudgetBackgroundObservationInterval(), now)
}

func allowAutonomiaActiveSessionsObservation(now time.Time) bool {
	return autonomiaActiveSessionsGate.AllowAt(autonomiaActiveSessionsInterval(), now)
}

func resetRuntimeBudgetObservationBackgroundGate() {
	runtimeBudgetObservationBackgroundGate.Reset()
}

func resetAutonomiaActiveSessionsObservationGate() {
	autonomiaActiveSessionsGate.Reset()
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

func tieneTrabajoOrquestablePendiente() (bool, string, error) {
	proyectos, err := db.ListarProyectosActivos()
	if err != nil {
		return false, "", err
	}
	for _, proyecto := range proyectos {
		if _, ok := pipelineLocalProyectoElegible(proyecto); !ok {
			continue
		}
		if pending, detail, err := proyectoTieneTrabajoOrquestablePendiente(proyecto); err != nil {
			return false, "", err
		} else if pending {
			return true, detail, nil
		}
	}
	return false, "", nil
}

func proyectoTieneTrabajoOrquestablePendiente(proyecto *db.Proyecto) (bool, string, error) {
	if proyecto == nil || proyecto.ID <= 0 {
		return false, "", nil
	}
	if pending, detail, err := proyectoTieneRuntimePendiente(proyecto.ID, strings.TrimSpace(proyecto.Slug)); err != nil {
		return false, "", err
	} else if pending {
		return true, detail, nil
	}
	for _, estado := range []db.EstadoTarea{
		db.TareaEnProgreso,
		db.TareaAsignada,
		db.TareaBloqueada,
		db.TareaLibre,
		db.TareaBacklog,
	} {
		tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyecto.ID, Estado: &estado})
		if err != nil {
			return false, "", err
		}
		if len(tareas) > 0 {
			return true, fmt.Sprintf("proyecto=%s tareas=%s count=%d", strings.TrimSpace(proyecto.Slug), string(estado), len(tareas)), nil
		}
	}
	return false, "", nil
}

func proyectoTieneRuntimePendiente(proyectoID int64, slug string) (bool, string, error) {
	for _, estado := range []string{"pendiente", "entregado"} {
		estado := estado
		items, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ProyectoID: &proyectoID, Estado: &estado})
		if err != nil {
			return false, "", err
		}
		if len(items) > 0 {
			return true, fmt.Sprintf("proyecto=%s mailbox=%s count=%d", strings.TrimSpace(slug), estado, len(items)), nil
		}
	}
	for _, estado := range []string{"pendiente", "tomada", "ejecutando"} {
		estado := estado
		orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
		if err != nil {
			return false, "", err
		}
		if len(orders) > 0 {
			return true, fmt.Sprintf("proyecto=%s runtime_orders=%s count=%d", strings.TrimSpace(slug), estado, len(orders)), nil
		}
	}
	return false, "", nil
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
	return db.PrepararPlanificacionAutomatica()
}

func (dbAutomationService) ProcesarPipelineLocalBatch() (int, error) {
	return procesarPipelineLocalBatch()
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
	reclamados, _ := db.ProcesarHigieneRuntimesAutonomosBatch()
	total := reclamados
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
		BatchTimeout: controlPlaneBatchTimeoutEfectivo(),
	}
	if debugControlPlane && debugLogger != nil {
		runner.Debugf = debugLogger.Printf
	}
	return runner
}

func controlPlaneBatchTimeoutEfectivo() time.Duration {
	timeout := 3 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("ORQUESTA_OLLAMA_TIMEOUT_MS")); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms > 0 {
			candidato := time.Duration(ms) * time.Millisecond
			if candidato > timeout {
				timeout = candidato + 30*time.Second
			}
		}
	}
	if timeout < 45*time.Second {
		return 45 * time.Second
	}
	return timeout
}

func procesarRuntimeTranscriptBatch() (int, error) {
	ingested, err := db.IngestarRuntimeTranscriptActivos()
	if err != nil {
		return ingested, err
	}
	despertarRuntimeOrdersTrasTranscript(ingested)
	registradas, err := procesarEntregasDesdeRuntimeTranscript()
	if err != nil {
		return ingested, err
	}
	despertarRuntimeOrdersTrasTranscript(registradas)
	despertarRuntimeMailboxTrasTranscript(registradas)
	ingested += registradas
	premiumRegistradas, err := procesarEntregasGitPremiumActivas()
	if err != nil {
		return ingested, err
	}
	despertarRuntimeOrdersTrasTranscript(premiumRegistradas)
	despertarRuntimeMailboxTrasTranscript(premiumRegistradas)
	ingested += premiumRegistradas
	if !controlPlaneConfigBoolOrDefault("runtime_transcript_auto_guidance_enabled", true) {
		return ingested, nil
	}
	processedSignals, err := procesarSignalsPendientesRuntimeTranscript()
	if err != nil {
		return ingested + processedSignals, err
	}
	return ingested + processedSignals, nil
}

func procesarEntregasGitPremiumActivas() (int, error) {
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Limit: 50})
	if err != nil {
		return 0, err
	}
	vistos := map[string]struct{}{}
	procesadas := 0
	for _, order := range orders {
		if !runtimeOrderEntregaGitPremiumActiva(order) {
			continue
		}
		proyectoID := int64(0)
		if order.ProyectoID != nil && *order.ProyectoID > 0 {
			proyectoID = *order.ProyectoID
		}
		if proyectoID <= 0 {
			continue
		}
		agente := strings.TrimSpace(order.Agente)
		if agente == "" {
			continue
		}
		key := fmt.Sprintf("%s:%d", agente, proyectoID)
		if _, ok := vistos[key]; ok {
			continue
		}
		vistos[key] = struct{}{}
		proyecto, err := runtimesService.GetProject(strconv.FormatInt(proyectoID, 10))
		if err != nil || proyecto == nil {
			if err != nil {
				return procesadas, err
			}
			continue
		}
		resultado, err := runtimesService.IntentarRegistrarEntregaGitMicroprogramacionActiva(
			agente,
			order.ProyectoID,
			proyecto.Slug,
			"runtime_premium_git_reconcile",
			"orquesta",
		)
		if err != nil {
			return procesadas, err
		}
		if resultado != nil {
			procesadas++
		}
	}
	return procesadas, nil
}

func runtimeOrderEntregaGitPremiumActiva(order *db.RuntimeOrder) bool {
	if order == nil || order.ProyectoID == nil || *order.ProyectoID <= 0 {
		return false
	}
	switch strings.TrimSpace(order.Tipo) {
	case "send_instruction":
		return runtimeOrderEntregaGitPremiumDesdeSendInstruction(order)
	case "start", "resume", "handoff":
		return runtimeOrderEntregaGitPremiumDesdeBootstrapLease(order)
	default:
		return false
	}
}

func runtimeOrderEntregaGitPremiumDesdeSendInstruction(order *db.RuntimeOrder) bool {
	if order == nil {
		return false
	}
	switch strings.TrimSpace(order.Estado) {
	case "pendiente", "encolada", "ejecutando":
	default:
		return false
	}
	if strings.TrimSpace(order.PayloadJSON) == "" {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(order.PayloadJSON), &payload); err != nil {
		return false
	}
	isPipelineLocal := strings.EqualFold(strings.TrimSpace(stringMapValue(payload, "source")), "pipeline_local") ||
		strings.EqualFold(strings.TrimSpace(stringMapValue(payload, "kind")), "pipeline_local") ||
		strings.EqualFold(strings.TrimSpace(stringMapValue(payload, "mailbox_kind")), "pipeline_local")
	if !isPipelineLocal {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(stringMapValue(payload, "carril"))) {
	case "premium_worktree", "revision_diff":
		return true
	default:
		return false
	}
}

func runtimeOrderEntregaGitPremiumDesdeBootstrapLease(order *db.RuntimeOrder) bool {
	if order == nil || strings.TrimSpace(order.ResultadoJSON) == "" {
		return false
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(order.ResultadoJSON), &result); err != nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(stringMapValue(result, "receipt_source")), "git_worktree") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(stringMapValue(result, "lease_state"))) {
	case "waiting_for_evidence", "delivered", "acked":
	default:
		return false
	}
	mailboxIDs, _ := result["mailbox_ids"].([]any)
	return len(mailboxIDs) > 0
}

func despertarRuntimeOrdersTrasTranscript(cantidad int) {
	if cantidad > 0 {
		wakeRuntimeOrdersAfterTranscript()
	}
}

func despertarRuntimeMailboxTrasTranscript(cantidad int) {
	if cantidad > 0 {
		wakeRuntimeMailboxAfterTranscript()
	}
}

func procesarSignalsPendientesRuntimeTranscript() (int, error) {
	signals, err := listarSignalsPendientesRuntimeTranscript()
	if err != nil {
		return 0, err
	}
	processedSignals := 0
	for i := len(signals) - 1; i >= 0; i-- {
		processed, err := procesarSignalPendienteRuntimeTranscript(signals[i])
		if err != nil {
			return processedSignals, err
		}
		if processed {
			processedSignals++
		}
	}
	return processedSignals, nil
}

func listarSignalsPendientesRuntimeTranscript() ([]*db.RuntimeTranscriptEntry, error) {
	return db.ListarRuntimeTranscript(db.FiltroRuntimeTranscript{
		SoloSenalesPend: true,
		Limit:           50,
	})
}

func procesarSignalPendienteRuntimeTranscript(item *db.RuntimeTranscriptEntry) (bool, error) {
	if item == nil || clasificacionSignalTranscript(item) == "" {
		return false, nil
	}
	note, err := procesarSignalTranscript(item)
	if err != nil {
		return false, err
	}
	if err := marcarSignalTranscriptManejado(item, note); err != nil {
		return false, err
	}
	return true, nil
}

func marcarSignalTranscriptManejado(item *db.RuntimeTranscriptEntry, note string) error {
	if item == nil || strings.TrimSpace(note) == "" {
		return nil
	}
	return db.MarcarRuntimeTranscriptManejado(item.ID, note)
}

func procesarEntregasDesdeRuntimeTranscript() (int, error) {
	items, err := runtimesService.ListRuntimeTranscript(db.FiltroRuntimeTranscript{Limit: 50})
	if err != nil {
		return 0, err
	}
	vistos := map[string]struct{}{}
	procesadas := 0
	for _, item := range items {
		if !runtimeTranscriptEntregaElegible(item) {
			continue
		}
		agente := agenteSignalTranscript(item)
		k, ok := claveEntregaRuntimeTranscript(item, agente)
		if !ok {
			continue
		}
		if _, ok := vistos[k]; ok {
			continue
		}
		vistos[k] = struct{}{}
		procesada, err := procesarEntregaRuntimeTranscript(item, agente)
		if err != nil {
			return procesadas, err
		}
		if procesada {
			procesadas++
		}
	}
	return procesadas, nil
}

type runtimeTranscriptEntregaEscenario string

const (
	runtimeTranscriptEntregaEscenarioDescartar    runtimeTranscriptEntregaEscenario = "descartar"
	runtimeTranscriptEntregaEscenarioCodigoPatch  runtimeTranscriptEntregaEscenario = "codigo_patch"
	runtimeTranscriptEntregaEscenarioFinalizacion runtimeTranscriptEntregaEscenario = "finalizacion"
	runtimeTranscriptEntregaEscenarioConsulta     runtimeTranscriptEntregaEscenario = "consulta"
	runtimeTranscriptEntregaEscenarioPlan         runtimeTranscriptEntregaEscenario = "plan"
	runtimeTranscriptEntregaEscenarioRuido        runtimeTranscriptEntregaEscenario = "ruido"
)

type runtimeTranscriptEntregaPolitica struct {
	Escenario           runtimeTranscriptEntregaEscenario
	PermiteMaterializar bool
	PermiteRegistrarGit bool
}

func resolverPoliticaEntregaRuntimeTranscript(item *db.RuntimeTranscriptEntry) runtimeTranscriptEntregaPolitica {
	if item == nil || item.ProyectoID == nil || *item.ProyectoID <= 0 {
		return runtimeTranscriptEntregaPolitica{Escenario: runtimeTranscriptEntregaEscenarioDescartar}
	}
	switch strings.ToLower(strings.TrimSpace(item.Stream)) {
	case "pty_out", "stdout", "stderr", "assistant":
	default:
		return runtimeTranscriptEntregaPolitica{Escenario: runtimeTranscriptEntregaEscenarioDescartar}
	}
	texto := textoEntregaRuntimeTranscript(item)
	normalized := normalizedEntregaRuntimeTranscript(item)
	if runtimeTranscriptTextoEsBootstrapPoolLocal(texto, normalized) {
		return runtimeTranscriptEntregaPolitica{Escenario: runtimeTranscriptEntregaEscenarioDescartar}
	}
	classification := clasificacionEntregaRuntimeTranscript(item)
	switch classification {
	case "approval_request", "waiting_human", "blocked", "needs_replan",
		"ready_for_review", "review_approved", "review_changes_requested", "review_blocked",
		"runtime_panic", "runtime_crash", "runtime_failure_signal":
		return runtimeTranscriptEntregaPolitica{Escenario: runtimeTranscriptEntregaEscenarioConsulta}
	}
	if runtimeTranscriptTextoPareceCodigoOPatch(texto, normalized) {
		return runtimeTranscriptEntregaPolitica{
			Escenario:           runtimeTranscriptEntregaEscenarioCodigoPatch,
			PermiteMaterializar: true,
			PermiteRegistrarGit: true,
		}
	}
	if runtimeTranscriptTextoPareceFinalizacion(normalized, classification) {
		return runtimeTranscriptEntregaPolitica{
			Escenario:           runtimeTranscriptEntregaEscenarioFinalizacion,
			PermiteRegistrarGit: true,
		}
	}
	if runtimeTranscriptTextoPareceConsultaCLI(normalized, classification) {
		return runtimeTranscriptEntregaPolitica{Escenario: runtimeTranscriptEntregaEscenarioConsulta}
	}
	if runtimeTranscriptTextoParecePlanTrabajo(normalized) {
		return runtimeTranscriptEntregaPolitica{Escenario: runtimeTranscriptEntregaEscenarioPlan}
	}
	if strings.TrimSpace(normalized) == "" {
		return runtimeTranscriptEntregaPolitica{Escenario: runtimeTranscriptEntregaEscenarioRuido}
	}
	return runtimeTranscriptEntregaPolitica{Escenario: runtimeTranscriptEntregaEscenarioRuido}
}

func clasificacionEntregaRuntimeTranscript(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(item.Classification))
}

func normalizedEntregaRuntimeTranscript(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return ""
	}
	normalized := strings.ToLower(strings.TrimSpace(item.NormalizedText))
	if normalized != "" {
		return normalized
	}
	return strings.ToLower(strings.Join(strings.Fields(textoEntregaRuntimeTranscript(item)), " "))
}

func runtimeTranscriptTextoPareceCodigoOPatch(texto, normalized string) bool {
	texto = strings.TrimSpace(texto)
	normalized = strings.TrimSpace(strings.ToLower(normalized))
	if texto == "" && normalized == "" {
		return false
	}
	patchMarkers := []string{
		"diff --git",
		"*** begin patch",
		"@@",
		"// file:",
	}
	for _, marker := range patchMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	switch {
	case strings.HasPrefix(texto, "--- "), strings.HasPrefix(texto, "+++ "):
		return true
	case strings.Contains(texto, "\n--- "), strings.Contains(texto, "\n+++ "):
		return true
	}
	codeMarkers := []string{
		"```go", "```ts", "```tsx", "```js", "```jsx", "```py", "```rs", "```java",
		"package ", "func ", "type ", "struct {", "interface {", "const ", "var ",
		"import (", "if err :=", "return err", "class ", "def ", "public ", "private ",
	}
	for _, marker := range codeMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func runtimeTranscriptTextoPareceFinalizacion(normalized, classification string) bool {
	switch classification {
	case "task_completed":
		return true
	}
	markers := []string{
		"he terminado",
		"he completado",
		"acabo de terminar",
		"tarea terminada",
		"tarea completada",
		"task completed",
		"finished the task",
		"completed the task",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func runtimeTranscriptTextoPareceConsultaCLI(normalized, classification string) bool {
	switch classification {
	case "approval_request", "waiting_human", "blocked", "needs_replan":
		return true
	}
	markers := []string{
		"puedo ",
		"quieres que",
		"te parece bien si",
		"debo ",
		"can i ",
		"should i ",
		"do you want me to",
		"what should i do next",
		"qué hago ahora",
		"que hago ahora",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func runtimeTranscriptTextoParecePlanTrabajo(normalized string) bool {
	markers := []string{
		"voy a ",
		"primero ",
		"después ",
		"despues ",
		"plan:",
		"i will ",
		"next i will",
		"first i will",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func runtimeTranscriptEntregaElegible(item *db.RuntimeTranscriptEntry) bool {
	politica := resolverPoliticaEntregaRuntimeTranscript(item)
	return politica.PermiteMaterializar || politica.PermiteRegistrarGit
}

func claveEntregaRuntimeTranscript(item *db.RuntimeTranscriptEntry, agente string) (string, bool) {
	if strings.TrimSpace(agente) == "" || item == nil || item.ProyectoID == nil {
		return "", false
	}
	return fmt.Sprintf("%s:%d", strings.TrimSpace(agente), *item.ProyectoID), true
}

func procesarEntregaRuntimeTranscript(item *db.RuntimeTranscriptEntry, agente string) (bool, error) {
	politica := resolverPoliticaEntregaRuntimeTranscript(item)
	if !politica.PermiteMaterializar && !politica.PermiteRegistrarGit {
		return false, nil
	}
	proyecto, err := resolverProyectoEntregaRuntimeTranscript(item)
	if err != nil || proyecto == nil {
		return false, nil
	}
	if politica.PermiteMaterializar {
		materializada, err := materializarEntregaRuntimeTranscript(item, agente, proyecto)
		if err != nil {
			return false, err
		}
		if materializada {
			return true, nil
		}
	}
	if !politica.PermiteRegistrarGit {
		return false, nil
	}
	return registrarEntregaGitRuntimeTranscript(item, agente, proyecto)
}

func resolverProyectoEntregaRuntimeTranscript(item *db.RuntimeTranscriptEntry) (*db.Proyecto, error) {
	return runtimesService.GetProject(strconv.FormatInt(*item.ProyectoID, 10))
}

func textoEntregaRuntimeTranscript(item *db.RuntimeTranscriptEntry) string {
	return textoSignalTranscript(item)
}

func etiquetaEntregaRuntimeTranscript(item *db.RuntimeTranscriptEntry) string {
	return etiquetaSignalTranscript(item)
}

func materializarEntregaRuntimeTranscript(item *db.RuntimeTranscriptEntry, agente string, proyecto *db.Proyecto) (bool, error) {
	resultadoMaterializado, err := runtimesService.MaterializarEntregaMicroprogramacionActivaDesdeRespuesta(
		agente,
		item.ProyectoID,
		proyecto.Slug,
		textoEntregaRuntimeTranscript(item),
		etiquetaEntregaRuntimeTranscript(item),
		"runtime_transcript_materializada",
	)
	if err != nil {
		return resolverFalloMaterializacionRuntimeTranscript(item, agente, err)
	}
	return resultadoMaterializado != nil, nil
}

func resolverFalloMaterializacionRuntimeTranscript(item *db.RuntimeTranscriptEntry, agente string, causa error) (bool, error) {
	if ordenCorreccionID, correccionErr := reencolarCorreccionMicroprogramacionDesdeTranscript(agente, item.ProyectoID, textoEntregaRuntimeTranscript(item), causa); correccionErr != nil {
		return false, correccionErr
	} else if ordenCorreccionID > 0 {
		return true, nil
	}
	return false, causa
}

func registrarEntregaGitRuntimeTranscript(item *db.RuntimeTranscriptEntry, agente string, proyecto *db.Proyecto) (bool, error) {
	resultado, err := runtimesService.IntentarRegistrarEntregaGitMicroprogramacionActiva(
		agente,
		item.ProyectoID,
		slugProyectoEntregaRuntimeTranscript(proyecto),
		etiquetaEntregaRuntimeTranscript(item),
		"orquesta",
	)
	if err != nil {
		return false, err
	}
	return resultado != nil, nil
}

func slugProyectoEntregaRuntimeTranscript(proyecto *db.Proyecto) string {
	return slugProyectoAutonomia(proyecto)
}

func reencolarCorreccionMicroprogramacionDesdeTranscript(agente string, proyectoID *int64, respuesta string, causa error) (int64, error) {
	if causa == nil {
		return 0, nil
	}
	ctx, err := resolverContextoCorreccionMicroprogramacionDesdeTranscript(agente, proyectoID, respuesta)
	if err != nil || ctx == nil || ctx.RuntimeOrderID <= 0 {
		return 0, err
	}
	return runtimesService.ReencolarCorreccionEntregaMicroprogramacion(ctx.RuntimeOrderID, causa.Error())
}

func resolverContextoCorreccionMicroprogramacionDesdeTranscript(agente string, proyectoID *int64, respuesta string) (*runtimesapp.ContextoEntregaMicroprogramacion, error) {
	return runtimesService.ResolverContextoEntregaMicroprogramacionParaRespuesta(strings.TrimSpace(agente), proyectoID, strings.TrimSpace(respuesta))
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
		if !runtimeHandleCuentaComoActivoEnMailboxBatch(fresh) {
			return nil, nil
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
	if !runtimeHandleCuentaComoActivoEnMailboxBatch(handle) {
		handle = nil
	}
	s.activeHandleLoaded[key] = struct{}{}
	s.activeHandles[key] = handle
	return handle, nil
}

func runtimeHandleCuentaComoActivoEnMailboxBatch(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(handle.Estado), "activo")
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
	return procesarRuntimeMailboxBatchConFiltro(db.FiltroRuntimeMailbox{Estado: &estado})
}

func procesarRuntimeMailboxBatchConFiltro(filter db.FiltroRuntimeMailbox) (int, error) {
	mailbox, err := runtimesService.ListRuntimeMailbox(filter)
	if err != nil {
		return 0, err
	}
	consumed := map[int64]struct{}{}
	snapshot := newRuntimeMailboxBatchSnapshot()
	quotaPipelineReconciled, err := reconciliarRuntimeMailboxPipelineDuplicadoEnCuotaBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled, err
	}
	reconciled, err := reconciliarRuntimeMailboxAgenteSinVidaBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled, err
	}
	pipelineCooldownReconciled, err := reconciliarRuntimeMailboxPipelineEnfriamientoBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + pipelineCooldownReconciled, err
	}
	reconciled += pipelineCooldownReconciled
	refreshCooldownReconciled, err := reconciliarRuntimeMailboxRefreshEnfriamientoBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + refreshCooldownReconciled, err
	}
	reconciled += refreshCooldownReconciled
	instructionCooldownReconciled, err := reconciliarRuntimeMailboxInstructionEnfriamientoBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + instructionCooldownReconciled, err
	}
	reconciled += instructionCooldownReconciled
	watchdogReconciled, err := reconciliarRuntimeMailboxWatchdogSinHandleBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + watchdogReconciled, err
	}
	reconciled += watchdogReconciled
	interactive, err := procesarRuntimeMailboxInteractivoBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + interactive, err
	}
	sessionResume, err := procesarRuntimeMailboxSessionResumeBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + interactive + sessionResume, err
	}
	bootstrapTMUX, err := procesarRuntimeMailboxBootstrapTMUXBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + interactive + sessionResume + bootstrapTMUX, err
	}
	bootstrapPipelineObserved, err := reconciliarRuntimeMailboxPipelineBootstrapActivaBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + interactive + sessionResume + bootstrapTMUX + bootstrapPipelineObserved, err
	}
	guidanceInbox, err := reconciliarRuntimeMailboxGuidanceDurableEnInboxBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + interactive + sessionResume + bootstrapTMUX + bootstrapPipelineObserved + guidanceInbox, err
	}
	restarts, err := procesarRuntimeMailboxCoordinatedRestartBatchConMailbox(mailbox, consumed, snapshot)
	if err != nil {
		return quotaPipelineReconciled + reconciled + interactive + sessionResume + bootstrapTMUX + bootstrapPipelineObserved + guidanceInbox + restarts, err
	}
	return quotaPipelineReconciled + reconciled + interactive + sessionResume + bootstrapTMUX + bootstrapPipelineObserved + guidanceInbox + restarts, nil
}

func reconciliarRuntimeMailboxPipelineDuplicadoEnCuotaBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	type clave struct {
		agente   string
		proyecto int64
	}
	newestByKey := map[clave]int64{}
	for _, msg := range mailbox {
		if msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Kind), "pipeline_local") {
			continue
		}
		agente := strings.TrimSpace(msg.ToAgente)
		if agente == "" {
			continue
		}
		proyectoID := int64(0)
		if msg.ProyectoID != nil && *msg.ProyectoID > 0 {
			proyectoID = *msg.ProyectoID
		}
		k := clave{agente: agente, proyecto: proyectoID}
		if newestByKey[k] == 0 || msg.ID > newestByKey[k] {
			newestByKey[k] = msg.ID
		}
	}
	total := 0
	for _, msg := range mailbox {
		if msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Kind), "pipeline_local") {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		agente := strings.TrimSpace(msg.ToAgente)
		if agente == "" {
			continue
		}
		compactar, err := runtimeMailboxPipelineDebeCompactarseEnBatch(agente, msg.ProyectoID, snapshot)
		if err != nil {
			return total, err
		}
		if !compactar {
			continue
		}
		proyectoID := int64(0)
		if msg.ProyectoID != nil && *msg.ProyectoID > 0 {
			proyectoID = *msg.ProyectoID
		}
		k := clave{agente: agente, proyecto: proyectoID}
		if newestByKey[k] == msg.ID {
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
		db.Audit("orquesta", "runtime_mailbox_pipeline_en_cuota_dedupe", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s proyecto_id=%s pipeline_local duplicado consumido por cuota", agente, runtimeMailboxProyectoDetalle(msg.ProyectoID)))
		consumed[msg.ID] = struct{}{}
		total++
	}
	return total, nil
}

func runtimeMailboxPipelineDebeCompactarseEnBatch(agente string, proyectoID *int64, snapshot *runtimeMailboxBatchSnapshot) (bool, error) {
	if snapshot == nil {
		return false, nil
	}
	ok, err := snapshot.quotaMatches(agente, "enfriamiento")
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	ok, err = snapshot.quotaMatches(agente, "agotado")
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	handle, err := snapshot.activeHandle(agente, proyectoID)
	if err != nil {
		return false, err
	}
	if runtimeHandleEstadoEsPausado(handle) {
		return true, nil
	}
	return false, nil
}

func reconciliarRuntimeMailboxPipelineEnfriamientoBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	for _, msg := range mailbox {
		if msg == nil || !runtimeMailboxPipelinePuedeConsumirsePorEnfriamiento(msg, snapshot) {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		dispatched, err := reconciliarRuntimeMailboxPipelineEnfriamientoMensaje(msg, consumed)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// reconciliarRuntimeMailboxPipelineEnfriamientoMensaje consume un pipeline_local ya
// cualificado como consumible por enfriamiento. Retorna true si fue consumido.
func reconciliarRuntimeMailboxPipelineEnfriamientoMensaje(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}) (bool, error) {
	if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
		return false, err
	} else if !canConsume {
		return false, nil
	}
	if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_pipeline_enfriamiento", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s kind=%s pipeline_local consumido por agente en enfriamiento",
			strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
	consumed[msg.ID] = struct{}{}
	return true, nil
}

func runtimeMailboxPipelinePuedeConsumirsePorEnfriamiento(msg *db.RuntimeMailboxMessage, snapshot *runtimeMailboxBatchSnapshot) bool {
	if msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Kind), "pipeline_local") {
		return false
	}
	ok, err := snapshot.quotaMatches(strings.TrimSpace(msg.ToAgente), "enfriamiento")
	if err != nil {
		return false
	}
	if ok {
		return true
	}
	ok, err = snapshot.quotaMatches(strings.TrimSpace(msg.ToAgente), "agotado")
	if err != nil {
		return false
	}
	return ok
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
		dispatched, err := reconciliarRuntimeMailboxRefreshEnfriamientoMensaje(msg, consumed)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// reconciliarRuntimeMailboxRefreshEnfriamientoMensaje consume un refresh (governance/skills)
// ya cualificado como consumible por enfriamiento. Retorna true si fue consumido.
func reconciliarRuntimeMailboxRefreshEnfriamientoMensaje(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}) (bool, error) {
	if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
		return false, err
	} else if !canConsume {
		return false, nil
	}
	if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_refresh_enfriamiento", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s kind=%s refresh consumido por agente en enfriamiento",
			strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
	consumed[msg.ID] = struct{}{}
	return true, nil
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
		dispatched, err := reconciliarRuntimeMailboxAgenteSinVidaMensaje(msg, consumed, snapshot)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// reconciliarRuntimeMailboxAgenteSinVidaMensaje consume un mensaje cuyo agente destino
// no está en la flota activa. Retorna true si el mensaje fue consumido.
func reconciliarRuntimeMailboxAgenteSinVidaMensaje(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (bool, error) {
	debeConsumirse, detalle, err := runtimeMailboxDebeConsumirsePorAgenteSinVida(msg, snapshot)
	if err != nil {
		return false, err
	}
	if !debeConsumirse {
		return false, nil
	}
	if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
		return false, err
	} else if !canConsume {
		return false, nil
	}
	if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_agente_sin_vida", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s kind=%s %s", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), detalle))
	consumed[msg.ID] = struct{}{}
	return true, nil
}

func reconciliarRuntimeMailboxInstructionEnfriamientoBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	for _, msg := range mailbox {
		if msg == nil || !runtimeMailboxInstructionPuedeConsumirsePorEnfriamiento(msg, snapshot) {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		dispatched, err := reconciliarRuntimeMailboxInstructionEnfriamientoMensaje(msg, consumed)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// reconciliarRuntimeMailboxInstructionEnfriamientoMensaje consume una instrucción
// ya cualificada como consumible por enfriamiento. Retorna true si fue consumida.
func reconciliarRuntimeMailboxInstructionEnfriamientoMensaje(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}) (bool, error) {
	if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
		return false, err
	} else if !canConsume {
		return false, nil
	}
	if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_instruction_enfriamiento", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s kind=%s instruction consumida por agente en enfriamiento",
			strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
	consumed[msg.ID] = struct{}{}
	return true, nil
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
		dispatched, err := reconciliarRuntimeMailboxWatchdogSinHandleMensaje(msg, consumed, snapshot, handle)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// reconciliarRuntimeMailboxWatchdogSinHandleMensaje consume un mensaje watchdog
// cuando el agente está en enfriamiento/agotado/pausado o no tiene handle activo.
// Retorna true si el mensaje fue consumido.
func reconciliarRuntimeMailboxWatchdogSinHandleMensaje(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot, handle *db.RuntimeHandle) (bool, error) {
	if watchdogPuedeConsumirsePorEnfriamiento(msg, handle, snapshot) {
		if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
			return false, err
		} else if !canConsume {
			return false, nil
		}
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return false, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return false, err
		}
		db.Audit("orquesta", "runtime_mailbox_watchdog_enfriamiento", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s proyecto_id=%s watchdog consumido por agente en enfriamiento",
				strings.TrimSpace(msg.ToAgente), runtimeMailboxProyectoDetalle(msg.ProyectoID)))
		consumed[msg.ID] = struct{}{}
		return true, nil
	}
	if handle != nil {
		return false, nil
	}
	if canConsume, err := runtimeMailboxPuedeConsumirseFueraDeOrden(msg); err != nil {
		return false, err
	} else if !canConsume {
		return false, nil
	}
	if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_watchdog_sin_handle", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s proyecto_id=%s watchdog consumido por ausencia de handle activo",
			strings.TrimSpace(msg.ToAgente), runtimeMailboxProyectoDetalle(msg.ProyectoID)))
	consumed[msg.ID] = struct{}{}
	return true, nil
}

func watchdogPuedeConsumirsePorEnfriamiento(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, snapshot *runtimeMailboxBatchSnapshot) bool {
	if msg == nil || snapshot == nil {
		return false
	}
	ok, err := snapshot.quotaMatches(strings.TrimSpace(msg.ToAgente), "enfriamiento")
	if err != nil {
		return false
	}
	if ok {
		return true
	}
	ok, err = snapshot.quotaMatches(strings.TrimSpace(msg.ToAgente), "agotado")
	if err != nil {
		return false
	}
	if ok {
		return true
	}
	if handle == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado")
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

func runtimeMailboxInstructionPuedeConsumirsePorEnfriamiento(msg *db.RuntimeMailboxMessage, snapshot *runtimeMailboxBatchSnapshot) bool {
	if msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Kind), "instruction") {
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
	if idSignalTranscript(item) > 0 {
		motivo = fmt.Sprintf("Auto-pausa por runtime_panic: transcript=%d", idSignalTranscript(item))
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
	agente, handle, note, handled, err := prepararDecisionSignalTranscript(item)
	if err != nil {
		return "", err
	}
	if handled {
		return note, nil
	}
	if esSignalFalloRuntime(item) {
		return procesarSignalFalloRuntime(item, agente)
	}
	return procesarSignalAutoGuidance(item, agente, handle)
}

func prepararDecisionSignalTranscript(item *db.RuntimeTranscriptEntry) (string, *db.RuntimeHandle, string, bool, error) {
	if signalTranscriptDebeIgnorarseAutoGuia(item) {
		return "", nil, "signal_bootstrap_pool_local_ignorado", true, nil
	}
	return prepararSignalTranscript(item)
}

func prepararSignalTranscript(item *db.RuntimeTranscriptEntry) (string, *db.RuntimeHandle, string, bool, error) {
	agente := agenteSignalTranscript(item)
	if agente == "" {
		return "", nil, "signal_sin_agente", true, nil
	}
	handle, err := resolverHandleEntregaTranscript(item)
	if err != nil {
		return "", nil, "", false, err
	}
	if handle == nil {
		return "", nil, "sin_runtime_activo", true, nil
	}
	if note, handled, err := resolverReviewGateDesdeSignalTranscript(item); err != nil {
		return "", nil, "", false, err
	} else if handled {
		return "", nil, note, true, nil
	}
	if note, handled, err := resolverCompletarTareaDesdeSignalTranscript(item, handle); err != nil {
		return "", nil, "", false, err
	} else if handled {
		return "", nil, note, true, nil
	}
	return agente, handle, "", false, nil
}

func procesarSignalFalloRuntime(item *db.RuntimeTranscriptEntry, agente string) (string, error) {
	notes := make([]string, 0, 3)
	if cooldownNote, err := resolverCooldownRuntimePanicSignal(agente, item); err != nil {
		return "", err
	} else {
		notes = acumularNotaAutonomia(notes, cooldownNote)
	}
	if supervisorNote, err := resolverSupervisionFalloRuntimeSignal(item); err != nil {
		return "", err
	} else {
		notes = acumularNotaAutonomia(notes, supervisorNote)
	}
	notes = acumularNotaAutonomia(notes, "runtime_failure_signal")
	return resumenNotasAutonomia(notes), nil
}

func resolverCooldownRuntimePanicSignal(agente string, item *db.RuntimeTranscriptEntry) (string, error) {
	return enfriarAgentePorRuntimePanic(agente, item)
}

func resolverSupervisionFalloRuntimeSignal(item *db.RuntimeTranscriptEntry) (string, error) {
	return notificarSupervisorSignalTranscript(item)
}

func procesarSignalAutoGuidance(item *db.RuntimeTranscriptEntry, agente string, handle *db.RuntimeHandle) (string, error) {
	payload := construirPayloadAutoGuidanceSignalTranscript(agente, item)
	orderID, orderType, err := encolarAutoGuidanceSignalTranscript(item, agente, handle, payload)
	if err != nil {
		return "", err
	}
	registrarAutoGuidanceSignalTranscript(orderID, orderType, agente, item, payload)
	return completarAutoGuidanceSignalTranscript(item, orderID)
}

func encolarAutoGuidanceSignalTranscript(item *db.RuntimeTranscriptEntry, agente string, handle *db.RuntimeHandle, payload map[string]any) (int64, string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
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
		return 0, "", err
	}
	return orderID, order.Tipo, nil
}

func registrarAutoGuidanceSignalTranscript(orderID int64, orderType, agente string, item *db.RuntimeTranscriptEntry, payload map[string]any) {
	db.Audit("orquesta", "runtime_transcript_signal", "runtime_order", orderID, detalleAutoGuidanceSignalTranscript(item, agente))
	emitirAutoGuidanceSignalTranscript(orderID, orderType, agente, item, payload)
}

func completarAutoGuidanceSignalTranscript(item *db.RuntimeTranscriptEntry, orderID int64) (string, error) {
	notes := []string{fmt.Sprintf("auto_guidance_order:%d", orderID)}
	if supervisorNote, err := notificarSupervisorSignalTranscript(item); err != nil {
		return "", err
	} else {
		notes = acumularNotaAutonomia(notes, supervisorNote)
	}
	return resumenNotasAutonomia(notes), nil
}

func signalTranscriptDebeIgnorarseAutoGuia(item *db.RuntimeTranscriptEntry) bool {
	if item == nil {
		return false
	}
	if !strings.EqualFold(clasificacionSignalTranscript(item), "waiting_human") {
		return false
	}
	return runtimeTranscriptTextoEsBootstrapPoolLocal(item.Text, item.NormalizedText)
}

func acumularNotaAutonomia(notes []string, note string) []string {
	note = strings.TrimSpace(note)
	if note == "" {
		return notes
	}
	return append(notes, note)
}

func resumenNotasAutonomia(notes []string) string {
	if len(notes) == 0 {
		return ""
	}
	return strings.Join(notes, ";")
}

func agregarNotaAutonomia(notes []string, note string) []string {
	return acumularNotaAutonomia(notes, note)
}

func construirPayloadAutoGuidanceSignalTranscript(agente string, item *db.RuntimeTranscriptEntry) map[string]any {
	return map[string]any{
		"to_agente":      strings.TrimSpace(agente),
		"from_agente":    "orquesta",
		"texto":          construirRespuestaSignalTranscript(item),
		"classification": clasificacionSignalTranscript(item),
		"transcript_id":  idSignalTranscript(item),
		"kind":           "instruction",
	}
}

func detalleAutoGuidanceSignalTranscript(item *db.RuntimeTranscriptEntry, agente string) string {
	return fmt.Sprintf("transcript_id=%d agente=%s signal=%s", idSignalTranscript(item), strings.TrimSpace(agente), clasificacionSignalTranscript(item))
}

func mensajeEventoAutoGuidanceSignalTranscript(agente string, item *db.RuntimeTranscriptEntry) string {
	return fmt.Sprintf("Orquesta envió guía automática a %s tras %s", strings.TrimSpace(agente), clasificacionSignalTranscript(item))
}

func emitirAutoGuidanceSignalTranscript(orderID int64, orderType, agente string, item *db.RuntimeTranscriptEntry, instruction map[string]any) {
	eventPayload := construirPayloadEventoAutoGuidance(item, orderID, orderType, instruction)
	payloadEvent, _ := json.Marshal(eventPayload)
	_, _ = db.RegistrarRuntimeEvent(&db.RuntimeEvent{
		RuntimeID:   item.RuntimeID,
		Kind:        "auto_guidance_sent",
		Level:       "info",
		Message:     mensajeEventoAutoGuidanceSignalTranscript(agente, item),
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
}

func runtimeTranscriptTextoEsBootstrapPoolLocal(texto, normalizado string) bool {
	texto = strings.ToLower(strings.TrimSpace(texto))
	if texto == "" {
		texto = strings.ToLower(strings.TrimSpace(normalizado))
	}
	if texto == "" {
		return false
	}
	if strings.Contains(texto, "bootstrap") {
		tieneEstado := strings.Contains(texto, "estado de sesión") ||
			strings.Contains(texto, "estado de sesion") ||
			strings.Contains(texto, "estado de la sesión") ||
			strings.Contains(texto, "estado de la sesion") ||
			strings.Contains(texto, "estado operativo") ||
			strings.Contains(texto, "estado de tareas") ||
			strings.Contains(texto, "estado del agente") ||
			strings.Contains(texto, "estado del runtime")
		tieneStandby := strings.Contains(texto, "en modo standby") ||
			strings.Contains(texto, "modo standby") ||
			strings.Contains(texto, "quedo a la espera") ||
			strings.Contains(texto, "quedo a la espera de") ||
			strings.Contains(texto, "sin tareas asignadas actualmente") ||
			strings.Contains(texto, "no se detectan tareas activas") ||
			strings.Contains(texto, "sesión: confirmada como activa") ||
			strings.Contains(texto, "sesion: confirmada como activa")
		if tieneEstado && tieneStandby {
			return true
		}
	}
	if strings.Contains(texto, "bootstrap de orquesta completado") {
		return strings.Contains(texto, "estado de sesión") || strings.Contains(texto, "estado de sesion")
	}
	if strings.Contains(texto, "bootstrap de sesión") {
		return strings.Contains(texto, "estado del agente") || strings.Contains(texto, "estado del runtime")
	}
	if strings.Contains(texto, "bootstrap aceptado") {
		if strings.Contains(texto, "inicializado y operativo") {
			return strings.Contains(texto, "estado de la sesión") || strings.Contains(texto, "estado de la sesion")
		}
	}
	if strings.Contains(texto, "bootstrap de gemmaapp") || strings.Contains(texto, "bootstrap completado para") {
		tieneEstadoSesion := strings.Contains(texto, "estado de la sesión") ||
			strings.Contains(texto, "estado de la sesion") ||
			strings.Contains(texto, "estado operativo") ||
			strings.Contains(texto, "estado de tareas")
		if !tieneEstadoSesion && !strings.Contains(texto, "estado del agente") && !strings.Contains(texto, "estado del runtime") {
			return false
		}
		return strings.Contains(texto, "sin tareas asignadas actualmente") ||
			strings.Contains(texto, "no se detectan tareas activas") ||
			strings.Contains(texto, "me encuentro en estado **idle**") ||
			strings.Contains(texto, "sesión: confirmada como activa") ||
			strings.Contains(texto, "sesion: confirmada como activa") ||
			strings.Contains(texto, "en modo standby") ||
			strings.Contains(texto, "quedo a la espera de") ||
			strings.Contains(texto, "quedo a la espera") ||
			strings.Contains(texto, "quedo a la espera de instrucciones")
	}
	return false
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
		"agente":             agenteSignalTranscript(item),
		"classification":     clasificacionSignalTranscript(item),
		"runtime_id":         item.RuntimeID,
		"runtime_order_id":   orderID,
		"runtime_order_type": strings.TrimSpace(orderType),
		"signal_text":        textoSignalTranscript(item),
		"transcript_id":      idSignalTranscript(item),
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
	agente := agenteSignalTranscript(item)
	classification := clasificacionSignalTranscript(item)
	if agente == "" {
		agente = "agente-desconocido"
	}
	texto := textoSignalTranscript(item)
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
	estado, ok := estadoReviewGateDesdeSignalTranscript(item)
	if !ok {
		return "", false, nil
	}
	gate, err := resolverGateAbiertaSignalTranscript(item)
	if err != nil || gate == nil {
		return "", false, err
	}
	findingsJSON, err := prepararFindingsReviewSignalTranscript(item)
	if err != nil {
		return "", false, err
	}
	note, err := resolverEstadoGateSignalTranscript(gate, estado, item, findingsJSON)
	if err != nil {
		return "", false, err
	}
	return note, true, nil
}

func resolverCompletarTareaDesdeSignalTranscript(item *db.RuntimeTranscriptEntry, handle *db.RuntimeHandle) (string, bool, error) {
	if item == nil || handle == nil || item.Classification != "task_completed" {
		return "", false, nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	tareaID := int64PtrFromMap(meta, "tarea_id")
	if tareaID == nil || *tareaID <= 0 {
		id, _ := db.GetTareaActivaIDPorAgenteProyecto(item.Agente, item.ProyectoID)
		if id > 0 {
			tareaID = &id
		}
	}
	if tareaID == nil || *tareaID <= 0 {
		return "task_completed_ignorado:sin_tarea_id", true, nil
	}
	// Verificar si la tarea ya está completada para evitar errores por señales duplicadas
	tareaActual, err := db.GetTarea(*tareaID)
	if err != nil {
		return "", false, err
	}
	if tareaActual != nil && (tareaActual.Estado != db.TareaEnProgreso && tareaActual.Estado != db.TareaAsignada) {
		return "task_completed_ignorado:ya_completada", true, nil
	}

	if err := db.CompletarTarea(*tareaID, item.Agente, "autonomo:signal_transcript"); err != nil {
		return "", false, err
	}
	db.Audit("server", "task_completed_autonomo", "tarea", *tareaID, "agente: "+item.Agente)
	return "task_completed_autonomo", true, nil
}

func int64PtrFromMap(m map[string]any, key string) *int64 {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch val := v.(type) {
	case int:
		i := int64(val)
		return &i
	case int64:
		return &val
	case float64:
		i := int64(val)
		return &i
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		if err == nil && i != 0 {
			return &i
		}
	}
	return nil
}

func prepararFindingsReviewSignalTranscript(item *db.RuntimeTranscriptEntry) (string, error) {
	return construirFindingsReviewTranscript(item)
}

func resolverGateAbiertaSignalTranscript(item *db.RuntimeTranscriptEntry) (*reviewapp.Gate, error) {
	proyectoRef, err := proyectoRefSignalTranscript(item)
	if err != nil || strings.TrimSpace(proyectoRef) == "" {
		return nil, err
	}
	gates, err := reviewService.List(construirReviewListInputSignalTranscript(item, proyectoRef))
	if err != nil {
		return nil, err
	}
	return firstOpenGate(gates), nil
}

func proyectoRefSignalTranscript(item *db.RuntimeTranscriptEntry) (string, error) {
	proyecto, err := proyectoIfExists(strconv.FormatInt(*item.ProyectoID, 10))
	if err != nil || proyecto == nil {
		return "", err
	}
	return slugProyectoAutonomia(proyecto), nil
}

func construirReviewListInputSignalTranscript(item *db.RuntimeTranscriptEntry, proyectoRef string) reviewapp.ListInput {
	return reviewapp.ListInput{
		ProyectoRef:    strings.TrimSpace(proyectoRef),
		ReviewerAgente: reviewerAgenteSignalTranscript(item),
		Limit:          20,
	}
}

func estadoReviewGateDesdeSignalTranscript(item *db.RuntimeTranscriptEntry) (string, bool) {
	switch clasificacionSignalTranscript(item) {
	case "review_approved":
		return reviewapp.GateStateApproved, true
	case "review_changes_requested":
		return reviewapp.GateStateChangesAsked, true
	case "review_blocked":
		return reviewapp.GateStateBlocked, true
	default:
		return "", false
	}
}

func resolverEstadoGateSignalTranscript(gate *reviewapp.Gate, estado string, item *db.RuntimeTranscriptEntry, findingsJSON string) (string, error) {
	resolved, err := reviewService.Resolve(construirResolveGateInputSignalTranscript(gate, estado, item, findingsJSON))
	if err != nil {
		return "", err
	}
	detail := auditarResolucionGateSignalTranscript(resolved.ID, item, estado)
	return "review_gate_resolved:" + detail, nil
}

func construirResolveGateInputSignalTranscript(gate *reviewapp.Gate, estado string, item *db.RuntimeTranscriptEntry, findingsJSON string) reviewapp.ResolveGateInput {
	return reviewapp.ResolveGateInput{
		ID:             gate.ID,
		Estado:         estado,
		ReviewerAgente: reviewerAgenteSignalTranscript(item),
		FindingsJSON:   findingsJSON,
	}
}

func reviewerAgenteSignalTranscript(item *db.RuntimeTranscriptEntry) string {
	return agenteSignalTranscript(item)
}

func detalleResolucionGateSignalTranscript(gateID int64, item *db.RuntimeTranscriptEntry, estado string) string {
	return fmt.Sprintf("gate=%d transcript=%d estado=%s", gateID, idSignalTranscript(item), strings.TrimSpace(estado))
}

func auditarResolucionGateSignalTranscript(gateID int64, item *db.RuntimeTranscriptEntry, estado string) string {
	detail := detalleResolucionGateSignalTranscript(gateID, item, estado)
	db.Audit("orquesta", "runtime_transcript_review_resolution", "review_gate", gateID, detail)
	return detail
}

func construirFindingsReviewTranscript(item *db.RuntimeTranscriptEntry) (string, error) {
	payload := construirPayloadFindingsReviewSignalTranscript(item)
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func construirPayloadFindingsReviewSignalTranscript(item *db.RuntimeTranscriptEntry) []map[string]any {
	return []map[string]any{{
		"source":         "runtime_transcript",
		"transcript_id":  idSignalTranscript(item),
		"classification": clasificacionSignalTranscript(item),
		"agente":         agenteSignalTranscript(item),
		"text":           textoSignalTranscript(item),
		"runtime_id":     item.RuntimeID,
		"handle_id":      item.HandleID,
	}}
}

func esSignalFalloRuntime(item *db.RuntimeTranscriptEntry) bool {
	if item == nil {
		return false
	}
	switch clasificacionSignalTranscript(item) {
	case "runtime_panic", "runtime_crash":
		return true
	default:
		return false
	}
}

func construirRespuestaSignalTranscript(item *db.RuntimeTranscriptEntry) string {
	base := "Orquesta: continua de forma autonoma dentro de la gobernanza efectiva del proyecto. Usa API/servicios para el flujo normal, no BD local salvo diagnostico o recuperacion, y mantén tests coherentes y eficientes. Si el cambio es delicado, crea checkpoint y sigue."
	normalized := normalizarSignalTranscriptTexto(item)
	clasificacion := clasificacionSignalTranscript(item)
	politicaPermisos := construirPoliticaPermisosSignalTranscript(clasificacion, normalized)
	coletilla := coletillaRespuestaSignalTranscript(clasificacion, politicaPermisos)
	if strings.TrimSpace(coletilla) == "" {
		return base
	}
	return base + " " + strings.TrimSpace(coletilla)
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

func clasificacionSignalTranscript(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return ""
	}
	return strings.TrimSpace(item.Classification)
}

func agenteSignalTranscript(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return ""
	}
	return strings.TrimSpace(item.Agente)
}

func textoSignalTranscript(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return ""
	}
	return strings.TrimSpace(item.Text)
}

func nombreAgenteAutonomia(agente *db.Agente) string {
	if agente == nil {
		return ""
	}
	return strings.TrimSpace(agente.Nombre)
}

func slugProyectoAutonomia(proyecto *db.Proyecto) string {
	if proyecto == nil {
		return ""
	}
	return strings.TrimSpace(proyecto.Slug)
}

func runtimeTranscriptTieneProyecto(item *db.RuntimeTranscriptEntry) bool {
	return item != nil && item.ProyectoID != nil && *item.ProyectoID > 0
}

func contextoSignalTranscriptActivo(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto) bool {
	return item != nil && policy != nil && proyecto != nil && policy.Enabled
}

func resolverContextoSignalTranscript(item *db.RuntimeTranscriptEntry) (*db.ProyectoAutonomia, *db.Proyecto, error) {
	if !runtimeTranscriptTieneProyecto(item) {
		return nil, nil, nil
	}
	policy, err := db.GetProyectoAutonomia(*item.ProyectoID)
	if err != nil || policy == nil || !policy.Enabled {
		return policy, nil, err
	}
	proyecto, err := proyectoIfExists(strconv.FormatInt(*item.ProyectoID, 10))
	if err != nil || proyecto == nil {
		return policy, proyecto, err
	}
	return policy, proyecto, nil
}

func objetivoGeneralAutonomia(policy *db.ProyectoAutonomia) string {
	if policy == nil {
		return ""
	}
	return strings.TrimSpace(policy.ObjetivoGeneral)
}

func definitionOfDoneAutonomia(policy *db.ProyectoAutonomia) string {
	if policy == nil {
		return ""
	}
	dod := strings.TrimSpace(policy.DefinitionOfDoneJSON)
	if dod == "{}" {
		return ""
	}
	return dod
}

func idSignalTranscript(item *db.RuntimeTranscriptEntry) int64 {
	if item == nil {
		return 0
	}
	return item.ID
}

func etiquetaSignalTranscript(item *db.RuntimeTranscriptEntry) string {
	return fmt.Sprintf("runtime_transcript_id=%d", idSignalTranscript(item))
}

func coletillaRespuestaSignalTranscript(clasificacion, politicaPermisos string) string {
	switch strings.TrimSpace(clasificacion) {
	case "approval_request":
		return strings.TrimSpace(politicaPermisos) + " Si dudas entre varias opciones seguras, elige la mas alineada con el proyecto y continua sin detenerte."
	case "waiting_human":
		return strings.TrimSpace(politicaPermisos) + " No te quedes esperando respuesta: formula el siguiente paso razonable, ejecuta y documenta los supuestos."
	case "ready_for_review":
		return "Deja un resumen breve, asegurate de que el frente queda verificable y sigue disponible para que Orquesta relance review si procede."
	case "needs_replan":
		return "Si no ves el siguiente paso, revisa backlog, tareas, propuestas y ultimo checkpoint, y continua por el frente mas util disponible."
	case "blocked":
		return "Si el bloqueo es de contexto, reevalua tareas, propuestas y estado del proyecto y avanza por el mejor siguiente paso disponible."
	default:
		return ""
	}
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
	policy, proyecto, err := resolverContextoSignalTranscript(item)
	if err != nil || !contextoSignalTranscriptActivo(item, policy, proyecto) {
		return "", err
	}
	if note, handled, err := intentarDerivarReviewSignalTranscript(item, policy, proyecto); err != nil {
		return "", err
	} else if handled {
		return note, nil
	}
	return ejecutarPlanSupervisorSignalTranscript(item, policy, proyecto)
}

func intentarDerivarReviewSignalTranscript(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto) (string, bool, error) {
	if clasificacionSignalTranscript(item) != "ready_for_review" {
		return "", false, nil
	}
	note, err := notificarReviewerSignalTranscript(item, policy, proyecto)
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(note) == "" {
		return "", false, nil
	}
	return note, true, nil
}

func ejecutarPlanSupervisorSignalTranscript(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto) (string, error) {
	notes := make([]string, 0, 3)
	supervisor, activado, err := resolverAgenteSupervisorSignalTranscript(proyecto.ID, policy, agenteSignalTranscript(item))
	if err != nil {
		return "", err
	}
	if note, err := asegurarTareaReplanAutonomiaSignal(item, policy, proyecto, supervisor); err != nil {
		return "", err
	} else {
		notes = agregarNotaAutonomia(notes, note)
	}
	if note, err := resolverEstadoAgenteObjetivoSignalTranscript(supervisor, activado, agenteSignalTranscript(item), proyecto, "inspeccionar_transcript_signal", construirInstruccionSupervisionSignalTranscript(policy, proyecto, nombreAgenteAutonomia(supervisor), item), item, "supervisor", "supervisor_start"); err != nil {
		return "", err
	} else {
		notes = agregarNotaAutonomia(notes, note)
	}
	return resumenNotasAutonomia(notes), nil
}

func resolverAgenteSupervisorSignalTranscript(proyectoID int64, policy *db.ProyectoAutonomia, agenteOrigen string) (*db.Agente, bool, error) {
	if policy == nil {
		return nil, false, nil
	}
	return resolverSupervisorAutonomiaOperativo(proyectoID, policy, "supervision_transcript_signal", strings.TrimSpace(agenteOrigen))
}

func resolverAgenteReviewSignalTranscript(proyectoID int64, reviewerPreferido, agenteOrigen string) (*db.Agente, bool, error) {
	return resolverAgentePreferidoOActivoSignalTranscript(
		proyectoID,
		reviewerPreferido,
		"review_transcript_signal",
		[]string{"revisor", "reviewer", "supervisor", "orquestador", "admin", "programador"},
		agenteOrigen,
	)
}

func resolverAgentePreferidoOActivoSignalTranscript(proyectoID int64, preferido, contexto string, roles []string, agenteOrigen string) (*db.Agente, bool, error) {
	agente, activado, err := prepararAgentePreferidoAutonomia(proyectoID, strings.TrimSpace(preferido), strings.TrimSpace(contexto))
	if err != nil || activado || agente != nil {
		return agente, activado, err
	}
	agente, err = seleccionarAgenteActivoProyectoPreferido(
		proyectoID,
		strings.TrimSpace(preferido),
		roles,
		strings.TrimSpace(agenteOrigen),
	)
	return agente, false, err
}

func asegurarTareaReplanAutonomiaSignal(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto, supervisor *db.Agente) (string, error) {
	if !contextoSignalTranscriptActivo(item, policy, proyecto) {
		return "", nil
	}
	return ejecutarPlanReplanSignalTranscript(item, policy, proyecto, supervisor)
}

func ejecutarPlanReplanSignalTranscript(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto, supervisor *db.Agente) (string, error) {
	if clasificacionSignalTranscript(item) != "needs_replan" {
		return "", nil
	}
	existe, err := existeTareaReplanAutonomiaPendiente(proyecto.ID)
	if err != nil {
		return "", err
	}
	if existe {
		return "replan_task_exists", nil
	}
	target := resolverAgenteObjetivoReplanAutonomiaSignal(policy, supervisor, agenteSignalTranscript(item))
	id, err := crearTareaReplanAutonomiaSignal(policy, proyecto, item, target)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("replan_task_created:%d", id), nil
}

func resolverAgenteObjetivoReplanAutonomiaSignal(policy *db.ProyectoAutonomia, supervisor *db.Agente, agenteOrigen string) string {
	agenteOrigen = strings.TrimSpace(agenteOrigen)
	if supervisor != nil && !strings.EqualFold(nombreAgenteAutonomia(supervisor), agenteOrigen) {
		return nombreAgenteAutonomia(supervisor)
	}
	if preferred := supervisorPreferidoAutonomia(policy); preferred != "" && !strings.EqualFold(preferred, agenteOrigen) {
		return preferred
	}
	return ""
}

func supervisorPreferidoAutonomia(policy *db.ProyectoAutonomia) string {
	if policy == nil {
		return ""
	}
	return strings.TrimSpace(policy.SupervisorAgente)
}

func reviewerPreferidoAutonomia(policy *db.ProyectoAutonomia) string {
	if policy == nil {
		return ""
	}
	return strings.TrimSpace(policy.ReviewerAgente)
}

func crearTareaReplanAutonomiaSignal(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, item *db.RuntimeTranscriptEntry, target string) (int64, error) {
	if policy == nil || proyecto == nil || item == nil {
		return 0, nil
	}
	id, err := tareasService.Create(tareasapp.CreateTaskInput{
		Titulo:      autonomiaReplanTaskTitle,
		Descripcion: construirDescripcionTareaReplanAutonomia(policy, proyecto, item),
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Agente:      target,
		Proyecto:    proyecto.Slug,
		Notas:       construirNotasTareaReplanAutonomiaSignal(item),
	})
	if err != nil {
		return 0, err
	}
	if err := arrancarTareaAutonomiaSiAsignada(id, target); err != nil {
		return 0, err
	}
	return id, nil
}

func existeTareaReplanAutonomiaPendiente(proyectoID int64) (bool, error) {
	if proyectoID <= 0 {
		return false, nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		return false, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaCompletada, db.TareaCancelada:
			continue
		}
		if esTareaReplanAutonomia(tarea) {
			return true, nil
		}
	}
	return false, nil
}

func construirNotasTareaReplanAutonomiaSignal(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return "autonomia:needs_replan"
	}
	return fmt.Sprintf("autonomia:needs_replan;transcript:%d;agente_origen:%s", idSignalTranscript(item), agenteSignalTranscript(item))
}

func esTareaReplanAutonomia(tarea *db.Tarea) bool {
	if tarea == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(tarea.Titulo), autonomiaReplanTaskTitle) {
		return true
	}
	return strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:needs_replan")
}

func construirDescripcionTareaReplanAutonomia(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, item *db.RuntimeTranscriptEntry) string {
	partes := []string{
		"Orquesta ha detectado en el transcript una señal de needs_replan: el agente no tiene claro el siguiente paso útil.",
		"Replanifica backlog, prioridades, propuestas, worktrees y checkpoints para abrir o reforzar el siguiente frente útil sin intervención humana.",
	}
	if slugProyectoAutonomia(proyecto) != "" {
		partes = append(partes, "Proyecto: "+slugProyectoAutonomia(proyecto)+".")
	}
	if item != nil && agenteSignalTranscript(item) != "" {
		partes = append(partes, "Agente origen: "+agenteSignalTranscript(item)+".")
	}
	if item != nil && textoSignalTranscript(item) != "" {
		partes = append(partes, "Señal transcript: "+textoSignalTranscript(item)+".")
	}
	if objetivoGeneralAutonomia(policy) != "" {
		partes = append(partes, "Objetivo general: "+objetivoGeneralAutonomia(policy)+".")
	}
	if definitionOfDoneAutonomia(policy) != "" {
		partes = append(partes, "Definition of done: "+definitionOfDoneAutonomia(policy)+".")
	}
	return strings.Join(partes, " ")
}

func notificarReviewerSignalTranscript(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto) (string, error) {
	if !contextoSignalTranscriptActivo(item, policy, proyecto) {
		return "", nil
	}
	return ejecutarPlanReviewerSignalTranscript(item, policy, proyecto)
}

func ejecutarPlanReviewerSignalTranscript(item *db.RuntimeTranscriptEntry, policy *db.ProyectoAutonomia, proyecto *db.Proyecto) (string, error) {
	reviewer, activado, err := resolverAgenteReviewSignalTranscript(proyecto.ID, reviewerPreferidoAutonomia(policy), agenteSignalTranscript(item))
	if err != nil {
		return "", err
	}
	return resolverEstadoAgenteObjetivoSignalTranscript(reviewer, activado, agenteSignalTranscript(item), proyecto, "inspeccionar_ready_for_review", construirInstruccionReviewSignalTranscript(policy, proyecto, nombreAgenteAutonomia(reviewer), item), item, "reviewer", "reviewer_start")
}

func resolverEstadoAgenteObjetivoSignalTranscript(agente *db.Agente, activado bool, agenteOrigen string, proyecto *db.Proyecto, accion, instruccion string, item *db.RuntimeTranscriptEntry, prefijoEstado, notaArranque string) (string, error) {
	if activado {
		return strings.TrimSpace(notaArranque), nil
	}
	if agente == nil || strings.EqualFold(nombreAgenteAutonomia(agente), strings.TrimSpace(agenteOrigen)) {
		return "", nil
	}
	return encolarNudgeSignalTranscriptAutonomia(nombreAgenteAutonomia(agente), proyecto, accion, instruccion, item, prefijoEstado)
}

func construirInstruccionSupervisionSignalTranscript(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, supervisor string, item *db.RuntimeTranscriptEntry) string {
	parts := []string{
		"Orquesta: un agente del proyecto ha emitido una señal de duda, espera o bloqueo en su transcript.",
		"Supervisa el frente ahora mismo: revisa el transcript, el contexto del proyecto, las tareas y propuestas activas, y decide el siguiente paso sin escalar a humano salvo que falten credenciales, secretos o un recurso externo real.",
	}
	if slugProyectoAutonomia(proyecto) != "" {
		parts = append(parts, fmt.Sprintf("Proyecto: %s.", slugProyectoAutonomia(proyecto)))
	}
	if strings.TrimSpace(supervisor) != "" {
		parts = append(parts, "Supervisor responsable: "+strings.TrimSpace(supervisor)+".")
	}
	if item != nil {
		if agenteSignalTranscript(item) != "" {
			parts = append(parts, "Agente origen: "+agenteSignalTranscript(item)+".")
		}
		if clasificacionSignalTranscript(item) != "" {
			parts = append(parts, "Clasificación: "+clasificacionSignalTranscript(item)+".")
		}
		if textoSignalTranscript(item) != "" {
			parts = append(parts, "Fragmento detectado: "+textoSignalTranscript(item)+".")
		}
	}
	if objetivoGeneralAutonomia(policy) != "" {
		parts = append(parts, "Objetivo general: "+objetivoGeneralAutonomia(policy)+".")
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
	if slugProyectoAutonomia(proyecto) != "" {
		parts = append(parts, fmt.Sprintf("Proyecto: %s.", slugProyectoAutonomia(proyecto)))
	}
	if strings.TrimSpace(reviewer) != "" {
		parts = append(parts, "Reviewer responsable: "+strings.TrimSpace(reviewer)+".")
	}
	if item != nil {
		if agenteSignalTranscript(item) != "" {
			parts = append(parts, "Agente origen: "+agenteSignalTranscript(item)+".")
		}
		if textoSignalTranscript(item) != "" {
			parts = append(parts, "Fragmento detectado: "+textoSignalTranscript(item)+".")
		}
	}
	if objetivoGeneralAutonomia(policy) != "" {
		parts = append(parts, "Objetivo general: "+objetivoGeneralAutonomia(policy)+".")
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
		dispatched, err := procesarRuntimeMailboxInteractivoMensaje(msg, consumed, snapshot)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// procesarRuntimeMailboxInteractivoMensaje valida y despacha un mensaje de tipo
// interactivo para el handle activo del agente destino.
// Retorna true si el mensaje fue procesado (encolado, reactivado u obsoleto consumido).
func procesarRuntimeMailboxInteractivoMensaje(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (bool, error) {
	texto, ok := construirInstruccionMailboxInteractivo(msg)
	if !ok {
		return false, nil
	}
	if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
		return false, err
	}
	if !runtimeMailboxSiguePendiente(msg.ID) {
		return false, nil
	}
	handle, err := snapshot.activeHandle(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil {
		return false, err
	}
	if handle == nil {
		return intentarReactivarRuntimeMailboxSinHandle(msg, snapshot)
	}
	if refreshed, _, _, err := snapshot.supervisedHandle(handle); err != nil {
		return false, err
	} else if refreshed != nil {
		handle = refreshed
	}
	if obsoleta, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg, handle, "interactive", time.Now().UTC()); err != nil {
		return false, err
	} else if obsoleta {
		consumed[msg.ID] = struct{}{}
		return true, nil
	}
	if db.RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliveryInteractive {
		return false, nil
	}
	if !runtimeHandleListaParaDispatchInteractivo(handle) {
		return false, nil
	}
	if !runtimeMailboxShouldReevaluate("interactive", msg.ID, handle.ID) {
		return false, nil
	}
	if abierta, err := existeRuntimeOrderAbiertaPorHandleEnSnapshot(snapshot, msg, handle.ID, "send_instruction"); err != nil {
		return false, err
	} else if abierta {
		return false, nil
	}
	if dedupe, err := existeIntentoSendInstructionMailboxParaHandleEnSnapshot(snapshot, msg, handle, ""); err != nil {
		return false, err
	} else if dedupe {
		return false, nil
	}
	orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, texto, "")
	if err != nil {
		return false, err
	}
	if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
		superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
		if err != nil {
			return false, err
		}
		if superseded > 0 {
			db.Audit("orquesta", "runtime_mailbox_interactivo_supersede", "runtime_mailbox", msg.ID,
				fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
		}
	}
	db.Audit("orquesta", "runtime_mailbox_interactivo", "runtime_order", orderID,
		fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
	consumed[msg.ID] = struct{}{}
	return true, nil
}

func intentarReactivarRuntimeMailboxPoolLocalSinHandle(msg *db.RuntimeMailboxMessage, snapshot *runtimeMailboxBatchSnapshot) (bool, error) {
	proyecto, err := resolverProyectoRuntimeMailboxReactivacion(msg, snapshot)
	if msg == nil || proyecto == nil {
		return false, nil
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if agente == "" {
		return false, nil
	}
	if !agenteUsaPoolLocalCompartidoParaReactivacion(agente, proyecto) {
		return false, nil
	}
	poolLocalSesion := agenteUsaSesionPoolLocalCompartido(agente, strings.TrimSpace(proyecto.Slug))
	permite, poolSlug, err := db.PoolLocalCompartidoPermiteActivacionAgenteProyecto(agente, strings.TrimSpace(proyecto.Slug))
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(poolSlug) == "" && !poolLocalSesion {
		return false, nil
	}
	if strings.TrimSpace(poolSlug) != "" && !permite {
		db.Audit("orquesta", "runtime_mailbox_pool_local_sin_capacidad", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s proyecto=%s pool=%s kind=%s", agente, strings.TrimSpace(proyecto.Slug), strings.TrimSpace(poolSlug), strings.TrimSpace(msg.Kind)))
		return false, nil
	}
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(agente, &proyecto.ID, "start", "resume", "handoff"); err != nil {
		return false, err
	} else if pendiente {
		return false, nil
	}
	if err := encolarControlAutonomiaProyecto(agente, proyecto, agenteControlAccionStart, "runtime_mailbox_pool_local_sin_handle"); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_pool_local_reactivacion", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s proyecto=%s pool=%s kind=%s", agente, strings.TrimSpace(proyecto.Slug), strings.TrimSpace(poolSlug), strings.TrimSpace(msg.Kind)))
	return true, nil
}

func intentarReactivarRuntimeMailboxSinHandle(msg *db.RuntimeMailboxMessage, snapshot *runtimeMailboxBatchSnapshot) (bool, error) {
	proyecto, err := resolverProyectoRuntimeMailboxReactivacion(msg, snapshot)
	if err != nil {
		return false, err
	}
	if msg == nil || proyecto == nil {
		return false, nil
	}
	if reactivado, err := intentarReactivarRuntimeMailboxPoolLocalSinHandle(msg, snapshot); err != nil || reactivado {
		return reactivado, err
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if agente == "" {
		return false, nil
	}
	infoAgente, err := agenteIfExists(agente)
	if err != nil {
		return false, err
	}
	if infoAgente == nil || !infoAgente.Habilitado {
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return false, err
		}
		db.Audit("orquesta", "runtime_mailbox_consumida_agente_retirado", "runtime_mailbox", msg.ID,
			fmt.Sprintf("agente=%s proyecto_id=%d kind=%s", agente, proyecto.ID, strings.TrimSpace(msg.Kind)))
		return true, nil
	}
	kind := strings.TrimSpace(msg.Kind)
	switch kind {
	case db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh, "watchdog":
		return false, nil
	}
	if kind != "pipeline_local" && kind != "microprogramacion" && kind != "autonomia" && kind != "nudge" {
		if tieneTrabajo, err := dbAgenteTieneTrabajoArrancable(agente, proyecto.ID); err != nil {
			return false, err
		} else if tieneTrabajo {
			return false, nil
		}
	}
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(agente, &proyecto.ID, "start", "resume", "handoff"); err != nil {
		return false, err
	} else if pendiente {
		return false, nil
	}
	reactivado, err := encolarReactivacionAgenteProyectoSiProcede(agente, proyecto, "runtime_mailbox_sin_handle")
	if err != nil {
		return false, err
	}
	if !reactivado {
		return false, nil
	}
	db.Audit("orquesta", "runtime_mailbox_reactivacion_sin_handle", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s proyecto_id=%d kind=%s", agente, proyecto.ID, strings.TrimSpace(msg.Kind)))
	return true, nil
}

func resolverProyectoRuntimeMailboxReactivacion(msg *db.RuntimeMailboxMessage, snapshot *runtimeMailboxBatchSnapshot) (*db.Proyecto, error) {
	if msg == nil {
		return nil, nil
	}
	if msg.ProyectoID != nil && *msg.ProyectoID > 0 {
		if snapshot != nil {
			return snapshot.project(*msg.ProyectoID)
		}
		return runtimesService.GetProject(strconv.FormatInt(*msg.ProyectoID, 10))
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if agente == "" {
		return nil, nil
	}
	return resolverProyectoReactivacionAgente(agente)
}

func agenteUsaSesionPoolLocalCompartido(agente, proyecto string) bool {
	agente = strings.TrimSpace(agente)
	proyecto = strings.TrimSpace(proyecto)
	if agente == "" || proyecto == "" {
		return false
	}
	if activa, err := sesionesAPIService.GetActiveSession(agente, proyecto); err == nil && sesionUsaPoolLocalCompartido(activa) {
		return true
	}
	if ultima, err := sesionesAPIService.GetLastSession(agente, proyecto); err == nil && sesionUsaPoolLocalCompartido(ultima) {
		return true
	}
	return false
}

func agenteUsaPoolLocalCompartidoParaReactivacion(agente string, proyecto *db.Proyecto) bool {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false
	}
	if proyecto != nil && agenteUsaSesionPoolLocalCompartido(agente, strings.TrimSpace(proyecto.Slug)) {
		return true
	}
	if proyecto != nil {
		if handle, err := resolverHandleReactivacionAgente(agente, &proyecto.ID); err == nil && runtimeHandleUsaPoolLocalCompartido(handle) {
			return true
		}
	}
	return runtimeagente.EsConectorFamiliaOllama(runtimeagente.ConectorPorDefectoAgente(agente), "")
}

func runtimeHandleUsaPoolLocalCompartido(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "remote_http") {
		return true
	}
	meta := strings.TrimSpace(handle.MetadataJSON)
	return strings.EqualFold(strings.TrimSpace(stringFromMetadataJSON(meta, "driver")), "ollama_pool_local") ||
		strings.EqualFold(strings.TrimSpace(stringFromMetadataJSON(meta, "conector_canonico")), "ollama_pool_local") ||
		strings.EqualFold(strings.TrimSpace(stringFromMetadataJSON(meta, "connector")), "ollama_pool_local")
}

func sesionUsaPoolLocalCompartido(sesion *db.Sesion) bool {
	if sesion == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(sesion.Herramienta), "ollama_pool_local") ||
		strings.EqualFold(strings.TrimSpace(sesion.ConectorSlug), "ollama_pool_local") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(stringFromMetadataJSON(sesion.ResumePayloadJSON, "driver")), "ollama_pool_local")
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
		dispatched, err := procesarRuntimeMailboxCoordinatedRestartMensaje(latest[key], snapshot)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// procesarRuntimeMailboxCoordinatedRestartMensaje despacha un mensaje coordinated_restart
// ya seleccionado como candidato (el más reciente para su agente/proyecto).
// Retorna true si se encoló el reinicio coordinado o se consumió como obsoleto.
func procesarRuntimeMailboxCoordinatedRestartMensaje(msg *db.RuntimeMailboxMessage, snapshot *runtimeMailboxBatchSnapshot) (bool, error) {
	if msg == nil || msg.ProyectoID == nil || *msg.ProyectoID <= 0 {
		return false, nil
	}
	agente := strings.TrimSpace(msg.ToAgente)
	proyectoID := *msg.ProyectoID
	proyecto, err := snapshot.project(proyectoID)
	if err != nil || proyecto == nil {
		return false, err
	}
	handle, err := snapshot.activeHandle(agente, msg.ProyectoID)
	if err != nil {
		return false, err
	}
	if handle == nil {
		return false, nil
	}
	if obsoleta, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg, handle, "coordinated_restart", time.Now().UTC()); err != nil {
		return false, err
	} else if obsoleta {
		return true, nil
	}
	motivo := fmt.Sprintf("runtime_mailbox_coordinated_restart:%s:%d", strings.TrimSpace(msg.Kind), msg.ID)
	stopOrderID, startOrderID, err := encolarReinicioCoordinadoMailbox(handle, proyecto, msg, motivo)
	if err != nil {
		return false, err
	}
	if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
		superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(agente, msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
		if err != nil {
			return false, err
		}
		if superseded > 0 {
			db.Audit("orquesta", "runtime_mailbox_coordinated_restart_supersede", "runtime_mailbox", msg.ID,
				fmt.Sprintf("agente=%s kind=%s superseded=%d", agente, strings.TrimSpace(msg.Kind), superseded))
		}
	}
	db.Audit("orquesta", "runtime_mailbox_coordinated_restart", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s proyecto=%s kind=%s stop_order_id=%d start_order_id=%d", agente, proyecto.Slug, strings.TrimSpace(msg.Kind), stopOrderID, startOrderID))
	return true, nil
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
		dispatched, err := procesarRuntimeMailboxSessionResumeMensaje(msg, consumed, snapshot)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// procesarRuntimeMailboxSessionResumeMensaje resuelve handle y runtime para un mensaje
// session_resume y lo despacha por el canal que corresponda (con sesión o fallback interactivo).
// Retorna true si el mensaje fue procesado (encolado, reactivado u obsoleto consumido).
func procesarRuntimeMailboxSessionResumeMensaje(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (bool, error) {
	texto, ok := construirInstruccionMailboxInteractivo(msg)
	if !ok {
		return false, nil
	}
	if err := coalescerRuntimeMailboxPendiente(msg); err != nil {
		return false, err
	}
	if !runtimeMailboxSiguePendiente(msg.ID) {
		return false, nil
	}
	handle, err := snapshot.activeHandle(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil {
		return false, err
	}
	if handle == nil {
		return intentarReactivarRuntimeMailboxSinHandle(msg, snapshot)
	}
	var runtimeInstance *db.RuntimeInstance
	if supervisedHandle, supervisedRuntime, _, err := snapshot.supervisedHandle(handle); err != nil {
		return false, err
	} else {
		if supervisedHandle != nil {
			handle = supervisedHandle
		}
		runtimeInstance = supervisedRuntime
	}
	if runtimeInstance == nil {
		runtimeInstance, err = runtimeCanonicoDesdeHandleAgenteProyecto(handle, strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
		if err != nil {
			return false, err
		}
	}
	handle, externalSessionID, err := snapshot.externalSessionHandle(handle, runtimeInstance)
	if err != nil {
		return false, err
	}
	if handle == nil {
		return false, nil
	}
	if strings.TrimSpace(externalSessionID) == "" {
		return procesarRuntimeMailboxSessionResumeFallbackInteractivo(msg, consumed, snapshot, handle, runtimeInstance, texto)
	}
	return procesarRuntimeMailboxSessionResumeConSesion(msg, consumed, snapshot, handle, runtimeInstance, texto, externalSessionID)
}

// procesarRuntimeMailboxSessionResumeFallbackInteractivo gestiona mensajes de resume
// cuando no hay externalSessionID: usa el canal interactivo transitorio como fallback.
func procesarRuntimeMailboxSessionResumeFallbackInteractivo(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot, handle *db.RuntimeHandle, runtimeInstance *db.RuntimeInstance, texto string) (bool, error) {
	if !runtimeHandleAdmiteFallbackInteractivoTransitorio(handle) {
		return false, nil
	}
	if !runtimeHandleListaParaDispatchInteractivo(handle) {
		return false, nil
	}
	if obsoleta, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg, handle, "interactive_transitory", time.Now().UTC()); err != nil {
		return false, err
	} else if obsoleta {
		consumed[msg.ID] = struct{}{}
		return true, nil
	}
	if observada, err := consumirRuntimeMailboxBootstrapObservadoSiProcede(msg, handle, runtimeInstance, "interactive_transitory"); err != nil {
		return false, err
	} else if observada {
		consumed[msg.ID] = struct{}{}
		return true, nil
	}
	return encolarRuntimeMailboxSessionResumeSiCorresponde(msg, consumed, snapshot, handle, runtimeInstance, texto, "", "runtime_mailbox_session_resume_interactive_fallback", "runtime_mailbox_session_resume_interactive_supersede")
}

// procesarRuntimeMailboxSessionResumeConSesion gestiona mensajes de resume cuando
// hay una externalSessionID: verifica condiciones TMUX y encola el resume normal.
func procesarRuntimeMailboxSessionResumeConSesion(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot, handle *db.RuntimeHandle, runtimeInstance *db.RuntimeInstance, texto, externalSessionID string) (bool, error) {
	if obsoleta, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg, handle, "session_resume", time.Now().UTC()); err != nil {
		return false, err
	} else if obsoleta {
		consumed[msg.ID] = struct{}{}
		return true, nil
	}
	if observada, err := consumirRuntimeMailboxBootstrapObservadoSiProcede(msg, handle, runtimeInstance, "session_resume"); err != nil {
		return false, err
	} else if observada {
		consumed[msg.ID] = struct{}{}
		return true, nil
	}
	if blocked, err := runtimeMailboxBloqueadaPorBootstrapPendiente(msg, handle, runtimeInstance); err != nil {
		return false, err
	} else if blocked {
		return false, nil
	}
	if !runtimeMailboxShouldReevaluate("session_resume", msg.ID, handle.ID) {
		return false, nil
	}
	if db.RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliverySessionResume {
		return false, nil
	}
	if !runtimeHandleListaParaDispatchSessionResumeTMUX(handle, runtimeInstance) {
		return false, nil
	}
	return encolarRuntimeMailboxSessionResumeSiCorresponde(msg, consumed, snapshot, handle, runtimeInstance, texto, externalSessionID, "runtime_mailbox_session_resume", "runtime_mailbox_session_resume_supersede")
}

func encolarRuntimeMailboxSessionResumeSiCorresponde(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot, handle *db.RuntimeHandle, runtimeInstance *db.RuntimeInstance, texto, externalSessionID, auditAction, supersedeAction string) (bool, error) {
	if blocked, err := runtimeMailboxBloqueadaPorBootstrapPendiente(msg, handle, runtimeInstance); err != nil {
		return false, err
	} else if blocked {
		return false, nil
	}
	if abierta, err := existeRuntimeOrderAbiertaPorHandleEnSnapshot(snapshot, msg, handle.ID, "send_instruction"); err != nil {
		return false, err
	} else if abierta {
		return false, nil
	}
	if dedupe, err := existeIntentoSendInstructionMailboxParaHandleEnSnapshot(snapshot, msg, handle, externalSessionID); err != nil {
		return false, err
	} else if dedupe {
		return false, nil
	}
	orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, texto, externalSessionID)
	if err != nil {
		return false, err
	}
	if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
		superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
		if err != nil {
			return false, err
		}
		if superseded > 0 {
			db.Audit("orquesta", strings.TrimSpace(supersedeAction), "runtime_mailbox", msg.ID,
				fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
		}
	}
	db.Audit("orquesta", strings.TrimSpace(auditAction), "runtime_order", orderID,
		fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
	consumed[msg.ID] = struct{}{}
	return true, nil
}

func runtimeHandleAdmiteFallbackInteractivoTransitorio(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	switch db.RuntimeHandleMailboxDeliveryMode(handle) {
	case runtimeagente.MailboxDeliveryCoordinatedRestart, runtimeagente.MailboxDeliveryBootstrapOnly:
		return false
	}
	if db.RuntimeHandleMailboxDeliveryMode(handle) == runtimeagente.MailboxDeliveryInteractive {
		return true
	}
	meta := mapFromJSON(strings.TrimSpace(handle.MetadataJSON))
	value := func(key string) string {
		if meta == nil {
			return ""
		}
		raw, _ := meta[key].(string)
		return strings.TrimSpace(raw)
	}
	if !strings.EqualFold(value("driver"), "process_pty_cli") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Transporte), "cli") || !strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") {
		return false
	}
	if value("external_session_id") != "" {
		return false
	}
	for _, key := range []string{"supervisor_ref", "stdin_path", "stdin_raw_path"} {
		if value(key) != "" {
			return true
		}
	}
	return false
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
			reactivado, err := intentarReactivarRuntimeMailboxSinHandle(msg, snapshot)
			if err != nil {
				return total, err
			}
			if reactivado {
				total++
			}
			continue
		}
		if supervisedHandle, _, _, err := snapshot.supervisedHandle(handle); err != nil {
			return total, err
		} else if supervisedHandle != nil {
			handle = supervisedHandle
		}
		dispatched, err := procesarRuntimeMailboxBootstrapTMUXMensaje(msg, consumed, snapshot, handle, texto)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// procesarRuntimeMailboxBootstrapTMUXMensaje valida y despacha un mensaje bootstrap_tmux
// para un handle ya resuelto. Retorna true si el mensaje fue procesado (encolado u obsoleto).
func procesarRuntimeMailboxBootstrapTMUXMensaje(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot, handle *db.RuntimeHandle, texto string) (bool, error) {
	if obsoleta, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msg, handle, "bootstrap_tmux", time.Now().UTC()); err != nil {
		return false, err
	} else if obsoleta {
		consumed[msg.ID] = struct{}{}
		return true, nil
	}
	if !runtimeMailboxShouldReevaluate("bootstrap_tmux", msg.ID, handle.ID) {
		return false, nil
	}
	if db.RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliveryBootstrapOnly {
		return false, nil
	}
	if !runtimeHandleListaParaDispatchBootstrapTMUX(handle) {
		return false, nil
	}
	if abierta, err := existeRuntimeOrderAbiertaPorHandleEnSnapshot(snapshot, msg, handle.ID, "send_instruction"); err != nil {
		return false, err
	} else if abierta {
		return false, nil
	}
	if dedupe, err := existeIntentoSendInstructionMailboxParaHandleEnSnapshot(snapshot, msg, handle, ""); err != nil {
		return false, err
	} else if dedupe {
		return false, nil
	}
	orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, texto, "")
	if err != nil {
		return false, err
	}
	if runtimeMailboxKindCoalescible(strings.TrimSpace(msg.Kind)) {
		superseded, err := db.ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), msg.ID)
		if err != nil {
			return false, err
		}
		if superseded > 0 {
			db.Audit("orquesta", "runtime_mailbox_bootstrap_tmux_supersede", "runtime_mailbox", msg.ID,
				fmt.Sprintf("agente=%s kind=%s superseded=%d", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), superseded))
		}
	}
	db.Audit("orquesta", "runtime_mailbox_bootstrap_tmux", "runtime_order", orderID,
		fmt.Sprintf("mailbox_id=%d agente=%s kind=%s", msg.ID, strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind)))
	consumed[msg.ID] = struct{}{}
	return true, nil
}

func reconciliarRuntimeMailboxGuidanceDurableEnInboxBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	for _, msg := range mailbox {
		if msg == nil || !runtimeMailboxGuidanceDurablePersistibleEnInbox(msg) {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		dispatched, err := reconciliarRuntimeMailboxGuidanceDurableEnInboxMensaje(msg, consumed, snapshot)
		if err != nil {
			return total, err
		}
		if dispatched {
			total++
		}
	}
	return total, nil
}

// reconciliarRuntimeMailboxGuidanceDurableEnInboxMensaje valida y escribe en inbox
// un mensaje guidance-durable para el agente destino.
// Retorna true si el mensaje fue entregado y consumido.
func reconciliarRuntimeMailboxGuidanceDurableEnInboxMensaje(msg *db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (bool, error) {
	handle, err := snapshot.activeHandle(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil {
		return false, err
	}
	if handle == nil {
		return false, nil
	}
	if !runtimeMailboxGuidanceDurableUsaInbox(handle) {
		return false, nil
	}
	if ok, err := runtimeMailboxGuidanceDurableTieneEntregaMailboxOnly(snapshot, msg, handle); err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}
	if msg.ProyectoID == nil || *msg.ProyectoID <= 0 {
		return false, nil
	}
	proyecto, err := snapshot.project(*msg.ProyectoID)
	if err != nil {
		return false, err
	}
	if proyecto == nil {
		return false, nil
	}
	tarea, err := tareaActivaAutonomiaProyectoAgente(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil {
		return false, err
	}
	if err := escribirInboxRuntimeMailboxDurable(proyecto, strings.TrimSpace(msg.ToAgente), tarea, msg, handle); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_guidance_durable_inbox", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s proyecto=%s kind=%s", strings.TrimSpace(msg.ToAgente), strings.TrimSpace(proyecto.Slug), strings.TrimSpace(msg.Kind)))
	consumed[msg.ID] = struct{}{}
	return true, nil
}

func runtimeMailboxBootstrapPendientePermiteGuidanceDurable(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle) bool {
	return runtimeMailboxGuidanceDurablePersistibleEnInbox(msg) && runtimeMailboxGuidanceDurableUsaInbox(handle)
}

func runtimeMailboxBloqueadaPorBootstrapPendiente(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, runtimeInstance *db.RuntimeInstance) (bool, error) {
	if msg == nil {
		return false, nil
	}
	covered, _, _, err := db.RuntimeMailboxCubiertoPorBootstrapPendiente(msg.ID, handle, runtimeInstance)
	if err != nil {
		return false, err
	}
	if !covered || runtimeMailboxBootstrapPendientePermiteGuidanceDurable(msg, handle) {
		return false, nil
	}
	if runtimeTMUXSessionResumeWorkerReady(handle) {
		return false, nil
	}
	return true, nil
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

func runtimeHandleBootstrapTMUXActivo(handle *db.RuntimeHandle) bool {
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
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "running", "ready", "working":
		return true
	default:
		return false
	}
}

func runtimeMailboxGuidanceDurablePersistibleEnInbox(msg *db.RuntimeMailboxMessage) bool {
	if msg == nil {
		return false
	}
	switch strings.TrimSpace(msg.Kind) {
	case "autonomia", "nudge", "watchdog", db.MailboxKindGovernanceRefresh, db.MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func runtimeMailboxGuidanceDurableUsaInbox(handle *db.RuntimeHandle) bool {
	if handle == nil || db.RuntimeHandlePermiteSendInputInteractivo(handle) {
		return false
	}
	meta := mapFromJSON(strings.TrimSpace(handle.MetadataJSON))
	driver := strings.TrimSpace(stringMapValue(meta, "driver"))
	return strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(driver, "tmux_cli_session")
}

func reconciliarRuntimeMailboxPipelineBootstrapActivaBatchConMailbox(mailbox []*db.RuntimeMailboxMessage, consumed map[int64]struct{}, snapshot *runtimeMailboxBatchSnapshot) (int, error) {
	total := 0
	now := time.Now().UTC()
	for _, msg := range mailbox {
		if msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Kind), "pipeline_local") {
			continue
		}
		if _, skip := consumed[msg.ID]; skip {
			continue
		}
		reconciled, err := reconciliarRuntimeMailboxPipelineBootstrapActivaSiProcede(msg, snapshot, now)
		if err != nil {
			return total, err
		}
		if reconciled {
			consumed[msg.ID] = struct{}{}
			total++
		}
	}
	return total, nil
}

func reconciliarRuntimeMailboxPipelineBootstrapActivaSiProcede(msg *db.RuntimeMailboxMessage, snapshot *runtimeMailboxBatchSnapshot, now time.Time) (bool, error) {
	if msg == nil || snapshot == nil || !strings.EqualFold(strings.TrimSpace(msg.Kind), "pipeline_local") {
		return false, nil
	}
	handle, err := snapshot.activeHandle(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil || handle == nil {
		return false, err
	}
	if supervisedHandle, _, _, err := snapshot.supervisedHandle(handle); err != nil {
		return false, err
	} else if supervisedHandle != nil {
		handle = supervisedHandle
	}
	if !runtimeMailboxGuidanceDurableUsaInbox(handle) {
		return false, nil
	}
	order, err := runtimeMailboxPipelineBootstrapPendingOrder(snapshot, msg, handle.ID)
	if err != nil || order == nil {
		return false, err
	}
	receiptAt, ok, err := runtimeMailboxPipelineBootstrapReceiptAt(msg, handle, now)
	if err != nil || !ok {
		return false, err
	}
	if err := completarRuntimeOrderMailboxBootstrapObservada(order, receiptAt); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_pipeline_bootstrap_progress", "runtime_mailbox", msg.ID,
		fmt.Sprintf("agente=%s handle_id=%d receipt_at=%s", strings.TrimSpace(msg.ToAgente), handle.ID, receiptAt.UTC().Format(time.RFC3339Nano)))
	return true, nil
}

func runtimeMailboxPipelineBootstrapReceiptAt(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, now time.Time) (time.Time, bool, error) {
	if msg == nil || handle == nil {
		return time.Time{}, false, nil
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON))
	if err != nil || snap == nil {
		return time.Time{}, false, err
	}
	view := snap.View(now.UTC(), 90*time.Second)
	if view == nil || !view.Alive || view.HeartbeatStale {
		return time.Time{}, false, nil
	}
	payload := mapFromJSON(strings.TrimSpace(msg.PayloadJSON))
	targetTaskID := int64PtrFromMap(payload, "tarea_objetivo_id")
	if targetTaskID == nil || *targetTaskID <= 0 {
		targetTaskID = int64PtrFromMap(payload, "tarea_id")
	}
	if msg.ProyectoID == nil || targetTaskID == nil || *targetTaskID <= 0 {
		return time.Time{}, false, nil
	}
	activeTaskID, err := db.GetTareaActivaIDPorAgenteProyecto(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil || activeTaskID <= 0 || activeTaskID != *targetTaskID {
		return time.Time{}, false, err
	}
	if view.LastProgressAt != nil && !view.LastProgressAt.IsZero() {
		receiptAt := view.LastProgressAt.UTC()
		if receiptAt.After(msg.CreatedAt.UTC()) {
			return receiptAt, true, nil
		}
	}
	if view.LastOutputAt != nil && !view.LastOutputAt.IsZero() {
		receiptAt := view.LastOutputAt.UTC()
		if receiptAt.After(msg.CreatedAt.UTC()) {
			return receiptAt, true, nil
		}
	}
	return time.Time{}, false, nil
}

func runtimeMailboxPipelineBootstrapPendingOrder(snapshot *runtimeMailboxBatchSnapshot, msg *db.RuntimeMailboxMessage, handleID int64) (*db.RuntimeOrder, error) {
	if snapshot == nil || msg == nil || handleID <= 0 {
		return nil, nil
	}
	orders, err := snapshot.ordersForAgentProject(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil {
		return nil, err
	}
	var fallback *db.RuntimeOrder
	for _, order := range orders {
		if !runtimeOrderMatchesMailboxSendInstructionAttempt(order, msg.ID, handleID) {
			if runtimeOrderMatchesMailboxSendInstructionByMailbox(order, msg.ID) && fallback == nil {
				fallback = order
			}
			continue
		}
		if !runtimeOrderPendienteBootstrapMailboxOnly(order) {
			continue
		}
		return order, nil
	}
	if runtimeOrderPendienteBootstrapMailboxOnly(fallback) {
		return fallback, nil
	}
	return nil, nil
}

func runtimeOrderMatchesMailboxSendInstructionByMailbox(order *db.RuntimeOrder, mailboxID int64) bool {
	if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" || mailboxID <= 0 {
		return false
	}
	return runtimeOrderMailboxIDFromJSON(order.PayloadJSON) == mailboxID
}

func runtimeOrderPendienteBootstrapMailboxOnly(order *db.RuntimeOrder) bool {
	if order == nil {
		return false
	}
	switch strings.TrimSpace(order.Estado) {
	case "pendiente", "tomada", "ejecutando":
	default:
		return false
	}
	reason := runtimeOrderDeferredReason(order.ResultadoJSON)
	return strings.HasPrefix(strings.TrimSpace(reason), "runtime_handle_bootstrap_only_mailbox_only")
}

func completarRuntimeOrderMailboxBootstrapObservada(order *db.RuntimeOrder, receiptAt time.Time) error {
	if order == nil {
		return nil
	}
	result := mapFromJSON(strings.TrimSpace(order.ResultadoJSON))
	if result == nil {
		result = map[string]any{}
	}
	result["ok"] = true
	result["deferred"] = false
	result["mailbox_only"] = true
	result["delivery_state"] = "delivered"
	result["dispatch_state"] = "delivered"
	result["receipt_source"] = "tmux_worker_progress"
	result["delivery_receipt_at"] = receiptAt.UTC().Format(time.RFC3339Nano)
	result["deferred_reason"] = "tmux_worker_progress"
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return db.MarcarRuntimeOrderEstado(order.ID, "completada", string(raw), "")
}

func runtimeMailboxGuidanceDurableTieneEntregaMailboxOnly(snapshot *runtimeMailboxBatchSnapshot, msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle) (bool, error) {
	if snapshot == nil || msg == nil || handle == nil {
		return false, nil
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if agente == "" {
		return false, nil
	}
	currentSessionID, err := runtimeMailboxGuidanceDurableExternalSessionID(handle)
	if err != nil {
		return false, err
	}
	currentSignature := runtimeMailboxDeliveryAttemptSignature(handle, currentSessionID)
	orders, err := snapshot.ordersForAgentProject(agente, msg.ProyectoID)
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if !runtimeOrderMatchesMailboxSendInstructionAttempt(order, msg.ID, handle.ID) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(order.Estado), "completada") &&
			runtimeOrderMailboxOnlyResult(order.ResultadoJSON) &&
			runtimeOrderMatchesCurrentMailboxDeliveryAttempt(order, currentSessionID, currentSignature) {
			return true, nil
		}
	}
	return false, nil
}

func runtimeMailboxGuidanceDurableExternalSessionID(handle *db.RuntimeHandle) (string, error) {
	if handle == nil {
		return "", nil
	}
	if ext := strings.TrimSpace(stringFromMetadataJSON(handle.MetadataJSON, "external_session_id")); ext != "" {
		return ext, nil
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		sesion, err := db.GetSesionByID(*handle.SesionID)
		if err != nil {
			return "", err
		}
		if sesion != nil && strings.TrimSpace(sesion.ExternalSessionID) != "" {
			return strings.TrimSpace(sesion.ExternalSessionID), nil
		}
	}
	return "", nil
}

func runtimeOrderMatchesCurrentMailboxDeliveryAttempt(order *db.RuntimeOrder, currentSessionID, currentSignature string) bool {
	if order == nil || !runtimeOrderMailboxOnlyResult(order.ResultadoJSON) {
		return false
	}
	orderSignature := runtimeOrderDeliveryAttemptSignature(order.PayloadJSON)
	if currentSignature != "" && orderSignature != "" && currentSignature == orderSignature {
		return true
	}
	orderSessionID := strings.TrimSpace(runtimeOrderExternalSessionIDFromJSON(order.PayloadJSON))
	if currentSessionID != "" && orderSessionID != "" && currentSessionID == orderSessionID {
		return true
	}
	if currentSignature != "" {
		return false
	}
	return runtimeOrderCompletedMailboxAttemptStillBlocks(order, currentSessionID, currentSignature)
}

func tareaActivaAutonomiaProyectoAgente(agente string, proyectoID *int64) (*db.Tarea, error) {
	if strings.TrimSpace(agente) == "" || proyectoID == nil || *proyectoID <= 0 {
		return nil, nil
	}
	tareaID, err := db.GetTareaActivaIDPorAgenteProyecto(strings.TrimSpace(agente), proyectoID)
	if err != nil || tareaID <= 0 {
		return nil, err
	}
	return db.GetTarea(tareaID)
}

func escribirInboxRuntimeMailboxDurable(proyecto *db.Proyecto, agente string, tarea *db.Tarea, msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle) error {
	if proyecto == nil || msg == nil {
		return nil
	}
	agente = strings.TrimSpace(agente)
	base := strings.TrimSpace(proyecto.RutaAbs)
	if ruta, err := resolverRutaInboxRuntimeMailboxDurable(proyecto, agente, handle); err != nil {
		return err
	} else if strings.TrimSpace(ruta) != "" {
		base = strings.TrimSpace(ruta)
	}
	if base == "" {
		return nil
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	path := filepath.Join(base, ".orquesta-inbox.md")
	return os.WriteFile(path, []byte(construirInboxRuntimeMailboxDurableMarkdown(proyecto, tarea, msg)), 0o644)
}

func resolverRutaInboxRuntimeMailboxDurable(proyecto *db.Proyecto, agente string, handle *db.RuntimeHandle) (string, error) {
	if proyecto == nil {
		return "", nil
	}
	agente = strings.TrimSpace(agente)
	if ruta, err := runtimeMailboxGuidanceDurableWorkingDir(handle); err != nil {
		return "", err
	} else if strings.TrimSpace(ruta) != "" {
		return strings.TrimSpace(ruta), nil
	}
	if worktree, err := (worktreeRuntimeService{}).ResolveActiveWorktree(strings.TrimSpace(proyecto.Slug), agente); err == nil && worktree != nil && strings.TrimSpace(worktree.RutaAbs) != "" {
		return strings.TrimSpace(worktree.RutaAbs), nil
	}
	estado := coordinacion.WorktreeActive
	filter := coordinacion.WorktreeFilter{ProjectID: &proyecto.ID, State: &estado}
	if agente != "" {
		filter.Agent = &agente
	}
	worktrees, err := db.ListarWorktreesCoordRaw(filter)
	if err != nil {
		return "", err
	}
	for _, item := range worktrees {
		if item == nil {
			continue
		}
		ruta := strings.TrimSpace(item.Path)
		if ruta == "" {
			continue
		}
		info, err := os.Stat(ruta)
		if err != nil || !info.IsDir() {
			continue
		}
		return ruta, nil
	}
	return strings.TrimSpace(proyecto.RutaAbs), nil
}

func runtimeMailboxGuidanceDurableWorkingDir(handle *db.RuntimeHandle) (string, error) {
	if handle == nil {
		return "", nil
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		sesion, err := db.GetSesionByID(*handle.SesionID)
		if err != nil {
			return "", err
		}
		if sesion != nil {
			cwd := strings.TrimSpace(sesion.CWD)
			if cwd != "" {
				if info, err := os.Stat(cwd); err == nil && info.IsDir() {
					return cwd, nil
				}
			}
		}
	}
	if cwd := strings.TrimSpace(stringFromMetadataJSON(handle.MetadataJSON, "working_dir")); cwd != "" {
		if info, err := os.Stat(cwd); err == nil && info.IsDir() {
			return cwd, nil
		}
	}
	return "", nil
}

func construirInboxRuntimeMailboxDurableMarkdown(proyecto *db.Proyecto, tarea *db.Tarea, msg *db.RuntimeMailboxMessage) string {
	base := strings.TrimSpace(construirInboxMicrocicloMarkdown(proyecto, tarea))
	if base == "" {
		base = "# Inbox Activa de Orquesta\n"
	}
	payload := map[string]any{}
	if strings.TrimSpace(msg.PayloadJSON) != "" {
		_ = json.Unmarshal([]byte(msg.PayloadJSON), &payload)
	}
	lineas := []string{
		strings.TrimRight(base, "\n"),
		"",
		"## Guidance Durable",
		fmt.Sprintf("- Mailbox: `%d`", msg.ID),
		fmt.Sprintf("- Kind: `%s`", strings.TrimSpace(msg.Kind)),
	}
	if accion := strings.TrimSpace(stringMapValue(payload, "accion")); accion != "" {
		lineas = append(lineas, fmt.Sprintf("- Accion: `%s`", accion))
	}
	if instruction := strings.TrimSpace(stringMapValue(payload, "instruction")); instruction != "" {
		lineas = append(lineas, "", "### Instruccion vigente", instruction)
	} else if texto := strings.TrimSpace(stringMapValue(payload, "texto")); texto != "" {
		lineas = append(lineas, "", "### Contexto", texto)
	}
	lineas = append(lineas,
		"",
		"## Regla de lectura",
		"- Esta inbox durable sustituye guidance repetida en caliente para este runtime.",
		"- Si retomas trabajo tras quedar idle o tras reinicio, vuelve a leer esta inbox antes de tocar codigo.",
	)
	return strings.TrimSpace(strings.Join(lineas, "\n")) + "\n"
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

func runtimeHandleListaParaDispatchSessionResumeTMUX(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) bool {
	if handle == nil {
		return false
	}
	if handle.SesionID == nil && handle.RuntimeID == nil {
		// A fresh canonical handle can still route session_resume via the
		// runtime/session external session even before the local link is hydrated.
		return true
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
	if !ready &&
		strings.EqualFold(strings.TrimSpace(view.State), "running") &&
		runtimeTMUXSessionResumePuedeDespacharPorRuntimeCanonico(runtime) {
		return true
	}
	return ready
}

func runtimeTMUXSessionResumeWorkerReady(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON))
	if err != nil || snap == nil {
		return false
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(view.Driver), "tmux_cli_session") &&
		!strings.EqualFold(strings.TrimSpace(view.Transport), "tmux") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "idle":
		return true
	default:
		return false
	}
}

func runtimeTMUXSessionResumePuedeDespacharPorRuntimeCanonico(runtime *db.RuntimeInstance) bool {
	if runtime == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(runtime.ProcessState), "running") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
	case "esperando_io", "waiting_io":
		return true
	default:
		return false
	}
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
	payload := map[string]any{}
	if strings.TrimSpace(msg.PayloadJSON) != "" {
		_ = json.Unmarshal([]byte(msg.PayloadJSON), &payload)
	}
	payload["to_agente"] = strings.TrimSpace(msg.ToAgente)
	payload["from_agente"] = strings.TrimSpace(msg.FromAgente)
	payload["texto"] = strings.TrimSpace(texto)
	payload["mailbox_id"] = msg.ID
	payload["mailbox_kind"] = strings.TrimSpace(msg.Kind)
	payload["delivery_attempt_signature"] = runtimeMailboxDeliveryAttemptSignature(handle, externalSessionID)
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
	orderID, err := db.EncolarRuntimeOrder(order)
	if err != nil {
		return 0, err
	}
	wakeRuntimeOrdersAfterMailbox()
	return orderID, nil
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
		if !runtimeOrderMatchesMailboxSendInstructionAttempt(order, msg.ID, handle.ID) {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			return true, nil
		case "completada":
			if !runtimeOrderCompletedMailboxAttemptStillBlocks(order, currentSessionID, currentSignature) {
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
		if !runtimeOrderMatchesMailboxSendInstructionAttempt(order, msg.ID, handle.ID) {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			if !runtimeOrderSigueBloqueandoMailbox(order, time.Now().UTC()) {
				continue
			}
			return true, nil
		case "completada":
			if !runtimeOrderCompletedMailboxAttemptStillBlocks(order, currentSessionID, currentSignature) {
				continue
			}
			return true, nil
		}
	}
	return false, nil
}

func runtimeOrderMatchesMailboxSendInstructionAttempt(order *db.RuntimeOrder, mailboxID, handleID int64) bool {
	if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" || mailboxID <= 0 || handleID <= 0 {
		return false
	}
	if runtimeOrderMailboxIDFromJSON(order.PayloadJSON) != mailboxID {
		return false
	}
	if order.HandleID != nil && *order.HandleID != handleID {
		return false
	}
	return true
}

func runtimeOrderCompletedMailboxAttemptStillBlocks(order *db.RuntimeOrder, currentSessionID, currentSignature string) bool {
	if order == nil || !runtimeOrderMailboxOnlyResult(order.ResultadoJSON) || runtimeOrderMailboxOnlyExpired(order) {
		return false
	}
	orderSignature := runtimeOrderDeliveryAttemptSignature(order.PayloadJSON)
	if currentSignature != "" {
		return orderSignature != "" && currentSignature == orderSignature
	}
	orderSessionID := strings.TrimSpace(runtimeOrderExternalSessionIDFromJSON(order.PayloadJSON))
	return !(currentSessionID != "" && orderSessionID != "" && currentSessionID != orderSessionID)
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
	payload := mapFromJSON(raw)
	if payload == nil {
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
	payload := mapFromJSON(raw)
	if payload == nil {
		return ""
	}
	value, _ := payload["external_session_id"]
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func runtimeOrderDeliveryAttemptSignature(raw string) string {
	payload := mapFromJSON(raw)
	if payload == nil {
		return ""
	}
	if text, ok := payload["delivery_attempt_signature"].(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func runtimeOrderMailboxKindFromJSON(raw string) string {
	payload := mapFromJSON(raw)
	if payload == nil {
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
	payload := mapFromJSON(raw)
	if payload == nil {
		return false
	}
	value, _ := payload["mailbox_only"]
	flag, ok := value.(bool)
	return ok && flag
}

func runtimeOrderDeferredReason(raw string) string {
	payload := mapFromJSON(raw)
	if payload == nil {
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
	if detalle, obsoleta, err := runtimeMailboxObsoletaPorTareaActivaActual(msg); err != nil {
		return false, err
	} else if obsoleta {
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return false, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return false, err
		}
		db.Audit("orquesta", "runtime_mailbox_obsoleta_tarea", "runtime_mailbox", msg.ID,
			fmt.Sprintf("lane=%s agente=%s kind=%s %s", strings.TrimSpace(lane), strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), strings.TrimSpace(detalle)))
		return true, nil
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
	if view.StartedAt == nil || view.StartedAt.IsZero() {
		return false, nil
	}
	if detalle, obsoleta := runtimeMailboxObsoletaPorProgresoTareaActual(msg, view); obsoleta {
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return false, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return false, err
		}
		db.Audit("orquesta", "runtime_mailbox_obsoleta_progress", "runtime_mailbox", msg.ID,
			fmt.Sprintf("lane=%s agente=%s kind=%s %s", strings.TrimSpace(lane), strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), strings.TrimSpace(detalle)))
		return true, nil
	}
	if detalle, obsoleta := runtimeMailboxObsoletaPorProgresoEfimero(msg, view); obsoleta {
		if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return false, err
		}
		if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return false, err
		}
		db.Audit("orquesta", "runtime_mailbox_obsoleta_progress_ephemeral", "runtime_mailbox", msg.ID,
			fmt.Sprintf("lane=%s agente=%s kind=%s %s", strings.TrimSpace(lane), strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), strings.TrimSpace(detalle)))
		return true, nil
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "running", "idle", "waiting_input":
	default:
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

func runtimeMailboxObsoletaPorProgresoTareaActual(msg *db.RuntimeMailboxMessage, view *runtimeagente.WorkerStatusView) (string, bool) {
	if msg == nil || view == nil {
		return "", false
	}
	payload := mapFromJSON(strings.TrimSpace(msg.PayloadJSON))
	tareaID := int64PtrFromMap(payload, "tarea_id")
	if tareaID == nil || *tareaID <= 0 || msg.ProyectoID == nil {
		return "", false
	}
	actualID, err := db.GetTareaActivaIDPorAgenteProyecto(strings.TrimSpace(msg.ToAgente), msg.ProyectoID)
	if err != nil || actualID <= 0 || actualID != *tareaID {
		return "", false
	}
	if view.LastProgressAt != nil && !view.LastProgressAt.IsZero() {
		if progressAt := view.LastProgressAt.UTC(); progressAt.After(msg.CreatedAt.UTC()) {
			return fmt.Sprintf("task_id=%d progress_at=%s created_at=%s", *tareaID, progressAt.Format(time.RFC3339Nano), msg.CreatedAt.UTC().Format(time.RFC3339Nano)), true
		}
	}
	if view.LastOutputAt != nil && !view.LastOutputAt.IsZero() {
		if outputAt := view.LastOutputAt.UTC(); outputAt.After(msg.CreatedAt.UTC()) {
			return fmt.Sprintf("task_id=%d output_at=%s created_at=%s", *tareaID, outputAt.Format(time.RFC3339Nano), msg.CreatedAt.UTC().Format(time.RFC3339Nano)), true
		}
	}
	return "", false
}

func runtimeMailboxObsoletaPorProgresoEfimero(msg *db.RuntimeMailboxMessage, view *runtimeagente.WorkerStatusView) (string, bool) {
	if msg == nil || view == nil || view.LastProgressAt == nil || view.LastProgressAt.IsZero() {
		return "", false
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Kind), "nudge") {
		return "", false
	}
	if !view.LastProgressAt.UTC().After(msg.CreatedAt.UTC()) {
		return "", false
	}
	return fmt.Sprintf("progress_at=%s created_at=%s", view.LastProgressAt.UTC().Format(time.RFC3339Nano), msg.CreatedAt.UTC().Format(time.RFC3339Nano)), true
}

func runtimeMailboxObsoletaPorTareaActivaActual(msg *db.RuntimeMailboxMessage) (string, bool, error) {
	if msg == nil || msg.ProyectoID == nil {
		return "", false, nil
	}
	payload := mapFromJSON(strings.TrimSpace(msg.PayloadJSON))
	tareaID := int64PtrFromMap(payload, "tarea_id")
	if tareaID == nil || *tareaID <= 0 {
		return "", false, nil
	}
	tareaObjetivo, err := tareasService.Get(*tareaID)
	if err != nil {
		return "", false, err
	}
	agente := strings.TrimSpace(msg.ToAgente)
	if tareaObjetivo == nil || tareaObjetivo.Agente == nil || !strings.EqualFold(strings.TrimSpace(*tareaObjetivo.Agente), agente) {
		return fmt.Sprintf("task_id=%d target_task_invalida=true proyecto_id=%d", *tareaID, *msg.ProyectoID), true, nil
	}
	switch tareaObjetivo.Estado {
	case db.TareaAsignada, db.TareaEnProgreso:
	default:
		return fmt.Sprintf("task_id=%d target_task_estado=%s proyecto_id=%d", *tareaID, strings.TrimSpace(string(tareaObjetivo.Estado)), *msg.ProyectoID), true, nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{Agente: &agente, ProyectoID: msg.ProyectoID})
	if err != nil {
		return "", false, err
	}
	candidatas := make([]*db.Tarea, 0, len(tareas))
	for _, tarea := range tareas {
		if tareaEsFrentePremiumActivoCompactable(tarea, agente) {
			candidatas = append(candidatas, tarea)
		}
	}
	canonica := seleccionarFrentePremiumCanonicSesionActiva(candidatas)
	if canonica == nil || canonica.ID == *tareaID {
		return "", false, nil
	}
	return fmt.Sprintf("task_id=%d active_task_id=%d proyecto_id=%d", *tareaID, canonica.ID, *msg.ProyectoID), true, nil
}

func consumirRuntimeMailboxBootstrapObservadoSiProcede(msg *db.RuntimeMailboxMessage, handle *db.RuntimeHandle, runtime *db.RuntimeInstance, lane string) (bool, error) {
	if msg == nil {
		return false, nil
	}
	observado, bootstrapOrderID, startOrderID, err := db.RuntimeMailboxEntregadoPorBootstrapObservado(msg.ID, handle, runtime)
	if err != nil || !observado {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
		return false, err
	}
	if err := db.MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
		return false, err
	}
	db.Audit("orquesta", "runtime_mailbox_bootstrap_receipt_consumed", "runtime_mailbox", msg.ID,
		fmt.Sprintf("lane=%s agente=%s kind=%s bootstrap_order_id=%d start_order_id=%d", strings.TrimSpace(lane), strings.TrimSpace(msg.ToAgente), strings.TrimSpace(msg.Kind), bootstrapOrderID, startOrderID))
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
		return tmuxLike
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
		if runtimeHandleBootstrapTMUXActivo(handle) {
			return false
		}
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
			if !runtimeOrderSigueBloqueandoMailbox(order, time.Now().UTC()) {
				continue
			}
		default:
			continue
		}
		if _, ok := tiposWanted[strings.TrimSpace(order.Tipo)]; ok {
			return true, nil
		}
	}
	return false, nil
}

func runtimeOrderSigueBloqueandoMailbox(order *db.RuntimeOrder, now time.Time) bool {
	if order == nil {
		return false
	}
	switch strings.TrimSpace(order.Estado) {
	case "pendiente":
		return true
	case "tomada", "ejecutando":
		if order.LeaseExpiresAt != nil && !order.LeaseExpiresAt.IsZero() && !order.LeaseExpiresAt.UTC().After(now.UTC()) {
			return false
		}
		return true
	default:
		return false
	}
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
	case "instruction", "pipeline_local":
		texto := stringMapValue(payload, "texto")
		if strings.EqualFold(strings.TrimSpace(msg.Kind), "pipeline_local") {
			texto = stringMapValue(payload, "instruction")
			if texto == "" {
				texto = stringMapValue(payload, "texto")
			}
		} else if texto == "" {
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

func procesarPipelineLocalBatch() (int, error) {
	proyectos, err := db.ListarProyectosActivos()
	if err != nil {
		return 0, err
	}
	procesados := 0
	for _, proyecto := range proyectos {
		slug, ok := pipelineLocalProyectoElegible(proyecto)
		if !ok {
			continue
		}
		proyectoProcesado, err := procesarProyectoPipelineLocal(proyecto, slug)
		if err != nil {
			continue
		}
		if proyectoProcesado {
			procesados++
		}
	}
	if procesados > 0 {
		resetStatusSnapshotCache()
	}
	return procesados, nil
}

func pipelineLocalProyectoElegible(proyecto *db.Proyecto) (string, bool) {
	if proyecto == nil || !proyecto.Activo || proyecto.Tipo != db.ProyectoRepo {
		return "", false
	}
	slug := strings.TrimSpace(proyecto.Slug)
	if slug == "" {
		return "", false
	}
	return slug, true
}

func procesarProyectoPipelineLocal(proyecto *db.Proyecto, slug string) (bool, error) {
	if cerrado, handled := intentarCerrarProyectoPipelineLocal(proyecto, slug); handled {
		return cerrado, nil
	}
	resultado, handled := ejecutarPasoProyectoPipelineLocal(proyecto, slug)
	if !handled {
		return false, nil
	}
	return procesarResultadoPipelineLocal(proyecto, slug, resultado)
}

func intentarCerrarProyectoPipelineLocal(proyecto *db.Proyecto, slug string) (bool, bool) {
	cerrado, err := procesarCierreProyectoAutonomiaSinSesion(proyecto)
	if err != nil {
		auditarPipelineLocalProyectoError("pipeline_local_project_close_error", proyecto, slug, err)
		return false, true
	}
	if !cerrado {
		return false, false
	}
	auditarPipelineLocalProyecto("pipeline_local_project_closed", proyecto, slug)
	return true, true
}

func ejecutarPasoProyectoPipelineLocal(proyecto *db.Proyecto, slug string) (*capacidadapp.ResultadoEjecucionPasoPipelineLocal, bool) {
	resultado, err := capacidadService.EjecutarSiguientePasoPipelineLocalDeterminista(slug)
	if err != nil {
		auditarPipelineLocalProyectoError("pipeline_local_error", proyecto, slug, err)
		return nil, false
	}
	return resultado, true
}

func procesarResultadoPipelineLocal(proyecto *db.Proyecto, slug string, resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal) (bool, error) {
	if resultado == nil {
		return false, nil
	}
	proyectoProcesado := resultado.FaseActivada != nil || resultado.TareaActualizada != nil
	despachoProcesado, err := procesarDespachoPipelineLocal(proyecto, slug, resultado)
	if err != nil {
		return false, err
	}
	return proyectoProcesado || despachoProcesado, nil
}

func procesarDespachoPipelineLocal(proyecto *db.Proyecto, slug string, resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal) (bool, error) {
	if proyecto == nil || resultado == nil || !despachoPipelineRequiereRuntime(resultado.Despacho) {
		return false, nil
	}
	key := pipelineLocalDispatchKey(slug, resultado.Despacho)
	if !pipelineLocalDispatchShouldAttempt(key) {
		return false, nil
	}
	dispatchRuntime, err := capacidadService.DespacharPipelineLocal(slug, resultado.Despacho)
	if err != nil {
		auditarPipelineLocalProyectoError("pipeline_local_dispatch_error", proyecto, slug, err)
		return false, nil
	}
	resultado.DispatchRuntime = dispatchRuntime
	if !pipelineLocalDispatchEncolado(dispatchRuntime) {
		return false, nil
	}
	auditarPipelineLocalDispatch(proyecto.ID, slug, resultado.Despacho, dispatchRuntime)
	return true, nil
}

func pipelineLocalDispatchEncolado(resultado *capacidadapp.ResultadoDespachoPipeline) bool {
	return resultado != nil && strings.EqualFold(strings.TrimSpace(resultado.Estado), "encolado")
}

func auditarPipelineLocalProyecto(accion string, proyecto *db.Proyecto, slug string) {
	db.Audit("server", strings.TrimSpace(accion), "proyecto", valorProyectoIDDesdePipelineLocal(proyecto), detalleProyectoPipelineLocal(slug, nil))
}

func auditarPipelineLocalProyectoError(accion string, proyecto *db.Proyecto, slug string, err error) {
	if err == nil {
		return
	}
	db.Audit("server", strings.TrimSpace(accion), "proyecto", valorProyectoIDDesdePipelineLocal(proyecto), detalleProyectoPipelineLocal(slug, err))
}

func valorProyectoIDDesdePipelineLocal(proyecto *db.Proyecto) int64 {
	if proyecto == nil {
		return 0
	}
	return proyecto.ID
}

func detalleProyectoPipelineLocal(slug string, err error) string {
	detalle := fmt.Sprintf("proyecto=%s", strings.TrimSpace(slug))
	if err == nil {
		return detalle
	}
	return detalle + " error=" + err.Error()
}

func auditarPipelineLocalDispatch(proyectoID int64, slug string, despacho *capacidadapp.DespachoPipelineLocal, dispatchRuntime *capacidadapp.ResultadoDespachoPipeline) {
	if despacho == nil || dispatchRuntime == nil {
		return
	}
	db.Audit("server", "pipeline_local_dispatch", "proyecto", proyectoID,
		fmt.Sprintf("proyecto=%s fase=%s carril=%s agente=%s tarea_id=%d start_order_id=%d runtime_order_id=%d",
			slug,
			strings.TrimSpace(despacho.Fase),
			strings.TrimSpace(despacho.Carril),
			strings.TrimSpace(despacho.AgenteSugerido),
			despacho.TareaObjetivoID,
			valorID(dispatchRuntime.StartOrderID),
			valorID(dispatchRuntime.RuntimeOrderID),
		))
}

func procesarCierreProyectoAutonomiaSinSesion(proyecto *db.Proyecto) (bool, error) {
	if proyecto == nil || proyecto.ID <= 0 {
		return false, nil
	}
	policy, err := supervisionService.GetProjectPolicy(strings.TrimSpace(proyecto.Slug))
	if err != nil {
		return false, err
	}
	if policy == nil || !policy.Enabled || !policy.AutoCloseProject || policy.EstadoAutonomia != db.AutonomiaProyectoCerrando {
		return false, nil
	}
	activa := true
	proyectoID := proyecto.ID
	sesiones, err := sesionesAPIService.ListInspectionSessions(db.FiltroSesionesInspeccion{
		ProyectoID: &proyectoID,
		Activa:     &activa,
		Limit:      10,
	})
	if err != nil {
		return false, err
	}
	if len(sesiones) > 0 {
		return false, nil
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil || !terminado {
		return false, err
	}
	if err := cerrarProyectoAutonomia(proyecto.ID, motivo, db.AutonomiaProyectoCerrado, policy); err != nil {
		return false, err
	}
	return true, nil
}

func valorID(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}

func procesarAutonomiaAgentesBatch() (int, error) {
	start := time.Now()
	activa := true
	sessionsStart := time.Now()
	sesiones, err := sesionesAPIService.ListInspectionSessions(db.FiltroSesionesInspeccion{Activa: &activa})
	if err != nil {
		return 0, err
	}
	sesiones = filtrarSesionesAutonomiaRelevantes(sesiones)
	autonomiaTickDebugf("listar_sesiones sesiones=%d duration=%s", len(sesiones), time.Since(sessionsStart).Round(time.Millisecond))
	snapshotStart := time.Now()
	snapshot, err := newAutonomiaBatchSnapshot(sesiones)
	if err != nil {
		return 0, err
	}
	autonomiaTickDebugf("snapshot agentes=%d duration=%s", len(snapshot.agentesByName), time.Since(snapshotStart).Round(time.Millisecond))
	procesadas := 0
	sesionesProcesadas := 0
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		agente := strings.TrimSpace(sesion.Agente)
		if agente == "" {
			continue
		}
		sesionStart := time.Now()
		n, err := ejecutarPasoAutonomiaConTimeout(
			fmt.Sprintf("sesion_activa:%s:%d", agente, valorProyectoID(sesion.ProyectoID)),
			autonomiaSessionStepTimeout(),
			func() (int, error) { return procesarAutonomiaSesionActivaBatchFn(sesion, snapshot) },
		)
		if err != nil {
			db.Audit("server", "autonomia_agente_error", "agente", 0, fmt.Sprintf("agente=%s error=%s", agente, err.Error()))
			continue
		}
		procesadas += n
		sesionesProcesadas++
		autonomiaTickDebugf("agente=%s proyecto_id=%d duration=%s procesadas=%d", agente, valorProyectoID(sesion.ProyectoID), time.Since(sesionStart).Round(time.Millisecond), n)
	}
	degradadosStart := time.Now()
	n, err := ejecutarPasoAutonomiaConTimeout(
		"agentes_degradados",
		autonomiaDegradedBatchTimeout(),
		func() (int, error) { return procesarAgentesDegradadosAutonomiaBatchFn() },
	)
	if err != nil {
		db.Audit("server", "autonomia_agentes_degradados_error", "runtime", 0, err.Error())
		autonomiaTickDebugf("agentes_degradados duration=%s procesadas=%d", time.Since(degradadosStart).Round(time.Millisecond), 0)
		if procesadas > 0 {
			resetStatusSnapshotCache()
		}
		return procesadas, nil
	}
	procesadas += n
	autonomiaTickDebugf("agentes_degradados duration=%s procesadas=%d", time.Since(degradadosStart).Round(time.Millisecond), n)
	autonomiaTickDebugf("batch sesiones=%d procesadas=%d duration=%s acciones=%d", len(sesiones), sesionesProcesadas, time.Since(start).Round(time.Millisecond), procesadas)
	if procesadas > 0 {
		resetStatusSnapshotCache()
	}
	return procesadas, nil
}

func filtrarSesionesAutonomiaRelevantes(sesiones []*db.Sesion) []*db.Sesion {
	if len(sesiones) == 0 {
		return nil
	}
	type clave struct {
		agente   string
		proyecto int64
	}
	mejores := map[clave]*db.Sesion{}
	for _, sesion := range sesiones {
		if !sesionAutonomiaRelevante(sesion) {
			continue
		}
		key := clave{
			agente:   strings.ToLower(strings.TrimSpace(sesion.Agente)),
			proyecto: valorProyectoID(sesion.ProyectoID),
		}
		actual := mejores[key]
		if actual == nil || sesionAutonomiaMasReciente(sesion, actual) {
			mejores[key] = sesion
		}
	}
	out := make([]*db.Sesion, 0, len(mejores))
	for _, sesion := range mejores {
		out = append(out, sesion)
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai := strings.ToLower(strings.TrimSpace(out[i].Agente))
		aj := strings.ToLower(strings.TrimSpace(out[j].Agente))
		if ai != aj {
			return ai < aj
		}
		pi := valorProyectoID(out[i].ProyectoID)
		pj := valorProyectoID(out[j].ProyectoID)
		if pi != pj {
			return pi < pj
		}
		if !out[i].Inicio.Equal(out[j].Inicio) {
			return out[i].Inicio.After(out[j].Inicio)
		}
		return out[i].ID > out[j].ID
	})
	return out
}

func sesionAutonomiaRelevante(sesion *db.Sesion) bool {
	if sesion == nil || !sesion.Activa || sesion.ProyectoID == nil || *sesion.ProyectoID <= 0 {
		return false
	}
	if strings.TrimSpace(sesion.Agente) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(sesion.Estado)) {
	case "", "activa", "active", "pausada", "paused":
		return true
	default:
		return false
	}
}

func sesionAutonomiaMasReciente(candidata, actual *db.Sesion) bool {
	if actual == nil {
		return true
	}
	if candidata == nil {
		return false
	}
	if !candidata.Inicio.Equal(actual.Inicio) {
		return candidata.Inicio.After(actual.Inicio)
	}
	return candidata.ID > actual.ID
}

func ejecutarPasoAutonomiaConTimeout(etiqueta string, timeout time.Duration, fn func() (int, error)) (int, error) {
	if fn == nil {
		return 0, nil
	}
	if timeout <= 0 {
		return fn()
	}
	type resultado struct {
		count int
		err   error
	}
	done := make(chan resultado, 1)
	go func() {
		count, err := fn()
		done <- resultado{count: count, err: err}
	}()
	select {
	case out := <-done:
		return out.count, out.err
	case <-time.After(timeout):
		return 0, fmt.Errorf("%s timeout tras %s", strings.TrimSpace(etiqueta), timeout)
	}
}

type runtimeProcessDegradadosSummary struct {
	Count                     int
	GhostAssignmentsCompacted int
	ReactivatedWithoutRuntime int
	IdleAutoassigned          int
}

func procesarAgentesDegradadosAutonomiaBatch() (int, error) {
	resumen, err := procesarAgentesDegradadosAutonomiaBatchDetallado()
	if err != nil {
		return 0, err
	}
	return resumen.Count, nil
}

func procesarAgentesDegradadosAutonomiaBatchDetallado() (runtimeProcessDegradadosSummary, error) {
	db.ResetRuntimeHandlesHotCache()
	rows, err := agentesService.BuildPanelRows()
	if err != nil {
		return runtimeProcessDegradadosSummary{}, err
	}
	tareasActivasPorAgente, tareasBloqueadasPorAgente, bloqueosPorTarea, err := cargarTareasYBloqueosAutonomiaPorAgente()
	if err != nil {
		return runtimeProcessDegradadosSummary{}, err
	}
	rowsPorAgente := rowsPorAgenteMap(rows)
	openTasksProjected := openTasksProjectedFromRows(rows)

	resumen := runtimeProcessDegradadosSummary{}
	now := time.Now().UTC()
	agentesFantasma := agentesAsignacionFantasmaCompactable(rows, tareasActivasPorAgente, tareasBloqueadasPorAgente)
	asignacionesFantasma, err := procesarAsignacionesPremiumFantasmaBatch(rows, tareasActivasPorAgente, tareasBloqueadasPorAgente)
	if err != nil {
		return resumen, err
	}
	resumen.GhostAssignmentsCompacted = asignacionesFantasma
	resumen.Count += asignacionesFantasma
	if asignacionesFantasma > 0 {
		rows, err = refrescarRowsCompactPorAgente(rows, agentesFantasma)
		if err != nil {
			return resumen, err
		}
		rowsPorAgente = rowsPorAgenteMap(rows)
		openTasksProjected = openTasksProjectedFromRows(rows)
	}
	migradas, err := procesarMigracionRuntimeLegacyTMUXBatch(rows, now)
	if err != nil {
		return resumen, err
	}
	resumen.Count += migradas
	huerfanasTMUX, err := procesarSesionesTMUXHuerfanasAutonomiaBatch(now)
	if err != nil {
		return resumen, err
	}
	resumen.Count += huerfanasTMUX
	agentesAtascados := agentesWorkersAtascadosRecuperables(rows, now)
	reiniciosAtascados, err := procesarWorkersAtascadosAutonomiaBatch(rows, now)
	if err != nil {
		return resumen, err
	}
	resumen.Count += reiniciosAtascados
	if reiniciosAtascados > 0 {
		rows, err = refrescarRowsCompactPorAgente(rows, agentesAtascados)
		if err != nil {
			return resumen, err
		}
		rowsPorAgente = rowsPorAgenteMap(rows)
		openTasksProjected = openTasksProjectedFromRows(rows)
	}
	limpiadasFuera, err := procesarTareasActivasFueraDeOrquestacionBatch(tareasActivasPorAgente, rowsPorAgente)
	if err != nil {
		return resumen, err
	}
	resumen.Count += limpiadasFuera
	compactadasExclusivas, err := procesarCompactacionExclusividadPremiumBatch(rows, tareasActivasPorAgente)
	if err != nil {
		return resumen, err
	}
	resumen.Count += compactadasExclusivas
	runtimesFueraAsignacion, err := procesarRuntimesFueraDeAsignacionActivaBatch(rows, now)
	if err != nil {
		return resumen, err
	}
	resumen.Count += runtimesFueraAsignacion
	redistribuidas, err := procesarSobrecargaAgentesAutonomiaBatch(rows, tareasActivasPorAgente, openTasksProjected, now)
	if err != nil {
		return resumen, err
	}
	resumen.Count += redistribuidas
	agentesSinRuntime := agentesSinRuntimeReactivables(rows, tareasActivasPorAgente, tareasBloqueadasPorAgente)
	reactivadosSinRuntime, err := procesarReactivacionAgentesSinRuntimeBatch(rows, tareasActivasPorAgente, tareasBloqueadasPorAgente)
	if err != nil {
		return resumen, err
	}
	resumen.ReactivatedWithoutRuntime = reactivadosSinRuntime
	resumen.Count += reactivadosSinRuntime
	if reactivadosSinRuntime > 0 {
		rows, err = refrescarRowsCompactPorAgente(rows, agentesSinRuntime)
		if err != nil {
			return resumen, err
		}
		rowsPorAgente = rowsPorAgenteMap(rows)
		tareasActivasPorAgente, tareasBloqueadasPorAgente, bloqueosPorTarea, err = cargarTareasYBloqueosAutonomiaPorAgente()
		if err != nil {
			return resumen, err
		}
		openTasksProjected = openTasksProjectedFromRows(rows)
	}
	autoasignadosPremiumSinSesion, err := procesarAutoasignacionPremiumSinSesionBatch(rows, tareasActivasPorAgente)
	if err != nil {
		return resumen, err
	}
	resumen.IdleAutoassigned = autoasignadosPremiumSinSesion
	resumen.Count += autoasignadosPremiumSinSesion
	recuperadasSobrecarga, err := procesarRecuperacionTareasBloqueadasSobrecargaBatch(rows, tareasBloqueadasPorAgente, bloqueosPorTarea, openTasksProjected, now)
	if err != nil {
		return resumen, err
	}
	resumen.Count += recuperadasSobrecarga
	intervencionActivas, err := procesarIntervencionTareasActivasDegradadasBatch(rows, tareasActivasPorAgente, openTasksProjected, now)
	if err != nil {
		return resumen, err
	}
	resumen.Count += intervencionActivas
	intervencionBloqueadas, err := procesarIntervencionTareasBloqueadasDegradadasBatch(rows, tareasBloqueadasPorAgente, bloqueosPorTarea, openTasksProjected, now)
	if err != nil {
		return resumen, err
	}
	resumen.Count += intervencionBloqueadas
	if resumen.Count > 0 {
		resetStatusSnapshotCache()
	}
	return resumen, nil
}

// procesarRecuperacionTareasBloqueadasSobrecargaBatch reactivca o reasigna tareas bloqueadas
// de agentes que ya están operativos pero tenían bloqueos por sobrecarga.
func procesarRecuperacionTareasBloqueadasSobrecargaBatch(rows []agentesapp.Row, tareasBloqueadasPorAgente map[string][]*db.Tarea, bloqueosPorTarea map[int64]db.ResumenBloqueo, openTasksProjected map[string]int, now time.Time) (int, error) {
	count := 0
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
				return count, err
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
				if err := desbloquearYReasignarTareaAutonomia(actual.ID, resolucion, relevo, fmt.Sprintf("Reasignada automáticamente desde %s a %s tras bloqueo por sobrecarga", agente, relevo)); err != nil {
					return count, err
				}
				openTasksProjected[relevo]++
				count++
				continue
			}
			resolucion := fmt.Sprintf("recuperación automática tras %s (%s)", row.EstadoOperativo, firstNonEmpty(strings.TrimSpace(row.DetalleOperativo), "worker recuperado"))
			if err := desbloquearYReactivarTareaAutonomia(actual.ID, resolucion, agente, fmt.Sprintf("Reactivada automáticamente en %s tras recuperar estado operativo", agente)); err != nil {
				return count, err
			}
			openTasksProjected[agente]++
			count++
		}
	}
	return count, nil
}

// procesarIntervencionTareasActivasDegradadasBatch reasigna o bloquea tareas activas
// cuyo agente está en un estado operativo que requiere intervención automática.
func procesarIntervencionTareasActivasDegradadasBatch(rows []agentesapp.Row, tareasActivasPorAgente map[string][]*db.Tarea, openTasksProjected map[string]int, now time.Time) (int, error) {
	count := 0
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
				return count, err
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
				if err := bloquearTareaAutonomiaSinRelevo(actual.ID, agente, motivo, construirNotaBloqueoAutonomiaSinRelevo(row, agente)); err != nil {
					return count, err
				}
				count++
				if openTasksProjected[agente] > 0 {
					openTasksProjected[agente]--
				}
				continue
			}

			if err := reasignarYArrancarTareaAutonomia(actual.ID, relevo, fmt.Sprintf("Reasignada automáticamente desde %s a %s: %s", agente, relevo, motivo)); err != nil {
				return count, err
			}
			if openTasksProjected[agente] > 0 {
				openTasksProjected[agente]--
			}
			openTasksProjected[relevo]++
			count++

			if err := encolarContinuacionTareaReasignadaSiCorresponde(relevo, actual.ProyectoID, actual.ID, agente, row.EstadoOperativo, "continúa con la tarea reasignada y deja evidencia de avance", nil); err != nil {
				return count, err
			}
		}
	}
	return count, nil
}

// procesarIntervencionTareasBloqueadasDegradadasBatch reasigna tareas bloqueadas
// cuyo agente está degradado y hay un relevo disponible.
func procesarIntervencionTareasBloqueadasDegradadasBatch(rows []agentesapp.Row, tareasBloqueadasPorAgente map[string][]*db.Tarea, bloqueosPorTarea map[int64]db.ResumenBloqueo, openTasksProjected map[string]int, now time.Time) (int, error) {
	count := 0
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
				return count, err
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
			if err := desbloquearYReasignarTareaAutonomia(actual.ID, resolucion, relevo, fmt.Sprintf("Reasignada automáticamente desde %s a %s: %s", agente, relevo, motivo)); err != nil {
				return count, err
			}
			openTasksProjected[relevo]++
			count++

			if err := encolarContinuacionTareaReasignadaSiCorresponde(relevo, actual.ProyectoID, actual.ID, agente, row.EstadoOperativo, "continúa con la tarea reasignada y deja evidencia de avance", nil); err != nil {
				return count, err
			}
		}
	}
	return count, nil
}

func rowsPorAgenteMap(rows []agentesapp.Row) map[string]agentesapp.Row {
	out := map[string]agentesapp.Row{}
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		nombre := strings.ToLower(strings.TrimSpace(row.Agente.Nombre))
		if nombre == "" {
			continue
		}
		out[nombre] = row
	}
	return out
}

func refrescarRowsCompactPorAgente(rows []agentesapp.Row, agentes map[string]struct{}) ([]agentesapp.Row, error) {
	if len(agentes) == 0 {
		return rows, nil
	}
	out := append([]agentesapp.Row(nil), rows...)
	for i, row := range out {
		if row.Agente == nil {
			continue
		}
		nombre := strings.TrimSpace(row.Agente.Nombre)
		if nombre == "" {
			continue
		}
		if _, ok := agentes[strings.ToLower(nombre)]; !ok {
			continue
		}
		detail, err := agentesService.BuildDetailCompact(nombre)
		if err != nil {
			return nil, err
		}
		if detail == nil || detail.Row.Agente == nil {
			continue
		}
		out[i] = detail.Row
	}
	return out, nil
}

func agentesAsignacionFantasmaCompactable(rows []agentesapp.Row, tareasActivasPorAgente map[string][]*db.Tarea, tareasBloqueadasPorAgente map[string][]*db.Tarea) map[string]struct{} {
	out := map[string]struct{}{}
	for _, row := range rows {
		if row.Agente == nil || row.Asignacion == nil || row.Asignacion.Estado != db.AsignacionActiva {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" || !asignacionPremiumFantasmaCompactable(row.Asignacion.Nota) {
			continue
		}
		if rowTieneRuntimeOHandleOperativo(row) || row.MailboxPending > 0 || row.OpenTasks > 0 || row.BlockedTasks > 0 {
			continue
		}
		if len(tareasActivasPorAgente[agente]) > 0 || len(tareasBloqueadasPorAgente[agente]) > 0 {
			continue
		}
		out[strings.ToLower(agente)] = struct{}{}
	}
	return out
}

func agentesWorkersAtascadosRecuperables(rows []agentesapp.Row, now time.Time) map[string]struct{} {
	out := map[string]struct{}{}
	for _, row := range rows {
		if row.Agente == nil || strings.TrimSpace(row.EstadoOperativo) != "atascado" {
			continue
		}
		if !row.WorkerFresh(now) || row.OpenTasks <= 0 || row.MailboxPending > 0 {
			continue
		}
		handle := row.Handle
		if handle == nil || !rowHasFreshTMUXWorkerForRecovery(row, now) {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(row.Agente.Nombre))] = struct{}{}
	}
	return out
}

func agentesSinRuntimeReactivables(rows []agentesapp.Row, tareasActivasPorAgente map[string][]*db.Tarea, tareasBloqueadasPorAgente map[string][]*db.Tarea) map[string]struct{} {
	out := map[string]struct{}{}
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		switch strings.TrimSpace(row.EstadoOperativo) {
		case "bloqueado_por_runtime", "caido":
		default:
			continue
		}
		if rowTieneRuntimeOHandleOperativo(row) {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" {
			continue
		}
		if row.MailboxPending <= 0 && len(tareasActivasPorAgente[agente]) == 0 && len(tareasBloqueadasPorAgente[agente]) == 0 {
			continue
		}
		out[strings.ToLower(agente)] = struct{}{}
	}
	return out
}

func asignacionPausadaPremiumAutoasignable(nota string) bool {
	switch strings.TrimSpace(nota) {
	case "sin_trabajo_reactivacion_automatica", "sin_trabajo_espera_automatica":
		return true
	default:
		return false
	}
}

func cargarTareasYBloqueosAutonomiaPorAgente() (map[string][]*db.Tarea, map[string][]*db.Tarea, map[int64]db.ResumenBloqueo, error) {
	tareas, err := db.ListarTareas(db.FiltroTareas{})
	if err != nil {
		return nil, nil, nil, err
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
		return nil, nil, nil, err
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
	return tareasActivasPorAgente, tareasBloqueadasPorAgente, bloqueosPorTarea, nil
}

func procesarAutoasignacionPremiumSinSesionBatch(rows []agentesapp.Row, tareasActivasPorAgente map[string][]*db.Tarea) (int, error) {
	procesadas := 0
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" || row.OpenTasks > 0 || row.MailboxPending > 0 || len(tareasActivasPorAgente[agente]) > 0 {
			continue
		}
		if rowTieneRuntimeUtilParaAutoasignacionPremium(row) {
			continue
		}
		if disponible, _, err := autonomiaCuentaCompartidaDisponible(agente); err != nil {
			return procesadas, err
		} else if !disponible {
			continue
		}
		proyecto, err := resolverProyectoAutoasignacionPremiumSinSesion(agente, row)
		if err != nil {
			return procesadas, err
		}
		if proyecto == nil || proyecto.ID <= 0 {
			continue
		}
		if !proyectoUsaContinuidadMicrocicloPremium(proyecto.ID) {
			continue
		}
		tarea, err := capacidadService.IntentarAutoasignarTareaPipelineLocal(proyecto.Slug, agente)
		if err != nil {
			return procesadas, err
		}
		if tarea == nil {
			tarea, err = asegurarFrentePremiumAgenteIdle(row.Agente, proyecto)
			if err != nil {
				return procesadas, err
			}
		}
		if tarea == nil {
			continue
		}
		if err := cerrarSemillaPremiumSiDerivaAFrenteAcotado(agente, proyecto.ID, tarea); err != nil {
			return procesadas, err
		}
		reactivado, err := encolarReactivacionAgenteProyectoSiProcede(agente, proyecto, "premium_idle_autoassigned")
		if err != nil {
			return procesadas, err
		}
		if !reactivado {
			continue
		}
		db.Audit("orquesta", "autonomia_premium_idle_autoassigned", "proyecto", proyecto.ID,
			fmt.Sprintf("agente=%s tarea_id=%d detalle=%s", agente, tarea.ID, strings.TrimSpace(row.DetalleOperativo)))
		procesadas++
	}
	return procesadas, nil
}

func resolverProyectoAutoasignacionPremiumSinSesion(agente string, row agentesapp.Row) (*db.Proyecto, error) {
	if proyectoID := rowProyectoIDPreferido(row); proyectoID != nil && *proyectoID > 0 {
		return runtimesService.GetProject(strconv.FormatInt(*proyectoID, 10))
	}
	if proyecto, err := resolverProyectoAutoasignacionDesdeAsignacionPausada(agente); err != nil {
		return nil, err
	} else if proyecto != nil {
		return proyecto, nil
	}
	if proyecto, err := resolverProyectoReactivacionAgente(agente); err != nil {
		return nil, err
	} else if proyecto != nil {
		return proyecto, nil
	}
	return nil, nil
}

func resolverProyectoAutoasignacionDesdeAsignacionPausada(agente string) (*db.Proyecto, error) {
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: &agente})
	if err != nil {
		return nil, err
	}
	var (
		best         *db.Asignacion
		bestProyecto *db.Proyecto
		bestScore    = -1
	)
	for _, asignacion := range asignaciones {
		if asignacion == nil || asignacion.ProyectoID <= 0 {
			continue
		}
		if asignacion.Estado != db.AsignacionPausada {
			continue
		}
		switch strings.TrimSpace(asignacion.Nota) {
		case "sin_trabajo_reactivacion_automatica", "sin_trabajo_espera_automatica":
		default:
			continue
		}
		proyecto, err := runtimesService.GetProject(strconv.FormatInt(asignacion.ProyectoID, 10))
		if err != nil {
			return nil, err
		}
		if proyecto == nil {
			continue
		}
		score := 0
		if proyectoUsaContinuidadMicrocicloPremium(proyecto.ID) {
			score += 100
		}
		if strings.EqualFold(strings.TrimSpace(asignacion.Nota), "sin_trabajo_reactivacion_automatica") {
			score += 10
		}
		if best == nil || score > bestScore || (score == bestScore && asignacion.ID > best.ID) {
			best = asignacion
			bestProyecto = proyecto
			bestScore = score
		}
	}
	if best == nil {
		return nil, nil
	}
	return bestProyecto, nil
}

func rowTieneRuntimeUtilParaAutoasignacionPremium(row agentesapp.Row) bool {
	if row.Runtime != nil {
		return true
	}
	if row.Handle == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(row.Handle.Estado)) {
	case "activo", "active", "pausado", "paused":
		return true
	default:
		return false
	}
}

func asegurarFrentePremiumAgenteIdle(agente *db.Agente, proyecto *db.Proyecto) (*capacidadapp.TareaPipelineLocal, error) {
	if agente == nil || proyecto == nil {
		return nil, nil
	}
	policy, err := supervisionService.GetProjectPolicy(strings.TrimSpace(proyecto.Slug))
	if err != nil || policy == nil || !policy.Enabled {
		return nil, err
	}
	res, err := asegurarTrabajoContinuoPremium(policy, proyecto, agente)
	if err != nil || res.SeedTaskID <= 0 {
		return nil, err
	}
	tarea, err := tareasService.Get(res.SeedTaskID)
	if err != nil || tarea == nil {
		return nil, err
	}
	return tareaPipelineLocalDesdeTareaDB(tarea), nil
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
			if err := bloquearTareaAutonomiaSinRelevo(actual.ID, strings.TrimSpace(agente), motivo, fmt.Sprintf("Bloqueada automáticamente: %s", motivo)); err != nil {
				return procesadas, err
			}
			procesadas++
		}
	}
	return procesadas, nil
}

func procesarCompactacionExclusividadPremiumBatch(rows []agentesapp.Row, tareasActivasPorAgente map[string][]*db.Tarea) (int, error) {
	procesadas := 0
	for _, row := range rows {
		if row.Agente == nil || row.Asignacion == nil {
			continue
		}
		if row.Asignacion.Estado != db.AsignacionActiva || !asignacionMantieneExclusividadPremium(row.Asignacion.Nota) {
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
			if !tareaDebeCompactarsePorExclusividadPremium(actual, row.Asignacion.ProyectoID, agente) {
				continue
			}
			if err := tareasService.MoveToBacklog(actual.ID); err != nil {
				return procesadas, err
			}
			procesadas++
		}
	}
	if procesadas > 0 {
		resetStatusSnapshotCache()
	}
	return procesadas, nil
}

func procesarRuntimesFueraDeAsignacionActivaBatch(rows []agentesapp.Row, now time.Time) (int, error) {
	procesadas := 0
	for _, row := range rows {
		if !rowRuntimeFueraDeAsignacionActivaDebePararse(row, now) {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		proyectoActual := rowProyectoIDActual(row)
		if agente == "" || proyectoActual == nil || *proyectoActual <= 0 {
			continue
		}
		if pendiente, err := existeRuntimeOrderAbiertaAutonomia(agente, proyectoActual, "stop", "pause", "restart", "resume"); err != nil {
			return procesadas, err
		} else if pendiente {
			continue
		}
		if reciente, err := existeRuntimeOrderAutonomiaReciente(agente, proyectoActual, "stop", "stop", 10*time.Minute); err != nil {
			return procesadas, err
		} else if reciente {
			continue
		}
		proyecto, err := runtimesService.GetProject(strconv.FormatInt(*proyectoActual, 10))
		if err != nil {
			return procesadas, err
		}
		if proyecto == nil {
			continue
		}
		motivo := fmt.Sprintf("La asignación activa del agente ha cambiado al proyecto %s", strings.TrimSpace(row.Asignacion.ProyectoSlug))
		if err := encolarControlAutonomiaProyecto(agente, proyecto, agenteControlAccionStop, motivo); err != nil {
			return procesadas, err
		}
		procesadas++
	}
	adicionales, err := procesarRuntimesFueraDeAsignacionActivaPorAsignacionBatch()
	if err != nil {
		return procesadas, err
	}
	return procesadas + adicionales, nil
}

func procesarRuntimesFueraDeAsignacionActivaPorAsignacionBatch() (int, error) {
	estado := db.AsignacionActiva
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Estado: &estado})
	if err != nil {
		return 0, err
	}
	procesadas := 0
	dedup := map[string]struct{}{}
	for _, asignacion := range asignaciones {
		if asignacion == nil || asignacion.ProyectoID <= 0 {
			continue
		}
		agente := strings.TrimSpace(asignacion.Agente)
		if agente == "" {
			continue
		}
		runtimes, err := db.ListarRuntimes(db.FiltroRuntimes{Agente: &agente})
		if err != nil {
			return procesadas, err
		}
		for _, runtime := range runtimes {
			if runtime == nil || runtime.ProyectoID == nil || *runtime.ProyectoID <= 0 || *runtime.ProyectoID == asignacion.ProyectoID {
				continue
			}
			if !runtimeEstaOperativoParaLimpieza(runtime) {
				continue
			}
			key := strings.ToLower(agente) + "|" + strconv.FormatInt(*runtime.ProyectoID, 10)
			if _, ok := dedup[key]; ok {
				continue
			}
			ok, err := encolarStopRuntimeFueraDeAsignacionActiva(agente, *runtime.ProyectoID, strings.TrimSpace(asignacion.ProyectoSlug))
			if err != nil {
				return procesadas, err
			}
			if ok {
				dedup[key] = struct{}{}
				procesadas++
			}
		}
		handles, err := db.ListarRuntimeHandles(&agente)
		if err != nil {
			return procesadas, err
		}
		for _, handle := range handles {
			if handle == nil || handle.ProyectoID == nil || *handle.ProyectoID <= 0 || *handle.ProyectoID == asignacion.ProyectoID {
				continue
			}
			if !handleEstaOperativoParaLimpieza(handle) {
				continue
			}
			key := strings.ToLower(agente) + "|" + strconv.FormatInt(*handle.ProyectoID, 10)
			if _, ok := dedup[key]; ok {
				continue
			}
			ok, err := encolarStopRuntimeFueraDeAsignacionActiva(agente, *handle.ProyectoID, strings.TrimSpace(asignacion.ProyectoSlug))
			if err != nil {
				return procesadas, err
			}
			if ok {
				dedup[key] = struct{}{}
				procesadas++
			}
		}
	}
	return procesadas, nil
}

func encolarStopRuntimeFueraDeAsignacionActiva(agente string, proyectoID int64, proyectoAsignado string) (bool, error) {
	if agente == "" || proyectoID <= 0 {
		return false, nil
	}
	proyectoIDPtr := proyectoID
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(agente, &proyectoIDPtr, "stop", "pause", "restart", "resume"); err != nil {
		return false, err
	} else if pendiente {
		return false, nil
	}
	if reciente, err := existeRuntimeOrderAutonomiaReciente(agente, &proyectoIDPtr, "stop", "stop", 10*time.Minute); err != nil {
		return false, err
	} else if reciente {
		return false, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		return false, err
	}
	if proyecto == nil {
		return false, nil
	}
	motivo := fmt.Sprintf("La asignación activa del agente ha cambiado al proyecto %s", strings.TrimSpace(proyectoAsignado))
	if err := encolarControlAutonomiaProyecto(agente, proyecto, agenteControlAccionStop, motivo); err != nil {
		return false, err
	}
	return true, nil
}

func runtimeEstaOperativoParaLimpieza(runtime *db.RuntimeInstance) bool {
	if runtime == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
	case "activo", "active", "running", "iniciando", "starting", "esperando_io", "waiting_io", "pausado", "paused":
		return true
	default:
		return false
	}
}

func handleEstaOperativoParaLimpieza(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
	case "activo", "active", "pausado", "paused":
		return true
	default:
		return false
	}
}

func rowRuntimeFueraDeAsignacionActivaDebePararse(row agentesapp.Row, now time.Time) bool {
	if row.Agente == nil || row.Asignacion == nil || row.Asignacion.Estado != db.AsignacionActiva {
		return false
	}
	proyectoActual := rowProyectoIDActual(row)
	if proyectoActual == nil || *proyectoActual <= 0 || *proyectoActual == row.Asignacion.ProyectoID {
		return false
	}
	if row.Handle != nil {
		switch strings.ToLower(strings.TrimSpace(row.Handle.Estado)) {
		case "activo", "active", "pausado", "paused":
			return true
		}
	}
	if row.Runtime != nil {
		switch strings.ToLower(strings.TrimSpace(row.Runtime.LogicalState)) {
		case "activo", "active", "running", "iniciando", "starting", "esperando_io", "waiting_io", "pausado", "paused":
			return true
		}
	}
	return row.WorkerFresh(now)
}

func rowProyectoIDActual(row agentesapp.Row) *int64 {
	for _, id := range []*int64{
		rowHandleProyectoID(row.Handle),
		rowRuntimeProyectoID(row.Runtime),
		rowSesionProyectoID(row.Sesion),
	} {
		if id != nil && *id > 0 {
			return id
		}
	}
	return nil
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
			if errorMigracionRuntimeLegacyTMUXIgnorable(err) {
				continue
			}
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

func errorMigracionRuntimeLegacyTMUXIgnorable(err error) bool {
	if err == nil {
		return true
	}
	raw := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case raw == "":
		return true
	case strings.Contains(raw, "can't find session"):
		return true
	case strings.Contains(raw, "no server running"):
		return true
	case strings.Contains(raw, "failed to connect to server"):
		return true
	default:
		return false
	}
}

func procesarAsignacionesPremiumFantasmaBatch(rows []agentesapp.Row, tareasActivasPorAgente map[string][]*db.Tarea, tareasBloqueadasPorAgente map[string][]*db.Tarea) (int, error) {
	procesadas := 0
	for _, row := range rows {
		if row.Agente == nil || row.Asignacion == nil {
			continue
		}
		if row.Asignacion.Estado != db.AsignacionActiva {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" {
			continue
		}
		if !asignacionPremiumFantasmaCompactable(row.Asignacion.Nota) {
			continue
		}
		if rowTieneRuntimeOHandleOperativo(row) || row.MailboxPending > 0 {
			continue
		}
		if row.OpenTasks > 0 || row.BlockedTasks > 0 {
			continue
		}
		if len(tareasActivasPorAgente[agente]) > 0 || len(tareasBloqueadasPorAgente[agente]) > 0 {
			continue
		}
		if row.Asignacion.ProyectoID <= 0 {
			continue
		}
		if err := db.PausarAsignacion(agente, row.Asignacion.ProyectoID, "sin_trabajo_reactivacion_automatica"); err != nil {
			return procesadas, err
		}
		db.Audit("orquesta", "autonomia_compacta_asignacion_premium_fantasma", "proyecto", row.Asignacion.ProyectoID,
			fmt.Sprintf("agente=%s nota=%s estado_operativo=%s", agente, strings.TrimSpace(row.Asignacion.Nota), strings.TrimSpace(row.EstadoOperativo)))
		procesadas++
	}
	return procesadas, nil
}

func asignacionPremiumFantasmaCompactable(nota string) bool {
	switch strings.ToLower(strings.TrimSpace(nota)) {
	case "microciclo_exclusivo", "reactivacion_automatica_trabajo_activo", "reactivacion_automatica":
		return true
	default:
		return false
	}
}

func procesarReactivacionAgentesSinRuntimeBatch(rows []agentesapp.Row, tareasActivasPorAgente map[string][]*db.Tarea, tareasBloqueadasPorAgente map[string][]*db.Tarea) (int, error) {
	resumenBloqueos, err := db.ListarResumenBloqueos()
	if err != nil {
		return 0, err
	}
	bloqueosPorTarea := map[int64]db.ResumenBloqueo{}
	for _, bloqueo := range resumenBloqueos {
		if bloqueo.ID > 0 {
			bloqueosPorTarea[bloqueo.ID] = bloqueo
		}
	}
	procesadas := 0
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		switch strings.TrimSpace(row.EstadoOperativo) {
		case "bloqueado_por_runtime", "caido":
		default:
			continue
		}
		if rowTieneRuntimeOHandleOperativo(row) {
			continue
		}
		agente := strings.TrimSpace(row.Agente.Nombre)
		if agente == "" {
			continue
		}
		if row.MailboxPending <= 0 && len(tareasActivasPorAgente[agente]) == 0 && len(tareasBloqueadasPorAgente[agente]) == 0 {
			continue
		}
		if disponible, _, err := autonomiaCuentaCompartidaDisponible(agente); err != nil {
			return procesadas, err
		} else if !disponible {
			continue
		}
		proyectoID := rowProyectoIDPreferido(row)
		var proyecto *db.Proyecto
		if proyectoID != nil && *proyectoID > 0 {
			proyecto, err = runtimesService.GetProject(strconv.FormatInt(*proyectoID, 10))
			if err != nil {
				return procesadas, err
			}
		} else {
			proyecto, err = resolverProyectoReactivacionAgente(agente)
			if err != nil {
				return procesadas, err
			}
		}
		if proyecto == nil || proyecto.ID <= 0 {
			continue
		}
		reactivado, err := encolarReactivacionAgenteProyectoSiProcede(agente, proyecto, "agente_sin_runtime_activo")
		if err != nil {
			return procesadas, err
		}
		if !reactivado {
			continue
		}
		desbloqueadas, err := desbloquearTareasBloqueadasRecuperablesSinRuntime(agente, proyecto.ID, tareasBloqueadasPorAgente[agente], bloqueosPorTarea)
		if err != nil {
			return procesadas, err
		}
		db.Audit("orquesta", "autonomia_agente_sin_runtime_reactivado", "proyecto", proyecto.ID,
			fmt.Sprintf("agente=%s detalle=%s mailbox_pending=%d open_tasks=%d blocked_tasks=%d", agente, strings.TrimSpace(row.DetalleOperativo), row.MailboxPending, len(tareasActivasPorAgente[agente]), len(tareasBloqueadasPorAgente[agente])))
		procesadas += 1 + desbloqueadas
	}
	return procesadas, nil
}

func desbloquearTareasBloqueadasRecuperablesSinRuntime(agente string, proyectoID int64, tareas []*db.Tarea, bloqueosPorTarea map[int64]db.ResumenBloqueo) (int, error) {
	procesadas := 0
	for _, tarea := range tareas {
		if tarea == nil || tarea.ID <= 0 {
			continue
		}
		actual, err := db.GetTarea(tarea.ID)
		if err != nil {
			return procesadas, err
		}
		if actual == nil || actual.Agente == nil || actual.ProyectoID == nil {
			continue
		}
		if actual.Estado != db.EstadoBloqueada || *actual.ProyectoID != proyectoID || !strings.EqualFold(strings.TrimSpace(*actual.Agente), agente) {
			continue
		}
		bloqueo, ok := bloqueosPorTarea[actual.ID]
		if !ok {
			continue
		}
		if !agentesapp.BloqueoAutonomiaRequiereIntervencion(agente, proyectoID, true, nil, nil, strings.TrimSpace(bloqueo.Motivo)) {
			resolucion := "reactivación automática al reponer runtime premium"
			if err := desbloquearYReactivarTareaAutonomia(actual.ID, resolucion, agente, fmt.Sprintf("Reactivada automáticamente en %s al reponer runtime premium", agente)); err != nil {
				return procesadas, err
			}
			procesadas++
		}
	}
	return procesadas, nil
}

func rowTieneRuntimeOHandleOperativo(row agentesapp.Row) bool {
	if row.Handle != nil {
		switch strings.ToLower(strings.TrimSpace(row.Handle.Estado)) {
		case "activo", "active", "pausado", "paused":
			return true
		}
	}
	if row.Runtime != nil {
		switch strings.ToLower(strings.TrimSpace(row.Runtime.LogicalState)) {
		case "activo", "active", "running", "iniciando", "starting", "esperando_io", "waiting_io", "pausado", "paused":
			return true
		}
	}
	return false
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
		stopReciente, err := runtimeOrderAutonomiaRecienteBloqueaRecuperacion(row, proyectoID, "stop", "stop", cooldown)
		if err != nil {
			return total, err
		}
		startReciente, err := runtimeOrderAutonomiaRecienteBloqueaRecuperacion(row, proyectoID, "start", "start", cooldown)
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
			if errorMigracionRuntimeLegacyTMUXIgnorable(err) {
				continue
			}
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
			if err := bloquearTareaAutonomiaSinRelevo(actual.ID, agente, motivo, fmt.Sprintf("Bloqueada automáticamente en %s por atasco persistente tras reinicios recientes", agente)); err != nil {
				return procesadas, err
			}
			if openTasksProjected[agente] > 0 {
				openTasksProjected[agente]--
			}
			procesadas++
			continue
		}
		if err := reasignarYArrancarTareaAutonomia(actual.ID, relevo, fmt.Sprintf("Reasignada automáticamente desde %s a %s por atasco persistente tras reinicios recientes", agente, relevo)); err != nil {
			return procesadas, err
		}
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
				if err := bloquearTareaAutonomiaSinRelevo(actual.ID, agente, "Sobrecarga operativa: sin relevo sano disponible", fmt.Sprintf("Bloqueada automáticamente en %s por sobrecarga operativa sin relevo sano", agente)); err != nil {
					return procesadas, err
				}
				if openTasksProjected[agente] > 0 {
					openTasksProjected[agente]--
				}
				procesadas++
				continue
			}
			if err := reasignarYArrancarTareaAutonomia(actual.ID, relevo, fmt.Sprintf("Redistribuida automáticamente desde %s a %s por sobrecarga operativa", agente, relevo)); err != nil {
				return procesadas, err
			}
			if openTasksProjected[agente] > 0 {
				openTasksProjected[agente]--
			}
			openTasksProjected[relevo]++
			procesadas++

			if err := encolarContinuacionTareaReasignadaSiCorresponde(relevo, actual.ProyectoID, actual.ID, agente, "sobrecarga_operativa", "continúa con la tarea redistribuida y deja evidencia de avance", map[string]any{"redistribuida_desde": agente}); err != nil {
				return procesadas, err
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
		rowAsignacionProyectoID(row.Asignacion),
		rowHandleProyectoID(row.Handle),
		rowRuntimeProyectoID(row.Runtime),
		rowSesionProyectoID(row.Sesion),
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

func construirNotaBloqueoAutonomiaSinRelevo(row agentesapp.Row, agente string) string {
	agente = strings.TrimSpace(agente)
	switch strings.TrimSpace(row.EstadoOperativo) {
	case "bloqueado_por_cuota":
		if agente != "" {
			return fmt.Sprintf("Bloqueada automáticamente en %s por cuota sin relevo sano", agente)
		}
		return "Bloqueada automáticamente por cuota sin relevo sano"
	case "mailbox_atascada":
		if agente != "" {
			return fmt.Sprintf("Bloqueada automáticamente en %s por mailbox atascada sin relevo sano", agente)
		}
		return "Bloqueada automáticamente por mailbox atascada sin relevo sano"
	default:
		return "Bloqueada automáticamente por degradación operativa sin relevo sano"
	}
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
	tareaPremiumAcotada := tareaDBTieneContratoPremiumAcotado(tarea)
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
		if tareaPremiumAcotada && openTasks > 0 {
			continue
		}
		if relevoPremiumExclusivoDebeOmitirse(row, tarea, openTasks) {
			continue
		}
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

func relevoPremiumExclusivoDebeOmitirse(row agentesapp.Row, tarea *db.Tarea, openTasks int) bool {
	if openTasks <= 0 || row.Asignacion == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(row.Asignacion.Nota), "microciclo_exclusivo") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(row.Asignacion.ProyectoSlug), "orquestador") {
		return false
	}
	if tarea == nil {
		return true
	}
	if tarea.ProyectoID == nil || *tarea.ProyectoID != row.Asignacion.ProyectoID {
		return true
	}
	return !tareaDBTieneContratoPremiumAcotado(tarea)
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
	return autonomiaDegradedTaskGate.AllowAt(strconv.FormatInt(tarea.ID, 10), autonomiaDegradedTaskCooldown, now)
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
	if n, err := procesarPrechecksAutonomiaSesionActiva(sesion, snapshot); err != nil || n > 0 {
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
		return procesarDecisionPausaSesionActivaAutonomia(sesion, proyecto, out, snapshot)
	case "supervisar_proyecto":
		// La supervisión rica del proyecto ya tiene su propio batch y señales
		// dedicadas. Repetir un nudge genérico por cada tick de una sesión viva
		// solo reinyecta guidance redundante sobre un runtime ya activo.
		return 0, nil
	case "votar_propuestas_pendientes", "pedir_intervencion":
		return procesarDecisionNudgeSesionActivaAutonomia(sesion, proyecto, out, snapshot)
	case "continuar_trabajo", "esperar_o_pedir_tarea":
		return procesarDecisionContinuacionSesionActivaAutonomia(sesion, proyecto, out, snapshot)
	default:
		return 0, nil
	}
}

func procesarPrechecksAutonomiaSesionActiva(sesion *db.Sesion, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	checks := []func() (int, error){
		func() (int, error) { return procesarCierreProyectoSesion(sesion) },
		func() (int, error) { return procesarReanudacionAutonomaSesion(sesion, snapshot) },
		func() (int, error) { return procesarAparcadoAutonomoSesion(sesion, snapshot) },
		func() (int, error) { return procesarRecuperacionRuntimeDegradadoSesion(sesion) },
		func() (int, error) { return procesarCompactacionExclusividadPremiumSesionActiva(sesion, snapshot) },
		func() (int, error) { return procesarRecuperacionTareasBloqueadasSesionActiva(sesion, snapshot) },
		func() (int, error) { return procesarCompactacionFrentesPremiumSesionActiva(sesion, snapshot) },
		func() (int, error) { return procesarDerivacionSemillaPremiumSesionActiva(sesion, snapshot) },
	}
	for _, check := range checks {
		n, err := check()
		if err != nil || n > 0 {
			if n > 0 {
				invalidarSnapshotAutonomiaProyecto(snapshot, sesion.Agente, valorProyectoID(sesion.ProyectoID))
			}
			return n, err
		}
	}
	return 0, nil
}

func procesarCompactacionExclusividadPremiumSesionActiva(sesion *db.Sesion, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	asignacion, err := db.GetAsignacionActivaAgente(strings.TrimSpace(sesion.Agente))
	if err != nil || asignacion == nil {
		return 0, err
	}
	if asignacion.ProyectoID != *sesion.ProyectoID || !asignacionMantieneExclusividadPremium(asignacion.Nota) {
		return 0, nil
	}
	agente := strings.TrimSpace(sesion.Agente)
	if agente == "" {
		return 0, nil
	}
	// La compactación de exclusividad debe ver todas las tareas reales del agente.
	// Un snapshot resumido puede ocultar frentes heredados que justamente queremos aparcar.
	tareas, err := tareasService.List(db.FiltroTareas{Agente: &agente})
	if err != nil {
		return 0, err
	}
	procesadas := 0
	for _, tarea := range tareas {
		if !tareaDebeCompactarsePorExclusividadPremium(tarea, asignacion.ProyectoID, agente) {
			continue
		}
		if err := tareasService.MoveToBacklog(tarea.ID); err != nil {
			return procesadas, err
		}
		procesadas++
	}
	if procesadas > 0 {
		resetStatusSnapshotCache()
		invalidarSnapshotAutonomiaProyecto(snapshot, agente, *sesion.ProyectoID)
	}
	return procesadas, nil
}

func tareaDebeCompactarsePorExclusividadPremium(tarea *db.Tarea, proyectoID int64, agente string) bool {
	if tarea == nil || tarea.ID <= 0 || tarea.Agente == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(*tarea.Agente), strings.TrimSpace(agente)) {
		return false
	}
	switch tarea.Estado {
	case db.TareaAsignada, db.TareaEnProgreso:
	default:
		return false
	}
	if taskManualTakeover(tarea) {
		return false
	}
	if tarea.ProyectoID == nil || *tarea.ProyectoID != proyectoID {
		return true
	}
	if strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:premium_frontier") {
		return false
	}
	return !tareaDBTieneContratoPremiumAcotado(tarea)
}

func asignacionMantieneExclusividadPremium(nota string) bool {
	switch strings.ToLower(strings.TrimSpace(nota)) {
	case "microciclo_exclusivo", "reactivacion_automatica_trabajo_activo":
		return true
	default:
		return false
	}
}

func procesarRecuperacionTareasBloqueadasSesionActiva(sesion *db.Sesion, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	agente := strings.TrimSpace(sesion.Agente)
	if agente == "" {
		return 0, nil
	}
	rows, err := agentesService.BuildPanelRows()
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	var row *agentesapp.Row
	for i := range rows {
		candidato := rows[i]
		if candidato.Agente == nil || !strings.EqualFold(strings.TrimSpace(candidato.Agente.Nombre), agente) {
			continue
		}
		if proyectoID := rowProyectoIDPreferido(candidato); proyectoID != nil && *proyectoID == *sesion.ProyectoID {
			row = &candidato
			break
		}
		if row == nil {
			row = &candidato
		}
	}
	if row == nil || !rowPermiteAutoRecuperacion(*row, now) {
		return 0, nil
	}
	var tareas []*db.Tarea
	if snapshot != nil {
		tareas, err = snapshot.tasks(agente, *sesion.ProyectoID)
		if err != nil {
			return 0, err
		}
	} else {
		tareas, err = tareasService.List(db.FiltroTareas{Agente: &agente, ProyectoID: sesion.ProyectoID})
		if err != nil {
			return 0, err
		}
	}
	if len(tareas) == 0 {
		return 0, nil
	}
	resumenBloqueos, err := db.ListarResumenBloqueos()
	if err != nil {
		return 0, err
	}
	bloqueosPorTarea := map[int64]db.ResumenBloqueo{}
	for _, bloqueo := range resumenBloqueos {
		bloqueosPorTarea[bloqueo.ID] = bloqueo
	}
	procesadas := 0
	for _, tarea := range tareas {
		if tarea == nil || tarea.Estado != db.TareaBloqueada || tarea.Agente == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(*tarea.Agente), agente) {
			continue
		}
		bloqueo, ok := bloqueosPorTarea[tarea.ID]
		if !ok || !esBloqueoAutonomiaAgenteRecuperable(agente, strings.TrimSpace(bloqueo.Agente), strings.TrimSpace(bloqueo.Motivo)) {
			continue
		}
		resolucion := fmt.Sprintf("recuperación automática de sesión activa (%s)", firstNonEmpty(strings.TrimSpace(row.EstadoOperativo), "worker recuperado"))
		if err := desbloquearYReactivarTareaAutonomia(tarea.ID, resolucion, agente, fmt.Sprintf("Reactivada automáticamente en %s desde la sesión activa", agente)); err != nil {
			return procesadas, err
		}
		procesadas++
	}
	if procesadas > 0 {
		resetStatusSnapshotCache()
		invalidarSnapshotAutonomiaProyecto(snapshot, agente, *sesion.ProyectoID)
	}
	return procesadas, nil
}

func procesarCompactacionFrentesPremiumSesionActiva(sesion *db.Sesion, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	agente := strings.TrimSpace(sesion.Agente)
	if agente == "" {
		return 0, nil
	}
	var (
		tareas []*db.Tarea
		err    error
	)
	if snapshot != nil {
		tareas, err = snapshot.tasks(agente, *sesion.ProyectoID)
	} else {
		tareas, err = tareasService.List(db.FiltroTareas{Agente: &agente, ProyectoID: sesion.ProyectoID})
	}
	if err != nil || len(tareas) <= 1 {
		return 0, err
	}
	candidatas := make([]*db.Tarea, 0, len(tareas))
	enProgreso := 0
	for _, tarea := range tareas {
		if !tareaEsFrentePremiumCompactable(tarea, agente) {
			continue
		}
		candidatas = append(candidatas, tarea)
		if tarea.Estado == db.TareaEnProgreso {
			enProgreso++
		}
	}
	if len(candidatas) <= 1 {
		return 0, nil
	}
	keep := seleccionarFrentePremiumCanonicSesionActiva(candidatas)
	if keep == nil {
		return 0, nil
	}
	multiplesEnProgreso := enProgreso > 1
	procesadas := 0
	for _, tarea := range candidatas {
		if tarea == nil || tarea.ID == keep.ID {
			continue
		}
		switch tarea.Estado {
		case db.TareaAsignada, db.TareaBloqueada:
		case db.TareaEnProgreso:
			if !multiplesEnProgreso {
				continue
			}
		default:
			continue
		}
		if err := tareasService.MoveToBacklog(tarea.ID); err != nil {
			return procesadas, err
		}
		procesadas++
	}
	if procesadas > 0 {
		resetStatusSnapshotCache()
		invalidarSnapshotAutonomiaProyecto(snapshot, agente, *sesion.ProyectoID)
	}
	return procesadas, nil
}

func tareaEsFrentePremiumActivoCompactable(tarea *db.Tarea, agente string) bool {
	return tareaEsFrentePremiumCompactable(tarea, agente) && tarea.Estado != db.TareaBloqueada
}

func tareaEsFrentePremiumCompactable(tarea *db.Tarea, agente string) bool {
	if tarea == nil || tarea.Agente == nil || !strings.EqualFold(strings.TrimSpace(*tarea.Agente), strings.TrimSpace(agente)) {
		return false
	}
	switch tarea.Estado {
	case db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada:
	default:
		return false
	}
	if strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:premium_frontier") {
		return false
	}
	return tareaPipelineLocalTieneContratoPremiumCmd(tareaPipelineLocalDesdeTareaDB(tarea))
}

func seleccionarFrentePremiumCanonicSesionActiva(tareas []*db.Tarea) *db.Tarea {
	var (
		mejor      *db.Tarea
		mejorScore = -1
	)
	for _, tarea := range tareas {
		score := puntuarFrentePremiumCanonicSesionActiva(tarea)
		if score < 0 {
			continue
		}
		if mejor == nil || score > mejorScore || (score == mejorScore && tarea.ID > mejor.ID) {
			mejor = tarea
			mejorScore = score
		}
	}
	return mejor
}

func puntuarFrentePremiumCanonicSesionActiva(tarea *db.Tarea) int {
	if tarea == nil {
		return -1
	}
	score := 0
	switch tarea.Estado {
	case db.TareaEnProgreso:
		score += 1000
	case db.TareaAsignada:
		score += 500
	case db.TareaBloqueada:
		score += 250
	default:
		return -1
	}
	tareaPipeline := tareaPipelineLocalDesdeTareaDB(tarea)
	if len(tareaPipeline.WriteSet) > 0 && strings.TrimSpace(tareaPipeline.TestsMinimos) != "" {
		score += 200
	}
	if strings.Contains(strings.TrimSpace(tarea.Notas), microcicloRefactorNotasTag) {
		score += 100
	}
	return score
}

func tareaPipelineLocalTieneContratoPremiumCmd(tarea *capacidadapp.TareaPipelineLocal) bool {
	if tarea == nil {
		return false
	}
	notas := strings.TrimSpace(tarea.Notas)
	if strings.Contains(notas, microcicloRefactorNotasTag) || strings.Contains(notas, "autonomia:premium_frontier") {
		return true
	}
	return len(tarea.WriteSet) > 0 && strings.TrimSpace(tarea.TestsMinimos) != ""
}

func tareaDBTieneContratoPremiumAcotado(tarea *db.Tarea) bool {
	if tarea == nil {
		return false
	}
	if strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:premium_frontier") {
		return false
	}
	return tareaPipelineLocalTieneContratoPremiumCmd(tareaPipelineLocalDesdeTareaDB(tarea))
}

func procesarDerivacionSemillaPremiumSesionActiva(sesion *db.Sesion, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	tareaID, err := db.GetTareaActivaIDPorAgenteProyecto(sesion.Agente, sesion.ProyectoID)
	if err != nil || tareaID <= 0 {
		return 0, err
	}
	tareaActual, err := tareasService.Get(tareaID)
	if err != nil || tareaActual == nil {
		return 0, err
	}
	if !strings.Contains(strings.TrimSpace(tareaActual.Notas), "autonomia:premium_frontier") {
		return 0, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil || proyecto == nil {
		return 0, err
	}
	var tareaNueva *capacidadapp.TareaPipelineLocal
	if tareaDerivada, err := derivarFrentePremiumAcotadoDesdeSemilla(proyecto.ID, sesion.Agente, tareaActual.ID); err != nil {
		return 0, err
	} else if tareaDerivada != nil {
		tareaNueva = tareaPipelineLocalDesdeTareaDB(tareaDerivada)
	}
	if tareaNueva == nil {
		tareaNueva, err = capacidadService.IntentarAutoasignarTareaPipelineLocal(proyecto.Slug, sesion.Agente)
	}
	if err != nil || tareaNueva == nil {
		return 0, err
	}
	if err := cerrarSemillaPremiumSiDerivaAFrenteAcotado(sesion.Agente, proyecto.ID, tareaNueva); err != nil {
		return 0, err
	}
	if ok, err := encolarNudgeAutonomiaConInvalidacion(
		snapshot,
		sesion.Agente,
		proyecto,
		"continuar_trabajo",
		fmt.Sprintf("Tarea #%d asignada automáticamente", tareaNueva.ID),
		"toma tarea asignada y sigue",
		map[string]any{"tarea_id": tareaNueva.ID, "motivo_autoasignacion": "semilla_premium_derivada"},
	); err != nil {
		return 0, err
	} else if ok {
		return 1, nil
	}
	return 0, nil
}

func tareaPipelineLocalDesdeTareaDB(tarea *db.Tarea) *capacidadapp.TareaPipelineLocal {
	if tarea == nil {
		return nil
	}
	out := &capacidadapp.TareaPipelineLocal{
		ID:           tarea.ID,
		Titulo:       strings.TrimSpace(tarea.Titulo),
		Descripcion:  strings.TrimSpace(tarea.Descripcion),
		Notas:        strings.TrimSpace(tarea.Notas),
		WriteSet:     capacidadapp.ExtraerWriteSetTextoPipelineLocal(strings.TrimSpace(tarea.Descripcion + "\n" + tarea.Notas)),
		SimbolosFoco: capacidadapp.ExtraerSimbolosFocoTextoPipelineLocal(strings.TrimSpace(tarea.Descripcion + "\n" + tarea.Notas)),
		TestsMinimos: capacidadapp.ExtraerTestsMinimosTextoPipelineLocal(strings.TrimSpace(tarea.Descripcion + "\n" + tarea.Notas)),
		Estado:       strings.TrimSpace(string(tarea.Estado)),
		Modulo:       strings.TrimSpace(tarea.Modulo),
		Prioridad:    strings.TrimSpace(string(tarea.Prioridad)),
	}
	if tarea.Agente != nil {
		out.Agente = strings.TrimSpace(*tarea.Agente)
	}
	return out
}

func derivarFrentePremiumAcotadoDesdeSemilla(proyectoID int64, agente string, excluirTaskID int64) (*db.Tarea, error) {
	agente = strings.TrimSpace(agente)
	if proyectoID <= 0 || agente == "" {
		return nil, nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		return nil, err
	}
	var candidata *db.Tarea
	mejor := -1
	for _, tarea := range tareas {
		if tarea == nil || tarea.ID == excluirTaskID {
			continue
		}
		if !tareaEsFrentePremiumAcotadoReutilizable(tarea) {
			continue
		}
		score := puntuarFrentePremiumAcotadoReutilizable(tarea, agente)
		if score < 0 {
			continue
		}
		if candidata == nil || score > mejor || (score == mejor && tarea.ID > candidata.ID) {
			candidata = tarea
			mejor = score
		}
	}
	if candidata == nil {
		return nil, nil
	}
	if candidata.Agente == nil || strings.TrimSpace(*candidata.Agente) == "" {
		switch candidata.Estado {
		case db.TareaLibre, db.TareaBacklog:
			if err := tareasService.Take(candidata.ID, agente); err != nil {
				return nil, err
			}
			return tareasService.Get(candidata.ID)
		}
	}
	actual := ""
	if candidata.Agente != nil {
		actual = strings.TrimSpace(*candidata.Agente)
	}
	if actual != "" && !strings.EqualFold(actual, agente) {
		return nil, nil
	}
	switch candidata.Estado {
	case db.TareaLibre, db.TareaBacklog:
		if err := tareasService.Take(candidata.ID, agente); err != nil {
			return nil, err
		}
		return tareasService.Get(candidata.ID)
	case db.TareaAsignada, db.TareaEnProgreso:
		return candidata, nil
	default:
		return nil, nil
	}
}

func tareaEsFrentePremiumAcotadoReutilizable(tarea *db.Tarea) bool {
	if tarea == nil {
		return false
	}
	if strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:premium_frontier") {
		return false
	}
	switch tarea.Estado {
	case db.TareaCompletada, db.TareaCancelada, db.TareaBloqueada:
		return false
	}
	if strings.Contains(strings.TrimSpace(tarea.Notas), microcicloRefactorNotasTag) {
		return true
	}
	contexto := strings.TrimSpace(tarea.Descripcion + "\n" + tarea.Notas)
	return len(capacidadapp.ExtraerWriteSetTextoPipelineLocal(contexto)) > 0 &&
		strings.TrimSpace(capacidadapp.ExtraerTestsMinimosTextoPipelineLocal(contexto)) != ""
}

func puntuarFrentePremiumAcotadoReutilizable(tarea *db.Tarea, agente string) int {
	if tarea == nil {
		return -1
	}
	actual := ""
	if tarea.Agente != nil {
		actual = strings.TrimSpace(*tarea.Agente)
	}
	switch tarea.Estado {
	case db.TareaEnProgreso:
		if strings.EqualFold(actual, agente) {
			return 40
		}
		return -1
	case db.TareaAsignada:
		if strings.EqualFold(actual, agente) {
			return 35
		}
		if actual == "" {
			return 25
		}
		return -1
	case db.TareaLibre:
		return 30
	case db.TareaBacklog:
		return 20
	default:
		return -1
	}
}

func procesarDecisionPausaSesionActivaAutonomia(sesion *db.Sesion, proyecto *db.Proyecto, out agenteTickOutput, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || proyecto == nil {
		return 0, nil
	}
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
	accion := agenteControlAccionPause
	if strings.TrimSpace(out.AccionRecomendada) == "pausar_y_reasignar" {
		accion = agenteControlAccionStop
	}
	if err := encolarControlAutonomiaProyecto(sesion.Agente, proyecto, accion, out.Motivo); err != nil {
		return 0, err
	}
	invalidarSnapshotAutonomiaProyecto(snapshot, sesion.Agente, proyecto.ID)
	return 1, nil
}

func procesarDecisionNudgeSesionActivaAutonomia(sesion *db.Sesion, proyecto *db.Proyecto, out agenteTickOutput, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || proyecto == nil {
		return 0, nil
	}
	if ok, err := encolarNudgeAutonomiaConInvalidacion(snapshot, sesion.Agente, proyecto, strings.TrimSpace(out.AccionRecomendada), out.Motivo, "", nil); err != nil {
		return 0, err
	} else if !ok {
		return 0, nil
	}
	return 1, nil
}

func procesarDecisionContinuacionSesionActivaAutonomia(sesion *db.Sesion, proyecto *db.Proyecto, out agenteTickOutput, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || proyecto == nil {
		return 0, nil
	}
	switch strings.TrimSpace(out.AccionRecomendada) {
	case "continuar_trabajo":
		return procesarDecisionContinuarTrabajoSesionActivaAutonomia(sesion, proyecto, out, snapshot)
	case "esperar_o_pedir_tarea":
		return procesarDecisionEsperarOPedirTareaSesionActivaAutonomia(sesion, proyecto, snapshot)
	default:
		return 0, nil
	}
}

func procesarDecisionContinuarTrabajoSesionActivaAutonomia(sesion *db.Sesion, proyecto *db.Proyecto, out agenteTickOutput, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || proyecto == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	tareaID, debe, err := sesionActivaDebeRecibirNudgeContinuacion(sesion, proyecto)
	if err != nil || !debe {
		return 0, err
	}
	throttleKey := autonomiaContinueNudgeThrottleKey(sesion.Agente, proyecto.ID, tareaID)
	if !autonomiaContinueNudgeShouldAttempt(sesion.Agente, proyecto.ID, tareaID) {
		return 0, nil
	}
	instruction := instruccionContinuacionSesionActivaAutonomia(tareaID)
	if ok, err := encolarNudgeAutonomiaConInvalidacion(
		snapshot,
		sesion.Agente,
		proyecto,
		"continuar_trabajo",
		out.Motivo,
		instruction,
		map[string]any{"tarea_id": tareaID, "motivo_autoasignacion": "sesion_activa_stale"},
	); err != nil {
		if throttleKey != "" {
			autonomiaContinueNudgeGate.Forget(throttleKey)
		}
		return 0, err
	} else if ok {
		return 1, nil
	}
	if throttleKey != "" {
		autonomiaContinueNudgeGate.Forget(throttleKey)
	}
	return 0, nil
}

func instruccionContinuacionSesionActivaAutonomia(tareaID int64) string {
	if tareaID <= 0 {
		return "Sigue con la tarea activa y cierra el siguiente slice útil del frente actual dentro del write-set y tests definidos."
	}
	tarea, err := tareasService.Get(tareaID)
	if err != nil || tarea == nil {
		return "Sigue con la tarea activa y cierra el siguiente slice útil del frente actual dentro del write-set y tests definidos."
	}
	if strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:premium_frontier") {
		return "Esta semilla premium no autoriza programar un frente amplio. Toma o reanuda exactamente una tarea premium acotada con WRITE_SET y tests mínimos; si no existe, créala por la app/CLI de Orquesta y continúa sobre ese frente derivado."
	}
	return "Sigue con la tarea activa y cierra el siguiente slice útil del frente actual dentro del write-set y tests definidos."
}

func procesarDecisionEsperarOPedirTareaSesionActivaAutonomia(sesion *db.Sesion, proyecto *db.Proyecto, snapshot *autonomiaBatchSnapshot) (int, error) {
	if sesion == nil || proyecto == nil {
		return 0, nil
	}
	if !autonomiaIdleAutoassignShouldAttempt(sesion.Agente, proyecto.ID) {
		return 0, nil
	}
	if err := revalidarYVerificarAgenteDisponibleParaTrabajo(sesion.Agente); err != nil {
		return 0, nil
	}
	tarea, err := capacidadService.IntentarAutoasignarTareaPipelineLocal(proyecto.Slug, sesion.Agente)
	if err != nil || tarea == nil {
		if err != nil {
			return 0, err
		}
		tarea, err = asegurarFrentePremiumSesionActivaIdle(sesion, proyecto)
		if err != nil || tarea == nil {
			return 0, err
		}
	}
	if err := cerrarSemillaPremiumSiDerivaAFrenteAcotado(sesion.Agente, proyecto.ID, tarea); err != nil {
		return 0, err
	}
	motivo := fmt.Sprintf("Tarea #%d asignada automáticamente", tarea.ID)
	if ok, err := encolarNudgeAutonomiaConInvalidacion(
		snapshot,
		sesion.Agente,
		proyecto,
		"continuar_trabajo",
		motivo,
		"toma tarea asignada y sigue",
		map[string]any{"tarea_id": tarea.ID, "motivo_autoasignacion": "sesion_activa_idle"},
	); err != nil {
		return 0, err
	} else if ok {
		return 1, nil
	}
	return 0, nil
}

func cerrarSemillaPremiumSiDerivaAFrenteAcotado(agente string, proyectoID int64, tarea *capacidadapp.TareaPipelineLocal) error {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 || tarea == nil {
		return nil
	}
	if strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:premium_frontier") {
		return nil
	}
	if len(tarea.WriteSet) == 0 || strings.TrimSpace(tarea.TestsMinimos) == "" {
		return nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		return err
	}
	for _, actual := range tareas {
		if actual == nil || actual.ID <= 0 || actual.ID == tarea.ID {
			continue
		}
		if actual.Agente == nil || !strings.EqualFold(strings.TrimSpace(*actual.Agente), agente) {
			continue
		}
		if !strings.Contains(strings.TrimSpace(actual.Notas), "autonomia:premium_frontier") {
			continue
		}
		switch actual.Estado {
		case db.TareaEnProgreso:
			if err := tareasService.Complete(actual.ID, agente, fmt.Sprintf("semilla satisfecha por tarea #%d", tarea.ID)); err != nil {
				return err
			}
		case db.TareaAsignada, db.TareaLibre, db.TareaBacklog:
			if err := tareasService.MoveToBacklog(actual.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func asegurarFrentePremiumSesionActivaIdle(sesion *db.Sesion, proyecto *db.Proyecto) (*capacidadapp.TareaPipelineLocal, error) {
	if sesion == nil || proyecto == nil {
		return nil, nil
	}
	policy, err := supervisionService.GetProjectPolicy(strings.TrimSpace(proyecto.Slug))
	if err != nil || policy == nil || !policy.Enabled {
		return nil, err
	}
	agente, err := agenteIfExists(strings.TrimSpace(sesion.Agente))
	if err != nil || agente == nil {
		return nil, err
	}
	res, err := asegurarTrabajoContinuoPremium(policy, proyecto, agente)
	if err != nil || res.SeedTaskID <= 0 {
		return nil, err
	}
	tarea, err := tareasService.Get(res.SeedTaskID)
	if err != nil || tarea == nil {
		return nil, err
	}
	out := &capacidadapp.TareaPipelineLocal{
		ID:           tarea.ID,
		Titulo:       strings.TrimSpace(tarea.Titulo),
		Descripcion:  strings.TrimSpace(tarea.Descripcion),
		Notas:        strings.TrimSpace(tarea.Notas),
		WriteSet:     capacidadapp.ExtraerWriteSetTextoPipelineLocal(strings.TrimSpace(tarea.Descripcion + "\n" + tarea.Notas)),
		SimbolosFoco: capacidadapp.ExtraerSimbolosFocoTextoPipelineLocal(strings.TrimSpace(tarea.Descripcion + "\n" + tarea.Notas)),
		TestsMinimos: capacidadapp.ExtraerTestsMinimosTextoPipelineLocal(strings.TrimSpace(tarea.Descripcion + "\n" + tarea.Notas)),
		Estado:       strings.TrimSpace(string(tarea.Estado)),
		Modulo:       strings.TrimSpace(tarea.Modulo),
		Prioridad:    strings.TrimSpace(string(tarea.Prioridad)),
	}
	if tarea.Agente != nil {
		out.Agente = strings.TrimSpace(*tarea.Agente)
	}
	return out, nil
}

func sesionActivaDebeRecibirNudgeContinuacion(sesion *db.Sesion, proyecto *db.Proyecto) (int64, bool, error) {
	if sesion == nil || proyecto == nil || sesion.ProyectoID == nil {
		return 0, false, nil
	}
	tareaID, err := db.GetTareaActivaIDPorAgenteProyecto(sesion.Agente, sesion.ProyectoID)
	if err != nil || tareaID <= 0 {
		return 0, false, err
	}
	handle, err := runtimeHandleSesionActivaParaContinuacion(sesion)
	if err != nil || handle == nil {
		return tareaID, false, err
	}
	if vigente, err := sesionActivaTieneContinuidadDurableVigente(sesion, handle); err != nil {
		return tareaID, false, err
	} else if vigente {
		return tareaID, false, nil
	}
	if !sesionActivaTMUXStaleParaContinuacion(handle, time.Now().UTC(), autonomiaContinueNudgeInterval()) {
		return tareaID, false, nil
	}
	return tareaID, true, nil
}

func sesionActivaTieneContinuidadDurableVigente(sesion *db.Sesion, handle *db.RuntimeHandle) (bool, error) {
	if sesion == nil || handle == nil || sesion.ProyectoID == nil || *sesion.ProyectoID <= 0 {
		return false, nil
	}
	if !runtimeMailboxGuidanceDurableUsaInbox(handle) {
		return false, nil
	}
	currentSessionID, err := runtimeMailboxGuidanceDurableExternalSessionID(handle)
	if err != nil {
		return false, err
	}
	currentSignature := runtimeMailboxDeliveryAttemptSignature(handle, currentSessionID)
	agente := strings.TrimSpace(sesion.Agente)
	if agente == "" {
		return false, nil
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     stringPtr(agente),
		ProyectoID: sesion.ProyectoID,
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if sesionActivaRuntimeOrderEsContinuidadDurableVigente(order, handle, currentSessionID, currentSignature) {
			return true, nil
		}
	}
	return false, nil
}

func sesionActivaRuntimeOrderEsContinuidadDurableVigente(order *db.RuntimeOrder, handle *db.RuntimeHandle, currentSessionID, currentSignature string) bool {
	if order == nil || handle == nil || strings.TrimSpace(order.Tipo) != "send_instruction" {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(runtimeOrderMailboxKindFromJSON(order.PayloadJSON)), "autonomia") {
		return false
	}
	payload := mapFromJSON(order.PayloadJSON)
	if !strings.EqualFold(strings.TrimSpace(stringMapValue(payload, "accion")), "continuar_trabajo") {
		return false
	}
	if !runtimeOrderMatchesCurrentHandleOSessionAttempt(order, handle.ID, currentSessionID, currentSignature) {
		return false
	}
	switch strings.TrimSpace(order.Estado) {
	case "pendiente", "tomada", "ejecutando":
		return runtimeOrderSigueBloqueandoMailbox(order, time.Now().UTC())
	case "completada":
		return runtimeOrderCompletedMailboxAttemptStillBlocksSesionActiva(order, currentSessionID, currentSignature)
	default:
		return false
	}
}

func runtimeOrderCompletedMailboxAttemptStillBlocksSesionActiva(order *db.RuntimeOrder, currentSessionID, currentSignature string) bool {
	if order == nil || !runtimeOrderMailboxOnlyResult(order.ResultadoJSON) {
		return false
	}
	if !runtimeOrderMatchesCurrentMailboxDeliveryAttempt(order, currentSessionID, currentSignature) {
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
	return time.Since(reference.UTC()) <= autonomiaContinueNudgeInterval()
}

func runtimeOrderMatchesCurrentHandleOSessionAttempt(order *db.RuntimeOrder, handleID int64, currentSessionID, currentSignature string) bool {
	if order == nil {
		return false
	}
	if order.HandleID != nil && handleID > 0 && *order.HandleID == handleID {
		return true
	}
	if currentSignature != "" {
		orderSignature := runtimeOrderDeliveryAttemptSignature(order.PayloadJSON)
		if orderSignature != "" && currentSignature == orderSignature {
			return true
		}
	}
	if currentSessionID != "" {
		orderSessionID := strings.TrimSpace(runtimeOrderExternalSessionIDFromJSON(order.PayloadJSON))
		if orderSessionID != "" && currentSessionID == orderSessionID {
			return true
		}
	}
	return false
}

func sesionActivaTMUXStaleParaContinuacion(handle *db.RuntimeHandle, now time.Time, staleAfter time.Duration) bool {
	if handle == nil {
		return false
	}
	_, view := runtimeHandleTMUXWorkerView(handle, now.UTC())
	if view == nil || !view.Alive || view.HeartbeatStale {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "running", "ready", "idle", "waiting_input":
	default:
		return false
	}
	if staleAfter <= 0 {
		staleAfter = 5 * time.Minute
	}
	lastActivity := time.Time{}
	switch {
	case view.LastProgressAt != nil && !view.LastProgressAt.IsZero():
		lastActivity = view.LastProgressAt.UTC()
	case view.ReadyAt != nil && !view.ReadyAt.IsZero():
		lastActivity = view.ReadyAt.UTC()
	case view.LastOutputAt != nil && !view.LastOutputAt.IsZero():
		lastActivity = view.LastOutputAt.UTC()
	}
	if lastActivity.IsZero() {
		return false
	}
	return !lastActivity.After(now.UTC().Add(-staleAfter))
}

func runtimeHandleSesionActivaParaContinuacion(sesion *db.Sesion) (*db.RuntimeHandle, error) {
	if sesion == nil {
		return nil, nil
	}
	if sesion.ID > 0 {
		handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
	}
	agente := strings.TrimSpace(sesion.Agente)
	if agente == "" {
		return nil, nil
	}
	if sesion.ProyectoID != nil && *sesion.ProyectoID > 0 {
		return db.GetRuntimeHandleCanonicoRecienteAgenteProyecto(agente, sesion.ProyectoID)
	}
	return db.GetRuntimeHandleCanonicoRecienteAgente(agente)
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
		if err := db.PausarAgenteHasta(agente, p.ResetAt.UTC(), motivo); err != nil {
			return err
		}
		return bloquearTareasActivasPorCuotaAutonomia(agente, motivo)
	}
	if err := db.PausarAgente(agente, 60, motivo); err != nil {
		return err
	}
	return bloquearTareasActivasPorCuotaAutonomia(agente, motivo)
}

func bloquearTareasActivasPorCuotaAutonomia(agente, motivo string) error {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{Agente: &agente})
	if err != nil {
		return err
	}
	motivoBloqueo := construirMotivoBloqueoCuotaAutonomia(agente, motivo)
	nota := fmt.Sprintf("Bloqueada automáticamente en %s por cuota sin relevo sano", agente)
	for _, tarea := range tareas {
		if tarea == nil || tarea.ID <= 0 {
			continue
		}
		actual, err := db.GetTarea(tarea.ID)
		if err != nil {
			return err
		}
		if actual == nil || actual.Agente == nil || !strings.EqualFold(strings.TrimSpace(*actual.Agente), agente) {
			continue
		}
		if actual.Estado != db.TareaAsignada && actual.Estado != db.TareaEnProgreso {
			continue
		}
		if err := bloquearTareaAutonomiaSinRelevo(actual.ID, agente, motivoBloqueo, nota); err != nil {
			return err
		}
	}
	return nil
}

func construirMotivoBloqueoCuotaAutonomia(agente, motivo string) string {
	agente = strings.TrimSpace(agente)
	motivo = strings.TrimSpace(motivo)
	if strings.Contains(strings.ToLower(motivo), "bloqueado_por_cuota") {
		return motivo
	}
	if motivo == "" {
		motivo = "worker bloqueado por cuota"
	}
	return fmt.Sprintf("Agente %s en estado bloqueado_por_cuota: %s", agente, motivo)
}

func procesarCierreProyectoSesion(sesion *db.Sesion) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	policy, err := supervisionService.GetProjectPolicy(proyecto.Slug)
	if err != nil {
		return 0, err
	}
	if policy == nil || !policy.Enabled || !policy.AutoCloseProject {
		return 0, nil
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil || !terminado {
		return 0, err
	}
	if err := cerrarProyectoAutonomia(proyecto.ID, motivo, db.AutonomiaProyectoCerrado, policy); err != nil {
		return 0, err
	}
	return pausarSesionAutonomiaPorMotivo(sesion, proyecto, "proyecto_terminado:"+motivo)
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
	return pausarSesionAutonomiaPorMotivo(sesion, proyecto, "bloqueo_humano:"+motivo)
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
			arrancable, err := tareaAutonomiaArrancableParaSesion(agente, tarea)
			if err != nil {
				return false, err
			}
			if arrancable {
				return true, nil
			}
		}
		return false, nil
	}
	return dbAgenteTieneTrabajoArrancable(agente, *sesion.ProyectoID)
}

func tareaAutonomiaArrancableParaSesion(agente string, tarea *db.Tarea) (bool, error) {
	if tarea == nil {
		return false, nil
	}
	switch tarea.Estado {
	case db.TareaAsignada, db.TareaEnProgreso:
		return true, nil
	case db.TareaBloqueada:
		return tareaBloqueadaRecuperableParaReanimacion(tarea.ID)
	default:
		return false, nil
	}
}

func reactivarTareasBloqueadasRecuperablesAutonomia(agente string, proyectoID int64, nota string) (int, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return 0, nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{
		Agente:     &agente,
		ProyectoID: &proyectoID,
	})
	if err != nil {
		return 0, err
	}
	procesadas := 0
	for _, tarea := range tareas {
		if tarea == nil || tarea.Estado != db.TareaBloqueada || tarea.Agente == nil || !strings.EqualFold(strings.TrimSpace(*tarea.Agente), agente) {
			continue
		}
		recuperable, err := tareaBloqueadaRecuperableParaReanimacion(tarea.ID)
		if err != nil {
			return procesadas, err
		}
		if !recuperable {
			continue
		}
		resolucion := "reactivación automática al salir de cuota o enfriamiento"
		if err := desbloquearYReactivarTareaAutonomia(tarea.ID, resolucion, agente, strings.TrimSpace(nota)); err != nil {
			return procesadas, err
		}
		procesadas++
	}
	return procesadas, nil
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
	compactadas, err := compactarFrentesExclusividadPremiumAgenteProyecto(strings.TrimSpace(sesion.Agente), *sesion.ProyectoID)
	if err != nil {
		return 0, err
	}
	estadoActiva := "activa"
	if err := db.GuardarSesionActiva(strings.TrimSpace(sesion.Agente), sesion.ProyectoID, db.SesionUpdate{
		Estado:    &estadoActiva,
		Heartbeat: true,
	}); err != nil {
		return 0, err
	}
	reactivadas, err := reactivarTareasBloqueadasRecuperablesAutonomia(
		strings.TrimSpace(sesion.Agente),
		*sesion.ProyectoID,
		fmt.Sprintf("Reactivada automáticamente en %s al salir de cuota o enfriamiento", strings.TrimSpace(sesion.Agente)),
	)
	if err != nil {
		return 0, err
	}
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(strings.TrimSpace(sesion.Agente), sesion.ProyectoID, "resume", "start", "pause", "handoff", "checkpoint"); err != nil {
		return 0, err
	} else if pendiente {
		return reactivadas, nil
	}
	handle, err := resolverHandleReactivacionAgente(strings.TrimSpace(sesion.Agente), sesion.ProyectoID)
	if err != nil {
		return reactivadas, err
	}
	accion := agenteControlAccionStart
	if handle != nil {
		runtimeCanonico, err := runtimeCanonicoDesdeHandleAgenteProyecto(handle, strings.TrimSpace(sesion.Agente), sesion.ProyectoID)
		if err != nil {
			return reactivadas, err
		}
		if runtimeLocalFallido(handle, runtimeCanonico) || runtimeRemotoDegradado(handle, runtimeCanonico) {
			accion = agenteControlAccionStart
		} else {
			switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
			case "activo":
				accion = agenteControlAccionResume
			case "pausado":
				if !db.RuntimeHandlePauseRequiresFreshStart(handle) {
					accion = agenteControlAccionResume
				}
			}
		}
	}
	if err := encolarControlAutonomiaProyecto(strings.TrimSpace(sesion.Agente), proyecto, accion, "desbloqueo_humano_auto"); err != nil {
		return compactadas + reactivadas, err
	}
	return 1 + reactivadas + compactadas, nil
}

func compactarFrentesExclusividadPremiumAgenteProyecto(agente string, proyectoID int64) (int, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return 0, nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{Agente: &agente})
	if err != nil {
		return 0, err
	}
	procesadas := 0
	for _, tarea := range tareas {
		if !tareaDebeCompactarsePorExclusividadPremium(tarea, proyectoID, agente) {
			continue
		}
		if err := tareasService.MoveToBacklog(tarea.ID); err != nil {
			return procesadas, err
		}
		procesadas++
	}
	if procesadas > 0 {
		resetStatusSnapshotCache()
	}
	return procesadas, nil
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
	var (
		fallback    *db.RuntimeHandle
		porProyecto *db.RuntimeHandle
	)
	for _, handle := range handles {
		if handle == nil || runtimeHandleEsCandidatoLegacyATMUX(handle) {
			continue
		}
		estado := strings.ToLower(strings.TrimSpace(handle.Estado))
		if estado != "activo" && estado != "pausado" && estado != "fallido" {
			continue
		}
		if proyectoID != nil && *proyectoID > 0 && handle.ProyectoID != nil && *handle.ProyectoID == *proyectoID {
			if runtimeHandlePreferibleParaRecuperacion(handle, porProyecto) {
				porProyecto = handle
			}
			continue
		}
		if runtimeHandlePreferibleParaRecuperacion(handle, fallback) {
			fallback = handle
		}
	}
	if porProyecto != nil {
		return porProyecto, nil
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
	if runtimeOperativoRecienteDistintoDeSesion(sesion) {
		return 0, nil
	}
	handle, runtime, err := runtimeRecuperacionSesionObjetivo(sesion)
	if err != nil {
		return 0, err
	}
	if handle == nil {
		proyecto, err := runtimesService.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
		if err != nil {
			return 0, err
		}
		reactivado, err := encolarReactivacionAgenteProyectoSiProcede(strings.TrimSpace(sesion.Agente), proyecto, "local_runtime_missing")
		if err != nil {
			return 0, err
		}
		if reactivado {
			return 1, nil
		}
		return 0, nil
	}
	if !esTransporteRemotoAutonomia(handle.Transporte) {
		return procesarRecuperacionRuntimeLocalSesion(sesion, handle, runtime)
	}
	if !runtimeRemotoDegradado(handle, runtime) {
		return 0, nil
	}
	return procesarRecuperacionRuntimeRemotoDegradadoSesion(sesion, handle, runtime)
}

// procesarRecuperacionRuntimeLocalSesion intenta recuperar una sesión cuyo runtime local
// ha fallado: sincroniza el handle, verifica si hay trabajo pendiente y encola un start.
func procesarRecuperacionRuntimeLocalSesion(sesion *db.Sesion, handle *db.RuntimeHandle, runtime *db.RuntimeInstance) (int, error) {
	if runtimeTMUXRecienteDebeSuplantarRecuperacionSesion(sesion, handle) {
		return 0, nil
	}
	var err error
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

	if err := encolarControlAutonomiaProyectoDetallado(apiAgenteControlRequest{
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

// procesarRecuperacionRuntimeRemotoDegradadoSesion emite un checkpoint seguido de un
// resume o start para recuperar una sesión con runtime remoto degradado. Si el conector
// asociado tiene el circuito abierto, pausa el proyecto en su lugar.
func procesarRecuperacionRuntimeRemotoDegradadoSesion(sesion *db.Sesion, handle *db.RuntimeHandle, runtime *db.RuntimeInstance) (int, error) {
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
			if err := encolarControlAutonomiaProyecto(strings.TrimSpace(sesion.Agente), proyecto, agenteControlAccionPause, motivo); err != nil {
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
	runtimeID := func() *int64 {
		if runtime != nil && runtime.ID > 0 {
			return &runtime.ID
		}
		return handle.RuntimeID
	}()
	if _, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      strings.TrimSpace(sesion.Agente),
		ProyectoID:  sesion.ProyectoID,
		RuntimeID:   runtimeID,
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
	if accion == agenteControlAccionResume {
		payload, err := json.Marshal(map[string]any{
			"accion":   agenteControlAccionResume,
			"motivo":   "remote_runtime_degraded",
			"por":      "orquesta",
			"proyecto": strings.TrimSpace(proyecto.Slug),
		})
		if err != nil {
			return 0, err
		}
		if _, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
			Agente:      strings.TrimSpace(sesion.Agente),
			ProyectoID:  sesion.ProyectoID,
			RuntimeID:   runtimeID,
			HandleID:    &handle.ID,
			Tipo:        agenteControlAccionResume,
			PayloadJSON: string(payload),
		}); err != nil {
			return 0, err
		}
		return 2, nil
	}
	if err := encolarControlAutonomiaProyecto(strings.TrimSpace(sesion.Agente), proyecto, accion, "remote_runtime_degraded"); err != nil {
		return 0, err
	}
	return 2, nil
}

func runtimeTMUXRecienteDebeSuplantarRecuperacionSesion(sesion *db.Sesion, handle *db.RuntimeHandle) bool {
	if sesion == nil || handle == nil {
		return false
	}
	if handle.SesionID == nil || *handle.SesionID == sesion.ID {
		return false
	}
	if !db.RuntimeHandleSnapshotIsFresh(handle, time.Minute) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") {
		return true
	}
	meta := mapFromJSON(strings.TrimSpace(handle.MetadataJSON))
	if strings.EqualFold(strings.TrimSpace(mapStringValue(meta, "driver")), "tmux_cli_session") {
		return true
	}
	return strings.TrimSpace(mapStringValue(meta, "tmux_session")) != ""
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
		runtime, err = db.GetRuntimeBySesionID(sesion.ID)
		if err != nil && err != sql.ErrNoRows {
			return nil, nil, err
		}
	}
	if handle == nil && agente != "" {
		var candidate *db.RuntimeHandle
		if proyectoID != nil && *proyectoID > 0 {
			candidate, err = db.GetRuntimeHandleOperativoRecienteAgenteProyecto(agente, proyectoID)
		} else {
			candidate, err = db.GetRuntimeHandleOperativoRecienteAgente(agente)
		}
		if err != nil {
			return nil, nil, err
		}
		if candidate != nil {
			handle = candidate
		}
	}
	if agente != "" {
		canonical, err := db.GetRuntimeHandleCanonicoRecienteAgenteProyecto(agente, proyectoID)
		if err != nil {
			return nil, nil, err
		}
		if runtimeHandlePreferibleParaRecuperacion(canonical, handle) {
			handle = canonical
		}
	}
	if handle != nil && sesionHandle != nil && handle.ID != sesionHandle.ID {
		runtime = nil
	}
	if runtime == nil {
		runtime, err = runtimeCanonicoDesdeHandleAgenteProyecto(handle, agente, proyectoID)
		if err != nil {
			return nil, nil, err
		}
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

func runtimeHandlePreferibleParaRecuperacion(candidate, current *db.RuntimeHandle) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	candidateTMUX := runtimeHandleTMUXRecienteParaRecuperacion(candidate)
	currentTMUX := runtimeHandleTMUXRecienteParaRecuperacion(current)
	if candidateTMUX != currentTMUX {
		return candidateTMUX
	}
	candidateFresh := db.RuntimeHandleSnapshotIsFresh(candidate, time.Minute)
	currentFresh := db.RuntimeHandleSnapshotIsFresh(current, time.Minute)
	if candidateFresh != currentFresh {
		return candidateFresh
	}
	return runtimeHandleRecencyAutonomia(candidate).After(runtimeHandleRecencyAutonomia(current))
}

func runtimeHandleTMUXRecienteParaRecuperacion(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if !db.RuntimeHandleSnapshotIsFresh(handle, time.Minute) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") {
		return true
	}
	meta := mapFromJSON(strings.TrimSpace(handle.MetadataJSON))
	if strings.EqualFold(strings.TrimSpace(mapStringValue(meta, "driver")), "tmux_cli_session") {
		return true
	}
	return strings.TrimSpace(mapStringValue(meta, "tmux_session")) != ""
}

func runtimeHandleRecencyAutonomia(handle *db.RuntimeHandle) time.Time {
	if handle == nil {
		return time.Time{}
	}
	if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() {
		return handle.LastSeenAt.UTC()
	}
	if !handle.UpdatedAt.IsZero() {
		return handle.UpdatedAt.UTC()
	}
	return handle.CreatedAt.UTC()
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
	if strings.EqualFold(strings.TrimSpace(sesion.Herramienta), "ollama_pool_local") ||
		strings.EqualFold(strings.TrimSpace(sesion.ConectorSlug), "ollama_pool_local") ||
		strings.EqualFold(strings.TrimSpace(stringFromMetadataJSON(handle.MetadataJSON, "driver")), "ollama_pool_local") {
		return false
	}
	if ext := strings.TrimSpace(sesion.ExternalSessionID); ext != "" {
		return true
	}
	if ext := strings.TrimSpace(stringFromMetadataJSON(handle.MetadataJSON, "external_session_id")); ext != "" {
		return true
	}
	if strings.TrimSpace(handle.HandleKind) == "process" {
		return false
	}
	ref := strings.TrimSpace(handle.HandleRef)
	if ref == "" {
		return false
	}
	return ref != strconv.FormatInt(sesion.ID, 10)
}

func runtimeOperativoRecienteDistintoDeSesion(sesion *db.Sesion) bool {
	if sesion == nil || sesion.ProyectoID == nil {
		return false
	}
	handle, err := db.GetRuntimeHandleOperativoRecienteAgenteProyecto(strings.TrimSpace(sesion.Agente), sesion.ProyectoID)
	if err != nil || handle == nil {
		return false
	}
	if handle.SesionID != nil && sesion.ID > 0 && *handle.SesionID == sesion.ID {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Estado), "activo") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") ||
		strings.EqualFold(strings.TrimSpace(stringFromMetadataJSON(handle.MetadataJSON, "driver")), "tmux_cli_session")
}

func reactivarAgenteTrasReanimacion(agente string) error {
	return reactivarAgenteTrasReanimacionConMotivo(agente, "reanimacion_automatica")
}

func reactivarAgenteTrasReanimacionConMotivo(agente, motivo string) error {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		motivo = "reanimacion_automatica"
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
	if proyecto != nil && proyecto.ID > 0 {
		if _, err := reactivarTareasBloqueadasRecuperablesAutonomia(
			agente,
			proyecto.ID,
			fmt.Sprintf("Reactivada automáticamente en %s al salir de cuota o enfriamiento", agente),
		); err != nil {
			return err
		}
	}
	_, err = encolarReactivacionAgenteProyectoSiProcede(agente, proyecto, motivo)
	return err
}

func encolarReactivacionAgenteProyectoSiProcede(agente string, proyecto *db.Proyecto, motivo string) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyecto == nil || proyecto.ID <= 0 {
		return false, nil
	}
	if agenteUsaPoolLocalCompartidoParaReactivacion(agente, proyecto) {
		if permite, poolSlug, err := db.PoolLocalCompartidoPermiteActivacionAgenteProyecto(agente, proyecto.Slug); err != nil {
			return false, err
		} else if !permite {
			db.Audit("orquesta", "reactivacion_pool_local_sin_capacidad", "agente", 0,
				fmt.Sprintf("agente=%s proyecto=%s pool=%s", agente, proyecto.Slug, poolSlug))
			return false, nil
		}
	}
	if pendiente, err := existeRuntimeOrderAbiertaAutonomia(agente, &proyecto.ID, "resume", "start", "handoff"); err != nil {
		return false, err
	} else if pendiente {
		return false, nil
	}
	handle, err := resolverHandleReactivacionAgente(agente, &proyecto.ID)
	if err != nil {
		return false, err
	}
	if handle != nil {
		switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
		case "pausado":
			accion := agenteControlAccionResume
			if db.RuntimeHandlePauseRequiresFreshStart(handle) {
				accion = agenteControlAccionStart
			}
			if err := encolarControlAutonomiaProyecto(agente, proyecto, accion, motivo); err != nil {
				return false, err
			}
			return true, nil
		case "fallido":
			if err := encolarControlAutonomiaProyecto(agente, proyecto, agenteControlAccionStart, motivo); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	tieneTrabajo, err := dbAgenteTieneTrabajoArrancable(agente, proyecto.ID)
	if err != nil {
		return false, err
	}
	if !tieneTrabajo {
		tieneBacklog, err := dbProyectoTieneBacklogReactivable(proyecto.ID)
		if err != nil {
			return false, err
		}
		if !tieneBacklog {
			return false, nil
		}
	}
	if err := encolarControlAutonomiaProyecto(agente, proyecto, agenteControlAccionStart, motivo); err != nil {
		return false, err
	}
	return true, nil
}

func resolverProyectoReactivacionAgente(agente string) (*db.Proyecto, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	if proyectoID, err := db.ObtenerProyectoActivoAgente(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return resolverProyectoReactivacionPorID(agente, proyectoID, "asignacion_activa")
	}
	if proyectoID, err := proyectoReactivacionDesdeTareas(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return resolverProyectoReactivacionPorID(agente, proyectoID, "tarea")
	}
	if proyectoID, err := proyectoReactivacionDesdeMailbox(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return resolverProyectoReactivacionPorID(agente, proyectoID, "mailbox")
	}
	if sesion, err := db.GetSesionAbierta(agente, nil); err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if sesion != nil && sesion.ProyectoID != nil && *sesion.ProyectoID > 0 {
		return resolverProyectoReactivacionPorID(agente, *sesion.ProyectoID, "sesion_abierta")
	}
	if sesion, err := db.ObtenerUltimaSesion(agente, nil); err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if sesion != nil && sesion.ProyectoID != nil && *sesion.ProyectoID > 0 {
		return resolverProyectoReactivacionPorID(agente, *sesion.ProyectoID, "ultima_sesion")
	}
	if handle, err := resolverHandleReactivacionAgente(agente, nil); err != nil {
		return nil, err
	} else if handle != nil && handle.ProyectoID != nil && *handle.ProyectoID > 0 {
		return resolverProyectoReactivacionPorID(agente, *handle.ProyectoID, "runtime_handle")
	}
	return nil, nil
}

func resolverProyectoReactivacionPorID(agente string, proyectoID int64, origen string) (*db.Proyecto, error) {
	if proyectoID <= 0 {
		return nil, nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		return nil, err
	}
	if proyecto == nil {
		return nil, fmt.Errorf("agente %s con proyecto de reactivacion inconsistente (%s=%d)", strings.TrimSpace(agente), strings.TrimSpace(origen), proyectoID)
	}
	return proyecto, nil
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

func proyectoReactivacionDesdeMailbox(agente string) (int64, error) {
	estado := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente: &agente,
		Estado:   &estado,
	})
	if err != nil {
		return 0, err
	}
	bestProjectID := int64(0)
	bestID := int64(0)
	for _, msg := range mailbox {
		if msg == nil || msg.ProyectoID == nil || *msg.ProyectoID <= 0 {
			continue
		}
		if bestProjectID == 0 || msg.ID > bestID {
			bestProjectID = *msg.ProyectoID
			bestID = msg.ID
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
	_, n, err := runtimeOrderAutonomiaRecienteMaxTimestamp(agente, proyectoID, tipo, accion, within)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func contarRuntimeOrdersRecientes(agente string, proyectoID *int64, tipo string, accion string, within time.Duration) (int, error) {
	_, total, err := runtimeOrderAutonomiaRecienteMaxTimestamp(agente, proyectoID, tipo, accion, within)
	return total, err
}

func runtimeOrderAutonomiaRecienteBloqueaRecuperacion(row agentesapp.Row, proyectoID *int64, tipo string, accion string, within time.Duration) (bool, error) {
	if row.Agente == nil {
		return false, nil
	}
	momento, total, err := runtimeOrderAutonomiaRecienteMaxTimestamp(strings.TrimSpace(row.Agente.Nombre), proyectoID, tipo, accion, within)
	if err != nil || total == 0 {
		return false, err
	}
	if row.WorkerHeartbeat != nil && !row.WorkerHeartbeat.IsZero() && row.WorkerHeartbeat.UTC().After(momento) {
		return false, nil
	}
	if row.WorkerUpdatedAt != nil && !row.WorkerUpdatedAt.IsZero() && row.WorkerUpdatedAt.UTC().After(momento) {
		return false, nil
	}
	return true, nil
}

func runtimeOrderAutonomiaRecienteMaxTimestamp(agente string, proyectoID *int64, tipo string, accion string, within time.Duration) (time.Time, int, error) {
	if within <= 0 {
		return time.Time{}, 0, nil
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
	})
	if err != nil {
		return time.Time{}, 0, err
	}
	cutoff := time.Now().UTC().Add(-within)
	total := 0
	latest := time.Time{}
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != strings.TrimSpace(tipo) {
			continue
		}
		if accion != "" && !runtimeOrderTieneAccion(order, accion) {
			continue
		}
		moment := runtimeOrderTimestamp(order)
		if moment.Before(cutoff) {
			continue
		}
		total++
		if moment.After(latest) {
			latest = moment
		}
	}
	return latest, total, nil
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

func invalidarSnapshotAutonomiaProyecto(snapshot *autonomiaBatchSnapshot, agente string, proyectoID int64) {
	if snapshot == nil {
		return
	}
	snapshot.invalidateAgent(strings.TrimSpace(agente))
	if proyectoID > 0 {
		snapshot.invalidateProject(proyectoID)
	}
}

func encolarNudgeAutonomiaConInvalidacion(snapshot *autonomiaBatchSnapshot, agente string, proyecto *db.Proyecto, accion, motivo, instruction string, extras map[string]any) (bool, error) {
	if proyecto == nil {
		return false, nil
	}
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(strings.TrimSpace(agente), &proyecto.ID, "nudge", strings.TrimSpace(accion)); err != nil {
		return false, err
	} else if pendiente {
		return false, nil
	}
	encolada, err := encolarNudgeAutonomiaDetallado(strings.TrimSpace(agente), proyecto, strings.TrimSpace(accion), motivo, instruction, extras)
	if err != nil || !encolada {
		return encolada, err
	}
	invalidarSnapshotAutonomiaProyecto(snapshot, strings.TrimSpace(agente), proyecto.ID)
	return true, nil
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

func encolarContinuacionTareaReasignada(agente string, proyecto *db.Proyecto, tareaID int64, origen, motivo, instruction string, extras map[string]any) (bool, error) {
	if tareaID <= 0 {
		return false, nil
	}
	payloadExtras := map[string]any{
		"tarea_id": tareaID,
	}
	origen = strings.TrimSpace(origen)
	if origen != "" {
		payloadExtras["reasignada_desde"] = origen
	}
	motivo = strings.TrimSpace(motivo)
	if motivo != "" {
		payloadExtras["motivo"] = motivo
	}
	for k, v := range extras {
		payloadExtras[k] = v
	}
	return encolarNudgeAutonomiaDetallado(
		agente,
		proyecto,
		"continuar_trabajo",
		fmt.Sprintf("Tarea #%d reasignada automáticamente", tareaID),
		instruction,
		payloadExtras,
	)
}

func encolarContinuacionTareaReasignadaSiCorresponde(agente string, proyectoID *int64, tareaID int64, origen, motivo, instruction string, extras map[string]any) error {
	if strings.TrimSpace(agente) == "" || proyectoID == nil || *proyectoID <= 0 || tareaID <= 0 {
		return nil
	}
	proyecto, err := runtimesService.GetProject(strconv.FormatInt(*proyectoID, 10))
	if err != nil || proyecto == nil {
		return err
	}
	pendiente, err := existeRuntimeOrderAutonomiaPendiente(strings.TrimSpace(agente), &proyecto.ID, "nudge", "continuar_trabajo")
	if err != nil || pendiente {
		return err
	}
	_, err = encolarContinuacionTareaReasignada(strings.TrimSpace(agente), proyecto, tareaID, origen, motivo, instruction, extras)
	return err
}

func encolarNudgeSignalTranscriptAutonomia(agente string, proyecto *db.Proyecto, accion, instruction string, item *db.RuntimeTranscriptEntry, prefijoEstado string) (string, error) {
	if strings.TrimSpace(agente) == "" || proyecto == nil || item == nil {
		return "", nil
	}
	if pendiente, err := existeRuntimeOrderAutonomiaPendiente(strings.TrimSpace(agente), &proyecto.ID, "nudge", strings.TrimSpace(accion)); err != nil {
		return "", err
	} else if pendiente {
		return estadoNudgeSignalTranscript(prefijoEstado, "pendiente"), nil
	}
	if encolada, err := encolarNudgeAutonomiaDetallado(strings.TrimSpace(agente), proyecto, strings.TrimSpace(accion), resumenSignalTranscriptAutonomia(item), instruction, payloadSignalTranscriptAutonomia(item)); err != nil {
		return "", err
	} else if !encolada {
		return estadoNudgeSignalTranscript(prefijoEstado, "omitido"), nil
	}
	return estadoNudgeSignalTranscript(prefijoEstado, "nudged"), nil
}

func payloadSignalTranscriptAutonomia(item *db.RuntimeTranscriptEntry) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"signal_classification": clasificacionSignalTranscript(item),
		"signal_transcript_id":  idSignalTranscript(item),
		"signal_agente":         agenteSignalTranscript(item),
		"signal_text":           textoSignalTranscript(item),
	}
}

func resumenSignalTranscriptAutonomia(item *db.RuntimeTranscriptEntry) string {
	if item == nil {
		return ""
	}
	return fmt.Sprintf("agente=%s signal=%s transcript=%d", agenteSignalTranscript(item), clasificacionSignalTranscript(item), idSignalTranscript(item))
}

func estadoNudgeSignalTranscript(prefijoEstado, sufijo string) string {
	prefijoEstado = strings.TrimSpace(prefijoEstado)
	sufijo = strings.TrimSpace(sufijo)
	if prefijoEstado == "" {
		return sufijo
	}
	if sufijo == "" {
		return prefijoEstado
	}
	if sufijo == "nudged" {
		return prefijoEstado + "_" + sufijo
	}
	return prefijoEstado + "_nudge_" + sufijo
}

func arrancarTareaAutonomiaSiAsignada(tareaID int64, agente string) error {
	if tareaID <= 0 || strings.TrimSpace(agente) == "" {
		return nil
	}
	return tareasService.Start(tareaID, strings.TrimSpace(agente))
}

func reasignarYArrancarTareaAutonomia(tareaID int64, relevo, nota string) error {
	if tareaID <= 0 || strings.TrimSpace(relevo) == "" {
		return nil
	}
	if err := tareasService.Reassign(tareaID, strings.TrimSpace(relevo)); err != nil {
		return err
	}
	if err := arrancarTareaAutonomiaSiAsignada(tareaID, strings.TrimSpace(relevo)); err != nil {
		return err
	}
	if strings.TrimSpace(nota) != "" {
		_ = tareasService.Note(tareaID, "orquesta", strings.TrimSpace(nota))
	}
	return nil
}

func bloquearTareaAutonomiaSinRelevo(tareaID int64, agente, motivo, nota string) error {
	if tareaID <= 0 || strings.TrimSpace(agente) == "" || strings.TrimSpace(motivo) == "" {
		return nil
	}
	if err := tareasService.Block(tareaID, strings.TrimSpace(agente), strings.TrimSpace(motivo)); err != nil {
		return err
	}
	if strings.TrimSpace(nota) != "" {
		_ = tareasService.Note(tareaID, "orquesta", strings.TrimSpace(nota))
	}
	return nil
}

func desbloquearYReasignarTareaAutonomia(tareaID int64, resolucion, relevo, nota string) error {
	if tareaID <= 0 || strings.TrimSpace(relevo) == "" {
		return nil
	}
	if err := tareasService.Unblock(tareaID, "orquesta", strings.TrimSpace(resolucion)); err != nil {
		return err
	}
	return reasignarYArrancarTareaAutonomia(tareaID, strings.TrimSpace(relevo), strings.TrimSpace(nota))
}

func desbloquearYReactivarTareaAutonomia(tareaID int64, resolucion, agente, nota string) error {
	if tareaID <= 0 || strings.TrimSpace(agente) == "" {
		return nil
	}
	if err := tareasService.Unblock(tareaID, "orquesta", strings.TrimSpace(resolucion)); err != nil {
		return err
	}
	if err := tareasService.Take(tareaID, strings.TrimSpace(agente)); err != nil {
		return err
	}
	if err := arrancarTareaAutonomiaSiAsignada(tareaID, strings.TrimSpace(agente)); err != nil {
		return err
	}
	if strings.TrimSpace(nota) != "" {
		_ = tareasService.Note(tareaID, "orquesta", strings.TrimSpace(nota))
	}
	return nil
}

func cerrarProyectoAutonomia(proyectoID int64, motivo string, estadoAutonomia db.EstadoAutonomiaProyecto, policy *db.ProyectoAutonomia) error {
	if proyectoID <= 0 {
		return nil
	}
	if err := db.MarcarProyectoCerrado(proyectoID, strings.TrimSpace(motivo)); err != nil {
		return err
	}
	if policy != nil && strings.TrimSpace(string(estadoAutonomia)) != "" {
		return persistirEstadoProyectoAutonomia(policy, estadoAutonomia)
	}
	return nil
}

func pausarSesionAutonomiaPorMotivo(sesion *db.Sesion, proyecto *db.Proyecto, motivo string) (int, error) {
	if sesion == nil || sesion.ProyectoID == nil || proyecto == nil {
		return 0, nil
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return 0, nil
	}
	if strings.EqualFold(strings.TrimSpace(sesion.Estado), "pausada") {
		if err := db.AparcarSesionActiva(sesion.Agente, sesion.ProyectoID); err != nil {
			return 0, err
		}
		if err := db.PausarAsignacion(sesion.Agente, proyecto.ID, motivo); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if satisfecha, err := pausaAutonomiaYaSatisfecha(sesion.Agente, &proyecto.ID, sesion); err != nil {
		return 0, err
	} else if satisfecha {
		return 0, nil
	}
	if err := encolarControlAutonomiaProyecto(sesion.Agente, proyecto, agenteControlAccionPause, motivo); err != nil {
		return 0, err
	}
	return 1, nil
}

func encolarControlAutonomiaProyecto(agente string, proyecto *db.Proyecto, accion, motivo string) error {
	if proyecto == nil {
		return nil
	}
	return encolarControlAutonomiaProyectoDetallado(apiAgenteControlRequest{
		Agente:   strings.TrimSpace(agente),
		Proyecto: strings.TrimSpace(proyecto.Slug),
		Accion:   accion,
		Motivo:   motivo,
		Por:      "orquesta",
	})
}

func encolarControlAutonomiaProyectoDetallado(req apiAgenteControlRequest) error {
	_, _, err := encolarControlAgenteLocal(req)
	return err
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
