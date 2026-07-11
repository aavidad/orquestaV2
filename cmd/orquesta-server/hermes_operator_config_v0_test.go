package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	operator "orquesta/modulos/orquesta-operator-mcp"
	operatorhermes "orquesta/modulos/orquesta-operator-mcp-hermes"
)

func TestHermesOperatorConfigV0LoadsFileRedactsAndWiresV0(t *testing.T) {
	projectDir := t.TempDir()
	makeProjectDirPrivateForTestV0(t, projectDir)
	if err := os.Mkdir(filepath.Join(projectDir, "secrets"), 0o700); err != nil {
		t.Fatalf("mkdir secrets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "secrets", "hermes.key"), []byte("hermes-file-secret\n"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	writeHermesOperatorConfigForTestV0(t, projectDir, `"enabled":true,"base_url":"https://hermes.example.test","mcp_path":"/operator/mcp","api_key_file":"secrets/hermes.key","status_tool":"remote.status","burst_tool":"remote.burst","outbox_tool":"remote.outbox","query_tool":"remote.query","status_connector_ref":"status-ref","burst_connector_ref":"burst-ref","outbox_connector_ref":"outbox-ref","query_connector_ref":"query-ref","timeout_seconds":17,"max_request_bytes":1234,"max_response_bytes":5678`)

	config, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0))
	if err != nil {
		t.Fatalf("server config: %v", err)
	}
	projectConfig := projectConfigFromServerConfigBestEffortV0(config)
	resolved, err := hermesOperatorConfigFromProjectConfigV0(projectConfig)
	if err != nil || !resolved.Enabled || resolved.APIKey != "" || resolved.APIKeyFile != "secrets/hermes.key" || resolved.BaseURL != "https://hermes.example.test" || resolved.MCPPath != "/operator/mcp" || resolved.TimeoutSeconds != 17 || resolved.MaxRequestBytes != 1234 || resolved.MaxResponseBytes != 5678 {
		t.Fatalf("resolved=%+v err=%v", resolved, err)
	}
	oldFactory := newHermesOperatorMCPConnectorV0
	defer func() { newHermesOperatorMCPConnectorV0 = oldFactory }()
	var captured operatorhermes.HermesOperatorMCPConfigV0
	newHermesOperatorMCPConnectorV0 = func(value operatorhermes.HermesOperatorMCPConfigV0) (operator.OperatorMCPConnectorV0, error) {
		captured = value
		return nil, nil
	}
	if _, err := hermesOperatorConnectorFromProjectConfigV0(config, projectConfig); err != nil {
		t.Fatalf("connector: %v", err)
	}
	if captured.APIKey != "hermes-file-secret" || captured.Timeout != 17*time.Second || captured.ToolNames.Status != "remote.status" || captured.ConnectorRefs.DirectedQuery != "query-ref" {
		t.Fatalf("captured=%+v", captured)
	}
	for key, want := range map[string]string{envHermesBaseURLV0: "hermes-base-url-configured", envHermesAPIKeyV0: "hermes-api-key-file-configured"} {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if !setting.Sensitive || setting.Source != configSettingSourceConfigFileV0 || setting.Value != want {
			t.Fatalf("%s setting=%+v", key, setting)
		}
	}
	counts := map[string]int{}
	for _, setting := range config.EffectiveConfig.Settings {
		counts[setting.Key]++
	}
	for _, key := range hermesOperatorEnvKeysV0() {
		if counts[key] != 1 {
			t.Fatalf("%s effective setting count=%d", key, counts[key])
		}
	}
	raw, err := json.Marshal(config.EffectiveConfig)
	if err != nil || strings.Contains(string(raw), "hermes-file-secret") || strings.Contains(string(raw), "secrets/hermes.key") || strings.Contains(string(raw), "hermes.example.test") {
		t.Fatalf("effective config leaked secret or URL: %s err=%v", raw, err)
	}
}

