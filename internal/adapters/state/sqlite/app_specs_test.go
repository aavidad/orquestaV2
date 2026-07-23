package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func TestRepositoryMigratesPopulatedV2ToCanonicalAppSpecsAndSecondOpenIsStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2", "orquesta.sqlite")
	seedPopulatedV2Database(t, path, false)
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if err != nil {
		t.Fatalf("migrate populated V2: %v", err)
	}
	ref := mustRef(t, "goal:v2-populated", goal.NewGoalRef)
	first, err := repository.GetGoal(context.Background(), ref)
	if err != nil {
		t.Fatalf("read migrated Goal: %v", err)
	}
	spec := first.Goal.AppSpec()
	if spec.Ref().String() != "app-spec:migrated:"+ref.String() || spec.Generation() != 1 ||
		spec.Objective() != "legacy v2-populated" || spec.Reason() != "migration.v2_to_v3" ||
		spec.ConfirmedBy() != first.Goal.Actor() || spec.Hash() == first.Goal.IntentHash() {
		t.Fatalf("canonical migrated AppSpec = %+v", spec.Snapshot())
	}
	if _, hasParent := spec.ParentRef(); hasParent {
		t.Fatal("migrated root has parent")
	}
	firstSnapshot := first.Goal.Snapshot()
	if err := repository.Close(); err != nil {
		t.Fatalf("close first open: %v", err)
	}

	repository, err = Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	second, err := repository.GetGoal(context.Background(), ref)
	if err != nil || !reflect.DeepEqual(second.Goal.Snapshot(), firstSnapshot) {
		t.Fatalf("second Open changed migrated Goal: record=%+v err=%v", second, err)
	}
	if got := tableCount(t, repository, "app_specs"); got != 1 {
		t.Fatalf("AppSpec rows after second Open = %d", got)
	}
	var version, receipts int
	if err := repository.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	if err := repository.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&receipts); err != nil {
		t.Fatalf("migration receipts: %v", err)
	}
	if version != recoverySchemaV21 || receipts != recoverySchemaV21 {
		t.Fatalf("migration state version=%d receipts=%d", version, receipts)
	}
}

func TestRepositoryCorruptV2BackfillRollsBackSchemaReceiptAndUserVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2-corrupt", "orquesta.sqlite")
	seedPopulatedV2Database(t, path, true)
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if repository != nil {
		_ = repository.Close()
		t.Fatal("corrupt V2 unexpectedly migrated")
	}
	if !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("corrupt migration error = %v", err)
	}

	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	if err != nil {
		t.Fatalf("inspect rolled back V2: %v", err)
	}
	defer database.Close()
	var version, receiptCount, stagingTables int
	if err := database.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("rolled back user_version: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = 3").Scan(&receiptCount); err != nil {
		t.Fatalf("rolled back receipt: %v", err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*) FROM sqlite_schema
WHERE type = 'table' AND name IN ('app_specs', 'goals_v3')`).Scan(&stagingTables); err != nil {
		t.Fatalf("rolled back schema: %v", err)
	}
	if version != 2 || receiptCount != 0 || stagingTables != 0 {
		t.Fatalf("partial migration escaped: version=%d receipt=%d staging=%d", version, receiptCount, stagingTables)
	}
	var legacyIntentColumn int
	if err := database.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('goals') WHERE name = 'intent_ref'`).Scan(&legacyIntentColumn); err != nil {
		t.Fatalf("legacy goals schema: %v", err)
	}
	if legacyIntentColumn != 1 {
		t.Fatalf("legacy goals table was replaced: intent_ref columns=%d", legacyIntentColumn)
	}
}

