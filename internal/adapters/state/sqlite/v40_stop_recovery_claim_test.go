package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

const v39AgentProviderStopKeyMigrationSHA256 = "sha256:8954122793c675d15aaf84903f67f2cc4c12134d87329f51b4d97acbdbc8d2e0"

func TestV40StopRecoveryClaimSurvivesRestartAndReclaimsSameAttempt(t *testing.T) {
	system, original, attempt, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-recovery")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	before := v28RecoveryLedgerCounts(t, system.repository.db)
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)

	first := claimV40StopRecovery(t, system, "claim:v40-stop-recovery:first")
	assertV40StopRecoveryClaim(t, system, first, original, attempt)
	if after := v28RecoveryLedgerCounts(t, system.repository.db); after != before {
		t.Fatalf("first recovery mutated physical ledgers before=%+v after=%+v", before, after)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("pending Stop recovery failed recovery validation: %s", sqliteTestErrorChain(err))
	}

	system.clock.Advance(first.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	second := claimV40StopRecovery(t, system, "claim:v40-stop-recovery:second")
	assertV40StopRecoveryClaim(t, system, second, original, attempt)
	if second.Fence <= first.Fence || second.DeliveryAttempt != first.DeliveryAttempt+1 {
		t.Fatalf("reclaim did not advance fence/delivery first=%+v second=%+v", first, second)
	}
	if after := v28RecoveryLedgerCounts(t, system.repository.db); after != before {
		t.Fatalf("reclaim created physical authority before=%+v after=%+v", before, after)
	}
}

func TestV40RecoveryValidationAcceptsReleasedStopRecoveryClaim(t *testing.T) {
	system, original, attempt, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-released")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV40StopRecovery(t, system, "claim:v40-stop-released")
	if err := system.repository.ValidateAgentStopRecoveryClaim(context.Background(), claim); err != nil {
		t.Fatalf("current Stop recovery rejected: %s", sqliteTestErrorChain(err))
	}
	restarted := openSQLiteV15Repository(t, system.path, system.clock.Now)
	if err := restarted.ValidateAgentStopRecoveryClaim(context.Background(), claim); err != nil {
		t.Fatalf("restart rejected current Stop recovery: %s", sqliteTestErrorChain(err))
	}
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found {
		t.Fatal("Stop recovery execution missing")
	}
	availableAt := claim.LeaseUntil.Add(time.Minute)
	sqliteTestNoError(t, system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: execution, AvailableAt: availableAt,
		OperationAt: system.clock.Now(), ErrorCode: "agent.stop_reconciliation_pending",
	}))
	if err := restarted.ValidateAgentStopRecoveryClaim(context.Background(), claim); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("released Stop claim remained current: %s", sqliteTestErrorChain(err))
	}
	requireV40StopRecoveryValidation(t, system, true)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("released Stop recovery failed full validation: %s", sqliteTestErrorChain(err))
	}
	var recoveryRef string
	var claimed int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT recovery_effect_attempt_ref,
claim_token IS NOT NULL FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(&recoveryRef, &claimed))
	if recoveryRef != attempt.Ref || claimed != 0 {
		t.Fatalf("released Stop recovery ref=%q claimed=%d", recoveryRef, claimed)
	}
	system.clock.Advance(availableAt.Sub(system.clock.Now()))
	next := claimV40StopRecovery(t, system, "claim:v40-stop-released-next")
	if next.RecoveryEffectAttemptRef != attempt.Ref || next.Fence <= claim.Fence ||
		next.DeliveryAttempt != claim.DeliveryAttempt+1 {
		t.Fatalf("reclaimed Stop recovery next=%+v previous=%+v", next, claim)
	}
	if err := restarted.ValidateAgentStopRecoveryClaim(context.Background(), next); err != nil {
		t.Fatalf("reclaimed Stop recovery rejected: %s", sqliteTestErrorChain(err))
	}
}

