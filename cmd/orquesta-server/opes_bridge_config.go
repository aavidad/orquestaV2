package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
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
}

func opesDrainConfigFromEnvV0() (opesDrainConfigV0, error) {
	return opesDrainConfigFromEnvWithBaseURLV0("")
}

func opesDrainConfigFromEnvWithBaseURLV0(
	orquestaBaseURLFallback string,
) (opesDrainConfigV0, error) {
	opesBaseURL := firstNonEmptyEnvV0(envOPESBaseURLV0, envOPESBaseURLLegacyV0)
	if opesBaseURL == "" {
		return opesDrainConfigV0{}, fmt.Errorf("ORQUESTA_OPES_BASE_URL u OPES_BASE_URL requerido")
	}
	dryRun := strings.TrimSpace(os.Getenv(envOPESBridgeDryRunV0)) == "1"
	orquestaBaseURL, err := orquestaBaseURLFromEnvOrStateOrFallbackV0(
		dryRun,
		orquestaBaseURLFallback,
	)
	if err != nil {
		return opesDrainConfigV0{}, err
	}
	inputLedger, err := opesBridgeInputLedgerFromEnvV0(dryRun)
	if err != nil {
		return opesDrainConfigV0{}, err
	}
	destination, err := opesDrainDestinationPolicyFromEnvV0(opesBaseURL, orquestaBaseURL, dryRun)
	if err != nil {
		return opesDrainConfigV0{}, err
	}
	jobType := strings.TrimSpace(os.Getenv(envOPESBridgeJobTypeV0))
	jobTypeSequence := opesBridgeJobTypeSequenceFromEnvV0()
	jobRef := strings.TrimSpace(os.Getenv(envOPESBridgeJobRefV0))
	programID := strings.TrimSpace(os.Getenv(envOPESBridgeProgramIDV0))
	topicID := strings.TrimSpace(os.Getenv(envOPESBridgeTopicIDV0))
	correlationID := strings.TrimSpace(os.Getenv(envOPESBridgeCorrelationIDV0))
	if len(jobTypeSequence) > 0 && jobType != "" {
		return opesDrainConfigV0{}, fmt.Errorf("ORQUESTA_OPES_BRIDGE_JOB_TYPE incompatible con ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE")
	}
	if len(jobTypeSequence) > 0 && jobRef != "" {
		return opesDrainConfigV0{}, fmt.Errorf("ORQUESTA_OPES_BRIDGE_JOB_REF incompatible con ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE")
	}
	return opesDrainConfigV0{
		OPESBaseURL:        strings.TrimRight(opesBaseURL, "/"),
		OrquestaBaseURL:    strings.TrimRight(orquestaBaseURL, "/"),
		Limit:              intEnvOrDefaultV0(envOPESBridgeLimitV0, 3),
		JobType:            jobType,
		JobTypeSequence:    jobTypeSequence,
		JobRef:             jobRef,
		ProgramID:          programID,
		TopicID:            topicID,
		CorrelationID:      correlationID,
		DryRun:             dryRun,
		SuperviseSubmitted: strings.TrimSpace(os.Getenv(envOPESBridgeSuperviseSubmittedV0)) == "1",
		ResidentDispatchWait: time.Duration(
			intEnvOrDefaultV0(envOPESBridgeWaitResidentSecondsV0, 0),
		) * time.Second,
		ResidentDispatchPollInterval: time.Duration(
			intEnvOrDefaultV0(envOPESBridgeWaitResidentIntervalMSV0, int(defaultOPESBridgeResidentDispatchIntervalV0/time.Millisecond)),
		) * time.Millisecond,
		HTTPTimeout: time.Duration(intEnvOrDefaultV0(envOPESBridgeTimeoutSecondsV0, 30)) * time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: intEnvOrDefaultV0(envOPESBridgePriorityV0, 70),
			RequestedBy:   "orquesta-opes-bridge",
		},
		InputLedger:          inputLedger,
		Destination:          destination,
		ExternalCapabilities: opesBridgeExternalCapabilitiesFromEnvV0(),
	}, nil
}

