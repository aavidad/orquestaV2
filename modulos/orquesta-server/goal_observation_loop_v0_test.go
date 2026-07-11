package orquestaserver

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0GoalObservationTickCorreSinDirectorResidenteV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: orquestagoal.GoalWorkStateV0{
					RunRef:  "run-ref-goal-observer-001",
					GoalRef: "goal-ref-goal-observer-001",
					Status:  orquestagoal.GoalStatusCompleteV0,
				},
				Result: orquestagoal.GoalWorkResultV0{
					Status:       orquestagoal.GoalStatusCompleteV0,
					GoalRef:      "goal-ref-goal-observer-001",
					EvidenceRefs: []string{"evidence-ref-goal-observer-001"},
				},
				Terminal:     true,
				Accepted:     true,
				EvidenceRefs: []string{"evidence-ref-goal-observer-001"},
			}},
			EvidenceRefs: []string{"evidence-ref-goal-observer-001"},
		}},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     store,
		Clock:          fixedClockV0{now: time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runGoalObservationTickV0(context.Background())

	if supervisor.goalObservationCalls != 1 ||
		!supervisor.lastGoalObservation.List.ActiveOnly ||
		supervisor.lastGoalObservation.List.MaxItems != 5 {
		t.Fatalf("goal calls=%d request=%+v", supervisor.goalObservationCalls, supervisor.lastGoalObservation)
	}
	if store.snapshotV0().GoalObserverStatus != "ok" ||
		store.snapshotV0().GoalObserverTicks != 1 ||
		store.snapshotV0().GoalObserverObserved != 1 ||
		store.snapshotV0().GoalObserverTerminal != 1 ||
		store.snapshotV0().GoalObserverErrorTicks != 0 {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0GoalObservationTickActualizaIdleSelfImprovementV0(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	const goalRef = "goal-ref-idle-generic-observer-001"
	const externalGoalRef = "external-goal-ref-idle-generic-observer-001"
	supervisor := &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: orquestagoal.GoalWorkStateV0{
					RunRef:          "run-ref-idle-generic-observer-001",
					GoalRef:         goalRef,
					ExternalGoalRef: externalGoalRef,
					Status:          orquestagoal.GoalStatusCompleteV0,
				},
				Result: orquestagoal.GoalWorkResultV0{
					Status:          orquestagoal.GoalStatusCompleteV0,
					GoalRef:         goalRef,
					ExternalGoalRef: externalGoalRef,
					Summary:         "automejora cerrada por observer generico",
					EvidenceRefs:    []string{"evidence-ref-idle-generic-required-001"},
				},
				Terminal:     true,
				Accepted:     true,
				EvidenceRefs: []string{"evidence-ref-idle-generic-required-001"},
			}},
			EvidenceRefs: []string{"evidence-ref-idle-generic-required-001"},
		}},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     store,
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted:        true,
		Status:          orquestagoal.GoalStatusAcceptedV0,
		Message:         "goal_first_launched",
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		GoalSpec: orquestagoal.GoalWorkSpecV0{
			GoalRef:       goalRef,
			Objective:     "cerrar automejora desde observer generico",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}},
			ClosurePolicy: orquestagoal.GoalClosurePolicyV0{RequiredEvidenceRefs: []string{"evidence-ref-idle-generic-required-001"}},
		},
		GoalReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
		},
	}, now.Add(-time.Minute))

	runtime.runGoalObservationTickV0(context.Background())

	if supervisor.goalObservationCalls != 1 {
		t.Fatalf("goal observer calls=%d", supervisor.goalObservationCalls)
	}
	if store.snapshotV0().GoalObserverStatus != "ok" ||
		store.snapshotV0().GoalObserverTicks != 1 ||
		store.snapshotV0().GoalObserverTerminal != 1 {
		t.Fatalf("goal observer state=%+v", store.snapshotV0())
	}
	if store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalClosureAcceptedReasonV0 ||
		store.snapshotV0().IdleSelfImprovementGoalResult == nil ||
		store.snapshotV0().IdleSelfImprovementGoalResult.GoalRef != goalRef ||
		store.snapshotV0().IdleSelfImprovementGoalClosure == nil ||
		!store.snapshotV0().IdleSelfImprovementGoalClosure.Accepted ||
		store.snapshotV0().IdleSelfImprovementFlight ||
		store.snapshotV0().LastError != "" {
		t.Fatalf("idle state no actualizado por observer generico: %+v", store.snapshotV0())
	}
}

func TestRuntimeV0GoalObservationHighConsumptionBloqueaIdleGoalSinProgresoV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 19, 0, 0, 0, time.UTC)
	const runRef = "run-ref-idle-progress-governor-001"
	const goalRef = "goal-ref-idle-progress-governor-001"
	const externalGoalRef = "external-goal-ref-idle-progress-governor-001"
	state := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, externalGoalRef)
	store := &memoryStateStoreV0{}
	goalStore := newMemoryGoalStateStoreV0()
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	supervisor := &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: state,
				Result: orquestagoal.GoalWorkResultV0{
					Status:          orquestagoal.GoalStatusRunningV0,
					GoalRef:         goalRef,
					ExternalGoalRef: externalGoalRef,
					ContextBudget:   orquestagoal.GoalContextBudgetV0{ContextBudgetTotalBytes: 250},
					EvidenceRefs:    []string{"evidence-ref-codex-goal-cached-input-tokens-250"},
				},
			}},
		}},
	}
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
		SelfWatchdog:                  SelfWatchdogConfigV0{NoProgressFor: time.Minute},
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStore,
		GoalStopper:    stopper,
		StateStore:     store,
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(idleSelfImprovementPreparedForProgressTestV0(state), now.Add(-3*time.Minute))
	runtime.tracker.MarkIdleSelfImprovementGoalProgressV0(idleSelfImprovementGoalProgressUpdateV0{
		UsefulProgressAt:        now.Add(-2 * time.Minute),
		UsefulProgressSignature: "sha256:previous-useful-progress",
		ObservedConsumption:     100,
	}, now.Add(-2*time.Minute))

	runtime.runGoalObservationTickV0(context.Background())

	if stopper.calls != 1 ||
		stopper.last.RunRef != runRef ||
		!stopper.last.RequireConfirmedBackendStop ||
		stopper.last.Reason != idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 ||
		stopper.last.RecommendedAction != idleSelfImprovementGoalReviewReplanRecommendedActionV0 {
		t.Fatalf("stopper calls=%d request=%+v", stopper.calls, stopper.last)
	}
	if store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 ||
		store.snapshotV0().IdleSelfImprovementGoalResult == nil ||
		store.snapshotV0().IdleSelfImprovementGoalResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		!containsStringForTestV0(store.snapshotV0().IdleSelfImprovementGoalResult.ReworkPlanRefs, idleSelfImprovementGoalReviewReplanRefV0) ||
		!containsStringForTestV0(store.snapshotV0().IdleSelfImprovementGoalResult.EvidenceRefs, "evidence-ref-goal-cooperative-stop-requested") ||
		!goalWorkResultHasIssueCodeV0(*store.snapshotV0().IdleSelfImprovementGoalResult, idleSelfImprovementGoalHighConsumptionNoProgressReasonV0) {
		t.Fatalf("idle state=%+v", store.snapshotV0())
	}
	loaded, err := goalStore.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if loaded.Status != orquestagoal.GoalStatusBlockedV0 ||
		loaded.LastResult == nil ||
		loaded.LastResult.Summary != idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 {
		t.Fatalf("goal state=%+v", loaded)
	}
}

