package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestOPESTopicRegistryEffectiveConfigRedactaToolPathV0(t *testing.T) {
	t.Setenv(envOPESTopicRegistryToolPathV0, "/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/tools/registro_trabajo_temas.py")
	t.Setenv(envOPESTopicRegistryAgentIDV0, "orquesta-registro")
	t.Setenv(envOPESTopicRegistryForceV0, "1")

	config := serverEffectiveConfigFromEnvV0(orquestaserver.ConfigV0{})

	if got := effectiveSettingValueForTopicRegistryTestV0(config.Settings, envOPESTopicRegistryEnabledV0); got != "true" {
		t.Fatalf("enabled=%q", got)
	}
	if got := effectiveSettingValueForTopicRegistryTestV0(config.Settings, envOPESTopicRegistryToolPathV0); got != "opes-topic-registry-tool-configured" {
		t.Fatalf("tool_path=%q", got)
	}
	if got := effectiveSettingValueForTopicRegistryTestV0(config.Settings, envOPESTopicRegistryAgentIDV0); got != "orquesta-registro" {
		t.Fatalf("agent_id=%q", got)
	}
	if got := effectiveSettingValueForTopicRegistryTestV0(config.Settings, envOPESTopicRegistryForceV0); got != "true" {
		t.Fatalf("force=%q", got)
	}
}

func TestOPESTopicRegistryConfigDescubreToolDesdeOPESProjectWorkDirV0(t *testing.T) {
	projectDir := t.TempDir()
	toolPath := filepath.Join(projectDir, "opes-salidas", "coordinacion_temarios", "tools", "registro_trabajo_temas.py")
	if err := os.MkdirAll(filepath.Dir(toolPath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(toolPath, []byte("#!/usr/bin/env python3\n"), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv(envOPESProjectWorkDirV0, projectDir)
	t.Setenv(envOPESTopicRegistryToolPathV0, "")
	t.Setenv(envOPESTopicRegistryEnabledV0, "")

	config := opesTopicRegistryConfigFromEnvV0()

	if !config.Enabled || config.ToolPath != toolPath {
		t.Fatalf("config=%+v tool=%s", config, toolPath)
	}
}

func TestOPESTopicRegistryConfigNoInyectaOPESLocalPorDefectoV0(t *testing.T) {
	t.Setenv(envOPESProjectWorkDirV0, "")
	t.Setenv(envOPESTopicRegistryToolPathV0, "")
	t.Setenv(envOPESTopicRegistryEnabledV0, "true")

	config := opesTopicRegistryConfigFromEnvV0()

	if !config.Enabled {
		t.Fatalf("enabled=false")
	}
	if config.ToolPath != "" {
		t.Fatalf("tool_path local inesperado=%q", config.ToolPath)
	}
}

func TestOPESTopicRegistryConfigDescubreToolDesdeFicheroCanonicoOPESV0(t *testing.T) {
	projectDir := t.TempDir()
	opesDir := filepath.Join(projectDir, "opes-workspace")
	toolPath := filepath.Join(opesDir, "opes-salidas", "coordinacion_temarios", "tools", "registro_trabajo_temas.py")
	if err := os.MkdirAll(filepath.Dir(toolPath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(toolPath, []byte("#!/usr/bin/env python3\n"), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"opes":{"project_workdir":"` + filepath.ToSlash(opesDir) + `"},
		"opes_topic_registry":{"enabled":true}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	projectConfig := projectConfigFromProjectDirBestEffortV0(projectDir)
	config := opesTopicRegistryConfigFromProjectConfigFileV0(projectConfig)

	if !config.Enabled || config.ToolPath != toolPath {
		t.Fatalf("config=%+v tool=%s", config, toolPath)
	}
}

func TestOPESTopicRegistryConfigLeeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	toolPath := filepath.Join(projectDir, "private", "registro_trabajo_temas.py")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"opes_topic_registry":{
			"enabled":true,
			"tool_path":"` + filepath.ToSlash(toolPath) + `",
			"agent_id":"agent-ref-file",
			"force":true
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	projectConfig := projectConfigFromServerConfigBestEffortV0(serverConfig)
	config := opesTopicRegistryConfigFromProjectConfigFileV0(projectConfig)
	if !config.Enabled || config.ToolPath != toolPath || config.AgentID != "agent-ref-file" || !config.Force {
		t.Fatalf("config=%+v tool=%s", config, toolPath)
	}
	settings := serverConfig.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envOPESTopicRegistryEnabledV0: "true",
		envOPESTopicRegistryAgentIDV0: "agent-ref-file",
		envOPESTopicRegistryForceV0:   "true",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s setting=%+v want value=%q source=config_file", key, setting, want)
		}
	}
	toolSetting := effectiveSettingForTestV0(settings, envOPESTopicRegistryToolPathV0)
	if toolSetting.Value != "opes-topic-registry-tool-configured" ||
		toolSetting.Source != configSettingSourceConfigFileV0 ||
		!toolSetting.Sensitive {
		t.Fatalf("tool setting=%+v", toolSetting)
	}
}

func effectiveSettingValueForTopicRegistryTestV0(
	settings []orquestaserver.ServerConfigSettingV0,
	key string,
) string {
	for _, setting := range settings {
		if setting.Key == key {
			return setting.Value
		}
	}
	return ""
}
