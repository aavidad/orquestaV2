package orquestaserver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
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
	if supervisor.lastCommand.MaxTicks != DefaultSupervisorMaxTicksV0 {
		t.Fatalf("max_ticks=%d want=%d", supervisor.lastCommand.MaxTicks, DefaultSupervisorMaxTicksV0)
	}
	if store.last.SupervisorTicks != 1 ||
		store.last.LastSupervisorStatus != "ok" ||
		store.last.LastSupervisorStop != orquestarunsupervisor.RunSupervisorStopNoExecutionV0 ||
		store.last.LastSupervisorStopPublic != "idle_no_execution" ||
		store.last.LastSupervisorStopCategory != "idle" ||
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
		store.last.LastSupervisorQueueRef != "global" ||
		len(store.last.RecentErrors) != 1 ||
		store.last.RecentErrors[0].Code != orquestarunsupervisor.RunSupervisorStopTickErrorV0 ||
		store.last.RecentErrors[0].Scope != "supervisor" {
		t.Fatalf("state error=%+v", store.last)
	}
	runtime.runSupervisorTickV0(context.Background())
	if supervisor.calls != 2 ||
		store.last.LastSupervisorStatus != "ok" ||
		store.last.LastError != "" ||
		store.last.LastSupervisorError != "" ||
		store.last.LastSupervisorExecutions != 2 ||
		store.last.SupervisorTicks != 2 ||
		store.last.SupervisorErrorTicks != 1 ||
		len(store.last.RecentErrors) != 1 {
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
	time.Sleep(10 * time.Millisecond)
	if !runtime.runSupervisorTickAsyncV0(context.Background()) {
		t.Fatalf("async tick did not restart after first completed")
	}
	supervisor.release()
	supervisor.waitDone(t)
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
	if !runtime.runSupervisorTickAsyncV0(context.Background()) {
		t.Fatalf("idle self improvement blocked supervisor tick")
	}
	close(supervisor.selfRelease)
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
						Outcome: "wait_external",
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
						Outcome: "wait_external",
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

func TestRuntimeV0SupervisorPreparaAutomejoraConCapacidadLibreAunqueHayaSkipsV0(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 3, 0, 0, time.UTC)
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			TotalSkips:      1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
						RunRef: "request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-aaa",
						Rank:   1,
					}, {
						RunRef: "request-ref-autoprogramming-backlog-t09-sanitizer-ccc",
						Rank:   2,
					}},
					Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
						RunRef:  "request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-aaa",
						Outcome: "wait_external",
					}},
					Skips: []orquestaruncoordinator.RunSkipSummaryV0{{
						RunRef: "request-ref-autoprogramming-backlog-t09-sanitizer-ccc",
						Rank:   2,
						Reason: "capacity_temporal",
					}},
				},
			}},
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-t10"}},
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

	runtime.runSupervisorTickV0(context.Background())

	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("automejora no preparada con skips no terminales: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.lastPlanRequest.Trigger != "capacity_free" ||
		supervisor.lastPlanRequest.Skips != 1 ||
		supervisor.lastPlanRequest.QueueSize != 2 ||
		supervisor.lastPlanRequest.FreeCapacity != 2 {
		t.Fatalf("plan=%+v", supervisor.lastPlanRequest)
	}
}

func TestRuntimeV0SupervisorNoPreparaAutomejoraSiCapacidadLlenaV0(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 5, 0, 0, time.UTC)
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
						RunRef: "run-ref-1", Rank: 1,
					}, {
						RunRef: "run-ref-2", Rank: 2,
					}},
				},
			}},
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-t10"}},
		selfStarted:  make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 2,
		IdleSelfImprovementTargetQueue: 2,
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

	if supervisor.planCalls != 0 || supervisor.selfCalls != 0 {
		t.Fatalf("plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
}

func TestRuntimeV0SupervisorDescuentaRunsRetryablesDePresionDeColaV0(t *testing.T) {
	now := time.Date(2026, 5, 25, 18, 20, 0, 0, time.UTC)
	retryableRunRef := "request-ref-autoprogramming-backlog-t34-director-wave-strict-guards-default-a3db5927"
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
						RunRef: retryableRunRef,
						Rank:   1,
					}, {
						RunRef: "request-ref-autoprogramming-backlog-t42-live",
						Rank:   2,
					}},
					Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
						RunRef:  retryableRunRef,
						Outcome: "wait_external",
					}},
				},
			}},
		}}},
		retryableRunRefs: []string{retryableRunRef},
		planRequests:     []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-retry-fill"}},
		selfStarted:      make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 2,
		IdleSelfImprovementTargetQueue: 2,
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
		t.Fatalf("automejora no preparada descontando retryables: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.lastPlanRequest.QueueSize != 1 ||
		supervisor.lastPlanRequest.FreeCapacity != 1 ||
		containsStringForTestV0(supervisor.lastPlanRequest.KnownRunRefs, retryableRunRef) ||
		containsStringForTestV0(supervisor.lastPlanRequest.KnownRequestRefs, retryableRunRef) {
		t.Fatalf("plan=%+v retryable=%s", supervisor.lastPlanRequest, retryableRunRef)
	}
}

