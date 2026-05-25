package orquestaweb

import (
	"encoding/json"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestWebAutoprogrammingStatusViewModelV0AgregaColaYRunVivos(t *testing.T) {
	vm := NewWebAutoprogrammingStatusViewModelV0("es", orquestamcp.MCPAutoprogrammingStatusToolResultV0{
		Estado:   orquestamcp.MCPAutoprogrammingStatusEstadoOKV0,
		QueueRef: "queue-ref-autop-status-web-001",
		RunRef:   "run-ref-autop-status-web-001",
		Queue: &orquestamcp.MCPRunQueuePriorityToolResultV0{
			Estado:   orquestamcp.MCPRunQueuePriorityEstadoOKV0,
			Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
			QueueRef: "queue-ref-autop-status-web-001",
			Count:    1,
			Ranked: []orquestamcp.MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-autop-status-web-queued",
				AppRef:        "app-ref-autop-status-web-queued",
				Status:        "ready",
				PriorityScore: 95,
			}},
		},
		Run: &orquestamcp.MCPDirectorStatsToolResultV0{
			Estado: orquestamcp.MCPDirectorStatsEstadoOKV0,
			RunRef: "run-ref-autop-status-web-001",
			Stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:       " run-ref-autop-status-web-001 ",
				ProjectRef:   " app-ref-autop-status-web-001 ",
				Status:       "running",
				CurrentPhase: "run_required_tests",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksTotal:     4,
					TasksClosed:    2,
					AgentsInFlight: 1,
					AgentsFailed:   1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					PercentComplete:   50,
					ProgressingAgents: 1,
					StalledAgents:     1,
				},
				Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
					Status:  "blocked",
					Blocked: true,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: " agent-ref-autop-status-web-001 ",
					Status:         "running",
					InFlight:       true,
					NeedsAttention: true,
					CanStop:        true,
					LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
						Status:              "stalled",
						NoProgressTicks:     3,
						RepeatedActionCount: 2,
					},
				}},
			},
		},
		Diagnostics: []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
			Code:         "run_closure_blocked",
			Scope:        "run",
			Message:      "cierre bloqueado",
			EvidenceRefs: []string{"blocker-ref-autop-status-web-001"},
		}},
	})

	if vm.SchemaVersion != "web_autoprogramming_status.v0" ||
		vm.Locale != "es-ES" ||
		vm.Estado != WebAutoprogrammingPrepareRunEstadoOKV0 ||
		!vm.QueueLive ||
		!vm.RunLive ||
		vm.QueueRef != "queue-ref-autop-status-web-001" ||
		vm.RunRef != "run-ref-autop-status-web-001" {
		t.Fatalf("vm=%+v", vm)
	}
	if len(vm.Runs) != 2 ||
		vm.Runs[0].RunRef != "run-ref-autop-status-web-queued" ||
		vm.Runs[0].StatsHref != "/director-stats?include_agent_progress=true&run_ref=run-ref-autop-status-web-queued" ||
		vm.Runs[1].RunRef != "run-ref-autop-status-web-001" ||
		vm.Runs[1].AppRef != "app-ref-autop-status-web-001" ||
		vm.Runs[1].CurrentPhase != "run_required_tests" ||
		vm.Runs[1].PercentComplete != 50 ||
		vm.Runs[1].TasksTotal != 4 ||
		vm.Runs[1].TasksClosed != 2 ||
		vm.Runs[1].AgentsInFlight != 1 ||
		vm.Runs[1].AgentsFailed != 1 ||
		vm.Runs[1].ProgressingAgents != 1 ||
		vm.Runs[1].StalledAgents != 1 ||
		vm.Runs[1].ClosureStatus != "blocked" ||
		!vm.Runs[1].NeedsAttention {
		t.Fatalf("runs=%+v", vm.Runs)
	}
	if len(vm.Agents) != 1 ||
		vm.Agents[0].AgentRef != "agent-ref-autop-status-web-001" ||
		vm.Agents[0].ProgressStatus != "stalled" ||
		vm.Agents[0].NoProgressTicks != 3 ||
		vm.Agents[0].RepeatedActionCount != 2 ||
		!vm.Agents[0].CanStop ||
		!vm.Agents[0].NeedsAttention {
		t.Fatalf("agents=%+v", vm.Agents)
	}
	if len(vm.Diagnostics) != 1 ||
		vm.Diagnostics[0].Code != "run_closure_blocked" ||
		len(vm.Diagnostics[0].EvidenceRefs) != 1 {
		t.Fatalf("diagnostics=%+v", vm.Diagnostics)
	}

	raw, err := json.Marshal(vm)
	if err != nil {
		t.Fatalf("marshal vm: %v", err)
	}
	got := strings.ToLower(string(raw))
	for _, forbidden := range []string{
		"process_ref",
		"session_ref",
		"runtime_provider",
		"provider",
		"oauth",
		"home",
		"transcript",
		"dsn",
		"sql",
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("viewmodel expone detalle prohibido %q: %s", forbidden, got)
		}
	}
}

func TestWebAutoprogrammingStatusViewModelV0SinPuertosNoInventaEstadoLive(t *testing.T) {
	vm := NewWebAutoprogrammingStatusViewModelV0("en", orquestamcp.MCPAutoprogrammingStatusToolResultV0{
		Estado: orquestamcp.MCPAutoprogrammingStatusEstadoErrorV0,
		Diagnostics: []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
			Code:  "queue_unbound",
			Scope: "queue",
		}},
		Errores: []orquestamcp.MCPValidationIssueV0{{
			Code:  "autoprogramming_status_no_disponible",
			Field: "ports",
		}},
	})

	if vm.Locale != "en-US" ||
		vm.Estado != WebAutoprogrammingPrepareRunEstadoErrorV0 ||
		vm.QueueLive ||
		vm.RunLive ||
		len(vm.Runs) != 0 ||
		len(vm.Agents) != 0 ||
		len(vm.Diagnostics) != 1 ||
		len(vm.ErroresPublicos) != 1 ||
		vm.ErroresPublicos[0].Code != "autoprogramming_status_no_disponible" {
		t.Fatalf("vm=%+v", vm)
	}
}
