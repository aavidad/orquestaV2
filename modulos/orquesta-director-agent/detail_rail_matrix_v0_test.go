package orquestadirectoragent

import (
	"testing"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func TestDirectorAgentDetailRailExternalMatrixV0(t *testing.T) {
	t.Setenv(orquestarails.DetailProhibitedRailsEnvV0, "on")

	decision := validDirectorAgentCreateMicrotaskDecisionV0()
	decision.Summary = "coordinar codex adapter, runtime, provider, model, git, db, sql, prompt policy y transcript policy."
	decision.CreateMicrotask.Task.Summary = "mantener web_application como alias reparable y token budget como ref opaca."
	decision.CreateMicrotask.Task.ContextRefs = []string{
		"context-ref-runtime-provider-model",
		"context-ref-token-budget",
		"context-ref-prompt-policy",
	}

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("decision bloqueada por falso positivo: %+v", issues)
	}
}

func TestDirectorAgentDetailRailMatrixRejectsSensitiveValueV0(t *testing.T) {
	t.Setenv(orquestarails.DetailProhibitedRailsEnvV0, "on")

	decision := validDirectorAgentDecisionV0()
	decision.Summary = "usar client_secret=abc123"

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_texto_invalido",
	)
}
