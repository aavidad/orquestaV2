package orquestaserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0ServerShutdownCongelaSupervisorResidenteV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &countingShutdownFreezeSupervisorV0{}
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"estado":              "ok",
			"status":              "waiting_drain",
			"shutdown_ready":      false,
			"runs_requested":      1,
			"runs_stopped":        1,
			"agents_in_flight":    1,
			"checkpoints_pending": 0,
		})
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef:      "global",
			MaxExecutions: 10,
		},
	}, RuntimeDepsV0{
		AppHandler: app,
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 25, 18, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	runtime.HandlerV0().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("shutdown status=%d body=%s", rec.Code, rec.Body.String())
	}
	state := runtime.StateV0()
	if !state.ShutdownInProgress ||
		!state.SupervisorFrozen ||
		state.ShutdownStatus != "waiting_drain" ||
		state.ShutdownAgentsInFlight != 1 {
		t.Fatalf("state shutdown=%+v", state)
	}

	runtime.runSupervisorTickV0(context.Background())
	if supervisor.calls != 0 {
		t.Fatalf("supervisor ejecutado durante shutdown: calls=%d", supervisor.calls)
	}
	state = runtime.StateV0()
	if state.LastSupervisorStatus != "skipped" ||
		state.LastSupervisorStop != "shutdown_in_progress" {
		t.Fatalf("state supervisor=%+v", state)
	}
}

func TestRuntimeV0ServerShutdownRechazadoNoCongelaSupervisorV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &countingShutdownFreezeSupervisorV0{}
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"estado":         "error",
			"status":         "requester_not_authorized",
			"shutdown_ready": true,
		})
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef:      "global",
			MaxExecutions: 10,
		},
	}, RuntimeDepsV0{
		AppHandler: app,
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 25, 18, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	runtime.HandlerV0().ServeHTTP(httptest.NewRecorder(), req)
	if runtime.StateV0().SupervisorFrozen || runtime.StateV0().ShutdownReady {
		t.Fatalf("shutdown rechazado publico ready o congelo supervisor: %+v", runtime.StateV0())
	}

	runtime.runSupervisorTickV0(context.Background())
	if supervisor.calls != 1 {
		t.Fatalf("supervisor no recuperado tras rechazo: calls=%d", supervisor.calls)
	}
}

func TestRuntimeV0ServerShutdownReadyDescongelaSupervisorV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &countingShutdownFreezeSupervisorV0{}
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"estado":         "ok",
			"status":         "ready",
			"shutdown_ready": true,
		})
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef:      "global",
			MaxExecutions: 10,
		},
	}, RuntimeDepsV0{
		AppHandler: app,
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 25, 18, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	runtime.HandlerV0().ServeHTTP(httptest.NewRecorder(), req)
	state := runtime.StateV0()
	if state.SupervisorFrozen || state.ShutdownInProgress || !state.ShutdownReady {
		t.Fatalf("shutdown ready dejo supervisor congelado: %+v", state)
	}

	runtime.runSupervisorTickV0(context.Background())
	if supervisor.calls != 1 {
		t.Fatalf("supervisor no recuperado tras ready: calls=%d", supervisor.calls)
	}
}

func TestRuntimeV0ServerShutdownReadyDetieneRuntimeHTTPV0(t *testing.T) {
	requireLocalTCPForServerTestV0(t)
	hook := newNotifyingRuntimeShutdownHookV0()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"estado":         "ok",
			"status":         "ready",
			"shutdown_ready": true,
		})
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:                "127.0.0.1:0",
		StateDir:            t.TempDir(),
		AuditDisabled:       true,
		TickInterval:        time.Hour,
		ShutdownGracePeriod: 500 * time.Millisecond,
	}, RuntimeDepsV0{
		AppHandler:    app,
		StateStore:    &threadSafeStateStoreV0{},
		ShutdownHooks: []RuntimeShutdownHookPortV0{hook},
		Clock:         fixedClockV0{now: time.Date(2026, 6, 29, 18, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(ctx) }()

	addr := waitRuntimeAddrV0(t, runtime)
	resp, err := http.Post("http://"+addr+serverShutdownRoutePathV0, "application/json", nil)
	if err != nil {
		t.Fatalf("POST shutdown: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shutdown status=%d", resp.StatusCode)
	}
	var payload serverShutdownHTTPProjectionV0
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode shutdown body: %v", err)
	}
	_ = resp.Body.Close()
	if !payload.ShutdownReady ||
		!payload.ExitPending ||
		payload.PID != os.Getpid() {
		t.Fatalf("shutdown response sin salida programada: %+v", payload)
	}
	if err := waitRuntimeDoneV0(t, done); err != nil {
		t.Fatalf("RunV0: %v", err)
	}
	waitForRuntimeTestV0(t, hook.called)
	if got := atomic.LoadInt32(&hook.calls); got != 1 {
		t.Fatalf("shutdown hook calls=%d", got)
	}
	state := runtime.StateV0()
	if state.Status != "stopped" ||
		state.ShutdownStatus != "stopped" ||
		!state.ShutdownReady ||
		state.ShutdownSignalName != "http_shutdown_ready" ||
		state.ShutdownSignalCount != 1 {
		t.Fatalf("runtime no paro por shutdown HTTP ready: %+v", state)
	}
}

