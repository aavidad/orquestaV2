package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerEffectiveConfigV0ProjectsTypedRuntimeAndPromotionFamiliesFromConfigFile(t *testing.T) {
	projectDir := t.TempDir()
	clearEffectiveConfigFamilyEnvV0(t)
	writeEffectiveConfigFamiliesProjectConfigV0(t, projectDir)
	t.Setenv(envCodexProjectWorkDirV0, projectDir)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envGeminiEnabledV0:                               "true",
		envGeminiCommandV0:                               "gemini_runtime-configured",
		envGeminiProjectWorkDirV0:                        "gemini_runtime-configured",
		envGeminiModelV0:                                 "gemini_runtime-configured",
		envRequiredTestRunnerEnabledV0:                   "true",
		envRequiredTestGoCommandV0:                       "required_test_runner-configured",
		envRequiredTestAllowedCommandsV0:                 "required_test_runner-configured",
		envRequiredTestOutputDirV0:                       "required_test_runner-configured",
		envRequiredTestEnvV0:                             "required_test_runner-configured",
		envRequiredTestMaxOutputBytesV0:                  "4096",
		envRequiredTestOutputMaxArtifactsV0:              "4",
		envServerAutoprogrammingPromotionEnabledV0:       "true",
		envServerAutoprogrammingPromotionArchiveDirV0:    "autoprogramming-promotion-configured",
		envServerAutoprogrammingPromotionRepoRefV0:       "autoprogramming-promotion-configured",
		envServerAutoprogrammingPromotionCommitMessageV0: "autoprogramming-promotion-configured",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s=%+v want value=%q source=config_file", key, setting, want)
		}
	}
	for _, key := range []string{
		envGeminiCommandV0,
		envGeminiProjectWorkDirV0,
		envGeminiModelV0,
		envRequiredTestGoCommandV0,
		envRequiredTestAllowedCommandsV0,
		envRequiredTestOutputDirV0,
		envRequiredTestEnvV0,
		envServerAutoprogrammingPromotionArchiveDirV0,
		envServerAutoprogrammingPromotionRepoRefV0,
		envServerAutoprogrammingPromotionCommitMessageV0,
	} {
		if setting := effectiveSettingForTestV0(settings, key); !setting.Sensitive {
			t.Fatalf("%s debe redactarse: %+v", key, setting)
		}
	}
}

func TestServerEffectiveConfigV0FamilyEnvOverridesConfigFile(t *testing.T) {
	projectDir := t.TempDir()
	clearEffectiveConfigFamilyEnvV0(t)
	writeEffectiveConfigFamiliesProjectConfigV0(t, projectDir)
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envGeminiModelV0, "gemini-env-model")
	t.Setenv(envRequiredTestGoCommandV0, "/env/bin/go")
	t.Setenv(envServerAutoprogrammingPromotionAppRefV0, "app-ref-env")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	for _, key := range []string{envGeminiModelV0, envRequiredTestGoCommandV0, envServerAutoprogrammingPromotionAppRefV0} {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if setting.Source != "explicit" || !setting.Sensitive {
			t.Fatalf("%s=%+v want explicit sensitive", key, setting)
		}
	}
}

func TestServerEffectiveConfigV0FamilyDefaultsRemainAuditable(t *testing.T) {
	clearEffectiveConfigFamilyEnvV0(t)
	config := serverEffectiveConfigFromEnvV0(orquestaserver.ConfigV0{})
	for key, want := range map[string]string{
		envGeminiEnabledV0:                         "false",
		envRequiredTestRunnerEnabledV0:             "false",
		envRequiredTestMaxOutputBytesV0:            "1048576",
		envRequiredTestOutputMaxArtifactsV0:        "200",
		envServerAutoprogrammingPromotionEnabledV0: "false",
	} {
		setting := effectiveSettingForTestV0(config.Settings, key)
		if setting.Value != want || setting.Source != "defaulted" {
			t.Fatalf("%s=%+v want value=%q source=defaulted", key, setting, want)
		}
	}
}

func clearEffectiveConfigFamilyEnvV0(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		envGeminiEnabledV0, envGeminiProjectWorkDirV0, envGeminiRuntimeWorkDirV0, envGeminiCommandV0,
		envGeminiHomeV0, envGeminiPathV0, envGeminiModelV0, envGeminiApprovalModeV0, envGeminiOutputFormatV0, envGeminiExtraArgsV0,
		envRequiredTestRunnerEnabledV0, envRequiredTestMaxOutputBytesV0, envRequiredTestOutputMaxArtifactsV0,
		envRequiredTestGoCommandV0, envRequiredTestAllowedCommandsV0, envRequiredTestOutputDirV0, envRequiredTestEnvV0,
		envServerAutoprogrammingPromotionEnabledV0, envServerAutoprogrammingPromotionArchiveDirV0,
		envServerAutoprogrammingPromotionRepoRefV0, envServerAutoprogrammingPromotionAppRefV0, envServerAutoprogrammingPromotionCommitMessageV0,
	} {
		unsetEffectiveConfigFamilyEnvV0(t, key)
	}
}

func unsetEffectiveConfigFamilyEnvV0(t *testing.T, key string) {
	t.Helper()
	value, present := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if present {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func writeEffectiveConfigFamiliesProjectConfigV0(t *testing.T, projectDir string) {
	t.Helper()
	config := `{
		"schema_version":"orquesta_config.v0",
		"gemini_runtime":{"enabled":true,"command_path":"/private/gemini","project_work_dir":"/private/project","model":"gemini-file-model"},
		"required_test_runner":{"enabled":true,"go_command":"/private/go","allowed_commands":{"lint":"/private/lint"},"output_dir":"/private/output","environment":{"TOKEN":"private"},"max_output_bytes":4096,"max_artifacts":4},
		"autoprogramming":{"promotion":{"enabled":true,"archive_dir":"/private/archive","repo_ref":"repo-ref-private","app_ref":"app-ref-private","commit_message":"private commit"}}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
