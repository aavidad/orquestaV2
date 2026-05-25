package orquestaruntimecodex

import (
	"context"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexExecResolverV0BloqueaSandboxNoPermitido(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.Sandbox = "read-only"

	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())

	requireCodexIssueFieldV0(t, issues, CodexConnectorValueInvalidV0, "sandbox")
}

func TestCodexExecResolverV0BloqueaApprovalInteractivoSinOptIn(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.ApprovalPolicy = CodexApprovalPolicyOnRequestV0

	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())

	requireCodexIssueFieldV0(t, issues, CodexConnectorValueInvalidV0, "approval_policy")
}

func TestCodexExecResolverV0AceptaApprovalInteractivoConOptInOperador(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.ApprovalPolicy = CodexApprovalPolicyOnRequestV0
	profile.InteractiveApprovalOptIn = true

	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestInferCodexRuntimeWorkDirPlacementV0DeclaraControlExterno(t *testing.T) {
	got := InferCodexRuntimeWorkDirPlacementV0("/tmp/project", "/tmp/runtime")

	if got != CodexRuntimeWorkDirExternalRootV0 {
		t.Fatalf("placement=%q", got)
	}
}

func requireCodexIssueFieldV0(
	t *testing.T,
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
	code CodexConnectorIssueCodeV0,
	field string,
) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == orquestaruntime.ExternalAgentConnectorErrorCodeV0(code) &&
			issue.Field == field {
			return
		}
	}
	t.Fatalf("no se encontro issue %q field=%q en %+v", code, field, issues)
}
