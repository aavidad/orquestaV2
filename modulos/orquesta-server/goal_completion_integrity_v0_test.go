package orquestaserver

import (
	"context"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestRuntimeV0GoalObservationDetectaCompleteSinResultYAutoinformeEquivocadoV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 21, 22, 0, 0, time.UTC)
	const runRef = "run-ref-idle-integrity-t292-001"
	const goalRef = "goal-ref-autoprogramming-backlog-t292-tests-congelados-001"
	const externalGoalRef = "external-goal-ref-idle-integrity-t292-001"
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
					Status:          orquestagoal.GoalStatusCompleteV0,
					GoalRef:         goalRef,
					ExternalGoalRef: externalGoalRef,
					Summary:         "MEJ-TASK-201 implementada: arnes determinista con backend falso",
				},
			}},
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
		GoalStateStore: goalStore,
		StateStore:     store,
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(idleSelfImprovementPreparedForProgressTestV0(state), now.Add(-3*time.Minute))

	runtime.runGoalObservationTickV0(context.Background())

	result := store.last.IdleSelfImprovementGoalResult
	if result == nil ||
		result.Status != orquestagoal.GoalStatusCompleteV0 ||
		!goalWorkResultHasIssueCodeV0(*result, idleSelfImprovementGoalCompletedWithoutResultReasonV0) ||
		!goalWorkResultHasIssueCodeV0(*result, idleSelfImprovementGoalSelfReportTaskMismatchReasonV0) ||
		!containsStringForTestV0(result.EvidenceRefs, idleSelfImprovementGoalCompletedWithoutResultEvidenceV0) ||
		!containsStringForTestV0(result.EvidenceRefs, idleSelfImprovementGoalSelfReportTaskMismatchEvidenceV0) {
		t.Fatalf("idle result=%+v", result)
	}
	loaded, err := goalStore.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if !containsStringForTestV0(loaded.EvidenceRefs, idleSelfImprovementGoalCompletedWithoutResultEvidenceV0) ||
		loaded.LastResult == nil ||
		!goalWorkResultHasIssueCodeV0(*loaded.LastResult, idleSelfImprovementGoalSelfReportTaskMismatchReasonV0) {
		t.Fatalf("goal state=%+v", loaded)
	}
}

func TestRuntimeV0GoalObservationCompleteConResultYAutoinformeCoherenteSinIssuesV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 21, 25, 0, 0, time.UTC)
	const runRef = "run-ref-idle-integrity-t292-002"
	const goalRef = "goal-ref-autoprogramming-backlog-t292-tests-congelados-002"
	const externalGoalRef = "external-goal-ref-idle-integrity-t292-002"
	state := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, externalGoalRef)
	goalStore := newMemoryGoalStateStoreV0()
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
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
		Supervisor:     &fakeSupervisorV0{},
		GoalStateStore: goalStore,
		StateStore:     &memoryStateStoreV0{},
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(idleSelfImprovementPreparedForProgressTestV0(state), now.Add(-3*time.Minute))

	observed := orquestagoal.GoalWorkObserveActiveResultV0{
		Observations: []orquestagoal.GoalWorkObserveResultV0{{
			State: state,
			Result: orquestagoal.GoalWorkResultV0{
				Status:          orquestagoal.GoalStatusCompleteV0,
				GoalRef:         goalRef,
				ExternalGoalRef: externalGoalRef,
				Summary:         "T292 completada segun backlog",
				ArtifactPaths: []string{
					"modulos/orquesta-autoprogramming/docs/orquesta_goal_result_" + goalRef + ".json",
				},
			},
		}},
	}

	audited := runtime.reconcileIdleSelfImprovementGoalCompletionIntegrityV0(context.Background(), observed)

	if len(audited.Issues) != 0 {
		t.Fatalf("issues inesperados: %+v", audited.Issues)
	}
	result := audited.Observations[0].Result
	if goalWorkResultHasIssueCodeV0(result, idleSelfImprovementGoalCompletedWithoutResultReasonV0) ||
		goalWorkResultHasIssueCodeV0(result, idleSelfImprovementGoalSelfReportTaskMismatchReasonV0) {
		t.Fatalf("result no debia llevar issues de integridad: %+v", result)
	}
}
