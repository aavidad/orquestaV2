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
			"required_runtime_commit_ref":"commit-ref-file",
			"allow_unfiltered":true,
			"input_ledger_disabled":true,
			"input_ledger_path":"/tmp/opes-bridge-ledger-file.json",
			"destination_evidence_ref":"evidence-ref-opes-bridge-file",
			"speech_synthesis":{
				"capability":"available",
				"capability_ref":"speech-ref-file",
				"evidence_refs":"evidence-ref-speech-file",
				"network_ready":"true",
				"tool_path_ready":"true",
				"provider_quota_ready":"true"
			},
			"speech_synthesis_no_progress_timeout_seconds":321,
			"speech_synthesis_provider_timeout_ready":"true",
			"speech_synthesis_tool_workdir":"` + filepath.ToSlash(projectDir) + `",
			"speech_synthesis_tool_command":"true",
			"speech_synthesis_tool_preflight_timeout_seconds":9,
			"remote_qa":{
				"capability":"available",
				"capability_ref":"remote-qa-ref-file",
				"evidence_refs":"evidence-ref-remote-qa-file",
				"network_ready":"true",
				"provider_quota_ready":"true"
			},
			"remote_qa_auth_state_ready":"true"
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
	if drainConfig.InputLedger != nil {
		t.Fatalf("input ledger debe quedar desactivado desde fichero")
	}
	if drainConfig.Destination.DestinationEvidenceRef != "evidence-ref-opes-bridge-file" {
		t.Fatalf("destination=%+v", drainConfig.Destination)
	}
	if len(drainConfig.ExternalCapabilities) != 2 {
		t.Fatalf("external capabilities=%+v", drainConfig.ExternalCapabilities)
	}
	speech := drainConfig.ExternalCapabilities[0]
	if speech.CapabilityRef != "speech-ref-file" ||
		!speech.Available ||
		!speech.NetworkReady ||
		!speech.ToolPathReady ||
		!speech.ProviderQuotaReady ||
		!speech.ProviderTimeoutReady ||
		speech.ProviderNoProgressTimeoutSeconds != 321 ||
		speech.CommandTimeoutSeconds != 1800 {
		t.Fatalf("speech capability=%+v", speech)
	}
	remoteQA := drainConfig.ExternalCapabilities[1]
	if remoteQA.CapabilityRef != "remote-qa-ref-file" ||
		!remoteQA.Available ||
		!remoteQA.NetworkReady ||
		!remoteQA.AuthStateReady ||
		!remoteQA.ProviderQuotaReady ||
		remoteQA.CommandTimeoutSeconds != 1200 {
		t.Fatalf("remote qa capability=%+v", remoteQA)
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
		envOPESBridgeAllowUnfilteredV0:             "true",
		envOPESBridgeInputLedgerDisabledV0:         "true",
		envOPESBridgeDestinationEvidenceV0:         "evidence-ref-opes-bridge-file",
		envOPESBridgeSpeechSynthesisCapabilityV0:   "available",
		envOPESBridgeRemoteQACapabilityV0:          "available",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s setting=%+v want value=%q source=config_file", key, setting, want)
		}
	}
	for key, want := range map[string]string{
		envOPESBridgeInputLedgerPathV0:                            "opes-bridge-input-ledger-path-configured",
		envOPESBridgeSpeechSynthesisToolWorkDirV0:                 "opes-speech-synthesis-tool-workdir-configured",
		envOPESBridgeSpeechSynthesisToolCommandV0:                 "opes-speech-synthesis-tool-command-configured",
		envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0:        "true",
		envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0: "9",
		envOPESBridgeRemoteQAAuthStateReadyV0:                     "true",
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

func TestOPESBridgeConfigV0EnvDeprecatedOverridePrevaleceSobreFicheroV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envOPESBridgeLimitV0, "9")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"opes":{"base_url":"http://127.0.0.1:18082"},
		"opes_bridge":{
			"dry_run":true,
			"program_id":"program-ref-file",
			"limit":4
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	drainConfig, err := opesDrainConfigFromEnvWithBaseURLV0("http://127.0.0.1:18100")
	if err != nil {
		t.Fatalf("opesDrainConfigFromEnvWithBaseURLV0: %v", err)
	}
	if drainConfig.Limit != 9 {
		t.Fatalf("limit=%d want env override 9", drainConfig.Limit)
	}
	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(serverConfig.EffectiveConfig.Settings, envOPESBridgeLimitV0)
	if setting.Value != "9" || setting.Source != "explicit" {
		t.Fatalf("limit setting=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(serverConfig.EffectiveConfig, "deprecated_env_used", envOPESBridgeLimitV0, "opes_bridge.*") {
		t.Fatalf("diagnostico deprecated OPES bridge ausente: %+v", serverConfig.EffectiveConfig.Diagnostics)
	}
}
