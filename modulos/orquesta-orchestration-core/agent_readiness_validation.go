package orquestacionnucleoapp

import (
	"regexp"
	"strings"
)

func validateAgentReadinessProbeRequestV0(
	request AgentReadinessProbeRequestV0,
) error {
	fields := []agentReadinessFieldV0{
		{field: "agent_readiness.run_id", value: request.RunID},
		{field: "agent_readiness.agent_request_id", value: request.AgentRequestID},
		{field: "agent_readiness.process_ref", value: request.ProcessRef},
		{field: "agent_readiness.session_ref", value: request.SessionRef},
		{field: "agent_readiness.launch_ref", value: request.LaunchRef},
		{field: "agent_readiness.readiness_ref", value: request.ReadinessRef},
	}
	if err := validateAgentReadinessFieldsV0(fields); err != nil {
		return err
	}
	return validateAgentReadinessEvidenceRefsV0(request.EvidenceRefs)
}

func validateAgentReadinessProbeResultV0(
	request AgentReadinessProbeRequestV0,
	result AgentReadinessProbeResultV0,
) error {
	if result.Status != AgentReadinessReadyV0 {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"readiness_probe.status",
			"agent_readiness_no_confirmada",
		)
	}
	if strings.TrimSpace(result.ReadinessRef) != request.ReadinessRef {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"readiness_probe.readiness_ref",
			"readiness_ref no coincide",
		)
	}
	return validateAgentReadinessEvidenceRefsV0(result.EvidenceRefs)
}

func validateAgentReadinessFieldsV0(
	fields []agentReadinessFieldV0,
) error {
	for _, item := range fields {
		trimmed := strings.TrimSpace(item.value)
		if trimmed == "" {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, item.field, "ref requerida")
		}
		if agentReadinessUnsafeTextV0(trimmed) {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, item.field, "detalle prohibido")
		}
		if !agentReadinessOpaqueRefPatternV0.MatchString(trimmed) {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, item.field, "ref no opaca")
		}
	}
	return nil
}

func validateAgentReadinessEvidenceRefsV0(
	refs []string,
) error {
	for _, ref := range refs {
		trimmed := strings.TrimSpace(ref)
		if trimmed == "" {
			continue
		}
		if agentReadinessUnsafeTextV0(trimmed) {
			return errorV0(
				ErrNucleoOrquestacionInvalidoV0,
				"agent_readiness.evidence_refs",
				"detalle prohibido",
			)
		}
		if !agentReadinessOpaqueRefPatternV0.MatchString(trimmed) {
			return errorV0(
				ErrNucleoOrquestacionInvalidoV0,
				"agent_readiness.evidence_refs",
				"ref no opaca",
			)
		}
	}
	return nil
}

func agentReadinessUnsafeTextV0(value string) bool {
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

type agentReadinessFieldV0 struct {
	field string
	value string
}

var agentReadinessOpaqueRefPatternV0 = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,511}$`)
