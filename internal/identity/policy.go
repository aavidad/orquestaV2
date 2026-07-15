package identity

import "errors"

type Role string

const (
	RolePlatformAdmin Role = "platform_admin"
	RoleProjectOwner  Role = "project_owner"
	RoleProjectAdmin  Role = "project_admin"
	RoleContributor   Role = "contributor"
	RoleReviewer      Role = "reviewer"
	RoleOperator      Role = "operator"
	RoleViewer        Role = "viewer"
)

type Permission string

const (
	PermissionProjectHierarchyManage  Permission = "project.hierarchy.manage"
	PermissionProjectMembershipManage Permission = "project.membership.manage"
	PermissionGoalsCreate             Permission = "goals.create"
	PermissionGoalsAmend              Permission = "goals.amend"
	PermissionGoalsGet                Permission = "goals.get"
	PermissionGoalsList               Permission = "goals.list"
	PermissionGoalsDirect             Permission = "goals.direct"
	PermissionArtifactsRead           Permission = "artifacts.read"
	PermissionProjectStatus           Permission = "project.status"
)

func ValidateRole(role Role) error {
	switch role {
	case RolePlatformAdmin, RoleProjectOwner, RoleProjectAdmin, RoleContributor,
		RoleReviewer, RoleOperator, RoleViewer:
		return nil
	default:
		return errors.New("identity.invalid_role")
	}
}

func ValidatePermission(permission Permission) error {
	switch permission {
	case PermissionProjectHierarchyManage, PermissionProjectMembershipManage,
		PermissionGoalsCreate, PermissionGoalsAmend, PermissionGoalsGet,
		PermissionGoalsList, PermissionGoalsDirect, PermissionArtifactsRead, PermissionProjectStatus:
		return nil
	default:
		return errors.New("identity.invalid_permission")
	}
}

// RoleAllows is the complete V10 default-deny policy. Unknown roles and
// permissions always return false.
func RoleAllows(role Role, permission Permission) bool {
	if ValidateRole(role) != nil || ValidatePermission(permission) != nil {
		return false
	}
	switch role {
	case RolePlatformAdmin, RoleProjectOwner:
		return true
	case RoleProjectAdmin:
		return permission != PermissionProjectHierarchyManage
	case RoleContributor:
		return permission == PermissionGoalsCreate || permission == PermissionGoalsAmend ||
			permission == PermissionGoalsGet || permission == PermissionGoalsList ||
			permission == PermissionArtifactsRead || permission == PermissionProjectStatus
	case RoleReviewer, RoleViewer:
		return permission == PermissionGoalsGet || permission == PermissionGoalsList ||
			permission == PermissionArtifactsRead || permission == PermissionProjectStatus
	case RoleOperator:
		return permission == PermissionGoalsCreate || permission == PermissionGoalsGet ||
			permission == PermissionGoalsList || permission == PermissionGoalsDirect ||
			permission == PermissionArtifactsRead ||
			permission == PermissionProjectStatus
	default:
		return false
	}
}

// CanDelegateMembershipRole is the complete membership delegation policy.
// Unknown roles and the platform-wide role are never valid project grants.
func CanDelegateMembershipRole(grantor Role, target Role) bool {
	if target == RolePlatformAdmin || ValidateRole(target) != nil {
		return false
	}
	switch grantor {
	case RolePlatformAdmin, RoleProjectOwner:
		return true
	case RoleProjectAdmin:
		return target == RoleContributor || target == RoleReviewer ||
			target == RoleOperator || target == RoleViewer
	default:
		return false
	}
}

// IsProjectAuthority identifies roles that preserve project ownership.
func IsProjectAuthority(role Role) bool {
	return role == RolePlatformAdmin || role == RoleProjectOwner
}
