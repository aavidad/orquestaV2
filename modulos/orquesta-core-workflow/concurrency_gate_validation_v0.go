package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

var forbiddenConcurrencyGateFragmentsV0 = forbiddenReviewReworkFragmentsV0

func validateRecordConcurrencyGatePayloadDataV0(payload RecordConcurrencyGateCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"run_ref":  payload.RunRef,
		"gate_ref": payload.GateRef,
		"plan_ref": payload.PlanRef,
		"decision": string(payload.Decision),
		"summary":  payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateConcurrencyGateCommonV0(payload, true); err != nil {
		return err
	}
	return validateRecordConcurrencyGatePayloadSizeV0(payload)
}

func validateConcurrencyGateRecordedPayloadDataV0(payload ConcurrencyGateRecordedPayloadV0) error {
	commandPayload := RecordConcurrencyGateCommandPayloadV0(payload)
	if err := requirePayloadFieldsV0(map[string]string{
		"run_ref":  commandPayload.RunRef,
		"gate_ref": commandPayload.GateRef,
		"plan_ref": commandPayload.PlanRef,
		"decision": string(commandPayload.Decision),
		"summary":  commandPayload.Summary,
	}); err != nil {
		return err
	}
	if err := validateConcurrencyGateCommonV0(commandPayload, false); err != nil {
		return err
	}
	return validateConcurrencyGateRecordedPayloadSizeV0(payload)
}

func validateConcurrencyGateCommonV0(payload RecordConcurrencyGateCommandPayloadV0, command bool) error {
	if !isSupportedConcurrencyGateDecisionV0(payload.Decision) {
		return concurrencyGatePayloadErrorV0(command, "payload.decision")
	}
	if concurrencyGateProjectionFieldUnsafeV0(payload) {
		return concurrencyGatePayloadErrorV0(command, "payload")
	}
	if concurrencyGatePayloadRefsInvalidV0(payload) {
		return concurrencyGatePayloadErrorV0(command, "payload.refs")
	}
	if err := validateConcurrencyGateDecisionConsistencyV0(payload, command); err != nil {
		return err
	}
	if concurrencyGateHasLongStringV0(concurrencyGateTextFieldsV0(payload)) {
		return concurrencyGatePayloadErrorV0(command, "payload")
	}
	if concurrencyGateHasForbiddenDetailsV0(concurrencyGateTextFieldsV0(payload)) {
		if command {
			return commandErrorV0(ErrDetalleProhibidoV0, "payload")
		}
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return nil
}

func validateConcurrencyGateDecisionConsistencyV0(payload RecordConcurrencyGateCommandPayloadV0, command bool) error {
	subjects := stringSetFromStringsV0(payload.SubjectClaimRefs)
	ready := stringSetFromStringsV0(payload.ReadyClaimRefs)
	blocked := stringSetFromStringsV0(payload.BlockedClaimRefs)
	switch payload.Decision {
	case ConcurrencyGateDecisionAllowRequestAgentV0:
		if len(subjects) == 0 || !allStringsInSetV0(payload.SubjectClaimRefs, ready) || anyStringInSetV0(payload.SubjectClaimRefs, blocked) {
			return concurrencyGatePayloadErrorV0(command, "payload.subject_claim_refs")
		}
	case ConcurrencyGateDecisionBlockRequestAgentV0:
		if len(subjects) == 0 || !anyStringInSetV0(payload.SubjectClaimRefs, blocked) {
			return concurrencyGatePayloadErrorV0(command, "payload.subject_claim_refs")
		}
	case ConcurrencyGateDecisionAskDirectorV0:
		return nil
	}
	return nil
}

func concurrencyGatePayloadErrorV0(command bool, field string) error {
	if command {
		return commandErrorV0(ErrPayloadInvalidoV0, field)
	}
	return eventErrorV0(ErrPayloadInvalidoV0, field)
}

func isSupportedConcurrencyGateDecisionV0(decision ConcurrencyGateDecisionV0) bool {
	switch decision {
	case ConcurrencyGateDecisionAllowRequestAgentV0,
		ConcurrencyGateDecisionBlockRequestAgentV0,
		ConcurrencyGateDecisionAskDirectorV0:
		return true
	default:
		return false
	}
}

func concurrencyGatePayloadRefsInvalidV0(payload RecordConcurrencyGateCommandPayloadV0) bool {
	refGroups := [][]string{
		payload.SubjectClaimRefs,
		payload.ReadyClaimRefs,
		payload.BlockedClaimRefs,
		payload.ConflictRefs,
		payload.EvidenceRefs,
	}
	for _, refs := range refGroups {
		if len(refs) > maxConcurrencyGateRefsV0 || concurrencyGateHasLongStringV0(refs) {
			return true
		}
	}
	return false
}

func concurrencyGateHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxConcurrencyGateStringV0 {
			return true
		}
	}
	return false
}

func concurrencyGateHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenConcurrencyGateFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func validateRecordConcurrencyGatePayloadSizeV0(payload RecordConcurrencyGateCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxConcurrencyGatePayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateConcurrencyGateRecordedPayloadSizeV0(payload ConcurrencyGateRecordedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxConcurrencyGatePayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func concurrencyGateTextFieldsV0(payload RecordConcurrencyGateCommandPayloadV0) []string {
	values := []string{payload.RunRef, payload.GateRef, payload.PlanRef, string(payload.Decision), payload.Summary}
	values = append(values, payload.SubjectClaimRefs...)
	values = append(values, payload.ReadyClaimRefs...)
	values = append(values, payload.BlockedClaimRefs...)
	values = append(values, payload.ConflictRefs...)
	values = append(values, payload.EvidenceRefs...)
	return values
}
