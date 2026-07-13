package orquestacoreleases

type AgentTimeoutDecisionV0 string

const (
	AgentTimeoutDecisionContinueV0    AgentTimeoutDecisionV0 = "continue"
	AgentTimeoutDecisionRetryV0       AgentTimeoutDecisionV0 = "retry"
	AgentTimeoutDecisionAskDirectorV0 AgentTimeoutDecisionV0 = "ask_director"
	AgentTimeoutDecisionStopAgentV0   AgentTimeoutDecisionV0 = "stop_agent"
	AgentTimeoutDecisionMarkFailedV0  AgentTimeoutDecisionV0 = "mark_failed"
	AgentTimeoutDecisionMarkStoppedV0 AgentTimeoutDecisionV0 = "mark_stopped"
	AgentTimeoutDecisionReplanTaskV0  AgentTimeoutDecisionV0 = "replan_task"
	AgentTimeoutDecisionAlertOnlyV0   AgentTimeoutDecisionV0 = "alert_only"
)

const (
	AgentTimeoutReasonHeartbeatCurrentV0 = "heartbeat_current"
	AgentTimeoutReasonHeartbeatTimeoutV0 = "heartbeat_timeout"
	AgentTimeoutReasonLaunchPendingV0    = "launch_pending"
	AgentTimeoutReasonLaunchTimeoutV0    = "launch_timeout"
	AgentTimeoutReasonHeartbeatStoppedV0 = "heartbeat_reported_stopped"
	AgentTimeoutReasonHeartbeatFailedV0  = "heartbeat_reported_failed"
	AgentTimeoutReasonTotalTimeoutV0     = "total_timeout"
)

type AgentLeaseEvaluationInputV0 struct {
	AssessmentRef    string                  `json:"assessment_ref"`
	RunRef           string                  `json:"run_ref"`
	AgentRequestID   string                  `json:"agent_request_id"`
	LeaseRef         string                  `json:"lease_ref"`
	LaunchObservedAt string                  `json:"launch_observed_at"`
	NowObservedAt    string                  `json:"now_observed_at"`
	Policy           AgentLeasePolicyV0      `json:"policy"`
	LastHeartbeat    *AgentHeartbeatReportV0 `json:"last_heartbeat,omitempty"`
	EvidenceRefs     []string                `json:"evidence_refs,omitempty"`
}

type AgentTimeoutAssessmentV0 struct {
	AssessmentRef  string                 `json:"assessment_ref"`
	RunRef         string                 `json:"run_ref"`
	AgentRequestID string                 `json:"agent_request_id"`
	LeaseRef       string                 `json:"lease_ref"`
	NowObservedAt  string                 `json:"now_observed_at"`
	Decision       AgentTimeoutDecisionV0 `json:"decision"`
	ReasonCode     string                 `json:"reason_code"`
	EvidenceRefs   []string               `json:"evidence_refs,omitempty"`
}

func EvaluateAgentLeaseV0(input AgentLeaseEvaluationInputV0) (AgentTimeoutAssessmentV0, error) {
	if issues := input.Validate(); len(issues) > 0 {
		return AgentTimeoutAssessmentV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}

	now, err := parseAgentLeaseInstantV0(input.NowObservedAt)
	if err != nil {
		return AgentTimeoutAssessmentV0{}, AgentLeaseValidationErrorV0{
			Issues: []AgentLeaseIssueV0{{Code: ErrAgentLeaseObservedAtV0, Field: "now_observed_at"}},
		}
	}
	launchObservedAt, err := parseAgentLeaseInstantV0(input.LaunchObservedAt)
	if err != nil {
		return AgentTimeoutAssessmentV0{}, AgentLeaseValidationErrorV0{
			Issues: []AgentLeaseIssueV0{{Code: ErrAgentLeaseObservedAtV0, Field: "launch_observed_at"}},
		}
	}

	decision := AgentTimeoutDecisionContinueV0
	reason := AgentTimeoutReasonLaunchPendingV0

	if input.LastHeartbeat != nil {
		heartbeat := *input.LastHeartbeat
		switch heartbeat.Status {
		case AgentHeartbeatStoppedV0:
			decision = AgentTimeoutDecisionMarkStoppedV0
			reason = AgentTimeoutReasonHeartbeatStoppedV0
		case AgentHeartbeatFailedV0:
			decision = AgentTimeoutDecisionMarkFailedV0
			reason = AgentTimeoutReasonHeartbeatFailedV0
		default:
			heartbeatObservedAt, err := parseAgentLeaseInstantV0(heartbeat.ObservedAt)
			if err != nil {
				return AgentTimeoutAssessmentV0{}, AgentLeaseValidationErrorV0{
					Issues: []AgentLeaseIssueV0{{Code: ErrAgentLeaseObservedAtV0, Field: "last_heartbeat.observed_at"}},
				}
			}
			if !agentLeaseDeadlineReachedV0(now, heartbeatObservedAt, input.Policy.HeartbeatTimeoutSeconds) {
				decision = AgentTimeoutDecisionContinueV0
				reason = AgentTimeoutReasonHeartbeatCurrentV0
			} else if agentLeaseDeadlineReachedV0(now, launchObservedAt, input.Policy.TotalTimeoutSeconds) {
				decision = agentTimeoutDecisionFromPolicyActionV0(input.Policy.TimeoutAction)
				reason = AgentTimeoutReasonTotalTimeoutV0
			} else {
				decision = agentTimeoutDecisionFromPolicyActionV0(input.Policy.TimeoutAction)
				reason = AgentTimeoutReasonHeartbeatTimeoutV0
			}
		}
	} else if agentLeaseDeadlineReachedV0(now, launchObservedAt, input.Policy.TotalTimeoutSeconds) {
		decision = agentTimeoutDecisionFromPolicyActionV0(input.Policy.TimeoutAction)
		reason = AgentTimeoutReasonTotalTimeoutV0
	} else if agentLeaseDeadlineReachedV0(now, launchObservedAt, input.Policy.LaunchTimeoutSeconds) {
		decision = agentTimeoutDecisionFromPolicyActionV0(input.Policy.TimeoutAction)
		reason = AgentTimeoutReasonLaunchTimeoutV0
	}

	assessment := AgentTimeoutAssessmentV0{
		AssessmentRef:  input.AssessmentRef,
		RunRef:         input.RunRef,
		AgentRequestID: input.AgentRequestID,
		LeaseRef:       input.LeaseRef,
		NowObservedAt:  input.NowObservedAt,
		Decision:       decision,
		ReasonCode:     reason,
		EvidenceRefs:   evidenceRefsForAgentTimeoutAssessmentV0(input),
	}
	if issues := ValidateAgentTimeoutAssessmentV0(assessment); len(issues) > 0 {
		return AgentTimeoutAssessmentV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}
	return assessment, nil
}

