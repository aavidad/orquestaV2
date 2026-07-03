package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestMCPAutoprogrammingStatusExecutorV0PublicaPresupuestoIdleV0(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		IdleSelfImprovementBudgetSource: fakeMCPAutoprogrammingIdleBudgetSourceV0{
			decision: orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0{
				Prepare:                              false,
				Reason:                               orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0,
				RequestedGoals:                       4,
				AllowedGoals:                         0,
				ContextBudgetBytesRemainingToday:     10,
				EstimatedNextContextBudgetBytes:      50,
				ContextBudgetEstimateEvidencePresent: true,
			},
			ok: true,
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		result.IdleSelfImprovementBudget == nil ||
		result.IdleSelfImprovementBudget.Reason != orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0 ||
		result.IdleSelfImprovementBudget.ContextBudgetBytesRemainingToday != 10 ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "idle_self_improvement_budget_deferred") {
		t.Fatalf("budget=%+v diagnostics=%+v", result.IdleSelfImprovementBudget, result.Diagnostics)
	}
}

func TestMCPTransportV0AutoprogrammingStatusPublicaPresupuestoIdleDesdeBindings(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		AutoprogrammingIdleSelfImprovementBudgetSource: fakeMCPAutoprogrammingIdleBudgetSourceV0{
			decision: orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0{
				Prepare:             true,
				Degraded:            true,
				Reason:              orquestaautoprogramming.AutoprogrammingIdleBudgetDegradedV0,
				RequestedGoals:      4,
				AllowedGoals:        1,
				GoalsRemainingToday: 1,
			},
			ok: true,
		},
	}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{},
	)
	if err != nil {
		t.Fatalf("call autoprogramming status: %v", err)
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, string(output))
	}
	if result.IdleSelfImprovementBudget == nil ||
		result.IdleSelfImprovementBudget.Reason != orquestaautoprogramming.AutoprogrammingIdleBudgetDegradedV0 ||
		result.IdleSelfImprovementBudget.AllowedGoals != 1 {
		t.Fatalf("budget=%+v", result.IdleSelfImprovementBudget)
	}
}

type fakeMCPAutoprogrammingIdleBudgetSourceV0 struct {
	decision orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0
	ok       bool
	err      error
}

func (source fakeMCPAutoprogrammingIdleBudgetSourceV0) AutoprogrammingIdleSelfImprovementBudgetV0(
	context.Context,
) (orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0, bool, error) {
	return source.decision, source.ok, source.err
}
