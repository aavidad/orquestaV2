package orquestaruntimeworktree

const (
	worktreeSnapshotFileBudgetCategoryV0      = "snapshot_file_budget"
	worktreeSnapshotFileCountBudgetCategoryV0 = "snapshot_file_count_budget"
	worktreeSnapshotTotalBudgetCategoryV0     = "snapshot_total_budget"
)

func worktreeSnapshotBudgetReceiptForIssueV0(
	issue WorktreeIssueV0,
) (WorktreeLocalArtifactExclusionReceiptV0, bool) {
	switch issue.Code {
	case WorktreeIssueSnapshotFileTooLargeV0:
		return worktreeSnapshotBudgetReceiptV0(
			worktreeSnapshotFileBudgetCategoryV0,
			WorktreeIssueSnapshotFileTooLargeV0,
		), true
	case WorktreeIssueSnapshotTooManyFilesV0:
		return worktreeSnapshotBudgetReceiptV0(
			worktreeSnapshotFileCountBudgetCategoryV0,
			WorktreeIssueSnapshotTooManyFilesV0,
		), true
	case WorktreeIssueSnapshotTooLargeV0:
		return worktreeSnapshotBudgetReceiptV0(
			worktreeSnapshotTotalBudgetCategoryV0,
			WorktreeIssueSnapshotTooLargeV0,
		), true
	default:
		return WorktreeLocalArtifactExclusionReceiptV0{}, false
	}
}

func worktreeSnapshotBudgetReceiptV0(
	category string,
	code WorktreeIssueCodeV0,
) WorktreeLocalArtifactExclusionReceiptV0 {
	return WorktreeLocalArtifactExclusionReceiptV0{
		Category:   category,
		ReasonCode: string(code),
		Count:      1,
	}
}

func worktreeSnapshotBudgetEvidenceRefsV0(
	receipts []WorktreeLocalArtifactExclusionReceiptV0,
) []string {
	var refs []string
	for _, receipt := range receipts {
		if receipt.Count <= 0 {
			continue
		}
		switch receipt.ReasonCode {
		case string(WorktreeIssueSnapshotFileTooLargeV0):
			refs = append(refs, "evidence-ref-worktree-snapshot-file-too-large-excluded-v0")
		case string(WorktreeIssueSnapshotTooManyFilesV0):
			refs = append(refs, "evidence-ref-worktree-snapshot-too-many-files-partial-v0")
		case string(WorktreeIssueSnapshotTooLargeV0):
			refs = append(refs, "evidence-ref-worktree-snapshot-total-budget-partial-v0")
		}
	}
	return compactWorktreeStringsV0(refs)
}
