package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestV19MigrationEmptyV18DatabaseCreatesCouncilFoundation(t *testing.T) {
	path := emptySQLiteV18Database(t)
	repository, err := Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4})
	if err != nil {
		t.Fatalf("empty V18 migration: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })
	assertSQLiteV19MigrationHealthy(t, repository.db)
	for _, table := range []string{"council_rounds", "council_facts", "council_decisions", "council_skips"} {
		var strict int
		sqliteTestNoError(t, repository.db.QueryRow(`SELECT strict FROM pragma_table_list WHERE name=?`, table).Scan(&strict))
		if strict != 1 {
			t.Fatalf("%s is not STRICT", table)
		}
	}
	for table, column := range map[string]string{
		"work_items": "council_policy", "executions": "council_subject_digest", "outbox": "council_resolution_kind",
	} {
		var count int
		sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`, table, column).Scan(&count))
		if count != 1 {
			t.Fatalf("%s.%s missing", table, column)
		}
	}
}

func TestV19MigrationRejectsEveryLiveWriteScopedV18FrontierAtomically(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, *sql.DB)
	}{
		{name: "pending_item", mutate: func(t *testing.T, database *sql.DB) {
			mustV19Exec(t, database, `UPDATE work_items SET state='pending',started_at=NULL,finished_at=NULL,execution_ref=NULL WHERE ref='work-item:v19-closed'`)
		}},
		{name: "running_item", mutate: func(t *testing.T, database *sql.DB) {
			mustV19Exec(t, database, `UPDATE work_items SET state='running',finished_at=NULL WHERE ref='work-item:v19-closed'`)
		}},
		{name: "pending_action", mutate: func(t *testing.T, database *sql.DB) {
			mustV19Exec(t, database, `UPDATE outbox SET completed_at=NULL,claim_token=NULL,claimed_by=NULL,claimed_until=NULL,delivery_attempt=0,fence=0 WHERE ref='action:v19-closed'`)
		}},
		{name: "claimed_action", mutate: func(t *testing.T, database *sql.DB) {
			mustV19Exec(t, database, `UPDATE outbox SET completed_at=NULL WHERE ref='action:v19-closed'`)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := populatedTerminalSQLiteV18Database(t)
			database, err := sql.Open(driverName, path)
			sqliteTestNoError(t, err)
			mustV19Exec(t, database, `INSERT INTO work_item_write_scopes(goal_ref,work_item_ref,scope,position) VALUES ('goal:v19-closed','work-item:v19-closed','internal/v19',0)`)
			test.mutate(t, database)
			sqliteTestNoError(t, database.Close())

			_, err = Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2})
			if err == nil || !strings.Contains(sqliteTestErrorChain(err), "sqlite.v19_upgrade_requires_v18_council_policy") {
				t.Fatalf("live V18 frontier migrated: %s", sqliteTestErrorChain(err))
			}
			assertV19MigrationNeverStarted(t, path)
		})
	}
}

func TestV19MigrationPreservesTerminalHistoryChecksumAndReopensIdempotently(t *testing.T) {
	path := populatedTerminalSQLiteV18Database(t)
	before := snapshotV18MigrationHistoryPath(t, path)
	repository, err := Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4})
	if err != nil {
		t.Fatalf("terminal V18 migration: %s", sqliteTestErrorChain(err))
	}
	assertSQLiteV19MigrationHealthy(t, repository.db)
	after := snapshotV18MigrationHistory(t, repository.db, before)
	assertV18MigrationHistoryEqual(t, before, after)
	var policy string
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT council_policy FROM work_items WHERE ref='work-item:v19-closed'`).Scan(&policy))
	if policy != "" {
		t.Fatalf("terminal history backfilled policy=%q", policy)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("V19 recovery validation: %v", err)
	}
	rewriteRecoveryTrigger(t, repository.db, "work_items_council_policy_immutable", func() {
		mustV19Exec(t, repository.db, `UPDATE work_items SET council_policy='required'
WHERE ref='work-item:v19-closed'`)
	})
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("V19 read-only policy recovery: %v", err)
	}
	sqliteTestNoError(t, repository.Close())

	reopened, err := Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4})
	if err != nil {
		t.Fatalf("reopen V19: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = reopened.Close() })
	assertSQLiteV19MigrationHealthy(t, reopened.db)
	var receipts int
	sqliteTestNoError(t, reopened.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV19).Scan(&receipts))
	if receipts != 1 {
		t.Fatalf("V19 migration receipts=%d", receipts)
	}
}