func TestRuntimeV0GoalObservationNoCortaConProgresoUtilRecienteV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 19, 15, 0, 0, time.UTC)
	const runRef = "run-ref-idle-progress-governor-002"
	const goalRef = "goal-ref-idle-progress-governor-002"
	const externalGoalRef = "external-goal-ref-idle-progress-governor-002"
	state := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, externalGoalRef)
	store := &memoryStateStoreV0{}
	goalStore := newMemoryGoalStateStoreV0()
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	supervisor := &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: state,
				Result: orquestagoal.GoalWorkResultV0{
					Status:          orquestagoal.GoalStatusRunningV0,
					GoalRef:         goalRef,
					ExternalGoalRef: externalGoalRef,
					ContextBudget:   orquestagoal.GoalContextBudgetV0{ContextBudgetTotalBytes: 100000},
					ArtifactPaths:   []string{"modulos/orquesta-server/generated_progress_marker.go"},
					EvidenceRefs:    []string{"evidence-ref-codex-goal-cached-input-tokens-100000"},
				},
			}},
		}},
	}
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
		SelfWatchdog:                  SelfWatchdogConfigV0{NoProgressFor: time.Minute},
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStore,
		GoalStopper:    stopper,
		StateStore:     store,
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(idleSelfImprovementPreparedForProgressTestV0(state), now.Add(-3*time.Minute))
	runtime.tracker.MarkIdleSelfImprovementGoalProgressV0(idleSelfImprovementGoalProgressUpdateV0{
		UsefulProgressAt:        now.Add(-2 * time.Minute),
		UsefulProgressSignature: "sha256:previous-useful-progress",
		ObservedConsumption:     100,
	}, now.Add(-2*time.Minute))

	runtime.runGoalObservationTickV0(context.Background())

	if stopper.calls != 0 {
		t.Fatalf("goal con progreso util no debe parar: calls=%d request=%+v", stopper.calls, stopper.last)
	}
	if store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalRunningReasonV0 ||
		store.snapshotV0().IdleSelfImprovementGoalResult == nil ||
		store.snapshotV0().IdleSelfImprovementGoalResult.Status != orquestagoal.GoalStatusRunningV0 ||
		store.snapshotV0().IdleSelfImprovementGoalUsefulProgressAt != formatTimeV0(now) ||
		store.snapshotV0().IdleSelfImprovementGoalUsefulProgressSignature == "sha256:previous-useful-progress" {
		t.Fatalf("idle state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0GoalObservationCheckpointInvalidoRepetidoCuentaSinProgresoV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 19, 30, 0, 0, time.UTC)
	const runRef = "run-ref-idle-progress-governor-003"
	const goalRef = "goal-ref-idle-progress-governor-003"
	const externalGoalRef = "external-goal-ref-idle-progress-governor-003"
	state := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, externalGoalRef)
	invalidCheckpoint := orquestagoal.GoalMaterializedArtifactV0{
		ArtifactRef:  "artifact-ref-invalid-checkpoint-001",
		Path:         "modulos/orquesta-server/checkpoint_started_t290.txt",
		ArtifactType: "checkpoint",
		Status:       "invalid",
	}
	invalidSignature := idleSelfImprovementGoalInvalidCheckpointSignatureV0(orquestagoal.GoalWorkResultV0{
		MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{invalidCheckpoint},
	})
	store := &memoryStateStoreV0{}
	goalStore := newMemoryGoalStateStoreV0()
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	supervisor := &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: state,
				Result: orquestagoal.GoalWorkResultV0{
					Status:                orquestagoal.GoalStatusRunningV0,
					GoalRef:               goalRef,
					ExternalGoalRef:       externalGoalRef,
					ContextBudget:         orquestagoal.GoalContextBudgetV0{ContextBudgetTotalBytes: 300},
					MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{invalidCheckpoint},
					EvidenceRefs:          []string{"evidence-ref-codex-goal-cached-input-tokens-300"},
				},
			}},
		}},
	}
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
		SelfWatchdog:                  SelfWatchdogConfigV0{NoProgressFor: time.Minute},
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStore,
		GoalStopper:    stopper,
		StateStore:     store,
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(idleSelfImprovementPreparedForProgressTestV0(state), now.Add(-3*time.Minute))
	runtime.tracker.MarkIdleSelfImprovementGoalProgressV0(idleSelfImprovementGoalProgressUpdateV0{
		UsefulProgressAt:           now.Add(-2 * time.Minute),
		UsefulProgressSignature:    "sha256:previous-useful-progress",
		ObservedConsumption:        100,
		InvalidCheckpointSignature: invalidSignature,
		InvalidCheckpointRepeats:   1,
	}, now.Add(-2*time.Minute))

	runtime.runGoalObservationTickV0(context.Background())

	if stopper.calls != 1 {
		t.Fatalf("checkpoint invalido repetido debe cortar por sin-progreso: calls=%d", stopper.calls)
	}
	if store.snapshotV0().IdleSelfImprovementGoalInvalidCheckpointRepeats != 2 ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 ||
		store.snapshotV0().IdleSelfImprovementGoalResult == nil ||
		!containsStringForTestV0(store.snapshotV0().IdleSelfImprovementGoalResult.EvidenceRefs, "evidence-ref-goal-invalid-checkpoint-repeated") {
		t.Fatalf("idle state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0GoalObserverHighConsumptionCheckpointOnlyPideStopCooperativoV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 22, 45, 0, 0, time.UTC)
	const runRef = "run-ref-goal-observer-checkpoint-only-001"
	const goalRef = "goal-ref-goal-observer-checkpoint-only-001"
	const externalGoalRef = "external-goal-ref-goal-observer-checkpoint-only-001"
	state := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, externalGoalRef)
	goalStore := newMemoryGoalStateStoreV0()
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	serverStore := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: state,
				Result: orquestagoal.GoalWorkResultV0{
					Status:          orquestagoal.GoalStatusRunningV0,
					GoalRef:         goalRef,
					ExternalGoalRef: externalGoalRef,
					Summary:         "codex_app_server_goal_status_active_high_token_usage tokens_used=139028",
					EvidenceRefs: []string{
						"evidence-ref-codex-app-server-goal-high-token-usage",
						goalObserverAppServerEarlyCheckpointPrefixV0 + ".orquesta-runtime/goal-receipts/goal-ref-001/checkpoint_started.txt",
					},
				},
			}},
		}},
	}
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStore,
		GoalStopper:    stopper,
		StateStore:     serverStore,
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runGoalObservationTickV0(context.Background())

	if stopper.calls != 1 ||
		stopper.last.RunRef != runRef ||
		!stopper.last.RequireConfirmedBackendStop ||
		stopper.last.Reason != goalObserverHighConsumptionCheckpointOnlyReasonV0 ||
		stopper.last.RecommendedAction != goalObserverHighConsumptionRecommendedActionV0 ||
		stopper.last.RequestedBy != goalObserverHighConsumptionRequestedByV0 {
		t.Fatalf("stopper calls=%d request=%+v", stopper.calls, stopper.last)
	}
	for _, want := range []string{
		goalObserverCheckpointOnlyHighConsumptionRefV0,
		goalObserverHighConsumptionStopEvidenceV0,
		"evidence-ref-codex-app-server-goal-high-token-usage",
	} {
		if !containsStringForTestV0(stopper.last.EvidenceRefs, want) {
			t.Fatalf("stopper evidence refs=%v want %s", stopper.last.EvidenceRefs, want)
		}
	}
	loaded, err := goalStore.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if loaded.Status != orquestagoal.GoalStatusRunningV0 ||
		loaded.LastResult == nil ||
		loaded.LastResult.Status != orquestagoal.GoalStatusRunningV0 ||
		loaded.LastClosure != nil ||
		!goalWorkResultHasIssueCodeV0(*loaded.LastResult, goalObserverHighConsumptionCheckpointOnlyReasonV0) {
		t.Fatalf("goal state=%+v", loaded)
	}
	if serverStore.snapshotV0().GoalObserverTerminal != 0 ||
		serverStore.snapshotV0().GoalObserverIssues != 1 {
		t.Fatalf("server state=%+v", serverStore.snapshotV0())
	}
}

