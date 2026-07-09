package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerConfigFromEnvV0RechazaBindRemotoSinOptInV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_ADDR", "0.0.0.0:8787")
	t.Setenv("ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM", "")
	t.Setenv("ORQUESTA_SERVER_CONTROL_TOKEN", "")

	if _, err := serverConfigFromEnvV0(); err == nil {
		t.Fatalf("serverConfigFromEnvV0 debe rechazar bind remoto sin opt-in")
	}
}

func TestServerConfigFromEnvV0AceptaBindRemotoConTokenOptInV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_ADDR", "0.0.0.0:8787")
	t.Setenv("ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM", "1")
	t.Setenv("ORQUESTA_SERVER_CONTROL_TOKEN", "secret-control-plane-token-0123456789")
	t.Setenv("ORQUESTA_SERVER_CONTROL_PRINCIPAL", "principal-ref-operator")
	t.Setenv("ORQUESTA_SERVER_CONTROL_PERMISSION_REF", "permission-ref-control-plane")
	t.Setenv("ORQUESTA_SERVER_CONTROL_PUBLIC_REASON", "remote_control_plane_opt_in")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.ControlPlane.RemoteAccessOptIn ||
		config.ControlPlane.Principal != "principal-ref-operator" ||
		config.ControlPlane.PermissionRef != "permission-ref-control-plane" {
		t.Fatalf("control plane config=%+v", config.ControlPlane)
	}
}

func TestServerConfigFromEnvV0LeeControlPlaneDesdeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, serverProjectConfigFileNameV0)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{"addr":"0.0.0.0:8787"},
		"control_plane":{
			"remote_access_opt_in":true,
			"token":"secret-control-plane-token-0123456789",
			"principal":"principal-ref-config",
			"permission_ref":"permission-ref-config",
			"public_reason":"remote_control_plane_config_file"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, err := serverConfigFromEnvWithProjectConfigPathV0(configPath)
	if err != nil {
		t.Fatalf("serverConfigFromEnvWithProjectConfigPathV0: %v", err)
	}
	if !config.ControlPlane.RemoteAccessOptIn ||
		config.ControlPlane.Token != "secret-control-plane-token-0123456789" ||
		config.ControlPlane.Principal != "principal-ref-config" ||
		config.ControlPlane.PermissionRef != "permission-ref-config" ||
		config.ControlPlane.PublicReason != "remote_control_plane_config_file" {
		t.Fatalf("control plane config=%+v", config.ControlPlane)
	}

	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envServerRemoteControlPlaneConfirmV0: "true",
		envServerControlPrincipalV0:          "principal-ref-config",
		envServerControlPermissionRefV0:      "permission-ref-config",
		envServerControlPublicReasonV0:       "remote_control_plane_config_file",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s setting=%+v want value=%q source=config_file", key, setting, want)
		}
	}
	tokenSetting := effectiveSettingForTestV0(settings, envServerControlTokenV0)
	if tokenSetting.Value != "present" ||
		tokenSetting.Source != configSettingSourceConfigFileV0 ||
		!tokenSetting.Sensitive {
		t.Fatalf("token setting=%+v", tokenSetting)
	}
	rawEffectiveConfig, err := json.Marshal(config.EffectiveConfig)
	if err != nil {
		t.Fatalf("marshal effective config: %v", err)
	}
	if strings.Contains(string(rawEffectiveConfig), "secret-control-plane-token") {
		t.Fatalf("effective_config filtra token crudo: %s", string(rawEffectiveConfig))
	}
}

func TestServerConfigFromEnvV0ControlPlaneEnvDeprecatedOverrideV0(t *testing.T) {
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, serverProjectConfigFileNameV0)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"control_plane":{
			"principal":"principal-ref-config",
			"permission_ref":"permission-ref-config"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv(envServerControlPrincipalV0, "principal-ref-env")

	config, err := serverConfigFromEnvWithProjectConfigPathV0(configPath)
	if err != nil {
		t.Fatalf("serverConfigFromEnvWithProjectConfigPathV0: %v", err)
	}
	if config.ControlPlane.Principal != "principal-ref-env" ||
		config.ControlPlane.PermissionRef != "permission-ref-config" {
		t.Fatalf("control plane config=%+v", config.ControlPlane)
	}
	principalSetting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerControlPrincipalV0)
	if principalSetting.Value != "principal-ref-env" || principalSetting.Source != "explicit" {
		t.Fatalf("principal setting=%+v", principalSetting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(
		config.EffectiveConfig,
		"deprecated_env_used",
		envServerControlPrincipalV0,
		"control_plane.*",
	) {
		t.Fatalf("diagnostico deprecated control_plane ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}
