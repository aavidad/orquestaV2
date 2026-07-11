package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaruntimeollama "orquesta/modulos/orquesta-runtime-ollama"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestOllamaModelManagerConfigFileV0LoadsTypedRuntimeModelsV0(t *testing.T) {
	projectDir := t.TempDir()
	makeProjectDirPrivateForTestV0(t, projectDir)
	secretDir := filepath.Join(projectDir, "secrets")
	if err := os.Mkdir(secretDir, 0o700); err != nil {
		t.Fatalf("mkdir secrets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "ollama.token"), []byte("file-token-should-not-leak\n"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	writeOllamaRuntimeModelsConfigForTestV0(t, projectDir, `"enabled":true,"base_url":"http://127.0.0.1:11435","timeout_seconds":17,"bearer_token_file":"secrets/ollama.token"`)

	config, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0))
	if err != nil {
		t.Fatalf("server config: %v", err)
	}
	projectConfig := projectConfigFromServerConfigBestEffortV0(config)
	port, err := runtimeModelManagerFromConfigV0(config, projectConfig)
	if err != nil {
		t.Fatalf("runtime models: %v", err)
	}
	manager, ok := port.(orquestaruntimeollama.OllamaModelManagerV0)
	if !ok || manager.BaseURL != "http://127.0.0.1:11435" || manager.BearerToken != "file-token-should-not-leak" || manager.HTTPClient.Timeout != 17*time.Second {
		t.Fatalf("manager=%#v", port)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envOllamaModelManagerBearerTokenV0)
	if setting.Value != "ollama-bearer-token-file-configured" || setting.Source != configSettingSourceConfigFileV0 || !setting.Sensitive {
		t.Fatalf("token setting=%+v", setting)
	}
	raw, err := json.Marshal(config.EffectiveConfig)
	if err != nil {
		t.Fatalf("marshal effective config: %v", err)
	}
	if strings.Contains(string(raw), "file-token-should-not-leak") || strings.Contains(string(raw), "secrets/ollama.token") {
		t.Fatalf("effective config filtra secreto: %s", raw)
	}
}

func TestOllamaModelManagerConfigFileV0EnvOverridesAreDeprecatedV0(t *testing.T) {
	projectDir := t.TempDir()
	writeOllamaRuntimeModelsConfigForTestV0(t, projectDir, `"enabled":false,"base_url":"http://127.0.0.1:11435","timeout_seconds":17`)
	t.Setenv(envOllamaModelManagerEnabledV0, "1")
	t.Setenv(envOllamaModelManagerBaseURLV0, "http://127.0.0.1:11436")
	t.Setenv(envOllamaModelManagerTimeoutSecondsV0, "19")

	config, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0))
	if err != nil {
		t.Fatalf("server config: %v", err)
	}
	projectConfig := projectConfigFromServerConfigBestEffortV0(config)
	port, err := runtimeModelManagerFromConfigV0(config, projectConfig)
	if err != nil {
		t.Fatalf("runtime models: %v", err)
	}
	manager := port.(orquestaruntimeollama.OllamaModelManagerV0)
	if manager.BaseURL != "http://127.0.0.1:11436" || manager.HTTPClient.Timeout != 19*time.Second {
		t.Fatalf("manager=%+v", manager)
	}
	for _, key := range []string{envOllamaModelManagerEnabledV0, envOllamaModelManagerBaseURLV0, envOllamaModelManagerTimeoutSecondsV0} {
		if setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key); setting.Source != "explicit" {
			t.Fatalf("%s setting=%+v", key, setting)
		}
		if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "deprecated_env_used", key, "runtime_models.*") {
			t.Fatalf("diagnostico %s ausente: %+v", key, config.EffectiveConfig.Diagnostics)
		}
	}
}

