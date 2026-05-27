package orquestaruntime

import (
	"fmt"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func runtimeSessionRotationRequestCoreValidV0(
	request RuntimeSessionRotationRequestV0,
	issues []RuntimeLaunchErrorV0,
) bool {
	if len(issues) == 0 {
		return true
	}
	for _, issue := range issues {
		if issue.Field == "handoff" {
			continue
		}
		if strings.HasPrefix(issue.Field, "handoff.") {
			continue
		}
		return false
	}
	return true
}

func runtimeSessionRotationDecisionV0(
	action string,
	reason string,
	request RuntimeSessionRotationRequestV0,
	issues []RuntimeLaunchErrorV0,
) RuntimeSessionRotationDecisionV0 {
	refs := append([]string(nil), request.EvidenceRefs...)
	switch action {
	case RuntimeSessionRotationActionLaunchNextV0:
		refs = append(refs, "evidence-ref-session-rotation-handoff-accepted")
	case RuntimeSessionRotationActionRequestHandoffV0:
		refs = append(refs, "evidence-ref-session-rotation-handoff-required")
	default:
		refs = append(refs, "evidence-ref-session-rotation-continued-current")
	}
	return RuntimeSessionRotationDecisionV0{
		Action:           action,
		Reason:           reason,
		SessionEpoch:     request.SessionEpoch,
		ComparisonMetric: runtimeSessionRotationMetricV0(action, reason, request, refs),
		EvidenceRefs:     compactRuntimeSessionRotationStringsV0(refs),
		Issues:           issues,
	}
}

func runtimeSessionRotationMetricV0(
	action string,
	reason string,
	request RuntimeSessionRotationRequestV0,
	refs []string,
) RuntimeSessionRotationMetricV0 {
	return RuntimeSessionRotationMetricV0{
		SchemaVersion:    "runtime_session_rotation_metric.v0",
		MetricRef:        runtimeSessionRotationMetricRefV0(request.RequestRef),
		BaselineAction:   RuntimeSessionRotationActionContinueCurrentV0,
		CandidateAction:  action,
		ContinuitySignal: runtimeSessionRotationContinuitySignalV0(action, reason),
		CostSignal:       "opt_in_low_priority_experiment",
		QualitySignal:    runtimeSessionRotationQualitySignalV0(action, request),
		DecisionReason:   reason,
		EvidenceRefs:     compactRuntimeSessionRotationStringsV0(refs),
	}
}

func runtimeSessionRotationMetricRefV0(requestRef string) string {
	requestRef = strings.TrimSpace(requestRef)
	if requestRef == "" {
		return "metric-ref-session-rotation-unknown"
	}
	return "metric-ref-session-rotation-" + requestRef
}

func runtimeSessionRotationContinuitySignalV0(action string, reason string) string {
	if action == RuntimeSessionRotationActionLaunchNextV0 {
		return "handoff_accepted_context_bounded"
	}
	if action == RuntimeSessionRotationActionRequestHandoffV0 {
		return "handoff_required_before_relaunch"
	}
	return reason
}

func runtimeSessionRotationQualitySignalV0(
	action string,
	request RuntimeSessionRotationRequestV0,
) string {
	if action == RuntimeSessionRotationActionLaunchNextV0 &&
		request.Handoff != nil &&
		len(request.Handoff.RequiredTests) > 0 {
		return "required_tests_declared_in_handoff"
	}
	if action == RuntimeSessionRotationActionLaunchNextV0 {
		return "handoff_evidence_declared"
	}
	return "current_session_preserved"
}

func validateSessionRotationStringsV0(
	v *runtimeLaunchRequestValidatorV0,
	field string,
	values []string,
	required bool,
	paths bool,
) {
	if required && len(values) == 0 {
		v.add(RuntimeLaunchRequestInvalidaV0, field)
		return
	}
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			v.add(RuntimeLaunchRequestInvalidaV0, indexedFieldV0(field, index))
			continue
		}
		if paths && !orquestarails.WorkspaceRelativePathAllowedV0(value, true) {
			v.add(WriteSetInvalidoV0, indexedFieldV0(field, index))
		}
		if looksLikeSecret(value) || (!paths && looksLikeConcreteHomePathInNarrativeV0(value)) {
			v.add(SecretoDetectadoV0, indexedFieldV0(field, index))
		}
	}
}

func validateSessionRotationOpaqueRefsV0(
	v *runtimeLaunchRequestValidatorV0,
	field string,
	values []string,
	required bool,
) {
	if required && len(values) == 0 {
		v.add(RuntimeLaunchRequestInvalidaV0, field)
		return
	}
	for index, value := range values {
		v.requireOpaque(indexedFieldV0(field, index), strings.TrimSpace(value), RuntimeLaunchRequestInvalidaV0)
	}
}

func compactRuntimeSessionRotationStringsV0(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func indexedFieldV0(field string, index int) string {
	return fmt.Sprintf("%s[%d]", field, index)
}

func looksLikeConcreteHomePathInNarrativeV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(low, "/home/") ||
		strings.HasPrefix(low, "~/") ||
		strings.Contains(low, "$home") ||
		strings.Contains(low, "${home}") ||
		(len(low) >= 3 && isASCIIAlpha(low[0]) && low[1] == ':' && (low[2] == '\\' || low[2] == '/'))
}
