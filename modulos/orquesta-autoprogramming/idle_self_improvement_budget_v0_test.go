package orquestaautoprogramming

import "testing"

func TestDecideAutoprogrammingIdleSelfImprovementBudgetV0SinConfigNoCambiaComportamiento(t *testing.T) {
	decision := DecideAutoprogrammingIdleSelfImprovementBudgetV0(
		AutoprogrammingIdleSelfImprovementBudgetDecisionInputV0{RequestedGoals: 3},
	)
	if !decision.Prepare ||
		decision.Reason != AutoprogrammingIdleBudgetUnconfiguredV0 ||
		decision.AllowedGoals != 3 ||
		decision.Degraded {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestDecideAutoprogrammingIdleSelfImprovementBudgetV0AplazaAgotado(t *testing.T) {
	decision := DecideAutoprogrammingIdleSelfImprovementBudgetV0(
		AutoprogrammingIdleSelfImprovementBudgetDecisionInputV0{
			Config: AutoprogrammingIdleSelfImprovementBudgetConfigV0{
				MaxGoalsPerDay: 2,
			},
			Usage: AutoprogrammingIdleSelfImprovementBudgetUsageV0{
				GoalsUsedToday: 2,
			},
			RequestedGoals: 1,
		},
	)
	if decision.Prepare ||
		decision.Reason != AutoprogrammingIdleBudgetDeferredV0 ||
		decision.AllowedGoals != 0 ||
		decision.GoalsRemainingToday != 0 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestDecideAutoprogrammingIdleSelfImprovementBudgetV0DegradaLote(t *testing.T) {
	decision := DecideAutoprogrammingIdleSelfImprovementBudgetV0(
		AutoprogrammingIdleSelfImprovementBudgetDecisionInputV0{
			Config: AutoprogrammingIdleSelfImprovementBudgetConfigV0{
				MaxContextBudgetBytesPerDay: 1000,
			},
			Usage: AutoprogrammingIdleSelfImprovementBudgetUsageV0{
				ContextBudgetBytesUsedToday:     100,
				EstimatedNextContextBudgetBytes: 300,
			},
			RequestedGoals: 4,
		},
	)
	if !decision.Prepare ||
		!decision.Degraded ||
		decision.Reason != AutoprogrammingIdleBudgetDegradedV0 ||
		decision.AllowedGoals != 3 ||
		decision.ContextBudgetBytesRemainingToday != 900 ||
		!decision.ContextBudgetEstimateEvidencePresent {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestDecideAutoprogrammingIdleSelfImprovementBudgetV0UsaEstimacionRealParaAplazar(t *testing.T) {
	decision := DecideAutoprogrammingIdleSelfImprovementBudgetV0(
		AutoprogrammingIdleSelfImprovementBudgetDecisionInputV0{
			Config: AutoprogrammingIdleSelfImprovementBudgetConfigV0{
				MaxContextBudgetBytesPerDay: 1000,
			},
			Usage: AutoprogrammingIdleSelfImprovementBudgetUsageV0{
				ContextBudgetBytesUsedToday:     800,
				EstimatedNextContextBudgetBytes: 300,
			},
			RequestedGoals: 1,
		},
	)
	if decision.Prepare ||
		decision.Reason != AutoprogrammingIdleBudgetDeferredV0 ||
		decision.EstimatedNextContextBudgetBytes != 300 ||
		!decision.ContextBudgetEstimateEvidencePresent {
		t.Fatalf("decision=%+v", decision)
	}
}
