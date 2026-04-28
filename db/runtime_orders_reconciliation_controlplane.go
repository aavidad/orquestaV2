package db

import (
	"database/sql"
	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
	"strings"
	"time"
)

func runtimeOrderControlEstadoDeseadoSatisfecho(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) (bool, string) {
	if order == nil {
		return false, ""
	}
	orderType := strings.ToLower(strings.TrimSpace(order.Tipo))
	now := time.Now().UTC()
	switch orderType {
	case "start", "resume":
		if handle != nil {
			switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
			case "pausado", "fallido", "cerrado", "fantasma":
				return false, ""
			}
			if runtimeHandleSesionTMUXAusente(handle) {
				return false, ""
			}
		}
		if runtime != nil {
			switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
			case "pausado", "stopped", "fallido", "degradado", "cerrado":
				return false, ""
			}
			switch strings.ToLower(strings.TrimSpace(runtime.ProcessState)) {
			case "fallido", "crashed", "exited", "remote_status_error":
				return false, ""
			}
		}
		if runtimeOrderTieneStopVivaPrevia(order) {
			return false, ""
		}
		if runtimeHandleActivoAPICompartido(handle, runtime) {
			return true, "runtime_handle_api_active"
		}
		if handle == nil && runtime != nil && runtimeOrderRuntimeOperativoConSesionActiva(order, runtime) {
			return true, "runtime_already_running"
		}
		if runtimeWorkerSnapshotSatisfaceControlActual(orderType, handle, runtimeWorkerSnapshot(handle, runtime), now) {
			return true, "runtime_worker_already_running"
		}
	case "stop":
		if runtimeWorkerSnapshotSatisfaceControlActual(orderType, handle, runtimeWorkerSnapshot(handle, runtime), now) {
			return true, "runtime_worker_already_stopped"
		}
		if handle == nil && runtime == nil {
			if order != nil {
				if sesion, err := GetSesionAbierta(strings.TrimSpace(order.Agente), order.ProyectoID); err == nil && sesion != nil {
					return false, ""
				}
			}
			return true, "runtime_already_stopped"
		}
		if handle != nil {
			switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
			case "cerrado", "fallido":
				return true, "runtime_handle_already_closed"
			case "pausado":
				return false, ""
			}
		}
		if runtime != nil {
			switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
			case "cerrado", "stopped", "fallido":
				return true, "runtime_already_stopped"
			}
		}
	}
	return false, ""
}

func runtimeOrderTieneStopVivaPrevia(order *RuntimeOrder) bool {
	if order == nil || order.ID <= 0 {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(order.Tipo), "start") && !strings.EqualFold(strings.TrimSpace(order.Tipo), "resume") {
		return false
	}
	query := `
SELECT 1
FROM runtime_orders
WHERE agente = ?
  AND lower(trim(tipo)) = 'stop'
  AND lower(trim(estado)) IN ('pendiente', 'ejecutando')
  AND id < ?`
	args := []any{strings.TrimSpace(order.Agente), order.ID}
	if order.ProyectoID != nil {
		query += ` AND proyecto_id = ?`
		args = append(args, *order.ProyectoID)
	} else {
		query += ` AND proyecto_id IS NULL`
	}
	query += ` LIMIT 1`
	var marker int
	if err := DB.QueryRow(query, args...).Scan(&marker); err != nil {
		return false
	}
	return marker == 1
}

func runtimeOrderRuntimeOperativoConSesionActiva(order *RuntimeOrder, runtime *RuntimeInstance) bool {
	if order == nil || runtime == nil {
		return false
	}
	now := time.Now().UTC()
	if runtime.SesionID != nil && *runtime.SesionID > 0 {
		if sesion, err := sesionIfExists(runtime.SesionID); err == nil && sesion != nil {
			if sesionCoincideProyectoOrden(sesion, order.ProyectoID) &&
				sesionOperativaPorHeartbeat(sesion.Activa, sesion.HeartbeatAt, sesion.Inicio, now) {
				return true
			}
		}
	}
	sesion, err := GetSesionActiva(strings.TrimSpace(order.Agente), order.ProyectoID)
	return err == nil && sesion != nil
}

