package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const guardianRetentionPolicyRefV0 = "guardian-retention-policy-ref-v0"

type guardianDurableWriteErrorV0 struct {
	Code string
	Err  error
}

func (err guardianDurableWriteErrorV0) Error() string {
	if err.Err == nil {
		return err.Code
	}
	return err.Code + ": " + err.Err.Error()
}

func (err guardianDurableWriteErrorV0) Unwrap() error { return err.Err }

type guardianRetentionPolicyV0 struct {
	MaxArtifacts map[string]int `json:"max_artifacts,omitempty"`
}

type guardianRetentionStatusV0 struct {
	PolicyRef      string         `json:"policy_ref,omitempty"`
	Status         string         `json:"status,omitempty"`
	ReasonCode     string         `json:"reason_code,omitempty"`
	CategoryCounts map[string]int `json:"category_counts,omitempty"`
	RemovedCounts  map[string]int `json:"removed_counts,omitempty"`
	IncompleteTmp  int            `json:"incomplete_tmp,omitempty"`
}

func guardianAttemptRefV0(config guardianConfigV0) string {
	for _, ref := range []string{config.PromotionRef, config.ShutdownRef, config.RunRef} {
		if strings.TrimSpace(ref) != "" {
			return strings.TrimSpace(ref)
		}
	}
	seed := config.OccurredAt.Format("20060102T150405.000000000Z")
	return "guardian-attempt-ref-" + guardianShortHashV0(seed)
}

func safeGuardianAttemptFilePartV0(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = "guardian-attempt-ref-empty"
	}
	safe := sanitizeFilenamePartV0(ref)
	return safe + "-" + guardianShortHashV0(ref)
}

func durableGuardianReasonCodeV0(err error) string {
	var writeErr guardianDurableWriteErrorV0
	if errors.As(err, &writeErr) && writeErr.Code != "" {
		return writeErr.Code
	}
	return "guardian_manifest_write_failed"
}

func applyGuardianRetentionV0(
	config guardianConfigV0,
	result guardianResultV0,
) guardianRetentionStatusV0 {
	policy := guardianRetentionPolicyForConfigV0(config)
	status := guardianRetentionStatusV0{
		PolicyRef:      guardianRetentionPolicyRefV0,
		Status:         "applied",
		CategoryCounts: map[string]int{},
		RemovedCounts:  map[string]int{},
	}
	protected := guardianRetentionProtectedPathsV0(config, result)
	for category, dir := range guardianRetentionCategoryDirsV0(config) {
		entries := guardianRetentionEntriesV0(dir)
		status.CategoryCounts[category] = len(entries)
		status.IncompleteTmp += countGuardianTmpEntriesV0(entries)
		removed := pruneGuardianRetentionEntriesV0(entries, policy.MaxArtifacts[category], protected)
		if removed > 0 {
			status.RemovedCounts[category] = removed
			status.CategoryCounts[category] -= removed
		}
	}
	if status.IncompleteTmp > 0 {
		status.Status = guardianStatusManifestIncompleteV0
		status.ReasonCode = "guardian_manifest_tmp_incomplete"
	}
	return status
}

func guardianRetentionPolicyForConfigV0(config guardianConfigV0) guardianRetentionPolicyV0 {
	defaults := map[string]int{
		"terminal_manifests": 100,
		"repair_packets":     100,
		"redacted_logs":      200,
		"raw_logs_opt_in":    0,
		"candidate_state":    10,
		"candidate_runtime":  10,
		"repair_runtime":     10,
	}
	for key, value := range config.RetentionPolicy.MaxArtifacts {
		defaults[key] = value
	}
	return guardianRetentionPolicyV0{MaxArtifacts: defaults}
}

func guardianRetentionCategoryDirsV0(config guardianConfigV0) map[string]string {
	return map[string]string{
		"terminal_manifests": filepath.Join(config.StateDir, "manifests"),
		"repair_packets":     filepath.Join(config.StateDir, "repair"),
		"redacted_logs":      filepath.Join(config.StateDir, "logs"),
		"candidate_state":    filepath.Join(config.StateDir, "candidate-state"),
		"candidate_runtime":  filepath.Join(config.StateDir, "candidate-runtime"),
		"repair_runtime":     config.RepairCodexRuntimeDir,
	}
}

func guardianRetentionProtectedPathsV0(
	config guardianConfigV0,
	result guardianResultV0,
) map[string]bool {
	protected := map[string]bool{}
	for _, path := range []string{
		config.ManifestPath,
		config.RepairPacketPath,
		config.CandidateStateDir,
		config.CandidateRuntimeDir,
		result.ManifestPath,
		result.RepairPacketPath,
	} {
		if strings.TrimSpace(path) != "" {
			protected[filepath.Clean(path)] = true
		}
	}
	for _, command := range result.Commands {
		if strings.TrimSpace(command.OutputPath) != "" {
			protected[filepath.Clean(command.OutputPath)] = true
		}
	}
	return protected
}

type guardianRetentionEntryV0 struct {
	Path    string
	ModUnix int64
	IsTmp   bool
}

func guardianRetentionEntriesV0(dir string) []guardianRetentionEntryV0 {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]guardianRetentionEntryV0, 0, len(items))
	for _, item := range items {
		info, err := item.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dir, item.Name())
		out = append(out, guardianRetentionEntryV0{
			Path:    filepath.Clean(path),
			ModUnix: info.ModTime().UnixNano(),
			IsTmp:   strings.Contains(item.Name(), ".tmp-"),
		})
	}
	return out
}

func countGuardianTmpEntriesV0(entries []guardianRetentionEntryV0) int {
	count := 0
	for _, entry := range entries {
		if entry.IsTmp {
			count++
		}
	}
	return count
}

func pruneGuardianRetentionEntriesV0(
	entries []guardianRetentionEntryV0,
	max int,
	protected map[string]bool,
) int {
	if max < 0 || len(entries) <= max {
		return 0
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ModUnix < entries[j].ModUnix })
	removed := 0
	for _, entry := range entries {
		if len(entries)-removed <= max {
			break
		}
		if protected[entry.Path] {
			continue
		}
		if err := os.RemoveAll(entry.Path); err == nil {
			removed++
		}
	}
	return removed
}
