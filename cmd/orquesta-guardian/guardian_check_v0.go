package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func runGuardianCheckPromoteV0(ctx context.Context, config guardianConfigV0) guardianResultV0 {
	result := baseGuardianResultV0(config)
	if err := ensureGuardianDirsV0(config); err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Phase = "prepare"
		result.Message = err.Error()
		return writeGuardianManifestV0(config, result)
	}
	lease, err := acquireGuardianPromotionLeaseV0(config, "check-promote")
	result.Lease = &lease
	if err != nil {
		result.Status = guardianStatusLeaseBusyV0
		result.Phase = "lease"
		result.Message = err.Error()
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-promotion-lease-busy")
		return writeGuardianManifestV0(config, result)
	}
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-promotion-lease-acquired")
	defer releaseGuardianPromotionLeaseV0(config, result.Lease)
	build := runGuardianPolicyShellCommandV0(ctx, config, "build", config.BuildCommand)
	result.Commands = append(result.Commands, build)
	if build.ExitCode != 0 {
		return failGuardianCandidateV0(ctx, config, result, "build", "candidate build failed")
	}
	for index, testCommand := range config.TestCommands {
		phase := "test-" + strconv.Itoa(index+1)
		test := runGuardianPolicyShellCommandV0(ctx, config, phase, testCommand)
		result.Commands = append(result.Commands, test)
		if test.ExitCode != 0 {
			return failGuardianCandidateV0(ctx, config, result, phase, "candidate required tests failed")
		}
	}
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-candidate-built")
	if len(config.TestCommands) > 0 {
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-candidate-tests-passed")
	}
	if config.SkipHealth {
		if config.Promote && len(config.SkipHealthEvidenceRefs) == 0 {
			result.Status = guardianStatusCandidateFailedV0
			result.Phase = "healthcheck"
			result.Message = "guardian_healthcheck_required"
			result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-healthcheck-required")
			return writeGuardianManifestV0(config, result)
		}
		result.EvidenceRefs = append(result.EvidenceRefs, config.SkipHealthEvidenceRefs...)
		result.EvidenceRefs = append(result.EvidenceRefs,
			"evidence-ref-guardian-candidate-not-live-checked",
			"evidence-ref-guardian-skip-health-breakglass",
		)
	} else {
		health := runGuardianCandidateHealthcheckV0(ctx, config)
		result.Commands = append(result.Commands, health)
		if health.ExitCode != 0 {
			return failGuardianCandidateV0(ctx, config, result, "healthcheck", "candidate healthcheck failed")
		}
		result.EvidenceRefs = append(result.EvidenceRefs,
			"evidence-ref-guardian-candidate-address-owned",
			"evidence-ref-guardian-candidate-readiness-target",
			"evidence-ref-guardian-candidate-attempt",
			"evidence-ref-guardian-candidate-readiness-passed",
		)
		if health.StopReceipt != nil && guardianCandidateStopPassedV0(*health.StopReceipt) {
			result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-candidate-process-stop-confirmed")
		}
	}
	if !config.Promote {
		result.Status = guardianStatusPromotedV0
		result.Phase = "verified"
		result.Message = "candidate verified without promotion"
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-candidate-verified")
		return writeGuardianManifestV0(config, result)
	}
	if err := verifyGuardianPromotionLeaseV0(config, lease, "promote"); err != nil {
		result.Status = guardianStatusLeaseLostV0
		result.Phase = "promote"
		result.Message = err.Error()
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-promotion-lease-lost")
		return writeGuardianManifestV0(config, result)
	}
	artifactManifest, err := promoteGuardianCandidateV0(config)
	result.ArtifactManifest = artifactManifest
	if err != nil {
		result.Status = guardianStatusPromotionIncompleteV0
		if artifactManifest != nil && artifactManifest.Status != "" {
			result.Status = artifactManifest.Status
		}
		result.Phase = "promote"
		result.Message = err.Error()
		return writeGuardianManifestV0(config, result)
	}
	result.Status = guardianStatusPromotedV0
	if config.SkipHealth {
		result.Status = guardianStatusPromotedBreakglassV0
	}
	result.Phase = "promote"
	result.Promoted = true
	result.EvidenceRefs = append(result.EvidenceRefs,
		"evidence-ref-guardian-candidate-promoted",
		"evidence-ref-guardian-artifact-promotion-manifest",
	)
	result.Message = "candidate promoted"
	return writeGuardianManifestV0(config, result)
}

func failGuardianCandidateV0(
	ctx context.Context,
	config guardianConfigV0,
	result guardianResultV0,
	phase string,
	message string,
) guardianResultV0 {
	result.Status = guardianStatusCandidateFailedV0
	result.Phase = phase
	result.Message = message
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-candidate-failed")
	receipt, err := writeGuardianRepairPacketV0(config, result)
	if err == nil {
		result.RepairPacketPath = receipt.Path
		result.RepairAttemptRef = receipt.RepairAttemptRef
		result.FailurePacketHash = receipt.FailurePacketHash
	}
	if receipt.Path != "" && (strings.TrimSpace(config.RepairCommand) != "" || config.RepairCodex) {
		attempt := guardianClaimRepairAttemptV0(config, phase, receipt)
		result.EvidenceRefs = append(result.EvidenceRefs, attempt.EvidenceRefs...)
		if !attempt.Allowed {
			result.RepairBlocked = true
			result.RepairBlockReason = attempt.ReasonCode
			return writeGuardianManifestV0(config, result)
		}
		repair := runGuardianRepairCommandV0(ctx, config, phase, receipt.Path, receipt)
		result.Commands = append(result.Commands, repair)
		result.RepairStarted = true
		if config.RepairCodex {
			result.RepairLaunchPath = strings.TrimSuffix(receipt.Path, filepath.Ext(receipt.Path)) + ".launch.json"
		}
	}
	return writeGuardianManifestV0(config, result)
}

func baseGuardianResultV0(config guardianConfigV0) guardianResultV0 {
	return guardianResultV0{
		SchemaVersion: guardianResultSchemaVersionV0,
		ProjectDir:    config.ProjectDir,
		CurrentBin:    config.CurrentBin,
		CandidateBin:  config.CandidateBin,
		LastGoodBin:   config.LastGoodBin,
		ManifestPath:  config.ManifestPath,
		PathPolicy:    config.PathPolicy,
		AttemptRef:    config.AttemptRef,
		PromotionRef:  config.PromotionRef,
		RunRef:        config.RunRef,
		WorktreeRef:   config.WorktreeRef,
		BranchRef:     config.BranchRef,
		ShutdownRef:   config.ShutdownRef,
		EvidenceRefs: []string{
			"evidence-ref-guardian-breakglass",
		},
	}
}

func ensureGuardianDirsV0(config guardianConfigV0) error {
	for _, dir := range []string{
		config.StateDir,
		filepath.Dir(config.CandidateBin),
		filepath.Dir(config.LastGoodBin),
		filepath.Dir(config.ManifestPath),
		filepath.Dir(config.RepairPacketPath),
		filepath.Join(config.StateDir, "logs"),
		config.CandidateStateDir,
		config.CandidateRuntimeDir,
		config.RepairCodexRuntimeDir,
		filepath.Join(config.StateDir, "repair-attempts"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}
