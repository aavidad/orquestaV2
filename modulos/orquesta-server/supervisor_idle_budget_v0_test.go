package orquestaserver

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0AutomejoraIdleAplazaPorPresupuestoAgotadoV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 20, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-no-debe-lanzarse"}},
		selfStarted:  make(chan struct{}, 1),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 1,
		IdleSelfImprovementTargetQueue: 1,
		IdleSelfImprovementBudget: orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetConfigV0{
			MaxGoalsPerDay: 1,
		},
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)
	runtime.tracker.updateV0(func(state *StateV0) {
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementRuns = 1
	})

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.planCalls != 0 || supervisor.selfCalls != 0 {
		t.Fatalf("presupuesto agotado no debe planificar: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0 ||
		store.last.IdleSelfImprovementBudget.Reason != orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0 ||
		store.last.IdleSelfImprovementBudget.AllowedGoals != 0 ||
		!strings.Contains(store.last.IdleSelfImprovementReason, orquestaautoprogramming.AutoprogrammingIdleBudgetDeferredV0) {
		t.Fatalf("state=%+v budget=%+v", store.last, store.last.IdleSelfImprovementBudget)
	}
}

func TestRuntimeV0AutomejoraIdleDegradaLotePorPresupuestoContextoV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 20, 5, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason:      orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0,
			TotalExecutions: 1,
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-budget-001",
		}, {
			RequestRef: "request-ref-budget-002",
		}, {
			RequestRef: "request-ref-budget-003",
		}},
		selfStarted: make(chan struct{}, 3),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 4,
		IdleSelfImprovementTargetQueue: 4,
		IdleSelfImprovementBudget: orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetConfigV0{
			MaxContextBudgetBytesPerDay: 250,
		},
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.updateV0(func(state *StateV0) {
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementGoalResult = &orquestagoal.GoalWorkResultV0{
			ContextBudget: orquestagoal.GoalContextBudgetV0{
				ContextBudgetTotalBytes: 100,
			},
			EvidenceRefs: []string{"evidence-ref-codex-goal-cached-input-tokens-512"},
		}
	})

	runtime.runSupervisorTickV0(context.Background())

	select {
	case <-supervisor.selfStarted:
	case <-time.After(time.Second):
		t.Fatalf("automejora degradada no arranco")
	}
	if supervisor.planCalls != 1 ||
		supervisor.lastPlanRequest.MaxRequests != 1 ||
		supervisor.selfCalls != 1 ||
		store.last.IdleSelfImprovementBudget.Reason != orquestaautoprogramming.AutoprogrammingIdleBudgetDegradedV0 ||
		!store.last.IdleSelfImprovementBudget.Degraded ||
		store.last.IdleSelfImprovementBudget.EstimatedNextContextBudgetBytes != 100 ||
		store.last.IdleSelfImprovementBudget.PromptCacheCachedInputTokensToday != 512 {
		t.Fatalf("plan=%+v self_calls=%d budget=%+v", supervisor.lastPlanRequest, supervisor.selfCalls, store.last.IdleSelfImprovementBudget)
	}
}
