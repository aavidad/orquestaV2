package main

func init() {
	serverEffectiveEnvRegistryV0[envWorktreeSnapshotMaxFilesV0] = serverEnvSettingMetadataV0{
		Scope:       "worktree_snapshot",
		Label:       "Max ficheros snapshot",
		Description: "Numero maximo de ficheros de producto que puede capturar el snapshot.",
	}
	serverEffectiveEnvRegistryV0[envWorktreeSnapshotMaxFileBytesV0] = serverEnvSettingMetadataV0{
		Scope:       "worktree_snapshot",
		Label:       "Max bytes fichero snapshot",
		Description: "Tamano maximo por fichero durante hash streaming del snapshot.",
	}
	serverEffectiveEnvRegistryV0[envWorktreeSnapshotMaxTotalBytesV0] = serverEnvSettingMetadataV0{
		Scope:       "worktree_snapshot",
		Label:       "Max bytes total snapshot",
		Description: "Tamano total maximo leido por snapshot de worktree.",
	}
}
