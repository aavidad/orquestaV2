package controlruntime

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type embeddedTmuxMonitorSpec struct {
	TmuxCommand         string `json:"tmux_command"`
	SessionName         string `json:"session_name"`
	WindowName          string `json:"window_name"`
	PaneID              string `json:"pane_id"`
	ChildPID            int    `json:"child_pid"`
	Agent               string `json:"agent"`
	Project             string `json:"project"`
	WorkingDir          string `json:"working_dir"`
	StartedAt           string `json:"started_at"`
	Command             string `json:"command"`
	LogPath             string `json:"log_path"`
	RuntimeManifestPath string `json:"runtime_manifest_path"`
	StatusPath          string `json:"status_path"`
	HeartbeatPath       string `json:"heartbeat_path"`
	ExternalSessionID   string `json:"external_session_id"`
	MailboxDeliveryMode string `json:"mailbox_delivery_mode"`
	OwnerPID            int    `json:"owner_pid"`
}

type tmuxPaneSnapshot struct {
	SessionName    string
	WindowName     string
	PaneID         string
	PanePID        int
	PaneDead       bool
	CurrentCommand string
	CurrentPath    string
}

const (
	embeddedTmuxMonitorHeartbeatInterval = 15 * time.Second
	embeddedTmuxProgressProbeInterval    = 60 * time.Second
	tmuxReadyInitialBackoff              = 150 * time.Millisecond
	tmuxReadyMaxBackoff                  = 2 * time.Second
	tmuxReadyTimeout                     = 15 * time.Second
	tmuxStartReadyTimeout                = 60 * time.Second
	tmuxSemanticDetectionTailBytes       = 8192
)

func MaybeRunEmbeddedTmuxMonitor(args []string) bool {
	if len(args) == 0 || strings.TrimSpace(args[0]) != "__tmux_monitor" {
		return false
	}
	if err := runEmbeddedTmuxMonitor(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
	return true
}

func runEmbeddedTmuxMonitor(args []string) error {
	fs := flag.NewFlagSet("__tmux_monitor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	specPath := fs.String("spec", "", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*specPath) == "" {
		return fmt.Errorf("tmux monitor sin --spec")
	}
	data, err := os.ReadFile(strings.TrimSpace(*specPath))
	if err != nil {
		return err
	}
	var spec embeddedTmuxMonitorSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return err
	}
	if strings.TrimSpace(spec.TmuxCommand) == "" {
		return fmt.Errorf("tmux monitor sin tmux_command")
	}
	if strings.TrimSpace(spec.PaneID) == "" {
		return fmt.Errorf("tmux monitor sin pane_id")
	}
	if strings.TrimSpace(spec.StatusPath) == "" || strings.TrimSpace(spec.HeartbeatPath) == "" {
		return fmt.Errorf("tmux monitor sin rutas de status/heartbeat")
	}

	_ = writeEmbeddedTmuxWorkerSnapshot(spec, workerStatusStarting, true, "", nil, spec.ChildPID, time.Now().UTC(), time.Time{})

	ticker := time.NewTicker(embeddedTmuxMonitorHeartbeatInterval)
	defer ticker.Stop()

	var lastProgressAt time.Time
	var lastProgressProbe time.Time
	for {
		if spec.OwnerPID > 0 {
			if alive, err := procesoVivoPID(spec.OwnerPID); err == nil && !alive {
				return nil
			}
		}
		snap, err := consultarTmuxPane(spec)
		now := time.Now().UTC()
		if err != nil {
			_ = writeEmbeddedTmuxWorkerSnapshot(spec, workerStatusFailed, false, err.Error(), nil, 0, now, lastProgressAt)
			return nil
		}
		if snap.PaneDead || snap.PanePID <= 0 {
			_ = writeEmbeddedTmuxWorkerSnapshot(spec, workerStatusStopped, false, "tmux pane finalizado", nil, snap.PanePID, now, lastProgressAt)
			return nil
		}
		if spec.ChildPID <= 0 {
			spec.ChildPID = snap.PanePID
		}
		state, stateErr := currentTMUXWorkerState(strings.TrimSpace(spec.TmuxCommand), strings.TrimSpace(spec.PaneID))
		if stateErr != nil {
			_ = writeEmbeddedTmuxWorkerSnapshot(spec, workerStatusFailed, false, stateErr.Error(), nil, snap.PanePID, now, lastProgressAt)
			return nil
		}
		if lastProgressProbe.IsZero() || now.Sub(lastProgressProbe) >= embeddedTmuxProgressProbeInterval {
			startedAt := time.Time{}
			if parsed := strings.TrimSpace(spec.StartedAt); parsed != "" {
				if ts, err := time.Parse(time.RFC3339Nano, parsed); err == nil {
					startedAt = ts.UTC()
				}
			}
			if progressAt, err := latestTMUXWorktreeProgressMoment(strings.TrimSpace(spec.WorkingDir), startedAt, now); err == nil && !progressAt.IsZero() && progressAt.After(lastProgressAt) {
				lastProgressAt = progressAt
			}
			lastProgressProbe = now
		}
		_ = writeEmbeddedTmuxWorkerSnapshot(spec, state, true, "", nil, snap.PanePID, now, lastProgressAt)
		<-ticker.C
	}
}

