package orquestaappcodexstack

import (
	"testing"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

func TestBuildStackV0NormalizaModelRoutingCeroConDefaultsEstricos(t *testing.T) {
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.Codex.ModelRouting = CodexModelRoutingConfigV0{}
	config.Claude.ModelRouting = ClaudeModelRoutingConfigV0{}

	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	if decision, model, err := codexModelRouteForTaskV0(stack.Codex, "task-normal"); err != nil || model != "gpt-5.5" || decision.ReasoningEffort != "medium" {
		t.Fatalf("codex default decision=%+v model=%q err=%v", decision, model, err)
	}
	if decision, model, err := claudeModelRouteForTaskV0(stack.Claude, "task-normal"); err != nil || model != "sonnet-5" || decision.ReasoningEffort != "medium" {
		t.Fatalf("claude default decision=%+v model=%q err=%v", decision, model, err)
	}
}

func TestBuildStackV0NoDefaultaModelRoutingParcial(t *testing.T) {
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.Codex.ModelRouting = CodexModelRoutingConfigV0{
		Policy: orquestacapacity.ModelRoutingPolicyV0{PolicyRef: "partial-policy"},
	}
	if _, err := BuildStackV0(config); err != nil {
		t.Fatalf("BuildStackV0 no debe resolver rutas parciales, err=%v", err)
	}
	if _, model, err := codexModelRouteForTaskV0(config.Codex, "task-normal"); err == nil || model != "" {
		t.Fatalf("routing Codex parcial debe fallar cerrado en la ruta, model=%q err=%v", model, err)
	}

	config = codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.Claude.ModelRouting = ClaudeModelRoutingConfigV0{
		ModelAlias: map[string]string{"sonnet": "sonnet-5"},
	}
	if decision, model, err := claudeModelRouteForTaskV0(config.Claude, "task-normal"); err == nil || model != "" {
		t.Fatalf("routing Claude parcial debe fallar cerrado en la ruta, decision=%+v model=%q err=%v", decision, model, err)
	}
}
