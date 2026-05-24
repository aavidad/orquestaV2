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
	if strings.TrimSpace(request.Method) == "notifications/initialized" {
		w.WriteHeader(http.StatusAccepted)
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
			Description: mcpToolDescriptionV0(tool),
			InputSchema: mcpToolInputSchemaV0(tool),
		})
	}
	return mcpToolListResultV0{Tools: tools}
}

func mcpToolDescriptionV0(tool orquestamcp.MCPTransportToolEnvelopeV0) string {
	parts := []string{strings.TrimSpace(tool.ResourceURI)}
	if strings.TrimSpace(tool.InputShape) != "" {
		parts = append(parts, "input: "+strings.TrimSpace(tool.InputShape))
	}
	if strings.TrimSpace(tool.OutputShape) != "" {
		parts = append(parts, "output: "+strings.TrimSpace(tool.OutputShape))
	}
	switch strings.TrimSpace(tool.Name) {
	case "orquesta.status.v0":
		parts = append(parts, "Use this for general Orquesta status, como va, queue summary, active projects and safe next actions. No internal refs required.")
	case "orquesta.tasks.list.v0":
		parts = append(parts, "Use this to list queued/active/completed visible tasks. No internal refs required.")
	case "orquesta.projects.list.v0":
		parts = append(parts, "Use this to list active projects visible from the queue. No internal refs required.")
	case "orquesta.agents.list.v0":
		parts = append(parts, "Use this to list agents for visible runs. No internal refs required.")
	case "orquesta.operator.command.v0":
		parts = append(parts, "Use this for human natural-language operational commands when no more specific Orquesta tool is obvious.")
	case "orquesta.operator.status.query.v0", "orquesta.operator.directed_query.v0":
		parts = append(parts, "Low-level operator connector tool. Do not use for general status; prefer orquesta.status.v0 unless connector refs are already known.")
	}
	return strings.Join(compactMCPRealStringsV0(parts), "\n")
}

