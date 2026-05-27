package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGuardianArtifactsV0ManifestRegistraHashesYTamanos(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:             dir,
		StateDir:               filepath.Join(dir, "guardian"),
		CurrentBin:             current,
		BuildCommand:           "printf candidate > {candidate_bin}",
		SkipHealth:             true,
		SkipHealthEvidenceRefs: testGuardianSkipHealthEvidenceRefsV0(),
		Promote:                true,
		CommandTimeout:         time.Second,
		OccurredAt:             time.Date(2026, 5, 27, 8, 0, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(t.Context(), config)

	if result.Status != guardianStatusPromotedBreakglassV0 || result.ArtifactManifest == nil {
		t.Fatalf("result=%+v", result)
	}
	manifest := result.ArtifactManifest
	if manifest.SchemaVersion != guardianArtifactManifestSchemaVersionV0 ||
		manifest.Candidate.SizeBytes != int64(len("candidate")) ||
		manifest.Current.SHA256 != guardianHashForTestV0("candidate") ||
		manifest.PreviousCurrent.SHA256 != guardianHashForTestV0("old") ||
		manifest.LastGood.SHA256 != guardianHashForTestV0("old") {
		t.Fatalf("manifest=%+v", manifest)
	}
	body := mustReadGuardianTestFileV0(t, config.ManifestPath)
	if !strings.Contains(body, guardianArtifactManifestSchemaVersionV0) ||
		!strings.Contains(body, guardianHashForTestV0("candidate")) {
		t.Fatalf("manifest persisted=%s", body)
	}
}

func TestGuardianArtifactsV0BloqueaArtefactoMayorAlPresupuesto(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:             dir,
		StateDir:               filepath.Join(dir, "guardian"),
		CurrentBin:             current,
		BuildCommand:           "printf candidate > {candidate_bin}",
		SkipHealth:             true,
		SkipHealthEvidenceRefs: testGuardianSkipHealthEvidenceRefsV0(),
		Promote:                true,
		CommandTimeout:         time.Second,
		ArtifactMaxBytes:       4,
		OccurredAt:             time.Date(2026, 5, 27, 8, 1, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(t.Context(), config)

	if result.Status != guardianStatusCandidateFailedV0 ||
		result.ArtifactManifest == nil ||
		result.ArtifactManifest.ReasonCode != "guardian_artifact_size_exceeded" {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
}

func TestGuardianArtifactsV0BloqueaSymlinkDeCandidato(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	target := filepath.Join(dir, "outside-target")
	candidate := filepath.Join(dir, "guardian", "candidate", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	mustWriteGuardianTestFileV0(t, target, "candidate")
	if err := os.MkdirAll(filepath.Dir(candidate), 0o700); err != nil {
		t.Fatalf("mkdir candidate: %v", err)
	}
	if err := os.Symlink(target, candidate); err != nil {
		t.Skipf("symlink no disponible: %v", err)
	}
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:             dir,
		StateDir:               filepath.Join(dir, "guardian"),
		CurrentBin:             current,
		CandidateBin:           candidate,
		BuildCommand:           "true",
		SkipHealth:             true,
		SkipHealthEvidenceRefs: testGuardianSkipHealthEvidenceRefsV0(),
		Promote:                true,
		CommandTimeout:         time.Second,
		OccurredAt:             time.Date(2026, 5, 27, 8, 2, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(t.Context(), config)

	if result.Status != guardianStatusCandidateFailedV0 ||
		result.ArtifactManifest == nil ||
		result.ArtifactManifest.ReasonCode != "guardian_artifact_symlink_path_blocked" {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
}

func TestGuardianArtifactsV0BloqueaRutaFueraDeRaizDeclarada(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "artifacts")
	outside := filepath.Join(dir, "outside")
	current := filepath.Join(root, "current")
	mustWriteGuardianTestFileV0(t, current, "old")
	mustWriteGuardianTestFileV0(t, filepath.Join(outside, "candidate"), "candidate")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:             dir,
		StateDir:               filepath.Join(dir, "guardian"),
		CurrentBin:             current,
		CandidateBin:           filepath.Join(outside, "candidate"),
		ArtifactRoot:           root,
		BuildCommand:           "true",
		SkipHealth:             true,
		SkipHealthEvidenceRefs: testGuardianSkipHealthEvidenceRefsV0(),
		Promote:                true,
		CommandTimeout:         time.Second,
		OccurredAt:             time.Date(2026, 5, 27, 8, 4, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(t.Context(), config)

	if result.Status != guardianStatusCandidateFailedV0 ||
		result.ArtifactManifest == nil ||
		result.ArtifactManifest.ReasonCode != "guardian_artifact_outside_root" {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
}

func TestGuardianArtifactsV0PublicaReasonCodeSinPaths(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:             dir,
		StateDir:               filepath.Join(dir, "guardian"),
		CurrentBin:             current,
		BuildCommand:           "printf candidate > {candidate_bin}",
		SkipHealth:             true,
		SkipHealthEvidenceRefs: testGuardianSkipHealthEvidenceRefsV0(),
		Promote:                true,
		CommandTimeout:         time.Second,
		ArtifactMaxBytes:       4,
		OccurredAt:             time.Date(2026, 5, 27, 8, 3, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(t.Context(), config)
	encoded, err := json.Marshal(publicGuardianResultV0(config, result))
	if err != nil {
		t.Fatalf("marshal public: %v", err)
	}

	if strings.Contains(string(encoded), dir) ||
		!strings.Contains(string(encoded), "guardian_artifact_size_exceeded") {
		t.Fatalf("public=%s", encoded)
	}
}

func guardianHashForTestV0(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
