package orquestaappcodexstack

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const autoprogrammingDirectorOpenReviewEvidenceV0 = "evidence-ref-autoprogramming-review-ready"

type AutoprogrammingDirectorDecisionSourceV0 struct {
	TaskStore orquestacionnucleoapp.WorkflowTaskStorePortV0
}

var _ orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0 = AutoprogrammingDirectorDecisionSourceV0{}

func (source AutoprogrammingDirectorDecisionSourceV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	ok, err := codexStackAutoprogrammingShouldOpenReviewV0(ctx, source.TaskStore, request.Run)
	if err != nil || !ok {
		return nil, err
	}
	refSuffix := autoprogrammingBridgeHashRefV0(request.Run.RunID + ":open-review")
	return []orquestadirectoragent.DirectorAgentDecisionV0{{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-autoprogramming-open-review-" + refSuffix,
		RunID:         request.Run.RunID,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-autoprogramming-open-review-" + refSuffix,
		Summary:       "Abrir revision por entrega completa de autoprogramacion.",
		EvidenceRefs:  []string{autoprogrammingDirectorOpenReviewEvidenceV0},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			Reason:  "entrega completa lista para review",
		},
	}}, nil
}

func codexStackAutoprogrammingShouldOpenReviewV0(
	ctx context.Context,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!stackDrainRunHasAllTasksDeliveredOrClosedV0(run) ||
		drainRunHasPendingExternalAgentsV0(run, nil) {
		return false, nil
	}
	return RunHasOpenAutoprogrammingTasksV0(ctx, taskStore, run)
}
