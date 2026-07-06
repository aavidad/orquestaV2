package orquestamcp

import (
	"context"
	"encoding/json"
	"errors"

	channel "orquesta/modulos/orquesta-operator-director-channel"
)

const (
	MCPOperatorDirectorMessageEstadoOKV0    = "ok"
	MCPOperatorDirectorMessageEstadoErrorV0 = "error"
)

type MCPOperatorDirectorMessageToolResultV0 struct {
	Estado    string                              `json:"estado"`
	Tool      string                              `json:"tool"`
	Issues    []channel.OperatorMessageIssueV0    `json:"issues,omitempty"`
	Response  *channel.OperatorDirectorResponseV0 `json:"response,omitempty"`
	ErrorCode string                              `json:"error_code,omitempty"`
}

type operatorDirectorMessageToolDescriptorV0 struct {
	Name        string
	Version     string
	ResourceURI string
	InputSchema string
	Output      string
}

func operatorDirectorMessageDescriptorV0() operatorDirectorMessageToolDescriptorV0 {
	return operatorDirectorMessageToolDescriptorV0{
		Name:        channel.OperatorDirectorMessageToolNameV0,
		Version:     "v0",
		ResourceURI: channel.OperatorDirectorMessageResourceURIV0,
		InputSchema: "OperatorMessageV0",
		Output:      "OperatorDirectorResponseV0",
	}
}

type MCPTransportOperatorDirectorMessageExecutorV0 struct {
	Service channel.OperatorDirectorChannelServiceV0
}

func (executor MCPTransportOperatorDirectorMessageExecutorV0) Execute(
	ctx context.Context,
	input channel.OperatorMessageV0,
) MCPOperatorDirectorMessageToolResultV0 {
	response, issues, err := channel.DispatchOperatorDirectorMessageV0(ctx, executor.Service, input)
	if len(issues) > 0 {
		return MCPOperatorDirectorMessageToolResultV0{
			Estado: MCPOperatorDirectorMessageEstadoErrorV0,
			Tool:   channel.OperatorDirectorMessageToolNameV0,
			Issues: issues,
		}
	}
	if err != nil {
		return MCPOperatorDirectorMessageToolResultV0{
			Estado:    MCPOperatorDirectorMessageEstadoErrorV0,
			Tool:      channel.OperatorDirectorMessageToolNameV0,
			ErrorCode: operatorDirectorMessageErrorCodeV0(err),
		}
	}
	return MCPOperatorDirectorMessageToolResultV0{
		Estado:   MCPOperatorDirectorMessageEstadoOKV0,
		Tool:     channel.OperatorDirectorMessageToolNameV0,
		Response: &response,
	}
}

func mcpOperatorDirectorMessageTransportHandlerV0(
	executor MCPTransportOperatorDirectorMessageExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input channel.OperatorMessageV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		return json.Marshal(executor.Execute(ctx, input))
	}
}

func operatorDirectorMessageErrorCodeV0(err error) string {
	switch {
	case errors.Is(err, channel.ErrOperatorDirectorDispatchPortUnavailableV0):
		return channel.ErrOperatorMessagePortUnavailableV0
	case errors.Is(err, channel.ErrOperatorDirectorStorePortUnavailableV0):
		return channel.ErrOperatorMessageStoreFailedV0
	default:
		return channel.ErrOperatorMessageDispatchFailedV0
	}
}
