package main

import (
	"testing"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
)

func TestServerConfigFromEnvV0UsaPresupuestoDeComandosParaFronteraParalela(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	got := config.SupervisorCommand.DrainLimits.MaxCommands
	if got != orquestadirectorrunner.DirectorCycleMaxCommandsV0 {
		t.Fatalf("max_commands=%d want %d", got, orquestadirectorrunner.DirectorCycleMaxCommandsV0)
	}
}
