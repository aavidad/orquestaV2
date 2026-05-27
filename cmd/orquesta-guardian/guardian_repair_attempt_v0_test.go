package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGuardianV0RepairAttemptIdempotenteNoRelanzaMismoPacket(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	repairSeen := filepath.Join(dir, "repair_seen.txt")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "exit 7",
		RepairCommand:  "printf x >> " + shellQuoteV0(repairSeen),
		AttemptRef:     "guardian-attempt-ref-repair-idem",
		SkipHealth:     true,
		Promote:        true,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
	})

	first := runGuardianCheckPromoteV0(context.Background(), config)
	if !first.RepairStarted || first.RepairAttemptRef == "" || first.FailurePacketHash == "" {
		t.Fatalf("first=%+v", first)
	}
	secondConfig := config
	secondConfig.ManifestPath = filepath.Join(config.StateDir, "manifests", "guardian-second.json")
	second := runGuardianCheckPromoteV0(context.Background(), secondConfig)
	if second.RepairStarted ||
		!second.RepairBlocked ||
		second.RepairBlockReason != "guardian_repair_attempt_duplicate" ||
		second.RepairAttemptRef != first.RepairAttemptRef ||
		second.FailurePacketHash != first.FailurePacketHash {
		t.Fatalf("second=%+v first=%+v", second, first)
	}
	if got := mustReadGuardianTestFileV0(t, repairSeen); got != "x" {
		t.Fatalf("repair_seen=%q", got)
	}
}

func TestGuardianV0RepairAttemptBudgetPermiteHashNuevoHastaLimite(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	repairSeen := filepath.Join(dir, "repair_seen.txt")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:        dir,
		StateDir:          filepath.Join(dir, "guardian"),
		CurrentBin:        current,
		BuildCommand:      "exit 7",
		RepairCommand:     "printf x >> " + shellQuoteV0(repairSeen),
		RepairMaxAttempts: 2,
		AttemptRef:        "guardian-attempt-ref-repair-budget",
		SkipHealth:        true,
		Promote:           true,
		CommandTimeout:    time.Second,
		OccurredAt:        time.Date(2026, 5, 27, 12, 1, 0, 0, time.UTC),
	})

	first := runGuardianCheckPromoteV0(context.Background(), config)
	secondConfig := config
	secondConfig.BuildCommand = "exit 8"
	secondConfig.ManifestPath = filepath.Join(config.StateDir, "manifests", "guardian-second.json")
	second := runGuardianCheckPromoteV0(context.Background(), secondConfig)
	thirdConfig := config
	thirdConfig.BuildCommand = "exit 9"
	thirdConfig.ManifestPath = filepath.Join(config.StateDir, "manifests", "guardian-third.json")
	third := runGuardianCheckPromoteV0(context.Background(), thirdConfig)

	if !first.RepairStarted || !second.RepairStarted || !strings.HasPrefix(first.FailurePacketHash, "sha256:") {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if first.FailurePacketHash == second.FailurePacketHash || first.RepairAttemptRef == second.RepairAttemptRef {
		t.Fatalf("hash/ref no cambio first=%+v second=%+v", first, second)
	}
	if third.RepairStarted ||
		!third.RepairBlocked ||
		third.RepairBlockReason != "guardian_repair_attempt_budget_exhausted" {
		t.Fatalf("third=%+v", third)
	}
	if got := mustReadGuardianTestFileV0(t, repairSeen); got != "xx" {
		t.Fatalf("repair_seen=%q", got)
	}
}
