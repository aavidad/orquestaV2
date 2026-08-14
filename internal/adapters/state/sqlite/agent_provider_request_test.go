package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestAgentProviderRequestRecordsBindsStagesReplaysAndReopens(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "provider-request-reopen")
	launch := sqliteAgentProviderRequest(attempt, ports.AgentProviderRequestLaunch)

	prepared, err := system.repository.RecordAgentProviderRequest(context.Background(), launch)
	if err != nil || !reflect.DeepEqual(prepared, launch) {
		t.Fatalf("RecordAgentProviderRequest()=%+v err=%v", prepared, err)
	}
	prepared.Body[0] ^= 0xff
	replayed, err := system.repository.RecordAgentProviderRequest(context.Background(), launch)
	if err != nil || !reflect.DeepEqual(replayed, launch) {
		t.Fatalf("replay=%+v err=%v", replayed, err)
	}

	bound, err := system.repository.BindAgentProviderLaunch(
		context.Background(), launch.Key, "ejecucion:docker_physical", 3,
	)
	if err != nil || bound.LaunchBindingRef != "ejecucion:docker_physical" || bound.LaunchBindingRevision != 3 {
		t.Fatalf("BindAgentProviderLaunch()=%+v err=%v", bound, err)
	}
	boundAgain, err := system.repository.BindAgentProviderLaunch(
		context.Background(), launch.Key, "ejecucion:docker_physical", 3,
	)
	if err != nil || !reflect.DeepEqual(boundAgain, bound) {
		t.Fatalf("binding replay=%+v err=%v", boundAgain, err)
	}

	for _, stage := range []ports.AgentProviderRequestStage{
		ports.AgentProviderRequestSessionStart,
		ports.AgentProviderRequestSessionInput,
	} {
		request := sqliteAgentProviderRequest(attempt, stage)
		request.TargetRef, request.ExpectedRevision = bound.LaunchBindingRef, bound.LaunchBindingRevision
		stored, err := system.repository.RecordAgentProviderRequest(context.Background(), request)
		if err != nil || !reflect.DeepEqual(stored, request) {
			t.Fatalf("stage %s stored=%+v err=%v", stage, stored, err)
		}
	}

	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	for _, stage := range []ports.AgentProviderRequestStage{
		ports.AgentProviderRequestLaunch,
		ports.AgentProviderRequestSessionStart,
		ports.AgentProviderRequestSessionInput,
	} {
		key := launch.Key
		key.Stage = stage
		resolved, found, err := reopened.ResolveAgentProviderRequest(context.Background(), key)
		if err != nil || !found || resolved.Key.Stage != stage {
			t.Fatalf("stage %s resolved=%+v found=%t err=%v", stage, resolved, found, err)
		}
		resolved.Body[0] ^= 0xff
		again, found, err := reopened.ResolveAgentProviderRequest(context.Background(), key)
		if err != nil || !found || reflect.DeepEqual(resolved.Body, again.Body) {
			t.Fatalf("stage %s aliases persisted body: found=%t err=%v", stage, found, err)
		}
	}
	missing := launch.Key
	missing.Stage = ports.AgentProviderRequestSessionInput
	missing.ActionFence++
	if resolved, found, err := reopened.ResolveAgentProviderRequest(context.Background(), missing); err != nil || found || !reflect.DeepEqual(resolved, ports.AgentProviderRequest{}) {
		t.Fatalf("missing resolved=%+v found=%t err=%v", resolved, found, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), reopened.db); err != nil {
		t.Fatalf("recovery validation: %v", err)
	}
}

