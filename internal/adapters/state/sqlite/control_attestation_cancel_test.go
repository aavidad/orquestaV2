package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSQLiteCancelAttestationBeforeClaimRetiresOnceAndRecovers(t *testing.T) {
	for _, target := range []application.ControlTarget{
		application.ControlTargetWorkItem,
		application.ControlTargetGoal,
	} {
		t.Run(string(target), func(t *testing.T) {
			attestor := &sqliteTestAttestor{}
			system, goalRef := seedSQLiteV17Committed(t, attestor)
			before, err := system.repository.GetGoal(context.Background(), goalRef)
			sqliteTestNoError(t, err)
			execution := before.Executions[0]
			request := bug453ControlRequest(
				before, application.ControlCancel, target,
				"request:cancel-attestation-before-claim:"+string(target),
			)

			result, err := system.orchestrator.Control(context.Background(), system.access, request)
			if err != nil || !result.Created || result.Control.Status != application.ControlConfirmed {
				t.Fatalf("cancel before claim: result=%+v err=%v cause=%v",
					result, err, errors.Unwrap(err))
			}
			after, err := system.repository.GetGoal(context.Background(), goalRef)
			sqliteTestNoError(t, err)
			assertBUG453CanceledLifecycle(t, after, target)
			assertBUG453ActionTerminal(
				t, system.repository.db, execution.Ref.String(),
				"completed", "application.action_retired", false, 0,
			)
			if calls, effects := attestor.counts(); calls != 0 || effects != 0 {
				t.Fatalf("attestor called before claim cancellation: calls/effects=%d/%d", calls, effects)
			}

			replayed, err := system.orchestrator.Control(context.Background(), system.access, request)
			if err != nil || replayed.Created || replayed.Control.Ref != result.Control.Ref {
				t.Fatalf("cancel replay: result=%+v err=%v", replayed, err)
			}
			afterReplay, err := system.repository.GetGoal(context.Background(), goalRef)
			sqliteTestNoError(t, err)
			if !reflect.DeepEqual(afterReplay, after) {
				t.Fatal("identical cancel replay mutated durable state")
			}
			assertBUG453RecoveryAndRestart(t, system, goalRef)
		})
	}
}

func TestSQLiteCancelQuarantinedAttestationPreservesPauseReceiptAndRecovery(t *testing.T) {
	attestor := &sqliteTestAttestor{failures: 1}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processed, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:cancel-quarantined-attestation",
	)
	if err == nil || err.Error() != "application.effect_unknown_applied" ||
		!processed.Processed || processed.Action != application.ActionAttestTest {
		t.Fatalf("seed quarantined attestation: result=%+v err=%v", processed, err)
	}

	beforePause, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	execution := beforePause.Executions[0]
	quarantineAt := bug453ActionTerminalAt(t, system.repository.db, execution.Ref.String())
	assertBUG453ActionTerminal(
		t, system.repository.db, execution.Ref.String(),
		"quarantined", "application.effect_unknown_applied", true, 1,
	)

	pauseRequest := bug453ControlRequest(
		beforePause, application.ControlPause, application.ControlTargetWorkItem,
		"request:pause-quarantined-attestation",
	)
	paused, err := system.orchestrator.Control(context.Background(), system.access, pauseRequest)
	if err != nil || !paused.Created || paused.Control.Status != application.ControlConfirmed {
		t.Fatalf("pause quarantined attestation: result=%+v err=%v", paused, err)
	}
	pausedRecord, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	item := pausedRecord.Goal.WorkItems()[0]
	effectivePause, found := pausedRecord.Goal.EffectivePause(item.Ref())
	if !found || !effectivePause {
		t.Fatalf("pause was not preserved: found=%v paused=%v", found, effectivePause)
	}
	pauseReplay, err := system.orchestrator.Control(context.Background(), system.access, pauseRequest)
	if err != nil || pauseReplay.Created || pauseReplay.Control.Ref != paused.Control.Ref {
		t.Fatalf("pause replay: result=%+v err=%v", pauseReplay, err)
	}
	if got := bug453ActionTerminalAt(t, system.repository.db, execution.Ref.String()); got != quarantineAt {
		t.Fatalf("pause rewrote quarantine timestamp: got=%d want=%d", got, quarantineAt)
	}

	cancelRequest := bug453ControlRequest(
		pausedRecord, application.ControlCancel, application.ControlTargetWorkItem,
		"request:cancel-quarantined-attestation",
	)
	canceled, err := system.orchestrator.Control(context.Background(), system.access, cancelRequest)
	if err != nil || !canceled.Created || canceled.Control.Status != application.ControlConfirmed {
		t.Fatalf("cancel quarantined attestation: result=%+v err=%v cause=%v",
			canceled, err, errors.Unwrap(err))
	}
	after, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	assertBUG453CanceledLifecycle(t, after, application.ControlTargetWorkItem)
	assertBUG453ActionTerminal(
		t, system.repository.db, execution.Ref.String(),
		"quarantined", "application.effect_unknown_applied", true, 1,
	)
	if got := bug453ActionTerminalAt(t, system.repository.db, execution.Ref.String()); got != quarantineAt {
		t.Fatalf("cancel rewrote existing quarantine timestamp: got=%d want=%d", got, quarantineAt)
	}
	beforeReplay := after
	replayed, err := system.orchestrator.Control(context.Background(), system.access, cancelRequest)
	if err != nil || replayed.Created || replayed.Control.Ref != canceled.Control.Ref {
		t.Fatalf("quarantined cancel replay: result=%+v err=%v", replayed, err)
	}
	afterReplay, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	if !reflect.DeepEqual(afterReplay, beforeReplay) {
		t.Fatal("quarantined cancel replay mutated durable state")
	}
	assertBUG453RecoveryAndRestart(t, system, goalRef)
}

