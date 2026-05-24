package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxVoteRequestPayloadBytesV0 = 2048
	maxVoteRequestStringV0       = 600
	maxVoteRequestEvidenceRefsV0 = 20
)

type RequestVoteCommandPayloadV0 struct {
	VoteRequestID              string                                `json:"vote_request_id"`
	PhaseID                    string                                `json:"phase_id"`
	DecisionTopicRef           string                                `json:"decision_topic_ref"`
	BrainstormRef              string                                `json:"brainstorm_ref"`
	Summary                    string                                `json:"summary"`
	MinimumRecommendedCapacity OrchestrationCapacityRecommendationV0 `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string                              `json:"evidence_refs,omitempty"`
}

type VoteRequestedPayloadV0 struct {
	VoteRequestID              string                                `json:"vote_request_id"`
	PhaseID                    string                                `json:"phase_id"`
	DecisionTopicRef           string                                `json:"decision_topic_ref"`
	BrainstormRef              string                                `json:"brainstorm_ref"`
	Summary                    string                                `json:"summary"`
	MinimumRecommendedCapacity OrchestrationCapacityRecommendationV0 `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string                              `json:"evidence_refs,omitempty"`
}

func NewRequestVoteCommandV0(meta OrchestrationCommandMetaV0, payload RequestVoteCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRequestVoteV0, normalizeRequestVotePayloadV0(payload))
}

func NewVoteRequestedEventV0(meta OrchestrationEventMetaV0, payload VoteRequestedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventVoteRequestedV0, normalizeVoteRequestedPayloadV0(payload))
}

func decodeRequestVoteCommandPayloadV0(raw json.RawMessage) (RequestVoteCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxVoteRequestPayloadBytesV0 {
		return RequestVoteCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RequestVoteCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RequestVoteCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRequestVotePayloadV0(payload)
	if err := validateRequestVoteCommandPayloadDataV0(payload); err != nil {
		return RequestVoteCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateVoteRequestedPayloadV0(event OrchestrationEventV0) error {
	var payload VoteRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateVoteRequestedPayloadDataV0(normalizeVoteRequestedPayloadV0(payload))
}

func handleRequestVoteCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRequestVoteCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureVoteCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return emptyCommandResultV0(), err
	}
	if voteAlreadyReflectedV0(current, payload.VoteRequestID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventVoteRequestedV0, payload.VoteRequestID, voteRequestedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewVoteRequestedEventV0(commandEventMetaV0(current, command, OrchestrationEventVoteRequestedV0), voteRequestedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyVoteRequestedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload VoteRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	payload = normalizeVoteRequestedPayloadV0(payload)
	if err := ensureVoteEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.VoteRequestID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Votes = appendUniqueCompactRefV0(next.Votes, payload.VoteRequestID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.VoteRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func voteRequestedPayloadFromCommandV0(payload RequestVoteCommandPayloadV0) VoteRequestedPayloadV0 {
	return VoteRequestedPayloadV0{
		VoteRequestID:              payload.VoteRequestID,
		PhaseID:                    payload.PhaseID,
		DecisionTopicRef:           payload.DecisionTopicRef,
		BrainstormRef:              payload.BrainstormRef,
		Summary:                    payload.Summary,
		MinimumRecommendedCapacity: payload.MinimumRecommendedCapacity,
		EvidenceRefs:               cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRequestVotePayloadV0(payload RequestVoteCommandPayloadV0) RequestVoteCommandPayloadV0 {
	return RequestVoteCommandPayloadV0{
		VoteRequestID:              strings.TrimSpace(payload.VoteRequestID),
		PhaseID:                    strings.TrimSpace(payload.PhaseID),
		DecisionTopicRef:           strings.TrimSpace(payload.DecisionTopicRef),
		BrainstormRef:              strings.TrimSpace(payload.BrainstormRef),
		Summary:                    strings.TrimSpace(payload.Summary),
		MinimumRecommendedCapacity: OrchestrationCapacityRecommendationV0(strings.TrimSpace(string(payload.MinimumRecommendedCapacity))),
		EvidenceRefs:               compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeVoteRequestedPayloadV0(payload VoteRequestedPayloadV0) VoteRequestedPayloadV0 {
	normalized := normalizeRequestVotePayloadV0(RequestVoteCommandPayloadV0(payload))
	return voteRequestedPayloadFromCommandV0(normalized)
}

func ensureVoteCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureVoteEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func voteAlreadyReflectedV0(current OrchestrationRunV0, voteRequestID string) bool {
	voteRequestID = strings.TrimSpace(voteRequestID)
	for _, existing := range current.Votes {
		if strings.TrimSpace(existing) == voteRequestID {
			return true
		}
	}
	return false
}
