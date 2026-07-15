package acceptance_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func v10AssertSingleDatabaseAudit(
	t *testing.T,
	system *v10RealSystem,
	source application.GoalRecord,
	amended application.GoalRecord,
	serviceGoal application.GoalRecord,
	grantAudit identity.MembershipAuditReceipt,
	revokeAudit identity.MembershipAuditReceipt,
	allowed identity.AuthorizationReceipt,
	denied identity.AuthorizationReceipt,
) {
	t.Helper()
	database, err := sql.Open("sqlite", system.databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var schemaVersion int
	if err := database.QueryRow(`PRAGMA user_version`).Scan(&schemaVersion); err != nil || schemaVersion < 6 {
		t.Fatalf("V10 schema version=%d err=%v", schemaVersion, err)
	}
	wantCounts := map[string]int{
		"workspaces": 2, "groups": 2, "projects": 2, "repositories": 2,
		"principals": 5, "project_memberships": 5, "membership_audit_receipts": 6,
		"goals": 3,
	}
	for table, want := range wantCounts {
		if got := v10TableCount(t, database, table); got != want {
			t.Errorf("single DB %s rows=%d want=%d", table, got, want)
		}
	}
	if grantAudit.Ref() == revokeAudit.Ref() || grantAudit.Action() != identity.MembershipAuditGranted ||
		revokeAudit.Action() != identity.MembershipAuditRevoked {
		t.Fatalf("grant/revoke audit identity collapsed: grant=%+v revoke=%+v", grantAudit, revokeAudit)
	}
	var storedAudits int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM membership_audit_receipts WHERE ref IN (?, ?)`,
		grantAudit.Ref(), revokeAudit.Ref(),
	).Scan(&storedAudits); err != nil {
		t.Fatal(err)
	}
	if storedAudits != 2 {
		t.Fatalf("membership audit receipts not persisted: count=%d", storedAudits)
	}
	var allowedCount, deniedCount, authorizationCount int
	if err := database.QueryRow(`
SELECT
    SUM(CASE WHEN outcome = 'allowed' THEN 1 ELSE 0 END),
    SUM(CASE WHEN outcome = 'denied' THEN 1 ELSE 0 END),
    COUNT(*)
FROM authorization_receipts`).Scan(&allowedCount, &deniedCount, &authorizationCount); err != nil {
		t.Fatal(err)
	}
	if allowedCount == 0 || deniedCount == 0 || allowedCount+deniedCount != authorizationCount {
		t.Fatalf("authorization receipts allowed=%d denied=%d total=%d",
			allowedCount, deniedCount, authorizationCount)
	}
	var storedAuthorizations int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM authorization_receipts WHERE ref IN (?, ?)`, allowed.Ref(), denied.Ref(),
	).Scan(&storedAuthorizations); err != nil {
		t.Fatal(err)
	}
	if storedAuthorizations != 2 {
		t.Fatalf("explicit allowed/denied receipts not persisted: count=%d", storedAuthorizations)
	}
	for _, record := range []application.GoalRecord{source, amended, serviceGoal} {
		var requestedBy, actorRef string
		if err := database.QueryRow(
			`SELECT requested_by_ref, actor_ref FROM goals WHERE ref = ?`, record.Goal.Ref().String(),
		).Scan(&requestedBy, &actorRef); err != nil {
			t.Fatal(err)
		}
		if requestedBy != record.RequestedBy.String() || actorRef != record.Goal.Actor().String() {
			t.Fatalf("Goal attribution drift ref=%s requested_by=%s/%s actor=%s/%s",
				record.Goal.Ref().String(), requestedBy, record.RequestedBy.String(),
				actorRef, record.Goal.Actor().String())
		}
	}

	membershipAuditCount := v10TableCount(t, database, "membership_audit_receipts")
	authorizationAuditCount := v10TableCount(t, database, "authorization_receipts")
	for name, mutation := range map[string]struct {
		statement string
		ref       string
	}{
		"membership_update": {
			statement: `UPDATE membership_audit_receipts SET role = 'viewer' WHERE ref = ?`, ref: grantAudit.Ref(),
		},
		"membership_delete": {
			statement: `DELETE FROM membership_audit_receipts WHERE ref = ?`, ref: revokeAudit.Ref(),
		},
		"authorization_update": {
			statement: `UPDATE authorization_receipts SET reason_code = 'tampered' WHERE ref = ?`, ref: allowed.Ref(),
		},
		"authorization_delete": {
			statement: `DELETE FROM authorization_receipts WHERE ref = ?`, ref: denied.Ref(),
		},
	} {
		if _, err := database.Exec(mutation.statement, mutation.ref); err == nil {
			t.Errorf("append-only audit allowed %s", name)
		}
	}
	if got := v10TableCount(t, database, "membership_audit_receipts"); got != membershipAuditCount {
		t.Fatalf("audit tamper changed membership receipts: %d/%d", got, membershipAuditCount)
	}
	if got := v10TableCount(t, database, "authorization_receipts"); got != authorizationAuditCount {
		t.Fatalf("audit tamper changed authorization receipts: %d/%d", got, authorizationAuditCount)
	}
}

