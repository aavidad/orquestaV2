package orquestaruntimecodexdelivery

import (
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (source CodexProgressObservationSourceV0) reportWithBudgetV0(
	report orquestaruntime.AgentProgressReportV0,
	state CodexProgressObservationStateV0,
) orquestaruntime.AgentProgressReportV0 {
	if !codexProgressBudgetPolicyEnabledV0(source.BudgetPolicy) {
		return report
	}
	now := codexProgressObservedAtV0(state)
	startedAt := codexProgressStartedAtV0(state, now)
	lastActivity := codexProgressLastActivityAtV0(state, startedAt)
	classification := ClassifyCodexBudgetActivityV0(CodexBudgetActivityInputV0{
		StartedAt:       startedAt,
		LastActivity:    lastActivity,
		LastAck:         state.LastAckAt,
		MaxExpected:     source.BudgetPolicy.MaxExpected,
		NoActivityLimit: source.BudgetPolicy.NoActivityLimit,
		ObservedAt:      now,
	})
	report.BudgetStatus = codexRuntimeBudgetStatusV0(classification.Status)
	report.BudgetReason = string(classification.Reason)
	report.StartedAt = codexProgressTimeStringV0(startedAt)
	report.LastActivityAt = codexProgressTimeStringV0(lastActivity)
	report.LastAckAt = codexProgressTimeStringV0(state.LastAckAt)
	report.AgeSeconds = codexProgressDurationSecondsV0(now.Sub(startedAt))
	report.SecondsSinceActivity = codexProgressDurationSecondsV0(now.Sub(lastActivity))
	report.SecondsSinceAck = codexProgressSecondsSinceAckV0(now, startedAt, state.LastAckAt)
	report.MaxExpectedSeconds = codexProgressDurationSecondsV0(source.BudgetPolicy.MaxExpected)
	report.NoActivityLimitSeconds = codexProgressDurationSecondsV0(source.BudgetPolicy.NoActivityLimit)
	report.DecisionRequired = codexProgressBudgetDecisionRequiredV0(report.BudgetStatus)
	return report
}

func codexProgressBudgetPolicyEnabledV0(policy CodexBudgetActivityPolicyV0) bool {
	return policy.MaxExpected > 0 || policy.NoActivityLimit > 0
}

func codexProgressBudgetDecisionRequiredV0(
	status orquestaruntime.AgentProgressBudgetStatusV0,
) bool {
	return status == orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0 ||
		status == orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0
}

func codexRuntimeBudgetStatusV0(
	status CodexBudgetActivityStatusV0,
) orquestaruntime.AgentProgressBudgetStatusV0 {
	switch status {
	case CodexBudgetActivityStalledV0:
		return orquestaruntime.AgentProgressBudgetStalledV0
	case CodexBudgetActivityOverBudgetButActiveV0:
		return orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0
	case CodexBudgetActivityOverBudgetNoActivityV0:
		return orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0
	default:
		return orquestaruntime.AgentProgressBudgetWorkingV0
	}
}

func codexProgressObservedAtV0(state CodexProgressObservationStateV0) time.Time {
	if state.ObservedAt.IsZero() {
		return orquestaruntime.NowUTCV0(nil)
	}
	return state.ObservedAt.UTC()
}

func codexProgressStartedAtV0(
	state CodexProgressObservationStateV0,
	fallback time.Time,
) time.Time {
	if state.FirstObservedAt.IsZero() {
		return fallback
	}
	return state.FirstObservedAt.UTC()
}

func codexProgressLastActivityAtV0(
	state CodexProgressObservationStateV0,
	fallback time.Time,
) time.Time {
	if state.LastActivityAt.IsZero() {
		return fallback
	}
	return state.LastActivityAt.UTC()
}

func codexProgressTimeStringV0(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func codexProgressDurationSecondsV0(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return int64(duration / time.Second)
}

func codexProgressSecondsSinceAckV0(
	now time.Time,
	startedAt time.Time,
	lastAckAt time.Time,
) int64 {
	if lastAckAt.IsZero() {
		return codexProgressDurationSecondsV0(now.Sub(startedAt))
	}
	return codexProgressDurationSecondsV0(now.Sub(lastAckAt))
}
