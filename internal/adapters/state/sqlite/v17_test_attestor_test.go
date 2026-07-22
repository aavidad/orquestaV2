package sqlite

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSQLiteAttestationPassIsAtomicAndSurvivesRestart(t *testing.T) {
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processSQLiteV16Actions(t, system, application.ActionAttestTest)
	assertSQLiteV17Pass(t, system.repository, goalRef)

	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	assertSQLiteV17Pass(t, system.repository, goalRef)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("V17 pass did not survive recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestConcurrentAttestationFenceAllowsOneWriter(t *testing.T) {
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)

	type outcome struct {
		result application.ProcessResult
		err    error
	}
	start := make(chan struct{})
	results := make(chan outcome, 2)
	var workers sync.WaitGroup
	for index := 0; index < 2; index++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			<-start
			result, err := system.orchestrator.ProcessNext(
				context.Background(), "worker:v17-concurrent:"+string(rune('a'+worker)),
			)
			results <- outcome{result: result, err: err}
		}(index)
	}
	close(start)
	workers.Wait()
	close(results)

	processed := 0
	for outcome := range results {
		if outcome.err != nil {
			t.Fatalf("concurrent attestation: %v", outcome.err)
		}
		if outcome.result.Processed {
			if outcome.result.Action != application.ActionAttestTest {
				t.Fatalf("unexpected concurrent action: %+v", outcome.result)
			}
			processed++
		}
	}
	if processed != 1 {
		t.Fatalf("concurrent processed attestations=%d want=1", processed)
	}
	if calls, effects := attestor.counts(); calls != 1 || effects != 1 {
		t.Fatalf("attestor calls/effects=%d/%d want=1/1", calls, effects)
	}
	assertSQLiteV17Pass(t, system.repository, goalRef)
}

func TestTestAttestationCrashFrontiersReplayExactlyOnce(t *testing.T) {
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	mustV10Exec(t, system.repository.db, `
CREATE TRIGGER v17_test_crash_before_attestation
BEFORE INSERT ON attestations WHEN NEW.kind='required_tests'
BEGIN SELECT RAISE(ABORT,'v17_test_crash'); END`)

	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v17-crash")
	if err == nil || err.Error() != "application.effect_unknown_applied" ||
		!result.Processed || result.Action != application.ActionAttestTest {
		t.Fatalf("injected crash result=%+v err=%v", result, err)
	}
	assertSQLiteV17UnknownAppliedCounts(t, system.repository.db)
	if calls, effects := attestor.counts(); calls != 1 || effects != 1 {
		t.Fatalf("pre-replay attestor calls/effects=%d/%d want=1/1", calls, effects)
	}
	mustV10Exec(t, system.repository.db, `DROP TRIGGER v17_test_crash_before_attestation`)
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, attestor)
	result, err = system.orchestrator.ProcessNext(context.Background(), "worker:v17-crash:recovery")
	if err != nil || result.Processed {
		t.Fatalf("quarantined attestation replayed result=%+v err=%v", result, err)
	}

	if calls, effects := attestor.counts(); calls != 1 || effects != 1 {
		t.Fatalf("recovery attestor calls/effects=%d/%d want=1/1", calls, effects)
	}
	assertSQLiteV17UnknownAppliedCounts(t, system.repository.db)
	var quarantined, attempts int
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM action_consumption_receipts
WHERE kind='attest_test' AND outcome='quarantined'
 AND error_code='application.effect_unknown_applied'`).Scan(&quarantined); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts attempt
JOIN effect_intents intent ON intent.ref=attempt.intent_ref WHERE intent.kind='attest_test'`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if quarantined != 1 || attempts != 1 {
		t.Fatalf("unknown attestation facts quarantined/attempts=%d/%d", quarantined, attempts)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("unknown attestation restart recovery: %v", err)
	}
	_ = goalRef
}

