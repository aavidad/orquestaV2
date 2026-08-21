package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestV37StopNonApplicationSurvivesRestartAndAuthorizesOneNewAttempt(t *testing.T) {
	system, claim, execution, attempt := seedV37ClaimedStop(t)
	outcome := v37StopOutcome(attempt, system.clock.Now())
	if err := system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: execution, AvailableAt: system.clock.Now(),
		ErrorCode: "agent.stop_definitely_not_applied", OperationAt: system.clock.Now(),
		EffectAttemptOutcome: &outcome,
	}); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := openSQLiteV15Repository(t, system.path, system.clock.Now)
	record, err := restarted.GetGoal(context.Background(), claim.Action.GoalRef)
	if err != nil || len(record.EffectAttemptOutcomes) != 1 || record.EffectAttemptOutcomes[0] != outcome {
		t.Fatalf("restart outcome=%+v err=%v", record.EffectAttemptOutcomes, err)
	}
	next, found, err := restarted.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v37-retry", Token: "claim:v37-retry", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy, CapacityCandidates: system.capacidad,
	})
	if err != nil || !found || next.Action.Ref != claim.Action.Ref || next.Fence <= claim.Fence {
		t.Fatalf("retry claim=%+v found=%v err=%v", next, found, err)
	}
	second := sqliteV15Attempt(next, system.clock.Now())
	if _, created, err := restarted.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: next, Attempt: second, OperationAt: system.clock.Now(),
	}); err != nil || !created {
		t.Fatalf("second attempt created=%v err=%v", created, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), restarted.db); err != nil {
		t.Fatalf("restart recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestV37RecoveryRejectsCorruptStopNonApplicationEvidence(t *testing.T) {
	system, claim, execution, attempt := seedV37ClaimedStop(t)
	outcome := v37StopOutcome(attempt, system.clock.Now())
	if err := system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: execution, AvailableAt: system.clock.Now(),
		ErrorCode: "agent.stop_definitely_not_applied", OperationAt: system.clock.Now(),
		EffectAttemptOutcome: &outcome,
	}); err != nil {
		t.Fatal(err)
	}
	rewriteRecoveryTrigger(t, system.repository.db, "effect_non_application_evidence_immutable_update", func() {
		mustV10Exec(t, system.repository.db, `UPDATE effect_non_application_evidence
SET observed_at=observed_at-1 WHERE attempt_ref=?`, attempt.Ref)
	})
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_v37_effect_non_application_invalid") {
		t.Fatalf("corrupt V37 evidence accepted: %v", err)
	}
}

func TestV37RecoveryRejectsRetryPastUnknownSecondStopAttempt(t *testing.T) {
	system, claim, execution, attempt := seedV37ClaimedStop(t)
	firstOutcome := v37StopOutcome(attempt, system.clock.Now())
	if err := system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: execution, AvailableAt: system.clock.Now(),
		ErrorCode: "agent.stop_definitely_not_applied", OperationAt: system.clock.Now(),
		EffectAttemptOutcome: &firstOutcome,
	}); err != nil {
		t.Fatal(err)
	}
	secondClaim := claimV37StopRetry(t, system, "claim:v37-second")
	second := sqliteV15Attempt(secondClaim, system.clock.Now())
	if _, created, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: secondClaim, Attempt: second, OperationAt: system.clock.Now(),
	}); err != nil || !created {
		t.Fatalf("second attempt created=%v err=%v", created, err)
	}
	// Simulate a corrupt writer that released an unknown-applied second attempt,
	// then prove recovery will not authorize a third physical invocation.
	mustV10Exec(t, system.repository.db, `UPDATE outbox SET claim_token=NULL,claimed_by=NULL,claimed_until=NULL,
available_at=?,last_error_code='agent.stop_unknown_applied' WHERE ref=?`, requiredTime(system.clock.Now()), claim.Action.Ref)
	thirdClaim := claimV37StopRetry(t, system, "claim:v37-third")
	third := sqliteV15Attempt(thirdClaim, system.clock.Now())
	if _, created, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: thirdClaim, Attempt: third, OperationAt: system.clock.Now(),
	}); err != nil || !created {
		t.Fatalf("third attempt seed created=%v err=%v", created, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_v17_unknown_applied_repeated") {
		t.Fatalf("retry past unknown-applied second attempt accepted: %v", err)
	}
}

func TestV37SecondStopAttemptUnknownAppliedCannotDispatchThird(t *testing.T) {
	system, claim, execution, attempt := seedV37ClaimedStop(t)
	firstOutcome := v37StopOutcome(attempt, system.clock.Now())
	if err := system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: execution, AvailableAt: system.clock.Now(),
		ErrorCode: "agent.stop_definitely_not_applied", OperationAt: system.clock.Now(),
		EffectAttemptOutcome: &firstOutcome,
	}); err != nil {
		t.Fatal(err)
	}
	controller := &v37UnknownStopController{sqliteV15External: system.external}
	system.orchestrator = newSQLiteV37StopOrchestrator(t, system, controller)
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v37-second-unknown")
	if err == nil || err.Error() != "application.effect_unknown_applied" ||
		!result.Processed || result.Action != application.ActionStopAgent || controller.calls != 1 {
		t.Fatalf("second result=%+v calls=%d err=%v", result, controller.calls, err)
	}
	result, err = system.orchestrator.ProcessNext(context.Background(), "worker:v37-third-forbidden")
	if err != nil || (result.Processed && result.Action == application.ActionStopAgent) || controller.calls != 1 {
		t.Fatalf("third result=%+v calls=%d err=%v", result, controller.calls, err)
	}
	var stopAttempts int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts WHERE action_ref=?`,
		claim.Action.Ref).Scan(&stopAttempts); err != nil || stopAttempts != 2 {
		t.Fatalf("stop attempts=%d err=%v", stopAttempts, err)
	}
}

type v37UnknownStopController struct {
	*sqliteV15External
	calls int
}

func (controller *v37UnknownStopController) Stop(
	context.Context, ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	controller.calls++
	return ports.AgentStopReceipt{}, errors.New("stop acknowledgement lost")
}

func newSQLiteV37StopOrchestrator(
	t *testing.T, system *sqliteV15System, controller application.AgentController,
) *application.Orchestrator {
	t.Helper()
	_, sources := prepararCapacidadSQLiteV15(t, system.repository, system.clock, 1_000)
	orchestrator, err := application.New(application.Dependencies{
		State: system.repository, Access: system.repository, Launcher: system.external, Observer: system.external,
		Controller: controller, Artifacts: system.external, Clock: system.clock, IDs: system.ids,
		MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6, ClaimLease: time.Minute,
		DirectorLeaseDuration: 30 * time.Second, EffectApprovalTTL: system.policy.EffectApprovalTTL,
		BudgetPolicy: system.policy, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: sqliteTestCapabilities(), CapacitySources: sources, CapacityObservationWait: time.Second,
	})
	sqliteTestNoError(t, err)
	return orchestrator
}

func claimV37StopRetry(t *testing.T, system *sqliteV15System, token string) application.ActionClaim {
	t.Helper()
	claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:" + token, Token: token, LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy, CapacityCandidates: system.capacidad,
	})
	if err != nil || !found || claim.Action.Kind != application.ActionStopAgent {
		t.Fatalf("retry claim=%+v found=%v err=%v", claim, found, err)
	}
	return claim
}
