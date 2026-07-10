package orquestaappcodexstack

import (
	"testing"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

func TestClaudeModelRouteForTaskV0TablaLifecycleSinHerencia(t *testing.T) {
	config := claudeRoutingConfigForTestV0()
	config.ModelRouting.TaskRoutes = map[string]orquestacapacity.ModelRoutingRequestV0{
		"trivial":  {Level: orquestacapacity.ModelRoutingLevelNormalV0, Trivial: true},
		"complex":  {Level: orquestacapacity.ModelRoutingLevelComplexV0},
		"critical": {Level: orquestacapacity.ModelRoutingLevelCriticalV0, ReasonRef: "reason", EvidenceRefs: []string{"evidence"}},
	}
	for _, test := range []struct{ task, model, effort string }{
		{"trivial", "haiku-4.5", "low"},
		{"normal", "sonnet-5", "medium"},
		{"complex", "sonnet-5", "high"},
		{"critical", "fable-5", "high"},
		{"critical-retry", "sonnet-5", "medium"},
		{"critical-child", "sonnet-5", "medium"},
	} {
		decision, model, err := claudeModelRouteForTaskV0(config, test.task)
		if err != nil || model != test.model || decision.ReasoningEffort != test.effort || decision.TaskRef != test.task {
			t.Fatalf("task=%s decision=%+v model=%q err=%v", test.task, decision, model, err)
		}
	}
}

func TestClaudeModelRouteForTaskV0RechazaOpusXHighMaxYOmisiones(t *testing.T) {
	base := claudeRoutingConfigForTestV0()
	base.ModelRouting.ModelAlias["sonnet"] = "opus-4.8"
	if _, _, err := claudeModelRouteForTaskV0(base, "normal"); err == nil {
		t.Fatal("opus automatico aceptado")
	}
	base = claudeRoutingConfigForTestV0()
	base.ModelRouting.TaskRoutes["xhigh"] = orquestacapacity.ModelRoutingRequestV0{Level: orquestacapacity.ModelRoutingLevelComplexV0, RequestedEffort: "xhigh"}
	if _, _, err := claudeModelRouteForTaskV0(base, "xhigh"); err == nil {
		t.Fatal("xhigh sin autorizacion aceptado")
	}
	base.ModelRouting.TaskRoutes["xhigh"] = orquestacapacity.ModelRoutingRequestV0{Level: orquestacapacity.ModelRoutingLevelComplexV0, RequestedEffort: "xhigh", ReasonRef: "reason", EvidenceRefs: []string{"evidence"}, XHighAuthorizationRef: "authorization-ref-task-xhigh"}
	if decision, model, err := claudeModelRouteForTaskV0(base, "xhigh"); err != nil || decision.ReasoningEffort != "xhigh" || model != "sonnet-5" {
		t.Fatalf("xhigh autorizado=%+v model=%q err=%v", decision, model, err)
	}
	base.ModelRouting.TaskRoutes["max"] = orquestacapacity.ModelRoutingRequestV0{Level: orquestacapacity.ModelRoutingLevelComplexV0, RequestedEffort: "max"}
	if _, _, err := claudeModelRouteForTaskV0(base, "max"); err == nil {
		t.Fatal("max aceptado")
	}
	delete(base.ModelRouting.ModelAlias, "sonnet")
	if _, _, err := claudeModelRouteForTaskV0(base, "normal"); err == nil {
		t.Fatal("omision de alias aceptada")
	}
}

func claudeRoutingConfigForTestV0() ClaudeRuntimeConfigV0 {
	return ClaudeRuntimeConfigV0{ModelRouting: ClaudeModelRoutingConfigV0{
		Policy: orquestacapacity.ModelRoutingPolicyV0{
			PolicyRef: "claude-policy", Strict: true,
			TrivialModelRef: "haiku", NormalModelRef: "sonnet", CriticalModelRef: "fable",
			TrivialEffort: "low", NormalEffort: "medium", ComplexEffort: "high", CriticalEffort: "high",
		},
		ModelAlias: map[string]string{"haiku": "haiku-4.5", "sonnet": "sonnet-5", "fable": "fable-5"},
		TaskRoutes: map[string]orquestacapacity.ModelRoutingRequestV0{},
	}}
}