func TestV40ValidateAgentStopRecoveryClaimRejectsCrossedCASWithoutMutation(t *testing.T) {
	system, original, _, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-validate-crossed")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV40StopRecovery(t, system, "claim:v40-stop-validate-crossed")
	before := readV28RecoveryOutbox(t, system.repository.db, claim.Action.Ref)
	mutations := map[string]func(*application.ActionClaim){
		"token":       func(value *application.ActionClaim) { value.Token += ":crossed" },
		"worker":      func(value *application.ActionClaim) { value.WorkerRef += ":crossed" },
		"fence":       func(value *application.ActionClaim) { value.Fence++ },
		"lease":       func(value *application.ActionClaim) { value.LeaseUntil = value.LeaseUntil.Add(time.Nanosecond) },
		"attempt ref": func(value *application.ActionClaim) { value.RecoveryEffectAttemptRef += ":crossed" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := claim
			mutate(&candidate)
			if err := system.repository.ValidateAgentStopRecoveryClaim(context.Background(), candidate); !application.IsStateError(err, application.StateConflict) {
				t.Fatalf("crossed %s error=%s", name, sqliteTestErrorChain(err))
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := system.repository.ValidateAgentStopRecoveryClaim(ctx, claim); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled validation error=%s", sqliteTestErrorChain(err))
	}
	if after := readV28RecoveryOutbox(t, system.repository.db, claim.Action.Ref); after != before {
		t.Fatalf("crossed validation mutated outbox before=%+v after=%+v", before, after)
	}
}

func TestV40StopRecoveryRequeueDoesNotEnableTerminalMutation(t *testing.T) {
	system, original, _, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-terminal-gate")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV40StopRecovery(t, system, "claim:v40-stop-terminal-gate")
	before := readV28RecoveryOutbox(t, system.repository.db, claim.Action.Ref)
	operationAt := system.clock.Now().UTC()
	err := system.repository.QuarantineAction(context.Background(), application.ActionQuarantinedState{
		Claim: claim, ErrorCode: "application.effect_unknown_applied", OperationAt: operationAt,
		Event: application.EventRecord{
			Ref: "event:v40-stop-terminal-gate", Kind: "action.quarantined",
			GoalRef: claim.Action.GoalRef, WorkItemRef: claim.Action.WorkItemRef,
			ExecutionRef: claim.Action.ExecutionRef, OccurredAt: operationAt,
		},
	})
	if !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("terminal Stop recovery mutation error=%s", sqliteTestErrorChain(err))
	}
	if after := readV28RecoveryOutbox(t, system.repository.db, claim.Action.Ref); after != before {
		t.Fatalf("terminal gate mutated outbox before=%+v after=%+v", before, after)
	}
	var consumptions int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT COUNT(*) FROM action_consumption_receipts
WHERE action_ref=?`, claim.Action.Ref).Scan(&consumptions))
	if consumptions != 0 {
		t.Fatalf("terminal gate consumptions=%d", consumptions)
	}
}

func TestV40RecoveryValidationRejectsTerminalStopRecoveryReceipt(t *testing.T) {
	system, original, attempt, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-terminal-receipt")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV40StopRecovery(t, system, "claim:v40-stop-terminal-receipt")
	completedAt := system.clock.Now().UTC()
	_, err := system.repository.db.Exec(`UPDATE outbox SET completed_at=? WHERE ref=?`,
		requiredTime(completedAt), claim.Action.Ref)
	sqliteTestNoError(t, err)
	_, err = system.repository.db.Exec(`INSERT INTO effect_receipts(
ref,intent_ref,intent_digest,approval_ref,attempt_ref,project_ref,goal_ref,work_item_ref,
execution_ref,plan_generation,app_spec_generation,spec_hash,actor_ref,action_ref,action_fence,
idempotency_key,external_ref,status,usage_tokens,usage_money_micros,usage_currency,
usage_active_time_ns,usage_process_slots,usage_disk_bytes,usage_known,usage_quality,confirmed_at)
SELECT ?,intent_ref,intent_digest,approval_ref,ref,project_ref,goal_ref,work_item_ref,
execution_ref,plan_generation,app_spec_generation,spec_hash,actor_ref,action_ref,action_fence,
idempotency_key,?,'stopped',0,0,'',0,0,0,0,'unknown',?
FROM effect_attempts WHERE ref=?`,
		"effect-receipt:v40-stop-terminal", "external:v40-stop-terminal",
		requiredTime(attempt.ClaimLeaseUntil.Add(-time.Nanosecond)), attempt.Ref)
	sqliteTestNoError(t, err)
	requireV40StopRecoveryValidation(t, system, false)
}

func TestV40RecoveryValidationRejectsConsumedTerminalStopRecovery(t *testing.T) {
	system, original, _, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-terminal-consumed")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV40StopRecovery(t, system, "claim:v40-stop-terminal-consumed")
	completedAt := system.clock.Now().UTC()
	_, err := system.repository.db.Exec(`UPDATE outbox
SET completed_at=?,quarantined_at=?,last_error_code='agent.stop_recovery_terminal_forbidden'
WHERE ref=?`, requiredTime(completedAt), requiredTime(completedAt), claim.Action.Ref)
	sqliteTestNoError(t, err)
	_, err = system.repository.db.Exec(`INSERT INTO action_consumption_receipts(
action_ref,governance_version,kind,goal_ref,work_item_ref,execution_ref,change_ref,
plan_generation,work_item_generation,fence,delivery_attempt,claim_token,worker_ref,
outcome,error_code,consumed_at)
SELECT ref,governance_version,kind,goal_ref,work_item_ref,execution_ref,change_ref,
plan_generation,work_item_generation,fence,delivery_attempt,claim_token,claimed_by,
'quarantined',last_error_code,completed_at FROM outbox WHERE ref=?`, claim.Action.Ref)
	sqliteTestNoError(t, err)
	requireV40StopRecoveryValidation(t, system, false)
}

func TestV40StopRecoveryClaimRejectsRelatedReceiptWithoutSecondAttempt(t *testing.T) {
	system, original, attempt, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-receipt")
	confirmedAt := attempt.ClaimLeaseUntil.Add(-time.Nanosecond)
	receipt := sqliteV15EffectReceipt(original, attempt, application.EffectStatusStopped, confirmedAt)
	transaction, err := beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, insertEffectReceipt(context.Background(), transaction, original, receipt, confirmedAt))
	sqliteTestNoError(t, commit(transaction))
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	before := v28RecoveryLedgerCounts(t, system.repository.db)
	outboxBefore := readV28RecoveryOutbox(t, system.repository.db, original.Action.Ref)

	claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v40-stop-receipt", Token: "claim:v40-stop-receipt",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
		BudgetPolicy: system.policy, ExcludeLaunch: true,
	})
	if !application.IsStateError(err, application.StateConflict) || found || claim != (application.ActionClaim{}) {
		t.Fatalf("receipted Stop recovery claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
	if after := v28RecoveryLedgerCounts(t, system.repository.db); after != before {
		t.Fatalf("receipted recovery mutated physical ledgers before=%+v after=%+v", before, after)
	}
	if after := readV28RecoveryOutbox(t, system.repository.db, original.Action.Ref); after != outboxBefore {
		t.Fatalf("receipted recovery mutated outbox before=%+v after=%+v", outboxBefore, after)
	}
}

func TestV40StopRecoveryClaimHasSingleConcurrentWinner(t *testing.T) {
	system, original, attempt, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-concurrent")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	peer := openSQLiteV15Repository(t, system.path, system.clock.Now)
	before := v28RecoveryLedgerCounts(t, system.repository.db)
	type outcome struct {
		claim application.ActionClaim
		found bool
		err   error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for index, repository := range []*Repository{system.repository, peer} {
		workers.Add(1)
		go func(index int, repository *Repository) {
			defer workers.Done()
			<-start
			claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
				WorkerRef:     "worker:v40-concurrent:" + string(rune('a'+index)),
				Token:         "claim:v40-concurrent:" + string(rune('a'+index)),
				LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
				BudgetPolicy: system.policy, ExcludeLaunch: true,
			})
			results <- outcome{claim: claim, found: found, err: err}
		}(index, repository)
	}
	close(start)
	workers.Wait()
	close(results)
	winners := 0
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent recovery error=%s", sqliteTestErrorChain(result.err))
		}
		if !result.found {
			continue
		}
		winners++
		if result.claim.Disposition != application.ActionClaimDispositionRecoverEffect ||
			result.claim.RecoveryEffectAttemptRef != attempt.Ref || result.claim.Fence <= attempt.ActionFence {
			t.Fatalf("concurrent winner=%+v attempt=%+v", result.claim, attempt)
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent recovery winners=%d", winners)
	}
	if after := v28RecoveryLedgerCounts(t, system.repository.db); after != before {
		t.Fatalf("concurrent recovery mutated physical ledgers before=%+v after=%+v", before, after)
	}
}

func TestV40StopRecoveryParksRevokedAuthorityWithoutClaimOrAttempt(t *testing.T) {
	system, original, _, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-revoked")
	owner := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
	admin := testPrincipal(t, "principal:v40-stop-admin", "actor:v40-stop-admin", identity.PrincipalKindHuman)
	grantTestMembership(t, system.repository, owner, admin, system.project,
		identity.RoleProjectOwner, "membership:v40-stop-admin", system.clock.Now())
	ownerMembership, err := system.repository.Membership(context.Background(), owner.Ref, system.project)
	sqliteTestNoError(t, err)
	authorization := authorizeTest(t, system.repository, admin, system.project,
		identity.PermissionProjectMembershipManage, owner.Ref.String(),
		"authorization:v40-stop-revoke-owner", system.clock.Now())
	revoke := testRevokeRequest(t, "membership:v40-stop-revoke-owner", admin, owner.Ref,
		system.project, ownerMembership.Revision(), system.clock.Now())
	_, _, changed, err := system.repository.RevokeMembership(context.Background(), application.MembershipRevokeState{
		AuthorizationReceipt: authorization, Request: revoke,
	})
	if err != nil || !changed {
		t.Fatalf("revoke owner changed=%t err=%v", changed, err)
	}
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	before := v28RecoveryLedgerCounts(t, system.repository.db)
	claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v40-stop-revoked", Token: "claim:v40-stop-revoked",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
		BudgetPolicy: system.policy, ExcludeLaunch: true,
	})
	if err != nil || found || claim != (application.ActionClaim{}) {
		t.Fatalf("revoked Stop recovery claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
	var code string
	var claimed, recoveryRef int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT last_error_code,
claim_token IS NOT NULL,recovery_effect_attempt_ref IS NOT NULL FROM outbox WHERE ref=?`,
		original.Action.Ref).Scan(&code, &claimed, &recoveryRef))
	if code != "governance.effect_approval_required" || claimed != 0 || recoveryRef != 0 {
		t.Fatalf("revoked recovery code=%q claimed=%d recovery_ref=%d", code, claimed, recoveryRef)
	}
	if after := v28RecoveryLedgerCounts(t, system.repository.db); after != before {
		t.Fatalf("revoked recovery mutated physical ledgers before=%+v after=%+v", before, after)
	}
}

