package orquestadirectoragentfilesource

import (
	"strings"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func normalizeDirectorAgentDecisionFileDescriptorsV0(
	descriptors []DirectorAgentDecisionFileDescriptorV0,
) []DirectorAgentDecisionFileDescriptorV0 {
	out := make([]DirectorAgentDecisionFileDescriptorV0, 0, len(descriptors))
	for _, descriptor := range descriptors {
		normalized := DirectorAgentDecisionFileDescriptorV0{
			DescriptorRef: strings.TrimSpace(descriptor.DescriptorRef),
			RunID:         strings.TrimSpace(descriptor.RunID),
			Path:          strings.TrimSpace(descriptor.Path),
		}
		if normalized.DescriptorRef == "" && normalized.Path == "" {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

func validateDirectorAgentDecisionFileDecisionsV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if len(decisions) == 0 {
		return nil, DirectorAgentFileSourceIssueV0{Field: "decisions"}
	}
	normalized := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(decisions))
	for _, decision := range decisions {
		decision = normalizeDirectorAgentDecisionFileDecisionV0(decision)
		if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(decision); len(issues) > 0 {
			return nil, DirectorAgentFileSourceIssueV0{Field: "decision." + issues[0].Field}
		}
		normalized = append(normalized, decision)
	}
	return normalized, nil
}

func normalizeDirectorAgentDecisionFileDecisionV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	if strings.TrimSpace(decision.SchemaVersion) == "" {
		decision.SchemaVersion = orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0
	}
	decision = normalizeDirectorAgentDecisionFileCommandTypeV0(decision)
	decision = normalizeDirectorAgentDecisionFilePhaseV0(decision)
	if decision.CreateMicrotask == nil {
		return decision
	}
	if strings.TrimSpace(decision.CreateMicrotask.Task.SchemaVersion) == "" {
		decision.CreateMicrotask.Task.SchemaVersion = orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0
	}
	return decision
}

func normalizeDirectorAgentDecisionFileCommandTypeV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	commandType := strings.TrimSpace(decision.CommandType)
	if decision.CreateMicrotask != nil &&
		(commandType == "" || commandType == "create_microtask.v0") {
		decision.CommandType = orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0
	}
	return decision
}

func normalizeDirectorAgentDecisionFilePhaseV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	switch {
	case decision.CreateMicrotask != nil:
		decision.PhaseID = normalizeDirectorAgentDecisionFileMicrotaskPhaseV0(decision)
	case decision.RequestBrainstorm != nil:
		decision.PhaseID = decision.RequestBrainstorm.PhaseID
	case decision.RequestVote != nil:
		decision.PhaseID = decision.RequestVote.PhaseID
	case decision.AcceptDecision != nil:
		decision.PhaseID = decision.AcceptDecision.PhaseID
	case decision.PublishContract != nil:
		decision.PhaseID = decision.PublishContract.PhaseID
	case decision.RequestCapacity != nil:
		decision.PhaseID = decision.RequestCapacity.PhaseID
	case decision.RequestAgent != nil:
		decision.PhaseID = decision.RequestAgent.PhaseID
	case decision.RequestReview != nil:
		decision.PhaseID = decision.RequestReview.PhaseID
	case decision.AcceptReview != nil:
		decision.PhaseID = decision.AcceptReview.PhaseID
	case decision.RequestRework != nil:
		decision.PhaseID = decision.RequestRework.PhaseID
	case decision.CloseTask != nil:
		decision.PhaseID = decision.CloseTask.PhaseID
	case decision.RegisterFinalValidation != nil:
		decision.PhaseID = decision.RegisterFinalValidation.PhaseID
	case decision.CloseRun != nil:
		decision.PhaseID = decision.CloseRun.PhaseID
	case decision.ProposePlanTeam != nil:
		decision.PhaseID = decision.ProposePlanTeam.Plan.PhaseID
	}
	return decision
}

func normalizeDirectorAgentDecisionFileMicrotaskPhaseV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) string {
	current := strings.TrimSpace(decision.PhaseID)
	taskPhase := strings.TrimSpace(decision.CreateMicrotask.Task.PhaseID)
	if current == "" || current == taskPhase {
		return orquestadirectoragent.DirectorAgentPlanningPhaseIDV0
	}
	return current
}
