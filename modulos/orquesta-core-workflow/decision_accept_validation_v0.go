package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func validateAcceptDecisionCommandPayloadDataV0(payload AcceptDecisionCommandPayloadV0) error {
	if err := validateAcceptDecisionRequiredV0(payload); err != nil {
		return err
	}
	if decisionAcceptStringsInvalidV0(payload.EvidenceRefs) || decisionAcceptHasLongStringV0(acceptDecisionTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return validateAcceptDecisionPayloadSizeV0(payload)
}

func validateArchitectureDecisionAcceptedPayloadDataV0(payload ArchitectureDecisionAcceptedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"decision_ref":        payload.DecisionRef,
		"phase_id":            payload.PhaseID,
		"vote_ref":            payload.VoteRef,
		"accepted_option_ref": payload.AcceptedOptionRef,
		"summary":             payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateDecisionAcceptedEventPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if decisionAcceptStringsInvalidV0(payload.EvidenceRefs) || decisionAcceptHasLongStringV0(architectureDecisionAcceptedTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return validateArchitectureDecisionAcceptedPayloadSizeV0(payload)
}

func validateAcceptDecisionRequiredV0(payload AcceptDecisionCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"decision_ref":        payload.DecisionRef,
		"phase_id":            payload.PhaseID,
		"vote_ref":            payload.VoteRef,
		"accepted_option_ref": payload.AcceptedOptionRef,
		"summary":             payload.Summary,
	}); err != nil {
		return err
	}
	return validateAcceptDecisionCommandPhaseIDV0(payload.PhaseID)
}

func validateAcceptDecisionCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseVotacionYDecisionV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateDecisionAcceptedEventPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseVotacionYDecisionV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateAcceptDecisionPayloadSizeV0(payload AcceptDecisionCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxDecisionAcceptPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateArchitectureDecisionAcceptedPayloadSizeV0(payload ArchitectureDecisionAcceptedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxDecisionAcceptPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func decisionAcceptStringsInvalidV0(values []string) bool {
	if len(values) > maxDecisionAcceptEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxDecisionAcceptStringV0 {
			return true
		}
	}
	return false
}

func decisionAcceptHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxDecisionAcceptStringV0 {
			return true
		}
	}
	return false
}

func acceptDecisionTextFieldsV0(payload AcceptDecisionCommandPayloadV0) []string {
	values := []string{payload.DecisionRef, payload.PhaseID, payload.VoteRef, payload.AcceptedOptionRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func architectureDecisionAcceptedTextFieldsV0(payload ArchitectureDecisionAcceptedPayloadV0) []string {
	values := []string{payload.DecisionRef, payload.PhaseID, payload.VoteRef, payload.AcceptedOptionRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
