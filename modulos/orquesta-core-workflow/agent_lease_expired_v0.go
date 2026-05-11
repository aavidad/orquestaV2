package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxAgentLeaseExpiredPayloadBytesV0 = 2048
	maxAgentLeaseExpiredStringV0       = 600
	maxAgentLeaseExpiredEvidenceRefsV0 = 20
)

type AgentLeaseRecommendedActionV0 string

const (
	AgentLeaseActionRetryV0       AgentLeaseRecommendedActionV0 = "retry"
	AgentLeaseActionAskDirectorV0 AgentLeaseRecommendedActionV0 = "ask_director"
	AgentLeaseActionStopAgentV0   AgentLeaseRecommendedActionV0 = "stop_agent"
	AgentLeaseActionMarkFailedV0  AgentLeaseRecommendedActionV0 = "mark_failed"
	AgentLeaseActionMarkStoppedV0 AgentLeaseRecommendedActionV0 = "mark_stopped"
	AgentLeaseActionReplanTaskV0  AgentLeaseRecommendedActionV0 = "replan_task"
	AgentLeaseActionAlertOnlyV0   AgentLeaseRecommendedActionV0 = "alert_only"
)

type RegisterAgentLeaseExpiredCommandPayloadV0 struct {
	RunRef            string                        `json:"run_ref"`
	AgentRequestID    string                        `json:"agent_request_id"`
	LeaseRef          string                        `json:"lease_ref"`
	ReasonCode        string                        `json:"reason_code"`
	ObservedAt        string                        `json:"observed_at"`
	RecommendedAction AgentLeaseRecommendedActionV0 `json:"recommended_action"`
	EvidenceRefs      []string                      `json:"evidence_refs,omitempty"`
}

type AgentLeaseExpiredPayloadV0 = RegisterAgentLeaseExpiredCommandPayloadV0

func NewRegisterAgentLeaseExpiredCommandV0(meta OrchestrationCommandMetaV0, payload RegisterAgentLeaseExpiredCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRegisterAgentLeaseExpiredV0, normalizeRegisterAgentLeaseExpiredPayloadV0(payload))
}

func NewAgentLeaseExpiredEventV0(meta OrchestrationEventMetaV0, payload AgentLeaseExpiredPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventAgentLeaseExpiredV0, normalizeAgentLeaseExpiredPayloadV0(payload))
}

func decodeRegisterAgentLeaseExpiredCommandPayloadV0(raw json.RawMessage) (RegisterAgentLeaseExpiredCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxAgentLeaseExpiredPayloadBytesV0 {
		return RegisterAgentLeaseExpiredCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RegisterAgentLeaseExpiredCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RegisterAgentLeaseExpiredCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRegisterAgentLeaseExpiredPayloadV0(payload)
	if err := validateRegisterAgentLeaseExpiredPayloadDataV0(payload); err != nil {
		return RegisterAgentLeaseExpiredCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateAgentLeaseExpiredPayloadV0(event OrchestrationEventV0) error {
	var payload AgentLeaseExpiredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateAgentLeaseExpiredPayloadDataV0(normalizeAgentLeaseExpiredPayloadV0(payload))
}

func handleRegisterAgentLeaseExpiredCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRegisterAgentLeaseExpiredCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRegisterAgentLeaseExpiredCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	matches, conflicts := agentLeaseExpiredMatchesPayloadV0(current, payload)
	if conflicts {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.lease_ref")
	}
	if matches {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventAgentLeaseExpiredV0, payload.LeaseRef, agentLeaseExpiredPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewAgentLeaseExpiredEventV0(commandEventMetaV0(current, command, OrchestrationEventAgentLeaseExpiredV0), agentLeaseExpiredPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyAgentLeaseExpiredEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload AgentLeaseExpiredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeAgentLeaseExpiredPayloadV0(payload)
	if err := ensureAgentLeaseExpiredEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	matches, conflicts := agentLeaseExpiredEventMatchesPayloadV0(current, payload)
	if conflicts {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.lease_ref")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.LeaseRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	if !matches {
		next.AgentLeaseExpirations = appendUniqueCompactRefV0(cloneStringsV0(next.AgentLeaseExpirations), agentLeaseExpiredProjectionRefV0(payload))
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.LeaseRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func agentLeaseExpiredPayloadFromCommandV0(payload RegisterAgentLeaseExpiredCommandPayloadV0) AgentLeaseExpiredPayloadV0 {
	return AgentLeaseExpiredPayloadV0{
		RunRef:            payload.RunRef,
		AgentRequestID:    payload.AgentRequestID,
		LeaseRef:          payload.LeaseRef,
		ReasonCode:        payload.ReasonCode,
		ObservedAt:        payload.ObservedAt,
		RecommendedAction: payload.RecommendedAction,
		EvidenceRefs:      cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRegisterAgentLeaseExpiredPayloadV0(payload RegisterAgentLeaseExpiredCommandPayloadV0) RegisterAgentLeaseExpiredCommandPayloadV0 {
	return RegisterAgentLeaseExpiredCommandPayloadV0{
		RunRef:            strings.TrimSpace(payload.RunRef),
		AgentRequestID:    strings.TrimSpace(payload.AgentRequestID),
		LeaseRef:          strings.TrimSpace(payload.LeaseRef),
		ReasonCode:        strings.TrimSpace(payload.ReasonCode),
		ObservedAt:        strings.TrimSpace(payload.ObservedAt),
		RecommendedAction: AgentLeaseRecommendedActionV0(strings.TrimSpace(string(payload.RecommendedAction))),
		EvidenceRefs:      compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeAgentLeaseExpiredPayloadV0(payload AgentLeaseExpiredPayloadV0) AgentLeaseExpiredPayloadV0 {
	normalized := normalizeRegisterAgentLeaseExpiredPayloadV0(RegisterAgentLeaseExpiredCommandPayloadV0(payload))
	return agentLeaseExpiredPayloadFromCommandV0(normalized)
}
