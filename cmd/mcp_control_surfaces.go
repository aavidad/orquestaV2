package cmd

import (
	"strings"
	"time"

	"orquesta/db"
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
		Description: "Ejecuta el barrido canónico de wake, hygiene, degradados, autonomia y reanimaciones; drena runtime_orders/mailbox y, si aún queda rearm seguro pendiente, lo aplica hasta converger o agotar el límite",
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
				"rearm":         map[string]any{"type": "boolean"},
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
var mcpServerSelfHealRearmLimit = 3
var mcpServerSelfHealDrainPassLimit = 3
var mcpServerSelfHealSettleAttempts = 3
var mcpServerSelfHealSettleDelay = 150 * time.Millisecond

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
	if drain, err := drainMCPServerSelfHealRuntimeWork(&result.Operational, enabled); err != nil {
		result.Drain = drain
		appendErr("drain", err)
	} else {
		result.Drain = drain
	}
	if enabled("rearm") {
		for attempts := 0; attempts < mcpServerSelfHealRearmLimit; attempts++ {
			rearm := result.Operational.Rearm
			if result.Operational.Operational || rearm == nil || !rearm.Needed || !rearm.Available {
				break
			}
			applied, err := serverOperationalApplyNextAction(rearm.Supervisor)
			if err != nil {
				appendErr("rearm", err)
				break
			}
			if applied != nil {
				result.RearmApplied = append(result.RearmApplied, applied)
			}
			result.Operational = mcpBuildServerOperationalInfoFn()
			if drain, err := drainMCPServerSelfHealRuntimeWork(&result.Operational, enabled); err != nil {
				result.Drain = mergeMCPServerSelfHealDrain(result.Drain, drain)
				appendErr("drain", err)
				break
			} else {
				result.Drain = mergeMCPServerSelfHealDrain(result.Drain, drain)
			}
		}
		if !result.Operational.Operational && result.Operational.Rearm != nil &&
			result.Operational.Rearm.Needed && result.Operational.Rearm.Available &&
			len(result.RearmApplied) >= mcpServerSelfHealRearmLimit {
			appendErr("rearm", errSelfHealRearmLimitReached)
		}
	}
	result.Operational = settleMCPServerSelfHealOperational(result.Operational)
	return toolResult(prettyJSON(result), result, len(result.Errors) > 0), nil
}

func settleMCPServerSelfHealOperational(info serverOperationalInfo) serverOperationalInfo {
	if info.Operational || mcpServerSelfHealSettleAttempts <= 0 {
		return info
	}
	for attempt := 0; attempt < mcpServerSelfHealSettleAttempts; attempt++ {
		if mcpServerSelfHealSettleDelay > 0 {
			time.Sleep(mcpServerSelfHealSettleDelay)
		}
		info = mcpBuildServerOperationalInfoFn()
		if info.Operational {
			return info
		}
	}
	return info
}

func drainMCPServerSelfHealRuntimeWork(info *serverOperationalInfo, enabled func(string) bool) (*apiRuntimeSelfHealDrainResponse, error) {
	if enabled == nil || (!enabled("orders") && !enabled("mailbox")) || mcpServerSelfHealDrainPassLimit <= 0 {
		return nil, nil
	}
	drain := &apiRuntimeSelfHealDrainResponse{}
	for pass := 0; pass < mcpServerSelfHealDrainPassLimit; pass++ {
		progress := false
		if enabled("orders") {
			count, err := apiRuntimeProcessOrdersExecutor()
			if err != nil {
				return drain, mcpServerSelfHealError("orders: " + strings.TrimSpace(err.Error()))
			}
			drain.Orders += count
			if count > 0 {
				progress = true
			}
		}
		if enabled("mailbox") {
			count, err := apiRuntimeProcessMailboxExecutor(db.FiltroRuntimeMailbox{})
			if err != nil {
				return drain, mcpServerSelfHealError("mailbox: " + strings.TrimSpace(err.Error()))
			}
			drain.Mailbox += count
			if count > 0 {
				progress = true
			}
		}
		if !progress {
			break
		}
		drain.Passes++
		if info != nil {
			*info = mcpBuildServerOperationalInfoFn()
			if info.Operational {
				break
			}
		}
	}
	if drain.Passes == 0 && drain.Orders == 0 && drain.Mailbox == 0 {
		return nil, nil
	}
	return drain, nil
}

func mergeMCPServerSelfHealDrain(base, add *apiRuntimeSelfHealDrainResponse) *apiRuntimeSelfHealDrainResponse {
	if base == nil {
		return add
	}
	if add == nil {
		return base
	}
	base.Passes += add.Passes
	base.Orders += add.Orders
	base.Mailbox += add.Mailbox
	return base
}

var errSelfHealRearmLimitReached = mcpServerSelfHealError("safe rearm limit reached before convergence")

type mcpServerSelfHealError string

func (e mcpServerSelfHealError) Error() string { return string(e) }
