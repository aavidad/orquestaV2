package main

import (
	"context"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerAutoprogrammingIdleBudgetSourceV0LeeEstadoDurable(t *testing.T) {
	config := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{StateDir: t.TempDir()})
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err != nil {
		t.Fatalf("NewFileStateStoreV0: %v", err)
	}
	if err := store.SaveServerStateV0(context.Background(), orquestaserver.StateV0{
		IdleSelfImprovementBudget: orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0{
			Reason:                          orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0,
			AllowedGoals:                    0,
			EstimatedNextContextBudgetBytes: 123,
		},
	}); err != nil {
		t.Fatalf("SaveServerStateV0: %v", err)
	}
	source, err := newServerAutoprogrammingIdleBudgetSourceV0(config)
	if err != nil {
		t.Fatalf("source: %v", err)
	}

	decision, ok, err := source.AutoprogrammingIdleSelfImprovementBudgetV0(context.Background())
	if err != nil {
		t.Fatalf("AutoprogrammingIdleSelfImprovementBudgetV0: %v", err)
	}
	if !ok ||
		decision.Reason != orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0 ||
		decision.EstimatedNextContextBudgetBytes != 123 {
		t.Fatalf("decision=%+v ok=%v", decision, ok)
	}
}

func TestServerAutoprogrammingIdleBudgetSourceV0OmiteDecisionVacia(t *testing.T) {
	config := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{StateDir: t.TempDir()})
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err != nil {
		t.Fatalf("NewFileStateStoreV0: %v", err)
	}
	if err := store.SaveServerStateV0(context.Background(), orquestaserver.StateV0{}); err != nil {
		t.Fatalf("SaveServerStateV0: %v", err)
	}
	source, err := newServerAutoprogrammingIdleBudgetSourceV0(config)
	if err != nil {
		t.Fatalf("source: %v", err)
	}

	decision, ok, err := source.AutoprogrammingIdleSelfImprovementBudgetV0(context.Background())
	if err != nil {
		t.Fatalf("AutoprogrammingIdleSelfImprovementBudgetV0: %v", err)
	}
	if ok || decision.Reason != "" {
		t.Fatalf("decision=%+v ok=%v", decision, ok)
	}
}
