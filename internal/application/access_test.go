package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestProjectAccessAllowsCollaboratorAndHidesOtherProject(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)}
	state := newMemoryRepository()
	accessRepository := newMemoryAccessRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("shared result"),
	}}}
	orchestrator, _ := newTestOrchestratorWithAccess(t, state, accessRepository, clock, agent)
	projectA := mustProjectRef(t, "project:a")
	projectB := mustProjectRef(t, "project:b")
	owner := testPrincipal(t, "principal:owner", "actor:owner", identity.PrincipalKindHuman)
	collaborator := testPrincipal(t, "principal:collaborator", "actor:collaborator", identity.PrincipalKindHuman)
	ownerAccess := mustAccess(t, owner, projectA)
	collaboratorAccess := mustAccess(t, collaborator, projectA)
	foreignAccess := mustAccess(t, collaborator, projectB)
	accessRepository.setRole(owner.Ref, projectA, identity.RoleProjectOwner)
	accessRepository.setRole(collaborator.Ref, projectA, identity.RoleContributor)

	source := v04SubmitAndClose(t, ctx, orchestrator, clock, ownerAccess, SubmitRequest{
		RequestRef: "request:collaboration-source", Statement: "source", Confirm: true,
	})
	if got, err := orchestrator.GetGoal(ctx, collaboratorAccess, source.Goal.Ref()); err != nil || got.Goal.Actor() != owner.ActorRef {
		t.Fatalf("collaborator cannot read project goal: goal=%+v err=%v", got.Goal.Snapshot(), err)
	}
	amended, err := orchestrator.Amend(ctx, collaboratorAccess, v04AmendRequest(source.Goal, "request:collaborator-amend"))
	if err != nil {
		t.Fatalf("collaborator amend: %v", err)
	}
	if amended.Record.Goal.Actor() != owner.ActorRef || amended.Record.Goal.AppSpec().Intent().Actor() != owner.ActorRef ||
		amended.Record.Goal.AppSpec().ConfirmedBy() != collaborator.ActorRef ||
		amended.Record.RequestedBy != collaborator.Ref {
		t.Fatalf("creator/caller attribution crossed: record=%+v spec=%+v", amended.Record, amended.Record.Goal.AppSpec().Snapshot())
	}
	if _, err := orchestrator.GetGoal(ctx, foreignAccess, source.Goal.Ref()); !IsStateError(err, StateNotFound) {
		t.Fatalf("cross-project goal leaked: %v", err)
	}
	listed, err := orchestrator.ListGoals(ctx, foreignAccess, 10)
	if err != nil || len(listed) != 0 {
		t.Fatalf("cross-project list leaked: %+v err=%v", listed, err)
	}
	status, err := orchestrator.Status(ctx, foreignAccess)
	if err != nil || status != (RepositoryStatus{}) {
		t.Fatalf("cross-project status leaked: %+v err=%v", status, err)
	}
}

func TestHumanAndServicePrincipalsFenceBusinessReplayIndependently(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)}
	state := newMemoryRepository()
	accessRepository := newMemoryAccessRepository()
	orchestrator, _ := newTestOrchestratorWithAccess(t, state, accessRepository, clock, &scriptedAgent{now: clock.Now})
	projectRef := mustProjectRef(t, "project:shared")
	human := testPrincipal(t, "principal:human", "actor:shared", identity.PrincipalKindHuman)
	service := testPrincipal(t, "principal:service", "actor:shared", identity.PrincipalKindService)
	humanAccess := mustAccess(t, human, projectRef)
	serviceAccess := mustAccess(t, service, projectRef)
	request := SubmitRequest{RequestRef: "request:same-business-key", Statement: "same", Confirm: true}

	humanResult, err := orchestrator.Submit(ctx, humanAccess, request)
	if err != nil {
		t.Fatal(err)
	}
	serviceResult, err := orchestrator.Submit(ctx, serviceAccess, request)
	if err != nil {
		t.Fatal(err)
	}
	if !humanResult.Created || !serviceResult.Created || humanResult.Record.Goal.Ref() == serviceResult.Record.Goal.Ref() ||
		humanResult.Record.RequestedBy != human.Ref || serviceResult.Record.RequestedBy != service.Ref ||
		submissionFingerprint(humanAccess, request) == submissionFingerprint(serviceAccess, request) {
		t.Fatalf("principal fence collapsed: human=%+v service=%+v", humanResult, serviceResult)
	}
	clock.Advance(time.Minute)
	humanReplay, err := orchestrator.Submit(ctx, humanAccess, request)
	if err != nil || humanReplay.Created || humanReplay.Record.Goal.Ref() != humanResult.Record.Goal.Ref() {
		t.Fatalf("human replay: %+v err=%v", humanReplay, err)
	}
	accessRepository.mu.Lock()
	requests := append([]identity.AuthorizationRequest(nil), accessRepository.authorizations...)
	accessRepository.mu.Unlock()
	seen := make(map[string]bool, len(requests))
	for _, authorization := range requests {
		if seen[authorization.RequestRef()] {
			t.Fatalf("authorization request ref reused: %s", authorization.RequestRef())
		}
		seen[authorization.RequestRef()] = true
	}
}

