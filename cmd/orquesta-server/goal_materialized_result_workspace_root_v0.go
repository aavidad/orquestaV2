package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

type serverGoalMaterializedResultWorkspaceRootV0 struct {
	CanonicalRoot   string
	WorkspaceLookup serverCodexGoalWorkspaceBindingLookupV0
}

func (resolver serverGoalMaterializedResultWorkspaceRootV0) ResolveGoalMaterializedResultProjectRootV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (string, error) {
	canonical := filepath.Clean(strings.TrimSpace(resolver.CanonicalRoot))
	if resolver.WorkspaceLookup == nil {
		return canonical, nil
	}
	goalRef := strings.TrimSpace(state.GoalRef)
	found, err := resolver.WorkspaceLookup.HasCodexGoalWorkspaceBindingV0(ctx, goalRef)
	if err != nil {
		return "", err
	}
	if !found {
		return canonical, nil
	}
	binding, err := resolver.WorkspaceLookup.ResolveCodexGoalWorkspaceV0(ctx, orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         goalRef,
		ExternalGoalRef: strings.TrimSpace(state.ExternalGoalRef),
	})
	if err != nil || strings.TrimSpace(binding.ProjectWorkDir) == "" {
		return "", fmt.Errorf("goal_materialized_result_workspace_unavailable")
	}
	return filepath.Clean(binding.ProjectWorkDir), nil
}

var _ orquestaappcodexstack.GoalMaterializedResultProjectRootResolverPortV0 = serverGoalMaterializedResultWorkspaceRootV0{}
