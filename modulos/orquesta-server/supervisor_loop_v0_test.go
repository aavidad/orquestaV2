package orquestaserver

import (
	"context"
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestSupervisorProjectionV0WaitUnhandledOutboxNoEsRunningV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
		supervisorResultWithUnhandledOutboxForTestV0("run-ref-t260-outbox-001"),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusWaitingOutboxV0 ||
		state.LastSupervisorStatus == "running" ||
		state.LastSupervisorStopPublic != SupervisorPublicStopWaitingOutboxV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryWaitOutboxV0 {
		t.Fatalf("state=%+v", state)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["waiting_outbox"]; got != 1 {
		t.Fatalf("waiting_outbox=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["running_live"]; got != 0 {
		t.Fatalf("running_live=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
}

func TestRuntimeV0SupervisorNoPreparaAutomejoraConWaitUnhandledOutboxV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 15, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{
			result: supervisorResultWithUnhandledOutboxForTestV0("run-ref-t260-outbox-002"),
		}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-no-outbox"}},
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
		t.Fatalf("wait_unhandled_outbox no debe planificar: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if store.last.LastSupervisorStatus != SupervisorPublicStatusWaitingOutboxV0 ||
		store.last.IdleSelfImprovementReason != SupervisorPublicStatusWaitingOutboxV0 {
		t.Fatalf("state=%+v", store.last)
	}
}

func TestSupervisorProjectionV0WaitExternalNoEsRunningV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 20, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
		supervisorResultWithWaitExternalForTestV0("run-ref-t260-wait-external"),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusWaitingExternalV0 ||
		state.LastSupervisorStatus == "running" ||
		state.LastSupervisorStopPublic != SupervisorPublicStopWaitingExternalV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryWaitExternalV0 {
		t.Fatalf("state=%+v", state)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["waiting_external"]; got != 1 {
		t.Fatalf("waiting_external=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["running_live"]; got != 0 {
		t.Fatalf("running_live=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
}

func TestSupervisorProjectionV0ProcessRefSinProcesoVerificableEsStalledV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 30, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
		supervisorResultWithUnverifiedProcessForTestV0("run-ref-t260-dead-process"),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusStalledV0 ||
		state.LastSupervisorStatus == "running" ||
		state.LastSupervisorStopPublic != SupervisorPublicStopStalledV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryExternalProcessV0 {
		t.Fatalf("state=%+v", state)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["stalled"]; got != 1 {
		t.Fatalf("stalled=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
	state = (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260-stop"},
		orquestarunsupervisor.RunSupervisorResultV0{StopReason: "running"},
		now,
	)
	if state.LastSupervisorStatus != SupervisorPublicStatusStalledV0 ||
		state.LastSupervisorStopPublic != SupervisorPublicStopStalledV0 {
		t.Fatalf("stop_reason running sin evidencia viva state=%+v", state)
	}
}

func TestSupervisorProjectionV0QueueRunningSinEvidenciaVivaEsStalledV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 35, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
		supervisorResultWithQueueRunningOnlyForTestV0("run-ref-t260-queue-running"),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusStalledV0 ||
		state.LastSupervisorStopPublic != SupervisorPublicStopStalledV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryExternalProcessV0 {
		t.Fatalf("state=%+v", state)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["stalled"]; got != 1 {
		t.Fatalf("stalled=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
}

func TestSupervisorProjectionV0LaunchFailedNoQuedaTapadoPorOutboxV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 40, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
		supervisorResultWithLaunchFailedForTestV0("run-ref-t260-launch-failed"),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusLaunchFailedV0 ||
		state.LastSupervisorStopPublic != SupervisorPublicStopLaunchFailedV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryExternalProcessV0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestSupervisorProjectionV0ProcessRefConEvidenciaVivaEsRunningLiveV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 45, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
		supervisorResultWithLiveProcessEvidenceForTestV0("run-ref-t260-live-process"),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusRunningLiveV0 ||
		state.LastSupervisorStopPublic != SupervisorPublicStopRunningLiveV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryExternalProcessV0 {
		t.Fatalf("state=%+v", state)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["running_live"]; got != 1 {
		t.Fatalf("running_live=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
}

func TestSupervisorProjectionV0ProcesoVivoNoQuedaTapadoPorHistoricoNoVerificadoV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 50, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
		supervisorResultWithLiveAndUnverifiedProcessForTestV0("run-ref-t260-live-process", "run-ref-t260-dead-process"),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusRunningLiveV0 ||
		state.LastSupervisorStopPublic != SupervisorPublicStopRunningLiveV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryExternalProcessV0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestSupervisorProjectionV0SnapshotProcesoRunningConOutboxEsRunningLiveV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
		supervisorResultWithLiveProcessSnapshotForTestV0("run-ref-t260-live-snapshot"),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusRunningLiveV0 ||
		state.LastSupervisorStopPublic != SupervisorPublicStopRunningLiveV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryExternalProcessV0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestRuntimeV0SupervisorDeduplicaRescatesYConservaCausaV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC)
	baseRef := "request-ref-autoprogramming-backlog-t262-rescue-dedupe"
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{
			RequestRef: baseRef + "-retry-a1", ProjectRef: "project-ref-t262", SuggestedArea: "tema-t262", WriteSet: []string{"modulos/orquesta-server"},
		}, {
			RequestRef: baseRef + "-alt-retry-b2", ProjectRef: "project-ref-t262", SuggestedArea: "tema-t262", WriteSet: []string{"modulos/orquesta-server"},
		}},
		selfStarted: make(chan struct{}, 2),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 3,
		AuditDisabled:                  true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)

	runtime.runSupervisorTickV0(context.Background())

	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("rescate no preparado")
	}
	select {
	case <-supervisor.selfStarted:
		t.Fatalf("rescate duplicado preparado: refs=%v", supervisor.selfRequestRefs)
	case <-time.After(50 * time.Millisecond):
	}
	request := supervisor.lastSelfRequest
	if supervisor.selfCalls != 1 ||
		request.ParentRunRef != baseRef ||
		request.SupersedesRunRef != baseRef ||
		request.ActiveAttemptRef != baseRef+"-retry-a1" ||
		!containsStringForTestV0(request.EvidenceRefs, "evidence-ref-idle-self-improvement-rescue-deduped") ||
		!containsStringForTestV0(request.ContextRefs, "deduped_rescue_ref:"+baseRef+"-alt-retry-b2") {
		t.Fatalf("calls=%d request=%+v", supervisor.selfCalls, request)
	}
}

func supervisorResultWithUnhandledOutboxForTestV0(runRef string) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
		TotalExecutions: 1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
					RunRef: runRef,
					Rank:   1,
				}},
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:      runRef,
					Outcome:     "",
					QueueStatus: "running",
					Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
						PendingOutboxCount: 1,
						PendingOutboxRefs:  []string{"outbox-ref-t260-001"},
					}},
				}},
			},
		}},
	}
}

