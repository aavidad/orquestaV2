package sqlite

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestRunningObservationRequeueDoesNotFabricateFailure(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	system.external.observationStatus = ports.AgentRunning
	submitted := system.submit(t, "request:observation-running-without-failure")

	processSQLiteObservationAction(t, system, application.ActionLaunchAgent)
	system.clock.Advance(time.Second)
	processSQLiteObservationAction(t, system, application.ActionObserveAgent)

	assertSQLiteObservationErrorCode(t, system, submitted, "")
}

func TestRunningObservationClearsPreviousTransientError(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	system.external.observationStatus = ports.AgentRunning
	submitted := system.submit(t, "request:observation-running-clears-transient")

	processSQLiteObservationAction(t, system, application.ActionLaunchAgent)
	claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:observation-transient", Token: "claim:observation-transient",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
	})
	if err != nil || !found || claim.Action.Kind != application.ActionObserveAgent {
		t.Fatalf("claim observation: claim=%+v found=%v err=%v", claim, found, err)
	}
	record, err := system.repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	retryAt := system.clock.Now().Add(time.Second)
	if err := system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: record.Executions[0], AvailableAt: retryAt,
		ErrorCode: "agent.observe_failed", OperationAt: system.clock.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	assertSQLiteObservationErrorCode(t, system, submitted, "agent.observe_failed")

	system.clock.Advance(time.Second)
	processSQLiteObservationAction(t, system, application.ActionObserveAgent)

	assertSQLiteObservationErrorCode(t, system, submitted, "")
}

func processSQLiteObservationAction(t *testing.T, system *sqliteV15System, want application.ActionKind) {
	t.Helper()
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:observation-requeue")
	if err != nil || !result.Processed || result.Action != want {
		t.Fatalf("process %s: result=%+v err=%v", want, result, err)
	}
}

func assertSQLiteObservationErrorCode(
	t *testing.T,
	system *sqliteV15System,
	submitted application.SubmitResult,
	want string,
) {
	t.Helper()
	var got string
	if err := system.repository.db.QueryRow(`
SELECT last_error_code
FROM outbox
WHERE goal_ref = ? AND kind = 'observe_agent'`,
		submitted.Record.Goal.Ref().String(),
	).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("observation last_error_code = %q, want %q", got, want)
	}
}
