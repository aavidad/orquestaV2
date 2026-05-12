package orquestaserver

import (
	"context"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0SupervisorPersisteTicksV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.calls != 1 {
		t.Fatalf("calls=%d", supervisor.calls)
	}
	if store.last.SupervisorTicks != 1 ||
		store.last.LastSupervisorStop != orquestarunsupervisor.RunSupervisorStopNoExecutionV0 {
		t.Fatalf("state=%+v", store.last)
	}
}

type fakeSupervisorV0 struct {
	calls int
}

func (fake *fakeSupervisorV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	fake.calls++
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

type memoryStateStoreV0 struct {
	last StateV0
}

func (store *memoryStateStoreV0) SaveServerStateV0(_ context.Context, state StateV0) error {
	store.last = state
	return nil
}

func (store *memoryStateStoreV0) LoadServerStateV0(context.Context) (StateV0, error) {
	return store.last, nil
}

type fixedClockV0 struct {
	now time.Time
}

func (clock fixedClockV0) Now() time.Time {
	return clock.now
}
