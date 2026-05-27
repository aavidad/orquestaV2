package orquestaserver

import (
	"context"
	"errors"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0SupervisorPlannerErrorNoUsaFallbackGenericoV0(t *testing.T) {
	now := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
		}}},
		planErr:     errors.New("planner_degraded"),
		selfStarted: make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 1,
		AuditDisabled:                  true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)

	runtime.runSupervisorTickV0(context.Background())

	select {
	case <-supervisor.selfStarted:
		t.Fatalf("planner error preparo fallback generico: request=%+v", supervisor.lastSelfRequest)
	default:
	}
	if supervisor.planCalls != 1 || supervisor.selfCalls != 0 {
		t.Fatalf("plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
}
