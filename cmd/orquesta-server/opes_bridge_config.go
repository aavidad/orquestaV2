package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	defaultOPESBridgeResidentDispatchIntervalV0            = 500 * time.Millisecond
	defaultOPESBridgeSpeechSynthesisToolPreflightTimeoutV0 = 5 * time.Second
	opesBridgeSpeechSynthesisToolPreflightEvidenceRefV0    = "evidence-ref-opes-audio-tool-preflight-ok"
	opesBridgeSpeechSynthesisToolMissingReasonV0           = "opes_audio_tool_missing"
	opesBridgeSpeechSynthesisToolPreflightTimeoutReasonV0  = "opes_audio_tool_preflight_timeout"
	opesBridgeSpeechSynthesisDefaultToolScriptRelV0        = "scripts/opes_audio_app.py"
	opesBridgeSpeechSynthesisDefaultToolPreflightCommandV0 = "python3 scripts/opes_audio_app.py --help"
)

type opesDrainConfigV0 struct {
	OPESBaseURL                  string
	OrquestaBaseURL              string
	Limit                        int
	JobType                      string
	JobTypeSequence              []string
	JobRef                       string
	ProgramID                    string
	TopicID                      string
	CorrelationID                string
	DryRun                       bool
	SuperviseSubmitted           bool
	ResidentDispatchWait         time.Duration
	ResidentDispatchPollInterval time.Duration
	HTTPTimeout                  time.Duration
	RunConfig                    orquestaopesbridge.JobRunConfigV0
	InputLedger                  externalBridgeInputLedgerV0
	Destination                  opesDrainDestinationPolicyV0
	ExternalCapabilities         []orquestadomainwork.DomainWorkExternalCapabilityV0
	RuntimeCompatibility         opesBridgeRuntimeCompatibilityPolicyV0
}

type opesBridgeRuntimeCompatibilityPolicyV0 struct {
	Required             bool
	RequiredBinarySHA256 string
	RequiredBuildRef     string
	RequiredCommitRef    string
}

func opesDrainConfigFromEnvV0() (opesDrainConfigV0, error) {
	return opesDrainConfigFromEnvWithBaseURLV0("")
}

func opesDrainConfigFromEnvWithBaseURLV0(
	orquestaBaseURLFallback string,
) (opesDrainConfigV0, error) {
	projectConfig := opesProjectConfigFromEnvBestEffortV0()
	opesBaseURL := opesBaseURLFromProjectConfigFileV0(projectConfig)
	if opesBaseURL == "" {
		return opesDrainConfigV0{}, fmt.Errorf("ORQUESTA_OPES_BASE_URL u OPES_BASE_URL requerido")
	}
	dryRun := opesBridgeDryRunFromProjectConfigFileV0(projectConfig)
	orquestaBaseURL, err := orquestaBaseURLFromEnvOrStateOrFallbackV0(
		dryRun,
		orquestaBaseURLFallback,
	)
	if err != nil {
		return opesDrainConfigV0{}, err
	}
	inputLedger, err := opesBridgeInputLedgerFromProjectConfigFileV0(projectConfig, dryRun)
	if err != nil {
		return opesDrainConfigV0{}, err
	}
	destination, err := opesDrainDestinationPolicyFromProjectConfigFileV0(projectConfig, opesBaseURL, orquestaBaseURL, dryRun)
	if err != nil {
		return opesDrainConfigV0{}, err
	}
	jobType := opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeJobTypeV0)
	jobTypeSequence := opesBridgeJobTypeSequenceFromProjectConfigFileV0(projectConfig)
	jobRef := opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeJobRefV0)
	programID := opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeProgramIDV0)
	topicID := opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeTopicIDV0)
	correlationID := opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeCorrelationIDV0)
	if len(jobTypeSequence) > 0 && jobType != "" {
		return opesDrainConfigV0{}, fmt.Errorf("ORQUESTA_OPES_BRIDGE_JOB_TYPE incompatible con ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE")
	}
	if len(jobTypeSequence) > 0 && jobRef != "" {
		return opesDrainConfigV0{}, fmt.Errorf("ORQUESTA_OPES_BRIDGE_JOB_REF incompatible con ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE")
	}
	return opesDrainConfigV0{
		OPESBaseURL:        strings.TrimRight(opesBaseURL, "/"),
		OrquestaBaseURL:    strings.TrimRight(orquestaBaseURL, "/"),
		Limit:              opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeLimitV0, 3),
		JobType:            jobType,
		JobTypeSequence:    jobTypeSequence,
		JobRef:             jobRef,
		ProgramID:          programID,
		TopicID:            topicID,
		CorrelationID:      correlationID,
		DryRun:             dryRun,
		SuperviseSubmitted: opesBridgeBoolValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSuperviseSubmittedV0, false),
		ResidentDispatchWait: time.Duration(
			opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeWaitResidentSecondsV0, 0),
		) * time.Second,
		ResidentDispatchPollInterval: time.Duration(
			opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeWaitResidentIntervalMSV0, int(defaultOPESBridgeResidentDispatchIntervalV0/time.Millisecond)),
		) * time.Millisecond,
		HTTPTimeout: time.Duration(opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeTimeoutSecondsV0, 30)) * time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgePriorityV0, 70),
			RequestedBy:   "orquesta-opes-bridge",
		},
		InputLedger:          inputLedger,
		Destination:          destination,
		ExternalCapabilities: opesBridgeExternalCapabilitiesFromProjectConfigFileV0(projectConfig),
		RuntimeCompatibility: opesBridgeRuntimeCompatibilityPolicyFromProjectConfigFileV0(projectConfig),
	}, nil
}

