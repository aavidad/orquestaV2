package orquestaserver

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0ShutdownEsperaPreparacionIdleAntesDeStoppedV0(t *testing.T) {
	requireLocalTCPForServerTestV0(t)
	now := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	store := &threadSafeStateStoreV0{}
	supervisor := newBlockingIdlePrepareSupervisorV0()
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:                        "127.0.0.1:0",
		StateDir:                    t.TempDir(),
		AuditDisabled:               true,
		TickInterval:                time.Hour,
		ShutdownGracePeriod:         500 * time.Millisecond,
		IdleSelfImprovementAfter:    time.Millisecond,
		IdleSelfImprovementWriteSet: []string{"modulos/orquesta-server"},
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(ctx) }()
	waitForRuntimeTestV0(t, supervisor.prepareStarted)

	cancel()
	select {
	case err := <-done:
		t.Fatalf("runtime paro antes de drenar preparacion idle: %v state=%+v", err, runtime.StateV0())
	case <-time.After(40 * time.Millisecond):
	}
	close(supervisor.releasePrepare)

	if err := waitRuntimeDoneV0(t, done); err != nil {
		t.Fatalf("RunV0: %v", err)
	}
	state := runtime.StateV0()
	if state.Status != "stopped" ||
		state.ShutdownStatus != "stopped" ||
		state.ShutdownAsyncWorkActive != 0 ||
		state.IdleSelfImprovementFlight {
		t.Fatalf("shutdown final sin quiescencia: %+v", state)
	}
	if store.LastV0().Status != "stopped" {
		t.Fatalf("estado durable no quedo stopped: %+v", store.LastV0())
	}
}

func TestRuntimeV0ShutdownTimeoutPublicaStopTimeoutV0(t *testing.T) {
	requireLocalTCPForServerTestV0(t)
	now := time.Date(2026, 5, 26, 10, 5, 0, 0, time.UTC)
	store := &threadSafeStateStoreV0{}
	supervisor := newBlockingIdlePrepareSupervisorV0()
	defer close(supervisor.releasePrepare)
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:                        "127.0.0.1:0",
		StateDir:                    t.TempDir(),
		AuditDisabled:               true,
		TickInterval:                time.Hour,
		ShutdownGracePeriod:         30 * time.Millisecond,
		IdleSelfImprovementAfter:    time.Millisecond,
		IdleSelfImprovementWriteSet: []string{"modulos/orquesta-server"},
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(ctx) }()
	waitForRuntimeTestV0(t, supervisor.prepareStarted)

	cancel()
	err = waitRuntimeDoneV0(t, done)
	if err == nil || !strings.Contains(err.Error(), "async_work_timeout") {
		t.Fatalf("RunV0 err=%v", err)
	}
	state := runtime.StateV0()
	if state.Status != "stop_timeout" ||
		state.ShutdownStatus != "stop_timeout" ||
		state.ShutdownAsyncWorkActive == 0 ||
		state.ShutdownStopTimeoutAt == "" {
		t.Fatalf("timeout no visible: %+v", state)
	}
}

type blockingIdlePrepareSupervisorV0 struct {
	prepareStarted chan struct{}
	releasePrepare chan struct{}
	once           sync.Once
}

func newBlockingIdlePrepareSupervisorV0() *blockingIdlePrepareSupervisorV0 {
	return &blockingIdlePrepareSupervisorV0{
		prepareStarted: make(chan struct{}),
		releasePrepare: make(chan struct{}),
	}
}

func (supervisor *blockingIdlePrepareSupervisorV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func (supervisor *blockingIdlePrepareSupervisorV0) PrepareIdleSelfImprovementV0(
	context.Context,
	IdleSelfImprovementRequestV0,
) (IdleSelfImprovementResultV0, error) {
	supervisor.once.Do(func() { close(supervisor.prepareStarted) })
	<-supervisor.releasePrepare
	return IdleSelfImprovementResultV0{
		Accepted:   true,
		RunRef:     "run-ref-idle-shutdown-test",
		RequestRef: "request-ref-idle-shutdown-test",
		Status:     "ok",
	}, nil
}

type threadSafeStateStoreV0 struct {
	mu   sync.Mutex
	last StateV0
}

func (store *threadSafeStateStoreV0) SaveServerStateV0(_ context.Context, state StateV0) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.last = state
	return nil
}

func (store *threadSafeStateStoreV0) LoadServerStateV0(context.Context) (StateV0, error) {
	return store.LastV0(), nil
}

func (store *threadSafeStateStoreV0) LastV0() StateV0 {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.last
}

func waitForRuntimeTestV0(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timeout esperando evento runtime")
	}
}

func waitRuntimeDoneV0(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal("timeout esperando RunV0")
		return nil
	}
}
