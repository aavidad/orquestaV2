package application

import (
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestAgentCapacityStateValidatesBindingsAmountsAndTransitions(t *testing.T) {
	project, _ := goal.NewProjectRef("project:capacity")
	goalRef, _ := goal.NewGoalRef("goal:capacity")
	workRef, _ := goal.NewWorkItemRef("work:capacity")
	executionRef, _ := goal.NewExecutionRef("execution:capacity")
	at := time.Date(2026, 7, 30, 23, 0, 0, 0, time.UTC)
	submission := AgentCapacityObservationSubmission{Ref: "capacity-observation:one", IdempotencyKey: "observation:one", Observation: validAgentCapacityObservation(t)}
	observation, firstErr := MaterializeAgentCapacityObservation(submission, 0)
	successor, successorErr := MaterializeAgentCapacityObservation(AgentCapacityObservationSubmission{Ref: "capacity-observation:two", IdempotencyKey: "observation:two", Observation: submission.Observation}, observation.Revision)
	if firstErr != nil || successorErr != nil || observation.ExpectedRevision != 0 ||
		observation.Revision != 1 || successor.ExpectedRevision != 1 || successor.Revision != 2 {
		t.Fatalf("revisiones inicial=%+v/%v sucesiva=%+v/%v", observation, firstErr, successor, successorErr)
	}
	if _, err := MaterializeAgentCapacityObservation(submission, ^uint64(0)); err == nil {
		t.Fatal("se aceptó desbordar la revisión")
	}
	reservation := AgentCapacityReservation{Ref: "capacity-reservation:one", ObservationRef: observation.Ref, ObservationRevision: observation.Revision, ActionRef: "action:capacity", EffectIntentRef: "effect-intent:capacity", IdempotencyKey: "reservation:capacity", ProjectRef: project, GoalRef: goalRef, WorkItemRef: workRef, ExecutionRef: executionRef, PlanGeneration: 1, WorkItemGeneration: 1, Revision: 1, Fence: 1, Allocation: AgentCapacityAllocation{Slots: 1}, State: AgentCapacityReserved, ReservedAt: at, UpdatedAt: at}
	cases := []struct {
		from, outcome    AgentCapacityReservationState
		cause            AgentCapacityTransitionCause
		attempt, receipt string
		settled          time.Time
		slots            int64
		valid            bool
	}{
		{AgentCapacityReserved, AgentCapacityConsumed, AgentCapacityCauseEffectReceipt, "effect-attempt:one", "effect-receipt:one", time.Time{}, 1, true}, {AgentCapacityReserved, AgentCapacityQuarantined, AgentCapacityCauseUnknownApplied, "", "", time.Time{}, 1, true}, {AgentCapacityReserved, AgentCapacityReleased, AgentCapacityCauseDefinitelyNotApplied, "", "", time.Time{}, 1, true}, {AgentCapacityQuarantined, AgentCapacityConsumed, AgentCapacityCauseEffectReceipt, "effect-attempt:one", "effect-receipt:one", time.Time{}, 1, true}, {AgentCapacityQuarantined, AgentCapacityConsumed, AgentCapacityCauseReconciliation, "effect-attempt:one", "effect-receipt:one", time.Time{}, 1, true}, {AgentCapacityQuarantined, AgentCapacityReleased, AgentCapacityCauseReconciliation, "", "", time.Time{}, 1, true}, {AgentCapacityConsumed, AgentCapacityReleased, AgentCapacityCauseExecutionTerminal, "", "", time.Time{}, 1, true}, {AgentCapacityConsumed, AgentCapacityReleased, AgentCapacityCauseReconciliation, "", "", time.Time{}, 1, true},
		{AgentCapacityReserved, AgentCapacityConsumed, AgentCapacityCauseEffectReceipt, "", "", time.Time{}, 1, false}, {AgentCapacityConsumed, AgentCapacityQuarantined, AgentCapacityCauseUnknownApplied, "", "", time.Time{}, 1, false}, {AgentCapacityReserved, AgentCapacityQuarantined, AgentCapacityCauseUnknownApplied, "effect-attempt:one", "effect-receipt:one", time.Time{}, 1, false}, {AgentCapacityReservationState("unknown"), AgentCapacityReleased, AgentCapacityCauseExecutionTerminal, "", "", time.Time{}, 1, false}, {AgentCapacityConsumed, AgentCapacityReleased, AgentCapacityCauseExecutionTerminal, "", "", at.Add(3 * time.Second), 1, false}, {AgentCapacityReserved, AgentCapacityConsumed, AgentCapacityCauseEffectReceipt, "effect-attempt:one", "effect-receipt:one", time.Time{}, 0, false},
	}
	for _, test := range cases {
		current := reservation
		if test.from != AgentCapacityReserved {
			current.State, current.Revision = test.from, 2
			current.LastTransitionRef, current.LastCauseRef = "capacity-transition:prior", "cause:prior"
			current.UpdatedAt = at.Add(time.Second)
		}
		current.SettledAt = test.settled
		current.Allocation.Slots = test.slots
		next := AgentCapacityTransition{Ref: "capacity-transition:next", ReservationRef: current.Ref, ProjectRef: project, Fence: 1, ExpectedRevision: current.Revision, Revision: current.Revision + 1, Outcome: test.outcome, Cause: test.cause, CauseRef: "cause:next", IdempotencyKey: "transition:next", RecordedAt: at.Add(2 * time.Second)}
		next.EffectAttemptRef, next.EffectReceiptRef = test.attempt, test.receipt
		if err := ValidateAgentCapacityTransition(current, next); (err == nil) != test.valid {
			t.Fatalf("%s→%s/%s válida=%v error=%v", test.from, test.outcome, test.cause, test.valid, err)
		}
	}
}
