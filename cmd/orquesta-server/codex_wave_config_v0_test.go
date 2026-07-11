package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexWaveConfigV0UsaSandboxWorkspaceWritePorDefecto(t *testing.T) {
	t.Setenv(envCodexWaveSandboxV0, "")

	config := mustCodexWaveConfigForTestV0(t, nil)

	if config.Sandbox != "workspace-write" {
		t.Fatalf("sandbox=%q want workspace-write", config.Sandbox)
	}
}

func TestCodexWaveConfigV0DangerFullAccessSoloOptInExplicito(t *testing.T) {
	config := mustCodexWaveConfigForTestV0(t, []string{"--sandbox", "danger-full-access"})

	if config.Sandbox != "danger-full-access" {
		t.Fatalf("sandbox=%q want danger-full-access", config.Sandbox)
	}
}

func TestCodexWaveConfigV0ConservaReasoningHighYXHigh(t *testing.T) {
	for _, effort := range []string{"high", "xhigh"} {
		t.Run(effort, func(t *testing.T) {
			config := mustCodexWaveConfigForTestV0(t, []string{"--reasoning-effort", effort})

			if config.ReasoningEffort != effort {
				t.Fatalf("reasoning_effort=%q want %q", config.ReasoningEffort, effort)
			}
		})
	}
}

func TestCodexWaveConfigsV0NoHeredanModeloNiPerfilGlobales(t *testing.T) {
	t.Setenv(envCodexModelV0, "modelo-global")
	t.Setenv(envCodexProfileV0, "perfil-global")
	t.Setenv(envCodexWaveModelV0, "")
	t.Setenv(envCodexWaveProfileV0, "")

	wave := mustCodexWaveConfigForTestV0(t, nil)
	if wave.Model != "" || wave.Profile != "" {
		t.Fatalf("ola heredo modelo/perfil globales: %+v", wave)
	}

	projectDir := t.TempDir()
	var stderr strings.Builder
	director, err := codexDirectorWaveConfigFromArgsV0([]string{
		"--dry-run", "--wave-ref", "wave-director", "--objective", "probar aislamiento",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(projectDir, ".orquesta-runtime", "codex-waves", "wave-director"),
		"--command", codexWaveTestExecutablePathV0(t),
	}, &stderr)
	if err != nil {
		t.Fatalf("codexDirectorWaveConfigFromArgsV0: %v stderr=%s", err, stderr.String())
	}
	if director.Wave.Model != "" || director.Wave.Profile != "" {
		t.Fatalf("director wave heredo modelo/perfil globales: %+v", director.Wave)
	}

	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	for _, key := range []string{envCodexWaveModelV0, envCodexWaveProfileV0} {
		if setting := effectiveSettingForTestV0(serverConfig.EffectiveConfig.Settings, key); setting.Value != "" {
			t.Fatalf("effective %s heredo global: %+v", key, setting)
		}
	}
}

