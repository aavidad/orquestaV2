package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

func TestV28ClaimNextActionAutomaticallyRecoversExactHistoricalLaunch(t *testing.T) {
	system, first, attempt := seedV27AmbiguousEffectAttempt(t, "v28-automatic-recovery")
	until := first.LeaseUntil
	if first.EffectApproval.ExpiresAt.After(until) {
		until = first.EffectApproval.ExpiresAt
	}
	system.clock.Advance(until.Sub(system.clock.Now()) + time.Nanosecond)
	before := v28RecoveryLedgerCounts(t, system.repository.db)

	claim, found, err := system.repository.ClaimNextAction(
		context.Background(), application.ClaimRequest{
			WorkerRef: "worker:v28-recovery", Token: "claim:v28-automatic-recovery",
			LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
			BudgetPolicy: system.policy, ExcludeLaunch: true,
		},
	)
	if err != nil || !found {
		t.Fatalf("automatic recovery found=%t err=%s", found, sqliteTestErrorChain(err))
	}
	if claim.Disposition != application.ActionClaimDispositionRecoverEffect ||
		claim.RecoveryEffectAttemptRef != attempt.Ref || claim.Fence <= attempt.ActionFence ||
		claim.EffectApproval != first.EffectApproval ||
		claim.BudgetReservation != first.BudgetReservation ||
		claim.CapacityReservation != first.CapacityReservation ||
		claim.ReferenciaColocacion != first.ReferenciaColocacion {
		t.Fatalf("recovery claim=%+v first=%+v attempt=%+v", claim, first, attempt)
	}
	after := v28RecoveryLedgerCounts(t, system.repository.db)
	if before != after {
		t.Fatalf("recovery created physical authority before=%+v after=%+v", before, after)
	}
	var durableRef string
	var durableFence int64
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT recovery_effect_attempt_ref,fence FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(&durableRef, &durableFence))
	if durableRef != attempt.Ref || durableFence != int64(claim.Fence) {
		t.Fatalf("durable recovery ref=%q fence=%d claim=%+v", durableRef, durableFence, claim)
	}
}

