package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func runtimeOrderTiposBootstrap() []string {
	return []string{"handoff", "resume", "start"}
}

func runtimeOrderTipoBootstrap(tipo string) bool {
	switch strings.TrimSpace(tipo) {
	case "handoff", "resume", "start":
		return true
	default:
		return false
	}
}

func runtimeOrderTipoDespachable(tipo string) bool {
	switch strings.TrimSpace(tipo) {
	case "sync_status", "checkpoint", "nudge", "discordia", "start", "pause", "resume", "stop", "restart", "send_instruction", "handoff":
		return true
	default:
		return false
	}
}

func runtimeOrderBootstrapStaleCoveredByFollowup(order *RuntimeOrder) (bool, string, string, error) {
	if order == nil {
		return false, "", "", nil
	}
	tipo := strings.TrimSpace(order.Tipo)
	if tipo != "handoff" && tipo != "resume" && tipo != "start" {
		return false, "", "", nil
	}
	result := mapFromJSON(order.ResultadoJSON)
	if len(result) == 0 {
		return false, "", "", nil
	}
	startOrderID := int64FromAny(result["start_order_id"])
	if startOrderID <= 0 || startOrderID == order.ID {
		return false, "", "", nil
	}
	startOrder, err := GetRuntimeOrder(startOrderID)
	if err != nil {
		return false, "", "", err
	}
	if startOrder == nil {
		return false, "", "", nil
	}
	if !startOrder.CreatedAt.IsZero() && !order.CreatedAt.IsZero() && startOrder.CreatedAt.Before(order.CreatedAt) {
		return false, "", "", nil
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":                true,
		"obsoleta":          true,
		"superseded":        true,
		"superseded_reason": "bootstrap_followup_order_exists",
		"start_order_id":    startOrderID,
	})
	return true, resultado, "bootstrap cubierto por start_order_id más reciente", nil
}

func reconciliarRuntimeOrderStale(id int64, tipo string, now, cutoff time.Time) (bool, error) {
	order, err := GetRuntimeOrder(id)
	if err != nil {
		return false, err
	}
	if order != nil {
		switch strings.TrimSpace(order.Tipo) {
		case "start", "resume", "stop":
			runtime, handle, err := resolverDestinoRuntimeOrderCanonico(order, nil, nil)
			if err != nil {
				return false, err
			}
			reason := ""
			if satisfied, satisfiedReason := runtimeOrderControlEstadoDeseadoSatisfecho(order, runtime, handle); satisfied {
				reason = satisfiedReason
			} else if runtimeOrderControlObsoletaPorWorkerRecuperado(order, runtime, handle) {
				reason = "runtime_worker_recovered_after_order"
			}
			if strings.TrimSpace(reason) != "" {
				if err := runtimeOrderPromoverEstadoObservadoSiSatisfecha(order, runtime, handle); err != nil {
					return false, err
				}
				resultado := map[string]any{
					"ok":       true,
					"obsoleta": true,
					"reason":   reason,
				}
				if runtime != nil && runtime.ID > 0 {
					resultado["runtime_id"] = runtime.ID
				}
				if handle != nil && handle.ID > 0 {
					resultado["handle_id"] = handle.ID
				}
				data, _ := json.Marshal(resultado)
				if err := MarcarRuntimeOrderEstado(id, "completada", string(data), ""); err != nil {
					return false, err
				}
				return true, nil
			}
		}
	}
	if order != nil && strings.TrimSpace(order.Tipo) == "send_instruction" {
		payload := mapFromJSON(order.PayloadJSON)
		if handled, err := reconciliarRuntimeOrderSendInstructionConMailboxActual(order, payload, now); err != nil {
			return false, err
		} else if handled {
			return true, nil
		}
		if reason, err := runtimeOrderSendInstructionDirectaAbsorbidaPorTrabajo(order, payload, now); err != nil {
			return false, err
		} else if reason != "" {
			if err := completarRuntimeOrderSendInstructionSupersedida(order, payload, reason); err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return false, err
				}
			}
			return true, nil
		}
	}
	if covered, resultado, detalle, coveredErr := runtimeOrderBootstrapStaleCoveredByFollowup(order); coveredErr != nil {
		return false, coveredErr
	} else if covered {
		if err := MarcarRuntimeOrderEstado(id, "completada", resultado, detalle); err != nil {
			return false, err
		}
		return true, nil
	}
	var (
		res     sql.Result
		execErr error
	)
	if runtimeOrderTipoDespachable(tipo) {
		res, execErr = DB.Exec(`
			UPDATE runtime_orders
			SET estado = 'pendiente',
			    started_at = NULL,
			    finished_at = NULL,
			    claimed_by = '',
			    lease_token = '',
			    lease_expires_at = NULL,
			    available_at = CURRENT_TIMESTAMP,
			    error_text = CASE
			        WHEN TRIM(COALESCE(error_text, '')) = '' THEN 'reencolada tras stale del control plane'
			        ELSE error_text || ?
			    END
			WHERE id = ?
			  AND estado IN ('tomada','ejecutando')
			  AND (
			        (lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
			     OR COALESCE(started_at, updated_at, created_at) <= ?
			  )`, "\nreencolada tras stale del control plane", id, now, cutoff)
	} else {
		res, execErr = DB.Exec(`
			UPDATE runtime_orders
			SET estado = 'expirada',
			    finished_at = CURRENT_TIMESTAMP,
			    claimed_by = '',
			    lease_token = '',
			    lease_expires_at = NULL,
			    error_text = CASE
			        WHEN TRIM(COALESCE(error_text, '')) = '' THEN 'orden expirada por stale sin dispatcher compatible'
			        ELSE error_text || ?
			    END
			WHERE id = ?
			  AND estado IN ('tomada','ejecutando')
			  AND (
			        (lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
			     OR COALESCE(started_at, updated_at, created_at) <= ?
			  )`, "\norden expirada por stale sin dispatcher compatible", id, now, cutoff)
	}
	if execErr != nil {
		return false, execErr
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}
