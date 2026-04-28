package db

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

var runtimeOrdersAsyncLongRunningInFlight atomic.Int32

func procesarRuntimeOrdersBatchTipos(tipos []string) (int, error) {
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	orders, err := seleccionarRuntimeOrdersBatch(SeleccionRuntimeOrdersBatch{
		Estado:       "pendiente",
		Tipos:        tipos,
		Limit:        limit,
		Now:          time.Now().UTC(),
		SortLess:     runtimeOrderBatchLess,
		RequireReady: true,
	})
	if err != nil {
		return 0, err
	}

	processed := 0
	start := time.Now()
	budget := runtimeOrdersBatchBudget()
	for _, candidate := range orders {
		if candidate == nil || candidate.ID <= 0 {
			continue
		}
		if budget > 0 && time.Since(start) >= budget {
			Audit("server", "runtime_orders_batch_deferred", "runtime_order", 0,
				fmt.Sprintf("budget=%s processed=%d", budget, processed))
			break
		}
		asyncReserved := false
		if runtimeOrderDebeEjecutarseAsync(candidate) {
			if !runtimeOrderReservarAsync() {
				continue
			}
			asyncReserved = true
		}
		order, err := claimRuntimeOrderByID(candidate.ID)
		if err != nil {
			if asyncReserved {
				runtimeOrderLiberarAsync()
			}
			return processed, err
		}
		if order == nil {
			if asyncReserved {
				runtimeOrderLiberarAsync()
			}
			continue
		}
		if err := MarcarRuntimeOrderEjecutando(order); err != nil {
			if asyncReserved {
				runtimeOrderLiberarAsync()
			}
			return processed, err
		}
		if asyncReserved {
			runtimeOrderLanzarAsync(order)
			processed++
			continue
		}
		if err := ejecutarRuntimeOrderBasicaFn(order); err != nil {
			_ = MarcarRuntimeOrderEstado(order.ID, "fallida", `{"ok":false}`, err.Error())
			processed++
			continue
		}
		processed++
	}
	return processed, nil
}

func runtimeOrderDebeEjecutarseAsync(order *RuntimeOrder) bool {
	if order == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(order.Tipo)) {
	case "start", "resume", "restart":
		if transporte, ok := runtimeOrderInferirTransporteControl(order); ok {
			switch strings.ToLower(strings.TrimSpace(transporte)) {
			case "api", "mcp_http", "otro":
				return false
			}
		}
		return true
	case "send_instruction":
		return runtimeOrderSendInstructionDebeEjecutarseAsync(order)
	default:
		return false
	}
}

func runtimeOrderSendInstructionDebeEjecutarseAsync(order *RuntimeOrder) bool {
	if order == nil || !strings.EqualFold(strings.TrimSpace(order.Tipo), "send_instruction") {
		return false
	}
	payload := mapFromJSON(order.PayloadJSON)
	transporte, transporteKnown := runtimeOrderInferirTransporteControl(order)
	return runtimeOrderSendInstructionDebeEjecutarseAsyncConTransporte(payload, transporte, transporteKnown)
}

func runtimeOrderSendInstructionDebeEjecutarseAsyncConTransporte(payload map[string]any, transporte string, transporteKnown bool) bool {
	if runtimeOrderSendInstructionProvieneMailbox(payload) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(stringFromMap(payload, "mailbox_kind", ""))) {
	case "pipeline_local", "microprogramacion", "autonomia", "nudge", "watchdog",
		strings.ToLower(MailboxKindGovernanceRefresh), strings.ToLower(MailboxKindSkillsRefresh):
		return true
	}
	if transporteKnown {
		switch strings.ToLower(strings.TrimSpace(transporte)) {
		case "api", "mcp_http", "otro":
			return false
		default:
			return true
		}
	}
	return true
}

