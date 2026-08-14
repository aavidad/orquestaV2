package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestAgentProviderStopRequestRecordsReplaysAndReopensExactBytes(t *testing.T) {
	system, _, _, request := seedSQLiteAgentProviderStopRequest(t, "provider-stop-reopen")
	stored := recordSQLiteAgentProviderStopRequest(t, system.repository, request)
	stored.Body[0] ^= 0xff
	replayed := recordSQLiteAgentProviderStopRequest(t, system.repository, request)
	if !reflect.DeepEqual(replayed, request) {
		t.Fatalf("replay=%+v want=%+v", replayed, request)
	}
	crossed := request
	crossed.Key.LaunchActionFence++
	if _, err := system.repository.RecordAgentProviderStopRequest(context.Background(), crossed); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("crossed launch replay error=%s", sqliteTestErrorChain(err))
	}

	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	resolved, found, err := reopened.ResolveAgentProviderStopRequest(context.Background(), request.Key)
	if err != nil || !found || !reflect.DeepEqual(resolved, request) {
		t.Fatalf("resolved=%+v found=%t err=%v", resolved, found, err)
	}
	if got, found, err := reopened.ResolveAgentProviderStopRequest(context.Background(), crossed.Key); err != nil || found || !reflect.DeepEqual(got, ports.AgentProviderStopRequest{}) {
		t.Fatalf("crossed launch resolve=%+v found=%t err=%v", got, found, err)
	}
	resolved.Body[0] ^= 0xff
	again, found, err := reopened.ResolveAgentProviderStopRequest(context.Background(), request.Key)
	if err != nil || !found || reflect.DeepEqual(resolved.Body, again.Body) {
		t.Fatalf("resolved body aliases storage: found=%t err=%v", found, err)
	}
	missing := request.Key
	missing.StopActionFence++
	if got, found, err := reopened.ResolveAgentProviderStopRequest(context.Background(), missing); err != nil || found || !reflect.DeepEqual(got, ports.AgentProviderStopRequest{}) {
		t.Fatalf("missing=%+v found=%t err=%v", got, found, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), reopened.db); err != nil {
		t.Fatalf("recovery validation: %s", sqliteTestErrorChain(err))
	}
}

func TestAgentProviderStopRequestRejectsReplayAndCausalDivergence(t *testing.T) {
	system, _, _, request := seedSQLiteAgentProviderStopRequest(t, "provider-stop-divergence")
	recordSQLiteAgentProviderStopRequest(t, system.repository, request)
	divergent := ports.CloneAgentProviderStopRequest(request)
	divergent.Body = []byte(`{"schema":"provider.stop.divergent.v1"}`)
	divergent.BodySHA256 = ports.AgentProviderRequestBodySHA256(divergent.Body)
	if _, err := system.repository.RecordAgentProviderStopRequest(context.Background(), divergent); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent replay error=%s", sqliteTestErrorChain(err))
	}

	otherExecution, _ := goal.NewExecutionRef("execution:provider-stop-crossed")
	mutations := map[string]func(*ports.AgentProviderStopRequest){
		"execution":    func(v *ports.AgentProviderStopRequest) { v.Key.ExecutionRef = otherExecution },
		"launch fence": func(v *ports.AgentProviderStopRequest) { v.Key.LaunchActionFence++ },
		"stop fence":   func(v *ports.AgentProviderStopRequest) { v.Key.StopActionFence++ },
		"attempt":      func(v *ports.AgentProviderStopRequest) { v.StopEffectAttemptRef += ":crossed" },
		"provider":     func(v *ports.AgentProviderStopRequest) { v.ProviderRef = "provider:other" },
		"idempotency":  func(v *ports.AgentProviderStopRequest) { v.IdempotencyKey += ":crossed" },
		"target":       func(v *ports.AgentProviderStopRequest) { v.TargetRef = "runtime:other" },
		"revision":     func(v *ports.AgentProviderStopRequest) { v.ExpectedRevision++ },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			fixture, _, _, candidate := seedSQLiteAgentProviderStopRequest(t, "provider-stop-crossed-"+name)
			mutate(&candidate)
			if _, err := fixture.repository.RecordAgentProviderStopRequest(context.Background(), candidate); !application.IsStateError(err, application.StateConflict) {
				t.Fatalf("crossed %s error=%s", name, sqliteTestErrorChain(err))
			}
		})
	}
}

func TestAgentProviderStopRequestRejectsFirstWriteAfterEffectReceipt(t *testing.T) {
	system, claim, attempt, request := seedSQLiteAgentProviderStopRequest(t, "provider-stop-receipted")
	receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusStopped, system.clock.Now())
	transaction, err := beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, insertEffectReceipt(context.Background(), transaction, claim, receipt, receipt.ConfirmedAt))
	sqliteTestNoError(t, commit(transaction))
	if _, err := system.repository.RecordAgentProviderStopRequest(context.Background(), request); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("post-receipt request error=%s", sqliteTestErrorChain(err))
	}
}

