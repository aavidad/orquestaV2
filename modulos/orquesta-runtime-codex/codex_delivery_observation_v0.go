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
	ack, issues := validateCodexDeliveryAckBytesForSpecV0(data, spec)
	if len(issues) > 0 {
		return CodexDeliveryObservationV0{}, issues
	}
	return BuildCodexDeliveryObservationV0(ack, spec)
}

func validateCodexDeliveryAckBytesForSpecV0(
	data []byte,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexAgentAckV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	if CodexAgentPacketRequiresStrictTerminalAckV0(spec.AgentPacket) {
		ack, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0(data, spec)
		if len(issues) == 0 {
			return ack, nil
		}
		if codexDeliveryObservationCanAcceptLegacyAckWithoutReceiptsV0(data, spec, issues) {
			return ValidateCodexAgentAckBytesForSpecV0(data, spec)
		}
		return ack, issues
	}
	return ValidateCodexAgentAckBytesForSpecV0(data, spec)
}

func codexDeliveryObservationCanAcceptLegacyAckWithoutReceiptsV0(
	data []byte,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	strictIssues []orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	if !codexDeliveryObservationIssuesOnlyMissingTestReceiptV0(strictIssues) {
		return false
	}
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) > 0 || len(ack.TestReceipts) > 0 {
		return false
	}
	return len(compactCodexAckStringsV0(spec.AgentPacket.Task.RequiredTests)) > 0
}

func codexDeliveryObservationIssuesOnlyMissingTestReceiptV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		if issue.Code != orquestaruntime.ExternalAgentConnectorErrorCodeV0(CodexConnectorAckArtifactV0) ||
			strings.TrimSpace(issue.Field) != "test_receipts" ||
			!codexAckIssueHasEvidenceV0(issue, "missing_required_test_receipt") {
			return false
		}
	}
	return true
}

func codexAckIssueHasEvidenceV0(
	issue orquestaruntime.ExternalAgentConnectorErrorV0,
	want string,
) bool {
	for _, evidence := range issue.Evidence {
		if evidence == want {
			return true
		}
	}
	return false
}

func BuildCodexDeliveryObservationV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexDeliveryObservationV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	ack = codexAgentAckWithSpecDefaultsV0(ack, spec)
	if issues := ValidateCodexAgentAckForSpecV0(ack, spec); len(issues) > 0 {
		return CodexDeliveryObservationV0{}, issues
	}
	if strings.TrimSpace(ack.Status) != codexAgentAckStatusCompletedV0 {
		return CodexDeliveryObservationV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorAckInvalidV0, "status", spec.CorrelationID, "status_not_completed"),
		}
	}
	return codexDeliveryObservationFromAckV0(ack, spec.AgentPacket), nil
}

func codexDeliveryObservationFromAckV0(
	ack CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) CodexDeliveryObservationV0 {
	evidenceRefs := []string{
		ack.AckRef,
		packet.DeliveryRefs.MailboxRef,
		packet.DeliveryRefs.ReadinessRef,
		packet.DeliveryRefs.CheckpointRef,
	}
	evidenceRefs = append(evidenceRefs, CodexRequiredTestReceiptEvidenceRefsV0(ack.TestReceipts)...)
	evidenceRefs = append(evidenceRefs, CodexAgentAckPendingRailEvidenceRefsV0(ack)...)
	return CodexDeliveryObservationV0{
		DeliveryRef:  strings.TrimSpace(ack.AckRef),
		PhaseID:      strings.TrimSpace(packet.Phase),
		TaskID:       strings.TrimSpace(ack.TaskRef),
		AgentRef:     strings.TrimSpace(ack.RequestID),
		Summary:      "Entrega compacta validada por recibo de agente.",
		EvidenceRefs: compactCodexDeliveryObservationRefsV0(evidenceRefs),
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
	return codexDeliveryObservationValuesContainV0(
		observation,
		codexDeliveryObservationValueUnsafeForCoreV0,
	)
}

func codexDeliveryObservationHasPendingRailV0(observation CodexDeliveryObservationV0) bool {
	return codexDeliveryObservationValuesContainV0(
		observation,
		codexDeliveryObservationValueHasPendingRailV0,
	)
}

func codexDeliveryObservationValuesContainV0(
	observation CodexDeliveryObservationV0,
	match func(string) bool,
) bool {
	values := []string{
		observation.DeliveryRef,
		observation.PhaseID,
		observation.TaskID,
		observation.AgentRef,
		observation.Summary,
	}
	values = append(values, observation.EvidenceRefs...)
	for _, value := range values {
		if match(value) {
			return true
		}
	}
	return false
}

func codexDeliveryObservationValueUnsafeForCoreV0(value string) bool {
	return codexAckTextContainsEffectiveSensitiveDetailV0(value)
}

func codexDeliveryObservationValueHasPendingRailV0(value string) bool {
	return !codexDeliveryObservationValueUnsafeForCoreV0(value) &&
		(codexTextContainsOperationalDetailMarkerV0(value) ||
			codexDeliveryObservationValueHasLegacyPendingMarkerV0(value))
}

func codexDeliveryObservationValueHasLegacyPendingMarkerV0(value string) bool {
	return codexTextHasLegacyPendingMarkerV0(value)
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
