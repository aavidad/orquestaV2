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

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

type codexWaveControlConfigV0 struct {
	WaveRef        string
	RuntimeWorkDir string
	AgentRef       string
	LogKind        string
	Mode           string
	Reason         string
	ConfirmStop    string
	ForceStop      bool
	Lines          int
	MaxBytes       int64
}

func codexWaveStatusCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := codexWaveControlConfigFromArgsV0("codex-wave-status", args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-status: %v\n", err)
		return 2
	}
	summary, err := codexWaveLoadTrustedRegistryV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-status: %v\n", err)
		return 1
	}
	codexWaveRefreshSummaryV0(&summary)
	if err := codexWaveSaveRegistryV0(summary); err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-status: %v\n", err)
		return 1
	}
	if err := writeCommandJSONOutputV0(stdout, codexWavePublicSummaryFromV0(summary)); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "codex-wave-status", "stdout", "json_encode", err)
	}
	return 0
}

func codexWaveStopCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := codexWaveControlConfigFromArgsV0("codex-wave-stop", args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-stop: %v\n", err)
		return 2
	}
	summary, err := codexWaveLoadTrustedRegistryV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-stop: %v\n", err)
		return 1
	}
	codexWaveApplyStopRequestV0(&summary, config, time.Now().UTC())
	codexWaveRefreshSummaryV0(&summary)
	if err := codexWaveSaveRegistryV0(summary); err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-stop: %v\n", err)
		return 1
	}
	if err := writeCommandJSONOutputV0(stdout, codexWavePublicSummaryFromV0(summary)); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "codex-wave-stop", "stdout", "json_encode", err)
	}
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
	summary, err := codexWaveLoadTrustedRegistryV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-tail: %v\n", err)
		return 1
	}
	report, err := codexWaveBuildTailReportV0(summary, config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-wave-tail: %v\n", err)
		if errors.Is(err, errCodexWaveTailBadRequestV0) {
			return 2
		}
		return 1
	}
	if err := writeCommandJSONOutputV0(stdout, report); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "codex-wave-tail", "stdout", "json_encode", err)
	}
	return 0
}

func codexWaveControlConfigFromArgsV0(command string, args []string, stderr io.Writer) (codexWaveControlConfigV0, error) {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	waveRef := flags.String("wave-ref", strings.TrimSpace(os.Getenv(envCodexWaveRefV0)), "ref de la ola")
	runtimeDir := flags.String("runtime-dir", strings.TrimSpace(os.Getenv(envCodexWaveRuntimeWorkDirV0)), "directorio runtime de la ola")
	agentRef := flags.String("agent-ref", "", "agente concreto")
	logKind := flags.String("file", "stdout", "stdout, stderr o last-message")
	mode := flags.String("mode", "summary", "summary o fragment")
	reason := flags.String("reason", strings.TrimSpace(os.Getenv(envCodexWaveTailReasonV0)), "razon de diagnostico")
	confirmStop := flags.String("confirm-stop", strings.TrimSpace(os.Getenv(envCodexWaveStopConfirmV0)), "confirmacion explicita: debe coincidir con wave-ref")
	forceStop := flags.Bool("force", boolEnvOrDefaultV0(envCodexWaveStopForceV0, false), "senalar proceso tras confirmacion explicita")
	lines := flags.Int("lines", 20, "lineas maximas a leer")
	maxBytes := flags.Int64("max-bytes", 8192, "bytes maximos a leer")
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
	if *maxBytes <= 0 || *maxBytes > codexWaveTailMaxBytesCeilingV0 {
		return codexWaveControlConfigV0{}, errors.New("max_bytes_out_of_range")
	}
	return codexWaveControlConfigV0{
		WaveRef:        strings.TrimSpace(*waveRef),
		RuntimeWorkDir: absRuntimeDir,
		AgentRef:       strings.TrimSpace(*agentRef),
		LogKind:        strings.TrimSpace(*logKind),
		Mode:           strings.TrimSpace(*mode),
		Reason:         strings.TrimSpace(*reason),
		ConfirmStop:    strings.TrimSpace(*confirmStop),
		ForceStop:      *forceStop,
		Lines:          *lines,
		MaxBytes:       *maxBytes,
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
	_, err = orquestaruntimecodex.WriteCodexControlFileBytesV0(orquestaruntimecodex.CodexControlFileWriteRequestV0{
		RootDir:     summary.RuntimeWorkDir,
		Path:        summary.RegistryPath,
		FileName:    codexWaveRegistryFileNameV0,
		ControlKind: "codex_wave_registry",
		Data:        append(data, '\n'),
		Mode:        orquestaruntimecodex.CodexControlFileWriteCreateOrReplaceV0,
		Perm:        0o600,
	})
	return err
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
		if codexWaveAgentProcessDoneV0(*agent) {
			agent.Status = "stopped"
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

func codexWaveAgentProcessDoneV0(agent codexWaveAgentSummaryV0) bool {
	if strings.TrimSpace(agent.WrapperPath) == "" {
		return false
	}
	_, err := os.Stat(codexWaveProcessDonePathV0(agent.WrapperPath))
	return err == nil
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
