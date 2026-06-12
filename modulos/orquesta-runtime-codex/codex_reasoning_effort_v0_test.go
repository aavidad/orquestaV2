package orquestaruntimecodex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexExecResolverV0ElevaReasoningCuandoPacketPideHigh(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.ReasoningEffort = "medium"
	spec := codexSpecForTestV0()
	spec.AgentPacket.CapacityLevel = "high"

	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), spec)
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}

	requireWrapperReasoningForTestV0(t, profile.RuntimeWorkDir, "high")
}

func TestCodexExecResolverV0ConservaXHighCuandoPacketLoPide(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.ReasoningEffort = "high"
	spec := codexSpecForTestV0()
	spec.AgentPacket.CapacityLevel = "xhigh"

	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), spec)
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}

	requireWrapperReasoningForTestV0(t, profile.RuntimeWorkDir, "xhigh")
}

func requireWrapperReasoningForTestV0(t *testing.T, runtimeWorkDir string, effort string) {
	t.Helper()
	wrapper, err := os.ReadFile(filepath.Join(runtimeWorkDir, CodexWrapperFileNameV0))
	if err != nil {
		t.Fatalf("read wrapper: %v", err)
	}
	want := "-c " + shellQuoteV0(`model_reasoning_effort="`+effort+`"`)
	if !strings.Contains(string(wrapper), want) {
		t.Fatalf("wrapper no fija reasoning: falta %q\n%s", want, wrapper)
	}
}