func TestRuntimeV0ShutdownReadyProgramaSalidaForzadaSiNoTerminaV0(t *testing.T) {
	forceExit := &recordingForceExitPortV0{codes: make(chan int, 1)}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:            t.TempDir(),
		AuditDisabled:       true,
		ShutdownGracePeriod: 20 * time.Millisecond,
	}, RuntimeDepsV0{
		ForceExit: forceExit,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.requestShutdownReadyV0()

	select {
	case code := <-forceExit.codes:
		if code != 0 {
			t.Fatalf("exit code=%d", code)
		}
	case <-time.After(time.Second):
		t.Fatalf("shutdown_ready no programo salida forzada")
	}
}

func TestRuntimeV0ServerShutdownReadyConAgenteVivoQuedaStopPendingV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &countingShutdownFreezeSupervisorV0{}
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"estado":              "ok",
			"status":              "stopped",
			"shutdown_ready":      true,
			"runs_requested":      1,
			"runs_stopped":        1,
			"agents_in_flight":    1,
			"checkpoints_pending": 0,
		})
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef:      "global",
			MaxExecutions: 10,
		},
	}, RuntimeDepsV0{
		AppHandler: app,
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	runtime.HandlerV0().ServeHTTP(httptest.NewRecorder(), req)
	state := runtime.StateV0()
	if !state.SupervisorFrozen ||
		!state.ShutdownInProgress ||
		state.ShutdownReady ||
		state.ShutdownStatus != "stop_pending" ||
		state.ShutdownAgentsInFlight != 1 {
		t.Fatalf("shutdown stop pendiente no reflejado: %+v", state)
	}
}

func TestShutdownProjectionFromHTTPV0ReadySinConfirmacionesQuedaStopPendingV0(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
	}{{
		name: "runs_sin_confirmar",
		body: map[string]any{
			"estado":         "ok",
			"status":         "ready",
			"shutdown_ready": true,
			"runs_requested": 2,
			"runs_stopped":   1,
		},
	}, {
		name: "checkpoint_pendiente",
		body: map[string]any{
			"estado":              "ok",
			"status":              "ready",
			"shutdown_ready":      true,
			"runs_requested":      1,
			"runs_stopped":        1,
			"checkpoints_pending": 1,
		},
	}, {
		name: "agente_checkpoint_pendiente",
		body: map[string]any{
			"estado":                    "ok",
			"status":                    "ready",
			"shutdown_ready":            true,
			"runs_requested":            1,
			"runs_stopped":              1,
			"checkpoint_agents_pending": 1,
		},
	}, {
		name: "async_work_pendiente",
		body: map[string]any{
			"estado":                     "ok",
			"status":                     "ready",
			"shutdown_ready":             true,
			"runs_requested":             1,
			"runs_stopped":               1,
			"shutdown_async_work_active": 1,
		},
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}

			projection, keepFrozen := shutdownProjectionFromHTTPV0(http.StatusOK, body)

			if !keepFrozen ||
				projection.Ready ||
				projection.Status != "stop_pending" {
				t.Fatalf("projection=%+v keep_frozen=%v", projection, keepFrozen)
			}
		})
	}
}

