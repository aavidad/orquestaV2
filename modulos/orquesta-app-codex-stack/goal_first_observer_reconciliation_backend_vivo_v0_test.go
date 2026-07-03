package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

// Regresion 2026-07-04: el work_ref de un ActiveWork de backend es la sesion
// tmux, no el goal_ref; con backend vivo y refs no coincidentes el goal NO
// puede decaer a goal_backend_gone_without_result.
func TestObserveActiveGoalWorksV0BackendVivoConRefsNoCoincidentesNoDecaeV0(t *testing.T) {
	ctx := context.Background()
	stack, _, _, _ := startGoalFirstQueueSyncStackForTestV0(t)
	observer := &goalFirstQueueShutdownObserverForTestV0{
		activeResult: orquestaservershutdown.ActiveShutdownWorkResultV0{
			ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{{
				Kind:            "goal_backend",
				WorkRef:         "orquesta-goal-aabbccdd00112233",
				ExternalWorkRef: "codex-goal-app-server-tmux",
				Status:          "backend_still_running",
			}},
		},
	}
	stack.Ports.GoalObserver = goalFirstReconciledObserverV0{
		Inner:      observer,
		StateStore: stack.Stores.AppGoalStateStore,
		MaterializedResultSource: stackGoalMaterializedRefsSourceV0{
			Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()}},
		},
	}

	result, err := stack.ObserveActiveGoalWorksV0(ctx, orquestagoal.GoalWorkObserveActiveRequestV0{
		List: orquestagoal.GoalWorkStateListRequestV0{MaxItems: 3},
	})

	if err != nil {
		t.Fatalf("ObserveActiveGoalWorksV0: %v", err)
	}
	for _, observation := range result.Observations {
		if observation.Result.Summary == goalFirstBackendGoneWithoutResultReasonV0 {
			t.Fatalf("goal decaido con backend vivo: %+v", observation)
		}
	}
}