func TestSQLiteCancelRacingStartedAttestationQuarantinesRatherThanRetires(t *testing.T) {
	blocking := newBUG453BlockingAttestor()
	system, goalRef := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
	system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, blocking)

	type processOutcome struct {
		result application.ProcessResult
		err    error
	}
	processDone := make(chan processOutcome, 1)
	go func() {
		result, err := system.orchestrator.ProcessNext(
			context.Background(), "worker:cancel-racing-attestation",
		)
		processDone <- processOutcome{result: result, err: err}
	}()
	blocking.waitEntered(t)
	defer blocking.release()

	before, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	execution := before.Executions[0]
	request := bug453ControlRequest(
		before, application.ControlCancel, application.ControlTargetWorkItem,
		"request:cancel-racing-attestation",
	)
	canceled, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || !canceled.Created || canceled.Control.Status != application.ControlConfirmed {
		t.Fatalf("cancel racing attestation: result=%+v err=%v cause=%v",
			canceled, err, errors.Unwrap(err))
	}
	afterCancel, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	assertBUG453CanceledLifecycle(t, afterCancel, application.ControlTargetWorkItem)
	assertBUG453ActionTerminal(
		t, system.repository.db, execution.Ref.String(),
		"quarantined", "application.effect_unknown_applied", true, 1,
	)
	assertBUG453NoTestCompletion(t, system.repository.db)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected cancel winner before late completion: %s", sqliteTestErrorChain(err))
	}

	blocking.release()
	var late processOutcome
	select {
	case late = <-processDone:
	case <-time.After(5 * time.Second):
		t.Fatal("late attestation completion did not settle")
	}
	if !late.result.Processed || late.result.Action != application.ActionAttestTest ||
		late.err == nil || !application.IsStateError(late.err, application.StateConflict) {
		t.Fatalf("late completion crossed cancellation CAS: result=%+v err=%v cause=%v",
			late.result, late.err, errors.Unwrap(late.err))
	}
	if calls, effects := blocking.delegate.counts(); calls != 1 || effects != 1 {
		t.Fatalf("physical attestation calls/effects=%d/%d want=1/1", calls, effects)
	}
	assertBUG453ActionTerminal(
		t, system.repository.db, execution.Ref.String(),
		"quarantined", "application.effect_unknown_applied", true, 1,
	)
	assertBUG453NoTestCompletion(t, system.repository.db)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected cancel winner after late completion: %s", sqliteTestErrorChain(err))
	}
}

func TestSQLiteAttestationCompletionThenCancellationPreservesTerminalEvidence(t *testing.T) {
	system, goalRef := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
	before, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	execution := before.Executions[0]
	cancelRequest := bug453ControlRequest(
		before, application.ControlCancel, application.ControlTargetWorkItem,
		"request:cancel-after-attestation-completion",
	)
	processSQLiteV16Actions(t, system, application.ActionAttestTest)

	result, err := system.orchestrator.Control(context.Background(), system.access, cancelRequest)
	if err != nil || !result.Created || result.Control.Status != application.ControlConfirmed {
		t.Fatalf("cancel after completed attestation: result=%+v err=%v cause=%v",
			result, err, errors.Unwrap(err))
	}
	assertBUG453ActionTerminal(
		t, system.repository.db, execution.Ref.String(),
		"completed", "", false, 1,
	)
	var attestations, effectReceipts int
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM attestations WHERE kind='required_tests'`,
	).Scan(&attestations))
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM effect_receipts WHERE action_ref=?`,
		"action:attest-test:"+execution.Ref.String(),
	).Scan(&effectReceipts))
	if attestations != 1 || effectReceipts != 1 {
		t.Fatalf("completed attestation facts=%d receipts=%d", attestations, effectReceipts)
	}
	after, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	item := after.Goal.WorkItems()[0]
	canceledExecutions := 0
	for _, current := range after.Executions {
		if current.State == application.ExecutionCanceled {
			canceledExecutions++
		}
	}
	if item.State() != goal.WorkItemStateCanceled || after.Goal.State() != goal.GoalStateFailed ||
		canceledExecutions != len(after.Executions) {
		t.Fatalf("post-attestation cancel item=%s goal=%s executions=%+v",
			item.State(), after.Goal.State(), after.Executions)
	}
	replayed, err := system.orchestrator.Control(context.Background(), system.access, cancelRequest)
	if err != nil || replayed.Created || replayed.Control.Ref != result.Control.Ref {
		t.Fatalf("post-attestation cancel replay: result=%+v err=%v", replayed, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected attestation completion winner: %s", sqliteTestErrorChain(err))
	}
}

