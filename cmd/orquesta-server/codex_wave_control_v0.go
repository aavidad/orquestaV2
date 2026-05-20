package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type codexWaveControlConfigV0 struct {
	WaveRef        string
	RuntimeWorkDir string
	AgentRef       string
	LogKind        string
	Lines          int
}

func codexWaveStatusCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := codexWaveControlConfigFromArgsV0("codex-wave-status", args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-status: %v\n", err)
		return 2
	}
	summary, err := codexWaveLoadRegistryV0(config.RuntimeWorkDir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-status: %v\n", err)
		return 1
	}
	codexWaveRefreshSummaryV0(&summary)
	if err := codexWaveSaveRegistryV0(summary); err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-status: %v\n", err)
		return 1
	}
	_ = json.NewEncoder(stdout).Encode(summary)
	return 0
}

func codexWaveStopCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := codexWaveControlConfigFromArgsV0("codex-wave-stop", args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-stop: %v\n", err)
		return 2
	}
	summary, err := codexWaveLoadRegistryV0(config.RuntimeWorkDir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-stop: %v\n", err)
		return 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range summary.Agents {
		if config.AgentRef != "" && summary.Agents[i].AgentRef != config.AgentRef {
			continue
		}
		if summary.Agents[i].PID <= 0 || !processAliveV0(summary.Agents[i].PID) {
			continue
		}
		if err := signalProcessV0(summary.Agents[i].PID); err != nil {
			summary.Errors = append(summary.Errors, codexWavePublicErrorV0{
				AgentRef: summary.Agents[i].AgentRef,
				Code:     "stop_failed",
			})
			continue
		}
		summary.Agents[i].StopRequestedAt = now
		summary.Agents[i].Status = "stop_requested"
	}
	codexWaveRefreshSummaryV0(&summary)
	if err := codexWaveSaveRegistryV0(summary); err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-stop: %v\n", err)
		return 1
	}
	_ = json.NewEncoder(stdout).Encode(summary)
	if len(summary.Errors) > 0 {
		return 1
	}
	return 0
}

func codexWaveTailCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := codexWaveControlConfigFromArgsV0("codex-wave-tail", args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-tail: %v\n", err)
		return 2
	}
	summary, err := codexWaveLoadRegistryV0(config.RuntimeWorkDir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-tail: %v\n", err)
		return 1
	}
	matched := 0
	for _, agent := range summary.Agents {
		if config.AgentRef != "" && agent.AgentRef != config.AgentRef {
			continue
		}
		matched++
		path, err := codexWaveAgentLogPathV0(agent, config.LogKind)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "codex-wave-tail: %v\n", err)
			return 2
		}
		text, err := codexWaveTailFileV0(path, config.Lines)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				text = ""
			} else {
				_, _ = fmt.Fprintf(stderr, "codex-wave-tail: %v\n", err)
				return 1
			}
		}
		_, _ = fmt.Fprintf(stdout, "== %s %s ==\n", agent.AgentRef, config.LogKind)
		if text != "" {
			_, _ = fmt.Fprint(stdout, text)
			if !strings.HasSuffix(text, "\n") {
				_, _ = fmt.Fprintln(stdout)
			}
		}
	}
	if matched == 0 {
		_, _ = fmt.Fprintf(stderr, "codex-wave-tail: agent_ref_not_found\n")
		return 1
	}
	return 0
}