func TestRepositoryDomainInvalidV2GoalRollsBackAppSpecMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2-domain-invalid", "orquesta.sqlite")
	seedPopulatedV2Database(t, path, false)
	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	if err != nil {
		t.Fatalf("open V2 mutation database: %v", err)
	}
	if _, err := database.Exec(`
UPDATE goals
SET closed_at = created_at + 1
WHERE ref = 'goal:v2-populated' AND state = 'running'`); err != nil {
		_ = database.Close()
		t.Fatalf("make SQL-valid/domain-invalid V2 Goal: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close V2 mutation database: %v", err)
	}

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if repository != nil {
		_ = repository.Close()
		t.Fatal("domain-invalid V2 Goal unexpectedly migrated")
	}
	if !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("domain-invalid migration error = %v", err)
	}

	database, err = sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	if err != nil {
		t.Fatalf("inspect rolled back V2: %v", err)
	}
	defer database.Close()
	var version, receiptCount, stagingTables, legacyCount int
	if err := database.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("rolled back user_version: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = 3").Scan(&receiptCount); err != nil {
		t.Fatalf("rolled back receipt: %v", err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*) FROM sqlite_schema
WHERE type = 'table' AND name IN ('app_specs', 'goals_v3')`).Scan(&stagingTables); err != nil {
		t.Fatalf("rolled back schema: %v", err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*)
FROM goals
WHERE ref = 'goal:v2-populated'
  AND intent_ref = 'intent:v2-populated'
  AND state = 'running'
  AND closed_at = created_at + 1
  AND plan_generation = 1`).Scan(&legacyCount); err != nil {
		t.Fatalf("rolled back legacy Goal: %v", err)
	}
	if version != 2 || receiptCount != 0 || stagingTables != 0 || legacyCount != 1 {
		t.Fatalf("partial migration escaped: version=%d receipt=%d staging=%d legacy=%d",
			version, receiptCount, stagingTables, legacyCount)
	}
}

func TestValidateGoalRecordConsistencyRejectsWorkItemExecutionStateMismatch(t *testing.T) {
	repository, _ := openTestRepository(t)
	record := createFailedSourceForAmend(t, repository, "inconsistent-execution-state")
	execution := &record.Executions[0]
	execution.State = application.ExecutionSucceeded
	execution.ProviderRef = "provider:state-mismatch"
	execution.ModelRef = "model:state-mismatch"
	execution.AgentRef = "agent:state-mismatch"
	execution.ExternalRef = "external:state-mismatch"
	execution.StartedAt = execution.CreatedAt
	execution.DeadlineAt = execution.CreatedAt.Add(time.Hour)
	execution.ProviderAcceptedAt = execution.CreatedAt
	execution.FinishedAt = execution.CreatedAt.Add(time.Second)
	execution.FailureCode = ""
	if err := validateExecution(*execution); err != nil {
		t.Fatalf("test execution must be valid in isolation: %v", err)
	}
	if err := validateGoalRecordConsistency(record, record.Goal.Ref().String()); err == nil {
		t.Fatal("failed WorkItem accepted with succeeded execution")
	}
}

func TestRepositoryImmutableIntentAppSpecAndGoalBindingTriggers(t *testing.T) {
	repository, _ := openTestRepository(t)
	state := newCreateFixture(t, "immutable", "request:immutable", "fingerprint:immutable", "actor:owner", "project:immutable")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create immutable fixture: %v", err)
	}
	intentRef := state.Goal.Intent().String()
	specRef := state.Goal.AppSpec().Ref().String()
	goalRef := state.Goal.Ref().String()
	mutations := []struct {
		name  string
		query string
		args  []any
	}{
		{"update intent", "UPDATE intents SET statement = 'changed' WHERE ref = ?", []any{intentRef}},
		{"delete intent", "DELETE FROM intents WHERE ref = ?", []any{intentRef}},
		{"update AppSpec", "UPDATE app_specs SET objective = 'changed' WHERE ref = ?", []any{specRef}},
		{"delete AppSpec", "DELETE FROM app_specs WHERE ref = ?", []any{specRef}},
		{"rebind Goal", "UPDATE goals SET app_spec_ref = app_spec_ref WHERE ref = ?", []any{goalRef}},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			if _, err := repository.db.Exec(mutation.query, mutation.args...); err == nil {
				t.Fatalf("mutation accepted: %s", mutation.name)
			}
		})
	}
	restored, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil || !reflect.DeepEqual(restored.Goal.Snapshot(), state.Goal.Snapshot()) {
		t.Fatalf("immutable binding changed: record=%+v err=%v", restored, err)
	}
}

