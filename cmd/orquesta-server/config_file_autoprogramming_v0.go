package main

type serverProjectConfigAutoprogrammingV0 struct {
	CheckpointOnlyHighConsumptionTokens *int                                          `json:"checkpoint_only_high_consumption_tokens,omitempty"`
	CheckpointOnlyMaxWaitSeconds        *int                                          `json:"checkpoint_only_max_wait_seconds,omitempty"`
	NoCheckpointWarningMaxWaitSeconds   *int                                          `json:"no_checkpoint_warning_max_wait_seconds,omitempty"`
	RequiredTestAttestationConfigFile   *string                                       `json:"required_test_attestation_config_file,omitempty"`
	Promotion                           serverProjectConfigAutoprogrammingPromotionV0 `json:"promotion,omitempty"`
}

type serverProjectConfigAutoprogrammingPromotionV0 struct {
	Enabled       *bool                                                 `json:"enabled,omitempty"`
	ArchiveDir    *string                                               `json:"archive_dir,omitempty"`
	RepoRef       *string                                               `json:"repo_ref,omitempty"`
	AppRef        *string                                               `json:"app_ref,omitempty"`
	CommitMessage *string                                               `json:"commit_message,omitempty"`
	Guardian      serverProjectConfigAutoprogrammingPromotionGuardianV0 `json:"guardian,omitempty"`
}

type serverProjectConfigAutoprogrammingPromotionGuardianV0 struct {
	Enabled                    *bool    `json:"enabled,omitempty"`
	Command                    *string  `json:"command,omitempty"`
	RunnerEnvAllowlist         []string `json:"runner_env_allowlist,omitempty"`
	StateDir                   *string  `json:"state_dir,omitempty"`
	CurrentBin                 *string  `json:"current_bin,omitempty"`
	CandidateBin               *string  `json:"candidate_bin,omitempty"`
	LastGoodBin                *string  `json:"last_good_bin,omitempty"`
	ArtifactRoot               *string  `json:"artifact_root,omitempty"`
	BuildCommand               *string  `json:"build_command,omitempty"`
	TestCommands               []string `json:"test_commands,omitempty"`
	HealthTimeout              *string  `json:"health_timeout,omitempty"`
	CommandTimeout             *string  `json:"command_timeout,omitempty"`
	ArtifactMaxBytes           *string  `json:"artifact_max_bytes,omitempty"`
	RepairCommand              *string  `json:"repair_command,omitempty"`
	RepairCodex                *bool    `json:"repair_codex,omitempty"`
	RepairCodexWriteSet        []string `json:"repair_codex_write_set,omitempty"`
	RepairCodexRequiredTests   []string `json:"repair_codex_required_tests,omitempty"`
	RepairCodexWorktreeRef     *string  `json:"repair_codex_worktree_ref,omitempty"`
	RepairCodexBranchRef       *string  `json:"repair_codex_branch_ref,omitempty"`
	RepairCodexRunRef          *string  `json:"repair_codex_run_ref,omitempty"`
	RepairCodexPromotionRef    *string  `json:"repair_codex_promotion_ref,omitempty"`
	RepairCodexSandbox         *string  `json:"repair_codex_sandbox,omitempty"`
	RepairCodexReasoning       *string  `json:"repair_codex_reasoning_effort,omitempty"`
	RepairCodexRuntimeDir      *string  `json:"repair_codex_runtime_dir,omitempty"`
	RepairCodexAllowBroad      *bool    `json:"repair_codex_allow_broad_sandbox,omitempty"`
	RepairCodexSandboxEvidence *string  `json:"repair_codex_sandbox_evidence_ref,omitempty"`
	CommandEffectEvidenceRefs  []string `json:"command_effect_evidence_refs,omitempty"`
	SkipHealthEvidenceRefs     []string `json:"skip_health_evidence_refs,omitempty"`
}
