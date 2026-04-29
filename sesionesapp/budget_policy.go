package sesionesapp

import (
	"encoding/json"
	"strings"
	"time"
)

type BudgetCandidate struct {
	RemainingPct *int
	WindowKind   string
	ResetAt      *time.Time
	Source       string
}

type BudgetQuotaSnapshot struct {
	RemainingSeconds  *int64
	RemainingMessages *int64
	RemainingTokens   *int64
	RemainingCredits  *float64
	Source            string
	RawSnapshotJSON   string
	CheckedAt         time.Time
	WindowStartedAt   *time.Time
	ResetAt           *time.Time
}

type BudgetEvaluation struct {
	Status           string
	ShouldHandoff    bool
	Reason           string
	ThresholdSeconds int64
	ThresholdRatio   float64
	RemainingRatio   *float64
}

func SelectEffectiveBudget(candidates []BudgetCandidate) BudgetCandidate {
	var weekly BudgetCandidate
	var short BudgetCandidate
	var fallback BudgetCandidate
	for _, item := range candidates {
		if item.RemainingPct == nil {
			continue
		}
		if BudgetWindowIsWeekly(item.WindowKind) {
			weekly = item
			continue
		}
		if short.RemainingPct == nil && BudgetWindowIsShort(item.WindowKind) {
			short = item
			continue
		}
		if fallback.RemainingPct == nil {
			fallback = item
		}
	}
	if weekly.RemainingPct != nil {
		if *weekly.RemainingPct <= 0 {
			return weekly
		}
		if short.RemainingPct != nil {
			return short
		}
		return weekly
	}
	if short.RemainingPct != nil {
		return short
	}
	if fallback.RemainingPct != nil {
		return fallback
	}
	return BudgetCandidate{}
}

func BudgetWindowIsWeekly(windowKind string) bool {
	return strings.EqualFold(strings.TrimSpace(windowKind), "weekly")
}

func BudgetWindowIsShort(windowKind string) bool {
	windowKind = strings.ToLower(strings.TrimSpace(windowKind))
	switch windowKind {
	case "5h", "session", "provider":
		return true
	}
	return strings.HasSuffix(windowKind, "m")
}

func ApplyEffectiveBudget(agent *Agente, candidate BudgetCandidate) {
	if agent == nil || candidate.RemainingPct == nil {
		return
	}
	agent.CuotaRestantePct = candidate.RemainingPct
	agent.PresupuestoVentana = candidate.WindowKind
	agent.PresupuestoResetAt = candidate.ResetAt
	if strings.TrimSpace(agent.PresupuestoFuente) == "" {
		agent.PresupuestoFuente = candidate.Source
	}
}

func BudgetSnapshotFresh(checkedAt time.Time, source string, now time.Time, observedMaxAgeSeconds, defaultMaxAgeSeconds int64) bool {
	if checkedAt.IsZero() {
		return false
	}
	maxAge := time.Duration(BudgetSnapshotMaxAgeSeconds(source, observedMaxAgeSeconds, defaultMaxAgeSeconds)) * time.Second
	if maxAge <= 0 {
		maxAge = 5 * time.Minute
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.Sub(checkedAt) <= maxAge
}

func BudgetSnapshotMaxAgeSeconds(source string, observedMaxAgeSeconds, defaultMaxAgeSeconds int64) int64 {
	if budgetSourceUsesObservedFreshness(source) && observedMaxAgeSeconds > 0 {
		return observedMaxAgeSeconds
	}
	return defaultMaxAgeSeconds
}

func budgetSourceUsesObservedFreshness(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "codex_token_count_observed", "codex_status_live", "codex_profile_status", "claude_rust_session_observed":
		return true
	default:
		return false
	}
}

func BudgetSnapshotContributesQuota(snapshot BudgetQuotaSnapshot) bool {
	if snapshot.RemainingSeconds != nil || snapshot.RemainingMessages != nil || snapshot.RemainingTokens != nil || snapshot.RemainingCredits != nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(snapshot.Source), "provider_backoff") {
		return true
	}
	rawSnapshot := strings.ToLower(strings.TrimSpace(snapshot.RawSnapshotJSON))
	if rawSnapshot == "" {
		return false
	}
	if strings.Contains(rawSnapshot, "\"rate_limits\"") && strings.Contains(rawSnapshot, "\"used_percent\"") {
		return true
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(snapshot.RawSnapshotJSON), &raw); err != nil || raw == nil {
		return false
	}
	rateLimits, _ := raw["rate_limits"].(map[string]any)
	if rateLimits == nil {
		return false
	}
	for _, key := range []string{"primary", "secondary"} {
		window, _ := rateLimits[key].(map[string]any)
		if window == nil {
			continue
		}
		if _, ok := budgetSnapshotFloat64(window["used_percent"]); ok {
			return true
		}
	}
	return false
}

