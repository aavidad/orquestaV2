package identity

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestOpaqueIdentityRefsPreserveValuesWithoutPathSemantics(t *testing.T) {
	valid := []string{"plain", "../opaque/not-a-path", "https://example.invalid/repo", "ámbito:uno"}
	for _, value := range valid {
		t.Run(value, func(t *testing.T) {
			principal, principalErr := NewPrincipalRef(value)
			workspace, workspaceErr := NewWorkspaceRef(value)
			group, groupErr := NewGroupRef(value)
			repository, repositoryErr := NewRepositoryRef(value)
			if principalErr != nil || workspaceErr != nil || groupErr != nil || repositoryErr != nil {
				t.Fatalf("opaque value rejected: %v %v %v %v", principalErr, workspaceErr, groupErr, repositoryErr)
			}
			if principal.String() != value || workspace.String() != value || group.String() != value || repository.String() != value {
				t.Fatal("opaque value changed")
			}
		})
	}
	for _, value := range []string{"", " leading", "trailing ", "null\x00ref", "line\nref"} {
		if ref, err := NewPrincipalRef(value); err == nil || ref.String() != "" {
			t.Errorf("invalid principal ref accepted: %q", value)
		}
		if ref, err := NewWorkspaceRef(value); err == nil || ref.String() != "" {
			t.Errorf("invalid workspace ref accepted: %q", value)
		}
		if ref, err := NewGroupRef(value); err == nil || ref.String() != "" {
			t.Errorf("invalid group ref accepted: %q", value)
		}
		if ref, err := NewRepositoryRef(value); err == nil || ref.String() != "" {
			t.Errorf("invalid repository ref accepted: %q", value)
		}
	}
}

func TestProjectHierarchyRequiresOneExactOpaqueParentChain(t *testing.T) {
	workspace, _ := NewWorkspaceRef("workspace:alpha")
	otherWorkspace, _ := NewWorkspaceRef("workspace:beta")
	group, _ := NewGroupRef("group:alpha")
	otherGroup, _ := NewGroupRef("group:beta")
	project, _ := goal.NewProjectRef("project:alpha")
	otherProject, _ := goal.NewProjectRef("project:beta")
	repository, _ := NewRepositoryRef("repository:alpha")
	valid := ProjectHierarchyInput{
		WorkspaceRef: workspace, GroupRef: group, GroupParentWorkspaceRef: workspace,
		ProjectRef: project, ProjectParentGroupRef: group, RepositoryRef: repository,
		RepositoryParentProjectRef: project,
	}
	hierarchy, err := NewProjectHierarchy(valid)
	if err != nil {
		t.Fatalf("NewProjectHierarchy() error = %v", err)
	}
	if !reflect.DeepEqual(hierarchy.Snapshot(), valid) {
		t.Fatalf("hierarchy snapshot = %+v", hierarchy.Snapshot())
	}

	cases := []struct {
		name string
		edit func(*ProjectHierarchyInput)
	}{
		{"missing", func(input *ProjectHierarchyInput) { input.RepositoryRef = RepositoryRef{} }},
		{"group_parent", func(input *ProjectHierarchyInput) { input.GroupParentWorkspaceRef = otherWorkspace }},
		{"project_parent", func(input *ProjectHierarchyInput) { input.ProjectParentGroupRef = otherGroup }},
		{"repository_parent", func(input *ProjectHierarchyInput) { input.RepositoryParentProjectRef = otherProject }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := valid
			test.edit(&candidate)
			if _, err := NewProjectHierarchy(candidate); err == nil {
				t.Fatal("invalid hierarchy accepted")
			}
		})
	}
}

