package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func runGuardianShellCommandV0(
	ctx context.Context,
	config guardianConfigV0,
	phase string,
	command string,
) guardianCommandResultV0 {
	start := time.Now()
	outputPath := filepath.Join(config.StateDir, "logs", sanitizeFilenamePartV0(phase)+"-"+start.UTC().Format("20060102T150405.000000000Z")+".log")
	decision := authorizeGuardianCommandEffectV0(config, phase, command)
	if decision.Status == guardianCommandEffectBlockedV0 {
		return blockedGuardianCommandResultV0(config, phase, command, decision.ReasonCode)
	}
	commandCtx, cancel := context.WithTimeout(ctx, config.CommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, "sh", "-c", command)
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(commandCtx, "cmd", "/C", command)
	}
	cmd.Dir = config.ProjectDir
	cmd.Env = guardianCommandEnvV0(config, nil)
	output := newGuardianOutputCaptureV0(config)
	cmd.Stdout = output
	cmd.Stderr = output
	err := cmd.Run()
	_ = output.WriteLog(outputPath)
	return applyGuardianCommandEffectV0(guardianCommandResultV0{
		Phase:            phase,
		Command:          command,
		ExitCode:         commandExitCodeV0(err),
		DurationMS:       time.Since(start).Milliseconds(),
		OutputPath:       outputPath,
		Error:            redactGuardianDiagnosticV0(config, commandErrorStringV0(err, commandCtx.Err())),
		OutputBytes:      output.BytesSeen(),
		OutputTruncated:  output.Truncated(),
		OutputReasonCode: output.ReasonCode(),
	}, decision)
}

func runGuardianCandidateHealthcheckV0(ctx context.Context, config guardianConfigV0) guardianCommandResultV0 {
	start := time.Now()
	outputPath := filepath.Join(config.StateDir, "logs", "healthcheck-"+start.UTC().Format("20060102T150405.000000000Z")+".log")
	addr, addrPolicy, err := guardianCandidateAddressPolicyV0(config.CandidateAddr)
	if err != nil {
		_ = writeGuardianRedactedLogV0(config, outputPath, err.Error())
		return applyGuardianCommandEffectV0(guardianCommandResultV0{Phase: "healthcheck", Command: "candidate healthcheck", ExitCode: 1, DurationMS: time.Since(start).Milliseconds(), OutputPath: outputPath, Error: redactGuardianDiagnosticV0(config, err.Error()), ReasonCode: "candidate_addr_unowned"}, authorizeGuardianCommandEffectV0(config, "healthcheck", "candidate healthcheck"))
	}
	candidateState := config.CandidateStateDir
	candidateRuntime := config.CandidateRuntimeDir
	_ = os.MkdirAll(candidateState, 0o700)
	_ = os.MkdirAll(candidateRuntime, 0o700)
	commandCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, config.CandidateBin, "run")
	cmd.Dir = config.ProjectDir
	cmd.Env = guardianCommandEnvV0(config, []string{
		"ORQUESTA_SERVER_ADDR=" + addr,
		"ORQUESTA_SERVER_STATE_DIR=" + candidateState,
		"ORQUESTA_SERVER_RUNTIME_WORKDIR=" + candidateRuntime,
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0",
		"ORQUESTA_OPES_BRIDGE_DRY_RUN=1",
	})
	policy := guardianPrepareCandidateCommandV0(cmd)
	output := newGuardianOutputCaptureV0(config)
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		_ = writeGuardianRedactedLogV0(config, outputPath, err.Error())
		return applyGuardianCommandEffectV0(guardianCommandResultV0{Phase: "healthcheck", Command: config.CandidateBin + " run", ExitCode: 1, DurationMS: time.Since(start).Milliseconds(), OutputPath: outputPath, Error: redactGuardianDiagnosticV0(config, err.Error()), ProcessPolicy: &policy}, authorizeGuardianCommandEffectV0(config, "healthcheck", "candidate healthcheck"))
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	targetAddr := addr
	if addrPolicy.Dynamic {
		var err error
		targetAddr, err = waitGuardianCandidateStateAddrV0(ctx, candidateState, config.HealthTimeout)
		if err != nil {
			receipt := stopGuardianCandidateProcessV0(cmd, waitDone, 2*time.Second)
			_ = output.WriteLog(outputPath)
			return applyGuardianCommandEffectV0(guardianCommandResultV0{Phase: "healthcheck", Command: config.CandidateBin + " run", ExitCode: 1, DurationMS: time.Since(start).Milliseconds(), OutputPath: outputPath, Error: redactGuardianDiagnosticV0(config, err.Error()), ReasonCode: "candidate_addr_unowned", OutputBytes: output.BytesSeen(), OutputTruncated: output.Truncated(), OutputReasonCode: output.ReasonCode(), ProcessPolicy: &policy, StopReceipt: &receipt}, authorizeGuardianCommandEffectV0(config, "healthcheck", "candidate healthcheck"))
		}
	}
	_, err = waitGuardianCandidateReadinessV0(ctx, "http://"+targetAddr, config.HealthTimeout)
	if err == nil {
		err = guardianCandidateProcessStillOwnedV0(waitDone)
	}
	receipt := stopGuardianCandidateProcessV0(cmd, waitDone, 2*time.Second)
	if err == nil && (receipt.Ambiguous || !receipt.TreeStopConfirmed) {
		err = errGuardianCandidateStopReceiptV0(receipt)
	}
	_ = output.WriteLog(outputPath)
	if err != nil {
		return applyGuardianCommandEffectV0(guardianCommandResultV0{Phase: "healthcheck", Command: config.CandidateBin + " run", ExitCode: 1, DurationMS: time.Since(start).Milliseconds(), OutputPath: outputPath, Error: redactGuardianDiagnosticV0(config, err.Error()), ReasonCode: guardianCandidateReadinessReasonWithOwnershipV0(err), OutputBytes: output.BytesSeen(), OutputTruncated: output.Truncated(), OutputReasonCode: output.ReasonCode(), ProcessPolicy: &policy, StopReceipt: &receipt}, authorizeGuardianCommandEffectV0(config, "healthcheck", "candidate healthcheck"))
	}
	return applyGuardianCommandEffectV0(guardianCommandResultV0{Phase: "healthcheck", Command: config.CandidateBin + " run", ExitCode: 0, DurationMS: time.Since(start).Milliseconds(), OutputPath: outputPath, ReasonCode: "candidate_address_owned", OutputBytes: output.BytesSeen(), OutputTruncated: output.Truncated(), OutputReasonCode: output.ReasonCode(), ProcessPolicy: &policy, StopReceipt: &receipt}, authorizeGuardianCommandEffectV0(config, "healthcheck", "candidate healthcheck"))
}

