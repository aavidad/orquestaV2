package db

import (
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"strings"
	"time"
)

func runtimeOrderSendInstructionEsGuidanceDurableServidor(payload map[string]any) bool {
	if !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false
	}
	switch strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) {
	case "nudge", "autonomia", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func runtimeHandlePermiteSessionResumeDirecto(handle *RuntimeHandle, sessionResumeRequested bool) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	for _, key := range []string{"supervisor_ref", "stdin_path", "stdin_raw_path"} {
		if strings.TrimSpace(stringFromMap(meta, key, "")) != "" {
			return true
		}
	}
	return sessionResumeRequested && !runtimeHandleUsaCodexTTYInestable(meta)
}

func runtimeHandleSolicitaSessionResume(handle *RuntimeHandle, deliveryMode string) bool {
	if handle == nil {
		return false
	}
	if runtimeagente.NormalizeMailboxDeliveryMode(deliveryMode) == runtimeagente.MailboxDeliverySessionResume {
		return true
	}
	meta := mapFromJSON(handle.MetadataJSON)
	caps := mapFromJSON(handle.CapabilitiesJSON)
	explicitBootstrapOnly := false
	for _, raw := range []string{
		stringFromMap(caps, "mailbox_delivery_mode", ""),
		stringFromMap(meta, "mailbox_delivery_mode", ""),
	} {
		switch runtimeagente.NormalizeMailboxDeliveryMode(raw) {
		case runtimeagente.MailboxDeliverySessionResume:
			return true
		case runtimeagente.MailboxDeliveryBootstrapOnly:
			explicitBootstrapOnly = true
		}
	}
	if explicitBootstrapOnly {
		return false
	}
	return false
}

func runtimeOrderSendInstructionDebeRetenerOrdenDirecta(handle *RuntimeHandle, runtime *RuntimeInstance, deliveryMode, externalSessionID string) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	requiresDurableRuntime := runtimeHandleUsaCodexTTYInestable(meta) ||
		strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") ||
		strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session")
	switch runtimeagente.NormalizeMailboxDeliveryMode(deliveryMode) {
	case runtimeagente.MailboxDeliveryBootstrapOnly:
		return requiresDurableRuntime
	case runtimeagente.MailboxDeliverySessionResume:
		return strings.TrimSpace(externalSessionID) != "" || runtimeHandleTieneExternalSessionID(handle, runtime)
	case runtimeagente.MailboxDeliveryCoordinatedRestart:
		return requiresDurableRuntime
	default:
		return false
	}
}

func intentarEntregarRuntimeOrderSendInstructionBootstrapTMUX(order *RuntimeOrder, payload map[string]any, handle *RuntimeHandle, runtime *RuntimeInstance, obj controlruntime.ObjetivoProceso) (bool, error) {
	if order == nil || handle == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, nil
	}
	if !runtimeHandleListaParaDispatchBootstrapTMUX(handle, time.Now().UTC()) {
		return false, nil
	}
	prompt, ok := runtimeOrderSendInstructionTMUXBootstrapPrompt(payload)
	if !ok {
		return false, nil
	}
	result, err := controlruntime.EnviarInstruccionTMUXVerificada(obj, prompt)
	if err != nil {
		return true, reencolarRuntimeOrderSendInstruction(order, payload, "tmux bootstrap dispatch error: "+strings.TrimSpace(err.Error()))
	}
	if !result.Attempted {
		return false, nil
	}
	if result.Delivered {
		if err := RegistrarRuntimeTranscriptInput(handle, runtime, prompt); err != nil {
			return true, err
		}
		if mailboxID := runtimeOrderSendInstructionMailboxID(payload); mailboxID > 0 {
			if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
				return true, err
			}
		}
		reason := "tmux bootstrap dispatch delivered"
		if result.Reason != "" {
			reason += ":" + strings.TrimSpace(result.Reason)
		}
		return true, retenerRuntimeOrderSendInstructionNotificada(order, payload, reason, time.Now().UTC())
	}
	reason := "tmux bootstrap dispatch unconfirmed"
	if result.Reason != "" {
		reason += ":" + strings.TrimSpace(result.Reason)
	}
	return true, reencolarRuntimeOrderSendInstruction(order, payload, reason)
}

func runtimeHandleListaParaDispatchBootstrapTMUX(handle *RuntimeHandle, now time.Time) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") {
		return false
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return false
	}
	ready, _ := snap.ReadyForTextDispatch(now, time.Minute)
	return ready
}