func runtimeOrderInferirTransporteControl(order *RuntimeOrder) (string, bool) {
	if order == nil {
		return "", false
	}
	if handle, err := resolverHandleParaOrden(order); err == nil && handle != nil {
		if transporte := strings.TrimSpace(handle.Transporte); transporte != "" {
			return transporte, true
		}
	}
	if runtime, err := resolverRuntimeParaOrden(order); err == nil && runtime != nil {
		if conector, err := GetConector(strings.TrimSpace(runtime.Connector)); err == nil && conector != nil {
			if transporte := strings.TrimSpace(conector.Transporte); transporte != "" {
				return transporte, true
			}
		}
	}
	payload := mapFromJSON(order.PayloadJSON)
	if conectorRef := strings.TrimSpace(stringFromMap(payload, "conector", "")); conectorRef != "" {
		if conector, err := GetConector(conectorRef); err == nil && conector != nil {
			if transporte := strings.TrimSpace(conector.Transporte); transporte != "" {
				return transporte, true
			}
		}
	}
	return "", false
}

func runtimeOrderAsyncLimit() int32 {
	return 4
}

func runtimeOrderReservarAsync() bool {
	limit := runtimeOrderAsyncLimit()
	if limit <= 0 {
		return false
	}
	for {
		current := runtimeOrdersAsyncLongRunningInFlight.Load()
		if current >= limit {
			return false
		}
		if runtimeOrdersAsyncLongRunningInFlight.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func runtimeOrderLiberarAsync() {
	for {
		current := runtimeOrdersAsyncLongRunningInFlight.Load()
		if current <= 0 {
			return
		}
		if runtimeOrdersAsyncLongRunningInFlight.CompareAndSwap(current, current-1) {
			return
		}
	}
}

func runtimeOrderLanzarAsync(order *RuntimeOrder) {
	go func(order *RuntimeOrder) {
		defer runtimeOrderLiberarAsync()
		defer func() {
			if recovered := recover(); recovered != nil {
				_ = MarcarRuntimeOrderEstado(order.ID, "fallida", `{"ok":false}`, fmt.Sprintf("panic runtime order async: %v", recovered))
			}
		}()
		if err := ejecutarRuntimeOrderBasica(order); err != nil {
			_ = MarcarRuntimeOrderEstado(order.ID, "fallida", `{"ok":false}`, err.Error())
		}
	}(order)
}

func runtimeOrderBatchLess(left, right *RuntimeOrder) bool {
	leftPriority := runtimeOrderBatchPriority(left)
	rightPriority := runtimeOrderBatchPriority(right)
	if leftPriority != rightPriority {
		return leftPriority < rightPriority
	}
	leftMoment := runtimeOrderBatchMoment(left)
	rightMoment := runtimeOrderBatchMoment(right)
	if !leftMoment.Equal(rightMoment) {
		return leftMoment.Before(rightMoment)
	}
	if left == nil {
		return false
	}
	if right == nil {
		return true
	}
	return left.ID < right.ID
}

func runtimeOrderBatchPriority(order *RuntimeOrder) int {
	if order == nil {
		return 99
	}
	switch strings.ToLower(strings.TrimSpace(order.Tipo)) {
	case "stop":
		return 0
	case "pause", "restart":
		return 1
	case "resume", "start":
		return 2
	case "sync_status", "checkpoint":
		return 3
	case "send_instruction":
		return 4
	case "handoff":
		return 5
	case "nudge", "discordia":
		return 6
	default:
		return 10
	}
}

func runtimeOrderBatchMoment(order *RuntimeOrder) time.Time {
	if order == nil {
		return time.Unix(1<<62, 0).UTC()
	}
	if !order.AvailableAt.IsZero() {
		return order.AvailableAt.UTC()
	}
	if !order.UpdatedAt.IsZero() {
		return order.UpdatedAt.UTC()
	}
	return order.CreatedAt.UTC()
}

func reconciliarRuntimeOrdersPendientesMailboxConsumido() (int, error) {
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{
		Estado: &estado,
		Tipos:  []string{"send_instruction"},
		Limit:  limit,
	})
	if err != nil {
		return 0, err
	}
	ids := make([]int64, 0, limit)
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" {
			continue
		}
		ids = append(ids, order.ID)
		if len(ids) >= limit {
			break
		}
	}

	processed := 0
	for _, id := range ids {
		order, err := GetRuntimeOrder(id)
		if err != nil {
			return processed, err
		}
		if order == nil || !strings.EqualFold(strings.TrimSpace(order.Estado), "pendiente") || strings.TrimSpace(order.Tipo) != "send_instruction" {
			continue
		}
		payload := mapFromJSON(order.PayloadJSON)
		handled, err := reconciliarRuntimeOrderSendInstructionConMailboxActual(order, payload, time.Now().UTC())
		if err != nil {
			return processed, err
		}
		if handled {
			processed++
		}
	}
	return processed, nil
}

