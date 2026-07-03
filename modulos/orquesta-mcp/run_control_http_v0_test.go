package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestMCPRunControlHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPRunControlHTTPExecutorV0{
		result: MCPRunControlToolResultV0{
			Estado:        MCPRunControlEstadoOKV0,
			CorrelationID: "corr-run-control-http-result-001",
			RunRef:        "run-ref-control-http-001",
			Status:        "paused",
		},
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunControlToolInputV0{
		Action:        "pause",
		RunRef:        "run-ref-control-http-001",
		RequestID:     "request-ref-run-control-http-001",
		CorrelationID: "corr-run-control-http-input-001",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunControlHTTPPathV0, body)

	NewMCPRunControlHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-result-001" ||
		executor.input.RunRef != "run-ref-control-http-001" {
		t.Fatalf("status=%d headers=%v input=%+v body=%s", rec.Code, rec.Header(), executor.input, rec.Body.String())
	}
}

func TestMCPRunControlHTTPHandlerV0MetodoYExecutorNil(t *testing.T) {
	for _, tc := range []struct {
		name    string
		method  string
		handler http.Handler
		want    int
	}{
		{name: "metodo", method: http.MethodGet, handler: NewMCPRunControlHTTPHandlerV0(&fakeMCPRunControlHTTPExecutorV0{}), want: http.StatusMethodNotAllowed},
		{name: "nil", method: http.MethodPost, handler: NewMCPRunControlHTTPHandlerV0(nil), want: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, MCPRunControlHTTPPathV0, bytes.NewBufferString(`{}`))
			req.Header.Set("X-Correlation-ID", "corr-run-control-http-error-001")

			tc.handler.ServeHTTP(rec, req)

			if rec.Code != tc.want ||
				rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-error-001" {
				t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
			}
		})
	}
}

func TestMCPRunControlHTTPHandlerV0BackendGoalActivoEsConflictV0(t *testing.T) {
	executor := &fakeMCPRunControlHTTPExecutorV0{
		result: MCPRunControlToolResultV0{
			Estado:        MCPRunControlEstadoErrorV0,
			RequestID:     "request-ref-run-control-http-conflict-001",
			CorrelationID: "corr-run-control-http-conflict-001",
			Action:        "stop",
			RunRef:        "run-ref-control-http-conflict-001",
			Status:        "stop_requested",
			Errores: []MCPValidationIssueV0{{
				Code:    "control_not_propagated_to_goal_backend",
				Field:   "goal_backend",
				Message: "goal backend sigue activo tras control local",
			}},
		},
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunControlToolInputV0{
		Action:    "stop",
		RunRef:    "run-ref-control-http-conflict-001",
		RequestID: "request-ref-run-control-http-conflict-001",
		Forced:    true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunControlHTTPPathV0, body)

	NewMCPRunControlHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-conflict-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPRunControlToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunControlEstadoErrorV0 ||
		result.Status != "stop_requested" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "control_not_propagated_to_goal_backend" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunControlHTTPHandlerV0CancelExternalCleanupSinForceV0(t *testing.T) {
	runRef := "run-ref-control-http-external-cleanup-cancel-001"
	goalRef := "goal-ref-control-http-external-cleanup-cancel-001"
	externalGoalRef := "thread-ref-" + goalRef
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Cancelar por HTTP tras cleanup externo gobernado.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState: runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunControlToolInputV0{
		Action:        "cancel",
		RunRef:        runRef,
		RequestID:     "request-ref-run-control-http-external-cleanup-cancel-001",
		CorrelationID: "corr-run-control-http-external-cleanup-cancel-001",
		RequestedBy:   "orquesta-director",
		Reason:        "seguir run_control_reconcile_external_cleanup cancelando por HTTP",
		Forced:        false,
		EvidenceRefs:  []string{mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunControlHTTPPathV0, body)

	NewMCPRunControlHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-external-cleanup-cancel-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPRunControlToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "cancel" ||
		result.Status != string(orquestaruncontrol.RunControlStatusCanceledV0) ||
		result.RecommendedAction != "replan_narrow_context" ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_external_cleanup") ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-goal-external-cleanup-reconciled") ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v", result)
	}
	if port.complete.TargetStatus != orquestaruncontrol.RunControlStatusCanceledV0 {
		t.Fatalf("complete=%+v", port.complete)
	}
}

func TestMCPRunControlHTTPHandlerV0StopExternalCleanupSinForceV0(t *testing.T) {
	runRef := "run-ref-control-http-external-cleanup-stop-001"
	goalRef := "goal-ref-control-http-external-cleanup-stop-001"
	externalGoalRef := "thread-ref-" + goalRef
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Parar por HTTP tras cleanup externo gobernado.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState:  runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
		stopStatus: orquestaruncontrol.RunControlStatusStoppedV0,
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunControlToolInputV0{
		Action:        "stop",
		RunRef:        runRef,
		RequestID:     "request-ref-run-control-http-external-cleanup-stop-001",
		CorrelationID: "corr-run-control-http-external-cleanup-stop-001",
		RequestedBy:   "orquesta-director",
		Reason:        "seguir run_control_reconcile_external_cleanup parando por HTTP",
		Forced:        false,
		EvidenceRefs:  []string{mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunControlHTTPPathV0, body)

	NewMCPRunControlHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-external-cleanup-stop-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPRunControlToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "stop" ||
		result.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		result.RecommendedAction != "replan_narrow_context" ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_external_cleanup") ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-goal-external-cleanup-reconciled") ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v", result)
	}
	if port.complete.TargetStatus != orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("complete=%+v", port.complete)
	}
}

