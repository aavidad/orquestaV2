package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
)

func TestCodexUsageMetricsFromEnvV0OptIn(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_USAGE_ACCOUNTING", "")
	if got := codexUsageMetricsFromEnvV0(nil); got != nil {
		t.Fatalf("usage metrics sin opt-in: %#v", got)
	}
	t.Setenv("ORQUESTA_CODEX_USAGE_ACCOUNTING", "runtime_logs")
	if got := codexUsageMetricsFromEnvV0(nil); got != nil {
		t.Fatalf("usage metrics no debe leer logs runtime sin reporte redactado")
	}
	t.Setenv("ORQUESTA_CODEX_USAGE_ACCOUNTING", "redacted_report")
	t.Setenv("ORQUESTA_CODEX_USAGE_LOG_MAX_BYTES", "2048")
	if got := codexUsageMetricsFromEnvV0(nil); got == nil {
		t.Fatalf("usage metrics no habilitado con opt-in")
	}
	t.Setenv("ORQUESTA_CODEX_USAGE_ACCOUNTING", "runtime_usage_report")
	if got := codexUsageMetricsFromEnvV0(nil); got == nil {
		t.Fatalf("usage metrics no habilitado con runtime_usage_report")
	}
}

func TestCodexUsageMetricsFromProjectConfigV0OptIn(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"codex_usage_accounting":{
			"mode":"redacted_report",
			"log_max_bytes":4096
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	provider := codexUsageMetricsFromProjectConfigV0(projectDir, nil)
	source, ok := provider.(orquestaappcodexstack.CodexStackRuntimeUsageMetricsSourceV0)
	if !ok || source.MaxBytes != 4096 {
		t.Fatalf("provider=%T %+v", provider, provider)
	}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	mode := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexUsageAccountingV0)
	maxBytes := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexUsageLogMaxBytesV0)
	if mode.Value != "redacted_report" ||
		mode.Source != configSettingSourceConfigFileV0 ||
		maxBytes.Value != "4096" ||
		maxBytes.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("settings mode=%+v max=%+v", mode, maxBytes)
	}
}

func TestCodexUsageMetricsFromProjectConfigV0EnvGana(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexUsageAccountingV0, "runtime_usage_report")
	t.Setenv(envCodexUsageLogMaxBytesV0, "8192")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"codex_usage_accounting":{
			"mode":"redacted_report",
			"log_max_bytes":4096
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	provider := codexUsageMetricsFromProjectConfigV0(projectDir, nil)
	source, ok := provider.(orquestaappcodexstack.CodexStackRuntimeUsageMetricsSourceV0)
	if !ok || source.MaxBytes != 8192 {
		t.Fatalf("provider=%T %+v", provider, provider)
	}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	mode := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexUsageAccountingV0)
	maxBytes := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexUsageLogMaxBytesV0)
	if mode.Value != "runtime_usage_report" ||
		mode.Source != "explicit" ||
		maxBytes.Value != "8192" ||
		maxBytes.Source != "explicit" {
		t.Fatalf("settings mode=%+v max=%+v", mode, maxBytes)
	}
}

func TestInt64EnvOrDefaultV0(t *testing.T) {
	const key = "ORQUESTA_CODEX_USAGE_ACCOUNTING_TEST_VALUE"
	_ = os.Unsetenv(key)
	if got := int64EnvOrDefaultV0(key, 10); got != 10 {
		t.Fatalf("default=%d", got)
	}
	t.Setenv(key, "42")
	if got := int64EnvOrDefaultV0(key, 10); got != 42 {
		t.Fatalf("value=%d", got)
	}
}
