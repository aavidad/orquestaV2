package orquestaserver

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	stopreason "orquesta/modulos/orquesta-run-supervisor/stopreason"
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
	if store.snapshotV0().LastSupervisorStatus != SupervisorPublicStatusWaitingOutboxV0 ||
		store.snapshotV0().IdleSelfImprovementReason != SupervisorPublicStatusWaitingOutboxV0 {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0IdleSelfImprovementAfterZeroDesactivaPlanificacionV0(t *testing.T) {
	now := time.Date(2026, 7, 1, 23, 45, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-disabled"}},
		selfStarted:  make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementDisabled:    true,
		IdleSelfImprovementAfter:       0,
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
	markNoExecutionSinceForTestV0(runtime, now.Add(-2*time.Minute))

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.planCalls != 0 || supervisor.selfCalls != 0 {
		t.Fatalf("automejora apagada no debe planificar: plan_calls=%d self_calls=%d",
			supervisor.planCalls,
			supervisor.selfCalls,
		)
	}
	select {
	case <-supervisor.selfStarted:
		t.Fatalf("automejora apagada preparo trabajo")
	case <-time.After(50 * time.Millisecond):
	}
	if store.snapshotV0().IdleSelfImprovementReason != "disabled" ||
		store.snapshotV0().IdleSelfImprovementFlight {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0IdleSelfImprovementDefaultDisparaTrasSesentaSegundosV0(t *testing.T) {
	now := time.Date(2026, 7, 2, 13, 20, 0, 0, time.UTC)
	clock := &mutableClockForGoalFirstChainTestV0{now: now}
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{
			result: orquestarunsupervisor.RunSupervisorResultV0{
				StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
			},
		}, {
			result: orquestarunsupervisor.RunSupervisorResultV0{
				StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
			},
		}},
		planRequests: []IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-backlog-default-idle-after-60s",
		}},
		selfStarted: make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      clock,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	markNoExecutionObservedAtForTestV0(runtime, now.Add(-59*time.Second))
	runtime.runSupervisorTickV0(context.Background())
	select {
	case <-supervisor.selfStarted:
		t.Fatalf("automejora disparo antes de 60s")
	case <-time.After(50 * time.Millisecond):
	}
	if supervisor.planCalls != 0 ||
		store.snapshotV0().IdleSelfImprovementReason != "idle_window_waiting" {
		t.Fatalf("antes de umbral plan_calls=%d state=%+v", supervisor.planCalls, store.snapshotV0())
	}

	clock.now = now.Add(time.Second)
	runtime.runSupervisorTickV0(context.Background())
	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("automejora no disparo al cumplir 60s: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if supervisor.planCalls != 1 ||
		supervisor.selfCalls != 1 ||
		supervisor.lastPlanRequest.Trigger != idleSelfImprovementTriggerIdleV0 ||
		supervisor.lastSelfRequest.RequestRef != "request-ref-backlog-default-idle-after-60s" {
		t.Fatalf("plan=%+v self_calls=%d last_self=%+v",
			supervisor.lastPlanRequest,
			supervisor.selfCalls,
			supervisor.lastSelfRequest,
		)
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

func TestSupervisorProjectionV0RuntimeErrorConAckPendienteEsWaitExternalV0(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 25, 0, 0, time.UTC)
	runRef := "run-ref-srv-task-022-ack-pending"
	result := orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      "runtime_error",
		TotalExecutions: 1,
		StopProjection: stopreason.ProjectionV0{
			SchemaVersion: stopreason.SchemaVersionV0,
			PublicReason:  "runtime_error",
			Category:      "blocked",
		},
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{RunRef: runRef, Rank: 1}},
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
	}
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-srv-task-022"},
		result,
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusWaitingExternalV0 ||
		state.LastSupervisorStopPublic != SupervisorPublicStopWaitingExternalV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryWaitExternalV0 {
		t.Fatalf("state=%+v", state)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["waiting_external"]; got != 1 {
		t.Fatalf("waiting_external=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["stalled"]; got != 0 {
		t.Fatalf("stalled=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
}

func TestServerPublicStatusV0DistingueOutboxWaitExternoYProcesoVerificadoV0(t *testing.T) {
	cases := []struct {
		name       string
		stopPublic string
		category   string
	}{
		{
			name:       "outbox pendiente",
			stopPublic: SupervisorPublicStopWaitingOutboxV0,
			category:   SupervisorPublicCategoryWaitOutboxV0,
		},
		{
			name:       "wait externo",
			stopPublic: SupervisorPublicStopWaitingExternalV0,
			category:   SupervisorPublicCategoryWaitExternalV0,
		},
		{
			name:       "proceso externo verificado",
			stopPublic: SupervisorPublicStopRunningLiveV0,
			category:   SupervisorPublicCategoryExternalProcessV0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			public := NewServerPublicStatusV0(StateV0{
				Status:                     SupervisorPublicStatusOKV0,
				LastSupervisorStatus:       SupervisorPublicStatusOKV0,
				LastSupervisorStopPublic:   tc.stopPublic,
				LastSupervisorStopCategory: tc.category,
			})
			if public.LastSupervisorStopPublic != tc.stopPublic ||
				public.LastSupervisorStopCategory != tc.category {
				t.Fatalf("public stop=%q category=%q want %q/%q",
					public.LastSupervisorStopPublic,
					public.LastSupervisorStopCategory,
					tc.stopPublic,
					tc.category,
				)
			}
		})
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

func TestSupervisorProjectionV0ErrorRecuperableConservaEstadoOperativoV0(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 20, 0, 0, time.UTC)
	cases := []struct {
		name         string
		result       orquestarunsupervisor.RunSupervisorResultV0
		wantStatus   string
		wantStop     string
		wantCategory string
		wantCounter  string
	}{
		{
			name:         "outbox pendiente",
			result:       supervisorResultWithUnhandledOutboxForTestV0("run-ref-t260-outbox-error"),
			wantStatus:   SupervisorPublicStatusWaitingOutboxV0,
			wantStop:     SupervisorPublicStopWaitingOutboxV0,
			wantCategory: SupervisorPublicCategoryWaitOutboxV0,
			wantCounter:  "waiting_outbox",
		},
		{
			name:         "wait externo",
			result:       supervisorResultWithWaitExternalForTestV0("run-ref-t260-external-error"),
			wantStatus:   SupervisorPublicStatusWaitingExternalV0,
			wantStop:     SupervisorPublicStopWaitingExternalV0,
			wantCategory: SupervisorPublicCategoryWaitExternalV0,
			wantCounter:  "waiting_external",
		},
		{
			name:         "proceso vivo",
			result:       supervisorResultWithLiveProcessEvidenceForTestV0("run-ref-t260-live-error"),
			wantStatus:   SupervisorPublicStatusRunningLiveV0,
			wantStop:     SupervisorPublicStopRunningLiveV0,
			wantCategory: SupervisorPublicCategoryExternalProcessV0,
			wantCounter:  "running_live",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := (&StatusTrackerV0{}).MarkSupervisorErrorV0(
				orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
				tc.result,
				"supervisor timeout con estado operacional recuperable",
				now,
			)
			if state.LastSupervisorStatus != tc.wantStatus ||
				state.LastSupervisorStopPublic != tc.wantStop ||
				state.LastSupervisorStopCategory != tc.wantCategory ||
				state.LastSupervisorError != "" ||
				state.LastError != "" {
				t.Fatalf("state=%+v", state)
			}
			if state.LastSupervisorOperationalMessage == nil ||
				state.LastSupervisorOperationalMessage.Status != tc.wantStatus ||
				state.LastSupervisorOperationalMessage.Counters[tc.wantCounter] != 1 ||
				state.LastSupervisorOperationalMessage.Counters["supervisor_error_advisory"] != 1 {
				t.Fatalf("message=%+v", state.LastSupervisorOperationalMessage)
			}
			if state.SupervisorLastError == "" || len(state.RecentErrors) != 1 {
				t.Fatalf("error evidence missing state=%+v", state)
			}
		})
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

func TestSupervisorProjectionV0NoMezclaEvidenciaVivaDeOtroRunV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 11, 5, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-t260"},
		supervisorResultWithCrossRunLiveProcessSnapshotForTestV0(
			"run-ref-t260-process-unverified",
			"run-ref-t260-other-live",
		),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusStalledV0 ||
		state.LastSupervisorStopPublic != SupervisorPublicStopStalledV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryExternalProcessV0 {
		t.Fatalf("state=%+v", state)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["running_live"]; got != 0 {
		t.Fatalf("running_live=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["stalled"]; got != 1 {
		t.Fatalf("stalled=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
}

func TestSupervisorErrorProjectionV0ProcesoVivoOAckPendienteNoEsTerminalV0(t *testing.T) {
	now := time.Date(2026, 6, 20, 18, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		result     orquestarunsupervisor.RunSupervisorResultV0
		wantStatus string
		wantStop   string
		wantCat    string
		wantCount  string
		wantEvid   string
	}{
		{
			name: "proceso vivo verificado",
			result: orquestarunsupervisor.RunSupervisorResultV0{
				StopReason:      orquestarunsupervisor.RunSupervisorStopTickErrorV0,
				ErrorRunRefs:    []string{"run-ref-srv-task-022-live"},
				TotalExecutions: 1,
				Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
					Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
						Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
							RunRef:       "run-ref-srv-task-022-live",
							Outcome:      "wait_external",
							QueueStatus:  "candidate_pending",
							EvidenceRefs: []string{"process_live"},
						}},
					},
				}},
			},
			wantStatus: SupervisorPublicStatusRunningLiveV0,
			wantStop:   SupervisorPublicStopRunningLiveV0,
			wantCat:    SupervisorPublicCategoryExternalProcessV0,
			wantCount:  "running_live",
			wantEvid:   "process_live",
		},
		{
			name: "ack tardio pendiente",
			result: orquestarunsupervisor.RunSupervisorResultV0{
				StopReason:      orquestarunsupervisor.RunSupervisorStopTickErrorV0,
				ErrorRunRefs:    []string{"run-ref-srv-task-022-ack"},
				TotalExecutions: 1,
				Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
					Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
						Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
							RunRef:      "run-ref-srv-task-022-ack",
							Outcome:     "wait_external",
							QueueStatus: "candidate_pending",
							Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
								Kind:         "ack_pending",
								Status:       "wait_external",
								RunRef:       "run-ref-srv-task-022-ack",
								PendingCount: 1,
								EvidenceRefs: []string{
									"evidence-ref-srv-task-022-ack-pending",
								},
							}},
						}},
					},
				}},
			},
			wantStatus: SupervisorPublicStatusWaitingExternalV0,
			wantStop:   SupervisorPublicStopWaitingExternalV0,
			wantCat:    SupervisorPublicCategoryWaitExternalV0,
			wantCount:  "waiting_external",
			wantEvid:   "evidence-ref-srv-task-022-ack-pending",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := (&StatusTrackerV0{}).MarkSupervisorErrorV0(
				orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-srv-task-022"},
				tc.result,
				"supervise_timeout_waiting_for_ack",
				now,
			)
			if state.LastSupervisorStatus != tc.wantStatus ||
				state.LastSupervisorStatus == "error" ||
				state.LastSupervisorStopPublic != tc.wantStop ||
				state.LastSupervisorStopCategory != tc.wantCat ||
				state.LastSupervisorError != "" ||
				state.LastError != "" ||
				state.SupervisorLastError != "supervise_timeout_waiting_for_ack" ||
				state.LastSupervisorOperationalMessage == nil ||
				state.LastSupervisorOperationalMessage.Status != tc.wantStatus ||
				state.LastSupervisorOperationalMessage.Counters[tc.wantCount] != 1 ||
				state.LastSupervisorOperationalMessage.Counters["supervisor_error_advisory"] != 1 ||
				len(state.RecentErrors) != 1 ||
				!containsStringForTestV0(state.LastSupervisorOperationalMessage.EvidenceRefs, tc.wantEvid) ||
				!containsStringForTestV0(state.RecentErrors[0].EvidenceRefs, tc.wantEvid) {
				t.Fatalf("state=%+v", state)
			}
			if tc.name == "proceso vivo verificado" &&
				state.LastSupervisorOperationalMessage.Counters["external_wait_live_process"] != 1 {
				t.Fatalf("external_wait_live_process missing counters=%v", state.LastSupervisorOperationalMessage.Counters)
			}
		})
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