// Validate preserves the public input validation boundary for lease evaluation.
func (input AgentLeaseEvaluationInputV0) Validate() []AgentLeaseIssueV0 {
	return ValidateAgentLeaseEvaluationInputV0(input)
}

func DecodeAgentTimeoutAssessmentV0(data []byte) (AgentTimeoutAssessmentV0, error) {
	if issues := detectForbiddenAgentLeaseJSONDetailsV0(data); len(issues) > 0 {
		return AgentTimeoutAssessmentV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}
	var assessment AgentTimeoutAssessmentV0
	if err := decodeStrictAgentLeaseJSONV0(data, &assessment); err != nil {
		return AgentTimeoutAssessmentV0{}, err
	}
	if issues := ValidateAgentTimeoutAssessmentV0(assessment); len(issues) > 0 {
		return AgentTimeoutAssessmentV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}
	return assessment, nil
}

func (assessment AgentTimeoutAssessmentV0) Validate() []AgentLeaseIssueV0 {
	return ValidateAgentTimeoutAssessmentV0(assessment)
}

func (assessment AgentTimeoutAssessmentV0) Valid() bool {
	return len(ValidateAgentTimeoutAssessmentV0(assessment)) == 0
}

func agentTimeoutDecisionFromPolicyActionV0(action AgentLeaseTimeoutActionV0) AgentTimeoutDecisionV0 {
	switch action {
	case AgentLeaseTimeoutRetryV0:
		return AgentTimeoutDecisionRetryV0
	case AgentLeaseTimeoutStopAgentV0:
		return AgentTimeoutDecisionStopAgentV0
	case AgentLeaseTimeoutAskDirectorV0:
		return AgentTimeoutDecisionAskDirectorV0
	case AgentLeaseTimeoutReplanTaskV0:
		return AgentTimeoutDecisionReplanTaskV0
	case AgentLeaseTimeoutAlertOnlyV0:
		return AgentTimeoutDecisionAlertOnlyV0
	default:
		return AgentTimeoutDecisionAskDirectorV0
	}
}

func evidenceRefsForAgentTimeoutAssessmentV0(input AgentLeaseEvaluationInputV0) []string {
	refs := make([]string, 0, len(input.EvidenceRefs)+len(input.Policy.EvidenceRefs)+4)
	refs = appendUniqueAgentLeaseRefsV0(refs, input.EvidenceRefs...)
	refs = appendUniqueAgentLeaseRefsV0(refs, input.Policy.EvidenceRefs...)
	if input.LastHeartbeat != nil {
		refs = appendUniqueAgentLeaseRefsV0(refs, input.LastHeartbeat.HeartbeatRef)
		refs = appendUniqueAgentLeaseRefsV0(refs, input.LastHeartbeat.ProgressReportRef)
		refs = appendUniqueAgentLeaseRefsV0(refs, input.LastHeartbeat.EvidenceRefs...)
	}
	return refs
}

func appendUniqueAgentLeaseRefsV0(refs []string, candidates ...string) []string {
	for _, candidate := range candidates {
		if candidate == "" || containsAgentLeaseRefV0(refs, candidate) {
			continue
		}
		refs = append(refs, candidate)
	}
	return refs
}

func containsAgentLeaseRefV0(refs []string, candidate string) bool {
	for _, ref := range refs {
		if ref == candidate {
			return true
		}
	}
	return false
}
