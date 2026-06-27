package orquestaweb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestRESTArrancarDirectorAppClientV0EnviaPOSTJSONYProyectaDirector(t *testing.T) {
	var received arrancarDirectorAppRequestEnvelopeV0
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != ArrancarDirectorAppEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get(ArrancarDirectorAppCorrelationV0); got != "req-director-client-001" {
			t.Fatalf("correlation=%q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeDirectorSuccessV0(t, w)
	}))
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	client.Limits = WebArrancarDirectorAppLimitsV0{
		MaxBursts:            4,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		MaxCommands:          9,
		MaxOutboxPerCycle:    5,
		MaxExternalWaits:     1,
	}

	form := minimalFormForClientV0("req-director-client-001")
	form.ProjectSource = WebNuevaAppProjectSourceFormV0{
		Kind:       "github",
		GitURL:     "https://example.test/director.git",
		ProjectRef: "project-ref-director-001",
	}
	vm, err := client.ArrancarDirectorApp(context.Background(), form)
	if err != nil {
		t.Fatalf("ArrancarDirectorApp: %v", err)
	}
	if received.AppSpecRequest.RequestID != "req-director-client-001" ||
		received.AppSpecRequest.Source != WebNuevaAppSourceV0 ||
		received.MaxBursts != 4 ||
		received.MaxStepsPerBurst != 3 ||
		received.MaxDispatchesPerWait != 2 ||
		received.MaxCommands != 9 ||
		received.MaxOutboxPerCycle != 5 ||
		received.MaxExternalWaits != 1 {
		t.Fatalf("payload inesperado: %+v", received)
	}
	if received.AppSpecRequest.ProjectSource.Kind != "github" ||
		received.AppSpecRequest.ProjectSource.GitURL != "https://example.test/director.git" ||
		received.AppSpecRequest.ProjectSource.ProjectRef != "project-ref-director-001" {
		t.Fatalf("project_source director inesperado: %+v", received.AppSpecRequest.ProjectSource)
	}
	if vm.Estado != WebNuevaAppEstadoDirector ||
		vm.Director == nil ||
		vm.Director.RunRef != "run-ref-director-client-001" ||
		len(vm.Director.DirectorTasks) != 1 ||
		vm.Director.DirectorTasks[0].Capacity != "high" {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestRESTArrancarDirectorAppClientV0OmiteModoDirectorVacioYPreservaLegacyExplicitoV0(t *testing.T) {
	client := NewRESTArrancarDirectorAppClientV0("http://example.test", time.Second)
	normalForm := minimalFormForClientV0("req-director-normal-mode")
	normalRequest := normalForm.ToAppSpecRequestV0()

	normalPayload, err := json.Marshal(client.payloadV0(normalForm, normalRequest))
	if err != nil {
		t.Fatalf("marshal normal: %v", err)
	}
	if strings.Contains(string(normalPayload), "director_execution_mode") {
		t.Fatalf("payload normal no debe serializar director_execution_mode: %s", normalPayload)
	}

	legacyForm := minimalFormForClientV0("req-director-legacy-mode")
	legacyForm.DirectorExecutionMode = "legacy_director_loop"
	legacyRequest := legacyForm.ToAppSpecRequestV0()
	legacyPayload, err := json.Marshal(client.payloadV0(legacyForm, legacyRequest))
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	if !strings.Contains(string(legacyPayload), `"director_execution_mode":"legacy_director_loop"`) {
		t.Fatalf("payload legacy no conserva modo explicito: %s", legacyPayload)
	}
}

func TestRESTPreviewDirectorAppClientV0EnviaPOSTJSONYProyectaGoalPreview(t *testing.T) {
	var received arrancarDirectorAppRequestEnvelopeV0
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != PreviewDirectorAppEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(WebPreviewDirectorAppResultV0{
			Estado:                ArrancarDirectorAppEstadoOKV0,
			RequestID:             "req-director-preview-client-001",
			RunRef:                "run-ref-director-preview-client-001",
			DirectorExecutionMode: "goal_first",
			GoalSpec: orquestagoal.GoalWorkSpecV0{
				GoalRef:      "goal-ref-director-preview-client-001",
				RunRef:       "run-ref-director-preview-client-001",
				WorkKind:     "new_app",
				DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
				Objective:    "Construir Agenda",
				WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "generated-apps/agenda"}},
				RequiredTests: []orquestagoal.GoalRequiredTestV0{{
					TestRef: "test-ref-agenda",
					Command: "verificar app",
				}},
			},
			Estimate: WebNuevaAppGoalPreviewEstimateV0{
				TokenBudget: 12000,
				CostTier:    "low",
			},
			EvidenceRefs: []string{"evidence-ref-web-preview-client-001"},
		})
	}))
	defer server.Close()

	client := NewRESTPreviewDirectorAppClientV0(server.URL, time.Second)
	client.Limits = WebArrancarDirectorAppLimitsV0{MaxCommands: 9}
	vm, err := client.PreviewDirectorApp(context.Background(), minimalFormForClientV0("req-director-preview-client-001"))
	if err != nil {
		t.Fatalf("PreviewDirectorApp: %v", err)
	}
	if received.AppSpecRequest.RequestID != "req-director-preview-client-001" || received.MaxCommands != 9 {
		t.Fatalf("payload inesperado: %+v", received)
	}
	if vm.Estado != WebNuevaAppEstadoGoalPreview ||
		vm.GoalPreview == nil ||
		vm.GoalPreview.GoalRef != "goal-ref-director-preview-client-001" ||
		len(vm.GoalPreview.WriteSet) != 1 ||
		vm.GoalPreview.Estimate.CostTier != "low" {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestRESTArrancarDirectorAppClientV0ErroresPublicosNoSonTransporte(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(WebArrancarDirectorAppResultV0{
			Estado:    ArrancarDirectorAppEstadoErrorV0,
			RequestID: "req-director-invalid",
			Errores: []WebNuevaAppIssueV0{{
				Code:    "app_spec_invalida",
				Field:   "nombre",
				Message: "nombre requerido",
			}},
		})
	}))
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	vm, err := client.ArrancarDirectorApp(
		context.Background(),
		minimalFormForClientV0("req-director-invalid"),
	)
	if err != nil {
		t.Fatalf("error publico no debe ser transporte: %v", err)
	}
	if vm.Estado != WebNuevaAppEstadoInvalida ||
		len(vm.ErroresPublicos) != 1 ||
		vm.ErroresPublicos[0].Code != "app_spec_invalida" {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestRESTArrancarDirectorAppClientV0RechazaOKSinRunRef(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(WebArrancarDirectorAppResultV0{
			Estado:                ArrancarDirectorAppEstadoOKV0,
			RequestID:             "req-director-goal-001",
			DirectorExecutionMode: "goal_first",
			GoalRef:               "goal-ref-web-director-001",
			ExternalGoalRef:       "thread-ref-web-director-001",
			GoalStatus:            "running",
		}); err != nil {
			t.Fatalf("encode success: %v", err)
		}
	}))
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	_, err := client.ArrancarDirectorApp(context.Background(), minimalFormForClientV0("req-director-goal-001"))
	var clientErr WebNuevaAppClientErrorV0
	if !errors.As(err, &clientErr) ||
		clientErr.Code != WebNuevaAppErrRespuestaInvalidaV0 ||
		clientErr.StatusCode != http.StatusOK {
		t.Fatalf("error inesperado: %#v", err)
	}
}

