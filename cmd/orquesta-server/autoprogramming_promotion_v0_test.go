package main

import (
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestAutoprogrammingPromotionConfigFromEnvV0OptInYRefsOpacasV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(t.TempDir(), "state")
	config := orquestaserver.ConfigV0{ProjectWorkDir: projectDir, StateDir: stateDir}

	disabled := autoprogrammingPromotionConfigFromEnvV0(config)
	if disabled.Enabled || disabled.Port != nil {
		t.Fatalf("disabled=%+v", disabled)
	}

	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED", "1")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_REPO_REF", "repo-ref-autoprogramming-test")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_APP_REF", "app-ref-autoprogramming-test")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_COMMIT_MESSAGE", "test: promote staging")
	enabled := autoprogrammingPromotionConfigFromEnvV0(config)
	if !enabled.Enabled ||
		enabled.Port == nil ||
		enabled.RepoRef != "repo-ref-autoprogramming-test" ||
		enabled.AppRef != "app-ref-autoprogramming-test" ||
		enabled.CommitMessage != "test: promote staging" {
		t.Fatalf("enabled=%+v", enabled)
	}
}
