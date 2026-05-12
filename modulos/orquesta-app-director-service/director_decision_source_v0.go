package orquestaappdirectorservice

import (
	"context"

	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func consumeStartAppDirectorDecisionsV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, bool, error) {
	if ports.DirectorDecisionSource == nil {
		return loop, false, nil
	}
	decisions, err := ports.DirectorDecisionSource.ListDirectorAgentDecisionsV0(
		ctx,
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
			Run:           loop.Run,
			OccurredAt:    request.OccurredAt,
			CorrelationID: request.CorrelationID,
			RequestedBy:   request.RequestedBy,
		},
	)
	if err != nil {
		return loop, false, err
	}
	progressed := false
	for _, decision := range decisions {
		reflected, err := startAppDirectorDecisionAlreadyReflectedV0(
			request,
			loop.Run,
			decision,
		)
		if err != nil {
			return loop, progressed, err
		}
		if reflected {
			continue
		}
		if startAppDirectorDecisionDeferredV0(loop.Run, decision) {
			continue
		}
		if err := guardStartAppDirectorClosurePolicyV0(loop.Run, decision); err != nil {
			return loop, progressed, err
		}
		applied, err := orquestadirectoragentworkflow.ApplyDirectorAgentDecisionV0(
			ctx,
			orquestadirectoragentworkflow.ApplyDirectorAgentDecisionRequestV0{
				Decision:      decision,
				OccurredAt:    request.OccurredAt,
				CorrelationID: request.CorrelationID,
				RequestedBy:   request.RequestedBy,
			},
			orquestadirectoragentworkflow.ApplyDirectorAgentDecisionPortsV0{
				RunStore:  ports.RunStore,
				EventSink: ports.EventSink,
				TaskStore: ports.DirectorTaskStore,
			},
		)
		if err != nil {
			if startAppDirectorDecisionTransitionPendingV0(err) {
				return loop, progressed, nil
			}
			return loop, progressed, err
		}
		if len(applied.Issues) > 0 {
			return loop, progressed, AppDirectorServiceIssueV0{
				Field: "director_decision." + applied.Issues[0].Field,
			}
		}
		loop.Run = applied.Run
		if applied.EventsCount > 0 {
			progressed = true
		}
	}
	return loop, progressed, nil
}
