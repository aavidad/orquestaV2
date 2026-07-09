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

func TestSmokeGoalFirstToolOutputPolicyAdversarialRealV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	base := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")
	wrapper := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh")

	for _, want := range []string{
		"ORQUESTA_GOAL_FIRST_SMOKE_TOOL_OUTPUT_POLICY_ADVERSARIAL_MODE",
		"assert_tool_output_policy_transport_accepted",
		"tool_output_policy_transport=accepted",
		"evidence-ref-codex-app-server-thread-output-sanitized",
		"codex_app_server_thread_read_response_too_large",
		"checkpoint_started_bug079.txt",
		"assert_bug079_probe_result_executed",
		"probe_result.txt",
		"bug200_probe_not_executed",
		"no_probe_result",
		"stdout gigante",
		"smoke_goal_first_tool_output_policy_adversarial_real=ok",
	} {
		if !strings.Contains(base, want) {
			t.Fatalf("smoke base no cubre modo adversarial toolOutputPolicy: falta %q", want)
		}
	}

	for _, want := range []string{
		"ORQUESTA_GOAL_FIRST_SMOKE_TOOL_OUTPUT_POLICY_ADVERSARIAL_MODE=1",
		"scripts/smoke_goal_first_app_server_real.sh",
		"tool_output_policy_transport=accepted",
		"evidence-ref-codex-app-server-thread-output-sanitized",
		"codex_app_server_thread_read_response_too_large",
	} {
		if !strings.Contains(wrapper, want) {
			t.Fatalf("wrapper adversarial toolOutputPolicy incompleto: falta %q", want)
		}
	}
}
