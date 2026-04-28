package db

import (
	"encoding/json"
	"strings"
	"time"

	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
)

func SincronizarRuntimeHandleExternalSessionID(handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeHandle, string, error) {
	if handle == nil {
		return nil, "", nil
	}
	if runtime == nil {
		var err error
		runtime, err = runtimeHandleRuntime(handle)
		if err != nil {
			return nil, "", err
		}
	}
	externalSessionID, err := runtimeHandleEffectiveExternalSessionID(handle, runtime)
	if err != nil || strings.TrimSpace(externalSessionID) == "" {
		return handle, strings.TrimSpace(externalSessionID), err
	}

	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		meta = map[string]any{}
	}
	if strings.TrimSpace(stringFromMap(meta, "external_session_id", "")) != externalSessionID {
		meta["external_session_id"] = externalSessionID
		metaJSON, _ := json.Marshal(meta)
		if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, string(metaJSON), handle.ID); err != nil {
			return nil, "", err
		}
		runtimeHandleHotReset()
	}
	if runtime != nil && runtime.ID > 0 && strings.TrimSpace(runtime.ExternalSessionID) != externalSessionID {
		if _, err := DB.Exec(`UPDATE runtime_instances SET external_session_id=?, last_event_at=CURRENT_TIMESTAMP WHERE id=?`, externalSessionID, runtime.ID); err != nil {
			return nil, "", err
		}
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		if _, err := DB.Exec(`UPDATE sesiones SET external_session_id=? WHERE id=?`, externalSessionID, *handle.SesionID); err != nil {
			return nil, "", err
		}
	}
	fresh, err := GetRuntimeHandle(handle.ID)
	if err != nil {
		return nil, "", err
	}
	return fresh, externalSessionID, nil
}

func SincronizarRuntimeHandleSupervisado(handle *RuntimeHandle, runtime *RuntimeInstance, source string) (*RuntimeHandle, *RuntimeInstance, string, error) {
	if handle == nil {
		return nil, runtime, "", nil
	}
	if runtime == nil {
		var err error
		runtime, err = runtimeHandleRuntime(handle)
		if err != nil {
			return nil, nil, "", err
		}
	}
	if runtimeHandleBloqueaReactivacionPorCanalRoto(handle) {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return nil, nil, "", err
		}
		fresh, err := GetRuntimeHandle(handle.ID)
		if err != nil {
			return nil, nil, "", err
		}
		if runtime != nil && runtime.ID > 0 {
			if refreshedRuntime, err := GetRuntime(runtime.ID); err == nil && refreshedRuntime != nil {
				runtime = refreshedRuntime
			}
		}
		return fresh, runtime, "", nil
	}
	if observed, _, err := observarProcesoLocalRuntime(handle, runtime, source); err != nil {
		return nil, nil, "", err
	} else if observed {
		fresh, err := GetRuntimeHandle(handle.ID)
		if err != nil {
			return nil, nil, "", err
		}
		handle = fresh
		if runtime != nil && runtime.ID > 0 {
			if refreshedRuntime, err := GetRuntime(runtime.ID); err == nil && refreshedRuntime != nil {
				runtime = refreshedRuntime
			}
		} else if handle != nil {
			if refreshedRuntime, err := runtimeHandleRuntime(handle); err == nil && refreshedRuntime != nil {
				runtime = refreshedRuntime
			}
		}
	}
	if handle == nil {
		return nil, runtime, "", nil
	}
	if runtimepolicy.RuntimeHandleTMUXSessionMissing(handle.MetadataJSON) {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return nil, nil, "", err
		}
		fresh, err := GetRuntimeHandle(handle.ID)
		if err != nil {
			return nil, nil, "", err
		}
		handle = fresh
		if runtime != nil && runtime.ID > 0 {
			if refreshedRuntime, err := GetRuntime(runtime.ID); err == nil && refreshedRuntime != nil {
				runtime = refreshedRuntime
			}
		}
	}
	handle, externalSessionID, err := SincronizarRuntimeHandleExternalSessionID(handle, runtime)
	if err != nil {
		return nil, nil, "", err
	}
	handle, err = normalizarRuntimeHandleTMUXCanonico(handle)
	if err != nil {
		return nil, nil, "", err
	}
	handle, err = compactarMetadataHandleRuntimePersistida(handle)
	if err != nil {
		return nil, nil, "", err
	}
	return handle, runtime, externalSessionID, nil
}

