package orquestaserver

import (
	"context"
	"errors"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
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

func TestRuntimeV0SupervisorDistingueColaIdleConGoalBackendActivoV0(t *testing.T) {
	now := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
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
	runtime.tracker.MarkGoalObserverV0(
		activeGoalObservationResultForSupervisorTestV0(
			"run-ref-bug-069-goal-backend-active-001",
			"goal-ref-bug-069-goal-backend-active-001",
		),
		now.Add(-time.Second),
	)

	runtime.runSupervisorTickV0(context.Background())

	if store.last.LastSupervisorQueueSize != 0 ||
		store.last.LastSupervisorStatus != SupervisorPublicStatusQueueIdleGoalBackendActiveV0 ||
		store.last.LastSupervisorStopPublic != SupervisorPublicStopQueueIdleButGoalBackendActiveV0 ||
		store.last.LastSupervisorStopCategory != SupervisorPublicCategoryGoalBackendV0 {
		t.Fatalf("state=%+v", store.last)
	}
	message := store.last.LastSupervisorOperationalMessage
	if message == nil ||
		message.ReasonCode != SupervisorPublicStatusQueueIdleGoalBackendActiveV0 ||
		message.Counters["goal_backend_active"] != 1 ||
		!containsStringForTestV0(message.RunRefs, "run-ref-bug-069-goal-backend-active-001") ||
		!containsStringForTestV0(message.GoalRefs, "goal-ref-bug-069-goal-backend-active-001") {
		t.Fatalf("operational message=%+v", message)
	}
	if store.last.IdleSelfImprovementReason != SupervisorPublicStopQueueIdleButGoalBackendActiveV0 {
		t.Fatalf("idle self-improvement reason=%q", store.last.IdleSelfImprovementReason)
	}
	events := readAuditEventsForTestV0(t, AuditPathV0(runtime.config))
	if len(events) < 2 {
		t.Fatalf("events=%+v", events)
	}
	summary, ok := events[1].Payload["result_summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("result_summary=%+v", events[1].Payload["result_summary"])
	}
	if summary["public_stop_reason"] != SupervisorPublicStopQueueIdleButGoalBackendActiveV0 ||
		summary["stop_category"] != SupervisorPublicCategoryGoalBackendV0 ||
		summary["goal_backend_active"] != float64(1) {
		t.Fatalf("summary=%+v", summary)
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

func activeGoalObservationResultForSupervisorTestV0(
	runRef string,
	goalRef string,
) orquestagoal.GoalWorkObserveActiveResultV0 {
	externalGoalRef := "external-" + goalRef
	evidenceRef := "evidence-ref-" + goalRef
	return orquestagoal.GoalWorkObserveActiveResultV0{
		Observations: []orquestagoal.GoalWorkObserveResultV0{{
			State: orquestagoal.GoalWorkStateV0{
				RunRef:          runRef,
				GoalRef:         goalRef,
				ExternalGoalRef: externalGoalRef,
				Status:          orquestagoal.GoalStatusRunningV0,
			},
			Result: orquestagoal.GoalWorkResultV0{
				Status:          orquestagoal.GoalStatusRunningV0,
				GoalRef:         goalRef,
				ExternalGoalRef: externalGoalRef,
				EvidenceRefs:    []string{evidenceRef},
			},
			EvidenceRefs: []string{evidenceRef},
		}},
		EvidenceRefs: []string{evidenceRef},
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