func TestViewerDeniedBeforeMutationAndAllowedExactReads(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 11, 0, 0, 0, time.UTC)}
	state := newMemoryRepository()
	accessRepository := newMemoryAccessRepository()
	orchestrator, _ := newTestOrchestratorWithAccess(t, state, accessRepository, clock, &scriptedAgent{now: clock.Now})
	projectRef := mustProjectRef(t, "project:viewer")
	owner := testPrincipal(t, "principal:owner-view", "actor:owner-view", identity.PrincipalKindHuman)
	viewer := testPrincipal(t, "principal:viewer", "actor:viewer", identity.PrincipalKindHuman)
	ownerAccess := mustAccess(t, owner, projectRef)
	viewerAccess := mustAccess(t, viewer, projectRef)
	accessRepository.setRole(viewer.Ref, projectRef, identity.RoleViewer)
	submitted, err := orchestrator.Submit(ctx, ownerAccess, SubmitRequest{
		RequestRef: "request:viewer-source", Statement: "visible", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.GetGoal(ctx, viewerAccess, submitted.Record.Goal.Ref()); err != nil {
		t.Fatalf("viewer exact read denied: %v", err)
	}
	before := len(state.records)
	_, err = orchestrator.Submit(ctx, viewerAccess, SubmitRequest{
		RequestRef: "request:viewer-write", Statement: "forbidden", Confirm: true,
	})
	if !errors.Is(err, errForbidden) || len(state.records) != before {
		t.Fatalf("viewer mutation result: records=%d/%d err=%v", len(state.records), before, err)
	}
}

func TestMissingMembershipConcealsObjectReadsButNotOtherDenials(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 11, 30, 0, 0, time.UTC)}
	state := newMemoryRepository()
	accessRepository := newMemoryAccessRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("private"),
	}}}
	orchestrator, _ := newTestOrchestratorWithAccess(t, state, accessRepository, clock, agent)
	projectRef := mustProjectRef(t, "project:private")
	owner := testPrincipal(t, "principal:private-owner", "actor:private-owner", identity.PrincipalKindHuman)
	outsider := testPrincipal(t, "principal:outsider", "actor:outsider", identity.PrincipalKindHuman)
	ownerAccess := mustAccess(t, owner, projectRef)
	outsiderAccess := mustAccess(t, outsider, projectRef)
	accessRepository.setRole(outsider.Ref, projectRef, "")
	source := v04SubmitAndClose(t, ctx, orchestrator, clock, ownerAccess, SubmitRequest{
		RequestRef: "request:private-source", Statement: "private", Confirm: true,
	})
	if _, err := orchestrator.GetGoal(ctx, outsiderAccess, source.Goal.Ref()); !IsStateError(err, StateNotFound) {
		t.Fatalf("missing membership disclosed goal denial: %v", err)
	}
	if _, err := orchestrator.GetArtifact(
		ctx, outsiderAccess, source.Goal.Ref(), source.Artifacts[0].Stored.Ref,
	); !IsStateError(err, StateNotFound) {
		t.Fatalf("missing membership disclosed artifact denial: %v", err)
	}
	if _, err := orchestrator.ListGoals(ctx, outsiderAccess, 10); !errors.Is(err, errForbidden) {
		t.Fatalf("list missing membership did not return forbidden: %v", err)
	}
	if _, err := orchestrator.Status(ctx, outsiderAccess); !errors.Is(err, errForbidden) {
		t.Fatalf("status missing membership did not return forbidden: %v", err)
	}
	if _, err := orchestrator.Submit(ctx, outsiderAccess, SubmitRequest{
		RequestRef: "request:outsider-write", Statement: "no", Confirm: true,
	}); !errors.Is(err, errForbidden) {
		t.Fatalf("write missing membership did not return forbidden: %v", err)
	}
}

