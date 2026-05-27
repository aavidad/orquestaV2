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

func TestGuardianRetentionV0IndexaIntentoYEscrituraDurable(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:   dir,
		StateDir:     filepath.Join(dir, "guardian"),
		CurrentBin:   current,
		BuildCommand: "printf candidate > {candidate_bin}",
		SkipHealth:   true,
		Promote:      false,
		PromotionRef: "promotion-ref-stable-001",
		OccurredAt:   time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusPromotedV0 || result.AttemptRef != config.PromotionRef {
		t.Fatalf("result=%+v attempt=%s", result, config.AttemptRef)
	}
	if !strings.Contains(filepath.Base(config.ManifestPath), safeGuardianAttemptFilePartV0(config.PromotionRef)) {
		t.Fatalf("manifest path no indexado por ref: %s", config.ManifestPath)
	}
	body := mustReadGuardianTestFileV0(t, config.ManifestPath)
	if strings.Contains(body, ".tmp-") || !strings.Contains(body, `"schema_version"`) {
		t.Fatalf("manifest durable invalido: %s", body)
	}
	var manifest guardianResultV0
	if err := json.Unmarshal([]byte(body), &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.RetentionStatus.PolicyRef != guardianRetentionPolicyRefV0 ||
		manifest.RetentionStatus.Status == "" {
		t.Fatalf("retention_status=%+v", manifest.RetentionStatus)
	}
}

func TestGuardianRetentionV0DetectaConflictoDeReplay(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:   dir,
		StateDir:     filepath.Join(dir, "guardian"),
		CurrentBin:   current,
		BuildCommand: "true",
		SkipHealth:   true,
		Promote:      false,
		AttemptRef:   "guardian-attempt-ref-conflict",
		OccurredAt:   time.Date(2026, 5, 27, 12, 1, 0, 0, time.UTC),
	})
	mustWriteGuardianTestFileV0(t, config.ManifestPath, `{"schema_version":"old"}`+"\n")

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusManifestIncompleteV0 ||
		result.RetentionStatus.ReasonCode != "guardian_manifest_payload_conflict" {
		t.Fatalf("result=%+v", result)
	}
}

func TestGuardianRetentionV0LimpiaPorCategoriaSinBorrarEvidenciaActual(t *testing.T) {
	dir := t.TempDir()
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir: dir,
		StateDir:   filepath.Join(dir, "guardian"),
		CurrentBin: filepath.Join(dir, "bin", "orquesta-server-latest"),
		AttemptRef: "guardian-attempt-ref-retention",
		OccurredAt: time.Date(2026, 5, 27, 12, 2, 0, 0, time.UTC),
		RetentionPolicy: guardianRetentionPolicyV0{MaxArtifacts: map[string]int{
			"redacted_logs": 1,
		}},
	})
	currentLog := filepath.Join(config.StateDir, "logs", "current.log")
	oldLog := filepath.Join(config.StateDir, "logs", "old.log")
	mustWriteGuardianTestFileV0(t, oldLog, "old")
	mustWriteGuardianTestFileV0(t, currentLog, "current")
	result := baseGuardianResultV0(config)
	result.Commands = []guardianCommandResultV0{{Phase: "build", OutputPath: currentLog}}

	status := applyGuardianRetentionV0(config, result)
	if status.RemovedCounts["redacted_logs"] != 1 {
		t.Fatalf("status=%+v", status)
	}
	if _, err := os.Stat(currentLog); err != nil {
		t.Fatalf("current log borrado: %v", err)
	}
	if _, err := os.Stat(oldLog); !os.IsNotExist(err) {
		t.Fatalf("old log sigue presente: %v", err)
	}
}