func opesBridgeRuntimeCompatibilityPolicyFromEnvV0() opesBridgeRuntimeCompatibilityPolicyV0 {
	return opesBridgeRuntimeCompatibilityPolicyFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func opesBridgeRuntimeCompatibilityPolicyFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
) opesBridgeRuntimeCompatibilityPolicyV0 {
	policy := opesBridgeRuntimeCompatibilityPolicyV0{
		Required:             opesBridgeBoolValueFromProjectConfigFileV0(config, envOPESBridgeRequireRuntimeCompatibilityV0, false),
		RequiredBinarySHA256: opesBridgeStringValueFromProjectConfigFileV0(config, envOPESBridgeRequiredRuntimeBinarySHA256V0),
		RequiredBuildRef:     opesBridgeStringValueFromProjectConfigFileV0(config, envOPESBridgeRequiredRuntimeBuildRefV0),
		RequiredCommitRef:    opesBridgeStringValueFromProjectConfigFileV0(config, envOPESBridgeRequiredRuntimeCommitRefV0),
	}
	if policy.RequiredBinarySHA256 != "" || policy.RequiredBuildRef != "" || policy.RequiredCommitRef != "" {
		policy.Required = true
	}
	return policy
}

func orquestaBaseURLFromEnvOrStateOrFallbackV0(
	dryRun bool,
	fallback string,
) (string, error) {
	if value := strings.TrimSpace(os.Getenv(envOrquestaServerURLV0)); value != "" {
		return value, nil
	}
	if value := strings.TrimSpace(os.Getenv(envOrquestaBaseURLV0)); value != "" {
		return value, nil
	}
	if value, ok, err := orquestaBaseURLFromRuntimeDirEnvV0(); ok || err != nil {
		return value, err
	}
	if value := strings.TrimSpace(fallback); value != "" {
		return value, nil
	}
	if dryRun {
		return "dry-run", nil
	}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		return "", err
	}
	state, err := loadStateV0(config)
	if err != nil {
		return "", fmt.Errorf("ORQUESTA_BASE_URL requerido si no hay estado de servidor")
	}
	if strings.TrimSpace(state.Addr) == "" {
		return "", fmt.Errorf("estado de servidor sin addr")
	}
	return "http://" + strings.TrimSpace(state.Addr), nil
}

func orquestaBaseURLFromRuntimeDirEnvV0() (string, bool, error) {
	runtimeDir := strings.TrimSpace(os.Getenv(envOrquestaRuntimeDirV0))
	if runtimeDir == "" {
		return "", false, nil
	}
	path := filepath.Join(runtimeDir, "base_url.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", true, fmt.Errorf("ORQUESTA_RUNTIME_DIR/base_url.txt no legible")
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

func firstNonEmptyEnvV0(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func opesProjectConfigFromEnvBestEffortV0() serverProjectConfigFileV0 {
	return projectConfigFromProjectDirBestEffortV0(os.Getenv(envCodexProjectWorkDirV0))
}

func opesBaseURLFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	if value := strings.TrimSpace(os.Getenv(envOPESBaseURLV0)); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv(envOPESBaseURLLegacyV0)); value != "" {
		return value
	}
	return stringProjectConfigOrEnvOrDefaultV0(envOPESBaseURLV0, config.OPES.BaseURL, "")
}

func opesBridgeEnabledFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return opesBridgeBoolValueFromProjectConfigFileV0(config, envOPESBridgeEnabledV0, false)
}

func opesBridgeConfirmFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return opesBridgeBoolValueFromProjectConfigFileV0(config, envOPESBridgeConfirmV0, false)
}

func opesBridgeDryRunFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return opesBridgeBoolValueFromProjectConfigFileV0(config, envOPESBridgeDryRunV0, false)
}

func opesBridgeBoolValueFromProjectConfigFileV0(config serverProjectConfigFileV0, key string, fallback bool) bool {
	switch key {
	case envOPESTemporalConfirmV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.OPES.TemporalConfirm, fallback)
	case envOPESBridgeEnabledV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.Enabled, fallback)
	case envOPESBridgeConfirmV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.Confirm, fallback)
	case envOPESBridgeDryRunV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.DryRun, fallback)
	case envOPESBridgeSuperviseSubmittedV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SuperviseSubmitted, fallback)
	case envOPESBridgeAllowUnfilteredV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.AllowUnfiltered, fallback)
	case envOPESBridgeInputLedgerDisabledV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.InputLedgerDisabled, fallback)
	case envOPESBridgeProductiveConfirmV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.ProductiveConfirm, fallback)
	case envOPESBridgeRequireRuntimeCompatibilityV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RequireRuntimeCompatibility, fallback)
	default:
		return boolEnvOrDefaultV0(key, fallback)
	}
}

func opesBridgeIntValueFromProjectConfigFileV0(config serverProjectConfigFileV0, key string, fallback int) int {
	switch key {
	case envOPESBridgeLimitV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.Limit, fallback)
	case envOPESBridgeTimeoutSecondsV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.TimeoutSeconds, fallback)
	case envOPESBridgePriorityV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.Priority, fallback)
	case envOPESBridgeIntervalSecondsV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.IntervalSeconds, fallback)
	case envOPESBridgeInitialDelaySecondsV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.InitialDelaySeconds, fallback)
	case envOPESBridgeMaxTicksV0:
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return intEnvOrDefaultV0(key, fallback)
		}
		if config.OPESBridge.MaxTicks != nil && *config.OPESBridge.MaxTicks >= 0 {
			return *config.OPESBridge.MaxTicks
		}
		return fallback
	case envOPESBridgeWaitResidentSecondsV0:
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return intEnvOrDefaultV0(key, fallback)
		}
		if config.OPESBridge.WaitResidentSeconds != nil && *config.OPESBridge.WaitResidentSeconds >= 0 {
			return *config.OPESBridge.WaitResidentSeconds
		}
		return fallback
	case envOPESBridgeWaitResidentIntervalMSV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.WaitResidentIntervalMS, fallback)
	case envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesisNoProgress, fallback)
	case envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesisToolPreflight, fallback)
	default:
		return intEnvOrDefaultV0(key, fallback)
	}
}

func opesBridgeMaxTicksFromProjectConfigFileV0(config serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrZeroV0(envOPESBridgeMaxTicksV0, config.OPESBridge.MaxTicks)
}

func opesBridgeDurationSecondsFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	key string,
	fallback time.Duration,
) time.Duration {
	switch key {
	case envOPESBridgeIntervalSecondsV0:
		return durationSecondsProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.IntervalSeconds, fallback)
	case envOPESBridgeInitialDelaySecondsV0:
		return durationSecondsProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.InitialDelaySeconds, fallback)
	default:
		return durationSecondsEnvOrDefaultV0(key, fallback)
	}
}

