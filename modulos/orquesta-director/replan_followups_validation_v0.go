package orquestadirector

import "strings"

func validateReplanFollowupsInputV0(input ReplanFollowupsInputV0) error {
	required := map[string]string{
		"decision_command_meta.run_id":          input.DecisionCommandMeta.RunID,
		"decision_command_meta.command_id":      input.DecisionCommandMeta.CommandID,
		"decision_command_meta.idempotency_key": input.DecisionCommandMeta.IdempotencyKey,
		"decision_command_meta.occurred_at":     input.DecisionCommandMeta.OccurredAt,
		"decision_payload.replan_ref":           input.DecisionPayload.ReplanRef,
		"decision_payload.run_ref":              input.DecisionPayload.RunRef,
		"decision_payload.task_ref":             input.DecisionPayload.TaskRef,
		"decision_payload.source_ref":           input.DecisionPayload.SourceRef,
		"decision_payload.accepted_action":      string(input.DecisionPayload.AcceptedAction),
		"decision_payload.summary":              input.DecisionPayload.Summary,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return replanFollowupsErrorV0(input, "campo requerido", field)
		}
	}
	if input.DecisionCommandMeta.RunID != input.DecisionPayload.RunRef {
		return replanFollowupsErrorV0(input, "run_ref no coincide", "decision_payload.run_ref")
	}
	if err := validateReplanFollowupCandidateRunRefsV0(input); err != nil {
		return err
	}
	return validateReplanFollowupBlockedAgentsV0(input)
}

func validateReplanFollowupCandidateRunRefsV0(input ReplanFollowupsInputV0) error {
	if input.OpenPhaseCandidate != nil &&
		input.OpenPhaseCandidate.CommandMeta.RunID != input.DecisionCommandMeta.RunID {
		return replanFollowupsErrorV0(input, "run_id no coincide", "open_phase_candidate.command_meta.run_id")
	}
	for _, candidate := range input.MicrotaskCandidates {
		if candidate.CommandMeta.RunID != input.DecisionCommandMeta.RunID {
			return replanFollowupsErrorV0(input, "run_id no coincide", "microtask_candidates.command_meta.run_id")
		}
		if strings.TrimSpace(candidate.Payload.Task.RunID) != input.DecisionCommandMeta.RunID {
			return replanFollowupsErrorV0(input, "run_id no coincide", "microtask_candidates.payload.task.run_id")
		}
	}
	if input.CapacityCandidate != nil &&
		input.CapacityCandidate.CommandMeta.RunID != input.DecisionCommandMeta.RunID {
		return replanFollowupsErrorV0(input, "run_id no coincide", "capacity_candidate.command_meta.run_id")
	}
	if input.AgentCandidate != nil &&
		input.AgentCandidate.CommandMeta.RunID != input.DecisionCommandMeta.RunID {
		return replanFollowupsErrorV0(input, "run_id no coincide", "agent_candidate.command_meta.run_id")
	}
	if input.AskDirectorCandidate != nil &&
		input.AskDirectorCandidate.CommandMeta.RunID != input.DecisionCommandMeta.RunID {
		return replanFollowupsErrorV0(input, "run_id no coincide", "ask_director_candidate.command_meta.run_id")
	}
	return nil
}

func validateReplanFollowupBlockedAgentsV0(input ReplanFollowupsInputV0) error {
	if input.AgentCandidate == nil {
		return nil
	}
	agentRef := strings.TrimSpace(input.AgentCandidate.Payload.AgentRequestID)
	for _, blocked := range input.BlockedAgentRefs {
		if strings.TrimSpace(blocked) == agentRef {
			return replanFollowupsErrorV0(input, "agent_ref bloqueado", "agent_candidate.payload.agent_request_id")
		}
	}
	return nil
}

func replanFollowupsErrorV0(
	input ReplanFollowupsInputV0,
	message string,
	field string,
) ReplanFollowupsErrorV0 {
	return ReplanFollowupsErrorV0{
		Code:          ErrDirectorReplanFollowupsInvalidoV0,
		Message:       message,
		Field:         field,
		CorrelationID: strings.TrimSpace(input.DecisionCommandMeta.CorrelationID),
	}
}