func TestRuntimeV0IdleSelfImprovementGoalFirstLanzaGoalSpecV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	goalStates := newMemoryGoalStateStoreV0()
	externalState := goalWorkStateForServerTestV0(
		"run-ref-external-work-active-001",
		"goal-ref-external-work-active-001",
		"external-goal-ref-external-work-active-001",
		"external_work",
		orquestagoal.GoalStatusRunningV0,
	)
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), externalState); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 external: %v", err)
	}
	supervisor := &goalFirstSupervisorForTestV0{started: make(chan struct{}, 1)}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                         t.TempDir(),
		TickInterval:                     time.Hour,
		IdleSelfImprovementAfter:         time.Minute,
		IdleSelfImprovementGoalFirst:     true,
		IdleSelfImprovementProjectRef:    "project-ref-orquesta",
		IdleSelfImprovementWriteSet:      []string{"modulos/orquesta-server/supervisor_loop_v0.go"},
		IdleSelfImprovementRequiredTests: []string{"go test -count=1 ./..."},
		IdleSelfImprovementContextRefs:   []string{"doc-ref-goal-first-codex"},
		IdleSelfImprovementEvidenceRefs:  []string{"evidence-ref-backlog-t260"},
		IdleSelfImprovementAcceptance:    []string{"mantener automejora bajo gobierno externo"},
		IdleSelfImprovementCompactRules:  []string{"AGENTS.md"},
		AuditDisabled:                    true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStates,
		StateStore:     store,
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)

	runtime.runSupervisorTickV0(context.Background())

	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatalf("goal-first no lanzado")
	}
	waitRuntimeAsyncWorkForTestV0(t, runtime)
	spec := supervisor.lastSpec
	if supervisor.launchCalls != 1 ||
		spec.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		spec.RunRef != spec.RequestRef ||
		spec.WorkKind != "idle_self_improvement" ||
		spec.ProjectRef != "project-ref-orquesta" ||
		len(spec.WriteSet) != 1 ||
		spec.WriteSet[0].Path != "modulos/orquesta-server/supervisor_loop_v0.go" ||
		len(spec.RequiredTests) != 1 ||
		spec.RequiredTests[0].Command != "go test -count=1 ./..." ||
		!spec.ClosurePolicy.RequireRequiredTests ||
		!spec.ReworkPolicy.PreferNewGoal ||
		spec.ReworkPolicy.MaxReworkGoals != 1 ||
		!spec.ReworkPolicy.PreserveArtifacts ||
		!containsStringForTestV0(spec.AcceptanceCriteria, "mantener automejora bajo gobierno externo") ||
		!containsStringForTestV0(spec.EvidenceRefs, "evidence-ref-backlog-t260") ||
		!containsGoalContextRefForTestV0(spec.ContextRefs, "doc-ref-goal-first-codex") ||
		!containsGoalRuleRefForTestV0(spec.RuleRefs, "AGENTS.md", orquestagoal.GoalRuleEnforcementAdvisoryV0) ||
		!containsStringForTestV0(spec.SkillRefs, DefaultIdleSelfImprovementSkillRefAutoV0) ||
		!containsStringForTestV0(spec.SkillRefs, DefaultIdleSelfImprovementSkillRefIntV0) {
		t.Fatalf("spec=%+v calls=%d", spec, supervisor.launchCalls)
	}
	goalState, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 idle: %v", err)
	}
	if goalState.Status != orquestagoal.GoalStatusRunningV0 ||
		goalState.RunRef != spec.RunRef ||
		goalState.Spec.WorkKind != "idle_self_improvement" ||
		goalState.ExternalGoalRef != "external-"+spec.GoalRef ||
		!containsStringForTestV0(goalState.EvidenceRefs, "evidence-ref-idle-self-improvement-goal-state-v0") {
		t.Fatalf("goal_state=%+v", goalState)
	}
	active, err := goalStates.ListGoalWorkStatesV0(context.Background(), orquestagoal.GoalWorkStateListRequestV0{
		ActiveOnly: true,
	})
	if err != nil {
		t.Fatalf("ListGoalWorkStatesV0 active: %v", err)
	}
	if len(active) != 2 ||
		!containsGoalWorkStateRunRefForTestV0(active, spec.RunRef) ||
		!containsGoalWorkStateRunRefForTestV0(active, externalState.RunRef) {
		t.Fatalf("active=%+v idle=%s external=%s", active, spec.RunRef, externalState.RunRef)
	}
	if store.snapshotV0().IdleSelfImprovementReason == "" ||
		!strings.Contains(store.snapshotV0().IdleSelfImprovementReason, "goal_ref="+spec.GoalRef) ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementGoalSpec == nil ||
		store.snapshotV0().IdleSelfImprovementGoalSpec.GoalRef != spec.GoalRef ||
		store.snapshotV0().IdleSelfImprovementGoalReceipt == nil ||
		store.snapshotV0().IdleSelfImprovementGoalReceipt.GoalRef != spec.GoalRef ||
		!containsStringForTestV0(store.snapshotV0().IdleSelfImprovementOperationalMessage.GoalRefs, spec.GoalRef) ||
		!containsStringForTestV0(store.snapshotV0().IdleSelfImprovementOperationalMessage.EvidenceRefs, "evidence-ref-idle-self-improvement-goal-first-launched") {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0IdleGoalFirstPrefierePreparacionCompletaSiComposicionLaDeclaraV0(t *testing.T) {
	now := time.Date(2026, 7, 11, 11, 0, 0, 0, time.UTC)
	direct := &goalFirstSupervisorForTestV0{started: make(chan struct{}, 1)}
	supervisor := &goalFirstPreparedSupervisorForTestV0{
		goalFirstSupervisorForTestV0: direct,
		prepared:                     make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                     t.TempDir(),
		TickInterval:                 time.Hour,
		IdleSelfImprovementAfter:     time.Minute,
		IdleSelfImprovementGoalFirst: true,
		AuditDisabled:                true,
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
	case <-supervisor.prepared:
	case <-time.After(time.Second):
		t.Fatal("preparacion goal-first completa no invocada")
	}
	waitRuntimeAsyncWorkForTestV0(t, runtime)
	if supervisor.prepareCalls != 1 || direct.launchCalls != 0 {
		t.Fatalf("prepare_calls=%d direct_launch_calls=%d", supervisor.prepareCalls, direct.launchCalls)
	}
}

func TestRuntimeV0IdleSelfImprovementGoalFirstCompactaObjectiveLargoV0(t *testing.T) {
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		IdleSelfImprovementProjectRef: "project-ref-orquesta",
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     &goalFirstSupervisorForTestV0{},
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     &memoryStateStoreV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	longSummary := "Corregir automejora goal-first. " + strings.Repeat("contexto operativo amplio ", 240)
	spec := runtime.idleSelfImprovementGoalWorkSpecV0(IdleSelfImprovementRequestV0{
		RequestRef:     "request-ref-idle-objective-largo",
		ProjectRef:     "project-ref-orquesta",
		FailureSummary: longSummary,
		ContextRefs:    []string{"doc-ref-contexto-completo"},
		AcceptanceCriteria: []string{
			"mantener evidencia durable",
			strings.Repeat("criterio extenso ", 220),
		},
		EvidenceRefs: []string{"evidence-ref-objective-largo"},
	})

	if got := len([]rune(spec.Objective)); got > idleSelfImprovementGoalObjectiveMaxRunesV0 {
		t.Fatalf("objective len=%d max=%d", got, idleSelfImprovementGoalObjectiveMaxRunesV0)
	}
	for _, want := range []string{
		"Corregir automejora goal-first",
		"objective_compacted",
		"original_sha256=",
		"full_context_in_goal_spec_refs",
	} {
		if !strings.Contains(spec.Objective, want) {
			t.Fatalf("objective compactado no contiene %q:\n%s", want, spec.Objective)
		}
	}
	if !containsGoalContextRefForTestV0(spec.ContextRefs, "doc-ref-contexto-completo") ||
		!containsStringForTestV0(spec.AcceptanceCriteria, "mantener evidencia durable") ||
		!containsStringForTestV0(spec.EvidenceRefs, "evidence-ref-objective-largo") {
		t.Fatalf("spec perdio refs/criterios fuera del objective: %+v", spec)
	}
}

func TestRuntimeV0IdleSelfImprovementGoalFirstPropagaSkillRefsDeRequestV0(t *testing.T) {
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		IdleSelfImprovementProjectRef: "project-ref-orquesta",
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     &goalFirstSupervisorForTestV0{},
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     &memoryStateStoreV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	spec := runtime.idleSelfImprovementGoalWorkSpecV0(IdleSelfImprovementRequestV0{
		RequestRef:     "request-ref-idle-skill-curada",
		ProjectRef:     "project-ref-orquesta",
		FailureSummary: "Anadir validacion con go test.",
		WriteSet:       []string{"modulos/orquesta-autoprogramming"},
		RequiredTests:  []string{"go test -count=1 ./modulos/orquesta-autoprogramming"},
		SkillRefs: []string{
			"skill-ref-orquesta-programacion-tests-v0",
			"skill-ref-orquesta-programacion-tests-v0",
		},
	})

	if !containsStringForTestV0(spec.SkillRefs, "skill-ref-orquesta-programacion-tests-v0") {
		t.Fatalf("skill_refs=%+v", spec.SkillRefs)
	}
	if !containsGoalContextRefForTestV0(spec.ContextRefs, "skill-ref-orquesta-programacion-tests-v0") {
		t.Fatalf("context_refs=%+v", spec.ContextRefs)
	}
}

func TestRuntimeV0IdleSelfImprovementGoalFirstSinLauncherNoCaeALegacyV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 30, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{selfStarted: make(chan struct{}, 1)}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                     t.TempDir(),
		TickInterval:                 time.Hour,
		IdleSelfImprovementAfter:     time.Minute,
		IdleSelfImprovementGoalFirst: true,
		AuditDisabled:                true,
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
		t.Fatalf("goal-first sin launcher no debe usar legacy: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	select {
	case <-supervisor.selfStarted:
		t.Fatalf("goal-first sin launcher preparo automejora legacy")
	case <-time.After(50 * time.Millisecond):
	}
	if store.snapshotV0().IdleSelfImprovementReason != idleSelfImprovementGoalLauncherUnavailableReasonV0 {
		t.Fatalf("reason=%q state=%+v", store.snapshotV0().IdleSelfImprovementReason, store.snapshotV0())
	}
}

