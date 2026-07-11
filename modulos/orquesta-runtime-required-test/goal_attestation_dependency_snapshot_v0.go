package orquestaruntimerequiredtest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) observeDependencySnapshotV0() (string, bool, error) {
	if adapter.config.DependencySnapshotPath == "" {
		return "", true, nil
	}
	path, hash, err := normalizeGoalRequiredTestDependencySnapshotV0(adapter.config.DependencySnapshotPath)
	valid := err == nil && path == adapter.config.DependencySnapshotPath && hash == adapter.config.DependencySnapshotSHA256
	ref, evidenceErr := adapter.writeDependencySnapshotEvidenceV0(hash, valid)
	if evidenceErr != nil {
		return "", false, evidenceErr
	}
	return ref, valid, nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) writeDependencySnapshotEvidenceV0(observedHash string, valid bool) (string, error) {
	refHash := localGoalAttestationHashV0(strings.Join([]string{adapter.config.DependencySnapshotSHA256, observedHash, fmt.Sprintf("%t", valid)}, "\x00"))
	ref := "goal-required-test-module-cache-snapshot-evidence-ref-" + refHash
	dir := filepath.Join(adapter.config.RuntimeRoot, "evidence", "snapshots")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	payload := struct {
		SchemaVersion  string `json:"schema_version"`
		EvidenceRef    string `json:"evidence_ref"`
		ExpectedSHA256 string `json:"expected_sha256"`
		ObservedSHA256 string `json:"observed_sha256"`
		Valid          bool   `json:"valid"`
	}{"orquesta.runtime_required_test.module_cache_snapshot_evidence.v0", ref, adapter.config.DependencySnapshotSHA256, observedHash, valid}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, refHash+".json"), append(encoded, '\n'), 0o600); err != nil {
		return "", err
	}
	return ref, nil
}

func goalRequiredTestGoAllowedV0(allowed map[string]string) bool {
	for name, path := range allowed {
		if strings.TrimSpace(name) == "go" || filepath.Base(path) == "go" {
			return true
		}
	}
	return false
}

func normalizeGoalRequiredTestDependencySnapshotV0(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", nil
	}
	if !filepath.IsAbs(value) || filepath.Clean(value) != value {
		return "", "", fmt.Errorf("goal_required_test_dependency_snapshot_invalid")
	}
	canonical, err := filepath.EvalSymlinks(value)
	if err != nil || canonical != value {
		return "", "", fmt.Errorf("goal_required_test_dependency_snapshot_not_canonical")
	}
	info, err := os.Lstat(canonical)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", "", fmt.Errorf("goal_required_test_dependency_snapshot_invalid")
	}
	hash := sha256.New()
	err = filepath.WalkDir(canonical, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o222 != 0 {
			return fmt.Errorf("goal_required_test_dependency_snapshot_not_read_only")
		}
		rel, err := filepath.Rel(canonical, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		_, _ = io.WriteString(hash, info.Mode().String()+"\x00"+rel+"\x00")
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(hash, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}
	return canonical, hex.EncodeToString(hash.Sum(nil)), nil
}
