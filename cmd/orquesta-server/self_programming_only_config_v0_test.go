package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSelfProgrammingOnlyConfigV0AceptaPerfilAisladoV0(t *testing.T) {
	root := t.TempDir()
	setSelfProgrammingOnlyBaseEnvForTestV0(t, root)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !pathInsideSelfProgrammingRootV0(config.ProjectWorkDir, root) ||
		!pathInsideSelfProgrammingRootV0(config.RuntimeWorkDir, root) ||
		!pathInsideSelfProgrammingRootV0(config.StateDir, root) {
		t.Fatalf("dirs fuera de root: project=%s runtime=%s state=%s root=%s",
			config.ProjectWorkDir, config.RuntimeWorkDir, config.StateDir, root)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerSelfProgrammingOnlyV0); got != "true" {
		t.Fatalf("self programming effective=%q", got)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envCodexGoalBackendV0); got != codexGoalBackendAppServerTmuxV0 {
		t.Fatalf("backend=%q", got)
	}
}

func TestSelfProgrammingOnlyConfigV0RechazaOPESBaseURLV0(t *testing.T) {
	root := t.TempDir()
	setSelfProgrammingOnlyBaseEnvForTestV0(t, root)
	t.Setenv(envOPESBaseURLV0, "http://127.0.0.1:18086")

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), "self_programming_only_env_forbidden:"+envOPESBaseURLV0) {
		t.Fatalf("err=%v", err)
	}
}

func TestSelfProgrammingOnlyConfigV0RechazaProyectoFueraDeRootV0(t *testing.T) {
	root := t.TempDir()
	setSelfProgrammingOnlyBaseEnvForTestV0(t, root)
	t.Setenv(envCodexProjectWorkDirV0, filepath.Join(t.TempDir(), "orquesta"))

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), "self_programming_only_project_work_dir_outside_root") {
		t.Fatalf("err=%v", err)
	}
}

func TestSelfProgrammingOnlyConfigV0RechazaBackendProxyV0(t *testing.T) {
	root := t.TempDir()
	setSelfProgrammingOnlyBaseEnvForTestV0(t, root)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), "self_programming_only_requires_app_server_tmux") {
		t.Fatalf("err=%v", err)
	}
}

func TestSelfProgrammingOnlyConfigV0RechazaDomainWorkHTTPV0(t *testing.T) {
	root := t.TempDir()
	setSelfProgrammingOnlyBaseEnvForTestV0(t, root)
	t.Setenv(envDomainWorkHTTPBaseURLV0, "http://127.0.0.1:19000")

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), "self_programming_only_env_forbidden:"+envDomainWorkHTTPBaseURLV0) {
		t.Fatalf("err=%v", err)
	}
}

func TestSelfProgrammingOnlyConfigV0PermitePromotionAisladaDentroRootV0(t *testing.T) {
	root := t.TempDir()
	setSelfProgrammingOnlyBaseEnvForTestV0(t, root)
	t.Setenv(envServerAutoprogrammingPromotionEnabledV0, "true")
	t.Setenv(envServerAutoprogrammingPromotionArchiveDirV0, filepath.Join(root, "archive"))

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerSelfProgrammingOnlyV0); got != "true" {
		t.Fatalf("self programming effective=%q", got)
	}
}

func TestSelfProgrammingOnlyConfigV0RechazaPromotionArchiveFueraDeRootV0(t *testing.T) {
	root := t.TempDir()
	setSelfProgrammingOnlyBaseEnvForTestV0(t, root)
	t.Setenv(envServerAutoprogrammingPromotionEnabledV0, "true")
	t.Setenv(envServerAutoprogrammingPromotionArchiveDirV0, filepath.Join(t.TempDir(), "archive"))

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), "self_programming_only_promotion_archive_dir_outside_root") {
		t.Fatalf("err=%v", err)
	}
}

func setSelfProgrammingOnlyBaseEnvForTestV0(t *testing.T, root string) {
	t.Helper()
	for _, key := range selfProgrammingOnlyMustBeEmptyEnvV0() {
		t.Setenv(key, "")
	}
	for _, key := range selfProgrammingOnlyMustBeFalseEnvV0() {
		t.Setenv(key, "false")
	}
	t.Setenv(envServerSelfProgrammingOnlyV0, "true")
	t.Setenv(envServerSelfProgrammingRootV0, root)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexProjectWorkDirV0, filepath.Join(root, "project"))
	t.Setenv(envCodexRuntimeWorkDirV0, filepath.Join(root, "runtime"))
	t.Setenv(envServerStateDirV0, filepath.Join(root, "state"))
	t.Setenv(envServerIdleSelfImprovementProjectWorkDirV0, filepath.Join(root, "project"))
	t.Setenv(envServerResidentDirectorEnabledV0, "false")
	t.Setenv(envOPESBridgeDryRunV0, "true")
	t.Setenv(envOPESRegistryFinalPkgDryRunV0, "true")
}
