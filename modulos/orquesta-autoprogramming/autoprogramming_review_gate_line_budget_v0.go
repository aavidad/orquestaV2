package orquestaautoprogramming

import "strings"

const (
	AutoprogrammingReviewGateStrictLineBudgetIssueV0 = "go_file_line_budget_strict_blocking"
)

func autoprogrammingReviewGateStrictGoLineBudgetIssuesV0(
	input AutoprogrammingReviewGateInputV0,
) []AutoprogrammingReviewGateIssueV0 {
	var issues []AutoprogrammingReviewGateIssueV0
	maxLines := autoprogrammingReviewGateMaxLinesV0(input)
	accepted := autoprogrammingReviewGateAcceptedPartitionFollowupsV0(input)
	for _, file := range input.Files {
		path := strings.TrimSpace(file.Path)
		if !strings.HasSuffix(path, ".go") || path == "" {
			continue
		}
		current := autoprogrammingReviewGateLineCountV0(file)
		baseline := autoprogrammingReviewGateBaselineLineCountV0(file)
		if !autoprogrammingReviewGateTrustedLineCountV0(file) {
			issues = append(issues, autoprogrammingReviewGateStrictLineBudgetIssueV0(
				path,
				"line_count_source_missing",
			))
			continue
		}
		if current <= maxLines {
			continue
		}
		if _, ok := accepted[path]; ok {
			continue
		}
		switch {
		case baseline > maxLines && current > baseline:
			issues = append(issues, autoprogrammingReviewGateStrictLineBudgetIssueV0(
				path,
				"baseline_growth",
			))
		case baseline <= maxLines:
			issues = append(issues, autoprogrammingReviewGateStrictLineBudgetIssueV0(
				path,
				"file_over_limit",
			))
		}
	}
	return issues
}

func autoprogrammingReviewGateTrustedLineCountV0(
	file AutoprogrammingReviewGateFileV0,
) bool {
	switch strings.TrimSpace(file.LineCountSource) {
	case "snapshot", "worktree", "project_file":
		return true
	default:
		return false
	}
}

func autoprogrammingReviewGateStrictGoLineBudgetV0(
	input AutoprogrammingReviewGateInputV0,
) bool {
	return input.StrictGoLineBudget
}

func autoprogrammingReviewGateAcceptedPartitionFollowupsV0(
	input AutoprogrammingReviewGateInputV0,
) map[string]struct{} {
	out := map[string]struct{}{}
	for _, path := range compactStringsV0(input.AcceptedPartitionFollowups) {
		out[path] = struct{}{}
	}
	return out
}

func autoprogrammingReviewGateStrictLineBudgetIssueV0(
	path string,
	reason string,
) AutoprogrammingReviewGateIssueV0 {
	return autoprogrammingReviewGateIssueV0(
		AutoprogrammingReviewGateStrictLineBudgetIssueV0,
		"files",
		"presupuesto estricto de lineas Go bloqueado: "+reason+":"+path,
	)
}
