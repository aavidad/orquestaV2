package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestRepositoryV10AuthorizationScopesProjectsPrincipalsAndDefaultsToDeny(t *testing.T) {
	repository, _ := openTestRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	projectA := mustRef(t, "project:auth-a", goal.NewProjectRef)
	projectB := mustRef(t, "project:auth-b", goal.NewProjectRef)
	owner := testPrincipal(t, "principal:owner", "actor:shared", identity.PrincipalKindHuman)
	service := testPrincipal(t, "principal:service", "actor:shared", identity.PrincipalKindHuman)
	human := testPrincipal(t, "principal:human", "actor:shared", identity.PrincipalKindHuman)
	provisionTestAccess(t, repository, owner, projectA, identity.RoleProjectOwner, now)
	provisionTestAccess(t, repository, owner, projectB, identity.RoleProjectOwner, now)

	grantTestMembership(t, repository, owner, service, projectA, identity.RoleContributor, "grant:service", now)
	grantTestMembership(t, repository, owner, human, projectA, identity.RoleViewer, "grant:human", now)

	allowed := authorizeTest(t, repository, service, projectA, identity.PermissionGoalsCreate, projectA.String(), "auth:service:create", now)
	if allowed.Decision().Outcome() != identity.AuthorizationAllowed || allowed.Decision().Role() != identity.RoleContributor {
		t.Fatalf("service authorization = %+v", allowed.Decision())
	}
	deniedRole := authorizeTest(t, repository, human, projectA, identity.PermissionGoalsCreate, projectA.String(), "auth:human:create", now)
	if deniedRole.Decision().Outcome() != identity.AuthorizationDenied || deniedRole.Decision().Role() != identity.RoleViewer {
		t.Fatalf("viewer default deny = %+v", deniedRole.Decision())
	}
	deniedProject := authorizeTest(t, repository, service, projectB, identity.PermissionGoalsGet, "goal:unknown", "auth:service:project-b", now)
	if deniedProject.Decision().Outcome() != identity.AuthorizationDenied || deniedProject.Decision().Role() != "" {
		t.Fatalf("cross-project default deny = %+v", deniedProject.Decision())
	}
	unknownProject := mustRef(t, "project:unknown", goal.NewProjectRef)
	unknown := authorizeTest(t, repository, service, unknownProject, identity.PermissionGoalsGet, "goal:unknown", "auth:unknown-project", now)
	if unknown.Decision().Outcome() != identity.AuthorizationDenied {
		t.Fatalf("unknown project authorization = %+v", unknown.Decision())
	}

	replayed := authorizeTest(t, repository, service, projectA, identity.PermissionGoalsCreate, projectA.String(), "auth:service:create", now)
	if replayed.Ref() != allowed.Ref() || !replayed.RecordedAt().Equal(allowed.RecordedAt()) {
		t.Fatalf("authorization replay changed: first=%s replay=%s", allowed.Ref(), replayed.Ref())
	}
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "auth:service:create", Principal: service, ProjectRef: projectA,
		Permission: identity.PermissionGoalsCreate, ResourceRef: "resource:changed", RequestedAt: now,
	})
	sqliteTestNoError(t, err)
	if _, err := repository.Authorize(ctx, request); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent authorization replay = %v", err)
	}
}