func TestV28ExcludeLaunchDoesNotAdmitResolvedHistoricalAttempt(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:v28-resolved-excluded")
	system.external.launchErr = sqliteV15DefinitelyUnapplied{}
	if result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v28-resolved-first",
	); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("first definitely-unapplied launch result=%+v err=%v", result, err)
	}
	system.external.mu.Lock()
	system.external.launchErr = nil
	system.external.mu.Unlock()
	system.clock.Advance(2 * time.Second)

	request := application.ClaimRequest{
		WorkerRef: "worker:v28-resolved", Token: "claim:v28-resolved-excluded",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
		BudgetPolicy: system.policy, ExcludeLaunch: true,
	}
	claim, found, err := system.repository.ClaimNextAction(context.Background(), request)
	if err != nil || found || claim != (application.ActionClaim{}) {
		t.Fatalf("excluded resolved launch claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}

	request.Token, request.ExcludeLaunch = "claim:v28-resolved-normal", false
	request.CapacityCandidates = system.capacidad
	claim, found, err = system.repository.ClaimNextAction(context.Background(), request)
	if err != nil || !found || claim.Disposition != application.ActionClaimDispositionNormal ||
		claim.Action.Kind != application.ActionLaunchAgent || claim.RecoveryEffectAttemptRef != "" {
		t.Fatalf("resolved normal retry claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
}

func TestV28ClaimNextActionRejectsResolvedReceiptAndRollsBack(t *testing.T) {
	system, first, attempt := seedV27AmbiguousEffectAttempt(t, "v28-receipt-conflict")
	confirmedAt := attempt.ClaimLeaseUntil.Add(-time.Nanosecond)
	receipt := sqliteV15EffectReceipt(first, attempt, application.EffectStatusAccepted, confirmedAt)
	transaction, err := beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, insertEffectReceipt(
		context.Background(), transaction, first, receipt, confirmedAt,
	))
	sqliteTestNoError(t, commit(transaction))
	until := first.LeaseUntil
	if first.EffectApproval.ExpiresAt.After(until) {
		until = first.EffectApproval.ExpiresAt
	}
	system.clock.Advance(until.Sub(system.clock.Now()) + time.Nanosecond)
	countsBefore := v28RecoveryLedgerCounts(t, system.repository.db)
	outboxBefore := readV28RecoveryOutbox(t, system.repository.db, first.Action.Ref)

	claim, found, err := system.repository.ClaimNextAction(
		context.Background(), application.ClaimRequest{
			WorkerRef: "worker:v28-receipt-conflict", Token: "claim:v28-receipt-conflict",
			LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
			BudgetPolicy: system.policy, ExcludeLaunch: true,
		},
	)
	if !application.IsStateError(err, application.StateConflict) || found ||
		claim != (application.ActionClaim{}) {
		t.Fatalf("receipt footprint claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
	if countsAfter := v28RecoveryLedgerCounts(t, system.repository.db); countsAfter != countsBefore {
		t.Fatalf("receipt conflict mutated ledgers before=%+v after=%+v", countsBefore, countsAfter)
	}
	if outboxAfter := readV28RecoveryOutbox(t, system.repository.db, first.Action.Ref); outboxAfter != outboxBefore {
		t.Fatalf("receipt conflict mutated outbox before=%+v after=%+v", outboxBefore, outboxAfter)
	}
}

func TestV28RecoveryClaimRequiresDurableSchemaColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recovery-v27.db")
	database := openFastV18MigrationFixture(t, path)
	t.Cleanup(func() { _ = database.Close() })
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(
		context.Background(), database, migrations[:recoverySchemaV38AttemptLease],
	))
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	repository := &Repository{db: database, path: path, now: func() time.Time { return now }}
	transaction, err := beginTransaction(context.Background(), repository)
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	_, err = claimSelectedCandidate(
		context.Background(), transaction,
		application.ClaimRequest{WorkerRef: "worker:v27", Token: "claim:v27"},
		claimSelection{
			disposition: application.ActionClaimDispositionRecoverEffect,
			recovery:    &claimRecoverySeed{},
		},
		now, now.Add(time.Minute),
	)
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("legacy schema accepted recovery claim: %s", sqliteTestErrorChain(err))
	}
	var fences int
	sqliteTestNoError(t, transaction.QueryRow(`SELECT COUNT(*) FROM work_item_fences`).Scan(&fences))
	if fences != 0 {
		t.Fatalf("legacy recovery guard allocated fences=%d", fences)
	}
}

func TestV28RecoveryParksWhenOnlySelectedApproverWasRevoked(t *testing.T) {
	fixture := seedV28RevokedRecoveryApprover(t)
	claim, found, err := fixture.system.repository.ClaimNextAction(
		context.Background(), application.ClaimRequest{
			WorkerRef: "worker:v28-revoked-approver", Token: "claim:v28-revoked-approver",
			LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
			BudgetPolicy: fixture.system.policy,
		},
	)
	if err != nil || found || claim != (application.ActionClaim{}) {
		t.Fatalf("revoked approver claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
	owner := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
	ownerMembership, err := fixture.system.repository.Membership(context.Background(), owner.Ref, fixture.system.project)
	sqliteTestNoError(t, err)
	record, err := fixture.system.repository.GetGoal(context.Background(), fixture.goalRef)
	sqliteTestNoError(t, err)
	var code string
	var claimed int
	sqliteTestNoError(t, fixture.system.repository.db.QueryRow(`
SELECT last_error_code,claim_token IS NOT NULL FROM outbox WHERE ref=?`, fixture.claim.Action.Ref).Scan(&code, &claimed))
	if !ownerMembership.IsActive() || len(record.BudgetReservations) != 1 ||
		len(record.BudgetSettlements) != 0 || len(record.EffectAttempts) != 1 ||
		record.Executions[0].BudgetReservationRef != fixture.claim.BudgetReservationRef ||
		code != "governance.effect_approval_required" || claimed != 0 {
		t.Fatalf("owner_active=%t reservations=%d settlements=%d attempts=%d code=%q claimed=%d",
			ownerMembership.IsActive(), len(record.BudgetReservations), len(record.BudgetSettlements),
			len(record.EffectAttempts), code, claimed)
	}
}

type v28RevokedApproverFixture struct {
	system  *sqliteV15System
	claim   application.ActionClaim
	goalRef goal.GoalRef
}

func seedV28RevokedRecoveryApprover(t *testing.T) v28RevokedApproverFixture {
	t.Helper()
	system := newSQLiteV15System(t, 1)
	owner := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
	approver := testPrincipal(t, "principal:v28-approver", "actor:v28-approver", identity.PrincipalKindHuman)
	membership := grantTestMembership(t, system.repository, owner, approver, system.project,
		identity.RoleProjectAdmin, "membership:v28-approver", system.clock.Now())
	approverAccess, err := application.NewAccess(approver, system.project)
	sqliteTestNoError(t, err)
	created, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v28-revoked-approver", Statement: "sensitive recovery approval", Confirm: true,
		Plan: &application.PlanSpec{
			Phases:    []application.PhaseSpec{{Ref: "phase-instance:v28-revoked-approver", Key: "phase:v28-revoked-approver", TemplateRef: "phase-template:v28-revoked-approver"}},
			WorkItems: []application.WorkItemSpec{{Key: "work", Objective: "sensitive recovery", Phase: "phase:v28-revoked-approver", Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle, SecurityCriticality: governance.SecurityCriticalitySensitive, ReasoningEffort: governance.ReasoningEffortMedium}},
		},
	})
	sqliteTestNoError(t, err)
	intent := created.Record.EffectIntents[0]
	approved, err := system.orchestrator.DecideEffect(context.Background(), approverAccess, application.DecideEffectRequest{
		RequestRef: "approval:v28-revoked-approver", GoalRef: created.Record.Goal.Ref(),
		IntentRef: intent.Ref, ExpectedIntentDigest: intent.Digest,
		Decision: application.EffectApproved, Reason: "bounded independent approval",
	})
	if err != nil || !approved.Created {
		t.Fatalf("approve recovery intent=%+v err=%v", approved, err)
	}
	claim := claimSQLiteV15(t, system, "claim:v28-before-approver-revocation")
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	_, _, err = system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	})
	sqliteTestNoError(t, err)
	authorization := authorizeTest(t, system.repository, owner, system.project,
		identity.PermissionProjectMembershipManage, approver.Ref.String(),
		"authorization:v28-revoke-approver", system.clock.Now())
	revoke := testRevokeRequest(t, "membership:v28-revoke-approver", owner, approver.Ref,
		system.project, membership.Revision(), system.clock.Now())
	_, _, changed, err := system.repository.RevokeMembership(context.Background(), application.MembershipRevokeState{
		AuthorizationReceipt: authorization, Request: revoke,
	})
	if err != nil || !changed {
		t.Fatalf("revoke approver changed=%t err=%v", changed, err)
	}
	system.clock.Advance(2 * time.Minute)
	restartSQLiteV15System(t, system)
	return v28RevokedApproverFixture{system: system, claim: claim, goalRef: created.Record.Goal.Ref()}
}