func TestRuntimeV0ServerShutdownConservaAsyncWorkActiveV0(t *testing.T) {
	stateDir := t.TempDir()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"estado":"ok",
			"status":"ready",
			"shutdown_ready":true,
			"runs_requested":1,
			"runs_stopped":1,
			"shutdown_async_work_active":1
		}`))
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler: app,
		Clock:      fixedClockV0{now: time.Date(2026, 7, 3, 23, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	runtime.HandlerV0().ServeHTTP(httptest.NewRecorder(), req)
	state := runtime.StateV0()
	public := NewServerPublicStatusV0(state)

	if !state.ShutdownInProgress ||
		state.ShutdownReady ||
		state.ShutdownStatus != "stop_pending" ||
		state.ShutdownAsyncWorkActive != 1 ||
		public.ShutdownAsyncWorkActive != 1 {
		t.Fatalf("async_work_active no conservado: state=%+v public=%+v", state, public)
	}
}

func TestRuntimeV0ServerShutdownSnapshotPrevioConservaAsyncWorkActiveV0(t *testing.T) {
	stateDir := t.TempDir()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"estado":"ok",
			"status":"waiting_drain",
			"shutdown_ready":false,
			"runs_requested":1,
			"runs_stopped":1
		}`))
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler: app,
		Clock:      fixedClockV0{now: time.Date(2026, 7, 3, 23, 5, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkRuntimeStoppingV0("async_work_draining", 2, time.Date(2026, 7, 3, 23, 4, 0, 0, time.UTC))

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	runtime.HandlerV0().ServeHTTP(httptest.NewRecorder(), req)
	state := runtime.StateV0()
	public := NewServerPublicStatusV0(state)

	if !state.ShutdownInProgress ||
		state.ShutdownReady ||
		state.ShutdownStatus != "waiting_drain" ||
		state.ShutdownAsyncWorkActive != 2 ||
		public.ShutdownAsyncWorkActive != 2 {
		t.Fatalf("snapshot previo no conservo async_work_active: state=%+v public=%+v", state, public)
	}
}

func TestRuntimeV0ServerShutdownConservaCheckpointAgentsPendingV0(t *testing.T) {
	stateDir := t.TempDir()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"estado":"ok",
			"status":"ready",
			"shutdown_ready":true,
			"runs_requested":1,
			"runs_stopped":1,
			"checkpoint_agents_pending":1
		}`))
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler: app,
		Clock:      fixedClockV0{now: time.Date(2026, 7, 2, 21, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	runtime.HandlerV0().ServeHTTP(httptest.NewRecorder(), req)
	state := runtime.StateV0()
	public := NewServerPublicStatusV0(state)

	if !state.ShutdownInProgress ||
		state.ShutdownReady ||
		state.ShutdownStatus != "stop_pending" ||
		state.ShutdownCheckpointAgentsPending != 1 ||
		public.ShutdownCheckpointAgentsPending != 1 {
		t.Fatalf("checkpoint_agents_pending no conservado: state=%+v public=%+v", state, public)
	}
}

func TestShutdownProjectionFromHTTPV0RechazadoNoPublicaReadyV0(t *testing.T) {
	body := []byte(`{"estado":"error","status":"requester_not_authorized","shutdown_ready":true}`)

	projection, keepFrozen := shutdownProjectionFromHTTPV0(http.StatusForbidden, body)

	if keepFrozen || projection.Ready {
		t.Fatalf("projection=%+v keep_frozen=%v", projection, keepFrozen)
	}
}

func TestShutdownProjectionFromHTTPV0StopPendingNoPublicaReadyV0(t *testing.T) {
	body := []byte(`{"estado":"ok","status":"stop_pending","shutdown_ready":true}`)

	projection, keepFrozen := shutdownProjectionFromHTTPV0(http.StatusOK, body)

	if !keepFrozen || projection.Ready || projection.Status != "stop_pending" {
		t.Fatalf("projection=%+v keep_frozen=%v", projection, keepFrozen)
	}
}

func TestShutdownProjectionFromHTTPV0ConservaActiveWorkRefsV0(t *testing.T) {
	body := []byte(`{
		"estado":"ok",
		"status":"backend_still_running",
		"shutdown_ready":false,
		"active_work_count":1,
		"active_works":[{
			"kind":"goal_backend",
			"run_ref":"run-ref-shutdown-active-001",
			"work_ref":"goal-ref-shutdown-active-001",
			"external_work_ref":"thread-ref-shutdown-active-001",
			"status":"backend_still_running"
		}]
	}`)

	projection, keepFrozen := shutdownProjectionFromHTTPV0(http.StatusConflict, body)

	if keepFrozen ||
		projection.Ready ||
		projection.Status != "backend_still_running" ||
		projection.ActiveWorkCount != 1 ||
		!hasShutdownProjectionRefForTestV0(projection.ActiveWorkRefs, "shutdown-active-work-goal-backend-run-ref-shutdown-active-001") ||
		!hasShutdownProjectionRefForTestV0(projection.ActiveWorkRefs, "shutdown-active-work-goal-backend-goal-ref-shutdown-active-001") ||
		!hasShutdownProjectionRefForTestV0(projection.ActiveWorkRefs, "shutdown-active-work-goal-backend-thread-ref-shutdown-active-001") {
		t.Fatalf("projection=%+v keep_frozen=%v", projection, keepFrozen)
	}
}

func TestShutdownProjectionFromHTTPV0ReadyConActiveWorkQuedaStopPendingV0(t *testing.T) {
	body := []byte(`{
		"estado":"ok",
		"status":"ready",
		"shutdown_ready":true,
		"active_works":[{
			"kind":"goal_backend",
			"run_ref":"run-ref-ready-active-001",
			"work_ref":"goal-ref-ready-active-001",
			"external_work_ref":"thread-ref-ready-active-001",
			"status":"backend_still_running"
		}]
	}`)

	projection, keepFrozen := shutdownProjectionFromHTTPV0(http.StatusOK, body)

	if !keepFrozen ||
		projection.Ready ||
		projection.Status != "stop_pending" ||
		projection.ActiveWorkCount != 1 ||
		!hasShutdownProjectionRefForTestV0(projection.ActiveWorkRefs, "shutdown-active-work-goal-backend-goal-ref-ready-active-001") {
		t.Fatalf("projection=%+v keep_frozen=%v", projection, keepFrozen)
	}
}

func TestShutdownProjectionFromHTTPV0ReadyConActiveWorkRefsQuedaStopPendingV0(t *testing.T) {
	body := []byte(`{
		"estado":"ok",
		"status":"ready",
		"shutdown_ready":true,
		"active_work_refs":[
			"goal-ref-ready-ref-001",
			"shutdown-active-work-goal-backend-thread-ref-ready-ref-001",
			"/tmp/oq-gsrv-secret/socket"
		]
	}`)

	projection, keepFrozen := shutdownProjectionFromHTTPV0(http.StatusOK, body)

	if !keepFrozen ||
		projection.Ready ||
		projection.Status != "stop_pending" ||
		projection.ActiveWorkCount != 3 ||
		!hasShutdownProjectionRefForTestV0(projection.ActiveWorkRefs, "shutdown-active-work-goal-ref-ready-ref-001") ||
		!hasShutdownProjectionRefForTestV0(projection.ActiveWorkRefs, "shutdown-active-work-goal-backend-thread-ref-ready-ref-001") ||
		!hasShutdownProjectionRefForTestV0(projection.ActiveWorkRefs, "shutdown-active-work-ref-redacted") ||
		hasShutdownProjectionRefForTestV0(projection.ActiveWorkRefs, "/tmp/oq-gsrv-secret/socket") {
		t.Fatalf("projection=%+v keep_frozen=%v", projection, keepFrozen)
	}
}

func TestShutdownProjectionFromHTTPV0ReadyConGoalActionsQuedaStopPendingV0(t *testing.T) {
	body := []byte(`{
		"estado":"ok",
		"status":"ready",
		"shutdown_ready":true,
		"goal_actions":[{
			"kind":"goal_backend",
			"work_ref":"goal-ref-action-freeze-001",
			"status":"backend_still_running",
			"action_taken":"cleanup_required",
			"action_evidence_refs":["evidence-ref-shutdown-goal-action-cleanup-required"]
		}]
	}`)

	projection, keepFrozen := shutdownProjectionFromHTTPV0(http.StatusOK, body)

	if !keepFrozen ||
		projection.Ready ||
		projection.Status != "stop_pending" ||
		len(projection.GoalActions) != 1 ||
		projection.GoalActions[0].ActionTaken != "cleanup_required" ||
		len(blockingShutdownGoalActionsV0(projection.GoalActions)) != 1 {
		t.Fatalf("projection=%+v keep_frozen=%v", projection, keepFrozen)
	}
}

func TestServerPublicStatusV0ExponeShutdownGoalActionsV0(t *testing.T) {
	status := NewServerPublicStatusV0(StateV0{
		Status:             "running",
		ShutdownInProgress: true,
		ShutdownStatus:     "backend_still_running",
		ShutdownGoalActions: []ShutdownGoalActionV0{{
			Kind:        "goal_backend",
			WorkRef:     "goal-ref-public-action-001",
			Status:      "backend_still_running",
			ActionTaken: "cleanup_required",
		}},
	})

	if len(status.ShutdownGoalActions) != 1 ||
		status.ShutdownGoalActions[0].ActionTaken != "cleanup_required" ||
		status.ShutdownGoalActions[0].WorkRef != "goal-ref-public-action-001" {
		t.Fatalf("status no expone goal_actions: %+v", status)
	}
}

func TestRuntimeV0ServerShutdownSnapshotPrevioSobreviveRespuestaSinCuerpoV0(t *testing.T) {
	stateDir := t.TempDir()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler: app,
		Clock:      fixedClockV0{now: time.Date(2026, 7, 2, 18, 0, 0, 0, time.UTC)},
		ShutdownSnapshot: fakeShutdownSnapshotPortV0{
			result: ShutdownSnapshotResultV0{
				Status:          "backend_still_running",
				ActiveWorkCount: 1,
				ActiveWorks: []ShutdownSnapshotWorkV0{{
					Kind:            "goal_backend",
					RunRef:          "run-ref-shutdown-snapshot-001",
					WorkRef:         "goal-ref-shutdown-snapshot-001",
					ExternalWorkRef: "thread-ref-shutdown-snapshot-001",
					Status:          "backend_still_running",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	rec := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("shutdown status=%d body=%s", rec.Code, rec.Body.String())
	}
	state := runtime.StateV0()
	if !state.ShutdownInProgress ||
		state.ShutdownReady ||
		state.ShutdownStatus != "backend_still_running" ||
		state.ShutdownActiveWorkCount != 1 ||
		!hasShutdownProjectionRefForTestV0(state.ShutdownActiveWorkRefs, "shutdown-active-work-goal-backend-run-ref-shutdown-snapshot-001") ||
		!hasShutdownProjectionRefForTestV0(state.ShutdownActiveWorkRefs, "shutdown-active-work-goal-backend-goal-ref-shutdown-snapshot-001") {
		t.Fatalf("snapshot previo no sobrevivio: %+v", state)
	}
	store, err := NewFileStateStoreV0(StatePathV0(ConfigV0{StateDir: stateDir}))
	if err != nil {
		t.Fatalf("NewFileStateStoreV0: %v", err)
	}
	persisted, err := store.LoadServerStateV0(context.Background())
	if err != nil {
		t.Fatalf("LoadServerStateV0: %v", err)
	}
	if persisted.ShutdownStatus != state.ShutdownStatus ||
		persisted.ShutdownActiveWorkCount != state.ShutdownActiveWorkCount ||
		!hasShutdownProjectionRefForTestV0(persisted.ShutdownActiveWorkRefs, "shutdown-active-work-goal-backend-goal-ref-shutdown-snapshot-001") {
		t.Fatalf("snapshot previo no persistido: state=%+v persisted=%+v", state, persisted)
	}
}

func TestRuntimeV0ServerShutdownReadyNoBorraSnapshotPrevioActivoV0(t *testing.T) {
	stateDir := t.TempDir()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"estado":"ok","status":"ready","shutdown_ready":true}`))
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler: app,
		Clock:      fixedClockV0{now: time.Date(2026, 7, 3, 23, 10, 0, 0, time.UTC)},
		ShutdownSnapshot: fakeShutdownSnapshotPortV0{
			result: ShutdownSnapshotResultV0{
				Status:          "backend_still_running",
				ActiveWorkCount: 1,
				ActiveWorks: []ShutdownSnapshotWorkV0{{
					Kind:            "goal_backend",
					RunRef:          "run-ref-shutdown-ready-snapshot-001",
					WorkRef:         "goal-ref-shutdown-ready-snapshot-001",
					ExternalWorkRef: "thread-ref-shutdown-ready-snapshot-001",
					Status:          "backend_still_running",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	rec := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(rec, req)

	state := runtime.StateV0()
	if !state.ShutdownInProgress ||
		state.ShutdownReady ||
		state.ShutdownStatus != "stop_pending" ||
		state.ShutdownActiveWorkCount != 1 ||
		!state.SupervisorFrozen ||
		!hasShutdownProjectionRefForTestV0(state.ShutdownActiveWorkRefs, "shutdown-active-work-goal-backend-goal-ref-shutdown-ready-snapshot-001") {
		t.Fatalf("ready falso borro snapshot previo activo: %+v", state)
	}
}

func TestRuntimeV0ServerShutdownReadyTrasCleanupNoHeredaSnapshotPrevioActivoV0(t *testing.T) {
	stateDir := t.TempDir()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"estado":"ok",
			"status":"ready",
			"shutdown_ready":true,
			"runs_requested":0,
			"runs_stopped":0,
			"evidence_refs":[
				"evidence-ref-codex-app-server-tmux-cleanup-requested",
				"evidence-ref-codex-app-server-tmux-configured-cleaned",
				"evidence-ref-shutdown-goal-backend-cleanup-requested"
			]
		}`))
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler: app,
		Clock:      fixedClockV0{now: time.Date(2026, 7, 3, 23, 20, 0, 0, time.UTC)},
		ShutdownSnapshot: fakeShutdownSnapshotPortV0{
			result: ShutdownSnapshotResultV0{
				Status:          "backend_still_running",
				ActiveWorkCount: 1,
				ActiveWorks: []ShutdownSnapshotWorkV0{{
					Kind:            "goal_backend",
					RunRef:          "run-ref-shutdown-cleanup-snapshot-001",
					WorkRef:         "goal-ref-shutdown-cleanup-snapshot-001",
					ExternalWorkRef: "thread-ref-shutdown-cleanup-snapshot-001",
					Status:          "backend_still_running",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	rec := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("shutdown status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverShutdownHTTPProjectionV0
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if !payload.ShutdownReady ||
		!payload.ExitPending ||
		payload.PID != os.Getpid() {
		t.Fatalf("shutdown ready tras cleanup sin salida programada: %+v", payload)
	}
	state := runtime.StateV0()
	if state.ShutdownInProgress ||
		state.SupervisorFrozen ||
		!state.ShutdownReady ||
		state.ShutdownStatus != "ready" ||
		state.ShutdownActiveWorkCount != 0 ||
		len(state.ShutdownActiveWorkRefs) != 0 {
		t.Fatalf("ready tras cleanup heredo snapshot previo activo: %+v", state)
	}
}

func TestRuntimeV0ServerShutdownSnapshotVacioLimpiaTrabajoPrevioYSolicitaSalidaV0(t *testing.T) {
	stateDir := t.TempDir()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"estado":"ok","status":"ready","shutdown_ready":true}`))
	})
	snapshots := &sequenceShutdownSnapshotPortV0{results: []ShutdownSnapshotResultV0{{
		Status:          "backend_still_running",
		ActiveWorkCount: 1,
		ActiveWorks: []ShutdownSnapshotWorkV0{{
			Kind:    "goal_backend",
			WorkRef: "goal-ref-shutdown-sequence-001",
			Status:  "backend_still_running",
		}},
	}, {}}}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler:       app,
		Clock:            fixedClockV0{now: time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)},
		ShutdownSnapshot: snapshots,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	first := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(first, httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil))
	var firstPayload serverShutdownHTTPProjectionV0
	if err := json.Unmarshal(first.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first shutdown response: %v body=%s", err, first.Body.String())
	}
	if firstPayload.ShutdownReady || firstPayload.Status != "stop_pending" || firstPayload.ExitPending {
		t.Fatalf("respuesta con trabajo conservado publica ready: %+v", firstPayload)
	}
	select {
	case <-runtime.shutdownReadyRequested:
		t.Fatal("solicito salida con snapshot activo")
	default:
	}

	second := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(second, httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil))
	var secondPayload serverShutdownHTTPProjectionV0
	if err := json.Unmarshal(second.Body.Bytes(), &secondPayload); err != nil {
		t.Fatalf("decode second shutdown response: %v body=%s", err, second.Body.String())
	}
	if !secondPayload.ShutdownReady || secondPayload.Status != "ready" || !secondPayload.ExitPending || secondPayload.PID != os.Getpid() {
		t.Fatalf("respuesta limpia no solicita salida: %+v", secondPayload)
	}
	state := runtime.StateV0()
	if state.ShutdownInProgress || state.SupervisorFrozen || !state.ShutdownReady ||
		state.ShutdownActiveWorkCount != 0 || len(state.ShutdownActiveWorkRefs) != 0 {
		t.Fatalf("snapshot vacio no limpio shutdown: %+v", state)
	}
	select {
	case <-runtime.shutdownReadyRequested:
	default:
		t.Fatal("shutdown limpio no solicito salida")
	}
}

