package orquestacionnucleoapp

import (
	"context"

	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func (service ServiceV0) ExecuteDirectorBriefingActionV0(
	ctx context.Context,
	request DirectorBriefingExecutionRequestV0,
) (DirectorBriefingExecutionResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return DirectorBriefingExecutionResultV0{}, err
	}
	request = normalizeDirectorBriefingExecutionRequestV0(request)
	action, err := selectDirectorBriefingExecutionActionV0(request)
	if err != nil {
		return DirectorBriefingExecutionResultV0{}, err
	}

	switch action.Kind {
	case orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0:
		return service.executeDirectorBriefingRunStepV0(ctx, request, action)
	case orquestadirectorsupervisor.DirectorSupervisorActionKindDispatchOutboxV0:
		return service.executeDirectorBriefingDispatchOutboxV0(ctx, request, action)
	case orquestadirectorsupervisor.DirectorSupervisorActionKindCloseOrIdleV0:
		if request.ExternalActionHandler != nil {
			return executeDirectorBriefingExternalActionV0(ctx, request, action)
		}
		return DirectorBriefingExecutionResultV0{
			Status:       DirectorBriefingExecutionStatusCloseOrIdlePendingV0,
			RunRef:       request.Briefing.RunRef,
			Action:       action,
			EvidenceRefs: mergeBriefingExecutionEvidenceV0(request, action),
		}, nil
	default:
		return executeDirectorBriefingExternalActionV0(ctx, request, action)
	}
}

func (service ServiceV0) executeDirectorBriefingRunStepV0(
	ctx context.Context,
	request DirectorBriefingExecutionRequestV0,
	action orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0,
) (DirectorBriefingExecutionResultV0, error) {
	if request.OccurredAt == "" {
		return DirectorBriefingExecutionResultV0{},
			errorV0(ErrDirectorBriefingExecutionInvalidV0, "occurred_at", "occurred_at requerido")
	}
	burst, err := service.RunSupervisedBurstV0(ctx, SupervisedBurstRequestV0{
		RunRef:        request.Briefing.RunRef,
		OccurredAt:    request.OccurredAt,
		MaxSteps:      1,
		CorrelationID: request.CorrelationID,
		EvidenceRefs:  mergeBriefingExecutionEvidenceV0(request, action),
		WaitAgentRefs: request.WaitAgentRefs,
	})
	result := DirectorBriefingExecutionResultV0{
		Status:       DirectorBriefingExecutionStatusRunStepV0,
		RunRef:       request.Briefing.RunRef,
		Action:       action,
		Burst:        &burst.Burst,
		Run:          &burst.Run,
		EvidenceRefs: burst.Burst.EvidenceRefs,
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func (service ServiceV0) executeDirectorBriefingDispatchOutboxV0(
	ctx context.Context,
	request DirectorBriefingExecutionRequestV0,
	action orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0,
) (DirectorBriefingExecutionResultV0, error) {
	maxDispatches := request.MaxDispatchesPerWait
	if maxDispatches <= 0 {
		maxDispatches = 1
	}
	dispatched, err := service.dispatchProgressiveWaitV0(ctx, ProgressiveLoopRequestV0{
		RunRef:               request.Briefing.RunRef,
		OccurredAt:           request.OccurredAt,
		MaxDispatchesPerWait: maxDispatches,
		CorrelationID:        request.CorrelationID,
		EvidenceRefs:         mergeBriefingExecutionEvidenceV0(request, action),
		WaitAgentRefs:        request.WaitAgentRefs,
		WaitScopeApplied:     request.WaitScopeApplied,
		Dispatchers:          scopedDirectorBriefingDispatchersV0(request.Dispatchers, action.TargetRefs),
		BatchDispatchers:     scopedDirectorBriefingBatchDispatchersV0(request.BatchDispatchers, action.TargetRefs),
	})
	status := DirectorBriefingExecutionStatusOutboxNoProgressV0
	if hasProgressiveDispatchProgressV0(dispatched) {
		status = DirectorBriefingExecutionStatusOutboxDispatchedV0
	}
	result := DirectorBriefingExecutionResultV0{
		Status:          status,
		RunRef:          request.Briefing.RunRef,
		Action:          action,
		Dispatches:      dispatched.Once,
		BatchDispatches: dispatched.Batch,
		EvidenceRefs:    mergeBriefingDispatchEvidenceV0(request, action, dispatched),
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func executeDirectorBriefingExternalActionV0(
	ctx context.Context,
	request DirectorBriefingExecutionRequestV0,
	action orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0,
) (DirectorBriefingExecutionResultV0, error) {
	if request.ExternalActionHandler == nil {
		return DirectorBriefingExecutionResultV0{
			Status:       DirectorBriefingExecutionStatusExternalPendingV0,
			RunRef:       request.Briefing.RunRef,
			Action:       action,
			EvidenceRefs: mergeBriefingExecutionEvidenceV0(request, action),
		}, nil
	}
	external, err := request.ExternalActionHandler.ExecuteDirectorBriefingExternalActionV0(
		ctx,
		DirectorBriefingExternalActionRequestV0{
			Briefing:      request.Briefing,
			Action:        action,
			OccurredAt:    request.OccurredAt,
			CorrelationID: request.CorrelationID,
			EvidenceRefs:  mergeBriefingExecutionEvidenceV0(request, action),
		},
	)
	if external.RunRef == "" {
		external.RunRef = request.Briefing.RunRef
	}
	if external.ActionRef == "" {
		external.ActionRef = action.ActionRef
	}
	status := external.Status
	if err != nil {
		status = DirectorBriefingExecutionStatusExternalFailedV0
		if external.Status == "" {
			external.Status = DirectorBriefingExecutionStatusExternalFailedV0
		}
	} else if external.Status == "" {
		external.Status = DirectorBriefingExecutionStatusExternalAppliedV0
		status = external.Status
	}
	result := DirectorBriefingExecutionResultV0{
		Status:       status,
		RunRef:       request.Briefing.RunRef,
		Action:       action,
		External:     &external,
		EvidenceRefs: compactStringsV0(append(mergeBriefingExecutionEvidenceV0(request, action), external.EvidenceRefs...)),
	}
	if err != nil {
		return result, err
	}
	return result, nil
}
