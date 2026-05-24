package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxBrainstormRequestPayloadBytesV0 = 2048
	maxBrainstormRequestStringV0       = 600
	maxBrainstormRequestEvidenceRefsV0 = 20
)

type RequestBrainstormCommandPayloadV0 struct {
	BrainstormRequestID        string                                `json:"brainstorm_request_id"`
	PhaseID                    string                                `json:"phase_id"`
	TopicRef                   string                                `json:"topic_ref"`
	Summary                    string                                `json:"summary"`
	MinimumRecommendedCapacity OrchestrationCapacityRecommendationV0 `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string                              `json:"evidence_refs,omitempty"`
}

type BrainstormRequestedPayloadV0 struct {
	BrainstormRequestID        string                                `json:"brainstorm_request_id"`
	PhaseID                    string                                `json:"phase_id"`
	TopicRef                   string                                `json:"topic_ref"`
	Summary                    string                                `json:"summary"`
	MinimumRecommendedCapacity OrchestrationCapacityRecommendationV0 `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string                              `json:"evidence_refs,omitempty"`
}

func NewRequestBrainstormCommandV0(meta OrchestrationCommandMetaV0, payload RequestBrainstormCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRequestBrainstormV0, normalizeRequestBrainstormPayloadV0(payload))
}

func NewBrainstormRequestedEventV0(meta OrchestrationEventMetaV0, payload BrainstormRequestedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventBrainstormRequestedV0, normalizeBrainstormRequestedPayloadV0(payload))
}

func decodeRequestBrainstormCommandPayloadV0(raw json.RawMessage) (RequestBrainstormCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxBrainstormRequestPayloadBytesV0 {
		return RequestBrainstormCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RequestBrainstormCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RequestBrainstormCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRequestBrainstormPayloadV0(payload)
	if err := validateRequestBrainstormCommandPayloadDataV0(payload); err != nil {
		return RequestBrainstormCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateBrainstormRequestedPayloadV0(event OrchestrationEventV0) error {
	var payload BrainstormRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateBrainstormRequestedPayloadDataV0(normalizeBrainstormRequestedPayloadV0(payload))
}

func handleRequestBrainstormCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRequestBrainstormCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureBrainstormCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return emptyCommandResultV0(), err
	}
	if brainstormAlreadyReflectedV0(current, payload.BrainstormRequestID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventBrainstormRequestedV0, payload.BrainstormRequestID, brainstormRequestedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewBrainstormRequestedEventV0(commandEventMetaV0(current, command, OrchestrationEventBrainstormRequestedV0), brainstormRequestedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyBrainstormRequestedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload BrainstormRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	payload = normalizeBrainstormRequestedPayloadV0(payload)
	if err := ensureBrainstormEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.BrainstormRequestID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Brainstorms = appendUniqueCompactRefV0(next.Brainstorms, payload.BrainstormRequestID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.BrainstormRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func brainstormRequestedPayloadFromCommandV0(payload RequestBrainstormCommandPayloadV0) BrainstormRequestedPayloadV0 {
	return BrainstormRequestedPayloadV0{
		BrainstormRequestID:        payload.BrainstormRequestID,
		PhaseID:                    payload.PhaseID,
		TopicRef:                   payload.TopicRef,
		Summary:                    payload.Summary,
		MinimumRecommendedCapacity: payload.MinimumRecommendedCapacity,
		EvidenceRefs:               cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRequestBrainstormPayloadV0(payload RequestBrainstormCommandPayloadV0) RequestBrainstormCommandPayloadV0 {
	return RequestBrainstormCommandPayloadV0{
		BrainstormRequestID:        strings.TrimSpace(payload.BrainstormRequestID),
		PhaseID:                    strings.TrimSpace(payload.PhaseID),
		TopicRef:                   strings.TrimSpace(payload.TopicRef),
		Summary:                    strings.TrimSpace(payload.Summary),
		MinimumRecommendedCapacity: OrchestrationCapacityRecommendationV0(strings.TrimSpace(string(payload.MinimumRecommendedCapacity))),
		EvidenceRefs:               compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeBrainstormRequestedPayloadV0(payload BrainstormRequestedPayloadV0) BrainstormRequestedPayloadV0 {
	normalized := normalizeRequestBrainstormPayloadV0(RequestBrainstormCommandPayloadV0(payload))
	return brainstormRequestedPayloadFromCommandV0(normalized)
}

func ensureBrainstormCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureBrainstormEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func brainstormAlreadyReflectedV0(current OrchestrationRunV0, brainstormRequestID string) bool {
	brainstormRequestID = strings.TrimSpace(brainstormRequestID)
	for _, existing := range current.Brainstorms {
		if strings.TrimSpace(existing) == brainstormRequestID {
			return true
		}
	}
	return false
}
