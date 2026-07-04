package orquestamcp

import (
	"context"
	"encoding/json"
)

func mcpNuevaAppWizardTransportHandlerV0(
	port MCPNuevaAppWizardExecutorPortV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPNuevaAppWizardToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPNuevaAppWizardToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := NewMCPNuevaAppWizardToolExecutorV0(port).Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