func TestRuntimeV0IdleSelfImprovementGoalFirstObservaGoalPendienteV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 13, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &goalObservationSupervisorForTestV0{
		result: orquestagoal.GoalWorkResultV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-autoprogramming-observe-001",
			ExternalGoalRef: "external-goal-ref-observe-001",
			Summary:         "goal sigue vivo",
			EvidenceRefs:    []string{"evidence-ref-goal-running"},
		},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                     t.TempDir(),
		TickInterval:                 time.Hour,
		IdleSelfImprovementAfter:     time.Minute,
		IdleSelfImprovementGoalFirst: true,
		AuditDisabled:                true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted:        true,
		RunRef:          "external-goal-ref-observe-001",
		RequestRef:      "request-ref-observe-001",
		Status:          orquestagoal.GoalStatusAcceptedV0,
		Message:         "goal_first_launched",
		GoalRef:         "goal-ref-autoprogramming-observe-001",
		ExternalGoalRef: "external-goal-ref-observe-001",
	}, now.Add(-time.Minute))

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.observeCalls != 1 ||
		supervisor.lastObservation.GoalRef != "goal-ref-autoprogramming-observe-001" ||
		supervisor.lastObservation.ExternalGoalRef != "external-goal-ref-observe-001" {
		t.Fatalf("observe_calls=%d request=%+v", supervisor.observeCalls, supervisor.lastObservation)
	}
	if store.snapshotV0().IdleSelfImprovementReason == "" ||
		!strings.Contains(store.snapshotV0().IdleSelfImprovementReason, idleSelfImprovementGoalRunningReasonV0) ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalRunningReasonV0 ||
		!containsStringForTestV0(store.snapshotV0().IdleSelfImprovementOperationalMessage.GoalRefs, "goal-ref-autoprogramming-observe-001") {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0IdleSelfImprovementGoalFirstUsaObserverGenericoSiDisponibleV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 13, 10, 0, 0, time.UTC)
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		IdleSelfImprovementAfter:      time.Minute,
		IdleSelfImprovementGoalFirst:  true,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     &fakeSupervisorV0{},
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     &memoryStateStoreV0{},
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted:        true,
		RunRef:          "external-goal-ref-observe-generico-001",
		RequestRef:      "request-ref-observe-generico-001",
		Status:          orquestagoal.GoalStatusAcceptedV0,
		Message:         "goal_first_launched",
		GoalRef:         "goal-ref-autoprogramming-observe-generico-001",
		ExternalGoalRef: "external-goal-ref-observe-generico-001",
	}, now.Add(-time.Minute))

	if runtime.observePendingIdleSelfImprovementGoalV0(context.Background(), now) {
		t.Fatalf("helper idle especifico no debe observar si el observer generico esta disponible")
	}
}

