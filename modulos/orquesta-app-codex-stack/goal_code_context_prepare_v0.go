package orquestaappcodexstack

import (
	"context"
	"path/filepath"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

type goalLauncherWithCodeContextPrepareV0 struct {
	Inner       orquestagoal.GoalWorkLauncherPortV0
	CodeContext orquestacontext.CodeContextQueryPortV0
}

func goalLauncherWithCodeContextPrepareFromConfigV0(
	launcher orquestagoal.GoalWorkLauncherPortV0,
	codeContext orquestacontext.CodeContextQueryPortV0,
) orquestagoal.GoalWorkLauncherPortV0 {
	if launcher == nil || codeContext == nil {
		return launcher
	}
	return goalLauncherWithCodeContextPrepareV0{
		Inner:       launcher,
		CodeContext: codeContext,
	}
}

func (launcher goalLauncherWithCodeContextPrepareV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	spec, cacheStatus := launcher.prepareCodeContextV0(ctx, spec)
	receipt, err := launcher.Inner.LaunchGoalWorkV0(ctx, spec)
	if cacheStatus != "" && receipt.ContextBudget.CodeContextCacheStatus == "" {
		receipt.ContextBudget.CodeContextCacheStatus = cacheStatus
	}
	return receipt, err
}

func (launcher goalLauncherWithCodeContextPrepareV0) prepareCodeContextV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkSpecV0, string) {
	if launcher.CodeContext == nil ||
		!goalSpecHasCodeWriteSetV0(spec) ||
		goalSpecHasCodeContextPreparedRefV0(spec) {
		return spec, ""
	}
	query := orquestacontext.CodeContextQueryV0{
		SchemaVersion: orquestacontext.CodeContextQuerySchemaVersionV0,
		RequestRef:    "request-ref-code-context-prepare-" + codeContextGoalRefPartV0(spec.GoalRef),
		RepositoryRef: firstCodeContextGoalValueV0(spec.ProjectRef, spec.RunRef, "repo-ref-goal-code-context"),
		WorktreeRef:   spec.RunRef,
		QueryKind:     orquestacontext.CodeContextQueryKindRepoMapV0,
		Query:         firstCodeContextGoalValueV0(spec.Objective, spec.WorkKind, "code"),
		Scope:         codeContextGoalScopesV0(spec.WriteSet),
		MaxResults:    8,
		MaxBytes:      6000,
		RequestedBy:   "orquesta-app-codex-stack-goal-code-context-prepare",
	}
	result, err := launcher.CodeContext.QueryCodeContextV0(ctx, query)
	if err != nil || result.Estado != orquestacontext.CodeContextEstadoOKV0 {
		spec.ContextRefs = append(spec.ContextRefs, orquestagoal.GoalContextRefV0{
			Kind:    "code_context",
			Ref:     "code_context_prepare_failed",
			Purpose: "analizador de codigo no precargado; consultar orquesta.codebase.query.v0 en vivo antes de leer ficheros completos",
		})
		return spec, ""
	}
	ref := "code_context_prepared:repo_map"
	if result.QueryHash != "" {
		ref += ":" + result.QueryHash
	}
	spec.ContextRefs = append(spec.ContextRefs, orquestagoal.GoalContextRefV0{
		Kind:    "code_context",
		Ref:     ref,
		Purpose: "repo_map precargado por Orquesta; cache_status=" + firstCodeContextGoalValueV0(result.CacheStatus, "miss"),
	})
	spec.EvidenceRefs = uniqueCodeContextGoalStringsV0(append(spec.EvidenceRefs, result.EvidenceRefs...))
	return spec, result.CacheStatus
}

func goalSpecHasCodeWriteSetV0(spec orquestagoal.GoalWorkSpecV0) bool {
	for _, scope := range spec.WriteSet {
		path := strings.ToLower(strings.TrimSpace(scope.Path))
		purpose := strings.ToLower(strings.TrimSpace(scope.Purpose))
		if strings.HasPrefix(path, "cmd/") ||
			strings.HasPrefix(path, "modulos/") ||
			strings.HasSuffix(path, ".go") ||
			strings.Contains(purpose, "codigo") ||
			strings.Contains(purpose, "code") {
			return true
		}
	}
	return false
}

func goalSpecHasCodeContextPreparedRefV0(spec orquestagoal.GoalWorkSpecV0) bool {
	for _, ref := range spec.ContextRefs {
		if strings.HasPrefix(strings.TrimSpace(ref.Ref), "code_context_prepared:") {
			return true
		}
	}
	return false
}

func codeContextGoalScopesV0(writeSet []orquestagoal.GoalWriteScopeV0) []string {
	out := make([]string, 0, len(writeSet))
	for _, scope := range writeSet {
		path := filepath.ToSlash(filepath.Clean(strings.TrimSpace(scope.Path)))
		if path == "" || path == "." || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "/") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(path), ".go") {
			path = filepath.ToSlash(filepath.Dir(path))
		}
		out = append(out, path)
	}
	if len(out) == 0 {
		return []string{"cmd", "modulos"}
	}
	return uniqueCodeContextGoalStringsV0(out)
}

func codeContextGoalRefPartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "goal"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "goal"
	}
	if len(out) > 40 {
		return strings.Trim(out[:40], "-")
	}
	return out
}

func firstCodeContextGoalValueV0(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func uniqueCodeContextGoalStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