func TestRepositoryV10MembershipCASReplayRaceAndImmutableAudit(t *testing.T) {
	repository, _ := openTestRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	project := mustRef(t, "project:membership", goal.NewProjectRef)
	owner := testPrincipal(t, "principal:membership-owner", "actor:membership-owner", identity.PrincipalKindHuman)
	target := testPrincipal(t, "principal:membership-target", "actor:membership-target", identity.PrincipalKindService)
	provisionTestAccess(t, repository, owner, project, identity.RoleProjectOwner, now)

	authorization := authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		target.Ref.String(), "auth:membership:grant", now,
	)
	request := testGrantRequest(t, "membership:grant", owner, target.Ref, project, identity.RoleContributor, 0, now)
	state := application.MembershipGrantState{AuthorizationReceipt: authorization, Request: request, Target: target}
	membership, audit, created, err := repository.GrantMembership(ctx, state)
	if err != nil || !created || membership.Revision() != 1 || !membership.IsActive() ||
		audit.Revision() != 1 || audit.Action() != identity.MembershipAuditGranted {
		t.Fatalf("grant = membership:%+v audit:%+v created:%v err:%v", membership, audit, created, err)
	}
	replayState := state
	replayState.AuthorizationReceipt = authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		target.Ref.String(), "auth:membership:grant:retry", now,
	)
	replayed, replayAudit, created, err := repository.GrantMembership(ctx, replayState)
	if err != nil || created || replayed.Snapshot() != membership.Snapshot() || replayAudit.Ref() != audit.Ref() {
		t.Fatalf("grant replay = membership:%+v audit:%+v created:%v err:%v", replayed, replayAudit, created, err)
	}

	divergent := state
	divergent.Request = testGrantRequest(t, "membership:grant", owner, target.Ref, project, identity.RoleReviewer, 0, now)
	if _, _, _, err := repository.GrantMembership(ctx, divergent); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent grant replay = %v", err)
	}

	revokeAuthorization := authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		target.Ref.String(), "auth:membership:revoke", now,
	)
	revokeRequest := testRevokeRequest(t, "membership:revoke", owner, target.Ref, project, 1, now)
	revoked, revokeAudit, changed, err := repository.RevokeMembership(ctx, application.MembershipRevokeState{
		AuthorizationReceipt: revokeAuthorization, Request: revokeRequest,
	})
	if err != nil || !changed || revoked.Status() != identity.MembershipRevoked ||
		revoked.Revision() != 2 || revokeAudit.Action() != identity.MembershipAuditRevoked {
		t.Fatalf("revoke = membership:%+v audit:%+v changed:%v err:%v", revoked, revokeAudit, changed, err)
	}
	revokeReplayAuthorization := authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		target.Ref.String(), "auth:membership:revoke:retry", now,
	)
	if _, _, changed, err := repository.RevokeMembership(ctx, application.MembershipRevokeState{
		AuthorizationReceipt: revokeReplayAuthorization, Request: revokeRequest,
	}); err != nil || changed {
		t.Fatalf("revoke replay changed=%v err=%v", changed, err)
	}

	for _, statement := range []string{
		"UPDATE membership_audit_receipts SET role = 'viewer' WHERE ref = ?",
		"DELETE FROM membership_audit_receipts WHERE ref = ?",
	} {
		if _, err := repository.db.Exec(statement, audit.Ref()); err == nil {
			t.Fatalf("immutable audit accepted: %s", statement)
		}
	}

	raceTarget := testPrincipal(t, "principal:race-target", "actor:race-target", identity.PrincipalKindHuman)
	const contenders = 2
	start := make(chan struct{})
	var successes atomic.Int64
	var conflicts atomic.Int64
	var wait sync.WaitGroup
	raceStates := make([]application.MembershipGrantState, contenders)
	for index := 0; index < contenders; index++ {
		suffix := string(rune('a' + index))
		auth := authorizeTest(t, repository, owner, project, identity.PermissionProjectMembershipManage,
			raceTarget.Ref.String(), "auth:race:"+suffix, now)
		req := testGrantRequest(t, "membership:race:"+suffix, owner, raceTarget.Ref,
			project, identity.RoleContributor, 0, now)
		raceStates[index] = application.MembershipGrantState{
			AuthorizationReceipt: auth, Request: req, Target: raceTarget,
		}
	}
	for index := 0; index < contenders; index++ {
		state := raceStates[index]
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, _, changed, err := repository.GrantMembership(ctx, state)
			if err == nil && changed {
				successes.Add(1)
			} else if application.IsStateError(err, application.StateConflict) {
				conflicts.Add(1)
			}
		}()
	}
	close(start)
	wait.Wait()
	if successes.Load() != 1 || conflicts.Load() != 1 {
		t.Fatalf("membership CAS race successes=%d conflicts=%d", successes.Load(), conflicts.Load())
	}

	ownerRevokeAuthorization := authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		owner.Ref.String(), "auth:last-owner:revoke", now,
	)
	ownerRevoke := testRevokeRequest(t, "membership:last-owner:revoke", owner, owner.Ref, project, 1, now)
	if _, _, _, err := repository.RevokeMembership(ctx, application.MembershipRevokeState{
		AuthorizationReceipt: ownerRevokeAuthorization, Request: ownerRevoke,
	}); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("last project owner revocation = %v", err)
	}
	demotionAuthorization := authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		owner.Ref.String(), "auth:last-owner:demote", now,
	)
	demotionRequest := testGrantRequest(
		t, "membership:last-owner:demote", owner, owner.Ref, project, identity.RoleViewer, 1, now,
	)
	if _, _, _, err := repository.GrantMembership(ctx, application.MembershipGrantState{
		AuthorizationReceipt: demotionAuthorization, Request: demotionRequest, Target: owner,
	}); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("last project owner demotion = %v", err)
	}
	secondOwner := testPrincipal(t, "principal:second-owner", "actor:second-owner", identity.PrincipalKindHuman)
	staleTarget := testPrincipal(t, "principal:stale-target", "actor:stale-target", identity.PrincipalKindHuman)
	staleMembershipAuthorization := authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		staleTarget.Ref.String(), "auth:membership:before-role-change", now,
	)
	grantTestMembership(t, repository, owner, secondOwner, project, identity.RoleProjectOwner, "grant:second-owner", now)
	demotionAuthorization = authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		owner.Ref.String(), "auth:owner-transfer:demote", now,
	)
	demotionRequest = testGrantRequest(
		t, "membership:owner-transfer:demote", owner, owner.Ref, project, identity.RoleViewer, 1, now,
	)
	if _, _, changed, err := repository.GrantMembership(ctx, application.MembershipGrantState{
		AuthorizationReceipt: demotionAuthorization, Request: demotionRequest, Target: owner,
	}); err != nil || !changed {
		t.Fatalf("owner transfer demotion changed=%v err=%v", changed, err)
	}
	staleGrant := testGrantRequest(
		t, "membership:stale-actor-authority", owner, staleTarget.Ref, project,
		identity.RoleViewer, 0, now,
	)
	if _, _, _, err := repository.GrantMembership(ctx, application.MembershipGrantState{
		AuthorizationReceipt: staleMembershipAuthorization, Request: staleGrant, Target: staleTarget,
	}); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("stale membership authority = %v", err)
	}
	ownerRevokeAuthorization = authorizeTest(
		t, repository, secondOwner, project, identity.PermissionProjectMembershipManage,
		owner.Ref.String(), "auth:owner-transfer:revoke", now,
	)
	ownerRevoke = testRevokeRequest(t, "membership:owner-transfer:revoke", secondOwner, owner.Ref, project, 2, now)
	if _, _, changed, err := repository.RevokeMembership(ctx, application.MembershipRevokeState{
		AuthorizationReceipt: ownerRevokeAuthorization, Request: ownerRevoke,
	}); err != nil || !changed {
		t.Fatalf("owner transfer revoke changed=%v err=%v", changed, err)
	}
}

