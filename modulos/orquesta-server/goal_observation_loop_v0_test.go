package orquestaserver

import (
	"context"
	"errors"
	"sync"
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
	if store.last.GoalObserverStatus != "ok" ||
		store.last.GoalObserverTicks != 1 ||
		store.last.GoalObserverObserved != 1 ||
		store.last.GoalObserverTerminal != 1 ||
		store.last.GoalObserverErrorTicks != 0 {
		t.Fatalf("state=%+v", store.last)
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
	if store.last.GoalObserverStatus != "ok" ||
		store.last.GoalObserverTicks != 1 ||
		store.last.GoalObserverTerminal != 1 {
		t.Fatalf("goal observer state=%+v", store.last)
	}
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalClosureAcceptedReasonV0 ||
		store.last.IdleSelfImprovementGoalResult == nil ||
		store.last.IdleSelfImprovementGoalResult.GoalRef != goalRef ||
		store.last.IdleSelfImprovementGoalClosure == nil ||
		!store.last.IdleSelfImprovementGoalClosure.Accepted ||
		store.last.IdleSelfImprovementFlight ||
		store.last.LastError != "" {
		t.Fatalf("idle state no actualizado por observer generico: %+v", store.last)
	}
}

func TestRuntimeV0GoalObservationBloqueaIdleGoalConsumoCrecienteSinProgresoV0(t *testing.T) {
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
		stopper.last.Reason != idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 ||
		stopper.last.RecommendedAction != idleSelfImprovementGoalReviewReplanRecommendedActionV0 {
		t.Fatalf("stopper calls=%d request=%+v", stopper.calls, stopper.last)
	}
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 ||
		store.last.IdleSelfImprovementGoalResult == nil ||
		store.last.IdleSelfImprovementGoalResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		!containsStringForTestV0(store.last.IdleSelfImprovementGoalResult.ReworkPlanRefs, idleSelfImprovementGoalReviewReplanRefV0) ||
		!containsStringForTestV0(store.last.IdleSelfImprovementGoalResult.EvidenceRefs, "evidence-ref-goal-cooperative-stop-requested") ||
		!goalWorkResultHasIssueCodeV0(*store.last.IdleSelfImprovementGoalResult, idleSelfImprovementGoalHighConsumptionNoProgressReasonV0) {
		t.Fatalf("idle state=%+v", store.last)
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
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalRunningReasonV0 ||
		store.last.IdleSelfImprovementGoalResult == nil ||
		store.last.IdleSelfImprovementGoalResult.Status != orquestagoal.GoalStatusRunningV0 ||
		store.last.IdleSelfImprovementGoalUsefulProgressAt != formatTimeV0(now) ||
		store.last.IdleSelfImprovementGoalUsefulProgressSignature == "sha256:previous-useful-progress" {
		t.Fatalf("idle state=%+v", store.last)
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
	if store.last.IdleSelfImprovementGoalInvalidCheckpointRepeats != 2 ||
		store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 ||
		store.last.IdleSelfImprovementGoalResult == nil ||
		!containsStringForTestV0(store.last.IdleSelfImprovementGoalResult.EvidenceRefs, "evidence-ref-goal-invalid-checkpoint-repeated") {
		t.Fatalf("idle state=%+v", store.last)
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
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalClosureAcceptedReasonV0 ||
		store.last.IdleSelfImprovementGoalClosure == nil ||
		!store.last.IdleSelfImprovementGoalClosure.Accepted ||
		store.last.IdleSelfImprovementGoalResult == nil ||
		!containsStringForTestV0(store.last.IdleSelfImprovementGoalResult.EvidenceRefs, "evidence-ref-idle-materialized-port") {
		t.Fatalf("state=%+v", store.last)
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
	if store.last.GoalObserverStatus != "skipped" ||
		store.last.GoalObserverTicks != 1 ||
		store.last.GoalObserverObserved != 1 {
		t.Fatalf("state=%+v", store.last)
	}
	if store.last.GoalObserverOperationalMessage == nil ||
		store.last.GoalObserverOperationalMessage.Counters["observed"] != 1 ||
		store.last.GoalObserverOperationalMessage.Counters["terminal"] != 0 ||
		!containsStringForTestV0(store.last.GoalObserverOperationalMessage.RunRefs, state.RunRef) ||
		!containsStringForTestV0(store.last.GoalObserverOperationalMessage.GoalRefs, state.GoalRef) {
		t.Fatalf("active goal snapshot lost after fingerprint skip: %+v", store.last.GoalObserverOperationalMessage)
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
	if store.last.GoalObserverStatus != "error" ||
		store.last.GoalObserverErrorTicks != 1 ||
		store.last.LastError == "" ||
		len(store.last.RecentErrors) != 1 ||
		store.last.RecentErrors[0].Scope != "goal_observer" {
		t.Fatalf("error state=%+v", store.last)
	}

	runtime.runGoalObservationTickV0(context.Background())
	if store.last.GoalObserverStatus != "ok_with_issues" ||
		store.last.GoalObserverErrorTicks != 1 ||
		store.last.LastError != "" ||
		store.last.GoalObserverIssues != 1 {
		t.Fatalf("recovered state=%+v", store.last)
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
	if store.last.GoalObserverTickActive {
		t.Fatalf("goal observer tick stayed active: %+v", store.last)
	}
	if supervisor.goalObservationCalls != 2 || store.last.GoalObserverTicks != 2 {
		t.Fatalf("calls=%d state=%+v", supervisor.goalObservationCalls, store.last)
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
