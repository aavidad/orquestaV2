package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	mcpRealHTTPPathV0        = "/mcp"
	mcpJSONRPCVersionV0      = "2.0"
	mcpJSONRPCMaxBodyBytesV0 = 1 << 20
)

type mcpRealTransportRegistryV0 struct {
	resourcesByName map[string]orquestamcp.MCPTransportResourceEnvelopeV0
	resourcesByURI  map[string]orquestamcp.MCPTransportResourceEnvelopeV0
	tools           map[string]orquestamcp.MCPTransportToolEnvelopeV0
}

type mcpJSONRPCRequestV0 struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpJSONRPCResponseV0 struct {
	JSONRPC string             `json:"jsonrpc"`
	ID      json.RawMessage    `json:"id,omitempty"`
	Result  any                `json:"result,omitempty"`
	Error   *mcpJSONRPCErrorV0 `json:"error,omitempty"`
}

type mcpJSONRPCErrorV0 struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data,omitempty"`
}

type mcpResourceReadParamsV0 struct {
	Name string `json:"name,omitempty"`
	URI  string `json:"uri,omitempty"`
}

type mcpToolCallParamsV0 struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type mcpResourceListResultV0 struct {
	Resources []mcpResourceDescriptorV0 `json:"resources"`
}

type mcpResourceDescriptorV0 struct {
	Name        string `json:"name"`
	URI         string `json:"uri"`
	MimeType    string `json:"mimeType"`
	Description string `json:"description,omitempty"`
}

type mcpToolListResultV0 struct {
	Tools []mcpToolDescriptorV0 `json:"tools"`
}

type mcpToolDescriptorV0 struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema string `json:"inputSchema,omitempty"`
}

type mcpResourceReadResultV0 struct {
	Contents []mcpTextContentV0 `json:"contents"`
}

type mcpToolCallResultV0 struct {
	Content []mcpTextContentV0 `json:"content"`
	IsError bool               `json:"isError,omitempty"`
}