func reconciliarRuntimeOrdersPendientesControlObsoletas() (int, error) {
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	orders, err := seleccionarRuntimeOrdersBatch(SeleccionRuntimeOrdersBatch{
		Estado:       "pendiente",
		Tipos:        []string{"start", "resume", "stop"},
		Limit:        limit,
		Now:          time.Now().UTC(),
		SortLess:     runtimeOrderByIDLess,
		RequireReady: true,
	})
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, candidate := range orders {
		if candidate == nil || candidate.ID <= 0 {
			continue
		}
		order, err := getRuntimeOrderPreferHotIndex(candidate.ID)
		if err != nil {
			return processed, err
		}
		if order == nil || !strings.EqualFold(strings.TrimSpace(order.Estado), "pendiente") {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(order.Tipo), "start") {
			if sesionAjena, err := runtimeOrderStartSesionAbiertaAjena(order.Agente, order.ProyectoID); err != nil {
				return processed, err
			} else if sesionAjena != nil {
				proyectoAjenoID := int64(0)
				if sesionAjena.ProyectoID != nil {
					proyectoAjenoID = *sesionAjena.ProyectoID
				}
				reason := fmt.Sprintf("sesion_activa_en_otro_proyecto:%d", proyectoAjenoID)
				if !SesionEsOperativa(sesionAjena) {
					if err := cerrarSesionFantasmaActiva(sesionAjena.ID); err != nil {
						return processed, err
					}
					reason = fmt.Sprintf("sesion_fantasma_en_otro_proyecto:%d", proyectoAjenoID)
				} else if err := asegurarStopSesionActivaAjena(order, sesionAjena); err != nil {
					return processed, err
				}
				if err := reencolarRuntimeOrderControlAt(order, reason, time.Now().UTC().Add(runtimeOrderControlRetryDelay())); err != nil {
					return processed, err
				}
				processed++
				continue
			}
		}
		runtime, handle, err := resolverDestinoRuntimeOrderCanonico(order, nil, nil)
		if err != nil {
			return processed, err
		}
		reason := ""
		if satisfied, satisfiedReason := runtimeOrderControlEstadoDeseadoSatisfecho(order, runtime, handle); satisfied {
			reason = satisfiedReason
		} else {
			if !runtimeOrderControlObsoletaPorWorkerRecuperado(order, runtime, handle) {
				continue
			}
			reason = "runtime_worker_recovered_after_order"
		}
		if err := runtimeOrderPromoverEstadoObservadoSiSatisfecha(order, runtime, handle); err != nil {
			return processed, err
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
		if err := MarcarRuntimeOrderEstado(order.ID, "completada", string(data), ""); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func reconciliarRuntimeOrdersEjecutandoControlSatisfechas() (int, error) {
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	orders, err := seleccionarRuntimeOrdersBatch(SeleccionRuntimeOrdersBatch{
		Estado:   "ejecutando",
		Tipos:    []string{"start", "resume", "stop"},
		Limit:    limit,
		SortLess: runtimeOrderByIDLess,
	})
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, candidate := range orders {
		if candidate == nil || candidate.ID <= 0 {
			continue
		}
		order, err := getRuntimeOrderPreferHotIndex(candidate.ID)
		if err != nil {
			return processed, err
		}
		if order == nil || !strings.EqualFold(strings.TrimSpace(order.Estado), "ejecutando") {
			continue
		}
		runtime, handle, err := resolverDestinoRuntimeOrderCanonico(order, nil, nil)
		if err != nil {
			return processed, err
		}
		reason := ""
		if satisfied, satisfiedReason := runtimeOrderControlEstadoDeseadoSatisfecho(order, runtime, handle); satisfied {
			reason = satisfiedReason
		} else if runtimeOrderControlObsoletaPorWorkerRecuperado(order, runtime, handle) {
			reason = "runtime_worker_recovered_after_order"
		}
		if strings.TrimSpace(reason) == "" {
			continue
		}
		if err := runtimeOrderPromoverEstadoObservadoSiSatisfecha(order, runtime, handle); err != nil {
			return processed, err
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
		if err := MarcarRuntimeOrderEstado(order.ID, "completada", string(data), ""); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}
