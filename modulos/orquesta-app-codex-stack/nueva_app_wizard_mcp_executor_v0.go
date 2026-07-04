package orquestaappcodexstack

import (
	"context"
	"encoding/json"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaweb "orquesta/modulos/orquesta-web"
)

type CodexStackNuevaAppWizardExecutorV0 struct {
	Assistant orquestaweb.WebNuevaAppIntakeAssistantPortV0
}

func NewCodexStackNuevaAppWizardExecutorV0(
	assistant orquestaweb.WebNuevaAppIntakeAssistantPortV0,
) CodexStackNuevaAppWizardExecutorV0 {
	return CodexStackNuevaAppWizardExecutorV0{Assistant: assistant}
}

func (executor CodexStackNuevaAppWizardExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPNuevaAppWizardToolInputV0,
) (orquestamcp.MCPNuevaAppWizardToolResultV0, error) {
	input = orquestamcp.NormalizeMCPNuevaAppWizardInputV0(input)
	payload, err := json.Marshal(input)
	if err != nil {
		return orquestamcp.MCPNuevaAppWizardToolResultV0{}, err
	}
	var request orquestaweb.WebNuevaAppIntakeGuidedRequestV0
	if err := json.Unmarshal(payload, &request); err != nil {
		return orquestamcp.MCPNuevaAppWizardToolResultV0{}, err
	}
	if request.SessionID == "" {
		request.SessionID = input.SessionID
	}
	response := orquestaweb.NewNuevaAppIntakeGuidedHTTPHandlerWithAssistantV0(
		executor.Assistant,
	).NewResponseV0(ctx, request)
	rawResponse, err := json.Marshal(response)
	if err != nil {
		return orquestamcp.MCPNuevaAppWizardToolResultV0{}, err
	}
	return orquestamcp.NewMCPNuevaAppWizardResultFromJSONV0(input, rawResponse)
}
