package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestRESTAutoprogrammingPrepareRunClientV0PreservaWorktreeAisladaYRamaOpaca(t *testing.T) {
	var got orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != WebAutoprogrammingPrepareRunInboundEndpointV0 {
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get(AppChangeCorrelationV0) != "corr-autoprog-web-001" {
			t.Fatalf("correlation header=%q", r.Header.Get(AppChangeCorrelationV0))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:           orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0,
			Accepted:         true,
			RunRef:           "run-autoprog-web-001",
			ProjectRef:       "project-ref-orquesta",
			WorktreeRef:      "worktree-ref-opaque-001",
			BranchRef:        "branch-ref-opaque-001",
			PhaseID:          "programacion",
			WorkflowTaskRefs: []string{"workflow-task-ref-001"},
			WaitAgentRefs:    []string{"agent-ref-001"},
			GoalSpecs: []orquestagoal.GoalWorkSpecV0{{
				SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
				GoalRef:       "goal-ref-web-autoprog-001",
				RequestRef:    "source-task-ref-001",
				RunRef:        "run-autoprog-web-001",
				ProjectRef:    "project-ref-orquesta",
				WorkKind:      "autoprogramming",
				Objective:     "Preservar contrato Goal desde web.",
				DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
				WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-web"}},
			}},
			Goals: []orquestamcp.MCPAutoprogrammingGoalRunV0{{
				DirectorExecutionMode: "goal_first",
				RunRef:                "run-autoprog-web-001",
				GoalRef:               "goal-ref-web-autoprog-001",
				GoalStatus:            orquestagoal.GoalStatusRunningV0,
			}},
			Continue: &orquestamcp.MCPAutoprogrammingContinueRequestV0{
				RunRef:                     "run-autoprog-web-001",
				OperationalDirectorPlanRef: "operational-director-plan-web-001",
				CorrelationID:              "corr-autoprog-web-001",
				WaitAgentRefs:              []string{"agent-ref-001"},
				MaxBursts:                  2,
			},
		})
	}))
	defer server.Close()

	client := NewRESTAutoprogrammingPrepareRunClientV0(server.URL, time.Second)
	vm, err := client.PrepararAutoprogrammingRun(context.Background(), validWebAutoprogrammingPrepareRunCommandV0())
	if err != nil {
		t.Fatalf("PrepararAutoprogrammingRun error: %v", err)
	}
	if got.RequestID != "request-ref-autoprog-web-001" ||
		got.CorrelationID != "corr-autoprog-web-001" ||
		got.AutoprogrammingRequest.RequestRef != "source-task-ref-001" {
		t.Fatalf("input refs=%+v", got)
	}
	request := got.AutoprogrammingRequest
	if !request.WorktreeIsolated ||
		request.WorktreeRef != "worktree-ref-opaque-001" ||
		request.BranchRef != "branch-ref-opaque-001" {
		t.Fatalf("worktree/branch no preservados: %+v", request)
	}
	if len(request.Tasks) != 1 ||
		request.Tasks[0].TaskRef != "task-ref-web-autoprog-001" ||
		request.Tasks[0].Area != "modulos/orquesta-web" ||
		request.Tasks[0].Objective != "Preservar contrato explicito web." ||
		len(request.Tasks[0].AcceptanceCriteria) != 1 ||
		request.Tasks[0].AcceptanceCriteria[0] != "criterio web visible" {
		t.Fatalf("tasks=%+v", request.Tasks)
	}
	if len(request.WriteSet) != 1 || request.WriteSet[0] != "modulos/orquesta-web" {
		t.Fatalf("write_set=%+v", request.WriteSet)
	}
	if len(request.RequiredTests) != 1 || request.RequiredTests[0] != "go test -count=1 ./modulos/orquesta-web" {
		t.Fatalf("required_tests=%+v", request.RequiredTests)
	}
	if vm.Estado != WebAutoprogrammingPrepareRunEstadoOKV0 ||
		!vm.Accepted ||
		vm.WorktreeRef != "worktree-ref-opaque-001" ||
		vm.BranchRef != "branch-ref-opaque-001" ||
		len(vm.GoalSpecSummaries) != 1 ||
		vm.GoalSpecSummaries[0].RunRef != "run-autoprog-web-001" ||
		vm.Goal == nil ||
		vm.Goal.RunRef != "run-autoprog-web-001" ||
		len(vm.Goals) != 1 ||
		vm.Goals[0].RunRef != "run-autoprog-web-001" ||
		vm.GoalSpecSummaries[0].DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		vm.GoalSpecSummaries[0].SpecHash == "" ||
		vm.Continue == nil ||
		vm.Continue.OperationalDirectorPlanRef != "operational-director-plan-web-001" ||
		len(vm.Continue.WaitAgentRefs) != 1 {
		t.Fatalf("viewmodel=%+v", vm)
	}
}