func TestGoalObserverEvidenceCheckpointSoloAceptaRefsCanonicasV0(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "checkpoint-started", ref: goalObserverAppServerCheckpointStartedEvidenceV0, want: true},
		{
			name: "early-checkpoint-materialized",
			ref:  goalObserverAppServerEarlyCheckpointPrefixV0 + ".orquesta-runtime/goal-receipts/goal-ref-001/checkpoint_started.txt",
			want: true,
		},
		{name: "prefijo-sin-ref", ref: goalObserverAppServerEarlyCheckpointPrefixV0, want: false},
		{name: "historica", ref: "evidence-ref-historical-checkpoint-review-2026", want: false},
		{name: "no-checkpoint", ref: goalObserverNoCheckpointHighConsumptionRefV0, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := goalObserverEvidenceRefLooksLikeCheckpointV0(test.ref); got != test.want {
				t.Fatalf("ref=%q got=%t want=%t", test.ref, got, test.want)
			}
		})
	}
}

func TestRuntimeV0GoalObserverHighConsumptionSinCheckpointPideStopCooperativoV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 23, 0, 0, 0, time.UTC)
	const runRef = "run-ref-goal-observer-no-checkpoint-001"
	const goalRef = "goal-ref-goal-observer-no-checkpoint-001"
	const externalGoalRef = "external-goal-ref-goal-observer-no-checkpoint-001"
	state := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, externalGoalRef)
	goalStore := newMemoryGoalStateStoreV0()
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	supervisor := &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: state,
				Result: orquestagoal.GoalWorkResultV0{
					Status:          orquestagoal.GoalStatusRunningV0,
					GoalRef:         goalRef,
					ExternalGoalRef: externalGoalRef,
					Summary:         "codex_app_server_goal_status_active_high_token_usage tokens_used=166009",
					EvidenceRefs: []string{
						"evidence-ref-codex-app-server-goal-high-token-usage",
						"evidence-ref-historical-checkpoint-review-2026",
						goalObserverNoCheckpointHighConsumptionRefV0,
					},
				},
			}},
		}},
	}
	supervisor.goalObservationResult = append(
		supervisor.goalObservationResult,
		supervisor.goalObservationResult[0],
	)
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStore,
		GoalStopper:    stopper,
		StateStore:     &memoryStateStoreV0{},
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runGoalObservationTickV0(context.Background())
	runtime.runGoalObservationTickV0(context.Background())

	if stopper.calls != 1 ||
		!stopper.last.RequireConfirmedBackendStop ||
		stopper.last.Reason != goalObserverHighConsumptionNoCheckpointReasonV0 ||
		stopper.last.RecommendedAction != goalObserverHighConsumptionRecommendedActionV0 {
		t.Fatalf("stopper calls=%d request=%+v", stopper.calls, stopper.last)
	}
	if !containsStringForTestV0(stopper.last.EvidenceRefs, goalObserverNoCheckpointHighConsumptionRefV0) {
		t.Fatalf("stopper evidence refs=%v", stopper.last.EvidenceRefs)
	}
	loaded, err := goalStore.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if loaded.Status != orquestagoal.GoalStatusRunningV0 ||
		loaded.LastResult == nil ||
		loaded.LastResult.Status != orquestagoal.GoalStatusRunningV0 ||
		loaded.LastClosure != nil ||
		!goalWorkResultHasIssueCodeV0(*loaded.LastResult, goalObserverHighConsumptionNoCheckpointReasonV0) {
		t.Fatalf("goal state=%+v", loaded)
	}
}

func TestRuntimeV0GoalObserverAltoConsumoAdvisoryPermiteCierrePosteriorV0(t *testing.T) {
	ctx := context.Background()
	const runRef = "run-ref-goal-observer-advisory-reconcile-001"
	const goalRef = "goal-ref-goal-observer-advisory-reconcile-001"
	goalStore := newMemoryGoalStateStoreV0()
	start, err := orquestagoal.StartGoalWorkV0(
		ctx,
		orquestagoal.GoalWorkStartRequestV0{
			RunRef: runRef,
			Spec: orquestagoal.GoalWorkSpecV0{
				GoalRef:      goalRef,
				Objective:    "Permitir que un cierre real supere un aviso de consumo.",
				DirectorKind: orquestagoal.GoalDirectorKindRuntimeGoalV0,
				WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}},
			},
		},
		orquestagoal.GoalWorkLifecyclePortsV0{
			Launcher:   goalObservationLauncherForTestV0{},
			StateStore: goalStore,
		},
	)
	if err != nil {
		t.Fatalf("StartGoalWorkV0: %v", err)
	}
	observer := &goalObservationSequenceObserverForTestV0{results: []orquestagoal.GoalWorkResultV0{
		{
			Status:       orquestagoal.GoalStatusRunningV0,
			GoalRef:      goalRef,
			Summary:      "codex_app_server_goal_status_active_high_token_usage tokens_used=166009",
			EvidenceRefs: []string{"evidence-ref-codex-app-server-goal-high-token-usage"},
		},
		{
			Status:       orquestagoal.GoalStatusCompleteV0,
			GoalRef:      goalRef,
			EvidenceRefs: []string{"evidence-ref-goal-observer-real-complete-001"},
		},
	}}
	supervisor := newGoalObservationLifecycleSupervisorForTestV0(
		goalStore,
		observer,
		goalObservationClosureValidatorForTestV0{},
	)
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	serverStore := &memoryStateStoreV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStore,
		GoalStopper:    stopper,
		StateStore:     serverStore,
		Clock:          fixedClockV0{now: time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runGoalObservationTickV0(ctx)
	advisory, err := goalStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 advisory: %v", err)
	}
	if advisory.Status != orquestagoal.GoalStatusRunningV0 ||
		advisory.LastResult == nil ||
		advisory.LastResult.Status != orquestagoal.GoalStatusRunningV0 ||
		advisory.LastClosure != nil ||
		stopper.calls != 1 ||
		serverStore.snapshotV0().GoalObserverTerminal != 0 {
		t.Fatalf("advisory state=%+v stopper=%+v server=%+v", advisory, stopper, serverStore.snapshotV0())
	}

	runtime.runGoalObservationTickV0(ctx)
	completed, err := goalStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 complete: %v", err)
	}
	if completed.Status != orquestagoal.GoalStatusCompleteV0 ||
		completed.LastResult == nil ||
		completed.LastResult.Status != orquestagoal.GoalStatusCompleteV0 ||
		completed.LastClosure == nil ||
		!completed.LastClosure.Accepted ||
		stopper.calls != 1 ||
		serverStore.snapshotV0().GoalObserverTerminal != 1 {
		t.Fatalf("completed state=%+v stopper=%+v server=%+v", completed, stopper, serverStore.snapshotV0())
	}
	if start.State.Status != orquestagoal.GoalStatusRunningV0 || observer.calls != 2 {
		t.Fatalf("start=%+v observer calls=%d", start, observer.calls)
	}
}

