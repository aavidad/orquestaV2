package orquestaserver

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0SupervisorRecuperaPanicYExponeMetricasV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &panicThenMetricsSupervisorV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef: "global",
		},
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 23, 11, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.runSupervisorTickV0(context.Background())
	if store.snapshotV0().LastSupervisorStatus != "error" ||
		!strings.Contains(store.snapshotV0().LastSupervisorError, "panic:tick roto") ||
		!strings.Contains(store.snapshotV0().SupervisorLastError, "panic:tick roto") ||
		store.snapshotV0().SupervisorLastErrorAt == "" ||
		store.snapshotV0().SupervisorErrorTicks != 1 {
		t.Fatalf("state panic=%+v", store.snapshotV0())
	}
	runtime.runSupervisorTickV0(context.Background())
	if store.snapshotV0().LastSupervisorStatus != "ok" ||
		!strings.Contains(store.snapshotV0().SupervisorLastError, "panic:tick roto") ||
		store.snapshotV0().LastSupervisorQueueSize != 3 ||
		store.snapshotV0().LastSupervisorTickNumber != 2 ||
		store.snapshotV0().LastSupervisorExecutions != 1 ||
		store.snapshotV0().LastSupervisorSkips != 2 ||
		store.snapshotV0().SupervisorExecutions != 1 ||
		store.snapshotV0().SupervisorSkips != 2 ||
		store.snapshotV0().SupervisorTicks != 2 {
		t.Fatalf("state recovered=%+v", store.snapshotV0())
	}
}

type panicThenMetricsSupervisorV0 struct {
	calls int
}

func (supervisor *panicThenMetricsSupervisorV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	supervisor.calls++
	if supervisor.calls == 1 {
		panic("tick roto")
	}
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxTicksV0,
		TotalExecutions: 1,
		TotalSkips:      2,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{RunRef: "run-a"}},
			},
		}, {
			TickNumber: 2,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{RunRef: "run-a"}},
				Skips: []orquestaruncoordinator.RunSkipSummaryV0{
					{RunRef: "run-b", Reason: "blocked"},
					{RunRef: "run-c", Reason: "excluded"},
				},
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{
					{RunRef: "run-a"},
					{RunRef: "run-b"},
					{RunRef: "run-c"},
				},
			},
		}},
	}, nil
}
