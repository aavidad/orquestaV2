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

func TestRuntimeLaunchBootstrapPromptLooksLikeOrchestratedAgentCLINormalized(t *testing.T) {
	if !RuntimeLaunchBootstrapPromptLooksLikeOrchestratedAgentCLI(&runtimeagente.LaunchPlan{
		Driver:  " CLI ",
		Comando: " /usr/bin/CODEX-PERFIL --help ",
	}) {
		t.Fatal("esperaba true para CLI case-insensitive")
	}
	if RuntimeLaunchBootstrapPromptLooksLikeOrchestratedAgentCLI(&runtimeagente.LaunchPlan{}) {
		t.Fatal("esperaba false para plan sin driver ni comando relevante")
	}
}

func TestRuntimeLaunchBootstrapPromptLooksLikeOllamaCLI(t *testing.T) {
	if !RuntimeLaunchBootstrapPromptLooksLikeOllamaCLI(&runtimeagente.LaunchPlan{
		Args: []string{"--profile", "ollama"},
	}) {
		t.Fatal("esperaba true para plan con argumento ollama")
	}
	if RuntimeLaunchBootstrapPromptLooksLikeOllamaCLI(&runtimeagente.LaunchPlan{
		Driver:  "cli",
		Comando: "bash",
	}) {
		t.Fatal("esperaba false para plan no ollama")
	}
}

func TestRuntimeLaunchBootstrapPromptCompactTaskSummary(t *testing.T) {
	tasks := []RuntimeLaunchBootstrapTask{
		{ID: 1, Estado: "completada", Titulo: "cerrada"},
		{
			ID:          2,
			Estado:      "en_progreso",
			Titulo:      "extraer policy",
			Descripcion: "Write-set exclusivo: db/runtime_bootstrap_prompt.go y tests asociados. Tests minimos del slice: go test ./db -run 'TestBuildLaunchBootstrapPrompt.*' -count=1 y go test ./runtimesapp -count=1.",
		},
	}
	got := RuntimeLaunchBootstrapPromptCompactTaskSummary(tasks)
	want := "Tarea activa: #2 [en_progreso] extraer policy. Alcance inmediato: Write-set exclusivo: db/runtime_bootstrap_prompt.go y tests asociados. Tests minimos: go test ./db -run 'TestBuildLaunchBootstrapPrompt.*' ; go test ./runtimesapp."
	if got != want {
		t.Fatalf("summary mismatch\nwant: %q\ngot:  %q", want, got)
	}
}

func TestRuntimeLaunchBootstrapPromptCompactTaskDescriptionPrioritaria(t *testing.T) {
	raw := "Frente actual: limpiar prompt. Write-set exclusivo: db/runtime_bootstrap_prompt.go. Tests minimos del slice: go test ./db -run 'TestBuildLaunchBootstrapPrompt.*' -count=1 y go test ./runtimesapp -count=1."
	got := runtimeLaunchBootstrapPromptCompactTaskDescription(raw)
	want := "Write-set exclusivo: db/runtime_bootstrap_prompt.go. Tests minimos: go test ./db -run 'TestBuildLaunchBootstrapPrompt.*' ; go test ./runtimesapp"
	if got != want {
		t.Fatalf("description mismatch\nwant: %q\ngot:  %q", want, got)
	}
}
