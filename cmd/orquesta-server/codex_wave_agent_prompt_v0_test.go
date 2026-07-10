package main

import (
	"strings"
	"testing"
)

func TestCodexWaveAgentPromptV0SeparaRecibosTecnicosDelRepositorio(t *testing.T) {
	prompt := codexWaveAgentPromptV0(codexWaveConfigV0{
		WaveRef:        "wave-receipts",
		RuntimeWorkDir: "/runtime/wave-receipts",
		Agents:         2,
		Prompt:         "corrige solo el write-set",
	}, "wave-receipts-agent-02", 2)

	for _, want := range []string{
		"No escribas checkpoint_started_* ni orquesta_goal_result_* dentro del repositorio",
		"/runtime/wave-receipts/agent-02",
		"corrige solo el write-set",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, prompt)
		}
	}
}
