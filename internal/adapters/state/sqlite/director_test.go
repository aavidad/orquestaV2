package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type sqliteDirectorSystem struct {
	repository    *Repository
	path          string
	clock         *sqliteMembershipClock
	ids           *sqliteDirectorIDs
	orchestrator  *application.Orchestrator
	project       goal.ProjectRef
	owner         identity.Principal
	service       identity.Principal
	ownerAccess   application.Access
	serviceAccess application.Access
	goal          application.GoalRecord
}

type sqliteDirectorIDs struct {
	mu   sync.Mutex
	next int
}

func (ids *sqliteDirectorIDs) NewID(ctx context.Context, prefix string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ids.mu.Lock()
	defer ids.mu.Unlock()
	ids.next++
	return prefix + ":sqlite-director:" + fmt.Sprint(ids.next), nil
}

func (ids *sqliteDirectorIDs) Count() int {
	ids.mu.Lock()
	defer ids.mu.Unlock()
	return ids.next
}

func TestRepositoryV12DirectorLeaseRenewTakeoverReplayAndRace(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteDirectorSystem(t)
	claimRequest := application.ClaimDirectorRequest{
		RequestRef: "director-claim:owner", GoalRef: system.goal.Goal.Ref(),
	}
	claimed, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, claimRequest)
	if err != nil || !claimed.Changed || claimed.Lease.Fence != 1 || claimed.Lease.Token == "" {
		t.Fatalf("claim=%+v err=%v", claimed, err)
	}
	authorizations, ids := sqliteDirectorEffectCounts(t, system)
	replayed, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, claimRequest)
	if err != nil || replayed.Changed || replayed.Lease != claimed.Lease {
		t.Fatalf("claim replay=%+v err=%v", replayed, err)
	}
	assertSQLiteDirectorEffectCounts(t, system, authorizations, ids)

	system.clock.Advance(10 * time.Second)
	renewRequest := application.RenewDirectorRequest{
		RequestRef: "director-renew:owner", GoalRef: system.goal.Goal.Ref(),
		Token: claimed.Lease.Token, Fence: claimed.Lease.Fence,
	}
	renewed, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, renewRequest)
	if err != nil || !renewed.Changed || renewed.Lease.Token != claimed.Lease.Token ||
		renewed.Lease.Fence != claimed.Lease.Fence || !renewed.Lease.LeaseUntil.After(claimed.Lease.LeaseUntil) {
		t.Fatalf("renew=%+v err=%v", renewed, err)
	}
	authorizations, ids = sqliteDirectorEffectCounts(t, system)
	renewReplay, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, renewRequest)
	if err != nil || renewReplay.Changed || renewReplay.Lease != renewed.Lease {
		t.Fatalf("renew replay=%+v err=%v", renewReplay, err)
	}
	assertSQLiteDirectorEffectCounts(t, system, authorizations, ids)
	system.clock.Advance(5 * time.Second)
	secondRenewRequest := renewRequest
	secondRenewRequest.RequestRef = "director-renew:owner:second"
	secondRenewed, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, secondRenewRequest)
	if err != nil || !secondRenewed.Changed ||
		!secondRenewed.Lease.LeaseUntil.After(renewed.Lease.LeaseUntil) {
		t.Fatalf("second renew=%+v err=%v", secondRenewed, err)
	}

	restarted := reopenSQLiteDirectorSystem(t, system)
	authorizations, ids = sqliteDirectorEffectCounts(t, restarted)
	restartReplay, err := restarted.orchestrator.RenewDirector(ctx, restarted.ownerAccess, renewRequest)
	if err != nil || restartReplay.Changed || restartReplay.Lease != renewed.Lease {
		t.Fatalf("restart renew replay=%+v err=%v", restartReplay, err)
	}
	assertSQLiteDirectorEffectCounts(t, restarted, authorizations, ids)

	restarted.clock.Advance(time.Minute)
	reclaimed, err := restarted.orchestrator.ClaimDirector(ctx, restarted.ownerAccess, application.ClaimDirectorRequest{
		RequestRef: "director-claim:owner-reclaim", GoalRef: restarted.goal.Goal.Ref(),
	})
	if err != nil || !reclaimed.Changed || reclaimed.Lease.Fence != 2 ||
		reclaimed.Lease.PrincipalRef != restarted.owner.Ref || reclaimed.Lease.Token == claimed.Lease.Token {
		t.Fatalf("same-principal reclaim=%+v err=%v cause=%v", reclaimed, err, errors.Unwrap(err))
	}
	restarted.clock.Advance(time.Minute)
	takeover, err := restarted.orchestrator.ClaimDirector(ctx, restarted.serviceAccess, application.ClaimDirectorRequest{
		RequestRef: "director-claim:service", GoalRef: restarted.goal.Goal.Ref(),
	})
	if err != nil || !takeover.Changed || takeover.Lease.Fence != 3 ||
		takeover.Lease.PrincipalRef != restarted.service.Ref || takeover.Lease.Token == claimed.Lease.Token {
		t.Fatalf("takeover=%+v err=%v", takeover, err)
	}
	authorizations, ids = sqliteDirectorEffectCounts(t, restarted)
	if _, err := restarted.orchestrator.ClaimDirector(ctx, restarted.ownerAccess, claimRequest); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("superseded claim replay=%v", err)
	}
	if _, err := restarted.orchestrator.RenewDirector(ctx, restarted.ownerAccess, renewRequest); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("superseded renew replay=%v", err)
	}
	conflicting := claimRequest
	conflicting.GoalRef = mustRef(t, "goal:other-replay", goal.NewGoalRef)
	if _, err := restarted.orchestrator.ClaimDirector(ctx, restarted.ownerAccess, conflicting); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("claim fingerprint conflict=%v", err)
	}
	assertSQLiteDirectorEffectCounts(t, restarted, authorizations, ids)

	var leases, claimReceipts, renewReceipts int
	if err := restarted.repository.db.QueryRow(`SELECT COUNT(*) FROM director_leases`).Scan(&leases); err != nil {
		t.Fatal(err)
	}
	if err := restarted.repository.db.QueryRow(`SELECT COUNT(*) FROM director_lease_receipts WHERE action = 'claim'`).Scan(&claimReceipts); err != nil {
		t.Fatal(err)
	}
	if err := restarted.repository.db.QueryRow(`SELECT COUNT(*) FROM director_lease_receipts WHERE action = 'renew'`).Scan(&renewReceipts); err != nil {
		t.Fatal(err)
	}
	if leases != 1 || claimReceipts != 3 || renewReceipts != 2 {
		t.Fatalf("lease authority=%d claim receipts=%d renew receipts=%d", leases, claimReceipts, renewReceipts)
	}

	t.Run("concurrent first claim has one winner", func(t *testing.T) {
		race := newSQLiteDirectorSystem(t)
		ownerAuthorization := authorizeTest(
			t, race.repository, race.owner, race.project, identity.PermissionGoalsDirect,
			race.goal.Goal.Ref().String(), "authorization:race-owner", race.clock.Now(),
		)
		serviceAuthorization := authorizeTest(
			t, race.repository, race.service, race.project, identity.PermissionGoalsDirect,
			race.goal.Goal.Ref().String(), "authorization:race-service", race.clock.Now(),
		)
		states := []application.ClaimDirectorState{
			{
				RequestRef: "race:owner", RequestFingerprint: "fingerprint:race-owner",
				AuthorizationReceipt: ownerAuthorization, PrincipalRef: race.owner.Ref,
				ProjectRef: race.project, GoalRef: race.goal.Goal.Ref(), Token: "token:race-owner",
				LeaseDuration: 30 * time.Second, RequestedAt: race.clock.Now(),
			},
			{
				RequestRef: "race:service", RequestFingerprint: "fingerprint:race-service",
				AuthorizationReceipt: serviceAuthorization, PrincipalRef: race.service.Ref,
				ProjectRef: race.project, GoalRef: race.goal.Goal.Ref(), Token: "token:race-service",
				LeaseDuration: 30 * time.Second, RequestedAt: race.clock.Now(),
			},
		}
		start := make(chan struct{})
		var wait sync.WaitGroup
		var successes, blocked int
		var lock sync.Mutex
		for _, state := range states {
			state := state
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				_, changed, err := race.repository.ClaimDirector(ctx, state)
				lock.Lock()
				defer lock.Unlock()
				if err == nil && changed {
					successes++
				} else if application.IsStateError(err, application.StateAlreadyClaimed) {
					blocked++
				}
			}()
		}
		close(start)
		wait.Wait()
		if successes != 1 || blocked != 1 {
			t.Fatalf("concurrent claims successes=%d blocked=%d", successes, blocked)
		}
	})

	t.Run("concurrent identical app claim persists one authority", func(t *testing.T) {
		race := newSQLiteDirectorSystem(t)
		var authorizationsBefore int
		if err := race.repository.db.QueryRow(`SELECT COUNT(*) FROM authorization_receipts`).Scan(&authorizationsBefore); err != nil {
			t.Fatal(err)
		}
		request := application.ClaimDirectorRequest{
			RequestRef: "director-claim:identical", GoalRef: race.goal.Goal.Ref(),
		}
		start := make(chan struct{})
		results := make([]application.DirectorLeaseResult, 2)
		errorsFound := make([]error, 2)
		var wait sync.WaitGroup
		for index := range results {
			index := index
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				results[index], errorsFound[index] = race.orchestrator.ClaimDirector(
					ctx, race.ownerAccess, request,
				)
			}()
		}
		close(start)
		wait.Wait()
		changed := 0
		for index, result := range results {
			if errorsFound[index] != nil {
				t.Fatalf("identical claim %d: %v", index, errorsFound[index])
			}
			if result.Changed {
				changed++
			}
		}
		if changed != 1 || results[0].Lease != results[1].Lease {
			t.Fatalf("identical results=%+v changed=%d", results, changed)
		}
		var authorizationsAfter, leases, receipts int
		if err := race.repository.db.QueryRow(`SELECT COUNT(*) FROM authorization_receipts`).Scan(&authorizationsAfter); err != nil {
			t.Fatal(err)
		}
		if err := race.repository.db.QueryRow(`SELECT COUNT(*) FROM director_leases`).Scan(&leases); err != nil {
			t.Fatal(err)
		}
		if err := race.repository.db.QueryRow(`SELECT COUNT(*) FROM director_lease_receipts`).Scan(&receipts); err != nil {
			t.Fatal(err)
		}
		if authorizationsAfter != authorizationsBefore+1 || leases != 1 || receipts != 1 {
			t.Fatalf(
				"identical durable effects authorization=%d/%d leases=%d receipts=%d",
				authorizationsAfter, authorizationsBefore+1, leases, receipts,
			)
		}
	})
}

