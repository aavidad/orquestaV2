package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
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

func TestSelfProgrammingOnlyStatusV0ExponePerfilAisladoSinFiltrarRootV0(t *testing.T) {
	root := t.TempDir()
	setSelfProgrammingOnlyBaseEnvForTestV0(t, root)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	tracker := orquestaserver.NewStatusTrackerV0(config, time.Unix(0, 0).UTC())
	tracker.MarkServingV0("127.0.0.1:19039", time.Unix(1, 0).UTC())
	handler := orquestaserver.NewHandlerV0(orquestaserver.HandlerConfigV0{Tracker: tracker})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, orquestaserver.ServerStatusEndpointV0, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, root) {
		t.Fatalf("status filtra self-programming root %q: %s", root, body)
	}
	var status orquestaserver.ServerPublicStatusV0
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v body=%s", err, body)
	}
	settings := status.EffectiveConfig.Settings
	if got := effectiveSettingValueForTestV0(settings, envServerSelfProgrammingOnlyV0); got != "true" {
		t.Fatalf("self programming status setting=%q", got)
	}
	rootSetting := effectiveSettingForTestV0(settings, envServerSelfProgrammingRootV0)
	if rootSetting.Value != orquestaserver.ServerStatusConfigHiddenValueV0 || !rootSetting.Sensitive {
		t.Fatalf("self root no redactado: %+v", rootSetting)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalBackendV0); got != codexGoalBackendAppServerTmuxV0 {
		t.Fatalf("goal backend status=%q", got)
	}
	if status.ConfigVisibility != orquestaserver.ServerStatusConfigVisibilityPublicRedactedV0 ||
		!containsStringV0(status.HiddenConfigFields, "effective_config."+envServerSelfProgrammingRootV0) ||
		status.ProjectWorkDirRef == "" ||
		status.RuntimeWorkDirRef == "" ||
		status.ResidentDirectorStatus != "disabled" {
		t.Fatalf("status publico insuficiente: %+v", status)
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
	outside := filepath.Join(t.TempDir(), "orquesta")
	t.Setenv(envCodexProjectWorkDirV0, outside)

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), "self_programming_only_project_work_dir_outside_root") {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Stat(outside); !os.IsNotExist(statErr) {
		t.Fatalf("project_work_dir fuera de root no debe crearse: stat=%v", statErr)
	}
}

func TestSelfProgrammingOnlyConfigV0RechazaDirsFueraDeRootSinCrearlosV0(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  string
		code string
	}{
		{
			name: "runtime",
			key:  envCodexRuntimeWorkDirV0,
			code: "self_programming_only_runtime_work_dir_outside_root",
		},
		{
			name: "state",
			key:  envServerStateDirV0,
			code: "self_programming_only_state_dir_outside_root",
		},
		{
			name: "idle",
			key:  envServerIdleSelfImprovementProjectWorkDirV0,
			code: "self_programming_only_idle_self_improvement_project_work_dir_outside_root",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			setSelfProgrammingOnlyBaseEnvForTestV0(t, root)
			outside := filepath.Join(t.TempDir(), tc.name)
			t.Setenv(tc.key, outside)

			_, err := serverConfigFromEnvV0()
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("err=%v", err)
			}
			if _, statErr := os.Stat(outside); !os.IsNotExist(statErr) {
				t.Fatalf("%s fuera de root no debe crearse: stat=%v", tc.key, statErr)
			}
		})
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

func TestSelfProgrammingOnlyConfigV0RechazaPromotionArchivePorSymlinkFueraDeRootV0(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "archive-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	setSelfProgrammingOnlyBaseEnvForTestV0(t, root)
	t.Setenv(envServerAutoprogrammingPromotionEnabledV0, "true")
	t.Setenv(envServerAutoprogrammingPromotionArchiveDirV0, filepath.Join(link, "archive"))

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