func TestRepositoryV10ApplicationMembershipReplayAcceptsFreshAuthorizationReceipt(t *testing.T) {
	ctx := context.Background()
	repository, _ := openTestRepository(t)
	clock := &sqliteMembershipClock{now: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)}
	ids := &sqliteMembershipIDs{}
	stub := sqliteMembershipExternalStub{}
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Access: repository, Launcher: stub, Observer: stub, Artifacts: stub,
		Clock: clock, IDs: ids, MaxOutputBytes: 1024,
		MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 2,
		MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: sqliteTestBudgetPolicy(clock.Now()),
		ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: sqliteTestCapabilities(),
	})
	sqliteTestNoError(t, err)
	project := mustRef(t, "project:application-membership", goal.NewProjectRef)
	owner := testPrincipal(t, "principal:application-owner", "actor:application-owner", identity.PrincipalKindHuman)
	target := testPrincipal(t, "principal:application-target", "actor:application-target", identity.PrincipalKindService)
	provisionTestAccess(t, repository, owner, project, identity.RoleProjectOwner, clock.Now())
	access, err := application.NewAccess(owner, project)
	sqliteTestNoError(t, err)
	grant := testGrantRequest(
		t, "membership:application-grant", owner, target.Ref, project,
		identity.RoleContributor, 0, clock.Now(),
	)
	first, firstAudit, changed, err := orchestrator.GrantMembership(ctx, access, grant, target)
	if err != nil || !changed {
		t.Fatalf("application grant changed=%v err=%v", changed, err)
	}
	clock.Advance(time.Minute)
	replayed, replayAudit, changed, err := orchestrator.GrantMembership(ctx, access, grant, target)
	if err != nil || changed || replayed.Snapshot() != first.Snapshot() || replayAudit.Ref() != firstAudit.Ref() {
		t.Fatalf("application grant replay changed=%v membership=%+v audit=%s err=%v",
			changed, replayed, replayAudit.Ref(), err)
	}

	revoke := testRevokeRequest(
		t, "membership:application-revoke", owner, target.Ref, project, 1, clock.Now(),
	)
	revoked, revokedAudit, changed, err := orchestrator.RevokeMembership(ctx, access, revoke)
	if err != nil || !changed {
		t.Fatalf("application revoke changed=%v err=%v", changed, err)
	}
	clock.Advance(time.Minute)
	replayedRevoke, replayedRevokeAudit, changed, err := orchestrator.RevokeMembership(ctx, access, revoke)
	if err != nil || changed || replayedRevoke.Snapshot() != revoked.Snapshot() ||
		replayedRevokeAudit.Ref() != revokedAudit.Ref() {
		t.Fatalf("application revoke replay changed=%v membership=%+v audit=%s err=%v",
			changed, replayedRevoke, replayedRevokeAudit.Ref(), err)
	}
	var authorizationReceipts, auditReceipts int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM authorization_receipts`).Scan(&authorizationReceipts); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM membership_audit_receipts`).Scan(&auditReceipts); err != nil {
		t.Fatal(err)
	}
	if authorizationReceipts != 4 || auditReceipts != 3 { // bootstrap + grant + revoke
		t.Fatalf("receipt counts authorization=%d audit=%d", authorizationReceipts, auditReceipts)
	}
}

