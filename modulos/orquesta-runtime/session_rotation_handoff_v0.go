package orquestaruntime

import "strings"

const (
	RuntimeSessionRotationRequestSchemaVersionV0 = "runtime_session_rotation_request.v0"
	RuntimeSessionRotationHandoffSchemaVersionV0 = "runtime_session_rotation_handoff.v0"

	RuntimeSessionRotationModeOptInLowPriorityV0 = "opt_in_low_priority"

	RuntimeSessionRotationActionContinueCurrentV0 = "continue_current_session"
	RuntimeSessionRotationActionRequestHandoffV0  = "request_complete_handoff"
	RuntimeSessionRotationActionLaunchNextV0      = "launch_replacement_session"

	RuntimeSessionRotationOutgoingStatusHandoffReadyV0 = "handoff_ready"
)

type RuntimeSessionRotationRequestV0 struct {
	SchemaVersion      string                           `json:"schema_version"`
	RequestRef         string                           `json:"request_ref"`
	CorrelationID      string                           `json:"correlation_id"`
	RunRef             string                           `json:"run_ref"`
	TaskRef            string                           `json:"task_ref"`
	AgentRef           string                           `json:"agent_ref"`
	SessionRef         string                           `json:"session_ref"`
	SessionEpoch       string                           `json:"session_epoch"`
	ExperimentRef      string                           `json:"experiment_ref"`
	Mode               string                           `json:"mode"`
	RotationReason     string                           `json:"rotation_reason"`
	CurrentProcessLive bool                             `json:"current_process_live"`
	Handoff            *RuntimeSessionRotationHandoffV0 `json:"handoff,omitempty"`
	EvidenceRefs       []string                         `json:"evidence_refs,omitempty"`
}

type RuntimeSessionRotationHandoffV0 struct {
	SchemaVersion     string   `json:"schema_version"`
	HandoffRef        string   `json:"handoff_ref"`
	OutgoingAckRef    string   `json:"outgoing_ack_ref"`
	OutgoingStatus    string   `json:"outgoing_status"`
	PendingStatus     string   `json:"pending_status"`
	OriginalObjective string   `json:"original_objective"`
	ProgressSummary   string   `json:"progress_summary"`
	NextAction        string   `json:"next_action"`
	PendingWork       []string `json:"pending_work"`
	TouchedFiles      []string `json:"touched_files,omitempty"`
	Risks             []string `json:"risks,omitempty"`
	RequiredTests     []string `json:"required_tests,omitempty"`
	CausalRefs        []string `json:"causal_refs"`
	EvidenceRefs      []string `json:"evidence_refs"`
}

type RuntimeSessionRotationDecisionV0 struct {
	Action             string                         `json:"action"`
	Reason             string                         `json:"reason"`
	LaunchAllowed      bool                           `json:"launch_allowed"`
	StopCurrentAllowed bool                           `json:"stop_current_allowed"`
	HandoffAccepted    bool                           `json:"handoff_accepted"`
	SessionEpoch       string                         `json:"session_epoch,omitempty"`
	ComparisonMetric   RuntimeSessionRotationMetricV0 `json:"comparison_metric"`
	EvidenceRefs       []string                       `json:"evidence_refs,omitempty"`
	Issues             []RuntimeLaunchErrorV0         `json:"issues,omitempty"`
}