func TestRetiredAttestationRejectsDivergentAttemptWithoutPartialWrite(t *testing.T) {
	blocking := newBUG453BlockingAttestor()
	system, _ := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
	system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, blocking)
	processDone := make(chan error, 1)
	go func() {
		_, err := system.orchestrator.ProcessNext(
			context.Background(), "worker:divergent-attestation-attempt",
		)
		processDone <- err
	}()
	blocking.waitEntered(t)
	defer blocking.release()

	var actionRef string
	var originalFence int64
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT action.ref,attempt.action_fence
FROM outbox action JOIN effect_attempts attempt ON attempt.action_ref=action.ref
WHERE action.kind='attest_test'`).Scan(&actionRef, &originalFence))
	rewriteRecoveryTrigger(t, system.repository.db, "effect_attempts_immutable_update", func() {
		mustV10Exec(t, system.repository.db, `
UPDATE effect_attempts SET action_fence=action_fence+1 WHERE action_ref=?`, actionRef)
	})

	transaction, err := system.repository.writer.BeginTx(context.Background(), nil)
	sqliteTestNoError(t, err)
	err = consumeRetiredAction(
		context.Background(), transaction, actionRef, "control:divergent-attempt", system.clock.Now(),
	)
	_ = transaction.Rollback()
	if err == nil || !application.IsStateError(err, application.StateConflict) ||
		!recoveryErrorContains(err, "sqlite.action_retirement_attestation_attempt_ambiguous") {
		t.Fatalf("divergent attempt did not fail closed: err=%v cause=%v", err, errors.Unwrap(err))
	}
	var active, receipts int
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT completed_at IS NULL AND quarantined_at IS NULL FROM outbox WHERE ref=?`,
		actionRef,
	).Scan(&active))
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT COUNT(*) FROM action_consumption_receipts WHERE action_ref=?`,
		actionRef,
	).Scan(&receipts))
	if active != 1 || receipts != 0 {
		t.Fatalf("divergent attempt caused partial write: active=%d receipts=%d", active, receipts)
	}

	rewriteRecoveryTrigger(t, system.repository.db, "effect_attempts_immutable_update", func() {
		mustV10Exec(t, system.repository.db, `
UPDATE effect_attempts SET action_fence=? WHERE action_ref=?`, originalFence, actionRef)
	})
	blocking.release()
	select {
	case err := <-processDone:
		if err != nil {
			t.Fatalf("restored attempt did not complete: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("restored attempt completion timed out")
	}
}

func bug453ControlRequest(
	record application.GoalRecord,
	operation application.ControlOperation,
	target application.ControlTarget,
	requestRef string,
) application.ControlRequest {
	request := application.ControlRequest{
		RequestRef: requestRef, Operation: operation, Target: target,
		GoalRef: record.Goal.Ref(), ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedPlanGeneration:    record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(),
		ExpectedSpecHash:          record.Goal.SpecHash(), Reason: "exercise causal attestation control",
	}
	if target == application.ControlTargetWorkItem {
		item := record.Goal.WorkItems()[0]
		request.WorkItemRef = item.Ref()
		request.ExpectedWorkItemRevision = item.Revision()
	}
	return request
}

func assertBUG453CanceledLifecycle(
	t *testing.T,
	record application.GoalRecord,
	target application.ControlTarget,
) {
	t.Helper()
	item := record.Goal.WorkItems()[0]
	if item.State() != goal.WorkItemStateCanceled ||
		len(record.Executions) != 1 ||
		record.Executions[0].State != application.ExecutionCanceled {
		t.Fatalf("canceled lifecycle item=%s executions=%+v", item.State(), record.Executions)
	}
	wantGoal := goal.GoalStateFailed
	if target == application.ControlTargetGoal {
		wantGoal = goal.GoalStateCanceled
	}
	if record.Goal.State() != wantGoal {
		t.Fatalf("canceled goal state=%s want=%s", record.Goal.State(), wantGoal)
	}
}

func assertBUG453ActionTerminal(
	t *testing.T,
	database *sql.DB,
	executionRef string,
	outcome string,
	code string,
	quarantined bool,
	attempts int,
) {
	t.Helper()
	var completed, quarantine, receiptCount, attemptCount int
	var gotOutcome, gotCode string
	err := database.QueryRow(`
