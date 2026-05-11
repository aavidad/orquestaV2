package orquestacorereplanner

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func NewAgentFailedReplanSignalV0(signal AgentFailedReplanSignalV0) (AgentFailedReplanSignalV0, error) {
	normalized := NormalizeAgentFailedReplanSignalV0(signal)
	if err := ValidateAgentFailedReplanSignalV0(normalized); err != nil {
		return AgentFailedReplanSignalV0{}, err
	}
	return normalized, nil
}

func NormalizeAgentFailedReplanSignalV0(signal AgentFailedReplanSignalV0) AgentFailedReplanSignalV0 {
	return AgentFailedReplanSignalV0{
		SignalRef:         strings.TrimSpace(signal.SignalRef),
		RunRef:            strings.TrimSpace(signal.RunRef),
		TaskRef:           strings.TrimSpace(signal.TaskRef),
		AgentRequestID:    strings.TrimSpace(signal.AgentRequestID),
		SourceRef:         strings.TrimSpace(signal.SourceRef),
		FailureReasonCode: strings.TrimSpace(signal.FailureReasonCode),
		Retryable:         signal.Retryable,
		RequestedAction:   ReplanRecommendedActionV0(strings.TrimSpace(string(signal.RequestedAction))),
		ReplacementRole:   strings.TrimSpace(signal.ReplacementRole),
		ReasonRef:         strings.TrimSpace(signal.ReasonRef),
		Summary:           strings.TrimSpace(signal.Summary),
		EvidenceRefs:      normalizeReplanProposalStringsV0(signal.EvidenceRefs),
	}
}

func AgentFailedToReplanProposalV0(input AgentFailedReplanInputV0) (ReplanProposalV0, error) {
	failed := normalizeAgentFailedPayloadForReplanV0(input.AgentFailed)
	signal, err := AgentFailedReplanSignalFromAgentFailedV0(input, failed)
	if err != nil {
		return ReplanProposalV0{}, err
	}
	return ReplanProposalFromAgentFailedReplanSignalV0(input.ReplanRef, signal)
}

func AgentFailedReplanSignalFromAgentFailedV0(
	input AgentFailedReplanInputV0,
	failed orquestacoreworkflow.AgentFailedPayloadV0,
) (AgentFailedReplanSignalV0, error) {
	failed = normalizeAgentFailedPayloadForReplanV0(failed)
	signal := AgentFailedReplanSignalV0{
		SignalRef:         input.SignalRef,
		RunRef:            input.RunRef,
		TaskRef:           input.TaskRef,
		AgentRequestID:    failed.AgentRequestID,
		SourceRef:         failed.AgentRequestID,
		FailureReasonCode: failed.ReasonCode,
		Retryable:         failed.Retryable,
		RequestedAction:   input.RequestedAction,
		ReplacementRole:   input.ReplacementRole,
		ReasonRef:         input.ReasonRef,
		Summary:           input.Summary,
		EvidenceRefs: agentFailedReplanEvidenceRefsV0(
			[]string{failed.LaunchRef},
			failed.EvidenceRefs,
			input.EvidenceRefs,
		),
	}
	return NewAgentFailedReplanSignalV0(signal)
}

func ReplanProposalFromAgentFailedReplanSignalV0(replanRef string, signal AgentFailedReplanSignalV0) (ReplanProposalV0, error) {
	normalized, err := NewAgentFailedReplanSignalV0(signal)
	if err != nil {
		return ReplanProposalV0{}, err
	}
	replacementRole := ""
	if normalized.RequestedAction == ReplanActionReplaceAgentV0 {
		replacementRole = normalized.ReplacementRole
	}

	proposal := ReplanProposalV0{
		ReplanRef:         replanRef,
		RunRef:            normalized.RunRef,
		TaskRef:           normalized.TaskRef,
		SourceRef:         normalized.SourceRef,
		ReasonCode:        agentFailedReplanReasonCodeV0(normalized),
		RecommendedAction: normalized.RequestedAction,
		ReplacementRole:   replacementRole,
		Summary:           normalized.Summary,
		EvidenceRefs:      agentFailedReplanEvidenceRefsV0([]string{normalized.SignalRef}, normalized.EvidenceRefs),
	}
	return NewReplanProposalV0(proposal)
}

func normalizeAgentFailedPayloadForReplanV0(payload orquestacoreworkflow.AgentFailedPayloadV0) orquestacoreworkflow.AgentFailedPayloadV0 {
	return orquestacoreworkflow.AgentFailedPayloadV0{
		AgentRequestID: strings.TrimSpace(payload.AgentRequestID),
		LaunchRef:      strings.TrimSpace(payload.LaunchRef),
		ReasonCode:     strings.TrimSpace(payload.ReasonCode),
		Retryable:      payload.Retryable,
		EvidenceRefs:   normalizeReplanProposalStringsV0(payload.EvidenceRefs),
	}
}

func agentFailedReplanReasonCodeV0(signal AgentFailedReplanSignalV0) string {
	if strings.TrimSpace(signal.ReasonRef) != "" {
		return signal.ReasonRef
	}
	return "agent_failed_" + signal.FailureReasonCode
}

func agentFailedReplanEvidenceRefsV0(groups ...[]string) []string {
	seen := map[string]bool{}
	var refs []string
	for _, group := range groups {
		for _, raw := range group {
			ref := strings.TrimSpace(raw)
			if ref == "" || seen[ref] {
				continue
			}
			seen[ref] = true
			refs = append(refs, ref)
		}
	}
	return refs
}