func TestRuntimeV0GoalObserverAdvisoryNoSobrescribeTerminalPreexistenteV0(t *testing.T) {
	for _, status := range []string{
		orquestagoal.GoalStatusCompleteV0,
		orquestagoal.GoalStatusBlockedV0,
		orquestagoal.GoalStatusInvalidV0,
	} {
		t.Run(status, func(t *testing.T) {
			ctx := context.Background()
			runRef := "run-ref-goal-observer-terminal-race-" + status
			goalRef := "goal-ref-goal-observer-terminal-race-" + status
			observedState := mustIdleSelfImprovementGoalStateForProgressTestV0(
				t,
				runRef,
				goalRef,
				"external-"+goalRef,
			)
			terminalEvidence := "evidence-ref-goal-observer-terminal-preexisting-" + status
			terminalResult := orquestagoal.GoalWorkResultV0{
				Status:       status,
				GoalRef:      goalRef,
				EvidenceRefs: []string{terminalEvidence},
			}
			closure := orquestagoal.GoalClosureValidationV0{
				Status:       orquestagoal.GoalStatusBlockedV0,
				NeedsRework:  true,
				EvidenceRefs: []string{terminalEvidence},
			}
			if status == orquestagoal.GoalStatusCompleteV0 {
				closure.Status = orquestagoal.GoalStatusAcceptedV0
				closure.Accepted = true
				closure.NeedsRework = false
			}
			terminalState := observedState
			terminalState.Status = status
			terminalState.LastResult = &terminalResult
			terminalState.LastClosure = &closure
			terminalState.EvidenceRefs = append(
				append([]string(nil), observedState.EvidenceRefs...),
				terminalEvidence,
			)
			goalStore := newMemoryGoalStateStoreV0()
			if err := goalStore.SaveGoalWorkStateV0(ctx, terminalState); err != nil {
				t.Fatalf("SaveGoalWorkStateV0: %v", err)
			}
			staleObservation := orquestagoal.GoalWorkObserveActiveResultV0{
				Observations: []orquestagoal.GoalWorkObserveResultV0{{
					State: observedState,
					Result: orquestagoal.GoalWorkResultV0{
						Status:       orquestagoal.GoalStatusRunningV0,
						GoalRef:      goalRef,
						Summary:      "codex_app_server_goal_status_active_high_token_usage tokens_used=190000",
						EvidenceRefs: []string{"evidence-ref-codex-app-server-goal-high-token-usage"},
					},
				}},
			}
			supervisor := &fakeSupervisorV0{
				goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{staleObservation},
			}
			stopper := &fakeGoalCooperativeStopperForTestV0{}
			serverStore := &memoryStateStoreV0{}
			runtime, err := NewRuntimeV0(ConfigV0{
				StateDir:                      t.TempDir(),
				TickInterval:                  time.Hour,
				GoalObserverEnabledConfigured: true,
				GoalObserverEnabled:           true,
				GoalObserverMaxItems:          5,
				ResidentDirectorEnabled:       false,
				AuditDisabled:                 true,
			}, RuntimeDepsV0{
				Supervisor:     supervisor,
				GoalStateStore: goalStore,
				GoalStopper:    stopper,
				StateStore:     serverStore,
			})
			if err != nil {
				t.Fatalf("NewRuntimeV0: %v", err)
			}

			reconciled := runtime.reconcileGoalObserverHighConsumptionV0(ctx, staleObservation)
			if len(reconciled.Observations) != 1 ||
				!reconciled.Observations[0].Terminal ||
				reconciled.Observations[0].State.Status != status ||
				reconciled.Observations[0].Result.Status != status {
				t.Fatalf("reconciled=%+v", reconciled)
			}
			runtime.runGoalObservationTickV0(ctx)

			loaded, err := goalStore.LoadGoalWorkStateV0(ctx, runRef)
			if err != nil {
				t.Fatalf("LoadGoalWorkStateV0: %v", err)
			}
			if loaded.Status != status ||
				loaded.LastResult == nil ||
				loaded.LastResult.Status != status ||
				loaded.LastClosure == nil ||
				loaded.LastClosure.Status != closure.Status ||
				!containsStringForTestV0(loaded.EvidenceRefs, terminalEvidence) ||
				stopper.calls != 0 ||
				serverStore.snapshotV0().GoalObserverTerminal != 1 {
				t.Fatalf("terminal state=%+v stopper=%+v server=%+v", loaded, stopper, serverStore.snapshotV0())
			}
		})
	}
}

func TestRuntimeV0GoalObserverPublicaTerminalQueApareceDuranteGovernanceV0(t *testing.T) {
	ctx := context.Background()
	const runRef = "run-ref-goal-observer-terminal-during-governance"
	const goalRef = "goal-ref-goal-observer-terminal-during-governance"
	running := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, "external-"+goalRef)
	terminal := running
	terminal.Status = orquestagoal.GoalStatusCompleteV0
	terminalResult := orquestagoal.GoalWorkResultV0{
		Status:       orquestagoal.GoalStatusCompleteV0,
		GoalRef:      goalRef,
		EvidenceRefs: []string{"evidence-ref-terminal-during-governance"},
	}
	terminalClosure := orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusAcceptedV0,
		Accepted:     true,
		EvidenceRefs: []string{"evidence-ref-terminal-during-governance"},
	}
	terminal.LastResult = &terminalResult
	terminal.LastClosure = &terminalClosure
	goalStore := &goalStateTerminalDuringGovernanceStoreV0{running: running, terminal: terminal}
	stale := orquestagoal.GoalWorkObserveActiveResultV0{Observations: []orquestagoal.GoalWorkObserveResultV0{{
		State: running,
		Result: orquestagoal.GoalWorkResultV0{
			Status:       orquestagoal.GoalStatusRunningV0,
			GoalRef:      goalRef,
			Summary:      "codex_app_server_goal_status_active_high_token_usage tokens_used=190000",
			EvidenceRefs: []string{"evidence-ref-codex-app-server-goal-high-token-usage"},
		},
	}}}
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	serverStore := &memoryStateStoreV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir: t.TempDir(), TickInterval: time.Hour,
		GoalObserverEnabledConfigured: true, GoalObserverEnabled: true,
		GoalObserverMaxItems: 5, AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor:     &fakeSupervisorV0{goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{stale}},
		GoalStateStore: goalStore, GoalStopper: stopper, StateStore: serverStore,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runGoalObservationTickV0(ctx)

	if stopper.calls != 0 || goalStore.saves != 0 || serverStore.snapshotV0().GoalObserverTerminal != 1 {
		t.Fatalf("stopper=%+v store=%+v server=%+v", stopper, goalStore, serverStore.snapshotV0())
	}
}

