package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCapacityDispatcherV0InyectaPolicyPort(t *testing.T) {
	dispatcher := capacityDispatcherV0(ConfigV0{Capacity: CapacityConfigV0{
		Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
		ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		PolicyRef:       "capacity-policy-ref-stack-test",
		PoolRef:         "capacity-pool-ref-stack-test",
		ModelRef:        "capacity-model-ref-stack-test",
		QuotaRef:        "capacity-quota-ref-stack-test",
		EvidenceRefs:    []string{"evidence-ref-stack-capacity-policy"},
	}})
	executor, ok := dispatcher.Executor.(orquestacionnucleoapp.CapacityDecisionExecutorV0)
	if !ok {
		t.Fatalf("executor=%T", dispatcher.Executor)
	}
	decision, err := executor.Policy.DecideCapacityV0(context.Background(), orquestacionnucleoapp.CapacityDecisionPolicyRequestV0{
		DecisionRef: "capacity-decision-ref-stack-test",
		Request: orquestacoreworkflow.CapacityDecisionRequestV0{
			CapacityRequestID: "capacity-ref-stack-test",
			RunID:             "run-ref-stack-test",
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			ReasonCode:        "policy-port",
			Summary:           "Validar wiring de policy.",
		},
	})
	if err != nil {
		t.Fatalf("DecideCapacityV0: %v", err)
	}
	if decision.PoolRef != "capacity-pool-ref-stack-test" ||
		decision.ModelRef != "capacity-model-ref-stack-test" ||
		decision.QuotaRef != "capacity-quota-ref-stack-test" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestStackCapacityPolicyAdapterV0MapeaContratoCapacity(t *testing.T) {
	adapter := StackCapacityPolicyAdapterV0{Policy: orquestacapacity.DefaultCapacityPolicyV0{
		PolicyRef: "capacity-policy-ref-adapter-test",
		PoolRef:   "capacity-pool-ref-adapter-test",
		ModelRef:  "capacity-model-ref-adapter-test",
		QuotaRef:  "capacity-quota-ref-adapter-test",
	}}
	decision, err := adapter.DecideCapacityV0(context.Background(), orquestacionnucleoapp.CapacityDecisionPolicyRequestV0{
		DecisionRef:  "capacity-decision-ref-adapter-test",
		DefaultTier:  orquestacoreworkflow.OrchestrationCapacityMediumV0,
		EvidenceRefs: []string{"evidence-ref-adapter-test"},
		Request: orquestacoreworkflow.CapacityDecisionRequestV0{
			CapacityRequestID: "capacity-ref-adapter-test",
			RunID:             "run-ref-adapter-test",
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			ReasonCode:        "policy-port",
			Summary:           "Validar adapter.",
		},
	})
	if err != nil {
		t.Fatalf("DecideCapacityV0: %v", err)
	}
	if decision.PolicyRef != "capacity-policy-ref-adapter-test" ||
		decision.Tier != orquestacoreworkflow.OrchestrationCapacityMediumV0 {
		t.Fatalf("decision=%+v", decision)
	}
}
