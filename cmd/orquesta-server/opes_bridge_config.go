package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
)

const defaultOPESBridgeResidentDispatchIntervalV0 = 500 * time.Millisecond

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
	raw := strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisCapabilityV0))
	if raw == "" {
		return []orquestadomainwork.DomainWorkExternalCapabilityV0{}
	}
	available, reason := opesBridgeCapabilityAvailabilityFromEnvV0(raw)
	reason = firstNonEmptyV0(strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisReasonV0)), reason)
	capabilityRef := strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisCapabilityRefV0))
	if capabilityRef == "" {
		capabilityRef = orquestadomainwork.DomainWorkExternalCapabilityKindSpeechSynthesisV0
	}
	return []orquestadomainwork.DomainWorkExternalCapabilityV0{
		orquestadomainwork.NormalizeDomainWorkExternalCapabilityV0(
			orquestadomainwork.DomainWorkExternalCapabilityV0{
				CapabilityRef:     capabilityRef,
				Kind:              orquestadomainwork.DomainWorkExternalCapabilityKindSpeechSynthesisV0,
				Available:         available,
				OperationalReason: reason,
				EvidenceRefs:      parseOPESBridgeJobTypeSequenceV0(os.Getenv(envOPESBridgeSpeechSynthesisEvidenceRefsV0)),
			},
		),
	}
}

func opesBridgeCapabilityAvailabilityFromEnvV0(raw string) (bool, string) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "si", "available", "enabled", "ready", "ok":
		return true, ""
	case "0", "false", "no", "n", "unavailable", "disabled", "missing", "blocked":
		return false, orquestadomainwork.DomainWorkExternalCapabilityReasonMissingV0 +
			":" + orquestadomainwork.DomainWorkExternalCapabilityKindSpeechSynthesisV0
	default:
		return false, "speech_synthesis_capability_invalid"
	}
}
