package orquestaruntimecodexdelivery

import (
	"context"
	"path/filepath"
	"testing"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestAgentScopedCodexReceiptAckPathResolverV0ResuelveACKPorRunYAgente(t *testing.T) {
	baseDir := t.TempDir()
	resolver := AgentScopedCodexReceiptAckPathResolverV0{BaseDir: baseDir}

	resolution, err := resolver.ResolveCodexReceiptAckPathV0(
		context.Background(),
		CodexReceiptAckPathRequestV0{
			RunID:    "run-team-001",
			AgentRef: "agent-agenda-web",
		},
	)

	if err != nil {
		t.Fatalf("ResolveCodexReceiptAckPathV0: %v", err)
	}
	want := filepath.Join(baseDir, "run-team-001", "agent-agenda-web", orquestaruntimecodex.CodexAgentAckFileNameV0)
	if resolution.AckPath != want {
		t.Fatalf("ack_path=%q want %q", resolution.AckPath, want)
	}
}

func TestCodexReceiptAgentRuntimeDirV0RechazaBaseRelativa(t *testing.T) {
	if _, err := CodexReceiptAgentRuntimeDirV0("tmp/runtime", "run-001", "agent-001"); err == nil {
		t.Fatalf("esperaba error")
	}
}

func TestAgentScopedCodexReceiptAckPathResolverV0RechazaFileNameConPath(t *testing.T) {
	resolver := AgentScopedCodexReceiptAckPathResolverV0{
		BaseDir:  t.TempDir(),
		FileName: "../agent_ack.json",
	}

	if _, err := resolver.ResolveCodexReceiptAckPathV0(
		context.Background(),
		CodexReceiptAckPathRequestV0{AgentRef: "agent-001"},
	); err == nil {
		t.Fatalf("esperaba error")
	}
}
