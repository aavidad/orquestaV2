package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestExternalWorkRunProjectWorkDirGuardConfigV0NoInyectaOPESLocalPorDefecto(t *testing.T) {
	t.Setenv(envOPESProjectWorkDirV0, "")

	config := externalWorkRunProjectWorkDirGuardConfigFromEnvV0(orquestaserver.ConfigV0{
		ProjectWorkDir: "/home/alberto/Trabajo/orquesta",
	})

	if config.ProjectWorkDir != "/home/alberto/Trabajo/orquesta" {
		t.Fatalf("project_work_dir=%q", config.ProjectWorkDir)
	}
	if len(config.Rules) != 2 {
		t.Fatalf("rules=%+v", config.Rules)
	}
	for _, rule := range config.Rules {
		if rule.RequiredProjectWorkDir != "" {
			t.Fatalf("rule=%+v", rule)
		}
		if len(rule.AllowedLocalWriteSetPrefixes) != 1 ||
			rule.AllowedLocalWriteSetPrefixes[0] != opesLocalExternalWriteSetPrefixV0 {
			t.Fatalf("local prefixes=%+v", rule)
		}
	}
}

func TestExternalWorkRunProjectWorkDirGuardConfigV0PermiteOverride(t *testing.T) {
	t.Setenv(envOPESProjectWorkDirV0, "/tmp/opes-workspace")

	config := externalWorkRunProjectWorkDirGuardConfigFromEnvV0(orquestaserver.ConfigV0{
		ProjectWorkDir: "/tmp/other",
	})

	if got := config.Rules[0].RequiredProjectWorkDir; got != "/tmp/opes-workspace" {
		t.Fatalf("required=%q", got)
	}
}

func TestExternalWorkRunProjectWorkDirGuardConfigV0LeeFicheroCanonico(t *testing.T) {
	projectDir := t.TempDir()
	opesDir := filepath.Join(projectDir, "opes-workspace")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"opes":{"project_workdir":"` + filepath.ToSlash(opesDir) + `"}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config := externalWorkRunProjectWorkDirGuardConfigFromEnvV0(orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
	})

	if got := config.Rules[0].RequiredProjectWorkDir; got != opesDir {
		t.Fatalf("required=%q want %q", got, opesDir)
	}
}