func TestV19MigrationPreservesHistoricalDirectorDecisionReplayAndImmutability(t *testing.T) {
	path := populatedTerminalSQLiteV18Database(t)
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	historical := seedV18HistoricalDirectorDecision(t, database)
	sqliteTestNoError(t, database.Close())

	repository, err := Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4})
	if err != nil {
		t.Fatalf("historical Director V18 migration: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })
	assertSQLiteV19MigrationHealthy(t, repository.db)

	var cause, councilSubject string
	var councilDecisionRef, councilDecisionDigest sql.NullString
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT cause,council_subject_digest,
council_decision_ref,council_decision_digest FROM director_decisions WHERE ref=?`, historical.ref).
		Scan(&cause, &councilSubject, &councilDecisionRef, &councilDecisionDigest))
	if cause != "" || councilSubject != "" || councilDecisionRef.Valid || councilDecisionDigest.Valid {
		t.Fatalf("historical Director Council copy cause=%q Council=%q/%+v/%+v",
			cause, councilSubject, councilDecisionRef, councilDecisionDigest)
	}

	if _, err := repository.db.Exec(`INSERT INTO director_decisions(
 ref,request_ref,request_fingerprint,authorization_receipt_ref,goal_ref,project_ref,principal_ref,
 lease_fence,source_goal_revision,source_plan_generation,cause,source_work_item_ref,
 source_work_item_revision,source_execution_ref,source_execution_attempt,applied_goal_revision,
 applied_plan_generation,reason,decided_at,council_subject_digest,council_decision_ref,council_decision_digest
) SELECT 'director-decision:v19-duplicate','director-request:v19-duplicate','fingerprint:v19-duplicate',
 authorization_receipt_ref,goal_ref,project_ref,principal_ref,lease_fence,source_goal_revision,
 source_plan_generation,cause,source_work_item_ref,source_work_item_revision,source_execution_ref,
 source_execution_attempt,applied_goal_revision,applied_plan_generation,reason,decided_at,'',NULL,NULL
FROM director_decisions WHERE ref=?`, historical.ref); err == nil {
		t.Fatal("director decision generation uniqueness lost in V19 migration")
	}
	if _, err := repository.db.Exec(`UPDATE director_decisions SET reason='tampered' WHERE ref=?`, historical.ref); err == nil ||
		!strings.Contains(err.Error(), "sqlite.director_decision_immutable") {
		t.Fatalf("historical Director decision update=%v", err)
	}
	if _, err := repository.db.Exec(`DELETE FROM director_decisions WHERE ref=?`, historical.ref); err == nil ||
		!strings.Contains(err.Error(), "sqlite.director_decision_immutable") {
		t.Fatalf("historical Director decision delete=%v", err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("historical Director V19 recovery: %s", sqliteTestErrorChain(err))
	}

	goalRef := mustRef(t, historical.goalRef, goal.NewGoalRef)
	projectRef := mustRef(t, historical.projectRef, goal.NewProjectRef)
	principalRef := mustRef(t, historical.principalRef, identity.NewPrincipalRef)
	decision, found, err := readDirectorDecisionByRequest(
		context.Background(), repository.db, principalRef, projectRef, goalRef, historical.requestRef,
	)
	sqliteTestNoError(t, err)
	if !found || decision.Ref != historical.ref || decision.RequestRef != historical.requestRef ||
		decision.RequestFingerprint != historical.fingerprint || decision.GoalRef != goalRef ||
		decision.PrincipalRef != principalRef || int64(decision.LeaseFence) != historical.fence ||
		int64(decision.SourceGoalRevision) != historical.sourceRevision ||
		int64(decision.SourcePlanGeneration) != historical.sourceGeneration ||
		int64(decision.AppliedGoalRevision) != historical.appliedRevision ||
		int64(decision.AppliedPlanGeneration) != historical.appliedGeneration || decision.Reason != historical.reason ||
		decision.DecidedAt.UnixNano() != historical.decidedAt || decision.Cause != "" ||
		decision.CouncilSubjectDigest != "" || decision.CouncilDecisionRef != "" ||
		decision.CouncilDecisionDigest != "" {
		t.Fatalf("historical Director reader=%+v found=%t", decision, found)
	}
	record, err := repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	replayed, created, err := repository.ApplyDirectorPlan(context.Background(), application.ApplyDirectorPlanState{
		RequestRef: historical.requestRef, RequestFingerprint: historical.fingerprint,
		AuthorizationReceipt: decision.AuthorizationReceipt, PrincipalRef: principalRef, ProjectRef: projectRef,
		GoalRef: goalRef, LeaseToken: "lease-token:v19-historical", LeaseFence: decision.LeaseFence,
		ExpectedGoalRevision: decision.SourceGoalRevision, ExpectedPlanGeneration: decision.SourcePlanGeneration,
		Goal: record.Goal, Decision: decision, OperationAt: decision.DecidedAt,
	})
	if err != nil || created || replayed != decision {
		t.Fatalf("historical Director replay=%+v created=%t err=%s", replayed, created, sqliteTestErrorChain(err))
	}
}

type v18HistoricalDirectorDecision struct {
	ref, requestRef, fingerprint, authorizationRef string
	goalRef, projectRef, principalRef              string
	fence                                          int64
	sourceRevision, sourceGeneration               int64
	appliedRevision, appliedGeneration             int64
	reason                                         string
	decidedAt                                      int64
}

func seedV18HistoricalDirectorDecision(t *testing.T, database *sql.DB) v18HistoricalDirectorDecision {
	t.Helper()
	var result v18HistoricalDirectorDecision
	var createdAt int64
	var actorValue, principalKind, principalMethod string
	sqliteTestNoError(t, database.QueryRow(`SELECT goal.ref,goal.project_ref,goal.requested_by_ref,
goal.revision,goal.plan_generation,goal.created_at,principal.actor_ref,principal.kind,principal.authentication_method
FROM goals goal JOIN principals principal ON principal.ref=goal.requested_by_ref
WHERE goal.ref='goal:v19-closed'`).Scan(
		&result.goalRef, &result.projectRef, &result.principalRef, &result.appliedRevision,
		&result.appliedGeneration, &createdAt, &actorValue, &principalKind, &principalMethod,
	))
	if result.appliedRevision <= 1 || result.appliedGeneration <= 0 {
		t.Fatalf("historical Director goal generation=%d/%d", result.appliedRevision, result.appliedGeneration)
	}
	result.ref = "director-decision:v18-historical"
	result.requestRef = "director-request:v18-historical"
	result.fence = 1
	result.sourceRevision = result.appliedRevision - 1
	result.sourceGeneration = result.appliedGeneration - 1
	result.reason = "historical Director extension"
	claimAt := createdAt + int64(time.Second)
	result.decidedAt = claimAt + int64(time.Second)
	leaseUntil := result.decidedAt + int64(time.Minute)
	grantAt := claimAt - int64(time.Nanosecond)
	mustV19Exec(t, database, `INSERT INTO workspaces(ref) VALUES ('workspace:v18-historical-director')`)
	mustV19Exec(t, database, `INSERT INTO groups(ref,workspace_ref) VALUES ('group:v18-historical-director','workspace:v18-historical-director')`)
	mustV19Exec(t, database, `INSERT INTO projects(ref,group_ref) VALUES (?,'group:v18-historical-director')`, result.projectRef)
	mustV19Exec(t, database, `INSERT INTO project_memberships(
principal_ref,project_ref,role,revision,status,granted_by_ref,granted_at,revoked_by_ref,revoked_at
) VALUES (?,?,'project_owner',1,'active',?,?,NULL,NULL)`,
		result.principalRef, result.projectRef, result.principalRef, grantAt)
	mustV19Exec(t, database, `INSERT INTO membership_audit_receipts(
ref,request_ref,request_fingerprint,action,actor_ref,target_ref,project_ref,role,previous_revision,revision,occurred_at
) VALUES ('membership-audit:v18-historical-director','membership-request:v18-historical-director',
'membership-fingerprint:v18-historical-director','membership.granted',?,?,?,'project_owner',0,1,?)`,
		result.principalRef, result.principalRef, result.projectRef, grantAt)
	principalRef := mustRef(t, result.principalRef, identity.NewPrincipalRef)
	actorRef := mustRef(t, actorValue, goal.NewActorRef)
	principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKind(principalKind), principalMethod)
	sqliteTestNoError(t, err)
	projectRef := mustRef(t, result.projectRef, goal.NewProjectRef)
	makeAuthorization := func(requestRef string, at int64) (string, string) {
		request, requestErr := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
			RequestRef: requestRef, Principal: principal, ProjectRef: projectRef,
			Permission: identity.PermissionGoalsDirect, ResourceRef: result.goalRef,
			RequestedAt: time.Unix(0, at).UTC(),
		})
		sqliteTestNoError(t, requestErr)
		fingerprint := authorizationRequestFingerprint(request)
		return deterministicRef("authorization-receipt", fingerprint), fingerprint
	}
	claimAuthorizationRef, claimFingerprint := makeAuthorization("director-claim:v18-historical", claimAt)
	result.authorizationRef, result.fingerprint = makeAuthorization(result.requestRef, result.decidedAt)
	claimReceiptRef := deterministicRef("director-lease-receipt", canonicalFingerprint(
		"director-lease-receipt.v1", "claim", result.principalRef, "director-claim:v18-historical",
		claimFingerprint, result.goalRef, result.projectRef,
	))
	for _, authorization := range []struct{ ref, request, fingerprint string }{
		{claimAuthorizationRef, "director-claim:v18-historical", claimFingerprint},
		{result.authorizationRef, result.requestRef, result.fingerprint},
	} {
		at := claimAt
		if authorization.request == result.requestRef {
			at = result.decidedAt
		}
		mustV19Exec(t, database, `INSERT INTO authorization_receipts(
ref,request_ref,request_fingerprint,principal_ref,project_ref,permission,resource_ref,requested_at,
outcome,role,membership_revision,reason_code,decided_at,recorded_at
) VALUES (?,?,?,?,?,'goals.direct',?,?, 'allowed','project_owner',1,'rbac.allowed',?,?)`,
			authorization.ref, authorization.request, authorization.fingerprint, result.principalRef,
			result.projectRef, result.goalRef, at, at, at, at)
	}
	mustV19Exec(t, database, `INSERT INTO director_lease_receipts(
ref,action,request_ref,request_fingerprint,authorization_receipt_ref,goal_ref,project_ref,principal_ref,
fence,lease_until,occurred_at
) VALUES (?,'claim','director-claim:v18-historical',
	?,?,?,?,?,1,?,?)`,
		claimReceiptRef, claimFingerprint, claimAuthorizationRef, result.goalRef, result.projectRef,
		result.principalRef, leaseUntil, claimAt)
	mustV19Exec(t, database, `INSERT INTO director_leases(
goal_ref,project_ref,principal_ref,token,fence,lease_until,claim_request_ref,claim_request_fingerprint,
claim_authorization_receipt_ref,renew_request_ref,renew_request_fingerprint,renew_authorization_receipt_ref,updated_at
) VALUES (?,?,?,'lease-token:v18-historical',1,?,'director-claim:v18-historical',
	?, ?,NULL,NULL,NULL,?)`,
		result.goalRef, result.projectRef, result.principalRef, leaseUntil, claimFingerprint, claimAuthorizationRef, claimAt)
	mustV19Exec(t, database, `INSERT INTO director_decisions(
ref,request_ref,request_fingerprint,authorization_receipt_ref,goal_ref,project_ref,principal_ref,lease_fence,
source_goal_revision,source_plan_generation,cause,source_work_item_ref,source_work_item_revision,
source_execution_ref,source_execution_attempt,applied_goal_revision,applied_plan_generation,reason,decided_at
) VALUES (?,?,?,?,?,?,?,?,?,?,'',NULL,0,NULL,0,?,?,?,?)`,
		result.ref, result.requestRef, result.fingerprint, result.authorizationRef, result.goalRef,
		result.projectRef, result.principalRef, result.fence, result.sourceRevision, result.sourceGeneration,
		result.appliedRevision, result.appliedGeneration, result.reason, result.decidedAt)
	mustV19Exec(t, database, `INSERT INTO events(ref,kind,goal_ref,work_item_ref,execution_ref,occurred_at)
VALUES (?,'director.plan_applied',?,NULL,NULL,?)`,
		"event:director-plan-applied:"+result.ref, result.goalRef, result.decidedAt)
	return result
}

func TestV19MigrationRejectsPartialHistoricalInterruptPairAtomically(t *testing.T) {
	for _, test := range []struct {
		name   string
		update string
	}{
		{
			name: "cause_without_time",
			update: `UPDATE work_items SET state='superseded',revision=4,control_sequence=1,
interrupt_cause='execution_failed',interrupted_at=NULL WHERE ref='work-item:v19-closed'`,
		},
		{
			name: "time_without_cause",
			update: `UPDATE work_items SET state='superseded',revision=4,control_sequence=1,
interrupt_cause='',interrupted_at=finished_at WHERE ref='work-item:v19-closed'`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := populatedTerminalSQLiteV18Database(t)
			database, err := sql.Open(driverName, path)
			sqliteTestNoError(t, err)
			mustV19Exec(t, database, test.update)
			sqliteTestNoError(t, database.Close())

			_, err = Open(context.Background(), Options{
				Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
			})
			if err == nil || !strings.Contains(sqliteTestErrorChain(err), "constraint failed") {
				t.Fatalf("partial V18 interrupt pair migrated: %s", sqliteTestErrorChain(err))
			}
			assertV19MigrationNeverStarted(t, path)
		})
	}
}

func TestV19MigrationPreservesValidInterruptedPair(t *testing.T) {
	path := populatedTerminalSQLiteV18Database(t)
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	rewriteRecoveryTrigger(t, database, "artifact_occurrences_immutable_delete", func() {
		rewriteRecoveryTrigger(t, database, "attestations_immutable_delete", func() {
			rewriteRecoveryTrigger(t, database, "artifacts_immutable_delete", func() {
				mustV19Exec(t, database, `DELETE FROM attestation_test_outcomes`)
				mustV19Exec(t, database, `DELETE FROM artifact_occurrences`)
				mustV19Exec(t, database, `DELETE FROM attestations`)
				mustV19Exec(t, database, `DELETE FROM artifacts`)
			})
		})
	})
	mustV19Exec(t, database, `UPDATE goals SET state='running',revision=6,control_sequence=1,closed_at=NULL
WHERE ref='goal:v19-closed'`)
	mustV19Exec(t, database, `UPDATE work_items SET state='interrupted',revision=4,control_sequence=1,
interrupt_cause='execution_failed',interrupted_at=finished_at,finished_at=NULL
WHERE ref='work-item:v19-closed'`)
	mustV19Exec(t, database, `UPDATE executions SET state='failed',failure_code='application.execution_failed'
WHERE ref='execution:v19-closed'`)
	sqliteTestNoError(t, database.Close())

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
	})
	if err != nil {
		t.Fatalf("valid interrupted V18 migration: %s", sqliteTestErrorChain(err))
	}
	assertSQLiteV19MigrationHealthy(t, repository.db)
	var state, cause string
	var interruptedAt int64
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT state,interrupt_cause,interrupted_at
FROM work_items WHERE ref='work-item:v19-closed'`).Scan(&state, &cause, &interruptedAt))
	if state != "interrupted" || cause != "execution_failed" || interruptedAt == 0 {
		t.Fatalf("valid interrupted history=%q/%q/%d", state, cause, interruptedAt)
	}
	sqliteTestNoError(t, repository.Close())

	reopened, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
	})
	if err != nil {
		t.Fatalf("valid interrupted V19 reopen: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = reopened.Close() })
	assertSQLiteV19MigrationHealthy(t, reopened.db)
}

func TestV19MigrationPreservesSupersededTerminalWriterWithoutPolicy(t *testing.T) {
	path := populatedTerminalSQLiteV18Database(t)
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	rewriteRecoveryTrigger(t, database, "artifact_occurrences_immutable_delete", func() {
		rewriteRecoveryTrigger(t, database, "attestations_immutable_delete", func() {
			rewriteRecoveryTrigger(t, database, "artifacts_immutable_delete", func() {
				mustV19Exec(t, database, `DELETE FROM attestation_test_outcomes`)
				mustV19Exec(t, database, `DELETE FROM artifact_occurrences`)
				mustV19Exec(t, database, `DELETE FROM attestations`)
				mustV19Exec(t, database, `DELETE FROM artifacts`)
			})
		})
	})
	mustV19Exec(t, database, `UPDATE goals SET state='failed',revision=9,control_sequence=1,plan_generation=2
WHERE ref='goal:v19-closed'`)
	mustV19Exec(t, database, `UPDATE work_items SET state='superseded',revision=4,control_sequence=1
WHERE ref='work-item:v19-closed'`)
	mustV19Exec(t, database, `UPDATE executions SET state='failed',failure_code='application.execution_superseded'
WHERE ref='execution:v19-closed'`)
	mustV19Exec(t, database, `INSERT INTO work_items(
 ref,goal_ref,actor_ref,project_ref,objective,phase_key,role_key,parent_ref,output_contract,
 skip_reason,interrupt_cause,rework_of,state,revision,paused,cancel_requested,control_sequence,
 position,created_at,started_at,interrupted_at,finished_at,execution_ref,handoff_required,
 governance_version,budget_demand_ref,budget_tokens,budget_money_micros,budget_currency,
 budget_active_time_ns,budget_process_slots,budget_disk_bytes,security_criticality,reasoning_effort)
SELECT 'work-item:v19-successor',goal_ref,actor_ref,project_ref,'terminal rework successor',
 phase_key,role_key,NULL,output_contract,'','',ref,'failed',3,0,0,0,1,created_at,started_at,
 NULL,finished_at,'execution:v19-successor',0,governance_version,budget_demand_ref,budget_tokens,
 budget_money_micros,budget_currency,budget_active_time_ns,budget_process_slots,budget_disk_bytes,
 security_criticality,reasoning_effort
FROM work_items WHERE ref='work-item:v19-closed'`)
	mustV19Exec(t, database, `INSERT INTO executions(
 ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,
 plan_generation,app_spec_generation,spec_hash,repository_ref,execution_workspace_ref,state,
 purpose,review_subject_digest,artifact_media_type,idempotency_key,max_output_bytes,provider_ref,
 model_ref,agent_ref,external_ref,governance_version,budget_reservation_ref,effect_intent_ref,
 launch_receipt_ref,created_at,deadline_at,started_at,provider_accepted_at,last_observed_at,
 provider_observed_at,finished_at,failure_code,recipient_mailbox_retired)
SELECT 'execution:v19-successor',goal_ref,'work-item:v19-successor',1,max_execution_attempts,NULL,
 2,app_spec_generation,spec_hash,'','','failed','work','',artifact_media_type,
 'idempotency:v19-successor',max_output_bytes,provider_ref,model_ref,agent_ref,
 'external:v19-successor',0,NULL,NULL,NULL,created_at,deadline_at,started_at,
 provider_accepted_at,last_observed_at,provider_observed_at,finished_at,
 'application.execution_failed',0
FROM executions WHERE ref='execution:v19-closed'`)
	mustV19Exec(t, database, `INSERT INTO work_item_write_scopes(goal_ref,work_item_ref,scope,position)
VALUES ('goal:v19-closed','work-item:v19-closed','internal/v19',0)`)
	sqliteTestNoError(t, database.Close())

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("superseded terminal V18 migration: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })
	record, err := repository.GetGoal(context.Background(), mustRef(t, "goal:v19-closed", goal.NewGoalRef))
	sqliteTestNoError(t, err)
	source, found := record.Goal.WorkItem(mustRef(t, "work-item:v19-closed", goal.NewWorkItemRef))
	successor, successorFound := record.Goal.WorkItem(mustRef(t, "work-item:v19-successor", goal.NewWorkItemRef))
	reworkOf, linked := successor.ReworkOf()
	_, policyFound := source.CouncilPolicy()
	if !found || !successorFound || source.State() != goal.WorkItemStateSuperseded ||
		successor.State() != goal.WorkItemStateFailed || !linked || reworkOf != source.Ref() || policyFound {
		t.Fatalf("superseded migration source=%+v successor=%+v linked=%t policy=%t",
			source, successor, linked, policyFound)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("superseded terminal recovery: %s", sqliteTestErrorChain(err))
	}
}

func emptySQLiteV18Database(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	sqliteTestNoError(t, os.Chmod(directory, 0o700))
	path := filepath.Join(directory, "v18-empty.db")
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(context.Background(), database, migrations[:recoverySchemaV18]))
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))
	return path
}

