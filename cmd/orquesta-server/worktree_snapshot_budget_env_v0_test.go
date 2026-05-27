package main

import "testing"

func TestCodexServerWorktreeSnapshotReadBudgetFromEnvV0(t *testing.T) {
	t.Setenv(envWorktreeSnapshotMaxFilesV0, "7")
	t.Setenv(envWorktreeSnapshotMaxFileBytesV0, "8")
	t.Setenv(envWorktreeSnapshotMaxTotalBytesV0, "9")

	budget := codexServerWorktreeSnapshotReadBudgetFromEnvV0()
	if budget.MaxFiles != 7 || budget.MaxFileBytes != 8 || budget.MaxTotalBytes != 9 {
		t.Fatalf("budget=%+v", budget)
	}
}
