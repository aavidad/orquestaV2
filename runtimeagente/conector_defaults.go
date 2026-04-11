package runtimeagente

import (
	"strings"
)

func ConectorPorDefectoAgente(agente string) string {
	switch familiaAgente(agente) {
	case "ollama":
		return "ollama-cli"
	default:
		return "codex-cli"
	}
}

func AplicarDefaultsConector(conector ConnectorConfig, perfilSolicitado, modeloSolicitado, razonamientoSolicitado, perfilActual, modeloActual, razonamientoActual string) (string, string, string, error) {
	metadata, err := parseAnyMapJSON(conector.MetadataJSON)
	if err != nil {
		return "", "", "", err
	}
	perfilActual = strings.TrimSpace(perfilActual)
	modeloActual = strings.TrimSpace(modeloActual)
	razonamientoActual = strings.TrimSpace(strings.ToLower(razonamientoActual))
	if strings.TrimSpace(perfilSolicitado) == "" && perfilActual == "" {
		if perfil := stringMetadata(metadata, "default_task_profile"); perfil != "" {
			perfilActual = perfil
		}
	}
	if strings.TrimSpace(modeloSolicitado) == "" && modeloActual == "" {
		if modelo := stringMetadata(metadata, "default_model"); modelo != "" {
			modeloActual = modelo
		}
	}
	if strings.TrimSpace(razonamientoSolicitado) == "" && razonamientoActual == "" {
		if razonamiento := stringMetadata(metadata, "default_reasoning_effort"); razonamiento != "" {
			razonamientoActual = strings.TrimSpace(strings.ToLower(razonamiento))
		}
	}
	return perfilActual, modeloActual, razonamientoActual, nil
}

func familiaAgente(agente string) string {
	agente = strings.ToLower(strings.TrimSpace(agente))
	switch {
	case strings.HasPrefix(agente, "ollama"),
		strings.HasPrefix(agente, "gemma"),
		strings.HasPrefix(agente, "llama"),
		strings.HasPrefix(agente, "qwen"):
		// gemma, llama y qwen corren vía ollama CLI
		return "ollama"
	case strings.HasPrefix(agente, "codex"):
		return "codex"
	case strings.HasPrefix(agente, "claude"):
		return "claude"
	case strings.HasPrefix(agente, "gemini"):
		return "gemini"
	default:
		return ""
	}
}
