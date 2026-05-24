package orquestaruntimeworktree

import (
	"sort"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func normalizeWorktreeRelPathV0(value string, allowRoot bool) (string, bool) {
	return orquestarails.NormalizeWorkspaceRelativePathV0(value, allowRoot)
}

func normalizeWorktreePathListV0(values []string, allowRoot bool) ([]string, []WorktreeIssueV0) {
	paths := make([]string, 0, len(values))
	var issues []WorktreeIssueV0
	seen := map[string]bool{}
	for _, value := range values {
		path, ok := normalizeWorktreeRelPathV0(value, allowRoot)
		if !ok {
			issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "path"))
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, issues
}

func worktreePathIgnoredV0(path string, prefixes []string) bool {
	path = strings.TrimSpace(path)
	for _, prefix := range prefixes {
		if prefix == "." || path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func worktreePathAllowedV0(path string, writeSet []string) bool {
	for _, allowed := range writeSet {
		if allowed == "." || path == allowed || strings.HasPrefix(path, allowed+"/") {
			return true
		}
	}
	return false
}

func compactWorktreeStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