func normalizarRuntimeHandleTMUXCanonico(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil || handle.ID <= 0 {
		return handle, nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	transport := strings.TrimSpace(stringFromMap(meta, "transport", ""))
	snap, snapErr := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if snapErr == nil && snap != nil {
		if driver == "" {
			driver = strings.TrimSpace(snap.Driver())
		}
		if transport == "" {
			transport = strings.TrimSpace(snap.Transport())
		}
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		!strings.EqualFold(driver, "tmux_cli_session") &&
		!strings.EqualFold(transport, "tmux") &&
		!runtimeHandleEsTMUXCanonico(handle) {
		return handle, nil
	}
	canonicalRef := runtimeHandleTMUXSessionRefObserved(handle)
	if canonicalRef == "" {
		if snapErr != nil {
			return handle, nil
		}
		if snap != nil {
			canonicalRef = strings.TrimSpace(snap.RuntimeRef())
		}
	}
	if canonicalRef == "" {
		return handle, nil
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") &&
		strings.TrimSpace(handle.HandleRef) == canonicalRef {
		if err := cerrarRuntimeHandlesSupersededPorCanonicoTMUX(handle); err != nil {
			return nil, err
		}
		return handle, nil
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte='tmux',
		    handle_kind='session',
		    handle_ref=?,
		    last_seen_at=CURRENT_TIMESTAMP
	WHERE id=?`, canonicalRef, handle.ID); err != nil {
		return nil, err
	}
	runtimeHandleHotReset()
	fresh, err := getRuntimeHandleRaw(handle.ID)
	if err != nil || fresh == nil {
		return fresh, err
	}
	if err := cerrarRuntimeHandlesSupersededPorCanonicoTMUX(fresh); err != nil {
		return nil, err
	}
	return fresh, nil
}

func cerrarRuntimeHandlesSupersededPorCanonicoTMUX(handle *RuntimeHandle) error {
	if handle == nil || handle.ID <= 0 || !runtimeHandleEsTMUXCanonico(handle) {
		return nil
	}
	now := time.Now().UTC()
	candidatos, err := listarRuntimeHandlesActivosCandidatos(handle.Agente, handle.ProyectoID)
	if err != nil {
		return err
	}
	for _, candidato := range candidatos {
		if candidato == nil || candidato.ID == handle.ID {
			continue
		}
		if !runtimeHandlePreferible(handle, candidato, now) {
			continue
		}
		if err := marcarRuntimeHandleSuperseded(candidato); err != nil {
			return err
		}
	}
	return nil
}

func compactarMetadataHandleRuntimePersistida(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil || handle.ID <= 0 {
		return handle, nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		return handle, nil
	}
	before, _ := json.Marshal(meta)
	compactarMetadataRuntimeHandle(meta)
	after, _ := json.Marshal(meta)
	if string(before) == string(after) {
		return handle, nil
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, string(after), handle.ID); err != nil {
		return nil, err
	}
	runtimeHandleHotReset()
	return getRuntimeHandleRaw(handle.ID)
}

func SincronizarRuntimeHandleWorkingDir(handle *RuntimeHandle, runtime *RuntimeInstance, agente string, proyectoID *int64) (*RuntimeHandle, string, error) {
	current := ""
	metaWorkingDir := ""
	metaCWD := ""
	if handle != nil {
		meta := mapFromJSON(handle.MetadataJSON)
		metaCWD = strings.TrimSpace(stringFromMap(meta, "cwd", ""))
		metaWorkingDir = strings.TrimSpace(stringFromMap(meta, "working_dir", ""))
	}
	if runtime != nil {
		current = strings.TrimSpace(runtime.CWD)
	}
	if current == "" && handle != nil {
		current = metaCWD
		if current == "" {
			current = metaWorkingDir
		}
	}
	if proyectoID == nil && runtime != nil {
		proyectoID = runtime.ProyectoID
	}
	if proyectoID == nil || *proyectoID <= 0 {
		return handle, current, nil
	}
	proyecto, err := GetProyecto(jsonNumber(*proyectoID))
	if err != nil || proyecto == nil {
		return handle, current, err
	}
	preferred := RutaTrabajoPreferidaAgenteProyecto(agente, proyecto, current)
	if preferred == "" {
		preferred = current
	}
	sessionID := int64(0)
	if handle != nil && handle.SesionID != nil && *handle.SesionID > 0 {
		sessionID = *handle.SesionID
	}
	if preferred == current {
		handleNeedsRepair := handle != nil && preferred != "" && (strings.TrimSpace(metaWorkingDir) != preferred || strings.TrimSpace(metaCWD) != preferred)
		runtimeNeedsRepair := runtime != nil && runtime.ID > 0 && preferred != "" && strings.TrimSpace(runtime.CWD) != preferred
		sessionNeedsRepair := sessionID > 0 && preferred != ""
		if handleNeedsRepair {
			meta := mapFromJSON(handle.MetadataJSON)
			if meta == nil {
				meta = map[string]any{}
			}
			meta["working_dir"] = preferred
			meta["cwd"] = preferred
			metaJSON, _ := json.Marshal(meta)
			if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, string(metaJSON), handle.ID); err != nil {
				return nil, "", err
			}
		}
		if runtimeNeedsRepair {
			if _, err := DB.Exec(`UPDATE runtime_instances SET cwd=?, last_event_at=CURRENT_TIMESTAMP WHERE id=?`, preferred, runtime.ID); err != nil {
				return nil, "", err
			}
		}
		if sessionNeedsRepair {
			if _, err := DB.Exec(`UPDATE sesiones SET cwd=? WHERE id=?`, preferred, sessionID); err != nil {
				return nil, "", err
			}
		}
		if handleNeedsRepair || runtimeNeedsRepair || sessionNeedsRepair {
			runtimeHandleHotReset()
			fresh, err := GetRuntimeHandle(handle.ID)
			if err != nil {
				return nil, "", err
			}
			return fresh, preferred, nil
		}
		return handle, preferred, nil
	}
	if handle != nil {
		meta := mapFromJSON(handle.MetadataJSON)
		if meta == nil {
			meta = map[string]any{}
		}
		meta["working_dir"] = preferred
		meta["cwd"] = preferred
		metaJSON, _ := json.Marshal(meta)
		if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, string(metaJSON), handle.ID); err != nil {
			return nil, "", err
		}
		runtimeHandleHotReset()
	}
	if runtime != nil && runtime.ID > 0 && strings.TrimSpace(runtime.CWD) != preferred {
		if _, err := DB.Exec(`UPDATE runtime_instances SET cwd=?, last_event_at=CURRENT_TIMESTAMP WHERE id=?`, preferred, runtime.ID); err != nil {
			return nil, "", err
		}
	}
	if sessionID > 0 {
		if _, err := DB.Exec(`UPDATE sesiones SET cwd=? WHERE id=?`, preferred, sessionID); err != nil {
			return nil, "", err
		}
	}
	if handle == nil {
		return nil, preferred, nil
	}
	fresh, err := GetRuntimeHandle(handle.ID)
	if err != nil {
		return nil, "", err
	}
	return fresh, preferred, nil
}

func runtimeHandleEffectiveExternalSessionID(handle *RuntimeHandle, runtime *RuntimeInstance) (string, error) {
	if handle == nil {
		return "", nil
	}
	if ext := strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "external_session_id", "")); ext != "" {
		return ext, nil
	}
	if runtime != nil && strings.TrimSpace(runtime.ExternalSessionID) != "" {
		return strings.TrimSpace(runtime.ExternalSessionID), nil
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		sesion, err := sesionIfExists(handle.SesionID)
		if err != nil {
			return "", err
		}
		if sesion != nil && strings.TrimSpace(sesion.ExternalSessionID) != "" {
			return strings.TrimSpace(sesion.ExternalSessionID), nil
		}
	}
	if runtime != nil && runtime.ID > 0 {
		if sample, err := NormalizarUltimaMuestraRuntime(runtime.ID); err == nil && sample != nil && strings.TrimSpace(sample.SessionRef) != "" {
			return strings.TrimSpace(sample.SessionRef), nil
		}
	}
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	return controlruntime.DetectExternalSessionID(obj)
}

func runtimeHandleExternalSessionIncompatible(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.ToLower(strings.TrimSpace(runtimepolicy.RuntimeHandleDriver(handle.MetadataJSON)))
	if driver != "tmux_cli_session" &&
		!strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		!strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") {
		return false
	}
	externalSessionID := strings.ToLower(strings.TrimSpace(stringFromMap(meta, "external_session_id", "")))
	if externalSessionID == "" {
		if effective, err := runtimeHandleEffectiveExternalSessionID(handle, nil); err == nil {
			externalSessionID = strings.ToLower(strings.TrimSpace(effective))
		}
	}
	if !strings.HasPrefix(externalSessionID, "ollama-pool-") {
		return false
	}
	commandHints := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(handle.HandleRef),
		strings.TrimSpace(stringFromMap(meta, "rendered_command", "")),
		strings.TrimSpace(stringFromMap(meta, "wrapped_command", "")),
		strings.TrimSpace(stringFromMap(meta, "command", "")),
		strings.TrimSpace(stringFromMap(meta, "herramienta", "")),
		strings.TrimSpace(stringFromMap(meta, "connector", "")),
		strings.TrimSpace(stringFromMap(meta, "conector", "")),
	}, " "))
	if strings.Contains(commandHints, "ollama run ") || strings.Contains(commandHints, "ollama serve") || strings.Contains(commandHints, "ollama_pool_local") {
		return false
	}
	for _, token := range []string{"claude", "gemini", "codex"} {
		if strings.Contains(commandHints, token) {
			return true
		}
	}
	return false
}
