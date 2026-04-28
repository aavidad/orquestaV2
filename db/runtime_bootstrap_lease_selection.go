package db

import (
	"strings"
	"time"
)

type bootstrapRuntimeLeaseSelection struct {
	order        *RuntimeOrder
	startOrderID int64
	mailboxIDs   []int64
	sesionID     int64
	score        int
}

func runtimeBootstrapLeaseSesionID(order, startOrder *RuntimeOrder) int64 {
	if order == nil {
		return 0
	}
	result := mapFromJSON(order.ResultadoJSON)
	if sesionID := int64FromAny(result["sesion_id"]); sesionID > 0 {
		return sesionID
	}
	if startOrder == nil {
		return 0
	}
	return int64FromAny(mapFromJSON(startOrder.ResultadoJSON)["sesion_id"])
}

func runtimeBootstrapLeaseAffinityScore(order, startOrder *RuntimeOrder, handle *RuntimeHandle, runtime *RuntimeInstance, sesionID int64) int {
	if order == nil {
		return 0
	}
	result := mapFromJSON(order.ResultadoJSON)
	score := 0
	if handle != nil && handle.ID > 0 {
		switch {
		case order.HandleID != nil && *order.HandleID == handle.ID:
			score += 8
		case int64FromAny(result["handle_id"]) == handle.ID:
			score += 8
		case startOrder != nil && startOrder.HandleID != nil && *startOrder.HandleID == handle.ID:
			score += 8
		case startOrder != nil && int64FromAny(mapFromJSON(startOrder.ResultadoJSON)["handle_id"]) == handle.ID:
			score += 8
		}
	}
	if runtime != nil && runtime.ID > 0 {
		switch {
		case order.RuntimeID != nil && *order.RuntimeID == runtime.ID:
			score += 4
		case int64FromAny(result["runtime_id"]) == runtime.ID:
			score += 4
		case startOrder != nil && startOrder.RuntimeID != nil && *startOrder.RuntimeID == runtime.ID:
			score += 4
		case startOrder != nil && int64FromAny(mapFromJSON(startOrder.ResultadoJSON)["runtime_id"]) == runtime.ID:
			score += 4
		}
	}
	if sesionID > 0 && runtimeBootstrapLeaseSesionID(order, startOrder) == sesionID {
		score += 2
	}
	return score
}

func runtimeBootstrapLeaseLinkedToStart(order *RuntimeOrder, startOrder *RuntimeOrder) bool {
	if order == nil || startOrder == nil {
		return false
	}
	if strings.TrimSpace(startOrder.Tipo) != "start" {
		return false
	}
	result := mapFromJSON(order.ResultadoJSON)
	return int64FromAny(result["start_order_id"]) == startOrder.ID
}

func runtimeBootstrapLeaseResumeDerivadoPrefiereStart(order *RuntimeOrder, startOrder *RuntimeOrder) bool {
	return order != nil &&
		strings.EqualFold(strings.TrimSpace(order.Tipo), "resume") &&
		runtimeBootstrapLeaseLinkedToStart(order, startOrder)
}

func runtimeBootstrapLeaseFuenteLigadaPrefiereSobreStart(order *RuntimeOrder, startOrder *RuntimeOrder) bool {
	return order != nil &&
		!strings.EqualFold(strings.TrimSpace(order.Tipo), "resume") &&
		runtimeBootstrapLeaseLinkedToStart(order, startOrder)
}

func runtimeBootstrapLeaseMailboxIDsVigentes(mailboxIDs []int64) ([]int64, error) {
	if len(mailboxIDs) == 0 {
		return nil, nil
	}
	out := make([]int64, 0, len(mailboxIDs))
	for _, mailboxID := range mailboxIDs {
		if mailboxID <= 0 {
			continue
		}
		msg, err := GetRuntimeMailbox(mailboxID)
		if err != nil {
			return nil, err
		}
		if msg == nil {
			continue
		}
		switch strings.TrimSpace(msg.Estado) {
		case "pendiente", "entregado":
			out = append(out, mailboxID)
		}
	}
	return out, nil
}