func TestRepositoryV10ProvisionRestartAndRevocationFenceGoalMutation(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state", "orquesta.sqlite")
	firstNow := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	repository, err := Open(ctx, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
		Now: func() time.Time { return firstNow },
	})
	sqliteTestNoError(t, err)
	project := mustRef(t, "project:restart", goal.NewProjectRef)
	owner := testPrincipal(t, "principal:restart-owner", "actor:restart-owner", identity.PrincipalKindHuman)
	contributor := testPrincipal(t, "principal:restart-contributor", "actor:restart-contributor", identity.PrincipalKindHuman)
	hierarchy := testHierarchy(t, project)
	if err := repository.ProvisionLocalAccess(ctx, owner, hierarchy, identity.RoleProjectOwner, firstNow); err != nil {
		t.Fatalf("first provision: %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	secondNow := firstNow.Add(time.Hour)
	repository, err = Open(ctx, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
		Now: func() time.Time { return secondNow },
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = repository.Close() })
	if err := repository.ProvisionLocalAccess(ctx, owner, hierarchy, identity.RoleProjectOwner, secondNow); err != nil {
		t.Fatalf("restart provision: %v", err)
	}
	intruder := testPrincipal(t, "principal:bootstrap-intruder", "actor:bootstrap-intruder", identity.PrincipalKindHuman)
	if err := repository.ProvisionLocalAccess(
		ctx, intruder, hierarchy, identity.RoleProjectOwner, secondNow,
	); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("second bootstrap authority = %v", err)
	}
	var intruderPrincipals int
	if err := repository.db.QueryRow(
		`SELECT COUNT(*) FROM principals WHERE ref = ?`, intruder.Ref.String(),
	).Scan(&intruderPrincipals); err != nil {
		t.Fatal(err)
	}
	if intruderPrincipals != 0 {
		t.Fatalf("failed second bootstrap persisted principal=%d", intruderPrincipals)
	}
	if got := tableCount(t, repository, "membership_audit_receipts"); got != 1 {
		t.Fatalf("bootstrap audit rows = %d", got)
	}
	differentRepository, err := identity.NewRepositoryRef("repository:restart-different")
	sqliteTestNoError(t, err)
	differentHierarchyInput := hierarchy.Snapshot()
	differentHierarchyInput.RepositoryRef = differentRepository
	differentHierarchy, err := identity.NewProjectHierarchy(differentHierarchyInput)
	sqliteTestNoError(t, err)
	if err := repository.ProvisionLocalAccess(
		ctx, owner, differentHierarchy, identity.RoleProjectOwner, secondNow,
	); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("different bootstrap hierarchy = %v", err)
	}
	var repositories int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM repositories WHERE project_ref = ?`, project.String()).Scan(&repositories); err != nil {
		t.Fatal(err)
	}
	if repositories != 1 || tableCount(t, repository, "membership_audit_receipts") != 1 {
		t.Fatalf("divergent bootstrap escaped repositories=%d audits=%d",
			repositories, tableCount(t, repository, "membership_audit_receipts"))
	}
	membership, err := repository.Membership(ctx, owner.Ref, project)
	if err != nil || membership.Revision() != 1 || !membership.GrantedAt().Equal(firstNow) {
		t.Fatalf("restarted membership = %+v err=%v", membership, err)
	}
	grantTestMembership(t, repository, owner, contributor, project, identity.RoleContributor, "grant:restart-contributor", secondNow)

	state := newCreateFixture(t, "revoked-write", "request:revoked-write", "fingerprint:revoked-write", contributor.ActorRef.String(), project.String())
	state.RequestedBy = contributor.Ref
	state.AuthorizationReceipt = authorizeTest(
		t, repository, contributor, project, identity.PermissionGoalsCreate, project.String(), "auth:create:before-revoke", secondNow,
	)
	revokeAuthorization := authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		contributor.Ref.String(), "auth:contributor-revoke", secondNow,
	)
	revoke := testRevokeRequest(t, "membership:contributor-revoke", owner, contributor.Ref, project, 1, secondNow)
	if _, _, changed, err := repository.RevokeMembership(ctx, application.MembershipRevokeState{
		AuthorizationReceipt: revokeAuthorization, Request: revoke,
	}); err != nil || !changed {
		t.Fatalf("self revoke changed=%v err=%v", changed, err)
	}
	if _, _, err := repository.CreateGoal(ctx, state); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("stale authorization mutated goal: %v", err)
	}
	if _, err := repository.GetGoal(ctx, state.Goal.Ref()); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("revoked write left goal: %v", err)
	}
}

func TestRepositoryV10RevokedCollaboratorAndReceiptSubstitutionFenceAmend(t *testing.T) {
	ctx := context.Background()
	repository, _ := openTestRepository(t)
	source := createFailedSourceForAmend(t, repository, "v10-stale-amend-source")
	owner := legacyTestPrincipal(t, source.RequestedBy, source.Goal.Actor())
	contributor := testPrincipal(t, "principal:stale-amender", "actor:stale-amender", identity.PrincipalKindHuman)
	state := newAmendFixture(
		t, source, "v10-stale-amend", "request:v10-stale-amend", "fingerprint:v10-stale-amend",
		"stale amendment", "authorization fence",
	)
	state.Successor = v10ReconfirmSuccessor(t, source, state, contributor.ActorRef)
	at := state.Successor.CreatedAt()
	grantTestMembership(t, repository, owner, contributor, source.Goal.Project(), identity.RoleContributor, "grant:stale-amender", at)
	state.RequestedBy = contributor.Ref
	state.AuthorizationReceipt = authorizeTest(
		t, repository, contributor, source.Goal.Project(), identity.PermissionGoalsAmend,
		source.Goal.Ref().String(), "auth:amend:before-revoke", at,
	)
	substitution := state
	substitution.AuthorizationReceipt = authorizeTest(
		t, repository, contributor, source.Goal.Project(), identity.PermissionGoalsGet,
		source.Goal.Ref().String(), "auth:get:not-amend", at,
	)
	if _, _, err := repository.AmendGoal(ctx, substitution); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("authorization receipt substitution = %v", err)
	}
	revokeAuthorization := authorizeTest(
		t, repository, owner, source.Goal.Project(), identity.PermissionProjectMembershipManage,
		contributor.Ref.String(), "auth:stale-amender:revoke", at,
	)
	revoke := testRevokeRequest(
		t, "membership:stale-amender:revoke", owner, contributor.Ref, source.Goal.Project(), 1, at,
	)
	if _, _, changed, err := repository.RevokeMembership(ctx, application.MembershipRevokeState{
		AuthorizationReceipt: revokeAuthorization, Request: revoke,
	}); err != nil || !changed {
		t.Fatalf("revoke stale amender changed=%v err=%v", changed, err)
	}
	if _, _, err := repository.AmendGoal(ctx, state); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("revoked collaborator amendment = %v", err)
	}
	if _, err := repository.GetGoal(ctx, state.Successor.Ref()); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("stale amendment left successor = %v", err)
	}
}

func testPrincipal(
	t *testing.T,
	principalValue, actorValue string,
	kind identity.PrincipalKind,
) identity.Principal {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef(principalValue)
	sqliteTestNoError(t, err)
	actorRef := mustRef(t, actorValue, goal.NewActorRef)
	principal, err := identity.NewPrincipal(principalRef, actorRef, kind, "test-auth")
	sqliteTestNoError(t, err)
	return principal
}

func testHierarchy(t *testing.T, project goal.ProjectRef) identity.ProjectHierarchy {
	t.Helper()
	workspace, err := identity.NewWorkspaceRef("workspace:test")
	sqliteTestNoError(t, err)
	group, err := identity.NewGroupRef("group:" + project.String())
	sqliteTestNoError(t, err)
	repositoryRef, err := identity.NewRepositoryRef("repository:" + project.String())
	sqliteTestNoError(t, err)
	hierarchy, err := identity.NewProjectHierarchy(identity.ProjectHierarchyInput{
		WorkspaceRef: workspace, GroupRef: group, GroupParentWorkspaceRef: workspace,
		ProjectRef: project, ProjectParentGroupRef: group,
		RepositoryRef: repositoryRef, RepositoryParentProjectRef: project,
	})
	sqliteTestNoError(t, err)
	return hierarchy
}

func provisionTestAccess(
	t *testing.T,
	repository *Repository,
	principal identity.Principal,
	project goal.ProjectRef,
	role identity.Role,
	at time.Time,
) {
	t.Helper()
	if err := repository.ProvisionLocalAccess(context.Background(), principal, testHierarchy(t, project), role, at); err != nil {
		t.Fatalf("provision test access: %v", err)
	}
}

func authorizeTest(
	t *testing.T,
	repository *Repository,
	principal identity.Principal,
	project goal.ProjectRef,
	permission identity.Permission,
	resourceRef, requestRef string,
	at time.Time,
) identity.AuthorizationReceipt {
	t.Helper()
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: requestRef, Principal: principal, ProjectRef: project,
		Permission: permission, ResourceRef: resourceRef, RequestedAt: at,
	})
	sqliteTestNoError(t, err)
	receipt, err := repository.Authorize(context.Background(), request)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	return receipt
}

func grantTestMembership(
	t *testing.T,
	repository *Repository,
	owner, target identity.Principal,
	project goal.ProjectRef,
	role identity.Role,
	requestRef string,
	at time.Time,
) identity.Membership {
	t.Helper()
	authorization := authorizeTest(
		t, repository, owner, project, identity.PermissionProjectMembershipManage,
		target.Ref.String(), "authorization:"+requestRef, at,
	)
	request := testGrantRequest(t, requestRef, owner, target.Ref, project, role, 0, at)
	membership, _, changed, err := repository.GrantMembership(context.Background(), application.MembershipGrantState{
		AuthorizationReceipt: authorization, Request: request, Target: target,
	})
	if err != nil || !changed {
		t.Fatalf("grant test membership changed=%v err=%v", changed, err)
	}
	return membership
}

func testGrantRequest(
	t *testing.T,
	requestRef string,
	actor identity.Principal,
	target identity.PrincipalRef,
	project goal.ProjectRef,
	role identity.Role,
	revision identity.MembershipRevision,
	at time.Time,
) identity.MembershipGrantRequest {
	t.Helper()
	request, err := identity.NewMembershipGrantRequest(identity.MembershipGrantRequestInput{
		RequestRef: requestRef, Actor: actor, TargetRef: target, ProjectRef: project,
		Role: role, ExpectedRevision: revision, RequestedAt: at,
	})
	sqliteTestNoError(t, err)
	return request
}

func testRevokeRequest(
	t *testing.T,
	requestRef string,
	actor identity.Principal,
	target identity.PrincipalRef,
	project goal.ProjectRef,
	revision identity.MembershipRevision,
	at time.Time,
) identity.MembershipRevokeRequest {
	t.Helper()
	request, err := identity.NewMembershipRevokeRequest(identity.MembershipRevokeRequestInput{
		RequestRef: requestRef, Actor: actor, TargetRef: target, ProjectRef: project,
		ExpectedRevision: revision, RequestedAt: at,
	})
	sqliteTestNoError(t, err)
	return request
}

func TestRepositoryV10RejectsConflictingPrincipalIdentity(t *testing.T) {
	repository, _ := openTestRepository(t)
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	project := mustRef(t, "project:principal-conflict", goal.NewProjectRef)
	principal := testPrincipal(t, "principal:stable", "actor:first", identity.PrincipalKindHuman)
	provisionTestAccess(t, repository, principal, project, identity.RoleProjectOwner, now)
	conflicting := testPrincipal(t, "principal:stable", "actor:second", identity.PrincipalKindHuman)
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "auth:principal-conflict", Principal: conflicting, ProjectRef: project,
		Permission: identity.PermissionGoalsGet, ResourceRef: "goal:any", RequestedAt: now,
	})
	sqliteTestNoError(t, err)
	if _, err := repository.Authorize(context.Background(), request); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("conflicting principal = %v", err)
	}
}

func TestRepositoryV10MigratedActorCanProvisionNewAuthenticationPrincipal(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy", "orquesta.sqlite")
	seedPopulatedV1Database(t, path)
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	repository, err := Open(ctx, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("migrate V09 state: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })
	actorRef := mustRef(t, "actor:v1", goal.NewActorRef)
	principalRef, err := identity.NewPrincipalRef("actor:v1")
	sqliteTestNoError(t, err)
	localPrincipal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, "local_token",
	)
	sqliteTestNoError(t, err)
	project := mustRef(t, "project:v1", goal.NewProjectRef)
	if err := repository.ProvisionLocalAccess(
		ctx, localPrincipal, testHierarchy(t, project), identity.RoleProjectOwner, now,
	); err != nil {
		t.Fatalf("provision same migrated actor with new authentication: %v", err)
	}

	var migratedPrincipals, localPrincipals, migratedGoals, memberships int
	queries := []struct {
		query string
		args  []any
		out   *int
	}{
		{`SELECT COUNT(*) FROM principals
          WHERE ref = 'migration:v09:actor:v1' AND actor_ref = 'actor:v1'
            AND authentication_method = 'migration.v09'`, nil, &migratedPrincipals},
		{`SELECT COUNT(*) FROM principals
          WHERE ref = 'actor:v1' AND actor_ref = 'actor:v1'
            AND authentication_method = 'local_token'`, nil, &localPrincipals},
		{`SELECT COUNT(*) FROM goals g
          JOIN app_specs spec ON spec.ref = g.app_spec_ref
          WHERE g.requested_by_ref = 'migration:v09:' || spec.confirmed_by`, nil, &migratedGoals},
		{`SELECT COUNT(*) FROM project_memberships
          WHERE principal_ref = ? AND project_ref = ? AND status = 'active'`,
			[]any{localPrincipal.Ref.String(), project.String()}, &memberships},
	}
	for _, query := range queries {
		if err := repository.db.QueryRow(query.query, query.args...).Scan(query.out); err != nil {
			t.Fatal(err)
		}
	}
	if migratedPrincipals != 1 || localPrincipals != 1 || migratedGoals != 2 || memberships != 1 {
		t.Fatalf("upgrade principal split migrated=%d local=%d goals=%d memberships=%d",
			migratedPrincipals, localPrincipals, migratedGoals, memberships)
	}
	running, err := repository.GetGoal(ctx, mustRef(t, "goal:v1-running", goal.NewGoalRef))
	if err != nil || running.RequestedBy.String() != "migration:v09:actor:v1" {
		t.Fatalf("migrated goal requested_by=%s err=%v", running.RequestedBy.String(), err)
	}
}

func TestRepositoryV10InvalidInputsRemainTyped(t *testing.T) {
	repository, _ := openTestRepository(t)
	if _, err := repository.Membership(context.Background(), identity.PrincipalRef{}, goal.ProjectRef{}); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("invalid membership query = %v", err)
	}
	if err := repository.ProvisionLocalAccess(context.Background(), identity.Principal{}, identity.ProjectHierarchy{}, "", time.Time{}); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("invalid provision = %v", err)
	}
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	project := mustRef(t, "project:viewer-bootstrap", goal.NewProjectRef)
	viewer := testPrincipal(t, "principal:viewer-bootstrap", "actor:viewer-bootstrap", identity.PrincipalKindHuman)
	if err := repository.ProvisionLocalAccess(
		context.Background(), viewer, testHierarchy(t, project), identity.RoleViewer, now,
	); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("non-authority bootstrap = %v", err)
	}
	var principals, projects int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM principals`).Scan(&principals); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&projects); err != nil {
		t.Fatal(err)
	}
	if principals != 0 || projects != 0 {
		t.Fatalf("invalid bootstrap left rows principals=%d projects=%d", principals, projects)
	}
}

