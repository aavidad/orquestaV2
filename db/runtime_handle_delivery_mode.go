package db

import (
	"strings"

	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
)

func runtimeHandlePreservesExternalSession(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if boolFromMap(meta, "preserve_external_session_on_stop") || boolFromMap(meta, "requires_human_reauth") {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "auth_mode", "")), "oauth") {
		return true
	}
	caps := mapFromJSON(handle.CapabilitiesJSON)
	if _, ok := caps["can_stop_without_reauth"]; ok && !boolFromMap(caps, "can_stop_without_reauth") {
		return true
	}
	return false
}

func RuntimeHandlePauseRequiresFreshStart(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") {
		return false
	}
	if runtimeHandlePreservesExternalSession(handle) {
		return false
	}
	if RuntimeHandleMailboxDeliveryMode(handle) == runtimeagente.MailboxDeliveryBootstrapOnly {
		return true
	}
	for _, raw := range []string{
		stringFromMap(meta, "rendered_command", ""),
		stringFromMap(meta, "wrapped_command", ""),
		stringFromMap(meta, "command", ""),
	} {
		lower := strings.ToLower(strings.TrimSpace(raw))
		if strings.HasPrefix(lower, "ollama run ") || strings.HasPrefix(lower, "ollama serve ") {
			return true
		}
	}
	return false
}

func runtimeHandleMailboxDeliveryModeExplicit(handle *RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	meta := mapFromJSON(handle.MetadataJSON)
	caps := mapFromJSON(handle.CapabilitiesJSON)
	for _, raw := range []string{
		stringFromMap(caps, "mailbox_delivery_mode", ""),
		stringFromMap(meta, "mailbox_delivery_mode", ""),
	} {
		if mode := runtimeagente.NormalizeMailboxDeliveryMode(raw); mode != "" {
			return mode
		}
	}
	return ""
}

func RuntimeHandleMailboxDeliveryMode(handle *RuntimeHandle) string {
	if handle == nil {
		return runtimeagente.MailboxDeliveryInteractive
	}
	meta := mapFromJSON(handle.MetadataJSON)
	caps := mapFromJSON(handle.CapabilitiesJSON)
	legacyTMUXPreferredCLI := runtimepolicy.RuntimeHandleUsaLegacyCLITMUXPreferred(meta)
	externalSessionReady := runtimeHandleTieneExternalSessionID(handle, nil)
	localCLIBrokerNoInteractive := runtimepolicy.RuntimeHandleUsaTMUXPreferredCLI(meta) && !RuntimeHandlePermiteSendInputInteractivo(handle)
	tmuxSessionResumeReady := strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") &&
		strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) &&
		externalSessionReady
	processSessionResumeReady := runtimeHandleUsaCodexTTYInestable(meta) &&
		strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "process_pty_cli") &&
		strings.EqualFold(strings.TrimSpace(handle.Transporte), "cli") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) &&
		externalSessionReady
	normalizeMode := func(raw string) string {
		mode := runtimeagente.NormalizeMailboxDeliveryMode(raw)
		if mode == "" {
			return ""
		}
		if mode == runtimeagente.MailboxDeliverySessionResume && !externalSessionReady {
			if runtimepolicy.RuntimeHandleUsaTMUXPreferredCLI(meta) {
				return runtimeagente.MailboxDeliveryBootstrapOnly
			}
			if runtimeHandlePuedeInteractuarAntesDeSessionResume(handle, meta) {
				return runtimeagente.MailboxDeliveryInteractive
			}
			return runtimeagente.MailboxDeliveryBootstrapOnly
		}
		if legacyTMUXPreferredCLI && mode == runtimeagente.MailboxDeliveryInteractive {
			return runtimeagente.MailboxDeliveryBootstrapOnly
		}
		return mode
	}
	explicitMode := normalizeMode(runtimeHandleMailboxDeliveryModeExplicit(handle))
	if explicitMode != "" {
		if explicitMode == runtimeagente.MailboxDeliveryBootstrapOnly && (processSessionResumeReady || tmuxSessionResumeReady) {
			return runtimeagente.MailboxDeliverySessionResume
		}
		return explicitMode
	}
	if tmuxSessionResumeReady || processSessionResumeReady {
		return runtimeagente.MailboxDeliverySessionResume
	}
	if localCLIBrokerNoInteractive {
		return runtimeagente.MailboxDeliveryBootstrapOnly
	}
	if runtimeHandleUsaCodexTTYInestable(meta) &&
		!boolFromMap(meta, "mailbox_restart_safe") &&
		!boolFromMap(caps, "mailbox_restart_safe") &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) {
		return runtimeagente.MailboxDeliveryBootstrapOnly
	}
	if !RuntimeHandlePermiteSendInputInteractivo(handle) {
		return runtimeagente.MailboxDeliveryBootstrapOnly
	}
	return runtimeagente.MailboxDeliveryInteractive
}