func TestRESTAutoprogrammingPrepareRunClientV0ConservaGoalSingularLegacyComoGoals(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:   orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0,
			Accepted: true,
			RunRef:   "run-autoprog-web-legacy-goal-001",
			Goal: &orquestamcp.MCPAutoprogrammingGoalRunV0{
				DirectorExecutionMode: "goal_first",
				RunRef:                "run-autoprog-web-legacy-goal-001",
				GoalRef:               "goal-ref-web-legacy-001",
				GoalStatus:            orquestagoal.GoalStatusRunningV0,
			},
		})
	}))
	defer server.Close()

	client := NewRESTAutoprogrammingPrepareRunClientV0(server.URL, time.Second)
	vm, err := client.PrepararAutoprogrammingRun(context.Background(), validWebAutoprogrammingPrepareRunCommandV0())
	if err != nil {
		t.Fatalf("PrepararAutoprogrammingRun error: %v", err)
	}
	if vm.Goal == nil ||
		vm.Goal.RunRef != "run-autoprog-web-legacy-goal-001" ||
		len(vm.Goals) != 1 ||
		vm.Goals[0].GoalRef != "goal-ref-web-legacy-001" {
		t.Fatalf("viewmodel=%+v", vm)
	}
}

func TestRESTAutoprogrammingPrepareRunClientV0RenderizaErrorPublicoHTTP(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:   orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0,
			Accepted: false,
			Errores: []orquestamcp.MCPValidationIssueV0{{
				Code:    "autoprogramming_prepare_run_no_configurado",
				Field:   "executor",
				Message: "autoprogramming_prepare_run_no_configurado",
			}},
		})
	}))
	defer server.Close()

	client := NewRESTAutoprogrammingPrepareRunClientV0(server.URL, time.Second)
	vm, err := client.PrepararAutoprogrammingRun(context.Background(), validWebAutoprogrammingPrepareRunCommandV0())
	if err != nil {
		t.Fatalf("error no esperado: %v", err)
	}
	if vm.Estado != WebAutoprogrammingPrepareRunEstadoErrorV0 ||
		len(vm.ErroresPublicos) != 1 ||
		vm.ErroresPublicos[0].Code != "autoprogramming_prepare_run_no_configurado" {
		t.Fatalf("viewmodel=%+v", vm)
	}
}

func TestRESTAutoprogrammingPrepareRunClientV0RespuestaInvalida(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"estado": ""})
	}))
	defer server.Close()

	client := NewRESTAutoprogrammingPrepareRunClientV0(server.URL, time.Second)
	_, err := client.PrepararAutoprogrammingRun(context.Background(), validWebAutoprogrammingPrepareRunCommandV0())
	if !IsWebAutoprogrammingPrepareRunClientErrorCodeV0(
		err,
		WebAutoprogrammingPrepareRunErrRespuestaInvalidaV0,
	) {
		t.Fatalf("error=%v", err)
	}
}

func TestRESTAutoprogrammingPrepareRunClientV0ConsultaStatusNormalizaYDecodifica(t *testing.T) {
	var got orquestamcp.MCPAutoprogrammingStatusToolInputV0
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != WebAutoprogrammingStatusInboundEndpointV0 {
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPAutoprogrammingStatusToolResultV0{
			Estado:   orquestamcp.MCPAutoprogrammingStatusEstadoOKV0,
			RunRef:   "run-autoprog-status-001",
			QueueRef: "queue-autoprog-status-001",
		})
	}))
	defer server.Close()

	client := NewRESTAutoprogrammingPrepareRunClientV0(server.URL, time.Second)
	viewModel, err := client.ConsultarAutoprogrammingStatus(context.Background(), WebAutoprogrammingStatusQueryV0{
		RequestID: " request-autoprog-status-001 ",
		RunRef:    " run-autoprog-status-001 ",
	})
	if err != nil {
		t.Fatalf("ConsultarAutoprogrammingStatus error: %v", err)
	}
	if got.RequestID != "request-autoprog-status-001" ||
		got.CorrelationID != "request-autoprog-status-001" ||
		got.RunRef != "run-autoprog-status-001" ||
		!bool(got.IncludeAgentProgress) {
		t.Fatalf("status input no normalizado: %+v", got)
	}
	if viewModel.Estado != WebAutoprogrammingPrepareRunEstadoOKV0 ||
		viewModel.RunRef != "run-autoprog-status-001" ||
		viewModel.QueueRef != "queue-autoprog-status-001" {
		t.Fatalf("status viewmodel no decodificado: %+v", viewModel)
	}
}

