package orquestacoreleases

import "time"

const agentLeaseInstantLayoutV0 = "2006-01-02T15:04:05Z"

func ValidateAgentLeaseEvaluationInputV0(input AgentLeaseEvaluationInputV0) []AgentLeaseIssueV0 {
	v := agentLeaseValidatorV0{}
	v.requireOpaque("assessment_ref", input.AssessmentRef)
	v.requireOpaque("run_ref", input.RunRef)
	v.requireOpaque("agent_request_id", input.AgentRequestID)
	v.requireOpaque("lease_ref", input.LeaseRef)
	v.requireLeaseObservedAt("launch_observed_at", input.LaunchObservedAt)
	v.requireLeaseObservedAt("now_observed_at", input.NowObservedAt)
	v.validateEvidenceRefs("evidence_refs", input.EvidenceRefs)
	v.issues = append(v.issues, ValidateAgentLeasePolicyV0(input.Policy)...)
	if input.LastHeartbeat != nil {
		v.issues = append(v.issues, ValidateAgentHeartbeatReportV0(*input.LastHeartbeat)...)
		if input.LastHeartbeat.RunRef != input.RunRef {
			v.add(ErrAgentLeaseInvalidoV0, "last_heartbeat.run_ref")
		}
		if input.LastHeartbeat.AgentRequestID != input.AgentRequestID {
			v.add(ErrAgentLeaseInvalidoV0, "last_heartbeat.agent_request_id")
		}
		if input.LastHeartbeat.LeaseRef != input.LeaseRef {
			v.add(ErrAgentLeaseInvalidoV0, "last_heartbeat.lease_ref")
		}
	}
	v.validateEvaluationClockOrder(input)
	return v.issues
}

func ValidateAgentTimeoutAssessmentV0(assessment AgentTimeoutAssessmentV0) []AgentLeaseIssueV0 {
	v := agentLeaseValidatorV0{}
	v.requireOpaque("assessment_ref", assessment.AssessmentRef)
	v.requireOpaque("run_ref", assessment.RunRef)
	v.requireOpaque("agent_request_id", assessment.AgentRequestID)
	v.requireOpaque("lease_ref", assessment.LeaseRef)
	v.requireLeaseObservedAt("now_observed_at", assessment.NowObservedAt)
	v.requireAssessmentDecision("decision", assessment.Decision)
	v.requireOpaque("reason_code", assessment.ReasonCode)
	v.validateEvidenceRefs("evidence_refs", assessment.EvidenceRefs)
	return v.issues
}

func ValidateAgentLeaseExpiredV0(expired AgentLeaseExpiredV0) []AgentLeaseIssueV0 {
	v := agentLeaseValidatorV0{}
	v.requireOpaque("run_ref", expired.RunRef)
	v.requireOpaque("agent_request_id", expired.AgentRequestID)
	v.requireOpaque("lease_ref", expired.LeaseRef)
	v.requireOpaque("reason_code", expired.ReasonCode)
	v.requireLeaseObservedAt("observed_at", expired.ObservedAt)
	v.requireLeaseExpiredRecommendedAction("recommended_action", expired.RecommendedAction)
	v.validateEvidenceRefs("evidence_refs", expired.EvidenceRefs)
	return v.issues
}

func (v *agentLeaseValidatorV0) requireLeaseObservedAt(field, value string) {
	if !validAgentLeaseUTCInstantV0(value) {
		v.add(ErrAgentLeaseObservedAtV0, field)
	}
}

func (v *agentLeaseValidatorV0) requireAssessmentDecision(field string, decision AgentTimeoutDecisionV0) {
	if !isOneOfAgentLeaseV0(string(decision),
		string(AgentTimeoutDecisionContinueV0),
		string(AgentTimeoutDecisionRetryV0),
		string(AgentTimeoutDecisionAskDirectorV0),
		string(AgentTimeoutDecisionStopAgentV0),
		string(AgentTimeoutDecisionMarkFailedV0),
		string(AgentTimeoutDecisionMarkStoppedV0),
		string(AgentTimeoutDecisionReplanTaskV0),
		string(AgentTimeoutDecisionAlertOnlyV0),
	) {
		v.add(ErrAgentLeaseDecisionInvalidaV0, field)
	}
}

func (v *agentLeaseValidatorV0) requireLeaseExpiredRecommendedAction(field string, decision AgentTimeoutDecisionV0) {
	if !isOneOfAgentLeaseV0(string(decision),
		string(AgentTimeoutDecisionRetryV0),
		string(AgentTimeoutDecisionAskDirectorV0),
		string(AgentTimeoutDecisionStopAgentV0),
		string(AgentTimeoutDecisionMarkFailedV0),
		string(AgentTimeoutDecisionMarkStoppedV0),
		string(AgentTimeoutDecisionReplanTaskV0),
		string(AgentTimeoutDecisionAlertOnlyV0),
	) {
		v.add(ErrAgentLeaseDecisionInvalidaV0, field)
	}
}

func (v *agentLeaseValidatorV0) validateEvaluationClockOrder(input AgentLeaseEvaluationInputV0) {
	if !validAgentLeaseUTCInstantV0(input.LaunchObservedAt) || !validAgentLeaseUTCInstantV0(input.NowObservedAt) {
		return
	}
	launchObservedAt, err := parseAgentLeaseInstantV0(input.LaunchObservedAt)
	if err != nil {
		v.add(ErrAgentLeaseObservedAtV0, "launch_observed_at")
		return
	}
	nowObservedAt, err := parseAgentLeaseInstantV0(input.NowObservedAt)
	if err != nil {
		v.add(ErrAgentLeaseObservedAtV0, "now_observed_at")
		return
	}
	if nowObservedAt.Before(launchObservedAt) {
		v.add(ErrAgentLeaseTiempoInvalidoV0, "now_observed_at")
	}
	if input.LastHeartbeat == nil || !validAgentLeaseUTCInstantV0(input.LastHeartbeat.ObservedAt) {
		return
	}
	heartbeatObservedAt, err := parseAgentLeaseInstantV0(input.LastHeartbeat.ObservedAt)
	if err != nil {
		v.add(ErrAgentLeaseObservedAtV0, "last_heartbeat.observed_at")
		return
	}
	if heartbeatObservedAt.Before(launchObservedAt) {
		v.add(ErrAgentLeaseTiempoInvalidoV0, "last_heartbeat.observed_at")
	}
	if heartbeatObservedAt.After(nowObservedAt) {
		v.add(ErrAgentLeaseTiempoInvalidoV0, "last_heartbeat.observed_at")
	}
}

func parseAgentLeaseInstantV0(value string) (time.Time, error) {
	return time.Parse(agentLeaseInstantLayoutV0, value)
}

func agentLeaseDeadlineReachedV0(now time.Time, observedAt time.Time, timeoutSeconds int) bool {
	deadline := observedAt.Add(time.Duration(timeoutSeconds) * time.Second)
	return !now.Before(deadline)
}