type goalStateTerminalDuringGovernanceStoreV0 struct {
	running  orquestagoal.GoalWorkStateV0
	terminal orquestagoal.GoalWorkStateV0
	loads    int
	saves    int
}

func (store *goalStateTerminalDuringGovernanceStoreV0) LoadGoalWorkStateV0(
	context.Context,
	string,
) (orquestagoal.GoalWorkStateV0, error) {
	store.loads++
	if store.loads == 1 {
		return store.running, nil
	}
	return store.terminal, nil
}

func (store *goalStateTerminalDuringGovernanceStoreV0) SaveGoalWorkStateV0(
	context.Context,
	orquestagoal.GoalWorkStateV0,
) error {
	store.saves++
	return nil
}

func TestRuntimeV0GoalObserverNoParaSiHayArtefactoUtilV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 23, 15, 0, 0, time.UTC)
	const runRef = "run-ref-goal-observer-useful-artifact-001"
	const goalRef = "goal-ref-goal-observer-useful-artifact-001"
	const externalGoalRef = "external-goal-ref-goal-observer-useful-artifact-001"
	state := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, externalGoalRef)
	goalStore := newMemoryGoalStateStoreV0()
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	supervisor := &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: state,
				Result: orquestagoal.GoalWorkResultV0{
					Status:          orquestagoal.GoalStatusRunningV0,
					GoalRef:         goalRef,
					ExternalGoalRef: externalGoalRef,
					Summary:         "codex_app_server_goal_status_active_high_token_usage tokens_used=120000",
					ArtifactPaths:   []string{"modulos/orquesta-server/avance_util.go"},
					EvidenceRefs:    []string{"evidence-ref-codex-app-server-goal-high-token-usage"},
				},
			}},
		}},
	}
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStore,
		GoalStopper:    stopper,
		StateStore:     &memoryStateStoreV0{},
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runGoalObservationTickV0(context.Background())

	if stopper.calls != 0 {
		t.Fatalf("goal con artefacto util no debe parar: calls=%d request=%+v", stopper.calls, stopper.last)
	}
	loaded, err := goalStore.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if loaded.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("goal state=%+v", loaded)
	}
}

func TestRuntimeV0IdleFallbackReconciliaResultMaterializadoPorPuertoV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 15, 30, 0, 0, time.UTC)
	const runRef = "run-ref-idle-materialized-port-001"
	const goalRef = "goal-ref-idle-materialized-port-001"
	const externalGoalRef = "external-goal-ref-idle-materialized-port-001"
	store := &memoryStateStoreV0{}
	goalStore := newMemoryGoalStateStoreV0()
	requiredEvidence := "evidence-ref-idle-materialized-required-001"
	spec := orquestagoal.GoalWorkSpecV0{
		RunRef:       runRef,
		GoalRef:      goalRef,
		Objective:    "cerrar automejora idle desde result materializado",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequiredEvidenceRefs: []string{requiredEvidence},
		},
	}
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		SchemaVersion:   orquestagoal.GoalWorkStateSchemaV0,
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec:            spec,
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	supervisor := &fakeSupervisorV0{
		materializedGoalResult: IdleSelfImprovementMaterializedGoalResultV0{
			Found: true,
			Result: orquestagoal.GoalWorkResultV0{
				SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
				Status:          orquestagoal.GoalStatusCompleteV0,
				GoalRef:         goalRef,
				ExternalGoalRef: externalGoalRef,
				Summary:         "cerrado por result materializado",
				EvidenceRefs:    []string{requiredEvidence},
			},
			EvidenceRefs: []string{"evidence-ref-idle-materialized-port"},
		},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           false,
		IdleSelfImprovementGoalFirst:  true,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStore,
		StateStore:     store,
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted:        true,
		Status:          orquestagoal.GoalStatusAcceptedV0,
		Message:         "goal_first_launched",
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		GoalSpec:        spec,
		GoalReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
		},
	}, now.Add(-time.Minute))

	if !runtime.observePendingIdleSelfImprovementGoalV0(context.Background(), now) {
		t.Fatalf("fallback idle no observo")
	}
	if supervisor.materializedGoalCalls != 1 {
		t.Fatalf("materialized calls=%d", supervisor.materializedGoalCalls)
	}
	if store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalClosureAcceptedReasonV0 ||
		store.snapshotV0().IdleSelfImprovementGoalClosure == nil ||
		!store.snapshotV0().IdleSelfImprovementGoalClosure.Accepted ||
		store.snapshotV0().IdleSelfImprovementGoalResult == nil ||
		!containsStringForTestV0(store.snapshotV0().IdleSelfImprovementGoalResult.EvidenceRefs, "evidence-ref-idle-materialized-port") {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0GoalObservationLoopCierraGoalActivoSinDirectorResidenteV0(t *testing.T) {
	requireLocalTCPForServerTestV0(t)
	ctx := context.Background()
	goalStore := newMemoryGoalStateStoreV0()
	runRef := "run-ref-goal-observer-e2e-001"
	goalRef := "goal-ref-goal-observer-e2e-001"
	start, err := orquestagoal.StartGoalWorkV0(
		ctx,
		orquestagoal.GoalWorkStartRequestV0{
			RunRef: runRef,
			Spec: orquestagoal.GoalWorkSpecV0{
				GoalRef:      goalRef,
				Objective:    "Cerrar un goal activo desde el observador residente.",
				DirectorKind: orquestagoal.GoalDirectorKindRuntimeGoalV0,
				WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
				RequiredTests: []orquestagoal.GoalRequiredTestV0{{
					TestRef: "test-ref-goal-observer-e2e-001",
					Command: "go test ./modulos/orquesta-server",
				}},
			},
			EvidenceRefs: []string{"evidence-ref-goal-observer-started-001"},
		},
		orquestagoal.GoalWorkLifecyclePortsV0{
			Launcher:   goalObservationLauncherForTestV0{},
			StateStore: goalStore,
		},
	)
	if err != nil {
		t.Fatalf("StartGoalWorkV0: %v", err)
	}
	supervisor := newGoalObservationLifecycleSupervisorForTestV0(
		goalStore,
		goalObservationResultObserverForTestV0{
			result: orquestagoal.GoalWorkResultV0{
				Status:  orquestagoal.GoalStatusCompleteV0,
				GoalRef: goalRef,
				RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
					TestRef:      "test-ref-goal-observer-e2e-001",
					Status:       "passed",
					EvidenceRefs: []string{"evidence-ref-goal-observer-test-001"},
				}},
				EvidenceRefs: []string{"evidence-ref-goal-observer-complete-001"},
			},
		},
		goalObservationClosureValidatorForTestV0{},
	)
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:                          "127.0.0.1:0",
		StateDir:                      t.TempDir(),
		TickInterval:                  20 * time.Millisecond,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
		ShutdownGracePeriod:           time.Second,
	}, RuntimeDepsV0{
		Supervisor:       supervisor,
		ResidentDirector: panicResidentDirectorForGoalObservationTestV0{},
		GoalStateStore:   goalStore,
		StateStore:       &threadSafeStateStoreV0{},
		Clock:            fixedClockV0{now: time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(runCtx) }()

	waitForRuntimeTestV0(t, supervisor.observed)
	cancel()
	if err := waitRuntimeDoneV0(t, done); err != nil {
		t.Fatalf("RunV0: %v", err)
	}
	state, err := goalStore.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusCompleteV0 ||
		state.LastClosure == nil ||
		!state.LastClosure.Accepted ||
		state.LastResult == nil ||
		state.LastResult.Status != orquestagoal.GoalStatusCompleteV0 ||
		supervisor.goalObservationCalls() == 0 ||
		start.State.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("state=%+v calls=%d start=%+v", state, supervisor.goalObservationCalls(), start.State)
	}
	serverState := runtime.StateV0()
	if serverState.ResidentDirectorTicks != 0 ||
		serverState.GoalObserverTicks == 0 ||
		serverState.GoalObserverTerminal == 0 {
		t.Fatalf("server state=%+v", serverState)
	}
}

