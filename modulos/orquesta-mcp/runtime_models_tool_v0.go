package orquestamcp

import (
	"context"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	MCPRuntimeModelsToolNameV0    = "orquesta.runtime.models.v0"
	MCPRuntimeModelsToolVersionV0 = "v0"
	MCPRuntimeModelsResourceURIV0 = "orquesta://contracts/runtime-models/v0"
	MCPRuntimeModelsEstadoOKV0    = "ok"
	MCPRuntimeModelsEstadoErrorV0 = "error"
)

type MCPRuntimeModelsToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPRuntimeModelsToolInputV0 struct {
	Action       string   `json:"action"`
	ProviderRef  string   `json:"provider_ref,omitempty"`
	EndpointRef  string   `json:"endpoint_ref,omitempty"`
	Model        string   `json:"model,omitempty"`
	KeepAlive    string   `json:"keep_alive,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type MCPRuntimeModelsToolResultV0 struct {
	Estado       string                                      `json:"estado"`
	Action       string                                      `json:"action,omitempty"`
	ListResult   *orquestaruntime.RuntimeModelListResultV0   `json:"list_result,omitempty"`
	ActionResult *orquestaruntime.RuntimeModelActionResultV0 `json:"action_result,omitempty"`
	Errores      []MCPValidationIssueV0                      `json:"errores_publicos,omitempty"`
}

func MCPRuntimeModelsDescriptorV0() MCPRuntimeModelsToolDescriptorV0 {
	return MCPRuntimeModelsToolDescriptorV0{
		Name:        MCPRuntimeModelsToolNameV0,
		Version:     MCPRuntimeModelsToolVersionV0,
		InputSchema: "envelope:{action:list|status|pull|serve|stop,provider_ref?,endpoint_ref?,model?,keep_alive?,tags?,evidence_refs?}",
		Output:      "ok:{list_result?|action_result?}|error:{errores_publicos}",
		ResourceURI: MCPRuntimeModelsResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino para gestion de modelos",
			"delega en RuntimeModelManagerPortV0 inyectado",
			"ollama/local/cloud quedan fuera del nucleo",
			"no instala ni sirve modelos si el puerto no esta configurado",
		},
	}
}

type MCPRuntimeModelsToolExecutorV0 struct {
	Port orquestaruntime.RuntimeModelManagerPortV0
}

func (executor MCPRuntimeModelsToolExecutorV0) Execute(
	ctx context.Context,
	input MCPRuntimeModelsToolInputV0,
) (MCPRuntimeModelsToolResultV0, error) {
	if executor.Port == nil {
		return newMCPRuntimeModelsErrorV0(input, MCPTransportToolUnboundV0, "port"), nil
	}
	action := normalizeMCPRuntimeModelsActionV0(input.Action)
	switch action {
	case "list":
		result, err := executor.Port.ListRuntimeModelsV0(ctx, runtimeModelListRequestFromMCPV0(input))
		if err != nil {
			return MCPRuntimeModelsToolResultV0{}, err
		}
		return newMCPRuntimeModelsListResultV0(input, result), nil
	case "status":
		result, err := executor.Port.RuntimeModelStatusV0(ctx, runtimeModelListRequestFromMCPV0(input))
		if err != nil {
			return MCPRuntimeModelsToolResultV0{}, err
		}
		return newMCPRuntimeModelsListResultV0(input, result), nil
	case "pull":
		result, err := executor.Port.PullRuntimeModelV0(ctx, runtimeModelActionRequestFromMCPV0(input))
		if err != nil {
			return MCPRuntimeModelsToolResultV0{}, err
		}
		return newMCPRuntimeModelsActionResultV0(input, result), nil
	case "serve":
		result, err := executor.Port.ServeRuntimeModelV0(ctx, runtimeModelActionRequestFromMCPV0(input))
		if err != nil {
			return MCPRuntimeModelsToolResultV0{}, err
		}
		return newMCPRuntimeModelsActionResultV0(input, result), nil
	case "stop":
		result, err := executor.Port.StopRuntimeModelV0(ctx, runtimeModelActionRequestFromMCPV0(input))
		if err != nil {
			return MCPRuntimeModelsToolResultV0{}, err
		}
		return newMCPRuntimeModelsActionResultV0(input, result), nil
	default:
		return newMCPRuntimeModelsErrorV0(input, MCPTransportToolInputInvalidV0, "action"), nil
	}
}

func runtimeModelListRequestFromMCPV0(input MCPRuntimeModelsToolInputV0) orquestaruntime.RuntimeModelListRequestV0 {
	return orquestaruntime.RuntimeModelListRequestV0{
		ProviderRef: strings.TrimSpace(input.ProviderRef),
		EndpointRef: strings.TrimSpace(input.EndpointRef),
		Tags:        compactStringsMCPV0(input.Tags),
	}
}

func runtimeModelActionRequestFromMCPV0(input MCPRuntimeModelsToolInputV0) orquestaruntime.RuntimeModelActionRequestV0 {
	return orquestaruntime.RuntimeModelActionRequestV0{
		ProviderRef: strings.TrimSpace(input.ProviderRef),
		EndpointRef: strings.TrimSpace(input.EndpointRef),
		Model:       strings.TrimSpace(input.Model),
		KeepAlive:   strings.TrimSpace(input.KeepAlive),
		Tags:        compactStringsMCPV0(input.Tags),
		Evidence:    compactStringsMCPV0(input.EvidenceRefs),
	}
}

func newMCPRuntimeModelsListResultV0(
	input MCPRuntimeModelsToolInputV0,
	result orquestaruntime.RuntimeModelListResultV0,
) MCPRuntimeModelsToolResultV0 {
	action := normalizeMCPRuntimeModelsActionV0(input.Action)
	return MCPRuntimeModelsToolResultV0{
		Estado:     MCPRuntimeModelsEstadoOKV0,
		Action:     action,
		ListResult: &result,
		Errores:    []MCPValidationIssueV0{},
	}
}

func newMCPRuntimeModelsActionResultV0(
	input MCPRuntimeModelsToolInputV0,
	result orquestaruntime.RuntimeModelActionResultV0,
) MCPRuntimeModelsToolResultV0 {
	action := normalizeMCPRuntimeModelsActionV0(input.Action)
	return MCPRuntimeModelsToolResultV0{
		Estado:       MCPRuntimeModelsEstadoOKV0,
		Action:       action,
		ActionResult: &result,
		Errores:      []MCPValidationIssueV0{},
	}
}

func newMCPRuntimeModelsErrorV0(input MCPRuntimeModelsToolInputV0, code string, field string) MCPRuntimeModelsToolResultV0 {
	return MCPRuntimeModelsToolResultV0{
		Estado: MCPRuntimeModelsEstadoErrorV0,
		Action: normalizeMCPRuntimeModelsActionV0(input.Action),
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(code),
		}},
	}
}

func normalizeMCPRuntimeModelsActionV0(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}
