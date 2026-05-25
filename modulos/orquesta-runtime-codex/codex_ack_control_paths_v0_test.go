package orquestaruntimecodex

import "testing"

func TestCodexAgentAckReceiptV0RechazaRuntimeLocalConWriteSetRaizV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{"."}
	cases := []string{
		".orquesta-local-runtime-20260525/run/agent_ack.json",
		".orquesta-runtime-20260525/run/debug.log",
		"orquesta-local-runtime-20260525/run/agent_packet.json",
	}
	for _, file := range cases {
		t.Run(file, func(t *testing.T) {
			ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["` + file + `"],"tests":["go test ./..."],"notes":["alcance raiz no exporta control"]}`

			_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

			requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
		})
	}
}
