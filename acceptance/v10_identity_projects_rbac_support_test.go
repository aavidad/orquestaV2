package acceptance_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type v10RealSystem struct {
	clock        *v06Clock
	ids          *v06IDs
	v06          v06Fixture
	databasePath string
	repository   *statesqlite.Repository
	orchestrator *application.Orchestrator
	agent        *v06Agent
	artifacts    *v06ArtifactStore
	owner        identity.Principal
	contributor  identity.Principal
	viewer       identity.Principal
	outsider     identity.Principal
	service      identity.Principal
	projectA     identity.ProjectHierarchy
	projectB     identity.ProjectHierarchy
}

type v10GrantResult struct {
	request    identity.MembershipGrantRequest
	membership identity.Membership
	audit      identity.MembershipAuditReceipt
}

func v10NewRealSystem(t *testing.T, fixture v10Fixture) *v10RealSystem {
	t.Helper()
	baseTime, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime)
	if err != nil {
		t.Fatal(err)
	}
	v06 := evidenceDecodeStrictJSON[v06Fixture](
		t, filepath.Join(evidenceRepositoryRoot(t), filepath.FromSlash(v06FixturePath)),
	)
	clock := &v06Clock{now: baseTime}
	databasePath := filepath.Join(t.TempDir(), "state", "v10.sqlite")
	repository, err := statesqlite.Open(context.Background(), statesqlite.Options{
		Path: databasePath, BusyTimeout: 5 * time.Second, MaxOpenConnections: 8, Now: clock.Now,
	})
	if err != nil {
		t.Fatalf("open V10 SQLite: %v", err)
	}
	system := &v10RealSystem{
		clock: clock, ids: &v06IDs{}, v06: v06, databasePath: databasePath, repository: repository,
		artifacts: newV06ArtifactStore(),
	}
	system.owner = v10Principal(t, fixture.Scenario.HumanActorRefs[0], identity.PrincipalKindHuman)
	system.contributor = v10Principal(t, fixture.Scenario.HumanActorRefs[1], identity.PrincipalKindHuman)
	system.viewer = v10Principal(t, fixture.Scenario.HumanActorRefs[2], identity.PrincipalKindHuman)
	system.outsider = v10Principal(t, fixture.Scenario.HumanActorRefs[3], identity.PrincipalKindHuman)
	system.service = v10Principal(t, fixture.Scenario.ServiceActorRef, identity.PrincipalKindService)
	system.projectA = v10Hierarchy(t, fixture, 0)
	system.projectB = v10Hierarchy(t, fixture, 1)
	if err := repository.ProvisionLocalAccess(
		context.Background(), system.owner, system.projectA, identity.RoleProjectOwner, clock.Now(),
	); err != nil {
		_ = repository.Close()
		t.Fatalf("provision project A owner: %v", err)
	}
	if err := repository.ProvisionLocalAccess(
		context.Background(), system.outsider, system.projectB, identity.RoleProjectOwner, clock.Now(),
	); err != nil {
		_ = repository.Close()
		t.Fatalf("provision project B owner: %v", err)
	}
	system.agent = newV06Agent(clock, v06Capabilities(v06.OpaqueRequirements, true), "accepted")
	system.orchestrator = v06NewOrchestrator(
		t, repository, clock, system.ids, system.agent, system.artifacts, v06,
	)
	return system
}

