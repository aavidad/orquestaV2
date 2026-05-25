package orquestaserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			"estado": "error",
			"status": "requester_not_authorized",
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
	if runtime.StateV0().SupervisorFrozen {
		t.Fatalf("shutdown rechazado congelo supervisor: %+v", runtime.StateV0())
	}

	runtime.runSupervisorTickV0(context.Background())
	if supervisor.calls != 1 {
		t.Fatalf("supervisor no recuperado tras rechazo: calls=%d", supervisor.calls)
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
