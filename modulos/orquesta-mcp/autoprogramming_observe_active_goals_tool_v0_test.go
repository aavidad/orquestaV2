package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestMCPAutoprogrammingObserveActiveGoalsToolExecutorV0ListaYObservaActivos(t *testing.T) {
	store := newMCPAutoprogrammingObserveActiveGoalsStoreForTestV0(t,
		"run-ref-observe-active-goals-001",
		"goal-ref-observe-active-goals-001",
		orquestagoal.GoalStatusRunningV0,
	)
	store.mustSaveV0(t,
		"run-ref-observe-active-goals-complete-001",
		"goal-ref-observe-active-goals-complete-001",
		orquestagoal.GoalStatusCompleteV0,
	)
	accepted, err := store.LoadGoalWorkStateV0(context.Background(), "run-ref-observe-active-goals-complete-001")
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 accepted base: %v", err)
	}
	accepted.RunRef = "run-ref-observe-active-goals-accepted-001"
	accepted.GoalRef = "goal-ref-observe-active-goals-accepted-001"
	accepted.Spec.RunRef = accepted.RunRef
	accepted.Spec.GoalRef = accepted.GoalRef
	accepted.LaunchReceipt.GoalRef = accepted.GoalRef
	accepted.Status = orquestagoal.GoalStatusCompleteV0
	accepted.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:   orquestagoal.GoalStatusAcceptedV0,
		Accepted: true,
	}
	if err := store.SaveGoalWorkStateV0(context.Background(), accepted); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 accepted: %v", err)
	}
	observer := &fakeMCPAutoprogrammingObserveActiveGoalsExecutorForTestV0{
		results: map[string]MCPAutoprogrammingObserveGoalToolResultV0{
			"run-ref-observe-active-goals-001": {
				Estado:       MCPAutoprogrammingObserveGoalEstadoOKV0,
				RunRef:       "run-ref-observe-active-goals-001",
				GoalRef:      "goal-ref-observe-active-goals-001",
				GoalStatus:   orquestagoal.GoalStatusRunningV0,
				EvidenceRefs: []string{"evidence-ref-observe-active-goals-001"},
			},
			"run-ref-observe-active-goals-complete-001": {
				Estado:       MCPAutoprogrammingObserveGoalEstadoOKV0,
				RunRef:       "run-ref-observe-active-goals-complete-001",
				GoalRef:      "goal-ref-observe-active-goals-complete-001",
				GoalStatus:   orquestagoal.GoalStatusCompleteV0,
				EvidenceRefs: []string{"evidence-ref-observe-active-goals-complete-001"},
			},
		},
	}

	result, err := NewMCPAutoprogrammingObserveActiveGoalsToolExecutorV0(store, observer).Execute(
		context.Background(),
		MCPAutoprogrammingObserveActiveGoalsToolInputV0{
			RequestID: "request-ref-observe-active-goals-001",
		},
	)

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingObserveActiveGoalsEstadoOKV0 ||
		len(result.Observations) != 2 ||
		len(result.Issues) != 0 ||
		len(observer.inputs) != 2 ||
		!hasInputRunMCPAutoprogrammingObserveActiveGoalsTestV0(observer.inputs, "run-ref-observe-active-goals-001") ||
		!hasInputRunMCPAutoprogrammingObserveActiveGoalsTestV0(observer.inputs, "run-ref-observe-active-goals-complete-001") ||
		hasInputRunMCPAutoprogrammingObserveActiveGoalsTestV0(observer.inputs, "run-ref-observe-active-goals-accepted-001") ||
		!hasStringMCPAutoprogrammingObserveActiveGoalsTestV0(result.EvidenceRefs, "evidence-ref-observe-active-goals-001") ||
		!hasStringMCPAutoprogrammingObserveActiveGoalsTestV0(result.EvidenceRefs, "evidence-ref-observe-active-goals-complete-001") {
		t.Fatalf("result=%+v inputs=%+v", result, observer.inputs)
	}
}

