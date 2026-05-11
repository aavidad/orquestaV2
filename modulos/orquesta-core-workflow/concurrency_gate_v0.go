package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxConcurrencyGatePayloadBytesV0 = 4096
	maxConcurrencyGateStringV0       = 800
	maxConcurrencyGateRefsV0         = 50
)

type ConcurrencyGateDecisionV0 string

const (
	ConcurrencyGateDecisionAllowRequestAgentV0 ConcurrencyGateDecisionV0 = "allow_request_agent"
	ConcurrencyGateDecisionBlockRequestAgentV0 ConcurrencyGateDecisionV0 = "block_request_agent"
	ConcurrencyGateDecisionAskDirectorV0       ConcurrencyGateDecisionV0 = "ask_director"
)

type RecordConcurrencyGateCommandPayloadV0 struct {
	RunRef           string                    `json:"run_ref"`
	GateRef          string                    `json:"gate_ref"`
	PlanRef          string                    `json:"plan_ref"`
	SubjectClaimRefs []string                  `json:"subject_claim_refs,omitempty"`
	ReadyClaimRefs   []string                  `json:"ready_claim_refs,omitempty"`
	BlockedClaimRefs []string                  `json:"blocked_claim_refs,omitempty"`
	ConflictRefs     []string                  `json:"conflict_refs,omitempty"`
	Decision         ConcurrencyGateDecisionV0 `json:"decision"`
	Summary          string                    `json:"summary"`
	EvidenceRefs     []string                  `json:"evidence_refs,omitempty"`
}

type ConcurrencyGateRecordedPayloadV0 = RecordConcurrencyGateCommandPayloadV0

func NewRecordConcurrencyGateCommandV0(meta OrchestrationCommandMetaV0, payload RecordConcurrencyGateCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRecordConcurrencyGateV0, normalizeRecordConcurrencyGatePayloadV0(payload))
}

func NewConcurrencyGateRecordedEventV0(meta OrchestrationEventMetaV0, payload ConcurrencyGateRecordedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventConcurrencyGateRecordedV0, normalizeConcurrencyGateRecordedPayloadV0(payload))
}

func decodeRecordConcurrencyGateCommandPayloadV0(raw json.RawMessage) (RecordConcurrencyGateCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxConcurrencyGatePayloadBytesV0 {
		return RecordConcurrencyGateCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RecordConcurrencyGateCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RecordConcurrencyGateCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRecordConcurrencyGatePayloadV0(payload)
	if err := validateRecordConcurrencyGatePayloadDataV0(payload); err != nil {
		return RecordConcurrencyGateCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateConcurrencyGateRecordedPayloadV0(event OrchestrationEventV0) error {
	var payload ConcurrencyGateRecordedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateConcurrencyGateRecordedPayloadDataV0(normalizeConcurrencyGateRecordedPayloadV0(payload))
}

func handleRecordConcurrencyGateCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRecordConcurrencyGateCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRecordConcurrencyGateCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	matches, conflicts := concurrencyGateMatchesPayloadV0(current, payload)
	if conflicts {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.gate_ref")
	}
	if matches {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventConcurrencyGateRecordedV0, payload.GateRef, concurrencyGateRecordedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewConcurrencyGateRecordedEventV0(commandEventMetaV0(current, command, OrchestrationEventConcurrencyGateRecordedV0), concurrencyGateRecordedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyConcurrencyGateRecordedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload ConcurrencyGateRecordedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeConcurrencyGateRecordedPayloadV0(payload)
	if err := ensureConcurrencyGateRecordedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	matches, conflicts := concurrencyGateEventMatchesPayloadV0(current, payload)
	if conflicts {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.gate_ref")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.GateRef); err != nil {
		return current, err
	}
	next := cloneRunForReducerV0(current)
	if !matches {
		next.ConcurrencyGates = appendUniqueCompactRefV0(cloneStringsV0(next.ConcurrencyGates), concurrencyGateProjectionRefV0(payload))
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.GateRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func concurrencyGateRecordedPayloadFromCommandV0(payload RecordConcurrencyGateCommandPayloadV0) ConcurrencyGateRecordedPayloadV0 {
	return ConcurrencyGateRecordedPayloadV0{
		RunRef:           payload.RunRef,
		GateRef:          payload.GateRef,
		PlanRef:          payload.PlanRef,
		SubjectClaimRefs: cloneStringsV0(payload.SubjectClaimRefs),
		ReadyClaimRefs:   cloneStringsV0(payload.ReadyClaimRefs),
		BlockedClaimRefs: cloneStringsV0(payload.BlockedClaimRefs),
		ConflictRefs:     cloneStringsV0(payload.ConflictRefs),
		Decision:         payload.Decision,
		Summary:          payload.Summary,
		EvidenceRefs:     cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRecordConcurrencyGatePayloadV0(payload RecordConcurrencyGateCommandPayloadV0) RecordConcurrencyGateCommandPayloadV0 {
	return RecordConcurrencyGateCommandPayloadV0{
		RunRef:           strings.TrimSpace(payload.RunRef),
		GateRef:          strings.TrimSpace(payload.GateRef),
		PlanRef:          strings.TrimSpace(payload.PlanRef),
		SubjectClaimRefs: compactUniqueStringsV0(payload.SubjectClaimRefs),
		ReadyClaimRefs:   compactUniqueStringsV0(payload.ReadyClaimRefs),
		BlockedClaimRefs: compactUniqueStringsV0(payload.BlockedClaimRefs),
		ConflictRefs:     compactUniqueStringsV0(payload.ConflictRefs),
		Decision:         ConcurrencyGateDecisionV0(strings.TrimSpace(string(payload.Decision))),
		Summary:          strings.TrimSpace(payload.Summary),
		EvidenceRefs:     compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeConcurrencyGateRecordedPayloadV0(payload ConcurrencyGateRecordedPayloadV0) ConcurrencyGateRecordedPayloadV0 {
	normalized := normalizeRecordConcurrencyGatePayloadV0(RecordConcurrencyGateCommandPayloadV0(payload))
	return concurrencyGateRecordedPayloadFromCommandV0(normalized)
}
