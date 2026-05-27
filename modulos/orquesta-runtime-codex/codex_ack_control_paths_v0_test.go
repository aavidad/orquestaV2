package orquestaruntimecodex

import (
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexAgentAckReceiptV0RechazaRuntimeLocalConWriteSetRaizV0(t *testing.T) {
	spec := codexSpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{"."}
	cases := []string{
		".orquesta-local-runtime-20260525/run/agent_ack.json",
		".orquesta-runtime-20260525/run/debug.log",
		"orquesta-local-runtime-20260525/run/agent_packet.json",
		".orquesta-server/state.json",
		".orquesta-smoke-work/run.log",
		".orquesta-logs/daemon.log",
		"logs/server.log",
		".ssl-key.log",
		"orquesta.env",
		"orquesta.db",
		".orquesta-inbox.md",
	}
	for _, file := range cases {
		t.Run(file, func(t *testing.T) {
			ack := `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["` + file + `"],"tests":["go test ./..."],"notes":["alcance raiz no exporta control"]}`

			_, issues := ValidateCodexAgentAckBytesForSpecV0([]byte(ack), spec)

			requireCodexIssueV0(t, issues, CodexConnectorAckArtifactV0)
			if !codexAckIssuesEvidencePrefixForTestV0(issues, "local_artifact_excluded:") {
				t.Fatalf("issues sin recibo compacto: %+v", issues)
			}
		})
	}
}

func codexAckIssuesEvidencePrefixForTestV0(issues []orquestaruntime.ExternalAgentConnectorErrorV0, prefix string) bool {
	for _, issue := range issues {
		for _, evidence := range issue.Evidence {
			if strings.HasPrefix(evidence, prefix) {
				return true
			}
		}
	}
	return false
}