func TestHermesOperatorConfigV0EnvOverridesAndIsDeprecatedV0(t *testing.T) {
	projectDir := t.TempDir()
	writeHermesOperatorConfigForTestV0(t, projectDir, `"enabled":false,"base_url":"https://file.example.test","timeout_seconds":17`)
	t.Setenv(envHermesEnabledV0, "true")
	t.Setenv(envHermesBaseURLV0, "https://env.example.test")
	t.Setenv(envHermesTimeoutSecondsV0, "19")
	t.Setenv(envHermesAPIKeyV0, "legacy-secret")
	config, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0))
	if err != nil {
		t.Fatalf("server config: %v", err)
	}
	resolved, err := hermesOperatorConfigFromProjectConfigV0(projectConfigFromServerConfigBestEffortV0(config))
	if err != nil || !resolved.Enabled || resolved.BaseURL != "https://env.example.test" || resolved.TimeoutSeconds != 19 || resolved.APIKey != "legacy-secret" {
		t.Fatalf("resolved=%+v err=%v", resolved, err)
	}
	for _, key := range []string{envHermesEnabledV0, envHermesBaseURLV0, envHermesTimeoutSecondsV0, envHermesAPIKeyV0} {
		if setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key); setting.Source != "explicit" {
			t.Fatalf("%s setting=%+v", key, setting)
		}
		configPath := "hermes_operator.*"
		if key == envHermesAPIKeyV0 {
			configPath = "hermes_operator.api_key_file"
		}
		if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "deprecated_env_used", key, configPath) {
			t.Fatalf("missing diagnostic for %s: %+v", key, config.EffectiveConfig.Diagnostics)
		}
	}
}

func TestHermesOperatorConfigV0DisabledDoesNotReadSecretOrConnectV0(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		prepare func(*testing.T, string)
	}{
		{name: "missing"},
		{name: "insecure", prepare: func(t *testing.T, projectDir string) {
			if err := os.WriteFile(filepath.Join(projectDir, "key"), []byte("secret"), 0o644); err != nil {
				t.Fatalf("write key: %v", err)
			}
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			projectDir := t.TempDir()
			makeProjectDirPrivateForTestV0(t, projectDir)
			if testCase.prepare != nil {
				testCase.prepare(t, projectDir)
			}
			writeHermesOperatorConfigForTestV0(t, projectDir, `"enabled":false,"api_key_file":"key"`)
			config, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0))
			if err != nil {
				t.Fatalf("disabled startup must not read secret: %v", err)
			}
			setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envHermesAPIKeyV0)
			if setting.Value != "hermes-api-key-file-configured" || setting.Source != configSettingSourceConfigFileV0 || !setting.Sensitive {
				t.Fatalf("api key projection=%+v", setting)
			}
			oldFactory := newHermesOperatorMCPConnectorV0
			defer func() { newHermesOperatorMCPConnectorV0 = oldFactory }()
			factoryCalls := 0
			newHermesOperatorMCPConnectorV0 = func(value operatorhermes.HermesOperatorMCPConfigV0) (operator.OperatorMCPConnectorV0, error) {
				factoryCalls++
				return nil, nil
			}
			connector, err := hermesOperatorConnectorFromProjectConfigV0(config, projectConfigFromServerConfigBestEffortV0(config))
			if err != nil || connector != nil || factoryCalls != 0 {
				t.Fatalf("disabled connector=%v err=%v factory_calls=%d", connector, err, factoryCalls)
			}
		})
	}
}