func mcpToolInputSchemaV0(tool orquestamcp.MCPTransportToolEnvelopeV0) map[string]any {
	description := strings.TrimSpace(tool.InputShape)
	if description == "" {
		description = "JSON envelope"
	}
	schema := map[string]any{
		"type":                 "object",
		"description":          description,
		"additionalProperties": true,
	}
	properties, required := mcpToolSchemaFieldsV0(tool)
	if len(properties) > 0 {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func mcpToolSchemaFieldsV0(
	tool orquestamcp.MCPTransportToolEnvelopeV0,
) (map[string]any, []string) {
	if properties, required, ok := mcpOperatorLowLevelToolSchemaV0(tool.Name); ok {
		return properties, required
	}
	fields := mcpInputShapeFieldsV0(tool.InputShape)
	properties := map[string]any{}
	required := []string{}
	for _, field := range fields {
		properties[field.name] = mcpInputShapeFieldSchemaV0(field)
		if field.required {
			required = append(required, field.name)
		}
	}
	return properties, required
}

type mcpInputShapeFieldV0 struct {
	name     string
	required bool
	value    string
}

func mcpInputShapeFieldsV0(inputShape string) []mcpInputShapeFieldV0 {
	inside, ok := mcpTextBetweenFirstBracesV0(inputShape)
	if !ok {
		return nil
	}
	tokens := splitTopLevelMCPRealV0(inside, ',')
	fields := make([]mcpInputShapeFieldV0, 0, len(tokens))
	for _, token := range tokens {
		field, ok := mcpInputShapeFieldFromTokenV0(token)
		if ok {
			fields = append(fields, field)
		}
	}
	return fields
}

func mcpInputShapeFieldFromTokenV0(token string) (mcpInputShapeFieldV0, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return mcpInputShapeFieldV0{}, false
	}
	namePart := token
	value := ""
	if idx := strings.Index(token, ":"); idx >= 0 {
		namePart = token[:idx]
		value = strings.TrimSpace(token[idx+1:])
	}
	namePart = strings.TrimSpace(namePart)
	required := !strings.HasSuffix(namePart, "?")
	namePart = strings.TrimSuffix(namePart, "?")
	if namePart == "" || strings.ContainsAny(namePart, "{}()|") {
		return mcpInputShapeFieldV0{}, false
	}
	return mcpInputShapeFieldV0{name: namePart, required: required, value: value}, true
}

func mcpInputShapeFieldSchemaV0(field mcpInputShapeFieldV0) map[string]any {
	value := strings.TrimSpace(field.value)
	name := strings.TrimSpace(field.name)
	if strings.Contains(value, "|") && !strings.Contains(value, "{") && !strings.Contains(value, "(") {
		return map[string]any{"type": "string", "enum": splitEnumMCPRealV0(value)}
	}
	switch {
	case strings.HasSuffix(name, "_refs") || strings.HasPrefix(name, "include_") && strings.HasSuffix(name, "s"):
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	case strings.HasPrefix(name, "include_") ||
		strings.HasPrefix(name, "auto_") ||
		strings.HasPrefix(name, "allow_") ||
		strings.HasPrefix(name, "use_") ||
		name == "forced" ||
		name == "resident_mode" ||
		name == "worktree_isolated" ||
		name == "raise_operator_question":
		return map[string]any{"type": "boolean"}
	case strings.Contains(name, "limit") ||
		strings.Contains(name, "max_") ||
		strings.Contains(name, "score") ||
		strings.Contains(name, "ticks") ||
		strings.Contains(name, "executions") ||
		strings.Contains(name, "waits"):
		return map[string]any{"type": "integer"}
	case strings.Contains(value, "{") || strings.Contains(value, "V0") || strings.Contains(value, "Request"):
		return map[string]any{"type": "object", "description": value}
	default:
		return map[string]any{"type": "string", "description": value}
	}
}

func mcpOperatorLowLevelToolSchemaV0(name string) (map[string]any, []string, bool) {
	switch strings.TrimSpace(name) {
	case "orquesta.operator.status.query.v0":
		return map[string]any{
			"request_ref":          map[string]any{"type": "string"},
			"subject_ref":          map[string]any{"type": "string"},
			"status_connector_ref": map[string]any{"type": "string"},
			"include_sections":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		}, []string{"request_ref", "subject_ref", "status_connector_ref"}, true
	case "orquesta.operator.directed_query.v0":
		return map[string]any{
			"query_ref":           map[string]any{"type": "string"},
			"target_ref":          map[string]any{"type": "string"},
			"query_connector_ref": map[string]any{"type": "string"},
			"question":            map[string]any{"type": "string"},
			"evidence_refs":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		}, []string{"query_ref", "target_ref", "query_connector_ref", "question"}, true
	case "orquesta.operator.outbox.pending.v0":
		return map[string]any{
			"request_ref":          map[string]any{"type": "string"},
			"subject_ref":          map[string]any{"type": "string"},
			"outbox_connector_ref": map[string]any{"type": "string"},
			"limit":                map[string]any{"type": "integer"},
			"include_kinds":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		}, []string{"request_ref", "subject_ref", "outbox_connector_ref", "limit"}, true
	case "orquesta.operator.supervised_burst.v0":
		return map[string]any{
			"request_ref":         map[string]any{"type": "string"},
			"run_ref":             map[string]any{"type": "string"},
			"burst_connector_ref": map[string]any{"type": "string"},
			"supervision_ref":     map[string]any{"type": "string"},
			"max_steps":           map[string]any{"type": "integer"},
			"evidence_refs":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		}, []string{"request_ref", "run_ref", "burst_connector_ref", "supervision_ref", "max_steps"}, true
	default:
		return nil, nil, false
	}
}

func mcpTextBetweenFirstBracesV0(value string) (string, bool) {
	start := strings.Index(value, "{")
	if start < 0 {
		return "", false
	}
	depth := 0
	for idx := start; idx < len(value); idx++ {
		switch value[idx] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return value[start+1 : idx], true
			}
		}
	}
	return "", false
}

func splitTopLevelMCPRealV0(value string, separator rune) []string {
	var out []string
	start := 0
	braceDepth := 0
	parenDepth := 0
	for idx, item := range value {
		switch item {
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		default:
			if item == separator && braceDepth == 0 && parenDepth == 0 {
				out = append(out, strings.TrimSpace(value[start:idx]))
				start = idx + len(string(item))
			}
		}
	}
	out = append(out, strings.TrimSpace(value[start:]))
	return out
}

func splitEnumMCPRealV0(value string) []string {
	parts := strings.Split(value, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func compactMCPRealStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
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