func TestAgentProviderRequestRejectsDivergenceAndOutOfOrderStages(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "provider-request-order")
	launch := sqliteAgentProviderRequest(attempt, ports.AgentProviderRequestLaunch)
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), launch); err != nil {
		t.Fatal(err)
	}

	divergent := ports.CloneAgentProviderRequest(launch)
	divergent.Body = []byte(`{"schema":"divergent.v1"}`)
	divergent.BodySHA256 = ports.AgentProviderRequestBodySHA256(divergent.Body)
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), divergent); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent replay err=%v", err)
	}

	start := sqliteAgentProviderRequest(attempt, ports.AgentProviderRequestSessionStart)
	start.TargetRef, start.ExpectedRevision = "ejecucion:docker_physical", 3
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), start); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("session before binding err=%v", err)
	}
	if _, err := system.repository.BindAgentProviderLaunch(context.Background(), launch.Key, start.TargetRef, start.ExpectedRevision); err != nil {
		t.Fatal(err)
	}
	input := sqliteAgentProviderRequest(attempt, ports.AgentProviderRequestSessionInput)
	input.TargetRef, input.ExpectedRevision = start.TargetRef, start.ExpectedRevision
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), input); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("input before session start err=%v", err)
	}
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), start); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.BindAgentProviderLaunch(context.Background(), launch.Key, "ejecucion:other", 3); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent binding err=%v", err)
	}
}

func TestAgentProviderRequestRejectsCrossedExecutionAndFenceAtPersistence(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "provider-request-crossed-causality")
	launch := sqliteAgentProviderRequest(attempt, ports.AgentProviderRequestLaunch)

	wrongFence := ports.CloneAgentProviderRequest(launch)
	wrongFence.Key.ActionFence++
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), wrongFence); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("crossed fence err=%v", err)
	}

	wrongExecution := ports.CloneAgentProviderRequest(launch)
	wrongExecution.Key.ExecutionRef, _ = goal.NewExecutionRef("execution:provider-request-crossed")
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), wrongExecution); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("crossed execution err=%v", err)
	}

	if resolved, found, err := system.repository.ResolveAgentProviderRequest(context.Background(), launch.Key); err != nil || found || !reflect.DeepEqual(resolved, ports.AgentProviderRequest{}) {
		t.Fatalf("unexpected persisted request=%+v found=%t err=%v", resolved, found, err)
	}
}

func TestAgentProviderRequestConcurrentLaunchBindingIsSetOnce(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "provider-request-concurrent-binding")
	launch := sqliteAgentProviderRequest(attempt, ports.AgentProviderRequestLaunch)
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), launch); err != nil {
		t.Fatal(err)
	}

	type bindResult struct {
		request ports.AgentProviderRequest
		err     error
	}
	start := make(chan struct{})
	results := make(chan bindResult, 2)
	for index, externalRef := range []string{"ejecucion:docker_first", "ejecucion:docker_second"} {
		revision := uint64(index + 1)
		go func() {
			<-start
			request, err := system.repository.BindAgentProviderLaunch(
				context.Background(), launch.Key, externalRef, revision,
			)
			results <- bindResult{request: request, err: err}
		}()
	}
	close(start)

	var winner ports.AgentProviderRequest
	successes, conflicts := 0, 0
	for range 2 {
		result := <-results
		switch {
		case result.err == nil:
			successes++
			winner = result.request
		case application.IsStateError(result.err, application.StateConflict):
			conflicts++
		default:
			t.Fatalf("unexpected binding error: %v", result.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	resolved, found, err := system.repository.ResolveAgentProviderRequest(context.Background(), launch.Key)
	if err != nil || !found || !reflect.DeepEqual(resolved, winner) {
		t.Fatalf("resolved=%+v winner=%+v found=%t err=%v", resolved, winner, found, err)
	}
}

func TestAgentProviderRequestRecoveryRejectsBodyCorruption(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "provider-request-corruption")
	launch := sqliteAgentProviderRequest(attempt, ports.AgentProviderRequestLaunch)
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), launch); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.db.Exec(`DROP TRIGGER agent_provider_requests_bind_launch_once`); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.db.Exec(`
UPDATE agent_provider_requests SET body=?
WHERE execution_ref=? AND action_fence=? AND stage='launch'`,
		[]byte(`{"schema":"corrupt.v1"}`), launch.Key.ExecutionRef.String(), launch.Key.ActionFence,
	); err != nil {
		t.Fatal(err)
	}
	transaction, err := beginReadTransaction(context.Background(), system.repository)
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback()
	if err := validateRecoveryV38AgentProviderRequests(context.Background(), transaction); err == nil ||
		!recoveryErrorContains(err, recoveryV38AgentProviderRequestInvalid) {
		t.Fatalf("corrupt recovery err=%v", err)
	}
}