func promoteGuardianCandidateV0(config guardianConfigV0) (*guardianArtifactManifestV0, error) {
	manifest, err := promoteGuardianCandidateArtifactsV0(config)
	if err != nil {
		return manifest, guardianArtifactErrorV0(err)
	}
	return manifest, nil
}

func restoreLastGoodCommandV0(config guardianConfigV0) guardianResultV0 {
	result := baseGuardianResultV0(config)
	if err := ensureGuardianDirsV0(config); err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Phase = "restore"
		result.Message = err.Error()
		return writeGuardianManifestV0(config, result)
	}
	lease, err := acquireGuardianPromotionLeaseV0(config, "restore")
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
	if err := verifyGuardianPromotionLeaseV0(config, lease, "restore"); err != nil {
		result.Status = guardianStatusLeaseLostV0
		result.Phase = "restore"
		result.Message = err.Error()
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-promotion-lease-lost")
		return writeGuardianManifestV0(config, result)
	}
	artifactManifest, err := restoreGuardianLastGoodArtifactsV0(config)
	result.ArtifactManifest = artifactManifest
	if err != nil {
		result.Status = guardianStatusLastGoodUnverifiedV0
		if artifactManifest != nil && artifactManifest.Status != "" {
			result.Status = artifactManifest.Status
		}
		result.Phase = "restore"
		result.Message = err.Error()
		return writeGuardianManifestV0(config, result)
	}
	result.Status = guardianStatusRestoredV0
	result.Phase = "restore"
	result.Restored = true
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-last-good-restored")
	result.Message = "last_good restored"
	return writeGuardianManifestV0(config, result)
}
