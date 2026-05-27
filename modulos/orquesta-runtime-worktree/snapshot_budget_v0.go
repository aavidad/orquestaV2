package orquestaruntimeworktree

func DefaultWorktreeSnapshotReadBudgetV0() WorktreeSnapshotReadBudgetV0 {
	return WorktreeSnapshotReadBudgetV0{
		MaxFiles:      WorktreeDefaultSnapshotMaxFilesV0,
		MaxFileBytes:  WorktreeDefaultSnapshotMaxFileBytesV0,
		MaxTotalBytes: WorktreeDefaultSnapshotMaxTotalBytesV0,
	}
}

func WorktreeSnapshotReadBudgetFromLimitsV0(
	maxFiles int,
	maxFileBytes int64,
	maxTotalBytes int64,
) WorktreeSnapshotReadBudgetV0 {
	return normalizeWorktreeSnapshotReadBudgetV0(WorktreeSnapshotReadBudgetV0{
		MaxFiles:      maxFiles,
		MaxFileBytes:  maxFileBytes,
		MaxTotalBytes: maxTotalBytes,
	})
}

func normalizeWorktreeSnapshotReadBudgetV0(
	budget WorktreeSnapshotReadBudgetV0,
) WorktreeSnapshotReadBudgetV0 {
	defaults := DefaultWorktreeSnapshotReadBudgetV0()
	if budget.MaxFiles <= 0 {
		budget.MaxFiles = defaults.MaxFiles
	}
	if budget.MaxFileBytes <= 0 {
		budget.MaxFileBytes = defaults.MaxFileBytes
	}
	if budget.MaxTotalBytes <= 0 {
		budget.MaxTotalBytes = defaults.MaxTotalBytes
	}
	return budget
}

func worktreeSnapshotRequestBudgetV0(
	request WorktreeSnapshotRequestV0,
) WorktreeSnapshotReadBudgetV0 {
	return normalizeWorktreeSnapshotReadBudgetV0(WorktreeSnapshotReadBudgetV0{
		MaxFiles:      request.MaxFiles,
		MaxFileBytes:  request.MaxFileBytes,
		MaxTotalBytes: request.MaxTotalBytes,
	})
}

func worktreeVerifySnapshotBudgetV0(
	request WorktreeVerifyRequestV0,
) WorktreeSnapshotReadBudgetV0 {
	if request.MaxSnapshotFiles > 0 ||
		request.MaxSnapshotFileBytes > 0 ||
		request.MaxSnapshotTotalBytes > 0 {
		return normalizeWorktreeSnapshotReadBudgetV0(WorktreeSnapshotReadBudgetV0{
			MaxFiles:      request.MaxSnapshotFiles,
			MaxFileBytes:  request.MaxSnapshotFileBytes,
			MaxTotalBytes: request.MaxSnapshotTotalBytes,
		})
	}
	return normalizeWorktreeSnapshotReadBudgetV0(request.Baseline.ReadBudget)
}
