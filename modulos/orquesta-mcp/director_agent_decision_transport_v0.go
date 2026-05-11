package orquestamcp

import (
	"context"
	"encoding/json"
)

type MCPTransportDirectorAgentDecisionExecutorV0 interface {
	Execute(
		context.Context,
		MCPDirectorAgentDecisionToolInputV0,
	) (MCPDirectorAgentDecisionToolResultV0, error)
}

func mcpDirectorAgentDecisionTransportHandlerV0(
	port MCPTransportDirectorAgentDecisionExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPDirectorAgentDecisionToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPDirectorAgentDecisionToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