func TestAgentProviderRequestRecoveryRejectsCrossedExecutionProvider(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "provider-request-crossed-provider")
	launch := sqliteAgentProviderRequest(attempt, ports.AgentProviderRequestLaunch)
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), launch); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.BindAgentProviderLaunch(
		context.Background(), launch.Key, "ejecucion:docker_physical", 3,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.db.Exec(`DROP TRIGGER executions_provider_identity_write_once`); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.db.Exec(`
UPDATE executions SET provider_ref=?,model_ref=?,agent_ref=?,external_ref=? WHERE ref=?`,
		"provider:other", "model:other", "agent:other", "ejecucion:docker_physical",
		attempt.Subject.ExecutionRef.String(),
	); err != nil {
		t.Fatal(err)
	}
	transaction, err := beginReadTransaction(context.Background(), system.repository)
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback()
	if err := validateRecoveryV38AgentProviderRequests(context.Background(), transaction); err == nil ||
		!recoveryErrorContains(err, recoveryV38AgentProviderRequestInvalid) {
		t.Fatalf("crossed provider recovery err=%v", err)
	}
}

func TestAgentProviderRequestCannotFirstBindAfterEffectReceipt(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:provider-request-receipted")
	claim := claimSQLiteV15(t, system, "claim:provider-request-receipted")
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	if _, created, err := system.repository.RecordEffectAttempt(
		context.Background(), application.RecordEffectAttemptState{
			Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
		},
	); err != nil || !created {
		t.Fatalf("RecordEffectAttempt created=%t err=%v", created, err)
	}
	launch := sqliteAgentProviderRequest(attempt, ports.AgentProviderRequestLaunch)
	if _, err := system.repository.RecordAgentProviderRequest(context.Background(), launch); err != nil {
		t.Fatal(err)
	}
	receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusAccepted, system.clock.Now())
	transaction, err := beginTransaction(context.Background(), system.repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := insertEffectReceipt(context.Background(), transaction, claim, receipt, receipt.ConfirmedAt); err != nil {
		_ = transaction.Rollback()
		t.Fatal(err)
	}
	if err := commit(transaction); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.BindAgentProviderLaunch(
		context.Background(), launch.Key, "ejecucion:too_late", 3,
	); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("late binding err=%v", err)
	}
}

func TestV37AgentProviderRequestMigrationHasStrictCausalShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-provider-request-v36.db")
	database := agentCapacityDatabase(t, path, recoverySchemaV23WizardGapsSnapshot)
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = repository.Close() })

	var version, strict, migrations int
	var name, tableSQL string
	sqliteTestNoError(t, repository.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, repository.db.QueryRow(`
SELECT name FROM schema_migrations WHERE version=?`, recoverySchemaV38AgentProviderRequest).Scan(&name))
	sqliteTestNoError(t, repository.db.QueryRow(`
SELECT strict FROM pragma_table_list WHERE name='agent_provider_requests'`).Scan(&strict))
	sqliteTestNoError(t, repository.db.QueryRow(`
SELECT sql FROM sqlite_schema WHERE type='table' AND name='agent_provider_requests'`).Scan(&tableSQL))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations))
	if version != recoverySchemaLatest || migrations != recoverySchemaLatest ||
		name != "037_agent_provider_requests.sql" || strict != 1 ||
		!strings.Contains(tableSQL, "FOREIGN KEY(effect_attempt_ref,execution_ref,action_fence)") {
		t.Fatalf("version=%d migrations=%d name=%q strict=%d sql=%s", version, migrations, name, strict, tableSQL)
	}
}

func sqliteAgentProviderRequest(
	attempt application.EffectAttempt,
	stage ports.AgentProviderRequestStage,
) ports.AgentProviderRequest {
	body := []byte(`{"schema":"` + string(stage) + `.v1"}`)
	return ports.AgentProviderRequest{
		Key: ports.AgentProviderRequestKey{
			ExecutionRef: attempt.Subject.ExecutionRef,
			ActionFence:  attempt.ActionFence,
			Stage:        stage,
		},
		EffectAttemptRef: attempt.Ref,
		ProviderRef:      "provider:docker",
		IdempotencyKey:   "provider-request:" + string(stage) + ":" + attempt.Ref,
		Body:             body,
		BodySHA256:       ports.AgentProviderRequestBodySHA256(body),
	}
}
