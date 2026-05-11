package orquestacoreworkflow

import "strings"

func handleAssessAgentWorkCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeAssessAgentWorkCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureAssessAgentWorkCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	if agentAssessmentAlreadyReflectedV0(current, payload.AssessmentRef) {
		if err := ensureAgentWorkAssessedEffectMatchesV0(current, command, payload); err != nil {
			return emptyCommandResultV0(), err
		}
		if payload.Action == AgentAssessmentActionStopAgentV0 && !agentStopAlreadyReflectedV0(current, payload.AgentRequestID) {
			return pendingStopAgentAssessmentResultV0(current, command, payload)
		}
		if payload.Action == AgentAssessmentActionStopAgentV0 && !agentStopConfirmedAlreadyReflectedV0(current, payload.AgentRequestID) {
			return pendingStopAgentAssessmentOutboxResultV0(current, command, payload)
		}
		return idempotentCommandResultV0(), nil
	}
	if !runPhaseIsCurrentV0(current, OrchestrationPhaseIDV0(payload.PhaseID)) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	assessed, err := NewAgentWorkAssessedEventV0(commandEventMetaV0(current, command, OrchestrationEventAgentWorkAssessedV0), agentWorkAssessedPayloadFromCommandV0(payload))
	if err != nil || payload.Action != AgentAssessmentActionStopAgentV0 {
		return eventCommandResultV0(assessed, err)
	}
	if agentStopAlreadyReflectedV0(current, payload.AgentRequestID) {
		return eventCommandResultV0(assessed, nil)
	}
	return stopAgentAssessmentResultV0(current, command, payload, assessed)
}

func stopAgentAssessmentResultV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload AssessAgentWorkCommandPayloadV0, assessed OrchestrationEventV0) (OrchestrationCommandResultV0, error) {
	stopPayload := stopAgentPayloadFromAssessmentV0(payload)
	stop, err := NewAgentStopRequestedEventV0(stopEventMetaFromAssessmentV0(command, nextCommandSequenceV0(current)+1), agentStopRequestedPayloadFromCommandV0(stopPayload))
	if err != nil {
		return emptyCommandResultV0(), err
	}
	outbox, err := newStopRuntimeAgentOutboxV0(commandWithIdempotencyKeyV0(command, stop.IdempotencyKey), stop, stopPayload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Events: []OrchestrationEventV0{assessed, stop}, Outbox: []OutboxMessageV0{outbox}}, nil
}

func pendingStopAgentAssessmentResultV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload AssessAgentWorkCommandPayloadV0) (OrchestrationCommandResultV0, error) {
	stopPayload := stopAgentPayloadFromAssessmentV0(payload)
	stop, err := NewAgentStopRequestedEventV0(stopEventMetaFromAssessmentV0(command, nextCommandSequenceV0(current)), agentStopRequestedPayloadFromCommandV0(stopPayload))
	if err != nil {
		return emptyCommandResultV0(), err
	}
	outbox, err := newStopRuntimeAgentOutboxV0(commandWithIdempotencyKeyV0(command, stop.IdempotencyKey), stop, stopPayload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Events: []OrchestrationEventV0{stop}, Outbox: []OutboxMessageV0{outbox}}, nil
}

func pendingStopAgentAssessmentOutboxResultV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload AssessAgentWorkCommandPayloadV0) (OrchestrationCommandResultV0, error) {
	matches, err := assessmentStopEffectMatchesV0(current, command, payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if !matches {
		return idempotentCommandResultV0(), nil
	}
	stopPayload := stopAgentPayloadFromAssessmentV0(payload)
	stop, err := NewAgentStopRequestedEventV0(stopEventMetaFromAssessmentV0(command, nextCommandSequenceV0(current)), agentStopRequestedPayloadFromCommandV0(stopPayload))
	if err != nil {
		return emptyCommandResultV0(), err
	}
	outbox, err := newStopRuntimeAgentOutboxV0(commandWithIdempotencyKeyV0(command, stop.IdempotencyKey), stop, stopPayload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Outbox: []OutboxMessageV0{outbox}}, nil
}

func stopEventMetaFromAssessmentV0(command OrchestrationCommandV0, sequence int64) OrchestrationEventMetaV0 {
	stopKey := assessmentStopIdempotencyKeyV0(command)
	return OrchestrationEventMetaV0{
		EventID:        commandEventIDV0(commandWithIdempotencyKeyV0(command, stopKey), OrchestrationEventAgentStopRequestedV0),
		RunID:          strings.TrimSpace(command.RunID),
		Sequence:       sequence,
		IdempotencyKey: stopKey,
		CorrelationID:  strings.TrimSpace(command.CorrelationID),
		CausationID:    strings.TrimSpace(command.CommandID),
		OccurredAt:     strings.TrimSpace(command.OccurredAt),
	}
}

func assessmentStopIdempotencyKeyV0(command OrchestrationCommandV0) string {
	return strings.TrimSpace(command.IdempotencyKey) + "-stop-agent"
}

func commandWithIdempotencyKeyV0(command OrchestrationCommandV0, idempotencyKey string) OrchestrationCommandV0 {
	command.IdempotencyKey = idempotencyKey
	return command
}
