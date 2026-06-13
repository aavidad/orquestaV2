package orquestaappdirectorservice

import (
	"context"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func continueOperationalDirectorPlanStatePostLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
	managedLoop orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) (existingDirectorAutonomyLoopResultV0, error) {
	var err error
	loop, err = operationalDirectorPlanStatePostLoopScopeV0(ctx, request, ports, loop, loopRequest)
	if err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	if loop.Status == orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 && loop.PendingOutboxCount > 0 {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, nil
	}
	changed, err := applyOperationalDirectorPlanStateAfterLoopV0(ctx, request, ports, loop)
	if err != nil || !changed {
		if err != nil {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
		expired, err := applyOperationalDirectorPlanStateAfterExternalWaitExhaustedV0(ctx, request, ports, managedLoop)
		if err != nil || expired {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	loop, err = operationalDirectorPlanStateActiveWaitLoopV0(ctx, request, ports, loop)
	if err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	_, _, activeReview, err := operationalDirectorActiveReviewDeliveriesStepV0(ctx, request, ports)
	if err != nil || !activeReview {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	nextRequest, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, ports)
	if err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	if _, err := ensureOperationalDirectorReviewPhaseV0(ctx, nextRequest, ports); err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	followup, err := runExistingDirectorAutonomyLoopV0(ctx, nextRequest, ports)
	if err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: nextRequest, Loop: followup.Loop, LoopRequest: followup.LoopRequest}, err
	}
	followupLoop := operationalDirectorScopedQuiescentLoopV0(followup.Request, followup.Loop, followup.LoopRequest)
	if _, err := applyOperationalDirectorPlanStateAfterLoopV0(ctx, followup.Request, ports, followupLoop); err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: followup.Request, Loop: followupLoop, LoopRequest: followup.LoopRequest}, err
	}
	return existingDirectorAutonomyLoopResultV0{Request: followup.Request, Loop: followupLoop, LoopRequest: followup.LoopRequest}, nil
}

func operationalDirectorPlanStatePostLoopScopeV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, error) {
	if continueOperationalDirectorPlanRefV0(request) == "" ||
		loop.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		loop.PendingOutboxCount > 0 {
		return loop, nil
	}
	waitAgentRefs := compactServiceRefsV0(loopRequest.WaitAgentRefs)
	if len(waitAgentRefs) == 0 {
		refs, err := operationalDirectorPlanStateActiveWaitScopeAgentRefsV0(ctx, request, ports)
		if err != nil {
			return loop, err
		}
		waitAgentRefs = refs
	}
	if len(waitAgentRefs) == 0 {
		return loop, nil
	}
	scopedRun := loop.Run
	if ports.RunStore != nil {
		latestRun, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
		if err != nil {
			return loop, err
		}
		scopedRun = latestRun
	}
	if appDirectorRunHasPendingAgentRefsV0(scopedRun, waitAgentRefs) {
		return loop, nil
	}
	loop.Run = scopedRun
	loop.Status = orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0
	return loop, nil
}

func operationalDirectorPlanStateActiveWaitLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, error) {
	if continueOperationalDirectorPlanRefV0(request) == "" || ports.OperationalPlanStateStore == nil {
		return loop, nil
	}
	waitAgentRefs, err := operationalDirectorPlanStateActiveWaitAgentRefsV0(ctx, request, ports)
	if err != nil || len(waitAgentRefs) == 0 {
		return loop, err
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, continueOperationalDirectorPlanRefV0(request))
	if err != nil {
		return loop, err
	}
	scopedRun := loop.Run
	if ports.RunStore != nil {
		latestRun, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
		if err != nil {
			return loop, err
		}
		scopedRun = latestRun
	}
	loop.Run = scopedRun
	if (loop.Status == orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		loop.Status == orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0) &&
		appDirectorRunHasPendingAgentRefsV0(scopedRun, waitAgentRefs) {
		loop.Status = orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0
		if err := persistOperationalDirectorPlanActiveWaitStateV0(ctx, request, ports, state); err != nil {
			return loop, err
		}
	}
	return loop, nil
}
