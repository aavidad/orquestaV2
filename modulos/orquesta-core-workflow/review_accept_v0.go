package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxReviewAcceptPayloadBytesV0 = 2048
	maxReviewAcceptStringV0       = 600
	maxReviewAcceptEvidenceRefsV0 = 20
)

type AcceptReviewCommandPayloadV0 struct {
	AcceptedReviewRef string   `json:"accepted_review_ref"`
	PhaseID           string   `json:"phase_id"`
	ReviewRequestID   string   `json:"review_request_id"`
	DeliveryRef       string   `json:"delivery_ref"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type ReviewAcceptedPayloadV0 struct {
	AcceptedReviewRef string   `json:"accepted_review_ref"`
	PhaseID           string   `json:"phase_id"`
	ReviewRequestID   string   `json:"review_request_id"`
	DeliveryRef       string   `json:"delivery_ref"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

func NewAcceptReviewCommandV0(meta OrchestrationCommandMetaV0, payload AcceptReviewCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandAcceptReviewV0, normalizeAcceptReviewPayloadV0(payload))
}

func NewReviewAcceptedEventV0(meta OrchestrationEventMetaV0, payload ReviewAcceptedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventReviewAcceptedV0, normalizeReviewAcceptedPayloadV0(payload))
}

func decodeAcceptReviewCommandPayloadV0(raw json.RawMessage) (AcceptReviewCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxReviewAcceptPayloadBytesV0 {
		return AcceptReviewCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload AcceptReviewCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return AcceptReviewCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeAcceptReviewPayloadV0(payload)
	if err := validateAcceptReviewCommandPayloadDataV0(payload); err != nil {
		return AcceptReviewCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateReviewAcceptedPayloadV0(event OrchestrationEventV0) error {
	var payload ReviewAcceptedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateReviewAcceptedPayloadDataV0(normalizeReviewAcceptedPayloadV0(payload))
}

func handleAcceptReviewCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeAcceptReviewCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureAcceptReviewCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	if acceptedReviewAlreadyReflectedV0(current, payload.AcceptedReviewRef) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventReviewAcceptedV0, payload.AcceptedReviewRef, reviewAcceptedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewReviewAcceptedEventV0(commandEventMetaV0(current, command, OrchestrationEventReviewAcceptedV0), reviewAcceptedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyReviewAcceptedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload ReviewAcceptedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeReviewAcceptedPayloadV0(payload)
	if err := ensureReviewAcceptedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.AcceptedReviewRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.AcceptedReviews = appendUniqueCompactRefV0(next.AcceptedReviews, payload.AcceptedReviewRef)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.AcceptedReviewRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func reviewAcceptedPayloadFromCommandV0(payload AcceptReviewCommandPayloadV0) ReviewAcceptedPayloadV0 {
	return ReviewAcceptedPayloadV0{
		AcceptedReviewRef: payload.AcceptedReviewRef,
		PhaseID:           payload.PhaseID,
		ReviewRequestID:   payload.ReviewRequestID,
		DeliveryRef:       payload.DeliveryRef,
		Summary:           payload.Summary,
		EvidenceRefs:      cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeAcceptReviewPayloadV0(payload AcceptReviewCommandPayloadV0) AcceptReviewCommandPayloadV0 {
	return AcceptReviewCommandPayloadV0{
		AcceptedReviewRef: strings.TrimSpace(payload.AcceptedReviewRef),
		PhaseID:           strings.TrimSpace(payload.PhaseID),
		ReviewRequestID:   strings.TrimSpace(payload.ReviewRequestID),
		DeliveryRef:       strings.TrimSpace(payload.DeliveryRef),
		Summary:           strings.TrimSpace(payload.Summary),
		EvidenceRefs:      compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeReviewAcceptedPayloadV0(payload ReviewAcceptedPayloadV0) ReviewAcceptedPayloadV0 {
	normalized := normalizeAcceptReviewPayloadV0(AcceptReviewCommandPayloadV0(payload))
	return reviewAcceptedPayloadFromCommandV0(normalized)
}

func acceptedReviewAlreadyReflectedV0(current OrchestrationRunV0, acceptedReviewRef string) bool {
	acceptedReviewRef = strings.TrimSpace(acceptedReviewRef)
	for _, existing := range current.AcceptedReviews {
		if strings.TrimSpace(existing) == acceptedReviewRef {
			return true
		}
	}
	return false
}
