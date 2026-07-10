package orquestacapacity

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveModelRoutingV0TablaNeutralCompleta(t *testing.T) {
	policy := modelRoutingPolicyForTestV0()
	for _, test := range []struct {
		name     string
		request  ModelRoutingRequestV0
		model    string
		effort   string
		rejected bool
	}{
		{"trivial", ModelRoutingRequestV0{TaskRef: "task-trivial", Level: ModelRoutingLevelNormalV0, Trivial: true}, "model-ref-small", "low", false},
		{"normal", ModelRoutingRequestV0{TaskRef: "task-normal", Level: ModelRoutingLevelNormalV0}, "model-ref-standard", "medium", false},
		{"complex", ModelRoutingRequestV0{TaskRef: "task-complex", Level: ModelRoutingLevelComplexV0}, "model-ref-standard", "high", false},
		{"critical causal", ModelRoutingRequestV0{TaskRef: "task-critical", Level: ModelRoutingLevelCriticalV0, ReasonRef: "reason-ref", EvidenceRefs: []string{"evidence-ref"}}, "model-ref-critical", "high", false},
		{"critical sin causalidad", ModelRoutingRequestV0{TaskRef: "task-critical", Level: ModelRoutingLevelCriticalV0}, "", "", true},
		{"nivel vacio", ModelRoutingRequestV0{TaskRef: "task-empty"}, "", "", true},
		{"task vacia", ModelRoutingRequestV0{Level: ModelRoutingLevelNormalV0}, "", "", true},
		{"nivel desconocido", ModelRoutingRequestV0{TaskRef: "task-invalid", Level: "ad_hoc"}, "", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ResolveModelRoutingV0(policy, test.request)
			if got.SelectedModelRef != test.model || got.ReasoningEffort != test.effort || got.Rejected != test.rejected {
				t.Fatalf("decision=%+v", got)
			}
		})
	}
}

func TestResolveModelRoutingV0XHighSoloConAutorizacionCausalPorTask(t *testing.T) {
	policy := modelRoutingPolicyForTestV0()
	request := ModelRoutingRequestV0{
		TaskRef:         "task-xhigh",
		Level:           ModelRoutingLevelComplexV0,
		ReasonRef:       "reason-ref-xhigh",
		EvidenceRefs:    []string{"evidence-ref-xhigh"},
		RequestedEffort: "xhigh",
	}
	if got := ResolveModelRoutingV0(policy, request); !got.Rejected || got.RejectionRef != "model-routing-xhigh-authorization-missing" {
		t.Fatalf("xhigh sin authorization_ref=%+v", got)
	}
	request.XHighAuthorizationRef = "authorization-ref-task-xhigh"
	if got := ResolveModelRoutingV0(policy, request); got.Rejected || got.ReasoningEffort != "xhigh" || got.TaskRef != request.TaskRef {
		t.Fatalf("xhigh causal=%+v", got)
	}
	request.RequestedEffort = "max"
	if got := ResolveModelRoutingV0(policy, request); !got.Rejected || got.RejectionRef != "model-routing-effort-override-prohibited" {
		t.Fatalf("max=%+v", got)
	}
}

func TestResolveModelRoutingV0PoliticaVaciaFallaCerrado(t *testing.T) {
	for _, policy := range []ModelRoutingPolicyV0{
		{},
		{PolicyRef: "policy-ref", TrivialModelRef: "small", NormalModelRef: "normal", CriticalModelRef: "critical", TrivialEffort: "low", NormalEffort: "medium", ComplexEffort: "high", CriticalEffort: "high"},
		{PolicyRef: "policy-ref", TrivialModelRef: "small", NormalModelRef: "normal", CriticalModelRef: "critical"},
		{PolicyRef: "policy-ref", TrivialModelRef: "small", NormalModelRef: "normal", CriticalModelRef: "critical", TrivialEffort: "low", NormalEffort: "medium", ComplexEffort: "max", CriticalEffort: "high"},
	} {
		got := ResolveModelRoutingV0(policy, ModelRoutingRequestV0{TaskRef: "task-ref", Level: ModelRoutingLevelNormalV0})
		if !got.Rejected || got.SelectedModelRef != "" || got.ReasoningEffort != "" {
			t.Fatalf("decision=%+v", got)
		}
	}
}

func TestNeutralCorePackagesDoNotContainConcreteProviderModelsRecursivelyV0(t *testing.T) {
	for _, pkg := range []string{
		"orquesta-core",
		"orquesta-core-workflow",
		"orquesta-domain-work",
		"orquesta-goal",
		"orquesta-orchestration-core",
	} {
		err := filepath.WalkDir(filepath.Join("..", pkg), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || filepath.Ext(path) != ".go" {
				return err
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			body := strings.ToLower(string(data))
			for _, concrete := range []string{"gpt-5.6", "haiku-4.5", "sonnet-5", "fable-5", "opus-4.8"} {
				if strings.Contains(body, concrete) {
					t.Fatalf("modelo concreto %q en paquete neutral: %s", concrete, path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func modelRoutingPolicyForTestV0() ModelRoutingPolicyV0 {
	return ModelRoutingPolicyV0{
		PolicyRef:        "policy-ref-model-routing-v0",
		Strict:           true,
		TrivialModelRef:  "model-ref-small",
		NormalModelRef:   "model-ref-standard",
		CriticalModelRef: "model-ref-critical",
		TrivialEffort:    "low",
		NormalEffort:     "medium",
		ComplexEffort:    "high",
		CriticalEffort:   "high",
	}
}