func supervisorResultWithLiveProcessEvidenceForTestV0(runRef string) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
		TotalExecutions: 1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
					RunRef: runRef,
					Rank:   1,
				}},
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:       runRef,
					Outcome:      "process_ref_registered",
					QueueStatus:  "running",
					EvidenceRefs: []string{"evidence-ref-codex-supervisor-process-live"},
				}},
			},
		}},
	}
}

func supervisorResultWithWaitExternalForTestV0(runRef string) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
		TotalExecutions: 1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{RunRef: runRef, Rank: 1}},
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:       runRef,
					Outcome:      "wait_external",
					QueueStatus:  "running",
					EvidenceRefs: []string{"agent-ref-t260-wait-external"},
				}},
			},
		}},
	}
}

func supervisorResultWithLiveProcessSnapshotForTestV0(runRef string) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
		TotalExecutions: 1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:      runRef,
					Outcome:     "process_ref_registered",
					QueueStatus: "running",
					Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
						Kind:               "process_runtime_snapshot",
						Status:             "running",
						RunRef:             runRef,
						PendingOutboxCount: 1,
						PendingOutboxRefs:  []string{"outbox-ref-t260-live"},
					}},
				}},
			},
		}},
	}
}

func supervisorResultWithQueueRunningOnlyForTestV0(runRef string) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
		TotalExecutions: 1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{RunRef: runRef, Rank: 1}},
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:      runRef,
					QueueStatus: "running",
				}},
			},
		}},
	}
}

func supervisorResultWithUnverifiedProcessForTestV0(runRef string) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
		TotalExecutions: 1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{
					RunRef: runRef,
					Rank:   1,
				}},
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:       runRef,
					Outcome:      "process_ref_registered",
					QueueStatus:  "running",
					EvidenceRefs: []string{"process-ref-t260-dead"},
					Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
						PendingOutboxCount: 1,
						PendingOutboxRefs:  []string{"outbox-ref-t260-after-process-ref"},
					}},
				}},
			},
		}},
	}
}

func supervisorResultWithLiveAndUnverifiedProcessForTestV0(
	liveRunRef string,
	unverifiedRunRef string,
) orquestarunsupervisor.RunSupervisorResultV0 {
	live := supervisorResultWithLiveProcessEvidenceForTestV0(liveRunRef)
	unverified := supervisorResultWithUnverifiedProcessForTestV0(unverifiedRunRef)
	live.Ticks[0].Result.Ranked = append(live.Ticks[0].Result.Ranked, unverified.Ticks[0].Result.Ranked...)
	live.Ticks[0].Result.Executions = append(live.Ticks[0].Result.Executions, unverified.Ticks[0].Result.Executions...)
	live.TotalExecutions += unverified.TotalExecutions
	return live
}

func supervisorResultWithLaunchFailedForTestV0(runRef string) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
		TotalExecutions: 1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:      runRef,
					Outcome:     SupervisorPublicStatusLaunchFailedV0,
					QueueStatus: "running",
					Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
						Kind:               SupervisorPublicStatusLaunchFailedV0,
						Status:             "failed",
						PendingOutboxCount: 1,
						PendingOutboxRefs:  []string{"outbox-ref-t260-launch-failed"},
					}},
				}},
			},
		}},
	}
}