type RuntimeSessionRotationMetricV0 struct {
	SchemaVersion    string   `json:"schema_version"`
	MetricRef        string   `json:"metric_ref"`
	BaselineAction   string   `json:"baseline_action"`
	CandidateAction  string   `json:"candidate_action"`
	ContinuitySignal string   `json:"continuity_signal"`
	CostSignal       string   `json:"cost_signal"`
	QualitySignal    string   `json:"quality_signal"`
	DecisionReason   string   `json:"decision_reason"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

func EvaluateRuntimeSessionRotationV0(
	request RuntimeSessionRotationRequestV0,
) RuntimeSessionRotationDecisionV0 {
	request = normalizeRuntimeSessionRotationRequestV0(request)
	issues := ValidateRuntimeSessionRotationRequestV0(request)
	if !runtimeSessionRotationRequestCoreValidV0(request, issues) {
		return runtimeSessionRotationDecisionV0(
			RuntimeSessionRotationActionContinueCurrentV0,
			"session_rotation_contract_invalid",
			request,
			issues,
		)
	}
	if request.Mode != RuntimeSessionRotationModeOptInLowPriorityV0 {
		return runtimeSessionRotationDecisionV0(
			RuntimeSessionRotationActionContinueCurrentV0,
			"session_rotation_not_opt_in",
			request,
			nil,
		)
	}
	if request.Handoff == nil {
		return runtimeSessionRotationDecisionV0(
			RuntimeSessionRotationActionRequestHandoffV0,
			"session_rotation_handoff_insufficient",
			request,
			[]RuntimeLaunchErrorV0{{
				Code:          RuntimeLaunchRequestInvalidaV0,
				Field:         "handoff",
				CorrelationID: request.CorrelationID,
			}},
		)
	}
	if handoffIssues := ValidateRuntimeSessionRotationHandoffV0(*request.Handoff, request.CorrelationID); len(handoffIssues) > 0 {
		return runtimeSessionRotationDecisionV0(
			RuntimeSessionRotationActionRequestHandoffV0,
			"session_rotation_handoff_insufficient",
			request,
			handoffIssues,
		)
	}
	decision := runtimeSessionRotationDecisionV0(
		RuntimeSessionRotationActionLaunchNextV0,
		"session_rotation_handoff_accepted",
		request,
		nil,
	)
	decision.LaunchAllowed = true
	decision.HandoffAccepted = true
	return decision
}

func ValidateRuntimeSessionRotationRequestV0(
	request RuntimeSessionRotationRequestV0,
) []RuntimeLaunchErrorV0 {
	v := runtimeLaunchRequestValidatorV0{correlationID: request.CorrelationID}
	v.requireConst("schema_version", request.SchemaVersion, RuntimeSessionRotationRequestSchemaVersionV0, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("request_ref", request.RequestRef, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("correlation_id", request.CorrelationID, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("run_ref", request.RunRef, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("task_ref", request.TaskRef, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("agent_ref", request.AgentRef, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("session_ref", request.SessionRef, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("session_epoch", request.SessionEpoch, RuntimeLaunchRequestInvalidaV0)
	v.optionalOpaque("experiment_ref", request.ExperimentRef)
	if request.Mode != "" && request.Mode != RuntimeSessionRotationModeOptInLowPriorityV0 {
		v.add(RuntimeLaunchRequestInvalidaV0, "mode")
	}
	v.requireNonEmpty("rotation_reason", request.RotationReason)
	if request.Handoff == nil {
		v.add(RuntimeLaunchRequestInvalidaV0, "handoff")
	}
	for index, ref := range request.EvidenceRefs {
		v.optionalOpaque(indexedFieldV0("evidence_refs", index), ref)
	}
	return v.errors
}

func ValidateRuntimeSessionRotationHandoffV0(
	handoff RuntimeSessionRotationHandoffV0,
	correlationID string,
) []RuntimeLaunchErrorV0 {
	v := runtimeLaunchRequestValidatorV0{correlationID: correlationID}
	v.requireConst("handoff.schema_version", handoff.SchemaVersion, RuntimeSessionRotationHandoffSchemaVersionV0, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("handoff.handoff_ref", handoff.HandoffRef, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("handoff.outgoing_ack_ref", handoff.OutgoingAckRef, AckRefRequeridoV0)
	v.requireConst("handoff.outgoing_status", handoff.OutgoingStatus, RuntimeSessionRotationOutgoingStatusHandoffReadyV0, RuntimeLaunchRequestInvalidaV0)
	v.requireNonEmpty("handoff.pending_status", handoff.PendingStatus)
	v.requireNonEmpty("handoff.original_objective", handoff.OriginalObjective)
	v.requireNonEmpty("handoff.progress_summary", handoff.ProgressSummary)
	v.requireNonEmpty("handoff.next_action", handoff.NextAction)
	validateSessionRotationStringsV0(&v, "handoff.pending_work", handoff.PendingWork, true, false)
	validateSessionRotationStringsV0(&v, "handoff.touched_files", handoff.TouchedFiles, false, true)
	validateSessionRotationStringsV0(&v, "handoff.required_tests", handoff.RequiredTests, false, false)
	validateSessionRotationOpaqueRefsV0(&v, "handoff.causal_refs", handoff.CausalRefs, true)
	validateSessionRotationOpaqueRefsV0(&v, "handoff.evidence_refs", handoff.EvidenceRefs, true)
	return v.errors
}

func normalizeRuntimeSessionRotationRequestV0(
	request RuntimeSessionRotationRequestV0,
) RuntimeSessionRotationRequestV0 {
	request.SchemaVersion = strings.TrimSpace(request.SchemaVersion)
	request.RequestRef = strings.TrimSpace(request.RequestRef)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.TaskRef = strings.TrimSpace(request.TaskRef)
	request.AgentRef = strings.TrimSpace(request.AgentRef)
	request.SessionRef = strings.TrimSpace(request.SessionRef)
	request.SessionEpoch = strings.TrimSpace(request.SessionEpoch)
	request.ExperimentRef = strings.TrimSpace(request.ExperimentRef)
	request.Mode = strings.TrimSpace(request.Mode)
	request.RotationReason = strings.TrimSpace(request.RotationReason)
	request.EvidenceRefs = compactRuntimeSessionRotationStringsV0(request.EvidenceRefs)
	if request.Handoff != nil {
		handoff := normalizeRuntimeSessionRotationHandoffV0(*request.Handoff)
		request.Handoff = &handoff
	}
	return request
}

func normalizeRuntimeSessionRotationHandoffV0(
	handoff RuntimeSessionRotationHandoffV0,
) RuntimeSessionRotationHandoffV0 {
	handoff.SchemaVersion = strings.TrimSpace(handoff.SchemaVersion)
	handoff.HandoffRef = strings.TrimSpace(handoff.HandoffRef)
	handoff.OutgoingAckRef = strings.TrimSpace(handoff.OutgoingAckRef)
	handoff.OutgoingStatus = strings.TrimSpace(handoff.OutgoingStatus)
	handoff.PendingStatus = strings.TrimSpace(handoff.PendingStatus)
	handoff.OriginalObjective = strings.TrimSpace(handoff.OriginalObjective)
	handoff.ProgressSummary = strings.TrimSpace(handoff.ProgressSummary)
	handoff.NextAction = strings.TrimSpace(handoff.NextAction)
	handoff.PendingWork = compactRuntimeSessionRotationStringsV0(handoff.PendingWork)
	handoff.TouchedFiles = compactRuntimeSessionRotationStringsV0(handoff.TouchedFiles)
	handoff.Risks = compactRuntimeSessionRotationStringsV0(handoff.Risks)
	handoff.RequiredTests = compactRuntimeSessionRotationStringsV0(handoff.RequiredTests)
	handoff.CausalRefs = compactRuntimeSessionRotationStringsV0(handoff.CausalRefs)
	handoff.EvidenceRefs = compactRuntimeSessionRotationStringsV0(handoff.EvidenceRefs)
	return handoff
}
