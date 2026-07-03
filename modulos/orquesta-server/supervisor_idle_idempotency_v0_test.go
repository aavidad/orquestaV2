package orquestaserver

import (
	"context"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestRuntimeV0IdleSelfImprovementSaltaBloqueoIdenticoYPublicaWatchdogV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 15, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: false,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionObservedAtForTestV0(runtime, now.Add(-2*time.Minute))

	runtime.runSupervisorTickV0(context.Background())
	waitRuntimeAsyncWorkForTestV0(t, runtime)
	runtime.runSupervisorTickV0(context.Background())
	runtime.runSupervisorTickV0(context.Background())
	runtime.runSupervisorTickV0(context.Background(), supervisorTickCauseWatchdogV0)

	if supervisor.selfCalls != 1 {
		t.Fatalf("prepare repetido: selfCalls=%d refs=%v", supervisor.selfCalls, supervisor.selfRequestRefs)
	}
	events := readAuditEventsForTestV0(t, AuditPathV0(runtime.config))
	if got := countAuditEventsForTestV0(events, "idle_self_improvement_scheduled", "scheduled"); got != 1 {
		t.Fatalf("scheduled events=%d events=%+v", got, events)
	}
	if got := countAuditEventsForTestV0(events, "idle_self_improvement_check", "blocked"); got != 1 {
		t.Fatalf("blocked check events=%d events=%+v", got, events)
	}
	watchdog := findAuditEventWithSkippedIdenticalForTestV0(events)
	if watchdog == nil {
		t.Fatalf("watchdog compacto no publicado: events=%+v", events)
	}
	if watchdog.Payload["skipped_identical"] != float64(2) ||
		watchdog.Payload["status"] != "blocked" ||
		watchdog.Payload["tick_cause"] != supervisorTickCauseWatchdogV0 {
		t.Fatalf("watchdog payload=%+v", watchdog.Payload)
	}
}

func TestStatusTrackerV0IdleSelfImprovementErrorLiberaPublicacionParaRetryV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 15, 10, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{}, now)
	publication := newIdleSelfImprovementPublicationV0(
		"scheduled",
		"scheduled",
		[]string{"request-ref-idle-retry-001"},
	)
	if skip, skipped := tracker.RegisterIdleSelfImprovementPublicationV0(publication); skip || skipped != 0 {
		t.Fatalf("primera publicacion skip=%v skipped=%d", skip, skipped)
	}
	if skip, skipped := tracker.RegisterIdleSelfImprovementPublicationV0(publication); !skip || skipped != 1 {
		t.Fatalf("duplicado no bloqueado: skip=%v skipped=%d", skip, skipped)
	}

	tracker.MarkIdleSelfImprovementErrorV0("fallo_transitorio", now.Add(-61*time.Second))

	if skip, skipped := tracker.RegisterIdleSelfImprovementPublicationV0(publication); skip || skipped != 0 {
		t.Fatalf("error no libero retry identico: skip=%v skipped=%d", skip, skipped)
	}
}

func TestStatusTrackerV0PrepareFailedLiberaPublicacionParaRetryV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 15, 15, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{}, now)
	publication := newIdleSelfImprovementPublicationV0(
		"scheduled",
		"scheduled",
		[]string{"request-ref-idle-prepare-failed-001"},
	)
	if skip, skipped := tracker.RegisterIdleSelfImprovementPublicationV0(publication); skip || skipped != 0 {
		t.Fatalf("primera publicacion skip=%v skipped=%d", skip, skipped)
	}
	if skip, skipped := tracker.RegisterIdleSelfImprovementPublicationV0(publication); !skip || skipped != 1 {
		t.Fatalf("duplicado no bloqueado: skip=%v skipped=%d", skip, skipped)
	}

	tracker.MarkIdleSelfImprovementPrepareFailedV0(IdleSelfImprovementResultV0{
		RequestRef: "request-ref-idle-prepare-failed-001",
		Status:     "error",
		Message:    "prepare_failed",
	}, now.Add(-61*time.Second))

	if skip, skipped := tracker.RegisterIdleSelfImprovementPublicationV0(publication); skip || skipped != 0 {
		t.Fatalf("prepare_failed no libero retry identico: skip=%v skipped=%d", skip, skipped)
	}
}

func TestRuntimeV0SupervisorDespiertaPorGoalTerminalObservadoV0(t *testing.T) {
	supervisor := &blockingGoalObserverSupervisorV0{
		blockingSupervisorV0: newBlockingSupervisorV0(),
		result: orquestagoal.GoalWorkObserveActiveResultV0{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: orquestagoal.GoalWorkStateV0{
					RunRef:  "run-ref-terminal-wakeup-001",
					GoalRef: "goal-ref-terminal-wakeup-001",
					Status:  orquestagoal.GoalStatusCompleteV0,
				},
				Result: orquestagoal.GoalWorkResultV0{
					Status:       orquestagoal.GoalStatusCompleteV0,
					GoalRef:      "goal-ref-terminal-wakeup-001",
					EvidenceRefs: []string{"evidence-ref-terminal-wakeup-001"},
				},
				Terminal: true,
			}},
		},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:             t.TempDir(),
		TickInterval:         time.Hour,
		GoalObserverInterval: time.Hour,
		AuditDisabled:        true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		StateStore:     &memoryStateStoreV0{},
		GoalStateStore: newMemoryGoalStateStoreV0(),
		Clock:          fixedClockV0{now: time.Date(2026, 7, 3, 15, 30, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runtime.runSupervisorLoopV0(ctx)

	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatalf("tick inicial no arranco")
	}
	supervisor.release()
	supervisor.waitDone(t)
	waitRuntimeAsyncWorkForTestV0(t, runtime)

	runtime.runGoalObservationTickV0(ctx)

	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatalf("terminal de goal no desperto supervisor por canal")
	}
	supervisor.release()
	supervisor.waitDone(t)
}

type blockingGoalObserverSupervisorV0 struct {
	*blockingSupervisorV0
	result orquestagoal.GoalWorkObserveActiveResultV0
}

func (supervisor *blockingGoalObserverSupervisorV0) ObserveActiveGoalWorksV0(
	context.Context,
	orquestagoal.GoalWorkObserveActiveRequestV0,
) (orquestagoal.GoalWorkObserveActiveResultV0, error) {
	return supervisor.result, nil
}

func countAuditEventsForTestV0(events []AuditEventV0, eventName string, status string) int {
	count := 0
	for _, event := range events {
		if event.Event == eventName && event.Status == status {
			count++
		}
	}
	return count
}

func findAuditEventWithSkippedIdenticalForTestV0(events []AuditEventV0) *AuditEventV0 {
	for index := range events {
		if events[index].Event == "idle_self_improvement_check" &&
			events[index].Status == "skipped" &&
			events[index].Payload["skipped_identical"] != nil {
			return &events[index]
		}
	}
	return nil
}
