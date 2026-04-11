package controlruntime

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type embeddedBrokerSpec struct {
	Command             string            `json:"command"`
	Args                []string          `json:"args"`
	Env                 map[string]string `json:"env"`
	WorkingDir          string            `json:"working_dir"`
	StdinPath           string            `json:"stdin_path"`
	LogPath             string            `json:"log_path"`
	ReadyPath           string            `json:"ready_path"`
	Agent               string            `json:"agent"`
	Project             string            `json:"project"`
	ExternalSessionID   string            `json:"external_session_id"`
	MailboxDeliveryMode string            `json:"mailbox_delivery_mode"`
	CanSendInput        bool              `json:"can_send_input"`
	StatusPath          string            `json:"status_path"`
	HeartbeatPath       string            `json:"heartbeat_path"`
}

type embeddedBrokerReady struct {
	ChildPID  int    `json:"child_pid"`
	StartedAt string `json:"started_at"`
}

const embeddedBrokerHeartbeatInterval = 15 * time.Second

type embeddedBrokerRuntimeState struct {
	mu           sync.RWMutex
	startedAt    time.Time
	lastOutputAt time.Time
	alive        bool
	exitCode     *int
	exitError    string
}

func newEmbeddedBrokerRuntimeState(startedAt time.Time) *embeddedBrokerRuntimeState {
	return &embeddedBrokerRuntimeState{
		startedAt: startedAt.UTC(),
		alive:     true,
	}
}

func (s *embeddedBrokerRuntimeState) markOutput(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastOutputAt = now.UTC()
}

func (s *embeddedBrokerRuntimeState) markExit(exitCode *int, exitError string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alive = false
	if exitCode != nil {
		code := *exitCode
		s.exitCode = &code
	}
	s.exitError = strings.TrimSpace(exitError)
}

func (s *embeddedBrokerRuntimeState) snapshot() embeddedBrokerRuntimeSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := embeddedBrokerRuntimeSnapshot{
		StartedAt: s.startedAt.UTC(),
		Alive:     s.alive,
		ExitError: s.exitError,
	}
	if !s.lastOutputAt.IsZero() {
		snap.LastOutputAt = s.lastOutputAt.UTC()
	}
	if s.exitCode != nil {
		code := *s.exitCode
		snap.ExitCode = &code
	}
	return snap
}

type embeddedBrokerRuntimeSnapshot struct {
	StartedAt    time.Time
	LastOutputAt time.Time
	Alive        bool
	ExitCode     *int
	ExitError    string
}

type embeddedBrokerLogWriter struct {
	w          io.Writer
	markOutput func(time.Time)
}

func (w *embeddedBrokerLogWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	if n > 0 && w.markOutput != nil {
		w.markOutput(time.Now().UTC())
	}
	return n, err
}

