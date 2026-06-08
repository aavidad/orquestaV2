package orquestadirectorcandidates

import "strings"

func normalizeInputV0(input SchedulableWorkCandidateInputV0) SchedulableWorkCandidateInputV0 {
	input.CandidateRef = strings.TrimSpace(input.CandidateRef)
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.PhaseID = strings.TrimSpace(input.PhaseID)
	input.TaskRef = strings.TrimSpace(input.TaskRef)
	input.SubjectClaimRefs = normalizeRefsV0(input.SubjectClaimRefs)
	input.GateEvidenceRefs = normalizeRefsV0(input.GateEvidenceRefs)
	input.EvidenceRefs = normalizeRefsV0(input.EvidenceRefs)
	input.Commands = normalizeCommandsV0(input.Commands)
	input.Capacity = normalizeCapacityV0(input.Capacity)
	input.Agent = normalizeAgentV0(input.Agent)
	input.ScopeClaims = normalizeScopeClaimsV0(input.ScopeClaims)
	return input
}

func normalizeCommandsV0(commands WorkCandidateCommandsV0) WorkCandidateCommandsV0 {
	commands.CapacityCommandID = strings.TrimSpace(commands.CapacityCommandID)
	commands.CapacityIdempotencyKey = strings.TrimSpace(commands.CapacityIdempotencyKey)
	commands.GateCommandID = strings.TrimSpace(commands.GateCommandID)
	commands.GateIdempotencyKey = strings.TrimSpace(commands.GateIdempotencyKey)
	commands.AgentCommandID = strings.TrimSpace(commands.AgentCommandID)
	commands.AgentIdempotencyKey = strings.TrimSpace(commands.AgentIdempotencyKey)
	commands.CorrelationID = strings.TrimSpace(commands.CorrelationID)
	commands.RequestedBy = strings.TrimSpace(commands.RequestedBy)
	commands.OccurredAt = strings.TrimSpace(commands.OccurredAt)
	return commands
}

func normalizeCapacityV0(capacity WorkCandidateCapacityInputV0) WorkCandidateCapacityInputV0 {
	capacity.CapacityRequestID = strings.TrimSpace(capacity.CapacityRequestID)
	capacity.ReasonCode = strings.TrimSpace(capacity.ReasonCode)
	capacity.Summary = strings.TrimSpace(capacity.Summary)
	capacity.EvidenceRefs = normalizeRefsV0(capacity.EvidenceRefs)
	return capacity
}

func normalizeAgentV0(agent WorkCandidateAgentInputV0) WorkCandidateAgentInputV0 {
	agent.AgentRequestID = strings.TrimSpace(agent.AgentRequestID)
	agent.ClaimRef = strings.TrimSpace(agent.ClaimRef)
	agent.Role = strings.TrimSpace(agent.Role)
	agent.Summary = strings.TrimSpace(agent.Summary)
	agent.EvidenceRefs = normalizeRefsV0(agent.EvidenceRefs)
	agent.SkillRefs = normalizeRefsV0(agent.SkillRefs)
	return agent
}
