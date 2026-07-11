package orquestaruntimerequiredtest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
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
	return goalRequiredTestGoCommandV0(allowed) != ""
}

func goalRequiredTestGoCommandV0(allowed map[string]string) string {
	if _, ok := allowed["go"]; ok {
		return "go"
	}
	names := make([]string, 0, len(allowed))
	for name, path := range allowed {
		if filepath.Base(path) == "go" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return names[0]
}

func goalRequiredTestProjectHasGoModuleV0(project string) (bool, error) {
	info, err := os.Lstat(filepath.Join(project, "go.mod"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("goal_required_test_go_module_invalid")
	}
	return true, nil
}

func ensureGoalRequiredTestGoSnapshotPreflightV0(commands []string, goCommand string) []string {
	if goCommand == "" {
		return commands
	}
	required := []string{goCommand, "mod", "download", "all"}
	for _, command := range commands {
		tokens, err := splitCommandV0(command)
		if err == nil && slices.Equal(tokens, required) {
			return commands
		}
	}
	return append([]string{strings.Join(required, " ")}, commands...)
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) materializeDependencySnapshotV0(runDir string) (string, error) {
	destination, err := os.MkdirTemp(runDir, "go-mod-cache-")
	if err != nil {
		return "", err
	}
	source := adapter.config.DependencySnapshotPath
	if source == "" {
		return destination, nil
	}
	_, beforeHash, err := normalizeGoalRequiredTestDependencySnapshotV0(source)
	if err != nil || beforeHash != adapter.config.DependencySnapshotSHA256 {
		_ = os.RemoveAll(destination)
		return "", fmt.Errorf("goal_required_test_dependency_snapshot_changed")
	}
	if err := copyGoalRequiredTestDependencySnapshotV0(source, destination); err != nil {
		_ = os.RemoveAll(destination)
		return "", err
	}
	_, afterHash, err := normalizeGoalRequiredTestDependencySnapshotV0(source)
	if err != nil || afterHash != beforeHash {
		_ = os.RemoveAll(destination)
		return "", fmt.Errorf("goal_required_test_dependency_snapshot_changed")
	}
	_, copiedHash, err := normalizeGoalRequiredTestDependencySnapshotV0(destination)
	if err != nil || copiedHash != beforeHash {
		_ = os.RemoveAll(destination)
		return "", fmt.Errorf("goal_required_test_dependency_snapshot_copy_invalid")
	}
	if err := makeGoalRequiredTestDependencySnapshotWritableV0(destination); err != nil {
		_ = os.RemoveAll(destination)
		return "", err
	}
	return destination, nil
}

func copyGoalRequiredTestDependencySnapshotV0(source string, destination string) error {
	type directoryModeV0 struct {
		path string
		mode os.FileMode
	}
	directories := make([]directoryModeV0, 0)
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			directories = append(directories, directoryModeV0{path: destination, mode: info.Mode().Perm()})
			return nil
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			if err := os.Mkdir(target, 0o700); err != nil {
				return err
			}
			directories = append(directories, directoryModeV0{path: target, mode: info.Mode().Perm()})
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("goal_required_test_dependency_snapshot_entry_invalid")
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeOutputErr := output.Close()
		closeInputErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeOutputErr != nil {
			return closeOutputErr
		}
		if closeInputErr != nil {
			return closeInputErr
		}
		return os.Chmod(target, info.Mode().Perm())
	})
	if err != nil {
		return err
	}
	for index := len(directories) - 1; index >= 0; index-- {
		if err := os.Chmod(directories[index].path, directories[index].mode); err != nil {
			return err
		}
	}
	return nil
}

func makeGoalRequiredTestDependencySnapshotWritableV0(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("goal_required_test_dependency_snapshot_entry_invalid")
		}
		return os.Chmod(path, 0o600)
	})
}

func removeGoalRequiredTestExecutionDirV0(root string) error {
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.RemoveAll(root)
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
