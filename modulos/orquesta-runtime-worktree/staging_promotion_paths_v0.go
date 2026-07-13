package orquestaruntimeworktree

import "strings"

func stagingPromotionProductEntriesV0(
	entries []stagingPromotionGitStatusEntryV0,
) ([]stagingPromotionGitStatusEntryV0, []WorktreeIssueV0) {
	product := make([]stagingPromotionGitStatusEntryV0, 0, len(entries))
	var controlPaths []string
	for _, entry := range entries {
		if IsWorktreeControlPathV0(entry.Path) {
			controlPaths = append(controlPaths, entry.Path)
			continue
		}
		product = append(product, entry)
	}
	return product, worktreeControlPathIssuesV0(controlPaths)
}

func stagingPromotionChangedPathsV0(entries []stagingPromotionGitStatusEntryV0) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	return compactWorktreeStringsV0(paths)
}

func stagingPromotionWriteSetIssuesV0(
	entries []stagingPromotionGitStatusEntryV0,
	writeSet []string,
) []WorktreeIssueV0 {
	var issues []WorktreeIssueV0
	for _, entry := range entries {
		if IsWorktreeControlPathV0(entry.Path) {
			continue
		}
		if strings.Contains(entry.Code, "D") {
			issues = append(issues, worktreeIssueV0(WorktreeIssueRemovedPathV0, entry.Path))
			continue
		}
		if !worktreePathAllowedV0(entry.Path, writeSet) {
			issues = append(issues, worktreeIssueV0(WorktreeIssueOutsideWriteSetV0, entry.Path))
		}
	}
	return issues
}
