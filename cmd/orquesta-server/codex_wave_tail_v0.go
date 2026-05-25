package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const (
	codexWaveTailSchemaVersionV0   = "orquesta_codex_wave_tail.v0"
	codexWaveTailMaxBytesCeilingV0 = int64(32 * 1024)
)

var errCodexWaveTailBadRequestV0 = errors.New("codex_wave_tail_bad_request")

type codexWaveTailReportV0 struct {
	SchemaVersion string                   `json:"schema_version"`
	WaveRef       string                   `json:"wave_ref"`
	LogKind       string                   `json:"log_kind"`
	Mode          string                   `json:"mode"`
	Reason        string                   `json:"reason_ref"`
	LineLimit     int                      `json:"line_limit"`
	ByteLimit     int64                    `json:"byte_limit"`
	Agents        []codexWaveTailAgentV0   `json:"agents"`
	Errors        []codexWavePublicErrorV0 `json:"errors,omitempty"`
}

type codexWaveTailAgentV0 struct {
	AgentRef         string   `json:"agent_ref"`
	Status           string   `json:"status"`
	BytesObserved    int64    `json:"bytes_observed"`
	BytesRead        int64    `json:"bytes_read"`
	LinesReturned    int      `json:"lines_returned"`
	Truncated        bool     `json:"truncated"`
	RedactionApplied bool     `json:"redaction_applied"`
	Summary          string   `json:"summary"`
	Fragment         []string `json:"fragment,omitempty"`
}

func codexWaveBuildTailReportV0(
	summary codexWaveLaunchSummaryV0,
	config codexWaveControlConfigV0,
) (codexWaveTailReportV0, error) {
	if strings.TrimSpace(config.Reason) == "" {
		return codexWaveTailReportV0{}, errors.Join(errCodexWaveTailBadRequestV0, errors.New("diagnostic_reason_required"))
	}
	mode, err := codexWaveTailModeV0(config.Mode)
	if err != nil {
		return codexWaveTailReportV0{}, err
	}
	if err := codexWaveTailRegistryScopeV0(summary, config); err != nil {
		return codexWaveTailReportV0{}, err
	}
	report := codexWaveTailReportV0{
		SchemaVersion: codexWaveTailSchemaVersionV0,
		WaveRef:       summary.WaveRef,
		LogKind:       strings.TrimSpace(config.LogKind),
		Mode:          mode,
		Reason:        config.Reason,
		LineLimit:     config.Lines,
		ByteLimit:     config.MaxBytes,
	}
	for _, agent := range summary.Agents {
		if config.AgentRef != "" && agent.AgentRef != config.AgentRef {
			continue
		}
		item, err := codexWaveTailAgentReportV0(summary, agent, config, mode)
		if err != nil {
			return codexWaveTailReportV0{}, err
		}
		report.Agents = append(report.Agents, item)
	}
	if len(report.Agents) == 0 {
		return codexWaveTailReportV0{}, errors.Join(errCodexWaveTailBadRequestV0, errors.New("agent_ref_not_found"))
	}
	return report, nil
}

func codexWaveTailAgentReportV0(
	summary codexWaveLaunchSummaryV0,
	agent codexWaveAgentSummaryV0,
	config codexWaveControlConfigV0,
	mode string,
) (codexWaveTailAgentV0, error) {
	path, err := codexWaveAgentLogPathV0(agent, config.LogKind)
	if err != nil {
		return codexWaveTailAgentV0{}, err
	}
	if err := codexWaveTailLogAccessV0(summary, agent, path); err != nil {
		return codexWaveTailAgentV0{}, err
	}
	text, bytesRead, truncated, err := codexWaveTailFileLimitedV0(path, config.Lines, config.MaxBytes)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return codexWaveTailAgentV0{}, err
	}
	redacted, changed := orquestarails.RedactOperationalTextForFieldV0("codex_wave_tail", config.LogKind, text)
	lines := codexWaveTailLinesV0(redacted, config.Lines)
	item := codexWaveTailAgentV0{
		AgentRef:         agent.AgentRef,
		Status:           agent.Status,
		BytesObserved:    fileSizeOrZeroV0(path),
		BytesRead:        bytesRead,
		LinesReturned:    len(lines),
		Truncated:        truncated,
		RedactionApplied: changed,
		Summary:          codexWaveTailSummaryTextV0(lines, changed, truncated),
	}
	if mode == "fragment" {
		item.Fragment = lines
	}
	return item, nil
}

