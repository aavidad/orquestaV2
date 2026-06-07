package orquestaruntimecodex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestCodexExecResolverV0PromptAdvierteContextoRequiredRefOnly(t *testing.T) {
	profile := codexProfileForTestV0(t)
	spec := codexSpecForTestV0()
	spec.AgentPacket.Context.Entries[0].Mode = orquestacontext.ContextMaterializationModeRefOnlyV0
	spec.AgentPacket.Context.Entries[0].Content = ""
	spec.AgentPacket.Context.Entries[0].Bytes = 0
	spec.AgentPacket.Context.Entries[0].RefOnlyReason = orquestacontext.ContextRefOnlyReasonMaterializationMissingV0
	spec.AgentPacket.Context.Entries[0].RequiredRefAction = orquestacontext.ContextRequiredRefActionReadLocalV0
	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), spec)
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}

	prompt, err := os.ReadFile(filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0))
	if err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	for _, want := range []string{
		"CONTEXTO REF_ONLY",
		"required_ref_action",
		"contexto_ref_only_resuelto",
		`"notes":["contexto_ref_only_resuelto: \u003cmotivo\u003e"]`,
	} {
		if !strings.Contains(string(prompt), want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, string(prompt))
		}
	}
}