func marcarRuntimeMailboxEntregadoPorIDs(ids []int64) error {
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if err := MarcarRuntimeMailboxEntregado(id); err != nil {
			return err
		}
	}
	return nil
}

func AckBootstrapRuntimeLease(startOrderID, bootstrapOrderID int64, mailboxIDs []int64, sesionID int64, ackSource string) error {
	if startOrderID <= 0 && bootstrapOrderID <= 0 && len(mailboxIDs) == 0 {
		return nil
	}
	ackSource = strings.TrimSpace(ackSource)
	if ackSource == "" {
		ackSource = "agente_tick"
	}
	if err := marcarBootstrapRuntimeLeaseEntregado(startOrderID, bootstrapOrderID, mailboxIDs, sesionID, ackSource, time.Now().UTC()); err != nil {
		return err
	}
	if err := marcarRuntimeMailboxConsumidoPorIDs(mailboxIDs); err != nil {
		return err
	}
	if bootstrapOrderID > 0 {
		order, err := GetRuntimeOrder(bootstrapOrderID)
		if err != nil {
			return err
		}
		if order != nil && order.Estado != "fallida" && order.Estado != "cancelada" && order.Estado != "expirada" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"ok":          true,
				"bootstrap":   true,
				"acked_by":    ackSource,
				"lease_state": "acked",
				"sesion_id":   sesionID,
				"mailbox_ids": mailboxIDs,
			})
			targetState := strings.TrimSpace(order.Estado)
			targetError := order.ErrorText
			if targetState == "" || targetState == "ejecutando" || targetState == "pendiente" || targetState == "tomada" {
				targetState = "completada"
				targetError = ""
			}
			if err := MarcarRuntimeOrderEstado(order.ID, targetState, resultado, targetError); err != nil {
				return err
			}
			if err := registrarEvidenciaHandoffReanudadoPorOrden(order, sesionID); err != nil {
				return err
			}
		}
	}
	if startOrderID > 0 {
		order, err := GetRuntimeOrder(startOrderID)
		if err != nil {
			return err
		}
		if order != nil && order.Estado != "fallida" && order.Estado != "cancelada" && order.Estado != "expirada" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"ok":                 true,
				"acked_by":           ackSource,
				"lease_state":        "acked",
				"sesion_id":          sesionID,
				"bootstrap_order_id": bootstrapOrderID,
				"bootstrap_mailbox":  mailboxIDs,
			})
			targetState := strings.TrimSpace(order.Estado)
			targetError := order.ErrorText
			if targetState == "" || targetState == "ejecutando" || targetState == "pendiente" || targetState == "tomada" {
				targetState = "completada"
				targetError = ""
			}
			if err := MarcarRuntimeOrderEstado(order.ID, targetState, resultado, targetError); err != nil {
				return err
			}
		}
	}
	return nil
}

func marcarBootstrapRuntimeLeaseEntregado(startOrderID, bootstrapOrderID int64, mailboxIDs []int64, sesionID int64, ackSource string, deliveredAt time.Time) error {
	if startOrderID <= 0 && bootstrapOrderID <= 0 && len(mailboxIDs) == 0 {
		return nil
	}
	ackSource = strings.TrimSpace(ackSource)
	if ackSource == "" {
		ackSource = "agente_tick"
	}
	if deliveredAt.IsZero() {
		deliveredAt = time.Now().UTC()
	}
	if err := marcarRuntimeMailboxEntregadoPorIDs(mailboxIDs); err != nil {
		return err
	}
	return marcarBootstrapRuntimeLeaseObservada(startOrderID, bootstrapOrderID, mailboxIDs, sesionID, ackSource, deliveredAt)
}

