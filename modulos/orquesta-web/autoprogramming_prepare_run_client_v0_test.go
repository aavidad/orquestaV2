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
		len(vm.GoalSpecs) != 1 ||
		vm.GoalSpecs[0].RunRef != "run-autoprog-web-001" ||
		vm.Goal == nil ||
		vm.Goal.RunRef != "run-autoprog-web-001" ||
		len(vm.Goals) != 1 ||
		vm.Goals[0].RunRef != "run-autoprog-web-001" ||
		vm.GoalSpecs[0].DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
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
