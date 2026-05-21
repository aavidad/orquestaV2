package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func maybeCloseOperationalDirectorV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, []orquestacionnucleoapp.ErrorV0, error) {
	if loop.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		loop.PendingOutboxCount > 0 {
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
		if blockErr := operationalDirectorPlanStateBlockedAfterClosureV0(
			ctx,
			request,
			ports,
			"operational-closure-issues",
			closure.Issues,
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

func operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	reason string,
	issues []orquestacionnucleoapp.ErrorV0,
) error {
	atReplanOrClose, err := operationalDirectorPlanStateIsAtReplanOrCloseV0(ctx, request, ports)
	if err != nil || !atReplanOrClose {
		return err
	}
	return operationalDirectorPlanStateBlockedAfterClosureV0(ctx, request, ports, reason, issues)
}

func operationalDirectorPlanStateIsAtReplanOrCloseV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (bool, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return false, nil
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 {
		return false, nil
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		return false, nil
	}
	return step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 &&
		step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0, nil
}

func operationalDirectorPlanStateBlocksClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (bool, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return false, nil
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 {
		return true, nil
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		return false, nil
	}
	if step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 &&
		step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return false, nil
	}
	return true, nil
}

func operationalDirectorPlanStateRequiredTestEvidenceRefsForClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) ([]string, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return nil, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return nil, err
	}
	refs := []string(nil)
	for _, step := range state.Steps {
		refs = append(refs, step.RequiredTestEvidenceRefs...)
	}
	return compactServiceRefsV0(refs), nil
}

func operationalDirectorPlanStateClosedAfterClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	closureRequest orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) error {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil || ports.OperationalPlanStateWriter == nil {
		return nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return nil
	}
	reason := "operational-closure-succeeded"
	closureRefs := operationalDirectorPlanStateClosureEvidenceRefsV0(closureRequest)
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == state.ActiveStepID ||
			(state.ActiveStepID == "" && step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0) {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepClosedV0
			nextStep.RequiredTestEvidenceRefs = compactServiceRefsV0(append(
				nextStep.RequiredTestEvidenceRefs,
				closureRequest.RequiredTestEvidenceRefs...,
			))
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, closureRefs...))
			nextStep.BlockerRefs = nil
			nextStep.Reason = reason
			if state.ActiveStepID == "" {
				state.ActiveStepID = step.StepID
			}
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0
	state.PendingAgentRefs = nil
	state.BlockerRefs = nil
	state.EvidenceRefs = compactServiceRefsV0(append(
		append(state.EvidenceRefs, closureRefs...),
		"evidence-ref-app-director-operational-plan-state-closure-succeeded-v0",
	))
	state.ClosureReason = reason
	state.Steps = nextSteps
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return err
	}
	return ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func operationalDirectorPlanStateBlockedAfterClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	reason string,
	issues []orquestacionnucleoapp.ErrorV0,
) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "operational-closure-blocked"
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil || ports.OperationalPlanStateWriter == nil {
		return nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return nil
	}
	blockerRefs := operationalDirectorPlanStateClosureBlockerRefsV0(reason, issues)
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == state.ActiveStepID ||
			(state.ActiveStepID == "" && step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0) {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			nextStep.BlockerRefs = compactServiceRefsV0(append(nextStep.BlockerRefs, blockerRefs...))
			nextStep.Reason = reason
			if state.ActiveStepID == "" {
				state.ActiveStepID = step.StepID
			}
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.BlockerRefs = compactServiceRefsV0(append(state.BlockerRefs, blockerRefs...))
	state.EvidenceRefs = compactServiceRefsV0(append(
		state.EvidenceRefs,
		"evidence-ref-app-director-operational-plan-state-closure-blocked-v0",
	))
	state.ClosureReason = reason
	state.Steps = nextSteps
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return err
	}
	return ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func operationalDirectorPlanStateClosureEvidenceRefsV0(
	request orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) []string {
	refs := []string{
		request.DeliveryRef,
		request.AcceptedReviewRef,
		request.ValidationRef,
		request.ClosureRef,
	}
	refs = append(refs, request.RequiredTestEvidenceRefs...)
	refs = append(refs, request.EvidenceRefs...)
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanStateClosureBlockerRefsV0(
	reason string,
	issues []orquestacionnucleoapp.ErrorV0,
) []string {
	refs := []string{reason}
	for _, issue := range issues {
		if field := strings.TrimSpace(issue.Field); field != "" {
			refs = append(refs, field)
		}
		if code := strings.TrimSpace(issue.Code); code != "" {
			refs = append(refs, code)
		}
	}
	return compactServiceRefsV0(refs)
}

func appDirectorClosureRequestWithDefaultsV0(
	request ContinueAppDirectorRequestV0,
	closureRequest orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) orquestacionnucleoapp.OperationalDirectorClosureRequestV0 {
	if closureRequest.RunRef == "" {
		closureRequest.RunRef = request.RunRef
	}
	if closureRequest.OccurredAt == "" {
		closureRequest.OccurredAt = request.OccurredAt
	}
	if closureRequest.CorrelationID == "" {
		closureRequest.CorrelationID = request.CorrelationID
	}
	if closureRequest.RequestedBy == "" {
		closureRequest.RequestedBy = request.RequestedBy
	}
	return closureRequest
}
