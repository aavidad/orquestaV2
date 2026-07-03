package main

import (
	"context"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverAutoprogrammingIdleBudgetSourceV0 struct {
	store orquestaserver.StateStorePortV0
}

func newServerAutoprogrammingIdleBudgetSourceV0(
	config orquestaserver.ConfigV0,
) (serverAutoprogrammingIdleBudgetSourceV0, error) {
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err != nil {
		return serverAutoprogrammingIdleBudgetSourceV0{}, err
	}
	return serverAutoprogrammingIdleBudgetSourceV0{store: store}, nil
}

func (source serverAutoprogrammingIdleBudgetSourceV0) AutoprogrammingIdleSelfImprovementBudgetV0(
	ctx context.Context,
) (orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0, bool, error) {
	if source.store == nil {
		return orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0{}, false, nil
	}
	state, err := source.store.LoadServerStateV0(ctx)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0{}, false, err
	}
	decision := state.IdleSelfImprovementBudget
	if !serverAutoprogrammingIdleBudgetDecisionPresentV0(decision) {
		return orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0{}, false, nil
	}
	return decision, true, nil
}

func serverAutoprogrammingIdleBudgetDecisionPresentV0(
	decision orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0,
) bool {
	return decision.Prepare ||
		decision.Degraded ||
		strings.TrimSpace(decision.Reason) != "" ||
		decision.RequestedGoals != 0 ||
		decision.AllowedGoals != 0 ||
		decision.MaxGoalsPerDay != 0 ||
		decision.GoalsUsedToday != 0 ||
		decision.GoalsRemainingToday != 0 ||
		decision.MaxContextBudgetBytesPerDay != 0 ||
		decision.ContextBudgetBytesUsedToday != 0 ||
		decision.ContextBudgetBytesRemainingToday != 0 ||
		decision.EstimatedNextContextBudgetBytes != 0 ||
		decision.PromptCacheCachedInputTokensToday != 0 ||
		decision.ContextBudgetEstimateEvidencePresent
}
