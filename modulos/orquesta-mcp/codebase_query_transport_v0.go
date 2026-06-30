package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpCodebaseQueryTransportHandlerV0(
	executor MCPTransportCodebaseQueryExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPCodebaseQueryToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if executor == nil {
			return mcpTransportToolErrorPayloadV0(MCPCodebaseQueryToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
