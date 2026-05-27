package orquestacionnucleoapp

import (
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	SessionRotationDirectiveContinueCurrentV0 = "continue_current_session"
	SessionRotationDirectiveAskHandoffV0      = "ask_director_complete_handoff"
	SessionRotationDirectiveLaunchNextV0      = "launch_replacement_session"
)

type SessionRotationDirectiveV0 struct {
	Directive            string                                         `json:"directive"`
	Reason               string                                         `json:"reason"`
	CanLaunchReplacement bool                                           `json:"can_launch_replacement"`
	CanStopCurrent       bool                                           `json:"can_stop_current"`
	DirectorQuestion     string                                         `json:"director_question,omitempty"`
	SessionEpoch         string                                         `json:"session_epoch,omitempty"`
	HandoffRef           string                                         `json:"handoff_ref,omitempty"`
	ComparisonMetric     orquestaruntime.RuntimeSessionRotationMetricV0 `json:"comparison_metric"`
	EvidenceRefs         []string                                       `json:"evidence_refs,omitempty"`
	Issues               []orquestaruntime.RuntimeLaunchErrorV0         `json:"issues,omitempty"`
}

func BuildSessionRotationDirectiveV0(
	request orquestaruntime.RuntimeSessionRotationRequestV0,
) SessionRotationDirectiveV0 {
	decision := orquestaruntime.EvaluateRuntimeSessionRotationV0(request)
	directive := SessionRotationDirectiveV0{
		Directive:            SessionRotationDirectiveContinueCurrentV0,
		Reason:               decision.Reason,
		CanLaunchReplacement: decision.LaunchAllowed,
		CanStopCurrent:       decision.StopCurrentAllowed,
		SessionEpoch:         strings.TrimSpace(decision.SessionEpoch),
		ComparisonMetric:     decision.ComparisonMetric,
		EvidenceRefs:         append([]string(nil), decision.EvidenceRefs...),
		Issues:               append([]orquestaruntime.RuntimeLaunchErrorV0(nil), decision.Issues...),
	}
	if request.Handoff != nil {
		directive.HandoffRef = strings.TrimSpace(request.Handoff.HandoffRef)
	}
	switch decision.Action {
	case orquestaruntime.RuntimeSessionRotationActionLaunchNextV0:
		directive.Directive = SessionRotationDirectiveLaunchNextV0
	case orquestaruntime.RuntimeSessionRotationActionRequestHandoffV0:
		directive.Directive = SessionRotationDirectiveAskHandoffV0
		directive.DirectorQuestion = "complete_handoff_or_continue_current_session"
		directive.CanLaunchReplacement = false
		directive.CanStopCurrent = false
	default:
		directive.CanLaunchReplacement = false
		directive.CanStopCurrent = false
	}
	return directive
}