func TestRuntimeV0SupervisorNoPreparaAutomejoraConProveedorAuthBloqueadoV0(t *testing.T) {
	now := time.Date(2026, 5, 25, 17, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-no-debe-usarse"}},
		selfStarted:  make(chan struct{}, 1),
		blocker: IdleSelfImprovementBlockerResultV0{
			Blocked:      true,
			Reason:       "provider_auth_blocked",
			RunRefs:      []string{"run-ref-auth-blocked-001"},
			EvidenceRefs: []string{"evidence-ref-auth-config-blocker"},
			Message:      "Autenticacion externa bloqueada; requiere reautorizacion.",
		},
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
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.planCalls != 0 || supervisor.selfCalls != 0 {
		t.Fatalf("bloqueo proveedor no debe planificar: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if store.last.IdleSelfImprovementReason != "provider_auth_blocked" {
		t.Fatalf("reason=%q state=%+v", store.last.IdleSelfImprovementReason, store.last)
	}
}

func TestStatusTrackerV0ConservaAutomejoraAceptadaDuranteBloqueoV0(t *testing.T) {
	now := time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{}, now)
	tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted: true, RunRef: "run-ref-1", RequestRef: "request-ref-1",
	}, now)
	state := tracker.MarkIdleSelfImprovementCheckedV0("attempt_blocked", now.Add(time.Minute))
	if !strings.Contains(state.IdleSelfImprovementReason, "run_ref=run-ref-1") ||
		!strings.Contains(state.IdleSelfImprovementReason, "pending=accepted_attempt") {
		t.Fatalf("reason=%q", state.IdleSelfImprovementReason)
	}
}

type fakeSupervisorV0 struct {
	calls            int
	selfCalls        int
	planCalls        int
	lastCommand      orquestarunsupervisor.RunSupervisorCommandV0
	lastSelfRequest  IdleSelfImprovementRequestV0
	lastPlanRequest  IdleSelfImprovementPlanRequestV0
	results          []fakeSupervisorResultV0
	planRequests     []IdleSelfImprovementRequestV0
	blocker          IdleSelfImprovementBlockerResultV0
	retryableRunRefs []string
	selfRequestRefs  []string
	selfStarted      chan struct{}
	selfRelease      chan struct{}
}

func (fake *fakeSupervisorV0) PrepareIdleSelfImprovementV0(
	_ context.Context,
	request IdleSelfImprovementRequestV0,
) (IdleSelfImprovementResultV0, error) {
	fake.selfCalls++
	fake.lastSelfRequest = request
	fake.selfRequestRefs = append(fake.selfRequestRefs, request.RequestRef)
	if fake.selfStarted != nil {
		fake.selfStarted <- struct{}{}
	}
	if fake.selfRelease != nil {
		<-fake.selfRelease
	}
	return IdleSelfImprovementResultV0{
		Accepted: true, RunRef: "run-ref-idle-self-improvement-001", RequestRef: request.RequestRef, Status: "ok",
	}, nil
}

func (fake *fakeSupervisorV0) IdleSelfImprovementBlockersV0(
	context.Context,
	IdleSelfImprovementBlockerRequestV0,
) (IdleSelfImprovementBlockerResultV0, error) {
	return fake.blocker, nil
}

func (fake *fakeSupervisorV0) RetryableIdleSelfImprovementRunRefsV0(
	context.Context,
	IdleSelfImprovementRunFreshnessRequestV0,
) (IdleSelfImprovementRunFreshnessResultV0, error) {
	return IdleSelfImprovementRunFreshnessResultV0{
		RetryableRunRefs: append([]string(nil), fake.retryableRunRefs...),
	}, nil
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
		Ticks:           []orquestarunsupervisor.RunSupervisorTickSummaryV0{{TickNumber: 1}},
		TotalExecutions: 1,
		TotalSkips:      2,
		StopReason:      orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

type memoryStateStoreV0 struct{ last StateV0 }

func (store *memoryStateStoreV0) SaveServerStateV0(_ context.Context, state StateV0) error {
	store.last = state
	return nil
}
func (store *memoryStateStoreV0) LoadServerStateV0(context.Context) (StateV0, error) {
	return store.last, nil
}

type fixedClockV0 struct{ now time.Time }

func (clock fixedClockV0) Now() time.Time { return clock.now }

func markNoExecutionSinceForTestV0(runtime *RuntimeV0, now time.Time) {
	runtime.tracker.MarkSupervisorV0(runtime.config.SupervisorCommand, orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, now.Add(-61*time.Second))
}

func containsStringForTestV0(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
