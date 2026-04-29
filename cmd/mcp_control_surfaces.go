package cmd

import (
	"log"
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
	info := normalizeServerOperationalInfo(mcpBuildServerOperationalInfoFn())
	return toolResult(prettyJSON(info), info, false), nil
}

func callMCPServerSelfHeal(args map[string]any) (map[string]any, error) {
	started := time.Now()
	phaseDurations := map[string]time.Duration{}
	markPhase := func(name string, start time.Time) {
		phaseDurations[name] = time.Since(start)
	}
	logSlow := func() {
		total := time.Since(started)
		if total < 500*time.Millisecond {
			return
		}
		log.Printf("orquesta[self-heal-slow] total=%s wake=%s hygiene=%s degradados=%s autonomia=%s reanimaciones=%s operational=%s drain=%s rearm=%s settle=%s",
			total,
			phaseDurations["wake"],
			phaseDurations["hygiene"],
			phaseDurations["degradados"],
			phaseDurations["autonomia"],
			phaseDurations["reanimaciones"],
			phaseDurations["operational"],
			phaseDurations["drain"],
			phaseDurations["rearm"],
			phaseDurations["settle"],
		)
	}
	defer logSlow()

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

	phaseStart := time.Now()
	if enabled("orders") {
		result.Wake.Orders = apiRuntimeWakeOrdersFn()
	}
	if enabled("mailbox") {
		result.Wake.Mailbox = apiRuntimeWakeMailboxFn()
	}
	if enabled("warm") {
		result.Wake.Warm = apiRuntimeWakeWarmFn()
	}
	markPhase("wake", phaseStart)
	phaseStart = time.Now()
	if enabled("hygiene") {
		count, err := runtimeProcessHygieneBatch()
		result.Hygiene = apiRuntimeProcessAutonomiaResponse{OK: err == nil, Count: count}
		appendErr("hygiene", err)
	}
	markPhase("hygiene", phaseStart)
	phaseStart = time.Now()
	result.Operational = normalizeServerOperationalInfo(mcpBuildServerOperationalInfoFn())
	syncMCPServerSelfHealRecovery(&result)
	markPhase("operational", phaseStart)
	if mcpServerSelfHealCanShortCircuit(result.Operational) {
		mcpServerSelfHealMarkSkippedHeavyPhases(&result, enabled)
		phaseStart = time.Now()
		result.Operational = settleMCPServerSelfHealOperational(result.Operational)
		syncMCPServerSelfHealRecovery(&result)
		markPhase("settle", phaseStart)
		return toolResult(prettyJSON(result), result, len(result.Errors) > 0), nil
	}
	phaseStart = time.Now()
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
	markPhase("degradados", phaseStart)
	phaseStart = time.Now()
	if enabled("autonomia") {
		count, err := runtimeProcessAutonomiaBatch()
		result.Autonomia = apiRuntimeProcessAutonomiaResponse{OK: err == nil, Count: count}
		appendErr("autonomia", err)
	}
	markPhase("autonomia", phaseStart)
	phaseStart = time.Now()
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
	markPhase("reanimaciones", phaseStart)
	phaseStart = time.Now()
	result.Operational = normalizeServerOperationalInfo(mcpBuildServerOperationalInfoFn())
	syncMCPServerSelfHealRecovery(&result)
	markPhase("operational", phaseStart)
	phaseStart = time.Now()
	if drain, err := drainMCPServerSelfHealRuntimeWork(&result.Operational, enabled); err != nil {
		result.Drain = drain
		syncMCPServerSelfHealRecovery(&result)
		appendErr("drain", err)
	} else {
		result.Drain = drain
		syncMCPServerSelfHealRecovery(&result)
	}
	markPhase("drain", phaseStart)
	phaseStart = time.Now()
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
			result.Operational = normalizeServerOperationalInfo(mcpBuildServerOperationalInfoFn())
			syncMCPServerSelfHealRecovery(&result)
			if drain, err := drainMCPServerSelfHealRuntimeWork(&result.Operational, enabled); err != nil {
				result.Drain = mergeMCPServerSelfHealDrain(result.Drain, drain)
				syncMCPServerSelfHealRecovery(&result)
				appendErr("drain", err)
				break
			} else {
				result.Drain = mergeMCPServerSelfHealDrain(result.Drain, drain)
				syncMCPServerSelfHealRecovery(&result)
			}
		}
		if !result.Operational.Operational && result.Operational.Rearm != nil &&
			result.Operational.Rearm.Needed && result.Operational.Rearm.Available &&
			len(result.RearmApplied) >= mcpServerSelfHealRearmLimit {
			appendErr("rearm", errSelfHealRearmLimitReached)
		}
	}
	markPhase("rearm", phaseStart)
	phaseStart = time.Now()
	result.Operational = settleMCPServerSelfHealOperational(result.Operational)
	syncMCPServerSelfHealRecovery(&result)
	markPhase("settle", phaseStart)
	return toolResult(prettyJSON(result), result, len(result.Errors) > 0), nil
}

func mcpServerSelfHealCanShortCircuit(info serverOperationalInfo) bool {
	info = normalizeServerOperationalInfo(info)
	if !info.Operational {
		return false
	}
	if info.NextRecoveryPlan != nil {
		return false
	}
	if info.Recovery != nil {
		return false
	}
	if info.Rearm != nil && info.Rearm.Needed {
		return false
	}
	return true
}

func mcpServerSelfHealMarkSkippedHeavyPhases(result *apiRuntimeSelfHealResponse, enabled func(string) bool) {
	if result == nil || enabled == nil {
		return
	}
	if enabled("degradados") && !result.Degradados.OK && result.Degradados.Count == 0 {
		result.Degradados.OK = true
	}
	if enabled("autonomia") && !result.Autonomia.OK && result.Autonomia.Count == 0 {
		result.Autonomia.OK = true
	}
	if enabled("reanimaciones") && !result.Reanimaciones.OK &&
		result.Reanimaciones.Count == 0 && result.Reanimaciones.Errors == 0 {
		result.Reanimaciones.OK = true
	}
}

func syncMCPServerSelfHealRecovery(result *apiRuntimeSelfHealResponse) {
	if result == nil {
		return
	}
	result.Operational = normalizeServerOperationalInfo(result.Operational)
	result.NextRecoveryAction = buildOpenClawNextRecoveryAction(result.Operational)
	result.NextRecoveryPlan = result.Operational.NextRecoveryPlan
	if result.NextRecoveryPlan == nil {
		result.NextRecoveryPlan = buildServerOperationalRecoveryPlan(result.Operational, result.NextRecoveryAction)
	}
}

func settleMCPServerSelfHealOperational(info serverOperationalInfo) serverOperationalInfo {
	if info.Operational || mcpServerSelfHealSettleAttempts <= 0 {
		return info
	}
	for attempt := 0; attempt < mcpServerSelfHealSettleAttempts; attempt++ {
		if mcpServerSelfHealSettleDelay > 0 {
			time.Sleep(mcpServerSelfHealSettleDelay)
		}
		info = normalizeServerOperationalInfo(mcpBuildServerOperationalInfoFn())
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
			*info = normalizeServerOperationalInfo(mcpBuildServerOperationalInfoFn())
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
