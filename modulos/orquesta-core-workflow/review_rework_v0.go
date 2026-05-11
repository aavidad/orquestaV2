package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxReviewReworkPayloadBytesV0 = 2048
	maxReviewReworkStringV0       = 600
	maxReviewReworkEvidenceRefsV0 = 20
)

type RequestReworkCommandPayloadV0 struct {
	ReworkRequestRef string   `json:"rework_request_ref"`
	PhaseID          string   `json:"phase_id"`
	ReviewResultRef  string   `json:"review_result_ref"`
	ReviewRequestID  string   `json:"review_request_id"`
	DeliveryRef      string   `json:"delivery_ref"`
	Summary          string   `json:"summary"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

type ReworkRequestedPayloadV0 struct {
	ReworkRequestRef string   `json:"rework_request_ref"`
	PhaseID          string   `json:"phase_id"`
	ReviewResultRef  string   `json:"review_result_ref"`
	ReviewRequestID  string   `json:"review_request_id"`
	DeliveryRef      string   `json:"delivery_ref"`
	Summary          string   `json:"summary"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

func NewRequestReworkCommandV0(meta OrchestrationCommandMetaV0, payload RequestReworkCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRequestReworkV0, normalizeRequestReworkPayloadV0(payload))
}

func NewReworkRequestedEventV0(meta OrchestrationEventMetaV0, payload ReworkRequestedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventReworkRequestedV0, normalizeReworkRequestedPayloadV0(payload))
}

func decodeRequestReworkCommandPayloadV0(raw json.RawMessage) (RequestReworkCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxReviewReworkPayloadBytesV0 {
		return RequestReworkCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RequestReworkCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RequestReworkCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRequestReworkPayloadV0(payload)
	if err := validateRequestReworkCommandPayloadDataV0(payload); err != nil {
		return RequestReworkCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateReworkRequestedPayloadV0(event OrchestrationEventV0) error {
	var payload ReworkRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateReworkRequestedPayloadDataV0(normalizeReworkRequestedPayloadV0(payload))
}

func handleRequestReworkCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRequestReworkCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRequestReworkCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	matches, conflicts := reworkRequestMatchesPayloadV0(current, payload)
	if conflicts {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.rework_request_ref")
	}
	if matches {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventReworkRequestedV0, payload.ReworkRequestRef, reworkRequestedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewReworkRequestedEventV0(commandEventMetaV0(current, command, OrchestrationEventReworkRequestedV0), reworkRequestedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyReworkRequestedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload ReworkRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeReworkRequestedPayloadV0(payload)
	if err := ensureReworkRequestedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	matches, conflicts := reworkRequestEventMatchesPayloadV0(current, payload)
	if conflicts {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.rework_request_ref")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.ReworkRequestRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	if !matches {
		next.ReworkRequests = appendUniqueCompactRefV0(cloneStringsV0(next.ReworkRequests), reworkRequestProjectionRefV0(payload))
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.ReworkRequestRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func reworkRequestedPayloadFromCommandV0(payload RequestReworkCommandPayloadV0) ReworkRequestedPayloadV0 {
	return ReworkRequestedPayloadV0{
		ReworkRequestRef: payload.ReworkRequestRef,
		PhaseID:          payload.PhaseID,
		ReviewResultRef:  payload.ReviewResultRef,
		ReviewRequestID:  payload.ReviewRequestID,
		DeliveryRef:      payload.DeliveryRef,
		Summary:          payload.Summary,
		EvidenceRefs:     cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRequestReworkPayloadV0(payload RequestReworkCommandPayloadV0) RequestReworkCommandPayloadV0 {
	return RequestReworkCommandPayloadV0{
		ReworkRequestRef: strings.TrimSpace(payload.ReworkRequestRef),
		PhaseID:          strings.TrimSpace(payload.PhaseID),
		ReviewResultRef:  strings.TrimSpace(payload.ReviewResultRef),
		ReviewRequestID:  strings.TrimSpace(payload.ReviewRequestID),
		DeliveryRef:      strings.TrimSpace(payload.DeliveryRef),
		Summary:          strings.TrimSpace(payload.Summary),
		EvidenceRefs:     compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeReworkRequestedPayloadV0(payload ReworkRequestedPayloadV0) ReworkRequestedPayloadV0 {
	normalized := normalizeRequestReworkPayloadV0(RequestReworkCommandPayloadV0(payload))
	return reworkRequestedPayloadFromCommandV0(normalized)
}
