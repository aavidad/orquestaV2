package sqlite

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// authorizeLegacyCreateState keeps pre-V10 state-adapter fixtures focused on
// their original lifecycle assertions while exercising the real V10 access
// path. New V10 tests construct identity explicitly instead.
func authorizeLegacyCreateState(
	t *testing.T,
	repository *Repository,
	state application.CreateGoalState,
) application.CreateGoalState {
	t.Helper()
	principal := legacyTestPrincipal(t, state.RequestedBy, state.Goal.Actor())
	ensureLegacyProjectAccess(t, repository, principal, state.Goal.Project(), state.Goal.CreatedAt())
	state.RequestedBy = principal.Ref
	state.AuthorizationReceipt = authorizeTest(
		t, repository, principal, state.Goal.Project(), identity.PermissionGoalsCreate,
		state.Goal.Project().String(), legacyAuthorizationRequestRef("create", principal.Ref, state.Goal.Project(), state.RequestRef),
		state.Goal.CreatedAt(),
	)
	return state
}

func authorizeLegacyAmendState(
	t *testing.T,
	repository *Repository,
	state application.AmendGoalState,
) application.AmendGoalState {
	t.Helper()
	principal := legacyTestPrincipal(t, state.RequestedBy, state.Successor.Actor())
	ensureLegacyProjectAccess(t, repository, principal, state.ProjectRef, state.Successor.CreatedAt())
	state.RequestedBy = principal.Ref
	state.AuthorizationReceipt = authorizeTest(
		t, repository, principal, state.ProjectRef, identity.PermissionGoalsAmend,
		state.SourceGoalRef.String(), legacyAuthorizationRequestRef("amend", principal.Ref, state.ProjectRef, state.RequestRef),
		state.Successor.CreatedAt(),
	)
	return state
}

func ensureLegacyProjectAccess(
	t *testing.T,
	repository *Repository,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	at time.Time,
) {
	t.Helper()
	err := repository.ProvisionLocalAccess(
		context.Background(), principal, testHierarchy(t, projectRef), identity.RoleProjectOwner, at,
	)
	if err == nil {
		return
	}
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("provision legacy access: %v", err)
	}
	current, membershipErr := repository.Membership(context.Background(), principal.Ref, projectRef)
	if membershipErr == nil && current.IsActive() && identity.RoleAllows(current.Role(), identity.PermissionGoalsCreate) {
		return
	}
	if membershipErr != nil && !application.IsStateError(membershipErr, application.StateNotFound) {
		t.Fatalf("read legacy membership: %v", membershipErr)
	}
	owner := readLegacyProjectOwner(t, repository, projectRef)
	authorization := authorizeTest(
		t, repository, owner, projectRef, identity.PermissionProjectMembershipManage,
		principal.Ref.String(), legacyAuthorizationRequestRef("membership", owner.Ref, projectRef, principal.Ref.String()), at,
	)
	expected := identity.MembershipRevision(0)
	if membershipErr == nil {
		expected = current.Revision()
	}
	request, requestErr := identity.NewMembershipGrantRequest(identity.MembershipGrantRequestInput{
		RequestRef: legacyAuthorizationRequestRef("membership-grant", owner.Ref, projectRef, principal.Ref.String()),
		Actor:      owner, TargetRef: principal.Ref, ProjectRef: projectRef,
		Role: identity.RoleProjectOwner, ExpectedRevision: expected, RequestedAt: at,
	})
	if requestErr != nil {
		t.Fatal(requestErr)
	}
	if _, _, changed, grantErr := repository.GrantMembership(context.Background(), application.MembershipGrantState{
		AuthorizationReceipt: authorization, Request: request, Target: principal,
	}); grantErr != nil || !changed {
		t.Fatalf("grant legacy project access changed=%v err=%v", changed, grantErr)
	}
}

func readLegacyProjectOwner(
	t *testing.T,
	repository *Repository,
	projectRef goal.ProjectRef,
) identity.Principal {
	t.Helper()
	var principalValue, actorValue, kind, method string
	err := repository.db.QueryRow(`
SELECT p.ref, p.actor_ref, p.kind, p.authentication_method
FROM project_memberships m
JOIN principals p ON p.ref = m.principal_ref
WHERE m.project_ref = ? AND m.status = 'active'
  AND m.role IN ('platform_admin', 'project_owner')
ORDER BY CASE m.role WHEN 'platform_admin' THEN 0 ELSE 1 END, p.ref
LIMIT 1`, projectRef.String()).Scan(&principalValue, &actorValue, &kind, &method)
	if err != nil {
		t.Fatalf("read legacy project owner: %v", err)
	}
	principalRef, err := identity.NewPrincipalRef(principalValue)
	if err != nil {
		t.Fatal(err)
	}
	actorRef, err := goal.NewActorRef(actorValue)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKind(kind), method)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func legacyTestPrincipal(
	t *testing.T,
	principalRef identity.PrincipalRef,
	actorRef goal.ActorRef,
) identity.Principal {
	t.Helper()
	var err error
	if principalRef.String() == "" {
		principalRef, err = identity.NewPrincipalRef(actorRef.String())
		if err != nil {
			t.Fatal(err)
		}
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, "sqlite_legacy_test",
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func legacyAuthorizationRequestRef(
	kind string,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	businessRequestRef string,
) string {
	return "legacy-authorization-" + canonicalFingerprint(
		"legacy-authorization.v1", kind, principalRef.String(), projectRef.String(), businessRequestRef,
	)
}

func createLegacyGoal(
	t *testing.T,
	repository *Repository,
	state application.CreateGoalState,
) (application.GoalRecord, bool, error) {
	t.Helper()
	return repository.CreateGoal(context.Background(), authorizeLegacyCreateState(t, repository, state))
}

func amendLegacyGoal(
	t *testing.T,
	repository *Repository,
	state application.AmendGoalState,
) (application.GoalRecord, bool, error) {
	t.Helper()
	return repository.AmendGoal(context.Background(), authorizeLegacyAmendState(t, repository, state))
}