func sesionCoincideProyectoOrden(sesion *Sesion, proyectoID *int64) bool {
	if sesion == nil {
		return false
	}
	if proyectoID == nil || *proyectoID <= 0 {
		return sesion.Activa
	}
	return sesion.ProyectoID != nil && *sesion.ProyectoID == *proyectoID
}

func runtimeOrderPromoverEstadoObservadoSiSatisfecha(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) error {
	if order == nil || runtime == nil {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(order.Tipo)) {
	case "start", "resume":
		if handle == nil {
			sesionID := int64(0)
			if runtime.SesionID != nil && *runtime.SesionID > 0 {
				sesionID = *runtime.SesionID
			}
			var sesion *Sesion
			var err error
			switch {
			case sesionID > 0:
				sesion, err = GetSesionByID(sesionID)
			default:
				sesion, err = GetSesionActiva(strings.TrimSpace(order.Agente), order.ProyectoID)
			}
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if sesion != nil {
				if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
					return err
				}
				if refreshed, err := GetRuntimeHandleBySesionID(sesion.ID); err != nil {
					return err
				} else if refreshed != nil {
					handle = refreshed
				}
			}
		}
		return runtimePromoverEstadoObservadoDesdeHandle(handle, runtime)
	default:
		return nil
	}
}

func runtimePromoverEstadoObservadoDesdeHandle(handle *RuntimeHandle, runtime *RuntimeInstance) error {
	if runtime == nil {
		return nil
	}
	logicalState := strings.TrimSpace(runtime.LogicalState)
	processState := strings.TrimSpace(runtime.ProcessState)
	if workerLogicalState, workerProcessState, ok := runtimeObservedLogicalStateFromStructuredWorker(handle); ok {
		logicalState = workerLogicalState
		if strings.TrimSpace(workerProcessState) != "" {
			processState = workerProcessState
		}
	} else if runtimeHandleActivoAPICompartido(handle, runtime) {
		if logicalState == "" || strings.EqualFold(logicalState, "esperando_io") || strings.EqualFold(logicalState, "disponible") {
			logicalState = "activo"
		}
		if processState == "" || strings.EqualFold(processState, "desconocido") {
			processState = "running"
		}
	}
	if logicalState == "" {
		return nil
	}
	if processState == "" {
		processState = "running"
	}
	_, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state = ?,
		    process_state = ?,
		    last_event_at = CURRENT_TIMESTAMP,
		    last_heartbeat_at = CURRENT_TIMESTAMP
		WHERE id = ?`, logicalState, processState, runtime.ID)
	return err
}

func runtimeHandleActivoAPICompartido(handle *RuntimeHandle, runtime *RuntimeInstance) bool {
	if handle == nil || runtime == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Estado), "activo") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Transporte), "api") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
	case "", "disponible", "working", "esperando_io":
	default:
		return false
	}
	switch strings.ToLower(strings.TrimSpace(runtime.ProcessState)) {
	case "", "running", "desconocido":
	default:
		return false
	}
	return true
}

func runtimeHandleSesionTMUXAusente(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	return runtimepolicy.RuntimeHandleTMUXSessionMissing(handle.MetadataJSON)
}

func runtimeWorkerSnapshotSatisfaceControlActual(orderType string, handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot, now time.Time) bool {
	if snap == nil {
		return false
	}
	if runtimeHandleSesionTMUXAusente(handle) {
		return strings.EqualFold(strings.TrimSpace(orderType), "stop")
	}
	view := snap.View(now.UTC(), time.Minute)
	if view == nil {
		return false
	}
	state := strings.ToLower(strings.TrimSpace(view.State))
	switch strings.ToLower(strings.TrimSpace(orderType)) {
	case "start", "resume":
		if view.HeartbeatStale || !view.Alive || strings.TrimSpace(view.ExitError) != "" {
			return false
		}
		return state == "ready" || state == "running" || state == "starting"
	case "stop":
		switch state {
		case "stopped", "failed", "exited", "closed", "stale":
			return true
		}
		if strings.TrimSpace(view.ExitError) != "" {
			return true
		}
		return !view.Alive && !view.HeartbeatStale
	default:
		return false
	}
}

func runtimeOrderControlObsoletaPorWorkerRecuperado(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) bool {
	if order == nil {
		return false
	}
	orderType := strings.TrimSpace(order.Tipo)
	switch orderType {
	case "start", "resume":
	default:
		return false
	}
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		return false
	}
	if runtimeHandleSesionTMUXAusente(handle) {
		return false
	}
	if runtime != nil {
		state := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
		if state == "pausado" || state == "stopped" {
			return false
		}
	}
	recovery := runtimeWorkerRecoveryMoment(handle, runtime)
	if recovery.IsZero() {
		return false
	}
	baseline := order.CreatedAt.UTC()
	if order.StartedAt != nil && !order.StartedAt.IsZero() && order.StartedAt.UTC().After(baseline) {
		baseline = order.StartedAt.UTC()
	}
	if snap := runtimeWorkerSnapshot(handle, runtime); snap != nil {
		return runtimeWorkerSnapshotRecoveredAfterBaseline(orderType, snap, baseline)
	}
	return recovery.After(baseline)
}

func runtimeWorkerSnapshotRecoveredAfterBaseline(orderType string, snap *runtimeagente.WorkerSnapshot, baseline time.Time) bool {
	if snap == nil || !snap.Alive() || snap.IsHeartbeatStale(time.Now().UTC(), time.Minute) {
		return false
	}
	if strings.TrimSpace(snap.ExitError()) != "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(snap.EffectiveState())) {
	case "ready", "running":
	default:
		return false
	}
	candidates := []*time.Time{
		snap.ReadyTime(),
		snap.StartedTime(),
	}
	if strings.EqualFold(strings.TrimSpace(orderType), "resume") {
		candidates = append(candidates, snap.LastProgressTime(), snap.LastOutputTime())
	}
	for _, candidate := range candidates {
		if candidate == nil || candidate.IsZero() {
			continue
		}
		if baseline.IsZero() || candidate.UTC().After(baseline) {
			return true
		}
	}
	return false
}

func runtimeWorkerRecoveryMoment(handle *RuntimeHandle, runtime *RuntimeInstance) time.Time {
	latest := time.Time{}
	if runtime != nil {
		if ts := runtimeMomentForRecovery(runtime); !ts.IsZero() && ts.After(latest) {
			latest = ts
		}
	}
	if handle != nil {
		if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() && handle.LastSeenAt.UTC().After(latest) {
			latest = handle.LastSeenAt.UTC()
		}
		if !handle.UpdatedAt.IsZero() && handle.UpdatedAt.UTC().After(latest) {
			latest = handle.UpdatedAt.UTC()
		}
	}
	if snap := runtimeWorkerSnapshot(handle, runtime); snap != nil && snap.Alive() && !snap.IsHeartbeatStale(time.Now().UTC(), time.Minute) {
		for _, candidate := range []*time.Time{
			snap.ReadyTime(),
			snap.LastProgressTime(),
			snap.LastOutputTime(),
			snap.HeartbeatTime(),
			snap.UpdatedTime(),
			snap.StartedTime(),
		} {
			if candidate != nil && !candidate.IsZero() && candidate.UTC().After(latest) {
				latest = candidate.UTC()
			}
		}
	}
	return latest
}

func runtimeMomentForRecovery(runtime *RuntimeInstance) time.Time {
	if runtime == nil {
		return time.Time{}
	}
	for _, value := range []*time.Time{runtime.UltimaActividadAt, runtime.LastHeartbeatAt, runtime.LastEventAt} {
		if value != nil && !value.IsZero() {
			return value.UTC()
		}
	}
	if !runtime.UpdatedAt.IsZero() {
		return runtime.UpdatedAt.UTC()
	}
	return runtime.CreatedAt.UTC()
}
