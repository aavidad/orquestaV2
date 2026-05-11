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
	for _, decision := range decisions {
		if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(decision); len(issues) > 0 {
			return nil, DirectorAgentFileSourceIssueV0{Field: "decision." + issues[0].Field}
		}
	}
	return decisions, nil
}