func populatedTerminalSQLiteV18Database(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	sqliteTestNoError(t, os.Chmod(directory, 0o700))
	path := filepath.Join(directory, "v18-terminal.db")
	sqliteTestNoError(t, preparePrivateDatabase(path))
	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	sqliteTestNoError(t, err)
	database.SetMaxOpenConns(1)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	tx, err := database.Begin()
	sqliteTestNoError(t, err)
	if _, err = tx.Exec(migrations[0].sql); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO schema_migrations(version,name,checksum) VALUES (1,?,?)`, migrations[0].name, migrations[0].checksum); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`PRAGMA user_version=1`); err != nil {
		t.Fatal(err)
	}
	seedV1Goal(t, tx, "v19-closed", "succeeded", 6, "succeeded", 3, "succeeded", time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC), true)
	sqliteTestNoError(t, tx.Commit())
	sqliteTestNoError(t, migrateSQLiteV19Prefix(t, database, migrations, 1, recoverySchemaV18))
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))
	return path
}

func migrateSQLiteV19Prefix(t *testing.T, database *sql.DB, migrations []migration, current, target int) error {
	t.Helper()
	connection, err := database.Conn(context.Background())
	if err != nil {
		return err
	}
	defer connection.Close()
	if _, err = connection.ExecContext(context.Background(), `PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	tx, err := connection.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, _, err = applyMigrationSteps(context.Background(), tx, migrations[:target], current); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	_, err = connection.ExecContext(context.Background(), `PRAGMA foreign_keys=ON`)
	return err
}

