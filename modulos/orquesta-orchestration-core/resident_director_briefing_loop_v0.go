package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func (service ServiceV0) RunResidentDirectorBriefingLoopV0(
	ctx context.Context,
	request ResidentDirectorBriefingLoopRequestV0,
) (ResidentDirectorBriefingLoopResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ResidentDirectorBriefingLoopResultV0{}, err
	}
	request = normalizeResidentDirectorBriefingLoopRequestV0(request)
	if err := validateResidentDirectorBriefingLoopRequestV0(request); err != nil {
		return ResidentDirectorBriefingLoopResultV0{}, err
	}

	result := ResidentDirectorBriefingLoopResultV0{
		Status:       ResidentDirectorBriefingLoopStatusCompletedV0,
		RunRef:       request.RunRef,
		EvidenceRefs: compactStringsV0(request.EvidenceRefs),
	}
	var previous *ResidentDirectorBriefingLoopStepV0

	for stepNumber := 1; stepNumber <= request.MaxActions; stepNumber++ {
		briefing, err := request.BriefingSource.BuildResidentDirectorBriefingV0(
			ctx,
			ResidentDirectorBriefingBuildRequestV0{
				RunRef:          request.RunRef,
				ObjectiveRef:    request.ObjectiveRef,
				ContextRefs:     request.ContextRefs,
				StepNumber:      stepNumber,
				PreviousStep:    previous,
				CorrelationID:   request.CorrelationID,
				EvidenceRefs:    result.EvidenceRefs,
				LastExecutionID: residentDirectorBriefingLastExecutionIDV0(previous),
			},
		)
		if err != nil {
			return result, err
		}
		briefing = normalizeResidentDirectorBriefingV0(briefing, request)
		if err := validateResidentDirectorBriefingV0(briefing, request.RunRef); err != nil {
			return result, err
		}

		step := ResidentDirectorBriefingLoopStepV0{
			StepNumber: stepNumber,
			Briefing:   briefing,
		}
		result.LastBriefing = &step.Briefing

		action := residentDirectorBriefingNextActionV0(briefing)
		if action == nil {
			result.Status = ResidentDirectorBriefingLoopStatusCompletedV0
			result.Steps = append(result.Steps, step)
			return result, nil
		}
		if !residentDirectorBriefingActionAutoApplicableV0(*action) {
			result.Status = ResidentDirectorBriefingLoopStatusNeedsDirectorV0
			result.Steps = append(result.Steps, step)
			return result, nil
		}

		execution, err := service.ExecuteDirectorBriefingActionV0(
			ctx,
			DirectorBriefingExecutionRequestV0{
				Briefing:              briefing,
				ActionRef:             action.ActionRef,
				OccurredAt:            request.OccurredAt,
				CorrelationID:         request.CorrelationID,
				EvidenceRefs:          result.EvidenceRefs,
				WaitAgentRefs:         request.WaitAgentRefs,
				WaitScopeApplied:      request.WaitScopeApplied,
				MaxDispatchesPerWait:  request.MaxDispatchesPerWait,
				Dispatchers:           request.Dispatchers,
				BatchDispatchers:      request.BatchDispatchers,
				ExternalActionHandler: request.ExternalActionHandler,
			},
		)
		step.Execution = &execution
		result.Steps = append(result.Steps, step)
		result.LastExecution = step.Execution
		result.ExecutedActions++
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, execution.EvidenceRefs...))
		if err != nil {
			result.Status = residentDirectorBriefingLoopStatusForExecutionV0(execution)
			return result, err
		}

		status := residentDirectorBriefingLoopStatusForExecutionV0(execution)
		switch status {
		case "":
			previous = &result.Steps[len(result.Steps)-1]
		case ResidentDirectorBriefingLoopStatusCompletedV0,
			ResidentDirectorBriefingLoopStatusExternalPendingV0,
			ResidentDirectorBriefingLoopStatusExternalFailedV0,
			ResidentDirectorBriefingLoopStatusNoProgressV0:
			result.Status = status
			return result, nil
		default:
			result.Status = status
			return result, nil
		}
	}

	result.Status = ResidentDirectorBriefingLoopStatusBudgetExhaustedV0
	return result, nil
}

func normalizeResidentDirectorBriefingLoopRequestV0(
	request ResidentDirectorBriefingLoopRequestV0,
) ResidentDirectorBriefingLoopRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.ObjectiveRef = strings.TrimSpace(request.ObjectiveRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.ContextRefs = compactStringsV0(request.ContextRefs)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	request.WaitAgentRefs = compactStringsV0(request.WaitAgentRefs)
	if request.MaxActions <= 0 {
		request.MaxActions = 1
	}
	return request
}