func TestRoleAllowsUsesExactDefaultDenyMatrix(t *testing.T) {
	roles := []Role{
		RolePlatformAdmin, RoleProjectOwner, RoleProjectAdmin, RoleContributor,
		RoleReviewer, RoleOperator, RoleViewer,
	}
	permissions := []Permission{
		PermissionProjectHierarchyManage, PermissionProjectMembershipManage,
		PermissionGoalsCreate, PermissionGoalsAmend, PermissionGoalsGet,
		PermissionGoalsList, PermissionBudgetsManage, PermissionEffectsApprove,
		PermissionCouncilSkip, PermissionArtifactsRead, PermissionProjectStatus,
	}
	want := map[Role]map[Permission]bool{
		RolePlatformAdmin: allowAll(permissions),
		RoleProjectOwner:  allowAll(permissions),
		RoleProjectAdmin: allowed(
			PermissionProjectMembershipManage, PermissionGoalsCreate, PermissionGoalsAmend,
			PermissionGoalsGet, PermissionGoalsList, PermissionBudgetsManage,
			PermissionEffectsApprove, PermissionArtifactsRead, PermissionProjectStatus,
		),
		RoleContributor: allowed(
			PermissionGoalsCreate, PermissionGoalsAmend, PermissionGoalsGet,
			PermissionGoalsList, PermissionArtifactsRead, PermissionProjectStatus,
		),
		RoleReviewer: allowed(
			PermissionGoalsGet, PermissionGoalsList, PermissionEffectsApprove,
			PermissionArtifactsRead, PermissionProjectStatus,
		),
		RoleOperator: allowed(
			PermissionGoalsCreate, PermissionGoalsGet, PermissionGoalsList,
			PermissionEffectsApprove, PermissionCouncilSkip, PermissionArtifactsRead, PermissionProjectStatus,
		),
		RoleViewer: allowed(
			PermissionGoalsGet, PermissionGoalsList, PermissionArtifactsRead, PermissionProjectStatus,
		),
	}
	for _, role := range roles {
		if err := ValidateRole(role); err != nil {
			t.Errorf("canonical role %q invalid: %v", role, err)
		}
		for _, permission := range permissions {
			if err := ValidatePermission(permission); err != nil {
				t.Errorf("canonical permission %q invalid: %v", permission, err)
			}
			if got := RoleAllows(role, permission); got != want[role][permission] {
				t.Errorf("RoleAllows(%q,%q)=%v want %v", role, permission, got, want[role][permission])
			}
		}
	}
	for _, pair := range [][2]string{{"", "goals.get"}, {"owner", "goals.get"}, {"viewer", ""}, {"viewer", "unknown"}} {
		if RoleAllows(Role(pair[0]), Permission(pair[1])) {
			t.Errorf("unknown policy pair allowed: %q/%q", pair[0], pair[1])
		}
	}
}

func TestMembershipDelegationUsesOneExactDefaultDenyMatrix(t *testing.T) {
	roles := []Role{
		RolePlatformAdmin, RoleProjectOwner, RoleProjectAdmin, RoleContributor,
		RoleReviewer, RoleOperator, RoleViewer,
	}
	want := map[Role]map[Role]bool{
		RolePlatformAdmin: allowedRoles(
			RoleProjectOwner, RoleProjectAdmin, RoleContributor, RoleReviewer, RoleOperator, RoleViewer,
		),
		RoleProjectOwner: allowedRoles(
			RoleProjectOwner, RoleProjectAdmin, RoleContributor, RoleReviewer, RoleOperator, RoleViewer,
		),
		RoleProjectAdmin: allowedRoles(RoleContributor, RoleReviewer, RoleOperator, RoleViewer),
	}
	for _, grantor := range roles {
		for _, target := range roles {
			if got := CanDelegateMembershipRole(grantor, target); got != want[grantor][target] {
				t.Errorf("CanDelegateMembershipRole(%q,%q)=%v want %v", grantor, target, got, want[grantor][target])
			}
		}
	}
	for _, pair := range [][2]Role{{"", RoleViewer}, {RoleProjectOwner, ""}, {"owner", RoleViewer}, {RoleProjectOwner, "read_only"}} {
		if CanDelegateMembershipRole(pair[0], pair[1]) {
			t.Errorf("unknown delegation pair allowed: %q/%q", pair[0], pair[1])
		}
	}
	for _, test := range []struct {
		role Role
		want bool
	}{{RolePlatformAdmin, true}, {RoleProjectOwner, true}, {RoleProjectAdmin, false}, {RoleViewer, false}, {"", false}} {
		if got := IsProjectAuthority(test.role); got != test.want {
			t.Errorf("IsProjectAuthority(%q)=%v want %v", test.role, got, test.want)
		}
	}
}