func v10AssertRestart(
	t *testing.T,
	system *v10RealSystem,
	source application.GoalRecord,
	amended application.GoalRecord,
	serviceGoal application.GoalRecord,
) {
	t.Helper()
	ctx := context.Background()
	beforeStatus, err := system.repository.Status(ctx, system.projectA.ProjectRef())
	if err != nil {
		t.Fatal(err)
	}
	beforeDatabase, err := sql.Open("sqlite", system.databasePath)
	if err != nil {
		t.Fatal(err)
	}
	beforeMembershipAudits := v10TableCount(t, beforeDatabase, "membership_audit_receipts")
	beforeAuthorizations := v10TableCount(t, beforeDatabase, "authorization_receipts")
	beforeGoals := v10TableCount(t, beforeDatabase, "goals")
	if err := beforeDatabase.Close(); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	system.repository = nil
	system.clock.Advance(time.Hour)
	restarted, err := statesqlite.Open(ctx, statesqlite.Options{
		Path: system.databasePath, BusyTimeout: 5 * time.Second,
		MaxOpenConnections: 8, Now: system.clock.Now,
	})
	if err != nil {
		t.Fatalf("restart V10 SQLite: %v", err)
	}
	system.repository = restarted
	if err := restarted.ProvisionLocalAccess(
		ctx, system.owner, system.projectA, identity.RoleProjectOwner, system.clock.Now(),
	); err != nil {
		t.Fatalf("restart project A provision: %v", err)
	}
	if err := restarted.ProvisionLocalAccess(
		ctx, system.outsider, system.projectB, identity.RoleProjectOwner, system.clock.Now(),
	); err != nil {
		t.Fatalf("restart project B provision: %v", err)
	}
	system.orchestrator = v06NewOrchestrator(
		t, restarted, system.clock, system.ids, system.agent, system.artifacts, system.v06,
	)

	wantMemberships := []struct {
		principal identity.Principal
		project   goal.ProjectRef
		role      identity.Role
		status    identity.MembershipStatus
		revision  identity.MembershipRevision
	}{
		{system.owner, system.projectA.ProjectRef(), identity.RoleProjectOwner, identity.MembershipActive, 1},
		{system.contributor, system.projectA.ProjectRef(), identity.RoleContributor, identity.MembershipActive, 1},
		{system.viewer, system.projectA.ProjectRef(), identity.RoleViewer, identity.MembershipActive, 1},
		{system.service, system.projectA.ProjectRef(), identity.RoleContributor, identity.MembershipRevoked, 2},
		{system.outsider, system.projectB.ProjectRef(), identity.RoleProjectOwner, identity.MembershipActive, 1},
	}
	for _, want := range wantMemberships {
		membership, err := restarted.Membership(ctx, want.principal.Ref, want.project)
		if err != nil || membership.Role() != want.role || membership.Status() != want.status ||
			membership.Revision() != want.revision {
			t.Errorf("restart membership principal=%s membership=%+v err=%v",
				want.principal.Ref.String(), membership.Snapshot(), err)
		}
	}
	for _, want := range []application.GoalRecord{source, amended, serviceGoal} {
		got, err := restarted.GetGoal(ctx, want.Goal.Ref())
		if err != nil || got.RequestedBy != want.RequestedBy ||
			!reflect.DeepEqual(got.Goal.Snapshot(), want.Goal.Snapshot()) ||
			!reflect.DeepEqual(got.Artifacts, want.Artifacts) {
			t.Errorf("restart Goal ref=%s got=%+v err=%v", want.Goal.Ref().String(), got, err)
		}
	}
	contributorAccess := v10Access(t, system.contributor, system.projectA.ProjectRef())
	if got, err := system.orchestrator.GetGoal(ctx, contributorAccess, source.Goal.Ref()); err != nil || got.Goal.Ref() != source.Goal.Ref() {
		t.Fatalf("restart collaborator Goal read: %+v err=%v", got, err)
	}
	if content, err := system.orchestrator.GetArtifact(
		ctx, contributorAccess, source.Goal.Ref(), source.Artifacts[0].Stored.Ref,
	); err != nil || content.Ref != source.Artifacts[0].Stored.Ref {
		t.Fatalf("restart shared artifact read: %+v err=%v", content, err)
	}
	outsiderBAccess := v10Access(t, system.outsider, system.projectB.ProjectRef())
	if status, err := system.orchestrator.Status(ctx, outsiderBAccess); err != nil || status != (application.RepositoryStatus{}) {
		t.Fatalf("restart project B isolation: status=%+v err=%v", status, err)
	}
	afterStatus, err := restarted.Status(ctx, system.projectA.ProjectRef())
	if err != nil || afterStatus != beforeStatus {
		t.Fatalf("restart lifecycle status: before=%+v after=%+v err=%v", beforeStatus, afterStatus, err)
	}
	afterDatabase, err := sql.Open("sqlite", system.databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer afterDatabase.Close()
	if got := v10TableCount(t, afterDatabase, "membership_audit_receipts"); got != beforeMembershipAudits {
		t.Fatalf("restart duplicated immutable membership audit: %d/%d", got, beforeMembershipAudits)
	}
	if got := v10TableCount(t, afterDatabase, "goals"); got != beforeGoals {
		t.Fatalf("restart changed Goals: %d/%d", got, beforeGoals)
	}
	if got := v10TableCount(t, afterDatabase, "authorization_receipts"); got < beforeAuthorizations {
		t.Fatalf("restart lost authorization audit: %d/%d", got, beforeAuthorizations)
	}
	for _, table := range []string{"workspaces", "groups", "projects", "repositories"} {
		if got := v10TableCount(t, afterDatabase, table); got != 2 {
			t.Fatalf("restart hierarchy %s rows=%d", table, got)
		}
	}
}

func v10TableCount(t *testing.T, database *sql.DB, table string) int {
	t.Helper()
	allowed := map[string]bool{
		"workspaces": true, "groups": true, "projects": true, "repositories": true,
		"principals": true, "project_memberships": true,
		"membership_audit_receipts": true, "authorization_receipts": true, "goals": true,
	}
	if !allowed[table] {
		t.Fatalf("non-allowlisted V10 table %q", table)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
