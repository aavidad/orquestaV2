package orquestaappcodexstack

import (
	"context"
	"errors"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

// LocalGoalRequiredTestAttestationExecutorPortV0 is the explicit local edge.
// Its implementation may use an isolated test environment, but this package
// never chooses a shell, command, provider or credential on its own.
type LocalGoalRequiredTestAttestationExecutorPortV0 interface {
	ExecuteLocalGoalRequiredTestAttestationV0(context.Context, orquestagoal.GoalRequiredTestAttestationRequestV0) ([]orquestagoal.GoalRequiredTestAttestationV0, error)
}

// LocalGoalRequiredTestAttestorV0 adapts an operator-injected local executor
// to the neutral attestor port. It is opt-in: a nil Executor never falls back
// to the implementer or to a runtime provider.
type LocalGoalRequiredTestAttestorV0 struct {
	Executor LocalGoalRequiredTestAttestationExecutorPortV0
}

func (attestor LocalGoalRequiredTestAttestorV0) AttestGoalRequiredTestsV0(
	ctx context.Context,
	request orquestagoal.GoalRequiredTestAttestationRequestV0,
) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	if attestor.Executor == nil {
		return nil, errors.New("local_goal_required_test_attestation_executor_unavailable")
	}
	return attestor.Executor.ExecuteLocalGoalRequiredTestAttestationV0(ctx, request)
}
