package application

import (
	"context"
	"sync"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type memoryAccessRepository struct {
	mu             sync.Mutex
	defaultRole    identity.Role
	roles          map[string]identity.Role
	memberships    map[string]identity.Membership
	mutations      map[string]memoryMembershipMutation
	authorizations []identity.AuthorizationRequest
	authorizeHook  func(identity.AuthorizationRequest) (identity.AuthorizationReceipt, error)
}

type memoryMembershipMutation struct {
	grant      identity.MembershipGrantRequest
	revoke     identity.MembershipRevokeRequest
	membership identity.Membership
	audit      identity.MembershipAuditReceipt
}

func newMemoryAccessRepository() *memoryAccessRepository {
	return &memoryAccessRepository{
		defaultRole: identity.RolePlatformAdmin,
		roles:       make(map[string]identity.Role),
		memberships: make(map[string]identity.Membership),
		mutations:   make(map[string]memoryMembershipMutation),
	}
}

func (repository *memoryAccessRepository) setRole(principal identity.PrincipalRef, project goal.ProjectRef, role identity.Role) {
	repository.mu.Lock()
	repository.roles[memoryMembershipKey(principal, project)] = role
	repository.mu.Unlock()
}

func (repository *memoryAccessRepository) seedMembership(membership identity.Membership) {
	repository.mu.Lock()
	key := memoryMembershipKey(membership.PrincipalRef(), membership.ProjectRef())
	repository.memberships[key] = membership
	if membership.IsActive() {
		repository.roles[key] = membership.Role()
	} else {
		repository.roles[key] = ""
	}
	repository.mu.Unlock()
}

func (repository *memoryAccessRepository) Authorize(
	_ context.Context,
	request identity.AuthorizationRequest,
) (identity.AuthorizationReceipt, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.authorizations = append(repository.authorizations, request)
	if repository.authorizeHook != nil {
		return repository.authorizeHook(request)
	}
	role, exists := repository.roles[memoryMembershipKey(request.Principal().Ref, request.ProjectRef())]
	if !exists {
		role = repository.defaultRole
	}
	outcome := identity.AuthorizationDenied
	reason := "access.denied"
	revision := identity.MembershipRevision(0)
	if identity.RoleAllows(role, request.Permission()) {
		outcome = identity.AuthorizationAllowed
		reason = "access.allowed"
		if role != identity.RolePlatformAdmin {
			revision = 1
		}
	}
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: request, Outcome: outcome, Role: role,
		MembershipRevision: revision, ReasonCode: reason, DecidedAt: request.RequestedAt(),
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	return identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref: "authorization-receipt:" + request.RequestRef(), Decision: decision,
		RecordedAt: request.RequestedAt(),
	})
}

func (repository *memoryAccessRepository) Membership(
	_ context.Context,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
) (identity.Membership, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	membership, ok := repository.memberships[memoryMembershipKey(principalRef, projectRef)]
	if !ok {
		return identity.Membership{}, &StateError{Code: StateNotFound}
	}
	return membership, nil
}

func (repository *memoryAccessRepository) GrantMembership(
	_ context.Context,
	state MembershipGrantState,
) (identity.Membership, identity.MembershipAuditReceipt, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	request := state.Request
	mutationKey := memoryMutationKey(request.Actor().Ref, request.ProjectRef(), request.RequestRef())
	if previous, exists := repository.mutations[mutationKey]; exists {
		if previous.grant != request {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateConflict}
		}
		return previous.membership, previous.audit, false, nil
	}
	if !memoryMembershipAuthorizationValid(
		state.AuthorizationReceipt, request.Actor(), request.ProjectRef(), request.TargetRef(),
	) || state.Target.Ref != request.TargetRef() {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateInvalid}
	}
	key := memoryMembershipKey(request.TargetRef(), request.ProjectRef())
	current, exists := repository.memberships[key]
	currentRevision := identity.MembershipRevision(0)
	if exists {
		currentRevision = current.Revision()
	}
	if currentRevision != request.ExpectedRevision() {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateConflict}
	}
	membership, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: request.TargetRef(), ProjectRef: request.ProjectRef(), Role: request.Role(),
		Revision: currentRevision + 1, Status: identity.MembershipActive,
		GrantedBy: request.Actor().Ref, GrantedAt: request.RequestedAt(),
	})
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	audit, err := identity.NewMembershipAuditReceipt(identity.MembershipAuditReceiptInput{
		Ref: "membership-audit:" + request.RequestRef(), RequestRef: request.RequestRef(),
		Action: identity.MembershipAuditGranted, ActorRef: request.Actor().Ref,
		TargetRef: request.TargetRef(), ProjectRef: request.ProjectRef(), Role: request.Role(),
		PreviousRevision: currentRevision, Revision: currentRevision + 1, OccurredAt: request.RequestedAt(),
	})
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	repository.memberships[key] = membership
	repository.roles[key] = request.Role()
	repository.mutations[mutationKey] = memoryMembershipMutation{grant: request, membership: membership, audit: audit}
	return membership, audit, true, nil
}

