package orquestacorereplanner

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func NewAgentReworkSignalV0(signal AgentReworkSignalV0) (AgentReworkSignalV0, error) {
	normalized := NormalizeAgentReworkSignalV0(signal)
	if err := ValidateAgentReworkSignalV0(normalized); err != nil {
		return AgentReworkSignalV0{}, err
	}
	return normalized, nil
}

func NormalizeAgentReworkSignalV0(signal AgentReworkSignalV0) AgentReworkSignalV0 {
	return AgentReworkSignalV0{
		SignalRef:        strings.TrimSpace(signal.SignalRef),
		RunRef:           strings.TrimSpace(signal.RunRef),
		TaskRef:          strings.TrimSpace(signal.TaskRef),
		AgentRequestID:   strings.TrimSpace(signal.AgentRequestID),
		SourceRef:        strings.TrimSpace(signal.SourceRef),
		AssessmentStatus: strings.TrimSpace(signal.AssessmentStatus),
		AssessmentAction: strings.TrimSpace(signal.AssessmentAction),
		RequestedAction:  ReplanRecommendedActionV0(strings.TrimSpace(string(signal.RequestedAction))),
		ReplacementRole:  strings.TrimSpace(signal.ReplacementRole),
		ReasonRef:        strings.TrimSpace(signal.ReasonRef),
		Summary:          strings.TrimSpace(signal.Summary),
		EvidenceRefs:     normalizeReplanProposalStringsV0(signal.EvidenceRefs),
	}
}

func AgentWorkAssessmentToReplanProposalV0(input AgentWorkAssessmentReplanInputV0) (*ReplanProposalV0, error) {
	assessment := normalizeAgentWorkAssessedForReplanV0(input.Assessment)
	if assessment.Verdict == orquestacoreworkflow.AgentAssessmentVerdictAcceptableV0 &&
		assessment.Action == orquestacoreworkflow.AgentAssessmentActionContinueV0 {
		return nil, nil
	}

	signal, err := AgentReworkSignalFromAgentWorkAssessedV0(input, assessment)
	if err != nil {
		return nil, err
	}

	proposal, err := ReplanProposalFromAgentReworkSignalV0(input.ReplanRef, signal)
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

func AgentReworkSignalFromAgentWorkAssessedV0(input AgentWorkAssessmentReplanInputV0, assessment orquestacoreworkflow.AgentWorkAssessedPayloadV0) (AgentReworkSignalV0, error) {
	assessment = normalizeAgentWorkAssessedForReplanV0(assessment)

	taskRef := strings.TrimSpace(assessment.TaskRef)
	if taskRef == "" {
		taskRef = strings.TrimSpace(input.TaskRef)
	}
	summary := strings.TrimSpace(input.Summary)
	if summary == "" {
		summary = assessment.Summary
	}

	signal := AgentReworkSignalV0{
		SignalRef:        input.SignalRef,
		RunRef:           input.RunRef,
		TaskRef:          taskRef,
		AgentRequestID:   assessment.AgentRequestID,
		SourceRef:        assessment.AssessmentRef,
		AssessmentStatus: assessment.Verdict,
		AssessmentAction: assessment.Action,
		RequestedAction:  input.RequestedAction,
		ReplacementRole:  input.ReplacementRole,
		ReasonRef:        input.ReasonRef,
		Summary:          summary,
		EvidenceRefs:     agentReworkEvidenceRefsV0(assessment.EvidenceRefs, input.EvidenceRefs),
	}
	return NewAgentReworkSignalV0(signal)
}

func ReplanProposalFromAgentReworkSignalV0(replanRef string, signal AgentReworkSignalV0) (ReplanProposalV0, error) {
	normalized, err := NewAgentReworkSignalV0(signal)
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
		ReasonCode:        agentReworkReasonCodeV0(normalized),
		RecommendedAction: normalized.RequestedAction,
		ReplacementRole:   replacementRole,
		Summary:           normalized.Summary,
		EvidenceRefs:      agentReworkEvidenceRefsV0([]string{normalized.SignalRef}, normalized.EvidenceRefs),
	}
	return NewReplanProposalV0(proposal)
}

func normalizeAgentWorkAssessedForReplanV0(payload orquestacoreworkflow.AgentWorkAssessedPayloadV0) orquestacoreworkflow.AgentWorkAssessedPayloadV0 {
	return orquestacoreworkflow.AgentWorkAssessedPayloadV0{
		AssessmentRef:  strings.TrimSpace(payload.AssessmentRef),
		PhaseID:        strings.TrimSpace(payload.PhaseID),
		AgentRequestID: strings.TrimSpace(payload.AgentRequestID),
		TaskRef:        strings.TrimSpace(payload.TaskRef),
		DeliveryRef:    strings.TrimSpace(payload.DeliveryRef),
		Verdict:        strings.TrimSpace(payload.Verdict),
		Action:         strings.TrimSpace(payload.Action),
		Severity:       strings.TrimSpace(payload.Severity),
		Summary:        strings.TrimSpace(payload.Summary),
		EvidenceRefs:   normalizeReplanProposalStringsV0(payload.EvidenceRefs),
	}
}

func agentReworkReasonCodeV0(signal AgentReworkSignalV0) string {
	if strings.TrimSpace(signal.ReasonRef) != "" {
		return signal.ReasonRef
	}
	return "agent_" + string(signal.AssessmentStatus)
}

func agentReworkEvidenceRefsV0(groups ...[]string) []string {
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
