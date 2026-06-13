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
