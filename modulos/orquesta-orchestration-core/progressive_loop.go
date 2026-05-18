package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func (service ServiceV0) RunProgressiveLoopV0(
	ctx context.Context,
	request ProgressiveLoopRequestV0,
) (ProgressiveLoopResultV0, error) {
	request = normalizeProgressiveLoopRequestV0(request)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := service.validateProgressiveLoopV0(request); err != nil {
		return ProgressiveLoopResultV0{}, err
	}
	result := ProgressiveLoopResultV0{}
	gate, err := service.evaluateProgressiveRunControlGateV0(ctx, request.RunRef)
	if err != nil {
		result.Status = ProgressiveLoopStatusStopErrorV0
		return service.finishProgressiveLoopV0(ctx, request, result), err
	}
	if !gate.Allowed {
		return service.finishProgressiveRunControlGateV0(ctx, request, result, gate)
	}
	for burstNumber := 1; burstNumber <= request.MaxBursts; burstNumber++ {
		burst, err := service.runProgressiveBurstV0(ctx, request, burstNumber)
		result = appendProgressiveBurstV0(result, burstNumber, burst)
		if err != nil {
			result.Status = ProgressiveLoopStatusStopErrorV0
			return service.finishProgressiveLoopV0(ctx, request, result), err
		}
		if burst.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
			result.Status = progressiveStatusFromActionV0(burst.Burst.FinalAction)
			return service.finishProgressiveLoopV0(ctx, request, result), nil
		}
		result = service.captureProgressiveFirstPendingV0(ctx, request, result)
		gate, err := service.evaluateProgressiveRunControlGateV0(ctx, request.RunRef)
		if err != nil {
			result.Status = ProgressiveLoopStatusStopErrorV0
			return service.finishProgressiveLoopV0(ctx, request, result), err
		}
		if !gate.Allowed {
			return service.finishProgressiveRunControlGateV0(ctx, request, result, gate)
		}
		dispatched, err := service.dispatchProgressiveWaitV0(ctx, request)
		result.Dispatches = append(result.Dispatches, dispatched.Once...)
		result.BatchDispatches = append(result.BatchDispatches, dispatched.Batch...)
		if err != nil {
			result.Status = ProgressiveLoopStatusStopErrorV0
			return service.finishProgressiveLoopV0(ctx, request, result), err
		}
		if !hasProgressiveDispatchProgressV0(dispatched) {
			result.Status = ProgressiveLoopStatusWaitUnhandledOutboxV0
			return service.finishProgressiveLoopV0(ctx, request, result), nil
		}
	}
	result = service.finishProgressiveLoopV0(ctx, request, result)
	result.Status = progressiveStatusAfterMaxBurstsV0(result.Run, request.WaitAgentRefs, request.WaitScopeApplied)
	return result, nil
}

func normalizeProgressiveLoopRequestV0(
	request ProgressiveLoopRequestV0,
) ProgressiveLoopRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	request.WaitAgentRefs = compactStringsV0(request.WaitAgentRefs)
	if request.MaxBursts <= 0 {
		request.MaxBursts = 1
	}
	if request.MaxStepsPerBurst <= 0 {
		request.MaxStepsPerBurst = 1
	}
	if request.MaxDispatchesPerWait <= 0 {
		request.MaxDispatchesPerWait = 1
	}
	return request
}

func (service ServiceV0) validateProgressiveLoopV0(
	request ProgressiveLoopRequestV0,
) error {
	return service.validateV0(SupervisedBurstRequestV0{
		RunRef:     request.RunRef,
		OccurredAt: request.OccurredAt,
		MaxSteps:   request.MaxStepsPerBurst,
	})
}
