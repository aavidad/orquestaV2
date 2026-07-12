package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0{
		result: MCPAutoprogrammingObserveGoalToolResultV0{
			Estado:     MCPAutoprogrammingObserveGoalEstadoOKV0,
			RunRef:     "run-ref-autoprogramming-goal-http-001",
			GoalRef:    "goal-ref-autoprogramming-goal-http-001",
			GoalStatus: "running",
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingObserveGoalToolInputV0{
		RequestID: "request-ref-autoprogramming-observe-goal-http-001",
		RunRef:    "run-ref-autoprogramming-goal-http-001",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingObserveGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if executor.input.RunRef != "run-ref-autoprogramming-goal-http-001" ||
		result.GoalRef != "goal-ref-autoprogramming-goal-http-001" {
		t.Fatalf("input=%+v result=%+v", executor.input, result)
	}
}

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0TimeoutDevuelveJSONPublico(t *testing.T) {
	executor := newBlockingMCPAutoprogrammingObserveGoalHTTPExecutorV0()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingObserveGoalToolInputV0{
		RequestID:     "request-ref-autoprogramming-observe-goal-http-timeout-001",
		CorrelationID: "corr-autoprogramming-observe-goal-http-timeout-input-001",
		RunRef:        "run-ref-autoprogramming-goal-http-timeout-001",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-autoprogramming-observe-goal-http-timeout-header-001")
	rec := httptest.NewRecorder()

	handler := newMCPAutoprogrammingObserveGoalHTTPHandlerWithTimeoutV0(executor, time.Millisecond)
	handler.ServeHTTP(rec, req)
	<-executor.started

	if rec.Code != http.StatusAccepted || rec.Header().Get("Location") != MCPAutoprogrammingObserveGoalHTTPPathV0 ||
		rec.Header().Get("Retry-After") != "1" ||
		rec.Header().Get("X-Correlation-ID") != "corr-autoprogramming-observe-goal-http-timeout-header-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingObserveGoalEstadoOKV0 || result.OperationRef == "" ||
		!result.Partial ||
		result.RunRef != "run-ref-autoprogramming-goal-http-timeout-001" ||
		result.RecommendedAction != "observe_later" ||
		result.Summary == "" ||
		len(result.Errores) != 0 ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-observe-goal-timeout") {
		t.Fatalf("result=%+v", result)
	}
	select {
	case <-executor.done:
		t.Fatalf("el deadline HTTP cancelo el executor durable")
	default:
	}
	replayRunning := httptest.NewRecorder()
	replayRunningRequest := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, strings.NewReader(`{"request_id":"request-ref-autoprogramming-observe-goal-http-timeout-distinto-002","run_ref":"run-ref-autoprogramming-goal-http-timeout-001"}`))
	handler.ServeHTTP(replayRunning, replayRunningRequest)
	if replayRunning.Code != http.StatusAccepted || executor.calls != 1 {
		t.Fatalf("replay running status=%d calls=%d body=%s", replayRunning.Code, executor.calls, replayRunning.Body.String())
	}
	close(executor.release)
	<-executor.done
	replay := httptest.NewRecorder()
	replayRequest := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, strings.NewReader(`{"request_id":"request-ref-autoprogramming-observe-goal-http-timeout-001","run_ref":"run-ref-autoprogramming-goal-http-timeout-001"}`))
	handler.ServeHTTP(replay, replayRequest)
	if replay.Code != http.StatusOK || executor.calls != 1 {
		t.Fatalf("replay status=%d calls=%d body=%s", replay.Code, executor.calls, replay.Body.String())
	}
}

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0ClienteCanceladoNoCancelaTrabajoV0(t *testing.T) {
	executor := newBlockingMCPAutoprogrammingObserveGoalHTTPExecutorV0()
	handler := newMCPAutoprogrammingObserveGoalHTTPHandlerWithTimeoutV0(executor, time.Millisecond)
	request := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, strings.NewReader(
		`{"request_id":"req-autoprogramming-http-client-cancel-001","run_ref":"run-ref-autoprogramming-http-client-cancel-001"}`,
	))
	clientContext, cancelClient := context.WithCancel(request.Context())
	cancelClient()
	request = request.WithContext(clientContext)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	select {
	case <-executor.done:
		t.Fatalf("la cancelacion del cliente cancelo el trabajo durable")
	default:
	}
	close(executor.release)
	<-executor.done
}

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0ExecutorTimeoutOrCancellationDevuelveTimeoutPublico(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{
			name: "deadline exceeded",
			err:  fmt.Errorf("observer residente bloqueado en /private/runtime: %w", context.DeadlineExceeded),
		},
		{
			name: "cancelled",
			err:  fmt.Errorf("observer residente cancelado con token sensible: %w", context.Canceled),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			executor := &fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0{err: testCase.err}
			input := MCPAutoprogrammingObserveGoalToolInputV0{
				RequestID: "request-ref-autoprogramming-observe-goal-http-contention-001",
				RunRef:    "run-ref-autoprogramming-goal-http-contention-001",
			}
			body := bytes.NewBuffer(nil)
			if err := json.NewEncoder(body).Encode(input); err != nil {
				t.Fatalf("encode: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, body)
			rec := httptest.NewRecorder()

			NewMCPAutoprogrammingObserveGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

			if rec.Code != http.StatusGatewayTimeout {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "/private/runtime") || strings.Contains(rec.Body.String(), "token sensible") {
				t.Fatalf("body expone detalle interno: %s", rec.Body.String())
			}
			var result MCPAutoprogrammingObserveGoalToolResultV0
			if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if result.Estado != MCPAutoprogrammingObserveGoalEstadoErrorV0 ||
				!result.Partial ||
				result.RunRef != input.RunRef ||
				result.RecommendedAction != "observe_later" ||
				len(result.Errores) != 1 ||
				result.Errores[0].Code != MCPAutoprogrammingObserveGoalHTTPTimeoutCodeV0 {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0TimeoutIncluyeSnapshotParcialV0(t *testing.T) {
	executor := &blockingSnapshotMCPAutoprogrammingObserveGoalHTTPExecutorV0{
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
		snapshot: MCPObserveAppDirectorGoalToolResultV0{
			Estado:            MCPObserveAppDirectorGoalEstadoOKV0,
			Partial:           true,
			RunRef:            "run-ref-autoprogramming-goal-http-timeout-snapshot-001",
			GoalRef:           "goal-ref-autoprogramming-goal-http-timeout-snapshot-001",
			ExternalGoalRef:   "thread-ref-autoprogramming-goal-http-timeout-snapshot-001",
			GoalStatus:        "blocked",
			RecommendedAction: "replan",
			ArtifactRefs:      []string{"artifact-ref-checkpoint-timeout-snapshot-001"},
			EvidenceRefs:      []string{"evidence-ref-autoprogramming-timeout-snapshot-001"},
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingObserveGoalToolInputV0{
		RequestID: "request-ref-autoprogramming-observe-goal-http-timeout-snapshot-001",
		RunRef:    "run-ref-autoprogramming-goal-http-timeout-snapshot-001",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-autoprogramming-observe-goal-http-timeout-snapshot-001")
	rec := httptest.NewRecorder()

	newMCPAutoprogrammingObserveGoalHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingObserveGoalEstadoOKV0 || result.OperationRef == "" ||
		!result.Partial ||
		result.GoalRef != "goal-ref-autoprogramming-goal-http-timeout-snapshot-001" ||
		result.ExternalGoalRef != "thread-ref-autoprogramming-goal-http-timeout-snapshot-001" ||
		result.GoalStatus != "blocked" ||
		result.RecommendedAction != "replan" ||
		len(result.Errores) != 0 ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.ArtifactRefs, "artifact-ref-checkpoint-timeout-snapshot-001") ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-timeout-snapshot-001") {
		t.Fatalf("result=%+v", result)
	}
	select {
	case <-executor.done:
		t.Fatalf("el deadline HTTP cancelo el executor durable")
	default:
	}
	close(executor.release)
	<-executor.done
}

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0TimeoutNoPublicaClosureBloqueadoSiGoalSigueRunningV0(t *testing.T) {
	executor := &blockingSnapshotMCPAutoprogrammingObserveGoalHTTPExecutorV0{
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
		snapshot: MCPObserveAppDirectorGoalToolResultV0{
			Estado:             MCPObserveAppDirectorGoalEstadoOKV0,
			Partial:            true,
			RunRef:             "run-ref-autoprogramming-goal-http-timeout-running-001",
			GoalRef:            "goal-ref-autoprogramming-goal-http-timeout-running-001",
			ExternalGoalRef:    "thread-ref-autoprogramming-goal-http-timeout-running-001",
			GoalStatus:         "running",
			RecommendedAction:  "replan",
			ClosureStatus:      "blocked",
			ClosureNeedsRework: true,
			ClosureIssues: []MCPValidationIssueV0{{
				Code:  MCPGoalFirstPartialArtifactsWrittenV0,
				Field: "goal_first.partial_artifacts",
			}},
			ArtifactRefs: []string{"artifact-ref-checkpoint-timeout-running-001"},
			EvidenceRefs: []string{"evidence-ref-autoprogramming-timeout-running-001"},
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingObserveGoalToolInputV0{
		RequestID: "request-ref-autoprogramming-observe-goal-http-timeout-running-001",
		RunRef:    "run-ref-autoprogramming-goal-http-timeout-running-001",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, body)
	rec := httptest.NewRecorder()

	newMCPAutoprogrammingObserveGoalHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.GoalStatus != "running" ||
		result.RecommendedAction != "observe_later" ||
		result.ClosureStatus != "" ||
		result.ClosureNeedsRework ||
		result.ClosureAccepted ||
		len(result.ClosureIssues) != 0 ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.ArtifactRefs, "artifact-ref-checkpoint-timeout-running-001") ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-timeout-running-001") {
		t.Fatalf("result=%+v", result)
	}
	select {
	case <-executor.done:
		t.Fatalf("el deadline HTTP cancelo el executor durable")
	default:
	}
	close(executor.release)
	<-executor.done
}

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0RunRefRequerido(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0{}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingObserveGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingObserveGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "run_ref" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0ErrorOperativoNoDevuelve500(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0{
		err: errors.New("goal state no encontrado: run-ref-autoprogramming-goal-http-missing-state-001"),
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingObserveGoalToolInputV0{
		RequestID: "request-ref-autoprogramming-observe-goal-http-missing-state-001",
		RunRef:    "run-ref-autoprogramming-goal-http-missing-state-001",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingObserveGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingObserveGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_observe_goal_state_not_found" ||
		result.Errores[0].Field != "goal_state" ||
		result.Errores[0].Message != "goal_state_not_found" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0ErrorRealDevuelve500(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0{
		err: errors.New("fallo inesperado del executor"),
	}
	req := httptest.NewRequest(
		http.MethodPost,
		MCPAutoprogrammingObserveGoalHTTPPathV0,
		strings.NewReader(`{"run_ref":"run-ref-autoprogramming-observe-goal-http-real-error-001"}`),
	)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingObserveGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0 struct {
	input  MCPAutoprogrammingObserveGoalToolInputV0
	result MCPAutoprogrammingObserveGoalToolResultV0
	err    error
}

func (executor *fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPAutoprogrammingObserveGoalToolInputV0,
) (MCPAutoprogrammingObserveGoalToolResultV0, error) {
	executor.input = input
	if executor.result.Estado == "" {
		executor.result = MCPAutoprogrammingObserveGoalToolResultV0{
			Estado: MCPAutoprogrammingObserveGoalEstadoOKV0,
			RunRef: input.RunRef,
		}
	}
	return executor.result, executor.err
}

type blockingMCPAutoprogrammingObserveGoalHTTPExecutorV0 struct {
	started chan struct{}
	release chan struct{}
	done    chan struct{}
	once    sync.Once
	calls   int
}

func newBlockingMCPAutoprogrammingObserveGoalHTTPExecutorV0() *blockingMCPAutoprogrammingObserveGoalHTTPExecutorV0 {
	return &blockingMCPAutoprogrammingObserveGoalHTTPExecutorV0{
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (executor *blockingMCPAutoprogrammingObserveGoalHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingObserveGoalToolInputV0,
) (MCPAutoprogrammingObserveGoalToolResultV0, error) {
	executor.calls++
	executor.once.Do(func() { close(executor.started) })
	select {
	case <-executor.release:
	case <-ctx.Done():
		return MCPAutoprogrammingObserveGoalToolResultV0{}, ctx.Err()
	}
	close(executor.done)
	return MCPAutoprogrammingObserveGoalToolResultV0{
		Estado: MCPAutoprogrammingObserveGoalEstadoOKV0,
		RunRef: input.RunRef,
	}, nil
}

type blockingSnapshotMCPAutoprogrammingObserveGoalHTTPExecutorV0 struct {
	started  chan struct{}
	release  chan struct{}
	done     chan struct{}
	once     sync.Once
	snapshot MCPObserveAppDirectorGoalToolResultV0
}

func (executor *blockingSnapshotMCPAutoprogrammingObserveGoalHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingObserveGoalToolInputV0,
) (MCPAutoprogrammingObserveGoalToolResultV0, error) {
	executor.once.Do(func() { close(executor.started) })
	select {
	case <-executor.release:
	case <-ctx.Done():
		return MCPAutoprogrammingObserveGoalToolResultV0{}, ctx.Err()
	}
	close(executor.done)
	return MCPAutoprogrammingObserveGoalToolResultV0{
		Estado: MCPAutoprogrammingObserveGoalEstadoOKV0,
		RunRef: input.RunRef,
	}, nil
}

func (executor *blockingSnapshotMCPAutoprogrammingObserveGoalHTTPExecutorV0) ObserveAppDirectorGoalTimeoutSnapshotV0(
	_ context.Context,
	_ MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error) {
	return executor.snapshot, nil
}