type mcpTextContentV0 struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	URI      string `json:"uri,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

func newMCPRealHTTPHandlerV0(bindings orquestamcp.MCPTransportBindingsV0) (http.Handler, error) {
	registry := &mcpRealTransportRegistryV0{
		resourcesByName: map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		resourcesByURI:  map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		tools:           map[string]orquestamcp.MCPTransportToolEnvelopeV0{},
	}
	if err := orquestamcp.RegisterMCPTransportV0(registry, bindings); err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc(mcpRealHTTPPathV0, registry.serveJSONRPCV0)
	return mux, nil
}

func (registry *mcpRealTransportRegistryV0) RegisterResourceV0(
	resource orquestamcp.MCPTransportResourceEnvelopeV0,
) error {
	registry.resourcesByName[resource.Name] = resource
	registry.resourcesByURI[resource.URI] = resource
	return nil
}

func (registry *mcpRealTransportRegistryV0) RegisterToolV0(
	tool orquestamcp.MCPTransportToolEnvelopeV0,
) error {
	registry.tools[tool.Name] = tool
	return nil
}

func (registry *mcpRealTransportRegistryV0) serveJSONRPCV0(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "mcp_method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var request mcpJSONRPCRequestV0
	if err := json.NewDecoder(io.LimitReader(r.Body, mcpJSONRPCMaxBodyBytesV0)).Decode(&request); err != nil {
		_ = json.NewEncoder(w).Encode(mcpJSONRPCResponseV0{
			JSONRPC: mcpJSONRPCVersionV0,
			Error:   mcpRPCErrorV0(-32700, "mcp_parse_error", "mcp_json_invalid"),
		})
		return
	}
	result, rpcErr := registry.handleJSONRPCV0(r.Context(), request)
	response := mcpJSONRPCResponseV0{JSONRPC: mcpJSONRPCVersionV0, ID: request.ID}
	if rpcErr != nil {
		response.Error = rpcErr
	} else {
		response.Result = result
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (registry *mcpRealTransportRegistryV0) handleJSONRPCV0(
	ctx context.Context,
	request mcpJSONRPCRequestV0,
) (any, *mcpJSONRPCErrorV0) {
	switch strings.TrimSpace(request.Method) {
	case "resources/list":
		return registry.resourceListV0(), nil
	case "resources/read":
		return registry.readResourceV0(ctx, request.Params)
	case "tools/list":
		return registry.toolListV0(), nil
	case "tools/call":
		return registry.callToolV0(ctx, request.Params)
	default:
		return nil, mcpRPCErrorV0(-32601, "mcp_method_not_found", "mcp_method_not_found")
	}
}

func (registry *mcpRealTransportRegistryV0) resourceListV0() mcpResourceListResultV0 {
	names := sortedKeysMCPRealV0(registry.resourcesByName)
	resources := make([]mcpResourceDescriptorV0, 0, len(names))
	for _, name := range names {
		resource := registry.resourcesByName[name]
		resources = append(resources, mcpResourceDescriptorV0{
			Name:        resource.Name,
			URI:         resource.URI,
			MimeType:    resource.ContentType,
			Description: resource.SummaryKey,
		})
	}
	return mcpResourceListResultV0{Resources: resources}
}

func (registry *mcpRealTransportRegistryV0) readResourceV0(
	ctx context.Context,
	raw json.RawMessage,
) (any, *mcpJSONRPCErrorV0) {
	var params mcpResourceReadParamsV0
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_resource_params_invalid")
	}
	resource, ok := registry.resourcesByName[strings.TrimSpace(params.Name)]
	if !ok {
		resource, ok = registry.resourcesByURI[strings.TrimSpace(params.URI)]
	}
	if !ok {
		return nil, mcpRPCErrorV0(-32602, "mcp_resource_not_found", "mcp_resource_not_found")
	}
	payload, err := resource.Handler(ctx)
	if err != nil {
		return nil, mcpRPCErrorV0(-32000, "mcp_resource_handler_error", "mcp_resource_handler_error")
	}
	return mcpResourceReadResultV0{Contents: []mcpTextContentV0{{
		Type:     "text",
		Text:     string(payload),
		URI:      resource.URI,
		MimeType: resource.ContentType,
	}}}, nil
}

func (registry *mcpRealTransportRegistryV0) toolListV0() mcpToolListResultV0 {
	names := sortedKeysMCPRealV0(registry.tools)
	tools := make([]mcpToolDescriptorV0, 0, len(names))
	for _, name := range names {
		tool := registry.tools[name]
		tools = append(tools, mcpToolDescriptorV0{
			Name:        tool.Name,
			Description: tool.ResourceURI,
			InputSchema: tool.InputShape,
		})
	}
	return mcpToolListResultV0{Tools: tools}
}

func (registry *mcpRealTransportRegistryV0) callToolV0(
	ctx context.Context,
	raw json.RawMessage,
) (any, *mcpJSONRPCErrorV0) {
	var params mcpToolCallParamsV0
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_tool_params_invalid")
	}
	tool, ok := registry.tools[strings.TrimSpace(params.Name)]
	if !ok {
		return nil, mcpRPCErrorV0(-32602, "mcp_tool_not_found", "mcp_tool_not_found")
	}
	arguments := params.Arguments
	if len(arguments) == 0 || string(arguments) == "null" {
		arguments = json.RawMessage(`{}`)
	}
	payload, err := tool.Handler(ctx, arguments)
	if err != nil {
		return nil, mcpRPCErrorV0(-32000, "mcp_tool_handler_error", "mcp_tool_handler_error")
	}
	return mcpToolCallResultV0{
		Content: []mcpTextContentV0{{
			Type:     "text",
			Text:     string(payload),
			MimeType: "application/json",
		}},
		IsError: mcpJSONPayloadIsErrorV0(payload),
	}, nil
}

func mcpJSONPayloadIsErrorV0(payload json.RawMessage) bool {
	var envelope struct {
		Estado    string `json:"estado"`
		ErrorCode string `json:"error_code"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return true
	}
	return strings.EqualFold(envelope.Estado, "error") || strings.TrimSpace(envelope.ErrorCode) != ""
}

func mcpRPCErrorV0(code int, message string, publicCode string) *mcpJSONRPCErrorV0 {
	return &mcpJSONRPCErrorV0{
		Code:    code,
		Message: message,
		Data:    map[string]string{"error_code": publicCode},
	}
}

func sortedKeysMCPRealV0[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
