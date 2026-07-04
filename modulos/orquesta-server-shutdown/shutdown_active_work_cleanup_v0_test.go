package orquestaservershutdown

import (
	"context"
	"testing"
)

func TestShutdownServerV0CleanupGoalBackendsNoLimpiaSiHayGoalFirstActivoV0(t *testing.T) {
	deps := newServerShutdownDepsForTestV0(nil)
	deps.active.works = []ActiveShutdownWorkV0{
		{Kind: "goal_first", RunRef: "run-goal-first-active", WorkRef: "goal-ref-active", Status: "running"},
		{
			Kind:            "goal_backend",
			WorkRef:         "orquesta-goal-backend-with-active-goal",
			ExternalWorkRef: "codex-goal-app-server-tmux",
			Status:          ServerShutdownStatusBackendStillRunningV0,
		},
	}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy:         "orquesta-director",
		Reason:              "no limpiar backend si goal_first sigue activo",
		CleanupGoalBackends: true,
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != ServerShutdownStatusBackendStillRunningV0 ||
		result.ActiveWorkCount != 2 ||
		deps.active.calls != 2 ||
		deps.cleaner.calls != 0 ||
		!shutdownGoalActionForTestV0(result.GoalActions, ServerShutdownGoalActionCleanupRequiredV0) {
		t.Fatalf("result=%+v active_calls=%d cleaner_calls=%d", result, deps.active.calls, deps.cleaner.calls)
	}
}