func TestAuthorizationReceiptMustMatchExactRequest(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)}
	state := newMemoryRepository()
	accessRepository := newMemoryAccessRepository()
	accessRepository.authorizeHook = func(request identity.AuthorizationRequest) (identity.AuthorizationReceipt, error) {
		wrong, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
			RequestRef: request.RequestRef(), Principal: request.Principal(), ProjectRef: request.ProjectRef(),
			Permission: request.Permission(), ResourceRef: "project:substituted", RequestedAt: request.RequestedAt(),
		})
		if err != nil {
			return identity.AuthorizationReceipt{}, err
		}
		decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
			Request: wrong, Outcome: identity.AuthorizationAllowed, Role: identity.RolePlatformAdmin,
			ReasonCode: "access.allowed", DecidedAt: wrong.RequestedAt(),
		})
		if err != nil {
			return identity.AuthorizationReceipt{}, err
		}
		return identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
			Ref: "authorization-receipt:substituted", Decision: decision, RecordedAt: wrong.RequestedAt(),
		})
	}
	orchestrator, _ := newTestOrchestratorWithAccess(t, state, accessRepository, clock, &scriptedAgent{now: clock.Now})
	projectRef := mustProjectRef(t, "project:exact")
	principal := testPrincipal(t, "principal:exact", "actor:exact", identity.PrincipalKindHuman)
	_, err := orchestrator.Submit(ctx, mustAccess(t, principal, projectRef), SubmitRequest{
		RequestRef: "request:exact", Statement: "must be fenced", Confirm: true,
	})
	if !errors.Is(err, errForbidden) || len(state.records) != 0 {
		t.Fatalf("substituted receipt accepted: records=%d err=%v", len(state.records), err)
	}
}

func TestMembershipGrantRevokeCASAndReplay(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 13, 0, 0, 0, time.UTC)}
	state := newMemoryRepository()
	accessRepository := newMemoryAccessRepository()
	orchestrator, _ := newTestOrchestratorWithAccess(t, state, accessRepository, clock, &scriptedAgent{now: clock.Now})
	projectRef := mustProjectRef(t, "project:members")
	admin := testPrincipal(t, "principal:platform", "actor:platform", identity.PrincipalKindHuman)
	target := testPrincipal(t, "principal:worker-service", "actor:worker-service", identity.PrincipalKindService)
	access := mustAccess(t, admin, projectRef)
	grant := mustGrantRequest(t, "membership-request:grant", admin, target.Ref, projectRef, identity.RoleContributor, 0, clock.Now())

	membership, audit, created, err := orchestrator.GrantMembership(ctx, access, grant, target)
	if err != nil || !created || !membership.IsActive() || membership.Revision() != 1 ||
		audit.Action() != identity.MembershipAuditGranted {
		t.Fatalf("grant: membership=%+v audit=%+v created=%v err=%v", membership.Snapshot(), audit, created, err)
	}
	clock.Advance(time.Minute)
	replayed, replayAudit, created, err := orchestrator.GrantMembership(ctx, access, grant, target)
	if err != nil || created || replayed.Snapshot() != membership.Snapshot() || replayAudit.Ref() != audit.Ref() {
		t.Fatalf("grant replay: membership=%+v created=%v err=%v", replayed.Snapshot(), created, err)
	}
	staleGrant := mustGrantRequest(t, "membership-request:stale-grant", admin, target.Ref, projectRef, identity.RoleReviewer, 0, clock.Now())
	if _, _, _, err := orchestrator.GrantMembership(ctx, access, staleGrant, target); !IsStateError(err, StateConflict) {
		t.Fatalf("stale grant accepted: %v", err)
	}
	revoke := mustRevokeRequest(t, "membership-request:revoke", admin, target.Ref, projectRef, 1, clock.Now())
	revoked, revokeAudit, changed, err := orchestrator.RevokeMembership(ctx, access, revoke)
	if err != nil || !changed || revoked.Status() != identity.MembershipRevoked || revoked.Revision() != 2 ||
		revokeAudit.Action() != identity.MembershipAuditRevoked {
		t.Fatalf("revoke: membership=%+v audit=%+v changed=%v err=%v", revoked.Snapshot(), revokeAudit, changed, err)
	}
	clock.Advance(time.Minute)
	replayedRevoke, replayedAudit, changed, err := orchestrator.RevokeMembership(ctx, access, revoke)
	if err != nil || changed || replayedRevoke.Snapshot() != revoked.Snapshot() || replayedAudit.Ref() != revokeAudit.Ref() {
		t.Fatalf("revoke replay: membership=%+v changed=%v err=%v", replayedRevoke.Snapshot(), changed, err)
	}
	staleRevoke := mustRevokeRequest(t, "membership-request:stale-revoke", admin, target.Ref, projectRef, 1, clock.Now())
	if _, _, _, err := orchestrator.RevokeMembership(ctx, access, staleRevoke); !IsStateError(err, StateConflict) {
		t.Fatalf("stale revoke accepted: %v", err)
	}
}

