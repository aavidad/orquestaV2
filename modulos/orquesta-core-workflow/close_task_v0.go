package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxCloseTaskPayloadBytesV0 = 2048
	maxCloseTaskStringV0       = 600
	maxCloseTaskEvidenceRefsV0 = 20
)

type CloseTaskCommandPayloadV0 struct {
	TaskID            string   `json:"task_id"`
	PhaseID           string   `json:"phase_id"`
	DeliveryRef       string   `json:"delivery_ref"`
	AcceptedReviewRef string   `json:"accepted_review_ref"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type TaskClosedPayloadV0 struct {
	TaskID            string   `json:"task_id"`
	PhaseID           string   `json:"phase_id"`
	DeliveryRef       string   `json:"delivery_ref"`
	AcceptedReviewRef string   `json:"accepted_review_ref"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

func NewCloseTaskCommandV0(meta OrchestrationCommandMetaV0, payload CloseTaskCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandCloseTaskV0, normalizeCloseTaskPayloadV0(payload))
}

func NewTaskClosedEventV0(meta OrchestrationEventMetaV0, payload TaskClosedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventTaskClosedV0, normalizeTaskClosedPayloadV0(payload))
}

func decodeCloseTaskCommandPayloadV0(raw json.RawMessage) (CloseTaskCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxCloseTaskPayloadBytesV0 {
		return CloseTaskCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload CloseTaskCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return CloseTaskCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeCloseTaskPayloadV0(payload)
	if err := validateCloseTaskCommandPayloadDataV0(payload); err != nil {
		return CloseTaskCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateTaskClosedPayloadV0(event OrchestrationEventV0) error {
	var payload TaskClosedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateTaskClosedPayloadDataV0(normalizeTaskClosedPayloadV0(payload))
}

func handleCloseTaskCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeCloseTaskCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureCloseTaskCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	if taskClosedAlreadyReflectedV0(current, payload.TaskID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventTaskClosedV0, payload.TaskID, taskClosedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewTaskClosedEventV0(commandEventMetaV0(current, command, OrchestrationEventTaskClosedV0), taskClosedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyTaskClosedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload TaskClosedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeTaskClosedPayloadV0(payload)
	if err := ensureTaskClosedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.TaskID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.ClosedTasks = appendUniqueCompactRefV0(next.ClosedTasks, payload.TaskID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.TaskID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func taskClosedPayloadFromCommandV0(payload CloseTaskCommandPayloadV0) TaskClosedPayloadV0 {
	return TaskClosedPayloadV0{
		TaskID:            payload.TaskID,
		PhaseID:           payload.PhaseID,
		DeliveryRef:       payload.DeliveryRef,
		AcceptedReviewRef: payload.AcceptedReviewRef,
		Summary:           payload.Summary,
		EvidenceRefs:      cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeCloseTaskPayloadV0(payload CloseTaskCommandPayloadV0) CloseTaskCommandPayloadV0 {
	return CloseTaskCommandPayloadV0{
		TaskID:            strings.TrimSpace(payload.TaskID),
		PhaseID:           strings.TrimSpace(payload.PhaseID),
		DeliveryRef:       strings.TrimSpace(payload.DeliveryRef),
		AcceptedReviewRef: strings.TrimSpace(payload.AcceptedReviewRef),
		Summary:           strings.TrimSpace(payload.Summary),
		EvidenceRefs:      compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeTaskClosedPayloadV0(payload TaskClosedPayloadV0) TaskClosedPayloadV0 {
	normalized := normalizeCloseTaskPayloadV0(CloseTaskCommandPayloadV0(payload))
	return taskClosedPayloadFromCommandV0(normalized)
}

func taskClosedAlreadyReflectedV0(current OrchestrationRunV0, taskID string) bool {
	taskID = strings.TrimSpace(taskID)
	for _, existing := range current.ClosedTasks {
		if strings.TrimSpace(existing) == taskID {
			return true
		}
	}
	return false
}
