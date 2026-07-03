package orquestaserver

import (
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementBudgetDecisionV0(
	decision orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementBudget = decision
	})
}