func TestProjectAdminCannotTouchAdministrativeMemberships(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 14, 0, 0, 0, time.UTC)}
	state := newMemoryRepository()
	accessRepository := newMemoryAccessRepository()
	orchestrator, _ := newTestOrchestratorWithAccess(t, state, accessRepository, clock, &scriptedAgent{now: clock.Now})
	projectRef := mustProjectRef(t, "project:delegation")
	admin := testPrincipal(t, "principal:project-admin", "actor:project-admin", identity.PrincipalKindHuman)
	owner := testPrincipal(t, "principal:project-owner", "actor:project-owner", identity.PrincipalKindHuman)
	newTarget := testPrincipal(t, "principal:new-target", "actor:new-target", identity.PrincipalKindHuman)
	adminMembership := mustMembership(t, admin.Ref, projectRef, identity.RoleProjectAdmin, admin.Ref, clock.Now())
	ownerMembership := mustMembership(t, owner.Ref, projectRef, identity.RoleProjectOwner, owner.Ref, clock.Now())
	accessRepository.seedMembership(adminMembership)
	accessRepository.seedMembership(ownerMembership)
	access := mustAccess(t, admin, projectRef)

	revokeOwner := mustRevokeRequest(t, "membership-request:revoke-owner", admin, owner.Ref, projectRef, 1, clock.Now())
	if _, _, _, err := orchestrator.RevokeMembership(ctx, access, revokeOwner); !errors.Is(err, errForbidden) {
		t.Fatalf("project admin revoked owner: %v", err)
	}
	demoteOwner := mustGrantRequest(t, "membership-request:demote-owner", admin, owner.Ref, projectRef, identity.RoleViewer, 1, clock.Now())
	if _, _, _, err := orchestrator.GrantMembership(ctx, access, demoteOwner, owner); !errors.Is(err, errForbidden) {
		t.Fatalf("project admin demoted owner: %v", err)
	}
	grantAdmin := mustGrantRequest(t, "membership-request:grant-admin", admin, newTarget.Ref, projectRef, identity.RoleProjectAdmin, 0, clock.Now())
	if _, _, _, err := orchestrator.GrantMembership(ctx, access, grantAdmin, newTarget); !errors.Is(err, errForbidden) {
		t.Fatalf("project admin granted admin: %v", err)
	}
	grantPlatform := mustGrantRequest(t, "membership-request:grant-platform", admin, newTarget.Ref, projectRef, identity.RolePlatformAdmin, 0, clock.Now())
	if _, _, _, err := orchestrator.GrantMembership(ctx, access, grantPlatform, newTarget); !errors.Is(err, errForbidden) {
		t.Fatalf("project platform_admin grant accepted: %v", err)
	}
	grantViewer := mustGrantRequest(t, "membership-request:grant-viewer", admin, newTarget.Ref, projectRef, identity.RoleViewer, 0, clock.Now())
	if _, _, created, err := orchestrator.GrantMembership(ctx, access, grantViewer, newTarget); err != nil || !created {
		t.Fatalf("project admin could not grant viewer: created=%v err=%v", created, err)
	}
}

func mustProjectRef(t *testing.T, value string) goal.ProjectRef {
	t.Helper()
	ref, err := goal.NewProjectRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func mustAccess(t *testing.T, principal identity.Principal, projectRef goal.ProjectRef) Access {
	t.Helper()
	access, err := NewAccess(principal, projectRef)
	if err != nil {
		t.Fatal(err)
	}
	return access
}

func mustGrantRequest(
	t *testing.T,
	requestRef string,
	actor identity.Principal,
	targetRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	role identity.Role,
	expected identity.MembershipRevision,
	at time.Time,
) identity.MembershipGrantRequest {
	t.Helper()
	request, err := identity.NewMembershipGrantRequest(identity.MembershipGrantRequestInput{
		RequestRef: requestRef, Actor: actor, TargetRef: targetRef, ProjectRef: projectRef,
		Role: role, ExpectedRevision: expected, RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func mustRevokeRequest(
	t *testing.T,
	requestRef string,
	actor identity.Principal,
	targetRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	expected identity.MembershipRevision,
	at time.Time,
) identity.MembershipRevokeRequest {
	t.Helper()
	request, err := identity.NewMembershipRevokeRequest(identity.MembershipRevokeRequestInput{
		RequestRef: requestRef, Actor: actor, TargetRef: targetRef, ProjectRef: projectRef,
		ExpectedRevision: expected, RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func mustMembership(
	t *testing.T,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	role identity.Role,
	grantedBy identity.PrincipalRef,
	at time.Time,
) identity.Membership {
	t.Helper()
	membership, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: principalRef, ProjectRef: projectRef, Role: role,
		Revision: 1, Status: identity.MembershipActive, GrantedBy: grantedBy, GrantedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return membership
}
