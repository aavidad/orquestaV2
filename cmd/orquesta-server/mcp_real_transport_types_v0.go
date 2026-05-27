package main

import (
	"encoding/json"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

type mcpRealTransportRegistryV0 struct {
	resourcesByName map[string]orquestamcp.MCPTransportResourceEnvelopeV0
	resourcesByURI  map[string]orquestamcp.MCPTransportResourceEnvelopeV0
	tools           map[string]orquestamcp.MCPTransportToolEnvelopeV0
	observer        mcpRealExecutionObserverV0
}

type mcpRealExecutionObserverV0 func(orquestaobservability.MCPExecutionObservationV0)

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
	Name             string                                    `json:"name"`
	URI              string                                    `json:"uri"`
	MimeType         string                                    `json:"mimeType"`
	Description      string                                    `json:"description,omitempty"`
	DescriptorSource orquestamcp.MCPResourceDescriptorSourceV0 `json:"descriptor_source"`
}

type mcpToolListResultV0 struct {
	Tools []mcpToolDescriptorV0 `json:"tools"`
}

type mcpToolDescriptorV0 struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
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

type mcpInitializeResultV0 struct {
	ProtocolVersion string                    `json:"protocolVersion"`
	Capabilities    mcpServerCapabilitiesV0   `json:"capabilities"`
	ServerInfo      mcpServerImplementationV0 `json:"serverInfo"`
}

type mcpServerCapabilitiesV0 struct {
	Resources map[string]any `json:"resources,omitempty"`
	Tools     map[string]any `json:"tools,omitempty"`
}

type mcpServerImplementationV0 struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
