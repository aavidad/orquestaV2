package db

import (
	"database/sql"
	"encoding/json"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
	"strings"
	"time"
)

func runtimeObservedLogicalStateFromStructuredWorker(handle *RuntimeHandle) (string, string, bool) {
	if handle == nil {
		return "", "", false
	}
	meta := mapFromJSON(strings.TrimSpace(handle.MetadataJSON))
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	transport := strings.TrimSpace(handle.Transporte)
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return "", "", false
	}
	if driver == "" {
		driver = strings.TrimSpace(snap.Driver())
	}
	if transport == "" {
		transport = strings.TrimSpace(snap.Transport())
	}
	if !strings.EqualFold(driver, "tmux_cli_session") && !strings.EqualFold(transport, "tmux") {
		return "", "", false
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil || view.HeartbeatStale || !view.Alive {
		return "", "", false
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "running", "idle":
		return "activo", "running", true
	default:
		return "", "", false
	}
}

func aplicarEstadoWorkerEstructuradoSinProceso(handle *RuntimeHandle, runtime *RuntimeInstance, source string) (bool, map[string]any, error) {
	logicalState, processState, ok := runtimeObservedLogicalStateFromStructuredWorker(handle)
	if !ok || handle == nil {
		return false, nil, nil
	}
	handleState := "activo"
	if strings.EqualFold(strings.TrimSpace(logicalState), "pausado") {
		handleState = "pausado"
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, handleState, handle.ID); err != nil {
		return true, nil, err
	}
	runtimeHandleHotReset()
	handle.Estado = handleState
	if runtime != nil && runtime.ID > 0 {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state = ?,
			    process_state = ?,
			    last_event_at = CURRENT_TIMESTAMP,
			    last_heartbeat_at = CURRENT_TIMESTAMP
			WHERE id = ?`, logicalState, processState, runtime.ID); err != nil {
			return true, nil, err
		}
		runtime.LogicalState = logicalState
		runtime.ProcessState = processState
	}
	if syncedHandle, _, err := SincronizarRuntimeHandleExternalSessionID(handle, runtime); err != nil {
		return true, nil, err
	} else if syncedHandle != nil {
		handle = syncedHandle
	}
	if err := marcarSesionHeartbeatSupervisada(handle, runtime, nil, logicalState); err != nil {
		return true, nil, err
	}
	if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, source); err != nil {
		return true, nil, err
	}
	return true, map[string]any{
		"process_alive":             true,
		"process_state":             processState,
		"logical_state":             logicalState,
		"observed_structured_worker": true,
	}, nil
}

func aplicarEstadoLocalObservado(handle *RuntimeHandle, estado *controlruntime.EstadoLocal) error {
	if handle == nil || estado == nil {
		return nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	for k, v := range mapFromJSON(estado.MetadataJSON) {
		meta[k] = v
	}
	normalizarMetadataTranscriptObservada(meta)
	compactarMetadataRuntimeHandle(meta)
	meta["local_last_status_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	metaJSON, _ := json.Marshal(meta)
	caps := handle.CapabilitiesJSON
	if strings.TrimSpace(estado.CapabilitiesJSON) != "" {
		caps = strings.TrimSpace(estado.CapabilitiesJSON)
	}
	estadoHandle := strings.TrimSpace(handle.Estado)
	if runtimeHandleBloqueaReactivacionPorCanalRoto(handle) {
		estadoHandle = "fallido"
	} else if rawEstado := strings.TrimSpace(estado.HandleEstado); rawEstado != "" {
		estadoHandle = normalizarEstadoHandleObservado(rawEstado, estadoHandle)
	}
	_, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?,
		    metadata_json = ?,
		    capabilities_json = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		estadoHandle, string(metaJSON), caps, handle.ID,
	)
	if err == nil {
		handle.Estado = estadoHandle
		handle.MetadataJSON = string(metaJSON)
		handle.CapabilitiesJSON = caps
	}
	return runtimeHandleHotResetOnSuccess(err)
}

func marcarProcesoLocalNoDisponible(handle *RuntimeHandle, runtime *RuntimeInstance) error {
	if handle != nil && handle.ID > 0 {
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado = 'fallido',
			    last_seen_at = CURRENT_TIMESTAMP
			WHERE id = ?`, handle.ID); err != nil {
			return err
		}
		runtimeHandleHotReset()
	}
	if runtime != nil && runtime.ID > 0 {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state = 'degradado',
			    process_state = 'missing',
			    last_event_at = CURRENT_TIMESTAMP
			WHERE id = ?`, runtime.ID); err != nil {
			return err
		}
	}
	return nil
}

func MarcarRuntimeHandleCanalRoto(handle *RuntimeHandle, runtime *RuntimeInstance, reason string) error {
	if handle == nil {
		return nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		meta = map[string]any{}
	}
	reason = strings.TrimSpace(reason)
	if reason != "" {
		meta["pty_last_broken_pipe_error"] = reason
	}
	meta["pty_last_broken_pipe_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	metaJSON, _ := json.Marshal(meta)
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = 'fallido',
		    metadata_json = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, string(metaJSON), handle.ID); err != nil {
		return err
	}
	runtimeHandleHotReset()
	if runtime == nil {
		var err error
		runtime, err = runtimeHandleRuntime(handle)
		if err != nil {
			return err
		}
	}
	if runtime != nil && runtime.ID > 0 {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state = 'degradado',
			    process_state = 'missing',
			    last_event_at = CURRENT_TIMESTAMP
			WHERE id = ?`, runtime.ID); err != nil {
			return err
		}
	}
	Audit("orquesta", "runtime_handle_broken_pipe", "runtime_handle", handle.ID, reason)
	return nil
}