func assertSQLiteV19MigrationHealthy(t *testing.T, database *sql.DB) {
	t.Helper()
	var version, receipt, violations, directorDecisionUniqueIndex int
	var workItemIndexes, workItemTriggers, workItemForeignKeys, staleWorkItemReferences int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=? AND name='014_council.sql'`, recoverySchemaV19).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_index_list('director_decisions')
WHERE name='director_decisions_goal_idx' AND "unique"=1`).Scan(&directorDecisionUniqueIndex))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema WHERE type='index'
AND name IN ('work_items_parent_idx','work_items_mailbox_lineage_idx','work_items_rework_idx')`).Scan(&workItemIndexes))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema WHERE type='trigger'
AND name IN ('work_items_handoff_required_immutable','work_items_governance_insert_guard',
'work_items_governance_update_guard','work_items_council_policy_immutable')`).Scan(&workItemTriggers))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_list('work_items')
WHERE "table" IN ('goals','goal_phases','work_items')`).Scan(&workItemForeignKeys))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema
WHERE sql IS NOT NULL AND sql LIKE '%work_items_v19%'`).Scan(&staleWorkItemReferences))
	if version != recoverySchemaV21 || receipt != 1 || violations != 0 || directorDecisionUniqueIndex != 1 ||
		workItemIndexes != 3 || workItemTriggers != 4 || workItemForeignKeys != 7 || staleWorkItemReferences != 0 {
		t.Fatalf("V19 migration version=%d receipt=%d foreign_keys=%d director_decisions_unique_index=%d work_item_indexes=%d triggers=%d work_item_fks=%d stale_refs=%d",
			version, receipt, violations, directorDecisionUniqueIndex, workItemIndexes, workItemTriggers,
			workItemForeignKeys, staleWorkItemReferences)
	}
}

func assertV19MigrationNeverStarted(t *testing.T, path string) {
	t.Helper()
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	defer database.Close()
	var version, receipt, column int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV19).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('work_items') WHERE name='council_policy'`).Scan(&column))
	if version != recoverySchemaV18 || receipt != 0 || column != 0 {
		t.Fatalf("rejected migration mutated version=%d receipt=%d column=%d", version, receipt, column)
	}
}

func mustV19Exec(t *testing.T, database *sql.DB, statement string, arguments ...any) {
	t.Helper()
	if _, err := database.Exec(statement, arguments...); err != nil {
		t.Fatal(err)
	}
}
