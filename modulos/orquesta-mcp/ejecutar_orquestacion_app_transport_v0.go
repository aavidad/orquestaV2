package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpEjecutarOrquestacionAppTransportHandlerV0(
	port MCPTransportEjecutarOrquestacionAppExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPEjecutarOrquestacionAppToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPEjecutarOrquestacionAppToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
