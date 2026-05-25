package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpDirectorStatsTransportHandlerV0(
	port MCPTransportDirectorStatsExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPDirectorStatsToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return mcpTransportToolErrorPayloadV0(MCPDirectorStatsToolNameV0, MCPTransportToolInputInvalidV0)
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPDirectorStatsToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			if result.Estado == MCPDirectorStatsEstadoErrorV0 && len(result.Errores) > 0 {
				return json.Marshal(result)
			}
			payload := newMCPDirectorStatsHTTPErrorV0(
				"executor",
				publicMCPExecutorErrorMessageFromErrorV0("director_stats_executor_error", err),
				firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
			)
			payload.RunRef = input.RunRef
			return json.Marshal(payload)
		}
		return json.Marshal(result)
	}
}
