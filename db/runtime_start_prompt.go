package db

import (
	"encoding/json"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"strings"
	"time"
)

func inyectarPromptArranque(handle *RuntimeHandle, runtime *RuntimeInstance, plan *runtimeagente.LaunchPlan, order *RuntimeOrder) error {
	if handle == nil || runtime == nil || plan == nil {
		return nil
	}
	if runtimeHandleOmitePromptArranqueInteractivo(handle, runtime, plan) {
		return nil
	}
	if plan.LaunchPromptEmbedded && plan.CanSendInput != nil && !*plan.CanSendInput {
		return nil
	}
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	texto := controlruntime.NormalizarInstruccionProceso(obj, promptArranquePendiente(plan))
	if texto == "" {
		return nil
	}
	if plan.LaunchPromptDelayMS > 0 {
		time.Sleep(time.Duration(plan.LaunchPromptDelayMS) * time.Millisecond)
	}
	aplicado, _, err := controlruntime.EnviarInstruccionProceso(obj, texto)
	if err != nil {
		if runtimeOrderPromptArranqueDebeDiferirse(handle, err) {
			return encolarPromptArranqueEnMailbox(order, texto)
		}
		return err
	}
	if aplicado {
		return RegistrarRuntimeTranscriptInput(handle, runtime, texto)
	}
	return encolarPromptArranqueEnMailbox(order, texto)
}

func runtimeOrderPromptArranqueDebeDiferirse(handle *RuntimeHandle, runtimeErr error) bool {
	if handle == nil || runtimeErr == nil {
		return false
	}
	raw := strings.ToLower(strings.TrimSpace(runtimeErr.Error()))
	switch {
	case strings.Contains(raw, "requiere carpeta de confianza"),
		strings.Contains(raw, "requiere trust"),
		strings.Contains(raw, "untrusted folder"),
		strings.Contains(raw, "requiere autenticacion manual"),
		strings.Contains(raw, "bloqueado por cuota"),
		strings.Contains(raw, "tmux pane no listo para send-keys"):
		return true
	}
	reason, ok := runtimeOrderSendInstructionDiferirPorErrorTMUX(handle, runtimeErr)
	if !ok {
		return false
	}
	switch strings.TrimSpace(reason) {
	case "worker_not_ready", "worker_blocked_trust", "worker_blocked_auth", "worker_blocked_quota", "worker_starting":
		return true
	default:
		return false
	}
}

func encolarPromptArranqueEnMailbox(order *RuntimeOrder, texto string) error {
	if order == nil || strings.TrimSpace(texto) == "" {
		return nil
	}
	payloadJSON, _ := json.Marshal(map[string]any{
		"texto":       texto,
		"kind":        "instruction",
		"from_agente": "orquesta",
		"motivo":      "launch_prompt_start",
	})
	_, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    strings.TrimSpace(order.Agente),
		ProyectoID:  order.ProyectoID,
		Kind:        "instruction",
		PayloadJSON: string(payloadJSON),
	})
	return err
}

func runtimeHandleOmitePromptArranqueInteractivo(handle *RuntimeHandle, runtime *RuntimeInstance, plan *runtimeagente.LaunchPlan) bool {
	if handle == nil || runtime == nil || plan == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") {
		rendered := strings.TrimSpace(stringFromMap(meta, "rendered_command", ""))
		if rendered == "" {
			rendered = runtimeagente.RenderCommand(plan)
		}
		if controlruntime.RenderedCommandLooksLikeTMUXPreferredCLI(rendered) &&
			!strings.Contains(strings.ToLower(strings.TrimSpace(rendered)), "ollama") {
			return true
		}
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "api") {
		driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
		if strings.EqualFold(driver, "ollama_pool_local") {
			return true
		}
	}
	if strings.EqualFold(strings.TrimSpace(runtime.Connector), "ollama_pool_local") {
		return true
	}
	return false
}
