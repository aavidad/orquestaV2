package orquestaruntimecodex

import (
	"os"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type CodexDeliveryObservationV0 struct {
	DeliveryRef  string   `json:"delivery_ref"`
	PhaseID      string   `json:"phase_id"`
	TaskID       string   `json:"task_id"`
	AgentRef     string   `json:"agent_ref"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

func ReadCodexDeliveryObservationFileV0(
	path string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexDeliveryObservationV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	data, err := os.ReadFile(path)
	if err != nil {
		issue := codexIssueV0(CodexConnectorAckInvalidV0, CodexAgentAckFileNameV0, spec.CorrelationID, "read_failed")
		if os.IsNotExist(err) {
			issue.Retryable = true
			issue.Evidence = []string{"ack_not_ready"}
		}
		return CodexDeliveryObservationV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			issue,
		}
	}
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) > 0 {
		return CodexDeliveryObservationV0{}, issues
	}
	return BuildCodexDeliveryObservationV0(ack, spec)
}

func BuildCodexDeliveryObservationV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexDeliveryObservationV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	if issues := ValidateCodexAgentAckForSpecV0(ack, spec); len(issues) > 0 {
		return CodexDeliveryObservationV0{}, issues
	}
	if strings.TrimSpace(ack.Status) != codexAgentAckStatusCompletedV0 {
		return CodexDeliveryObservationV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorAckInvalidV0, "status", spec.CorrelationID, "status_not_completed"),
		}
	}
	observation := codexDeliveryObservationFromAckV0(ack, spec.AgentPacket)
	if codexDeliveryObservationUnsafeForCoreV0(observation) {
		return CodexDeliveryObservationV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(
				CodexConnectorAckForbiddenV0,
				"delivery_observation",
				spec.CorrelationID,
				"core_forbidden_detail",
			),
		}
	}
	return observation, nil
}

func codexDeliveryObservationFromAckV0(
	ack CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) CodexDeliveryObservationV0 {
	return CodexDeliveryObservationV0{
		DeliveryRef: strings.TrimSpace(ack.AckRef),
		PhaseID:     strings.TrimSpace(packet.Phase),
		TaskID:      strings.TrimSpace(ack.TaskRef),
		AgentRef:    strings.TrimSpace(ack.RequestID),
		Summary:     "Entrega compacta validada por recibo de agente.",
		EvidenceRefs: compactCodexDeliveryObservationRefsV0([]string{
			ack.AckRef,
			packet.DeliveryRefs.MailboxRef,
			packet.DeliveryRefs.ReadinessRef,
			packet.DeliveryRefs.CheckpointRef,
		}),
	}
}

func compactCodexDeliveryObservationRefsV0(values []string) []string {
	seen := map[string]bool{}
	refs := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		refs = append(refs, value)
	}
	return refs
}

func codexDeliveryObservationUnsafeForCoreV0(observation CodexDeliveryObservationV0) bool {
	values := []string{
		observation.DeliveryRef,
		observation.PhaseID,
		observation.TaskID,
		observation.AgentRef,
		observation.Summary,
	}
	values = append(values, observation.EvidenceRefs...)
	for _, value := range values {
		if codexDeliveryObservationValueUnsafeForCoreV0(value) {
			return true
		}
	}
	return false
}

func codexDeliveryObservationValueUnsafeForCoreV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, fragment := range []string{
		"db",
		"database",
		"sql",
		"dsn",
		"runtime",
		"provider",
		"proveedor",
		"model",
		"modelo",
		"home",
		"oauth",
		"codex",
		"claude",
		"ollama",
		"vllm",
		"adapter",
		"adaptador",
		"filesystem",
		"git",
		"docker",
		"tmux",
		"secret",
		"secreto",
		"token",
		"password",
		"credential",
		"credencial",
		"api_key",
	} {
		if codexDeliveryObservationContainsFragmentV0(lower, fragment) {
			return true
		}
	}
	return false
}

func codexDeliveryObservationContainsFragmentV0(lowerValue string, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	if fragment == "" {
		return false
	}
	start := 0
	for {
		index := strings.Index(lowerValue[start:], fragment)
		if index < 0 {
			return false
		}
		absolute := start + index
		if codexDeliveryObservationHasTokenBoundaryV0(lowerValue, absolute, absolute+len(fragment)) {
			return true
		}
		start = absolute + len(fragment)
	}
}

func codexDeliveryObservationHasTokenBoundaryV0(value string, start int, end int) bool {
	before := start == 0 || !codexDeliveryObservationIsAsciiLetterOrDigitV0(value[start-1])
	after := end >= len(value) || !codexDeliveryObservationIsAsciiLetterOrDigitV0(value[end])
	return before && after
}

func codexDeliveryObservationIsAsciiLetterOrDigitV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}