func opesBridgeStringValueFromProjectConfigFileV0(config serverProjectConfigFileV0, key string) string {
	switch key {
	case envOPESProjectWorkDirV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPES.ProjectWorkDir, "")
	case envOPESBridgeJobTypeV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.JobType, "")
	case envOPESBridgeJobRefV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.JobRef, "")
	case envOPESBridgeProgramIDV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.ProgramID, "")
	case envOPESBridgeTopicIDV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.TopicID, "")
	case envOPESBridgeCorrelationIDV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.CorrelationID, "")
	case envOPESBridgeInputLedgerPathV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.InputLedgerPath, "")
	case envOPESBridgeDestinationEvidenceV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.DestinationEvidenceRef, "")
	case envOPESBridgeRequiredRuntimeBinarySHA256V0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RequiredRuntimeBinarySHA256, "")
	case envOPESBridgeRequiredRuntimeBuildRefV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RequiredRuntimeBuildRef, "")
	case envOPESBridgeRequiredRuntimeCommitRefV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RequiredRuntimeCommitRef, "")
	case envOPESBridgeSpeechSynthesisCapabilityV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesis.Capability, "")
	case envOPESBridgeSpeechSynthesisCapabilityRefV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesis.CapabilityRef, "")
	case envOPESBridgeSpeechSynthesisEvidenceRefsV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesis.EvidenceRefs, "")
	case envOPESBridgeSpeechSynthesisReasonV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesis.Reason, "")
	case envOPESBridgeSpeechSynthesisNetworkReadyV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesis.NetworkReady, "")
	case envOPESBridgeSpeechSynthesisToolPathReadyV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesis.ToolPathReady, "")
	case envOPESBridgeSpeechSynthesisQuotaReadyV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesis.ProviderQuotaReady, "")
	case envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesisProgressReady, "")
	case envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesisTimeoutReady, "")
	case envOPESBridgeSpeechSynthesisToolWorkDirV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesisToolWorkDir, "")
	case envOPESBridgeSpeechSynthesisToolCommandV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.SpeechSynthesisToolCommand, "")
	case envOPESBridgeRemoteQACapabilityV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RemoteQA.Capability, "")
	case envOPESBridgeRemoteQACapabilityRefV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RemoteQA.CapabilityRef, "")
	case envOPESBridgeRemoteQAEvidenceRefsV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RemoteQA.EvidenceRefs, "")
	case envOPESBridgeRemoteQAReasonV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RemoteQA.Reason, "")
	case envOPESBridgeRemoteQANetworkReadyV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RemoteQA.NetworkReady, "")
	case envOPESBridgeRemoteQAAuthStateReadyV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RemoteQAAuthStateReady, "")
	case envOPESBridgeRemoteQAQuotaReadyV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.OPESBridge.RemoteQA.ProviderQuotaReady, "")
	default:
		return strings.TrimSpace(os.Getenv(key))
	}
}

