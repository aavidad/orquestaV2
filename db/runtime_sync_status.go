package db

import (
	"encoding/json"
	"fmt"
	"orquesta/internal/controlruntime"
	"strings"
)

func errorObservacionLocalIgnorable(err error) bool {
	if err == nil {
		return false
	}
	raw := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case raw == "":
		return false
	case strings.Contains(raw, "can't find session"):
		return true
	case strings.Contains(raw, "no server running"):
		return true
	case strings.Contains(raw, "failed to connect to server"):
		return true
	case strings.Contains(raw, "error connecting to /tmp/tmux"):
		return true
	case strings.Contains(raw, "tmux session missing"):
		return true
	default:
		return false
	}
}

func ejecutarRuntimeOrderBasica(order *RuntimeOrder) error {
	switch strings.TrimSpace(order.Tipo) {
	case "sync_status":
		return ejecutarRuntimeOrderSyncStatus(order)
	case "checkpoint":
		return ejecutarRuntimeOrderCheckpoint(order)
	case "nudge":
		return ejecutarRuntimeOrderNudge(order)
	case "discordia":
		return ejecutarRuntimeOrderDiscordia(order)
	case "start":
		return ejecutarRuntimeOrderStart(order)
	case "pause":
		return ejecutarRuntimeOrderPause(order)
	case "resume":
		return ejecutarRuntimeOrderResume(order)
	case "stop":
		return ejecutarRuntimeOrderStop(order)
	case "restart":
		return ejecutarRuntimeOrderRestart(order)
	case "send_instruction":
		return ejecutarRuntimeOrderSendInstruction(order)
	case "handoff":
		return ejecutarRuntimeOrderHandoff(order)
	default:
		return fmt.Errorf("tipo de orden aún no soportado por el dispatcher básico: %s", order.Tipo)
	}
}

func ejecutarRuntimeOrderSyncStatus(order *RuntimeOrder) error {
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return err
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return err
	}
	result, err := supervisarRuntimeHandle(handle, runtime, "runtime_sync_status")
	if err != nil {
		return err
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func supervisarRuntimeHandle(handle *RuntimeHandle, runtime *RuntimeInstance, source string) (map[string]any, error) {
	result := map[string]any{
		"ok":               true,
		"handle_id":        nil,
		"runtime_id":       nil,
		"handle":           nil,
		"runtime":          nil,
		"sin_handle":       handle == nil,
		"sin_runtime":      runtime == nil,
		"observed_remote":  false,
		"observed_process": false,
	}
	if handle != nil {
		estadoRemoto, observedRemote, err := controlruntime.ConsultarEstadoRemoto(controlruntime.ObjetivoProceso{
			HandleKind:   handle.HandleKind,
			HandleRef:    handle.HandleRef,
			MetadataJSON: handle.MetadataJSON,
		})
		if err != nil {
			failures, degraded, obsErr := registrarFalloObservacionRemota(handle, runtime, err)
			if obsErr != nil {
				return nil, obsErr
			}
			result["ok"] = false
			result["observed_remote_error"] = true
			result["remote_sync_failures"] = failures
			result["remote_degraded"] = degraded
			result["remote_error"] = strings.TrimSpace(err.Error())
		} else if observedRemote && estadoRemoto != nil {
			if err := aplicarEstadoRemotoObservado(handle, runtime, estadoRemoto); err != nil {
				return nil, err
			}
			result["observed_remote"] = true
			if strings.TrimSpace(estadoRemoto.RawJSON) != "" {
				result["remote_status"] = json.RawMessage(estadoRemoto.RawJSON)
			}
			if err := marcarSesionHeartbeatSupervisada(handle, runtime, nil, strings.TrimSpace(estadoRemoto.LogicalState)); err != nil {
				return nil, err
			}
			if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, source); err != nil {
				return nil, err
			}
		} else if observedProcess, processResult, err := observarProcesoLocalRuntime(handle, runtime, source); err != nil {
			return nil, err
		} else if observedProcess {
			result["observed_process"] = true
			for k, v := range processResult {
				result[k] = v
			}
		}
		if handle.ID > 0 {
			handle, err = GetRuntimeHandle(handle.ID)
			if err != nil {
				return nil, err
			}
		}
	}
	if runtime != nil && runtime.ID > 0 {
		freshRuntime, err := runtimeInstanceIfExists(&runtime.ID)
		if err != nil {
			return nil, err
		}
		if freshRuntime != nil {
			runtime = freshRuntime
		}
	}
	if handle != nil {
		result["handle_id"] = handle.ID
		result["handle"] = map[string]any{
			"estado":      handle.Estado,
			"transporte":  handle.Transporte,
			"handle_kind": handle.HandleKind,
			"handle_ref":  handle.HandleRef,
		}
	}
	if runtime != nil {
		result["runtime_id"] = runtime.ID
		result["runtime"] = map[string]any{
			"logical_state": runtime.LogicalState,
			"process_state": runtime.ProcessState,
			"connector":     runtime.Connector,
			"pid":           runtime.PID,
		}
	}
	return result, nil
}

