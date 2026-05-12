package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
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
	if payload.Action != AgentAssessmentActionStopAgentV0 {
		return true
	}
	return payload.Verdict == AgentAssessmentVerdictGarbageV0 ||
		payload.Verdict == AgentAssessmentVerdictLoopDetectedV0 ||
		payload.Verdict == AgentAssessmentVerdictCapacityLimitedV0
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
	switch strings.TrimSpace(value) {
	case AgentAssessmentVerdictAcceptableV0, AgentAssessmentVerdictNeedsRevisionV0,
		AgentAssessmentVerdictGarbageV0, AgentAssessmentVerdictLoopDetectedV0,
		AgentAssessmentVerdictCapacityLimitedV0:
		return true
	default:
		return false
	}
}

func validAssessmentActionV0(value string) bool {
	switch strings.TrimSpace(value) {
	case AgentAssessmentActionContinueV0, AgentAssessmentActionRequestRevisionV0,
		AgentAssessmentActionStopAgentV0, AgentAssessmentActionAskDirectorV0:
		return true
	default:
		return false
	}
}

func validAssessmentSeverityV0(value string) bool {
	switch strings.TrimSpace(value) {
	case AgentAssessmentSeverityLowV0, AgentAssessmentSeverityMediumV0,
		AgentAssessmentSeverityHighV0, AgentAssessmentSeverityCriticalV0:
		return true
	default:
		return false
	}
}
