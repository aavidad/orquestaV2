package orquestaruntimecodexdelivery

import "time"

type CodexBudgetActivityStatusV0 string

const (
	CodexBudgetActivityWorkingV0              CodexBudgetActivityStatusV0 = "working"
	CodexBudgetActivityStalledV0              CodexBudgetActivityStatusV0 = "stalled"
	CodexBudgetActivityOverBudgetButActiveV0  CodexBudgetActivityStatusV0 = "over_budget_but_active"
	CodexBudgetActivityOverBudgetNoActivityV0 CodexBudgetActivityStatusV0 = "over_budget_no_activity"
)

type CodexBudgetActivityReasonV0 string

const (
	CodexBudgetActivityReasonWorkingV0              CodexBudgetActivityReasonV0 = "within_budget_activity_recent"
	CodexBudgetActivityReasonStalledV0              CodexBudgetActivityReasonV0 = "within_budget_no_recent_activity"
	CodexBudgetActivityReasonOverBudgetButActiveV0  CodexBudgetActivityReasonV0 = "over_budget_activity_recent"
	CodexBudgetActivityReasonOverBudgetNoActivityV0 CodexBudgetActivityReasonV0 = "over_budget_no_recent_activity"
)

type CodexBudgetActivityPolicyV0 struct {
	MaxExpected     time.Duration `json:"max_expected"`
	NoActivityLimit time.Duration `json:"no_activity_limit"`
}

type CodexBudgetActivityInputV0 struct {
	StartedAt       time.Time     `json:"started_at"`
	LastActivity    time.Time     `json:"last_activity,omitempty"`
	LastAck         time.Time     `json:"last_ack,omitempty"`
	MaxExpected     time.Duration `json:"max_expected"`
	NoActivityLimit time.Duration `json:"no_activity_limit"`
	ObservedAt      time.Time     `json:"observed_at,omitempty"`
}

type CodexBudgetActivityClassificationV0 struct {
	Status CodexBudgetActivityStatusV0 `json:"status"`
	Reason CodexBudgetActivityReasonV0 `json:"reason"`
}

func ClassifyCodexBudgetActivityV0(
	input CodexBudgetActivityInputV0,
) CodexBudgetActivityClassificationV0 {
	input = normalizeCodexBudgetActivityInputV0(input)
	overBudget := codexBudgetActivityOverBudgetV0(input)
	noActivity := codexBudgetActivityNoActivityV0(input)
	switch {
	case overBudget && noActivity:
		return codexBudgetActivityResultV0(
			CodexBudgetActivityOverBudgetNoActivityV0,
			CodexBudgetActivityReasonOverBudgetNoActivityV0,
		)
	case overBudget:
		return codexBudgetActivityResultV0(
			CodexBudgetActivityOverBudgetButActiveV0,
			CodexBudgetActivityReasonOverBudgetButActiveV0,
		)
	case noActivity:
		return codexBudgetActivityResultV0(
			CodexBudgetActivityStalledV0,
			CodexBudgetActivityReasonStalledV0,
		)
	default:
		return codexBudgetActivityResultV0(
			CodexBudgetActivityWorkingV0,
			CodexBudgetActivityReasonWorkingV0,
		)
	}
}

func normalizeCodexBudgetActivityInputV0(
	input CodexBudgetActivityInputV0,
) CodexBudgetActivityInputV0 {
	if input.ObservedAt.IsZero() {
		input.ObservedAt = time.Now().UTC()
	}
	input.ObservedAt = input.ObservedAt.UTC()
	input.StartedAt = input.StartedAt.UTC()
	input.LastActivity = input.LastActivity.UTC()
	input.LastAck = input.LastAck.UTC()
	return input
}

func codexBudgetActivityOverBudgetV0(input CodexBudgetActivityInputV0) bool {
	if input.StartedAt.IsZero() || input.MaxExpected <= 0 {
		return false
	}
	return input.ObservedAt.Sub(input.StartedAt) > input.MaxExpected
}

func codexBudgetActivityNoActivityV0(input CodexBudgetActivityInputV0) bool {
	if input.NoActivityLimit <= 0 {
		return false
	}
	lastActivity := codexBudgetActivityLastSignalAtV0(input)
	if lastActivity.IsZero() {
		return true
	}
	return input.ObservedAt.Sub(lastActivity) > input.NoActivityLimit
}

func codexBudgetActivityLastSignalAtV0(input CodexBudgetActivityInputV0) time.Time {
	last := input.StartedAt
	last = codexBudgetActivityMaxSignalV0(input, last, input.LastActivity)
	last = codexBudgetActivityMaxSignalV0(input, last, input.LastAck)
	return last
}

func codexBudgetActivityMaxSignalV0(
	input CodexBudgetActivityInputV0,
	current time.Time,
	next time.Time,
) time.Time {
	if next.IsZero() || next.Before(input.StartedAt) {
		return current
	}
	if next.After(input.ObservedAt) {
		next = input.ObservedAt
	}
	if current.IsZero() || next.After(current) {
		return next
	}
	return current
}

func codexBudgetActivityResultV0(
	status CodexBudgetActivityStatusV0,
	reason CodexBudgetActivityReasonV0,
) CodexBudgetActivityClassificationV0 {
	return CodexBudgetActivityClassificationV0{
		Status: status,
		Reason: reason,
	}
}