func consultarTmuxPane(spec embeddedTmuxMonitorSpec) (*tmuxPaneSnapshot, error) {
	format := "#{session_name}|#{window_name}|#{pane_id}|#{pane_pid}|#{pane_dead}|#{pane_current_command}|#{pane_current_path}"
	cmd := exec.Command(strings.TrimSpace(spec.TmuxCommand), "display-message", "-p", "-t", strings.TrimSpace(spec.PaneID), format)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(parts) < 7 {
		return nil, fmt.Errorf("salida tmux invalida: %q", strings.TrimSpace(string(out)))
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
	return &tmuxPaneSnapshot{
		SessionName:    strings.TrimSpace(parts[0]),
		WindowName:     strings.TrimSpace(parts[1]),
		PaneID:         strings.TrimSpace(parts[2]),
		PanePID:        pid,
		PaneDead:       strings.TrimSpace(parts[4]) == "1",
		CurrentCommand: strings.TrimSpace(parts[5]),
		CurrentPath:    strings.TrimSpace(parts[6]),
	}, nil
}

func writeEmbeddedTmuxWorkerSnapshot(spec embeddedTmuxMonitorSpec, state string, alive bool, exitError string, exitCode *int, childPID int, now time.Time, lastProgressAt time.Time) error {
	lastOutputAt := ""
	if info, err := os.Stat(strings.TrimSpace(spec.LogPath)); err == nil {
		lastOutputAt = info.ModTime().UTC().Format(time.RFC3339Nano)
	}
	lastProgress := ""
	if !lastProgressAt.IsZero() {
		lastProgress = lastProgressAt.UTC().Format(time.RFC3339Nano)
	}
	readyAt := ""
	if data, err := os.ReadFile(strings.TrimSpace(spec.StatusPath)); err == nil {
		var prev workerStatus
		if json.Unmarshal(data, &prev) == nil && strings.TrimSpace(prev.ReadyAt) != "" {
			readyAt = strings.TrimSpace(prev.ReadyAt)
		}
	}
	if readyAt == "" {
		if data, err := os.ReadFile(strings.TrimSpace(spec.HeartbeatPath)); err == nil {
			var prev workerHeartbeat
			if json.Unmarshal(data, &prev) == nil && strings.TrimSpace(prev.ReadyAt) != "" {
				readyAt = strings.TrimSpace(prev.ReadyAt)
			}
		}
	}
	if readyAt == "" && (strings.EqualFold(strings.TrimSpace(state), workerStatusReady) || strings.EqualFold(strings.TrimSpace(state), workerStatusRunning)) {
		readyAt = now.UTC().Format(time.RFC3339Nano)
	}
	status := workerStatus{
		State:               strings.TrimSpace(state),
		UpdatedAt:           now.UTC().Format(time.RFC3339Nano),
		Alive:               alive,
		ChildPID:            childPID,
		Agent:               strings.TrimSpace(spec.Agent),
		Project:             strings.TrimSpace(spec.Project),
		WorkingDir:          strings.TrimSpace(spec.WorkingDir),
		LogPath:             strings.TrimSpace(spec.LogPath),
		ExternalSessionID:   strings.TrimSpace(spec.ExternalSessionID),
		MailboxDeliveryMode: strings.TrimSpace(spec.MailboxDeliveryMode),
		ReadyAt:             readyAt,
		LastOutputAt:        lastOutputAt,
		LastProgressAt:      lastProgress,
		ExitCode:            exitCode,
		ExitError:           strings.TrimSpace(exitError),
	}
	heartbeat := workerHeartbeat{
		Alive:             alive,
		HeartbeatAt:       now.UTC().Format(time.RFC3339Nano),
		StartedAt:         now.UTC().Format(time.RFC3339Nano),
		ChildPID:          childPID,
		Agent:             strings.TrimSpace(spec.Agent),
		Project:           strings.TrimSpace(spec.Project),
		ExternalSessionID: strings.TrimSpace(spec.ExternalSessionID),
		ReadyAt:           readyAt,
		LastOutputAt:      lastOutputAt,
		LastProgressAt:    lastProgress,
		ExitCode:          exitCode,
		ExitError:         strings.TrimSpace(exitError),
	}
	if data, err := os.ReadFile(strings.TrimSpace(spec.HeartbeatPath)); err == nil {
		var prev workerHeartbeat
		if json.Unmarshal(data, &prev) == nil && strings.TrimSpace(prev.StartedAt) != "" {
			heartbeat.StartedAt = strings.TrimSpace(prev.StartedAt)
		}
	}
	if err := writeWorkerStatusFile(strings.TrimSpace(spec.StatusPath), status); err != nil {
		return err
	}
	return writeWorkerHeartbeatFile(strings.TrimSpace(spec.HeartbeatPath), heartbeat)
}

func latestTMUXWorktreeProgressMoment(workingDir string, baseline, fallback time.Time) (time.Time, error) {
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" {
		return time.Time{}, nil
	}
	cmd := exec.Command("git", "-C", workingDir, "status", "--porcelain", "--untracked-files=all")
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r", ""), "\n")
	latest := time.Time{}
	dirty := false
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" || len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if idx := strings.LastIndex(path, " -> "); idx >= 0 {
			path = strings.TrimSpace(path[idx+4:])
		}
		path = strings.Trim(path, "\"")
		if path == "" || !tmuxWorktreePathCountsAsProgress(path) {
			continue
		}
		dirty = true
		fullPath := filepath.Join(workingDir, filepath.FromSlash(path))
		if info, statErr := os.Stat(fullPath); statErr == nil {
			modAt := info.ModTime().UTC()
			if !baseline.IsZero() && modAt.Before(baseline) {
				continue
			}
			if modAt.After(latest) {
				latest = modAt
			}
		}
	}
	if !dirty {
		return time.Time{}, nil
	}
	if !latest.IsZero() {
		return latest, nil
	}
	if info, err := os.Stat(filepath.Join(workingDir, ".git", "index")); err == nil {
		modAt := info.ModTime().UTC()
		if baseline.IsZero() || !modAt.Before(baseline) {
			return modAt, nil
		}
	}
	if !baseline.IsZero() {
		return time.Time{}, nil
	}
	if !fallback.IsZero() {
		return fallback.UTC(), nil
	}
	return time.Time{}, nil
}