type sqliteMembershipClock struct{ now time.Time }

func (clock *sqliteMembershipClock) Now() time.Time { return clock.now }
func (clock *sqliteMembershipClock) Advance(duration time.Duration) {
	clock.now = clock.now.Add(duration)
}

type sqliteMembershipIDs struct{ next int }

func (ids *sqliteMembershipIDs) NewID(ctx context.Context, prefix string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ids.next++
	return fmt.Sprintf("%s:sqlite-membership:%d", prefix, ids.next), nil
}

type sqliteMembershipExternalStub struct{}

func (sqliteMembershipExternalStub) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return sqliteTestCapabilities(), nil
}
func (sqliteMembershipExternalStub) Launch(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	return ports.AgentLaunchReceipt{}, fmt.Errorf("sqlite.membership_test.launch_unexpected")
}
func (sqliteMembershipExternalStub) Observe(context.Context, goal.ExecutionRef) (ports.AgentObservation, error) {
	return ports.AgentObservation{}, fmt.Errorf("sqlite.membership_test.observe_unexpected")
}
func (sqliteMembershipExternalStub) Put(context.Context, ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	return ports.StoredArtifact{}, fmt.Errorf("sqlite.membership_test.put_unexpected")
}
func (sqliteMembershipExternalStub) Get(context.Context, goal.ArtifactRef, int64) (ports.ArtifactContent, error) {
	return ports.ArtifactContent{}, fmt.Errorf("sqlite.membership_test.get_unexpected")
}
