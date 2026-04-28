package db

import (
	"encoding/json"
	"strings"
)

func inferirTransporteSesion(s *Sesion) string {
	if s == nil {
		return "cli"
	}
	if strings.TrimSpace(s.ConectorSlug) != "" {
		if conector, err := GetConector(s.ConectorSlug); err == nil && strings.TrimSpace(conector.Transporte) != "" {
			return conector.Transporte
		}
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(s.Herramienta)), "mcp") {
		return "mcp_stdio"
	}
	return "cli"
}

func inferirHandleKindSesion(s *Sesion) string {
	if s == nil {
		return "session"
	}
	if s.PID != nil && *s.PID > 0 {
		return "process"
	}
	if strings.TrimSpace(s.ExternalSessionID) != "" {
		return "session"
	}
	return "session"
}

func inferirHandleRefSesion(s *Sesion) string {
	if s == nil {
		return ""
	}
	if s.PID != nil && *s.PID > 0 {
		return strings.TrimSpace(jsonNumber(*s.PID))
	}
	if strings.TrimSpace(s.ExternalSessionID) != "" {
		return strings.TrimSpace(s.ExternalSessionID)
	}
	return strings.TrimSpace(jsonNumber(s.ID))
}

func inferirEstadoHandleSesion(s *Sesion) string {
	if s == nil {
		return "fallido"
	}
	if s.Fin != nil || !s.Activa || s.Estado == "cerrada" || s.Estado == "fallida" {
		return "cerrado"
	}
	if s.Estado == "pausada" {
		return "pausado"
	}
	return "activo"
}

func inferirCapabilitiesSesion(s *Sesion) string {
	caps := map[string]any{
		"can_send_input":       true,
		"can_checkpoint":       true,
		"can_resume":           strings.TrimSpace(s.ExternalSessionID) != "",
		"can_capture_pid":      s.PID != nil && *s.PID > 0,
		"can_track_continuity": true,
	}
	for key, value := range extraerCapacidadesResumeSesion(s) {
		caps[key] = value
	}
	data, _ := json.Marshal(caps)
	return string(data)
}

func inferirMetadataSesion(s *Sesion) string {
	if s == nil {
		return "{}"
	}
	meta := map[string]any{
		"herramienta":         strings.TrimSpace(s.Herramienta),
		"external_session_id": strings.TrimSpace(s.ExternalSessionID),
		"branch":              strings.TrimSpace(s.Branch),
		"cwd":                 strings.TrimSpace(s.CWD),
	}
	for key, value := range extraerMetadataResumeSesion(s) {
		meta[key] = value
	}
	data, _ := json.Marshal(meta)
	return string(data)
}

func extraerMetadataResumeSesion(s *Sesion) map[string]any {
	if s == nil {
		return nil
	}
	envelope := ParseResumePayloadEnvelope(strings.TrimSpace(s.ResumePayloadJSON))
	if len(envelope) == 0 {
		return nil
	}
	out := map[string]any{}
	for _, key := range []string{
		"driver",
		"transport",
		"endpoint",
		"launch_path",
		"resume_path",
		"input_path",
		"status_path",
		"pause_path",
		"continue_path",
		"stop_path",
		"auth_header",
		"auth_token_env",
		"auth_mode",
		"mailbox_delivery_mode",
		"pool_compartido",
		"perfil_operativo",
		"can_send_input",
		"worktree_path",
		"tmux_session",
		"tmux_pane_id",
	} {
		if value, ok := envelope[key]; ok {
			out[key] = value
		}
	}
	if perfil, ok := envelope["perfil_ejecucion"].(map[string]any); ok {
		for _, key := range []string{
			"driver",
			"transport",
			"endpoint",
			"launch_path",
			"resume_path",
			"input_path",
			"status_path",
			"pause_path",
			"continue_path",
			"stop_path",
			"mailbox_delivery_mode",
			"pool_slug",
			"modelo",
			"perfil_tarea",
			"razonamiento",
			"perfil_operativo",
			"can_send_input",
			"worktree_path",
			"tmux_session",
			"tmux_pane_id",
		} {
			if value, ok := perfil[key]; ok {
				out[key] = value
			}
		}
	}
	return out
}

func extraerCapacidadesResumeSesion(s *Sesion) map[string]any {
	meta := extraerMetadataResumeSesion(s)
	if len(meta) == 0 {
		return nil
	}
	out := map[string]any{}
	for _, key := range []string{
		"can_send_input",
		"can_pause",
		"can_stop",
		"can_resume",
		"can_track_continuity",
		"can_stop_without_reauth",
		"mailbox_delivery_mode",
	} {
		if value, ok := meta[key]; ok {
			out[key] = value
		}
	}
	return out
}
