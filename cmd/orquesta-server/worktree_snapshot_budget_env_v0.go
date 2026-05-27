package main

import (
	"strconv"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	envWorktreeSnapshotMaxFilesV0      = "ORQUESTA_WORKTREE_SNAPSHOT_MAX_FILES"
	envWorktreeSnapshotMaxFileBytesV0  = "ORQUESTA_WORKTREE_SNAPSHOT_MAX_FILE_BYTES"
	envWorktreeSnapshotMaxTotalBytesV0 = "ORQUESTA_WORKTREE_SNAPSHOT_MAX_TOTAL_BYTES"
)

func codexServerWorktreeSnapshotReadBudgetFromEnvV0() orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0 {
	defaults := orquestaruntimeworktree.DefaultWorktreeSnapshotReadBudgetV0()
	return orquestaruntimeworktree.WorktreeSnapshotReadBudgetFromLimitsV0(
		intEnvOrDefaultV0(envWorktreeSnapshotMaxFilesV0, defaults.MaxFiles),
		int64(intEnvOrDefaultV0(envWorktreeSnapshotMaxFileBytesV0, int(defaults.MaxFileBytes))),
		int64(intEnvOrDefaultV0(envWorktreeSnapshotMaxTotalBytesV0, int(defaults.MaxTotalBytes))),
	)
}

func codexServerWorktreeSnapshotBudgetSettingsV0(
	budget orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0,
) []orquestaserver.ServerConfigSettingV0 {
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingV0(
			envWorktreeSnapshotMaxFilesV0,
			strconv.Itoa(budget.MaxFiles),
			"worktree_snapshot",
			"Max ficheros snapshot",
			"Numero maximo de ficheros de producto que puede capturar el snapshot.",
		),
		serverConfigSettingV0(
			envWorktreeSnapshotMaxFileBytesV0,
			strconv.FormatInt(budget.MaxFileBytes, 10),
			"worktree_snapshot",
			"Max bytes fichero snapshot",
			"Tamano maximo por fichero durante hash streaming del snapshot.",
		),
		serverConfigSettingV0(
			envWorktreeSnapshotMaxTotalBytesV0,
			strconv.FormatInt(budget.MaxTotalBytes, 10),
			"worktree_snapshot",
			"Max bytes total snapshot",
			"Tamano total maximo leido por snapshot de worktree.",
		),
	}
}
