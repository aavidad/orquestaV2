package orquestaserver

import (
	"context"
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

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
						Outcome: "drained",
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
						Outcome: "drained",
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

func TestRuntimeV0SupervisorDescuentaRequestRefsRetryablesConEvidenciaV0(t *testing.T) {
	now := time.Date(2026, 5, 27, 11, 20, 0, 0, time.UTC)
	retryableRunRef := "request-ref-autoprogramming-backlog-srv-task-011-guardian-retry-001-retry-002"
	retryableRequestRef := "request-ref-autoprogramming-backlog-srv-task-011-guardian-retry-001"
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
						RunRef: retryableRunRef,
						Rank:   1,
					}},
					Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
						RunRef:  retryableRunRef,
						Outcome: "drained",
					}},
				},
			}},
		}}},
		retryableRequestRefs:  []string{retryableRequestRef},
		retryableEvidenceRefs: []string{"evidence-ref-guardian-retryable-result"},
		planRequests:          []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-guardian-replacement"}},
		selfStarted:           make(chan struct{}, 1),
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
		t.Fatalf("automejora no preparada descontando request retryable: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.lastPlanRequest.QueueSize != 0 ||
		supervisor.lastPlanRequest.FreeCapacity != 2 ||
		containsStringForTestV0(supervisor.lastPlanRequest.KnownRunRefs, retryableRunRef) ||
		containsStringForTestV0(supervisor.lastPlanRequest.KnownRequestRefs, retryableRequestRef) ||
		!containsStringForTestV0(supervisor.lastPlanRequest.RetryableRequestRefs, retryableRequestRef) ||
		!containsStringForTestV0(supervisor.lastPlanRequest.RetryableEvidenceRefs, "evidence-ref-guardian-retryable-result") {
		t.Fatalf("plan=%+v", supervisor.lastPlanRequest)
	}
}
