package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func validateRegisterPhaseArtifactCommandPayloadDataV0(payload RegisterPhaseArtifactCommandPayloadV0) error {
	if err := validateRegisterPhaseArtifactRequiredV0(payload); err != nil {
		return err
	}
	if phaseArtifactStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if phaseArtifactProjectionFieldUnsafeV0(payload.ArtifactRef, payload.AgentRef) ||
		phaseArtifactHasLongStringV0(registerPhaseArtifactTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if phaseArtifactHasForbiddenDetailsV0(registerPhaseArtifactTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateRegisterPhaseArtifactPayloadSizeV0(payload)
}

func validatePhaseArtifactRegisteredPayloadDataV0(payload PhaseArtifactRegisteredPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"artifact_ref": payload.ArtifactRef,
		"phase_id":     payload.PhaseID,
		"agent_ref":    payload.AgentRef,
		"summary":      payload.Summary,
	}); err != nil {
		return err
	}
	if err := validatePhaseArtifactEventPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if phaseArtifactStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if phaseArtifactProjectionFieldUnsafeV0(payload.ArtifactRef, payload.AgentRef) ||
		phaseArtifactHasLongStringV0(phaseArtifactRegisteredTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if phaseArtifactHasForbiddenDetailsV0(phaseArtifactRegisteredTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validatePhaseArtifactRegisteredPayloadSizeV0(payload)
}

func validateRegisterPhaseArtifactRequiredV0(payload RegisterPhaseArtifactCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"artifact_ref": payload.ArtifactRef,
		"phase_id":     payload.PhaseID,
		"agent_ref":    payload.AgentRef,
		"summary":      payload.Summary,
	}); err != nil {
		return err
	}
	return validatePhaseArtifactCommandPhaseIDV0(payload.PhaseID)
}

func validatePhaseArtifactCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase == OrchestrationPhaseProgramacionV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validatePhaseArtifactEventPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase == OrchestrationPhaseProgramacionV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateRegisterPhaseArtifactPayloadSizeV0(payload RegisterPhaseArtifactCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxPhaseArtifactPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validatePhaseArtifactRegisteredPayloadSizeV0(payload PhaseArtifactRegisteredPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxPhaseArtifactPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func phaseArtifactStringsInvalidV0(values []string) bool {
	if len(values) > maxPhaseArtifactEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxPhaseArtifactStringV0 {
			return true
		}
	}
	return false
}

func phaseArtifactHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxPhaseArtifactStringV0 {
			return true
		}
	}
	return false
}

func phaseArtifactHasForbiddenDetailsV0(values []string) bool {
	return textValuesContainForbiddenOperationalSensitiveDetailV0(values)
}

func registerPhaseArtifactTextFieldsV0(payload RegisterPhaseArtifactCommandPayloadV0) []string {
	values := []string{payload.ArtifactRef, payload.PhaseID, payload.AgentRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func phaseArtifactRegisteredTextFieldsV0(payload PhaseArtifactRegisteredPayloadV0) []string {
	values := []string{payload.ArtifactRef, payload.PhaseID, payload.AgentRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
