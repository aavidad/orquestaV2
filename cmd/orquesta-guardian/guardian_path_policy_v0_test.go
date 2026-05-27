package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGuardianPathPolicyV0BloqueaBinarioFueraDeRaicesDeclaradas(t *testing.T) {
	projectDir := t.TempDir()
	outsideDir := t.TempDir()

	_, err := normalizeGuardianConfigV0(guardianConfigV0{
		ProjectDir:     projectDir,
		StateDir:       filepath.Join(projectDir, "guardian"),
		CurrentBin:     filepath.Join(outsideDir, "orquesta-server"),
		CommandTimeout: time.Second,
		OccurredAt:     time.Now().UTC(),
	})

	if err == nil || !strings.Contains(err.Error(), guardianConfigPathOutsideAllowedRootV0+":current_bin") {
		t.Fatalf("err=%v", err)
	}
}

func TestGuardianPathPolicyV0AceptaArtifactRootExternoDeclarado(t *testing.T) {
	projectDir := t.TempDir()
	artifactRoot := t.TempDir()

	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     projectDir,
		StateDir:       filepath.Join(projectDir, "guardian"),
		ArtifactRoot:   artifactRoot,
		CurrentBin:     filepath.Join(artifactRoot, "current", "orquesta-server"),
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 27, 14, 0, 0, 0, time.UTC),
	})

	if config.PathPolicy.SchemaVersion != guardianPathPolicySchemaVersionV0 ||
		!guardianPathPolicyHasV0(config.PathPolicy, "current_bin", guardianPathClassificationBinaryV0) ||
		!guardianPathPolicyHasV0(config.PathPolicy, "artifact_root", guardianPathClassificationControlRootV0) {
		t.Fatalf("path_policy=%+v", config.PathPolicy)
	}
}

func TestGuardianPathPolicyV0PublicaRefsSinPathsLocales(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "printf candidate > {candidate_bin}",
		SkipHealth:     true,
		Promote:        false,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 27, 14, 1, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(t.Context(), config)
	encoded, err := json.Marshal(publicGuardianResultV0(config, result))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(encoded)
	if strings.Contains(body, dir) || !strings.Contains(body, guardianPathClassificationProductRootV0) ||
		!strings.Contains(body, guardianPathClassificationBinaryV0) {
		t.Fatalf("public path policy invalida: %s", body)
	}
}

func TestGuardianPathPolicyV0BloqueaStateDirSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink requiere privilegios en windows")
	}
	projectDir := t.TempDir()
	targetDir := filepath.Join(projectDir, "real-state")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	linkPath := filepath.Join(projectDir, "state-link")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Skipf("symlink no disponible: %v", err)
	}

	_, err := normalizeGuardianConfigV0(guardianConfigV0{
		ProjectDir:     projectDir,
		StateDir:       linkPath,
		CurrentBin:     filepath.Join(projectDir, "bin", "orquesta-server"),
		CommandTimeout: time.Second,
		OccurredAt:     time.Now().UTC(),
	})

	if err == nil || !strings.Contains(err.Error(), guardianConfigPathSymlinkBlockedV0+":state_dir") {
		t.Fatalf("err=%v", err)
	}
}

func guardianPathPolicyHasV0(policy guardianPathRootPolicyV0, field string, classification string) bool {
	for _, item := range policy.Items {
		if item.Field == field && item.Classification == classification && item.Ref != "" {
			return true
		}
	}
	return false
}
