package main

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

const (
	mcpJSONRPCMaxIDStringBytesV0      = 128
	mcpJSONRPCDefaultMaxParamsBytesV0 = 64 << 10
	mcpJSONRPCReadMaxParamsBytesV0    = 8 << 10
	mcpJSONRPCToolMaxParamsBytesV0    = 256 << 10
	mcpJSONRPCInitMaxParamsBytesV0    = 16 << 10
)

func validateMCPJSONRPCRequestV0(
	request mcpJSONRPCRequestV0,
) (json.RawMessage, *mcpJSONRPCErrorV0, bool) {
	hasID := mcpJSONRPCIDPresentV0(request.ID)
	safeID, idOK := safeMCPJSONRPCIDV0(request.ID)
	if hasID && !idOK {
		return nil, mcpRPCErrorV0(-32600, "mcp_invalid_request", "mcp_id_invalid"), false
	}
	if strings.TrimSpace(request.JSONRPC) != mcpJSONRPCVersionV0 {
		return safeID, mcpRPCErrorV0(-32600, "mcp_invalid_request", "mcp_jsonrpc_version_invalid"), false
	}
	method := strings.TrimSpace(request.Method)
	if method == "" {
		return safeID, mcpRPCErrorV0(-32600, "mcp_invalid_request", "mcp_method_required"), false
	}
	if !hasID {
		if method == "notifications/initialized" {
			return nil, nil, true
		}
		return nil, mcpRPCErrorV0(-32600, "mcp_invalid_request", "mcp_notification_unsupported"), false
	}
	if method == "notifications/initialized" {
		return safeID, mcpRPCErrorV0(-32600, "mcp_invalid_request", "mcp_notification_id_not_allowed"), false
	}
	if code := validateMCPJSONRPCParamsBudgetV0(method, request.Params); code != "" {
		return safeID, mcpRPCErrorV0(-32602, "mcp_invalid_params", code), false
	}
	return safeID, nil, false
}

func mcpJSONRPCIDPresentV0(raw json.RawMessage) bool {
	return len(bytes.TrimSpace(raw)) > 0
}

func safeMCPJSONRPCIDV0(raw json.RawMessage) (json.RawMessage, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, true
	}
	if len(trimmed) > mcpJSONRPCMaxIDStringBytesV0 {
		return nil, false
	}
	if bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("null"), true
	}
	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return nil, false
		}
		if strings.TrimSpace(value) == "" || len([]byte(value)) > mcpJSONRPCMaxIDStringBytesV0 {
			return nil, false
		}
		return append(json.RawMessage(nil), trimmed...), true
	}
	var value json.Number
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return nil, false
	}
	text := value.String()
	if strings.ContainsAny(text, ".eE") || len(text) > 32 {
		return nil, false
	}
	if _, err := strconv.ParseInt(text, 10, 64); err != nil {
		return nil, false
	}
	return append(json.RawMessage(nil), trimmed...), true
}

func validateMCPJSONRPCParamsBudgetV0(method string, params json.RawMessage) string {
	limit := int64(mcpJSONRPCDefaultMaxParamsBytesV0)
	switch method {
	case "initialize":
		limit = mcpJSONRPCInitMaxParamsBytesV0
	case "ping", "resources/list", "tools/list":
		limit = 1024
	case "resources/read":
		limit = mcpJSONRPCReadMaxParamsBytesV0
	case "tools/call":
		limit = mcpJSONRPCToolMaxParamsBytesV0
	}
	if int64(len(bytes.TrimSpace(params))) > limit {
		return serverPublicErrMCPParamsTooLargeV0
	}
	return ""
}

func mcpRPCCodeForDecodeErrorV0(code string) int {
	switch code {
	case "mcp_batch_unsupported", "mcp_request_must_be_object":
		return -32600
	default:
		return -32700
	}
}
