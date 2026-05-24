package orquestaruntimecodex

import (
	pathpkg "path"
	"path/filepath"
	"strings"
)

func codexAckHasInvalidPathV0(values []string) bool {
	for _, value := range values {
		if _, ok := normalizeCodexAckFilePathV0(value); !ok {
			return true
		}
	}
	return false
}

func codexAckHasForbiddenArtifactPathV0(values []string) bool {
	for _, value := range values {
		if codexAckArtifactPathForbiddenV0(value) {
			return true
		}
	}
	return false
}

func codexAckArtifactPathForbiddenV0(value string) bool {
	path, ok := normalizeCodexAckFilePathV0(value)
	if !ok {
		return true
	}
	if strings.HasPrefix(path, ".orquesta-runtime/") ||
		strings.HasPrefix(path, "orquesta-runtime/") {
		return true
	}
	switch pathpkg.Base(path) {
	case CodexAgentAckFileNameV0,
		CodexAgentPacketFileNameV0,
		CodexDirectorDecisionsFileNameV0,
		"agent_prompt.txt",
		"codex_stdout.log",
		"codex_stderr.log",
		"codex_last_message.txt":
		return true
	default:
		return false
	}
}

func normalizeCodexAckPathsV0(values []string) []string {
	paths := make([]string, 0, len(values))
	for _, value := range values {
		path, ok := normalizeCodexAckFilePathV0(value)
		if ok {
			paths = append(paths, path)
		}
	}
	return paths
}

func normalizeCodexAckWriteSetPathsV0(values []string) []string {
	paths := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "." {
			paths = append(paths, ".")
			continue
		}
		path, ok := normalizeCodexAckPathV0(value)
		if ok {
			paths = append(paths, path)
		}
	}
	return paths
}

func normalizeCodexAckFilePathV0(value string) (string, bool) {
	path, ok := normalizeCodexAckPathV0(value)
	if !ok || strings.ContainsAny(path, "*?[") {
		return "", false
	}
	return path, true
}

func normalizeCodexAckPathV0(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" ||
		strings.Contains(trimmed, "://") ||
		strings.HasPrefix(trimmed, "~") ||
		strings.Contains(trimmed, "$HOME") ||
		filepath.IsAbs(trimmed) {
		return "", false
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func codexAckMissingWriteSetV0(writeSet []string, files []string) []string {
	missing := make([]string, 0)
	for _, required := range writeSet {
		if required == "." {
			continue
		}
		found := false
		for _, file := range files {
			if codexAckPathMatchesWriteSetEntryV0(file, required) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, required)
		}
	}
	return missing
}

func codexAckPathMatchesWriteSetEntryV0(path string, entry string) bool {
	if entry == "." {
		return true
	}
	if path == entry || strings.HasPrefix(path, entry+"/") {
		return true
	}
	if strings.HasSuffix(entry, "/**") {
		prefix := strings.TrimSuffix(entry, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if strings.Contains(entry, "**") && codexAckPathGlobstarMatchV0(entry, path) {
		return true
	}
	if !strings.ContainsAny(entry, "*?[") {
		return false
	}
	matched, err := pathpkg.Match(entry, path)
	return err == nil && matched
}

func codexAckPathGlobstarMatchV0(pattern string, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	memo := map[[2]int]bool{}
	var match func(int, int) bool
	match = func(patternIndex int, pathIndex int) bool {
		key := [2]int{patternIndex, pathIndex}
		if value, ok := memo[key]; ok {
			return value
		}
		var ok bool
		defer func() {
			memo[key] = ok
		}()
		if patternIndex == len(patternParts) {
			ok = pathIndex == len(pathParts)
			return ok
		}
		part := patternParts[patternIndex]
		if part == "**" {
			if match(patternIndex+1, pathIndex) {
				ok = true
				return ok
			}
			for next := pathIndex; next < len(pathParts); next++ {
				if match(patternIndex+1, next+1) {
					ok = true
					return ok
				}
			}
			return false
		}
		if pathIndex >= len(pathParts) {
			return false
		}
		matched, err := pathpkg.Match(part, pathParts[pathIndex])
		if err != nil || !matched {
			return false
		}
		ok = match(patternIndex+1, pathIndex+1)
		return ok
	}
	return match(0, 0)
}
