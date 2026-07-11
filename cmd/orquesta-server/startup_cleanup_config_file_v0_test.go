package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestStartupCleanupConfigFileV0ResuelveFicheroYEffectiveConfigV0(t *testing.T) {
	startupCleanupClearEnvForTestV0(t)
	config := startupCleanupServerConfigForTestV0(t, serverProjectConfigStartupCleanupV0{
		Mode:       startupCleanupStringPointerForTestV0(startupCleanupModeSelectiveV0),
		ScopeRefs:  startupCleanupStringsPointerForTestV0([]string{"project-ref-a", "run-ref-b"}),
		QueueLimit: startupCleanupIntPointerForTestV0(17),
	})

	port := startupCheckFromEnvV0(orquestaAppCodexStackZeroForStartupCleanupTestV0(), config)
	check, ok := port.(serverStartupCheckV0)
	if !ok || check.Mode != startupCleanupModeSelectiveV0 || check.QueueLimit != 17 ||
		!reflect.DeepEqual(check.ScopeRefs, []string{"project-ref-a", "run-ref-b"}) {
		t.Fatalf("startup check=%T %+v", port, port)
	}
	for _, key := range []string{envStartupCleanupModeV0, envStartupCleanupScopeRefsV0, envStartupQueueLimitV0} {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("setting %s=%+v", key, setting)
		}
	}
}

func TestStartupCleanupConfigFileV0EnvPrecedeFicheroV0(t *testing.T) {
	t.Setenv(envStartupCleanupModeV0, startupCleanupModeForcedStopV0)
	t.Setenv(envStartupCleanupScopeRefsV0, "run-ref-env")
	t.Setenv(envStartupQueueLimitV0, "23")
	config := startupCleanupServerConfigForTestV0(t, serverProjectConfigStartupCleanupV0{
		Mode:       startupCleanupStringPointerForTestV0(startupCleanupModeSelectiveV0),
		ScopeRefs:  startupCleanupStringsPointerForTestV0([]string{"run-ref-file"}),
		QueueLimit: startupCleanupIntPointerForTestV0(17),
	})

	check, ok := startupCheckFromEnvV0(orquestaAppCodexStackZeroForStartupCleanupTestV0(), config).(serverStartupCheckV0)
	if !ok || check.Mode != startupCleanupModeForcedStopV0 || check.QueueLimit != 23 ||
		!reflect.DeepEqual(check.ScopeRefs, []string{"run-ref-env"}) {
		t.Fatalf("startup check=%+v", check)
	}
}

func TestStartupCleanupConfigFileV0DaemonProyectaValoresCanonicosV0(t *testing.T) {
	startupCleanupClearEnvForTestV0(t)
	config := startupCleanupServerConfigForTestV0(t, serverProjectConfigStartupCleanupV0{
		Mode:       startupCleanupStringPointerForTestV0(startupCleanupModeSelectiveV0),
		ScopeRefs:  startupCleanupStringsPointerForTestV0([]string{"project-ref-daemon"}),
		QueueLimit: startupCleanupIntPointerForTestV0(31),
	})
	got := serverDaemonStartEnvironmentV0(nil, config)
	for key, value := range map[string]string{
		envStartupCleanupModeV0:      startupCleanupModeSelectiveV0,
		envStartupCleanupScopeRefsV0: "project-ref-daemon",
		envStartupQueueLimitV0:       "31",
	} {
		if !daemonStartEnvHasPairForTestV0(got, key, value) {
			t.Fatalf("daemon no proyecta %s=%s: %v", key, value, got)
		}
	}
}

func startupCleanupServerConfigForTestV0(
	t *testing.T,
	startup serverProjectConfigStartupCleanupV0,
) orquestaserver.ConfigV0 {
	t.Helper()
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	payload, err := json.Marshal(serverProjectConfigFileV0{
		SchemaVersion:  serverProjectConfigSchemaVersionV0,
		StartupCleanup: startup,
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(projectDir, serverProjectConfigFileNameV0)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := serverConfigFromEnvWithProjectConfigPathV0(path)
	if err != nil {
		t.Fatalf("server config: %v", err)
	}
	return config
}

func startupCleanupStringPointerForTestV0(value string) *string      { return &value }
func startupCleanupStringsPointerForTestV0(value []string) *[]string { return &value }
func startupCleanupIntPointerForTestV0(value int) *int               { return &value }

func startupCleanupClearEnvForTestV0(t *testing.T) {
	t.Helper()
	for _, key := range []string{envStartupCleanupModeV0, envStartupCleanupScopeRefsV0, envStartupQueueLimitV0} {
		t.Setenv(key, "")
	}
}

func orquestaAppCodexStackZeroForStartupCleanupTestV0() orquestaappcodexstack.StackV0 {
	return orquestaappcodexstack.StackV0{}
}
