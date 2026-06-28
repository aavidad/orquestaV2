package orquestacli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestAutoprogrammingCliClientV0PrepararRunPreservaWorktreeYRamaOpaca(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != AutoprogrammingPrepareRunCliEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		var input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decode: %v", err)
		}
		req := input.AutoprogrammingRequest
		if !req.WorktreeIsolated || req.WorktreeRef != "worktree-ref-opaque-001" ||
			req.BranchRef != "branch-ref-opaque-001" {
			t.Fatalf("scope no preservado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:           orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0,
			Accepted:         true,
			RunRef:           "run-ref-autoprog-001",
			ProjectRef:       req.ProjectRef,
			WorktreeRef:      req.WorktreeRef,
			BranchRef:        req.BranchRef,
			WorkflowTaskRefs: []string{"workflow-task-ref-001"},
			WaitAgentRefs:    []string{"agent-ref-001"},
		})
	}))
	defer server.Close()

	client := mustNewAutoprogrammingCliClientV0(t, server.URL)
	env := client.PrepararRun(context.Background(), autoprogInvocationV0(server.URL), orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		AutoprogrammingRequest: orquestaautoprogramming.AutoprogrammingRequestV0{
			RequestRef:       "request-ref-autoprog-001",
			ProjectRef:       "project-ref-orquesta",
			WorktreeRef:      "worktree-ref-opaque-001",
			WorktreeIsolated: true,
			BranchRef:        "branch-ref-opaque-001",
			Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{{
				TaskRef: "task-ref-001",
				Area:    "cli",
			}},
			WriteSet:      []string{"modulos/orquesta-cli"},
			RequiredTests: []string{"go test -count=1 ./modulos/orquesta-cli"},
		},
	})

	if !env.OK || env.Contract != CliContractAutoprogrammingV0 {
		t.Fatalf("env=%+v", env)
	}
}

func TestAutoprogrammingCliClientV0ConsultaColaYRunPorAPI(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path] = true
		switch r.URL.Path {
		case AutoprogrammingRunQueueCliEndpointV0:
			var input orquestamcp.MCPRunQueuePriorityToolInputV0
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatalf("decode queue: %v", err)
			}
			if input.Action != orquestamcp.MCPRunQueuePriorityActionRankV0 || input.Limit != 3 {
				t.Fatalf("queue input=%+v", input)
			}
			_ = json.NewEncoder(w).Encode(orquestamcp.MCPRunQueuePriorityToolResultV0{
				Estado: orquestamcp.MCPRunQueuePriorityEstadoOKV0,
				Count:  1,
				Ranked: []orquestamcp.MCPRunQueueRankedCandidateCompactV0{{RunRef: "run-ref-queue-001"}},
			})
		case AutoprogrammingRunStatsCliEndpointV0:
			var input orquestamcp.MCPDirectorStatsToolInputV0
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatalf("decode stats: %v", err)
			}
			if input.RunRef != "run-ref-queue-001" || !input.IncludeProcessRefs {
				t.Fatalf("stats input=%+v", input)
			}
			_ = json.NewEncoder(w).Encode(orquestamcp.MCPDirectorStatsToolResultV0{
				Estado: orquestamcp.MCPDirectorStatsEstadoOKV0,
				RunRef: input.RunRef,
				Stats:  &orquestacionnucleoapp.DirectorRunStatsV0{RunRef: input.RunRef},
			})
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := mustNewAutoprogrammingCliClientV0(t, server.URL)
	queueEnv := client.ConsultarCola(context.Background(), autoprogInvocationV0(server.URL), orquestamcp.MCPRunQueuePriorityToolInputV0{Limit: 3})
	runEnv := client.ConsultarRun(context.Background(), autoprogInvocationV0(server.URL), orquestamcp.MCPDirectorStatsToolInputV0{
		RunRef:             "run-ref-queue-001",
		IncludeProcessRefs: true,
	})

	if !queueEnv.OK || !runEnv.OK || !seen[AutoprogrammingRunQueueCliEndpointV0] || !seen[AutoprogrammingRunStatsCliEndpointV0] {
		t.Fatalf("queue=%+v run=%+v seen=%+v", queueEnv, runEnv, seen)
	}
}

