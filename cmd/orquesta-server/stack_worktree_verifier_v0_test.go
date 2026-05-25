package main

import (
	"path/filepath"
	"testing"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestBuildStackFromEnvV0CableaVerificadorWorktreeV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	source, ok := stack.Ports.DeliverySource.(orquestaruntimecodexdelivery.CodexDeliveryObservationSourceV0)
	if !ok || source.WorktreeVerifier == nil {
		t.Fatalf("delivery source sin verificador worktree: %T", stack.Ports.DeliverySource)
	}
}