func TestV40RecoveryValidationRejectsCrossedStopAttempt(t *testing.T) {
	system, original, _, _ := seedSQLiteAgentProviderStopRequest(t, "v40-stop-crossed")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV40StopRecovery(t, system, "claim:v40-stop-crossed")
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	var launchAttemptRef string
	for _, attempt := range record.EffectAttempts {
		if attempt.ActionRef != claim.Action.Ref {
			launchAttemptRef = attempt.Ref
			break
		}
	}
	if launchAttemptRef == "" {
		t.Fatal("launch attempt fixture missing")
	}
	_, err = system.repository.db.Exec(`DROP TRIGGER outbox_recovery_effect_claim_guard`)
	sqliteTestNoError(t, err)
	_, err = system.repository.db.Exec(`UPDATE outbox SET recovery_effect_attempt_ref=? WHERE ref=?`,
		launchAttemptRef, claim.Action.Ref)
	sqliteTestNoError(t, err)
	transaction, err := system.repository.db.BeginTx(context.Background(), nil)
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	if err := validateRecoveryV40EffectRecoveryClaim(context.Background(), transaction); err == nil ||
		!strings.Contains(sqliteTestErrorChain(err), "sqlite.recovery_v28_effect_recovery_claim_invalid") {
		t.Fatalf("crossed Stop recovery passed validation: %s", sqliteTestErrorChain(err))
	}
}

