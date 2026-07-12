package orquestaappcodexstack

import (
	"testing"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

func TestCodexModelRouteForTaskV0TablaLifecycleSinHerencia(t *testing.T) {
	config := CodexRuntimeConfigV0{Model: "global-prohibido", ModelRouting: CodexModelRoutingConfigV0{
		Policy: orquestacapacity.ModelRoutingPolicyV0{
			PolicyRef: "policy-ref-route", Strict: true,
			TrivialModelRef: "luna", NormalModelRef: "terra", CriticalModelRef: "sol",
			TrivialEffort: "low", NormalEffort: "medium", ComplexEffort: "high", CriticalEffort: "high",
		},
		ModelAlias: map[string]string{"luna": "gpt-5.6-luna", "terra": "gpt-5.6-terra", "sol": "gpt-5.6-sol"},
		TaskRoutes: map[string]orquestacapacity.ModelRoutingRequestV0{
			"task-trivial":  {Level: orquestacapacity.ModelRoutingLevelNormalV0, Trivial: true},
			"task-complex":  {Level: orquestacapacity.ModelRoutingLevelComplexV0},
			"task-critical": {Level: orquestacapacity.ModelRoutingLevelCriticalV0, ReasonRef: "reason-ref", EvidenceRefs: []string{"evidence-ref"}},
		},
	}}
	for _, test := range []struct{ task, model, effort string }{
		{"task-trivial", "gpt-5.6-luna", "low"},
		{"task-normal", "gpt-5.6-terra", "medium"},
		{"task-complex", "gpt-5.6-terra", "high"},
		{"task-critical", "gpt-5.6-sol", "high"},
		{"task-critical-retry", "gpt-5.6-terra", "medium"},
		{"task-critical-child", "gpt-5.6-terra", "medium"},
	} {
		decision, model, err := codexModelRouteForTaskV0(config, test.task)
		if err != nil || model != test.model || decision.ReasoningEffort != test.effort || decision.TaskRef != test.task {
			t.Fatalf("task=%s model=%s decision=%+v err=%v", test.task, model, decision, err)
		}
	}
}

func TestCodexModelRouteForTaskV0CriticalYXHighRequierenCausalidad(t *testing.T) {
	config := codexRoutingConfigForTestV0()
	config.ModelRouting.TaskRoutes["task-critical"] = orquestacapacity.ModelRoutingRequestV0{Level: orquestacapacity.ModelRoutingLevelCriticalV0}
	if decision, model, err := codexModelRouteForTaskV0(config, "task-critical"); err == nil || !decision.Rejected || model != "" {
		t.Fatalf("critical=%+v model=%q err=%v", decision, model, err)
	}
	config.ModelRouting.TaskRoutes["task-xhigh"] = orquestacapacity.ModelRoutingRequestV0{Level: orquestacapacity.ModelRoutingLevelComplexV0, RequestedEffort: "xhigh", ReasonRef: "reason", EvidenceRefs: []string{"evidence"}}
	if decision, _, err := codexModelRouteForTaskV0(config, "task-xhigh"); err == nil || !decision.Rejected {
		t.Fatalf("xhigh=%+v err=%v", decision, err)
	}
	config.ModelRouting.TaskRoutes["task-xhigh"] = orquestacapacity.ModelRoutingRequestV0{Level: orquestacapacity.ModelRoutingLevelComplexV0, RequestedEffort: "xhigh", ReasonRef: "reason", EvidenceRefs: []string{"evidence"}, XHighAuthorizationRef: "authorization-ref-task-xhigh"}
	if decision, model, err := codexModelRouteForTaskV0(config, "task-xhigh"); err != nil || decision.ReasoningEffort != "xhigh" || model != "gpt-5.6-terra" {
		t.Fatalf("xhigh autorizado=%+v model=%q err=%v", decision, model, err)
	}
}

func TestCodexModelRouteForTaskV0NoUsaModeloGlobalComoFallback(t *testing.T) {
	decision, model, err := codexModelRouteForTaskV0(CodexRuntimeConfigV0{Model: "gpt-5.6-sol"}, "task-normal")
	if err != nil || model != "gpt-5.5" || decision.ReasoningEffort != "medium" || decision.Rejected {
		t.Fatalf("decision=%+v model=%q err=%v", decision, model, err)
	}
}

func codexRoutingConfigForTestV0() CodexRuntimeConfigV0 {
	return CodexRuntimeConfigV0{ModelRouting: CodexModelRoutingConfigV0{
		Policy: orquestacapacity.ModelRoutingPolicyV0{
			PolicyRef: "policy", Strict: true,
			TrivialModelRef: "luna", NormalModelRef: "terra", CriticalModelRef: "sol",
			TrivialEffort: "low", NormalEffort: "medium", ComplexEffort: "high", CriticalEffort: "high",
		},
		ModelAlias: map[string]string{"luna": "gpt-5.6-luna", "terra": "gpt-5.6-terra", "sol": "gpt-5.6-sol"},
		TaskRoutes: map[string]orquestacapacity.ModelRoutingRequestV0{},
	}}
}
