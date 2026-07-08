package orquestaservershutdown

import (
	"context"
	"testing"
)

func TestShutdownServerV0CleanupGoalBackendsParcialNoConservaAccionBloqueanteDeWorkLimpioV0(t *testing.T) {
	deps := newServerShutdownDepsForTestV0(nil)
	cleanedWork := ActiveShutdownWorkV0{
		Kind:            "goal_backend",
		WorkRef:         "orquesta-goal-cleanup-partial-cleaned",
		ExternalWorkRef: "codex-goal-app-server-tmux-cleaned",
		Status:          ServerShutdownStatusBackendStillRunningV0,
		EvidenceRefs:    []string{"evidence-ref-backend-partial-cleaned-before"},
	}
	stillLiveWork := ActiveShutdownWorkV0{
		Kind:            "goal_backend",
		WorkRef:         "orquesta-goal-cleanup-partial-still-live",
		ExternalWorkRef: "codex-goal-app-server-tmux-still-live",
		Status:          ServerShutdownStatusBackendStillRunningV0,
		EvidenceRefs:    []string{"evidence-ref-backend-partial-still-live"},
	}
	deps.active.reads = [][]ActiveShutdownWorkV0{
		{cleanedWork, stillLiveWork},
		{stillLiveWork},
	}
	deps.cleaner.result = ActiveShutdownWorkCleanupResultV0{
		CleanedWorkCount: 1,
		EvidenceRefs:     []string{"evidence-ref-backend-partial-cleaned"},
	}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy:         "orquesta-director",
		Reason:              "cleanup parcial de backends goal propios",
		CorrelationID:       "corr-cleanup-backend-partial",
		CleanupGoalBackends: true,
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != ServerShutdownStatusBackendStillRunningV0 ||
		result.ActiveWorkCount != 1 ||
		result.ActiveWorks[0].WorkRef != stillLiveWork.WorkRef ||
		deps.active.calls != 2 ||
		deps.cleaner.calls != 1 ||
		!shutdownGoalActionForWorkForTestV0(result.GoalActions, cleanedWork.WorkRef, ServerShutdownGoalActionCleanupCompletedV0) ||
		shutdownGoalActionForWorkForTestV0(result.GoalActions, cleanedWork.WorkRef, ServerShutdownGoalActionCleanupRequestedV0) ||
		!shutdownGoalActionForWorkForTestV0(result.GoalActions, stillLiveWork.WorkRef, ServerShutdownGoalActionCleanupAttemptedV0) ||
		shutdownGoalActionForWorkForTestV0(result.GoalActions, stillLiveWork.WorkRef, ServerShutdownGoalActionCleanupCompletedV0) {
		t.Fatalf("result=%+v active_calls=%d cleaner=%+v", result, deps.active.calls, deps.cleaner)
	}
}