func budgetSnapshotFloat64(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		f, err := json.Number(strings.TrimSpace(v)).Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func BudgetAccountCandidateScore(snapshot BudgetQuotaSnapshot, now time.Time) int64 {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if snapshot.CheckedAt.IsZero() {
		return -1
	}
	var score int64
	if BudgetSnapshotContributesQuota(snapshot) {
		score += 1_000_000_000
	}
	age := now.Sub(snapshot.CheckedAt.UTC())
	switch {
	case age <= 15*time.Minute:
		score += 100_000_000
	case age <= time.Hour:
		score += 75_000_000
	case age <= 6*time.Hour:
		score += 50_000_000
	case age <= 24*time.Hour:
		score += 25_000_000
	}
	switch strings.ToLower(strings.TrimSpace(snapshot.Source)) {
	case "codex_status_live", "claude_status_live":
		score += 70_000_000
	case "codex_profile_status":
		score += 65_000_000
	case "codex_token_count_observed", "claude_rust_session_observed":
		score += 60_000_000
	case "provider_backoff":
		score += 10_000_000
	default:
		score += 5_000_000
	}
	score += snapshot.CheckedAt.UTC().Unix()
	return score
}

func EvaluateBudgetSnapshot(snapshot BudgetQuotaSnapshot, thresholdSeconds int64, thresholdRatio float64) BudgetEvaluation {
	evaluation := BudgetEvaluation{
		Status:           "ok",
		ThresholdSeconds: thresholdSeconds,
		ThresholdRatio:   thresholdRatio,
	}
	evaluation.RemainingRatio = budgetRemainingRatio(snapshot)
	if budgetExhaustedByCount(snapshot) {
		evaluation.Status = "agotado"
		evaluation.ShouldHandoff = true
		evaluation.Reason = "presupuesto agotado"
		return evaluation
	}
	if snapshot.RemainingSeconds != nil && *snapshot.RemainingSeconds <= thresholdSeconds {
		evaluation.Status = "handoff_preventivo"
		evaluation.ShouldHandoff = true
		evaluation.Reason = "threshold_seconds"
		return evaluation
	}
	if evaluation.RemainingRatio != nil {
		if *evaluation.RemainingRatio <= thresholdRatio {
			evaluation.Status = "handoff_preventivo"
			evaluation.ShouldHandoff = true
			evaluation.Reason = "threshold_ratio"
			return evaluation
		}
		evaluation.Reason = "ratio_ok"
		return evaluation
	}
	if snapshot.RemainingSeconds != nil {
		evaluation.Reason = "remaining_seconds_only"
		return evaluation
	}
	evaluation.Status = "sin_datos"
	evaluation.Reason = "sin telemetría suficiente para decidir handoff"
	return evaluation
}

func budgetExhaustedByCount(snapshot BudgetQuotaSnapshot) bool {
	return int64PtrLTEZero(snapshot.RemainingSeconds) ||
		int64PtrLTEZero(snapshot.RemainingMessages) ||
		int64PtrLTEZero(snapshot.RemainingTokens) ||
		float64PtrLTEZero(snapshot.RemainingCredits) ||
		budgetSnapshotRateLimitExhausted(snapshot)
}

func int64PtrLTEZero(v *int64) bool {
	return v != nil && *v <= 0
}

func float64PtrLTEZero(v *float64) bool {
	return v != nil && *v <= 0
}

func budgetRemainingRatio(snapshot BudgetQuotaSnapshot) *float64 {
	if budgetExhaustedByCount(snapshot) {
		ratio := 0.0
		return &ratio
	}
	if snapshot.RemainingSeconds != nil && snapshot.WindowStartedAt != nil && snapshot.ResetAt != nil {
		total := snapshot.ResetAt.Sub(*snapshot.WindowStartedAt).Seconds()
		if total > 0 {
			ratio := float64(*snapshot.RemainingSeconds) / total
			if ratio < 0 {
				ratio = 0
			}
			return &ratio
		}
	}
	return budgetSnapshotRemainingRatio(snapshot)
}

func budgetSnapshotRemainingRatio(snapshot BudgetQuotaSnapshot) *float64 {
	var raw map[string]any
	if err := json.Unmarshal([]byte(snapshot.RawSnapshotJSON), &raw); err != nil || raw == nil {
		return nil
	}
	rateLimits, _ := raw["rate_limits"].(map[string]any)
	if rateLimits == nil {
		return nil
	}
	primary, _ := rateLimits["primary"].(map[string]any)
	if primary == nil {
		return nil
	}
	used, ok := budgetSnapshotFloat64(primary["used_percent"])
	if !ok {
		return nil
	}
	ratio := 1 - (used / 100)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return &ratio
}

func budgetSnapshotRateLimitExhausted(snapshot BudgetQuotaSnapshot) bool {
	ratio := budgetSnapshotRemainingRatio(snapshot)
	return ratio != nil && *ratio <= 0
}
