package orquestaserver

import (
	"strings"
	"time"
)

const (
	ShutdownSignalStatusStoppingV0  = "stopping_by_signal"
	ShutdownSignalStatusEscalatedV0 = "signal_escalated"
)

type ShutdownSignalPolicyV0 struct {
	HandledSignals     []string `json:"handled_signals,omitempty"`
	CooperativeSignal  string   `json:"cooperative_signal,omitempty"`
	SecondSignalAction string   `json:"second_signal_action,omitempty"`
}

type ShutdownSignalCauseV0 struct {
	SignalName string
	Count      int
	Escalated  bool
}

func NormalizeShutdownSignalPolicyV0(policy ShutdownSignalPolicyV0) ShutdownSignalPolicyV0 {
	policy.HandledSignals = compactConfigStringsV0(policy.HandledSignals)
	policy.CooperativeSignal = compactSignalPolicyValueV0(policy.CooperativeSignal)
	policy.SecondSignalAction = compactSignalPolicyValueV0(policy.SecondSignalAction)
	return policy
}

func (tracker *StatusTrackerV0) MarkRuntimeStoppingBySignalV0(
	cause ShutdownSignalCauseV0,
	active int,
	now time.Time,
) StateV0 {
	cause = normalizeShutdownSignalCauseV0(cause)
	return tracker.updateV0(func(state *StateV0) {
		state.Status = ShutdownSignalStatusStoppingV0
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = true
		state.ShutdownStatus = ShutdownSignalStatusStoppingV0
		state.ShutdownReady = false
		state.ShutdownAsyncWorkActive = nonNegativeServerIntV0(active)
		state.ShutdownSignalName = cause.SignalName
		state.ShutdownSignalCount = cause.Count
		state.ShutdownSignalEscalated = cause.Escalated
		state.SupervisorFrozen = true
	})
}

func normalizeShutdownSignalCauseV0(cause ShutdownSignalCauseV0) ShutdownSignalCauseV0 {
	cause.SignalName = compactSignalPolicyValueV0(cause.SignalName)
	if cause.SignalName == "" {
		cause.SignalName = "signal_unknown"
	}
	if cause.Count <= 0 {
		cause.Count = 1
	}
	return cause
}

func compactSignalPolicyValueV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' {
			builder.WriteRune(r)
			continue
		}
		if builder.Len() > 0 {
			break
		}
	}
	return builder.String()
}
