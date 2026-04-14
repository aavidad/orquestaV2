package sesionesapp

import (
	"strings"
	"time"
)

type BudgetCandidate struct {
	RemainingPct *int
	WindowKind   string
	ResetAt      *time.Time
	Source       string
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
