package orquestacoreleases

// AgentLeaseExpiredV0 is the compact candidate event that workflow may record
// after an external timeout assessment. It carries only opaque refs and an
// observed instant supplied by an adapter/dispatcher.
type AgentLeaseExpiredV0 struct {
	RunRef            string                 `json:"run_ref"`
	AgentRequestID    string                 `json:"agent_request_id"`
	LeaseRef          string                 `json:"lease_ref"`
	ReasonCode        string                 `json:"reason_code"`
	ObservedAt        string                 `json:"observed_at"`
	RecommendedAction AgentTimeoutDecisionV0 `json:"recommended_action"`
	EvidenceRefs      []string               `json:"evidence_refs,omitempty"`
}

// AgentLeaseExpiredFromAssessmentV0 translates non-continue assessments into a
// candidate event. It does not execute the recommended action or inspect runtime
// state; durable commands in another boundary must materialize that later.
func AgentLeaseExpiredFromAssessmentV0(assessment AgentTimeoutAssessmentV0) (AgentLeaseExpiredV0, bool, error) {
	if issues := ValidateAgentTimeoutAssessmentV0(assessment); len(issues) > 0 {
		return AgentLeaseExpiredV0{}, false, AgentLeaseValidationErrorV0{Issues: issues}
	}
	if assessment.Decision == AgentTimeoutDecisionContinueV0 {
		return AgentLeaseExpiredV0{}, false, nil
	}

	expired := AgentLeaseExpiredV0{
		RunRef:            assessment.RunRef,
		AgentRequestID:    assessment.AgentRequestID,
		LeaseRef:          assessment.LeaseRef,
		ReasonCode:        assessment.ReasonCode,
		ObservedAt:        assessment.NowObservedAt,
		RecommendedAction: assessment.Decision,
		EvidenceRefs:      append([]string(nil), assessment.EvidenceRefs...),
	}
	if issues := ValidateAgentLeaseExpiredV0(expired); len(issues) > 0 {
		return AgentLeaseExpiredV0{}, false, AgentLeaseValidationErrorV0{Issues: issues}
	}
	return expired, true, nil
}

func DecodeAgentLeaseExpiredV0(data []byte) (AgentLeaseExpiredV0, error) {
	if issues := detectForbiddenAgentLeaseJSONDetailsV0(data); len(issues) > 0 {
		return AgentLeaseExpiredV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}
	var expired AgentLeaseExpiredV0
	if err := decodeStrictAgentLeaseJSONV0(data, &expired); err != nil {
		return AgentLeaseExpiredV0{}, err
	}
	if issues := ValidateAgentLeaseExpiredV0(expired); len(issues) > 0 {
		return AgentLeaseExpiredV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}
	return expired, nil
}

func (expired AgentLeaseExpiredV0) Validate() []AgentLeaseIssueV0 {
	return ValidateAgentLeaseExpiredV0(expired)
}

func (expired AgentLeaseExpiredV0) Valid() bool {
	return len(ValidateAgentLeaseExpiredV0(expired)) == 0
}
