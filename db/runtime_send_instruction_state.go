package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func retenerRuntimeOrderSendInstructionDiferidaAMailbox(order *RuntimeOrder, payload map[string]any, reason string) error {
	if order == nil {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "mailbox retenido hasta runtime entregable"
	}
	if payload == nil {
		payload = map[string]any{}
	}
	mailboxID, err := asegurarRuntimeOrderSendInstructionMailboxPendiente(order, payload)
	if err != nil {
		return err
	}
	payload["mailbox_id"] = mailboxID
	if strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) == "" {
		payload["mailbox_kind"] = "instruction"
	}
	if strings.TrimSpace(stringFromMap(payload, "to_agente", "")) == "" && strings.TrimSpace(order.Agente) != "" {
		payload["to_agente"] = strings.TrimSpace(order.Agente)
	}
	updatedPayloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	nextAttempt := time.Now().UTC().Add(runtimeOrderSendInstructionRetryDelay())
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":  "pending",
		"ok":              false,
		"mailbox_id":      mailboxID,
		"deferred":        true,
		"deferred_reason": reason,
		"mailbox_only":    true,
		"delivery_state":  "queued",
		"retry_after":     nextAttempt.Format(time.RFC3339Nano),
	})
	resultado = mergeRuntimeOrderResultJSON(resultado, runtimeOrderSendInstructionTrackingResult(payload))
	_, err = DB.Exec(`
		UPDATE runtime_orders
		SET payload_json = ?,
		    estado = 'pendiente',
		    resultado_json = ?,
		    error_text = CASE
		        WHEN TRIM(COALESCE(error_text, '')) = '' THEN ?
		        ELSE error_text || ?
		    END,
		    started_at = NULL,
		    finished_at = NULL,
		    claimed_by = '',
		    lease_token = '',
		    lease_expires_at = NULL,
		    available_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		string(updatedPayloadJSON),
		resultado,
		reason,
		"\n"+reason,
		nextAttempt,
		order.ID,
	)
	if err != nil {
		return err
	}
	if err := runtimeOrdersHotIndexSyncByID(order.ID); err != nil {
		return err
	}
	if err := registrarDispatchLedgerRuntimeOrder(order, payload, "pending", "queued", "", reason); err != nil {
		return err
	}
	return registrarWorkQueueRuntimeOrder(order, payload, "queued", reason)
}

func retenerRuntimeOrderSendInstructionNotificada(order *RuntimeOrder, payload map[string]any, reason string, notifiedAt time.Time) error {
	if order == nil {
		return nil
	}
	actual, err := GetRuntimeOrder(order.ID)
	if err == nil && runtimeOrderYaTieneEntregaValida(actual) {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "mailbox notificado pendiente de recibo"
	}
	if notifiedAt.IsZero() {
		notifiedAt = time.Now().UTC()
	}
	nextAttempt := time.Now().UTC().Add(runtimeOrderSendInstructionRetryDelay())
	mailboxOnly := runtimeOrderSendInstructionMailboxOnly(reason, payload)
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":       "notified",
		"ok":                   false,
		"mailbox_id":           runtimeOrderSendInstructionMailboxID(payload),
		"deferred":             true,
		"deferred_reason":      reason,
		"mailbox_only":         mailboxOnly,
		"delivery_state":       "notified",
		"delivery_notified_at": notifiedAt.Format(time.RFC3339Nano),
		"delivery_receipt_at":  "",
		"receipt_source":       "",
		"retry_after":          nextAttempt.Format(time.RFC3339Nano),
	})
	resultado = mergeRuntimeOrderResultJSON(resultado, runtimeOrderSendInstructionTrackingResult(payload))
	_, err = DB.Exec(`
		UPDATE runtime_orders
		SET estado = 'pendiente',
		    resultado_json = ?,
		    error_text = CASE
		        WHEN TRIM(COALESCE(error_text, '')) = '' THEN ?
		        ELSE error_text || ?
		    END,
		    started_at = NULL,
		    finished_at = NULL,
		    claimed_by = '',
		    lease_token = '',
		    lease_expires_at = NULL,
		    available_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		resultado,
		reason,
		"\n"+reason,
		nextAttempt,
		order.ID,
	)
	if err != nil {
		return err
	}
	if err := runtimeOrdersHotIndexSyncByID(order.ID); err != nil {
		return err
	}
	if err := registrarDispatchLedgerRuntimeOrder(order, payload, "notified", "notified", "", reason); err != nil {
		return err
	}
	return registrarWorkQueueRuntimeOrder(order, payload, "notified", reason)
}

func runtimeOrderYaTieneEntregaValida(order *RuntimeOrder) bool {
	if order == nil {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(order.Estado)) {
	case "completada", "fallida", "cancelada", "expirada":
		return true
	}
	result := mapFromJSON(order.ResultadoJSON)
	if result == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(stringFromMap(result, "delivery_state", "")), "delivered") {
		return true
	}
	if strings.TrimSpace(stringFromMap(result, "receipt_source", "")) != "" {
		return true
	}
	return false
}

