package orquestaserver

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type ExternalBridgeLifecycleUpdateV0 struct {
	Component    string
	Status       string
	TickNumber   int
	TickActive   bool
	Success      bool
	ErrorCode    string
	StopReason   string
	Filters      []string
	Counters     map[string]int
	OccurredAt   time.Time
	EvidenceRefs []string
}

func (runtime *RuntimeV0) MarkExternalBridgeLifecycleV0(
	ctx context.Context,
	update ExternalBridgeLifecycleUpdateV0,
) {
	if runtime == nil || runtime.tracker == nil {
		return
	}
	now := update.OccurredAt
	if now.IsZero() && runtime.clock != nil {
		now = runtime.clock.Now()
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	state := runtime.tracker.MarkExternalBridgeLifecycleV0(update, now)
	runtime.persistStateTransitionV0(ctx, state, "external_bridge_lifecycle")
	if strings.TrimSpace(update.ErrorCode) != "" {
		runtime.auditEventV0(ctx, "external_bridge_tick", update.Status, update.ErrorCode, map[string]interface{}{
			"component":     strings.TrimSpace(update.Component),
			"tick_ref":      state.ExternalBridgeLastTickRef,
			"filters":       compactServerDiagnosticStringsV0(update.Filters),
			"counters":      copyServerIntMapV0(update.Counters),
			"stop_reason":   strings.TrimSpace(update.StopReason),
			"evidence_refs": compactServerDiagnosticStringsV0(update.EvidenceRefs),
		})
	}
}

func (tracker *StatusTrackerV0) MarkExternalBridgeLifecycleV0(
	update ExternalBridgeLifecycleUpdateV0,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		status := strings.TrimSpace(update.Status)
		if status == "" {
			status = "running"
		}
		state.LastHeartbeatAt = formatTimeV0(now)
		state.ExternalBridgeComponent = firstNonEmptyServerDiagnosticV0(update.Component, state.ExternalBridgeComponent)
		state.ExternalBridgeStatus = status
		state.ExternalBridgeTickActive = update.TickActive
		if update.TickNumber > 0 {
			state.ExternalBridgeLastTickRef = externalBridgeTickRefV0(update.Component, update.TickNumber)
			state.ExternalBridgeLastTickAt = formatTimeV0(now)
			state.ExternalBridgeTicks++
		}
		if len(update.Filters) > 0 {
			state.ExternalBridgeFilters = compactServerDiagnosticStringsV0(update.Filters)
		}
		if update.Counters != nil {
			state.ExternalBridgeCounters = copyServerIntMapV0(update.Counters)
			state.ExternalBridgeEvidenceRefs = compactServerDiagnosticStringsV0(update.EvidenceRefs)
		}
		if update.Success {
			state.ExternalBridgeLastSuccess = formatTimeV0(now)
			state.ExternalBridgeLastError = ""
			state.ExternalBridgeLastErrorAt = ""
			state.ExternalBridgeOperationalMessage = nil
			if status == "degraded" || status == "timeout" {
				state.ExternalBridgeStatus = "running"
			}
		}
		updateExternalBridgeErrorV0(state, update, now)
		if strings.TrimSpace(update.StopReason) != "" {
			state.ExternalBridgeStopReason = strings.TrimSpace(update.StopReason)
		}
	})
}

func updateExternalBridgeErrorV0(
	state *StateV0,
	update ExternalBridgeLifecycleUpdateV0,
	now time.Time,
) {
	code := strings.TrimSpace(update.ErrorCode)
	if code == "" {
		return
	}
	state.ExternalBridgeLastError = projectServerOperationalMessageV0("external_bridge", code)
	state.ExternalBridgeLastErrorAt = formatTimeV0(now)
	state.ExternalBridgeOperationalMessage = projectServerOperationalMessageRecordV0(
		serverOperationalMessageInputV0{
			Scope:        "external_bridge",
			ReasonCode:   code,
			Status:       strings.TrimSpace(update.Status),
			Message:      code,
			EvidenceRefs: update.EvidenceRefs,
			Counters:     update.Counters,
		},
	)
	state.ExternalBridgeErrorTicks++
	appendRecentServerErrorV0(
		state,
		now,
		state.ExternalBridgeLastError,
		"external_bridge",
		state.ExternalBridgeLastError,
		update.EvidenceRefs,
	)
}

func externalBridgeTickRefV0(component string, tickNumber int) string {
	component = strings.TrimSpace(component)
	if component == "" {
		component = "external_bridge_loop"
	}
	return component + "-tick-ref-" + strconv.Itoa(tickNumber)
}

func copyServerIntMapV0(values map[string]int) map[string]int {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]int, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if value < 0 {
			value = 0
		}
		out[key] = value
	}
	return out
}
