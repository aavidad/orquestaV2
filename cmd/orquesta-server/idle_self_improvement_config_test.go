package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestServerConfigFromEnvV0ConfiguraAutomejoraIdleV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.IdleSelfImprovementAfter != 60*time.Second ||
		config.IdleSelfImprovementDisabled ||
		config.IdleSelfImprovementMaxRequests != 10 ||
		config.IdleSelfImprovementTargetQueue != 10 ||
		len(config.IdleSelfImprovementWriteSet) == 0 ||
		config.IdleSelfImprovementRequiredTests[0] != "go test -count=1 ./..." {
		t.Fatalf("idle config=%+v", config)
	}

	t.Setenv("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS", "0")
	config, err = serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 disabled: %v", err)
	}
	if !config.IdleSelfImprovementDisabled || config.IdleSelfImprovementAfter != 0 {
		t.Fatalf("idle disabled config=%+v", config)
	}
}

func TestServerConfigFromEnvV0DesactivaAutomejoraIdleEnOPESSinWorkdirSeparadoV0(t *testing.T) {
	root := t.TempDir()
	opesDir := filepath.Join(root, "OPES")
	t.Setenv(envCodexProjectWorkDirV0, opesDir)
	t.Setenv(envOPESProjectWorkDirV0, opesDir)
	t.Setenv(envServerIdleSelfImprovementAfterV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementDisabled || config.IdleSelfImprovementAfter != 0 {
		t.Fatalf("automejora idle debe quedar desactivada en OPES sin workdir separado: %+v", config)
	}
	if config.IdleSelfImprovementProjectWorkDir != opesDir {
		t.Fatalf("idle_self_improvement_project_work_dir=%q want %q", config.IdleSelfImprovementProjectWorkDir, opesDir)
	}
}

func TestServerConfigFromEnvV0DesactivaAutomejoraIdleSiWorkdirExplicitoEsOPESV0(t *testing.T) {
	root := t.TempDir()
	opesDir := filepath.Join(root, "OPES")
	orquestaDir := filepath.Join(root, "orquesta")
	t.Setenv(envCodexProjectWorkDirV0, orquestaDir)
	t.Setenv(envOPESProjectWorkDirV0, opesDir)
	t.Setenv(envServerIdleSelfImprovementProjectWorkDirV0, opesDir)
	t.Setenv(envServerIdleSelfImprovementAfterV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementDisabled || config.IdleSelfImprovementAfter != 0 {
		t.Fatalf("automejora idle debe quedar desactivada si apunta al workdir OPES: %+v", config)
	}
}
