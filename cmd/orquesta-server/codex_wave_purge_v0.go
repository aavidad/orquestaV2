package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const codexWavePurgeReportSchemaVersionV0 = "orquesta_codex_wave_runtime_purge_report.v0"

type codexWavePurgeReportV0 struct {
	SchemaVersion string   `json:"schema_version"`
	WaveRef       string   `json:"wave_ref"`
	RuntimeRef    string   `json:"runtime_ref"`
	Status        string   `json:"status"`
	FileCount     int      `json:"file_count"`
	DirCount      int      `json:"dir_count"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
	BlockedBy     []string `json:"blocked_by,omitempty"`
}

func codexWaveApplyRuntimePurgeV0(config codexWaveConfigV0) (*codexWavePurgeReportV0, error) {
	report, err := codexWaveBuildPurgeReportV0(config)
	if err != nil {
		return nil, err
	}
	if len(report.BlockedBy) > 0 {
		report.Status = "blocked"
		return report, errors.New("runtime_purge_blocked")
	}
	if report.Status == "not_found" || config.PurgeReportOnly {
		if report.Status != "not_found" {
			report.Status = "report_only"
		}
		return report, nil
	}
	if err := os.RemoveAll(config.RuntimeWorkDir); err != nil {
		return report, err
	}
	report.Status = "purged"
	report.EvidenceRefs = append(report.EvidenceRefs, "runtime-purge-ref-"+safeCodexWavePurgeRefPartV0(config.WaveRef))
	return report, nil
}

func codexWaveBuildPurgeReportV0(config codexWaveConfigV0) (*codexWavePurgeReportV0, error) {
	if err := codexWaveValidatePurgeRequestV0(config); err != nil {
		return nil, err
	}
	report := &codexWavePurgeReportV0{
		SchemaVersion: codexWavePurgeReportSchemaVersionV0,
		WaveRef:       strings.TrimSpace(config.WaveRef),
		RuntimeRef:    "runtime-ref-" + safeCodexWavePurgeRefPartV0(config.WaveRef),
		Status:        "ready",
		EvidenceRefs:  []string{"runtime-purge-manifest-ref-" + safeCodexWavePurgeRefPartV0(config.WaveRef)},
	}
	if _, err := os.Stat(config.RuntimeWorkDir); errors.Is(err, os.ErrNotExist) {
		report.Status = "not_found"
		return report, nil
	} else if err != nil {
		return nil, err
	}
	if err := filepath.WalkDir(config.RuntimeWorkDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == config.RuntimeWorkDir {
			return nil
		}
		if entry.IsDir() {
			report.DirCount++
		} else {
			report.FileCount++
		}
		codexWaveAddPurgeBlockersV0(report, path, entry)
		return nil
	}); err != nil {
		return nil, err
	}
	codexWaveAddRegistryPurgeBlockersV0(report, config.RuntimeWorkDir)
	report.BlockedBy = compactCodexWavePurgeStringsV0(report.BlockedBy)
	return report, nil
}

func codexWaveValidatePurgeRequestV0(config codexWaveConfigV0) error {
	if strings.TrimSpace(config.WaveRef) == "" {
		return errors.New("wave_ref_required_for_purge")
	}
	if strings.TrimSpace(config.PurgeConfirm) != strings.TrimSpace(config.WaveRef) {
		return errors.New("runtime_purge_confirmation_required")
	}
	if strings.TrimSpace(config.RuntimeWorkDir) == "" {
		return errors.New("runtime_dir_purge_unsafe")
	}
	cleanDir := filepath.Clean(config.RuntimeWorkDir)
	if cleanDir == "." || cleanDir == string(filepath.Separator) || filepath.Base(cleanDir) != config.WaveRef {
		return errors.New("runtime_dir_purge_ref_mismatch")
	}
	if !codexWaveRuntimeUnderAllowedRootV0(cleanDir, config.ProjectWorkDir) {
		return errors.New("runtime_dir_purge_root_not_allowed")
	}
	return nil
}

func codexWaveRuntimeUnderAllowedRootV0(runtimeDir string, projectWorkDir string) bool {
	runtimeDir, err := filepath.Abs(runtimeDir)
	if err != nil {
		return false
	}
	for _, root := range codexWaveAllowedPurgeRootsV0(projectWorkDir) {
		rel, err := filepath.Rel(root, runtimeDir)
		if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
			continue
		}
		return true
	}
	return false
}

func codexWaveAllowedPurgeRootsV0(projectWorkDir string) []string {
	roots := make([]string, 0, 2)
	if strings.TrimSpace(projectWorkDir) != "" {
		roots = append(roots, filepath.Join(projectWorkDir, ".orquesta-runtime", "codex-waves"))
		roots = append(roots, filepath.Join(filepath.Dir(filepath.Clean(projectWorkDir)), "runtime"))
	}
	if base := strings.TrimSpace(os.Getenv(envCodexRuntimeWorkDirV0)); base != "" {
		roots = append(roots, filepath.Join(base, "codex-waves"))
	}
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		if abs, err := filepath.Abs(root); err == nil {
			out = append(out, filepath.Clean(abs))
		}
	}
	return out
}

func codexWaveAddPurgeBlockersV0(report *codexWavePurgeReportV0, path string, entry fs.DirEntry) {
	name := entry.Name()
	dir := filepath.Dir(path)
	switch name {
	case "agent_ack.json":
		report.BlockedBy = append(report.BlockedBy, "agent_ack_unreconciled")
	case "director_decisions.json":
		report.BlockedBy = append(report.BlockedBy, "director_decisions_pending")
	case "orquesta_shutdown_request.json":
		if _, err := os.Stat(filepath.Join(dir, "agent_shutdown_checkpoint_ack.json")); errors.Is(err, os.ErrNotExist) {
			report.BlockedBy = append(report.BlockedBy, "checkpoint_pending")
		}
	default:
		low := strings.ToLower(name)
		if strings.Contains(low, "outbox") || strings.Contains(low, "plan_state") {
			report.BlockedBy = append(report.BlockedBy, "open_state_or_outbox_pending")
		}
	}
}

func codexWaveAddRegistryPurgeBlockersV0(report *codexWavePurgeReportV0, runtimeDir string) {
	summary, err := codexWaveLoadRegistryV0(runtimeDir)
	if err != nil {
		return
	}
	for _, agent := range summary.Agents {
		status := strings.TrimSpace(agent.Status)
		if agent.PID > 0 && processAliveV0(agent.PID) {
			report.BlockedBy = append(report.BlockedBy, "agent_live")
			continue
		}
		if status == "running" || status == "stop_requested" {
			report.BlockedBy = append(report.BlockedBy, "agent_live")
		}
	}
}

func safeCodexWavePurgeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-").Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func compactCodexWavePurgeStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
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

func codexWavePurgeReportJSONV0(report *codexWavePurgeReportV0) string {
	if report == nil {
		return ""
	}
	data, _ := json.Marshal(report)
	return string(data)
}
