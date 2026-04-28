package db

import (
	"encoding/json"
	"strings"
	"time"

	"orquesta/internal/controlruntime"
)

func registrarDispatchLedgerRuntimeOrder(order *RuntimeOrder, payload map[string]any, dispatchState, deliveryState, receiptSource, reason string) error {
	if order == nil {
		return nil
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil || handle == nil {
		return err
	}
	return controlruntime.RecordDispatchLedgerFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON), controlruntime.DispatchLedgerRecordInput{
		RuntimeOrderID:           order.ID,
		MailboxID:                runtimeOrderSendInstructionMailboxID(payload),
		HandleID:                 handle.ID,
		DeliveryAttemptSignature: runtimeOrderSendInstructionDeliveryAttemptSignature(payload),
		ExternalSessionID:        runtimeOrderSendInstructionExternalSessionID(payload),
		DispatchState:            strings.TrimSpace(dispatchState),
		DeliveryState:            strings.TrimSpace(deliveryState),
		ReceiptSource:            strings.TrimSpace(receiptSource),
		Reason:                   strings.TrimSpace(reason),
		RecordedAt:               time.Now().UTC(),
	})
}

func registrarWorkQueueRuntimeOrder(order *RuntimeOrder, payload map[string]any, state, reason string) error {
	if order == nil || payload == nil {
		return nil
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil || handle == nil {
		return err
	}
	return controlruntime.RecordWorkQueueFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON), controlruntime.WorkQueueRecordInput{
		MailboxID:       runtimeOrderSendInstructionMailboxID(payload),
		RuntimeOrderID:  order.ID,
		Kind:            strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")),
		Action:          strings.TrimSpace(stringFromMap(payload, "accion", "")),
		TaskID:          int64FromAny(payload["tarea_id"]),
		VerificationKey: strings.TrimSpace(stringFromMap(payload, "verification_key", "")),
		State:           strings.TrimSpace(state),
		Title:           strings.TrimSpace(runtimeOrderSendInstructionTexto(payload)),
		Reason:          strings.TrimSpace(reason),
		RecordedAt:      time.Now().UTC(),
	})
}

func actualizarRuntimeOrderPayloadJSON(orderID int64, payload map[string]any) error {
	if orderID <= 0 || payload == nil {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET payload_json = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		string(raw),
		orderID,
	); err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(orderID)
}

func objetivoProcesoSendInstruction(order *RuntimeOrder, handle *RuntimeHandle, runtime *RuntimeInstance, workingDir string) controlruntime.ObjetivoProceso {
	obj := controlruntime.ObjetivoProceso{}
	if handle != nil {
		obj.HandleKind = handle.HandleKind
		obj.HandleRef = handle.HandleRef
		obj.MetadataJSON = handle.MetadataJSON
	}
	if strings.TrimSpace(workingDir) != "" || (order != nil && order.ID > 0) {
		meta := mapFromJSON(obj.MetadataJSON)
		if meta == nil {
			meta = map[string]any{}
		}
		meta["working_dir"] = strings.TrimSpace(workingDir)
		if order != nil && order.ID > 0 {
			meta["runtime_order_id"] = order.ID
		}
		if metaJSON, marshalErr := json.Marshal(meta); marshalErr == nil {
			obj.MetadataJSON = string(metaJSON)
		}
	}
	if runtime != nil &&
		!runtimeOrderBloqueaFallbackPID(order, runtime, handle) &&
		(handle == nil || strings.TrimSpace(handle.HandleKind) == "" || strings.TrimSpace(handle.HandleKind) == "process") {
		obj.PID = runtime.PID
	}
	return obj
}

func runtimeOrderSendInstructionExternalSessionID(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	return strings.TrimSpace(stringFromMap(payload, "external_session_id", ""))
}

func runtimeOrderSendInstructionRetryDelay() time.Duration {
	seconds := configIntOrDefault("runtime_send_instruction_retry_seconds", 15)
	if seconds <= 0 {
		seconds = 15
	}
	return time.Duration(seconds) * time.Second
}

func runtimeOrderSendInstructionDeliveryAttemptSignature(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	return strings.TrimSpace(stringFromMap(payload, "delivery_attempt_signature", ""))
}

func runtimeOrderSendInstructionDispatchLedgerEvidence(handle *RuntimeHandle, order *RuntimeOrder, payload map[string]any) (bool, string, time.Time, error) {
	if handle == nil || order == nil || payload == nil {
		return false, "", time.Time{}, nil
	}
	entry, err := controlruntime.FindDispatchLedgerEntryFromMetadataJSON(
		strings.TrimSpace(handle.MetadataJSON),
		runtimeOrderSendInstructionMailboxID(payload),
		runtimeOrderSendInstructionDeliveryAttemptSignature(payload),
		runtimeOrderSendInstructionExternalSessionID(payload),
	)
	if err != nil || entry == nil {
		return false, "", time.Time{}, err
	}
	switch strings.ToLower(strings.TrimSpace(entry.DeliveryState)) {
	case "delivered", "consumed":
	default:
		return false, "", time.Time{}, nil
	}
	receiptAt := time.Now().UTC()
	if parsed, ok := parseDispatchLedgerRecordedAt(entry.RecordedAt); ok {
		receiptAt = parsed
	}
	source := strings.TrimSpace(entry.ReceiptSource)
	if source == "" {
		source = "dispatch_ledger"
	}
	return true, source, receiptAt, nil
}