func MaybeRunEmbeddedBroker(args []string) bool {
	if len(args) == 0 || strings.TrimSpace(args[0]) != "__pty_broker" {
		return false
	}
	if err := runEmbeddedBroker(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
	return true
}

func runEmbeddedBroker(args []string) error {
	fs := flag.NewFlagSet("__pty_broker", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	specPath := fs.String("spec", "", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*specPath) == "" {
		return fmt.Errorf("broker pty sin --spec")
	}
	data, err := os.ReadFile(strings.TrimSpace(*specPath))
	if err != nil {
		return err
	}
	var spec embeddedBrokerSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return err
	}
	if strings.TrimSpace(spec.Command) == "" {
		return fmt.Errorf("broker pty sin command")
	}
	if strings.TrimSpace(spec.StdinPath) == "" || strings.TrimSpace(spec.LogPath) == "" || strings.TrimSpace(spec.ReadyPath) == "" {
		return fmt.Errorf("broker pty sin rutas obligatorias")
	}
	if err := os.MkdirAll(filepath.Dir(spec.LogPath), 0o700); err != nil {
		return err
	}
	logFile, err := os.OpenFile(spec.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()
	_, _ = fmt.Fprintf(logFile, "Orquesta PTY broker started on %s [COMMAND=%q]\n", time.Now().UTC().Format(time.RFC3339Nano), renderBrokerCommand(spec.Command, spec.Args))

	cmd := exec.Command(spec.Command, spec.Args...)
	cmd.Dir = strings.TrimSpace(spec.WorkingDir)
	env := append([]string(nil), os.Environ()...)
	for k, v := range spec.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	runtimeState := newEmbeddedBrokerRuntimeState(time.Now().UTC())

	ptmx, err := pty.Start(cmd)
	if err != nil {
		runtimeState.markExit(nil, errorString(err))
		_ = writeEmbeddedBrokerArtifacts(spec, 0, workerStatusFailed, runtimeState.snapshot())
		return err
	}
	defer func() { _ = ptmx.Close() }()

	ready := embeddedBrokerReady{
		ChildPID:  cmd.Process.Pid,
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}

	stopCh := make(chan os.Signal, 4)
	signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(stopCh)
	go func() {
		for sig := range stopCh {
			if cmd.Process == nil {
				continue
			}
			_ = cmd.Process.Signal(sig)
		}
	}()

	copyErrCh := make(chan error, 2)
	stopHeartbeat := make(chan struct{})
	defer close(stopHeartbeat)
	stopInput := make(chan struct{})
	var stopInputOnce sync.Once
	stopInputClose := func() {
		stopInputOnce.Do(func() {
			close(stopInput)
		})
	}
	defer stopInputClose()
	go func() {
		ticker := time.NewTicker(embeddedBrokerHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stopHeartbeat:
				return
			case <-ticker.C:
				_ = writeEmbeddedBrokerArtifacts(spec, ready.ChildPID, embeddedBrokerLifecycleState(runtimeState.snapshot()), runtimeState.snapshot())
			}
		}
	}()
	go func() {
		_, err := io.Copy(&embeddedBrokerLogWriter{w: logFile, markOutput: runtimeState.markOutput}, ptmx)
		copyErrCh <- err
	}()
	go func() {
		copyErrCh <- bridgeEmbeddedBrokerInput(spec.StdinPath, ptmx, stopInput)
	}()

	if err := writeEmbeddedBrokerArtifacts(spec, ready.ChildPID, workerStatusStarting, runtimeState.snapshot()); err != nil {
		return err
	}
	readyData, _ := json.MarshalIndent(ready, "", "  ")
	readyData = append(readyData, '\n')
	if err := os.WriteFile(spec.ReadyPath, readyData, 0o600); err != nil {
		return err
	}

	waitErr := cmd.Wait()
	stopInputClose()
	_ = ptmx.Close()
	select {
	case <-time.After(250 * time.Millisecond):
	case <-copyErrCh:
	}

	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	_, _ = fmt.Fprintf(logFile, "\nOrquesta PTY broker finished on %s [EXIT_CODE=%d]\n", time.Now().UTC().Format(time.RFC3339Nano), exitCode)
	var exitCodePtr *int
	if exitCode >= 0 {
		exitCodeCopy := exitCode
		exitCodePtr = &exitCodeCopy
	}
	runtimeState.markExit(exitCodePtr, errorString(waitErr))
	_ = writeEmbeddedBrokerArtifacts(spec, ready.ChildPID, embeddedBrokerLifecycleState(runtimeState.snapshot()), runtimeState.snapshot())
	if waitErr != nil {
		return waitErr
	}
	return nil
}

func bridgeEmbeddedBrokerInput(stdinPath string, ptmx *os.File, stop <-chan struct{}) error {
	fd, err := syscall.Open(stdinPath, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	pipe := os.NewFile(uintptr(fd), stdinPath)
	if pipe == nil {
		_ = syscall.Close(fd)
		return fmt.Errorf("stdin fifo no disponible")
	}
	defer pipe.Close()

	buf := make([]byte, 4096)
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		n, err := pipe.Read(buf)
		if n > 0 {
			if _, writeErr := ptmx.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			continue
		}
		if err != nil {
			if err == io.EOF || err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
				time.Sleep(25 * time.Millisecond)
				continue
			}
			return err
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func renderBrokerCommand(command string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, strings.TrimSpace(command))
	for _, arg := range args {
		parts = append(parts, strings.TrimSpace(arg))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func writeEmbeddedBrokerArtifacts(spec embeddedBrokerSpec, childPID int, state string, snap embeddedBrokerRuntimeSnapshot) error {
	status := workerStatus{
		State:               strings.TrimSpace(state),
		UpdatedAt:           time.Now().UTC().Format(time.RFC3339Nano),
		Alive:               snap.Alive,
		ChildPID:            childPID,
		Agent:               strings.TrimSpace(spec.Agent),
		Project:             strings.TrimSpace(spec.Project),
		WorkingDir:          strings.TrimSpace(spec.WorkingDir),
		LogPath:             strings.TrimSpace(spec.LogPath),
		ExternalSessionID:   strings.TrimSpace(spec.ExternalSessionID),
		MailboxDeliveryMode: strings.TrimSpace(spec.MailboxDeliveryMode),
		ExitError:           strings.TrimSpace(snap.ExitError),
	}
	if snap.ExitCode != nil {
		code := *snap.ExitCode
		status.ExitCode = &code
	}
	if !snap.LastOutputAt.IsZero() {
		status.LastOutputAt = snap.LastOutputAt.UTC().Format(time.RFC3339Nano)
	}
	if err := writeWorkerStatusFile(spec.StatusPath, status); err != nil {
		return err
	}
	heartbeat := workerHeartbeat{
		Alive:             snap.Alive,
		HeartbeatAt:       time.Now().UTC().Format(time.RFC3339Nano),
		StartedAt:         snap.StartedAt.UTC().Format(time.RFC3339Nano),
		ChildPID:          childPID,
		Agent:             strings.TrimSpace(spec.Agent),
		Project:           strings.TrimSpace(spec.Project),
		ExternalSessionID: strings.TrimSpace(spec.ExternalSessionID),
		ExitError:         strings.TrimSpace(snap.ExitError),
	}
	if snap.ExitCode != nil {
		code := *snap.ExitCode
		heartbeat.ExitCode = &code
	}
	if !snap.LastOutputAt.IsZero() {
		heartbeat.LastOutputAt = snap.LastOutputAt.UTC().Format(time.RFC3339Nano)
	}
	return writeWorkerHeartbeatFile(spec.HeartbeatPath, heartbeat)
}

func embeddedBrokerLifecycleState(snap embeddedBrokerRuntimeSnapshot) string {
	if snap.Alive {
		if snap.LastOutputAt.IsZero() {
			return workerStatusStarting
		}
		return workerStatusRunning
	}
	if snap.ExitCode != nil && *snap.ExitCode == 0 {
		return workerStatusStopped
	}
	return workerStatusFailed
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
