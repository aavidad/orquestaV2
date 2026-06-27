package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpAutoprogrammingObserveGoalTransportHandlerV0(
	port MCPTransportAutoprogrammingObserveGoalExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPAutoprogrammingObserveGoalToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPAutoprogrammingObserveGoalToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			if result.Estado == MCPAutoprogrammingObserveGoalEstadoErrorV0 && len(result.Errores) > 0 {
				return json.Marshal(result)
			}
			if publicResult, ok := NewMCPAutoprogrammingObserveGoalErrorResultFromErrorV0(input, err); ok {
				return json.Marshal(publicResult)
			}
			payload := NewMCPAutoprogrammingObserveGoalErrorResultV0(
				input,
				"autoprogramming_observe_goal_execute_error",
				"executor",
				publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_observe_goal_execute_error", err),
			)
			return json.Marshal(payload)
		}
		return json.Marshal(result)
	}
}
