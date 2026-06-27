package orquestaserver

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
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
	if store.last.IdleSelfImprovementReason == "" ||
		!strings.Contains(store.last.IdleSelfImprovementReason, "goal_ref="+spec.GoalRef) ||
		store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementGoalSpec == nil ||
		store.last.IdleSelfImprovementGoalSpec.GoalRef != spec.GoalRef ||
		store.last.IdleSelfImprovementGoalReceipt == nil ||
		store.last.IdleSelfImprovementGoalReceipt.GoalRef != spec.GoalRef ||
		!containsStringForTestV0(store.last.IdleSelfImprovementOperationalMessage.GoalRefs, spec.GoalRef) ||
		!containsStringForTestV0(store.last.IdleSelfImprovementOperationalMessage.EvidenceRefs, "evidence-ref-idle-self-improvement-goal-first-launched") {
		t.Fatalf("state=%+v", store.last)
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
	if store.last.IdleSelfImprovementReason != idleSelfImprovementGoalLauncherUnavailableReasonV0 {
		t.Fatalf("reason=%q state=%+v", store.last.IdleSelfImprovementReason, store.last)
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
	if store.last.IdleSelfImprovementReason == "" ||
		!strings.Contains(store.last.IdleSelfImprovementReason, idleSelfImprovementGoalRunningReasonV0) ||
		store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalRunningReasonV0 ||
		!containsStringForTestV0(store.last.IdleSelfImprovementOperationalMessage.GoalRefs, "goal-ref-autoprogramming-observe-001") {
		t.Fatalf("state=%+v", store.last)
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
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalCompletePendingClosureV0 ||
		store.last.IdleSelfImprovementGoalClosure != nil ||
		store.last.IdleSelfImprovementOK != 1 {
		t.Fatalf("state=%+v", store.last)
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
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalClosureAcceptedReasonV0 ||
		store.last.IdleSelfImprovementOperationalMessage.Status != orquestagoal.GoalStatusAcceptedV0 ||
		store.last.IdleSelfImprovementGoalResult == nil ||
		store.last.IdleSelfImprovementGoalResult.GoalRef != goalRef ||
		store.last.IdleSelfImprovementGoalClosure == nil ||
		!store.last.IdleSelfImprovementGoalClosure.Accepted ||
		!containsStringForTestV0(store.last.IdleSelfImprovementOperationalMessage.EvidenceRefs, "evidence-ref-required-closure") {
		t.Fatalf("state=%+v", store.last)
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

type goalFirstSupervisorForTestV0 struct {
	launchCalls int
	lastSpec    orquestagoal.GoalWorkSpecV0
	started     chan struct{}
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
