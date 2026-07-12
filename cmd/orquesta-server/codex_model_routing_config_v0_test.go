package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

func TestCodexModelRoutingConfigV0RoundTripYEffectiveRefs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orquesta.config.json")
	raw := `{"schema_version":"orquesta_config.v0","codex_model_routing":{"policy_ref":"policy-ref-test","strict":true,"aliases":{"codex-model-ref-luna-v0":"gpt-5.6-luna","codex-model-ref-terra-v0":"gpt-5.6-terra","codex-model-ref-sol-v0":"gpt-5.6-sol"},"task_routes":{"task-ref-critical":{"level":"critical","reason_ref":"reason-ref","evidence_refs":["evidence-ref"]}}}}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	project, ok, err := loadServerProjectConfigPathV0(path)
	if err != nil || !ok {
		t.Fatalf("project=%+v ok=%t err=%v", project, ok, err)
	}
	config := codexModelRoutingFromProjectConfigFileV0(project)
	if !config.Policy.Strict || config.Policy.PolicyRef != "policy-ref-test" || len(codexModelRoutingAliasRefsV0(config)) != 3 {
		t.Fatalf("config=%+v", config)
	}
	if source := codexModelRoutingConfigSourceV0(project); source != configSettingSourceConfigFileV0 {
		t.Fatalf("source=%q", source)
	}
	if got := codexModelRoutingEffortsSummaryV0(config); got != "trivial=low,normal=medium,complex=high,critical=high" {
		t.Fatalf("efforts=%q", got)
	}
}

func TestCodexModelRoutingConfigV0VaciosFallanCerrado(t *testing.T) {
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
			config := codexModelRoutingFromProjectConfigFileV0(serverProjectConfigFileV0{
				CodexModelRouting: &serverProjectConfigCodexModelRoutingV0{PolicyRef: test.policyRef, Strict: test.strict},
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

func TestCodexModelRoutingConfigV0AusenteMaterializaAliasesCanonicos(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orquesta.config.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	project, ok, err := loadServerProjectConfigPathV0(path)
	if err != nil || !ok {
		t.Fatalf("project=%+v ok=%t err=%v", project, ok, err)
	}
	config := codexModelRoutingFromProjectConfigFileV0(project)
	decision := orquestacapacity.ResolveModelRoutingV0(config.Policy, orquestacapacity.ModelRoutingRequestV0{TaskRef: "legacy-goal", Level: orquestacapacity.ModelRoutingLevelNormalV0})
	if decision.Rejected || config.ModelAlias[decision.SelectedModelRef] != "gpt-5.5" {
		t.Fatalf("decision=%+v aliases=%+v", decision, config.ModelAlias)
	}
}

func TestCodexModelRoutingConfigV0BloqueParcialRechaza(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orquesta.config.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0","codex_model_routing":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	project, ok, err := loadServerProjectConfigPathV0(path)
	if err != nil || !ok {
		t.Fatalf("project=%+v ok=%t err=%v", project, ok, err)
	}
	config := codexModelRoutingFromProjectConfigFileV0(project)
	decision := orquestacapacity.ResolveModelRoutingV0(config.Policy, orquestacapacity.ModelRoutingRequestV0{TaskRef: "partial-goal", Level: orquestacapacity.ModelRoutingLevelNormalV0})
	if decision.Rejected || config.ModelAlias[decision.SelectedModelRef] != "" {
		t.Fatalf("decision=%+v aliases=%+v", decision, config.ModelAlias)
	}
}
