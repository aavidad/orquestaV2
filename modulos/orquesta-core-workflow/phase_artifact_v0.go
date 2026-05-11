package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxPhaseArtifactPayloadBytesV0 = 2048
	maxPhaseArtifactStringV0       = 600
	maxPhaseArtifactEvidenceRefsV0 = 20
)

type RegisterPhaseArtifactCommandPayloadV0 struct {
	ArtifactRef  string   `json:"artifact_ref"`
	PhaseID      string   `json:"phase_id"`
	AgentRef     string   `json:"agent_ref"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type PhaseArtifactRegisteredPayloadV0 RegisterPhaseArtifactCommandPayloadV0

func NewRegisterPhaseArtifactCommandV0(meta OrchestrationCommandMetaV0, payload RegisterPhaseArtifactCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRegisterPhaseArtifactV0, normalizeRegisterPhaseArtifactPayloadV0(payload))
}

func NewPhaseArtifactRegisteredEventV0(meta OrchestrationEventMetaV0, payload PhaseArtifactRegisteredPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventPhaseArtifactRegisteredV0, normalizePhaseArtifactRegisteredPayloadV0(payload))
}

func decodeRegisterPhaseArtifactCommandPayloadV0(raw json.RawMessage) (RegisterPhaseArtifactCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxPhaseArtifactPayloadBytesV0 {
		return RegisterPhaseArtifactCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RegisterPhaseArtifactCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RegisterPhaseArtifactCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRegisterPhaseArtifactPayloadV0(payload)
	if err := validateRegisterPhaseArtifactCommandPayloadDataV0(payload); err != nil {
		return RegisterPhaseArtifactCommandPayloadV0{}, err
	}
	return payload, nil
}

func validatePhaseArtifactRegisteredPayloadV0(event OrchestrationEventV0) error {
	var payload PhaseArtifactRegisteredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validatePhaseArtifactRegisteredPayloadDataV0(normalizePhaseArtifactRegisteredPayloadV0(payload))
}

func handleRegisterPhaseArtifactCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRegisterPhaseArtifactCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRegisterPhaseArtifactCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	matches, conflicts := phaseArtifactMatchesPayloadV0(current, payload)
	if conflicts {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload")
	}
	if matches {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventPhaseArtifactRegisteredV0, payload.ArtifactRef, phaseArtifactRegisteredPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewPhaseArtifactRegisteredEventV0(commandEventMetaV0(current, command, OrchestrationEventPhaseArtifactRegisteredV0), phaseArtifactRegisteredPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyPhaseArtifactRegisteredEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload PhaseArtifactRegisteredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizePhaseArtifactRegisteredPayloadV0(payload)
	if err := ensurePhaseArtifactRegisteredEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	matches, conflicts := phaseArtifactEventMatchesPayloadV0(current, payload)
	if conflicts {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.ArtifactRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	if !matches {
		next.PhaseArtifacts = appendUniqueCompactRefV0(next.PhaseArtifacts, phaseArtifactProjectionRefV0(payload))
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.ArtifactRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func phaseArtifactRegisteredPayloadFromCommandV0(payload RegisterPhaseArtifactCommandPayloadV0) PhaseArtifactRegisteredPayloadV0 {
	return PhaseArtifactRegisteredPayloadV0{
		ArtifactRef:  payload.ArtifactRef,
		PhaseID:      payload.PhaseID,
		AgentRef:     payload.AgentRef,
		Summary:      payload.Summary,
		EvidenceRefs: cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRegisterPhaseArtifactPayloadV0(payload RegisterPhaseArtifactCommandPayloadV0) RegisterPhaseArtifactCommandPayloadV0 {
	return RegisterPhaseArtifactCommandPayloadV0{
		ArtifactRef:  strings.TrimSpace(payload.ArtifactRef),
		PhaseID:      strings.TrimSpace(payload.PhaseID),
		AgentRef:     strings.TrimSpace(payload.AgentRef),
		Summary:      strings.TrimSpace(payload.Summary),
		EvidenceRefs: compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizePhaseArtifactRegisteredPayloadV0(payload PhaseArtifactRegisteredPayloadV0) PhaseArtifactRegisteredPayloadV0 {
	normalized := normalizeRegisterPhaseArtifactPayloadV0(RegisterPhaseArtifactCommandPayloadV0(payload))
	return phaseArtifactRegisteredPayloadFromCommandV0(normalized)
}
