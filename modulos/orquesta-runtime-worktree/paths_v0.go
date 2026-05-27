package orquestaruntimeworktree

import (
	pathpkg "path"
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
		if worktreePathMatchesIgnorePrefixV0(path, strings.TrimSpace(prefix)) {
			return true
		}
	}
	return false
}

func worktreePathMatchesIgnorePrefixV0(path string, prefix string) bool {
	if prefix == "." || path == prefix || strings.HasPrefix(path, prefix+"/") {
		return true
	}
	if !worktreeDefaultControlIgnorePrefixV0(prefix) {
		return false
	}
	segment, _, _ := strings.Cut(path, "/")
	return strings.HasPrefix(segment, prefix+"-")
}

func worktreeDefaultControlIgnorePrefixV0(prefix string) bool {
	for _, controlPrefix := range worktreeDefaultControlPrefixesV0 {
		if prefix == controlPrefix {
			return true
		}
	}
	return false
}

func worktreePathAllowedV0(path string, writeSet []string) bool {
	for _, allowed := range writeSet {
		if worktreePathMatchesWriteSetEntryV0(path, allowed) {
			return true
		}
	}
	return false
}

func worktreePathMatchesWriteSetEntryV0(path string, entry string) bool {
	if entry == "." || path == entry || strings.HasPrefix(path, entry+"/") {
		return true
	}
	if strings.HasSuffix(entry, "/**") {
		prefix := strings.TrimSuffix(entry, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if strings.Contains(entry, "**") && worktreePathGlobstarMatchV0(entry, path) {
		return true
	}
	if !strings.ContainsAny(entry, "*?[") {
		return false
	}
	matched, err := pathpkg.Match(entry, path)
	return err == nil && matched
}

func worktreePathGlobstarMatchV0(pattern string, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	memo := map[[2]int]bool{}
	var match func(int, int) bool
	match = func(patternIndex int, pathIndex int) bool {
		key := [2]int{patternIndex, pathIndex}
		if value, ok := memo[key]; ok {
			return value
		}
		ok := false
		defer func() { memo[key] = ok }()
		if patternIndex == len(patternParts) {
			ok = pathIndex == len(pathParts)
			return ok
		}
		if patternParts[patternIndex] == "**" {
			ok = match(patternIndex+1, pathIndex)
			for next := pathIndex; !ok && next < len(pathParts); next++ {
				ok = match(patternIndex+1, next+1)
			}
			return ok
		}
		if pathIndex >= len(pathParts) {
			return false
		}
		matched, err := pathpkg.Match(patternParts[patternIndex], pathParts[pathIndex])
		ok = err == nil && matched && match(patternIndex+1, pathIndex+1)
		return ok
	}
	return match(0, 0)
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
