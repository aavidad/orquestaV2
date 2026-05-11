package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpRunControlTransportHandlerV0(
	port MCPTransportRunControlExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPRunControlToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPRunControlToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
