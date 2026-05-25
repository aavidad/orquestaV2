package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpRunSupervisorTransportHandlerV0(
	port MCPTransportRunSupervisorExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPRunSupervisorToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPRunSupervisorToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			if result.Estado == MCPRunSupervisorEstadoErrorV0 && len(result.Errores) > 0 {
				return json.Marshal(result)
			}
			payload := NewMCPRunSupervisorErrorResultV0(
				input,
				"run_supervisor_execute_error",
				"executor",
				publicMCPExecutorErrorMessageFromErrorV0("run_supervisor_execute_error", err),
			)
			return json.Marshal(payload)
		}
		return json.Marshal(result)
	}
}
