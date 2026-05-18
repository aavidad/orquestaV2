package orquestacionnucleoapp

import (
	"context"
	"fmt"

	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func (service ServiceV0) runProgressiveBurstV0(
	ctx context.Context,
	request ProgressiveLoopRequestV0,
	burstNumber int,
) (SupervisedBurstResultV0, error) {
	return service.RunSupervisedBurstV0(ctx, SupervisedBurstRequestV0{
		RunRef:        request.RunRef,
		OccurredAt:    request.OccurredAt,
		MaxSteps:      request.MaxStepsPerBurst,
		CorrelationID: progressiveCorrelationIDV0(request, burstNumber),
		EvidenceRefs:  request.EvidenceRefs,
		WaitAgentRefs: request.WaitAgentRefs,
	})
}

func appendProgressiveBurstV0(
	result ProgressiveLoopResultV0,
	burstNumber int,
	burst SupervisedBurstResultV0,
) ProgressiveLoopResultV0 {
	result.Run = burst.Run
	result.TotalExecutedSteps += burst.Burst.ExecutedSteps
	result.FinalAction = burst.Burst.FinalAction
	result.Bursts = append(result.Bursts, ProgressiveLoopBurstV0{
		BurstNumber:  burstNumber,
		Executed:     burst.Burst.ExecutedSteps,
		FinalAction:  burst.Burst.FinalAction,
		EvidenceRefs: append([]string(nil), burst.Burst.EvidenceRefs...),
	})
	return result
}

func (service ServiceV0) finishProgressiveLoopV0(
	ctx context.Context,
	request ProgressiveLoopRequestV0,
	result ProgressiveLoopResultV0,
) ProgressiveLoopResultV0 {
	if run, err := service.RunStore.LoadRunV0(ctx, request.RunRef); err == nil {
		result.Run = run
	}
	pending, issues := service.OutboxLedger.ListPending(ctx,
		orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
			RunRef: request.RunRef,
		},
	)
	if len(issues) == 0 {
		result.PendingOutboxRefs = outboxRefsV0(pending)
		result.PendingOutboxCount = len(pending)
	}
	return result
}

func (service ServiceV0) captureProgressiveFirstPendingV0(
	ctx context.Context,
	request ProgressiveLoopRequestV0,
	result ProgressiveLoopResultV0,
) ProgressiveLoopResultV0 {
	if result.FirstPendingCount > 0 || len(result.FirstPendingRefs) > 0 {
		return result
	}
	pending, issues := service.OutboxLedger.ListPending(ctx,
		orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
			RunRef: request.RunRef,
		},
	)
	if len(issues) == 0 {
		result.FirstPendingRefs = outboxRefsV0(pending)
		result.FirstPendingCount = len(pending)
	}
	return result
}

func progressiveStatusFromActionV0(
	action orquestadirectorsupervisor.DirectorSupervisorActionV0,
) ProgressiveLoopStatusV0 {
	switch action {
	case orquestadirectorsupervisor.DirectorSupervisorActionStopQuiescentV0:
		return ProgressiveLoopStatusQuiescentV0
	case orquestadirectorsupervisor.DirectorSupervisorActionWaitExternalV0:
		return ProgressiveLoopStatusWaitExternalV0
	case orquestadirectorsupervisor.DirectorSupervisorActionNeedsDirectorV0:
		return ProgressiveLoopStatusNeedsDirectorV0
	case orquestadirectorsupervisor.DirectorSupervisorActionBlockedV0:
		return ProgressiveLoopStatusBlockedV0
	case orquestadirectorsupervisor.DirectorSupervisorActionStopMaxStepsV0:
		return ProgressiveLoopStatusStopMaxStepsV0
	case orquestadirectorsupervisor.DirectorSupervisorActionStopErrorV0:
		return ProgressiveLoopStatusStopErrorV0
	default:
		return ProgressiveLoopStatusStopErrorV0
	}
}

func progressiveCorrelationIDV0(
	request ProgressiveLoopRequestV0,
	burstNumber int,
) string {
	if request.CorrelationID != "" {
		return fmt.Sprintf("%s-burst-%03d", request.CorrelationID, burstNumber)
	}
	return fmt.Sprintf("corr-%s-burst-%03d", request.RunRef, burstNumber)
}
