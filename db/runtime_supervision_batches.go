package db

import (
	"fmt"
	"orquesta/internal/controlruntime"
	"strings"
	"time"
)

var runtimeSupervisionHandleFn = supervisarRuntimeHandle
var runtimeSupervisionBatchBudgetOverride time.Duration
var runtimeOrdersBatchBudgetOverride time.Duration
var ejecutarRuntimeOrderBasicaFn = ejecutarRuntimeOrderBasica
var runtimeTMUXCommandPathFn = controlruntime.TMUXCommandPath
var runtimeTMUXListSessionsFn = controlruntime.ListTMUXSessions
var runtimeTMUXKillSessionFn = controlruntime.KillTMUXSession

func runtimeSupervisionBatchBudget() time.Duration {
	if runtimeSupervisionBatchBudgetOverride > 0 {
		return runtimeSupervisionBatchBudgetOverride
	}
	seconds := configIntOrDefault("runtime_supervision_batch_budget_seconds", 2)
	if seconds <= 0 {
		seconds = 2
	}
	return time.Duration(seconds) * time.Second
}

func ProcesarRuntimeSupervisionBatch() (int, error) {
	limit := configIntOrDefault("runtime_supervision_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	intervalSeconds := configIntOrDefault("runtime_supervision_interval_seconds", 60)
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	if intervalSeconds < 60 {
		intervalSeconds = 60
	}
	cutoff := time.Now().UTC().Add(-time.Duration(intervalSeconds) * time.Second)

	rows, err := DB.Query(runtimeHandleSelectBase()+`
		WHERE estado IN ('activo','pausado')
		  AND COALESCE(last_seen_at, updated_at, created_at) <= ?
		ORDER BY COALESCE(last_seen_at, updated_at, created_at), id
		LIMIT ?`, cutoff, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	handles := make([]*RuntimeHandle, 0, limit)
	for rows.Next() {
		handle, err := scanRuntimeHandle(rows)
		if err != nil {
			return 0, err
		}
		if handle != nil {
			handles = append(handles, handle)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	processed := 0
	start := time.Now()
	budget := runtimeSupervisionBatchBudget()
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if budget > 0 && time.Since(start) >= budget {
			Audit("server", "runtime_supervision_batch_deferred", "runtime_handle", 0,
				fmt.Sprintf("budget=%s processed=%d", budget, processed))
			break
		}
		if abierta, err := existeRuntimeOrderAbiertaAgenteProyecto(strings.TrimSpace(handle.Agente), handle.ProyectoID, 0,
			"sync_status", "start", "resume", "pause", "stop", "restart", "handoff"); err != nil {
			return processed, err
		} else if abierta {
			continue
		}
		runtime, err := runtimeHandleRuntime(handle)
		if err != nil {
			return processed, err
		}
		if _, err := runtimeSupervisionHandleFn(handle, runtime, "runtime_supervision"); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func ProcesarRuntimeOrdersBasicasBatch() (int, error) {
	return procesarRuntimeOrdersBatchTipos(runtimeOrderTiposBasicos())
}

func runtimeOrdersBatchBudget() time.Duration {
	if runtimeOrdersBatchBudgetOverride > 0 {
		return runtimeOrdersBatchBudgetOverride
	}
	seconds := configIntOrDefault("runtime_orders_batch_budget_seconds", 2)
	if seconds <= 0 {
		seconds = 2
	}
	return time.Duration(seconds) * time.Second
}
