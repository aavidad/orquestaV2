package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxFunctionContractPublishPayloadBytesV0 = 2048
	maxFunctionContractPublishStringV0       = 600
	maxFunctionContractPublishRefsV0         = 20
)

type PublishFunctionContractCommandPayloadV0 struct {
	ContractRef   string   `json:"contract_ref"`
	PhaseID       string   `json:"phase_id"`
	DecisionRef   string   `json:"decision_ref"`
	Summary       string   `json:"summary"`
	FunctionNames []string `json:"function_names,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type FunctionContractPublishedPayloadV0 struct {
	ContractRef   string   `json:"contract_ref"`
	PhaseID       string   `json:"phase_id"`
	DecisionRef   string   `json:"decision_ref"`
	Summary       string   `json:"summary"`
	FunctionNames []string `json:"function_names,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

func NewPublishFunctionContractCommandV0(meta OrchestrationCommandMetaV0, payload PublishFunctionContractCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandPublishFunctionContractV0, normalizePublishFunctionContractPayloadV0(payload))
}

func NewFunctionContractPublishedEventV0(meta OrchestrationEventMetaV0, payload FunctionContractPublishedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventFunctionContractPublishedV0, normalizeFunctionContractPublishedPayloadV0(payload))
}

func decodePublishFunctionContractCommandPayloadV0(raw json.RawMessage) (PublishFunctionContractCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxFunctionContractPublishPayloadBytesV0 {
		return PublishFunctionContractCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload PublishFunctionContractCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return PublishFunctionContractCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizePublishFunctionContractPayloadV0(payload)
	if err := validatePublishFunctionContractCommandPayloadDataV0(payload); err != nil {
		return PublishFunctionContractCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateFunctionContractPublishedPayloadV0(event OrchestrationEventV0) error {
	var payload FunctionContractPublishedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateFunctionContractPublishedPayloadDataV0(normalizeFunctionContractPublishedPayloadV0(payload))
}

func handlePublishFunctionContractCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodePublishFunctionContractCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureFunctionContractPublishCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	if functionContractAlreadyReflectedV0(current, payload.ContractRef) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventFunctionContractPublishedV0, payload.ContractRef, functionContractPublishedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewFunctionContractPublishedEventV0(commandEventMetaV0(current, command, OrchestrationEventFunctionContractPublishedV0), functionContractPublishedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyFunctionContractPublishedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload FunctionContractPublishedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeFunctionContractPublishedPayloadV0(payload)
	if err := ensureFunctionContractPublishedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.ContractRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.FunctionContracts = appendUniqueCompactRefV0(next.FunctionContracts, payload.ContractRef)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.ContractRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func functionContractPublishedPayloadFromCommandV0(payload PublishFunctionContractCommandPayloadV0) FunctionContractPublishedPayloadV0 {
	return FunctionContractPublishedPayloadV0{
		ContractRef:   payload.ContractRef,
		PhaseID:       payload.PhaseID,
		DecisionRef:   payload.DecisionRef,
		Summary:       payload.Summary,
		FunctionNames: cloneStringsV0(payload.FunctionNames),
		EvidenceRefs:  cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizePublishFunctionContractPayloadV0(payload PublishFunctionContractCommandPayloadV0) PublishFunctionContractCommandPayloadV0 {
	return PublishFunctionContractCommandPayloadV0{
		ContractRef:   strings.TrimSpace(payload.ContractRef),
		PhaseID:       strings.TrimSpace(payload.PhaseID),
		DecisionRef:   strings.TrimSpace(payload.DecisionRef),
		Summary:       strings.TrimSpace(payload.Summary),
		FunctionNames: compactUniqueStringsV0(payload.FunctionNames),
		EvidenceRefs:  compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeFunctionContractPublishedPayloadV0(payload FunctionContractPublishedPayloadV0) FunctionContractPublishedPayloadV0 {
	normalized := normalizePublishFunctionContractPayloadV0(PublishFunctionContractCommandPayloadV0(payload))
	return functionContractPublishedPayloadFromCommandV0(normalized)
}

func functionContractAlreadyReflectedV0(current OrchestrationRunV0, contractRef string) bool {
	contractRef = strings.TrimSpace(contractRef)
	for _, existing := range current.FunctionContracts {
		if strings.TrimSpace(existing) == contractRef {
			return true
		}
	}
	return false
}