func TestRuntimeV0ServerShutdownSnapshotVacioRetiraAccionBloqueantePreviaV0(t *testing.T) {
	stateDir := t.TempDir()
	responses := [][]byte{
		[]byte(`{
			"estado":"ok",
			"status":"backend_still_running",
			"shutdown_ready":false,
			"active_work_count":1,
			"goal_actions":[{
				"kind":"goal_backend",
				"work_ref":"goal-ref-shutdown-action-sequence-001",
				"status":"backend_still_running",
				"action_taken":"cleanup_required"
			}]
		}`),
		[]byte(`{"estado":"ok","status":"ready","shutdown_ready":true}`),
	}
	call := 0
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		index := call
		if index >= len(responses) {
			index = len(responses) - 1
		}
		call++
		_, _ = w.Write(responses[index])
	})
	snapshots := &sequenceShutdownSnapshotPortV0{results: []ShutdownSnapshotResultV0{{
		Status:          "backend_still_running",
		ActiveWorkCount: 1,
		ActiveWorks: []ShutdownSnapshotWorkV0{{
			Kind:    "goal_backend",
			WorkRef: "goal-ref-shutdown-action-sequence-001",
			Status:  "backend_still_running",
		}},
	}, {}}}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler:       app,
		Clock:            fixedClockV0{now: time.Date(2026, 7, 11, 10, 30, 0, 0, time.UTC)},
		ShutdownSnapshot: snapshots,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	first := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(first, httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil))
	if state := runtime.StateV0(); len(state.ShutdownGoalActions) != 1 || state.ShutdownReady {
		t.Fatalf("primera accion bloqueante no persistida: %+v", state)
	}

	second := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(second, httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil))
	var payload serverShutdownHTTPProjectionV0
	if err := json.Unmarshal(second.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode second shutdown response: %v body=%s", err, second.Body.String())
	}
	if !payload.ShutdownReady || payload.Status != "ready" || !payload.ExitPending || len(payload.GoalActions) != 0 {
		t.Fatalf("snapshot vacio heredo accion previa: %+v", payload)
	}
	state := runtime.StateV0()
	if state.ShutdownInProgress || state.SupervisorFrozen || !state.ShutdownReady || len(state.ShutdownGoalActions) != 0 {
		t.Fatalf("estado final conserva accion previa: %+v", state)
	}
}

