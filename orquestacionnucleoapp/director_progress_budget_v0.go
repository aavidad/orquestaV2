package orquestacionnucleoapp

import (
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func directorProgressTemporalFromReportV0(
	report orquestaruntime.AgentProgressReportV0,
) DirectorProgressTemporalV0 {
	return DirectorProgressTemporalV0{
		Classification:         strings.TrimSpace(string(report.BudgetStatus)),
		BudgetReason:           strings.TrimSpace(report.BudgetReason),
		AgeSeconds:             report.AgeSeconds,
		SecondsSinceActivity:   report.SecondsSinceActivity,
		SecondsSinceAck:        report.SecondsSinceAck,
		MaxExpectedSeconds:     report.MaxExpectedSeconds,
		NoActivityLimitSeconds: report.NoActivityLimitSeconds,
		StartedAt:              strings.TrimSpace(report.StartedAt),
		LastActivityAt:         strings.TrimSpace(report.LastActivityAt),
		LastAckAt:              strings.TrimSpace(report.LastAckAt),
		DecisionRequired:       report.DecisionRequired,
	}
}
