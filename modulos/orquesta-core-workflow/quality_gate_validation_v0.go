package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func validateRecordQualityGatePayloadDataV0(payload RecordQualityGateCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"run_ref":     payload.RunRef,
		"gate_ref":    payload.GateRef,
		"phase_id":    payload.PhaseID,
		"subject_ref": payload.SubjectRef,
		"decision":    string(payload.Decision),
		"summary":     payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateQualityGateCommonV0(payload, true); err != nil {
		return err
	}
	return validateRecordQualityGatePayloadSizeV0(payload)
}

func validateQualityGateRecordedPayloadDataV0(payload QualityGateRecordedPayloadV0) error {
	commandPayload := RecordQualityGateCommandPayloadV0(payload)
	if err := requirePayloadFieldsV0(map[string]string{
		"run_ref":     commandPayload.RunRef,
		"gate_ref":    commandPayload.GateRef,
		"phase_id":    commandPayload.PhaseID,
		"subject_ref": commandPayload.SubjectRef,
		"decision":    string(commandPayload.Decision),
		"summary":     commandPayload.Summary,
	}); err != nil {
		return err
	}
	if err := validateQualityGateCommonV0(commandPayload, false); err != nil {
		return err
	}
	return validateQualityGateRecordedPayloadSizeV0(payload)
}

func validateQualityGateCommonV0(payload RecordQualityGateCommandPayloadV0, command bool) error {
	if !isSupportedQualityGateDecisionV0(payload.Decision) {
		return qualityGatePayloadErrorV0(command, "payload.decision")
	}
	if err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID)); err != nil {
		return qualityGatePayloadErrorV0(command, "payload.phase_id")
	}
	if qualityGateProjectionFieldUnsafeV0(payload) {
		return qualityGatePayloadErrorV0(command, "payload")
	}
	if qualityGateRefsInvalidForPayloadV0(payload) {
		return qualityGatePayloadErrorV0(command, "payload.refs")
	}
	if err := validateQualityGateDecisionConsistencyV0(payload, command); err != nil {
		return err
	}
	if qualityGateHasLongStringV0(qualityGateTextFieldsV0(payload)) {
		return qualityGatePayloadErrorV0(command, "payload")
	}
	if qualityGateHasForbiddenDetailsV0(qualityGateTextFieldsV0(payload)) {
		if command {
			return commandErrorV0(ErrDetalleProhibidoV0, "payload")
		}
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return nil
}

func validateQualityGateDecisionConsistencyV0(payload RecordQualityGateCommandPayloadV0, command bool) error {
	switch payload.Decision {
	case QualityGateDecisionAcceptedV0:
		return nil
	case QualityGateDecisionReworkRequiredV0, QualityGateDecisionBlockedV0, QualityGateDecisionAskDirectorV0:
		if len(payload.IssueRefs) == 0 {
			return qualityGatePayloadErrorV0(command, "payload.issue_refs")
		}
	}
	return nil
}

func qualityGatePayloadErrorV0(command bool, field string) error {
	if command {
		return commandErrorV0(ErrPayloadInvalidoV0, field)
	}
	return eventErrorV0(ErrPayloadInvalidoV0, field)
}

func isSupportedQualityGateDecisionV0(decision QualityGateDecisionV0) bool {
	switch decision {
	case QualityGateDecisionAcceptedV0,
		QualityGateDecisionReworkRequiredV0,
		QualityGateDecisionBlockedV0,
		QualityGateDecisionAskDirectorV0:
		return true
	default:
		return false
	}
}

func qualityGateRefsInvalidForPayloadV0(payload RecordQualityGateCommandPayloadV0) bool {
	refGroups := [][]string{payload.IssueRefs, payload.EvidenceRefs}
	for _, refs := range refGroups {
		if len(refs) > maxQualityGateRefsV0 || qualityGateHasLongStringV0(refs) {
			return true
		}
		for _, ref := range refs {
			if strings.TrimSpace(ref) == "" {
				return true
			}
		}
	}
	return false
}

func qualityGateHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxQualityGateStringV0 {
			return true
		}
	}
	return false
}

func qualityGateHasForbiddenDetailsV0(values []string) bool {
	return textValuesContainForbiddenOperationalSensitiveDetailV0(values)
}

func validateRecordQualityGatePayloadSizeV0(payload RecordQualityGateCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxQualityGatePayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateQualityGateRecordedPayloadSizeV0(payload QualityGateRecordedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxQualityGatePayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func qualityGateTextFieldsV0(payload RecordQualityGateCommandPayloadV0) []string {
	values := []string{
		payload.RunRef,
		payload.GateRef,
		payload.PhaseID,
		payload.SubjectRef,
		string(payload.Decision),
		payload.Summary,
	}
	values = append(values, payload.IssueRefs...)
	return append(values, payload.EvidenceRefs...)
}