func observarProcesoLocalRuntime(handle *RuntimeHandle, runtime *RuntimeInstance, source string) (bool, map[string]any, error) {
	if handle == nil {
		return false, nil, nil
	}
	if runtimeHandleTMUXCurrentPathMismatch(handle) && !runtimeHandlePreservesExternalSession(handle) {
		if err := marcarRuntimeHandleFantasma(handle); err != nil {
			return true, map[string]any{
				"process_alive": false,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		return true, map[string]any{
			"process_alive":  false,
			"observed_local": true,
			"process_error":  "tmux current_path stale or deleted",
		}, nil
	}
	if repaired, restarted, err := controlruntime.EnsureTMUXMonitorFromMetadataJSON(handle.MetadataJSON); err != nil {
		if errorObservacionLocalIgnorable(err) {
			if markErr := marcarProcesoLocalNoDisponible(handle, runtime); markErr != nil {
				return true, map[string]any{
					"process_alive": false,
					"process_error": strings.TrimSpace(markErr.Error()),
				}, markErr
			}
			return true, map[string]any{
				"process_alive": false,
				"process_error": strings.TrimSpace(err.Error()),
			}, nil
		}
		return true, map[string]any{
			"process_alive": false,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	} else if strings.TrimSpace(repaired) != "" && strings.TrimSpace(repaired) != strings.TrimSpace(handle.MetadataJSON) {
		if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, repaired, handle.ID); err != nil {
			return true, map[string]any{
				"process_alive": false,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		runtimeHandleHotReset()
		handle.MetadataJSON = repaired
		if restarted {
			handle.Estado = "activo"
		}
	}
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	estado, observed, err := controlruntime.ConsultarEstadoLocal(obj)
	if err != nil {
		if errorObservacionLocalIgnorable(err) {
			if markErr := marcarProcesoLocalNoDisponible(handle, runtime); markErr != nil {
				return true, map[string]any{
					"process_alive": false,
					"process_error": strings.TrimSpace(markErr.Error()),
				}, markErr
			}
			return true, map[string]any{
				"process_alive": false,
				"process_error": strings.TrimSpace(err.Error()),
			}, nil
		}
		return true, map[string]any{
			"process_alive": false,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	}
	if !observed {
		vivo, pid, err := controlruntime.ProcesoVivo(obj)
		if err != nil {
			if errorObservacionLocalIgnorable(err) {
				if markErr := marcarProcesoLocalNoDisponible(handle, runtime); markErr != nil {
					return true, map[string]any{
						"process_alive": false,
						"process_error": strings.TrimSpace(markErr.Error()),
					}, markErr
				}
				return true, map[string]any{
					"process_alive": false,
					"process_error": strings.TrimSpace(err.Error()),
				}, nil
			}
			return true, map[string]any{
				"process_alive": false,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		if pid <= 0 {
			if revived, result, err := aplicarEstadoWorkerEstructuradoSinProceso(handle, runtime, source); revived || err != nil {
				return true, result, err
			}
			return false, nil, nil
		}
		estado = &controlruntime.EstadoLocal{
			PID:  pid,
			Vivo: vivo,
		}
		observed = true
	}
	if estado == nil {
		if revived, result, err := aplicarEstadoWorkerEstructuradoSinProceso(handle, runtime, source); revived || err != nil {
			return true, result, err
		}
		return true, map[string]any{
			"process_alive": false,
		}, nil
	}
	pid := estado.PID
	vivo := estado.Vivo
	if !vivo && pid == 0 {
		if revived, result, err := aplicarEstadoWorkerEstructuradoSinProceso(handle, runtime, source); revived || err != nil {
			return true, result, err
		}
		return true, map[string]any{
			"process_alive": false,
		}, nil
	}
	if superseded, err := runtimeHandleDebeCederAWorkerEstructurado(handle); err != nil {
		return true, map[string]any{
			"process_alive": vivo,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	} else if superseded {
		if err := marcarRuntimeHandleSuperseded(handle); err != nil {
			return true, map[string]any{
				"process_alive": vivo,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		return true, map[string]any{
			"process_alive":      vivo,
			"process_superseded": true,
		}, nil
	}
	if runtimeHandleBloqueaReactivacionPorCanalRoto(handle) {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return true, map[string]any{
				"process_alive": vivo,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		return true, map[string]any{
			"process_alive":  false,
			"observed_local": true,
			"process_error":  "runtime handle blocked by broken channel",
		}, nil
	}
	if err := aplicarEstadoLocalObservado(handle, estado); err != nil {
		return true, map[string]any{
			"process_alive": vivo,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	}
	if runtimeHandleBloqueaReactivacionPorCanalRoto(handle) {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return true, map[string]any{
				"process_alive": vivo,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		return true, map[string]any{
			"process_alive":  false,
			"observed_local": true,
			"process_error":  "runtime handle blocked by broken channel",
		}, nil
	}
	if !vivo {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return true, nil, err
		}
		return true, map[string]any{
			"process_alive": false,
		}, nil
	}

	estadoHandle := strings.TrimSpace(handle.Estado)
	if estadoHandle == "" {
		estadoHandle = "activo"
	}
	if estadoHandle != "pausado" {
		estadoHandle = "activo"
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, estadoHandle, handle.ID); err != nil {
		return true, nil, err
	}
	runtimeHandleHotReset()

	pid64 := int64(pid)
	logicalState := ""
	processState := ""
	if runtime != nil {
		runtimeSnapshot := *runtime
		runtimeSnapshot.PID = &pid64
		if sample, err := RegistrarMuestraGenericProcess(runtime.ID, &runtimeSnapshot); err == nil && sample != nil {
			logicalState = strings.TrimSpace(sample.LogicalState)
			if normalized, normErr := NormalizarMuestraRuntime(sample); normErr == nil && normalized != nil {
				processState = strings.TrimSpace(normalized.State)
			}
		} else {
			if _, updErr := DB.Exec(`
				UPDATE runtime_instances
				SET pid = ?,
				    process_state = ?,
				    last_event_at = CURRENT_TIMESTAMP,
				    last_heartbeat_at = CURRENT_TIMESTAMP
				WHERE id = ?`, pid64, "running", runtime.ID); updErr != nil {
				return true, nil, updErr
			}
		}
	}
	if logicalState == "" {
		if estadoHandle == "pausado" {
			logicalState = "pausado"
		} else {
			logicalState = "disponible"
		}
	}
	if processState == "" {
		if estadoHandle == "pausado" {
			processState = "stopped"
		} else {
			processState = "running"
		}
	}
	if workerLogicalState, workerProcessState, ok := runtimeObservedLogicalStateFromStructuredWorker(handle); ok {
		logicalState = workerLogicalState
		if strings.TrimSpace(workerProcessState) != "" {
			processState = workerProcessState
		}
	}
	if runtime != nil {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET pid = ?,
			    logical_state = ?,
			    process_state = ?,
			    last_event_at = CURRENT_TIMESTAMP,
			    last_heartbeat_at = CURRENT_TIMESTAMP
			WHERE id = ?`, pid64, logicalState, processState, runtime.ID); err != nil {
			return true, nil, err
		}
	}
	if handle != nil {
		syncedHandle, _, err := SincronizarRuntimeHandleExternalSessionID(handle, runtime)
		if err != nil {
			return true, nil, err
		}
		if syncedHandle != nil {
			handle = syncedHandle
		}
	}
	if err := marcarSesionHeartbeatSupervisada(handle, runtime, &pid64, logicalState); err != nil {
		return true, nil, err
	}
	if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, source); err != nil {
		return true, nil, err
	}
	return true, map[string]any{
		"process_alive":  true,
		"process_pid":    pid64,
		"process_state":  processState,
		"logical_state":  logicalState,
		"observed_local": true,
	}, nil
}