func runtimeHandleBloqueaReactivacionPorCanalRoto(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	rawMeta := strings.ToLower(strings.TrimSpace(handle.MetadataJSON))
	if strings.Contains(rawMeta, "external_session_id incompatible with tmux premium runtime") {
		return true
	}
	reason := strings.ToLower(strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "pty_last_broken_pipe_error", "")))
	if reason == "" {
		return false
	}
	return strings.Contains(reason, "external_session_id incompatible")
}

func aplicarEstadoRemotoObservado(handle *RuntimeHandle, runtime *RuntimeInstance, estado *controlruntime.EstadoRemoto) error {
	if estado == nil {
		return nil
	}
	observedAt := time.Now().UTC().Format(time.RFC3339)
	conector, err := resolverConectorRuntime(handle, runtime)
	if err != nil {
		return err
	}
	if handle != nil {
		meta := mapFromJSON(handle.MetadataJSON)
		meta["remote_last_status_at"] = observedAt
		meta["remote_sync_failures"] = 0
		delete(meta, "remote_last_error")
		delete(meta, "remote_last_error_at")
		if rawEstado := strings.TrimSpace(estado.HandleEstado); rawEstado != "" {
			meta["remote_handle_state"] = rawEstado
		}
		for k, v := range mapFromJSON(estado.MetadataJSON) {
			meta[k] = v
		}
		normalizarMetadataTranscriptObservada(meta)
		compactarMetadataRuntimeHandle(meta)
		metaJSON, _ := json.Marshal(meta)
		caps := handle.CapabilitiesJSON
		if strings.TrimSpace(estado.CapabilitiesJSON) != "" {
			caps = strings.TrimSpace(estado.CapabilitiesJSON)
		}
		estadoHandle := strings.TrimSpace(handle.Estado)
		if rawEstado := strings.TrimSpace(estado.HandleEstado); rawEstado != "" {
			estadoHandle = normalizarEstadoHandleObservado(rawEstado, estadoHandle)
		}
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado = ?,
			    metadata_json = ?,
			    capabilities_json = ?,
			    last_seen_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			estadoHandle, string(metaJSON), caps, handle.ID,
		); err != nil {
			return err
		}
		runtimeHandleHotReset()
	}
	if runtime != nil {
		logical := strings.TrimSpace(runtime.LogicalState)
		if strings.TrimSpace(estado.LogicalState) != "" {
			logical = strings.TrimSpace(estado.LogicalState)
		}
		process := strings.TrimSpace(runtime.ProcessState)
		if strings.TrimSpace(estado.ProcessState) != "" {
			process = strings.TrimSpace(estado.ProcessState)
		}
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state = ?,
			    process_state = ?,
			    last_event_at = CURRENT_TIMESTAMP,
			    last_heartbeat_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			logical, process, runtime.ID,
		); err != nil {
			return err
		}
	}
	if conector != nil {
		if err := RegistrarExitoConector(conector.ID); err != nil {
			return err
		}
	}
	return nil
}

func normalizarMetadataTranscriptObservada(meta map[string]any) {
	if meta == nil {
		return
	}
	pending := runtimepolicy.CompactPendingTranscript(strings.TrimSpace(stringFromMap(meta, "transcript_log_pending", "")))
	if pending == "" {
		delete(meta, "transcript_log_pending")
	} else {
		meta["transcript_log_pending"] = pending
	}
}

