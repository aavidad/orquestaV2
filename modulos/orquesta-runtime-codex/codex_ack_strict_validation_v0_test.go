package orquestaruntimecodex

import "testing"

func TestStrictCompletedCodexAgentAckV0RechazaACKMinimoHidratableV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.RequiredTests = nil
	ack := `{"schema_version":"codex_agent_ack.v0","status":"completed"}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckInvalidV0)
}

func TestStrictCompletedCodexAgentAckV0RechazaCorrelacionNormalizableV0(t *testing.T) {
	spec := codexSpecForTestV0()
	ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-erronea","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."]}`

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(ack), spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckCorrelationV0)
}

func TestStrictCompletedCodexAgentAckV0AceptaACKCompletoSinIssuesV0(t *testing.T) {
	spec := codexSpecForTestV0()

	_, issues := ValidateStrictCompletedCodexAgentAckBytesForSpecV0([]byte(codexValidAckJSONV0()), spec)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}
