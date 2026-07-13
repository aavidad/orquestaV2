package main

import (
	"strconv"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func codexServerWorktreeSnapshotReadBudgetFromEnvV0() orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0 {
	defaults := orquestaruntimeworktree.DefaultWorktreeSnapshotReadBudgetV0()
	return orquestaruntimeworktree.WorktreeSnapshotReadBudgetFromLimitsV0(
		intEnvOrDefaultV0(envWorktreeSnapshotMaxFilesV0, defaults.MaxFiles),
		int64(intEnvOrDefaultV0(envWorktreeSnapshotMaxFileBytesV0, int(defaults.MaxFileBytes))),
		int64(intEnvOrDefaultV0(envWorktreeSnapshotMaxTotalBytesV0, int(defaults.MaxTotalBytes))),
	)
}

func codexServerWorktreeSnapshotReadBudgetFromProjectConfigV0(
	projectDir string,
) orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0 {
	return codexServerWorktreeSnapshotReadBudgetFromConfigFileV0(
		projectConfigFromProjectDirBestEffortV0(projectDir),
	)
}

func codexServerWorktreeSnapshotReadBudgetFromConfigFileV0(
	config serverProjectConfigFileV0,
) orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0 {
	defaults := orquestaruntimeworktree.DefaultWorktreeSnapshotReadBudgetV0()
	return orquestaruntimeworktree.WorktreeSnapshotReadBudgetFromLimitsV0(
		intProjectConfigOrEnvOrDefaultV0(
			envWorktreeSnapshotMaxFilesV0,
			config.WorktreeSnapshot.MaxFiles,
			defaults.MaxFiles,
		),
		int64(intProjectConfigOrEnvOrDefaultV0(
			envWorktreeSnapshotMaxFileBytesV0,
			config.WorktreeSnapshot.MaxFileBytes,
			int(defaults.MaxFileBytes),
		)),
		int64(intProjectConfigOrEnvOrDefaultV0(
			envWorktreeSnapshotMaxTotalBytesV0,
			config.WorktreeSnapshot.MaxTotalBytes,
			int(defaults.MaxTotalBytes),
		)),
	)
}

func codexServerWorktreeSnapshotBudgetSettingsV0(
	serverConfig orquestaserver.ConfigV0,
	budget orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0,
) []orquestaserver.ServerConfigSettingV0 {
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryWithSourceV0(
			envWorktreeSnapshotMaxFilesV0,
			strconv.Itoa(budget.MaxFiles),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envWorktreeSnapshotMaxFilesV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envWorktreeSnapshotMaxFileBytesV0,
			strconv.FormatInt(budget.MaxFileBytes, 10),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envWorktreeSnapshotMaxFileBytesV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envWorktreeSnapshotMaxTotalBytesV0,
			strconv.FormatInt(budget.MaxTotalBytes, 10),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envWorktreeSnapshotMaxTotalBytesV0),
		),
	}
}
