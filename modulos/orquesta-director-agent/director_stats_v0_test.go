package orquestadirectoragent

import "testing"

func TestValidateDirectorAgentCompactStatsV0AceptaStatsCompactas(t *testing.T) {
	stats := DirectorAgentCompactStatsV0{
		SchemaVersion: DirectorAgentCompactStatsSchemaVersionV0,
		RunID:         "run-ref-001",
		Status:        "activa",
		CurrentPhase:  "programacion",
		Totals: DirectorAgentStatsTotalsV0{
			Tasks:            3,
			CapacityRequests: 1,
			Agents:           1,
		},
		PendingRefs:  []string{"tasks_open", "capacity_pending", "tasks_open"},
		EvidenceRefs: []string{"workflow-run:run-ref-001"},
	}

	normalized := NormalizeDirectorAgentCompactStatsV0(stats)
	if len(normalized.PendingRefs) != 2 {
		t.Fatalf("pending_refs=%+v", normalized.PendingRefs)
	}
	if issues := ValidateDirectorAgentCompactStatsV0(normalized); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentCompactStatsV0RechazaDetalleOperativo(t *testing.T) {
	stats := DirectorAgentCompactStatsV0{
		SchemaVersion: DirectorAgentCompactStatsSchemaVersionV0,
		RunID:         "run-ref-001",
		Status:        "activa",
		CurrentPhase:  "programacion",
		PendingRefs:   []string{"/home/user/secret"},
	}

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentCompactStatsV0(stats),
		"director_agent_ref_invalida",
	)
}
