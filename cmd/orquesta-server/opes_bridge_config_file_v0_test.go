package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOPESBridgeConfigLeeFicheroCanonicoCamposSegurosV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"opes":{"base_url":"http://127.0.0.1:18082"},
		"opes_bridge":{
			"dry_run":true,
			"job_type_sequence":["draft_content_block","generate_visual_asset"],
			"program_id":"program-ref-file",
			"topic_id":"topic-ref-file",
			"correlation_id":"corr-ref-file",
			"limit":4,
			"timeout_seconds":11,
			"priority":81,
			"interval_seconds":12,
			"initial_delay_seconds":2,
			"max_ticks":13,
			"supervise_submitted":true,
			"wait_resident_seconds":5,
			"wait_resident_interval_ms":250,
			"require_runtime_compatibility":true,
			"required_runtime_binary_sha256":"sha256-ref-file",
			"required_runtime_build_ref":"build-ref-file",
			"required_runtime_commit_ref":"commit-ref-file"
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	drainConfig, err := opesDrainConfigFromEnvWithBaseURLV0("http://127.0.0.1:18100")
	if err != nil {
		t.Fatalf("opesDrainConfigFromEnvWithBaseURLV0: %v", err)
	}
	if drainConfig.OPESBaseURL != "http://127.0.0.1:18082" ||
		drainConfig.OrquestaBaseURL != "http://127.0.0.1:18100" ||
		!drainConfig.DryRun ||
		drainConfig.ProgramID != "program-ref-file" ||
		drainConfig.TopicID != "topic-ref-file" ||
		drainConfig.CorrelationID != "corr-ref-file" ||
		drainConfig.Limit != 4 ||
		drainConfig.HTTPTimeout != 11*time.Second ||
		drainConfig.RunConfig.PriorityScore != 81 ||
		!drainConfig.SuperviseSubmitted ||
		drainConfig.ResidentDispatchWait != 5*time.Second ||
		drainConfig.ResidentDispatchPollInterval != 250*time.Millisecond ||
		!drainConfig.RuntimeCompatibility.Required ||
		drainConfig.RuntimeCompatibility.RequiredBinarySHA256 != "sha256-ref-file" ||
		drainConfig.RuntimeCompatibility.RequiredBuildRef != "build-ref-file" ||
		drainConfig.RuntimeCompatibility.RequiredCommitRef != "commit-ref-file" {
		t.Fatalf("drainConfig=%+v", drainConfig)
	}
	if got := filepath.ToSlash(filepath.Join(drainConfig.JobTypeSequence...)); got != "draft_content_block/generate_visual_asset" {
		t.Fatalf("job_type_sequence=%v", drainConfig.JobTypeSequence)
	}

	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := serverConfig.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envOPESBridgeJobTypeSequenceV0:             "draft_content_block,generate_visual_asset",
		envOPESBridgeProgramIDV0:                   "program-ref-file",
		envOPESBridgeTopicIDV0:                     "topic-ref-file",
		envOPESBridgeCorrelationIDV0:               "corr-ref-file",
		envOPESBridgeLimitV0:                       "4",
		envOPESBridgeTimeoutSecondsV0:              "11",
		envOPESBridgePriorityV0:                    "81",
		envOPESBridgeIntervalSecondsV0:             "12",
		envOPESBridgeInitialDelaySecondsV0:         "2",
		envOPESBridgeMaxTicksV0:                    "13",
		envOPESBridgeSuperviseSubmittedV0:          "true",
		envOPESBridgeWaitResidentSecondsV0:         "5",
		envOPESBridgeWaitResidentIntervalMSV0:      "250",
		envOPESBridgeRequireRuntimeCompatibilityV0: "true",
		envOPESBridgeRequiredRuntimeBinarySHA256V0: "sha256-ref-file",
		envOPESBridgeRequiredRuntimeBuildRefV0:     "build-ref-file",
		envOPESBridgeRequiredRuntimeCommitRefV0:    "commit-ref-file",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s setting=%+v want value=%q source=config_file", key, setting, want)
		}
	}
	base := effectiveSettingForTestV0(settings, envOPESBaseURLV0)
	if base.Value != "opes-base-url-configured" ||
		base.Source != configSettingSourceConfigFileV0 ||
		!base.Sensitive {
		t.Fatalf("opes base setting=%+v", base)
	}
}
