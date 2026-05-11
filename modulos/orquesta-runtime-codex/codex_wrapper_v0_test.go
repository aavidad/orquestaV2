package orquestaruntimecodex

import (
	"strings"
	"testing"
)

func TestCodexWrapperV0WorkspaceWriteAutorizaRuntimeParaACK(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.Sandbox = "workspace-write"

	wrapper := BuildCodexWrapperScriptV0(profile)
	want := "--add-dir " + shellQuoteV0(profile.RuntimeWorkDir)
	if !strings.Contains(wrapper, want) {
		t.Fatalf("wrapper no autoriza runtime para ACK: falta %q\n%s", want, wrapper)
	}
}

func TestCodexWrapperV0NoAnadeRuntimeWritableSinWorkspaceWrite(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.Sandbox = "danger-full-access"

	wrapper := BuildCodexWrapperScriptV0(profile)
	if strings.Contains(wrapper, "--add-dir ") {
		t.Fatalf("wrapper no debe anadir add-dir fuera de workspace-write:\n%s", wrapper)
	}
}

func TestCodexWrapperV0FijaEsfuerzoDeRazonamiento(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.ReasoningEffort = "medium"

	wrapper := BuildCodexWrapperScriptV0(profile)
	want := "-c " + shellQuoteV0(`model_reasoning_effort="medium"`)
	if !strings.Contains(wrapper, want) {
		t.Fatalf("wrapper no fija esfuerzo de razonamiento: falta %q\n%s", want, wrapper)
	}
}

func TestCodexProfileV0WorkspaceWriteRechazaRuntimeFueraDelProyecto(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.RuntimeWorkDir = t.TempDir()

	issues := ValidateCodexConnectorProfileV0(profile)

	requireCodexIssueV0(t, issues, CodexConnectorPathInvalidV0)
}
