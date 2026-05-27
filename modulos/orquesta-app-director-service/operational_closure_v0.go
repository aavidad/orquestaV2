package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func maybeCloseOperationalDirectorV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, []orquestacionnucleoapp.ErrorV0, error) {
	if loop.Status == orquestacionnucleoapp.ProgressiveLoopStatusNeedsDirectorV0 {
		atReplanOrClose, err := operationalDirectorPlanStateIsAtReplanOrCloseV0(ctx, request, ports)
		if err != nil || !atReplanOrClose {
			return loop, nil, err
		}
		loop.Status = orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0
	}
	if loop.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		return loop, nil, nil
	}
	if loop.PendingOutboxCount > 0 {
		if err := operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
			ctx,
			request,
			ports,
			"operational-closure-outbox-pending",
			nil,
		); err != nil {
			return loop, nil, err
		}
		return loop, nil, nil
	}
	if ports.OperationalClosureSource == nil {
		if err := operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
			ctx,
			request,
			ports,
			"operational-closure-source-unavailable",
			nil,
		); err != nil {
			return loop, nil, err
		}
		return loop, nil, nil
	}
	if ports.DirectorTaskStore == nil {
		if err := operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
			ctx,
			request,
			ports,
			"operational-closure-task-store-unavailable",
			nil,
		); err != nil {
			return loop, nil, err
		}
		return loop, nil, nil
	}
	if loop.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		if err := operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
			ctx,
			request,
			ports,
			"operational-closure-run-not-active",
			nil,
		); err != nil {
			return loop, nil, err
		}
		return loop, nil, nil
	}
	blocked, err := operationalDirectorPlanStateBlocksClosureV0(ctx, request, ports)
	if err != nil || blocked {
		return loop, nil, err
	}
	requiredTestEvidenceRefs, err := operationalDirectorPlanStateRequiredTestEvidenceRefsForClosureV0(ctx, request, ports)
	if err != nil {
		return loop, nil, err
	}
	closureRequest, ok, err := ports.OperationalClosureSource.BuildOperationalDirectorClosureRequestV0(
		ctx,
		AppDirectorOperationalClosureRequestV0{
			Run:                      loop.Run,
			LoopStatus:               loop.Status,
			OccurredAt:               request.OccurredAt,
			CorrelationID:            request.CorrelationID,
			RequestedBy:              request.RequestedBy,
			WaitAgentRefs:            append([]string(nil), loopRequest.WaitAgentRefs...),
			WaitScopeApplied:         loopRequest.WaitScopeApplied,
			WaitCohortRef:            request.WaitCohortRef,
			WaitWaveRef:              request.WaitWaveRef,
			WaitParentTaskRef:        request.WaitParentTaskRef,
			RequiredTestEvidenceRefs: requiredTestEvidenceRefs,
			EvidenceRefs:             []string{"evidence-ref-app-director-operational-closure-v0"},
		},
	)
	if err != nil || !ok {
		if blockErr := operationalDirectorPlanStateBlockedAfterClosureV0(
			ctx,
			request,
			ports,
			"operational-closure-source-unavailable",
			nil,
		); blockErr != nil {
			return loop, nil, blockErr
		}
		return loop, nil, err
	}
	closureRequest = appDirectorClosureRequestWithDefaultsV0(request, closureRequest)
	beforeClosureRun := loop.Run
	closure, err := (orquestacionnucleoapp.OperationalDirectorClosureV0{
		RunStore:                  ports.RunStore,
		EventSink:                 ports.EventSink,
		EventReader:               ports.EventReader,
		TaskStore:                 ports.DirectorTaskStore,
		RequiredTestEvidenceStore: ports.RequiredTestEvidenceStore,
		RequestedBy:               request.RequestedBy,
	}).CloseOperationalDirectorRunV0(ctx, closureRequest)
	if err != nil {
		return loop, nil, err
	}
	if len(closure.Issues) > 0 || closure.Run.RunID == "" {
		if closure.Run.RunID != "" && appDirectorClosureOnlyOpenTasksIssuesV0(closure.Issues) {
			loop.Run = closure.Run
			if appDirectorClosureClosedTaskProgressV0(beforeClosureRun, closure.Run) {
				loop.Status = orquestacionnucleoapp.ProgressiveLoopStatusNeedsDirectorV0
				return loop, nil, nil
			}
			reopened, reopenErr := operationalDirectorPlanStateReopenReviewForClosureOpenTasksNoProgressV0(
				ctx,
				request,
				ports,
				closure.Run,
			)
			if reopenErr != nil {
				return loop, closure.Issues, reopenErr
			}
			loop.Status = orquestacionnucleoapp.ProgressiveLoopStatusNeedsDirectorV0
			if reopened {
				return loop, nil, nil
			}
			return loop, closure.Issues, nil
		}
		replanRefs, replanErr := operationalDirectorPlanStateReplanClosureIssuesV0(
			ctx,
			request,
			ports,
			closure.Run,
			closureRequest,
			closure.Issues,
		)
		if replanErr != nil {
			return loop, closure.Issues, replanErr
		}
		if blockErr := operationalDirectorPlanStateBlockedAfterClosureV0(
			ctx,
			request,
			ports,
			"operational-closure-issues",
			closure.Issues,
			replanRefs...,
		); blockErr != nil {
			return loop, closure.Issues, blockErr
		}
		return loop, closure.Issues, nil
	}
	if err := operationalDirectorPlanStateClosedAfterClosureV0(ctx, request, ports, closureRequest); err != nil {
		return loop, nil, err
	}
	loop.Run = closure.Run
	return loop, nil, nil
}