func TestRuntimeV0ServerShutdownNuevoIntentoRetiraTimeoutYAsyncPreviosV0(t *testing.T) {
	stateDir := t.TempDir()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"estado":"ok","status":"ready","shutdown_ready":true}`))
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler:       app,
		Clock:            fixedClockV0{now: time.Date(2026, 7, 11, 10, 45, 0, 0, time.UTC)},
		ShutdownSnapshot: fakeShutdownSnapshotPortV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.persistStateTransitionV0(
		context.Background(),
		runtime.tracker.MarkRuntimeStopTimeoutV0("previous_stop_timeout", 2, time.Date(2026, 7, 11, 10, 40, 0, 0, time.UTC)),
		"test_previous_stop_timeout",
	)
	if previous := runtime.StateV0(); previous.ShutdownAsyncWorkActive != 2 || previous.ShutdownStopTimeoutAt == "" {
		t.Fatalf("fixture timeout previo invalido: %+v", previous)
	}

	recorder := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil))
	var payload serverShutdownHTTPProjectionV0
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode shutdown response: %v body=%s", err, recorder.Body.String())
	}
	state := runtime.StateV0()
	if !payload.ShutdownReady || !payload.ExitPending || payload.Status != "ready" ||
		state.ShutdownInProgress || !state.ShutdownReady ||
		state.ShutdownAsyncWorkActive != 0 || state.ShutdownStopTimeoutAt != "" {
		t.Fatalf("nuevo intento heredo timeout/async previos: payload=%+v state=%+v", payload, state)
	}
}

func TestRuntimeV0ServerShutdownConflictSinCuerpoConservaSnapshotPrevioActivoV0(t *testing.T) {
	stateDir := t.TempDir()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != serverShutdownRoutePathV0 {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusConflict)
	})
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		AppHandler: app,
		Clock:      fixedClockV0{now: time.Date(2026, 7, 3, 23, 15, 0, 0, time.UTC)},
		ShutdownSnapshot: fakeShutdownSnapshotPortV0{
			result: ShutdownSnapshotResultV0{
				Status:          "backend_still_running",
				ActiveWorkCount: 1,
				ActiveWorks: []ShutdownSnapshotWorkV0{{
					Kind:            "goal_backend",
					RunRef:          "run-ref-shutdown-conflict-snapshot-001",
					WorkRef:         "goal-ref-shutdown-conflict-snapshot-001",
					ExternalWorkRef: "thread-ref-shutdown-conflict-snapshot-001",
					Status:          "backend_still_running",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, serverShutdownRoutePathV0, nil)
	rec := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(rec, req)

	state := runtime.StateV0()
	if rec.Code != http.StatusConflict ||
		!state.ShutdownInProgress ||
		state.ShutdownReady ||
		state.ShutdownStatus != "backend_still_running" ||
		state.ShutdownActiveWorkCount != 1 ||
		!state.SupervisorFrozen ||
		!hasShutdownProjectionRefForTestV0(state.ShutdownActiveWorkRefs, "shutdown-active-work-goal-backend-goal-ref-shutdown-conflict-snapshot-001") {
		t.Fatalf("conflict sin cuerpo no conservo snapshot previo activo: code=%d state=%+v", rec.Code, state)
	}
}

type countingShutdownFreezeSupervisorV0 struct {
	calls int
}

func (supervisor *countingShutdownFreezeSupervisorV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	supervisor.calls++
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func hasShutdownProjectionRefForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type notifyingRuntimeShutdownHookV0 struct {
	called chan struct{}
	calls  int32
}

func newNotifyingRuntimeShutdownHookV0() *notifyingRuntimeShutdownHookV0 {
	return &notifyingRuntimeShutdownHookV0{called: make(chan struct{})}
}

func (hook *notifyingRuntimeShutdownHookV0) ShutdownV0(context.Context) error {
	if atomic.AddInt32(&hook.calls, 1) == 1 {
		close(hook.called)
	}
	return nil
}

type fakeShutdownSnapshotPortV0 struct {
	result ShutdownSnapshotResultV0
	err    error
}

type sequenceShutdownSnapshotPortV0 struct {
	results []ShutdownSnapshotResultV0
	next    int
}

func (port *sequenceShutdownSnapshotPortV0) SnapshotShutdownV0(
	context.Context,
	ShutdownSnapshotRequestV0,
) (ShutdownSnapshotResultV0, error) {
	if port.next >= len(port.results) {
		return ShutdownSnapshotResultV0{}, nil
	}
	result := port.results[port.next]
	port.next++
	return result, nil
}

func (fake fakeShutdownSnapshotPortV0) SnapshotShutdownV0(
	context.Context,
	ShutdownSnapshotRequestV0,
) (ShutdownSnapshotResultV0, error) {
	return fake.result, fake.err
}

type recordingForceExitPortV0 struct {
	codes chan int
}

func (port *recordingForceExitPortV0) ExitV0(code int) {
	port.codes <- code
}

func waitRuntimeAddrV0(t *testing.T, runtime *RuntimeV0) string {
	t.Helper()
	deadline := time.After(time.Second)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout esperando addr runtime: %+v", runtime.StateV0())
		case <-tick.C:
			if state := runtime.StateV0(); state.Status == "running" && state.Addr != "" {
				return state.Addr
			}
		}
	}
}
