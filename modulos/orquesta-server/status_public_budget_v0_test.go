package orquestaserver

import (
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestServerPublicStatusV0ExponePresupuestoAutomejoraIdleV0(t *testing.T) {
	status := NewServerPublicStatusV0(StateV0{
		Status: "running",
		IdleSelfImprovementBudget: orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0{
			Prepare:                          false,
			Reason:                           orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0,
			MaxContextBudgetBytesPerDay:      1000,
			ContextBudgetBytesUsedToday:      900,
			ContextBudgetBytesRemainingToday: 100,
			EstimatedNextContextBudgetBytes:  250,
		},
	})
	if status.IdleSelfImprovementBudget.Reason != orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0 ||
		status.IdleSelfImprovementBudget.ContextBudgetBytesRemainingToday != 100 ||
		status.IdleSelfImprovementBudget.EstimatedNextContextBudgetBytes != 250 {
		t.Fatalf("budget publico=%+v", status.IdleSelfImprovementBudget)
	}
}
