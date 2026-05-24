package orquestaagentprocessregistry

import (
	"regexp"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func NormalizeAgentProcessRegistryRecordV0(
	record AgentProcessRegistryRecordV0,
) AgentProcessRegistryRecordV0 {
	record.RunID = normalizeRefV0(record.RunID)
	record.AgentRequestID = normalizeRefV0(record.AgentRequestID)
	record.ProcessRef = normalizeRefV0(record.ProcessRef)
	record.SessionRef = normalizeRefV0(record.SessionRef)
	record.LaunchRef = normalizeRefV0(record.LaunchRef)
	record.ReadinessRef = normalizeRefV0(record.ReadinessRef)
	record.EvidenceRefs = compactRefsV0(record.EvidenceRefs)
	return record
}

func NormalizeAgentProcessRegistryLookupV0(
	runID string,
	agentRequestID string,
) (string, string) {
	return normalizeRefV0(runID), normalizeRefV0(agentRequestID)
}

func ValidateAgentProcessRegistryRecordV0(
	record AgentProcessRegistryRecordV0,
) error {
	if err := validateRefsV0([]registryFieldV0{
		{field: "agent_process.run_id", value: record.RunID},
		{field: "agent_process.agent_request_id", value: record.AgentRequestID},
		{field: "agent_process.process_ref", value: record.ProcessRef},
		{field: "agent_process.session_ref", value: record.SessionRef},
		{field: "agent_process.launch_ref", value: record.LaunchRef},
		{field: "agent_process.readiness_ref", value: record.ReadinessRef},
	}); err != nil {
		return err
	}
	return validateEvidenceRefsV0(record.EvidenceRefs)
}

func ValidateAgentProcessRegistryLookupV0(
	runID string,
	agentRequestID string,
) error {
	return validateRefsV0([]registryFieldV0{
		{field: "agent_process.run_id", value: runID},
		{field: "agent_process.agent_request_id", value: agentRequestID},
	})
}

func validateRefsV0(fields []registryFieldV0) error {
	for _, item := range fields {
		trimmed := normalizeRefV0(item.value)
		if trimmed == "" {
			return registryErrorV0(item.field, "ref requerida")
		}
		if unsafeRefV0(trimmed) {
			return registryErrorV0(item.field, "detalle prohibido")
		}
		if !opaqueRefPatternV0.MatchString(trimmed) {
			return registryErrorV0(item.field, "ref no opaca")
		}
	}
	return nil
}

func validateEvidenceRefsV0(refs []string) error {
	for _, ref := range refs {
		trimmed := normalizeRefV0(ref)
		if trimmed == "" {
			continue
		}
		if unsafeRefV0(trimmed) {
			return registryErrorV0("agent_process.evidence_refs", "detalle prohibido")
		}
		if !opaqueRefPatternV0.MatchString(trimmed) {
			return registryErrorV0("agent_process.evidence_refs", "ref no opaca")
		}
	}
	return nil
}

func normalizeRefV0(value string) string {
	return strings.TrimSpace(value)
}

func compactRefsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := normalizeRefV0(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func unsafeRefV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "://") ||
		strings.Contains(low, "/home/") ||
		strings.Contains(low, "\\users\\") ||
		strings.Contains(low, "oauth") ||
		strings.Contains(low, "token") ||
		strings.Contains(low, "secret") ||
		strings.Contains(low, "home=") ||
		strings.Contains(low, "provider=") ||
		strings.Contains(low, "proveedor=")
}

func registryErrorV0(field string, message string) ErrorV0 {
	return ErrorV0{
		Code:    ErrAgentProcessRegistryInvalidV0,
		Field:   field,
		Message: message,
	}
}

type registryFieldV0 struct {
	field string
	value string
}

var opaqueRefPatternV0 = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,511}$`)
