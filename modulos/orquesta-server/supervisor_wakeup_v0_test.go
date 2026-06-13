package orquestaserver

import (
	"context"
	"testing"
	"time"
)

func TestRuntimeV0SupervisorLoopDespiertaPorWakeupSinTickerV0(t *testing.T) {
	supervisor := newBlockingSupervisorV0()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: time.Date(2026, 6, 13, 1, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runtime.runSupervisorLoopV0(ctx)

	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatalf("tick inicial no arrancado")
	}
	supervisor.release()
	supervisor.waitDone(t)

	if !runtime.RequestSupervisorWakeupV0("queue_ready") {
		t.Fatalf("wakeup no aceptado")
	}
	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatalf("wakeup no disparo tick sin ticker")
	}
	supervisor.release()
	supervisor.waitDone(t)
}
