package orquestadirector

import (
	"regexp"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarails "orquesta/modulos/orquesta-rails"
)

func normalizeLaunchContextBundlePayloadV0(
	payload orquestacoreworkflow.LaunchRuntimeAgentRequestV0,
) orquestacoreworkflow.LaunchRuntimeAgentRequestV0 {
	return orquestacoreworkflow.LaunchRuntimeAgentRequestV0{
		AgentRequestID:     strings.TrimSpace(payload.AgentRequestID),
		RunID:              strings.TrimSpace(payload.RunID),
		PhaseID:            strings.TrimSpace(payload.PhaseID),
		TaskRef:            strings.TrimSpace(payload.TaskRef),
		CapacityRequestRef: strings.TrimSpace(payload.CapacityRequestRef),
		Role:               strings.TrimSpace(payload.Role),
		Summary:            strings.TrimSpace(payload.Summary),
		EvidenceRefs:       compactLaunchContextStringsV0(payload.EvidenceRefs),
	}
}

func validateLaunchContextBundlePayloadV0(
	payload orquestacoreworkflow.LaunchRuntimeAgentRequestV0,
) error {
	requiredOpaque := map[string]string{
		"payload.agent_request_id":     payload.AgentRequestID,
		"payload.run_id":               payload.RunID,
		"payload.phase_id":             payload.PhaseID,
		"payload.task_ref":             payload.TaskRef,
		"payload.capacity_request_ref": payload.CapacityRequestRef,
	}
	for field, value := range requiredOpaque {
		if !launchContextOpaqueRefV0(value) || launchContextUnsafeValueV0(field, value) {
			return launchContextBundleErrorV0(ErrDirectorContextBundleOutboxV0, field, nil)
		}
	}
	if !launchContextCompactTokenV0(payload.Role) ||
		launchContextUnsafeValueV0("payload.role", payload.Role) {
		return launchContextBundleErrorV0(ErrDirectorContextBundleOutboxV0, "payload.role", nil)
	}
	if payload.Summary == "" || len(payload.Summary) > 1000 ||
		launchContextUnsafeValueV0("payload.summary", payload.Summary) {
		return launchContextBundleErrorV0(ErrDirectorContextBundleOutboxV0, "payload.summary", nil)
	}
	for _, ref := range payload.EvidenceRefs {
		if !launchContextOpaqueRefV0(ref) || launchContextUnsafeValueV0("payload.evidence_refs", ref) {
			return launchContextBundleErrorV0(ErrDirectorContextBundleOutboxV0, "payload.evidence_refs", nil)
		}
	}
	return nil
}

func launchContextOpaqueRefV0(value string) bool {
	return launchContextOpaqueRefPatternV0.MatchString(strings.TrimSpace(value))
}

func launchContextCompactTokenV0(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 2 || len(value) > 80 {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			continue
		}
		if b == '.' || b == '_' || b == ':' || b == '-' {
			continue
		}
		return false
	}
	return true
}

func launchContextUnsafeValueV0(field string, value string) bool {
	return launchContextLooksLikeSecretV0(value) ||
		orquestarails.TextContainsOperationalRawDetailForFieldV0("launch_context_bundle", field, value)
}

func launchContextLooksLikeSecretV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(low, "bearer") ||
		strings.HasPrefix(low, "sk-") ||
		strings.HasPrefix(low, "ghp_") ||
		strings.HasPrefix(low, "xoxb-") ||
		strings.Contains(low, "access_token") ||
		strings.Contains(low, "refresh_token") ||
		strings.Contains(low, "api_key") ||
		strings.Contains(low, "secret=") ||
		strings.Count(value, ".") == 2 && strings.HasPrefix(value, "eyJ")
}

var launchContextOpaqueRefPatternV0 = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,511}$`)
