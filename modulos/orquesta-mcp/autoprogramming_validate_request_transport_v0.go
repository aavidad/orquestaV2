package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpAutoprogrammingValidateRequestTransportHandlerV0(
	executor MCPAutoprogrammingValidateRequestToolExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPAutoprogrammingValidateRequestToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