func TestRepositoryAmendGoalIsAtomicIdempotentAndRestartable(t *testing.T) {
	repository, path := openTestRepository(t)
	source := createFailedSourceForAmend(t, repository, "amend-source")
	sourceSnapshot := source.Goal.Snapshot()
	state := newAmendFixture(t, source, "amend-one", "request:amend-one", "fingerprint:amend-one", "rebuild cleanly", "operator correction")
	record, created, err := amendLegacyGoal(t, repository, state)
	if err != nil || !created {
		t.Fatalf("AmendGoal() created=%v error=%v", created, err)
	}
	assertAmendedRecord(t, record, source, state)
	unchanged, err := repository.GetGoal(context.Background(), source.Goal.Ref())
	if err != nil || !reflect.DeepEqual(unchanged.Goal.Snapshot(), sourceSnapshot) {
		t.Fatalf("source changed by amendment: record=%+v err=%v", unchanged, err)
	}

	replayState := newAmendFixture(t, source, "amend-replay", state.RequestRef, state.RequestFingerprint, "rebuild cleanly", "operator correction")
	replayed, created, err := amendLegacyGoal(t, repository, replayState)
	if err != nil || created || replayed.Goal.Ref() != record.Goal.Ref() {
		t.Fatalf("semantic amendment replay created=%v record=%+v err=%v", created, replayed, err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close before amendment restart: %v", err)
	}
	repository, err = Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8})
	if err != nil {
		t.Fatalf("restart after amendment: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	restarted, err := repository.GetGoal(context.Background(), record.Goal.Ref())
	if err != nil || !reflect.DeepEqual(restarted.Goal.Snapshot(), record.Goal.Snapshot()) {
		t.Fatalf("amendment restart mismatch: record=%+v err=%v", restarted, err)
	}
	if _, created, err := amendLegacyGoal(t, repository, replayState); err != nil || created {
		t.Fatalf("amendment replay after restart created=%v err=%v", created, err)
	}
}

func TestRepositoryPersistsAppSpecConfirmedByReviewerDifferentFromIntentActor(t *testing.T) {
	repository, _ := openTestRepository(t)
	source := createFailedSourceForAmend(t, repository, "reviewer-source")
	state := newAmendFixture(t, source, "reviewer-amend", "request:reviewer-amend", "fingerprint:reviewer-amend", "reviewed objective", "reviewer confirmation")
	original := state.Successor.AppSpec()
	reviewer := mustRef(t, "actor:reviewer", goal.NewActorRef)
	reviewed, err := source.Goal.AppSpec().Amend(goal.AppSpecInput{
		Ref: original.Ref(), Intent: original.Intent(), Objective: original.Objective(), Reason: original.Reason(),
		ConfirmedBy: reviewer, ConfirmedAt: original.ConfirmedAt(),
	})
	if err != nil {
		t.Fatalf("reviewer-confirmed AppSpec: %v", err)
	}
	successor, err := goal.NewSuccessorGoal(state.Successor.Ref(), source.Goal, reviewed, state.Successor.CreatedAt())
	if err != nil {
		t.Fatalf("reviewer-confirmed successor: %v", err)
	}
	state.Successor = successor
	record, created, err := amendLegacyGoal(t, repository, state)
	if err != nil || !created || record.Goal.AppSpec().ConfirmedBy() != reviewer {
		t.Fatalf("reviewer confirmation created=%v confirmed_by=%s err=%v", created, record.Goal.AppSpec().ConfirmedBy(), err)
	}
}

func TestRepositoryConcurrentSecondSuccessorAllowsOneAndRollsBackLoser(t *testing.T) {
	repository, _ := openTestRepository(t)
	source := createFailedSourceForAmend(t, repository, "concurrent-amend-source")
	states := []application.AmendGoalState{
		newAmendFixture(t, source, "concurrent-amend-a", "request:concurrent-amend-a", "fingerprint:concurrent-amend-a", "option a", "reason a"),
		newAmendFixture(t, source, "concurrent-amend-b", "request:concurrent-amend-b", "fingerprint:concurrent-amend-b", "option b", "reason b"),
	}
	for index := range states {
		states[index] = authorizeLegacyAmendState(t, repository, states[index])
	}
	beforeIntents := tableCount(t, repository, "intents")
	beforeSpecs := tableCount(t, repository, "app_specs")
	beforeGoals := tableCount(t, repository, "goals")
	start := make(chan struct{})
	results := make(chan error, len(states))
	var createdCount atomic.Int64
	var wait sync.WaitGroup
	for _, state := range states {
		state := state
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, created, err := repository.AmendGoal(context.Background(), state)
			if created {
				createdCount.Add(1)
			}
			results <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	conflicts := 0
	for err := range results {
		switch {
		case err == nil:
		case application.IsStateError(err, application.StateConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent amendment error: %v", err)
		}
	}
	if createdCount.Load() != 1 || conflicts != 1 {
		t.Fatalf("concurrent successors created=%d conflicts=%d", createdCount.Load(), conflicts)
	}
	if tableCount(t, repository, "intents") != beforeIntents+1 ||
		tableCount(t, repository, "app_specs") != beforeSpecs+1 ||
		tableCount(t, repository, "goals") != beforeGoals+1 {
		t.Fatal("losing successor left partial rows")
	}
}

func TestRepositoryAmendLateEventConflictRollsBackIntentSpecAndGoal(t *testing.T) {
	repository, _ := openTestRepository(t)
	source := createFailedSourceForAmend(t, repository, "rollback-amend-source")
	state := newAmendFixture(t, source, "rollback-amend", "request:rollback-amend", "fingerprint:rollback-amend", "rollback", "late event conflict")
	state.Events[0].Ref = "event:goal-created:rollback-amend-source"
	beforeIntents := tableCount(t, repository, "intents")
	beforeSpecs := tableCount(t, repository, "app_specs")
	beforeGoals := tableCount(t, repository, "goals")
	if _, _, err := amendLegacyGoal(t, repository, state); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("late amendment conflict = %v", err)
	}
	if tableCount(t, repository, "intents") != beforeIntents ||
		tableCount(t, repository, "app_specs") != beforeSpecs ||
		tableCount(t, repository, "goals") != beforeGoals {
		t.Fatal("late amendment failure committed partial rows")
	}
	if _, err := repository.GetGoal(context.Background(), state.Successor.Ref()); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("rolled back successor lookup = %v", err)
	}
}

func seedPopulatedV2Database(t *testing.T, path string, corruptIntentHash bool) {
	t.Helper()
	if err := preparePrivateDatabase(path); err != nil {
		t.Fatalf("prepare V2 path: %v", err)
	}
	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	if err != nil {
		t.Fatalf("open V2 seed: %v", err)
	}
	defer database.Close()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	transaction, err := database.Begin()
	if err != nil {
		t.Fatalf("begin V2 seed: %v", err)
	}
	defer transaction.Rollback()
	if _, err := transaction.Exec(migrations[0].sql); err != nil {
		t.Fatalf("apply V1 schema: %v", err)
	}
	base := time.Date(2026, 7, 14, 8, 0, 0, 0, time.UTC)
	seedV1Goal(t, transaction, "v2-populated", "running", 3, "pending", 1, "queued", base, false)
	if _, err := transaction.Exec(migrations[1].sql); err != nil {
		t.Fatalf("apply V2 schema: %v", err)
	}
	for index := 0; index < 2; index++ {
		migration := migrations[index]
		if _, err := transaction.Exec(
			"INSERT INTO schema_migrations(version, name, checksum) VALUES (?, ?, ?)",
			migration.version, migration.name, migration.checksum,
		); err != nil {
			t.Fatalf("record migration %d: %v", migration.version, err)
		}
	}
	if corruptIntentHash {
		if _, err := transaction.Exec("UPDATE intents SET hash = ?", "0000000000000000000000000000000000000000000000000000000000000000"); err != nil {
			t.Fatalf("corrupt V2 intent: %v", err)
		}
	}
	if _, err := transaction.Exec("PRAGMA user_version = 2"); err != nil {
		t.Fatalf("set V2 user_version: %v", err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatalf("commit V2 seed: %v", err)
	}
}

func createFailedSourceForAmend(t *testing.T, repository *Repository, suffix string) application.GoalRecord {
	t.Helper()
	state := newCreateFixture(t, suffix, "request:"+suffix, "fingerprint:"+suffix, "actor:amend", "project:amend")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create amendment source: %v", err)
	}
	claim := mustClaim(t, repository, "worker:"+suffix, "claim:"+suffix, state.Executions[0].CreatedAt)
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("get amendment source: %v", err)
	}
	item := onlyItem(t, record.Goal)
	failedAt := state.Executions[0].CreatedAt.Add(time.Second)
	preparedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, failedAt,
	)
	if err != nil {
		t.Fatalf("prepare amendment source: %v", err)
	}
	preparedExecution := record.Executions[0]
	preparedExecution.State = application.ExecutionDispatching
	repository.now = func() time.Time { return failedAt }
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: preparedGoal,
		Execution: preparedExecution, OperationAt: failedAt,
		Event: application.EventRecord{
			Ref: "event:execution-dispatching:" + suffix, Kind: "execution.dispatching",
			GoalRef: preparedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: preparedExecution.Ref,
			OccurredAt: failedAt,
		},
	}); err != nil {
		t.Fatalf("record amendment source preparation: %v", err)
	}
	runningItem := onlyItem(t, preparedGoal)
	expectedGoalRevision := preparedGoal.Revision()
	expectedItemRevision := runningItem.Revision()
	failedGoal, err := preparedGoal.FailWorkItem(
		preparedGoal.Revision(), runningItem.Revision(), runningItem.Ref(), failedAt,
	)
	if err == nil {
		failedGoal, err = failedGoal.Close(failedGoal.Revision(), goal.GoalOutcomeFailed, failedAt)
	}
	if err != nil {
		t.Fatalf("close amendment source: %v", err)
	}
	failedExecution := preparedExecution
	failedExecution.State = application.ExecutionFailed
	failedExecution.FinishedAt = failedAt
	failedExecution.FailureCode = "test.failed"
	if err := repository.RecordGoalFailed(context.Background(), application.GoalFailedState{
		Claim: claim, ExpectedGoalRevision: expectedGoalRevision, ExpectedItemRevision: expectedItemRevision,
		Goal: failedGoal, Execution: failedExecution, OperationAt: failedAt,
		Events: []application.EventRecord{
			{Ref: "event:work-failed:" + suffix, Kind: "work_item.failed", GoalRef: failedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: failedExecution.Ref, OccurredAt: failedAt},
			{Ref: "event:goal-failed:" + suffix, Kind: "goal.failed", GoalRef: failedGoal.Ref(), OccurredAt: failedAt},
		},
	}); err != nil {
		t.Fatalf("persist amendment source failure: %v", err)
	}
	terminal, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil || !terminal.Goal.IsTerminal() {
		t.Fatalf("terminal amendment source = %+v err=%v", terminal, err)
	}
	return terminal
}