func TestMCPAutoprogrammingObserveActiveGoalsDescriptorV0DeclaraObservacionesAccionables(t *testing.T) {
	descriptor := MCPAutoprogrammingObserveActiveGoalsDescriptorV0()
	if descriptor.Name != MCPAutoprogrammingObserveActiveGoalsToolNameV0 ||
		descriptor.ResourceURI != MCPAutoprogrammingObserveActiveGoalsResourceURIV0 ||
		descriptor.InputSchema == "" ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	if !strings.Contains(descriptor.Output, "evidence_refs?") ||
		!strings.Contains(descriptor.Output, "observations?[]{run_ref,goal_ref?,goal_status?,recommended_action?,evidence_refs?}") ||
		!strings.Contains(descriptor.Output, "error:{errores_publicos,operation_ref?") ||
		!strings.Contains(descriptor.Output, "operation_ref?") ||
		!strings.Contains(descriptor.Output, "next_actions?") ||
		!strings.Contains(descriptor.Output, "diagnostics?") {
		t.Fatalf("observe_active_goals debe declarar observaciones accionables y evidencia: %+v", descriptor)
	}
}

func TestMCPAutoprogrammingObserveActiveGoalsToolExecutorV0PublicaSnapshotUsageLimited(t *testing.T) {
	store := newMCPAutoprogrammingObserveActiveGoalsStoreForTestV0(t,
		"run-ref-observe-active-goals-usage-limited-001",
		"goal-ref-observe-active-goals-usage-limited-001",
		orquestagoal.GoalStatusBlockedV0,
	)
	state, err := store.LoadGoalWorkStateV0(context.Background(), "run-ref-observe-active-goals-usage-limited-001")
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		Status:       orquestagoal.GoalStatusBlockedV0,
		GoalRef:      state.GoalRef,
		Summary:      "codex_app_server_goal_status_usageLimited",
		EvidenceRefs: []string{"evidence-ref-goal-provider-limited"},
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code: "codex_app_server_goal_provider_limited",
		}},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  orquestagoal.ErrGoalClosureInvalidV0,
			Field: "status",
		}},
	}
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	observer := &fakeMCPAutoprogrammingObserveActiveGoalsExecutorForTestV0{}

	result, err := NewMCPAutoprogrammingObserveActiveGoalsToolExecutorV0(store, observer).Execute(
		context.Background(),
		MCPAutoprogrammingObserveActiveGoalsToolInputV0{
			RequestID: "request-ref-observe-active-goals-usage-limited-001",
		},
	)

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.Observations) != 1 ||
		len(observer.inputs) != 0 ||
		!result.Observations[0].Partial ||
		result.Observations[0].GoalStatus != orquestagoal.GoalStatusBlockedV0 ||
		result.Observations[0].RecommendedAction != "inspect_goal_backend_limits" ||
		!hasMCPAutoprogrammingObserveActiveGoalsIssueCodeForTestV0(result.Issues, "codex_app_server_goal_provider_limited") ||
		!hasMCPAutoprogrammingObserveActiveGoalsIssueCodeForTestV0(result.Issues, "autoprogramming_goal_backend_limited") ||
		!hasStringMCPAutoprogrammingObserveActiveGoalsTestV0(result.NextActions, "inspect_goal_backend_limits") ||
		!hasStringMCPAutoprogrammingObserveActiveGoalsTestV0(result.NextActions, "wait_for_goal_backend_capacity") ||
		!hasStringMCPAutoprogrammingObserveActiveGoalsTestV0(result.EvidenceRefs, "evidence-ref-goal-provider-limited") {
		t.Fatalf("result=%+v inputs=%+v", result, observer.inputs)
	}
}

