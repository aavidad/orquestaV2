package orquestagoal

import (
	"path/filepath"
	"strings"
)

func NormalizeGoalDestructiveChangeAuthorizationV0(
	authorization GoalDestructiveChangeAuthorizationV0,
) GoalDestructiveChangeAuthorizationV0 {
	authorization.Kind = GoalDestructiveChangeAuthorizationKindV0(strings.ToLower(strings.TrimSpace(string(authorization.Kind))))
	authorization.Path = filepath.ToSlash(strings.TrimSpace(authorization.Path))
	authorization.PreviousPath = filepath.ToSlash(strings.TrimSpace(authorization.PreviousPath))
	authorization.CurrentPath = filepath.ToSlash(strings.TrimSpace(authorization.CurrentPath))
	return authorization
}

func ValidGoalDestructiveChangeAuthorizationV0(authorization GoalDestructiveChangeAuthorizationV0) bool {
	authorization = NormalizeGoalDestructiveChangeAuthorizationV0(authorization)
	switch authorization.Kind {
	case GoalDestructiveChangeAuthorizationRenameV0:
		return authorization.Path == "" &&
			validGoalWriteScopePathV0(authorization.PreviousPath) &&
			validGoalWriteScopePathV0(authorization.CurrentPath)
	case GoalDestructiveChangeAuthorizationRemoveV0,
		GoalDestructiveChangeAuthorizationTruncateV0,
		GoalDestructiveChangeAuthorizationReplaceV0:
		return validGoalWriteScopePathV0(authorization.Path) &&
			authorization.PreviousPath == "" && authorization.CurrentPath == ""
	default:
		return false
	}
}

func goalDestructiveAuthorizationOutsideWriteSetV0(
	authorization GoalDestructiveChangeAuthorizationV0,
	writeSet []GoalWriteScopeV0,
) bool {
	paths := []string{authorization.Path}
	if authorization.Kind == GoalDestructiveChangeAuthorizationRenameV0 {
		paths = []string{authorization.PreviousPath, authorization.CurrentPath}
	}
	return len(goalArtifactPathsOutsideWriteSetV0(writeSet, paths)) > 0
}
