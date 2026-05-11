package orquestadirectorcandidates

import "strings"

type requiredFieldV0 struct {
	name  string
	value string
}

func validateInputV0(input SchedulableWorkCandidateInputV0) error {
	fields := []requiredFieldV0{
		{"candidate_ref", input.CandidateRef},
		{"run_ref", input.RunRef},
		{"phase_id", input.PhaseID},
		{"task_ref", input.TaskRef},
		{"commands.occurred_at", input.Commands.OccurredAt},
		{"capacity.capacity_request_id", input.Capacity.CapacityRequestID},
		{"capacity.reason_code", input.Capacity.ReasonCode},
		{"capacity.summary", input.Capacity.Summary},
		{"agent.agent_request_id", input.Agent.AgentRequestID},
		{"agent.claim_ref", input.Agent.ClaimRef},
		{"agent.role", input.Agent.Role},
		{"agent.summary", input.Agent.Summary},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return candidateErrorV0(field.name)
		}
	}
	if len(input.SubjectClaimRefs) == 0 {
		return candidateErrorV0("subject_claim_refs")
	}
	if len(input.Claims)+len(input.ScopeClaims) == 0 {
		return candidateErrorV0("claims")
	}
	return validateCommandRefsV0(input.Commands)
}

func validateCommandRefsV0(commands WorkCandidateCommandsV0) error {
	fields := []requiredFieldV0{
		{"commands.capacity_command_id", commands.CapacityCommandID},
		{"commands.capacity_idempotency_key", commands.CapacityIdempotencyKey},
		{"commands.gate_command_id", commands.GateCommandID},
		{"commands.gate_idempotency_key", commands.GateIdempotencyKey},
		{"commands.agent_command_id", commands.AgentCommandID},
		{"commands.agent_idempotency_key", commands.AgentIdempotencyKey},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return candidateErrorV0(field.name)
		}
	}
	return nil
}
