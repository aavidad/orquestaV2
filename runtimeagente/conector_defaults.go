package runtimeagente

import (
	"path/filepath"
	"strings"
)

func ConectorPorDefectoAgente(agente string) string {
	switch familiaAgente(agente) {
	case "ollama":
		return "ollama-cli"
	case "claude":
		return "claude-code"
	case "gemini":
		return "gemini-cli"
	case "codex":
		return "codex-cli"
	default:
		return "codex-cli"
	}
}

func ConectorCompatibleConAgente(agente, slug, comando string) bool {
	familia := familiaAgente(agente)
	if familia == "" {
		return true
	}
	familiaConector := familiaConector(slug, comando)
	if familiaConector == "" {
		return true
	}
	return familia == familiaConector
}

func EsConectorFamiliaOllama(slug, comando string) bool {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "ollama-cli" || slug == "ollama_pool_local" || slug == "ollama-pool-local" {
		return true
	}
	comando = strings.ToLower(strings.TrimSpace(comando))
	switch {
	case comando == "ollama", comando == "ollama-cli", strings.HasPrefix(comando, "ollama-perfil"):
		return true
	default:
		return false
	}
}

func familiaConector(slug, comando string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	comandoBase := strings.ToLower(strings.TrimSpace(filepath.Base(strings.TrimSpace(comando))))
	switch {
	case EsConectorFamiliaOllama(slug, comando):
		return "ollama"
	case slug == "codex-cli", comandoBase == "codex", comandoBase == "codex-cli", strings.HasPrefix(comandoBase, "codex-perfil"):
		return "codex"
	case slug == "claude-code", comandoBase == "claude", comandoBase == "claude-code":
		return "claude"
	case slug == "gemini-cli", comandoBase == "gemini":
		return "gemini"
	default:
		return ""
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

func ModeloPorDefectoAgente(agente string) string {
	agente = strings.ToLower(strings.TrimSpace(agente))
	switch {
	case strings.HasPrefix(agente, "gemma"):
		return "gemma4:26b"
	case strings.HasPrefix(agente, "qwen"):
		return "qwen2.5-coder:7b"
	default:
		return ""
	}
}

func ModeloAfinAgente(agente, modelo string) bool {
	agente = strings.ToLower(strings.TrimSpace(agente))
	modelo = strings.ToLower(strings.TrimSpace(modelo))
	if agente == "" || modelo == "" {
		return true
	}
	switch {
	case strings.HasPrefix(agente, "gemma"):
		return strings.HasPrefix(modelo, "gemma")
	case strings.HasPrefix(agente, "qwen"):
		return strings.HasPrefix(modelo, "qwen")
	case strings.HasPrefix(agente, "llama"):
		return strings.HasPrefix(modelo, "llama")
	case strings.HasPrefix(agente, "claude"):
		return strings.HasPrefix(modelo, "claude")
	case strings.HasPrefix(agente, "gemini"):
		return strings.HasPrefix(modelo, "gemini")
	case strings.HasPrefix(agente, "codex"):
		return modeloCompatibleFamiliaOpenAI(modelo)
	default:
		return true
	}
}

func ModeloPreferenteAgenteCompatible(agente string, conector ConnectorConfig) string {
	modelo := strings.TrimSpace(ModeloPorDefectoAgente(agente))
	if modelo == "" {
		return ""
	}
	if !ModeloCompatibleConConector(conector, modelo) {
		return ""
	}
	return modelo
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