func TestRuntimeV0IdleSelfImprovementGoalFirstCompleteNoCierraSinValidacionV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 13, 30, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &goalObservationSupervisorForTestV0{
		result: orquestagoal.GoalWorkResultV0{
			Status:       orquestagoal.GoalStatusCompleteV0,
			GoalRef:      "goal-ref-autoprogramming-complete-001",
			Summary:      "goal completo observado",
			EvidenceRefs: []string{"evidence-ref-goal-complete"},
		},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                     t.TempDir(),
		TickInterval:                 time.Hour,
		IdleSelfImprovementAfter:     time.Minute,
		IdleSelfImprovementGoalFirst: true,
		AuditDisabled:                true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted: true,
		Status:   orquestagoal.GoalStatusAcceptedV0,
		Message:  "goal_first_launched",
		GoalRef:  "goal-ref-autoprogramming-complete-001",
	}, now.Add(-time.Minute))

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.observeCalls != 1 {
		t.Fatalf("observe_calls=%d", supervisor.observeCalls)
	}
	if store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalCompletePendingClosureV0 ||
		store.snapshotV0().IdleSelfImprovementGoalClosure != nil ||
		store.snapshotV0().IdleSelfImprovementOK != 1 {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0IdleSelfImprovementGoalFirstCompleteValidaCierreConSpecPersistidoV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 14, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	const goalRef = "goal-ref-autoprogramming-complete-accepted-001"
	supervisor := &goalObservationSupervisorForTestV0{
		result: orquestagoal.GoalWorkResultV0{
			Status:       orquestagoal.GoalStatusCompleteV0,
			GoalRef:      goalRef,
			Summary:      "goal completo observado",
			EvidenceRefs: []string{"evidence-ref-required-closure"},
		},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                     t.TempDir(),
		TickInterval:                 time.Hour,
		IdleSelfImprovementAfter:     time.Minute,
		IdleSelfImprovementGoalFirst: true,
		AuditDisabled:                true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted: true,
		Status:   orquestagoal.GoalStatusAcceptedV0,
		Message:  "goal_first_launched",
		GoalRef:  goalRef,
		GoalSpec: orquestagoal.GoalWorkSpecV0{
			GoalRef:       goalRef,
			Objective:     "cerrar automejora goal-first con evidencia causal",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}},
			ClosurePolicy: orquestagoal.GoalClosurePolicyV0{RequiredEvidenceRefs: []string{"evidence-ref-required-closure"}},
		},
	}, now.Add(-time.Minute))

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.observeCalls != 1 {
		t.Fatalf("observe_calls=%d", supervisor.observeCalls)
	}
	if store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalClosureAcceptedReasonV0 ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.Status != orquestagoal.GoalStatusAcceptedV0 ||
		store.snapshotV0().IdleSelfImprovementGoalResult == nil ||
		store.snapshotV0().IdleSelfImprovementGoalResult.GoalRef != goalRef ||
		store.snapshotV0().IdleSelfImprovementGoalClosure == nil ||
		!store.snapshotV0().IdleSelfImprovementGoalClosure.Accepted ||
		!containsStringForTestV0(store.snapshotV0().IdleSelfImprovementOperationalMessage.EvidenceRefs, "evidence-ref-required-closure") {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0SelfAuditBacklogGoalFirstResidenteCierraYEncadenaSiguienteV0(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	clock := &mutableClockForGoalFirstChainTestV0{now: now}
	store := &memoryStateStoreV0{}
	goalStates := newMemoryGoalStateStoreV0()
	first := idleSelfImprovementRequestForGoalFirstChainTestV0("001", "modulos/orquesta-server")
	second := idleSelfImprovementRequestForGoalFirstChainTestV0("002", "modulos/orquesta-app-codex-stack")
	supervisor := &goalFirstChainSupervisorForTestV0{
		planBatches: [][]IdleSelfImprovementRequestV0{{first}, {second}},
		started:     make(chan struct{}, 2),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                         t.TempDir(),
		TickInterval:                     time.Hour,
		IdleSelfImprovementAfter:         time.Minute,
		IdleSelfImprovementMaxRequests:   1,
		IdleSelfImprovementGoalFirst:     true,
		IdleSelfImprovementProjectRef:    "project-ref-self-audit-chain",
		IdleSelfImprovementRequiredTests: []string{"go test -count=1 ./modulos/orquesta-server"},
		AuditDisabled:                    true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStates,
		StateStore:     store,
		Clock:          clock,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)

	runtime.runSupervisorTickV0(ctx)
	waitGoalFirstChainLaunchForTestV0(t, supervisor)
	waitRuntimeAsyncWorkForTestV0(t, runtime)
	firstSpec := supervisor.launchedSpecs[0]
	if supervisor.planCalls != 1 ||
		supervisor.launchCalls != 1 ||
		supervisor.selfCalls != 0 ||
		firstSpec.WorkKind != "idle_self_improvement" ||
		firstSpec.RequestRef != first.RequestRef ||
		firstSpec.RunRef != firstSpec.RequestRef ||
		firstSpec.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != "prepared" {
		t.Fatalf("primer tick inesperado: plan=%d launch=%d self=%d spec=%+v state=%+v",
			supervisor.planCalls, supervisor.launchCalls, supervisor.selfCalls, firstSpec, store.snapshotV0())
	}
	state, err := goalStates.LoadGoalWorkStateV0(ctx, firstSpec.RunRef)
	if err != nil || state.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("goal state primero err=%v state=%+v", err, state)
	}

	clock.now = now.Add(2 * time.Minute)
	runtime.runSupervisorTickV0(ctx)
	if supervisor.observeCalls != 1 ||
		supervisor.launchCalls != 1 ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalClosureAcceptedReasonV0 ||
		store.snapshotV0().IdleSelfImprovementGoalClosure == nil ||
		!store.snapshotV0().IdleSelfImprovementGoalClosure.Accepted ||
		store.snapshotV0().IdleSelfImprovementGoalResult == nil ||
		store.snapshotV0().IdleSelfImprovementGoalResult.Status != orquestagoal.GoalStatusCompleteV0 ||
		store.snapshotV0().IdleSelfImprovementFlight ||
		!containsStringForTestV0(store.snapshotV0().IdleSelfImprovementOperationalMessage.EvidenceRefs, "evidence-ref-required-test-001") {
		t.Fatalf("segundo tick cierre inesperado: observe=%d launch=%d state=%+v",
			supervisor.observeCalls, supervisor.launchCalls, store.snapshotV0())
	}

	clock.now = now.Add(4 * time.Minute)
	runtime.runSupervisorTickV0(ctx)
	waitGoalFirstChainLaunchForTestV0(t, supervisor)
	waitRuntimeAsyncWorkForTestV0(t, runtime)
	secondSpec := supervisor.launchedSpecs[1]
	if supervisor.planCalls != 2 ||
		supervisor.launchCalls != 2 ||
		supervisor.selfCalls != 0 ||
		secondSpec.RequestRef != second.RequestRef ||
		secondSpec.GoalRef == firstSpec.GoalRef ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != "prepared" {
		t.Fatalf("tercer tick no encadeno segundo goal: plan=%d launch=%d self=%d first=%+v second=%+v state=%+v",
			supervisor.planCalls, supervisor.launchCalls, supervisor.selfCalls, firstSpec, secondSpec, store.snapshotV0())
	}
	state, err = goalStates.LoadGoalWorkStateV0(ctx, secondSpec.RunRef)
	if err != nil || state.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("goal state segundo err=%v state=%+v", err, state)
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
					Outcome:     "running",
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

type goalFirstSupervisorForTestV0 struct {
	launchCalls int
	lastSpec    orquestagoal.GoalWorkSpecV0
	started     chan struct{}
}

type goalFirstPreparedSupervisorForTestV0 struct {
	*goalFirstSupervisorForTestV0
	prepareCalls int
	prepared     chan struct{}
}

func (fake *goalFirstPreparedSupervisorForTestV0) GoalFirstIdleSelfImprovementPreparationEnabledV0() bool {
	return true
}

func (fake *goalFirstPreparedSupervisorForTestV0) PrepareIdleSelfImprovementV0(
	_ context.Context,
	request IdleSelfImprovementRequestV0,
) (IdleSelfImprovementResultV0, error) {
	fake.prepareCalls++
	if fake.prepared != nil {
		fake.prepared <- struct{}{}
	}
	return IdleSelfImprovementResultV0{
		Accepted:   true,
		RequestRef: request.RequestRef,
		RunRef:     request.RequestRef,
		Status:     "prepared",
	}, nil
}

func (fake *goalFirstSupervisorForTestV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func (fake *goalFirstSupervisorForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	fake.launchCalls++
	fake.lastSpec = copyGoalWorkSpecForServerStateV0(spec)
	if fake.started != nil {
		fake.started <- struct{}{}
	}
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusAcceptedV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "external-" + spec.GoalRef,
		EvidenceRefs:    []string{"evidence-ref-goal-first-test"},
	}, nil
}

func goalWorkStateForServerTestV0(
	runRef string,
	goalRef string,
	externalGoalRef string,
	workKind string,
	status string,
) orquestagoal.GoalWorkStateV0 {
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      goalRef,
			RunRef:       runRef,
			Objective:    "goal activo de prueba",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WorkKind:     workKind,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          status,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
		},
	})
	if err != nil {
		panic(err)
	}
	return state
}

