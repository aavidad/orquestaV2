package orquestacoreworkflow

import (
	"encoding/json"
)

func validateAssessAgentWorkCommandPayloadDataV0(payload AssessAgentWorkCommandPayloadV0) error {
	if err := validateAssessAgentWorkRequiredV0(payload); err != nil {
		return err
	}
	if err := validateAssessmentEnumsForCommandV0(payload); err != nil {
		return err
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(assessmentTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateAssessmentPayloadSizeV0(payload)
}

func validateAgentWorkAssessedPayloadDataV0(payload AgentWorkAssessedPayloadV0) error {
	commandPayload := AssessAgentWorkCommandPayloadV0(payload)
	if err := requirePayloadFieldsV0(assessmentRequiredFieldsV0(commandPayload)); err != nil {
		return err
	}
	if !assessmentEnumsValidV0(commandPayload) || !assessmentActionMatchesVerdictV0(commandPayload) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(assessmentTextFieldsV0(commandPayload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateAssessmentEventPayloadSizeV0(payload)
}

func validateAssessmentEnumsForCommandV0(payload AssessAgentWorkCommandPayloadV0) error {
	if !validAssessmentVerdictV0(payload.Verdict) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.verdict")
	}
	if !validAssessmentActionV0(payload.Action) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.action")
	}
	if !validAssessmentSeverityV0(payload.Severity) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.severity")
	}
	if !assessmentActionMatchesVerdictV0(payload) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.action")
	}
	return nil
}

func validateAssessAgentWorkRequiredV0(payload AssessAgentWorkCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(assessmentRequiredFieldsV0(payload)); err != nil {
		return err
	}
	return ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID))
}

func assessmentRequiredFieldsV0(payload AssessAgentWorkCommandPayloadV0) map[string]string {
	return map[string]string{
		"assessment_ref":   payload.AssessmentRef,
		"phase_id":         payload.PhaseID,
		"agent_request_id": payload.AgentRequestID,
		"verdict":          payload.Verdict,
		"action":           payload.Action,
		"severity":         payload.Severity,
		"summary":          payload.Summary,
	}
}

func assessmentEnumsValidV0(payload AssessAgentWorkCommandPayloadV0) bool {
	return validAssessmentVerdictV0(payload.Verdict) &&
		validAssessmentActionV0(payload.Action) &&
		validAssessmentSeverityV0(payload.Severity)
}

func assessmentActionMatchesVerdictV0(payload AssessAgentWorkCommandPayloadV0) bool {
	action := normalizeAgentAssessmentActionV0(payload.Action)
	if action != AgentAssessmentActionStopAgentV0 {
		return true
	}
	verdict := normalizeAgentAssessmentVerdictV0(payload.Verdict)
	return verdict == AgentAssessmentVerdictGarbageV0 ||
		verdict == AgentAssessmentVerdictLoopDetectedV0 ||
		verdict == AgentAssessmentVerdictCapacityLimitedV0 ||
		verdict == AgentAssessmentVerdictTimeoutV0
}

func validateAssessmentPayloadSizeV0(payload AssessAgentWorkCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateAssessmentEventPayloadSizeV0(payload AgentWorkAssessedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func assessmentTextFieldsV0(payload AssessAgentWorkCommandPayloadV0) []string {
	values := []string{
		payload.AssessmentRef, payload.PhaseID, payload.AgentRequestID,
		payload.TaskRef, payload.DeliveryRef, payload.Verdict, payload.Action,
		payload.Severity, payload.Summary,
	}
	return append(values, payload.EvidenceRefs...)
}

func validAssessmentVerdictV0(value string) bool {
	switch normalizeAgentAssessmentVerdictV0(value) {
	case AgentAssessmentVerdictAcceptableV0, AgentAssessmentVerdictNeedsRevisionV0,
		AgentAssessmentVerdictGarbageV0, AgentAssessmentVerdictLoopDetectedV0,
		AgentAssessmentVerdictCapacityLimitedV0, AgentAssessmentVerdictTimeoutV0:
		return true
	default:
		return false
	}
}

func validAssessmentActionV0(value string) bool {
	switch normalizeAgentAssessmentActionV0(value) {
	case AgentAssessmentActionContinueV0, AgentAssessmentActionRequestRevisionV0,
		AgentAssessmentActionStopAgentV0, AgentAssessmentActionAskDirectorV0:
		return true
	default:
		return false
	}
}

func validAssessmentSeverityV0(value string) bool {
	switch normalizeAgentAssessmentSeverityV0(value) {
	case AgentAssessmentSeverityLowV0, AgentAssessmentSeverityMediumV0,
		AgentAssessmentSeverityHighV0, AgentAssessmentSeverityCriticalV0:
		return true
	default:
		return false
	}
}

func normalizeAgentAssessmentVerdictV0(value string) string {
	switch normalized := normalizeLooseEnumTokenV0(value); normalized {
	case AgentAssessmentVerdictAcceptableV0, "accepted", "accept", "approved", "ok", "pass":
		return AgentAssessmentVerdictAcceptableV0
	case AgentAssessmentVerdictNeedsRevisionV0, "needs_review", "needs_changes", "changes_requested",
		"request_changes", "revision", "revision_required":
		return AgentAssessmentVerdictNeedsRevisionV0
	case AgentAssessmentVerdictGarbageV0, "invalid", "not_useful", "unusable", "nonsense":
		return AgentAssessmentVerdictGarbageV0
	case AgentAssessmentVerdictLoopDetectedV0, "loop", "looping":
		return AgentAssessmentVerdictLoopDetectedV0
	case AgentAssessmentVerdictCapacityLimitedV0, "capacity", "capacity_limit", "capacity_blocked",
		"no_capacity":
		return AgentAssessmentVerdictCapacityLimitedV0
	case AgentAssessmentVerdictTimeoutV0, "time_out", "timed_out":
		return AgentAssessmentVerdictTimeoutV0
	default:
		return normalized
	}
}

func normalizeAgentAssessmentActionV0(value string) string {
	switch normalized := normalizeLooseEnumTokenV0(value); normalized {
	case AgentAssessmentActionContinueV0, "proceed", "keep_going", "keep_running":
		return AgentAssessmentActionContinueV0
	case AgentAssessmentActionRequestRevisionV0, "request_review", "ask_revision", "revise",
		"rework", "request_changes", "needs_revision":
		return AgentAssessmentActionRequestRevisionV0
	case AgentAssessmentActionStopAgentV0, "stop", "halt_agent", "terminate_agent":
		return AgentAssessmentActionStopAgentV0
	case AgentAssessmentActionAskDirectorV0, "ask", "director", "ask_supervisor", "escalate":
		return AgentAssessmentActionAskDirectorV0
	default:
		return normalized
	}
}

func normalizeAgentAssessmentSeverityV0(value string) string {
	switch normalized := normalizeLooseEnumTokenV0(value); normalized {
	case AgentAssessmentSeverityLowV0, "minor", "baja":
		return AgentAssessmentSeverityLowV0
	case AgentAssessmentSeverityMediumV0, "normal", "media":
		return AgentAssessmentSeverityMediumV0
	case AgentAssessmentSeverityHighV0, "major", "alta":
		return AgentAssessmentSeverityHighV0
	case AgentAssessmentSeverityCriticalV0, "blocker", "urgent", "critica", "critical_blocker":
		return AgentAssessmentSeverityCriticalV0
	default:
		return normalized
	}
}
