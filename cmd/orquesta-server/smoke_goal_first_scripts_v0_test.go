package main

import (
	"strings"
	"testing"
)

func TestSmokeGoalFirstAppServerRealExponeToolOutputPolicyTransportV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"assert_tool_output_policy_transport_observed",
		"tool_output_policy_transport_status",
		"tool_output_policy_transport=$status",
		"evidence-ref-codex-app-server-turn-start-tool-output-policy-sent",
		"evidence-ref-codex-app-server-turn-start-tool-output-policy-accepted",
		"evidence-ref-codex-app-server-turn-start-tool-output-policy-fallback",
		"sent_without_accept_or_fallback",
		"smoke sin evidencia de transporte toolOutputPolicy sent/accepted/fallback",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke no expone transporte toolOutputPolicy: falta %q", want)
		}
	}
}