type v28RecoveryCounts struct {
	attempts, budgetReservations, capacityReservations, capacityObservations, placements int
}

func v28RecoveryLedgerCounts(t *testing.T, database *sql.DB) v28RecoveryCounts {
	t.Helper()
	var counts v28RecoveryCounts
	for table, target := range map[string]*int{
		"effect_attempts": &counts.attempts, "budget_reservations": &counts.budgetReservations,
		"agent_capacity_reservations": &counts.capacityReservations,
		"agent_capacity_observations": &counts.capacityObservations,
		"agent_placement_bindings":    &counts.placements,
	} {
		sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM `+table).Scan(target))
	}
	return counts
}

type v28RecoveryOutbox struct {
	token, worker, recoveryRef sql.NullString
	lease                      sql.NullInt64
	deliveryAttempt, fence     int64
}

func readV28RecoveryOutbox(t *testing.T, database *sql.DB, actionRef string) v28RecoveryOutbox {
	t.Helper()
	var state v28RecoveryOutbox
	sqliteTestNoError(t, database.QueryRow(`SELECT claim_token,claimed_by,claimed_until,
delivery_attempt,fence,recovery_effect_attempt_ref FROM outbox WHERE ref=?`, actionRef).Scan(
		&state.token, &state.worker, &state.lease, &state.deliveryAttempt, &state.fence, &state.recoveryRef,
	))
	return state
}
