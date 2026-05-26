package orquestadirectoroperativo

import "testing"

func TestBuildOperationalDirectorAgentBudgetV0CalculaArbolRecursivo(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(func(request *OperationalDirectorRequestV0) {
		request.MaxParallelAgents = 2
		request.AllowRecursiveDelegation = true
		request.MaxDelegationDepth = 2
		request.MaxSubagentsPerAgent = 3
		request.MaxRecursiveAgents = 25
	}))

	budget := BuildOperationalDirectorAgentBudgetV0(result.Plan)

	if budget.PlannedAgents != 26 ||
		budget.MaxAgents != 25 ||
		!budget.Exceeded ||
		!budget.RecursiveLaunch {
		t.Fatalf("budget=%+v", budget)
	}
}

func TestBuildOperationalDirectorAgentBudgetV0BudgetCeroNoBloquea(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(func(request *OperationalDirectorRequestV0) {
		request.MaxParallelAgents = 2
		request.AllowRecursiveDelegation = true
		request.MaxDelegationDepth = 2
		request.MaxSubagentsPerAgent = 3
		request.MaxRecursiveAgents = 0
	}))

	budget := BuildOperationalDirectorAgentBudgetV0(result.Plan)

	if budget.PlannedAgents != 26 ||
		budget.MaxAgents != 0 ||
		budget.Exceeded {
		t.Fatalf("budget=%+v", budget)
	}
}

func TestBuildOperationalDirectorAgentBudgetV0CapacidadCanonicaDiezPadresSeisHijos(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(func(request *OperationalDirectorRequestV0) {
		request.MaxParallelAgents = 10
		request.AllowRecursiveDelegation = true
		request.MaxDelegationDepth = 1
		request.MaxSubagentsPerAgent = 6
		request.MaxRecursiveAgents = 70
	}))

	budget := BuildOperationalDirectorAgentBudgetV0(result.Plan)

	if result.Plan.MaxParallelAgents != 10 ||
		result.Plan.MaxSubagentsPerAgent != 6 ||
		budget.PlannedAgents != 70 ||
		budget.MaxAgents != 70 ||
		budget.Exceeded ||
		!budget.RecursiveLaunch {
		t.Fatalf("plan=%+v budget=%+v", result.Plan, budget)
	}
}
