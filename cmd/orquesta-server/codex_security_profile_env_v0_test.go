package main

import (
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestCodexRuntimeConfigV0ConservaSandboxInvalidoParaValidacionPublica(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_SANDBOX", "read-only")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "read-only" {
		t.Fatalf("sandbox normalizado silenciosamente: %q", config.Sandbox)
	}
}

func TestCodexRuntimeConfigV0ApprovalInteractivoRequiereOptInOperador(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_APPROVAL_POLICY", "on-request")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.InteractiveApprovalOptIn {
		t.Fatalf("approval interactivo quedo habilitado sin opt-in")
	}
}

func TestCodexRuntimeConfigV0ApprovalInteractivoOptInExplicito(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_ALLOW_INTERACTIVE_APPROVAL", "1")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if !config.InteractiveApprovalOptIn {
		t.Fatalf("approval interactivo no preservo opt-in explicito")
	}
}
