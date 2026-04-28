package cmd

import "time"

func mcpWorkspaceControlTool() mcpTool {
	return mcpTool{
		Name:        "orquesta.workspace.control",
		Title:       "Control global del workspace",
		Description: "Devuelve la vista global compacta del workspace para supervisión operativa",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"desde": map[string]any{"type": "string"},
			},
			"additionalProperties": false,
		},
	}
}

func mcpServerOperationalTool() mcpTool {
	return mcpTool{
		Name:        "orquesta.server.operational",
		Title:       "Estado operativo del servidor",
		Description: "Resume la salud operativa del control plane con degradación rápida canónica",
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
		},
	}
}

func callMCPWorkspaceControl(args map[string]any) (map[string]any, error) {
	since, err := parseWorkspaceControlSince(optionalStringArg(args, "desde"))
	if err != nil {
		return toolResult(err.Error(), nil, true), nil
	}
	report, err := buildWorkspaceControlReportSince(since)
	if err != nil {
		return nil, err
	}
	payload := apiWorkspaceControlResponse{Control: report}
	return toolResult(prettyJSON(payload), payload, false), nil
}

func buildMCPServerOperationalInfo() serverOperationalInfo {
	if snapshot, ok := readStatusSnapshotFreshUsable(); ok {
		return normalizeServerOperationalInfo(buildServerOperationalInfo(snapshot))
	}
	if status, ok := fetchStatusForOperationalFallback(statusFastTimeout); ok {
		return normalizeServerOperationalInfo(buildServerOperationalInfo(status))
	}
	ensureStatusRefreshAsync()
	info := normalizeServerOperationalInfo(degradedServerOperationalInfo())
	if info.Generated == "" {
		info.Generated = time.Now().UTC().Format(time.RFC3339)
	}
	return info
}

func callMCPServerOperational() (map[string]any, error) {
	info := buildMCPServerOperationalInfo()
	return toolResult(prettyJSON(info), info, false), nil
}