func marcarBootstrapRuntimeLeaseObservada(startOrderID, bootstrapOrderID int64, mailboxIDs []int64, sesionID int64, ackSource string, deliveredAt time.Time) error {
	if startOrderID <= 0 && bootstrapOrderID <= 0 {
		return nil
	}
	ackSource = strings.TrimSpace(ackSource)
	if ackSource == "" {
		ackSource = "agente_tick"
	}
	if deliveredAt.IsZero() {
		deliveredAt = time.Now().UTC()
	}
	if bootstrapOrderID > 0 {
		order, err := GetRuntimeOrder(bootstrapOrderID)
		if err != nil {
			return err
		}
		if order != nil && order.Estado != "fallida" && order.Estado != "cancelada" && order.Estado != "expirada" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"bootstrap":    true,
				"delivered_by": ackSource,
				"delivered_at": deliveredAt.Format(time.RFC3339Nano),
				"lease_state":  "delivered",
				"sesion_id":    sesionID,
				"mailbox_ids":  mailboxIDs,
			})
			if err := MarcarRuntimeOrderEstado(order.ID, order.Estado, resultado, order.ErrorText); err != nil {
				return err
			}
		}
	}
	if startOrderID > 0 {
		order, err := GetRuntimeOrder(startOrderID)
		if err != nil {
			return err
		}
		if order != nil && order.Estado != "fallida" && order.Estado != "cancelada" && order.Estado != "expirada" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"delivered_by":       ackSource,
				"delivered_at":       deliveredAt.Format(time.RFC3339Nano),
				"lease_state":        "delivered",
				"sesion_id":          sesionID,
				"bootstrap_order_id": bootstrapOrderID,
				"bootstrap_mailbox":  mailboxIDs,
			})
			if err := MarcarRuntimeOrderEstado(order.ID, order.Estado, resultado, order.ErrorText); err != nil {
				return err
			}
		}
	}
	return nil
}

func newBootstrapRuntimeLeaseSelection(order *RuntimeOrder, startOrderID int64, mailboxIDs []int64, sesionID int64, score int) bootstrapRuntimeLeaseSelection {
	return bootstrapRuntimeLeaseSelection{
		order:        order,
		startOrderID: startOrderID,
		mailboxIDs:   append([]int64(nil), mailboxIDs...),
		sesionID:     sesionID,
		score:        score,
	}
}

func runtimeBootstrapLeaseResolverTarget(handle *RuntimeHandle, runtime *RuntimeInstance) (string, *int64, int64) {
	agente := ""
	var proyectoID *int64
	var sesionID int64
	if handle != nil {
		agente = strings.TrimSpace(handle.Agente)
		proyectoID = handle.ProyectoID
		if handle.SesionID != nil {
			sesionID = *handle.SesionID
		}
	}
	if runtime != nil {
		if agente == "" {
			agente = strings.TrimSpace(runtime.Agente)
		}
		if proyectoID == nil {
			proyectoID = runtime.ProyectoID
		}
		if sesionID <= 0 && runtime.SesionID != nil {
			sesionID = *runtime.SesionID
		}
	}
	return agente, proyectoID, sesionID
}

func runtimeBootstrapLeaseSelectionCandidate(order *RuntimeOrder, mailboxID, targetSesionID int64, handle *RuntimeHandle, runtime *RuntimeInstance, observed bool) (*bootstrapRuntimeLeaseSelection, error) {
	startOrderID, mailboxIDs, orderSesionID := runtimeBootstrapLeaseFromOrder(order)
	startOrder, err := runtimeBootstrapLeaseStartOrder(order, startOrderID)
	if err != nil {
		return nil, err
	}
	if orderSesionID <= 0 {
		orderSesionID = runtimeBootstrapLeaseSesionID(order, startOrder)
	}
	if !observed && startOrder != nil && runtimeBootstrapLeaseTieneReceiptUtil(mapFromJSON(startOrder.ResultadoJSON)) {
		return nil, nil
	}
	mailboxIDs, err = runtimeBootstrapLeaseMailboxIDsVigentesParaSeleccion(order, startOrder, mailboxID)
	if err != nil {
		return nil, err
	}
	if len(mailboxIDs) == 0 || !runtimeBootstrapLeaseIncluyeMailbox(mailboxIDs, mailboxID) {
		return nil, nil
	}
	if targetSesionID > 0 && orderSesionID > 0 && orderSesionID != targetSesionID {
		return nil, nil
	}
	score := runtimeBootstrapLeaseAffinityScore(order, startOrder, handle, runtime, targetSesionID)
	candidate := newBootstrapRuntimeLeaseSelection(order, startOrderID, mailboxIDs, orderSesionID, score)
	return &candidate, nil
}