func containsGoalWorkStateRunRefForTestV0(
	states []orquestagoal.GoalWorkStateV0,
	runRef string,
) bool {
	for _, state := range states {
		if state.RunRef == runRef {
			return true
		}
	}
	return false
}

func containsGoalContextRefForTestV0(values []orquestagoal.GoalContextRefV0, target string) bool {
	for _, value := range values {
		if value.Ref == target {
			return true
		}
	}
	return false
}

func containsGoalRuleRefForTestV0(
	values []orquestagoal.GoalRuleRefV0,
	target string,
	enforcement string,
) bool {
	for _, value := range values {
		if value.Ref == target && value.Enforcement == enforcement {
			return true
		}
	}
	return false
}

type goalObservationSupervisorForTestV0 struct {
	observeCalls    int
	lastObservation orquestagoal.GoalObservationRequestV0
	result          orquestagoal.GoalWorkResultV0
}

func (fake *goalObservationSupervisorForTestV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func (fake *goalObservationSupervisorForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	fake.observeCalls++
	fake.lastObservation = request
	return fake.result, nil
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

func supervisorResultWithCrossRunLiveProcessSnapshotForTestV0(
	executionRunRef string,
	liveDiagnosticRunRef string,
) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
		TotalExecutions: 1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{RunRef: executionRunRef, Rank: 1}},
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:      executionRunRef,
					Outcome:     "process_ref_registered",
					QueueStatus: "running",
					Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
						Kind:   "process_runtime_snapshot",
						Status: "running",
						RunRef: liveDiagnosticRunRef,
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

type mutableClockForGoalFirstChainTestV0 struct {
	now time.Time
}

func (clock *mutableClockForGoalFirstChainTestV0) Now() time.Time {
	return clock.now
}

type goalFirstChainSupervisorForTestV0 struct {
	planBatches   [][]IdleSelfImprovementRequestV0
	planCalls     int
	launchCalls   int
	observeCalls  int
	selfCalls     int
	launchedSpecs []orquestagoal.GoalWorkSpecV0
	started       chan struct{}
}

func (fake *goalFirstChainSupervisorForTestV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func (fake *goalFirstChainSupervisorForTestV0) PlanIdleSelfImprovementV0(
	_ context.Context,
	request IdleSelfImprovementPlanRequestV0,
) (IdleSelfImprovementPlanResultV0, error) {
	fake.planCalls++
	index := fake.planCalls - 1
	if index < len(fake.planBatches) {
		return IdleSelfImprovementPlanResultV0{
			Requests: append([]IdleSelfImprovementRequestV0(nil), fake.planBatches[index]...),
		}, nil
	}
	return IdleSelfImprovementPlanResultV0{Requests: []IdleSelfImprovementRequestV0{request.BaseRequest}}, nil
}

func (fake *goalFirstChainSupervisorForTestV0) FilterIdleSelfImprovementRequestsV0(
	_ context.Context,
	request IdleSelfImprovementRequestFilterRequestV0,
) (IdleSelfImprovementRequestFilterResultV0, error) {
	return IdleSelfImprovementRequestFilterResultV0{Requests: append([]IdleSelfImprovementRequestV0(nil), request.Requests...)}, nil
}

func (fake *goalFirstChainSupervisorForTestV0) IdleSelfImprovementBlockersV0(
	context.Context,
	IdleSelfImprovementBlockerRequestV0,
) (IdleSelfImprovementBlockerResultV0, error) {
	return IdleSelfImprovementBlockerResultV0{}, nil
}

func (fake *goalFirstChainSupervisorForTestV0) RetryableIdleSelfImprovementRunRefsV0(
	context.Context,
	IdleSelfImprovementRunFreshnessRequestV0,
) (IdleSelfImprovementRunFreshnessResultV0, error) {
	return IdleSelfImprovementRunFreshnessResultV0{}, nil
}

func (fake *goalFirstChainSupervisorForTestV0) PrepareIdleSelfImprovementV0(
	context.Context,
	IdleSelfImprovementRequestV0,
) (IdleSelfImprovementResultV0, error) {
	fake.selfCalls++
	return IdleSelfImprovementResultV0{}, nil
}

func (fake *goalFirstChainSupervisorForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	fake.launchCalls++
	fake.launchedSpecs = append(fake.launchedSpecs, copyGoalWorkSpecForServerStateV0(spec))
	if fake.started != nil {
		fake.started <- struct{}{}
	}
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusAcceptedV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "external-" + spec.GoalRef,
		EvidenceRefs:    []string{"evidence-ref-goal-first-chain-launch"},
	}, nil
}

func (fake *goalFirstChainSupervisorForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	fake.observeCalls++
	var spec orquestagoal.GoalWorkSpecV0
	for _, candidate := range fake.launchedSpecs {
		if candidate.GoalRef == request.GoalRef {
			spec = candidate
			break
		}
	}
	result := orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		Summary:         "goal-first self-audit completo",
		ArtifactRefs:    []string{"artifact-ref-" + request.GoalRef},
		EvidenceRefs:    []string{"evidence-ref-required-test-001"},
	}
	for _, test := range spec.RequiredTests {
		result.RequiredTestResults = append(result.RequiredTestResults, orquestagoal.GoalRequiredTestResultV0{
			TestRef:      test.TestRef,
			Status:       "passed",
			EvidenceRefs: []string{"evidence-ref-required-test-001"},
		})
	}
	return result, nil
}

func idleSelfImprovementRequestForGoalFirstChainTestV0(
	suffix string,
	writeSet string,
) IdleSelfImprovementRequestV0 {
	return IdleSelfImprovementRequestV0{
		RequestRef:         "request-ref-self-audit-backlog-" + suffix,
		ProjectRef:         "project-ref-self-audit-chain",
		FailureSummary:     "hallazgo self-audit backlog " + suffix,
		WriteSet:           []string{writeSet},
		RequiredTests:      []string{"go test -count=1 ./" + writeSet},
		AcceptanceCriteria: []string{"cierre aceptado con evidencia durable " + suffix},
		ContextRefs:        []string{"context-ref-self-audit-backlog-" + suffix},
		EvidenceRefs:       []string{"evidence-ref-self-audit-backlog-" + suffix},
	}
}

func waitGoalFirstChainLaunchForTestV0(
	t *testing.T,
	supervisor *goalFirstChainSupervisorForTestV0,
) {
	t.Helper()
	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatalf("goal-first chain launch no observado")
	}
}
