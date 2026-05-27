package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestGuardianLeaseV0BloqueaIntentoActivoV0(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:             dir,
		StateDir:               filepath.Join(dir, "guardian"),
		CurrentBin:             current,
		PromotionRef:           "promotion-ref-lease-001",
		BuildCommand:           "printf candidate > {candidate_bin}",
		SkipHealth:             true,
		SkipHealthEvidenceRefs: testGuardianSkipHealthEvidenceRefsV0(),
		Promote:                true,
		CommandTimeout:         time.Second,
		LeaseTTL:               time.Hour,
		OccurredAt:             time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC),
	})
	if _, err := acquireGuardianPromotionLeaseV0(config, "check-promote"); err != nil {
		t.Fatalf("lease: %v", err)
	}

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusLeaseBusyV0 ||
		result.Lease == nil ||
		result.Lease.ReasonCode != guardianStatusLeaseBusyV0 ||
		len(result.Commands) != 0 {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
}

func TestGuardianLeaseV0BloqueaRestoreConIntentoActivoV0(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:   dir,
		StateDir:     filepath.Join(dir, "guardian"),
		CurrentBin:   current,
		PromotionRef: "promotion-ref-lease-restore",
		LeaseTTL:     time.Hour,
		OccurredAt:   time.Date(2026, 5, 27, 9, 2, 0, 0, time.UTC),
	})
	mustWriteGuardianTestFileV0(t, config.LastGoodBin, "last-good")
	if _, err := acquireGuardianPromotionLeaseV0(config, "check-promote"); err != nil {
		t.Fatalf("lease: %v", err)
	}

	result := restoreLastGoodCommandV0(config)
	if result.Status != guardianStatusLeaseBusyV0 ||
		result.Lease == nil ||
		result.Lease.ReasonCode != guardianStatusLeaseBusyV0 ||
		result.Restored {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
}

func TestGuardianLeaseV0UsaRelojDeConfigV0(t *testing.T) {
	dir := t.TempDir()
	occurredAt := time.Date(2026, 5, 27, 9, 3, 0, 0, time.UTC)
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:   dir,
		StateDir:     filepath.Join(dir, "guardian"),
		CurrentBin:   filepath.Join(dir, "bin", "orquesta-server-latest"),
		PromotionRef: "promotion-ref-clock",
		LeaseTTL:     15 * time.Minute,
		OccurredAt:   occurredAt,
	})

	lease, err := acquireGuardianPromotionLeaseV0(config, "restore")
	if err != nil {
		t.Fatalf("lease: %v", err)
	}
	if lease.AcquiredAt != occurredAt.Format(time.RFC3339) ||
		lease.DeadlineAt != occurredAt.Add(15*time.Minute).Format(time.RFC3339) {
		t.Fatalf("lease clock=%+v", lease)
	}
}

func TestGuardianLeaseV0VerificaAntesDePromocionarV0(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:   dir,
		StateDir:     filepath.Join(dir, "guardian"),
		CurrentBin:   current,
		PromotionRef: "promotion-ref-lease-lost",
		BuildCommand: "rm -f {state_dir}/leases/guardian-promotion-lease.json; " +
			"printf candidate > {candidate_bin}",
		TestCommands:           []string{"true"},
		SkipHealth:             true,
		SkipHealthEvidenceRefs: testGuardianSkipHealthEvidenceRefsV0(),
		Promote:                true,
		CommandTimeout:         time.Second,
		LeaseTTL:               time.Hour,
		OccurredAt:             time.Date(2026, 5, 27, 9, 1, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusLeaseLostV0 || result.Promoted {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
}
