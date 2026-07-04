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
	config, _, err := loadServerProjectConfigFileV0(projectDir)
	if err != nil {
		return codexServerWorktreeSnapshotReadBudgetFromEnvV0()
	}
	return codexServerWorktreeSnapshotReadBudgetFromConfigFileV0(config)
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
	projectDir string,
	budget orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0,
) []orquestaserver.ServerConfigSettingV0 {
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryWithSourceV0(
			envWorktreeSnapshotMaxFilesV0,
			strconv.Itoa(budget.MaxFiles),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envWorktreeSnapshotMaxFilesV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envWorktreeSnapshotMaxFileBytesV0,
			strconv.FormatInt(budget.MaxFileBytes, 10),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envWorktreeSnapshotMaxFileBytesV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envWorktreeSnapshotMaxTotalBytesV0,
			strconv.FormatInt(budget.MaxTotalBytes, 10),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envWorktreeSnapshotMaxTotalBytesV0),
		),
	}
}
