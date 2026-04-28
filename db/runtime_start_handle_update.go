package db

import (
	"encoding/json"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"strings"
)

func actualizarHandleRuntimeArranque(handleID int64, arranque *controlruntime.ProcesoArrancado, conector *Conector, plan *runtimeagente.LaunchPlan, resume runtimeagente.ResumeContext) error {
	if handleID == 0 || arranque == nil {
		return nil
	}
	meta := mapFromJSON(arranque.MetadataJSON)
	if meta == nil {
		meta = map[string]any{}
	}
	if strings.TrimSpace(arranque.StdinPath) != "" {
		meta["stdin_path"] = strings.TrimSpace(arranque.StdinPath)
	}
	if strings.TrimSpace(arranque.LogPath) != "" {
		meta["log_path"] = strings.TrimSpace(arranque.LogPath)
	}
	if strings.TrimSpace(arranque.WorkingDir) != "" {
		meta["working_dir"] = strings.TrimSpace(arranque.WorkingDir)
	}
	if strings.TrimSpace(arranque.WrappedCommand) != "" {
		meta["wrapped_command"] = strings.TrimSpace(arranque.WrappedCommand)
	}
	if strings.TrimSpace(arranque.RenderedCommand) != "" {
		meta["rendered_command"] = strings.TrimSpace(arranque.RenderedCommand)
	}
	if strings.TrimSpace(arranque.ExternalSessionID) != "" {
		meta["external_session_id"] = strings.TrimSpace(arranque.ExternalSessionID)
	}
	if conector != nil {
		meta["conector"] = strings.TrimSpace(conector.Slug)
	}
	if plan != nil {
		meta["modo_plan"] = strings.TrimSpace(plan.Modo)
		meta["continuity_prompt"] = strings.TrimSpace(plan.ContinuityPrompt)
		meta["bootstrap_prompt"] = strings.TrimSpace(plan.BootstrapPrompt)
		meta["launch_prompt_embedded"] = plan.LaunchPromptEmbedded
		meta["launch_prompt_mode"] = strings.TrimSpace(plan.LaunchPromptMode)
		meta["launch_prompt_delay_ms"] = plan.LaunchPromptDelayMS
	}
	if strings.TrimSpace(resume.ResumenContinuidad) != "" {
		meta["resumen_continuidad"] = strings.TrimSpace(resume.ResumenContinuidad)
	}
	compactarMetadataRuntimeHandle(meta)
	metaJSON, _ := json.Marshal(meta)
	caps := mapFromJSON(arranque.CapabilitiesJSON)
	if caps == nil {
		caps = map[string]any{}
	}
	if len(caps) == 0 {
		caps = map[string]any{
			"can_send_input":       true,
			"can_checkpoint":       true,
			"can_resume":           true,
			"can_capture_pid":      arranque.PID > 0,
			"can_track_continuity": true,
			"can_pause":            arranque.PID > 0,
			"can_stop":             true,
		}
	}
	capsJSON, _ := json.Marshal(caps)
	handleKind := strings.TrimSpace(arranque.HandleKind)
	if handleKind == "" {
		if arranque.PID > 0 {
			handleKind = "process"
		} else {
			handleKind = "session"
		}
	}
	handleRef := strings.TrimSpace(arranque.HandleRef)
	if handleRef == "" {
		if arranque.PID > 0 {
			handleRef = jsonNumber(int64(arranque.PID))
		} else if strings.TrimSpace(arranque.ExternalSessionID) != "" {
			handleRef = strings.TrimSpace(arranque.ExternalSessionID)
		}
	}
	transporte := inferirTransporteArranque(arranque)
	_, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte = ?,
		    handle_kind = ?,
		    handle_ref = ?,
		    metadata_json = ?,
		    capabilities_json = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, transporte, handleKind, handleRef, string(metaJSON), string(capsJSON), handleID)
	return runtimeHandleHotResetOnSuccess(err)
}
