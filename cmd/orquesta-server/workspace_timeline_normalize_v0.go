package main

import (
	"encoding/json"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func normalizeMCPToolArgumentsForToolV0(
	toolName string,
	arguments json.RawMessage,
) (json.RawMessage, *mcpJSONRPCErrorV0) {
	switch strings.TrimSpace(toolName) {
	case orquestamcp.MCPWorkspaceTimelineToolNameV0:
		return normalizeWorkspaceTimelineArgumentsV0(arguments)
	default:
		return arguments, nil
	}
}

func normalizeWorkspaceTimelineArgumentsV0(
	arguments json.RawMessage,
) (json.RawMessage, *mcpJSONRPCErrorV0) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &object); err != nil {
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "workspace_timeline_arguments_invalid")
	}
	raw, ok := object["time_window"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return arguments, nil
	}
	var preset string
	if err := json.Unmarshal(raw, &preset); err == nil {
		preset = strings.TrimSpace(preset)
		if preset == "" {
			delete(object, "time_window")
		} else {
			normalized, marshalErr := json.Marshal(map[string]string{"preset": preset})
			if marshalErr != nil {
				return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "workspace_timeline_arguments_invalid")
			}
			object["time_window"] = normalized
		}
		return marshalWorkspaceTimelineArgumentsV0(object)
	}
	var nested map[string]any
	if err := json.Unmarshal(raw, &nested); err != nil {
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "workspace_timeline_time_window_invalid")
	}
	return arguments, nil
}

func marshalWorkspaceTimelineArgumentsV0(
	object map[string]json.RawMessage,
) (json.RawMessage, *mcpJSONRPCErrorV0) {
	raw, err := json.Marshal(object)
	if err != nil {
		return nil, mcpRPCErrorV0(-32602, "mcp_invalid_params", "workspace_timeline_arguments_invalid")
	}
	return raw, nil
}
