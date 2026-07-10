package orquestaruntimeclaude

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func (backend ClaudeGoalBackendV0) resultCandidatePathsV0(spec orquestagoal.GoalWorkSpecV0) []string {
	goalFile := ClaudeGoalResultFilePrefixV0 + claudeGoalSafeRefV0(spec.GoalRef) + ".json"
	seen := map[string]bool{}
	var out []string
	addRuntime := func(name string) {
		path := filepath.Join(filepath.Clean(backend.RuntimeWorkDir), name)
		if seen[path] {
			return
		}
		seen[path] = true
		out = append(out, path)
	}
	add := func(rel string) {
		path, ok := backend.projectPathV0(rel)
		if !ok || seen[path] {
			return
		}
		seen[path] = true
		out = append(out, path)
	}
	// New launches write here. Project candidates remain below for governed
	// migration of historical results.
	addRuntime(goalFile)
	for _, scope := range spec.WriteSet {
		rel := strings.TrimSpace(filepath.ToSlash(scope.Path))
		if rel == "" {
			continue
		}
		if strings.HasSuffix(rel, ".json") {
			add(rel)
			continue
		}
		add(filepath.ToSlash(filepath.Join(rel, ClaudeGoalResultFileNameV0)))
		add(filepath.ToSlash(filepath.Join(rel, goalFile)))
		add(filepath.ToSlash(filepath.Join(rel, "docs", ClaudeGoalResultFileNameV0)))
		add(filepath.ToSlash(filepath.Join(rel, "docs", goalFile)))
	}
	return out
}

func claudeGoalRuntimeResultPathV0(runtimeDir string, goalRef string) string {
	return filepath.Join(filepath.Clean(strings.TrimSpace(runtimeDir)), ClaudeGoalResultFilePrefixV0+claudeGoalSafeRefV0(goalRef)+".json")
}

func (backend ClaudeGoalBackendV0) projectPathV0(rel string) (string, bool) {
	rel = filepath.Clean(strings.TrimSpace(rel))
	if rel == "" || rel == "." {
		rel = "."
	}
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false
	}
	path := filepath.Join(filepath.Clean(backend.ProjectWorkDir), rel)
	root := filepath.Clean(backend.ProjectWorkDir)
	back, err := filepath.Rel(root, path)
	if err != nil || back == ".." || filepath.IsAbs(back) || strings.HasPrefix(back, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return path, true
}

func claudeGoalSpecFileNameV0(goalRef string) string {
	return ClaudeGoalWorkSpecFilePrefixV0 + claudeGoalSafeRefV0(goalRef) + ".json"
}

func claudeGoalPromptFileNameV0(goalRef string) string {
	return ClaudeGoalPromptFilePrefixV0 + claudeGoalSafeRefV0(goalRef) + ".txt"
}

func claudeGoalExternalRefV0(goalRef string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(goalRef)))
	return "claude-goal-" + hex.EncodeToString(hash[:])[:16]
}

func claudeGoalSafeRefV0(goalRef string) string {
	goalRef = strings.TrimSpace(goalRef)
	if goalRef == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range goalRef {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	safe := strings.Trim(b.String(), "-")
	if safe == "" {
		return "unknown"
	}
	return safe
}

func firstNonEmptyClaudeGoalV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func compactClaudeGoalStringsV0(values []string) []string {
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
	return out
}
