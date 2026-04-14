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
