package orquestaserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shutdown status=%d", resp.StatusCode)
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

func (fake fakeShutdownSnapshotPortV0) SnapshotShutdownV0(
	context.Context,
	ShutdownSnapshotRequestV0,
) (ShutdownSnapshotResultV0, error) {
	return fake.result, fake.err
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
