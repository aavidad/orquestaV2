package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodexServerWorktreeSnapshotReadBudgetFromEnvV0(t *testing.T) {
	t.Setenv(envWorktreeSnapshotMaxFilesV0, "7")
	t.Setenv(envWorktreeSnapshotMaxFileBytesV0, "8")
	t.Setenv(envWorktreeSnapshotMaxTotalBytesV0, "9")

	budget := codexServerWorktreeSnapshotReadBudgetFromEnvV0()
	if budget.MaxFiles != 7 || budget.MaxFileBytes != 8 || budget.MaxTotalBytes != 9 {
		t.Fatalf("budget=%+v", budget)
	}
}

func TestCodexServerWorktreeSnapshotReadBudgetFromProjectConfigV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"worktree_snapshot":{
			"max_files":17,
			"max_file_bytes":18000,
			"max_total_bytes":19000
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	budget := codexServerWorktreeSnapshotReadBudgetFromProjectConfigV0(projectDir)
	if budget.MaxFiles != 17 || budget.MaxFileBytes != 18000 || budget.MaxTotalBytes != 19000 {
		t.Fatalf("budget=%+v", budget)
	}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	for _, key := range []string{
		envWorktreeSnapshotMaxFilesV0,
		envWorktreeSnapshotMaxFileBytesV0,
		envWorktreeSnapshotMaxTotalBytesV0,
	} {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("setting %s=%+v", key, setting)
		}
	}
}

func TestCodexServerWorktreeSnapshotReadBudgetEnvGanaAlProjectConfigV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envWorktreeSnapshotMaxFilesV0, "27")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"worktree_snapshot":{
			"max_files":17,
			"max_file_bytes":18000
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	budget := codexServerWorktreeSnapshotReadBudgetFromProjectConfigV0(projectDir)
	if budget.MaxFiles != 27 || budget.MaxFileBytes != 18000 {
		t.Fatalf("budget=%+v", budget)
	}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	files := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envWorktreeSnapshotMaxFilesV0)
	bytes := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envWorktreeSnapshotMaxFileBytesV0)
	if files.Source != "explicit" || bytes.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("settings files=%+v bytes=%+v", files, bytes)
	}
}
