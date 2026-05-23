package orquestaappdirectorintake

import "strings"

func normalizeHumanDirectorWriteSetHintsV0(
	values []string,
) ([]string, []HumanDirectorPlanIssueV0) {
	var issues []HumanDirectorPlanIssueV0
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		normalized, ok := normalizeHumanDirectorWriteSetHintV0(value)
		if !ok {
			issues = append(issues, humanDirectorPlanIssueV0(
				"unsafe_write_set_hint",
				"hints.write_set",
				strings.TrimSpace(value),
			))
			continue
		}
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	return out, issues
}

func normalizeHumanDirectorWriteSetHintV0(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if value == "." {
		return ".", true
	}
	if strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "~") ||
		strings.Contains(value, "$") ||
		strings.Contains(value, "\\") {
		return "", false
	}
	parts := compactDirectorIntakeStringsV0(strings.Split(value, "/"))
	for _, part := range parts {
		if part == "." || part == ".." {
			return "", false
		}
	}
	value = strings.TrimSuffix(value, "/**")
	value = strings.TrimSuffix(value, "/*")
	value = strings.Trim(value, "/")
	if value == "" {
		return "", true
	}
	return value, true
}

func splitHumanDirectorWriteSetOverlapsV0(
	writeSet []string,
) (overlapped []string, clean []string) {
	overlap := map[string]bool{}
	for i, left := range writeSet {
		for j, right := range writeSet {
			if i == j {
				continue
			}
			if humanDirectorScopeContainsV0(left, right) || humanDirectorScopeContainsV0(right, left) {
				overlap[left] = true
				overlap[right] = true
			}
		}
	}
	for _, path := range writeSet {
		if overlap[path] {
			overlapped = append(overlapped, path)
			continue
		}
		clean = append(clean, path)
	}
	return overlapped, clean
}

func humanDirectorScopeContainsV0(parent string, child string) bool {
	parent = strings.Trim(parent, "/")
	child = strings.Trim(child, "/")
	if parent == "" || child == "" || parent == child {
		return false
	}
	return parent == "." || strings.HasPrefix(child, parent+"/")
}

func needsHumanDirectorStudyBeforeV0(
	request HumanDirectorWorkIntakeRequestV0,
	writeSet []string,
) bool {
	if request.WorktreeRef == "" || request.BranchRef == "" {
		return true
	}
	if len(writeSet) == 0 {
		return true
	}
	if directorIntakeStringInSetV0(writeSet, ".") {
		return true
	}
	return len(request.Hints.Areas) > 1 && len(writeSet) < len(request.Hints.Areas)
}

func normalizeHumanDirectorAreasV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		area := normalizeHumanDirectorAreaV0(value)
		if area == "" || seen[area] {
			continue
		}
		seen[area] = true
		out = append(out, area)
	}
	return out
}

func normalizeHumanDirectorAreaV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "web_application", "frontend", "front":
		return "web_app"
	case "backend", "server", "api_rest":
		return "api"
	case "database", "storage", "db":
		return "persistence"
	default:
		return safeDirectorIntakeRefPartV0(value)
	}
}
