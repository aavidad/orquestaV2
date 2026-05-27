package orquestaserver

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0SupervisorAsyncCoalesceaUnTickPendienteV0(t *testing.T) {
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
	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatalf("pending async tick was not coalesced after first completed")
	}
	if runtime.runSupervisorTickAsyncV0(context.Background()) {
		t.Fatalf("third async tick overlapped while coalesced tick in flight")
	}
	supervisor.release()
	supervisor.waitDone(t)
	time.Sleep(10 * time.Millisecond)
	if store.last.SupervisorTickActive {
		t.Fatalf("supervisor tick stayed active after coalesced pulse: %+v", store.last)
	}
}
func TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0(t *testing.T) {
	now := time.Date(2026, 5, 23, 10, 2, 0, 0, time.UTC)
	noExecution := fakeSupervisorResultV0{result: orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}}
	supervisor := &fakeSupervisorV0{results: []fakeSupervisorResultV0{{
		result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Skips: []orquestaruncoordinator.RunSkipSummaryV0{{
						RunRef: "run-ref-autoprogramming-pending-001", Rank: 1, Reason: "pending_work",
					}},
				},
			}},
		},
	}, noExecution, noExecution, noExecution}, selfStarted: make(chan struct{}, 1), selfRelease: make(chan struct{})}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 1,
		IdleSelfImprovementTargetQueue: 1,
		AuditDisabled:                  true,
	}, RuntimeDepsV0{
		Supervisor: supervisor, StateStore: &memoryStateStoreV0{}, Clock: fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)
	runtime.runSupervisorTickV0(context.Background())
	if supervisor.selfCalls != 0 {
		t.Fatalf("self improvement started with pending primary work")
	}
	markNoExecutionSinceForTestV0(runtime, now)
	if !runtime.runSupervisorTickAsyncV0(context.Background()) {
		t.Fatalf("async tick not started")
	}
	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("self improvement not started")
	}
	waitSupervisorTickInactiveForTestV0(t, runtime)
	if !runtime.runSupervisorTickAsyncV0(context.Background()) {
		t.Fatalf("idle self improvement blocked supervisor tick")
	}
	close(supervisor.selfRelease)
	waitIdleSelfImprovementFinishedForTestV0(t, runtime)
	if supervisor.selfCalls != 1 ||
		!strings.Contains(supervisor.lastSelfRequest.FailureSummary, "1m0s") ||
		supervisor.lastSelfRequest.PriorityScore != DefaultIdleSelfImprovementPriorityScoreV0 ||
		!strings.Contains(strings.Join(supervisor.lastSelfRequest.EvidenceRefs, ","), "idle-no-execution") {
		t.Fatalf("self_calls=%d request=%+v", supervisor.selfCalls, supervisor.lastSelfRequest)
	}
	runtime.tracker.MarkIdleSelfImprovementErrorV0("fallo_transitorio", now.Add(-61*time.Second))
	runtime.runSupervisorTickV0(context.Background())
	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("retry no lanzado tras cooldown")
	}
}

func waitIdleSelfImprovementFinishedForTestV0(t *testing.T, runtime *RuntimeV0) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		_, _, inFlight, _ := runtime.tracker.IdleSelfImprovementWindowV0()
		if !inFlight {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("idle self improvement sigue en vuelo")
}

func waitSupervisorTickInactiveForTestV0(t *testing.T, runtime *RuntimeV0) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if runtime.tracker.SnapshotV0().SupervisorTickActive {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		return
	}
	t.Fatalf("supervisor tick sigue activo")
}
