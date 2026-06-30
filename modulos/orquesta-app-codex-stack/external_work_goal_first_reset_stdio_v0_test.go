package orquestaappcodexstack

import "testing"

func TestExternalWorkGoalFirstKnownLaunchFailureReasonV0ClasificaResetStdio(t *testing.T) {
	got := externalWorkGoalFirstKnownLaunchFailureReasonV0(
		"codex_app_server_wrapper_stdio_failed: void node::ResetStdio() at ../src/node.cc:751",
	)

	if got != "codex_app_server_wrapper_stdio_failed" {
		t.Fatalf("reason=%q", got)
	}
}
