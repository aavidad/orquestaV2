package sesionesapp

import (
	"testing"
	"time"
)

func TestSelectEffectiveBudgetPrefersShortWindowOverPositiveWeekly(t *testing.T) {
	weeklyPct := 70
	shortPct := 20

	got := SelectEffectiveBudget([]BudgetCandidate{
		{RemainingPct: &weeklyPct, WindowKind: "weekly", Source: "weekly"},
		{RemainingPct: &shortPct, WindowKind: "5h", Source: "short"},
	})
	if got.RemainingPct == nil || *got.RemainingPct != shortPct || got.WindowKind != "5h" {
		t.Fatalf("candidato efectivo inesperado: %+v", got)
	}
}

func TestSelectEffectiveBudgetPrefersExhaustedWeeklyWindow(t *testing.T) {
	weeklyPct := 0
	shortPct := 50

	got := SelectEffectiveBudget([]BudgetCandidate{
		{RemainingPct: &weeklyPct, WindowKind: "weekly"},
		{RemainingPct: &shortPct, WindowKind: "5h"},
	})
	if got.RemainingPct == nil || *got.RemainingPct != weeklyPct || got.WindowKind != "weekly" {
		t.Fatalf("deberia priorizar semanal agotada: %+v", got)
	}
}

func TestApplyEffectiveBudget(t *testing.T) {
	pct := 85
	agent := &Agente{}

	ApplyEffectiveBudget(agent, BudgetCandidate{
		RemainingPct: &pct,
		WindowKind:   "weekly",
		Source:       "derived_weekly",
	})
	if agent.CuotaRestantePct == nil || *agent.CuotaRestantePct != 85 {
		t.Fatalf("cuota inesperada: %+v", agent.CuotaRestantePct)
	}
	if agent.PresupuestoVentana != "weekly" || agent.PresupuestoFuente != "derived_weekly" {
		t.Fatalf("presupuesto aplicado inesperado: %+v", agent)
	}
}

func TestBudgetSnapshotFreshUsesObservedTTL(t *testing.T) {
	now := time.Now().UTC()
	if !BudgetSnapshotFresh(now.Add(-30*time.Minute), "codex_status_live", now, 3600, 300) {
		t.Fatal("codex_status_live deberia seguir fresco con TTL observado")
	}
}

func TestBudgetSnapshotFreshUsesDefaultTTL(t *testing.T) {
	now := time.Now().UTC()
	if BudgetSnapshotFresh(now.Add(-10*time.Minute), "manual", now, 3600, 300) {
		t.Fatal("manual no deberia seguir fresco con TTL por defecto")
	}
}

func TestBudgetSnapshotContributesQuota(t *testing.T) {
	remaining := int64(42)
	if !BudgetSnapshotContributesQuota(BudgetQuotaSnapshot{RemainingSeconds: &remaining}) {
		t.Fatal("remaining_seconds deberia aportar cuota")
	}
	if !BudgetSnapshotContributesQuota(BudgetQuotaSnapshot{Source: "provider_backoff"}) {
		t.Fatal("provider_backoff deberia aportar cuota")
	}
	if !BudgetSnapshotContributesQuota(BudgetQuotaSnapshot{RawSnapshotJSON: `{"rate_limits":{"primary":{"used_percent":12}}}`}) {
		t.Fatal("used_percent deberia aportar cuota")
	}
	if BudgetSnapshotContributesQuota(BudgetQuotaSnapshot{RawSnapshotJSON: `{"rate_limits":{"primary":{"window_minutes":300}}}`}) {
		t.Fatal("sin used_percent no deberia aportar cuota")
	}
}

func TestBudgetAccountCandidateScorePrefersFresherAndStrongerSource(t *testing.T) {
	now := time.Now().UTC()
	remaining := int64(30)
	live := BudgetAccountCandidateScore(BudgetQuotaSnapshot{
		RemainingSeconds: &remaining,
		Source:           "codex_status_live",
		CheckedAt:        now.Add(-10 * time.Minute),
	}, now)
	observed := BudgetAccountCandidateScore(BudgetQuotaSnapshot{
		RemainingSeconds: &remaining,
		Source:           "codex_token_count_observed",
		CheckedAt:        now.Add(-10 * time.Minute),
	}, now)
	stale := BudgetAccountCandidateScore(BudgetQuotaSnapshot{
		RemainingSeconds: &remaining,
		Source:           "codex_status_live",
		CheckedAt:        now.Add(-8 * time.Hour),
	}, now)
	if live <= observed {
		t.Fatalf("codex_status_live deberia puntuar por encima de observed: live=%d observed=%d", live, observed)
	}
	if observed <= stale {
		t.Fatalf("snapshot fresco deberia puntuar por encima de uno mas viejo: observed=%d stale=%d", observed, stale)
	}
}
