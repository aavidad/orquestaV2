package orquestacoreconcurrency

import (
	"path"
	"sort"
	"strings"
)

type ScopeRefV0 struct {
	Ref string `json:"ref"`
}

func NewScopeRefV0(raw string) (ScopeRefV0, []WorksetClaimIssueV0) {
	ref, issues := normalizeScopeRefV0("scope", raw)
	if len(issues) > 0 {
		return ScopeRefV0{}, issues
	}
	return ScopeRefV0{Ref: ref}, nil
}

func NormalizeScopeRefsV0(raw []string) ([]ScopeRefV0, []WorksetClaimIssueV0) {
	return normalizeScopeRefsV0("scopes", raw)
}

func ScopeRefsOverlapV0(left ScopeRefV0, right ScopeRefV0) bool {
	return scopeRefContainsV0(left.Ref, right.Ref) || scopeRefContainsV0(right.Ref, left.Ref)
}

func normalizeScopeRefsV0(field string, raw []string) ([]ScopeRefV0, []WorksetClaimIssueV0) {
	issues := make([]WorksetClaimIssueV0, 0)
	refs := make([]string, 0, len(raw))
	for index, value := range raw {
		normalized, valueIssues := normalizeScopeRefV0(field+"[]", value)
		for _, issue := range valueIssues {
			issue.Field = field
			issue.Index = index
			issues = append(issues, issue)
		}
		if len(valueIssues) == 0 {
			refs = append(refs, normalized)
		}
	}

	sort.Strings(refs)
	refs = compactScopeRefStringsV0(refs)

	normalized := make([]ScopeRefV0, 0, len(refs))
	for _, ref := range refs {
		normalized = append(normalized, ScopeRefV0{Ref: ref})
	}
	return normalized, issues
}

func normalizeScopeRefV0(field string, raw string) (string, []WorksetClaimIssueV0) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", []WorksetClaimIssueV0{newWorksetClaimIssueV0(ErrScopeRefVacioV0, field, "")}
	}

	value = strings.ReplaceAll(value, "\\", "/")
	if isHomeScopeRefV0(value) {
		return "", []WorksetClaimIssueV0{newWorksetClaimIssueV0(ErrScopeRefHomeV0, field, raw)}
	}
	if isAbsoluteScopeRefV0(value) {
		return "", []WorksetClaimIssueV0{newWorksetClaimIssueV0(ErrScopeRefAbsolutoV0, field, raw)}
	}
	if hasTraversalScopeRefV0(value) {
		return "", []WorksetClaimIssueV0{newWorksetClaimIssueV0(ErrScopeRefTraversalV0, field, raw)}
	}

	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	value = path.Clean(value)
	if value == "." || value == "" {
		return "", []WorksetClaimIssueV0{newWorksetClaimIssueV0(ErrScopeRefVacioV0, field, raw)}
	}
	if hasForbiddenScopeSegmentV0(value) {
		return "", []WorksetClaimIssueV0{newWorksetClaimIssueV0(ErrScopeRefReservadoV0, field, raw)}
	}
	return value, nil
}

func compactScopeRefStringsV0(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}

	compact := make([]string, 0, len(refs))
	for _, ref := range refs {
		if len(compact) > 0 {
			last := compact[len(compact)-1]
			if last == ref || scopeRefContainsV0(last, ref) {
				continue
			}
		}
		compact = append(compact, ref)
	}
	return compact
}

func scopeRefContainsV0(parent string, child string) bool {
	return parent == child || strings.HasPrefix(child, parent+"/")
}

func isAbsoluteScopeRefV0(value string) bool {
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return true
	}
	return len(value) >= 2 && isASCIILetterV0(value[0]) && value[1] == ':'
}

func isHomeScopeRefV0(value string) bool {
	upper := strings.ToUpper(value)
	return value == "~" ||
		strings.HasPrefix(value, "~/") ||
		strings.HasPrefix(upper, "$HOME/") ||
		upper == "$HOME" ||
		strings.HasPrefix(upper, "${HOME}/") ||
		upper == "${HOME}" ||
		strings.HasPrefix(upper, "%USERPROFILE%/") ||
		upper == "%USERPROFILE%" ||
		strings.HasPrefix(upper, "%HOMEPATH%/") ||
		upper == "%HOMEPATH%"
}

func hasTraversalScopeRefV0(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func hasForbiddenScopeSegmentV0(value string) bool {
	segments := strings.Split(value, "/")
	if len(segments) == 0 {
		return false
	}
	if isReservedScopeRootV0(segments[0]) {
		return true
	}
	for _, segment := range segments {
		if isSecretScopeSegmentV0(segment) {
			return true
		}
	}
	return false
}

func isReservedScopeRootV0(segment string) bool {
	switch strings.ToLower(segment) {
	case "db", "database", "databases",
		"provider", "providers",
		"model", "models", "modelo", "modelos",
		"runtime", "oauth":
		return true
	default:
		return false
	}
}

func isSecretScopeSegmentV0(segment string) bool {
	lower := strings.ToLower(segment)
	switch lower {
	case "secret", "secrets", "secreto", "secretos",
		"credential", "credentials":
		return true
	default:
		return strings.Contains(lower, "token") || strings.Contains(lower, "password")
	}
}

func isASCIILetterV0(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z')
}
