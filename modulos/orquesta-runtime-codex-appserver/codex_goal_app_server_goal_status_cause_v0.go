package orquestaruntimecodexappserver

import (
	"context"
	"fmt"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

const codexAppServerGoalHighTokenUsageThresholdV0 = 100000

func (backend serverCodexAppServerGoalBackendV0) codexAppServerStartImmediateLimitedReceiptV0(
	ctx context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
	threadID string,
	goalSetEvidence string,
) (orquestaruntimecodexgoal.CodexGoalStartReceiptV0, bool) {
	if backend.Protocol == nil {
		return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{}, false
	}
	goal, err := backend.Protocol.GetGoalV0(ctx, threadID)
	if err != nil || goal == nil {
		return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{}, false
	}
	code, evidenceRef := codexAppServerGoalStatusIssueCodeV0(goal.Status)
	if code == "" {
		return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{}, false
	}
	receipt := codexAppServerStartReceiptV0(packet, threadID, code)
	receipt.Status = orquestagoal.GoalStatusBlockedV0
	receipt.EvidenceRefs = compactServerStackStringsV0([]string{
		"evidence-ref-codex-app-server-thread-started",
		goalSetEvidence,
		"evidence-ref-codex-app-server-turn-started",
		evidenceRef,
	})
	return receipt, true
}

func codexAppServerObservationReceiptWithGoalStatusCauseV0(
	receipt orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
	status string,
) orquestaruntimecodexgoal.CodexGoalObservationReceiptV0 {
	code, evidenceRef := codexAppServerGoalStatusIssueCodeV0(status)
	if code == "" {
		return receipt
	}
	receipt.IssueCode = code
	receipt.EvidenceRefs = compactServerStackStringsV0(append(receipt.EvidenceRefs, evidenceRef))
	return receipt
}

func codexAppServerObservationReceiptWithGoalUsageV0(
	receipt orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
	goal *serverCodexAppServerThreadGoalV0,
) orquestaruntimecodexgoal.CodexGoalObservationReceiptV0 {
	if goal == nil || strings.TrimSpace(receipt.Status) != orquestagoal.GoalStatusRunningV0 {
		return receipt
	}
	if goal.TokensUsed < codexAppServerGoalHighTokenUsageThresholdV0 {
		return receipt
	}
	usage := []string{
		"codex_app_server_goal_status_active_high_token_usage",
		fmt.Sprintf("tokens_used=%d", goal.TokensUsed),
	}
	if goal.TimeUsedSeconds > 0 {
		usage = append(usage, fmt.Sprintf("time_used_seconds=%d", goal.TimeUsedSeconds))
	}
	if goal.TokenBudget != nil && *goal.TokenBudget > 0 {
		usage = append(usage, fmt.Sprintf("token_budget=%d", *goal.TokenBudget))
	}
	receipt.Summary = strings.Join(usage, " ")
	receipt.EvidenceRefs = compactServerStackStringsV0(append(
		receipt.EvidenceRefs,
		"evidence-ref-codex-app-server-goal-high-token-usage",
	))
	return receipt
}

func codexAppServerGoalStatusIssueCodeV0(status string) (string, string) {
	switch strings.TrimSpace(status) {
	case "usageLimited", "quotaLimited", "providerLimited":
		return "codex_app_server_goal_provider_limited", "evidence-ref-codex-app-server-goal-provider-limited"
	case "budgetLimited":
		return "codex_app_server_goal_budget_limited", "evidence-ref-codex-app-server-goal-budget-limited"
	case "policyLimited":
		return "codex_app_server_goal_policy_limited", "evidence-ref-codex-app-server-goal-policy-limited"
	default:
		return "", ""
	}
}

func codexAppServerGoalStatusToGoalWorkStatusV0(status string) string {
	switch strings.TrimSpace(status) {
	case "complete":
		return orquestagoal.GoalStatusCompleteV0
	case "blocked", "usageLimited", "budgetLimited", "policyLimited", "quotaLimited", "providerLimited":
		return orquestagoal.GoalStatusBlockedV0
	case "active", "paused":
		return orquestagoal.GoalStatusRunningV0
	default:
		return orquestagoal.GoalStatusInvalidV0
	}
}
