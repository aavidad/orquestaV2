package orquestaappcodexstack

import (
	"context"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

type compositeDirectorDecisionSourceV0 struct {
	Sources              []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0
	AppChangeStore       orquestaappchange.AppChangeRecordSourcePortV0
	DomainRequiredPolicy orquestadomainwork.DomainWorkRequiredTestPolicyPortV0
	Budget               orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetV0
}

var _ orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0 = compositeDirectorDecisionSourceV0{}

func (source compositeDirectorDecisionSourceV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	out := []orquestadirectoragent.DirectorAgentDecisionV0{}
	sourceCount := 0
	for _, child := range source.Sources {
		if child == nil {
			continue
		}
		sourceCount++
		if err := source.validateBudgetV0(request.Run, out, sourceCount, nil); err != nil {
			return nil, err
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
		if err := source.validateBudgetV0(request.Run, out, sourceCount, decisions); err != nil {
			return nil, err
		}
		out = append(out, decisions...)
	}
	return out, nil
}

func (source compositeDirectorDecisionSourceV0) validateBudgetV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	accepted []orquestadirectoragent.DirectorAgentDecisionV0,
	sourceCount int,
	incoming []orquestadirectoragent.DirectorAgentDecisionV0,
) error {
	candidate := append(append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), accepted...), incoming...)
	stats := orquestadirectoragentworkflow.CountDirectorAgentDecisionBatchV0(run, candidate)
	stats.Sources = sourceCount
	if incoming != nil {
		stats.OmittedDecisions = len(incoming)
	}
	return orquestadirectoragentworkflow.ValidateDirectorAgentDecisionBatchBudgetV0(
		source.Budget,
		stats,
	)
}