func newAmendFixture(
	t *testing.T,
	source application.GoalRecord,
	suffix, requestRef, fingerprint, statement, reason string,
) application.AmendGoalState {
	t.Helper()
	closedAt, ok := source.Goal.ClosedAt()
	if !ok {
		t.Fatal("amendment source is not closed")
	}
	at := closedAt.Add(time.Second)
	intent, err := goal.NewIntentManifest(goal.IntentManifestInput{
		Ref:   mustRef(t, "intent:"+suffix, goal.NewIntentRef),
		Actor: source.Goal.Actor(), Project: source.Goal.Project(), Statement: statement, SubmittedAt: at,
	})
	if err != nil {
		t.Fatalf("new amendment Intent: %v", err)
	}
	spec, err := source.Goal.AppSpec().Amend(goal.AppSpecInput{
		Ref: mustRef(t, "app-spec:"+suffix, goal.NewAppSpecRef), Intent: intent,
		Objective: statement, Reason: reason, ConfirmedBy: source.Goal.Actor(), ConfirmedAt: at,
	})
	if err != nil {
		t.Fatalf("new amendment AppSpec: %v", err)
	}
	successor, err := goal.NewSuccessorGoal(mustRef(t, "goal:"+suffix, goal.NewGoalRef), source.Goal, spec, at)
	if err != nil {
		t.Fatalf("new successor Goal: %v", err)
	}
	return application.AmendGoalState{
		RequestRef: requestRef, RequestFingerprint: fingerprint,
		RequestedBy: source.RequestedBy, ProjectRef: source.Goal.Project(), SourceGoalRef: source.Goal.Ref(),
		ExpectedSourceRevision: source.Goal.Revision(), ExpectedSourceSpecHash: source.Goal.SpecHash(),
		Successor: successor,
		Events: []application.EventRecord{{
			Ref: "event:goal-amended:" + suffix, Kind: "goal.amended", GoalRef: successor.Ref(), OccurredAt: at,
		}},
	}
}