func tmuxWorktreePathCountsAsProgress(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return false
	}
	switch {
	case strings.HasPrefix(path, ".orquesta-runtime/"),
		strings.HasPrefix(path, "logs/"),
		strings.HasPrefix(path, ".codex/"),
		path == ".ssl-key.log":
		return false
	default:
		return true
	}
}

func launchEmbeddedTmuxMonitor(spec embeddedTmuxMonitorSpec, runDir string) (int, error) {
	specPath := filepath.Join(runDir, "tmux-monitor.json")
	specData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return 0, err
	}
	specData = append(specData, '\n')
	if err := os.WriteFile(specPath, specData, 0o600); err != nil {
		return 0, err
	}
	command, args, err := embeddedTmuxMonitorInvocation(specPath)
	if err != nil {
		return 0, err
	}
	stdoutPath := filepath.Join(runDir, "tmux-monitor.stdout")
	stderrPath := filepath.Join(runDir, "tmux-monitor.stderr")
	stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 0, err
	}
	defer stdoutFile.Close()
	stderrFile, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 0, err
	}
	defer stderrFile.Close()
	cmd := exec.Command(command, args...)
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	return cmd.Process.Pid, nil
}

func currentTMUXWorkerState(tmuxCommand, paneID string) (string, error) {
	captured, err := captureTMUXPane(tmuxCommand, paneID)
	if err != nil {
		return workerStatusRunning, nil
	}
	dismissed, err := dismissTMUXTrustPromptIfPresent(tmuxCommand, paneID, captured)
	if err != nil {
		return "", err
	}
	if dismissed {
		time.Sleep(120 * time.Millisecond)
		if updated, captureErr := captureTMUXPane(tmuxCommand, paneID); captureErr == nil {
			captured = updated
		} else {
			return workerStatusStarting, nil
		}
	}
	return tmuxClassifyPaneState(captured), nil
}

