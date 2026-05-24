package orquestaappcodexstack

import (
	"context"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

type compositeDirectorDecisionSourceV0 struct {
	Sources              []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0
	AppChangeStore       orquestaappchange.AppChangeRecordSourcePortV0
	DomainRequiredPolicy orquestadomainwork.DomainWorkRequiredTestPolicyPortV0
}

var _ orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0 = compositeDirectorDecisionSourceV0{}

func (source compositeDirectorDecisionSourceV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	out := []orquestadirectoragent.DirectorAgentDecisionV0{}
	for _, child := range source.Sources {
		if child == nil {
			continue
		}
		decisions, err := child.ListDirectorAgentDecisionsV0(ctx, request)
		if err != nil {
			return nil, err
		}
		decisions = normalizeCompositeDirectorDecisionBatchV0(request.Run, decisions)
		decisions, err = source.normalizeDomainWorkRequiredTestsV0(ctx, request.Run, decisions)
		if err != nil {
			return nil, err
		}
		if err := validateCompositeDirectorDecisionBatchV0(
			request.Run,
			request.RequestKind,
			request.ObjectiveHints,
			decisions,
		); err != nil {
			return nil, err
		}
		out = append(out, decisions...)
	}
	return out, nil
}
