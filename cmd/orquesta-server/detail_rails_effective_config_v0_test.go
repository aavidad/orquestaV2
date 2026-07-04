package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerEffectiveConfigV0PublicaRailsDetalleCanonicos(t *testing.T) {
	t.Setenv(envDetailProhibitedRailsV0, "")
	t.Setenv(envDetailProhibitedRailsScopeV0, "")
	t.Setenv(envSecurityModeV0, "")
	t.Setenv(envRailsModeV0, "")
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envSecurityModeV0:               securityModeServerDefaultV0,
		envRailsModeV0:                  railsModeServerDefaultV0,
		envDetailProhibitedRailsV0:      detailProhibitedRailsServerDefaultV0,
		envDetailProhibitedRailsScopeV0: detailProhibitedRailsScopeServerDefaultV0,
	} {
		if got := effectiveSettingValueForDetailRailsTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q", key, got, want)
		}
		if source := effectiveSettingSourceForDetailRailsTestV0(settings, key); source != "defaulted" {
			t.Fatalf("%s source=%q want defaulted", key, source)
		}
	}
}

func TestServerEffectiveConfigV0PublicaRailsDetalleDesdeFicheroCanonico(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"rails_security":{
			"security_mode":"programming",
			"rails_mode":"enforced",
			"detail_prohibited_rails":"on",
			"detail_prohibited_rails_scope":"context_bundle_request.*"
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envSecurityModeV0:               "programming",
		envRailsModeV0:                  railsModeServerDefaultV0,
		envDetailProhibitedRailsV0:      "off",
		envDetailProhibitedRailsScopeV0: "context_bundle_request.*",
	} {
		if got := effectiveSettingValueForDetailRailsTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q", key, got, want)
		}
		if source := effectiveSettingSourceForDetailRailsTestV0(settings, key); source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s source=%q want config_file", key, source)
		}
	}
}

func TestServerEffectiveConfigV0ConservaFuenteFicheroTrasProyeccionDaemonRails(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(projectDir, "state")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{"state_dir":"` + filepath.ToSlash(stateDir) + `"},
		"rails_security":{
			"security_mode":"programming",
			"rails_mode":"enforced",
			"detail_prohibited_rails":"on",
			"detail_prohibited_rails_scope":"context_bundle_request.*"
		}
	}`
	configPath := filepath.Join(projectDir, serverProjectConfigFileNameV0)
	if err := os.WriteFile(configPath, []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	parentConfig, err := serverConfigFromEnvWithProjectConfigPathV0(configPath)
	if err != nil {
		t.Fatalf("serverConfigFromEnvWithProjectConfigPathV0 parent: %v", err)
	}
	for _, item := range serverDaemonStartEnvironmentV0(os.Environ(), parentConfig) {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		switch key {
		case envSecurityModeV0, envRailsModeV0, envDetailProhibitedRailsV0, envDetailProhibitedRailsScopeV0:
			t.Setenv(key, value)
		}
	}

	manualConfig, err := serverConfigFromEnvWithProjectConfigPathV0(configPath)
	if err != nil {
		t.Fatalf("serverConfigFromEnvWithProjectConfigPathV0 manual: %v", err)
	}
	if source := effectiveSettingSourceForDetailRailsTestV0(manualConfig.EffectiveConfig.Settings, envSecurityModeV0); source != "explicit" {
		t.Fatalf("run --config manual con env explicita debe conservar explicit, source=%q", source)
	}
	if _, err := serverDaemonRunArgsV0(parentConfig); err != nil {
		t.Fatalf("serverDaemonRunArgsV0: %v", err)
	}
	snapshotPath := filepath.Join(stateDir, serverDaemonConfigSnapshotDirV0, serverProjectConfigFileNameV0)
	childConfig, err := serverConfigFromEnvWithProjectConfigPathV0(snapshotPath)
	if err != nil {
		t.Fatalf("serverConfigFromEnvWithProjectConfigPathV0 child: %v", err)
	}
	for key, want := range map[string]string{
		envSecurityModeV0:               "programming",
		envRailsModeV0:                  railsModeServerDefaultV0,
		envDetailProhibitedRailsV0:      "off",
		envDetailProhibitedRailsScopeV0: "context_bundle_request.*",
	} {
		setting := effectiveSettingValueForDetailRailsTestV0(childConfig.EffectiveConfig.Settings, key)
		if setting != want {
			t.Fatalf("%s=%q want %q", key, setting, want)
		}
		source := effectiveSettingSourceForDetailRailsTestV0(childConfig.EffectiveConfig.Settings, key)
		if source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s source=%q want config_file", key, source)
		}
	}
}

func effectiveSettingValueForDetailRailsTestV0(settings []orquestaserver.ServerConfigSettingV0, key string) string {
	for _, setting := range settings {
		if setting.Key == key {
			return setting.Value
		}
	}
	return ""
}

func effectiveSettingSourceForDetailRailsTestV0(settings []orquestaserver.ServerConfigSettingV0, key string) string {
	for _, setting := range settings {
		if setting.Key == key {
			return setting.Source
		}
	}
	return ""
}
