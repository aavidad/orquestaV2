package orquestaruntimecodexappserver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

const codexAppServerGoalHighTokenUsageThresholdDefaultV0 = 100000

func (backend serverCodexAppServerGoalBackendV0) codexAppServerStartImmediateLimitedReceiptV0(
	ctx context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
	threadID string,
	goalSetEvidence string,
	turnStartEvidenceRefs []string,
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
	evidenceRefs := []string{
		"evidence-ref-codex-app-server-thread-started",
		goalSetEvidence,
	}
	evidenceRefs = append(evidenceRefs, turnStartEvidenceRefs...)
	evidenceRefs = append(evidenceRefs, evidenceRef)
	receipt.EvidenceRefs = compactServerStackStringsV0(evidenceRefs)
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

func (backend serverCodexAppServerGoalBackendV0) codexAppServerObservationReceiptWithGoalUsageV0(
	receipt orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
	goal *serverCodexAppServerThreadGoalV0,
) orquestaruntimecodexgoal.CodexGoalObservationReceiptV0 {
	if goal == nil {
		return receipt
	}
	threadID := strings.TrimSpace(firstNonEmptyServerStackV0(goal.ThreadID, receipt.ExternalGoalRef))
	now := backend.nowCodexAppServerGoalV0()
	observedAt, ok := goal.UpdatedAt.TimeV0(now)
	if !ok {
		observedAt = now
	}
	receipt.UsageObservation = orquestagoal.GoalUsageObservationV0{
		TokensAccumulated: int64(goal.TokensUsed),
		RuntimeSeconds:    int64(goal.TimeUsedSeconds),
		ObservedAt:        observedAt.UTC().Format(time.RFC3339Nano),
		EvidenceRefs:      []string{"evidence-ref-codex-app-server-goal-usage-observed"},
		SourceRef:         codexAppServerGoalUsageSourceRefV0(threadID),
	}
	if strings.TrimSpace(receipt.Status) != orquestagoal.GoalStatusRunningV0 ||
		goal.TokensUsed < backend.codexAppServerGoalHighTokenUsageThresholdV0() {
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

func codexAppServerGoalUsageSourceRefV0(threadID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(threadID)))
	return fmt.Sprintf("codex-app-server-goal-ref-%x", sum[:])
}

func (backend serverCodexAppServerGoalBackendV0) codexAppServerGoalHighTokenUsageThresholdV0() int {
	if backend.HighTokenUsageThreshold > 0 {
		return backend.HighTokenUsageThreshold
	}
	return codexAppServerGoalHighTokenUsageThresholdDefaultV0
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
