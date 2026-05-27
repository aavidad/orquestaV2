package main

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

func normalizeGuardianConfigV0(config guardianConfigV0) (guardianConfigV0, error) {
	return normalizeGuardianConfigForCommandV0(config, true)
}

func normalizeGuardianConfigForCommandV0(
	config guardianConfigV0,
	requireCurrentBin bool,
) (guardianConfigV0, error) {
	projectDir, err := filepath.Abs(strings.TrimSpace(config.ProjectDir))
	if err != nil {
		return guardianConfigV0{}, err
	}
	config.ProjectDir = projectDir
	config.StateDir = absPathFromBaseV0(config.ProjectDir, config.StateDir)
	if err := normalizeGuardianBinaryPathsV0(&config, requireCurrentBin); err != nil {
		return guardianConfigV0{}, err
	}
	if strings.TrimSpace(config.ArtifactRoot) != "" {
		config.ArtifactRoot = absPathFromBaseV0(config.ProjectDir, config.ArtifactRoot)
	}
	normalizeGuardianRepairConfigV0(&config)
	if err := validateGuardianRepairConfigV0(config); err != nil {
		return guardianConfigV0{}, err
	}
	normalizeGuardianRuntimeConfigV0(&config)
	return applyGuardianPathPolicyV0(config)
}

func normalizeGuardianBinaryPathsV0(config *guardianConfigV0, requireCurrentBin bool) error {
	if config.CurrentBin == "" {
		if requireCurrentBin {
			return errors.New("current-bin requerido")
		}
		return nil
	}
	config.CurrentBin = absPathFromBaseV0(config.ProjectDir, config.CurrentBin)
	if config.CandidateBin == "" {
		config.CandidateBin = filepath.Join(config.StateDir, "candidate", filepath.Base(config.CurrentBin))
	}
	config.CandidateBin = absPathFromBaseV0(config.ProjectDir, config.CandidateBin)
	if config.LastGoodBin == "" {
		config.LastGoodBin = filepath.Join(config.StateDir, "last_good", filepath.Base(config.CurrentBin))
	}
	config.LastGoodBin = absPathFromBaseV0(config.ProjectDir, config.LastGoodBin)
	return nil
}

func normalizeGuardianRepairConfigV0(config *guardianConfigV0) {
	if config.RepairCodexRuntimeDir == "" {
		config.RepairCodexRuntimeDir = filepath.Join(config.StateDir, "repair-codex-runtime")
	}
	config.RepairCodexRuntimeDir = absPathFromBaseV0(config.ProjectDir, config.RepairCodexRuntimeDir)
	if len(config.RepairCodexRequiredTests) == 0 {
		config.RepairCodexRequiredTests = append([]string(nil), config.TestCommands...)
	}
	if config.RepairCodexWorktreeRef == "" {
		config.RepairCodexWorktreeRef = "worktree-ref-guardian-repair"
	}
	if config.RepairCodexBranchRef == "" {
		config.RepairCodexBranchRef = "branch-ref-guardian-repair"
	}
}

func validateGuardianRepairConfigV0(config guardianConfigV0) error {
	if config.RepairCodex && len(config.RepairCodexWriteSet) == 0 {
		return errors.New("repair-codex-write-set requerido")
	}
	if config.RepairCodex && len(config.RepairCodexRequiredTests) == 0 {
		return errors.New("repair-codex-required-tests requerido")
	}
	if config.RepairCodex && guardianRepairCodexSandboxBroadV0(config.RepairCodexSandbox) &&
		(!config.RepairCodexAllowBroadSandbox || strings.TrimSpace(config.RepairCodexSandboxEvidenceRef) == "") {
		return errors.New("repair-codex-broad-sandbox-requires-opt-in")
	}
	return nil
}

func normalizeGuardianRuntimeConfigV0(config *guardianConfigV0) {
	if config.BuildCommand == "" {
		config.BuildCommand = guardianCanonicalBuildCommandV0(*config)
	}
	if config.HealthTimeout <= 0 {
		config.HealthTimeout = 20 * time.Second
	}
	if config.CommandTimeout <= 0 {
		config.CommandTimeout = 10 * time.Minute
	}
	if config.CommandOutputMaxBytes <= 0 {
		config.CommandOutputMaxBytes = defaultGuardianCommandOutputMaxBytesV0
	}
	if config.ArtifactMaxBytes <= 0 {
		config.ArtifactMaxBytes = defaultGuardianArtifactMaxBytesV0
	}
	config.EnvAllowlist = compactStringsV0(config.EnvAllowlist)
	config.CommandEffectEvidenceRefs = compactStringsV0(config.CommandEffectEvidenceRefs)
	config.SkipHealthEvidenceRefs = compactStringsV0(config.SkipHealthEvidenceRefs)
	config.RepairRetryEvidenceRefs = compactStringsV0(config.RepairRetryEvidenceRefs)
	if config.RepairMaxAttempts <= 0 {
		config.RepairMaxAttempts = 1
	}
	if len(config.EnvAllowlist) == 0 {
		config.EnvAllowlist = defaultGuardianEnvAllowlistV0()
	}
	if config.ShutdownNow {
		config.ShutdownForced = true
	}
	if config.LeaseTTL <= 0 {
		config.LeaseTTL = 30 * time.Minute
	}
	normalizeGuardianRefsAndPathsV0(config)
}

func normalizeGuardianRefsAndPathsV0(config *guardianConfigV0) {
	config.PromotionRef = strings.TrimSpace(config.PromotionRef)
	config.AttemptRef = strings.TrimSpace(config.AttemptRef)
	config.RunRef = strings.TrimSpace(config.RunRef)
	config.WorktreeRef = strings.TrimSpace(config.WorktreeRef)
	config.BranchRef = strings.TrimSpace(config.BranchRef)
	config.ShutdownRef = strings.TrimSpace(config.ShutdownRef)
	if config.AttemptRef == "" {
		config.AttemptRef = guardianAttemptRefV0(*config)
	}
	config.ShutdownEscalationEvidenceRef = strings.TrimSpace(config.ShutdownEscalationEvidenceRef)
	attemptFile := safeGuardianAttemptFilePartV0(config.AttemptRef)
	config.CandidateStateDir = filepath.Join(config.StateDir, "candidate-state", attemptFile)
	config.CandidateRuntimeDir = filepath.Join(config.StateDir, "candidate-runtime", attemptFile)
	config.ManifestPath = filepath.Join(config.StateDir, "manifests", "guardian-"+attemptFile+".json")
	config.RepairPacketPath = filepath.Join(config.StateDir, "repair", "repair-"+attemptFile+".json")
}
