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
		QueueHealth: &orquestamcp.MCPAutoprogrammingQueueHealthV0{
			Queued:       1,
			RunningLive:  1,
			RunningStale: 2,
			Blocked:      3,
			Lost:         4,
			Completed:    5,
			Failed:       6,
			ObservedRuns: 7,
			QueueRuns:    8,
			StatsRuns:    1,
		},
		StaleRunning: []orquestamcp.MCPAutoprogrammingActionableRunV0{{
			Code:              "provider_usage_limit_retry_after",
			Severity:          "blocked",
			RunRef:            " run-ref-autop-status-web-stale ",
			AppRef:            " opes ",
			Status:            " stopped ",
			Reason:            " provider_usage_limit_retry_after ",
			RecommendedAction: " wait_for_quota_and_relaunch_idempotently ",
			EvidenceRefs:      []string{" evidence-ref-provider-usage-limit-retry-after ", "evidence-ref-provider-usage-limit-retry-after"},
		}},
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
					Status:      "blocked",
					Blocked:     true,
					BlockedBy:   []string{" closure-required-tests ", "closure-required-tests"},
					BlockerRefs: []string{" blocker-ref-required-tests-001 ", "blocker-ref-required-tests-001"},
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
						EvidenceRefs:        []string{" evidence-ref-progress-web-001 ", "evidence-ref-progress-web-001"},
					},
				}},
			},
		},
		Diagnostics: []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
			Code:         "run_closure_blocked",
			Scope:        "run",
			Message:      "cierre bloqueado",
			EvidenceRefs: []string{" blocker-ref-autop-status-web-001 ", "blocker-ref-autop-status-web-001"},
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
	if vm.QueueHealth == nil ||
		vm.QueueHealth.Queued != 1 ||
		vm.QueueHealth.RunningLive != 1 ||
		vm.QueueHealth.RunningStale != 2 ||
		vm.QueueHealth.Blocked != 3 ||
		vm.QueueHealth.Lost != 4 ||
		vm.QueueHealth.Completed != 5 ||
		vm.QueueHealth.Failed != 6 ||
		vm.QueueHealth.ObservedRuns != 7 ||
		vm.QueueHealth.QueueRuns != 8 ||
		vm.QueueHealth.StatsRuns != 1 {
		t.Fatalf("queue_health=%+v", vm.QueueHealth)
	}
	if len(vm.StaleRunning) != 1 ||
		vm.StaleRunning[0].Code != "usage_limit_retry_after" ||
		vm.StaleRunning[0].Severity != "blocked" ||
		vm.StaleRunning[0].RunRef != "run-ref-autop-status-web-stale" ||
		vm.StaleRunning[0].RecommendedAction != "wait_for_quota_and_relaunch_idempotently" ||
		len(vm.StaleRunning[0].EvidenceRefs) != 1 ||
		vm.StaleRunning[0].EvidenceRefs[0] != "evidence-ref-usage-limit-retry-after" {
		t.Fatalf("stale_running=%+v", vm.StaleRunning)
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
		len(vm.Runs[1].ClosureBlockedBy) != 1 ||
		vm.Runs[1].ClosureBlockedBy[0] != "closure-required-tests" ||
		len(vm.Runs[1].ClosureBlockerRefs) != 1 ||
		vm.Runs[1].ClosureBlockerRefs[0] != "blocker-ref-required-tests-001" ||
		!vm.Runs[1].NeedsAttention {
		t.Fatalf("runs=%+v", vm.Runs)
	}
	if len(vm.Agents) != 1 ||
		vm.Agents[0].AgentRef != "agent-ref-autop-status-web-001" ||
		vm.Agents[0].ProgressStatus != "stalled" ||
		vm.Agents[0].NoProgressTicks != 3 ||
		vm.Agents[0].RepeatedActionCount != 2 ||
		len(vm.Agents[0].EvidenceRefs) != 1 ||
		vm.Agents[0].EvidenceRefs[0] != "evidence-ref-progress-web-001" ||
		!vm.Agents[0].CanStop ||
		!vm.Agents[0].NeedsAttention {
		t.Fatalf("agents=%+v", vm.Agents)
	}
	if len(vm.Diagnostics) != 1 ||
		vm.Diagnostics[0].Code != "run_closure_blocked" ||
		len(vm.Diagnostics[0].EvidenceRefs) != 1 ||
		vm.Diagnostics[0].EvidenceRefs[0] != "blocker-ref-autop-status-web-001" {
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
