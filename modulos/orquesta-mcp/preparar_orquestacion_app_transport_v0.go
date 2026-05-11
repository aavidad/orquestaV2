package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpPrepararOrquestacionAppTransportHandlerV0(
	_ context.Context,
	raw json.RawMessage,
) (json.RawMessage, error) {
	var input MCPPrepararOrquestacionAppToolInputV0
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, err
	}
	return json.Marshal(ExecuteMCPPrepararOrquestacionAppToolV0(input))
}
