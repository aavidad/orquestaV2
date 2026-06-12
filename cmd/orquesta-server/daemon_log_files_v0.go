package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const daemonLogSummarySchemaVersionV0 = "orquesta_server_daemon_log_summary.v0"

type daemonLogSummaryV0 struct {
	SchemaVersion     string `json:"schema_version"`
	Status            string `json:"status"`
	LogKind           string `json:"log_kind"`
	Owner             string `json:"owner"`
	PolicyRef         string `json:"policy_ref"`
	Mode              string `json:"mode"`
	Access            string `json:"access"`
	MaxBytes          int64  `json:"max_bytes"`
	MaxRotatedFiles   int    `json:"max_rotated_files"`
	RetentionDays     int    `json:"retention_days"`
	TerminalEvidence  bool   `json:"terminal_evidence"`
	RedactionRequired bool   `json:"redaction_required"`
}

func openDaemonOutputFilesV0(config orquestaserver.ConfigV0) (*os.File, *os.File, error) {
	policy := orquestaserver.NormalizeDaemonLogPolicyV0(config.DaemonLogPolicy)
	if policy.LocalRawEnabled {
		stdout, err := openDaemonRawLogV0(config.StateDir, "stdout.log", policy)
		if err != nil {
			return nil, nil, err
		}
		stderr, err := openDaemonRawLogV0(config.StateDir, "stderr.log", policy)
		if err != nil {
			_ = stdout.Close()
			return nil, nil, err
		}
		return stdout, stderr, nil
	}
	if err := writeDaemonLogSummaryV0(config.StateDir, "stdout.log", policy); err != nil {
		return nil, nil, err
	}
	if err := writeDaemonLogSummaryV0(config.StateDir, "stderr.log", policy); err != nil {
		return nil, nil, err
	}
	return openDaemonDiscardPairV0()
}

func writeDaemonLogSummaryV0(dir string, name string, policy orquestaserver.DaemonLogPolicyV0) error {
	path, err := daemonLogPathV0(dir, name)
	if err != nil {
		return err
	}
	if err := rotateDaemonLogIfNeededV0(path, policy); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("daemon_log_unavailable")
	}
	defer file.Close()
	summary := daemonLogSummaryV0{
		SchemaVersion:     daemonLogSummarySchemaVersionV0,
		Status:            "redacted_summary",
		LogKind:           strings.TrimSuffix(name, ".log"),
		Owner:             policy.Owner,
		PolicyRef:         policy.PolicyRef,
		Mode:              policy.Mode,
		Access:            policy.Access,
		MaxBytes:          policy.MaxBytes,
		MaxRotatedFiles:   policy.MaxRotatedFiles,
		RetentionDays:     policy.RetentionDays,
		TerminalEvidence:  false,
		RedactionRequired: true,
	}
	return json.NewEncoder(file).Encode(summary)
}

func openDaemonRawLogV0(dir string, name string, policy orquestaserver.DaemonLogPolicyV0) (*os.File, error) {
	path, err := daemonLogPathV0(dir, name)
	if err != nil {
		return nil, err
	}
	if err := rotateDaemonLogIfNeededV0(path, policy); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}

func daemonLogPathV0(dir string, name string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("daemon_log_dir_unavailable")
	}
	if filepath.Base(name) != name || filepath.Ext(name) != ".log" {
		return "", fmt.Errorf("daemon_log_name_invalid")
	}
	return filepath.Join(dir, name), nil
}

func rotateDaemonLogIfNeededV0(path string, policy orquestaserver.DaemonLogPolicyV0) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("daemon_log_unavailable")
	}
	if stat.Size() < policy.MaxBytes {
		return nil
	}
	rotated := fmt.Sprintf("%s.%s.%d", path, time.Now().UTC().Format("20060102T150405Z"), os.Getpid())
	if err := os.Rename(path, rotated); err != nil {
		return fmt.Errorf("daemon_log_rotate_failed")
	}
	return enforceDaemonLogRetentionV0(path, policy)
}

func enforceDaemonLogRetentionV0(path string, policy orquestaserver.DaemonLogPolicyV0) error {
	matches, err := filepath.Glob(path + ".*")
	if err != nil {
		return fmt.Errorf("daemon_log_retention_failed")
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	cutoff := time.Now().AddDate(0, 0, -policy.RetentionDays)
	for index, match := range matches {
		expired, err := daemonRotatedLogExpiredV0(match, cutoff)
		if err != nil {
			return err
		}
		if index < policy.MaxRotatedFiles && !expired {
			continue
		}
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("daemon_log_retention_failed")
		}
	}
	return nil
}

func daemonRotatedLogExpiredV0(path string, cutoff time.Time) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("daemon_log_retention_failed")
	}
	return stat.ModTime().Before(cutoff), nil
}

func openDaemonDiscardPairV0() (*os.File, *os.File, error) {
	stdout, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("daemon_log_unavailable")
	}
	stderr, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		_ = stdout.Close()
		return nil, nil, fmt.Errorf("daemon_log_unavailable")
	}
	return stdout, stderr, nil
}
