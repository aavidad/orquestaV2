package orquestaautoprogramming

import "strings"

func autoprogrammingRequestLiveWorkIssuesV0(
	liveWorks []AutoprogrammingLiveWorkV0,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	for _, live := range liveWorks {
		if !autoprogrammingLiveWorkIsActiveV0(live) {
			continue
		}
		if autoprogrammingLiveWorkDependencyRefV0(live) == "" {
			issues = append(issues, autoprogrammingRequestIssueV0(
				"live_work_ref_missing",
				"live_works",
				"trabajo vivo sin ref causal",
			))
		}
		for _, path := range compactStringsV0(live.WriteSet) {
			if autoprogrammingRequestWriteSetPathAllowedV0(path) {
				continue
			}
			issues = append(issues, autoprogrammingRequestIssueV0(
				"live_work_write_set_path_invalid",
				"live_works.write_set",
				"ruta viva no permitida: "+path,
			))
		}
	}
	return issues
}

func autoprogrammingLiveWorkIsActiveV0(live AutoprogrammingLiveWorkV0) bool {
	switch strings.ToLower(strings.TrimSpace(live.Status)) {
	case "closed", "completed", "done", "failed", "cancelled", "canceled":
		return false
	default:
		return len(compactStringsV0(live.WriteSet)) > 0
	}
}

func autoprogrammingLiveWorkDependencyRefV0(live AutoprogrammingLiveWorkV0) string {
	for _, value := range []string{live.TaskRef, live.WorkRef, live.AgentRef} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func autoprogrammingWriteSetsOverlapV0(left []string, right []string) bool {
	for _, a := range compactStringsV0(left) {
		for _, b := range compactStringsV0(right) {
			if autoprogrammingWriteSetPathsOverlapV0(a, b) {
				return true
			}
		}
	}
	return false
}

func autoprogrammingWriteSetPathsOverlapV0(left string, right string) bool {
	a := autoprogrammingNormalizeWriteSetPathForOverlapV0(left)
	b := autoprogrammingNormalizeWriteSetPathForOverlapV0(right)
	return a != "" && b != "" && (a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/"))
}

func autoprogrammingNormalizeWriteSetPathForOverlapV0(path string) string {
	return strings.Trim(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"), "/")
}

func appendUniqueStringV0(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
