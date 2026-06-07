package orquestamcp

import (
	"context"
	"encoding/json"
)

type MCPTransportDirectorSupervisorBriefingExecutorV0 interface {
	Execute(
		context.Context,
		MCPDirectorSupervisorBriefingToolInputV0,
	) (MCPDirectorSupervisorBriefingToolResultV0, error)
}

func mcpDirectorSupervisorBriefingTransportHandlerV0(
	executor MCPTransportDirectorSupervisorBriefingExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPDirectorSupervisorBriefingToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return mcpTransportToolErrorPayloadV0(MCPDirectorSupervisorBriefingToolNameV0, MCPTransportToolInputInvalidV0)
		}
		activeExecutor := executor
		if activeExecutor == nil {
			activeExecutor = MCPDirectorSupervisorBriefingToolExecutorV0{}
		}
		result, err := activeExecutor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