func TestAgentProviderStopRequestHonorsPreCanceledContext(t *testing.T) {
	system, _, _, request := seedSQLiteAgentProviderStopRequest(t, "provider-stop-canceled")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := system.repository.RecordAgentProviderStopRequest(ctx, request); !errors.Is(err, context.Canceled) {
		t.Fatalf("Record canceled error=%s", sqliteTestErrorChain(err))
	}
	if got, found, err := system.repository.ResolveAgentProviderStopRequest(context.Background(), request.Key); err != nil || found || !reflect.DeepEqual(got, ports.AgentProviderStopRequest{}) {
		t.Fatalf("Record canceled persisted=%+v found=%t err=%s", got, found, sqliteTestErrorChain(err))
	}
	if _, _, err := system.repository.ResolveAgentProviderStopRequest(ctx, request.Key); !errors.Is(err, context.Canceled) {
		t.Fatalf("Resolve canceled error=%s", sqliteTestErrorChain(err))
	}
}

func TestAgentProviderStopRequestConcurrentExactAndDivergentRecord(t *testing.T) {
	for name, mutate := range map[string]func(*ports.AgentProviderStopRequest){
		"exact": func(*ports.AgentProviderStopRequest) {},
		"divergent": func(request *ports.AgentProviderStopRequest) {
			request.Body = []byte(`{"schema":"provider.stop.concurrent.v1"}`)
			request.BodySHA256 = ports.AgentProviderRequestBodySHA256(request.Body)
		},
	} {
		t.Run(name, func(t *testing.T) {
			system, _, _, request := seedSQLiteAgentProviderStopRequest(t, "provider-stop-concurrent-"+name)
			second := ports.CloneAgentProviderStopRequest(request)
			mutate(&second)
			start := make(chan struct{})
			results := make(chan error, 2)
			for _, candidate := range []ports.AgentProviderStopRequest{request, second} {
				go func(candidate ports.AgentProviderStopRequest) {
					<-start
					_, err := system.repository.RecordAgentProviderStopRequest(context.Background(), candidate)
					results <- err
				}(candidate)
			}
			close(start)
			var successes, conflicts int
			for range 2 {
				switch err := <-results; {
				case err == nil:
					successes++
				case application.IsStateError(err, application.StateConflict):
					conflicts++
				default:
					t.Fatalf("concurrent Record error=%s", sqliteTestErrorChain(err))
				}
			}
			wantSuccesses, wantConflicts := 2, 0
			if name == "divergent" {
				wantSuccesses, wantConflicts = 1, 1
			}
			if successes != wantSuccesses || conflicts != wantConflicts {
				t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
			}
		})
	}
}

func TestAgentProviderStopRequestRecoveryAcceptsTerminalReceiptAndRejectsAcceptedReceipt(t *testing.T) {
	system, claim, attempt, request := seedSQLiteAgentProviderStopRequest(t, "provider-stop-terminal-receipt")
	recordSQLiteAgentProviderStopRequest(t, system.repository, request)
	receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusStopped, system.clock.Now())
	transaction, err := beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, insertEffectReceipt(context.Background(), transaction, claim, receipt, receipt.ConfirmedAt))
	sqliteTestNoError(t, commit(transaction))
	read, err := beginReadTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	if err := validateRecoveryV38AgentProviderStopRequests(context.Background(), read); err != nil {
		t.Fatalf("terminal receipt rejected: %s", sqliteTestErrorChain(err))
	}
	sqliteTestNoError(t, read.Rollback())
	_, err = system.repository.db.Exec(`DROP TRIGGER effect_receipts_immutable_update`)
	sqliteTestNoError(t, err)
	_, err = system.repository.db.Exec(`UPDATE effect_receipts SET status='accepted' WHERE attempt_ref=?`, attempt.Ref)
	sqliteTestNoError(t, err)
	read, err = beginReadTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	defer read.Rollback()
	if err := validateRecoveryV38AgentProviderStopRequests(context.Background(), read); err == nil ||
		!recoveryErrorContains(err, recoveryV38AgentProviderStopRequestInvalid) {
		t.Fatalf("accepted stop receipt recovered: %s", sqliteTestErrorChain(err))
	}
}

