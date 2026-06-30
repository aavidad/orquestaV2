package main

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerReadinessOKV0UsaReadinessNoHealthzV0(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
		case orquestaserver.ServerReadinessEndpointV0:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	if serverReadinessOKV0(strings.TrimPrefix(server.URL, "http://")) {
		t.Fatalf("readiness false no debe pasar aunque healthz este vivo")
	}
}

func TestWaitForStateHealthyV0AceptaReadinessEstableV0(t *testing.T) {
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	config := daemonReadinessConfigForTestV0(t)
	state := daemonReadinessStateForTestV0(strings.TrimPrefix(server.URL, "http://"), 1234)
	saveDaemonReadinessStateForTestV0(t, config, state)
	restore := setDaemonReadinessTimingsForTestV0(5*time.Millisecond, 5*time.Millisecond)
	defer restore()

	got, err := waitForStateHealthyV0(config, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("waitForStateHealthyV0: %v", err)
	}
	if got.PID != state.PID || got.ProcessRef != state.ProcessRef {
		t.Fatalf("state estable inesperado: %+v", got)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("readiness estable debe requerir al menos dos consultas, calls=%d", calls)
	}
}

func TestWaitForStateHealthyV0RechazaReadinessInestableTrasReadyV0(t *testing.T) {
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	config := daemonReadinessConfigForTestV0(t)
	state := daemonReadinessStateForTestV0(strings.TrimPrefix(server.URL, "http://"), 1234)
	saveDaemonReadinessStateForTestV0(t, config, state)
	restore := setDaemonReadinessTimingsForTestV0(5*time.Millisecond, 5*time.Millisecond)
	defer restore()

	_, err := waitForStateHealthyV0(config, 80*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "readiness_timeout") {
		t.Fatalf("readiness inestable debe expirar, err=%v", err)
	}
}

func TestWaitForStateHealthyV0RechazaCambioDeIdentidadTrasReadyV0(t *testing.T) {
	config := daemonReadinessConfigForTestV0(t)
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		call := atomic.AddInt32(&calls, 1)
		next := daemonReadinessStateForTestV0(r.Host, 2000+int(call))
		next.Addr = r.Host
		saveDaemonReadinessStateForTestV0(t, config, next)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	state := daemonReadinessStateForTestV0(strings.TrimPrefix(server.URL, "http://"), 1234)
	saveDaemonReadinessStateForTestV0(t, config, state)
	restore := setDaemonReadinessTimingsForTestV0(5*time.Millisecond, 5*time.Millisecond)
	defer restore()

	_, err := waitForStateHealthyV0(config, 80*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "readiness_timeout") {
		t.Fatalf("cambio de identidad debe expirar, err=%v", err)
	}
}

func daemonReadinessConfigForTestV0(t *testing.T) orquestaserver.ConfigV0 {
	t.Helper()
	return orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		StateDir:  t.TempDir(),
		StateFile: orquestaserver.DefaultStateFileV0,
	})
}

func daemonReadinessStateForTestV0(addr string, pid int) orquestaserver.StateV0 {
	return orquestaserver.StateV0{
		Status:         "running",
		PID:            pid,
		Addr:           addr,
		ProcessRef:     "process-ref-startup-test",
		DaemonEpochRef: "daemon-epoch-startup-test",
		StartedAt:      "2026-06-30T10:00:00Z",
		StartupStatus:  "ready",
		StartupReady:   true,
		RuntimeIdentity: orquestaserver.ServerRuntimeIdentityV0{
			BinarySHA256: "sha256-startup-test",
			BuildRef:     "build-ref-startup-test",
			CommitRef:    "commit-ref-startup-test",
			StartedAt:    "2026-06-30T10:00:00Z",
		},
	}
}

func saveDaemonReadinessStateForTestV0(
	t *testing.T,
	config orquestaserver.ConfigV0,
	state orquestaserver.StateV0,
) {
	t.Helper()
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err != nil {
		t.Fatalf("NewFileStateStoreV0: %v", err)
	}
	if err := store.SaveServerStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveServerStateV0: %v", err)
	}
}

func setDaemonReadinessTimingsForTestV0(poll time.Duration, stable time.Duration) func() {
	previousPoll := serverReadinessPollEveryV0
	previousStable := serverReadinessStableDelayV0
	serverReadinessPollEveryV0 = poll
	serverReadinessStableDelayV0 = stable
	return func() {
		serverReadinessPollEveryV0 = previousPoll
		serverReadinessStableDelayV0 = previousStable
	}
}