func TestHermesOperatorConfigV0EnabledMissingSecretStillBuildsEffectiveConfigV0(t *testing.T) {
	projectDir := t.TempDir()
	makeProjectDirPrivateForTestV0(t, projectDir)
	writeHermesOperatorConfigForTestV0(t, projectDir, `"enabled":true,"base_url":"https://hermes.example.test","api_key_file":"missing.key"`)
	config, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0))
	if err != nil {
		t.Fatalf("startup/effective config must not read secret: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envHermesAPIKeyV0)
	if setting.Value != "hermes-api-key-file-configured" || setting.Source != configSettingSourceConfigFileV0 || !setting.Sensitive {
		t.Fatalf("api key projection=%+v", setting)
	}
	if _, err := hermesOperatorConnectorFromProjectConfigV0(config, projectConfigFromServerConfigBestEffortV0(config)); err == nil || !strings.Contains(err.Error(), "hermes_operator_api_key_file_invalid") {
		t.Fatalf("missing secret wiring err=%v", err)
	}
}

func TestHermesOperatorConfigV0EnabledUnsafeSecretFailsOnlyAtWiringV0(t *testing.T) {
	projectDir := t.TempDir()
	makeProjectDirPrivateForTestV0(t, projectDir)
	if err := os.WriteFile(filepath.Join(projectDir, "key"), []byte("secret"), 0o644); err != nil {
		t.Fatalf("write key: %v", err)
	}
	if err := os.Chmod(filepath.Join(projectDir, "key"), 0o644); err != nil {
		t.Fatalf("chmod key: %v", err)
	}
	writeHermesOperatorConfigForTestV0(t, projectDir, `"enabled":true,"base_url":"https://hermes.example.test","api_key_file":"key"`)
	config, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0))
	if err != nil {
		t.Fatalf("startup must not read unsafe secret: %v", err)
	}
	oldFactory := newHermesOperatorMCPConnectorV0
	defer func() { newHermesOperatorMCPConnectorV0 = oldFactory }()
	factoryCalls := 0
	newHermesOperatorMCPConnectorV0 = func(value operatorhermes.HermesOperatorMCPConfigV0) (operator.OperatorMCPConnectorV0, error) {
		factoryCalls++
		return nil, nil
	}
	if _, err := hermesOperatorConnectorFromProjectConfigV0(config, projectConfigFromServerConfigBestEffortV0(config)); err == nil || !strings.Contains(err.Error(), "hermes_operator_api_key_file_invalid") {
		t.Fatalf("unsafe secret wiring err=%v", err)
	}
	if factoryCalls != 0 {
		t.Fatalf("factory called before secret validation: %d", factoryCalls)
	}
}

func TestHermesOperatorConfigV0RejectsAdversarialInputsV0(t *testing.T) {
	for _, testCase := range []struct{ name, body string }{
		{"unknown plaintext key", `"enabled":true,"base_url":"https://hermes.example.test","api_key":"plaintext"`},
		{"invalid url", `"enabled":true,"base_url":"ftp://hermes.example.test"`},
		{"url query", `"enabled":true,"base_url":"https://hermes.example.test?token=x"`},
		{"invalid path", `"enabled":true,"base_url":"https://hermes.example.test","mcp_path":"mcp"`},
		{"timeout zero", `"timeout_seconds":0`},
		{"request bytes negative", `"max_request_bytes":-1`},
		{"response bytes zero", `"max_response_bytes":0`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			projectDir := t.TempDir()
			writeHermesOperatorConfigForTestV0(t, projectDir, testCase.body)
			if _, err := serverConfigFromEnvWithProjectConfigPathV0(filepath.Join(projectDir, serverProjectConfigFileNameV0)); err == nil {
				t.Fatal("config invalida aceptada")
			}
		})
	}
	t.Run("invalid enabled env", func(t *testing.T) {
		t.Setenv(envHermesEnabledV0, "perhaps")
		if _, err := serverConfigFromEnvV0(); err == nil {
			t.Fatal("enabled env invalido aceptado")
		}
	})
}

func writeHermesOperatorConfigForTestV0(t *testing.T, projectDir string, hermes string) {
	t.Helper()
	content := `{"schema_version":"orquesta_config.v0","hermes_operator":{` + hermes + `}}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