func TestAgentProviderStopRequestRecoveryRejectsBodyAndBindingCorruption(t *testing.T) {
	for name, corrupt := range map[string]func(*testing.T, *sqliteV15System, ports.AgentProviderStopRequest){
		"body": func(t *testing.T, system *sqliteV15System, request ports.AgentProviderStopRequest) {
			_, err := system.repository.db.Exec(`UPDATE agent_provider_stop_requests SET body=?
WHERE execution_ref=? AND launch_action_fence=? AND stop_action_fence=?`, []byte(`{"corrupt":true}`),
				request.Key.ExecutionRef.String(), request.Key.LaunchActionFence, request.Key.StopActionFence)
			sqliteTestNoError(t, err)
		},
		"binding": func(t *testing.T, system *sqliteV15System, request ports.AgentProviderStopRequest) {
			_, err := system.repository.db.Exec(`UPDATE agent_provider_stop_requests SET expected_revision=expected_revision+1
WHERE execution_ref=? AND launch_action_fence=? AND stop_action_fence=?`,
				request.Key.ExecutionRef.String(), request.Key.LaunchActionFence, request.Key.StopActionFence)
			sqliteTestNoError(t, err)
		},
	} {
		t.Run(name, func(t *testing.T) {
			system, _, _, request := seedSQLiteAgentProviderStopRequest(t, "provider-stop-corrupt-"+name)
			recordSQLiteAgentProviderStopRequest(t, system.repository, request)
			_, err := system.repository.db.Exec(`DROP TRIGGER agent_provider_stop_requests_immutable_update`)
			sqliteTestNoError(t, err)
			corrupt(t, system, request)
			transaction, err := beginReadTransaction(context.Background(), system.repository)
			sqliteTestNoError(t, err)
			defer transaction.Rollback()
			err = validateRecoveryV38AgentProviderStopRequests(context.Background(), transaction)
			if err == nil || !recoveryErrorContains(err, recoveryV38AgentProviderStopRequestInvalid) {
				t.Fatalf("corruption accepted: %s", sqliteTestErrorChain(err))
			}
		})
	}
}

func TestV38AgentProviderStopRequestMigrationHasStrictCausalShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-provider-stop-v37.db")
	database := agentCapacityDatabase(t, path, recoverySchemaV38AgentProviderRequest)
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = repository.Close() })

	var version, strict int
	var name, tableSQL, triggerSQL string
	sqliteTestNoError(t, repository.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT name FROM schema_migrations WHERE version=?`,
		recoverySchemaV38AgentProviderStop).Scan(&name))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT strict FROM pragma_table_list
WHERE name='agent_provider_stop_requests'`).Scan(&strict))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT sql FROM sqlite_schema
WHERE type='table' AND name='agent_provider_stop_requests'`).Scan(&tableSQL))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='agent_provider_stop_requests_causal_insert'`).Scan(&triggerSQL))
	if version != recoverySchemaLatest || name != "038_agent_provider_stop_requests.sql" || strict != 1 ||
		!strings.Contains(tableSQL, "PRIMARY KEY(execution_ref,launch_action_fence,stop_action_fence)") ||
		!strings.Contains(tableSQL, "FOREIGN KEY(stop_effect_attempt_ref,execution_ref,stop_action_fence)") ||
		!strings.Contains(triggerSQL, "intent.kind='agent_stop'") ||
		!strings.Contains(triggerSQL, "launch.launch_binding_revision=NEW.expected_revision") {
		t.Fatalf("version=%d name=%q strict=%d table=%s trigger=%s", version, name, strict, tableSQL, triggerSQL)
	}
}

