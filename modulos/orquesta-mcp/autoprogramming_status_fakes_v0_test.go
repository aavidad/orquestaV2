package orquestamcp

import (
	"context"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func hasMCPAutoprogrammingDiagnosticCodeV0(diagnostics []MCPAutoprogrammingDiagnosticV0, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

type fakeMCPAutoprogrammingQueueStatusV0 struct {
	input MCPRunQueuePriorityToolInputV0
	empty bool
}

func (fake *fakeMCPAutoprogrammingQueueStatusV0) Execute(
	_ context.Context,
	input MCPRunQueuePriorityToolInputV0,
) (MCPRunQueuePriorityToolResultV0, error) {
	fake.input = input
	result := MCPRunQueuePriorityToolResultV0{
		Estado:        MCPRunQueuePriorityEstadoOKV0,
		CorrelationID: input.CorrelationID,
		Action:        MCPRunQueuePriorityActionRankV0,
		QueueRef:      input.QueueRef,
		Ranked:        []MCPRunQueueRankedCandidateCompactV0{},
		Errores:       []MCPValidationIssueV0{},
	}
	if fake.empty {
		return result, nil
	}
	result.Count = 1
	result.Ranked = []MCPRunQueueRankedCandidateCompactV0{{
		Rank:          1,
		RunRef:        "run-ref-autop-status-001",
		AppRef:        "app-ref-autop-status-001",
		Status:        "running",
		PriorityScore: 50,
	}}
	return result, nil
}

type fakeMCPAutoprogrammingRunStatusV0 struct {
	input MCPDirectorStatsToolInputV0
	stats *orquestacionnucleoapp.DirectorRunStatsV0
}

func (fake *fakeMCPAutoprogrammingRunStatusV0) Execute(
	_ context.Context,
	input MCPDirectorStatsToolInputV0,
) (MCPDirectorStatsToolResultV0, error) {
	fake.input = input
	stats := fake.stats
	if stats == nil {
		stats = defaultFakeMCPAutoprogrammingStatsV0(input.RunRef)
	}
	return MCPDirectorStatsToolResultV0{
		Estado:        MCPDirectorStatsEstadoOKV0,
		CorrelationID: input.CorrelationID,
		RunRef:        input.RunRef,
		Stats:         stats,
		Errores:       []MCPValidationIssueV0{},
	}, nil
}

func defaultFakeMCPAutoprogrammingStatsV0(runRef string) *orquestacionnucleoapp.DirectorRunStatsV0 {
	return &orquestacionnucleoapp.DirectorRunStatsV0{
		RunRef:     runRef,
		ProjectRef: "app-ref-autop-status-001",
		Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
			TasksTotal:     1,
			AgentsInFlight: 1,
		},
		Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
			Tasks: []orquestacionnucleoapp.DirectorTaskProgressV0{{
				TaskRef:             "task-ref-autop-status-001",
				Status:              "in_progress",
				AgentRequestID:      "agent-ref-autop-status-001",
				DeliveryRef:         "delivery-ref-autop-status-001",
				LastReportRef:       "report-ref-autop-status-001",
				ProgressStatus:      "progressing",
				Summary:             "cerrar panel operativo",
				NoProgressTicks:     1,
				RepeatedActionCount: 1,
				DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
					Classification:   "working",
					BudgetReason:     "within_budget",
					DecisionRequired: true,
				},
				EvidenceRefs: []string{"evidence-ref-task-autop-status-001"},
			}},
		},
		Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
			Status:      orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
			Blocked:     true,
			BlockedBy:   []string{"revision_final"},
			BlockerRefs: []string{"blocker-ref-autop-status-001"},
		},
		Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
			AgentRequestID: "agent-ref-autop-status-001",
			Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
			InFlight:       true,
			LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
				TaskRef:         "task-ref-autop-status-001",
				DeliveryRef:     "delivery-ref-autop-status-001",
				ReportRef:       "report-ref-autop-status-001",
				Status:          "progressing",
				NoProgressTicks: 1,
				DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
					Classification: "working",
					BudgetReason:   "within_budget",
				},
				EvidenceRefs: []string{"evidence-ref-agent-progress-001"},
			},
			Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
				ProcessRef:   "process-ref-autop-status-001",
				SessionRef:   "session-ref-autop-status-001",
				EvidenceRefs: []string{"evidence-ref-agent-process-001"},
			},
			Usage: &orquestacionnucleoapp.DirectorAgentUsageStatsV0{
				RuntimeKind:   "cli",
				CapacityLevel: "medium",
				QuotaStatus:   "available",
				TotalTokens:   123,
				EvidenceRefs:  []string{"evidence-ref-agent-usage-001"},
			},
		}},
	}
}
