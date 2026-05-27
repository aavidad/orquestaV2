package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func writeGuardianManifestV0(config guardianConfigV0, result guardianResultV0) guardianResultV0 {
	if result.ManifestPath == "" {
		result.ManifestPath = config.ManifestPath
	}
	result.RetentionStatus = applyGuardianRetentionV0(config, result)
	if err := writeJSONFileV0(config.ManifestPath, result); err != nil {
		result.Status = guardianStatusManifestIncompleteV0
		result.RetentionStatus.Status = "guardian_manifest_incomplete"
		result.RetentionStatus.ReasonCode = durableGuardianReasonCodeV0(err)
	}
	return result
}

func writeJSONFileV0(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeGuardianDurableFileV0(path, data, 0o600)
}

func writeGuardianTextFileV0(path string, content string) error {
	return writeGuardianDurableFileV0(path, []byte(content), 0o600)
}

func writeGuardianDurableFileV0(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if existing, err := os.ReadFile(path); err == nil {
		if bytes.Equal(existing, data) {
			return nil
		}
		return guardianDurableWriteErrorV0{Code: "guardian_manifest_payload_conflict", Err: nil}
	} else if !errors.Is(err, os.ErrNotExist) {
		return guardianDurableWriteErrorV0{Code: "guardian_manifest_existing_read_failed", Err: err}
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return guardianDurableWriteErrorV0{Code: "guardian_manifest_tmp_failed", Err: err}
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return guardianDurableWriteErrorV0{Code: "guardian_manifest_chmod_failed", Err: err}
	}
	if _, err := tmp.Write(data); err != nil {
		return guardianDurableWriteErrorV0{Code: "guardian_manifest_write_failed", Err: err}
	}
	if err := tmp.Sync(); err != nil {
		return guardianDurableWriteErrorV0{Code: "guardian_manifest_fsync_failed", Err: err}
	}
	if err := tmp.Close(); err != nil {
		return guardianDurableWriteErrorV0{Code: "guardian_manifest_close_failed", Err: err}
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return guardianDurableWriteErrorV0{Code: "guardian_manifest_rename_failed", Err: err}
	}
	cleanup = false
	if err := fsyncGuardianDirV0(filepath.Dir(path)); err != nil {
		return guardianDurableWriteErrorV0{Code: "guardian_manifest_dir_fsync_failed", Err: err}
	}
	return nil
}

func copyFileAtomicV0(src string, dst string, mode os.FileMode) error {
	_, err := copyGuardianArtifactAtomicV0(
		guardianConfigV0{ProjectDir: filepath.Dir(dst), ArtifactMaxBytes: defaultGuardianArtifactMaxBytesV0},
		src,
		dst,
		mode,
	)
	return err
}

func commandExitCodeV0(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func commandErrorStringV0(err error, ctxErr error) string {
	if ctxErr != nil {
		return ctxErr.Error()
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func expandGuardianCommandV0(command string, config guardianConfigV0) string {
	replacer := strings.NewReplacer(
		"{project_dir}", shellQuoteV0(config.ProjectDir),
		"{state_dir}", shellQuoteV0(config.StateDir),
		"{current_bin}", shellQuoteV0(config.CurrentBin),
		"{candidate_bin}", shellQuoteV0(config.CandidateBin),
		"{last_good_bin}", shellQuoteV0(config.LastGoodBin),
		"{repair_packet}", shellQuoteV0(config.RepairPacketPath),
	)
	return replacer.Replace(command)
}

func envOrDefaultV0(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBoolOrDefaultV0(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envDurationOrDefaultV0(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envIntOrDefaultV0(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt64OrDefaultV0(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitEnvCommandsV0(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return compactStringsV0(strings.Split(value, "\n"))
}

func splitEnvListV0(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	value = strings.ReplaceAll(value, "\n", ",")
	return compactStringsV0(strings.Split(value, ","))
}

func expandGuardianListValuesV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, splitEnvListV0(value)...)
	}
	return out
}

func absPathFromBaseV0(base string, path string) string {
	path = strings.TrimSpace(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(base, path)
}

func sanitizeFilenamePartV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "command"
	}
	return out
}

func shellQuoteV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func compactStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