func TestV40MigrationPreservesV39ChecksumAndProgressivelyWidensClaimGuard(t *testing.T) {
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	if len(migrations) != recoverySchemaLatest {
		t.Fatalf("migration count=%d latest=%d", len(migrations), recoverySchemaLatest)
	}
	v39 := migrations[recoverySchemaV38AgentProviderStopKey-1]
	v40 := migrations[recoverySchemaV38StopRecoveryClaim-1]
	if v39.name != "039_agent_provider_stop_physical_idempotency.sql" ||
		v39.checksum != v39AgentProviderStopKeyMigrationSHA256 ||
		v40.name != "040_stop_recovery_claim.sql" ||
		!strings.Contains(v40.sql, "NEW.kind='stop_agent' AND intent.kind='agent_stop'") {
		t.Fatalf("migration V39=%q/%q V40=%q", v39.name, v39.checksum, v40.name)
	}
	prefixV39, err := recoveryMigrationPrefix(migrations, recoverySchemaV38AgentProviderStopKey)
	sqliteTestNoError(t, err)
	prefixV40, err := recoveryMigrationPrefix(migrations, recoverySchemaV38StopRecoveryClaim)
	sqliteTestNoError(t, err)
	if len(prefixV39) != 39 || len(prefixV40) != 40 {
		t.Fatalf("migration prefixes V39=%d V40=%d", len(prefixV39), len(prefixV40))
	}

	path := filepath.Join(t.TempDir(), "canonical-v39.db")
	database := agentCapacityDatabase(t, path, recoverySchemaV38AgentProviderStopKey)
	var triggerV39 string
	sqliteTestNoError(t, database.QueryRow(`SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='outbox_recovery_effect_claim_guard'`).Scan(&triggerV39))
	if strings.Contains(triggerV39, "NEW.kind='stop_agent'") {
		t.Fatalf("canonical V39 unexpectedly allows Stop recovery: %s", triggerV39)
	}
	sqliteTestNoError(t, database.Close())
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: time.Now,
	})
	if err != nil {
		t.Fatalf("upgrade V39: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })
	var version, receipt int
	var triggerV40 string
	sqliteTestNoError(t, repository.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		recoverySchemaV38StopRecoveryClaim).Scan(&receipt))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='outbox_recovery_effect_claim_guard'`).Scan(&triggerV40))
	if version != recoverySchemaLatest || receipt != 1 ||
		!strings.Contains(triggerV40, "NEW.kind='stop_agent'") {
		t.Fatalf("upgrade version=%d receipt=%d trigger=%s", version, receipt, triggerV40)
	}
}

func claimV40StopRecovery(
	t *testing.T,
	system *sqliteV15System,
	token string,
) application.ActionClaim {
	t.Helper()
	claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v40-stop-recovery", Token: token,
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
		BudgetPolicy: system.policy, ExcludeLaunch: true,
	})
	if err != nil || !found || claim.Action.Kind != application.ActionStopAgent ||
		claim.Disposition != application.ActionClaimDispositionRecoverEffect {
		t.Fatalf("Stop recovery claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
	return claim
}

func assertV40StopRecoveryClaim(
	t *testing.T,
	system *sqliteV15System,
	claim, original application.ActionClaim,
	attempt application.EffectAttempt,
) {
	t.Helper()
	if claim.RecoveryEffectAttemptRef != attempt.Ref || claim.Fence <= attempt.ActionFence ||
		claim.EffectApproval != original.EffectApproval || claim.BudgetReservationRef != "" ||
		claim.BudgetReservation != (application.ActionClaim{}).BudgetReservation ||
		claim.CapacityReservation != (application.ActionClaim{}).CapacityReservation ||
		claim.ReferenciaColocacion.String() != "" {
		t.Fatalf("Stop recovery authority=%+v original=%+v attempt=%+v", claim, original, attempt)
	}
	var durableRef string
	var durableFence int64
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT recovery_effect_attempt_ref,fence
FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(&durableRef, &durableFence))
	if durableRef != attempt.Ref || durableFence != int64(claim.Fence) {
		t.Fatalf("durable Stop recovery ref=%q fence=%d claim=%+v", durableRef, durableFence, claim)
	}
}

func requireV40StopRecoveryValidation(t *testing.T, system *sqliteV15System, valid bool) {
	t.Helper()
	transaction, err := system.repository.db.BeginTx(context.Background(), nil)
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	err = validateRecoveryV40EffectRecoveryClaim(context.Background(), transaction)
	if valid && err != nil {
		t.Fatalf("valid Stop recovery rejected: %s", sqliteTestErrorChain(err))
	}
	if !valid && (err == nil ||
		!strings.Contains(sqliteTestErrorChain(err), "sqlite.recovery_v28_effect_recovery_claim_invalid")) {
		t.Fatalf("terminal Stop recovery passed validation: %s", sqliteTestErrorChain(err))
	}
}
