package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func writeGuardianRepairPacketV0(config guardianConfigV0, result guardianResultV0) (guardianRepairPacketReceiptV0, error) {
	packet := repairGuardianPacketV0(config, result)
	packet.RetentionStatus = applyGuardianRetentionV0(config, result)
	packet, receipt := finalizeGuardianRepairPacketV0(config, packet)
	if err := writeJSONFileV0(receipt.Path, packet); err != nil {
		if guardianRepairPacketConflictMatchesV0(receipt.Path, receipt.FailurePacketHash, err) {
			return receipt, nil
		}
		return guardianRepairPacketReceiptV0{}, err
	}
	return receipt, nil
}

func guardianRepairPacketConflictMatchesV0(path string, failureHash string, err error) bool {
	var writeErr guardianDurableWriteErrorV0
	if !errors.As(err, &writeErr) || writeErr.Code != "guardian_manifest_payload_conflict" {
		return false
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return false
	}
	var existing guardianRepairPacketV0
	if json.Unmarshal(data, &existing) != nil {
		return false
	}
	return strings.TrimSpace(existing.FailurePacketHash) == strings.TrimSpace(failureHash)
}

func runGuardianRepairCommandV0(
	ctx context.Context,
	config guardianConfigV0,
	phase string,
	packetPath string,
	receipt guardianRepairPacketReceiptV0,
) guardianCommandResultV0 {
	command, expandErr := expandGuardianCommandCheckedV0(config.RepairCommand, config)
	if expandErr != nil {
		return blockedGuardianCommandResultV0(config, "repair", config.RepairCommand, guardianCommandExpansionReasonV0)
	}
	if command == "" && config.RepairCodex {
		var err error
		var launch guardianCodexRepairLaunchV0
		launch, err = guardianCodexRepairCommandV0(config, phase, packetPath, receipt)
		if err != nil {
			return guardianCommandResultV0{
				Phase:    "repair",
				Command:  "guardian codex repair command",
				ExitCode: 1,
				Error:    err.Error(),
			}
		}
		command = launch.Command
	}
	decision := authorizeGuardianCommandEffectV0(config, "repair", command)
	if decision.Status == guardianCommandEffectBlockedV0 {
		return blockedGuardianCommandResultV0(config, "repair", command, decision.ReasonCode)
	}
	commandCtx, cancel := context.WithTimeout(ctx, config.CommandTimeout)
	defer cancel()
	start := time.Now()
	outputPath := filepath.Join(config.StateDir, "logs", "repair-"+start.UTC().Format("20060102T150405.000000000Z")+".log")
	cmd := exec.CommandContext(commandCtx, "sh", "-c", command)
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(commandCtx, "cmd", "/C", command)
	}
	cmd.Dir = config.ProjectDir
	cmd.Env = guardianCommandEnvV0(config, []string{
		"ORQUESTA_GUARDIAN_REPAIR_PACKET=" + packetPath,
		"ORQUESTA_GUARDIAN_FAILURE_PHASE=" + phase,
		"ORQUESTA_GUARDIAN_REPAIR_ATTEMPT_REF=" + receipt.RepairAttemptRef,
		"ORQUESTA_GUARDIAN_FAILURE_PACKET_HASH=" + receipt.FailurePacketHash,
	})
	output := newGuardianOutputCaptureV0(config)
	cmd.Stdout = output
	cmd.Stderr = output
	err := cmd.Run()
	_ = output.WriteLog(outputPath)
	return applyGuardianCommandEffectV0(guardianCommandResultV0{
		Phase:            "repair",
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

type guardianCodexRepairLaunchV0 struct {
	Command          string
	LaunchPacketPath string
	ObjectivePath    string
}

func guardianCodexRepairCommandV0(
	config guardianConfigV0,
	phase string,
	packetPath string,
	receipt guardianRepairPacketReceiptV0,
) (guardianCodexRepairLaunchV0, error) {
	inspection, err := inspectGuardianRepairPacketForLaunchV0(config, phase, packetPath)
	if err != nil {
		return guardianCodexRepairLaunchV0{}, err
	}
	if inspection.Packet.RepairAttemptRef != receipt.RepairAttemptRef ||
		inspection.Packet.FailurePacketHash != receipt.FailurePacketHash {
		return guardianCodexRepairLaunchV0{}, guardianRepairPacketInvalidV0("receipt_mismatch")
	}
	launchPath, err := writeGuardianRepairLaunchPacketFileV0(config, phase, packetPath, inspection)
	if err != nil {
		return guardianCodexRepairLaunchV0{}, err
	}
	objectivePath, err := writeGuardianCodexRepairPromptV0(config, phase, packetPath, launchPath, inspection)
	if err != nil {
		return guardianCodexRepairLaunchV0{}, err
	}
	waveRef := "guardian-repair-" + safeGuardianAttemptFilePartV0(config.AttemptRef)
	command := strings.Join([]string{
		shellQuoteV0(config.CurrentBin),
		"codex-launch-director-wave",
		"--agents 1",
		"--wave-ref " + shellQuoteV0(waveRef),
		"--project-dir " + shellQuoteV0(config.ProjectDir),
		"--runtime-dir " + shellQuoteV0(filepath.Join(config.RepairCodexRuntimeDir, waveRef)),
		"--sandbox " + shellQuoteV0(config.RepairCodexSandbox),
		"--approval-policy never",
		"--reasoning-effort " + shellQuoteV0(config.RepairCodexEffort),
		"--objective-file " + shellQuoteV0(objectivePath),
		"--write-set " + shellQuoteV0(strings.Join(config.RepairCodexWriteSet, ",")),
		"--required-tests " + shellQuoteV0(strings.Join(config.RepairCodexRequiredTests, ",")),
		"--worktree-ref " + shellQuoteV0(config.RepairCodexWorktreeRef),
		"--branch-ref " + shellQuoteV0(config.RepairCodexBranchRef),
		"--request-ref " + shellQuoteV0("request-ref-"+waveRef),
		"--run-ref " + shellQuoteV0(firstNonEmptyStringV0(config.RepairCodexRunRef, "run-ref-"+waveRef)),
		"--project-ref " + shellQuoteV0("orquesta-guardian"),
		"--domain-ref " + shellQuoteV0("guardian_repair"),
		"--allow-unmanaged-launch",
		"--unmanaged-launch-reason " + shellQuoteV0("guardian_break_glass_"+sanitizeFilenamePartV0(phase)),
		"--confirm-unmanaged-launch " + shellQuoteV0(waveRef),
	}, " ")
	return guardianCodexRepairLaunchV0{Command: command, LaunchPacketPath: launchPath, ObjectivePath: objectivePath}, nil
}

func writeGuardianRepairLaunchPacketFileV0(
	config guardianConfigV0,
	phase string,
	packetPath string,
	inspection guardianRepairPacketInspectionV0,
) (string, error) {
	result := guardianResultV0{
		Phase:             phase,
		ManifestPath:      config.ManifestPath,
		RepairAttemptRef:  inspection.Packet.RepairAttemptRef,
		FailurePacketHash: inspection.Packet.FailurePacketHash,
	}
	launch := buildGuardianRepairLaunchPacketV0(config, result, packetPath, inspection)
	launchPath := strings.TrimSuffix(packetPath, filepath.Ext(packetPath)) + ".launch.json"
	if err := writeJSONFileV0(launchPath, launch); err != nil {
		return "", err
	}
	return launchPath, nil
}

func writeGuardianCodexRepairPromptV0(
	config guardianConfigV0,
	phase string,
	packetPath string,
	launchPath string,
	inspection guardianRepairPacketInspectionV0,
) (string, error) {
	promptPath := strings.TrimSuffix(packetPath, filepath.Ext(packetPath)) + ".md"
	repairPacketRef := guardianRepairPacketRefV0(config, guardianResultV0{Phase: phase, RepairPacketPath: packetPath})
	launchPacketRef := guardianRepairLaunchRefV0(config, guardianResultV0{Phase: phase, RepairLaunchPath: launchPath})
	content := strings.Join([]string{
		"# Reparacion break-glass de Orquesta",
		"",
		"schema_version: " + guardianRepairPromptSchemaVersionV0,
		"attempt_ref: " + config.AttemptRef,
		"",
		"Trabaja solo con el contrato versionado del packet de lanzamiento.",
		"No borres trabajo de otros agentes. No hagas reset destructivo. Mantener arquitectura hexagonal.",
		"Si falta alcance, permisos o contexto, escribe CONSULTA AL DIRECTOR en agent_ack.json.",
		"",
		"Refs de entrada:",
		"- failure_phase: " + phase,
		"- repair_packet_ref: " + repairPacketRef,
		"- repair_launch_packet_ref: " + launchPacketRef,
		"- repair_attempt_ref: " + inspection.Packet.RepairAttemptRef,
		"- failure_packet_hash: " + inspection.Packet.FailurePacketHash,
		"- repair_packet_sha256: " + inspection.SHA256,
		"- repair_packet_bytes: " + inspection.ByteCount,
		"- repair_packet_summary: " + guardianRepairPacketPromptSummaryV0(config, inspection.Packet.Summary),
		"- repair_packet_command_refs: " + strings.Join(compactStringsV0(inspection.Packet.RequiredCommandRefs), ","),
		"",
		"Objetivo:",
		"- Repara build/tests/healthcheck del candidato de Orquesta.",
		"- Respeta write-set cerrado y pruebas requeridas del packet de lanzamiento.",
		"- Entrega agent_ack.json estructurado; stdout, prompt o exit code no cierran reparacion.",
		"",
	}, "\n")
	if err := writeGuardianTextFileV0(promptPath, content); err != nil {
		return "", err
	}
	return promptPath, nil
}

func firstNonEmptyStringV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
