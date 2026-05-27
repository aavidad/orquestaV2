package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sort"
	"strings"
)

func (registry *mcpRealTransportRegistryV0) callToolV0(
	ctx context.Context,
	raw json.RawMessage,
) (any, *mcpJSONRPCErrorV0) {
	var params mcpToolCallParamsV0
	if err := decodeMCPStrictObjectParamsV0(raw, &params); err != nil {
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_tool_params_invalid")
	}
	if strings.TrimSpace(params.Name) == "" || !mcpToolArgumentsShapeValidV0(params.Arguments) {
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
	var rpcErr *mcpJSONRPCErrorV0
	arguments, rpcErr = normalizeMCPToolArgumentsV0(arguments)
	if rpcErr != nil {
		return nil, rpcErr
	}
	arguments, rpcErr = normalizeMCPToolArgumentsForToolV0(tool.Name, arguments)
	if rpcErr != nil {
		return nil, rpcErr
	}
	payload, err := registry.executeToolHandlerV0(ctx, tool, arguments)
	if err != nil {
		if rpcErr := registry.mcpExecutionRPCErrorV0(tool.Name, tool.ExecutionBudget, err); rpcErr != nil {
			return nil, rpcErr
		}
		return nil, mcpRPCErrorV0(-32000, "mcp_tool_handler_error", "mcp_tool_handler_error")
	}
	if rpcErr := mcpOutputBudgetRPCErrorV0(mcpOutputKindToolV0, tool.Name, tool.OutputBudget, payload); rpcErr != nil {
		return nil, rpcErr
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

func decodeMCPStrictObjectParamsV0(raw json.RawMessage, output any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '{' {
		return io.ErrUnexpectedEOF
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func mcpToolArgumentsShapeValidV0(arguments json.RawMessage) bool {
	trimmed := bytes.TrimSpace(arguments)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return true
	}
	return trimmed[0] == '{' || trimmed[0] == '"'
}

func normalizeMCPToolArgumentsV0(
	arguments json.RawMessage,
) (json.RawMessage, *mcpJSONRPCErrorV0) {
	var probe any
	if err := json.Unmarshal(arguments, &probe); err != nil {
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_tool_arguments_invalid_json")
	}
	switch value := probe.(type) {
	case map[string]any:
		return arguments, nil
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_tool_arguments_string_empty")
		}
		var nested map[string]any
		if err := json.Unmarshal([]byte(value), &nested); err != nil {
			return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_tool_arguments_string_not_json_object")
		}
		raw, err := json.Marshal(nested)
		if err != nil {
			return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_tool_arguments_invalid")
		}
		return raw, nil
	default:
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "mcp_tool_arguments_must_be_object")
	}
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
