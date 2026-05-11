package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxQualityGatePayloadBytesV0 = 4096
	maxQualityGateStringV0       = 800
	maxQualityGateRefsV0         = 50
)

type QualityGateDecisionV0 string

const (
	QualityGateDecisionAcceptedV0       QualityGateDecisionV0 = "accepted"
	QualityGateDecisionReworkRequiredV0 QualityGateDecisionV0 = "rework_required"
	QualityGateDecisionBlockedV0        QualityGateDecisionV0 = "blocked"
	QualityGateDecisionAskDirectorV0    QualityGateDecisionV0 = "ask_director"
)

type RecordQualityGateCommandPayloadV0 struct {
	RunRef       string                `json:"run_ref"`
	GateRef      string                `json:"gate_ref"`
	PhaseID      string                `json:"phase_id"`
	SubjectRef   string                `json:"subject_ref"`
	Decision     QualityGateDecisionV0 `json:"decision"`
	IssueRefs    []string              `json:"issue_refs,omitempty"`
	Summary      string                `json:"summary"`
	EvidenceRefs []string              `json:"evidence_refs,omitempty"`
}

type QualityGateRecordedPayloadV0 = RecordQualityGateCommandPayloadV0

func NewRecordQualityGateCommandV0(meta OrchestrationCommandMetaV0, payload RecordQualityGateCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRecordQualityGateV0, normalizeRecordQualityGatePayloadV0(payload))
}

func NewQualityGateRecordedEventV0(meta OrchestrationEventMetaV0, payload QualityGateRecordedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventQualityGateRecordedV0, normalizeQualityGateRecordedPayloadV0(payload))
}

func decodeRecordQualityGateCommandPayloadV0(raw json.RawMessage) (RecordQualityGateCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxQualityGatePayloadBytesV0 {
		return RecordQualityGateCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RecordQualityGateCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RecordQualityGateCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRecordQualityGatePayloadV0(payload)
	if err := validateRecordQualityGatePayloadDataV0(payload); err != nil {
		return RecordQualityGateCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateQualityGateRecordedPayloadV0(event OrchestrationEventV0) error {
	var payload QualityGateRecordedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateQualityGateRecordedPayloadDataV0(normalizeQualityGateRecordedPayloadV0(payload))
}

func handleRecordQualityGateCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRecordQualityGateCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRecordQualityGateCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	matches, conflicts := qualityGateMatchesPayloadV0(current, payload)
	if conflicts {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.gate_ref")
	}
	if matches {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventQualityGateRecordedV0, payload.GateRef, qualityGateRecordedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewQualityGateRecordedEventV0(commandEventMetaV0(current, command, OrchestrationEventQualityGateRecordedV0), qualityGateRecordedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyQualityGateRecordedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload QualityGateRecordedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeQualityGateRecordedPayloadV0(payload)
	if err := ensureQualityGateRecordedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	matches, conflicts := qualityGateEventMatchesPayloadV0(current, payload)
	if conflicts {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.gate_ref")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.GateRef); err != nil {
		return current, err
	}
	next := cloneRunForReducerV0(current)
	if !matches {
		next.QualityGates = appendUniqueCompactRefV0(cloneStringsV0(next.QualityGates), qualityGateProjectionRefV0(payload))
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
