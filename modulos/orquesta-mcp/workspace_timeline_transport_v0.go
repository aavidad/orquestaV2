package orquestamcp

import (
	"context"
	"encoding/json"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func mcpWorkspaceTimelineTransportHandlerV0(
	source orquestaobservability.WorkspaceTimelineSourcePortV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input orquestaobservability.WorkspaceTimelineQueryV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		result, err := (MCPWorkspaceTimelineToolExecutorV0{Source: source}).Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