func TestRequestPrincipalContextRejectsMissingInvalidAndSpoofedValues(t *testing.T) {
	human := testPrincipal(t, "human", PrincipalKindHuman)
	service := testPrincipal(t, "service", PrincipalKindService)
	for _, principal := range []Principal{human, service} {
		bound, err := BindPrincipal(context.Background(), principal)
		if err != nil {
			t.Fatalf("BindPrincipal(%s) error = %v", principal.Kind, err)
		}
		got, err := (ContextProvider{}).Principal(bound)
		if err != nil || got != principal {
			t.Fatalf("ContextProvider(%s) = %+v, %v", principal.Kind, got, err)
		}
		if _, err := BindPrincipal(bound, principal); err == nil {
			t.Error("principal context was rebound")
		}
	}
	if _, err := PrincipalFromContext(context.Background()); err == nil {
		t.Error("missing principal accepted")
	}
	type publicSpoofKey string
	spoofed := context.WithValue(context.Background(), publicSpoofKey("principal"), human)
	if _, err := PrincipalFromContext(spoofed); err == nil {
		t.Error("public context key spoofed authenticated principal")
	}
	if _, err := BindPrincipal(nil, human); err == nil {
		t.Error("nil context accepted")
	}
	if _, err := BindPrincipal(context.Background(), Principal{}); err == nil {
		t.Error("invalid principal accepted")
	}
}

func TestPrincipalKindsAndAuthenticationMethodValidate(t *testing.T) {
	ref, _ := NewPrincipalRef("principal:one")
	actor, _ := goal.NewActorRef("actor:one")
	for _, kind := range []PrincipalKind{PrincipalKindHuman, PrincipalKindService} {
		principal, err := NewPrincipal(ref, actor, kind, "test_method")
		if err != nil || principal.Ref != ref || principal.ActorRef != actor || principal.Kind != kind {
			t.Errorf("NewPrincipal(%q) = %+v, %v", kind, principal, err)
		}
	}
	for _, test := range []struct {
		ref    PrincipalRef
		actor  goal.ActorRef
		kind   PrincipalKind
		method string
	}{
		{actor: actor, kind: PrincipalKindHuman, method: "method"},
		{ref: ref, kind: PrincipalKindHuman, method: "method"},
		{ref: ref, actor: actor, kind: PrincipalKind("robot"), method: "method"},
		{ref: ref, actor: actor, kind: PrincipalKindHuman, method: ""},
		{ref: ref, actor: actor, kind: PrincipalKindHuman, method: "line\nmethod"},
	} {
		if _, err := NewPrincipal(test.ref, test.actor, test.kind, test.method); err == nil {
			t.Errorf("invalid principal accepted: %+v", test)
		}
	}
}