func validateResidentDirectorBriefingLoopRequestV0(
	request ResidentDirectorBriefingLoopRequestV0,
) error {
	if request.RunRef == "" {
		return errorV0(ErrResidentDirectorBriefingLoopInvalidV0, "run_ref", "run_ref requerido")
	}
	if request.BriefingSource == nil {
		return errorV0(ErrResidentDirectorBriefingLoopInvalidV0, "briefing_source", "briefing_source requerido")
	}
	if request.MaxActions < 1 {
		return errorV0(ErrResidentDirectorBriefingLoopInvalidV0, "max_actions", "max_actions debe ser mayor que cero")
	}
	return nil
}

func normalizeResidentDirectorBriefingV0(
	briefing orquestadirectorsupervisor.DirectorSupervisorBriefingV0,
	request ResidentDirectorBriefingLoopRequestV0,
) orquestadirectorsupervisor.DirectorSupervisorBriefingV0 {
	briefing.SchemaVersion = strings.TrimSpace(briefing.SchemaVersion)
	if briefing.SchemaVersion == "" {
		briefing.SchemaVersion = orquestadirectorsupervisor.DirectorSupervisorBriefingSchemaV0
	}
	briefing.RunRef = strings.TrimSpace(briefing.RunRef)
	if briefing.RunRef == "" {
		briefing.RunRef = request.RunRef
	}
	if briefing.ObjectiveRef == "" {
		briefing.ObjectiveRef = request.ObjectiveRef
	}
	briefing.ContextRefs = compactStringsV0(append(briefing.ContextRefs, request.ContextRefs...))
	briefing.EvidenceRefs = compactStringsV0(append(briefing.EvidenceRefs, request.EvidenceRefs...))
	return briefing
}

func validateResidentDirectorBriefingV0(
	briefing orquestadirectorsupervisor.DirectorSupervisorBriefingV0,
	runRef string,
) error {
	if briefing.SchemaVersion != orquestadirectorsupervisor.DirectorSupervisorBriefingSchemaV0 {
		return errorV0(ErrResidentDirectorBriefingLoopInvalidV0, "briefing.schema_version", "schema_version invalido")
	}
	if strings.TrimSpace(briefing.RunRef) != runRef {
		return errorV0(ErrResidentDirectorBriefingLoopInvalidV0, "briefing.run_ref", "briefing de otro run")
	}
	action := residentDirectorBriefingNextActionV0(briefing)
	if action == nil {
		return nil
	}
	if strings.TrimSpace(action.RunRef) != runRef {
		return errorV0(ErrResidentDirectorBriefingLoopInvalidV0, "briefing.next_action.run_ref", "accion de otro run")
	}
	return nil
}

func residentDirectorBriefingNextActionV0(
	briefing orquestadirectorsupervisor.DirectorSupervisorBriefingV0,
) *orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0 {
	if briefing.NextAction != nil {
		return briefing.NextAction
	}
	if len(briefing.ActionQueue) == 0 {
		return nil
	}
	return &briefing.ActionQueue[0]
}

func residentDirectorBriefingActionAutoApplicableV0(
	action orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0,
) bool {
	if action.RequiresDirector {
		return false
	}
	return action.SafeToApply
}

func residentDirectorBriefingLoopStatusForExecutionV0(
	execution DirectorBriefingExecutionResultV0,
) string {
	switch execution.Status {
	case DirectorBriefingExecutionStatusRunStepV0,
		DirectorBriefingExecutionStatusOutboxDispatchedV0,
		DirectorBriefingExecutionStatusExternalAppliedV0:
		return ""
	case DirectorBriefingExecutionStatusCloseOrIdlePendingV0:
		return ResidentDirectorBriefingLoopStatusCompletedV0
	case DirectorBriefingExecutionStatusExternalPendingV0:
		return ResidentDirectorBriefingLoopStatusExternalPendingV0
	case DirectorBriefingExecutionStatusExternalFailedV0:
		return ResidentDirectorBriefingLoopStatusExternalFailedV0
	case DirectorBriefingExecutionStatusOutboxNoProgressV0:
		return ResidentDirectorBriefingLoopStatusNoProgressV0
	default:
		return ResidentDirectorBriefingLoopStatusNoProgressV0
	}
}

func residentDirectorBriefingLastExecutionIDV0(
	previous *ResidentDirectorBriefingLoopStepV0,
) string {
	if previous == nil || previous.Execution == nil {
		return ""
	}
	return previous.Execution.Action.ActionRef
}