func orquestaBaseURLFromEnvOrStateOrFallbackV0(
	dryRun bool,
	fallback string,
) (string, error) {
	if value := strings.TrimSpace(os.Getenv(envOrquestaBaseURLV0)); value != "" {
		return value, nil
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

func firstNonEmptyEnvV0(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func opesBridgeJobTypeSequenceFromEnvV0() []string {
	return parseOPESBridgeJobTypeSequenceV0(os.Getenv(envOPESBridgeJobTypeSequenceV0))
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
	out := []orquestadomainwork.DomainWorkExternalCapabilityV0{}
	if capability, ok := opesBridgeExternalCapabilityFromEnvV0(
		orquestadomainwork.DomainWorkExternalCapabilityKindSpeechSynthesisV0,
		envOPESBridgeSpeechSynthesisCapabilityV0,
		envOPESBridgeSpeechSynthesisCapabilityRefV0,
		envOPESBridgeSpeechSynthesisEvidenceRefsV0,
		envOPESBridgeSpeechSynthesisReasonV0,
	); ok {
		capability.NetworkReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromEnvV0(
			envOPESBridgeSpeechSynthesisNetworkReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"network_ready",
		)
		capability.ToolPathReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromEnvV0(
			envOPESBridgeSpeechSynthesisToolPathReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"tool_path_ready",
		)
		capability.ProviderQuotaReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromEnvV0(
			envOPESBridgeSpeechSynthesisQuotaReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"provider_quota_ready",
		)
		capability.ProgressHeartbeatReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromEnvV0(
			envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0,
			false,
			capability.OperationalReason,
			capability.Kind,
			"progress_heartbeat_ready",
		)
		capability.ProviderTimeoutReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromEnvV0(
			envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0,
			false,
			capability.OperationalReason,
			capability.Kind,
			"provider_timeout_ready",
		)
		if capability.ProviderTimeoutReady {
			capability.ProviderNoProgressTimeoutSeconds = intEnvOrDefaultV0(
				envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0,
				300,
			)
		}
		capability.CommandTimeoutSeconds = 1800
		capability = opesBridgeSpeechSynthesisToolPreflightFromEnvV0(capability)
		out = append(out, orquestadomainwork.NormalizeDomainWorkExternalCapabilityV0(capability))
	}
	if capability, ok := opesBridgeExternalCapabilityFromEnvV0(
		orquestadomainwork.DomainWorkExternalCapabilityKindRemoteQAProviderV0,
		envOPESBridgeRemoteQACapabilityV0,
		envOPESBridgeRemoteQACapabilityRefV0,
		envOPESBridgeRemoteQAEvidenceRefsV0,
		envOPESBridgeRemoteQAReasonV0,
	); ok {
		capability.NetworkReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromEnvV0(
			envOPESBridgeRemoteQANetworkReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"network_ready",
		)
		capability.AuthStateReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromEnvV0(
			envOPESBridgeRemoteQAAuthStateReadyV0,
			capability.Available,
			capability.OperationalReason,
			capability.Kind,
			"auth_state_ready",
		)
		capability.ProviderQuotaReady, capability.OperationalReason = opesBridgeCapabilityReadinessFromEnvV0(
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
	workdir := strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisToolWorkDirV0))
	commandRaw := strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisToolCommandV0))
	if workdir == "" && commandRaw == "" {
		return capability
	}
	if !capability.Available {
		return capability
	}
	if !capability.ToolPathReady &&
		strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisToolPathReadyV0)) != "" {
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
		intEnvOrDefaultV0(
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
	raw := strings.TrimSpace(os.Getenv(capabilityEnv))
	if raw == "" {
		return orquestadomainwork.DomainWorkExternalCapabilityV0{}, false
	}
	available, reason := opesBridgeCapabilityAvailabilityFromEnvV0(raw, kind)
	reason = firstNonEmptyV0(strings.TrimSpace(os.Getenv(reasonEnv)), reason)
	capabilityRef := strings.TrimSpace(os.Getenv(refEnv))
	if capabilityRef == "" {
		capabilityRef = kind
	}
	return orquestadomainwork.DomainWorkExternalCapabilityV0{
		CapabilityRef:     capabilityRef,
		Kind:              kind,
		Available:         available,
		OperationalReason: reason,
		EvidenceRefs:      parseOPESBridgeJobTypeSequenceV0(os.Getenv(evidenceEnv)),
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
	raw := strings.TrimSpace(os.Getenv(envName))
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
