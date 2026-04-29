package cmd

import (
	"strings"
	"time"
)

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

func mcpServerSelfHealTool() mcpTool {
	return mcpTool{
		Name:        "orquesta.server.self_heal",
		Title:       "Auto-reparar control plane",
		Description: "Ejecuta el barrido canónico de wake, hygiene, degradados, autonomia y reanimaciones, y devuelve el estado operativo final",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"orders":        map[string]any{"type": "boolean"},
				"mailbox":       map[string]any{"type": "boolean"},
				"warm":          map[string]any{"type": "boolean"},
				"hygiene":       map[string]any{"type": "boolean"},
				"degradados":    map[string]any{"type": "boolean"},
				"autonomia":     map[string]any{"type": "boolean"},
				"reanimaciones": map[string]any{"type": "boolean"},
			},
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

var mcpBuildServerOperationalInfoFn = buildMCPServerOperationalInfo

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
	info := mcpBuildServerOperationalInfoFn()
	return toolResult(prettyJSON(info), info, false), nil
}

func callMCPServerSelfHeal(args map[string]any) (map[string]any, error) {
	enabled := func(key string) bool {
		if args == nil {
			return true
		}
		if _, ok := args[key]; !ok {
			return true
		}
		return boolArgOrFalse(args, key)
	}
	result := apiRuntimeSelfHealResponse{OK: true}
	appendErr := func(phase string, err error) {
		if err == nil {
			return
		}
		result.OK = false
		result.Errors = append(result.Errors, strings.TrimSpace(phase)+": "+strings.TrimSpace(err.Error()))
	}

	if enabled("orders") {
		result.Wake.Orders = apiRuntimeWakeOrdersFn()
	}
	if enabled("mailbox") {
		result.Wake.Mailbox = apiRuntimeWakeMailboxFn()
	}
	if enabled("warm") {
		result.Wake.Warm = apiRuntimeWakeWarmFn()
	}
	if enabled("hygiene") {
		count, err := runtimeProcessHygieneBatch()
		result.Hygiene = apiRuntimeProcessAutonomiaResponse{OK: err == nil, Count: count}
		appendErr("hygiene", err)
	}
	if enabled("degradados") {
		summary, err := runtimeProcessDegradadosBatchDetailed()
		result.Degradados = apiRuntimeProcessAutonomiaResponse{
			OK:                        err == nil,
			Count:                     summary.Count,
			GhostAssignmentsCompacted: summary.GhostAssignmentsCompacted,
			ReactivatedWithoutRuntime: summary.ReactivatedWithoutRuntime,
			IdleAutoassigned:          summary.IdleAutoassigned,
		}
		appendErr("degradados", err)
	}
	if enabled("autonomia") {
		count, err := runtimeProcessAutonomiaBatch()
		result.Autonomia = apiRuntimeProcessAutonomiaResponse{OK: err == nil, Count: count}
		appendErr("autonomia", err)
	}
	if enabled("reanimaciones") {
		result.Reanimaciones = runtimeProcessReanimationsBatchFn()
		if !result.Reanimaciones.OK || result.Reanimaciones.Errors > 0 {
			result.OK = false
			if !result.Reanimaciones.OK {
				result.Errors = append(result.Errors, "reanimaciones: batch no OK")
			} else {
				result.Errors = append(result.Errors, "reanimaciones: batch con errores")
			}
		}
	}
	result.Operational = mcpBuildServerOperationalInfoFn()
	return toolResult(prettyJSON(result), result, len(result.Errors) > 0), nil
}
