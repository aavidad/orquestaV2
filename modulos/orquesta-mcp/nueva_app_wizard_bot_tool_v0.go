package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
)

const (
	MCPNuevaAppWizardBotToolNameV0    = "orquesta.nueva_app.wizard.bot.v0"
	MCPNuevaAppWizardBotToolVersionV0 = "v0"
	MCPNuevaAppWizardBotResourceURIV0 = "orquesta://contracts/nueva-app-wizard-bot/v0"
	MCPNuevaAppWizardBotEstadoOKV0    = "ok"
	MCPNuevaAppWizardBotEstadoErrorV0 = "error"
)

type MCPNuevaAppWizardBotToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPNuevaAppWizardBotToolInputV0 struct {
	RequestID     string          `json:"request_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	SessionRef    string          `json:"session_ref,omitempty"`
	UserText      string          `json:"user_text"`
	Locale        string          `json:"locale,omitempty"`
	Session       json.RawMessage `json:"session,omitempty"`
}

type MCPNuevaAppWizardBotToolResultV0 struct {
	Estado        string                 `json:"estado"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Reply         json.RawMessage        `json:"reply,omitempty"`
	Session       json.RawMessage        `json:"session,omitempty"`
	Errores       []MCPValidationIssueV0 `json:"errores_publicos,omitempty"`
}

type MCPNuevaAppWizardBotExecutorPortV0 interface {
	Execute(context.Context, MCPNuevaAppWizardBotToolInputV0) (MCPNuevaAppWizardBotToolResultV0, error)
}

type MCPNuevaAppWizardBotToolExecutorV0 struct {
	Port MCPNuevaAppWizardBotExecutorPortV0
}

func NewMCPNuevaAppWizardBotToolExecutorV0(port MCPNuevaAppWizardBotExecutorPortV0) MCPNuevaAppWizardBotToolExecutorV0 {
	return MCPNuevaAppWizardBotToolExecutorV0{Port: port}
}

func MCPNuevaAppWizardBotDescriptorV0() MCPNuevaAppWizardBotToolDescriptorV0 {
	return MCPNuevaAppWizardBotToolDescriptorV0{
		Name:        MCPNuevaAppWizardBotToolNameV0,
		Version:     MCPNuevaAppWizardBotToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,session_ref?,user_text,locale?,session?}",
		Output:      "ok:{reply,session}|error:{errores_publicos}",
		ResourceURI: MCPNuevaAppWizardBotResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"delega en el bot del wizard de nueva app",
			"no inventa preguntas ni opciones fuera del motor",
			"funciona sin proveedor LLM",
		},
	}
}

func (executor MCPNuevaAppWizardBotToolExecutorV0) Execute(
	ctx context.Context,
	input MCPNuevaAppWizardBotToolInputV0,
) (MCPNuevaAppWizardBotToolResultV0, error) {
	input = NormalizeMCPNuevaAppWizardBotInputV0(input)
	if executor.Port == nil {
		return NewMCPNuevaAppWizardBotErrorResultV0(input, MCPTransportToolUnboundV0, "executor"), nil
	}
	result, err := executor.Port.Execute(ctx, input)
	if err != nil {
		return result, err
	}
	return NormalizeMCPNuevaAppWizardBotResultV0(input, result), nil
}

func NormalizeMCPNuevaAppWizardBotInputV0(
	input MCPNuevaAppWizardBotToolInputV0,
) MCPNuevaAppWizardBotToolInputV0 {
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.SessionRef = strings.TrimSpace(input.SessionRef)
	input.UserText = strings.TrimSpace(input.UserText)
	input.Locale = strings.TrimSpace(input.Locale)
	if input.SessionRef == "" {
		input.SessionRef = firstNonEmptyMCPV0(input.RequestID, input.CorrelationID)
	}
	if input.RequestID == "" {
		input.RequestID = firstNonEmptyMCPV0(input.SessionRef, input.CorrelationID, generatedMCPNuevaAppRequestIDV0())
	}
	if input.CorrelationID == "" {
		input.CorrelationID = input.RequestID
	}
	if input.SessionRef == "" {
		input.SessionRef = input.RequestID
	}
	return input
}

func NormalizeMCPNuevaAppWizardBotResultV0(
	input MCPNuevaAppWizardBotToolInputV0,
	result MCPNuevaAppWizardBotToolResultV0,
) MCPNuevaAppWizardBotToolResultV0 {
	input = NormalizeMCPNuevaAppWizardBotInputV0(input)
	if result.Estado == "" {
		result.Estado = MCPNuevaAppWizardBotEstadoOKV0
	}
	result.RequestID = firstNonEmptyMCPV0(result.RequestID, input.RequestID)
	result.CorrelationID = firstNonEmptyMCPV0(result.CorrelationID, input.CorrelationID, result.RequestID)
	if result.Errores == nil {
		result.Errores = []MCPValidationIssueV0{}
	}
	return result
}

func NewMCPNuevaAppWizardBotErrorResultV0(
	input MCPNuevaAppWizardBotToolInputV0,
	code string,
	field string,
) MCPNuevaAppWizardBotToolResultV0 {
	input = NormalizeMCPNuevaAppWizardBotInputV0(input)
	return MCPNuevaAppWizardBotToolResultV0{
		Estado:        MCPNuevaAppWizardBotEstadoErrorV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(code),
		}},
	}
}

func mcpNuevaAppWizardBotTransportHandlerV0(
	port MCPNuevaAppWizardBotExecutorPortV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPNuevaAppWizardBotToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPNuevaAppWizardBotToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := NewMCPNuevaAppWizardBotToolExecutorV0(port).Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
