package orquestaserver

import (
	"context"
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0SupervisorNoPreparaAutomejoraConWaitExternalActivoV0(t *testing.T) {
	now := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
						RunRef: "run-ref-wait-external-001",
						Rank:   1,
					}},
					Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
						RunRef:  "run-ref-wait-external-001",
						Outcome: "wait_external",
						EvidenceRefs: []string{
							"evidence-ref-wait-external-live-001",
						},
					}},
				},
			}},
		}}},
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
		t.Fatalf("wait_external activo no debe planificar: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if store.snapshotV0().IdleSelfImprovementReason != idleSelfImprovementExternalWaitBlockedReasonV0 {
		t.Fatalf("reason=%q state=%+v", store.snapshotV0().IdleSelfImprovementReason, store.snapshotV0())
	}
}

func TestRuntimeV0SupervisorNoPreparaAutomejoraConAckPendienteV0(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 35, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	runRef := "run-ref-srv-task-022-ack-pending"
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      "runtime_error",
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
						RunRef: runRef,
						Rank:   1,
					}},
					Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
						RunRef:      runRef,
						Outcome:     "runtime_error",
						QueueStatus: "blocked",
						Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
							RunRef:       runRef,
							Kind:         "ack_pending",
							Status:       "pending",
							EvidenceRefs: []string{"ack_pending"},
						}},
					}},
				},
			}},
		}}},
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
		t.Fatalf("ack pendiente no debe planificar: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if store.snapshotV0().LastSupervisorStatus != SupervisorPublicStatusWaitingExternalV0 ||
		store.snapshotV0().IdleSelfImprovementReason != idleSelfImprovementExternalWaitBlockedReasonV0 {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0SupervisorNoBloqueaAutomejoraPorWaitExternalRetryableV0(t *testing.T) {
	now := time.Date(2026, 5, 26, 10, 30, 0, 0, time.UTC)
	waitRunRef := "request-ref-autoprogramming-backlog-t88-wait-stale-retry-001"
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
			Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
				Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
					Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
						RunRef: waitRunRef,
						Rank:   1,
					}},
					Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
						RunRef:       waitRunRef,
						Outcome:      "wait_external",
						EvidenceRefs: []string{"evidence-ref-wait-external-stale-001"},
					}},
				},
			}},
		}}},
		planRequests:     []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-retry-replacement-001"}},
		retryableRunRefs: []string{waitRunRef},
		selfStarted:      make(chan struct{}, 1),
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
		t.Fatalf("wait_external retryable no debe bloquear automejora: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.planCalls != 1 ||
		supervisor.lastPlanRequest.Trigger != idleSelfImprovementTriggerCapacityFreeV0 ||
		supervisor.lastPlanRequest.QueueSize != 0 ||
		containsStringForTestV0(supervisor.lastPlanRequest.KnownRunRefs, waitRunRef) {
		t.Fatalf("plan=%+v self_calls=%d", supervisor.lastPlanRequest, supervisor.selfCalls)
	}
}

func TestIdleSelfImprovementExternalWaitBlockV0ExtraeRefsYEvidenciaV0(t *testing.T) {
	block := detectIdleSelfImprovementExternalWaitBlockV0(orquestarunsupervisor.RunSupervisorResultV0{
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:       "run-ref-wait-external-002",
					Outcome:      "wait_external",
					EvidenceRefs: []string{"evidence-ref-wait-external-002"},
					Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
						FirstPendingRefs: []string{"agent-ref-pending-002"},
					}},
				}},
			},
		}},
	})

	if !block.Blocked ||
		!containsStringForTestV0(block.RunRefs, "run-ref-wait-external-002") ||
		!containsStringForTestV0(block.EvidenceRefs, "agent-ref-pending-002") {
		t.Fatalf("block=%+v", block)
	}
}
