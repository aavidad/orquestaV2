package candidateseal

import "testing"

func TestGateBBindingUsesCanonicalGateAIdentityAndStaysNoGo(t *testing.T) {
	gateA, err := Build(candidateInput("gate-b"))
	if err != nil {
		t.Fatal(err)
	}
	connector := GateBConnector{
		Module:         "github.com/aavidad/agente_microvm/conectores/orquesta",
		Version:        "v0.0.0-20260816192103-3d9c68f20721",
		ModuleSum:      "h1:ijYWT4pPHQmVL6UVx5rNXHRbDzveIn1ykXp16LwWhcs=",
		ContractSHA256: testDigest('b'),
	}
	binding, err := BuildGateBBinding(gateA, connector)
	if err != nil {
		t.Fatal(err)
	}
	if binding.GateACandidateSHA256 != gateA.CandidateSHA256 || binding.Status != GateBStatus ||
		!binding.PhysicalReceiptsNeeded || binding.BindingSHA256[:7] != "sha256:" {
		t.Fatalf("unexpected binding: %+v", binding)
	}
	if err := VerifyGateBBinding(binding, gateA); err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeGateBBinding(binding)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeGateBBinding(encoded)
	if err != nil || decoded != binding {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}
}

func TestGateBBindingRejectsCrossedCandidateConnectorAndPrematureState(t *testing.T) {
	one, _ := Build(candidateInput("one"))
	two, _ := Build(candidateInput("two"))
	connector := GateBConnector{Module: "connector", Version: "v1", ModuleSum: "h1:sum", ContractSHA256: testDigest('c')}
	binding, err := BuildGateBBinding(one, connector)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyGateBBinding(binding, two) == nil {
		t.Fatal("crossed Gate A candidate accepted")
	}
	mutations := []func(*GateBBinding){
		func(v *GateBBinding) { v.Status = "passed" },
		func(v *GateBBinding) { v.PhysicalReceiptsNeeded = false },
		func(v *GateBBinding) { v.Protocol = "agentmicrovm.local.v0" },
		func(v *GateBBinding) { v.Connector.ContractSHA256 = testDigest('d') },
		func(v *GateBBinding) { v.BindingSHA256 = testDigest('e') },
	}
	for index, mutate := range mutations {
		changed := binding
		mutate(&changed)
		if VerifyGateBBinding(changed, one) == nil {
			t.Fatalf("mutation %d accepted", index)
		}
	}
}
