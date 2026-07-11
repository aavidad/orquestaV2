package orquestaappcodexstack

import (
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func autoprogrammingWorktreeDestructiveAuthorizationsV0(
	authorizations []orquestagoal.GoalDestructiveChangeAuthorizationV0,
) []orquestaruntimeworktree.WorktreeDestructiveAuthorizationV0 {
	result := make([]orquestaruntimeworktree.WorktreeDestructiveAuthorizationV0, 0, len(authorizations))
	for _, authorization := range authorizations {
		mapped := orquestaruntimeworktree.WorktreeDestructiveAuthorizationV0{
			Path: authorization.Path, PreviousPath: authorization.PreviousPath, CurrentPath: authorization.CurrentPath,
		}
		switch authorization.Kind {
		case orquestagoal.GoalDestructiveChangeAuthorizationRenameV0:
			mapped.Kind = orquestaruntimeworktree.WorktreeDestructiveAuthorizationRenameV0
		case orquestagoal.GoalDestructiveChangeAuthorizationRemoveV0:
			mapped.Kind = orquestaruntimeworktree.WorktreeDestructiveAuthorizationRemoveV0
		case orquestagoal.GoalDestructiveChangeAuthorizationTruncateV0:
			mapped.Kind = orquestaruntimeworktree.WorktreeDestructiveAuthorizationTruncateV0
		case orquestagoal.GoalDestructiveChangeAuthorizationReplaceV0:
			mapped.Kind = orquestaruntimeworktree.WorktreeDestructiveAuthorizationReplaceV0
		default:
			continue
		}
		result = append(result, mapped)
	}
	return result
}
