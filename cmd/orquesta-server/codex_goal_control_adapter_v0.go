package main

import (
	"context"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
)

type serverCodexGoalControllerV0 interface {
	StopCodexGoalV0(context.Context, orquestaruntimecodexappserver.CodexGoalStopRequestV0) (orquestaruntimecodexappserver.CodexGoalStopResultV0, error)
}

type serverCodexGoalBackendControlV0 struct {
	Controller serverCodexGoalControllerV0
}

func serverGoalBackendControlFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestaappcodexstack.GoalBackendControlPortV0 {
	if backend.Controller == nil {
		return nil
	}
	return serverCodexGoalBackendControlV0{Controller: backend.Controller}
}

func (control serverCodexGoalBackendControlV0) ControlGoalBackendV0(
	ctx context.Context,
	request orquestaappcodexstack.GoalBackendControlRequestV0,
) (orquestaappcodexstack.GoalBackendControlResultV0, error) {
	result, err := control.Controller.StopCodexGoalV0(ctx, orquestaruntimecodexappserver.CodexGoalStopRequestV0{
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		Action:          request.Action,
		Reason:          request.Reason,
		Forced:          request.Forced,
		EvidenceRefs:    request.EvidenceRefs,
	})
	return orquestaappcodexstack.GoalBackendControlResultV0{
		Status:          result.Status,
		GoalRef:         result.GoalRef,
		ExternalGoalRef: result.ExternalGoalRef,
		GoalStatusSet:   result.GoalStatusSet,
		BackendStopped:  result.BackendStopped,
		IssueCode:       result.IssueCode,
		EvidenceRefs:    result.EvidenceRefs,
	}, err
}
