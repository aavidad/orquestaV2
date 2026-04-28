package db

import (
	"orquesta/runtimeagente"
	"strings"
)

func reconciliarRuntimeSendInstructionSessionResume(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) error {
	if runtime != nil {
		logicalState := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
		shouldReactivate := logicalState == "" ||
			logicalState == "pausado" ||
			logicalState == "disponible" ||
			logicalState == "degradado" ||
			logicalState == "esperando_io"
		if shouldReactivate {
			if _, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='activo',
		    process_state=CASE
		        WHEN trim(COALESCE(process_state,''))='' OR lower(trim(process_state))='stopped' THEN 'corriendo'
		        ELSE process_state
		    END,
		    updated_at=CURRENT_TIMESTAMP
		WHERE id=?`, runtime.ID); err != nil {
				return err
			}
		}
	}
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='activo',
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE id=?`, handle.ID); err != nil {
			return err
		}
		runtimeHandleHotReset()
	}
	if order != nil {
		if err := actualizarEstadoSesionParaOrden(order, "activa"); err != nil {
			return err
		}
	}
	return nil
}

func runtimeOrderSendInstructionDebeRetenerSinRuntimeActivo(order *RuntimeOrder, payload map[string]any) bool {
	if order == nil {
		return false
	}
	return runtimeagente.EsConectorFamiliaOllama(runtimeagente.ConectorPorDefectoAgente(order.Agente), "")
}

func runtimeOrderSendInstructionMailboxOnlyReason(deliveryMode string, payload map[string]any) string {
	reason := "runtime_handle_bootstrap_only_mailbox_only"
	switch runtimeagente.NormalizeMailboxDeliveryMode(deliveryMode) {
	case runtimeagente.MailboxDeliveryCoordinatedRestart:
		reason = "runtime_handle_coordinated_restart_mailbox_only"
	case runtimeagente.MailboxDeliverySessionResume:
		reason = "runtime_handle_session_resume_mailbox_only"
	}
	if kind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")); kind != "" {
		reason += ":" + kind
	}
	return reason
}

func runtimeOrderSendInstructionLongTextReason(handle *RuntimeHandle, payload map[string]any, texto string) string {
	if handle == nil {
		return ""
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if !runtimeHandleUsaCodexTTYInestable(meta) {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") ||
		strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") {
		return ""
	}
	texto = strings.TrimSpace(texto)
	if len(texto) <= 32 {
		return ""
	}
	reason := "codex_long_text_mailbox_only"
	if kind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")); kind != "" {
		reason += ":" + kind
	}
	return reason
}

func runtimeOrderSendInstructionDurableOnlyReason(handle *RuntimeHandle, payload map[string]any, texto, deliveryMode string) (string, bool) {
	if handle == nil {
		return "", false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if runtimeHandleUsaCodexTTYInestable(meta) && runtimeOrderSendInstructionEsGuidanceDurableServidor(payload) {
		if !runtimeHandleEsTMUXCanonico(handle) {
			return runtimeOrderSendInstructionMailboxOnlyReason(deliveryMode, payload), true
		}
	}
	if reason := runtimeOrderSendInstructionLongTextReason(handle, payload, texto); reason != "" {
		return reason, true
	}
	if !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return "", false
	}
	switch strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) {
	case "nudge":
		if runtimeHandleEsTMUXCanonico(handle) {
			return "", false
		}
		return runtimeOrderSendInstructionMailboxOnlyReason(deliveryMode, payload), true
	}
	return "", false
}

func runtimeOrderSendInstructionDebeIntentarSessionResume(handle *RuntimeHandle, runtime *RuntimeInstance, payload map[string]any, texto, externalSessionID, deliveryMode string, reboundToCanonical bool) bool {
	if handle == nil || RuntimeHandlePermiteSendInputInteractivo(handle) {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if runtimeHandleUsaCodexTTYInestable(meta) && runtimeOrderSendInstructionEsGuidanceDurableServidor(payload) {
		if !runtimeHandleEsTMUXCanonico(handle) {
			return false
		}
	}
	if strings.TrimSpace(externalSessionID) == "" {
		detected, err := runtimeHandleEffectiveExternalSessionID(handle, runtime)
		if err == nil {
			externalSessionID = strings.TrimSpace(detected)
		}
	}
	externalSessionKnown := strings.TrimSpace(externalSessionID) != "" || runtimeHandleTieneExternalSessionID(handle, runtime)
	sessionResumeRequested := runtimeHandleSolicitaSessionResume(handle, deliveryMode)
	if !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return externalSessionKnown && (reboundToCanonical || runtimeHandlePermiteSessionResumeDirecto(handle, sessionResumeRequested))
	}
	if reason := runtimeOrderSendInstructionLongTextReason(handle, payload, texto); reason != "" {
		return false
	}
	switch strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) {
	case "", "instruction":
		return externalSessionKnown || sessionResumeRequested
	case "nudge":
		return runtimeHandleEsTMUXCanonico(handle) && externalSessionKnown
	case "autonomia", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return false
	}
	return sessionResumeRequested && externalSessionKnown
}
