package orquestaserver

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0SupervisorNoPreparaAutomejoraConResidentPendingSinDispatchV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: supervisorResultWithResidentPendingForTestV0(
			"run-external-work-opes-b305970086cb77343865513f32df7a66",
		)}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-no-debe-usarse"}},
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
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.planCalls != 0 || supervisor.selfCalls != 0 {
		t.Fatalf("resident pending no debe planificar: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if !strings.HasPrefix(store.last.IdleSelfImprovementReason, idleSelfImprovementResidentPendingBlockedReasonV0) ||
		store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.Counters["resident_pending_without_dispatch"] != 1 {
		t.Fatalf("state=%+v", store.last)
	}
	if !strings.Contains(strings.Join(store.last.IdleSelfImprovementOperationalMessage.RunRefs, ","), "b305970086cb77343865513f32df7a66") {
		t.Fatalf("run_refs=%v", store.last.IdleSelfImprovementOperationalMessage.RunRefs)
	}
	if !containsStringForTestV0(
		store.last.IdleSelfImprovementOperationalMessage.EvidenceRefs,
		"evidence-ref-resident-director-wakeup-unavailable",
	) {
		t.Fatalf("evidence_refs=%v", store.last.IdleSelfImprovementOperationalMessage.EvidenceRefs)
	}
}

func TestRuntimeV0SupervisorDespiertaDirectorResidenteConResidentPendingSinDispatchV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 10, 5, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: supervisorResultWithResidentPendingForTestV0(
			"run-external-work-opes-b305970086cb77343865513f32df7a66",
		)}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-no-debe-usarse"}},
		selfStarted:  make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 3,
		IdleSelfImprovementTargetQueue: 4,
		ResidentDirectorEnabled:        true,
		AuditDisabled:                  true,
	}, RuntimeDepsV0{
		Supervisor:       supervisor,
		ResidentDirector: &fakeResidentDirectorV0{},
		StateStore:       store,
		Clock:            fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runSupervisorTickV0(context.Background())

	select {
	case wakeup := <-runtime.residentDirectorWakeups:
		if wakeup.Cause != idleSelfImprovementResidentPendingBlockedReasonV0 {
			t.Fatalf("wakeup=%+v", wakeup)
		}
	default:
		t.Fatalf("resident director wakeup no solicitado")
	}
	if supervisor.planCalls != 0 || supervisor.selfCalls != 0 {
		t.Fatalf("resident pending no debe planificar: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.Message != "resident_director_wakeup_requested" ||
		!containsStringForTestV0(
			store.last.IdleSelfImprovementOperationalMessage.EvidenceRefs,
			"evidence-ref-resident-director-wakeup-requested",
		) {
		t.Fatalf("state=%+v", store.last)
	}
}

func TestRuntimeV0SupervisorNoBloqueaAutomejoraPorResidentPendingRetryableV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 10, 30, 0, 0, time.UTC)
	runRef := "run-ref-resident-pending-retryable-001"
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: supervisorResultWithResidentPendingForTestV0(runRef)}},
		retryableRunRefs: []string{
			runRef,
		},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-replacement"}},
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
		t.Fatalf("resident pending retryable no debe bloquear automejora: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.lastPlanRequest.QueueSize != 0 ||
		containsStringForTestV0(supervisor.lastPlanRequest.KnownRunRefs, runRef) {
		t.Fatalf("plan=%+v", supervisor.lastPlanRequest)
	}
}

func TestRuntimeV0SupervisorNoBloqueaAutomejoraPorResidentPendingConProcesoVivoV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 11, 0, 0, 0, time.UTC)
	runRef := "run-external-work-opes-b305970086cb77343865513f32df7a66"
	result := supervisorResultWithResidentPendingForTestV0(runRef)
	result.Diagnostics = []orquestaruncoordinator.RunDrainDiagnosticV0{{
		Kind:   "process_runtime_snapshot",
		Status: "running",
		RunRef: runRef,
		EvidenceRefs: []string{
			"evidence-ref-codex-supervisor-process-live",
		},
	}}
	supervisor := &fakeSupervisorV0{
		results:      []fakeSupervisorResultV0{{result: result}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-fill-live-pending"}},
		selfStarted:  make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 3,
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
		t.Fatalf("resident pending con proceso vivo no debe bloquear capacidad: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.lastPlanRequest.Trigger != "capacity_free" ||
		supervisor.lastPlanRequest.QueueSize != 1 ||
		supervisor.lastPlanRequest.FreeCapacity != 2 {
		t.Fatalf("plan=%+v", supervisor.lastPlanRequest)
	}
}

func supervisorResultWithResidentPendingForTestV0(runRef string) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
		TotalExecutions: 0,
		TotalSkips:      1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
					RunRef: runRef,
					Rank:   1,
				}},
				Skips: []orquestaruncoordinator.RunSkipSummaryV0{{
					RunRef: runRef,
					Rank:   1,
					Reason: "resident_director_pending",
					Status: "resident_director_pending",
				}},
			},
		}},
	}
}
