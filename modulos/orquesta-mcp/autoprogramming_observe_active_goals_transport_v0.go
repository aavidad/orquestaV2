package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpAutoprogrammingObserveActiveGoalsTransportHandlerV0(
	port MCPTransportAutoprogrammingObserveActiveGoalsExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPAutoprogrammingObserveActiveGoalsToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPAutoprogrammingObserveActiveGoalsToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			payload := mcpAutoprogrammingObserveActiveGoalsErrorV0(
				input,
				"autoprogramming_observe_active_goals_execute_error",
				"executor",
				publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_observe_active_goals_execute_error", err),
			)
			return json.Marshal(payload)
		}
		return json.Marshal(result)
	}
}
