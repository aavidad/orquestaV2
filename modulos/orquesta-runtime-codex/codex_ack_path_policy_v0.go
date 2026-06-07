package orquestaruntimecodex

import (
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
)

var codexAckForbiddenControlDirBasesV0 = []string{
	".orquesta-runtime",
	".orquesta-codex-runtime",
	".orquesta-local-runtime",
	".orquesta-control",
	".orquesta-server",
	".orquesta-smoke-work",
	".orquesta-runs",
	".orquesta-worktrees",
	".orquesta-logs",
	"orquesta-runtime",
	"orquesta-codex-runtime",
	"orquesta-local-runtime",
	"logs",
	"tmp",
	".cache",
	"backups",
	"certs",
}

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
	if codexAckControlDirPathForbiddenV0(path) {
		return true
	}
	segment, _, _ := strings.Cut(path, "/")
	lowerSegment := strings.ToLower(segment)
	if lowerSegment == "orquesta.env" ||
		lowerSegment == "orquesta.db" ||
		strings.HasPrefix(lowerSegment, "orquesta.db-") ||
		lowerSegment == ".orquesta-inbox.md" {
		return true
	}
	switch pathpkg.Base(path) {
	case CodexAgentAckFileNameV0,
		CodexAgentPacketFileNameV0,
		CodexDirectorDecisionsFileNameV0,
		"agent_prompt.txt",
		"agent_shutdown_checkpoint_ack.json",
		"codex_stdout.log",
		"codex_stderr.log",
		"codex_last_message.txt",
		"orquesta_shutdown_request.json",
		"server.log",
		".ssl-key.log":
		return true
	default:
		return strings.HasSuffix(strings.ToLower(pathpkg.Base(path)), ".log")
	}
}

func codexAckControlDirPathForbiddenV0(path string) bool {
	segment, _, _ := strings.Cut(path, "/")
	for _, base := range codexAckForbiddenControlDirBasesV0 {
		if segment == base || strings.HasPrefix(segment, base+"-") {
			return true
		}
	}
	return false
}

func codexAckForbiddenArtifactEvidenceV0(values []string) string {
	categories := map[string]bool{}
	for _, value := range values {
		path, ok := normalizeCodexAckFilePathV0(value)
		if !ok {
			categories["invalid_path"] = true
			continue
		}
		if category := codexAckForbiddenArtifactCategoryV0(path); category != "" {
			categories[category] = true
		}
	}
	if len(categories) == 0 {
		return "local_artifact_excluded"
	}
	out := make([]string, 0, len(categories))
	for category := range categories {
		out = append(out, category)
	}
	sort.Strings(out)
	return "local_artifact_excluded:" + strings.Join(out, ",")
}

func codexAckForbiddenArtifactCategoryV0(path string) string {
	segment, _, _ := strings.Cut(path, "/")
	base := strings.ToLower(pathpkg.Base(path))
	lowerSegment := strings.ToLower(segment)
	switch {
	case lowerSegment == ".orquesta-server":
		return "server_state_dir"
	case lowerSegment == ".orquesta-smoke-work":
		return "smoke_work_dir"
	case lowerSegment == "orquesta.env":
		return "local_operator_config"
	case lowerSegment == "orquesta.db" || strings.HasPrefix(lowerSegment, "orquesta.db-"):
		return "local_database_state"
	case lowerSegment == ".orquesta-inbox.md":
		return "local_operator_notes"
	case lowerSegment == "certs" || base == ".ssl-key.log":
		return "local_secret_diagnostics"
	case lowerSegment == "logs" || lowerSegment == ".orquesta-logs" || strings.HasSuffix(base, ".log"):
		return "local_logs"
	case codexAckControlDirPathForbiddenV0(path):
		return "runtime_control_dir"
	case codexAckArtifactPathForbiddenV0(path):
		return "control_file"
	default:
		return ""
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
	if !ok {
		return "", false
	}
	if strings.ContainsAny(path, "*?[") {
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
