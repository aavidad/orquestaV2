package orquestaserver

import (
	"strconv"
	"strings"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const idleSelfImprovementBudgetEvidenceRefV0 = "evidence-ref-idle-self-improvement-budget-v0"

func (runtime *RuntimeV0) applyIdleSelfImprovementBudgetV0(
	decision idleSelfImprovementScheduleDecisionV0,
	now time.Time,
) idleSelfImprovementScheduleDecisionV0 {
	if !decision.Schedule {
		return decision
	}
	requested := decision.MaxRequests
	if requested <= 0 {
		requested = runtime.config.IdleSelfImprovementMaxRequests
	}
	budgetDecision := orquestaautoprogramming.DecideAutoprogrammingIdleSelfImprovementBudgetV0(
		orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionInputV0{
			Config:         runtime.config.IdleSelfImprovementBudget,
			Usage:          runtime.idleSelfImprovementBudgetUsageV0(now),
			RequestedGoals: requested,
		},
	)
	decision.BudgetDecision = budgetDecision
	if !budgetDecision.Prepare {
		decision.Schedule = false
		decision.AuditStatus = "blocked"
		decision.Reason = orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0
		decision.BlockerEvidence = compactConfigStringsV0(append(
			decision.BlockerEvidence,
			idleSelfImprovementBudgetEvidenceRefV0,
		))
		decision.BlockerMessage = "idle_self_improvement_budget_deferred"
		decision.BlockerRecoveryAction = "wait_for_budget_window_or_raise_declared_budget"
		decision.BlockerNextActions = compactConfigStringsV0(append(
			decision.BlockerNextActions,
			"observe_autoprogramming_status_budget",
			"wait_for_next_budget_window_or_adjust_budget",
		))
		return decision
	}
	if budgetDecision.Degraded && budgetDecision.AllowedGoals > 0 {
		decision.MaxRequests = budgetDecision.AllowedGoals
		decision.Reason = orquestaautoprogramming.AutoprogrammingIdleBudgetDegradedV0
	}
	return decision
}

func (runtime *RuntimeV0) idleSelfImprovementBudgetUsageV0(
	now time.Time,
) orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetUsageV0 {
	if runtime == nil || runtime.tracker == nil {
		return orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetUsageV0{}
	}
	state := runtime.tracker.SnapshotV0()
	estimate := idleSelfImprovementContextBudgetEstimateV0(state)
	usage := orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetUsageV0{
		EstimatedNextContextBudgetBytes:   estimate.ContextBudgetTotalBytes,
		PromptCacheCachedInputTokensToday: idleSelfImprovementPromptCacheTokensV0(state),
	}
	if idleSelfImprovementCheckSameDayV0(state.IdleSelfImprovementCheck, now) {
		usage.GoalsUsedToday = state.IdleSelfImprovementRuns
		usage.ContextBudgetBytesUsedToday = estimate.ContextBudgetTotalBytes
	}
	return usage
}

func idleSelfImprovementContextBudgetEstimateV0(state StateV0) orquestagoal.GoalContextBudgetV0 {
	metric := orquestagoal.GoalContextBudgetV0{}
	if state.IdleSelfImprovementGoalReceipt != nil {
		metric = orquestagoal.MergeGoalContextBudgetV0(metric, state.IdleSelfImprovementGoalReceipt.ContextBudget)
	}
	if state.IdleSelfImprovementGoalResult != nil {
		metric = orquestagoal.MergeGoalContextBudgetV0(metric, state.IdleSelfImprovementGoalResult.ContextBudget)
	}
	return orquestagoal.NormalizeGoalContextBudgetV0(metric)
}

func idleSelfImprovementCheckSameDayV0(value string, now time.Time) bool {
	if strings.TrimSpace(value) == "" || now.IsZero() {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return false
	}
	year, month, day := parsed.UTC().Date()
	nowYear, nowMonth, nowDay := now.UTC().Date()
	return year == nowYear && month == nowMonth && day == nowDay
}

func idleSelfImprovementPromptCacheTokensV0(state StateV0) int64 {
	var refs []string
	if state.IdleSelfImprovementGoalReceipt != nil {
		refs = append(refs, state.IdleSelfImprovementGoalReceipt.EvidenceRefs...)
	}
	if state.IdleSelfImprovementGoalResult != nil {
		refs = append(refs, state.IdleSelfImprovementGoalResult.EvidenceRefs...)
	}
	var total int64
	for _, ref := range refs {
		tokenText, ok := strings.CutPrefix(strings.TrimSpace(ref), "evidence-ref-codex-goal-cached-input-tokens-")
		if !ok || tokenText == "" {
			continue
		}
		value, err := strconv.ParseInt(tokenText, 10, 64)
		if err == nil && value > 0 {
			total += value
		}
	}
	return total
}