func TestMembershipStateAndMutationDTOsAreValidatedValueObjects(t *testing.T) {
	now := testTime()
	project := testProject(t)
	actor := testPrincipal(t, "admin", PrincipalKindHuman)
	target, _ := NewPrincipalRef("principal:member")
	activeInput := MembershipInput{
		PrincipalRef: target, ProjectRef: project, Role: RoleContributor,
		Revision: 1, Status: MembershipActive, GrantedBy: actor.Ref, GrantedAt: now,
	}
	active, err := NewMembership(activeInput)
	if err != nil {
		t.Fatalf("NewMembership(active) error = %v", err)
	}
	activeInput.Role = RoleViewer
	activeInput.Revision = 99
	if active.Role() != RoleContributor || active.Revision() != 1 || !active.IsActive() ||
		active.GrantedAt().Location() != time.UTC {
		t.Fatalf("active membership was mutable or not canonical: %+v", active.Snapshot())
	}

	revokedBy, _ := NewPrincipalRef("principal:revoker")
	revoked, err := NewMembership(MembershipInput{
		PrincipalRef: target, ProjectRef: project, Role: RoleContributor,
		Revision: 2, Status: MembershipRevoked, GrantedBy: actor.Ref, GrantedAt: now,
		RevokedBy: revokedBy, RevokedAt: now.Add(time.Minute),
	})
	if err != nil || revoked.IsActive() || revoked.RevokedBy() != revokedBy {
		t.Fatalf("revoked membership = %+v, %v", revoked.Snapshot(), err)
	}

	invalidMemberships := []MembershipInput{
		{},
		{PrincipalRef: target, ProjectRef: project, Role: Role("unknown"), Revision: 1, Status: MembershipActive, GrantedBy: actor.Ref, GrantedAt: now},
		{PrincipalRef: target, ProjectRef: project, Role: RoleViewer, Revision: 1, Status: MembershipActive, GrantedBy: actor.Ref, GrantedAt: now, RevokedBy: revokedBy},
		{PrincipalRef: target, ProjectRef: project, Role: RoleViewer, Revision: 1, Status: MembershipRevoked, GrantedBy: actor.Ref, GrantedAt: now, RevokedBy: revokedBy, RevokedAt: now},
		{PrincipalRef: target, ProjectRef: project, Role: RoleViewer, Revision: 2, Status: MembershipRevoked, GrantedBy: actor.Ref, GrantedAt: now, RevokedBy: revokedBy, RevokedAt: now.Add(-time.Second)},
	}
	for index, input := range invalidMemberships {
		if _, err := NewMembership(input); err == nil {
			t.Errorf("invalid membership %d accepted", index)
		}
	}

	grantInput := MembershipGrantRequestInput{
		RequestRef: "request:grant", Actor: actor, TargetRef: target, ProjectRef: project,
		Role: RoleContributor, ExpectedRevision: 0, RequestedAt: now,
	}
	grant, err := NewMembershipGrantRequest(grantInput)
	if err != nil {
		t.Fatalf("NewMembershipGrantRequest() error = %v", err)
	}
	grantInput.RequestRef = "changed"
	grantInput.Role = RoleViewer
	if grant.RequestRef() != "request:grant" || grant.Role() != RoleContributor ||
		grant.Actor() != actor || grant.TargetRef() != target || grant.ProjectRef() != project {
		t.Fatal("grant request changed through input alias")
	}

	revoke, err := NewMembershipRevokeRequest(MembershipRevokeRequestInput{
		RequestRef: "request:revoke", Actor: actor, TargetRef: target, ProjectRef: project,
		ExpectedRevision: 1, RequestedAt: now.Add(time.Minute),
	})
	if err != nil || revoke.ExpectedRevision() != 1 || revoke.TargetRef() != target {
		t.Fatalf("revoke request = %+v, %v", revoke, err)
	}
	if _, err := NewMembershipRevokeRequest(MembershipRevokeRequestInput{
		RequestRef: "request:revoke", Actor: actor, TargetRef: target,
		ProjectRef: project, RequestedAt: now,
	}); err == nil {
		t.Error("revoke without expected revision accepted")
	}

	for _, request := range []MembershipGrantRequestInput{
		{},
		{RequestRef: " request", Actor: actor, TargetRef: target, ProjectRef: project, Role: RoleViewer, RequestedAt: now},
		{RequestRef: "request", Actor: Principal{}, TargetRef: target, ProjectRef: project, Role: RoleViewer, RequestedAt: now},
		{RequestRef: "request", Actor: actor, TargetRef: target, ProjectRef: project, Role: Role("unknown"), RequestedAt: now},
	} {
		if _, err := NewMembershipGrantRequest(request); err == nil {
			t.Errorf("invalid grant accepted: %+v", request)
		}
	}
}