func TestRESTAutoprogrammingPrepareRunClientV0ConsultaStatusDefaultsToAppScope(t *testing.T) {
	var got orquestamcp.MCPAutoprogrammingStatusToolInputV0
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPAutoprogrammingStatusToolResultV0{Estado: orquestamcp.MCPAutoprogrammingStatusEstadoOKV0, ScopeMode: orquestamcp.MCPAutoprogrammingStatusScopeAppV0, Scope: "app-opaque-001"})
	}))
	defer server.Close()
	_, err := NewRESTAutoprogrammingPrepareRunClientV0(server.URL, time.Second).ConsultarAutoprogrammingStatus(context.Background(), WebAutoprogrammingStatusQueryV0{AppRef: " app-opaque-001 "})
	if err != nil || got.ScopeMode != orquestamcp.MCPAutoprogrammingStatusScopeAppV0 || got.Scope != "app-opaque-001" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestRESTAutoprogrammingStatusE2EV0FiltersOpaqueRunAcrossMCPHTTPAndWeb(t *testing.T) {
	allowed := "request-ref-status-opaque-allowed-001"
	foreign := "request-ref-status-opaque-allowed-001-shadow"
	server := newWebHTTPTestServerV0(t, orquestamcp.NewMCPAutoprogrammingStatusHTTPHandlerV0(
		orquestamcp.MCPAutoprogrammingStatusToolExecutorV0{
			Queue: webStatusScopeQueueV0{ranked: []orquestamcp.MCPRunQueueRankedCandidateCompactV0{{RunRef: allowed, Status: "running"}, {RunRef: foreign, Status: "running"}}},
			Stats: webStatusScopeStatsV0{},
		},
	))
	defer server.Close()

	vm, err := NewRESTAutoprogrammingPrepareRunClientV0(server.URL, time.Second).ConsultarAutoprogrammingStatus(context.Background(), WebAutoprogrammingStatusQueryV0{RunRef: allowed})
	if err != nil {
		t.Fatalf("status e2e: %v", err)
	}
	if vm.ScopeMode != orquestamcp.MCPAutoprogrammingStatusScopeRunV0 || vm.Scope != allowed || len(vm.Runs) != 2 || vm.Runs[0].RunRef != allowed || vm.Runs[1].RunRef != allowed || len(vm.Agents) != 1 {
		t.Fatalf("viewmodel debe ocultar run opaca ajena: %+v", vm)
	}
}

type webStatusScopeQueueV0 struct {
	ranked []orquestamcp.MCPRunQueueRankedCandidateCompactV0
}

func (fake webStatusScopeQueueV0) Execute(_ context.Context, input orquestamcp.MCPRunQueuePriorityToolInputV0) (orquestamcp.MCPRunQueuePriorityToolResultV0, error) {
	return orquestamcp.MCPRunQueuePriorityToolResultV0{Estado: orquestamcp.MCPRunQueuePriorityEstadoOKV0, QueueRef: input.QueueRef, Ranked: fake.ranked}, nil
}

type webStatusScopeStatsV0 struct{}

func (webStatusScopeStatsV0) Execute(_ context.Context, input orquestamcp.MCPDirectorStatsToolInputV0) (orquestamcp.MCPDirectorStatsToolResultV0, error) {
	return orquestamcp.MCPDirectorStatsToolResultV0{Estado: orquestamcp.MCPDirectorStatsEstadoOKV0, RunRef: input.RunRef, Stats: &orquestacionnucleoapp.DirectorRunStatsV0{RunRef: input.RunRef, ProjectRef: "app-ref-status", Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{AgentRequestID: "agent-ref-status", InFlight: true}}}}, nil
}

func validWebAutoprogrammingPrepareRunCommandV0() WebAutoprogrammingPrepareRunCommandV0 {
	return WebAutoprogrammingPrepareRunCommandV0{
		RequestID:     "request-ref-autoprog-web-001",
		CorrelationID: "corr-autoprog-web-001",
		Locale:        "es-ES",
		RequestedBy:   "operator-ref-web",
		RequestRef:    "source-task-ref-001",
		ProjectRef:    "project-ref-orquesta",
		WorktreeRef:   "worktree-ref-opaque-001",
		BranchRef:     "branch-ref-opaque-001",
		Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:            "task-ref-web-autoprog-001",
			Area:               "modulos/orquesta-web",
			Objective:          "Preservar contrato explicito web.",
			Context:            []string{"contexto compacto web"},
			ContextRefs:        []string{"doc-ref:web-autoprog"},
			AcceptanceCriteria: []string{"criterio web visible"},
			RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-web -run TestRESTAutoprogramming"},
			CompactRules:       []string{"no convertir branch_ref en rama git"},
		}},
		WriteSet:             []string{"modulos/orquesta-web"},
		RequiredTests:        []string{"go test -count=1 ./modulos/orquesta-web"},
		MaxBursts:            2,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 1,
	}
}