func TestCodexWaveConfigV0LeeFicheroCanonicoYEnvDeprecatedOverrideV0(t *testing.T) {
	projectDir := t.TempDir()
	sourceHome := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".orquesta-runtime", "codex-waves", "wave-file")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexWaveModelV0, "gpt-env-override")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"codex_wave":{
			"agents":3,
			"wave_ref":"wave-file",
			"runtime_workdir":"` + filepath.ToSlash(runtimeDir) + `",
			"source_code_home":"` + filepath.ToSlash(sourceHome) + `",
			"model":"gpt-file",
			"reasoning_effort":"high",
			"profile":"profile-file",
			"sandbox":"workspace-write",
			"approval_policy":"never",
			"extra_args":"--foo bar",
			"isolate_home":true,
			"strict_credential_projection":true,
			"project_memories":true,
			"projection_max_files":5,
			"projection_max_file_bytes":4096,
			"projection_max_total_bytes":8192,
			"purge_runtime":true,
			"purge_runtime_confirm":"wave-file",
			"purge_runtime_report":true,
			"allow_unmanaged_launch":true,
			"unmanaged_launch_reason":"breakglass-file",
			"unmanaged_launch_confirm":"wave-file",
			"path":"/bin:/usr/bin",
			"tail_reason":"tail-file",
			"stop_confirm":"wave-file",
			"stop_force":true
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	args := []string{
		"--prompt", "probar config",
		"--project-dir", projectDir,
		"--command", codexWaveTestExecutablePathV0(t),
		"--dry-run",
	}
	var stderr strings.Builder
	config, err := codexWaveConfigFromArgsV0(args, &stderr)
	if err != nil {
		t.Fatalf("codexWaveConfigFromArgsV0: %v stderr=%s", err, stderr.String())
	}
	if config.Agents != 3 ||
		config.WaveRef != "wave-file" ||
		config.RuntimeWorkDir != runtimeDir ||
		config.SourceCodeHome != sourceHome ||
		config.Model != "gpt-env-override" ||
		config.ReasoningEffort != "high" ||
		config.Profile != "profile-file" ||
		config.Sandbox != "workspace-write" ||
		config.ApprovalPolicy != "never" ||
		strings.Join(config.ExtraArgs, " ") != "--foo bar" ||
		!config.IsolateHome ||
		!config.CredentialProjectionPolicy.Strict ||
		!config.CredentialProjectionPolicy.ProjectMemories ||
		config.CredentialProjectionPolicy.MaxFiles != 5 ||
		config.CredentialProjectionPolicy.MaxFileBytes != 4096 ||
		config.CredentialProjectionPolicy.MaxTotalBytes != 8192 ||
		!config.PurgeRuntime ||
		config.PurgeConfirm != "wave-file" ||
		!config.PurgeReportOnly ||
		!config.AllowUnmanagedLaunch ||
		config.UnmanagedLaunchReason != "breakglass-file" ||
		config.UnmanagedLaunchConfirm != "wave-file" ||
		config.PathEnv != "/bin:/usr/bin" {
		t.Fatalf("codex wave config=%+v projection=%+v", config, config.CredentialProjectionPolicy)
	}

	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := serverConfig.EffectiveConfig.Settings
	if setting := effectiveSettingForTestV0(settings, envCodexWaveModelV0); setting.Value != "gpt-env-override" || setting.Source != "explicit" {
		t.Fatalf("model setting=%+v", setting)
	}
	if setting := effectiveSettingForTestV0(settings, envCodexWaveSandboxV0); setting.Value != "workspace-write" || setting.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("sandbox setting=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(serverConfig.EffectiveConfig, "deprecated_env_used", envCodexWaveModelV0, "codex_wave.*") {
		t.Fatalf("diagnostico deprecated codex_wave ausente: %+v", serverConfig.EffectiveConfig.Diagnostics)
	}
}

func mustCodexWaveConfigForTestV0(t *testing.T, extraArgs []string) codexWaveConfigV0 {
	t.Helper()
	projectDir := t.TempDir()
	args := []string{
		"--wave-ref", "wave-test",
		"--prompt", "probar config",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(projectDir, ".orquesta-runtime", "codex-waves", "wave-test"),
		"--command", codexWaveTestExecutablePathV0(t),
	}
	args = append(args, extraArgs...)
	var stderr strings.Builder
	config, err := codexWaveConfigFromArgsV0(args, &stderr)
	if err != nil {
		t.Fatalf("codexWaveConfigFromArgsV0: %v stderr=%s", err, stderr.String())
	}
	return config
}

func TestCodexWaveConfigV0RejectsRuntimeOutsideObservedRoots(t *testing.T) {
	projectDir := t.TempDir()
	args := []string{
		"--wave-ref", "wave-outside",
		"--prompt", "probar config",
		"--project-dir", projectDir,
		"--runtime-dir", t.TempDir(),
		"--command", codexWaveTestExecutablePathV0(t),
		"--dry-run",
	}
	var stderr strings.Builder
	if _, err := codexWaveConfigFromArgsV0(args, &stderr); err == nil || !strings.Contains(err.Error(), "runtime_dir_not_observable") {
		t.Fatalf("err=%v stderr=%s", err, stderr.String())
	}
}

func codexWaveTestExecutablePathV0(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return path
}
