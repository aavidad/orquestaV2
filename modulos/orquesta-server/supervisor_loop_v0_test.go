package orquestaserver

import (
	"context"
	"errors"
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
	if supervisor.lastCommand.MaxTicks != DefaultSupervisorMaxTicksV0 {
		t.Fatalf("max_ticks=%d want=%d", supervisor.lastCommand.MaxTicks, DefaultSupervisorMaxTicksV0)
	}
	if store.last.SupervisorTicks != 1 ||
		store.last.LastSupervisorStatus != "ok" ||
		store.last.LastSupervisorStop != orquestarunsupervisor.RunSupervisorStopNoExecutionV0 ||
		store.last.LastSupervisorQueueRef != "" {
		t.Fatalf("state=%+v", store.last)
	}
}

func TestRuntimeV0SupervisorRespetaComandoConfiguradoV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			MaxTicks:          5,
			MaxExecutions:     3,
			AllowRepeatedRuns: true,
		},
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.lastCommand.MaxTicks != 5 ||
		supervisor.lastCommand.MaxExecutions != 3 ||
		!supervisor.lastCommand.AllowRepeatedRuns {
		t.Fatalf("command=%+v", supervisor.lastCommand)
	}
	if store.last.LastSupervisorExecutions != 1 ||
		store.last.LastSupervisorSkips != 2 ||
		store.last.LastSupervisorResultTicks != 1 {
		t.Fatalf("state=%+v", store.last)
	}
}

func TestRuntimeV0SupervisorRegistraErrorYPermiteSiguienteTickV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{
			result: orquestarunsupervisor.RunSupervisorResultV0{
				StopReason:      orquestarunsupervisor.RunSupervisorStopTickErrorV0,
				TotalExecutions: 1,
				TotalSkips:      1,
			},
			err: errors.New("fallo_transitorio"),
		}, {
			result: orquestarunsupervisor.RunSupervisorResultV0{
				StopReason:      orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
				TotalExecutions: 2,
			},
		}},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef: "global",
		},
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runSupervisorTickV0(context.Background())

	if store.last.LastSupervisorStatus != "error" ||
		store.last.LastSupervisorError != "fallo_transitorio" ||
		store.last.SupervisorTicks != 1 ||
		store.last.SupervisorErrorTicks != 1 ||
		store.last.LastSupervisorQueueRef != "global" {
		t.Fatalf("state error=%+v", store.last)
	}

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.calls != 2 ||
		store.last.LastSupervisorStatus != "ok" ||
		store.last.LastError != "" ||
		store.last.LastSupervisorError != "" ||
		store.last.LastSupervisorExecutions != 2 ||
		store.last.SupervisorTicks != 2 ||
		store.last.SupervisorErrorTicks != 1 {
		t.Fatalf("state recovered=%+v calls=%d", store.last, supervisor.calls)
	}
}

func TestRuntimeV0SupervisorAsyncNoBloqueaTickResidenteV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := newBlockingSupervisorV0()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	if !runtime.runSupervisorTickAsyncV0(context.Background()) {
		t.Fatalf("first async tick not started")
	}
	<-supervisor.started
	if runtime.runSupervisorTickAsyncV0(context.Background()) {
		t.Fatalf("second async tick overlapped while first in flight")
	}
	supervisor.release()
	supervisor.waitDone(t)
	if !runtime.runSupervisorTickAsyncV0(context.Background()) {
		t.Fatalf("async tick did not restart after first completed")
	}
	supervisor.release()
	supervisor.waitDone(t)
}

type fakeSupervisorV0 struct {
	calls       int
	lastCommand orquestarunsupervisor.RunSupervisorCommandV0
	results     []fakeSupervisorResultV0
}

type blockingSupervisorV0 struct {
	started   chan struct{}
	releaseCh chan struct{}
	done      chan struct{}
	calls     int
}

func newBlockingSupervisorV0() *blockingSupervisorV0 {
	return &blockingSupervisorV0{
		started:   make(chan struct{}, 2),
		releaseCh: make(chan struct{}, 2),
		done:      make(chan struct{}, 2),
	}
}

func (fake *blockingSupervisorV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	fake.calls++
	fake.started <- struct{}{}
	<-fake.releaseCh
	fake.done <- struct{}{}
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func (fake *blockingSupervisorV0) release() {
	fake.releaseCh <- struct{}{}
}

func (fake *blockingSupervisorV0) waitDone(t *testing.T) {
	t.Helper()
	select {
	case <-fake.done:
	case <-time.After(time.Second):
		t.Fatalf("blocking supervisor did not finish")
	}
}

type fakeSupervisorResultV0 struct {
	result orquestarunsupervisor.RunSupervisorResultV0
	err    error
}

func (fake *fakeSupervisorV0) RunGlobalSupervisorV0(
	_ context.Context,
	command orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	fake.calls++
	fake.lastCommand = command
	if len(fake.results) >= fake.calls {
		next := fake.results[fake.calls-1]
		return next.result, next.err
	}
	return orquestarunsupervisor.RunSupervisorResultV0{
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
		}},
		TotalExecutions: 1,
		TotalSkips:      2,
		StopReason:      orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
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
