package orquestaserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRuntimeV0PersistStateFailureVisibleSinFiltrarDetallesV0(t *testing.T) {
	store := &failingStateStoreV0{err: errors.New("write failed at /tmp/runtime-secret/state.json")}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		AuditDisabled: true,
	}, RuntimeDepsV0{
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	if err := runtime.prepareStartupV0(context.Background()); err != nil {
		t.Fatalf("prepareStartupV0: %v", err)
	}

	state := runtime.StateV0()
	if state.StatePersistStatus != "degraded" ||
		state.StatePersistFailures != 1 ||
		state.StatePersistLastCode != "state_persist_failed" ||
		state.StatePersistLastTransition != "startup_ready" {
		t.Fatalf("state persist projection=%+v", state)
	}
	if state.LastError != "" {
		t.Fatalf("last_error must not turn persist failure into raw server error: %q", state.LastError)
	}
	if len(state.RecentErrors) == 0 ||
		state.RecentErrors[0].Code != "state_persist_failed" ||
		state.RecentErrors[0].Message != "state_persist_failed" {
		t.Fatalf("recent_errors=%+v", state.RecentErrors)
	}
	body, _ := json.Marshal(state)
	if strings.Contains(string(body), "runtime-secret") ||
		strings.Contains(string(body), "state.json") {
		t.Fatalf("state leaks store error detail: %s", string(body))
	}
}

func TestRuntimeV0PersistStateConfirmedRecuperaEstadoDegradadoV0(t *testing.T) {
	store := &failingStateStoreV0{err: errors.New("write failed")}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		AuditDisabled: true,
	}, RuntimeDepsV0{
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.persistStateTransitionV0(context.Background(), runtime.StateV0(), "startup_ready")
	store.err = nil

	runtime.persistStateTransitionV0(context.Background(), runtime.StateV0(), "supervisor_tick")

	state := runtime.StateV0()
	if state.StatePersistStatus != "ok" ||
		state.StatePersistFailures != 1 ||
		state.StatePersistLastConfirmed == "" ||
		state.StatePersistLastTransition != "supervisor_tick" {
		t.Fatalf("state persist recovered=%+v", state)
	}
}

type failingStateStoreV0 struct {
	err  error
	last StateV0
}

func (store *failingStateStoreV0) SaveServerStateV0(_ context.Context, state StateV0) error {
	if store.err != nil {
		return store.err
	}
	store.last = state
	return nil
}

func (store *failingStateStoreV0) LoadServerStateV0(context.Context) (StateV0, error) {
	return store.last, store.err
}
