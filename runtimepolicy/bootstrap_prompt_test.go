package runtimepolicy

import (
	"testing"

	"orquesta/runtimeagente"
)

func TestRuntimeLaunchBootstrapPromptCompact(t *testing.T) {
	canSendInput := false
	tests := []struct {
		name     string
		plan     *runtimeagente.LaunchPlan
		expected bool
	}{
		{
			name: "bootstrap only without input",
			plan: &runtimeagente.LaunchPlan{
				Driver:              "cli",
				CanSendInput:        &canSendInput,
				MailboxDeliveryMode: runtimeagente.MailboxDeliveryBootstrapOnly,
			},
			expected: true,
		},
		{
			name: "orchestrated cli command",
			plan: &runtimeagente.LaunchPlan{
				Driver:  "cli",
				Comando: "codex-perfil",
			},
			expected: true,
		},
		{
			name: "non compact",
			plan: &runtimeagente.LaunchPlan{
				Driver:  "cli",
				Comando: "bash",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RuntimeLaunchBootstrapPromptCompact(tt.plan); got != tt.expected {
				t.Fatalf("compacto esperado=%v got=%v", tt.expected, got)
			}
		})
	}
}

func TestRuntimeLaunchBootstrapPromptHasActiveTask(t *testing.T) {
	if !RuntimeLaunchBootstrapPromptHasActiveTask([]string{"asignada", "bloqueada"}) {
		t.Fatal("se esperaba true con estado activo")
	}
	if RuntimeLaunchBootstrapPromptHasActiveTask([]string{"libre", "completada"}) {
		t.Fatal("se esperaba false sin tareas activas")
	}
}