func TestTerminalAttestationReceiptPreventsRestartRerun(t *testing.T) {
	firstAttestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, firstAttestor)
	processSQLiteV16Actions(t, system, application.ActionAttestTest)
	if calls, effects := firstAttestor.counts(); calls != 1 || effects != 1 {
		t.Fatalf("first attestor calls/effects=%d/%d want=1/1", calls, effects)
	}

	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	restartedAttestor := &sqliteTestAttestor{}
	system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, restartedAttestor)
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v17-restart")
	if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("terminal attestation did not advance to durable review after restart: result=%+v err=%v", result, err)
	}
	if calls, effects := restartedAttestor.counts(); calls != 0 || effects != 0 {
		t.Fatalf("restarted attestor calls/effects=%d/%d want=0/0", calls, effects)
	}
	assertSQLiteV17Pass(t, system.repository, goalRef)
}

func TestSQLiteFailedAttestationIsAtomicAndRecoverable(t *testing.T) {
	attestor := &sqliteTestAttestor{verdict: ports.TestAttestationFailed}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processSQLiteV16Actions(t, system, application.ActionAttestTest)
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	item := record.Goal.WorkItems()[0]
	failed := 0
	for _, execution := range record.Executions {
		if execution.State == application.ExecutionFailed &&
			execution.FailureCode == "test_attestor.required_tests_failed" {
			failed++
		}
	}
	typedFailures := 0
	for _, attestation := range record.Attestations {
		if attestation.Kind == application.AttestationKindRequiredTests &&
			attestation.Verdict == application.AttestationVerdictFailed && len(attestation.Tests) == 1 &&
			attestation.Tests[0].ExitCode != 0 {
			typedFailures++
		}
	}
	if item.State() != goal.WorkItemStateInterrupted || failed != 1 || typedFailures != 1 ||
		len(record.ChangeSets) != 1 || len(record.IntegrationReceipts) != 0 {
		t.Fatalf("failed V17 attestation frontier item=%s failed=%d typed=%d record=%+v",
			item.State(), failed, typedFailures, record)
	}
	assertSQLiteV17TerminalCounts(t, system.repository.db, 1, 1, 1)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("failed V17 recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestSQLiteTransientAttestationRequeuesDurablyAndRetriesOnce(t *testing.T) {
	attestor := &sqliteTestAttestor{failures: 1}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v17-transient:first")
	if err == nil || err.Error() != "application.effect_unknown_applied" ||
		!result.Processed || result.Action != application.ActionAttestTest {
		t.Fatalf("transient attestation result=%+v err=%v", result, err)
	}
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	if err != nil || len(record.Executions) != 1 ||
		record.Executions[0].State != application.ExecutionAwaitingAttestation {
		t.Fatalf("transient frontier record=%+v err=%v", record, err)
	}
	var quarantined, attempts int
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM action_consumption_receipts
WHERE kind='attest_test' AND outcome='quarantined'
 AND error_code='application.effect_unknown_applied'`).Scan(&quarantined); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM effect_attempts attempt
JOIN effect_intents intent ON intent.ref=attempt.intent_ref WHERE intent.kind='attest_test'`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	assertSQLiteV17UnknownAppliedCounts(t, system.repository.db)
	if quarantined != 1 || attempts != 1 {
		t.Fatalf("transient durable frontier quarantined/attempts=%d/%d", quarantined, attempts)
	}
	system.clock.Advance(time.Second)
	if replay, replayErr := system.orchestrator.ProcessNext(
		context.Background(), "worker:v17-transient:replay",
	); replayErr != nil || replay.Processed {
		t.Fatalf("quarantined attestation replayed: result=%+v err=%v", replay, replayErr)
	}
	if calls, effects := attestor.counts(); calls != 1 || effects != 0 {
		t.Fatalf("transient retry calls/effects=%d/%d want=1/0", calls, effects)
	}
	assertSQLiteV17UnknownAppliedCounts(t, system.repository.db)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("transient retry V17 recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}
