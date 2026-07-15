package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestV10RecoveryRestoresExactV09ThenOpenMigratesToV10(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 7, 15, 15, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "v09-source", "state.sqlite")
	seedV09DatabaseForV10(t, path)
	database := openRawV10TestDatabase(t, path)
	repository := &Repository{db: database, path: path, now: func() time.Time { return at }}
	recovery, backupRoot, _ := newV09TestRecovery(t, repository, at, nil)

	schemaRef, _, err := validateRecoveryDatabase(ctx, database)
	if err != nil {
		t.Fatalf("validate exact V09: %v", err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if schemaRef != migrationSchemaRef(migrations[:recoverySchemaV09]) {
		t.Fatalf("V09 schema ref = %s", schemaRef)
	}
	receipt, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("backup V09: %v", err)
	}
	if receipt.SchemaRef != schemaRef {
		t.Fatalf("V09 receipt schema = %s want %s", receipt.SchemaRef, schemaRef)
	}
	if _, err := recovery.VerifyBackup(ctx, receipt.Ref); err != nil {
		t.Fatalf("verify V09: %v", err)
	}
	targetRef, _ := application.NewRecoveryTargetRef("recovery-target:v10-v09-exact")
	if _, err := recovery.RestoreBackup(ctx, receipt.Ref, targetRef); err != nil {
		t.Fatalf("restore V09: %v", err)
	}
	targetPath, err := recovery.TargetPath(targetRef)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(findRecoveryFile(t, backupRoot, recoveryPayloadName))
	if err != nil {
		t.Fatal(err)
	}
	restoredBytes, err := os.ReadFile(targetPath)
	if err != nil || !bytes.Equal(payload, restoredBytes) {
		t.Fatalf("restore changed V09 bytes: equal=%v err=%v", bytes.Equal(payload, restoredBytes), err)
	}

	rawTarget := openRawV10TestDatabase(t, targetPath)
	assertRecoverySchemaVersion(t, rawTarget, recoverySchemaV09)
	var v10Tables, requestedBy int
	if err := rawTarget.QueryRow(`SELECT COUNT(*) FROM sqlite_schema
WHERE type = 'table' AND name IN ('principals', 'project_memberships', 'authorization_receipts')`).Scan(&v10Tables); err != nil {
		t.Fatal(err)
	}
	if err := rawTarget.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('goals') WHERE name = 'requested_by_ref'`).Scan(&requestedBy); err != nil {
		t.Fatal(err)
	}
	if v10Tables != 0 || requestedBy != 0 {
		t.Fatalf("V09 restore was migrated early: tables=%d requested_by=%d", v10Tables, requestedBy)
	}
	if err := rawTarget.Close(); err != nil {
		t.Fatal(err)
	}
	if err := recovery.Close(); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	migrated, err := Open(ctx, Options{
		Path: targetPath, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("Open did not migrate restored V09: %v", err)
	}
	defer migrated.Close()
	assertRecoverySchemaVersion(t, migrated.db, recoverySchemaV12)
	var bound int
	if err := migrated.db.QueryRow(`SELECT COUNT(*) FROM goals g
JOIN app_specs spec ON spec.ref = g.app_spec_ref
WHERE g.ref = 'goal:v10-v09'
  AND g.requested_by_ref = 'migration:v09:' || spec.confirmed_by`).Scan(&bound); err != nil || bound != 1 {
		t.Fatalf("V09 requested_by backfill = %d err=%v", bound, err)
	}
}

func TestV10RecoveryRestoresExactV10IdentitySnapshot(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC)
	repository, _ := openTestRepository(t)
	actorRef, _ := goal.NewActorRef("actor:recovery-v10")
	projectRef, _ := goal.NewProjectRef("project:recovery-v10")
	_ = newRestartAccess(t, repository, actorRef, projectRef, at)
	principalRef, _ := identity.NewPrincipalRef(actorRef.String())
	principal, _ := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindHuman, "test")
	authorizationRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization-request:recovery-v10", Principal: principal,
		ProjectRef: projectRef, Permission: identity.PermissionGoalsGet,
		ResourceRef: "goal:recovery-v10", RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Authorize(ctx, authorizationRequest); err != nil {
		t.Fatalf("seed authorization receipt: %v", err)
	}
	recovery, backupRoot, _ := newV09TestRecovery(t, repository, at.Add(time.Minute), nil)
	receipt, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("backup V10: %v", err)
	}
	migrations, _ := loadMigrations()
	if receipt.SchemaRef != migrationSchemaRef(migrations[:recoverySchemaV12]) {
		t.Fatalf("V12 schema ref = %s", receipt.SchemaRef)
	}
	targetRef, _ := application.NewRecoveryTargetRef("recovery-target:v10-exact")
	if _, err := recovery.RestoreBackup(ctx, receipt.Ref, targetRef); err != nil {
		t.Fatalf("restore V10: %v", err)
	}
	targetPath, _ := recovery.TargetPath(targetRef)
	payload, _ := os.ReadFile(findRecoveryFile(t, backupRoot, recoveryPayloadName))
	restored, err := os.ReadFile(targetPath)
	if err != nil || !bytes.Equal(payload, restored) {
		t.Fatalf("restore changed V10 bytes: equal=%v err=%v", bytes.Equal(payload, restored), err)
	}
	raw := openRawV10TestDatabase(t, targetPath)
	defer raw.Close()
	assertRecoverySchemaVersion(t, raw, recoverySchemaV12)
	if schemaRef, _, err := validateRecoveryDatabase(ctx, raw); err != nil || schemaRef != receipt.SchemaRef {
		t.Fatalf("restored V10 semantics: schema=%s err=%v", schemaRef, err)
	}
}

func TestV10RecoveryRejectsV4AndVersionSpecificTampering(t *testing.T) {
	t.Run("V4 exact prefix is unsupported", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "v04", "state.sqlite")
		if err := preparePrivateDatabase(path); err != nil {
			t.Fatal(err)
		}
		database := openRawV10TestDatabase(t, path)
		defer database.Close()
		migrations, _ := loadMigrations()
		if err := applyRecoveryMigrationPrefix(context.Background(), database, migrations[:4]); err != nil {
			t.Fatal(err)
		}
		if _, _, err := validateRecoveryDatabase(context.Background(), database); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_schema_version_invalid") {
			t.Fatalf("V4 recovery accepted: %v", err)
		}
	})

	t.Run("V09 migration checksum prefix is exact", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "v09", "state.sqlite")
		seedV09DatabaseForV10(t, path)
		database := openRawV10TestDatabase(t, path)
		defer database.Close()
		mustV10Exec(t, database, `UPDATE schema_migrations SET checksum = ? WHERE version = 5`,
			"sha256:"+strings.Repeat("0", 64))
		if _, _, err := validateRecoveryDatabase(context.Background(), database); err == nil ||
			!recoveryErrorContains(err, "sqlite.migration_history_invalid") {
			t.Fatalf("tampered V09 checksum accepted: %v", err)
		}
	})

	t.Run("V10 authorization fingerprint is semantic", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		at := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
		actorRef, _ := goal.NewActorRef("actor:recovery-tamper")
		projectRef, _ := goal.NewProjectRef("project:recovery-tamper")
		_ = newRestartAccess(t, repository, actorRef, projectRef, at)
		principalRef, _ := identity.NewPrincipalRef(actorRef.String())
		principal, _ := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindHuman, "test")
		request, _ := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
			RequestRef: "authorization-request:tamper", Principal: principal, ProjectRef: projectRef,
			Permission: identity.PermissionGoalsGet, ResourceRef: "goal:tamper", RequestedAt: at,
		})
		if _, err := repository.Authorize(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		rewriteRecoveryTrigger(t, repository.db, "authorization_receipts_immutable_update", func() {
			mustV10Exec(t, repository.db, `UPDATE authorization_receipts
SET request_fingerprint = 'tampered' WHERE request_ref = ?`, request.RequestRef())
		})
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_authorization_fingerprint_invalid") {
			t.Fatalf("tampered authorization accepted: %v", err)
		}
	})

	t.Run("V10 membership requires latest audit", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		at := time.Date(2026, 7, 15, 18, 0, 0, 0, time.UTC)
		actorRef, _ := goal.NewActorRef("actor:recovery-audit")
		projectRef, _ := goal.NewProjectRef("project:recovery-audit")
		_ = newRestartAccess(t, repository, actorRef, projectRef, at)
		rewriteRecoveryTrigger(t, repository.db, "membership_audit_receipts_immutable_delete", func() {
			mustV10Exec(t, repository.db, `DELETE FROM membership_audit_receipts WHERE project_ref = ?`, projectRef.String())
		})
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_membership_audit_chain_invalid") {
			t.Fatalf("membership without audit accepted: %v", err)
		}
	})

	t.Run("V10 membership rejects duplicate revision", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		at := time.Date(2026, 7, 15, 18, 30, 0, 0, time.UTC)
		actorRef, _ := goal.NewActorRef("actor:recovery-duplicate-audit")
		projectRef, _ := goal.NewProjectRef("project:recovery-duplicate-audit")
		_ = newRestartAccess(t, repository, actorRef, projectRef, at)
		mustV10Exec(t, repository.db, `INSERT INTO membership_audit_receipts(
ref, request_ref, request_fingerprint, action, actor_ref, target_ref,
project_ref, role, previous_revision, revision, occurred_at
) VALUES (?, ?, ?, 'membership.granted', ?, ?, ?, 'project_owner', 0, 1, ?)`,
			"membership-audit:duplicate", "membership-request:duplicate", "membership-fingerprint:duplicate",
			actorRef.String(), actorRef.String(), projectRef.String(), at.UnixNano(),
		)
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_membership_audit_chain_invalid") {
			t.Fatalf("duplicate membership revision accepted: %v", err)
		}
	})

	t.Run("V10 membership binds latest grant time", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		at := time.Date(2026, 7, 15, 18, 45, 0, 0, time.UTC)
		actorRef, _ := goal.NewActorRef("actor:recovery-audit-time")
		projectRef, _ := goal.NewProjectRef("project:recovery-audit-time")
		_ = newRestartAccess(t, repository, actorRef, projectRef, at)
		rewriteRecoveryTrigger(t, repository.db, "membership_audit_receipts_immutable_update", func() {
			mustV10Exec(t, repository.db, `UPDATE membership_audit_receipts
SET occurred_at = occurred_at + 1 WHERE project_ref = ?`, projectRef.String())
		})
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_membership_audit_binding_invalid") {
			t.Fatalf("misbound membership grant time accepted: %v", err)
		}
	})

	t.Run("V10 authorization outcome reason tuple is semantic", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		at := time.Date(2026, 7, 15, 19, 0, 0, 0, time.UTC)
		actorRef, _ := goal.NewActorRef("actor:recovery-auth-tuple")
		projectRef, _ := goal.NewProjectRef("project:recovery-auth-tuple")
		_ = newRestartAccess(t, repository, actorRef, projectRef, at)
		principalRef, _ := identity.NewPrincipalRef(actorRef.String())
		principal, _ := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindHuman, "test")
		request, _ := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
			RequestRef: "authorization-request:tuple", Principal: principal, ProjectRef: projectRef,
			Permission: identity.PermissionGoalsCreate, ResourceRef: projectRef.String(), RequestedAt: at,
		})
		if _, err := repository.Authorize(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		rewriteRecoveryTrigger(t, repository.db, "authorization_receipts_immutable_update", func() {
			mustV10Exec(t, repository.db, `UPDATE authorization_receipts
SET reason_code = 'rbac.permission_denied' WHERE request_ref = ?`, request.RequestRef())
		})
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_authorization_tuple_invalid") {
			t.Fatalf("misbound authorization tuple accepted: %v", err)
		}
	})

	t.Run("V10 requested_by remains bound to confirmer", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		state := authorizeRecoveryCreate(t, repository, newCreateFixture(
			t, "recovery-requested-by", "request:recovery-requested-by",
			"fingerprint:recovery-requested-by", "actor:recovery-requested-by", "project:recovery-requested-by",
		))
		if _, _, err := repository.CreateGoal(context.Background(), state); err != nil {
			t.Fatal(err)
		}
		other := testPrincipal(
			t, "principal:recovery-other", "actor:recovery-other", identity.PrincipalKindHuman,
		)
		ensureLegacyProjectAccess(t, repository, other, state.Goal.Project(), state.Goal.CreatedAt())
		rewriteRecoveryTrigger(t, repository.db, "goals_app_spec_immutable", func() {
			mustV10Exec(t, repository.db, `UPDATE goals SET requested_by_ref = ? WHERE ref = ?`,
				other.Ref.String(), state.Goal.Ref().String())
		})
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_requested_by_binding_invalid") {
			t.Fatalf("misbound requested_by accepted: %v", err)
		}
	})
}

func TestV10RecoveryAuthorizationTupleContract(t *testing.T) {
	at := time.Date(2026, 7, 15, 20, 0, 0, 0, time.UTC)
	actorRef, _ := goal.NewActorRef("actor:recovery-tuple-contract")
	principalRef, _ := identity.NewPrincipalRef("principal:recovery-tuple-contract")
	principal, _ := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindHuman, "test")
	projectRef, _ := goal.NewProjectRef("project:recovery-tuple-contract")
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization-request:tuple-contract", Principal: principal, ProjectRef: projectRef,
		Permission: identity.PermissionGoalsCreate, ResourceRef: projectRef.String(), RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		outcome  identity.AuthorizationOutcome
		role     identity.Role
		revision identity.MembershipRevision
		reason   string
		valid    bool
	}{
		{"allowed", identity.AuthorizationAllowed, identity.RoleContributor, 1, authorizationReasonAllowed, true},
		{"project unknown", identity.AuthorizationDenied, "", 0, authorizationReasonProjectUnknown, true},
		{"membership missing", identity.AuthorizationDenied, "", 0, authorizationReasonMembershipMissing, true},
		{"membership revoked", identity.AuthorizationDenied, identity.RoleViewer, 1, authorizationReasonMembershipRevoked, true},
		{"permission denied", identity.AuthorizationDenied, identity.RoleViewer, 1, authorizationReasonPermissionDenied, true},
		{"allowed wrong reason", identity.AuthorizationAllowed, identity.RoleContributor, 1, authorizationReasonPermissionDenied, false},
		{"unknown with revision", identity.AuthorizationDenied, "", 1, authorizationReasonProjectUnknown, false},
		{"missing with role", identity.AuthorizationDenied, identity.RoleViewer, 1, authorizationReasonMembershipMissing, false},
		{"revoked without membership", identity.AuthorizationDenied, "", 0, authorizationReasonMembershipRevoked, false},
		{"denied role actually allows", identity.AuthorizationDenied, identity.RoleContributor, 1, authorizationReasonPermissionDenied, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
				Request: request, Outcome: test.outcome, Role: test.role,
				MembershipRevision: test.revision, ReasonCode: test.reason, DecidedAt: at,
			})
			if err != nil {
				t.Fatalf("construct decision: %v", err)
			}
			err = validateRecoveryAuthorizationTuple(decision)
			if (err == nil) != test.valid {
				t.Fatalf("tuple valid=%v err=%v", test.valid, err)
			}
		})
	}
}

func TestV10RecoveryAcceptsContiguousMembershipLifecycle(t *testing.T) {
	repository, _ := openTestRepository(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 15, 21, 0, 0, 0, time.UTC)
	projectRef, _ := goal.NewProjectRef("project:recovery-membership-lifecycle")
	owner := testPrincipal(t, "principal:recovery-owner", "actor:recovery-owner", identity.PrincipalKindHuman)
	target := testPrincipal(t, "principal:recovery-target", "actor:recovery-target", identity.PrincipalKindService)
	provisionTestAccess(t, repository, owner, projectRef, identity.RoleProjectOwner, at)
	grantTestMembership(
		t, repository, owner, target, projectRef, identity.RoleContributor,
		"membership:recovery-lifecycle-grant", at.Add(time.Second),
	)
	revokeAt := at.Add(2 * time.Second)
	revokeAuthorization := authorizeTest(
		t, repository, owner, projectRef, identity.PermissionProjectMembershipManage,
		target.Ref.String(), "authorization:recovery-lifecycle-revoke", revokeAt,
	)
	revokeRequest := testRevokeRequest(
		t, "membership:recovery-lifecycle-revoke", owner, target.Ref, projectRef, 1, revokeAt,
	)
	if _, _, changed, err := repository.RevokeMembership(ctx, application.MembershipRevokeState{
		AuthorizationReceipt: revokeAuthorization, Request: revokeRequest,
	}); err != nil || !changed {
		t.Fatalf("revoke lifecycle membership: changed=%v err=%v", changed, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, repository.db); err != nil {
		t.Fatalf("valid revoked membership rejected: %v", err)
	}
	regrantAt := at.Add(3 * time.Second)
	regrantAuthorization := authorizeTest(
		t, repository, owner, projectRef, identity.PermissionProjectMembershipManage,
		target.Ref.String(), "authorization:recovery-lifecycle-regrant", regrantAt,
	)
	regrantRequest := testGrantRequest(
		t, "membership:recovery-lifecycle-regrant", owner, target.Ref, projectRef,
		identity.RoleReviewer, 2, regrantAt,
	)
	if _, _, changed, err := repository.GrantMembership(ctx, application.MembershipGrantState{
		AuthorizationReceipt: regrantAuthorization, Request: regrantRequest, Target: target,
	}); err != nil || !changed {
		t.Fatalf("regrant lifecycle membership: changed=%v err=%v", changed, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, repository.db); err != nil {
		t.Fatalf("valid regranted membership rejected: %v", err)
	}
}

func TestRecoveryCanonicalInventoryAndSchemaRefAreVersioned(t *testing.T) {
	v09, err := canonicalSchemaInventoryDigest(recoverySchemaV09)
	if err != nil {
		t.Fatal(err)
	}
	v10, err := canonicalSchemaInventoryDigest(recoverySchemaV10)
	if err != nil {
		t.Fatal(err)
	}
	v12, err := canonicalSchemaInventoryDigest(recoverySchemaV12)
	if err != nil {
		t.Fatal(err)
	}
	if v09 == v10 || v09 == v12 || v10 == v12 ||
		!strings.HasPrefix(v09, "sha256:") || !strings.HasPrefix(v10, "sha256:") ||
		!strings.HasPrefix(v12, "sha256:") {
		t.Fatalf("canonical inventories not versioned: V09=%s V10=%s V12=%s", v09, v10, v12)
	}
	migrations, _ := loadMigrations()
	v09Ref := migrationSchemaRef(migrations[:recoverySchemaV09])
	v10Ref := migrationSchemaRef(migrations[:recoverySchemaV10])
	v12Ref := migrationSchemaRef(migrations[:recoverySchemaV12])
	if v09Ref == v10Ref || v09Ref == v12Ref || v10Ref == v12Ref ||
		!strings.HasPrefix(v09Ref, schemaRefPrefix) || !strings.HasPrefix(v10Ref, schemaRefPrefix) ||
		!strings.HasPrefix(v12Ref, schemaRefPrefix) {
		t.Fatalf("schema refs not versioned: V09=%s V10=%s V12=%s", v09Ref, v10Ref, v12Ref)
	}
}

func assertRecoverySchemaVersion(t *testing.T, database *sql.DB, want int) {
	t.Helper()
	var version, receipts int
	if err := database.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if version != want || receipts != want {
		t.Fatalf("schema version/receipts = %d/%d want %d", version, receipts, want)
	}
}

func rewriteRecoveryTrigger(t *testing.T, database *sql.DB, name string, mutate func()) {
	t.Helper()
	var statement string
	if err := database.QueryRow(`SELECT sql FROM sqlite_schema WHERE type = 'trigger' AND name = ?`, name).Scan(&statement); err != nil {
		t.Fatal(err)
	}
	mustV10Exec(t, database, "DROP TRIGGER "+quoteSQLiteIdentifier(name))
	mutate()
	mustV10Exec(t, database, statement)
}

func recoveryErrorContains(err error, text string) bool {
	for err != nil {
		if strings.Contains(err.Error(), text) {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