func waitForTMUXPaneReady(tmuxCommand, paneID string, timeout time.Duration) (bool, error) {
	if timeout <= 0 {
		timeout = tmuxReadyTimeout
	}
	startedAt := time.Now()
	delay := tmuxReadyInitialBackoff
	for time.Since(startedAt) < timeout {
		captured, err := captureTMUXPane(tmuxCommand, paneID)
		if err != nil {
			return false, err
		}
		dismissed, err := dismissTMUXTrustPromptIfPresent(tmuxCommand, paneID, captured)
		if err != nil {
			return false, err
		}
		state := tmuxClassifyPaneState(captured)
		if state == workerStatusBlockedAuth || state == workerStatusBlockedQuota {
			return false, nil
		}
		if !dismissed && state == workerStatusReady {
			return true, nil
		}
		remaining := timeout - time.Since(startedAt)
		if remaining <= 0 {
			break
		}
		sleep := delay
		if dismissed {
			sleep = 120 * time.Millisecond
			delay = tmuxReadyInitialBackoff
		} else if delay < tmuxReadyMaxBackoff {
			delay *= 2
			if delay > tmuxReadyMaxBackoff {
				delay = tmuxReadyMaxBackoff
			}
		}
		if sleep > remaining {
			sleep = remaining
		}
		time.Sleep(sleep)
	}
	return false, nil
}

func tmuxClassifyPaneState(captured string) string {
	switch {
	case tmuxPaneHasCodexAuthPrompt(captured):
		return workerStatusBlockedAuth
	case tmuxPaneHasUsageLimitPrompt(captured):
		return workerStatusBlockedQuota
	case tmuxPaneHasActiveTask(captured):
		return workerStatusRunning
	case tmuxPaneLooksReady(captured):
		return workerStatusReady
	default:
		return workerStatusStarting
	}
}

func captureTMUXPane(tmuxCommand, paneID string) (string, error) {
	cmd := exec.Command(strings.TrimSpace(tmuxCommand), "capture-pane", "-t", strings.TrimSpace(paneID), "-p")
	out, err := cmd.CombinedOutput()
	if err != nil {
		detalle := strings.TrimSpace(string(out))
		if detalle != "" {
			return "", fmt.Errorf("tmux capture-pane: %s", detalle)
		}
		return "", err
	}
	return string(out), nil
}

func dismissTMUXTrustPromptIfPresent(tmuxCommand, paneID, captured string) (bool, error) {
	if !tmuxPaneHasTrustPrompt(captured) {
		return false, nil
	}
	if err := sendTMUXKey(tmuxCommand, paneID, "C-m"); err != nil {
		return true, err
	}
	time.Sleep(120 * time.Millisecond)
	if err := sendTMUXKey(tmuxCommand, paneID, "C-m"); err != nil {
		return true, err
	}
	return true, nil
}

func sendTMUXKey(tmuxCommand, paneID, key string) error {
	cmd := exec.Command(strings.TrimSpace(tmuxCommand), "send-keys", "-t", strings.TrimSpace(paneID), strings.TrimSpace(key))
	out, err := cmd.CombinedOutput()
	if err != nil {
		detalle := strings.TrimSpace(string(out))
		if detalle != "" {
			return fmt.Errorf("tmux send-keys %s: %s", strings.TrimSpace(key), detalle)
		}
		return err
	}
	return nil
}

func tmuxPaneHasTrustPrompt(captured string) bool {
	captured = tmuxTailCapture(captured, 4096)
	lines := tmuxNormalizePaneLines(captured)
	if len(lines) == 0 {
		return false
	}
	tail := lines
	if len(tail) > 12 {
		tail = tail[len(tail)-12:]
	}
	hasQuestion := false
	hasChoices := false
	for _, line := range tail {
		trimmed := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(trimmed, "do you trust the contents of this directory?") {
			hasQuestion = true
		}
		if strings.Contains(trimmed, "yes, continue") ||
			strings.Contains(trimmed, "no, quit") ||
			strings.Contains(trimmed, "press enter to continue") {
			hasChoices = true
		}
	}
	return hasQuestion && hasChoices
}

