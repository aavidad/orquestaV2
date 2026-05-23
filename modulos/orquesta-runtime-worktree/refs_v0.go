package orquestaruntimeworktree

import (
	"regexp"
	"strings"
)

func validateWorktreeOpaqueRefV0(field string, value string, required bool) []WorktreeIssueV0 {
	value = strings.TrimSpace(value)
	if value == "" {
		if required {
			return []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueInvalidRequestV0, field)}
		}
		return nil
	}
	if worktreeRefLooksUnsafeV0(value) || !worktreeOpaqueRefPatternV0.MatchString(value) {
		return []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueInvalidRequestV0, field)}
	}
	return nil
}

func worktreeRefLooksUnsafeV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "/") ||
		strings.Contains(value, "\\") ||
		strings.Contains(value, "://") ||
		strings.HasPrefix(value, "~") ||
		strings.HasPrefix(low, "$home") ||
		strings.ContainsAny(value, "\x00\r\n") ||
		strings.Contains(low, "token") ||
		strings.Contains(low, "secret") ||
		strings.HasPrefix(low, "sk-")
}

var worktreeOpaqueRefPatternV0 = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,159}$`)
