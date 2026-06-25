package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpObserveAppDirectorGoalTransportHandlerV0(
	port MCPTransportObserveAppDirectorGoalExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPObserveAppDirectorGoalToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPObserveAppDirectorGoalToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			if result.Estado == MCPObserveAppDirectorGoalEstadoErrorV0 && len(result.Errores) > 0 {
				return json.Marshal(result)
			}
			payload := NewMCPObserveAppDirectorGoalErrorResultV0(
				input,
				"observe_app_director_goal_execute_error",
				"executor",
				publicMCPExecutorErrorMessageFromErrorV0("observe_app_director_goal_execute_error", err),
			)
			return json.Marshal(payload)
		}
		return json.Marshal(result)
	}
}
