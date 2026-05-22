package orquestadirectoroperativo

func BuildOperationalDirectorAgentBudgetV0(
	plan OperationalDirectorPlanV0,
) OperationalDirectorAgentBudgetV0 {
	planned := plan.MaxParallelAgents
	if planned < 0 {
		planned = 0
	}
	recursive := plan.RecursiveDelegation && plan.MaxDelegationDepth > 0 && plan.MaxSubagentsPerAgent > 0
	if recursive {
		currentDepthAgents := plan.MaxParallelAgents
		for depth := 1; depth <= plan.MaxDelegationDepth; depth++ {
			currentDepthAgents *= plan.MaxSubagentsPerAgent
			planned += currentDepthAgents
		}
	}
	return OperationalDirectorAgentBudgetV0{
		MaxAgents:       plan.MaxRecursiveAgents,
		PlannedAgents:   planned,
		Exceeded:        plan.MaxRecursiveAgents > 0 && planned > plan.MaxRecursiveAgents,
		RecursiveLaunch: recursive,
	}
}
