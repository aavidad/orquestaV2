package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpServerShutdownTransportHandlerV0(
	port MCPTransportServerShutdownExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPServerShutdownToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPServerShutdownToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			if result.Estado == MCPServerShutdownEstadoErrorV0 && len(result.Errores) > 0 {
				return json.Marshal(result)
			}
			payload := newMCPServerShutdownErrorV0(
				input,
				"server_shutdown_executor_error",
				"executor",
				publicMCPExecutorErrorMessageFromErrorV0("server_shutdown_executor_error", err),
			)
			return json.Marshal(payload)
		}
		return json.Marshal(result)
	}
}
