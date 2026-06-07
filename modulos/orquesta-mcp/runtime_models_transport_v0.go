package orquestamcp

import (
	"context"
	"encoding/json"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func mcpRuntimeModelsTransportHandlerV0(
	port orquestaruntime.RuntimeModelManagerPortV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPRuntimeModelsToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPRuntimeModelsToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := (MCPRuntimeModelsToolExecutorV0{Port: port}).Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