func codexWaveControlConfigFromArgsV0(command string, args []string, stderr io.Writer) (codexWaveControlConfigV0, error) {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	waveRef := flags.String("wave-ref", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_REF")), "ref de la ola")
	runtimeDir := flags.String("runtime-dir", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_RUNTIME_WORKDIR")), "directorio runtime de la ola")
	agentRef := flags.String("agent-ref", "", "agente concreto")
	logKind := flags.String("file", "stdout", "stdout, stderr o last-message")
	lines := flags.Int("lines", 80, "lineas a mostrar")
	if err := flags.Parse(args); err != nil {
		return codexWaveControlConfigV0{}, err
	}
	absRuntimeDir, err := codexWaveControlRuntimeDirV0(*runtimeDir, *waveRef)
	if err != nil {
		return codexWaveControlConfigV0{}, err
	}
	if *lines <= 0 {
		return codexWaveControlConfigV0{}, errors.New("lines_out_of_range")
	}
	return codexWaveControlConfigV0{
		WaveRef:        strings.TrimSpace(*waveRef),
		RuntimeWorkDir: absRuntimeDir,
		AgentRef:       strings.TrimSpace(*agentRef),
		LogKind:        strings.TrimSpace(*logKind),
		Lines:          *lines,
	}, nil
}

func codexWaveControlRuntimeDirV0(rawRuntimeDir string, rawWaveRef string) (string, error) {
	runtimeDir := strings.TrimSpace(rawRuntimeDir)
	waveRef := strings.TrimSpace(rawWaveRef)
	if runtimeDir != "" {
		return filepath.Abs(runtimeDir)
	}
	if waveRef == "" {
		return "", errors.New("wave_ref_required")
	}
	projectWorkDir, err := projectDirFromEnvV0()
	if err != nil {
		return "", err
	}
	return codexWaveRuntimeDirPathV0("", projectWorkDir, waveRef)
}

func codexWaveRegistryPathV0(runtimeWorkDir string) string {
	return filepath.Join(runtimeWorkDir, codexWaveRegistryFileNameV0)
}

func codexWaveLoadRegistryV0(runtimeWorkDir string) (codexWaveLaunchSummaryV0, error) {
	path := codexWaveRegistryPathV0(runtimeWorkDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return codexWaveLaunchSummaryV0{}, err
	}
	var summary codexWaveLaunchSummaryV0
	if err := json.Unmarshal(data, &summary); err != nil {
		return codexWaveLaunchSummaryV0{}, err
	}
	if summary.SchemaVersion != codexWaveSummarySchemaVersionV0 {
		return codexWaveLaunchSummaryV0{}, errors.New("codex_wave_registry_invalid")
	}
	summary.RegistryPath = path
	return summary, nil
}

func codexWaveSaveRegistryV0(summary codexWaveLaunchSummaryV0) error {
	if summary.RegistryPath == "" {
		summary.RegistryPath = codexWaveRegistryPathV0(summary.RuntimeWorkDir)
	}
	summary.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(summary.RegistryPath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(summary.RegistryPath, append(data, '\n'), 0o600)
}

func codexWaveRefreshSummaryV0(summary *codexWaveLaunchSummaryV0) {
	if summary == nil {
		return
	}
	for i := range summary.Agents {
		agent := &summary.Agents[i]
		agent.StdoutBytes = fileSizeOrZeroV0(agent.StdoutPath)
		agent.StderrBytes = fileSizeOrZeroV0(agent.StderrPath)
		agent.LastMessageBytes = fileSizeOrZeroV0(agent.LastMessagePath)
		if summary.DryRun {
			agent.Status = "dry_run"
			continue
		}
		if agent.PID > 0 && processAliveV0(agent.PID) {
			if agent.StopRequestedAt != "" {
				agent.Status = "stop_requested"
			} else {
				agent.Status = "running"
			}
			continue
		}
		if agent.PID > 0 {
			agent.Status = "stopped"
			continue
		}
		if agent.Status == "" {
			agent.Status = "unknown"
		}
	}
}

func fileSizeOrZeroV0(path string) int64 {
	if path == "" {
		return 0
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0
	}
	return info.Size()
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
		return "", errors.New("log_file_kind_invalid")
	}
}

func codexWaveTailFileV0(path string, lines int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := string(data)
	if lines <= 0 {
		return "", errors.New("lines_out_of_range")
	}
	parts := strings.SplitAfter(text, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) <= lines {
		return text, nil
	}
	return strings.Join(parts[len(parts)-lines:], ""), nil
}