func TestRepositoryV12AuthorizationReplayKeepsOriginalTime(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteDirectorSystem(t)
	requestRef := "authorization-request:stable"
	firstRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: requestRef, Principal: system.owner, ProjectRef: system.project,
		Permission: identity.PermissionGoalsGet, ResourceRef: system.goal.Goal.Ref().String(),
		RequestedAt: system.clock.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := system.repository.Authorize(ctx, firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	retryRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: requestRef, Principal: system.owner, ProjectRef: system.project,
		Permission: identity.PermissionGoalsGet, ResourceRef: system.goal.Goal.Ref().String(),
		RequestedAt: system.clock.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := system.repository.Authorize(ctx, retryRequest)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Ref() != first.Ref() || replayed.Decision().Request() != firstRequest ||
		!replayed.RecordedAt().Equal(first.RecordedAt()) {
		t.Fatalf("authorization replay first=%+v replay=%+v", first, replayed)
	}
	var persisted int
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM authorization_receipts WHERE principal_ref = ? AND request_ref = ?`,
		system.owner.Ref.String(), requestRef,
	).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != 1 {
		t.Fatalf("authorization receipts=%d", persisted)
	}
	divergent, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: requestRef, Principal: system.owner, ProjectRef: system.project,
		Permission: identity.PermissionGoalsGet, ResourceRef: "goal:divergent",
		RequestedAt: system.clock.Now().Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.Authorize(ctx, divergent); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent authorization replay=%v", err)
	}
}

func TestRepositoryV12DirectorPlanLateReplayRestartAndRecovery(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteDirectorSystem(t)
	claim, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, application.ClaimDirectorRequest{
		RequestRef: "director-claim:plan", GoalRef: system.goal.Goal.Ref(),
	})
	if err != nil {
		t.Fatal(err)
	}
	firstRequest := application.ProposeDirectorPlanRequest{
		RequestRef: "director-plan:first", GoalRef: system.goal.Goal.Ref(),
		ExpectedGoalRevision:   system.goal.Goal.Revision(),
		ExpectedPlanGeneration: system.goal.Goal.PlanGeneration(),
		LeaseToken:             claim.Lease.Token, LeaseFence: claim.Lease.Fence,
		Reason: "web_application alias and docs/*.md stay advisory",
		Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
			Key: "work:research", Objective: "inspect recoverable input",
			Phase: goal.DefaultPhaseKey().String(), Role: "role:researcher",
			WriteSet: []string{"docs"}, OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
	first, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, firstRequest)
	if err != nil || !first.Created {
		t.Fatalf("first proposal=%+v err=%v", first, err)
	}
	firstRecord, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil || firstRecord.Goal.PlanGeneration() != 2 ||
		len(firstRecord.Executions) != len(system.goal.Executions)+1 {
		t.Fatalf("first canonical Goal=%+v err=%v", firstRecord, err)
	}
	firstItem := firstRecord.Goal.WorkItems()[1]
	secondRequest := application.ProposeDirectorPlanRequest{
		RequestRef: "director-plan:second", GoalRef: firstRecord.Goal.Ref(),
		ExpectedGoalRevision:   firstRecord.Goal.Revision(),
		ExpectedPlanGeneration: firstRecord.Goal.PlanGeneration(),
		LeaseToken:             claim.Lease.Token, LeaseFence: claim.Lease.Fence,
		Reason: "append dependent review",
		Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
			Key: "work:review", Objective: "review research", Phase: goal.DefaultPhaseKey().String(),
			Role: "role:reviewer", Dependencies: []string{firstItem.Ref().String()},
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
	second, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, secondRequest)
	if err != nil || !second.Created {
		t.Fatalf("second proposal=%+v err=%v", second, err)
	}
	canonical, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil || canonical.Goal.PlanGeneration() != 3 {
		t.Fatalf("second canonical Goal generation=%d err=%v", canonical.Goal.PlanGeneration(), err)
	}
	before := sqliteDirectorPersistentCounts(t, system.repository)
	authorizations, ids := sqliteDirectorEffectCounts(t, system)
	late, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, firstRequest)
	if err != nil || late.Created || late.Decision.Ref != first.Decision.Ref ||
		late.Decision.AppliedPlanGeneration != 2 {
		t.Fatalf("late replay=%+v err=%v", late, err)
	}
	if got := sqliteDirectorPersistentCounts(t, system.repository); got != before {
		t.Fatalf("late replay mutated state got=%v want=%v", got, before)
	}
	assertSQLiteDirectorEffectCounts(t, system, authorizations, ids)

	restarted := reopenSQLiteDirectorSystem(t, system)
	authorizations, ids = sqliteDirectorEffectCounts(t, restarted)
	restartReplay, err := restarted.orchestrator.ProposeDirectorPlan(ctx, restarted.ownerAccess, firstRequest)
	if err != nil || restartReplay.Created || restartReplay.Decision.Ref != first.Decision.Ref ||
		restartReplay.Decision.AppliedPlanGeneration != 2 {
		t.Fatalf("restart replay=%+v err=%v", restartReplay, err)
	}
	assertSQLiteDirectorEffectCounts(t, restarted, authorizations, ids)

	recovery, _, _ := newV09TestRecovery(t, restarted.repository, restarted.clock.Now(), nil)
	backup, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("backup V12: %v cause=%v", err, errors.Unwrap(err))
	}
	if _, err := recovery.VerifyBackup(ctx, backup.Ref); err != nil {
		t.Fatalf("verify V12: %v", err)
	}
	targetRef, err := application.NewRecoveryTargetRef("recovery-target:v12-director")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.RestoreBackup(ctx, backup.Ref, targetRef); err != nil {
		t.Fatalf("restore V12: %v", err)
	}
	targetPath, err := recovery.TargetPath(targetRef)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Open(ctx, Options{
		Path: targetPath, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8, Now: restarted.clock.Now,
	})
	if err != nil {
		t.Fatalf("open restored V12: %v", err)
	}
	t.Cleanup(func() { _ = restored.Close() })
	restoredSystem := *restarted
	restoredSystem.repository = restored
	restoredSystem.orchestrator = newSQLiteDirectorOrchestrator(t, restored, restoredSystem.clock, restoredSystem.ids)
	restoredReplay, err := restoredSystem.orchestrator.ProposeDirectorPlan(ctx, restoredSystem.ownerAccess, firstRequest)
	if err != nil || restoredReplay.Created || restoredReplay.Decision.Ref != first.Decision.Ref ||
		restoredReplay.Decision.AppliedPlanGeneration != 2 {
		t.Fatalf("restored replay=%+v err=%v", restoredReplay, err)
	}

	columns := sqliteTableColumns(t, restored, "director_decisions")
	if columns["lease_token"] || columns["token"] {
		t.Fatalf("director decisions persist lease capability: %v", columns)
	}
}

func TestRepositoryV12MigratesPopulatedV6ToV7(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "v6", "state.sqlite")
	if err := preparePrivateDatabase(path); err != nil {
		t.Fatal(err)
	}
	database := openRawV10TestDatabase(t, path)
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if err := applyRecoveryMigrationPrefix(ctx, database, migrations[:recoverySchemaV10]); err != nil {
		t.Fatal(err)
	}
	repository := &Repository{db: database, path: path, now: func() time.Time { return at }}
	project := mustRef(t, "project:v6-v7", goal.NewProjectRef)
	owner := testPrincipal(t, "principal:v6-v7", "actor:v6-v7", identity.PrincipalKindHuman)
	provisionTestAccess(t, repository, owner, project, identity.RoleProjectOwner, at)
	state := newCreateFixture(t, "v6-v7", "request:v6-v7", "fingerprint:v6-v7", owner.ActorRef.String(), project.String())
	state.RequestedBy = owner.Ref
	state.AuthorizationReceipt = authorizeTest(
		t, repository, owner, project, identity.PermissionGoalsCreate,
		project.String(), "authorization:v6-v7:create", at,
	)
	if _, created, err := repository.CreateGoal(ctx, state); err != nil || !created {
		t.Fatalf("seed V6 goal created=%v err=%v cause=%v", created, err, errors.Unwrap(err))
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	migrated, err := Open(ctx, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: func() time.Time { return at },
	})
	if err != nil {
		t.Fatalf("migrate V6 to V7: %v", err)
	}
	t.Cleanup(func() { _ = migrated.Close() })
	assertRecoverySchemaVersion(t, migrated.db, recoverySchemaV16)
	if _, err := migrated.GetGoal(ctx, state.Goal.Ref()); err != nil {
		t.Fatalf("migrated Goal: %v", err)
	}
	directAuthorization := authorizeTest(
		t, migrated, owner, project, identity.PermissionGoalsDirect,
		state.Goal.Ref().String(), "authorization:v7:direct", at,
	)
	if directAuthorization.Decision().Outcome() != identity.AuthorizationAllowed {
		t.Fatalf("goals.direct after migration=%+v", directAuthorization.Decision())
	}
	if _, _, err := validateRecoveryDatabase(ctx, migrated.db); err != nil {
		t.Fatalf("V7 recovery validation: %v", err)
	}
	var foreignKeyFailures int
	rows, err := migrated.db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		foreignKeyFailures++
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if foreignKeyFailures != 0 {
		t.Fatalf("V6 to V7 foreign key failures=%d", foreignKeyFailures)
	}
}

func TestRepositoryV12DirectorGuardsAndRecoveryRejectCausalTampering(t *testing.T) {
	ctx := context.Background()
	t.Run("lease update requires causal receipt", func(t *testing.T) {
		system := newSQLiteDirectorSystem(t)
		if _, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, application.ClaimDirectorRequest{
			RequestRef: "director-claim:guard", GoalRef: system.goal.Goal.Ref(),
		}); err != nil {
			t.Fatal(err)
		}
		_, err := system.repository.db.Exec(`
UPDATE director_leases
SET lease_until = lease_until + 1000000,
    renew_request_ref = 'director-renew:missing-receipt',
    renew_request_fingerprint = 'fingerprint:missing-receipt',
    renew_authorization_receipt_ref = claim_authorization_receipt_ref,
    updated_at = updated_at + 1
WHERE goal_ref = ?`, system.goal.Goal.Ref().String())
		if err == nil || !strings.Contains(err.Error(), "sqlite.director_lease_update_invalid") {
			t.Fatalf("lease update without receipt=%v", err)
		}
	})

	t.Run("active lease binds latest renewal receipt", func(t *testing.T) {
		system := newSQLiteDirectorSystem(t)
		claim, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, application.ClaimDirectorRequest{
			RequestRef: "director-claim:renew-binding", GoalRef: system.goal.Goal.Ref(),
		})
		if err != nil {
			t.Fatal(err)
		}
		system.clock.Advance(time.Second)
		if _, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, application.RenewDirectorRequest{
			RequestRef: "director-renew:binding", GoalRef: system.goal.Goal.Ref(),
			Token: claim.Lease.Token, Fence: claim.Lease.Fence,
		}); err != nil {
			t.Fatal(err)
		}
		rewriteRecoveryTrigger(t, system.repository.db, "director_leases_update_guard", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE director_leases
SET renew_request_ref = NULL,
    renew_request_fingerprint = NULL,
    renew_authorization_receipt_ref = NULL
WHERE goal_ref = ?`, system.goal.Goal.Ref().String())
		})
		if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_director_active_renew_binding_invalid") {
			t.Fatalf("renewal binding tamper accepted=%v", err)
		}
	})

	t.Run("next fence starts only after previous expiry", func(t *testing.T) {
		system := newSQLiteDirectorSystem(t)
		if _, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, application.ClaimDirectorRequest{
			RequestRef: "director-claim:fence-one", GoalRef: system.goal.Goal.Ref(),
		}); err != nil {
			t.Fatal(err)
		}
		system.clock.Advance(time.Minute)
		if _, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, application.ClaimDirectorRequest{
			RequestRef: "director-claim:fence-two", GoalRef: system.goal.Goal.Ref(),
		}); err != nil {
			t.Fatal(err)
		}
		rewriteRecoveryTrigger(t, system.repository.db, "director_lease_receipts_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE director_lease_receipts
SET lease_until = (
    SELECT occurred_at + 1 FROM director_lease_receipts next
    WHERE next.goal_ref = director_lease_receipts.goal_ref
      AND next.fence = 2 AND next.action = 'claim'
)
WHERE goal_ref = ? AND fence = 1 AND action = 'claim'`, system.goal.Goal.Ref().String())
		})
		if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_director_takeover_before_expiry") {
			t.Fatalf("premature fence tamper accepted=%v", err)
		}
	})

	t.Run("takeover requires its causal claim receipt", func(t *testing.T) {
		system := newSQLiteDirectorSystem(t)
		if _, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, application.ClaimDirectorRequest{
			RequestRef: "director-claim:causal-one", GoalRef: system.goal.Goal.Ref(),
		}); err != nil {
			t.Fatal(err)
		}
		system.clock.Advance(time.Minute)
		if _, err := system.orchestrator.ClaimDirector(ctx, system.serviceAccess, application.ClaimDirectorRequest{
			RequestRef: "director-claim:causal-two", GoalRef: system.goal.Goal.Ref(),
		}); err != nil {
			t.Fatal(err)
		}
		rewriteRecoveryTrigger(t, system.repository.db, "director_lease_receipts_immutable_delete", func() {
			mustV10Exec(t, system.repository.db, `
DELETE FROM director_lease_receipts
WHERE goal_ref = ? AND fence = 2 AND action = 'claim'`, system.goal.Goal.Ref().String())
		})
		if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_director_claim_chain_incomplete") {
			t.Fatalf("takeover without causal receipt accepted=%v", err)
		}
	})

	t.Run("decision cannot point beyond Goal", func(t *testing.T) {
		system, decision := seedSQLiteDirectorDecision(t)
		rewriteRecoveryTrigger(t, system.repository.db, "director_decisions_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE director_decisions
SET source_plan_generation = source_plan_generation + 100,
    applied_plan_generation = applied_plan_generation + 100
WHERE ref = ?`, decision.Decision.Ref)
		})
		if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_director_decision_invalid") {
			t.Fatalf("future decision tamper accepted=%v", err)
		}
	})

	t.Run("latest decision must match live Goal generation", func(t *testing.T) {
		system, decision := seedSQLiteDirectorDecision(t)
		rewriteRecoveryTrigger(t, system.repository.db, "director_decisions_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE director_decisions
SET source_plan_generation = 0, applied_plan_generation = 1
WHERE ref = ?`, decision.Decision.Ref)
		})
		if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_director_decision_tail_invalid") {
			t.Fatalf("decision tail mismatch accepted=%v cause=%v", err, errors.Unwrap(err))
		}
	})

	t.Run("decision requires its causal event", func(t *testing.T) {
		system, decision := seedSQLiteDirectorDecision(t)
		if _, err := system.repository.db.Exec(`DELETE FROM events WHERE ref = ?`,
			"event:director-plan-applied:"+decision.Decision.Ref,
		); err != nil {
			t.Fatal(err)
		}
		if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_director_decision_event_invalid") {
			t.Fatalf("missing decision event accepted=%v", err)
		}
	})

	t.Run("decision audit is immutable", func(t *testing.T) {
		system, decision := seedSQLiteDirectorDecision(t)
		if _, err := system.repository.db.Exec(`UPDATE director_decisions SET reason = 'tampered' WHERE ref = ?`,
			decision.Decision.Ref,
		); err == nil || !strings.Contains(err.Error(), "sqlite.director_decision_immutable") {
			t.Fatalf("decision update=%v", err)
		}
		if _, err := system.repository.db.Exec(`DELETE FROM director_decisions WHERE ref = ?`,
			decision.Decision.Ref,
		); err == nil || !strings.Contains(err.Error(), "sqlite.director_decision_immutable") {
			t.Fatalf("decision delete=%v", err)
		}
	})
}

func seedSQLiteDirectorDecision(t *testing.T) (*sqliteDirectorSystem, application.DirectorPlanResult) {
	t.Helper()
	ctx := context.Background()
	system := newSQLiteDirectorSystem(t)
	claim, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, application.ClaimDirectorRequest{
		RequestRef: "director-claim:tamper", GoalRef: system.goal.Goal.Ref(),
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	decision, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, application.ProposeDirectorPlanRequest{
		RequestRef: "director-plan:tamper", GoalRef: current.Goal.Ref(),
		ExpectedGoalRevision: current.Goal.Revision(), ExpectedPlanGeneration: current.Goal.PlanGeneration(),
		LeaseToken: claim.Lease.Token, LeaseFence: claim.Lease.Fence, Reason: "causal recovery fixture",
		Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
			Key: "work:causal-fixture", Objective: "persist causal fixture",
			Phase: goal.DefaultPhaseKey().String(), Role: "role:reviewer",
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	})
	if err != nil || !decision.Created {
		t.Fatalf("seed Director decision=%+v err=%v", decision, err)
	}
	return system, decision
}

func TestRepositoryV14DirectorSplitReplanSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteDirectorSystem(t)
	claim, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, application.ClaimDirectorRequest{
		RequestRef: "director-claim:v14-split", GoalRef: system.goal.Goal.Ref(),
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	source := current.Goal.WorkItems()[0]
	if len(current.Executions) != 1 || current.Executions[0].State != application.ExecutionQueued {
		t.Fatalf("split source projection=%+v", current.Executions)
	}
	execution := current.Executions[0]
	request := application.ProposeDirectorPlanRequest{
		RequestRef: "director-plan:v14-split", GoalRef: current.Goal.Ref(),
		ExpectedGoalRevision: current.Goal.Revision(), ExpectedPlanGeneration: current.Goal.PlanGeneration(),
		LeaseToken: claim.Lease.Token, LeaseFence: claim.Lease.Fence,
		Cause: goal.ReplanCauseSplitPending, SourceWorkItemRef: source.Ref(),
		ExpectedWorkItemRevision: source.Revision(), SourceExecutionRef: execution.Ref,
		SourceExecutionAttempt: execution.AttemptNo, Reason: "persist exact split replan",
		Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
			Key: "successor", Objective: "sqlite split successor", Phase: source.Phase().String(),
			Role: source.Role().String(), OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
	decision, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, request)
	if err != nil || !decision.Created {
		t.Fatalf("split replan: result=%+v err=%v", decision, err)
	}

	restarted := reopenSQLiteDirectorSystem(t, system)
	persisted, err := restarted.repository.GetGoal(ctx, current.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	persistedSource, _ := persisted.Goal.WorkItem(source.Ref())
	if persistedSource.State() != goal.WorkItemStateSuperseded || len(persisted.Goal.WorkItems()) != 2 {
		t.Fatalf("restart source/items=%s/%d", persistedSource.State(), len(persisted.Goal.WorkItems()))
	}
	successor := persisted.Goal.WorkItems()[1]
	reworkOf, linked := successor.ReworkOf()
	oldExecution, oldFound := sqliteExecutionByRef(persisted.Executions, execution.Ref)
	var successorExecution application.ExecutionRecord
	for _, candidate := range persisted.Executions {
		if candidate.WorkItemRef == successor.Ref() {
			successorExecution = candidate
		}
	}
	if !linked || reworkOf != source.Ref() || !oldFound || oldExecution.State != application.ExecutionCanceled ||
		successorExecution.Ref.String() == "" || successorExecution.State != application.ExecutionQueued {
		t.Fatalf("restart causal projection: source=%s old=%+v successor=%+v", reworkOf, oldExecution, successorExecution)
	}
	var activeActions int
	if err := restarted.repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox
WHERE completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`).Scan(&activeActions); err != nil {
		t.Fatal(err)
	}
	if activeActions != 1 {
		t.Fatalf("restart active actions=%d", activeActions)
	}
	before := sqliteDirectorPersistentCounts(t, restarted.repository)
	replay, err := restarted.orchestrator.ProposeDirectorPlan(ctx, restarted.ownerAccess, request)
	if err != nil || replay.Created || replay.Decision.Ref != decision.Decision.Ref {
		t.Fatalf("restart split replay: result=%+v err=%v", replay, err)
	}
	if after := sqliteDirectorPersistentCounts(t, restarted.repository); after != before {
		t.Fatalf("restart replay mutated state: before=%v after=%v", before, after)
	}
	if _, _, err := validateRecoveryDatabase(ctx, restarted.repository.db); err != nil {
		t.Fatalf("recovery rejected V14 split replan: %v", err)
	}
}

func newSQLiteDirectorSystem(t *testing.T) *sqliteDirectorSystem {
	t.Helper()
	repository, path := openTestRepository(t)
	clock := &sqliteMembershipClock{now: time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &sqliteDirectorIDs{}
	project := mustRef(t, "project:v12-director", goal.NewProjectRef)
	owner := testPrincipal(t, "principal:v12-owner", "actor:v12-owner", identity.PrincipalKindHuman)
	service := testPrincipal(t, "principal:v12-service", "actor:v12-service", identity.PrincipalKindService)
	provisionTestAccess(t, repository, owner, project, identity.RoleProjectOwner, clock.Now())
	grantTestMembership(t, repository, owner, service, project, identity.RoleOperator, "membership:v12-service", clock.Now())
	ownerAccess, err := application.NewAccess(owner, project)
	if err != nil {
		t.Fatal(err)
	}
	serviceAccess, err := application.NewAccess(service, project)
	if err != nil {
		t.Fatal(err)
	}
	orchestrator := newSQLiteDirectorOrchestrator(t, repository, clock, ids)
	submitted, err := orchestrator.Submit(context.Background(), ownerAccess, application.SubmitRequest{
		RequestRef: "request:v12-goal", Statement: "coordinate transferable goal", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit V12 Goal: %v", err)
	}
	return &sqliteDirectorSystem{
		repository: repository, path: path, clock: clock, ids: ids, orchestrator: orchestrator,
		project: project, owner: owner, service: service,
		ownerAccess: ownerAccess, serviceAccess: serviceAccess, goal: submitted.Record,
	}
}

func newSQLiteDirectorOrchestrator(
	t *testing.T,
	repository *Repository,
	clock *sqliteMembershipClock,
	ids *sqliteDirectorIDs,
) *application.Orchestrator {
	t.Helper()
	stub := sqliteMembershipExternalStub{}
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Access: repository, Launcher: stub, Observer: stub, Artifacts: stub,
		Clock: clock, IDs: ids, MaxOutputBytes: 1024,
		MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3,
		MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: sqliteTestBudgetPolicy(clock.Now()),
		ClaimLease: time.Minute, DirectorLeaseDuration: 30 * time.Second,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: sqliteTestCapabilities(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return orchestrator
}

func reopenSQLiteDirectorSystem(t *testing.T, source *sqliteDirectorSystem) *sqliteDirectorSystem {
	t.Helper()
	if err := source.repository.Close(); err != nil {
		t.Fatal(err)
	}
	repository, err := Open(context.Background(), Options{
		Path: source.path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8, Now: source.clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	restarted := *source
	restarted.repository = repository
	restarted.orchestrator = newSQLiteDirectorOrchestrator(t, repository, source.clock, source.ids)
	return &restarted
}

func sqliteDirectorEffectCounts(t *testing.T, system *sqliteDirectorSystem) (int, int) {
	t.Helper()
	var authorizations int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM authorization_receipts`).Scan(&authorizations); err != nil {
		t.Fatal(err)
	}
	return authorizations, system.ids.Count()
}

func assertSQLiteDirectorEffectCounts(t *testing.T, system *sqliteDirectorSystem, authorizations, ids int) {
	t.Helper()
	gotAuthorizations, gotIDs := sqliteDirectorEffectCounts(t, system)
	if gotAuthorizations != authorizations || gotIDs != ids {
		t.Fatalf("replay effects authorization=%d/%d ids=%d/%d", gotAuthorizations, authorizations, gotIDs, ids)
	}
}

func sqliteDirectorPersistentCounts(t *testing.T, repository *Repository) [6]int {
	t.Helper()
	var result [6]int
	tables := []string{"director_decisions", "work_items", "executions", "outbox", "events", "authorization_receipts"}
	for index, table := range tables {
		if err := repository.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&result[index]); err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func sqliteTableColumns(t *testing.T, repository *Repository, table string) map[string]bool {
	t.Helper()
	rows, err := repository.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	result := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		result[name] = true
	}
	if err := rows.Err(); err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	return result
}