func TestAutoprogrammingCliClientV0ConsultaEstadoObserveGoalYSupervisaPorAPI(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path] = true
		switch r.URL.Path {
		case AutoprogrammingStatusCliEndpointV0:
			var input orquestamcp.MCPAutoprogrammingStatusToolInputV0
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatalf("decode status: %v", err)
			}
			if input.RunRef != "run-ref-status-001" || !input.IncludeAgentProgress {
				t.Fatalf("status input=%+v", input)
			}
			_ = json.NewEncoder(w).Encode(orquestamcp.MCPAutoprogrammingStatusToolResultV0{
				Estado: orquestamcp.MCPAutoprogrammingStatusEstadoOKV0,
				RunRef: input.RunRef,
				Operator: &orquestamcp.MCPAutoprogrammingOperatorV0{
					QueueLive: true,
					SafeActions: []orquestamcp.MCPAutoprogrammingSafeActionV0{{
						Action: "supervise",
						RunRef: input.RunRef,
					}},
				},
			})
		case AutoprogrammingObserveGoalCliEndpointV0:
			var input orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatalf("decode observe goal: %v", err)
			}
			if input.RunRef != "run-ref-status-001" || input.RequestedBy != "operator" {
				t.Fatalf("observe goal input=%+v", input)
			}
			_ = json.NewEncoder(w).Encode(orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{
				Estado:     orquestamcp.MCPAutoprogrammingObserveGoalEstadoOKV0,
				RunRef:     input.RunRef,
				GoalRef:    "goal-ref-cli-observe-001",
				GoalStatus: "complete",
			})
		case AutoprogrammingSuperviseCliEndpointV0:
			var input orquestamcp.MCPRunSupervisorToolInputV0
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatalf("decode supervise: %v", err)
			}
			if input.RunRef != "run-ref-status-001" ||
				input.DirectorExecutionMode != "legacy_director_loop" ||
				input.MaxTicks != 1 {
				t.Fatalf("supervise input=%+v", input)
			}
			_ = json.NewEncoder(w).Encode(orquestamcp.MCPRunSupervisorToolResultV0{
				Estado: orquestamcp.MCPRunSupervisorEstadoOKV0,
				RunRef: input.RunRef,
				Ticks:  1,
			})
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := mustNewAutoprogrammingCliClientV0(t, server.URL)
	statusEnv := client.ConsultarEstado(context.Background(), autoprogInvocationV0(server.URL), orquestamcp.MCPAutoprogrammingStatusToolInputV0{
		RunRef:               "run-ref-status-001",
		IncludeAgentProgress: true,
	})
	observeGoalEnv := client.ObservarGoal(context.Background(), autoprogInvocationV0(server.URL), orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{
		RunRef:      "run-ref-status-001",
		RequestedBy: "operator",
	})
	superviseEnv := client.Supervisar(context.Background(), autoprogInvocationV0(server.URL), orquestamcp.MCPRunSupervisorToolInputV0{
		DirectorExecutionMode: "legacy_director_loop",
		RunRef:                "run-ref-status-001",
		MaxTicks:              1,
	})

	if !statusEnv.OK || !observeGoalEnv.OK || !superviseEnv.OK ||
		!seen[AutoprogrammingStatusCliEndpointV0] ||
		!seen[AutoprogrammingObserveGoalCliEndpointV0] ||
		!seen[AutoprogrammingSuperviseCliEndpointV0] {
		t.Fatalf("status=%+v observe_goal=%+v supervise=%+v seen=%+v", statusEnv, observeGoalEnv, superviseEnv, seen)
	}
}

func mustNewAutoprogrammingCliClientV0(t *testing.T, serverURL string) *AutoprogrammingCliClientV0 {
	t.Helper()
	client, err := NewAutoprogrammingCliClientV0(serverURL, time.Second)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return client
}

func autoprogInvocationV0(serverURL string) CliInvocationContextV0 {
	return CliInvocationContextV0{
		RequestID:     "req-autoprog-cli-001",
		CorrelationID: "corr-autoprog-cli-001",
		ServerURL:     serverURL,
		Timeout:       time.Second,
	}
}