func assertAmendedRecord(
	t *testing.T,
	record application.GoalRecord,
	source application.GoalRecord,
	state application.AmendGoalState,
) {
	t.Helper()
	spec := record.Goal.AppSpec()
	parent, hasParent := spec.ParentRef()
	if record.RequestRef != state.RequestRef || record.RequestFingerprint != state.RequestFingerprint ||
		record.Goal.State() != goal.GoalStatePending || record.Goal.PlanGeneration() != 0 ||
		record.Goal.WorkItemCount() != 0 || len(record.Executions) != 0 || len(record.Artifacts) != 0 ||
		len(record.Attestations) != 0 || !hasParent || parent != source.Goal.AppSpec().Ref() ||
		spec.ParentHash() != source.Goal.SpecHash() || spec.Generation() != source.Goal.AppSpec().Generation()+1 {
		t.Fatalf("amended record mismatch: %+v", record)
	}
}

func TestRepositoryAppSpecParentTriggersRejectHashGenerationAndScopeMismatch(t *testing.T) {
	repository, _ := openTestRepository(t)
	source := createFailedSourceForAmend(t, repository, "trigger-parent-source")
	parent := source.Goal.AppSpec()
	closedAt, _ := source.Goal.ClosedAt()
	cases := []struct {
		name       string
		actorValue string
		generation int64
		parentHash string
	}{
		{"hash", source.Goal.Actor().String(), int64(parent.Generation() + 1), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"generation", source.Goal.Actor().String(), int64(parent.Generation() + 2), parent.Hash()},
		{"scope", "actor:other", int64(parent.Generation() + 1), parent.Hash()},
	}
	for index, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			transaction, err := repository.db.Begin()
			if err != nil {
				t.Fatalf("begin trigger test: %v", err)
			}
			defer transaction.Rollback()
			actor := mustRef(t, testCase.actorValue, goal.NewActorRef)
			intent, err := goal.NewIntentManifest(goal.IntentManifestInput{
				Ref:   mustRef(t, "intent:trigger-child:"+testIndex(index), goal.NewIntentRef),
				Actor: actor, Project: source.Goal.Project(), Statement: "invalid child", SubmittedAt: closedAt.Add(time.Second),
			})
			if err != nil {
				t.Fatalf("new trigger Intent: %v", err)
			}
			intentSnapshot := intent.Snapshot()
			if _, err := transaction.Exec(`
INSERT INTO intents(ref, actor_ref, project_ref, statement, submitted_at, hash)
VALUES (?, ?, ?, ?, ?, ?)`, intentSnapshot.Ref, intentSnapshot.ActorRef, intentSnapshot.ProjectRef,
				intentSnapshot.Statement, requiredTime(intentSnapshot.SubmittedAt), intentSnapshot.Hash); err != nil {
				t.Fatalf("insert trigger Intent: %v", err)
			}
			_, err = transaction.Exec(`
INSERT INTO app_specs(ref, intent_ref, generation, parent_ref, parent_hash, objective, reason,
                      confirmed_by, confirmed_at, hash)
VALUES (?, ?, ?, ?, ?, 'invalid child', 'test', ?, ?, ?)`,
				"app-spec:trigger-child:"+testIndex(index), intent.Ref().String(), testCase.generation,
				parent.Ref().String(), testCase.parentHash, actor.String(), requiredTime(closedAt.Add(time.Second)),
				"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			)
			if err == nil {
				t.Fatalf("invalid parent binding accepted: %+v", testCase)
			}
		})
	}
}