func runtimeHandleTieneExternalSessionID(handle *RuntimeHandle, runtime *RuntimeInstance) bool {
	if handle == nil {
		return false
	}
	if ext := strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "external_session_id", "")); ext != "" {
		return true
	}
	if runtime != nil && strings.TrimSpace(runtime.ExternalSessionID) != "" {
		return true
	}
	if handle.SesionID != nil {
		sesion, err := sesionIfExists(handle.SesionID)
		if err == nil && sesion != nil && strings.TrimSpace(sesion.ExternalSessionID) != "" {
			return true
		}
	}
	return false
}

func RuntimeHandlePermiteSendInputInteractivo(handle *RuntimeHandle) bool {
	if handle == nil {
		return true
	}
	meta := mapFromJSON(handle.MetadataJSON)
	caps := mapFromJSON(handle.CapabilitiesJSON)
	if runtimeHandlePuedeInteractuarAntesDeSessionResume(handle, meta) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(stringFromMap(meta, "auth_mode", ""))) {
	case "oauth":
		return false
	}
	deliveryMode := runtimeHandleMailboxDeliveryModeExplicit(handle)
	if deliveryMode == runtimeagente.MailboxDeliverySessionResume {
		if runtimeHandleTMUXSessionResumePuedeInteractuar(handle, meta) {
			return true
		}
		if runtimeHandleUsaCodexTTYInestable(meta) {
			return false
		}
	}
	if value, ok := meta["can_send_input"]; ok && !boolFromAny(value) {
		return false
	}
	if value, ok := caps["can_send_input"]; ok && !boolFromAny(value) {
		return false
	}
	if runtimeHandleUsaCodexTTYInestable(meta) {
		return false
	}
	if runtimeHandleLocalProcesoSinCanalInteractivo(handle, meta) {
		return false
	}
	return true
}

func runtimeHandleTMUXSessionResumePuedeInteractuar(handle *RuntimeHandle, meta map[string]any) bool {
	if handle == nil {
		return false
	}
	deliveryModeRaw := strings.TrimSpace(stringFromMap(meta, "mailbox_delivery_mode", ""))
	if deliveryModeRaw == "" {
		deliveryModeRaw = strings.TrimSpace(stringFromMap(mapFromJSON(handle.CapabilitiesJSON), "mailbox_delivery_mode", ""))
	}
	deliveryMode := runtimeagente.NormalizeMailboxDeliveryMode(deliveryModeRaw)
	if deliveryMode != runtimeagente.MailboxDeliverySessionResume {
		return false
	}
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	if !strings.EqualFold(driver, "tmux_cli_session") && !strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") {
		return false
	}
	if strings.TrimSpace(stringFromMap(meta, "tmux_session", "")) == "" && strings.TrimSpace(handle.HandleRef) == "" {
		return false
	}
	if strings.TrimSpace(stringFromMap(meta, "tmux_pane_id", "")) == "" && !strings.Contains(strings.TrimSpace(handle.HandleRef), "/%") {
		return false
	}
	return true
}

func runtimeHandlePuedeInteractuarAntesDeSessionResume(handle *RuntimeHandle, meta map[string]any) bool {
	if handle == nil || !runtimeHandleUsaCodexTTYInestable(meta) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "process_pty_cli") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Transporte), "cli") || !strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") {
		return false
	}
	if runtimeHandleTieneExternalSessionID(handle, nil) || strings.TrimSpace(stringFromMap(meta, "external_session_id", "")) != "" {
		return false
	}
	for _, key := range []string{"supervisor_ref", "stdin_path", "stdin_raw_path"} {
		if strings.TrimSpace(stringFromMap(meta, key, "")) != "" {
			return true
		}
	}
	return false
}

func runtimeHandleUsaCodexTTYInestable(meta map[string]any) bool {
	if runtimepolicy.RuntimeHandleUsaLegacyCLITMUXPreferred(meta) {
		return true
	}
	if meta == nil {
		return false
	}
	for _, candidate := range []string{
		stringFromMap(meta, "herramienta", ""),
		stringFromMap(meta, "conector", ""),
		stringFromMap(meta, "profile_status_wrapper", ""),
	} {
		if runtimepolicy.RuntimeHandleLooksLikeCodexCLIRef(candidate) {
			return true
		}
	}
	for _, candidate := range []string{
		stringFromMap(meta, "rendered_command", ""),
		stringFromMap(meta, "wrapped_command", ""),
	} {
		if runtimepolicy.RuntimeHandleLooksLikeCodexCommand(candidate) {
			return true
		}
	}
	return false
}

func runtimeHandleLocalProcesoSinCanalInteractivo(handle *RuntimeHandle, meta map[string]any) bool {
	if handle == nil {
		return false
	}
	if strings.TrimSpace(handle.HandleKind) != "process" {
		return false
	}
	if strings.TrimSpace(stringFromMap(meta, "stdin_path", "")) != "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "cli") {
		return true
	}
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	if strings.EqualFold(driver, "process_pty_cli") {
		return true
	}
	if strings.TrimSpace(stringFromMap(meta, "supervisor_ref", "")) != "" {
		return true
	}
	if _, ok := meta["supervisor_owner_pid"]; ok {
		return true
	}
	if strings.TrimSpace(stringFromMap(meta, "supervision_mode", "")) != "" {
		return true
	}
	return false
}