func TestRESTArrancarDirectorAppClientV0AceptaContextNil(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeDirectorSuccessV0(t, w)
	}))
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	vm, err := client.ArrancarDirectorApp(context.TODO(), minimalFormForClientV0("req-director-context-nil"))
	if err != nil {
		t.Fatalf("ArrancarDirectorApp: %v", err)
	}
	if vm.Estado != WebNuevaAppEstadoDirector {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestRESTArrancarDirectorAppClientV0Status500DevuelveErrorPublico(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "stack privado", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	_, err := client.ArrancarDirectorApp(
		context.Background(),
		minimalFormForClientV0("req-director-500"),
	)
	var clientErr WebNuevaAppClientErrorV0
	if !errors.As(err, &clientErr) ||
		clientErr.Code != WebNuevaAppErrTransporteV0 ||
		clientErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("error inesperado: %#v", err)
	}
}

func TestRESTArrancarDirectorAppClientV0Status500JSONConservaErrorPublicoGoal(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(WebArrancarDirectorAppResultV0{
			Estado:    ArrancarDirectorAppEstadoErrorV0,
			RequestID: "req-director-goal-degraded",
			Errores: []WebNuevaAppIssueV0{{
				Code:    "codex_app_server_control_socket_missing",
				Field:   "goal_backend",
				Message: "codex_app_server_control_socket_missing",
			}},
		})
	}))
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	vm, err := client.ArrancarDirectorApp(
		context.Background(),
		minimalFormForClientV0("req-director-goal-degraded"),
	)
	if err != nil {
		t.Fatalf("error publico goal degradado no debe ser transporte: %v", err)
	}
	if vm.Estado != WebNuevaAppEstadoInvalida ||
		len(vm.ErroresPublicos) != 1 ||
		vm.ErroresPublicos[0].Code != "codex_app_server_control_socket_missing" {
		t.Fatalf("vm=%+v", vm)
	}
}

func writeDirectorSuccessV0(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(WebArrancarDirectorAppResultV0{
		Estado:     ArrancarDirectorAppEstadoOKV0,
		RequestID:  "req-director-client-001",
		RunRef:     "run-ref-director-client-001",
		PhaseID:    "brainstorming_arquitectura",
		LoopStatus: "quiescent",
		DirectorTasks: []WebNuevaAppDirectorTaskV0{{
			TaskRef:        "task-ref-director-client-001",
			AgentRequestID: "agent-ref-director-client-001",
			Capacity:       "high",
		}},
		StartedAgents: []string{"agent-ref-director-client-001"},
		EvidenceRefs:  []string{"evidence-ref-director-client-001"},
	}); err != nil {
		t.Fatalf("encode success: %v", err)
	}
}
