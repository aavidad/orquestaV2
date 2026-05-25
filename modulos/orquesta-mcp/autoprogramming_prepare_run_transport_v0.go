package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpAutoprogrammingPrepareRunTransportHandlerV0(
	port MCPTransportAutoprogrammingPrepareRunExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPAutoprogrammingPrepareRunToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPAutoprogrammingPrepareRunToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			if result.Estado == MCPAutoprogrammingPrepareRunEstadoErrorV0 && len(result.Errores) > 0 {
				return json.Marshal(result)
			}
			payload := NewMCPAutoprogrammingPrepareRunErrorResultV0(
				input,
				"autoprogramming_prepare_run_executor_error",
				"executor",
				publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_prepare_run_executor_error", err),
			)
			return json.Marshal(payload)
		}
		return json.Marshal(result)
	}
}