func TestMCPAutoprogrammingObserveActiveGoalsToolExecutorV0ConservaIncidenciaYContinua(t *testing.T) {
	store := newMCPAutoprogrammingObserveActiveGoalsStoreForTestV0(t,
		"run-ref-observe-active-goals-001",
		"goal-ref-observe-active-goals-001",
		orquestagoal.GoalStatusRunningV0,
	)
	store.mustSaveV0(t,
		"run-ref-observe-active-goals-002",
		"goal-ref-observe-active-goals-002",
		orquestagoal.GoalStatusRunningV0,
	)
	observer := &fakeMCPAutoprogrammingObserveActiveGoalsExecutorForTestV0{
		results: map[string]MCPAutoprogrammingObserveGoalToolResultV0{
			"run-ref-observe-active-goals-002": {
				Estado:     MCPAutoprogrammingObserveGoalEstadoOKV0,
				RunRef:     "run-ref-observe-active-goals-002",
				GoalRef:    "goal-ref-observe-active-goals-002",
				GoalStatus: orquestagoal.GoalStatusRunningV0,
			},
		},
		errorsByRun: map[string]error{
			"run-ref-observe-active-goals-001": errMCPAutoprogrammingObserveActiveGoalsForTestV0("observe_goal_timeout"),
		},
	}

	result, err := NewMCPAutoprogrammingObserveActiveGoalsToolExecutorV0(store, observer).Execute(
		context.Background(),
		MCPAutoprogrammingObserveActiveGoalsToolInputV0{},
	)

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.Observations) != 1 ||
		result.Observations[0].RunRef != "run-ref-observe-active-goals-002" ||
		len(result.Issues) != 1 ||
		result.Issues[0].Code != "autoprogramming_observe_goal_failed" ||
		result.Issues[0].Field != "run-ref-observe-active-goals-001" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingObserveActiveGoalsToolExecutorV0MarcaEstadoNoOKComoIncidencia(t *testing.T) {
	store := newMCPAutoprogrammingObserveActiveGoalsStoreForTestV0(t,
		"run-ref-observe-active-goals-non-ok-001",
		"goal-ref-observe-active-goals-non-ok-001",
		orquestagoal.GoalStatusRunningV0,
	)
	observer := &fakeMCPAutoprogrammingObserveActiveGoalsExecutorForTestV0{
		results: map[string]MCPAutoprogrammingObserveGoalToolResultV0{
			"run-ref-observe-active-goals-non-ok-001": {
				Estado:     MCPAutoprogrammingObserveGoalEstadoErrorV0,
				RunRef:     "run-ref-observe-active-goals-non-ok-001",
				GoalRef:    "goal-ref-observe-active-goals-non-ok-001",
				GoalStatus: orquestagoal.GoalStatusRunningV0,
			},
		},
	}

	result, err := NewMCPAutoprogrammingObserveActiveGoalsToolExecutorV0(store, observer).Execute(
		context.Background(),
		MCPAutoprogrammingObserveActiveGoalsToolInputV0{},
	)

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.Observations) != 1 ||
		result.Observations[0].Estado != MCPAutoprogrammingObserveGoalEstadoErrorV0 ||
		len(result.Issues) != 1 ||
		result.Issues[0].Code != "autoprogramming_observe_goal_non_ok" ||
		result.Issues[0].Field != "run-ref-observe-active-goals-non-ok-001" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveActiveGoalsHTTPExecutorForTestV0{
		result: MCPAutoprogrammingObserveActiveGoalsToolResultV0{
			Estado: MCPAutoprogrammingObserveActiveGoalsEstadoOKV0,
			Observations: []MCPAutoprogrammingObserveGoalToolResultV0{{
				RunRef:     "run-ref-observe-active-goals-http-001",
				GoalStatus: orquestagoal.GoalStatusRunningV0,
			}},
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingObserveActiveGoalsToolInputV0{
		RequestID: "request-ref-observe-active-goals-http-001",
		MaxItems:  4,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveActiveGoalsHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingObserveActiveGoalsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if executor.input.MaxItems != 4 ||
		len(result.Observations) != 1 ||
		result.Observations[0].RunRef != "run-ref-observe-active-goals-http-001" {
		t.Fatalf("input=%+v result=%+v", executor.input, result)
	}
}

func TestMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0AceptaBackgroundSiTarda(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveActiveGoalsHTTPExecutorForTestV0{
		delay: 25 * time.Millisecond,
		result: MCPAutoprogrammingObserveActiveGoalsToolResultV0{
			Estado: MCPAutoprogrammingObserveActiveGoalsEstadoOKV0,
		},
	}
	body := bytes.NewBufferString(`{
		"request_id":"request-ref-observe-active-goals-background-001",
		"correlation_id":"corr-observe-active-goals-background-001",
		"run_refs":["run-ref-observe-active-goals-background-001"]
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveActiveGoalsHTTPPathV0, body)
	rec := httptest.NewRecorder()

	newMCPAutoprogrammingObserveActiveGoalsHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingObserveActiveGoalsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingObserveActiveGoalsEstadoOKV0 ||
		result.OperationRef == "" ||
		!strings.Contains(result.OperationRef, "request-ref-observe-active-goals-background-001") ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_observe_active_goals_background_accepted") ||
		len(result.NextActions) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0CancelaExecutorAlAceptarBackground(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveActiveGoalsHTTPExecutorForTestV0{
		waitForCancel:  true,
		cancelObserved: make(chan struct{}),
		result: MCPAutoprogrammingObserveActiveGoalsToolResultV0{
			Estado: MCPAutoprogrammingObserveActiveGoalsEstadoOKV0,
		},
	}
	req := httptest.NewRequest(
		http.MethodPost,
		MCPAutoprogrammingObserveActiveGoalsHTTPPathV0,
		bytes.NewBufferString(`{"request_id":"request-ref-observe-active-goals-cancel-001"}`),
	)
	rec := httptest.NewRecorder()

	newMCPAutoprogrammingObserveActiveGoalsHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	select {
	case <-executor.cancelObserved:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("executor no recibio cancelacion tras background accepted")
	}
}

func TestMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0ExecutorErrorDevuelveDiagnosticoPublico(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveActiveGoalsHTTPExecutorForTestV0{
		err: errors.New("fallo interno /home/alberto/Trabajo token=secret"),
	}
	req := httptest.NewRequest(
		http.MethodPost,
		MCPAutoprogrammingObserveActiveGoalsHTTPPathV0,
		bytes.NewBufferString(`{
			"request_id":"request-ref-observe-active-goals-error-001",
			"run_refs":["run-ref-observe-active-goals-error-001"]
		}`),
	)
	req.Header.Set("X-Correlation-ID", "corr-observe-active-goals-error-001")
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError ||
		rec.Header().Get("X-Correlation-ID") != "corr-observe-active-goals-error-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPAutoprogrammingObserveActiveGoalsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingObserveActiveGoalsEstadoErrorV0 ||
		result.OperationRef == "" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_observe_active_goals_execute_error" ||
		result.Errores[0].Message != "autoprogramming_observe_active_goals_execute_error" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_observe_active_goals_execute_error") ||
		!hasStringMCPAutoprogrammingObserveActiveGoalsTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-observe-active-goals-execute-error") ||
		strings.Contains(result.Errores[0].Message, "/home/alberto") ||
		strings.Contains(result.Errores[0].Message, "secret") {
		t.Fatalf("result=%+v", result)
	}
}

type errMCPAutoprogrammingObserveActiveGoalsForTestV0 string

func (err errMCPAutoprogrammingObserveActiveGoalsForTestV0) Error() string {
	return string(err)
}

type fakeMCPAutoprogrammingObserveActiveGoalsExecutorForTestV0 struct {
	inputs      []MCPAutoprogrammingObserveGoalToolInputV0
	results     map[string]MCPAutoprogrammingObserveGoalToolResultV0
	errorsByRun map[string]error
}

func (executor *fakeMCPAutoprogrammingObserveActiveGoalsExecutorForTestV0) Execute(
	_ context.Context,
	input MCPAutoprogrammingObserveGoalToolInputV0,
) (MCPAutoprogrammingObserveGoalToolResultV0, error) {
	executor.inputs = append(executor.inputs, input)
	if err := executor.errorsByRun[input.RunRef]; err != nil {
		return MCPAutoprogrammingObserveGoalToolResultV0{}, err
	}
	if result, ok := executor.results[input.RunRef]; ok {
		return result, nil
	}
	return MCPAutoprogrammingObserveGoalToolResultV0{
		Estado: MCPAutoprogrammingObserveGoalEstadoOKV0,
		RunRef: input.RunRef,
	}, nil
}

type fakeMCPAutoprogrammingObserveActiveGoalsHTTPExecutorForTestV0 struct {
	input          MCPAutoprogrammingObserveActiveGoalsToolInputV0
	result         MCPAutoprogrammingObserveActiveGoalsToolResultV0
	delay          time.Duration
	err            error
	waitForCancel  bool
	cancelObserved chan struct{}
}

func (executor *fakeMCPAutoprogrammingObserveActiveGoalsHTTPExecutorForTestV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) (MCPAutoprogrammingObserveActiveGoalsToolResultV0, error) {
	executor.input = input
	if executor.waitForCancel {
		if executor.cancelObserved == nil {
			executor.cancelObserved = make(chan struct{})
		}
		<-ctx.Done()
		close(executor.cancelObserved)
		return executor.result, nil
	}
	if executor.delay > 0 {
		time.Sleep(executor.delay)
	}
	if executor.err != nil {
		return MCPAutoprogrammingObserveActiveGoalsToolResultV0{}, executor.err
	}
	return executor.result, nil
}

type mcpAutoprogrammingObserveActiveGoalsStoreForTestV0 struct {
	states map[string]orquestagoal.GoalWorkStateV0
}

func newMCPAutoprogrammingObserveActiveGoalsStoreForTestV0(
	t *testing.T,
	runRef string,
	goalRef string,
	status string,
) *mcpAutoprogrammingObserveActiveGoalsStoreForTestV0 {
	t.Helper()
	store := &mcpAutoprogrammingObserveActiveGoalsStoreForTestV0{
		states: map[string]orquestagoal.GoalWorkStateV0{},
	}
	store.mustSaveV0(t, runRef, goalRef, status)
	return store
}

func (store *mcpAutoprogrammingObserveActiveGoalsStoreForTestV0) mustSaveV0(
	t *testing.T,
	runRef string,
	goalRef string,
	status string,
) {
	t.Helper()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Observar goal activo.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusRunningV0,
			EvidenceRefs: []string{"evidence-ref-observe-active-goals-store"},
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.Status = status
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		t.Fatalf("NewGoalWorkStateV0: %v", err)
	}
	store.states[runRef] = state
}

func (store *mcpAutoprogrammingObserveActiveGoalsStoreForTestV0) SaveGoalWorkStateV0(
	_ context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return err
	}
	store.states[normalized.RunRef] = normalized
	return nil
}

func (store *mcpAutoprogrammingObserveActiveGoalsStoreForTestV0) LoadGoalWorkStateV0(
	_ context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	return store.states[runRef], nil
}

func (store *mcpAutoprogrammingObserveActiveGoalsStoreForTestV0) ListGoalWorkStatesV0(
	_ context.Context,
	request orquestagoal.GoalWorkStateListRequestV0,
) ([]orquestagoal.GoalWorkStateV0, error) {
	request = orquestagoal.NormalizeGoalWorkStateListRequestV0(request)
	out := []orquestagoal.GoalWorkStateV0{}
	for _, state := range store.states {
		if orquestagoal.GoalWorkStateMatchesListRequestV0(state, request) {
			out = append(out, state)
		}
	}
	return out, nil
}

func hasStringMCPAutoprogrammingObserveActiveGoalsTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasInputRunMCPAutoprogrammingObserveActiveGoalsTestV0(
	inputs []MCPAutoprogrammingObserveGoalToolInputV0,
	want string,
) bool {
	for _, input := range inputs {
		if input.RunRef == want {
			return true
		}
	}
	return false
}

func hasMCPAutoprogrammingObserveActiveGoalsIssueCodeForTestV0(
	issues []MCPValidationIssueV0,
	want string,
) bool {
	for _, issue := range issues {
		if issue.Code == want {
			return true
		}
	}
	return false
}