func codexWaveTailRegistryScopeV0(summary codexWaveLaunchSummaryV0, config codexWaveControlConfigV0) error {
	if summary.WaveRef == "" || (config.WaveRef != "" && summary.WaveRef != config.WaveRef) {
		return errors.Join(errCodexWaveTailBadRequestV0, errors.New("wave_ref_mismatch"))
	}
	if !sameCleanPathV0(summary.RuntimeWorkDir, config.RuntimeWorkDir) {
		return errors.Join(errCodexWaveTailBadRequestV0, errors.New("runtime_scope_mismatch"))
	}
	return nil
}

func codexWaveTailLogAccessV0(summary codexWaveLaunchSummaryV0, agent codexWaveAgentSummaryV0, path string) error {
	if !pathUnderDirV0(agent.RuntimeWorkDir, path) || !pathUnderDirV0(summary.RuntimeWorkDir, path) {
		return errors.Join(errCodexWaveTailBadRequestV0, errors.New("log_path_out_of_scope"))
	}
	return nil
}

func codexWaveAgentLogPathV0(agent codexWaveAgentSummaryV0, kind string) (string, error) {
	switch strings.TrimSpace(kind) {
	case "stdout":
		return agent.StdoutPath, nil
	case "stderr":
		return agent.StderrPath, nil
	case "last-message":
		return agent.LastMessagePath, nil
	default:
		return "", errors.Join(errCodexWaveTailBadRequestV0, errors.New("log_file_kind_invalid"))
	}
}

func codexWaveTailFileLimitedV0(path string, lines int, maxBytes int64) (string, int64, bool, error) {
	if lines <= 0 {
		return "", 0, false, errors.Join(errCodexWaveTailBadRequestV0, errors.New("lines_out_of_range"))
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, false, err
	}
	if info.IsDir() {
		return "", 0, false, errors.New("log_path_is_dir")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", 0, false, err
	}
	defer file.Close()
	start := int64(0)
	truncated := info.Size() > maxBytes
	if truncated {
		start = info.Size() - maxBytes
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return "", 0, false, err
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes))
	if err != nil {
		return "", 0, false, err
	}
	text := string(data)
	if truncated {
		text = strings.TrimLeft(text, "\r\n")
	}
	return strings.Join(codexWaveTailLinesV0(text, lines), "\n"), int64(len(data)), truncated, nil
}

func codexWaveTailLinesV0(text string, lines int) []string {
	parts := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return parts
}

func codexWaveTailModeV0(mode string) (string, error) {
	switch strings.TrimSpace(mode) {
	case "", "summary":
		return "summary", nil
	case "fragment":
		return "fragment", nil
	default:
		return "", errors.Join(errCodexWaveTailBadRequestV0, errors.New("tail_mode_invalid"))
	}
}

func codexWaveTailSummaryTextV0(lines []string, redacted bool, truncated bool) string {
	parts := []string{"diagnostic_log_tail"}
	if len(lines) == 0 {
		parts = append(parts, "empty")
	}
	if redacted {
		parts = append(parts, "redacted")
	}
	if truncated {
		parts = append(parts, "truncated")
	}
	return strings.Join(parts, ":")
}

func sameCleanPathV0(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(strings.TrimSpace(left))
	rightAbs, rightErr := filepath.Abs(strings.TrimSpace(right))
	if leftErr != nil || rightErr != nil {
		return false
	}
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func pathUnderDirV0(dir string, path string) bool {
	dirAbs, dirErr := filepath.Abs(strings.TrimSpace(dir))
	pathAbs, pathErr := filepath.Abs(strings.TrimSpace(path))
	if dirErr != nil || pathErr != nil {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(dirAbs), filepath.Clean(pathAbs))
	return err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
