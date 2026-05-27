package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func autoprogrammingBridgeContinueRequestV0(
	request AutoprogrammingBridgeRequestV0,
	result AutoprogrammingBridgeResultV0,
) orquestaappdirectorservice.ContinueAppDirectorRequestV0 {
	return orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:               result.Run.RunID,
		OccurredAt:           request.OccurredAt,
		CorrelationID:        request.CorrelationID,
		RequestedBy:          request.RequestedBy,
		MaxBursts:            request.MaxBursts,
		MaxStepsPerBurst:     request.MaxStepsPerBurst,
		MaxDispatchesPerWait: request.MaxDispatchesPerWait,
		WaitAgentRefs:        append([]string(nil), result.WaitAgentRefs...),
		MaxCommands:          request.MaxCommands,
		MaxOutboxPerCycle:    request.MaxOutboxPerCycle,
	}
}

func autoprogrammingBridgeContinueRequestWithPlanStateV0(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
	result AutoprogrammingBridgeResultV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) (orquestaappdirectorservice.ContinueAppDirectorRequestV0, error) {
	continueRequest := autoprogrammingBridgeContinueRequestV0(request, result)
	stateRequest, err := orquestaappdirectorservice.EnsureOperationalDirectorPlanStateFromWorkflowTasksV0(
		ctx,
		orquestaappdirectorservice.ContinueAppDirectorRequestV0{
			RunRef:            result.Run.RunID,
			OccurredAt:        request.OccurredAt,
			CorrelationID:     request.CorrelationID,
			RequestedBy:       request.RequestedBy,
			MaxBursts:         request.MaxBursts,
			MaxStepsPerBurst:  request.MaxStepsPerBurst,
			MaxCommands:       request.MaxCommands,
			MaxOutboxPerCycle: request.MaxOutboxPerCycle,
		},
		ports,
	)
	if err != nil {
		return orquestaappdirectorservice.ContinueAppDirectorRequestV0{}, err
	}
	if strings.TrimSpace(stateRequest.OperationalDirectorPlanRef) != "" {
		continueRequest.OperationalDirectorPlanRef = stateRequest.OperationalDirectorPlanRef
	}
	return continueRequest, nil
}

func autoprogrammingBridgeWaitAgentRefsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	return compactStringsV0(refs)
}
