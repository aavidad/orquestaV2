package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func codexWaveAgentPromptV0(config codexWaveConfigV0, agentRef string, index int) string {
	var b strings.Builder
	b.WriteString("Eres un agente Codex lanzado por Orquesta en una ola operativa opt-in.\n\n")
	b.WriteString("Identidad:\n")
	b.WriteString("- wave_ref: " + config.WaveRef + "\n")
	b.WriteString("- agent_ref: " + agentRef + "\n")
	b.WriteString("- agente: " + strconv.Itoa(index) + " de " + strconv.Itoa(config.Agents) + "\n\n")
	b.WriteString("Reglas operativas:\n")
	b.WriteString("- Trabaja en el repositorio indicado por Orquesta y respeta AGENTS.md locales antes de editar.\n")
	b.WriteString("- No borres archivos ni codigo existente sin revisar primero su uso y dejar evidencia clara.\n")
	b.WriteString("- Manten el write-set estrecho y coordina mentalmente tu parte con el resto de la ola.\n")
	b.WriteString("- No escribas checkpoint_started_* ni orquesta_goal_result_* dentro del repositorio: son recibos de ejecucion, no artefactos de producto.\n")
	b.WriteString("- Si una instruccion exige un recibo tecnico, usa solo el runtime del agente: " + filepath.Join(config.RuntimeWorkDir, fmt.Sprintf("agent-%02d", index)) + ".\n")
	b.WriteString("- Al terminar, resume cambios, rutas tocadas, pruebas ejecutadas y bloqueos.\n\n")
	b.WriteString("Instrucciones del operador:\n")
	b.WriteString(codexWavePromptForAgentV0(config, index))
	b.WriteString("\n")
	return b.String()
}

func codexWavePromptForAgentV0(config codexWaveConfigV0, index int) string {
	if index > 0 && index <= len(config.AgentPrompts) {
		if prompt := strings.TrimSpace(config.AgentPrompts[index-1]); prompt != "" {
			return prompt
		}
	}
	return config.Prompt
}
