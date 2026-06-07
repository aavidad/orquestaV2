package orquestacoreworkflow

import "strings"

const (
	OrchestrationCommandNoopAlreadyReflectedV0 = "already_reflected"
)

type OrchestrationCommandResultV0 struct {
	Events     []OrchestrationEventV0 `json:"events"`
	Outbox     []OutboxMessageV0      `json:"outbox"`
	Idempotent bool                   `json:"idempotent,omitempty"`
	NoopReason string                 `json:"noop_reason,omitempty"`
}

func HandleCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	if err := ValidateOrchestrationCommandV0(command); err != nil {
		return emptyCommandResultV0(), err
	}

	handler, ok := lookupCommandHandlerV0(command.CommandType)
	if !ok {
		return emptyCommandResultV0(), commandErrorV0(ErrComandoNoSoportadoV0, "command_type")
	}
	return handler(current, command)
}

func handleStartRunCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeStartRunCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if startRunAlreadyReflectedV0(current, command, payload) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventRunStartedV0, command.RunID, runStartedPayloadFromCommandV0(command, payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	if !orchestrationRunIsEmptyV0(current) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "run")
	}
	event, err := NewRunStartedEventV0(commandEventMetaV0(current, command, OrchestrationEventRunStartedV0), runStartedPayloadFromCommandV0(command, payload))
	return eventCommandResultV0(event, err)
}

func handleOpenPhaseCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeOpenPhaseCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if reflected, err := phaseOpenedEffectKnownV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	} else if reflected {
		if err := ensurePhaseOpenedEffectMatchesV0(current, command, payload); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	phaseID := OrchestrationPhaseIDV0(payload.PhaseID)
	if phaseAlreadyOpenV0(current, phaseID) && strings.TrimSpace(current.LastEventID) == phaseOpenedEffectSubjectFromCommandV0(command) {
		return idempotentCommandResultV0(), nil
	}
	if !runContainsPhaseV0(current, phaseID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "phases")
	}
	event, err := NewPhaseOpenedEventV0(commandEventMetaV0(current, command, OrchestrationEventPhaseOpenedV0), phaseOpenedPayloadFromCommandV0(command, payload))
	return eventCommandResultV0(event, err)
}

func handleBlockRunCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeBlockRunCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if blockerAlreadyReflectedV0(current, payload.BlockerID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventRunBlockedV0, payload.BlockerID, runBlockedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewRunBlockedEventV0(commandEventMetaV0(current, command, OrchestrationEventRunBlockedV0), runBlockedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func handleResolveRunBlockerCommandV0(
	current OrchestrationRunV0,
	command OrchestrationCommandV0,
) (OrchestrationCommandResultV0, error) {
	payload, err := decodeResolveRunBlockerCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	repairPartialProjection := false
	if reflected, err := runBlockerResolvedEffectKnownV0(current, payload.BlockerID); err != nil {
		return emptyCommandResultV0(), err
	} else if reflected {
		if err := ensureRunBlockerResolvedEffectMatchesV0(current, command, payload); err != nil {
			return emptyCommandResultV0(), err
		}
		if !blockerAlreadyReflectedV0(current, payload.BlockerID) {
			return idempotentCommandResultV0(), nil
		}
		repairPartialProjection = true
	}
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if !blockerAlreadyReflectedV0(current, payload.BlockerID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.blocker_id")
	}
	meta := commandEventMetaV0(current, command, OrchestrationEventRunBlockerResolvedV0)
	if repairPartialProjection {
		meta.Sequence = current.LastSequence
	}
	event, err := NewRunBlockerResolvedEventV0(
		meta,
		runBlockerResolvedPayloadFromCommandV0(payload),
	)
	return eventCommandResultV0(event, err)
}

func ensureExistingRunForCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) error {
	if orchestrationRunIsEmptyV0(current) {
		return commandErrorV0(ErrTransicionInvalidaV0, "run")
	}
	if strings.TrimSpace(current.RunID) != strings.TrimSpace(command.RunID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "run_id")
	}
	if current.Status == OrchestrationRunStatusClosedV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "status")
	}
	if issues := ValidateOrchestrationRunV0(current); len(issues) > 0 {
		return commandErrorV0(ErrTransicionInvalidaV0, issues[0].Field)
	}
	return nil
}

func ensureActiveRunForCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) error {
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return err
	}
	if current.Status != OrchestrationRunStatusActiveV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "status")
	}
	return nil
}

func startRunAlreadyReflectedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload StartRunCommandPayloadV0) bool {
	if orchestrationRunIsEmptyV0(current) || len(ValidateOrchestrationRunV0(current)) > 0 {
		return false
	}
	return strings.TrimSpace(current.RunID) == strings.TrimSpace(command.RunID) &&
		strings.TrimSpace(current.ProjectRef) == payload.ProjectRef &&
		strings.TrimSpace(current.AppSpecRef) == payload.AppSpecRef
}

func phaseAlreadyOpenV0(current OrchestrationRunV0, phaseID OrchestrationPhaseIDV0) bool {
	if normalizePhaseIDV0(current.CurrentPhase) != normalizePhaseIDV0(phaseID) {
		return false
	}
	for _, phase := range current.Phases {
		if normalizePhaseIDV0(phase.ID) == normalizePhaseIDV0(phaseID) {
			return phase.Status == OrchestrationPhaseStatusActiveV0
		}
	}
	return false
}

func blockerAlreadyReflectedV0(current OrchestrationRunV0, blockerID string) bool {
	if current.Status != OrchestrationRunStatusBlockedV0 {
		return false
	}
	for _, blocker := range current.Blockers {
		if strings.TrimSpace(blocker) == strings.TrimSpace(blockerID) {
			return true
		}
	}
	return false
}

func commandEventMetaV0(current OrchestrationRunV0, command OrchestrationCommandV0, eventType string) OrchestrationEventMetaV0 {
	return OrchestrationEventMetaV0{
		EventID:        commandEventIDV0(command, eventType),
		RunID:          strings.TrimSpace(command.RunID),
		Sequence:       nextCommandSequenceV0(current),
		IdempotencyKey: strings.TrimSpace(command.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(command.CorrelationID),
		CausationID:    strings.TrimSpace(command.CommandID),
		OccurredAt:     strings.TrimSpace(command.OccurredAt),
	}
}

func commandEventIDV0(command OrchestrationCommandV0, eventType string) string {
	return "evt-" + strings.ToLower(eventType) + "-" + strings.TrimSpace(command.IdempotencyKey)
}

func nextCommandSequenceV0(current OrchestrationRunV0) int64 {
	if orchestrationRunIsEmptyV0(current) {
		return 1
	}
	return current.LastSequence + 1
}

func eventCommandResultV0(event OrchestrationEventV0, err error) (OrchestrationCommandResultV0, error) {
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{
		Events: []OrchestrationEventV0{event},
		Outbox: []OutboxMessageV0{},
	}, nil
}

func idempotentCommandResultV0() OrchestrationCommandResultV0 {
	return OrchestrationCommandResultV0{
		Events:     []OrchestrationEventV0{},
		Outbox:     []OutboxMessageV0{},
		Idempotent: true,
		NoopReason: OrchestrationCommandNoopAlreadyReflectedV0,
	}
}

func emptyCommandResultV0() OrchestrationCommandResultV0 {
	return OrchestrationCommandResultV0{
		Events: []OrchestrationEventV0{},
		Outbox: []OutboxMessageV0{},
	}
}
