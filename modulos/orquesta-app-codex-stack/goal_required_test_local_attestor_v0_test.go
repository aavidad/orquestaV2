package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestLocalGoalRequiredTestAttestorV0EsInyectadoYNoHaceFallback(t *testing.T) {
	if _, err := (LocalGoalRequiredTestAttestorV0{}).AttestGoalRequiredTestsV0(context.Background(), orquestagoal.GoalRequiredTestAttestationRequestV0{}); err == nil {
		t.Fatalf("esperaba executor local explicito")
	}
	executor := &localGoalRequiredTestAttestationExecutorForTestV0{}
	attestor := LocalGoalRequiredTestAttestorV0{Executor: executor}
	request := orquestagoal.GoalRequiredTestAttestationRequestV0{GoalRef: "goal-ref-local-attestor"}
	if _, err := attestor.AttestGoalRequiredTestsV0(context.Background(), request); err != nil || executor.calls != 1 || executor.request.GoalRef != request.GoalRef {
		t.Fatalf("calls=%d request=%+v err=%v", executor.calls, executor.request, err)
	}
}

type localGoalRequiredTestAttestationExecutorForTestV0 struct {
	calls   int
	request orquestagoal.GoalRequiredTestAttestationRequestV0
}

func (executor *localGoalRequiredTestAttestationExecutorForTestV0) ExecuteLocalGoalRequiredTestAttestationV0(_ context.Context, request orquestagoal.GoalRequiredTestAttestationRequestV0) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	executor.calls++
	executor.request = request
	return nil, nil
}
