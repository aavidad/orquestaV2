package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxDecisionAcceptPayloadBytesV0 = 2048
	maxDecisionAcceptStringV0       = 600
	maxDecisionAcceptEvidenceRefsV0 = 20
)

type AcceptDecisionCommandPayloadV0 struct {
	DecisionRef       string   `json:"decision_ref"`
	PhaseID           string   `json:"phase_id"`
	VoteRef           string   `json:"vote_ref"`
	AcceptedOptionRef string   `json:"accepted_option_ref"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type ArchitectureDecisionAcceptedPayloadV0 struct {
	DecisionRef       string   `json:"decision_ref"`
	PhaseID           string   `json:"phase_id"`
	VoteRef           string   `json:"vote_ref"`
	AcceptedOptionRef string   `json:"accepted_option_ref"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

func NewAcceptDecisionCommandV0(meta OrchestrationCommandMetaV0, payload AcceptDecisionCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandAcceptDecisionV0, normalizeAcceptDecisionPayloadV0(payload))
}

func NewArchitectureDecisionAcceptedEventV0(meta OrchestrationEventMetaV0, payload ArchitectureDecisionAcceptedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventArchitectureDecisionAcceptedV0, normalizeArchitectureDecisionAcceptedPayloadV0(payload))
}

func decodeAcceptDecisionCommandPayloadV0(raw json.RawMessage) (AcceptDecisionCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxDecisionAcceptPayloadBytesV0 {
		return AcceptDecisionCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload AcceptDecisionCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return AcceptDecisionCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeAcceptDecisionPayloadV0(payload)
	if err := validateAcceptDecisionCommandPayloadDataV0(payload); err != nil {
		return AcceptDecisionCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateArchitectureDecisionAcceptedPayloadV0(event OrchestrationEventV0) error {
	var payload ArchitectureDecisionAcceptedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateArchitectureDecisionAcceptedPayloadDataV0(normalizeArchitectureDecisionAcceptedPayloadV0(payload))
}

func handleAcceptDecisionCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeAcceptDecisionCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureAcceptDecisionCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return emptyCommandResultV0(), err
	}
	if !voteRefAlreadyReflectedV0(current, payload.VoteRef) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.vote_ref")
	}
	if decisionAlreadyReflectedV0(current, payload.DecisionRef) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventArchitectureDecisionAcceptedV0, payload.DecisionRef, architectureDecisionAcceptedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewArchitectureDecisionAcceptedEventV0(commandEventMetaV0(current, command, OrchestrationEventArchitectureDecisionAcceptedV0), architectureDecisionAcceptedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyArchitectureDecisionAcceptedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload ArchitectureDecisionAcceptedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	payload = normalizeArchitectureDecisionAcceptedPayloadV0(payload)
	if err := ensureAcceptDecisionEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return current, err
	}
	if !voteRefAlreadyReflectedV0(current, payload.VoteRef) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.vote_ref")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.DecisionRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Decisions = appendUniqueCompactRefV0(next.Decisions, payload.DecisionRef)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.DecisionRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func architectureDecisionAcceptedPayloadFromCommandV0(payload AcceptDecisionCommandPayloadV0) ArchitectureDecisionAcceptedPayloadV0 {
	return ArchitectureDecisionAcceptedPayloadV0{
		DecisionRef:       payload.DecisionRef,
		PhaseID:           payload.PhaseID,
		VoteRef:           payload.VoteRef,
		AcceptedOptionRef: payload.AcceptedOptionRef,
		Summary:           payload.Summary,
		EvidenceRefs:      cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeAcceptDecisionPayloadV0(payload AcceptDecisionCommandPayloadV0) AcceptDecisionCommandPayloadV0 {
	return AcceptDecisionCommandPayloadV0{
		DecisionRef:       strings.TrimSpace(payload.DecisionRef),
		PhaseID:           strings.TrimSpace(payload.PhaseID),
		VoteRef:           strings.TrimSpace(payload.VoteRef),
		AcceptedOptionRef: strings.TrimSpace(payload.AcceptedOptionRef),
		Summary:           strings.TrimSpace(payload.Summary),
		EvidenceRefs:      compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeArchitectureDecisionAcceptedPayloadV0(payload ArchitectureDecisionAcceptedPayloadV0) ArchitectureDecisionAcceptedPayloadV0 {
	normalized := normalizeAcceptDecisionPayloadV0(AcceptDecisionCommandPayloadV0(payload))
	return architectureDecisionAcceptedPayloadFromCommandV0(normalized)
}

func ensureAcceptDecisionCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureAcceptDecisionEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func decisionAlreadyReflectedV0(current OrchestrationRunV0, decisionRef string) bool {
	decisionRef = strings.TrimSpace(decisionRef)
	for _, existing := range current.Decisions {
		if strings.TrimSpace(existing) == decisionRef {
			return true
		}
	}
	return false
}

func voteRefAlreadyReflectedV0(current OrchestrationRunV0, voteRef string) bool {
	voteRef = strings.TrimSpace(voteRef)
	for _, existing := range current.Votes {
		if strings.TrimSpace(existing) == voteRef {
			return true
		}
	}
	return false
}
