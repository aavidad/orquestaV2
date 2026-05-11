package orquestacoreworkflow

import "strings"

func applyDirectorQuestionRaisedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload DirectorQuestionRaisedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeQuestionRaisedPayloadV0(payload)
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.QuestionID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.DirectorQuestions = appendUniqueCompactRefV0(next.DirectorQuestions, payload.QuestionID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.QuestionID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func applyDirectorQuestionAnsweredEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload DirectorQuestionAnsweredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeDirectorQuestionAnsweredPayloadV0(payload)
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	if !directorQuestionAlreadyReflectedV0(current, payload.QuestionID) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.question_id")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.AnswerID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.DirectorAnswers = appendUniqueCompactRefV0(next.DirectorAnswers, payload.AnswerID)
	next.DirectorAnsweredQuestions = appendUniqueCompactRefV0(next.DirectorAnsweredQuestions, payload.QuestionID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.AnswerID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	if payload.Unblocks {
		next.Blockers = removeCompactRefV0(next.Blockers, payload.BlockerID)
		if len(next.Blockers) == 0 && next.Status == OrchestrationRunStatusBlockedV0 {
			next.Status = OrchestrationRunStatusActiveV0
		}
	}
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func applyMicrotaskCreatedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload MicrotaskCreatedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	if err := ensureCreateMicrotaskEventPlanningCurrentV0(current); err != nil {
		return current, err
	}
	if err := ensureMicrotaskEventContractsReadyV0(current, payload.FunctionContractRefs); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.TaskID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Tasks = appendUniqueCompactRefV0(next.Tasks, payload.TaskID)
	for _, ref := range compactStringsV0(payload.FunctionContractRefs) {
		next.FunctionContracts = appendUniqueCompactRefV0(next.FunctionContracts, ref)
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.TaskID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func applyCapacityRequestedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload CapacityRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	if !runPhaseIsCurrentV0(current, OrchestrationPhaseIDV0(payload.PhaseID)) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.CapacityRequestID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.CapacityRequests = appendUniqueCompactRefV0(next.CapacityRequests, payload.CapacityRequestID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.CapacityRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func applyAgentRequestedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload AgentRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	if !runPhaseIsCurrentV0(current, OrchestrationPhaseIDV0(payload.PhaseID)) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	if !capacityDecisionAlreadyReflectedV0(current, payload.CapacityRequestRef) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.capacity_request_ref")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.AgentRequestID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Agents = appendUniqueCompactRefV0(next.Agents, payload.AgentRequestID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.AgentRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}