func TestMCPRunControlHTTPHandlerV0ResumeNoReconciliaExternalCleanupSinForceV0(t *testing.T) {
	runRef := "run-ref-control-http-external-cleanup-resume-001"
	goalRef := "goal-ref-control-http-external-cleanup-resume-001"
	externalGoalRef := "thread-ref-" + goalRef
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Reanudar por HTTP sin reconciliar cleanup externo.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState: runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusPausedV0, false),
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunControlToolInputV0{
		Action:        "resume",
		RunRef:        runRef,
		RequestID:     "request-ref-run-control-http-external-cleanup-resume-001",
		CorrelationID: "corr-run-control-http-external-cleanup-resume-001",
		RequestedBy:   "orquesta-director",
		Reason:        "reanudar sin reconciliar cleanup externo por HTTP",
		Forced:        false,
		EvidenceRefs:  []string{mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunControlHTTPPathV0, body)

	NewMCPRunControlHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-external-cleanup-resume-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPRunControlToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "resume" ||
		result.Status != string(orquestaruncontrol.RunControlStatusRunningV0) ||
		result.RecommendedAction != "" ||
		port.complete.RunRef != "" ||
		containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_external_cleanup") ||
		!containsStringMCPTestV0(port.resume.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v resume=%+v complete=%+v", result, port.resume, port.complete)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusRunningV0 ||
		state.LastResult != nil ||
		state.LastClosure != nil {
		t.Fatalf("state mutado por resume HTTP con evidencia cleanup externo: %+v", state)
	}
}

func TestMCPRunControlHTTPHandlerV0PauseNoReconciliaExternalCleanupSinForceV0(t *testing.T) {
	runRef := "run-ref-control-http-external-cleanup-pause-001"
	goalRef := "goal-ref-control-http-external-cleanup-pause-001"
	externalGoalRef := "thread-ref-" + goalRef
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Pausar por HTTP sin reconciliar cleanup externo.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState: runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunControlToolInputV0{
		Action:        "pause",
		RunRef:        runRef,
		RequestID:     "request-ref-run-control-http-external-cleanup-pause-001",
		CorrelationID: "corr-run-control-http-external-cleanup-pause-001",
		RequestedBy:   "orquesta-director",
		Reason:        "pausar sin reconciliar cleanup externo por HTTP",
		Forced:        false,
		EvidenceRefs:  []string{mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunControlHTTPPathV0, body)

	NewMCPRunControlHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-external-cleanup-pause-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPRunControlToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "pause" ||
		result.Status != string(orquestaruncontrol.RunControlStatusPausedV0) ||
		result.RecommendedAction != "" ||
		port.complete.RunRef != "" ||
		containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_external_cleanup") ||
		!containsStringMCPTestV0(port.pause.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v pause=%+v complete=%+v", result, port.pause, port.complete)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusRunningV0 ||
		state.LastResult != nil ||
		state.LastClosure != nil {
		t.Fatalf("state mutado por pause HTTP con evidencia cleanup externo: %+v", state)
	}
}

func TestMCPRunControlHTTPHandlerV0ExecutorErrorConservaEvidenciaV0(t *testing.T) {
	executor := &fakeMCPRunControlHTTPExecutorV0{
		err: errors.New("executor unavailable"),
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunControlToolInputV0{
		Action:        "stop",
		RunRef:        "run-ref-control-http-executor-error-001",
		RequestID:     "request-ref-run-control-http-executor-error-001",
		CorrelationID: "corr-run-control-http-executor-error-input-001",
		Forced:        true,
		EvidenceRefs: []string{
			mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
			mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
		},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunControlHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-run-control-http-executor-error-header-001")

	NewMCPRunControlHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-executor-error-header-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPRunControlToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunControlEstadoErrorV0 ||
		result.Action != "stop" ||
		result.RunRef != "run-ref-control-http-executor-error-001" ||
		!result.Forced ||
		len(result.EvidenceRefs) != 1 ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) ||
		len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != "run_control_http_error" ||
		result.Diagnostics[0].Scope != "run:run-ref-control-http-executor-error-001" ||
		!containsStringMCPTestV0(
			result.Diagnostics[0].EvidenceRefs,
			mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
		) ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_control_http_error" {
		t.Fatalf("executor error debe conservar evidencia compacta: %+v", result)
	}
}

func TestMCPRunControlHTTPHandlerV0TimeoutDevuelveJSONPublico(t *testing.T) {
	executor := &blockingMCPRunControlHTTPExecutorV0{done: make(chan struct{})}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunControlToolInputV0{
		Action:         "stop",
		RunRef:         "run-ref-control-http-timeout-001",
		RequestID:      "request-ref-run-control-http-timeout-001",
		CorrelationID:  "corr-run-control-http-timeout-input-001",
		IdempotencyKey: "idem-run-control-http-timeout-001",
		Forced:         true,
		EvidenceRefs: []string{
			mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
			"",
			mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
		},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunControlHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-run-control-http-timeout-header-001")

	newMCPRunControlHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-timeout-header-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPRunControlToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunControlEstadoErrorV0 ||
		result.Action != "stop" ||
		result.RunRef != "run-ref-control-http-timeout-001" ||
		!result.Forced ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "run_control_timeout") ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_control_timeout" {
		t.Fatalf("result=%+v", result)
	}
	if len(result.EvidenceRefs) != 1 ||
		len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Scope != "run:run-ref-control-http-timeout-001" ||
		!containsStringMCPTestV0(
			result.Diagnostics[0].EvidenceRefs,
			mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
		) {
		t.Fatalf("timeout debe conservar evidencia compacta y diagnostico: %+v", result)
	}
	select {
	case <-executor.done:
	case <-time.After(time.Second):
		t.Fatalf("executor no recibio cancelacion tras timeout HTTP")
	}
}

type fakeMCPRunControlHTTPExecutorV0 struct {
	input  MCPRunControlToolInputV0
	result MCPRunControlToolResultV0
	err    error
}

type blockingMCPRunControlHTTPExecutorV0 struct {
	done chan struct{}
}

func (executor *fakeMCPRunControlHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolResultV0, error) {
	executor.input = input
	return executor.result, executor.err
}

func (executor *blockingMCPRunControlHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolResultV0, error) {
	<-ctx.Done()
	close(executor.done)
	return MCPRunControlToolResultV0{
		Estado: MCPRunControlEstadoOKV0,
		Action: normalizeMCPRunControlActionV0(input.Action),
		RunRef: input.RunRef,
	}, nil
}
