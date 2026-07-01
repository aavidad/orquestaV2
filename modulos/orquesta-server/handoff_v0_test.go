package orquestaserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0ServerHandoffCongelaSinInvocarShutdownDeRunsV0(t *testing.T) {
	supervisor := &countingShutdownFreezeSupervisorV0{}
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == serverShutdownRoutePathV0 {
			t.Fatalf("handoff no debe delegar en shutdown")
		}
		http.NotFound(w, r)
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
		Clock:      fixedClockV0{now: time.Date(2026, 5, 27, 1, 10, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, serverHandoffRoutePathV0, nil)
	runtime.HandlerV0().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("handoff status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverHandoffHTTPProjectionV0
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("handoff json: %v", err)
	}
	if !payload.HandoffReady || !payload.AgentsPreserved || payload.Status != "handoff_requested" {
		t.Fatalf("handoff payload=%+v", payload)
	}
	state := runtime.StateV0()
	if !state.ShutdownInProgress ||
		!state.SupervisorFrozen ||
		state.ShutdownStatus != "handoff_requested" ||
		state.Status != "handoff_requested" {
		t.Fatalf("state handoff=%+v", state)
	}

	runtime.runSupervisorTickV0(context.Background())
	if supervisor.calls != 0 {
		t.Fatalf("supervisor ejecutado durante handoff: calls=%d", supervisor.calls)
	}
}

func TestRuntimeV0ServerHandoffCierraSoloServidorV0(t *testing.T) {
	requireLocalTCPForServerTestV0(t)
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:                "127.0.0.1:0",
		StateDir:            t.TempDir(),
		TickInterval:        time.Hour,
		ShutdownGracePeriod: 2 * time.Second,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef:      "global",
			MaxExecutions: 10,
		},
	}, RuntimeDepsV0{
		AppHandler: http.NotFoundHandler(),
		Supervisor: &countingShutdownFreezeSupervisorV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(ctx) }()
	addr := waitRuntimeAddrForTestV0(t, runtime)

	req, err := http.NewRequest(http.MethodPost, "http://"+addr+serverHandoffRoutePathV0, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("handoff post: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("handoff status=%d", resp.StatusCode)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunV0 handoff err=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("runtime no cerro tras handoff")
	}
	state := runtime.StateV0()
	if state.Status != "handoff_ready" ||
		state.ShutdownStatus != "handoff_ready" ||
		state.ShutdownInProgress ||
		!state.ShutdownReady {
		t.Fatalf("state final handoff=%+v", state)
	}
}

func waitRuntimeAddrForTestV0(t *testing.T, runtime *RuntimeV0) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		addr := strings.TrimSpace(runtime.StateV0().Addr)
		if addr != "" && !strings.HasSuffix(addr, ":0") {
			return addr
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runtime sin addr")
	return ""
}
