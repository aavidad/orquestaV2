package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxAgentLostPayloadBytesV0 = 2048
	maxAgentLostStringV0       = 600
	maxAgentLostEvidenceRefsV0 = 20
)

type RegisterAgentLostCommandPayloadV0 struct {
	AgentRequestID string   `json:"agent_request_id"`
	LossRef        string   `json:"loss_ref"`
	ReasonCode     string   `json:"reason_code"`
	ObservedAt     string   `json:"observed_at"`
	Retryable      bool     `json:"retryable"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type AgentLostPayloadV0 RegisterAgentLostCommandPayloadV0

func NewRegisterAgentLostCommandV0(
	meta OrchestrationCommandMetaV0,
	payload RegisterAgentLostCommandPayloadV0,
) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(
		meta,
		OrchestrationCommandRegisterAgentLostV0,
		normalizeRegisterAgentLostPayloadV0(payload),
	)
}

func NewAgentLostEventV0(meta OrchestrationEventMetaV0, payload AgentLostPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventAgentLostV0, normalizeAgentLostPayloadV0(payload))
}

func decodeRegisterAgentLostCommandPayloadV0(raw json.RawMessage) (RegisterAgentLostCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxAgentLostPayloadBytesV0 {
		return RegisterAgentLostCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RegisterAgentLostCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RegisterAgentLostCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRegisterAgentLostPayloadV0(payload)
	if err := validateRegisterAgentLostPayloadDataV0(payload); err != nil {
		return RegisterAgentLostCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateAgentLostPayloadV0(event OrchestrationEventV0) error {
	var payload AgentLostPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateAgentLostPayloadDataV0(normalizeAgentLostPayloadV0(payload))
}

func handleRegisterAgentLostCommandV0(
	current OrchestrationRunV0,
	command OrchestrationCommandV0,
) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRegisterAgentLostCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRegisterAgentLostCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	if agentLostAlreadyReflectedV0(current, payload.AgentRequestID) {
		if err := ensureCommandEffectMatchesV0(
			current,
			command,
			OrchestrationEventAgentLostV0,
			payload.AgentRequestID,
			agentLostPayloadFromCommandV0(payload),
		); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewAgentLostEventV0(
		commandEventMetaV0(current, command, OrchestrationEventAgentLostV0),
		agentLostPayloadFromCommandV0(payload),
	)
	return eventCommandResultV0(event, err)
}

func applyAgentLostEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload AgentLostPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeAgentLostPayloadV0(payload)
	if err := ensureAgentLostEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.AgentRequestID); err != nil {
		return current, err
	}
	next := cloneRunForReducerV0(current)
	next.LostAgents = appendUniqueCompactRefV0(next.LostAgents, payload.AgentRequestID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.AgentRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func normalizeRegisterAgentLostPayloadV0(
	payload RegisterAgentLostCommandPayloadV0,
) RegisterAgentLostCommandPayloadV0 {
	return RegisterAgentLostCommandPayloadV0{
		AgentRequestID: strings.TrimSpace(payload.AgentRequestID),
		LossRef:        strings.TrimSpace(payload.LossRef),
		ReasonCode:     strings.TrimSpace(payload.ReasonCode),
		ObservedAt:     strings.TrimSpace(payload.ObservedAt),
		Retryable:      payload.Retryable,
		EvidenceRefs:   compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeAgentLostPayloadV0(payload AgentLostPayloadV0) AgentLostPayloadV0 {
	normalized := normalizeRegisterAgentLostPayloadV0(RegisterAgentLostCommandPayloadV0(payload))
	return agentLostPayloadFromCommandV0(normalized)
}

func agentLostPayloadFromCommandV0(payload RegisterAgentLostCommandPayloadV0) AgentLostPayloadV0 {
	return AgentLostPayloadV0{
		AgentRequestID: payload.AgentRequestID,
		LossRef:        payload.LossRef,
		ReasonCode:     payload.ReasonCode,
		ObservedAt:     payload.ObservedAt,
		Retryable:      payload.Retryable,
		EvidenceRefs:   cloneStringsV0(payload.EvidenceRefs),
	}
}

func agentLostAlreadyReflectedV0(current OrchestrationRunV0, agentRequestID string) bool {
	return compactRefInListV0(current.LostAgents, agentRequestID)
}
