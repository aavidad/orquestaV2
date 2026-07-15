package identity

import "testing"

func TestGoalsDirectPermissionUsesDefaultDenyRolePolicy(t *testing.T) {
	if err := ValidatePermission(PermissionGoalsDirect); err != nil {
		t.Fatalf("validate goals.direct: %v", err)
	}
	allowed := map[Role]bool{
		RolePlatformAdmin: true,
		RoleProjectOwner:  true,
		RoleProjectAdmin:  true,
		RoleOperator:      true,
	}
	for _, role := range []Role{
		RolePlatformAdmin, RoleProjectOwner, RoleProjectAdmin, RoleContributor,
		RoleReviewer, RoleOperator, RoleViewer,
	} {
		if got := RoleAllows(role, PermissionGoalsDirect); got != allowed[role] {
			t.Errorf("RoleAllows(%s, goals.direct)=%v want=%v", role, got, allowed[role])
		}
	}
	if RoleAllows(Role("director"), PermissionGoalsDirect) ||
		RoleAllows(RoleOperator, Permission("goals.direct.alias")) {
		t.Fatal("unknown role or permission escaped default deny")
	}
}