func runtimeBootstrapLeaseStartOrder(order *RuntimeOrder, startOrderID int64) (*RuntimeOrder, error) {
	if startOrderID <= 0 || order == nil || startOrderID == order.ID {
		return nil, nil
	}
	return GetRuntimeOrder(startOrderID)
}

func runtimeBootstrapLeaseMailboxIDsVigentesParaSeleccion(order *RuntimeOrder, startOrder *RuntimeOrder, mailboxID int64) ([]int64, error) {
	var firstNonEmpty []int64
	for _, candidateMailboxIDs := range runtimeBootstrapLeaseMailboxIDsConFallbackFuente(order, startOrder) {
		vigentes, err := runtimeBootstrapLeaseMailboxIDsVigentes(candidateMailboxIDs)
		if err != nil {
			return nil, err
		}
		if len(vigentes) == 0 {
			continue
		}
		filtered := runtimeBootstrapLeaseMailboxIDsFiltrados(vigentes, mailboxID)
		if mailboxID > 0 && len(filtered) > 0 {
			return filtered, nil
		}
		if len(firstNonEmpty) == 0 {
			firstNonEmpty = filtered
		}
	}
	return firstNonEmpty, nil
}

func runtimeBootstrapLeaseMailboxIDsFiltrados(mailboxIDs []int64, mailboxID int64) []int64 {
	if mailboxID <= 0 {
		return append([]int64(nil), mailboxIDs...)
	}
	for _, candidateMailboxID := range mailboxIDs {
		if candidateMailboxID == mailboxID {
			return []int64{mailboxID}
		}
	}
	return nil
}

func runtimeBootstrapLeaseIncluyeMailbox(mailboxIDs []int64, mailboxID int64) bool {
	if mailboxID <= 0 {
		return true
	}
	for _, candidateMailboxID := range mailboxIDs {
		if candidateMailboxID == mailboxID {
			return true
		}
	}
	return false
}

func runtimeBootstrapLeaseSelectionMejora(candidate, selected bootstrapRuntimeLeaseSelection, targetSesionID int64) bool {
	candidateMatchesTargetSession := runtimeBootstrapLeaseSelectionMatchesTargetSession(candidate, targetSesionID)
	selectedMatchesTargetSession := runtimeBootstrapLeaseSelectionMatchesTargetSession(selected, targetSesionID)
	if candidateMatchesTargetSession != selectedMatchesTargetSession {
		return candidateMatchesTargetSession
	}
	if targetSesionID <= 0 {
		candidateHasSession := candidate.sesionID > 0
		selectedHasSession := selected.sesionID > 0
		if candidateHasSession != selectedHasSession {
			return candidateHasSession
		}
	}
	if candidate.score != selected.score {
		return candidate.score > selected.score
	}
	switch {
	case runtimeBootstrapLeaseResumeDerivadoPrefiereStart(candidate.order, selected.order):
		return false
	case runtimeBootstrapLeaseResumeDerivadoPrefiereStart(selected.order, candidate.order):
		return true
	case runtimeBootstrapLeaseFuenteLigadaPrefiereSobreStart(candidate.order, selected.order):
		return true
	case runtimeBootstrapLeaseFuenteLigadaPrefiereSobreStart(selected.order, candidate.order):
		return false
	default:
		return candidate.order.ID >= selected.order.ID
	}
}

