package db

import (
	"encoding/json"
	"fmt"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"strings"
)

func objetivoProcesoParaOrden(order *RuntimeOrder) (controlruntime.ObjetivoProceso, error) {
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return controlruntime.ObjetivoProceso{}, err
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return controlruntime.ObjetivoProceso{}, err
	}
	return objetivoProcesoDesdeHandleRuntimeOrden(order, handle, runtime), nil
}

func runtimeHandleID(handle *RuntimeHandle) any {
	if handle == nil {
		return nil
	}
	return handle.ID
}

func runtimeInstanceID(runtime *RuntimeInstance) any {
	if runtime == nil {
		return nil
	}
	return runtime.ID
}

func payloadJSONDesdePlan(plan *runtimeagente.LaunchPlan) string {
	if plan == nil {
		return "{}"
	}
	data, err := json.Marshal(map[string]any{
		"modo":                   strings.TrimSpace(plan.Modo),
		"native_resume":          plan.NativeResume,
		"continuity_prompt":      strings.TrimSpace(plan.ContinuityPrompt),
		"bootstrap_prompt":       strings.TrimSpace(plan.BootstrapPrompt),
		"launch_prompt_embedded": plan.LaunchPromptEmbedded,
		"launch_prompt_mode":     strings.TrimSpace(plan.LaunchPromptMode),
		"launch_prompt_delay_ms": plan.LaunchPromptDelayMS,
	})
	if err != nil {
		return "{}"
	}
	return string(data)
}

func payloadJSONDesdePlanYResume(plan *runtimeagente.LaunchPlan, resume runtimeagente.ResumeContext) string {
	if plan == nil {
		return strings.TrimSpace(resume.ResumePayloadJSON)
	}
	return MergeResumePayloadPerfilEjecucion(
		resume.ResumePayloadJSON,
		strings.TrimSpace(plan.PerfilTarea),
		strings.TrimSpace(plan.Modelo),
		strings.TrimSpace(plan.Razonamiento),
	)
}

func construirTranscriptArranque(plan *runtimeagente.LaunchPlan, resume runtimeagente.ResumeContext, bootstrap *bootstrapRuntimeData) string {
	parts := []string{"Orquesta ha arrancado o reanudado este agente bajo control plane."}
	if plan != nil {
		if strings.TrimSpace(plan.Modo) != "" {
			parts = append(parts, fmt.Sprintf("Modo: %s.", strings.TrimSpace(plan.Modo)))
		}
		if strings.TrimSpace(plan.WorkingDir) != "" {
			parts = append(parts, fmt.Sprintf("Directorio: %s.", strings.TrimSpace(plan.WorkingDir)))
		}
		if strings.TrimSpace(plan.BootstrapPrompt) != "" {
			parts = append(parts, "Bootstrap inicial: "+strings.TrimSpace(plan.BootstrapPrompt))
		}
		if strings.TrimSpace(plan.ContinuityPrompt) != "" {
			parts = append(parts, "Prompt de continuidad: "+strings.TrimSpace(plan.ContinuityPrompt))
		}
		if plan.LaunchPromptEmbedded {
			parts = append(parts, "El prompt inicial quedó embebido en el comando de arranque.")
		} else if strings.TrimSpace(promptArranquePendiente(plan)) != "" {
			parts = append(parts, "El prompt inicial quedó programado para inyección tras arranque.")
		}
	}
	if strings.TrimSpace(resume.ResumenContinuidad) != "" {
		parts = append(parts, "Resumen de continuidad: "+strings.TrimSpace(resume.ResumenContinuidad))
	}
	if bootstrap != nil && bootstrap.Checkpoint != nil && bootstrap.Checkpoint.ID > 0 {
		parts = append(parts, fmt.Sprintf("Checkpoint de referencia: #%d.", bootstrap.Checkpoint.ID))
	}
	return strings.Join(parts, " ")
}

func branchDesdeResume(ultima *Sesion, resume runtimeagente.ResumeContext) string {
	if strings.TrimSpace(resume.Branch) != "" {
		return strings.TrimSpace(resume.Branch)
	}
	if ultima == nil {
		return ""
	}
	return strings.TrimSpace(ultima.Branch)
}

func resumenContinuidadDesdeResume(ultima *Sesion, resume runtimeagente.ResumeContext) string {
	if strings.TrimSpace(resume.ResumenContinuidad) != "" {
		return strings.TrimSpace(resume.ResumenContinuidad)
	}
	if ultima == nil {
		return ""
	}
	return strings.TrimSpace(ultima.ResumenContinuidad)
}

func inferirTransporteArranque(arranque *controlruntime.ProcesoArrancado) string {
	if arranque == nil {
		return ""
	}
	meta := mapFromJSON(arranque.MetadataJSON)
	for _, candidate := range []string{
		stringFromMap(meta, "transport", ""),
		stringFromMap(meta, "transporte", ""),
	} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	switch strings.TrimSpace(stringFromMap(meta, "driver", "")) {
	case "tmux_cli_session":
		return "tmux"
	case "process_pty_cli":
		return "cli"
	case "remote_http", "ollama_pool_local":
		return "api"
	}
	if strings.TrimSpace(arranque.HandleKind) == "session" {
		return "tmux"
	}
	return "cli"
}