func marcarSesionHeartbeatSupervisada(handle *RuntimeHandle, runtime *RuntimeInstance, pid *int64, logicalState string) error {
	sesionID := int64(0)
	agente := ""
	if handle != nil {
		agente = strings.TrimSpace(handle.Agente)
		if handle.SesionID != nil {
			sesionID = *handle.SesionID
		}
	}
	if runtime != nil {
		if agente == "" {
			agente = strings.TrimSpace(runtime.Agente)
		}
		if sesionID <= 0 && runtime.SesionID != nil {
			sesionID = *runtime.SesionID
		}
	}
	if sesionID <= 0 {
		return nil
	}
	estado := "activa"
	switch strings.ToLower(strings.TrimSpace(logicalState)) {
	case "pausado":
		estado = "pausada"
	}
	args := []any{estado, sesionID}
	q := `UPDATE sesiones SET estado = ?, heartbeat_at = CURRENT_TIMESTAMP`
	if pid != nil && *pid > 0 {
		q += `, pid = ?`
		args = []any{estado, *pid, sesionID}
	}
	q += ` WHERE id = ? AND activa = 1`
	if _, err := DB.Exec(q, args...); err != nil {
		return err
	}
	if agente != "" {
		SetEstadoSesion(agente, strings.TrimSpace(logicalState))
	}
	return nil
}

func registrarFalloObservacionRemota(handle *RuntimeHandle, runtime *RuntimeInstance, remoteErr error) (int, bool, error) {
	if handle == nil {
		return 0, false, nil
	}
	conector, err := resolverConectorRuntime(handle, runtime)
	if err != nil {
		return 0, false, err
	}
	meta := mapFromJSON(handle.MetadataJSON)
	failures := intFromMap(meta, "remote_sync_failures") + 1
	meta["remote_sync_failures"] = failures
	meta["remote_last_error"] = strings.TrimSpace(remoteErr.Error())
	meta["remote_last_error_at"] = time.Now().UTC().Format(time.RFC3339)
	threshold := configIntOrDefault("runtime_remote_sync_failure_threshold", 3)
	if threshold <= 0 {
		threshold = 3
	}
	degraded := failures >= threshold
	handleState := strings.TrimSpace(handle.Estado)
	if degraded {
		handleState = "fallido"
	}
	metaJSON, _ := json.Marshal(meta)
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?,
		    metadata_json = ?
		WHERE id = ?`,
		handleState, string(metaJSON), handle.ID,
	); err != nil {
		return failures, degraded, err
	}
	if runtime != nil && degraded {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state = ?,
			    process_state = ?,
			    last_event_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			"degradado", "remote_status_error", runtime.ID,
		); err != nil {
			return failures, degraded, err
		}
	}
	if conector != nil {
		if _, err := RegistrarFalloConector(conector.ID, remoteErr.Error()); err != nil {
			return failures, degraded, err
		}
	}
	return failures, degraded, nil
}

func resolverConectorRuntime(handle *RuntimeHandle, runtime *RuntimeInstance) (*Conector, error) {
	slug := ""
	if runtime != nil {
		slug = strings.TrimSpace(runtime.Connector)
	}
	if slug == "" && handle != nil {
		slug = stringFromMap(mapFromJSON(handle.MetadataJSON), "conector", "")
	}
	if slug == "" && handle != nil && handle.SesionID != nil {
		sesion, err := sesionIfExists(handle.SesionID)
		if err != nil {
			return nil, err
		}
		if sesion != nil {
			slug = strings.TrimSpace(sesion.ConectorSlug)
		}
	}
	if slug == "" {
		return nil, nil
	}
	conector, err := GetConector(slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return conector, err
}

func normalizarEstadoHandleObservado(observado, actual string) string {
	switch strings.ToLower(strings.TrimSpace(observado)) {
	case "activo", "active", "running", "online", "ready", "idle", "esperando", "degradado", "warning":
		return "activo"
	case "pausado", "paused", "suspended":
		return "pausado"
	case "cerrado", "closed", "stopped", "finished", "finalizado":
		return "cerrado"
	case "fallido", "failed", "error", "crashed":
		return "fallido"
	}
	actual = strings.TrimSpace(actual)
	if actual == "" {
		return "activo"
	}
	return actual
}
