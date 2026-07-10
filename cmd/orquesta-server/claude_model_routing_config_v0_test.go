package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

func TestClaudeModelRoutingConfigV0RoundTripYEffectiveRefs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orquesta.config.json")
	raw := `{"schema_version":"orquesta_config.v0","claude_model_routing":{"policy_ref":"policy-ref-claude","strict":true,"aliases":{"claude-model-ref-haiku-v0":"haiku-4.5","claude-model-ref-sonnet-v0":"sonnet-5","claude-model-ref-fable-v0":"fable-5"},"task_routes":{"task-ref-critical":{"level":"critical","reason_ref":"reason-ref","evidence_refs":["evidence-ref"]}}}}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	project, ok, err := loadServerProjectConfigPathV0(path)
	if err != nil || !ok {
		t.Fatalf("project=%+v ok=%t err=%v", project, ok, err)
	}
	config := claudeModelRoutingFromProjectConfigFileV0(project)
	if !config.Policy.Strict || config.Policy.PolicyRef != "policy-ref-claude" || len(claudeModelRoutingAliasRefsV0(config)) != 3 {
		t.Fatalf("config=%+v", config)
	}
	if got := claudeModelRoutingEffortsSummaryV0(config); got != "trivial=low,normal=medium,complex=high,critical=high" {
		t.Fatalf("efforts=%q", got)
	}
}

func TestClaudeModelRoutingConfigV0VaciosFallanCerrado(t *testing.T) {
	empty := ""
	nonStrict := false
	for _, test := range []struct {
		name      string
		policyRef *string
		strict    *bool
		rejection string
	}{
		{name: "policy ref vacia", policyRef: &empty, rejection: "model-routing-policy-ref-missing"},
		{name: "strict desactivado", strict: &nonStrict, rejection: "model-routing-policy-not-strict"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := claudeModelRoutingFromProjectConfigFileV0(serverProjectConfigFileV0{
				ClaudeModelRouting: &serverProjectConfigClaudeModelRoutingV0{PolicyRef: test.policyRef, Strict: test.strict},
			})
			decision := orquestacapacity.ResolveModelRoutingV0(config.Policy, orquestacapacity.ModelRoutingRequestV0{
				TaskRef: "task-ref", Level: orquestacapacity.ModelRoutingLevelNormalV0,
			})
			if !decision.Rejected || decision.RejectionRef != test.rejection {
				t.Fatalf("decision=%+v", decision)
			}
		})
	}
}

func TestClaudeModelRoutingConfigV0AusenteMaterializaAliasesCanonicos(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orquesta.config.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	project, ok, err := loadServerProjectConfigPathV0(path)
	if err != nil || !ok {
		t.Fatalf("project=%+v ok=%t err=%v", project, ok, err)
	}
	config := claudeModelRoutingFromProjectConfigFileV0(project)
	decision, model, err := claudeGoalModelRouteV0(config)
	if err != nil || decision.Rejected || model != "sonnet-5" {
		t.Fatalf("decision=%+v model=%q err=%v", decision, model, err)
	}
}

func TestClaudeModelRoutingConfigV0BloqueParcialRechaza(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orquesta.config.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0","claude_model_routing":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	project, ok, err := loadServerProjectConfigPathV0(path)
	if err != nil || !ok {
		t.Fatalf("project=%+v ok=%t err=%v", project, ok, err)
	}
	config := claudeModelRoutingFromProjectConfigFileV0(project)
	if decision, model, err := claudeGoalModelRouteV0(config); err == nil || decision.Rejected || model != "" {
		t.Fatalf("decision=%+v model=%q err=%v", decision, model, err)
	}
}