func (repository *memoryAccessRepository) RevokeMembership(
	_ context.Context,
	state MembershipRevokeState,
) (identity.Membership, identity.MembershipAuditReceipt, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	request := state.Request
	mutationKey := memoryMutationKey(request.Actor().Ref, request.ProjectRef(), request.RequestRef())
	if previous, exists := repository.mutations[mutationKey]; exists {
		if previous.revoke != request {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateConflict}
		}
		return previous.membership, previous.audit, false, nil
	}
	if !memoryMembershipAuthorizationValid(
		state.AuthorizationReceipt, request.Actor(), request.ProjectRef(), request.TargetRef(),
	) {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateInvalid}
	}
	key := memoryMembershipKey(request.TargetRef(), request.ProjectRef())
	current, exists := repository.memberships[key]
	if !exists || !current.IsActive() || current.Revision() != request.ExpectedRevision() {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateConflict}
	}
	membership, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: current.PrincipalRef(), ProjectRef: current.ProjectRef(), Role: current.Role(),
		Revision: current.Revision() + 1, Status: identity.MembershipRevoked,
		GrantedBy: current.GrantedBy(), GrantedAt: current.GrantedAt(),
		RevokedBy: request.Actor().Ref, RevokedAt: request.RequestedAt(),
	})
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	audit, err := identity.NewMembershipAuditReceipt(identity.MembershipAuditReceiptInput{
		Ref: "membership-audit:" + request.RequestRef(), RequestRef: request.RequestRef(),
		Action: identity.MembershipAuditRevoked, ActorRef: request.Actor().Ref,
		TargetRef: request.TargetRef(), ProjectRef: request.ProjectRef(), Role: current.Role(),
		PreviousRevision: current.Revision(), Revision: current.Revision() + 1, OccurredAt: request.RequestedAt(),
	})
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	repository.memberships[key] = membership
	repository.roles[key] = ""
	repository.mutations[mutationKey] = memoryMembershipMutation{revoke: request, membership: membership, audit: audit}
	return membership, audit, true, nil
}

func memoryMembershipAuthorizationValid(
	receipt identity.AuthorizationReceipt,
	actor identity.Principal,
	projectRef goal.ProjectRef,
	targetRef identity.PrincipalRef,
) bool {
	request := receipt.Decision().Request()
	return receipt.Decision().Outcome() == identity.AuthorizationAllowed &&
		identity.RoleAllows(receipt.Decision().Role(), identity.PermissionProjectMembershipManage) &&
		request.Principal() == actor && request.ProjectRef() == projectRef &&
		request.Permission() == identity.PermissionProjectMembershipManage &&
		request.ResourceRef() == targetRef.String()
}

func memoryMembershipKey(principalRef identity.PrincipalRef, projectRef goal.ProjectRef) string {
	return principalRef.String() + "\x00" + projectRef.String()
}

func memoryMutationKey(actorRef identity.PrincipalRef, projectRef goal.ProjectRef, requestRef string) string {
	return actorRef.String() + "\x00" + projectRef.String() + "\x00" + requestRef
}

func accessForScope(
	t interface{ Fatalf(string, ...any) },
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
) Access {
	principalRef, err := identity.NewPrincipalRef(actorRef.String())
	if err != nil {
		t.Fatalf("principal ref: %v", err)
	}
	principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindHuman, "test")
	if err != nil {
		t.Fatalf("principal: %v", err)
	}
	access, err := NewAccess(principal, projectRef)
	if err != nil {
		t.Fatalf("access: %v", err)
	}
	return access
}

func testPrincipal(t *testing.T, ref, actor string, kind identity.PrincipalKind) identity.Principal {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef(ref)
	if err != nil {
		t.Fatal(err)
	}
	actorRef, err := goal.NewActorRef(actor)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(principalRef, actorRef, kind, "test")
	if err != nil {
		t.Fatal(err)
	}
	return principal
}
