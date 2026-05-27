package orquestamcp

import "testing"

func TestMCPTransportV0DeclaraExecutionBudgetPorPerfil(t *testing.T) {
	resources := MCPTransportResourcesV0()
	if len(resources) == 0 {
		t.Fatalf("resources vacio")
	}
	for _, resource := range resources {
		budget := NormalizeMCPTransportExecutionBudgetV0(
			resource.ExecutionBudget,
			MCPTransportExecutionProfileResourceReadV0,
		)
		if budget.Profile != MCPTransportExecutionProfileResourceReadV0 ||
			budget.MaxDuration <= 0 ||
			budget.TimeoutCode != MCPTransportExecutionTimeoutV0 ||
			budget.CancelledCode != MCPTransportExecutionCancelledV0 {
			t.Fatalf("resource %s sin execution budget read-only: %+v", resource.Name, budget)
		}
	}

	tools := MCPTransportToolsV0(MCPTransportBindingsV0{})
	profiles := map[string]bool{}
	for _, tool := range tools {
		budget := NormalizeMCPTransportExecutionBudgetV0(
			tool.ExecutionBudget,
			MCPTransportExecutionProfileControlPlaneMutationV0,
		)
		if budget.MaxDuration <= 0 || budget.TimeoutCode == "" || budget.CancelledCode == "" {
			t.Fatalf("tool %s sin execution budget: %+v", tool.Name, budget)
		}
		profiles[budget.Profile] = true
	}
	for _, profile := range []string{
		MCPTransportExecutionProfileDefaultToolV0,
		MCPTransportExecutionProfileControlPlaneMutationV0,
		MCPTransportExecutionProfileAutoprogrammingLongV0,
	} {
		if !profiles[profile] {
			t.Fatalf("perfil %s no declarado en tools: %+v", profile, profiles)
		}
	}
}