func opesBridgeJobTypeSequenceFromEnvV0() []string {
	return opesBridgeJobTypeSequenceFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func opesBridgeJobTypeSequenceFromProjectConfigFileV0(config serverProjectConfigFileV0) []string {
	if value := strings.TrimSpace(os.Getenv(envOPESBridgeJobTypeSequenceV0)); value != "" {
		return parseOPESBridgeJobTypeSequenceV0(value)
	}
	if config.OPESBridge.JobTypeSequence == nil {
		return []string{}
	}
	return parseOPESBridgeJobTypeSequenceV0(strings.Join(*config.OPESBridge.JobTypeSequence, " "))
}

func opesBridgeEffectiveConfigSettingsV0(
	config orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) []orquestaserver.ServerConfigSettingV0 {
	return []orquestaserver.ServerConfigSettingV0{
		opesBridgeBaseURLSettingV0(config, projectConfig),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeJobTypeV0,
			opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeJobTypeV0),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeJobTypeV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeJobRefV0,
			opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeJobRefV0),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeJobRefV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeProgramIDV0,
			opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeProgramIDV0),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeProgramIDV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeTopicIDV0,
			opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeTopicIDV0),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeTopicIDV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeCorrelationIDV0,
			opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeCorrelationIDV0),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeCorrelationIDV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeJobTypeSequenceV0,
			strings.Join(opesBridgeJobTypeSequenceFromProjectConfigFileV0(projectConfig), ","),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeJobTypeSequenceV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeLimitV0,
			strconv.Itoa(opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeLimitV0, 3)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeLimitV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeTimeoutSecondsV0,
			strconv.Itoa(opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeTimeoutSecondsV0, 30)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeTimeoutSecondsV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgePriorityV0,
			strconv.Itoa(opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgePriorityV0, 70)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgePriorityV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeIntervalSecondsV0,
			strconv.Itoa(int(opesBridgeDurationSecondsFromProjectConfigFileV0(projectConfig, envOPESBridgeIntervalSecondsV0, 60*time.Second)/time.Second)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeIntervalSecondsV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeInitialDelaySecondsV0,
			strconv.Itoa(int(opesBridgeDurationSecondsFromProjectConfigFileV0(projectConfig, envOPESBridgeInitialDelaySecondsV0, 2*time.Second)/time.Second)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeInitialDelaySecondsV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeMaxTicksV0,
			strconv.Itoa(opesBridgeMaxTicksFromProjectConfigFileV0(projectConfig)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeMaxTicksV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeSuperviseSubmittedV0,
			strconv.FormatBool(opesBridgeBoolValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSuperviseSubmittedV0, false)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSuperviseSubmittedV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeWaitResidentSecondsV0,
			strconv.Itoa(opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeWaitResidentSecondsV0, 0)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeWaitResidentSecondsV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeWaitResidentIntervalMSV0,
			strconv.Itoa(opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeWaitResidentIntervalMSV0, int(defaultOPESBridgeResidentDispatchIntervalV0/time.Millisecond))),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeWaitResidentIntervalMSV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeRequireRuntimeCompatibilityV0,
			strconv.FormatBool(opesBridgeRuntimeCompatibilityPolicyFromProjectConfigFileV0(projectConfig).Required),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRequireRuntimeCompatibilityV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeRequiredRuntimeBinarySHA256V0,
			opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRequiredRuntimeBinarySHA256V0),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRequiredRuntimeBinarySHA256V0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeRequiredRuntimeBuildRefV0,
			opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRequiredRuntimeBuildRefV0),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRequiredRuntimeBuildRefV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESBridgeRequiredRuntimeCommitRefV0,
			opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRequiredRuntimeCommitRefV0),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRequiredRuntimeCommitRefV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeAllowUnfilteredV0, strconv.FormatBool(opesBridgeBoolValueFromProjectConfigFileV0(projectConfig, envOPESBridgeAllowUnfilteredV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeAllowUnfilteredV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeInputLedgerDisabledV0, strconv.FormatBool(opesBridgeBoolValueFromProjectConfigFileV0(projectConfig, envOPESBridgeInputLedgerDisabledV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeInputLedgerDisabledV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envOPESBridgeInputLedgerPathV0, configuredRefValueV0(opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeInputLedgerPathV0), "opes-bridge-input-ledger-path-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeInputLedgerPathV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeProductiveConfirmV0, strconv.FormatBool(opesBridgeBoolValueFromProjectConfigFileV0(projectConfig, envOPESBridgeProductiveConfirmV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeProductiveConfirmV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeDestinationEvidenceV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeDestinationEvidenceV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeDestinationEvidenceV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisCapabilityV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisCapabilityV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisCapabilityV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisCapabilityRefV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisCapabilityRefV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisCapabilityRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisEvidenceRefsV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisEvidenceRefsV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisEvidenceRefsV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisReasonV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisReasonV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisReasonV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisNetworkReadyV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisNetworkReadyV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisNetworkReadyV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisToolPathReadyV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisToolPathReadyV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisToolPathReadyV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisQuotaReadyV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisQuotaReadyV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisQuotaReadyV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0, strconv.Itoa(opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0, 300)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisToolWorkDirV0, configuredRefValueV0(opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisToolWorkDirV0), "opes-speech-synthesis-tool-workdir-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisToolWorkDirV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisToolCommandV0, configuredRefValueV0(opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisToolCommandV0), "opes-speech-synthesis-tool-command-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisToolCommandV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0, strconv.Itoa(opesBridgeIntValueFromProjectConfigFileV0(projectConfig, envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0, int(defaultOPESBridgeSpeechSynthesisToolPreflightTimeoutV0/time.Second))), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeRemoteQACapabilityV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRemoteQACapabilityV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRemoteQACapabilityV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeRemoteQACapabilityRefV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRemoteQACapabilityRefV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRemoteQACapabilityRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeRemoteQAEvidenceRefsV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRemoteQAEvidenceRefsV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRemoteQAEvidenceRefsV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeRemoteQAReasonV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRemoteQAReasonV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRemoteQAReasonV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeRemoteQANetworkReadyV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRemoteQANetworkReadyV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRemoteQANetworkReadyV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeRemoteQAAuthStateReadyV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRemoteQAAuthStateReadyV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRemoteQAAuthStateReadyV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESBridgeRemoteQAQuotaReadyV0, opesBridgeStringValueFromProjectConfigFileV0(projectConfig, envOPESBridgeRemoteQAQuotaReadyV0), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBridgeRemoteQAQuotaReadyV0)),
	}
}

func opesBridgeBaseURLSettingV0(
	config orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) orquestaserver.ServerConfigSettingV0 {
	source := configSettingSourceFromConfigOrProjectConfigV0(config, envOPESBaseURLV0)
	if source == "defaulted" && strings.TrimSpace(os.Getenv(envOPESBaseURLV0)) == "" &&
		strings.TrimSpace(os.Getenv(envOPESBaseURLLegacyV0)) != "" {
		source = "legacy_alias"
	}
	return serverSensitiveConfigSettingFromRegistryWithSourceV0(
		envOPESBaseURLV0,
		sensitiveConfigValueFromSourceV0(
			opesBaseURLFromProjectConfigFileV0(projectConfig),
			"opes-base-url-configured",
			source,
		),
		source,
	)
}

func parseOPESBridgeJobTypeSequenceV0(raw string) []string {
	raw = strings.NewReplacer(",", " ", ";", " ", "\n", " ", "\t", " ").Replace(raw)
	parts := strings.Fields(raw)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func opesBridgeExternalCapabilitiesFromEnvV0() []orquestadomainwork.DomainWorkExternalCapabilityV0 {
	return opesBridgeExternalCapabilitiesFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func opesBridgeExternalCapabilitiesFromProjectConfigFileV0(config serverProjectConfigFileV0) []orquestadomainwork.DomainWorkExternalCapabilityV0 {
	out := []orquestadomainwork.DomainWorkExternalCapabilityV0{}
	if capability, ok := opesBridgeExternalCapabilityFromProjectConfigFileV0(
		config,
		orquestadomainwork.DomainWorkExternalCapabilityKindSpeechSynthesisV0,
		envOPESBridgeSpeechSynthesisCapabilityV0,
		envOPESBridgeSpeechSynthesisCapabilityRefV0,
		envOPESBridgeSpeechSynthesisEvidenceRefsV0,
		envOPESBridgeSpeechSynthesisReasonV0,
	); ok {
		capability.NetworkReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromProjectConfigFileV0(
			config,
			envOPESBridgeSpeechSynthesisNetworkReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"network_ready",
		)
		capability.ToolPathReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromProjectConfigFileV0(
			config,
			envOPESBridgeSpeechSynthesisToolPathReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"tool_path_ready",
		)
		capability.ProviderQuotaReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromProjectConfigFileV0(
			config,
			envOPESBridgeSpeechSynthesisQuotaReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"provider_quota_ready",
		)
		capability.ProgressHeartbeatReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromProjectConfigFileV0(
			config,
			envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0,
			false,
			capability.OperationalReason,
			capability.Kind,
			"progress_heartbeat_ready",
		)
		capability.ProviderTimeoutReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromProjectConfigFileV0(
			config,
			envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0,
			false,
			capability.OperationalReason,
			capability.Kind,
			"provider_timeout_ready",
		)
		if capability.ProviderTimeoutReady {
			capability.ProviderNoProgressTimeoutSeconds = opesBridgeIntValueFromProjectConfigFileV0(
				config,
				envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0,
				300,
			)
		}
		capability.CommandTimeoutSeconds = 1800
		capability = opesBridgeSpeechSynthesisToolPreflightFromProjectConfigFileV0(config, capability)
		out = append(out, orquestadomainwork.NormalizeDomainWorkExternalCapabilityV0(capability))
	}
	if capability, ok := opesBridgeExternalCapabilityFromProjectConfigFileV0(
		config,
		orquestadomainwork.DomainWorkExternalCapabilityKindRemoteQAProviderV0,
		envOPESBridgeRemoteQACapabilityV0,
		envOPESBridgeRemoteQACapabilityRefV0,
		envOPESBridgeRemoteQAEvidenceRefsV0,
		envOPESBridgeRemoteQAReasonV0,
	); ok {
		capability.NetworkReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromProjectConfigFileV0(
			config,
			envOPESBridgeRemoteQANetworkReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"network_ready",
		)
		capability.AuthStateReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromProjectConfigFileV0(
			config,
			envOPESBridgeRemoteQAAuthStateReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"auth_state_ready",
		)
		capability.ProviderQuotaReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromProjectConfigFileV0(
			config,
			envOPESBridgeRemoteQAQuotaReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"provider_quota_ready",
		)
		capability.CommandTimeoutSeconds = 1200
		out = append(out, orquestadomainwork.NormalizeDomainWorkExternalCapabilityV0(capability))
	}
	if out == nil {
		return []orquestadomainwork.DomainWorkExternalCapabilityV0{}
	}
	return out
}

func opesBridgeSpeechSynthesisToolPreflightFromEnvV0(
	capability orquestadomainwork.DomainWorkExternalCapabilityV0,
) orquestadomainwork.DomainWorkExternalCapabilityV0 {
	return opesBridgeSpeechSynthesisToolPreflightFromProjectConfigFileV0(serverProjectConfigFileV0{}, capability)
}

func opesBridgeSpeechSynthesisToolPreflightFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	capability orquestadomainwork.DomainWorkExternalCapabilityV0,
) orquestadomainwork.DomainWorkExternalCapabilityV0 {
	workdir := opesBridgeStringValueFromProjectConfigFileV0(config, envOPESBridgeSpeechSynthesisToolWorkDirV0)
	commandRaw := opesBridgeStringValueFromProjectConfigFileV0(config, envOPESBridgeSpeechSynthesisToolCommandV0)
	if workdir == "" && commandRaw == "" {
		return capability
	}
	if !capability.Available {
		return capability
	}
	if !capability.ToolPathReady &&
		opesBridgeStringValueFromProjectConfigFileV0(config, envOPESBridgeSpeechSynthesisToolPathReadyV0) != "" {
		return capability
	}
	args := opesBridgeSpeechSynthesisToolPreflightCommandV0(commandRaw)
	resolvedWorkdir, err := opesBridgeSpeechSynthesisToolPreflightWorkdirV0(workdir)
	if err != nil {
		return opesBridgeSpeechSynthesisToolPreflightFailedV0(capability, opesBridgeSpeechSynthesisToolMissingReasonV0)
	}
	if opesBridgeSpeechSynthesisToolPreflightChecksDefaultScriptV0(commandRaw, args) {
		scriptPath := filepath.Join(resolvedWorkdir, opesBridgeSpeechSynthesisDefaultToolScriptRelV0)
		if stat, err := os.Stat(scriptPath); err != nil || stat.IsDir() {
			return opesBridgeSpeechSynthesisToolPreflightFailedV0(capability, opesBridgeSpeechSynthesisToolMissingReasonV0)
		}
	}
	timeout := time.Duration(
		opesBridgeIntValueFromProjectConfigFileV0(
			config,
			envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0,
			int(defaultOPESBridgeSpeechSynthesisToolPreflightTimeoutV0/time.Second),
		),
	) * time.Second
	if timeout <= 0 {
		timeout = defaultOPESBridgeSpeechSynthesisToolPreflightTimeoutV0
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := opesBridgeRunSpeechSynthesisToolPreflightV0(ctx, resolvedWorkdir, args); err != nil {
		reason := opesBridgeSpeechSynthesisToolMissingReasonV0
		if ctx.Err() == context.DeadlineExceeded {
			reason = opesBridgeSpeechSynthesisToolPreflightTimeoutReasonV0
		}
		return opesBridgeSpeechSynthesisToolPreflightFailedV0(capability, reason)
	}
	capability.ToolPathReady = true
	capability.EvidenceRefs = append(capability.EvidenceRefs, opesBridgeSpeechSynthesisToolPreflightEvidenceRefV0)
	return capability
}

func opesBridgeSpeechSynthesisToolPreflightCommandV0(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = opesBridgeSpeechSynthesisDefaultToolPreflightCommandV0
	}
	return strings.Fields(raw)
}

func opesBridgeSpeechSynthesisToolPreflightWorkdirV0(raw string) (string, error) {
	workdir := strings.TrimSpace(raw)
	if workdir == "" {
		return "", nil
	}
	abs, err := filepath.Abs(workdir)
	if err != nil {
		return "", err
	}
	stat, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("opes speech synthesis workdir no es directorio")
	}
	return abs, nil
}

func opesBridgeSpeechSynthesisToolPreflightChecksDefaultScriptV0(commandRaw string, args []string) bool {
	if strings.TrimSpace(commandRaw) == "" {
		return true
	}
	for _, arg := range args {
		if filepath.Clean(strings.TrimSpace(arg)) == opesBridgeSpeechSynthesisDefaultToolScriptRelV0 {
			return true
		}
	}
	return false
}

func opesBridgeRunSpeechSynthesisToolPreflightV0(ctx context.Context, workdir string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("opes speech synthesis preflight command vacio")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if strings.TrimSpace(workdir) != "" {
		cmd.Dir = strings.TrimSpace(workdir)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("opes speech synthesis preflight fallo: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func opesBridgeSpeechSynthesisToolPreflightFailedV0(
	capability orquestadomainwork.DomainWorkExternalCapabilityV0,
	reason string,
) orquestadomainwork.DomainWorkExternalCapabilityV0 {
	capability.ToolPathReady = false
	capability.OperationalReason = strings.TrimSpace(reason)
	return capability
}

func opesBridgeExternalCapabilityFromEnvV0(
	kind string,
	capabilityEnv string,
	refEnv string,
	evidenceEnv string,
	reasonEnv string,
) (orquestadomainwork.DomainWorkExternalCapabilityV0, bool) {
	return opesBridgeExternalCapabilityFromProjectConfigFileV0(serverProjectConfigFileV0{}, kind, capabilityEnv, refEnv, evidenceEnv, reasonEnv)
}

func opesBridgeExternalCapabilityFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	kind string,
	capabilityEnv string,
	refEnv string,
	evidenceEnv string,
	reasonEnv string,
) (orquestadomainwork.DomainWorkExternalCapabilityV0, bool) {
	raw := opesBridgeStringValueFromProjectConfigFileV0(config, capabilityEnv)
	if raw == "" {
		return orquestadomainwork.DomainWorkExternalCapabilityV0{}, false
	}
	available, reason := opesBridgeCapabilityAvailabilityFromEnvV0(raw, kind)
	reason = firstNonEmptyV0(opesBridgeStringValueFromProjectConfigFileV0(config, reasonEnv), reason)
	capabilityRef := opesBridgeStringValueFromProjectConfigFileV0(config, refEnv)
	if capabilityRef == "" {
		capabilityRef = kind
	}
	return orquestadomainwork.DomainWorkExternalCapabilityV0{
		CapabilityRef:     capabilityRef,
		Kind:              kind,
		Available:         available,
		OperationalReason: reason,
		EvidenceRefs:      parseOPESBridgeJobTypeSequenceV0(opesBridgeStringValueFromProjectConfigFileV0(config, evidenceEnv)),
	}, true
}

func opesBridgeCapabilityAvailabilityFromEnvV0(raw string, kind string) (bool, string) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "si", "available", "enabled", "ready", "ok":
		return true, ""
	case "0", "false", "no", "n", "unavailable", "disabled", "missing", "blocked":
		return false, orquestadomainwork.DomainWorkExternalCapabilityReasonMissingV0 +
			":" + strings.TrimSpace(kind)
	default:
		return false, strings.TrimSpace(kind) + "_capability_invalid"
	}
}

func opesBridgeCapabilityReadinessFromEnvV0(
	envName string,
	defaultReady bool,
	currentReason string,
	kind string,
	check string,
) (bool, string) {
	return opesBridgeCapabilityReadinessFromProjectConfigFileV0(serverProjectConfigFileV0{}, envName, defaultReady, currentReason, kind, check)
}

func opesBridgeCapabilityReadinessFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	envName string,
	defaultReady bool,
	currentReason string,
	kind string,
	check string,
) (bool, string) {
	raw := opesBridgeStringValueFromProjectConfigFileV0(config, envName)
	if raw == "" {
		return defaultReady, strings.TrimSpace(currentReason)
	}
	ready, reason := opesBridgeCapabilityAvailabilityFromEnvV0(raw, kind)
	if ready {
		return true, strings.TrimSpace(currentReason)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || reason == orquestadomainwork.DomainWorkExternalCapabilityReasonMissingV0+":"+strings.TrimSpace(kind) {
		reason = orquestadomainwork.DomainWorkExternalCapabilityReasonMissingV0 + ":" +
			strings.TrimSpace(kind) + ":" + strings.TrimSpace(check)
	}
	if strings.TrimSpace(currentReason) != "" {
		return false, strings.TrimSpace(currentReason)
	}
	return false, reason
}