func TestRuntimeV0GoalObservationLoopUsaIntervaloPropioV0(t *testing.T) {
	requireLocalTCPForServerTestV0(t)
	supervisor := &fakeSupervisorV0{
		goalStarted: make(chan struct{}, 3),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:                          "127.0.0.1:0",
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverInterval:          10 * time.Millisecond,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
		ShutdownGracePeriod:           time.Second,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     &threadSafeStateStoreV0{},
		Clock:          fixedClockV0{now: time.Date(2026, 6, 27, 11, 30, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(runCtx) }()

	waitForRuntimeTestV0(t, supervisor.goalStarted)
	waitForRuntimeTestV0(t, supervisor.goalStarted)
	cancel()
	if err := waitRuntimeDoneV0(t, done); err != nil {
		t.Fatalf("RunV0: %v", err)
	}
	if supervisor.goalObservationCalls < 2 {
		t.Fatalf("goal observer calls=%d, no uso intervalo propio", supervisor.goalObservationCalls)
	}
}

func TestRuntimeV0GoalObservationFingerprintSaltaRunSinCambiosV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	goalStore := newMemoryGoalStateStoreV0()
	state := mustServerGoalStateForTestV0(
		t,
		"run-ref-goal-observer-fingerprint-001",
		"goal-ref-goal-observer-fingerprint-001",
		orquestagoal.GoalStatusRunningV0,
	)
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	supervisor := &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{
			Observations: []orquestagoal.GoalWorkObserveResultV0{{
				State: state,
				Result: orquestagoal.GoalWorkResultV0{
					Status:       orquestagoal.GoalStatusRunningV0,
					GoalRef:      state.GoalRef,
					EvidenceRefs: []string{"evidence-ref-goal-observer-fingerprint-001"},
				},
				EvidenceRefs: []string{"evidence-ref-goal-observer-fingerprint-001"},
			}},
			EvidenceRefs: []string{"evidence-ref-goal-observer-fingerprint-001"},
		}},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:      supervisor,
		GoalStateStore:  goalStore,
		GoalFingerprint: fakeGoalObservationFingerprintForTestV0{},
		StateStore:      store,
		Clock:           fixedClockV0{now: time.Date(2026, 6, 27, 10, 2, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runGoalObservationTickV0(context.Background())
	runtime.runGoalObservationTickV0(context.Background())

	if supervisor.goalObservationCalls != 1 {
		t.Fatalf("fingerprint estable no debe re-observar: calls=%d", supervisor.goalObservationCalls)
	}
	if store.snapshotV0().GoalObserverStatus != "skipped" ||
		store.snapshotV0().GoalObserverTicks != 1 ||
		store.snapshotV0().GoalObserverObserved != 1 {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
	if store.snapshotV0().GoalObserverOperationalMessage == nil ||
		store.snapshotV0().GoalObserverOperationalMessage.Counters["observed"] != 1 ||
		store.snapshotV0().GoalObserverOperationalMessage.Counters["terminal"] != 0 ||
		!containsStringForTestV0(store.snapshotV0().GoalObserverOperationalMessage.RunRefs, state.RunRef) ||
		!containsStringForTestV0(store.snapshotV0().GoalObserverOperationalMessage.GoalRefs, state.GoalRef) {
		t.Fatalf("active goal snapshot lost after fingerprint skip: %+v", store.snapshotV0().GoalObserverOperationalMessage)
	}
}

func TestRuntimeV0ForgetGoalObservationFingerprintV0(t *testing.T) {
	runRef := "run-ref-goal-observer-forget-001"
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor:     &fakeSupervisorV0{},
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     &memoryStateStoreV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	fingerprint := orquestagoal.GoalObservationFingerprintV0{
		RunRef:       runRef,
		GoalRef:      "goal-ref-goal-observer-forget-001",
		AckFilesHash: "ack-hash-stable",
		ProcessAlive: true,
		LastStatus:   orquestagoal.GoalStatusRunningV0,
		EvidenceHash: "evidence-hash-stable",
	}
	runtime.storeGoalObservationFingerprintV0(runRef, fingerprint)

	if !runtime.ForgetGoalObservationFingerprintV0(runRef) {
		t.Fatalf("fingerprint no olvidada")
	}
	if runtime.goalObservationFingerprintUnchangedV0(runRef, fingerprint) {
		t.Fatalf("fingerprint sigue presente")
	}
	if runtime.ForgetGoalObservationFingerprintV0(runRef) {
		t.Fatalf("segunda invalidacion no debe reportar cambio")
	}
}

func TestRuntimeV0GoalObservationDisabledNoArrancaV0(t *testing.T) {
	supervisor := &fakeSupervisorV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           false,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     &memoryStateStoreV0{},
		Clock:          fixedClockV0{now: time.Date(2026, 6, 27, 10, 5, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	if runtime.runGoalObservationTickAsyncV0(context.Background()) {
		t.Fatalf("goal observer tick arrancado pese a opt-out")
	}
	if supervisor.goalObservationCalls != 0 ||
		runtime.RequestGoalObservationWakeupV0("test_disabled") {
		t.Fatalf("goal observer activo calls=%d", supervisor.goalObservationCalls)
	}
}

func mustServerGoalStateForTestV0(
	t *testing.T,
	runRef string,
	goalRef string,
	status string,
) orquestagoal.GoalWorkStateV0 {
	t.Helper()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      goalRef,
			Objective:    "Implementar una tarea acotada con evidencias.",
			DirectorKind: orquestagoal.GoalDirectorKindRuntimeGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			ExternalGoalRef: "external-" + goalRef,
		},
		EvidenceRefs: []string{"evidence-ref-" + goalRef},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.Status = status
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		t.Fatalf("NewGoalWorkStateV0: %v", err)
	}
	return state
}

type fakeGoalObservationFingerprintForTestV0 struct{}

func (fakeGoalObservationFingerprintForTestV0) FingerprintGoalObservationV0(
	_ context.Context,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalObservationFingerprintV0, bool, error) {
	return orquestagoal.GoalObservationFingerprintV0{
		RunRef:       state.RunRef,
		GoalRef:      state.GoalRef,
		AckFilesHash: "ack-hash-stable",
		ProcessAlive: true,
		LastStatus:   state.Status,
		EvidenceHash: "evidence-hash-stable",
	}, true, nil
}

func TestRuntimeV0GoalObservationErrorVisibleYRecuperaV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		goalObservationErrs: []error{errors.New("goal observer temporalmente no disponible"), nil},
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{{}, {
			Issues: []orquestagoal.GoalWorkObserveActiveIssueV0{{
				RunRef:  "run-ref-goal-observer-issue-001",
				GoalRef: "goal-ref-goal-observer-issue-001",
				Code:    "observe_goal_failed",
			}},
		}},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     store,
		Clock:          fixedClockV0{now: time.Date(2026, 6, 27, 10, 10, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runGoalObservationTickV0(context.Background())
	if store.snapshotV0().GoalObserverStatus != "error" ||
		store.snapshotV0().GoalObserverErrorTicks != 1 ||
		store.snapshotV0().LastError == "" ||
		len(store.snapshotV0().RecentErrors) != 1 ||
		store.snapshotV0().RecentErrors[0].Scope != "goal_observer" {
		t.Fatalf("error state=%+v", store.snapshotV0())
	}

	runtime.runGoalObservationTickV0(context.Background())
	if store.snapshotV0().GoalObserverStatus != "ok_with_issues" ||
		store.snapshotV0().GoalObserverErrorTicks != 1 ||
		store.snapshotV0().LastError != "" ||
		store.snapshotV0().GoalObserverIssues != 1 {
		t.Fatalf("recovered state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0GoalObservationAsyncCoalesceaUnTickPendienteV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		goalStarted: make(chan struct{}, 2),
		goalRelease: make(chan struct{}),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     store,
		Clock:          fixedClockV0{now: time.Date(2026, 6, 27, 10, 15, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	if !runtime.runGoalObservationTickAsyncV0(context.Background()) {
		t.Fatalf("first goal observer async tick not started")
	}
	<-supervisor.goalStarted
	if runtime.runGoalObservationTickAsyncV0(context.Background()) {
		t.Fatalf("second goal observer tick overlapped while first in flight")
	}
	supervisor.goalRelease <- struct{}{}
	select {
	case <-supervisor.goalStarted:
	case <-time.After(time.Second):
		t.Fatalf("pending goal observer tick was not coalesced")
	}
	if runtime.runGoalObservationTickAsyncV0(context.Background()) {
		t.Fatalf("third goal observer tick overlapped while coalesced tick in flight")
	}
	supervisor.goalRelease <- struct{}{}
	waitRuntimeAsyncWorkForTestV0(t, runtime)
	if store.snapshotV0().GoalObserverTickActive {
		t.Fatalf("goal observer tick stayed active: %+v", store.snapshotV0())
	}
	if supervisor.goalObservationCalls != 2 || store.snapshotV0().GoalObserverTicks != 2 {
		t.Fatalf("calls=%d state=%+v", supervisor.goalObservationCalls, store.snapshotV0())
	}
}

func TestRuntimeV0GoalObservationTickTimeoutPublicaErrorAccionableV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &timeoutGoalObserverSupervisorForTestV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverTimeout:           20 * time.Millisecond,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     store,
		Clock:          fixedClockV0{now: time.Date(2026, 7, 4, 11, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.runGoalObservationTickV0(context.Background())

	if calls := atomic.LoadInt32(&supervisor.calls); calls != 1 {
		t.Fatalf("goal observer calls=%d", calls)
	}
	if store.snapshotV0().GoalObserverStatus != "error" ||
		store.snapshotV0().GoalObserverLastError != "goal_observer_timeout" ||
		store.snapshotV0().GoalObserverErrorTicks != 1 ||
		store.snapshotV0().GoalObserverTickActive {
		t.Fatalf("state=%+v", store.snapshotV0())
	}
}

func TestRuntimeV0GoalObservationTickTimeoutNoBloqueaSiBackendIgnoraContextoV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	supervisor := &ignoringContextGoalObserverSupervisorForTestV0{
		started:  make(chan struct{}),
		release:  make(chan struct{}),
		returned: make(chan struct{}),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverTimeout:           20 * time.Millisecond,
		AuditDisabled:                 true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     store,
		Clock:          fixedClockV0{now: time.Date(2026, 7, 4, 11, 30, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	firstDone := make(chan struct{})
	go func() {
		runtime.runGoalObservationTickV0(context.Background())
		close(firstDone)
	}()
	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatalf("goal observer backend call did not start")
	}
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatalf("goal observer tick blocked behind backend that ignored context")
	}
	if store.snapshotV0().GoalObserverStatus != "error" ||
		store.snapshotV0().GoalObserverLastError != "goal_observer_timeout" ||
		store.snapshotV0().GoalObserverErrorTicks != 1 {
		t.Fatalf("first timeout state=%+v", store.snapshotV0())
	}

	runtime.runGoalObservationTickV0(context.Background())
	if store.snapshotV0().GoalObserverStatus != "error" ||
		store.snapshotV0().GoalObserverLastError != "goal_observer_backend_call_in_flight" ||
		store.snapshotV0().GoalObserverErrorTicks != 2 {
		t.Fatalf("second timeout state=%+v", store.snapshotV0())
	}
	close(supervisor.release)
	select {
	case <-supervisor.returned:
	case <-time.After(time.Second):
		t.Fatalf("goal observer backend call did not return after release")
	}
	waitGoalObservationBackendInactiveForTestV0(t, runtime)
}

type timeoutGoalObserverSupervisorForTestV0 struct {
	calls int32
}

func (supervisor *timeoutGoalObserverSupervisorForTestV0) RunGlobalSupervisorV0(
	_ context.Context,
	_ orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	return orquestarunsupervisor.RunSupervisorResultV0{}, nil
}

func (supervisor *timeoutGoalObserverSupervisorForTestV0) ObserveActiveGoalWorksV0(
	ctx context.Context,
	_ orquestagoal.GoalWorkObserveActiveRequestV0,
) (orquestagoal.GoalWorkObserveActiveResultV0, error) {
	atomic.AddInt32(&supervisor.calls, 1)
	<-ctx.Done()
	return orquestagoal.GoalWorkObserveActiveResultV0{}, ctx.Err()
}

type ignoringContextGoalObserverSupervisorForTestV0 struct {
	started     chan struct{}
	release     chan struct{}
	returned    chan struct{}
	startedOnce sync.Once
}

func (supervisor *ignoringContextGoalObserverSupervisorForTestV0) RunGlobalSupervisorV0(
	_ context.Context,
	_ orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	return orquestarunsupervisor.RunSupervisorResultV0{}, nil
}

func (supervisor *ignoringContextGoalObserverSupervisorForTestV0) ObserveActiveGoalWorksV0(
	_ context.Context,
	_ orquestagoal.GoalWorkObserveActiveRequestV0,
) (orquestagoal.GoalWorkObserveActiveResultV0, error) {
	supervisor.startedOnce.Do(func() { close(supervisor.started) })
	<-supervisor.release
	close(supervisor.returned)
	return orquestagoal.GoalWorkObserveActiveResultV0{}, nil
}

func waitGoalObservationBackendInactiveForTestV0(t *testing.T, runtime *RuntimeV0) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for atomic.LoadInt32(&runtime.goalObservationBackendActive) != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("goal observer backend call stayed active")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type goalObservationLifecycleSupervisorForTestV0 struct {
	mu               sync.Mutex
	observed         chan struct{}
	observedOnce     sync.Once
	stateStore       orquestagoal.GoalWorkStateStorePortV0
	observer         orquestagoal.GoalWorkObservationPortV0
	closureValidator orquestagoal.GoalWorkClosureValidatorPortV0
	goalCalls        int
	supervisorCalls  int
}

func newGoalObservationLifecycleSupervisorForTestV0(
	stateStore orquestagoal.GoalWorkStateStorePortV0,
	observer orquestagoal.GoalWorkObservationPortV0,
	closureValidator orquestagoal.GoalWorkClosureValidatorPortV0,
) *goalObservationLifecycleSupervisorForTestV0 {
	return &goalObservationLifecycleSupervisorForTestV0{
		observed:         make(chan struct{}),
		stateStore:       stateStore,
		observer:         observer,
		closureValidator: closureValidator,
	}
}

func (supervisor *goalObservationLifecycleSupervisorForTestV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	supervisor.mu.Lock()
	supervisor.supervisorCalls++
	supervisor.mu.Unlock()
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func (supervisor *goalObservationLifecycleSupervisorForTestV0) ObserveActiveGoalWorksV0(
	ctx context.Context,
	request orquestagoal.GoalWorkObserveActiveRequestV0,
) (orquestagoal.GoalWorkObserveActiveResultV0, error) {
	supervisor.mu.Lock()
	supervisor.goalCalls++
	supervisor.mu.Unlock()
	result, err := orquestagoal.ObserveActiveGoalWorksV0(
		ctx,
		request,
		orquestagoal.GoalWorkLifecyclePortsV0{
			Observer:         supervisor.observer,
			ClosureValidator: supervisor.closureValidator,
			StateStore:       supervisor.stateStore,
		},
	)
	if err == nil && goalObservationResultHasTerminalForTestV0(result) {
		supervisor.observedOnce.Do(func() { close(supervisor.observed) })
	}
	return result, err
}

func (supervisor *goalObservationLifecycleSupervisorForTestV0) goalObservationCalls() int {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	return supervisor.goalCalls
}

type goalObservationLauncherForTestV0 struct{}

func (goalObservationLauncherForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	return orquestagoal.GoalLaunchReceiptV0{
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "external-" + spec.GoalRef,
		EvidenceRefs:    []string{"evidence-ref-goal-observer-launch-001"},
	}, nil
}

type goalObservationResultObserverForTestV0 struct {
	result orquestagoal.GoalWorkResultV0
}

func (observer goalObservationResultObserverForTestV0) ObserveGoalWorkV0(
	context.Context,
	orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	return observer.result, nil
}

type goalObservationSequenceObserverForTestV0 struct {
	results []orquestagoal.GoalWorkResultV0
	calls   int
}

func (observer *goalObservationSequenceObserverForTestV0) ObserveGoalWorkV0(
	context.Context,
	orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	index := observer.calls
	observer.calls++
	if index >= len(observer.results) {
		return orquestagoal.GoalWorkResultV0{}, nil
	}
	return observer.results[index], nil
}

type goalObservationClosureValidatorForTestV0 struct{}

func (goalObservationClosureValidatorForTestV0) ValidateGoalWorkClosureV0(
	context.Context,
	orquestagoal.GoalWorkSpecV0,
	orquestagoal.GoalWorkResultV0,
) (orquestagoal.GoalClosureValidationV0, error) {
	return orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusAcceptedV0,
		Accepted:     true,
		EvidenceRefs: []string{"evidence-ref-goal-observer-closure-001"},
	}, nil
}

type panicResidentDirectorForGoalObservationTestV0 struct{}

func (panicResidentDirectorForGoalObservationTestV0) RunResidentDirectorV0(
	context.Context,
	ResidentDirectorCommandV0,
) (ResidentDirectorResultV0, error) {
	panic("resident director must stay disabled in goal observation e2e")
}

func goalObservationResultHasTerminalForTestV0(
	result orquestagoal.GoalWorkObserveActiveResultV0,
) bool {
	for _, observation := range result.Observations {
		if observation.Terminal && observation.Accepted {
			return true
		}
	}
	return false
}

func mustIdleSelfImprovementGoalStateForProgressTestV0(
	t *testing.T,
	runRef string,
	goalRef string,
	externalGoalRef string,
) orquestagoal.GoalWorkStateV0 {
	t.Helper()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Gobernar progreso util de una automejora goal-first.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{
				{Path: "modulos/orquesta-server"},
			},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
		},
		EvidenceRefs: []string{"evidence-ref-" + goalRef},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	return state
}

func idleSelfImprovementPreparedForProgressTestV0(
	state orquestagoal.GoalWorkStateV0,
) IdleSelfImprovementResultV0 {
	return IdleSelfImprovementResultV0{
		Accepted:        true,
		RunRef:          state.RunRef,
		Status:          orquestagoal.GoalStatusAcceptedV0,
		Message:         "goal_first_launched",
		GoalRef:         state.GoalRef,
		ExternalGoalRef: state.ExternalGoalRef,
		GoalSpec:        state.Spec,
		GoalReceipt:     state.LaunchReceipt,
		EvidenceRefs:    []string{"evidence-ref-idle-progress-governor-launched"},
	}
}

type fakeGoalCooperativeStopperForTestV0 struct {
	calls int
	last  GoalCooperativeStopRequestV0
	err   error
}

func (fake *fakeGoalCooperativeStopperForTestV0) RequestGoalCooperativeStopV0(
	_ context.Context,
	request GoalCooperativeStopRequestV0,
) (GoalCooperativeStopResultV0, error) {
	fake.calls++
	fake.last = request
	if fake.err != nil {
		return GoalCooperativeStopResultV0{}, fake.err
	}
	return GoalCooperativeStopResultV0{
		Requested:    true,
		Status:       "stop_requested",
		EvidenceRefs: []string{"evidence-ref-fake-goal-cooperative-stop"},
	}, nil
}
