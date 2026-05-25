package orquestaruntimeworktree

import (
	"strconv"
	"strings"
)

func goLineBudgetViolationsV0(
	baseline WorktreeSnapshotV0,
	current WorktreeSnapshotV0,
	changedPaths []string,
	maxLines int,
	acceptedFollowups []string,
) []WorktreeGoLineBudgetViolationV0 {
	base := worktreeSnapshotMapV0(baseline)
	now := worktreeSnapshotMapV0(current)
	changed := worktreePathSetV0(changedPaths)
	accepted := worktreePathSetV0(acceptedFollowups)
	maxLines = worktreeGoLineBudgetMaxV0(maxLines)
	var out []WorktreeGoLineBudgetViolationV0
	for path, file := range now {
		if _, ok := changed[path]; !ok || !strings.HasSuffix(path, ".go") {
			continue
		}
		if _, ok := accepted[path]; ok || file.LineCount <= maxLines {
			continue
		}
		baseLines := base[path].LineCount
		switch {
		case baseLines > maxLines && file.LineCount > baseLines:
			out = append(out, worktreeGoLineBudgetViolationV0(path, baseLines, file.LineCount, maxLines, "baseline_growth"))
		case baseLines <= maxLines:
			out = append(out, worktreeGoLineBudgetViolationV0(path, baseLines, file.LineCount, maxLines, "file_over_limit"))
		}
	}
	return out
}

func worktreeGoLineBudgetMaxV0(maxLines int) int {
	if maxLines > 0 {
		return maxLines
	}
	return WorktreeDefaultGoFileLineBudgetV0
}

func worktreePathSetV0(paths []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, path := range compactWorktreeStringsV0(paths) {
		out[path] = struct{}{}
	}
	return out
}

func worktreeGoLineBudgetViolationV0(
	path string,
	baselineLines int,
	currentLines int,
	maxLines int,
	reason string,
) WorktreeGoLineBudgetViolationV0 {
	return WorktreeGoLineBudgetViolationV0{
		Path:          path,
		BaselineLines: baselineLines,
		CurrentLines:  currentLines,
		MaxLines:      maxLines,
		Reason:        reason,
	}
}

func worktreeGoLineBudgetEvidenceV0(
	violations []WorktreeGoLineBudgetViolationV0,
) []string {
	out := make([]string, 0, len(violations))
	for _, violation := range violations {
		out = append(out, violation.Path+":"+violation.Reason+
			":"+strconv.Itoa(violation.CurrentLines))
	}
	return compactWorktreeStringsV0(out)
}