func parseDispatchLedgerRecordedAt(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func runtimeMailboxReusableForPendingSendInstruction(msg *RuntimeMailboxMessage) bool {
	if msg == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(msg.Estado)) {
	case "", "pendiente", "entregado":
		return true
	default:
		return false
	}
}

func runtimeOrderSendInstructionMailboxPayloadJSON(order *RuntimeOrder, payload map[string]any) string {
	if payload == nil {
		return strings.TrimSpace(order.PayloadJSON)
	}
	sanitized := make(map[string]any, len(payload))
	for key, value := range payload {
		switch strings.TrimSpace(key) {
		case "mailbox_id":
			continue
		default:
			sanitized[key] = value
		}
	}
	if len(sanitized) == 0 {
		return strings.TrimSpace(order.PayloadJSON)
	}
	raw, err := json.Marshal(sanitized)
	if err != nil {
		return strings.TrimSpace(order.PayloadJSON)
	}
	return string(raw)
}

func asegurarRuntimeOrderSendInstructionMailboxPendiente(order *RuntimeOrder, payload map[string]any) (int64, error) {
	if order == nil {
		return 0, nil
	}
	if payload == nil {
		payload = map[string]any{}
	}
	if mailboxID := runtimeOrderSendInstructionMailboxID(payload); mailboxID > 0 {
		msg, err := GetRuntimeMailbox(mailboxID)
		if err != nil {
			return 0, err
		}
		if runtimeMailboxReusableForPendingSendInstruction(msg) {
			return mailboxID, nil
		}
		delete(payload, "mailbox_id")
	}
	existente, err := GetRuntimeMailboxByRuntimeOrderID(order.ID)
	if err != nil {
		return 0, err
	}
	if runtimeMailboxReusableForPendingSendInstruction(existente) {
		return existente.ID, nil
	}
	toAgente := stringFromMap(payload, "to_agente", order.Agente)
	fromAgente := stringFromMap(payload, "from_agente", "server")
	kind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", ""))
	if kind == "" {
		kind = "instruction"
	}
	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     fromAgente,
		ToAgente:       toAgente,
		ProyectoID:     order.ProyectoID,
		RuntimeOrderID: &order.ID,
		Kind:           kind,
		PayloadJSON:    runtimeOrderSendInstructionMailboxPayloadJSON(order, payload),
	})
	if err != nil {
		return 0, err
	}
	return msgID, nil
}

func runtimeOrderSendInstructionDebeDegradarseAMailboxPorError(order *RuntimeOrder, payload map[string]any, runtimeErr error) bool {
	if order == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) || runtimeErr == nil {
		return false
	}
	raw := strings.ToLower(strings.TrimSpace(runtimeErr.Error()))
	if !strings.Contains(raw, "session_resume timeout") {
		return false
	}
	switch strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) {
	case "autonomia", "nudge", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func runtimeOrderSendInstructionDebeRetenerNotificadaPorSessionResumeTimeout(order *RuntimeOrder, handle *RuntimeHandle, payload map[string]any, runtimeErr error) bool {
	if order == nil || handle == nil || runtimeErr == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false
	}
	raw := strings.ToLower(strings.TrimSpace(runtimeErr.Error()))
	if !strings.Contains(raw, "session_resume timeout") {
		return false
	}
	if !runtimeHandleUsaCodexTTYInestable(mapFromJSON(handle.MetadataJSON)) {
		return false
	}
	if runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return true
	}
	return runtimeHandleEsTMUXCanonico(handle) &&
		strings.EqualFold(strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")), "nudge")
}

func reencolarRuntimeOrderSendInstruction(order *RuntimeOrder, payload map[string]any, reason string) error {
	return reencolarRuntimeOrderSendInstructionAt(order, payload, reason, time.Now().UTC().Add(runtimeOrderSendInstructionRetryDelay()))
}

func reencolarRuntimeOrderSendInstructionAt(order *RuntimeOrder, payload map[string]any, reason string, nextAttempt time.Time) error {
	if order == nil || order.ID <= 0 {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "runtime no entregable todavía"
	}
	if nextAttempt.IsZero() {
		nextAttempt = time.Now().UTC().Add(runtimeOrderSendInstructionRetryDelay())
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":       "pending",
		"ok":                   false,
		"deferred":             true,
		"mailbox_id":           runtimeOrderSendInstructionMailboxID(payload),
		"delivery_state":       "queued",
		"delivery_notified_at": "",
		"delivery_receipt_at":  "",
		"receipt_source":       "",
		"retry_after":          nextAttempt.Format(time.RFC3339Nano),
		"deferred_reason":      reason,
	})
	resultado = mergeRuntimeOrderResultJSON(resultado, runtimeOrderSendInstructionTrackingResult(payload))
	_, err := DB.Exec(`
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
		WHERE id = ?
		  AND estado NOT IN ('completada','fallida','cancelada','expirada')`,
		resultado,
		reason,
		"\n"+reason,
		nextAttempt,
		order.ID,
	)
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(order.ID)
}
