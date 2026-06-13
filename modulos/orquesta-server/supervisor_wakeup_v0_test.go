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

func TestRuntimeV0ResidentDirectorLoopDespiertaPorWakeupSinTickerV0(t *testing.T) {
	director := newBlockingResidentDirectorV0()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                t.TempDir(),
		TickInterval:            time.Hour,
		ResidentDirectorEnabled: true,
		AuditDisabled:           true,
	}, RuntimeDepsV0{
		ResidentDirector: director,
		StateStore:       &memoryStateStoreV0{},
		Clock:            fixedClockV0{now: time.Date(2026, 6, 13, 1, 30, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runtime.runResidentDirectorLoopV0(ctx)

	select {
	case <-director.started:
	case <-time.After(time.Second):
		t.Fatalf("tick inicial del director residente no arrancado")
	}
	director.release()
	director.waitDone(t)

	if !runtime.RequestResidentDirectorWakeupV0("run_store_saved") {
		t.Fatalf("wakeup residente no aceptado")
	}
	select {
	case <-director.started:
	case <-time.After(time.Second):
		t.Fatalf("wakeup residente no disparo tick sin ticker")
	}
	director.release()
	director.waitDone(t)
}

func TestRuntimeV0ResidentDirectorWakeupRequiereOptInV0(t *testing.T) {
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		ResidentDirector: &fakeResidentDirectorV0{},
		StateStore:       &memoryStateStoreV0{},
		Clock:            fixedClockV0{now: time.Date(2026, 6, 13, 1, 45, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	if runtime.RequestResidentDirectorWakeupV0("run_store_saved") {
		t.Fatalf("wakeup residente aceptado sin opt-in")
	}
}