SELECT action.completed_at IS NOT NULL,action.quarantined_at IS NOT NULL,
       receipt.outcome,receipt.error_code,
       (SELECT COUNT(*) FROM action_consumption_receipts all_receipts WHERE all_receipts.action_ref=action.ref),
       (SELECT COUNT(*) FROM effect_attempts attempt WHERE attempt.action_ref=action.ref)
FROM outbox action JOIN action_consumption_receipts receipt ON receipt.action_ref=action.ref
WHERE action.ref=?`,
		"action:attest-test:"+executionRef,
	).Scan(&completed, &quarantine, &gotOutcome, &gotCode, &receiptCount, &attemptCount)
	sqliteTestNoError(t, err)
	wantQuarantine := 0
	if quarantined {
		wantQuarantine = 1
	}
	if completed != 1 || quarantine != wantQuarantine || gotOutcome != outcome ||
		gotCode != code || receiptCount != 1 || attemptCount != attempts {
		t.Fatalf("attestation terminal completed/quarantine=%d/%d outcome/code=%s/%s receipts/attempts=%d/%d",
			completed, quarantine, gotOutcome, gotCode, receiptCount, attemptCount)
	}
}

func bug453ActionTerminalAt(t *testing.T, database *sql.DB, executionRef string) int64 {
	t.Helper()
	var at int64
	sqliteTestNoError(t, database.QueryRow(`
SELECT quarantined_at FROM outbox WHERE ref=?`,
		"action:attest-test:"+executionRef,
	).Scan(&at))
	return at
}

func assertBUG453NoTestCompletion(t *testing.T, database *sql.DB) {
	t.Helper()
	var attestations, effectReceipts, consumptions int
	sqliteTestNoError(t, database.QueryRow(
		`SELECT COUNT(*) FROM attestations WHERE kind='required_tests'`,
	).Scan(&attestations))
	sqliteTestNoError(t, database.QueryRow(
		`SELECT COUNT(*) FROM effect_receipts WHERE status IN ('attested_passed','attested_failed')`,
	).Scan(&effectReceipts))
	sqliteTestNoError(t, database.QueryRow(
		`SELECT COUNT(*) FROM action_consumption_receipts WHERE kind='attest_test'`,
	).Scan(&consumptions))
	if attestations != 0 || effectReceipts != 0 || consumptions != 1 {
		t.Fatalf("unexpected late test completion attestations/effect-receipts/consumptions=%d/%d/%d",
			attestations, effectReceipts, consumptions)
	}
}

func assertBUG453RecoveryAndRestart(
	t *testing.T,
	system *sqliteV15System,
	goalRef goal.GoalRef,
) {
	t.Helper()
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected canceled attestation: %s", sqliteTestErrorChain(err))
	}
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	if _, err := system.repository.GetGoal(context.Background(), goalRef); err != nil {
		t.Fatalf("restart rejected canceled attestation: %s", sqliteTestErrorChain(err))
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("restart recovery rejected canceled attestation: %s", sqliteTestErrorChain(err))
	}
}

type bug453BlockingAttestor struct {
	delegate    *sqliteTestAttestor
	entered     chan struct{}
	releaseCh   chan struct{}
	enterOnce   sync.Once
	releaseOnce sync.Once
}

func newBUG453BlockingAttestor() *bug453BlockingAttestor {
	return &bug453BlockingAttestor{
		delegate:  &sqliteTestAttestor{},
		entered:   make(chan struct{}),
		releaseCh: make(chan struct{}),
	}
}

func (attestor *bug453BlockingAttestor) Attest(
	ctx context.Context,
	run ports.TestAttestationRun,
) (ports.TestAttestationResult, error) {
	attestor.enterOnce.Do(func() { close(attestor.entered) })
	select {
	case <-attestor.releaseCh:
	case <-ctx.Done():
		return ports.TestAttestationResult{}, ctx.Err()
	}
	return attestor.delegate.Attest(ctx, run)
}

func (attestor *bug453BlockingAttestor) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-attestor.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("attestation did not reach physical call")
	}
}

func (attestor *bug453BlockingAttestor) release() {
	attestor.releaseOnce.Do(func() { close(attestor.releaseCh) })
}
