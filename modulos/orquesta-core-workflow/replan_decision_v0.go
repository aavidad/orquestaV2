package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxReplanDecisionPayloadBytesV0 = 2048
	maxReplanDecisionStringV0       = 600
	maxReplanDecisionRefsV0         = 20
)

type ReplanDecisionActionV0 string

const (
	ReplanDecisionActionSplitTaskV0        ReplanDecisionActionV0 = "split_task"
	ReplanDecisionActionRetryTaskV0        ReplanDecisionActionV0 = "retry_task"
	ReplanDecisionActionReplaceAgentV0     ReplanDecisionActionV0 = "replace_agent"
	ReplanDecisionActionEscalateCapacityV0 ReplanDecisionActionV0 = "escalate_capacity"
	ReplanDecisionActionAskDirectorV0      ReplanDecisionActionV0 = "ask_director"
	ReplanDecisionActionAbortTaskV0        ReplanDecisionActionV0 = "abort_task"
)

type RecordReplanDecisionCommandPayloadV0 struct {
	ReplanRef      string                 `json:"replan_ref"`
	RunRef         string                 `json:"run_ref"`
	TaskRef        string                 `json:"task_ref"`
	SourceRef      string                 `json:"source_ref"`
	AcceptedAction ReplanDecisionActionV0 `json:"accepted_action"`
	FollowupRefs   []string               `json:"followup_refs"`
	Summary        string                 `json:"summary"`
	EvidenceRefs   []string               `json:"evidence_refs,omitempty"`
}

type ReplanDecisionRecordedPayloadV0 = RecordReplanDecisionCommandPayloadV0

func NewRecordReplanDecisionCommandV0(meta OrchestrationCommandMetaV0, payload RecordReplanDecisionCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRecordReplanDecisionV0, normalizeRecordReplanDecisionPayloadV0(payload))
}

func NewReplanDecisionRecordedEventV0(meta OrchestrationEventMetaV0, payload ReplanDecisionRecordedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventReplanDecisionRecordedV0, normalizeReplanDecisionRecordedPayloadV0(payload))
}

func decodeRecordReplanDecisionCommandPayloadV0(raw json.RawMessage) (RecordReplanDecisionCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxReplanDecisionPayloadBytesV0 {
		return RecordReplanDecisionCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RecordReplanDecisionCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RecordReplanDecisionCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRecordReplanDecisionPayloadV0(payload)
	if err := validateRecordReplanDecisionPayloadDataV0(payload); err != nil {
		return RecordReplanDecisionCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateReplanDecisionRecordedPayloadV0(event OrchestrationEventV0) error {
	var payload ReplanDecisionRecordedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateReplanDecisionRecordedPayloadDataV0(normalizeReplanDecisionRecordedPayloadV0(payload))
}

func handleRecordReplanDecisionCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRecordReplanDecisionCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRecordReplanDecisionCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	matches, conflicts := replanDecisionMatchesPayloadV0(current, payload)
	if conflicts {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.replan_ref")
	}
	if matches {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventReplanDecisionRecordedV0, payload.ReplanRef, replanDecisionRecordedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewReplanDecisionRecordedEventV0(commandEventMetaV0(current, command, OrchestrationEventReplanDecisionRecordedV0), replanDecisionRecordedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyReplanDecisionRecordedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload ReplanDecisionRecordedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeReplanDecisionRecordedPayloadV0(payload)
	if err := ensureReplanDecisionRecordedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	matches, conflicts := replanDecisionEventMatchesPayloadV0(current, payload)
	if conflicts {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.replan_ref")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.ReplanRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	if !matches {
		next.ReplanDecisions = appendUniqueCompactRefV0(cloneStringsV0(next.ReplanDecisions), replanDecisionProjectionRefV0(payload))
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.ReplanRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func replanDecisionRecordedPayloadFromCommandV0(payload RecordReplanDecisionCommandPayloadV0) ReplanDecisionRecordedPayloadV0 {
	return ReplanDecisionRecordedPayloadV0{
		ReplanRef:      payload.ReplanRef,
		RunRef:         payload.RunRef,
		TaskRef:        payload.TaskRef,
		SourceRef:      payload.SourceRef,
		AcceptedAction: payload.AcceptedAction,
		FollowupRefs:   cloneStringsV0(payload.FollowupRefs),
		Summary:        payload.Summary,
		EvidenceRefs:   cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRecordReplanDecisionPayloadV0(payload RecordReplanDecisionCommandPayloadV0) RecordReplanDecisionCommandPayloadV0 {
	return RecordReplanDecisionCommandPayloadV0{
		ReplanRef:      strings.TrimSpace(payload.ReplanRef),
		RunRef:         strings.TrimSpace(payload.RunRef),
		TaskRef:        strings.TrimSpace(payload.TaskRef),
		SourceRef:      strings.TrimSpace(payload.SourceRef),
		AcceptedAction: ReplanDecisionActionV0(strings.TrimSpace(string(payload.AcceptedAction))),
		FollowupRefs:   compactUniqueStringsV0(payload.FollowupRefs),
		Summary:        strings.TrimSpace(payload.Summary),
		EvidenceRefs:   compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeReplanDecisionRecordedPayloadV0(payload ReplanDecisionRecordedPayloadV0) ReplanDecisionRecordedPayloadV0 {
	normalized := normalizeRecordReplanDecisionPayloadV0(RecordReplanDecisionCommandPayloadV0(payload))
	return replanDecisionRecordedPayloadFromCommandV0(normalized)
}