func TestAuthorizationAndAuditReceiptsBindPrincipalRoleRevisionAndTime(t *testing.T) {
	now := testTime()
	project := testProject(t)
	for _, kind := range []PrincipalKind{PrincipalKindHuman, PrincipalKindService} {
		principal := testPrincipal(t, string(kind), kind)
		request, err := NewAuthorizationRequest(AuthorizationRequestInput{
			RequestRef: "request:authorize:" + string(kind), Principal: principal,
			ProjectRef: project, Permission: PermissionGoalsAmend,
			ResourceRef: "goal:shared", RequestedAt: now,
		})
		if err != nil {
			t.Fatalf("NewAuthorizationRequest(%s) error = %v", kind, err)
		}
		decision, err := NewAuthorizationDecision(AuthorizationDecisionInput{
			Request: request, Outcome: AuthorizationAllowed, Role: RoleContributor,
			MembershipRevision: 3, ReasonCode: "rbac.allowed", DecidedAt: now.Add(time.Second),
		})
		if err != nil {
			t.Fatalf("NewAuthorizationDecision(%s) error = %v", kind, err)
		}
		receipt, err := NewAuthorizationReceipt(AuthorizationReceiptInput{
			Ref: "audit:authorize:" + string(kind), Decision: decision,
			RecordedAt: now.Add(2 * time.Second),
		})
		if err != nil {
			t.Fatalf("NewAuthorizationReceipt(%s) error = %v", kind, err)
		}
		if receipt.Decision().Request().Principal() != principal ||
			receipt.Decision().Outcome() != AuthorizationAllowed ||
			receipt.Decision().MembershipRevision() != 3 ||
			receipt.Decision().DecidedAt().Location() != time.UTC || receipt.RecordedAt().Location() != time.UTC {
			t.Fatalf("authorization receipt lost identity: %+v", receipt)
		}
	}

	principal := testPrincipal(t, "viewer", PrincipalKindHuman)
	request, _ := NewAuthorizationRequest(AuthorizationRequestInput{
		RequestRef: "request:deny", Principal: principal, ProjectRef: project,
		Permission: PermissionGoalsAmend, ResourceRef: "goal:hidden", RequestedAt: now,
	})
	for _, input := range []AuthorizationRequestInput{
		{},
		{RequestRef: " request", Principal: principal, ProjectRef: project, Permission: PermissionGoalsGet, ResourceRef: "goal:x", RequestedAt: now},
		{RequestRef: "request", Principal: Principal{}, ProjectRef: project, Permission: PermissionGoalsGet, ResourceRef: "goal:x", RequestedAt: now},
		{RequestRef: "request", Principal: principal, ProjectRef: project, Permission: Permission("unknown"), ResourceRef: "goal:x", RequestedAt: now},
		{RequestRef: "request", Principal: principal, ProjectRef: project, Permission: PermissionGoalsGet, ResourceRef: "", RequestedAt: now},
	} {
		if _, err := NewAuthorizationRequest(input); err == nil {
			t.Errorf("invalid authorization request accepted: %+v", input)
		}
	}
	deniedDecision, err := NewAuthorizationDecision(AuthorizationDecisionInput{
		Request: request, Outcome: AuthorizationDenied,
		Role: RoleViewer, MembershipRevision: 1, ReasonCode: "rbac.permission_denied",
		DecidedAt: now,
	})
	if err != nil || deniedDecision.Outcome() != AuthorizationDenied {
		t.Fatalf("denied decision = %+v, %v", deniedDecision, err)
	}
	for _, input := range []AuthorizationDecisionInput{
		{},
		{Request: request, Outcome: AuthorizationAllowed, Role: RoleViewer, MembershipRevision: 1, ReasonCode: "allowed", DecidedAt: now},
		{Request: request, Outcome: AuthorizationOutcome("maybe"), ReasonCode: "unknown", DecidedAt: now},
		{Request: request, Outcome: AuthorizationDenied, ReasonCode: "denied", DecidedAt: now.Add(-time.Second)},
	} {
		if _, err := NewAuthorizationDecision(input); err == nil {
			t.Errorf("invalid authorization decision accepted: %+v", input)
		}
	}
	deniedReceipt, err := NewAuthorizationReceipt(AuthorizationReceiptInput{
		Ref: "audit:deny", Decision: deniedDecision, RecordedAt: now.Add(time.Second),
	})
	if err != nil || deniedReceipt.Decision().Outcome() != AuthorizationDenied {
		t.Fatalf("denied receipt = %+v, %v", deniedReceipt, err)
	}
	for _, input := range []AuthorizationReceiptInput{
		{},
		{Ref: "audit:early", Decision: deniedDecision, RecordedAt: now.Add(-time.Second)},
	} {
		if _, err := NewAuthorizationReceipt(input); err == nil {
			t.Errorf("invalid authorization receipt accepted: %+v", input)
		}
	}

	target, _ := NewPrincipalRef("principal:member")
	grantReceiptInput := MembershipAuditReceiptInput{
		Ref: "audit:grant", RequestRef: "request:grant", Action: MembershipAuditGranted,
		ActorRef: principal.Ref, TargetRef: target, ProjectRef: project, Role: RoleViewer,
		PreviousRevision: 0, Revision: 1, OccurredAt: now,
	}
	grantReceipt, err := NewMembershipAuditReceipt(grantReceiptInput)
	if err != nil {
		t.Fatalf("NewMembershipAuditReceipt(grant) error = %v", err)
	}
	grantReceiptInput.Role = RoleContributor
	if grantReceipt.Role() != RoleViewer || grantReceipt.Revision() != 1 {
		t.Fatal("membership audit receipt changed through input alias")
	}
	revokeReceipt, err := NewMembershipAuditReceipt(MembershipAuditReceiptInput{
		Ref: "audit:revoke", RequestRef: "request:revoke", Action: MembershipAuditRevoked,
		ActorRef: principal.Ref, TargetRef: target, ProjectRef: project, Role: RoleViewer,
		PreviousRevision: 1, Revision: 2, OccurredAt: now.Add(time.Minute),
	})
	if err != nil || revokeReceipt.Action() != MembershipAuditRevoked {
		t.Fatalf("revoke audit receipt = %+v, %v", revokeReceipt, err)
	}
	for _, input := range []MembershipAuditReceiptInput{
		{},
		{Ref: "audit", RequestRef: "request", Action: MembershipAuditRevoked, ActorRef: principal.Ref, TargetRef: target, ProjectRef: project, Role: RoleViewer, PreviousRevision: 0, Revision: 1, OccurredAt: now},
		{Ref: "audit", RequestRef: "request", Action: MembershipAuditGranted, ActorRef: principal.Ref, TargetRef: target, ProjectRef: project, Role: RoleViewer, PreviousRevision: 1, Revision: 3, OccurredAt: now},
	} {
		if _, err := NewMembershipAuditReceipt(input); err == nil {
			t.Errorf("invalid membership audit accepted: %+v", input)
		}
	}
}

func allowAll(permissions []Permission) map[Permission]bool {
	result := make(map[Permission]bool, len(permissions))
	for _, permission := range permissions {
		result[permission] = true
	}
	return result
}

func allowed(permissions ...Permission) map[Permission]bool {
	return allowAll(permissions)
}

func allowedRoles(roles ...Role) map[Role]bool {
	result := make(map[Role]bool, len(roles))
	for _, role := range roles {
		result[role] = true
	}
	return result
}

func testPrincipal(t *testing.T, suffix string, kind PrincipalKind) Principal {
	t.Helper()
	ref, _ := NewPrincipalRef("principal:" + suffix)
	actor, _ := goal.NewActorRef("actor:" + suffix)
	principal, err := NewPrincipal(ref, actor, kind, "test")
	if err != nil {
		t.Fatalf("NewPrincipal() error = %v", err)
	}
	return principal
}

func testProject(t *testing.T) goal.ProjectRef {
	t.Helper()
	ref, err := goal.NewProjectRef("project:test")
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func testTime() time.Time {
	return time.Date(2026, 7, 15, 8, 0, 0, 0, time.FixedZone("test", 2*60*60))
}
