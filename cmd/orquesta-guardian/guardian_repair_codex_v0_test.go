package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGuardianV0BuildFallidoPuedeLanzarCodexReparadorOptIn(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	argsPath := filepath.Join(dir, "codex_args.txt")
	script := "#!/bin/sh\nprintf \"%s\\n\" \"$@\" > " + shellQuoteV0(argsPath) + "\n"
	mustWriteGuardianTestFileV0(t, current, script)
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:   dir,
		StateDir:     filepath.Join(dir, "guardian"),
		CurrentBin:   current,
		BuildCommand: "exit 7",
		TestCommands: []string{"go test -count=1 ./cmd/orquesta-guardian"},
		RepairCodex:  true,
		RepairCodexWriteSet: []string{
			"cmd/orquesta-guardian",
			"docs/runbooks",
		},
		SkipHealth:     true,
		Promote:        true,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 6, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 || !result.RepairStarted {
		t.Fatalf("result=%+v", result)
	}
	guardianAssertRepairCodexCommandV0(t, argsPath)
	guardianAssertRepairCodexPacketsV0(t, result.RepairPacketPath, dir)
}

func TestGuardianV0RepairCodexSandboxAmplioRequiereOptIn(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	_, err := normalizeGuardianConfigV0(guardianConfigV0{
		ProjectDir:          dir,
		StateDir:            filepath.Join(dir, "guardian"),
		CurrentBin:          current,
		RepairCodex:         true,
		RepairCodexSandbox:  "danger-full-access",
		RepairCodexWriteSet: []string{"cmd/orquesta-guardian"},
		RepairCodexRequiredTests: []string{
			"go test -count=1 ./cmd/orquesta-guardian",
		},
		OccurredAt: time.Date(2026, 5, 24, 12, 7, 0, 0, time.UTC),
	})
	if err == nil || !strings.Contains(err.Error(), "repair-codex-broad-sandbox-requires-opt-in") {
		t.Fatalf("err=%v", err)
	}
}

func TestGuardianV0RepairCodexBloqueaPacketConRedactionInvalida(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:   dir,
		StateDir:     filepath.Join(dir, "guardian"),
		CurrentBin:   current,
		BuildCommand: "exit 7",
		RepairCodex:  true,
		RepairCodexWriteSet: []string{
			"cmd/orquesta-guardian",
		},
		RepairCodexRequiredTests: []string{
			"go test -count=1 ./cmd/orquesta-guardian",
		},
		SkipHealth:     true,
		Promote:        true,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 8, 0, 0, time.UTC),
	})
	packet := repairGuardianPacketV0(config, guardianResultV0{
		Phase:   "build",
		Message: "build failed",
	})
	packet, receipt := finalizeGuardianRepairPacketV0(config, packet)
	packet.RedactionLevel = "raw"
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	mustWriteGuardianTestFileV0(t, receipt.Path, string(body))

	_, err = guardianCodexRepairCommandV0(config, "build", receipt.Path, receipt)
	if err == nil || !strings.Contains(err.Error(), "guardian_repair_packet_invalid:redaction_level_invalid") {
		t.Fatalf("err=%v", err)
	}
}

func guardianAssertRepairCodexCommandV0(t *testing.T, argsPath string) {
	t.Helper()
	args := mustReadGuardianTestFileV0(t, argsPath)
	for _, want := range []string{
		"codex-launch-director-wave",
		"--agents",
		"1",
		"--objective-file",
		"--write-set",
		"--required-tests",
		"--allow-unmanaged-launch",
		"--confirm-unmanaged-launch",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("args no contiene %q: %s", want, args)
		}
	}
}

func guardianAssertRepairCodexPacketsV0(t *testing.T, repairPacketPath string, dir string) {
	t.Helper()
	promptPath := strings.TrimSuffix(repairPacketPath, filepath.Ext(repairPacketPath)) + ".md"
	if _, err := os.Stat(promptPath); err != nil {
		t.Fatalf("repair prompt: %v", err)
	}
	launchPacket := strings.TrimSuffix(repairPacketPath, filepath.Ext(repairPacketPath)) + ".launch.json"
	encodedLaunch := mustReadGuardianTestFileV0(t, launchPacket)
	for _, want := range []string{
		guardianRepairLaunchPacketSchemaVersionV0,
		"cmd/orquesta-guardian",
		"go test -count=1 ./cmd/orquesta-guardian",
		"codex_agent_ack.v0",
		"break_glass",
		"repair-attempt-ref-",
		"sha256:",
	} {
		if !strings.Contains(encodedLaunch, want) {
			t.Fatalf("launch packet no contiene %q: %s", want, encodedLaunch)
		}
	}
	promptText := mustReadGuardianTestFileV0(t, promptPath)
	if strings.Contains(promptText, encodedLaunch) || strings.Contains(promptText, dir) {
		t.Fatalf("prompt contiene payload o path local: %s", promptText)
	}
	for _, want := range []string{
		"repair_attempt_ref:",
		"failure_packet_hash: sha256:",
		"repair_packet_sha256:",
		"repair_packet_summary:",
		"repair_packet_command_refs:",
	} {
		if !strings.Contains(promptText, want) {
			t.Fatalf("prompt no contiene %q: %s", want, promptText)
		}
	}
}
