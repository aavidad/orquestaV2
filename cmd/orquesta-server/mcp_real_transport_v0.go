package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	mcpRealHTTPPathV0        = "/mcp"
	mcpJSONRPCVersionV0      = "2.0"
	mcpJSONRPCMaxBodyBytesV0 = 1 << 20
)

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
	return orquestahttpgateway.NewControlPlaneHTTPHeadersV0(mux), nil
}

func (registry *mcpRealTransportRegistryV0) RegisterResourceV0(
	resource orquestamcp.MCPTransportResourceEnvelopeV0,
) error {
	return registry.registerResourceV0(resource)
}

func (registry *mcpRealTransportRegistryV0) RegisterToolV0(
	tool orquestamcp.MCPTransportToolEnvelopeV0,
) error {
	return registry.registerToolV0(tool)
}

func (registry *mcpRealTransportRegistryV0) serveJSONRPCV0(w http.ResponseWriter, r *http.Request) {
	if handleServerPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		writeMCPJSONRPCMethodNotAllowedV0(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var request mcpJSONRPCRequestV0
	if code := decodeMCPJSONRPCRequestV0(w, r, &request); code != "" {
		_ = json.NewEncoder(w).Encode(mcpJSONRPCResponseV0{
			JSONRPC: mcpJSONRPCVersionV0,
			Error:   mcpRPCErrorV0(mcpRPCCodeForDecodeErrorV0(code), "mcp_parse_error", code),
		})
		return
	}
	safeID, validationErr, notificationAccepted := validateMCPJSONRPCRequestV0(request)
	if validationErr != nil {
		_ = json.NewEncoder(w).Encode(mcpJSONRPCResponseV0{
			JSONRPC: mcpJSONRPCVersionV0,
			ID:      safeID,
			Error:   validationErr,
		})
		return
	}
	if notificationAccepted {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	ctx := context.WithValue(r.Context(), mcpRealCorrelationContextKeyV0{}, mcpJSONRPCIDCorrelationV0(safeID))
	result, rpcErr := registry.handleJSONRPCV0(ctx, request)
	response := mcpJSONRPCResponseV0{JSONRPC: mcpJSONRPCVersionV0, ID: safeID}
	if rpcErr != nil {
		response.Error = rpcErr
	} else {
		response.Result = result
	}
	_ = json.NewEncoder(w).Encode(response)
}

func writeMCPJSONRPCMethodNotAllowedV0(w http.ResponseWriter) {
	setServerPublicHTTPAllowV0(w, http.MethodPost)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(mcpJSONRPCResponseV0{
		JSONRPC: mcpJSONRPCVersionV0,
		Error:   mcpRPCErrorV0(-32600, serverPublicErrMethodNotAllowedV0, serverPublicErrMethodNotAllowedV0),
	})
}

func (registry *mcpRealTransportRegistryV0) handleJSONRPCV0(
	ctx context.Context,
	request mcpJSONRPCRequestV0,
) (any, *mcpJSONRPCErrorV0) {
	switch strings.TrimSpace(request.Method) {
	case "initialize":
		return mcpInitializeResultV0{
			ProtocolVersion: "2025-03-26",
			Capabilities: mcpServerCapabilitiesV0{
				Resources: map[string]any{},
				Tools:     map[string]any{},
			},
			ServerInfo: mcpServerImplementationV0{
				Name:    "orquesta-mcp",
				Version: "v0",
			},
		}, nil
	case "ping":
		return map[string]any{}, nil
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
			Name:             resource.Name,
			URI:              resource.URI,
			MimeType:         resource.ContentType,
			Description:      resource.SummaryKey,
			DescriptorSource: resource.DescriptorSource,
		})
	}
	return mcpResourceListResultV0{Resources: resources}
}

func (registry *mcpRealTransportRegistryV0) readResourceV0(
	ctx context.Context,
	raw json.RawMessage,
) (any, *mcpJSONRPCErrorV0) {
	var params mcpResourceReadParamsV0
	if err := decodeMCPStrictObjectParamsV0(raw, &params); err != nil {
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_resource_params_invalid")
	}
	name := strings.TrimSpace(params.Name)
	uri := strings.TrimSpace(params.URI)
	if (name == "" && uri == "") || (name != "" && uri != "") {
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_resource_params_invalid")
	}
	resource, ok := registry.resourcesByName[name]
	if !ok {
		resource, ok = registry.resourcesByURI[uri]
	}
	if !ok {
		return nil, mcpRPCErrorV0(-32602, "mcp_resource_not_found", "mcp_resource_not_found")
	}
	payload, err := registry.executeResourceHandlerV0(ctx, resource)
	if err != nil {
		if rpcErr := registry.mcpResourceExecutionRPCErrorV0(resource.Name, resource.ExecutionBudget, err); rpcErr != nil {
			return nil, rpcErr
		}
		return nil, mcpRPCErrorV0(-32000, "mcp_resource_handler_error", "mcp_resource_handler_error")
	}
	if rpcErr := mcpOutputBudgetRPCErrorV0(mcpOutputKindResourceV0, resource.Name, resource.OutputBudget, payload); rpcErr != nil {
		return nil, rpcErr
	}
	return mcpResourceReadResultV0{Contents: []mcpTextContentV0{{
		Type:     "text",
		Text:     string(payload),
		URI:      resource.URI,
		MimeType: resource.ContentType,
	}}}, nil
}