func v10Principal(t *testing.T, actorValue string, kind identity.PrincipalKind) identity.Principal {
	t.Helper()
	actorRef, err := goal.NewActorRef(actorValue)
	if err != nil {
		t.Fatal(err)
	}
	principalValue := strings.Replace(actorValue, "actor:", "principal:", 1)
	principalRef, err := identity.NewPrincipalRef(principalValue)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(principalRef, actorRef, kind, "acceptance.v10")
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func v10Hierarchy(t *testing.T, fixture v10Fixture, index int) identity.ProjectHierarchy {
	t.Helper()
	workspaceRef, err := identity.NewWorkspaceRef(fixture.Scenario.WorkspaceRefs[index])
	if err != nil {
		t.Fatal(err)
	}
	groupRef, err := identity.NewGroupRef(fixture.Scenario.GroupRefs[index])
	if err != nil {
		t.Fatal(err)
	}
	projectRef, err := goal.NewProjectRef(fixture.Scenario.ProjectRefs[index])
	if err != nil {
		t.Fatal(err)
	}
	repositoryRef, err := identity.NewRepositoryRef(fixture.Scenario.RepositoryRefs[index])
	if err != nil {
		t.Fatal(err)
	}
	hierarchy, err := identity.NewProjectHierarchy(identity.ProjectHierarchyInput{
		WorkspaceRef: workspaceRef, GroupRef: groupRef, GroupParentWorkspaceRef: workspaceRef,
		ProjectRef: projectRef, ProjectParentGroupRef: groupRef,
		RepositoryRef: repositoryRef, RepositoryParentProjectRef: projectRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	return hierarchy
}

func v10Access(t *testing.T, principal identity.Principal, projectRef goal.ProjectRef) application.Access {
	t.Helper()
	access, err := application.NewAccess(principal, projectRef)
	if err != nil {
		t.Fatal(err)
	}
	return access
}

func v10GrantActive(
	t *testing.T,
	system *v10RealSystem,
	actor identity.Principal,
	access application.Access,
	target identity.Principal,
	role identity.Role,
	requestRef string,
) v10GrantResult {
	t.Helper()
	request := v10GrantRequest(
		t, requestRef, actor, target.Ref, system.projectA.ProjectRef(), role, 0, system.clock.Now(),
	)
	membership, audit, created, err := system.orchestrator.GrantMembership(
		context.Background(), access, request, target,
	)
	if err != nil || !created || !membership.IsActive() || membership.Revision() != 1 ||
		membership.Role() != role || membership.GrantedBy() != actor.Ref ||
		audit.Action() != identity.MembershipAuditGranted || audit.RequestRef() != requestRef ||
		audit.PreviousRevision() != 0 || audit.Revision() != 1 {
		t.Fatalf("grant %s: membership=%+v audit=%+v created=%v err=%v",
			requestRef, membership.Snapshot(), audit, created, err)
	}
	return v10GrantResult{request: request, membership: membership, audit: audit}
}

func v10GrantRequest(
	t *testing.T,
	requestRef string,
	actor identity.Principal,
	target identity.PrincipalRef,
	projectRef goal.ProjectRef,
	role identity.Role,
	expected identity.MembershipRevision,
	at time.Time,
) identity.MembershipGrantRequest {
	t.Helper()
	request, err := identity.NewMembershipGrantRequest(identity.MembershipGrantRequestInput{
		RequestRef: requestRef, Actor: actor, TargetRef: target, ProjectRef: projectRef,
		Role: role, ExpectedRevision: expected, RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func v10RevokeRequest(
	t *testing.T,
	requestRef string,
	actor identity.Principal,
	target identity.PrincipalRef,
	projectRef goal.ProjectRef,
	expected identity.MembershipRevision,
	at time.Time,
) identity.MembershipRevokeRequest {
	t.Helper()
	request, err := identity.NewMembershipRevokeRequest(identity.MembershipRevokeRequestInput{
		RequestRef: requestRef, Actor: actor, TargetRef: target, ProjectRef: projectRef,
		ExpectedRevision: expected, RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func v10AuthorizationRequest(
	t *testing.T,
	requestRef string,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	permission identity.Permission,
	resourceRef string,
	at time.Time,
) identity.AuthorizationRequest {
	t.Helper()
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: requestRef, Principal: principal, ProjectRef: projectRef,
		Permission: permission, ResourceRef: resourceRef, RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func v10AssertRolePolicy(
	t *testing.T,
	fixture v10Fixture,
	repository *statesqlite.Repository,
	projectRef goal.ProjectRef,
	role identity.Role,
	principals []identity.Principal,
	at time.Time,
) {
	t.Helper()
	for _, principal := range principals {
		for _, rawPermission := range fixture.Scenario.ProtectedActions {
			permission := identity.Permission(rawPermission)
			if err := identity.ValidatePermission(permission); err != nil {
				t.Fatal(err)
			}
			request := v10AuthorizationRequest(
				t, "authorization-request:v10-policy:"+principal.Ref.String()+":"+rawPermission,
				principal, projectRef, permission, "resource:v10-policy:"+rawPermission, at,
			)
			receipt, err := repository.Authorize(context.Background(), request)
			wantAllowed := identity.RoleAllows(role, permission)
			if err != nil || receipt.Decision().Role() != role ||
				(receipt.Decision().Outcome() == identity.AuthorizationAllowed) != wantAllowed {
				t.Fatalf("policy principal=%s role=%s permission=%s receipt=%+v err=%v",
					principal.Ref.String(), role, permission, receipt, err)
			}
		}
	}
}

func v10CloseGoal(t *testing.T, system *v10RealSystem, goalRef goal.GoalRef) application.GoalRecord {
	t.Helper()
	launch, err := system.orchestrator.ProcessNext(context.Background(), "worker:v10")
	if err != nil || !launch.Processed || launch.Action != application.ActionLaunchAgent || launch.GoalRef != goalRef {
		t.Fatalf("launch shared Goal: result=%+v err=%v", launch, err)
	}
	system.clock.Advance(v06Duration(t, system.v06.ClockAndRetry.ObservationDelay))
	observe, err := system.orchestrator.ProcessNext(context.Background(), "worker:v10")
	if err != nil || !observe.Processed || observe.Action != application.ActionObserveAgent || observe.GoalRef != goalRef {
		t.Fatalf("observe shared Goal: result=%+v err=%v", observe, err)
	}
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func v10Forbidden(err error) bool {
	return err != nil && err.Error() == "application.forbidden"
}