func completarRuntimeOrderSendInstructionEntregadaPorReceipt(order *RuntimeOrder, payload map[string]any, receiptSource string, receiptAt time.Time) error {
	if order == nil {
		return nil
	}
	if receiptAt.IsZero() {
		receiptAt = time.Now().UTC()
	}
	receiptSource = strings.TrimSpace(receiptSource)
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":      "delivered",
		"ok":                  true,
		"mailbox_id":          runtimeOrderSendInstructionMailboxID(payload),
		"mailbox_only":        true,
		"delivery_state":      "delivered",
		"delivery_receipt_at": receiptAt.Format(time.RFC3339Nano),
		"receipt_source":      receiptSource,
	})
	resultado = mergeRuntimeOrderResultJSON(resultado, runtimeOrderSendInstructionTrackingResult(payload))
	if err := MarcarRuntimeOrderEstado(order.ID, "completada", resultado, ""); err != nil {
		return err
	}
	if err := registrarDispatchLedgerRuntimeOrder(order, payload, "delivered", "delivered", receiptSource, "receipt confirmado"); err != nil {
		return err
	}
	workReason := strings.TrimSpace(receiptSource)
	if workReason == "" {
		workReason = "receipt_confirmado"
	}
	if err := registrarWorkQueueRuntimeOrder(order, payload, "delivered", workReason); err != nil {
		return err
	}
	runtimeOrderSendInstructionAutoStopLocalOllama(order, payload)
	return nil
}

func completarRuntimeOrderSendInstructionSupersedida(order *RuntimeOrder, payload map[string]any, reason string) error {
	if order == nil {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "mailbox_missing_after_notify"
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":                true,
		"obsoleta":          true,
		"superseded":        true,
		"superseded_reason": reason,
		"mailbox_id":        runtimeOrderSendInstructionMailboxID(payload),
		"delivery_state":    "superseded",
		"dispatch_state":    "superseded",
	})
	resultado = mergeRuntimeOrderResultJSON(resultado, runtimeOrderSendInstructionTrackingResult(payload))
	if err := MarcarRuntimeOrderEstado(order.ID, "completada", resultado, ""); err != nil {
		return err
	}
	if err := registrarDispatchLedgerRuntimeOrder(order, payload, "superseded", "superseded", "", reason); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err := registrarWorkQueueRuntimeOrder(order, payload, "superseded", reason); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

func fallarRuntimeOrderSendInstructionPorBloqueoAgente(order *RuntimeOrder, payload map[string]any, detalle string, blockedAt time.Time) error {
	if order == nil {
		return nil
	}
	detalle = strings.TrimSpace(detalle)
	if detalle == "" {
		detalle = "BLOQUEO: el agente no pudo ejecutar la microtarea"
	}
	if blockedAt.IsZero() {
		blockedAt = time.Now().UTC()
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":      "failed",
		"ok":                  false,
		"mailbox_id":          runtimeOrderSendInstructionMailboxID(payload),
		"delivery_state":      "blocked",
		"blocked":             true,
		"blocked_reason":      detalle,
		"delivery_receipt_at": blockedAt.Format(time.RFC3339Nano),
		"receipt_source":      "transcript_blocked",
	})
	resultado = mergeRuntimeOrderResultJSON(resultado, runtimeOrderSendInstructionTrackingResult(payload))
	if err := MarcarRuntimeOrderEstado(order.ID, "fallida", resultado, detalle); err != nil {
		return err
	}
	runtimeOrderSendInstructionAutoStopLocalOllama(order, payload)
	return nil
}

func completarRuntimeOrderSendInstructionDiferidaAMailbox(order *RuntimeOrder, payload map[string]any, reason string) error {
	if order == nil {
		return nil
	}
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) || runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, reason)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "mailbox retenido hasta runtime entregable"
	}
	mailboxID := runtimeOrderSendInstructionMailboxID(payload)
	if mailboxID <= 0 {
		existente, err := GetRuntimeMailboxByRuntimeOrderID(order.ID)
		if err != nil {
			return err
		}
		if existente != nil {
			mailboxID = existente.ID
		} else {
			toAgente := stringFromMap(payload, "to_agente", order.Agente)
			fromAgente := stringFromMap(payload, "from_agente", "server")
			msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
				FromAgente:     fromAgente,
				ToAgente:       toAgente,
				ProyectoID:     order.ProyectoID,
				RuntimeOrderID: &order.ID,
				Kind:           "instruction",
				PayloadJSON:    order.PayloadJSON,
			})
			if err != nil {
				return err
			}
			mailboxID = msgID
		}
	}
	mailboxOnly := runtimeOrderSendInstructionMailboxOnly(reason, payload)
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":  "pending",
		"ok":              true,
		"mailbox_id":      mailboxID,
		"deferred":        true,
		"deferred_reason": reason,
		"mailbox_only":    mailboxOnly,
		"delivery_state":  "queued",
	})
	if err := MarcarRuntimeOrderEstado(order.ID, "completada", resultado, ""); err != nil {
		return err
	}
	if err := registrarDispatchLedgerRuntimeOrder(order, payload, "pending", "queued", "", reason); err != nil {
		return err
	}
	return registrarWorkQueueRuntimeOrder(order, payload, "queued", reason)
}

func runtimeOrderSendInstructionMailboxOnly(reason string, payload map[string]any) bool {
	if runtimeOrderSendInstructionProvieneMailbox(payload) {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(reason)), "mailbox_only")
}
