package orquestacorereplanner

import "testing"

func TestCapReplanActionByRetryLimitV0EscalaTrasLimite(t *testing.T) {
	// Por debajo del limite: mantiene la accion resolutiva.
	for _, action := range []ReplanRecommendedActionV0{
		ReplanActionRetryTaskV0, ReplanActionSplitTaskV0,
		ReplanActionReplaceAgentV0, ReplanActionEscalateCapacityV0,
	} {
		if got := CapReplanActionByRetryLimitV0(action, 0, 3); got != action {
			t.Fatalf("count=0 action=%s got=%s want sin cambios", action, got)
		}
		if got := CapReplanActionByRetryLimitV0(action, 2, 3); got != action {
			t.Fatalf("count=2 action=%s got=%s want sin cambios", action, got)
		}
		// Al alcanzar el limite: escala a ask_director.
		if got := CapReplanActionByRetryLimitV0(action, 3, 3); got != ReplanActionAskDirectorV0 {
			t.Fatalf("count=3 action=%s got=%s want ask_director", action, got)
		}
		if got := CapReplanActionByRetryLimitV0(action, 10, 3); got != ReplanActionAskDirectorV0 {
			t.Fatalf("count=10 action=%s got=%s want ask_director", action, got)
		}
	}
}

func TestCapReplanActionByRetryLimitV0NoTocaTerminalesV0(t *testing.T) {
	if got := CapReplanActionByRetryLimitV0(ReplanActionAskDirectorV0, 99, 3); got != ReplanActionAskDirectorV0 {
		t.Fatalf("ask_director no debe cambiar: got=%s", got)
	}
	if got := CapReplanActionByRetryLimitV0(ReplanActionAbortTaskV0, 99, 3); got != ReplanActionAbortTaskV0 {
		t.Fatalf("abort_task no debe cambiar: got=%s", got)
	}
}

func TestCapReplanActionByRetryLimitV0LimiteDefecto(t *testing.T) {
	// limit<=0 usa el default (3).
	if got := CapReplanActionByRetryLimitV0(ReplanActionRetryTaskV0, DefaultReplanRetryLimitV0, 0); got != ReplanActionAskDirectorV0 {
		t.Fatalf("con limite por defecto debe escalar: got=%s", got)
	}
	if !ReplanRetryLimitExceededV0(DefaultReplanRetryLimitV0, 0) {
		t.Fatalf("ReplanRetryLimitExceededV0 debe ser true en el default")
	}
}
