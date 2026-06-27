package orquestaappdirectorservice

import (
	"context"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type ContinueAppDirectorLoopRuntimeV0 struct {
	Request      ContinueAppDirectorRequestV0
	Service      orquestacionnucleoapp.ServiceV0
	LoopRequest  orquestacionnucleoapp.ProgressiveLoopRequestV0
	Closed       bool
	ClosedResult ContinueAppDirectorResultV0
}

func BuildContinueAppDirectorLoopRuntimeV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorLoopRuntimeV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeContinueAppDirectorRequestV0(request)
	now, err := startAppDirectorNowV0(request.OccurredAt)
	if err != nil {
		return ContinueAppDirectorLoopRuntimeV0{}, err
	}
	if request.OccurredAt == "" {
		request.OccurredAt = now.Format("2006-01-02T15:04:05Z07:00")
	}
	if result, handled, err := continueAppDirectorGoalFirstContainerResultV0(ctx, request, ports); err != nil || handled {
		return ContinueAppDirectorLoopRuntimeV0{
			Request:      request,
			Closed:       handled,
			ClosedResult: result,
		}, err
	}
	if err := validateContinueAppDirectorRequestV0(request, ports); err != nil {
		return ContinueAppDirectorLoopRuntimeV0{}, err
	}
	if result, closed, err := closedOperationalDirectorPlanStateContinueResultV0(ctx, request, ports); err != nil || closed {
		return ContinueAppDirectorLoopRuntimeV0{
			Request:      request,
			Closed:       closed,
			ClosedResult: result,
		}, err
	}
	materialized, err := materializeContinueOperationalDirectorPlanV0(ctx, request, ports)
	if err != nil {
		return ContinueAppDirectorLoopRuntimeV0{}, err
	}
	request = continueRequestWithOperationalDirectorWaitV0(request, materialized)
	request, err = continueRequestWithOperationalDirectorPlanStateV0(ctx, request, ports)
	if err != nil {
		return ContinueAppDirectorLoopRuntimeV0{}, err
	}
	if err := ensureOperationalDirectorPassedRequiredTestsQualityGateV0(ctx, request, ports); err != nil {
		return ContinueAppDirectorLoopRuntimeV0{}, err
	}
	if _, err := ensureOperationalDirectorReviewPhaseV0(ctx, request, ports); err != nil {
		return ContinueAppDirectorLoopRuntimeV0{}, err
	}
	loopRequest, err := existingDirectorLoopRequestV0(ctx, request, ports)
	if err != nil {
		return ContinueAppDirectorLoopRuntimeV0{}, err
	}
	return ContinueAppDirectorLoopRuntimeV0{
		Request:     request,
		Service:     existingDirectorLoopServiceV0(request, ports),
		LoopRequest: loopRequest,
	}, nil
}
