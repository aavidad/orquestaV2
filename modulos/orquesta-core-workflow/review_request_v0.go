package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxReviewRequestPayloadBytesV0 = 2048
	maxReviewRequestStringV0       = 600
	maxReviewRequestEvidenceRefsV0 = 20
)

type RequestReviewCommandPayloadV0 struct {
	ReviewRequestID string   `json:"review_request_id"`
	PhaseID         string   `json:"phase_id"`
	DeliveryRef     string   `json:"delivery_ref"`
	Summary         string   `json:"summary"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type ReviewRequestedPayloadV0 struct {
	ReviewRequestID string   `json:"review_request_id"`
	PhaseID         string   `json:"phase_id"`
	DeliveryRef     string   `json:"delivery_ref"`
	Summary         string   `json:"summary"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

func NewRequestReviewCommandV0(meta OrchestrationCommandMetaV0, payload RequestReviewCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRequestReviewV0, normalizeRequestReviewPayloadV0(payload))
}

func NewReviewRequestedEventV0(meta OrchestrationEventMetaV0, payload ReviewRequestedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventReviewRequestedV0, normalizeReviewRequestedPayloadV0(payload))
}

func decodeRequestReviewCommandPayloadV0(raw json.RawMessage) (RequestReviewCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxReviewRequestPayloadBytesV0 {
		return RequestReviewCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RequestReviewCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RequestReviewCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRequestReviewPayloadV0(payload)
	if err := validateRequestReviewCommandPayloadDataV0(payload); err != nil {
		return RequestReviewCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateReviewRequestedPayloadV0(event OrchestrationEventV0) error {
	var payload ReviewRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateReviewRequestedPayloadDataV0(normalizeReviewRequestedPayloadV0(payload))
}

func handleRequestReviewCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRequestReviewCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRequestReviewCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	if reviewRequestAlreadyReflectedV0(current, payload.ReviewRequestID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventReviewRequestedV0, payload.ReviewRequestID, reviewRequestedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewReviewRequestedEventV0(commandEventMetaV0(current, command, OrchestrationEventReviewRequestedV0), reviewRequestedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyReviewRequestedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload ReviewRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeReviewRequestedPayloadV0(payload)
	if err := ensureReviewRequestedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.ReviewRequestID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Reviews = appendUniqueCompactRefV0(next.Reviews, payload.ReviewRequestID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.ReviewRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func reviewRequestedPayloadFromCommandV0(payload RequestReviewCommandPayloadV0) ReviewRequestedPayloadV0 {
	return ReviewRequestedPayloadV0{
		ReviewRequestID: payload.ReviewRequestID,
		PhaseID:         payload.PhaseID,
		DeliveryRef:     payload.DeliveryRef,
		Summary:         payload.Summary,
		EvidenceRefs:    cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRequestReviewPayloadV0(payload RequestReviewCommandPayloadV0) RequestReviewCommandPayloadV0 {
	return RequestReviewCommandPayloadV0{
		ReviewRequestID: strings.TrimSpace(payload.ReviewRequestID),
		PhaseID:         strings.TrimSpace(payload.PhaseID),
		DeliveryRef:     strings.TrimSpace(payload.DeliveryRef),
		Summary:         strings.TrimSpace(payload.Summary),
		EvidenceRefs:    compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeReviewRequestedPayloadV0(payload ReviewRequestedPayloadV0) ReviewRequestedPayloadV0 {
	normalized := normalizeRequestReviewPayloadV0(RequestReviewCommandPayloadV0(payload))
	return reviewRequestedPayloadFromCommandV0(normalized)
}

func reviewRequestAlreadyReflectedV0(current OrchestrationRunV0, reviewRequestID string) bool {
	reviewRequestID = strings.TrimSpace(reviewRequestID)
	for _, existing := range current.Reviews {
		if strings.TrimSpace(existing) == reviewRequestID {
			return true
		}
	}
	return false
}
