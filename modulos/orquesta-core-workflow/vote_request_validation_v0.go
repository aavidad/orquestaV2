package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func validateRequestVoteCommandPayloadDataV0(payload RequestVoteCommandPayloadV0) error {
	if err := validateVoteRequestRequiredV0(payload); err != nil {
		return err
	}
	if voteRequestStringsInvalidV0(payload.EvidenceRefs) || voteRequestHasLongStringV0(voteRequestTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if voteRequestHasForbiddenDetailsV0(voteRequestTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateVoteRequestPayloadSizeV0(payload)
}

func validateVoteRequestedPayloadDataV0(payload VoteRequestedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"vote_request_id":    payload.VoteRequestID,
		"phase_id":           payload.PhaseID,
		"decision_topic_ref": payload.DecisionTopicRef,
		"brainstorm_ref":     payload.BrainstormRef,
		"summary":            payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateVoteRequestEventCommonV0(payload.PhaseID, payload.MinimumRecommendedCapacity); err != nil {
		return err
	}
	if voteRequestStringsInvalidV0(payload.EvidenceRefs) || voteRequestHasLongStringV0(voteRequestedTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if voteRequestHasForbiddenDetailsV0(voteRequestedTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateVoteRequestedPayloadSizeV0(payload)
}

func validateVoteRequestRequiredV0(payload RequestVoteCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"vote_request_id":    payload.VoteRequestID,
		"phase_id":           payload.PhaseID,
		"decision_topic_ref": payload.DecisionTopicRef,
		"brainstorm_ref":     payload.BrainstormRef,
		"summary":            payload.Summary,
	}); err != nil {
		return err
	}
	return validateVoteRequestCommandCommonV0(payload.PhaseID, payload.MinimumRecommendedCapacity)
}

func validateVoteRequestCommandCommonV0(phaseID string, capacity OrchestrationCapacityRecommendationV0) error {
	if err := validateVoteCommandPhaseIDV0(phaseID); err != nil {
		return err
	}
	if strings.TrimSpace(string(capacity)) != "" && !isSupportedCapacityRecommendationV0(capacity) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.minimum_recommended_capacity")
	}
	return nil
}

func validateVoteRequestEventCommonV0(phaseID string, capacity OrchestrationCapacityRecommendationV0) error {
	if err := validateVoteEventPhaseIDV0(phaseID); err != nil {
		return err
	}
	if strings.TrimSpace(string(capacity)) != "" && !isSupportedCapacityRecommendationV0(capacity) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.minimum_recommended_capacity")
	}
	return nil
}

func validateVoteCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseVotacionYDecisionV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateVoteEventPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseVotacionYDecisionV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateVoteRequestPayloadSizeV0(payload RequestVoteCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxVoteRequestPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateVoteRequestedPayloadSizeV0(payload VoteRequestedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxVoteRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func voteRequestStringsInvalidV0(values []string) bool {
	if len(values) > maxVoteRequestEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxVoteRequestStringV0 {
			return true
		}
	}
	return false
}

func voteRequestHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxVoteRequestStringV0 {
			return true
		}
	}
	return false
}

func voteRequestHasForbiddenDetailsV0(values []string) bool {
	return textValuesContainForbiddenOperationalSensitiveDetailV0(values)
}

func voteRequestTextFieldsV0(payload RequestVoteCommandPayloadV0) []string {
	values := []string{payload.VoteRequestID, payload.PhaseID, payload.DecisionTopicRef, payload.BrainstormRef, payload.Summary, string(payload.MinimumRecommendedCapacity)}
	return append(values, payload.EvidenceRefs...)
}

func voteRequestedTextFieldsV0(payload VoteRequestedPayloadV0) []string {
	values := []string{payload.VoteRequestID, payload.PhaseID, payload.DecisionTopicRef, payload.BrainstormRef, payload.Summary, string(payload.MinimumRecommendedCapacity)}
	return append(values, payload.EvidenceRefs...)
}
