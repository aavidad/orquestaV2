package main

import (
	"context"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestServerShutdownSnapshotFromStackV0FusionaAppEIdleV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{}
	appHook := fakeServerGoalActiveShutdownHookV0{
		result: orquestaservershutdown.ActiveShutdownWorkResultV0{
			ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{{
				Kind:            "goal_backend",
				RunRef:          "run-ref-app-shutdown-snapshot",
				WorkRef:         "goal-ref-app-shutdown-snapshot",
				ExternalWorkRef: "thread-ref-app-shutdown-snapshot",
				Status:          orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0,
			}},
			EvidenceRefs: []string{"evidence-ref-app-shutdown-snapshot"},
		},
	}
	idleHook := fakeServerGoalActiveShutdownHookV0{
		result: orquestaservershutdown.ActiveShutdownWorkResultV0{
			ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{{
				Kind:    "goal_backend",
				RunRef:  "run-ref-idle-shutdown-snapshot",
				WorkRef: "goal-ref-idle-shutdown-snapshot",
				Status:  orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0,
			}},
			EvidenceRefs: []string{"evidence-ref-idle-shutdown-snapshot"},
		},
	}
	snapshot := serverShutdownSnapshotFromStackV0(orquestaappcodexstack.StackV0{}, serverCodexGoalBackendsV0{
		AppGoal: serverCodexGoalBackendV0{
			Starter:      serverCodexAppServerGoalBackendV0{Protocol: protocol},
			Observer:     serverCodexAppServerGoalBackendV0{Protocol: protocol},
			ShutdownHook: appHook,
		},
		IdleGoal: serverCodexGoalBackendV0{
			ShutdownHook: idleHook,
		},
	})

	result, err := snapshot.SnapshotShutdownV0(context.Background(), orquestaserver.ShutdownSnapshotRequestV0{
		EvidenceRefs: []string{"evidence-ref-request-shutdown-snapshot"},
	})
	if err != nil {
		t.Fatalf("SnapshotShutdownV0: %v", err)
	}
	if result.Status != orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 ||
		result.ActiveWorkCount != 2 ||
		!hasShutdownSnapshotWorkV0(result.ActiveWorks, "run-ref-app-shutdown-snapshot", "goal-ref-app-shutdown-snapshot") ||
		!hasShutdownSnapshotWorkV0(result.ActiveWorks, "run-ref-idle-shutdown-snapshot", "goal-ref-idle-shutdown-snapshot") {
		t.Fatalf("snapshot result=%+v", result)
	}
}

func hasShutdownSnapshotWorkV0(
	works []orquestaserver.ShutdownSnapshotWorkV0,
	runRef string,
	workRef string,
) bool {
	for _, work := range works {
		if work.RunRef == runRef && work.WorkRef == workRef {
			return true
		}
	}
	return false
}