func tmuxPaneLooksReady(captured string) bool {
	captured = tmuxTailCapture(captured, 4096)
	if tmuxPaneHasCodexAuthPrompt(captured) {
		return false
	}
	if tmuxPaneHasUsageLimitPrompt(captured) {
		return false
	}
	if tmuxPaneHasPendingSubmit(captured) {
		return false
	}
	lines := tmuxNormalizePaneLines(captured)
	if len(lines) == 0 || tmuxPaneIsBootstrapping(lines) {
		return false
	}
	tail := lines
	if len(tail) > 6 {
		tail = tail[len(tail)-6:]
	}
	last := strings.TrimSpace(tail[len(tail)-1])
	if tmuxLineLooksPrompt(last) {
		return true
	}
	for _, line := range tail {
		trimmed := strings.TrimSpace(line)
		if tmuxLineLooksPrompt(trimmed) {
			return true
		}
		if strings.Contains(strings.ToLower(trimmed), "how can i help") {
			return true
		}
	}
	return false
}

func tmuxPaneHasPendingSubmit(captured string) bool {
	normalized := strings.ToLower(tmuxSemanticTail(captured))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "press enter to send")
}

func tmuxPaneHasActiveTask(captured string) bool {
	captured = tmuxTailCapture(captured, 4096)
	lines := tmuxNormalizePaneLines(captured)
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	for _, line := range lines {
		normalized := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(normalized, "esc to interrupt") || strings.Contains(normalized, "background terminal running") {
			return true
		}
	}
	return false
}

func tmuxPaneHasCodexAuthPrompt(captured string) bool {
	normalized := tmuxSemanticTail(captured)
	if normalized == "" {
		return false
	}
	hasWelcome := strings.Contains(normalized, "welcome to codex")
	hasSignin := strings.Contains(normalized, "sign in with chatgpt") ||
		strings.Contains(normalized, "sign in with device code") ||
		strings.Contains(normalized, "provide your own api key")
	hasChoice := strings.Contains(normalized, "usage included with plus") ||
		strings.Contains(normalized, "press enter to continue")
	if hasSignin && hasChoice {
		return true
	}
	return hasWelcome && hasSignin
}

func tmuxPaneHasUsageLimitPrompt(captured string) bool {
	normalized := tmuxSemanticTail(captured)
	if normalized == "" {
		return false
	}
	hasUsageLimit := strings.Contains(normalized, "you've hit your usage limit") ||
		strings.Contains(normalized, "you have hit your usage limit") ||
		strings.Contains(normalized, "usage limit")
	hasRetryHint := strings.Contains(normalized, "send a request to your admin") ||
		strings.Contains(normalized, "try again at") ||
		strings.Contains(normalized, "try again later")
	return hasUsageLimit && hasRetryHint
}

func tmuxPaneIsBootstrapping(lines []string) bool {
	for _, line := range lines {
		normalized := strings.ToLower(strings.TrimSpace(line))
		switch {
		case strings.Contains(normalized, "loading"),
			strings.Contains(normalized, "initializing"),
			strings.Contains(normalized, "starting up"),
			strings.Contains(normalized, "model: loading"),
			strings.Contains(normalized, "connecting to"):
			return true
		}
	}
	return false
}

func tmuxLineLooksPrompt(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	if strings.HasPrefix(line, ">") {
		rest := strings.TrimSpace(strings.TrimPrefix(line, ">"))
		if rest != "" {
			if idx := strings.IndexByte(rest, '.'); idx > 0 {
				allDigits := true
				for _, r := range rest[:idx] {
					if r < '0' || r > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					return false
				}
			}
		}
		return true
	}
	if strings.HasPrefix(line, "\u203a") || strings.HasPrefix(line, "\u276f") {
		return true
	}
	return false
}

func tmuxNormalizePaneLines(captured string) []string {
	captured = sanitizeTMUXPaneText(captured)
	raw := strings.Split(strings.ReplaceAll(captured, "\r", ""), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func sanitizeTMUXPaneText(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	raw = ansiEscapePattern.ReplaceAllString(raw, "")
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 32 || r == 127:
			return -1
		default:
			return r
		}
	}, raw)
}

func tmuxSemanticTail(captured string) string {
	return tmuxNormalizeCapture(tmuxTailCapture(captured, tmuxSemanticDetectionTailBytes))
}

func tmuxTailCapture(captured string, maxBytes int) string {
	if maxBytes <= 0 || len(captured) <= maxBytes {
		return captured
	}
	start := len(captured) - maxBytes
	for start < len(captured) && start > 0 && (captured[start]&0xC0) == 0x80 {
		start++
	}
	return captured[start:]
}
