package orquestamcp

import "testing"

func TestRegisterMCPTransportV0DeclaraOutputBudgetComun(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	for name, resource := range transport.resources {
		budget := resource.OutputBudget
		if budget.MaxBytes <= 0 ||
			budget.Mode != MCPTransportOutputModeFullV0 ||
			budget.Freshness != MCPTransportOutputFreshnessStaticV0 ||
			budget.Overflow != MCPTransportResourcePayloadBlockedV0 {
			t.Fatalf("resource %s sin budget comun: %+v", name, budget)
		}
	}
	for name, tool := range transport.tools {
		budget := tool.OutputBudget
		if budget.MaxBytes <= 0 ||
			budget.Mode != MCPTransportOutputModeFullV0 ||
			budget.Freshness != MCPTransportOutputFreshnessLiveV0 ||
			budget.Overflow != MCPTransportToolPayloadBlockedV0 {
			t.Fatalf("tool %s sin budget comun: %+v", name, budget)
		}
	}
}
