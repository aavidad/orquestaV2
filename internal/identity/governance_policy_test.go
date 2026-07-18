package identity

import "testing"

func TestGovernancePermissionsUseExactDefaultDenyRolePolicy(t *testing.T) {
	roles := []Role{
		RolePlatformAdmin, RoleProjectOwner, RoleProjectAdmin, RoleContributor,
		RoleReviewer, RoleOperator, RoleViewer,
	}
	wantBudgetManager := map[Role]bool{
		RolePlatformAdmin: true,
		RoleProjectOwner:  true,
		RoleProjectAdmin:  true,
	}
	wantEffectApprover := map[Role]bool{
		RolePlatformAdmin: true,
		RoleProjectOwner:  true,
		RoleProjectAdmin:  true,
		RoleReviewer:      true,
		RoleOperator:      true,
	}
	for _, permission := range []Permission{PermissionBudgetsManage, PermissionEffectsApprove} {
		if err := ValidatePermission(permission); err != nil {
			t.Fatalf("ValidatePermission(%q): %v", permission, err)
		}
	}
	for _, role := range roles {
		if got := RoleAllows(role, PermissionBudgetsManage); got != wantBudgetManager[role] {
			t.Errorf("RoleAllows(%s, budgets.manage)=%v want=%v", role, got, wantBudgetManager[role])
		}
		if got := RoleAllows(role, PermissionEffectsApprove); got != wantEffectApprover[role] {
			t.Errorf("RoleAllows(%s, effects.approve)=%v want=%v", role, got, wantEffectApprover[role])
		}
	}
	if RoleAllows(Role("approver"), PermissionEffectsApprove) ||
		RoleAllows(RoleProjectOwner, Permission("effects.approve.alias")) ||
		RoleAllows(RoleProjectOwner, Permission("budgets.manage.alias")) {
		t.Fatal("unknown governance role or permission escaped default deny")
	}
}