func runtimeBootstrapLeaseSelectionMatchesTargetSession(selection bootstrapRuntimeLeaseSelection, targetSesionID int64) bool {
	return targetSesionID > 0 && selection.sesionID == targetSesionID
}

func resolverBootstrapRuntimeLeaseConFiltro(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance, observed bool) (*RuntimeOrder, int64, []int64, int64, error) {
	if handle != nil {
		meta := mapFromJSON(handle.MetadataJSON)
		driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
		transport := strings.TrimSpace(stringFromMap(meta, "transport", ""))
		if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") ||
			strings.EqualFold(driver, "tmux_cli_session") ||
			strings.EqualFold(transport, "tmux") {
			var err error
			handle, err = normalizarRuntimeHandleTMUXCanonico(handle)
			if err != nil {
				return nil, 0, nil, 0, err
			}
		}
	}
	agente, proyectoID, sesionID := runtimeBootstrapLeaseResolverTarget(handle, runtime)
	if agente == "" {
		return nil, 0, nil, 0, nil
	}

	targetSesionID := sesionID
	candidates, err := listarRuntimeOrdersBootstrapLeaseCandidatas(agente, proyectoID, observed)
	if err != nil {
		return nil, 0, nil, 0, err
	}
	var selected *bootstrapRuntimeLeaseSelection
	for _, order := range candidates {
		candidate, candidateErr := runtimeBootstrapLeaseSelectionCandidate(order, mailboxID, targetSesionID, handle, runtime, observed)
		if candidateErr != nil {
			return nil, 0, nil, 0, candidateErr
		}
		if candidate == nil {
			continue
		}
		if selected == nil {
			selected = candidate
			continue
		}
		if !runtimeBootstrapLeaseSelectionMejora(*candidate, *selected, targetSesionID) {
			continue
		}
		selected = candidate
	}
	if selected == nil {
		return nil, 0, nil, sesionID, nil
	}
	if targetSesionID <= 0 {
		targetSesionID = selected.sesionID
	}
	return selected.order, selected.startOrderID, append([]int64(nil), selected.mailboxIDs...), targetSesionID, nil
}

func listarRuntimeOrdersBootstrapLeaseCandidatas(agente string, proyectoID *int64, observed bool) ([]*RuntimeOrder, error) {
	candidates := make([]*RuntimeOrder, 0, 8)
	for _, estado := range []string{"ejecutando", "tomada", "pendiente", "completada"} {
		estado := estado
		orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return nil, err
		}
		for _, order := range orders {
			if runtimeOrderCalificaComoBootstrapLeaseCandidata(order, observed) {
				candidates = append(candidates, order)
			}
		}
	}
	return candidates, nil
}

func runtimeOrderCalificaComoBootstrapLeaseCandidata(order *RuntimeOrder, observed bool) bool {
	if order == nil {
		return false
	}
	switch strings.TrimSpace(order.Tipo) {
	case "handoff", "resume", "start":
	default:
		return false
	}
	res := mapFromJSON(order.ResultadoJSON)
	switch strings.TrimSpace(order.Estado) {
	case "pendiente":
		if observed && runtimeBootstrapLeaseTieneReceiptUtil(res) && runtimeBootstrapLeaseTieneCoberturaDeclarada(res) {
			return true
		}
		if !runtimeOrderMantieneBootstrapLeasePendiente(res) &&
			!boolFromAny(res["deferred"]) &&
			stringFromMap(res, "estado_dispatch", "") == "" {
			return false
		}
	case "completada":
		leaseState := strings.ToLower(strings.TrimSpace(stringFromMap(res, "lease_state", "")))
		if leaseState != "waiting_for_evidence" && leaseState != "delivered" {
			return false
		}
		if !runtimeBootstrapLeaseTieneCoberturaDeclarada(res) {
			return false
		}
	}
	return runtimeBootstrapLeaseTieneReceiptUtil(res) == observed
}
