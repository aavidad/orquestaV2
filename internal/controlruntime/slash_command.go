package controlruntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	defaultSlashCommandTimeout = 60 * time.Second
	defaultSlashCommandSettle  = 3 * time.Second
	defaultSlashCommandPoll    = 150 * time.Millisecond
	maxSlashCommandBytes       = 128 * 1024
)

var (
	ErrSlashCommandBrokenPipe = errors.New("runtime pty input broken")
	ErrSlashCommandNotDelivered = errors.New("runtime pty input not delivered")
)

type SlashCommandOptions struct {
	Timeout  time.Duration
	Settle   time.Duration
	Poll     time.Duration
	MaxBytes int
}

type SlashCommandResult struct {
	Command    string
	StartedAt  time.Time
	FinishedAt time.Time
	RawOutput  string
}

var ansiEscapePattern = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1b\\))`)

func RunSlashCommandLive(obj ObjetivoProceso, command string, opts SlashCommandOptions) (*SlashCommandResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("slash command vacio")
	}
	if !strings.HasPrefix(command, "/") {
		command = "/" + command
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultSlashCommandTimeout
	}
	if opts.Settle <= 0 {
		opts.Settle = defaultSlashCommandSettle
	}
	if opts.Poll <= 0 {
		opts.Poll = defaultSlashCommandPoll
	}
	if opts.MaxBytes <= 0 || opts.MaxBytes > maxSlashCommandBytes {
		opts.MaxBytes = maxSlashCommandBytes
	}

	if estado, observed, err := ConsultarEstadoLocal(obj); err == nil && observed && estado != nil && strings.TrimSpace(estado.MetadataJSON) != "" {
		obj.MetadataJSON = estado.MetadataJSON
	}
	logPath := strings.TrimSpace(stringValueFromMetadata(metadataMap(obj.MetadataJSON), "log_path"))
	if logPath == "" {
		slashCommandDebugf("slash command=%s status=no_log_path handle_kind=%s handle_ref=%s", command, strings.TrimSpace(obj.HandleKind), strings.TrimSpace(obj.HandleRef))
		return nil, fmt.Errorf("el runtime no expone log_path para slash commands")
	}
	if wait := slashCommandWarmupDelay(obj, command); wait > 0 {
		slashCommandDebugf("slash command=%s status=warmup_wait delay=%s handle_kind=%s handle_ref=%s", command, wait.Round(time.Millisecond), strings.TrimSpace(obj.HandleKind), strings.TrimSpace(obj.HandleRef))
		time.Sleep(wait)
	}
	startOffset := runtimeTraceFileSize(logPath)
	startedAt := time.Now().UTC()
	ok, _, err := sendSlashCommandTyped(obj, command)
	if err != nil {
		slashCommandDebugf("slash command=%s status=send_error err=%v", command, err)
		return nil, err
	}
	if !ok {
		slashCommandDebugf("slash command=%s status=not_delivered", command)
		return nil, fmt.Errorf("no se pudo entregar %s al runtime", command)
	}
	slashCommandDebugf("slash command=%s status=sent log_path=%s offset=%d", command, logPath, startOffset)

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	var (
		lastDataAt time.Time
		lastOutput string
		seenData   bool
	)
	ticker := time.NewTicker(opts.Poll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if seenData && strings.TrimSpace(lastOutput) != "" {
				slashCommandDebugf("slash command=%s status=timeout_with_output bytes=%d", command, len(lastOutput))
				return &SlashCommandResult{
					Command:    command,
					StartedAt:  startedAt,
					FinishedAt: time.Now().UTC(),
					RawOutput:  lastOutput,
				}, nil
			}
			slashCommandDebugf("slash command=%s status=timeout_without_output", command)
			return nil, fmt.Errorf("timeout esperando salida de %s", command)
		case <-ticker.C:
			output, changed, err := readRuntimeTraceDelta(logPath, startOffset, opts.MaxBytes)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			if changed {
				seenData = true
				lastDataAt = time.Now().UTC()
				lastOutput = output
				slashCommandDebugf("slash command=%s status=output_changed bytes=%d", command, len(output))
				if slashCommandOutputComplete(command, output) {
					slashCommandDebugf("slash command=%s status=complete bytes=%d duration=%s", command, len(lastOutput), time.Since(startedAt).Round(time.Millisecond))
					return &SlashCommandResult{
						Command:    command,
						StartedAt:  startedAt,
						FinishedAt: time.Now().UTC(),
						RawOutput:  lastOutput,
					}, nil
				}
				continue
			}
			if seenData && !lastDataAt.IsZero() && time.Since(lastDataAt) >= opts.Settle {
				slashCommandDebugf("slash command=%s status=ok bytes=%d duration=%s", command, len(lastOutput), time.Since(startedAt).Round(time.Millisecond))
				return &SlashCommandResult{
					Command:    command,
					StartedAt:  startedAt,
					FinishedAt: time.Now().UTC(),
					RawOutput:  lastOutput,
				}, nil
			}
		}
	}
}

func slashCommandWarmupDelay(obj ObjetivoProceso, command string) time.Duration {
	command = strings.TrimSpace(strings.ToLower(command))
	if command != "/status" {
		return 0
	}
	if !esRuntimeCodexLocal(obj) {
		return 0
	}
	meta := metadataMap(obj.MetadataJSON)
	startedAt := metadataTime(meta, "started_at")
	if startedAt.IsZero() {
		return 0
	}
	minAge := 50 * time.Second
	age := time.Since(startedAt.UTC())
	if age >= minAge {
		return 0
	}
	return minAge - age
}

func sendSlashCommandTyped(obj ObjetivoProceso, command string) (bool, int, error) {
	pid, ok, err := ResolverPID(obj)
	if err != nil {
		return false, pid, err
	}
	if !ok {
		return EnviarInstruccionProceso(obj, command)
	}
	vivo, _, err := ProcesoVivo(obj)
	if err != nil {
		return false, pid, err
	}
	if !vivo {
		return false, pid, nil
	}
	stdinPath := stdinPathFromMetadata(obj.MetadataJSON)
	if stdinPath == "" {
		return EnviarInstruccionProceso(obj, command)
	}
	fd, err := syscall.Open(stdinPath, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if err == syscall.ENXIO || err == syscall.ENOENT {
			slashCommandDebugf("slash command=%s status=stdin_open_unavailable stdin=%s err=%v", command, stdinPath, err)
			return false, pid, ErrSlashCommandBrokenPipe
		}
		return true, pid, err
	}
	f := os.NewFile(uintptr(fd), stdinPath)
	if f == nil {
		_ = syscall.Close(fd)
		return false, pid, nil
	}
	defer f.Close()
	texto := strings.TrimSpace(command)
	if texto == "" {
		return false, pid, nil
	}
	slashCommandDebugf("slash command=%s status=typing_start pid=%d stdin=%s", command, pid, stdinPath)
	if _, err := io.WriteString(f, "\u0003"); err == nil {
		time.Sleep(150 * time.Millisecond)
	}
	for _, r := range texto {
		if _, err := io.WriteString(f, string(r)); err != nil {
			if err == syscall.EPIPE {
				slashCommandDebugf("slash command=%s status=typed_broken_pipe pid=%d stdin=%s", command, pid, stdinPath)
				return false, pid, ErrSlashCommandBrokenPipe
			}
			return true, pid, err
		}
		time.Sleep(35 * time.Millisecond)
	}
	time.Sleep(600 * time.Millisecond)
	if _, err := io.WriteString(f, "\n"); err != nil {
		if err == syscall.EPIPE {
			slashCommandDebugf("slash command=%s status=enter1_broken_pipe pid=%d stdin=%s", command, pid, stdinPath)
			return false, pid, ErrSlashCommandBrokenPipe
		}
		return true, pid, err
	}
	time.Sleep(900 * time.Millisecond)
	if _, err := io.WriteString(f, "\n"); err != nil {
		if err == syscall.EPIPE {
			slashCommandDebugf("slash command=%s status=enter2_broken_pipe pid=%d stdin=%s", command, pid, stdinPath)
			return false, pid, ErrSlashCommandBrokenPipe
		}
		return true, pid, err
	}
	_ = appendRuntimeRawInput(stdinRawPathFromMetadata(obj.MetadataJSON), texto)
	slashCommandDebugf("slash command=%s status=typing_done pid=%d stdin=%s", command, pid, stdinPath)
	return true, pid, nil
}

func slashCommandOutputComplete(command, raw string) bool {
	command = strings.TrimSpace(strings.ToLower(command))
	text := normalizeSlashCommandOutput(raw)
	if text == "" {
		return false
	}
	switch command {
	case "/status":
		if strings.Contains(text, "Account:") &&
			strings.Contains(text, "5h limit:") &&
			strings.Contains(text, "Weekly limit:") {
			return true
		}
		if strings.HasPrefix(strings.ToLower(text), "status: ") {
			return true
		}
	}
	return false
}

func slashCommandDebugf(format string, args ...any) {
	if !runtimeStatusDebugEnabled() {
		return
	}
	log.Printf("orquesta[runtime_status][debug] "+format, args...)
}

func runtimeStatusDebugEnabled() bool {
	for _, raw := range []string{
		strings.TrimSpace(os.Getenv("ORQUESTA_DEBUG_RUNTIME_STATUS")),
		strings.TrimSpace(os.Getenv("ORQUESTA_DEBUG")),
	} {
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "on", "si", "sí":
			return true
		}
	}
	return false
}

func readRuntimeTraceDelta(path string, offset int64, maxBytes int) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", false, err
	}
	total := info.Size()
	if total <= offset {
		return "", false, nil
	}
	start := offset
	if total-start > int64(maxBytes) {
		start = total - int64(maxBytes)
	}
	buf := make([]byte, total-start)
	n, err := file.ReadAt(buf, start)
	if err != nil && err != io.EOF {
		return "", false, err
	}
	buf = bytes.ToValidUTF8(buf[:n], []byte("?"))
	return string(buf), true, nil
}

func runtimeTraceFileSize(path string) int64 {
	info, err := os.Stat(strings.TrimSpace(path))
	if err != nil {
		return 0
	}
	return info.Size()
}

func normalizeSlashCommandOutput(raw string) string {
	raw = ansiEscapePattern.ReplaceAllString(raw, "")
	replacer := strings.NewReplacer(
		"\r\n", "\n",
		"\r", "\n",
		"│", " ",
		"╭", " ",
		"╮", " ",
		"╰", " ",
		"╯", " ",
		"─", " ",
		">_", " ",
	)
	raw = replacer.Replace(raw)
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
