package orquestadirectoragent

import "strings"

func NormalizeDirectorAgentCompactStatsV0(
	stats DirectorAgentCompactStatsV0,
) DirectorAgentCompactStatsV0 {
	return DirectorAgentCompactStatsV0{
		SchemaVersion: strings.TrimSpace(stats.SchemaVersion),
		RunID:         strings.TrimSpace(stats.RunID),
		Status:        strings.TrimSpace(stats.Status),
		CurrentPhase:  strings.TrimSpace(stats.CurrentPhase),
		Totals:        stats.Totals,
		PendingRefs:   compactDirectorAgentStringsV0(stats.PendingRefs),
		EvidenceRefs:  compactDirectorAgentStringsV0(stats.EvidenceRefs),
	}
}

func ValidateDirectorAgentCompactStatsV0(
	stats DirectorAgentCompactStatsV0,
) []DirectorAgentDecisionIssueV0 {
	stats = NormalizeDirectorAgentCompactStatsV0(stats)
	v := directorAgentDecisionValidatorV0{}
	v.require("schema_version", stats.SchemaVersion, DirectorAgentCompactStatsSchemaVersionV0)
	v.requireRef("run_id", stats.RunID)
	v.requireRef("status", stats.Status)
	v.requireRef("current_phase", stats.CurrentPhase)
	v.requireEvidence("pending_refs", stats.PendingRefs)
	v.requireEvidence("evidence_refs", stats.EvidenceRefs)
	if stats.Totals.hasNegativeV0() {
		v.add("director_agent_stats_total_invalido", "totals")
	}
	return v.issues
}

func DirectorAgentCompactStatsValidV0(stats DirectorAgentCompactStatsV0) bool {
	return len(ValidateDirectorAgentCompactStatsV0(stats)) == 0
}

func (totals DirectorAgentStatsTotalsV0) hasNegativeV0() bool {
	return totals.Phases < 0 ||
		totals.Tasks < 0 ||
		totals.CapacityRequests < 0 ||
		totals.Agents < 0 ||
		totals.Deliveries < 0 ||
		totals.ReviewResults < 0 ||
		totals.ReworkRequests < 0 ||
		totals.ReplanDecisions < 0 ||
		totals.ClosedTasks < 0 ||
		totals.Validations < 0 ||
		totals.Closures < 0
}
