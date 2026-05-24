package main

import (
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