func seedSQLiteAgentProviderStopRequest(
	t *testing.T,
	suffix string,
) (*sqliteV15System, application.ActionClaim, application.EffectAttempt, ports.AgentProviderStopRequest) {
	t.Helper()
	system, firstClaim, launchAttempt := seedV27AmbiguousEffectAttempt(t, suffix)
	launch := sqliteAgentProviderRequest(launchAttempt, ports.AgentProviderRequestLaunch)
	launch.ProviderRef, launch.IdempotencyKey = "provider:codex", launchAttempt.IdempotencyKey
	recordSQLiteAgentProviderRequest(t, system.repository, launch)
	targetRef := "external:" + launch.Key.ExecutionRef.String()
	bindSQLiteAgentProviderLaunch(t, system.repository, launch.Key, targetRef, 3)
	until := firstClaim.LeaseUntil
	if firstClaim.EffectApproval.ExpiresAt.After(until) {
		until = firstClaim.EffectApproval.ExpiresAt
	}
	system.clock.Advance(until.Sub(system.clock.Now()) + time.Nanosecond)
	recoveryClaim := claimSQLiteV28Recovery(t, system, "claim:provider-stop-launch-recovery:"+suffix, launchAttempt.Ref)
	accepted := sqliteV28RecoveryLaunchAcceptedState(t, system, recoveryClaim, launchAttempt)
	accepted.EffectReceipt.ExternalRef = targetRef
	accepted.Execution.ExternalRef = targetRef
	sqliteTestNoError(t, system.repository.RecordLaunchAccepted(context.Background(), accepted))

	running, err := system.repository.GetGoal(context.Background(), accepted.Execution.GoalRef)
	sqliteTestNoError(t, err)
	item, found := running.Goal.WorkItem(accepted.Execution.WorkItemRef)
	if !found {
		t.Fatal("running WorkItem missing")
	}
	control, err := system.orchestrator.Control(context.Background(), system.access, sqliteExactStopRequest(
		running, item, accepted.Execution, "control:provider-stop:"+suffix, ports.AgentStopForced,
	))
	if err != nil || !control.Created {
		t.Fatalf("Control()=%+v err=%v", control, err)
	}
	controlled, err := system.repository.GetGoal(context.Background(), accepted.Execution.GoalRef)
	sqliteTestNoError(t, err)
	var stopIntent application.EffectIntent
	for _, intent := range controlled.EffectIntents {
		if intent.ActionRef == "action:stop:"+control.Control.Ref+":"+accepted.Execution.Ref.String() {
			stopIntent = intent
		}
	}
	decision, err := system.orchestrator.DecideEffect(context.Background(), system.access, application.DecideEffectRequest{
		RequestRef: "approval:provider-stop:" + suffix, GoalRef: controlled.Goal.Ref(),
		IntentRef: stopIntent.Ref, ExpectedIntentDigest: stopIntent.Digest,
		Decision: application.EffectApproved, Reason: "authorize provider stop fixture",
	})
	if err != nil || !decision.Created {
		t.Fatalf("DecideEffect()=%+v intent=%+v err=%v", decision, stopIntent, err)
	}
	stopClaim := claimSQLiteAgentProviderStop(t, system, "claim:provider-stop:"+suffix)
	stopAttempt := sqliteV15Attempt(stopClaim, system.clock.Now())
	stored, created, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: stopClaim, Attempt: stopAttempt, OperationAt: system.clock.Now(),
	})
	if err != nil || !created || stored != stopAttempt {
		t.Fatalf("stop attempt=%+v created=%t err=%v", stored, created, err)
	}
	body := []byte(`{"schema":"provider.stop.v1"}`)
	request := ports.AgentProviderStopRequest{
		Key: ports.AgentProviderStopRequestKey{
			ExecutionRef: accepted.Execution.Ref, LaunchActionFence: launchAttempt.ActionFence,
			StopActionFence: stopAttempt.ActionFence,
		},
		StopEffectAttemptRef: stopAttempt.Ref, ProviderRef: accepted.Execution.ProviderRef,
		IdempotencyKey: stopAttempt.IdempotencyKey, TargetRef: targetRef, ExpectedRevision: 3,
		Body: body, BodySHA256: ports.AgentProviderRequestBodySHA256(body),
	}
	return system, stopClaim, stopAttempt, request
}

func claimSQLiteAgentProviderStop(
	t *testing.T,
	system *sqliteV15System,
	token string,
) application.ActionClaim {
	t.Helper()
	ctx := context.Background()
	transaction, err := beginTransaction(ctx, system.repository)
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	now := system.clock.Now().UTC()
	candidates, err := readClaimCandidateWindow(ctx, transaction, now, false, false, nil)
	sqliteTestNoError(t, err)
	filtered := candidates[:0]
	for _, candidate := range candidates {
		if candidate.action.Kind == application.ActionStopAgent {
			filtered = append(filtered, candidate)
		}
	}
	requirements, err := readAgentRequirementsBatch(ctx, transaction, filtered)
	sqliteTestNoError(t, err)
	selected, found, err := selectClaimCandidate(
		ctx, transaction, filtered, requirements, sqliteTestCapabilities(), system.capacidad, false, now,
	)
	if err != nil || !found {
		t.Fatalf("select stop candidate found=%t err=%v candidates=%d", found, err, len(filtered))
	}
	claim, err := claimSelectedCandidate(ctx, transaction, application.ClaimRequest{
		WorkerRef: "worker:provider-stop", Token: token, LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
		CapacityCandidates: system.capacidad,
	}, selected, now, now.Add(time.Minute))
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, commit(transaction))
	return claim
}

func recordSQLiteAgentProviderStopRequest(
	t *testing.T,
	repository *Repository,
	request ports.AgentProviderStopRequest,
) ports.AgentProviderStopRequest {
	t.Helper()
	stored, err := repository.RecordAgentProviderStopRequest(context.Background(), request)
	if err != nil {
		t.Fatalf("RecordAgentProviderStopRequest: %s", sqliteTestErrorChain(err))
	}
	return stored
}
