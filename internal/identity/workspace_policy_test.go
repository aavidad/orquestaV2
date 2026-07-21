package identity

import "testing"

func TestChangesIntegratePermissionUsesExactDefaultDenyPolicy(t *testing.T) {
	if err := ValidatePermission(PermissionChangesIntegrate); err != nil {
		t.Fatalf("ValidatePermission(changes.integrate): %v", err)
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
		if got := RoleAllows(role, PermissionChangesIntegrate); got != allowed[role] {
			t.Errorf("RoleAllows(%s, changes.integrate)=%v want=%v", role, got, allowed[role])
		}
	}
	if RoleAllows(Role("operator_alias"), PermissionChangesIntegrate) ||
		RoleAllows(RoleOperator, Permission("changes.integrate.alias")) {
		t.Fatal("unknown role or permission was allowed")
	}
}