func TestReadServerProjectSecretFileV0RejectsUnsafePathsV0(t *testing.T) {
	projectDir := t.TempDir()
	makeProjectDirPrivateForTestV0(t, projectDir)
	config := serverConfigForSecretFileTestV0(projectDir)
	if err := os.WriteFile(filepath.Join(projectDir, "token"), []byte("safe"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	if token, err := readServerProjectSecretFileV0(config, "token"); err != nil || token != "safe" {
		t.Fatalf("safe token=%q err=%v", token, err)
	}
	if _, err := readServerProjectSecretFileV0(config, "../token"); err == nil {
		t.Fatal("path traversal debe rechazarse")
	}
	if err := os.Chmod(filepath.Join(projectDir, "token"), 0o644); err != nil {
		t.Fatalf("chmod token: %v", err)
	}
	if _, err := readServerProjectSecretFileV0(config, "token"); err == nil {
		t.Fatal("permisos inseguros deben rechazarse")
	}
	if err := os.Chmod(filepath.Join(projectDir, "token"), 0o600); err != nil {
		t.Fatalf("chmod token: %v", err)
	}
	if err := os.Symlink("token", filepath.Join(projectDir, "token-link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := readServerProjectSecretFileV0(config, "token-link"); err == nil {
		t.Fatal("symlink debe rechazarse")
	}
	unsafeDir := filepath.Join(projectDir, "unsafe")
	if err := os.Mkdir(unsafeDir, 0o770); err != nil {
		t.Fatalf("mkdir unsafe: %v", err)
	}
	if err := os.Chmod(unsafeDir, 0o770); err != nil {
		t.Fatalf("chmod unsafe: %v", err)
	}
	if err := os.WriteFile(filepath.Join(unsafeDir, "token"), []byte("unsafe"), 0o600); err != nil {
		t.Fatalf("write unsafe token: %v", err)
	}
	if _, err := readServerProjectSecretFileV0(config, "unsafe/token"); err == nil {
		t.Fatal("directorio padre escribible por grupo debe rechazarse")
	}
	safeDir := filepath.Join(projectDir, "safe")
	if err := os.Mkdir(safeDir, 0o700); err != nil {
		t.Fatalf("mkdir safe: %v", err)
	}
	if err := os.Symlink("safe", filepath.Join(projectDir, "safe-link")); err != nil {
		t.Fatalf("symlink intermedio: %v", err)
	}
	if _, err := readServerProjectSecretFileV0(config, "safe-link/token"); err == nil {
		t.Fatal("symlink intermedio debe rechazarse")
	}
	oversized := make([]byte, serverProjectSecretFileMaxBytesV0+1)
	if err := os.WriteFile(filepath.Join(projectDir, "oversized"), oversized, 0o600); err != nil {
		t.Fatalf("write oversized: %v", err)
	}
	if _, err := readServerProjectSecretFileV0(config, "oversized"); err == nil {
		t.Fatal("secreto sobredimensionado debe rechazarse")
	}
	rootLink := filepath.Join(t.TempDir(), "project-link")
	if err := os.Symlink(projectDir, rootLink); err != nil {
		t.Fatalf("root symlink: %v", err)
	}
	if _, err := readServerProjectSecretFileV0(serverConfigForSecretFileTestV0(rootLink), "token"); err == nil {
		t.Fatal("raiz symlink debe rechazarse")
	}
}

func TestOllamaModelManagerConfigFileV0RejectsInvalidInputsV0(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		config string
	}{
		{name: "scheme", config: `"enabled":true,"base_url":"ftp://127.0.0.1/model"`},
		{name: "userinfo", config: `"enabled":true,"base_url":"http://user@127.0.0.1"`},
		{name: "query", config: `"enabled":true,"base_url":"http://127.0.0.1?token=x"`},
		{name: "timeout zero", config: `"enabled":true,"timeout_seconds":0`},
		{name: "timeout max", config: `"enabled":true,"timeout_seconds":3601`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			projectDir := t.TempDir()
			writeOllamaRuntimeModelsConfigForTestV0(t, projectDir, testCase.config)
			if _, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0)); err == nil {
				t.Fatal("config invalida aceptada")
			}
		})
	}
}

func TestOllamaModelManagerConfigFileV0RejectsInvalidEnvV0(t *testing.T) {
	for _, testCase := range []struct {
		key   string
		value string
	}{
		{key: envOllamaModelManagerEnabledV0, value: "not-bool"},
		{key: envOllamaModelManagerTimeoutSecondsV0, value: "text"},
		{key: envOllamaModelManagerTimeoutSecondsV0, value: "0"},
		{key: envOllamaModelManagerTimeoutSecondsV0, value: "-1"},
		{key: envOllamaModelManagerTimeoutSecondsV0, value: "3601"},
		{key: envOllamaModelManagerTimeoutSecondsV0, value: "999999999999999999999"},
	} {
		t.Run(testCase.key+"="+testCase.value, func(t *testing.T) {
			projectDir := t.TempDir()
			writeOllamaRuntimeModelsConfigForTestV0(t, projectDir, `"enabled":true`)
			t.Setenv(testCase.key, testCase.value)
			if _, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0)); err == nil {
				t.Fatal("env invalida aceptada")
			}
		})
	}
}

func TestOllamaModelManagerConfigFileV0EnabledFalseGanaABaseURLV0(t *testing.T) {
	projectDir := t.TempDir()
	writeOllamaRuntimeModelsConfigForTestV0(t, projectDir, `"enabled":false,"base_url":"http://127.0.0.1:11435"`)
	config, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0))
	if err != nil {
		t.Fatalf("server config: %v", err)
	}
	port, err := runtimeModelManagerFromConfigV0(config, projectConfigFromServerConfigBestEffortV0(config))
	if err != nil || port != nil {
		t.Fatalf("disabled port=%#v err=%v", port, err)
	}
}

func TestReadServerProjectSecretFileV0UsaProjectWorkDirAunqueConfigSeaSnapshotV0(t *testing.T) {
	projectDir := t.TempDir()
	makeProjectDirPrivateForTestV0(t, projectDir)
	if err := os.WriteFile(filepath.Join(projectDir, "token"), []byte("source-token"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	config := serverConfigForSecretFileTestV0(projectDir)
	config.ProjectConfigFilePath = filepath.Join(t.TempDir(), "config-snapshots", serverProjectConfigFileNameV0)
	token, err := readServerProjectSecretFileV0(config, "token")
	if err != nil || token != "source-token" {
		t.Fatalf("token=%q err=%v", token, err)
	}
}

func writeOllamaRuntimeModelsConfigForTestV0(t *testing.T, projectDir string, runtimeModels string) {
	t.Helper()
	content := `{"schema_version":"orquesta_config.v0","runtime_models":{` + runtimeModels + `}}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func serverConfigForSecretFileTestV0(projectDir string) orquestaserver.ConfigV0 {
	return orquestaserver.ConfigV0{
		ProjectWorkDir:        projectDir,
		ProjectConfigFilePath: filepath.Join(projectDir, serverProjectConfigFileNameV0),
	}
}

func makeProjectDirPrivateForTestV0(t *testing.T, projectDir string) {
	t.Helper()
	if err := os.Chmod(projectDir, 0o700); err != nil {
		t.Fatalf("chmod project dir: %v", err)
	}
}