func runtimeHandleListaParaDispatchInteractivo(handle *RuntimeHandle, now time.Time) (bool, string) {
	if handle == nil {
		return false, "worker_missing"
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	transport := strings.TrimSpace(handle.Transporte)
	isTMUX := strings.EqualFold(driver, "tmux_cli_session") || strings.EqualFold(transport, "tmux")
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		if isTMUX {
			return false, "worker_snapshot_missing"
		}
		return true, ""
	}
	view := snap.View(now, time.Minute)
	if view == nil {
		if isTMUX {
			return false, "worker_view_missing"
		}
		return true, ""
	}
	if !strings.EqualFold(strings.TrimSpace(view.Driver), "tmux_cli_session") &&
		!strings.EqualFold(strings.TrimSpace(view.Transport), "tmux") {
		return true, ""
	}
	return snap.ReadyForTextDispatch(now, time.Minute)
}

func runtimeOrderSendInstructionDiferirPorErrorTMUX(handle *RuntimeHandle, runtimeErr error) (string, bool) {
	if handle == nil || runtimeErr == nil {
		return "", false
	}
	ready, reason := runtimeHandleListaParaDispatchInteractivo(handle, time.Now().UTC())
	if !ready && strings.TrimSpace(reason) != "" {
		return reason, true
	}
	raw := strings.ToLower(strings.TrimSpace(runtimeErr.Error()))
	switch {
	case strings.Contains(raw, "tmux pane no listo para send-keys"):
		return "worker_not_ready", true
	case strings.Contains(raw, "can't find pane"),
		strings.Contains(raw, "no such pane"),
		strings.Contains(raw, "tmux capture-pane"),
		strings.Contains(raw, "worker_status"):
		return "worker_not_ready", true
	case strings.Contains(raw, "tmux pane requiere carpeta de confianza"),
		strings.Contains(raw, "tmux pane requiere trust"),
		strings.Contains(raw, "untrusted folder"):
		return "worker_blocked_trust", true
	case strings.Contains(raw, "tmux pane requiere autenticacion manual"):
		return "worker_blocked_auth", true
	case strings.Contains(raw, "tmux pane bloqueado por cuota"):
		return "worker_blocked_quota", true
	default:
		return "", false
	}
}

func runtimeOrderSendInstructionTMUXBootstrapPrompt(payload map[string]any) (string, bool) {
	if !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return "", false
	}
	texto := strings.TrimSpace(stringFromMap(payload, "texto", ""))
	if texto == "" {
		return "", false
	}
	texto = strings.Join(strings.Fields(strings.ReplaceAll(texto, "\r", " ")), " ")
	if texto == "" || len(texto) > 160 {
		return "", false
	}
	return texto, true
}

func runtimeOrderSendInstructionUsaTMUXPaneActivityEvidence(handle *RuntimeHandle, payload map[string]any) bool {
	if handle == nil {
		return false
	}
	if runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return runtimeHandleEsTMUXCanonico(handle)
	}
	if !runtimeHandleEsTMUXCanonico(handle) {
		return false
	}
	if runtimeagente.NormalizeMailboxDeliveryMode(RuntimeHandleMailboxDeliveryMode(handle)) != runtimeagente.MailboxDeliverySessionResume {
		return false
	}
	mailboxKind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", ""))
	if strings.EqualFold(mailboxKind, "nudge") {
		return true
	}
	return strings.EqualFold(mailboxKind, "autonomia") &&
		strings.EqualFold(strings.TrimSpace(stringFromMap(payload, "accion", "")), "continuar_trabajo")
}

func completarRuntimeMailboxDesdePayload(payload map[string]any, proyectoID *int64) error {
	if payload == nil {
		return nil
	}
	mailboxID := int64FromAny(payload["mailbox_id"])
	if mailboxID <= 0 {
		return nil
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		return err
	}
	if err := MarcarRuntimeMailboxConsumido(mailboxID); err != nil {
		return err
	}
	kind := strings.TrimSpace(stringFromAny(payload["mailbox_kind"]))
	toAgente := strings.TrimSpace(stringFromAny(payload["to_agente"]))
	if runtimeMailboxKindSupersedible(kind) && toAgente != "" {
		_, err := ConsumirRuntimeMailboxPendienteSupersedido(toAgente, proyectoID, kind, mailboxID)
		return err
	}
	return nil
}
