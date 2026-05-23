package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpAppVCSTransportHandlerV0(port MCPAppVCSExecutorPortV0) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPAppVCSToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPAppVCSToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := NewMCPAppVCSToolExecutorV0(port).Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
