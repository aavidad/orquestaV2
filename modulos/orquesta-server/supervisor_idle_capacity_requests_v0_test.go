package orquestaserver

import (
	"context"
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0SupervisorPreparaAutomejoraConCapacidadLibreV0(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
						RunRef: "request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-aaa-retry-bbb",
						Rank:   1,
					}, {
						RunRef: "request-ref-autoprogramming-backlog-t09-sanitizer-ccc",
						Rank:   2,
					}},
					Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
						RunRef:  "request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-aaa-retry-bbb",
						Outcome: "drained",
					}},
				},
			}},
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-backlog-t10",
		}, {
			RequestRef: "request-ref-backlog-scanner",
		}},
		selfStarted: make(chan struct{}, 2),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 3,
		IdleSelfImprovementTargetQueue: 4,
		AuditDisabled:                  true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runSupervisorTickV0(context.Background())

	for i := 0; i < 2; i++ {
		select {
		case <-supervisor.selfStarted:
		case <-time.After(time.Second):
			t.Fatalf("automejora por capacidad no preparada: i=%d calls=%d", i, supervisor.selfCalls)
		}
	}
	if supervisor.planCalls != 1 ||
		supervisor.lastPlanRequest.Trigger != "capacity_free" ||
		supervisor.lastPlanRequest.MaxRequests != 2 ||
		supervisor.lastPlanRequest.QueueSize != 2 ||
		supervisor.lastPlanRequest.FreeCapacity != 2 ||
		!containsStringForTestV0(
			supervisor.lastPlanRequest.KnownRequestRefs,
			"request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-aaa",
		) ||
		supervisor.selfCalls != 2 {
		t.Fatalf("plan=%+v self_calls=%d", supervisor.lastPlanRequest, supervisor.selfCalls)
	}
}

func TestRuntimeV0SupervisorPreparaAutomejoraConColaVaciaBajoObjetivoV0(t *testing.T) {
	now := time.Date(2026, 7, 1, 22, 31, 19, 0, time.UTC)
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{},
			}},
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-backlog-scanner",
		}},
		selfStarted: make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 3,
		IdleSelfImprovementTargetQueue: 4,
		AuditDisabled:                  true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runSupervisorTickV0(context.Background())

	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("automejora no preparada con cola vacia: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.planCalls != 1 ||
		supervisor.lastPlanRequest.Trigger != "capacity_free" ||
		supervisor.lastPlanRequest.QueueSize != 0 ||
		supervisor.lastPlanRequest.FreeCapacity != 4 ||
		supervisor.lastPlanRequest.MaxRequests != 3 ||
		supervisor.selfCalls != 1 ||
		supervisor.lastSelfRequest.RequestRef != "request-ref-backlog-scanner" {
		t.Fatalf("plan=%+v self_calls=%d last_self=%+v",
			supervisor.lastPlanRequest,
			supervisor.selfCalls,
			supervisor.lastSelfRequest,
		)
	}
}

func TestRuntimeV0SupervisorPreparaCapacidadAunqueRelojIdleEsteDesactivadoV0(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-backlog-capacity-with-idle-clock-off",
		}},
		selfStarted: make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                        t.TempDir(),
		TickInterval:                    time.Hour,
		IdleSelfImprovementAfter:        0,
		IdleSelfImprovementIdleDisabled: true,
		IdleSelfImprovementMaxRequests:  3,
		IdleSelfImprovementTargetQueue:  4,
		AuditDisabled:                   true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now.Add(-2*time.Hour))

	runtime.runSupervisorTickV0(context.Background())

	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("capacidad libre no preparo con reloj idle apagado: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.planCalls != 1 ||
		supervisor.lastPlanRequest.Trigger != "capacity_free" ||
		supervisor.lastPlanRequest.QueueSize != 0 ||
		supervisor.lastPlanRequest.FreeCapacity != 4 ||
		supervisor.selfCalls != 1 {
		t.Fatalf("plan=%+v self_calls=%d", supervisor.lastPlanRequest, supervisor.selfCalls)
	}
}

func TestRuntimeV0SupervisorFiltraRequestsDeAutomejoraPorPuertoV0(t *testing.T) {
	now := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{RunRef: "run-ref-live", Rank: 1}},
				},
			}},
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-autoprogramming-backlog-tareas-futuras-001",
		}, {
			RequestRef: "request-ref-autoprogramming-backlog-t33-real-001",
		}},
		filterRequests: []IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-autoprogramming-backlog-t33-real-001",
		}},
		selfStarted: make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 2,
		IdleSelfImprovementTargetQueue: 3,
		AuditDisabled:                  true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runSupervisorTickV0(context.Background())

	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("automejora filtrada no preparada: filter_calls=%d self_calls=%d", supervisor.filterCalls, supervisor.selfCalls)
	}
	if supervisor.filterCalls != 1 ||
		len(supervisor.lastFilterRequest.Requests) != 2 ||
		supervisor.selfCalls != 1 ||
		supervisor.lastSelfRequest.RequestRef != "request-ref-autoprogramming-backlog-t33-real-001" {
		t.Fatalf("filter_calls=%d last_filter=%+v self_calls=%d last_self=%+v",
			supervisor.filterCalls, supervisor.lastFilterRequest, supervisor.selfCalls, supervisor.lastSelfRequest)
	}
}

func TestRuntimeV0SupervisorCapacidadLibreNoEsBloqueadaPorCooldownV0(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 2, 0, 0, time.UTC)
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
						RunRef: "request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-aaa",
						Rank:   1,
					}},
					Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
						RunRef:  "request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-aaa",
						Outcome: "drained",
					}},
				},
			}},
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-scanner"}},
		selfStarted:  make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 3,
		IdleSelfImprovementTargetQueue: 4,
		AuditDisabled:                  true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted: true,
		RunRef:   "request-ref-autoprogramming-backlog-t07-prev",
	}, now.Add(-10*time.Second))

	runtime.runSupervisorTickV0(context.Background())

	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("capacity_free quedo bloqueado por cooldown: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.lastPlanRequest.Trigger != "capacity_free" ||
		supervisor.lastPlanRequest.FreeCapacity != 3 {
		t.Fatalf("plan=%+v", supervisor.lastPlanRequest)
	}
}
