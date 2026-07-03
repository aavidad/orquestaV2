package orquestaruntimecodexappserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	codexAppServerTmuxDirV0             = "goal-srv"
	codexAppServerTmuxMarkerFileV0      = "owner.json"
	codexAppServerTmuxOwnerSchemaV0     = "orquesta_codex_app_server_tmux_owner.v0"
	codexAppServerTmuxEvidenceOwnedV0   = "orquesta-codex-goal-app-server-tmux-v0"
	codexAppServerTmuxDefaultTimeoutV0  = 3 * time.Second
	codexAppServerTmuxCleanupTimeoutV0  = 10 * time.Second
	codexAppServerTmuxMinStartupV0      = 60 * time.Second
	codexAppServerTmuxMaxSocketPathV0   = 107
	codexAppServerTmuxSocketPollEveryV0 = 50 * time.Millisecond
)

type serverCodexAppServerTmuxBackendV0 struct {
	CommandPath            string
	PathEnv                string
	SocketPath             string
	SessionName            string
	HomeDir                string
	CodeHomeDir            string
	RuntimeWorkDir         string
	ProjectWorkDir         string
	SourceCodeHomeDir      string
	Timeout                time.Duration
	ShutdownCleanupTimeout time.Duration
}

type codexAppServerTmuxOwnerMarkerV0 struct {
	SchemaVersion string `json:"schema_version"`
	OwnerRef      string `json:"owner_ref"`
	SessionName   string `json:"session_name"`
	SocketRef     string `json:"socket_ref"`
}

func (backend serverCodexAppServerTmuxBackendV0) EnsureV0(
	ctx context.Context,
	preflight serverCodexAppServerProbePortV0,
) error {
	timeout := backend.Timeout
	if timeout <= 0 {
		timeout = codexAppServerTmuxDefaultTimeoutV0
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	socketPath := strings.TrimSpace(backend.SocketPath)
	sessionName := strings.TrimSpace(backend.SessionName)
	if socketPath == "" || sessionName == "" {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_config_missing", Err: errors.New("codex_app_server_tmux_config_missing")}
	}
	if err := ensureCodexAppServerTmuxRuntimeDirV0(socketPath); err != nil {
		return err
	}
	tmuxPath, err := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	if err != nil {
		return err
	}
	hasSession, err := backend.tmuxHasSessionV0(runCtx, tmuxPath)
	if err != nil {
		return err
	}
	if hasSession {
		if !backend.tmuxOwnerMarkerExistsV0() {
			return codexAppServerCallErrorV0{
				Code: "codex_app_server_tmux_session_unowned",
				Err:  errors.New("codex_app_server_tmux_session_unowned"),
			}
		}
		if backend.ensureTmuxSocketPrivateV0() == nil && preflight.ProbeV0(runCtx) == nil {
			return nil
		}
		if err := backend.tmuxKillSessionV0(runCtx, tmuxPath); err != nil {
			return err
		}
	}
	_ = os.Remove(socketPath)
	if err := backend.prepareTmuxCodeHomeV0(); err != nil {
		return err
	}
	if err := backend.tmuxStartSessionV0(runCtx, tmuxPath); err != nil {
		backend.cleanupTmuxSessionAfterStartupFailureV0(tmuxPath)
		return err
	}
	if err := backend.writeTmuxOwnerMarkerV0(); err != nil {
		backend.cleanupTmuxSessionAfterStartupFailureV0(tmuxPath)
		return err
	}
	if err := backend.waitForTmuxSocketV0(runCtx, tmuxPath, preflight); err != nil {
		backend.cleanupTmuxSessionAfterStartupFailureV0(tmuxPath)
		return err
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxHasSessionV0(
	ctx context.Context,
	tmuxPath string,
) (bool, error) {
	output, err := backend.runTmuxCommandV0(ctx, tmuxPath, "has-session", "-t", strings.TrimSpace(backend.SessionName))
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, codexAppServerTmuxCommandErrorV0("codex_app_server_tmux_has_session_failed", output, err)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxKillSessionV0(
	ctx context.Context,
	tmuxPath string,
) error {
	output, err := backend.runTmuxCommandV0(ctx, tmuxPath, "kill-session", "-t", strings.TrimSpace(backend.SessionName))
	if err != nil {
		return codexAppServerTmuxCommandErrorV0("codex_app_server_tmux_kill_failed", output, err)
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxPanePIDV0(
	ctx context.Context,
	tmuxPath string,
) string {
	output, err := backend.runTmuxCommandV0(
		ctx,
		tmuxPath,
		"display-message",
		"-p",
		"-t",
		strings.TrimSpace(backend.SessionName),
		"#{pane_pid}",
	)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}

func (backend serverCodexAppServerTmuxBackendV0) waitTmuxPaneExitedV0(
	ctx context.Context,
	panePID string,
) error {
	pid, err := strconv.Atoi(strings.TrimSpace(panePID))
	if err != nil || pid <= 0 {
		return nil
	}
	ticker := time.NewTicker(codexAppServerTmuxSocketPollEveryV0)
	defer ticker.Stop()
	for {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return nil
		}
		select {
		case <-ctx.Done():
			return codexAppServerCallErrorV0{
				Code: "codex_app_server_tmux_pane_exit_timeout",
				Err:  ctx.Err(),
			}
		case <-ticker.C:
		}
	}
}

func (backend serverCodexAppServerTmuxBackendV0) ShutdownV0(ctx context.Context) error {
	return backend.shutdownTmuxSessionV0(ctx, false)
}

func (backend serverCodexAppServerTmuxBackendV0) ShutdownConfiguredSessionAfterStartupFailureV0(ctx context.Context) error {
	return backend.shutdownTmuxSessionV0(ctx, true)
}

func codexAppServerTmuxSocketPresentV0(socketPath string) bool {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return false
	}
	info, err := os.Lstat(socketPath)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func codexAppServerTmuxPIDAliveV0(panePID string) bool {
	pid, err := strconv.Atoi(strings.TrimSpace(panePID))
	if err != nil || pid <= 0 {
		return false
	}
	err = syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func (backend serverCodexAppServerTmuxBackendV0) shutdownTmuxSessionV0(ctx context.Context, allowConfiguredOrphan bool) error {
	return backend.shutdownTmuxSessionWithOptionsV0(ctx, allowConfiguredOrphan, false)
}

func (backend serverCodexAppServerTmuxBackendV0) shutdownTmuxSessionWithOptionsV0(
	ctx context.Context,
	allowConfiguredOrphan bool,
	continueAfterPaneExitTimeout bool,
) error {
	timeout := backend.Timeout
	if timeout <= 0 {
		timeout = codexAppServerTmuxDefaultTimeoutV0
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	owned := backend.tmuxOwnerMarkerExistsV0()
	if !owned && !allowConfiguredOrphan {
		return nil
	}
	if !owned && !backend.tmuxConfiguredOrphanCleanupAllowedV0() {
		return nil
	}
	tmuxPath, err := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	if err != nil {
		return err
	}
	hasSession, err := backend.tmuxHasSessionV0(runCtx, tmuxPath)
	if err != nil {
		return err
	}
	panePID := ""
	if hasSession {
		panePID = backend.tmuxPanePIDV0(runCtx, tmuxPath)
		if err := backend.tmuxKillSessionV0(runCtx, tmuxPath); err != nil {
			return err
		}
		if err := backend.waitTmuxPaneExitedV0(runCtx, panePID); err != nil {
			if !continueAfterPaneExitTimeout {
				return err
			}
		}
	}
	backend.stopCodexAppServerSocketProcessesV0(runCtx)
	backend.stopCodexAppServerRuntimeOwnedProcessesV0(runCtx)
	_ = os.Remove(strings.TrimSpace(backend.SocketPath))
	_ = os.Remove(backend.tmuxOwnerMarkerPathV0())
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) shutdownCleanupTimeoutV0() time.Duration {
	if backend.ShutdownCleanupTimeout > 0 {
		return backend.ShutdownCleanupTimeout
	}
	return codexAppServerTmuxCleanupTimeoutV0
}

func (backend serverCodexAppServerTmuxBackendV0) shutdownTmuxSessionForCleanupV0(ctx context.Context) error {
	cleanup := backend
	cleanup.Timeout = backend.shutdownCleanupTimeoutV0()
	return cleanup.shutdownTmuxSessionWithOptionsV0(ctx, true, true)
}

func (backend serverCodexAppServerTmuxBackendV0) stopCodexAppServerSocketProcessesV0(ctx context.Context) {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return
	}
	pids := codexAppServerTmuxSocketProcessPIDsV0(ctx, socketPath)
	for _, pid := range pids {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	backend.waitCodexAppServerSocketProcessesGoneV0(ctx, socketPath)
	killCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, pid := range codexAppServerTmuxSocketProcessPIDsV0(killCtx, socketPath) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	backend.waitCodexAppServerSocketProcessesGoneV0(killCtx, socketPath)
}

func (backend serverCodexAppServerTmuxBackendV0) stopCodexAppServerRuntimeOwnedProcessesV0(ctx context.Context) {
	runtimeDir := strings.TrimSpace(backend.RuntimeWorkDir)
	if runtimeDir == "" || !backend.tmuxConfiguredOrphanCleanupAllowedV0() {
		return
	}
	runtimeGoalDir := filepath.Join(filepath.Clean(runtimeDir), codexAppServerTmuxDirV0)
	pids := codexAppServerTmuxRuntimeOwnedProcessPIDsV0(ctx, runtimeGoalDir)
	for _, pid := range pids {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	backend.waitCodexAppServerRuntimeOwnedProcessesGoneV0(ctx, runtimeGoalDir)
	killCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, pid := range codexAppServerTmuxRuntimeOwnedProcessPIDsV0(killCtx, runtimeGoalDir) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	backend.waitCodexAppServerRuntimeOwnedProcessesGoneV0(killCtx, runtimeGoalDir)
}

func (backend serverCodexAppServerTmuxBackendV0) waitCodexAppServerRuntimeOwnedProcessesGoneV0(ctx context.Context, runtimeGoalDir string) {
	ticker := time.NewTicker(codexAppServerTmuxSocketPollEveryV0)
	defer ticker.Stop()
	for {
		if len(codexAppServerTmuxRuntimeOwnedProcessPIDsV0(ctx, runtimeGoalDir)) == 0 {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (backend serverCodexAppServerTmuxBackendV0) waitCodexAppServerSocketProcessesGoneV0(ctx context.Context, socketPath string) {
	ticker := time.NewTicker(codexAppServerTmuxSocketPollEveryV0)
	defer ticker.Stop()
	for {
		if len(codexAppServerTmuxSocketProcessPIDsV0(ctx, socketPath)) == 0 {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func codexAppServerTmuxSocketProcessPIDsV0(ctx context.Context, socketPath string) []int {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil
	}
	output, err := exec.CommandContext(ctx, "ps", "-eo", "pid=,args=").Output()
	if err != nil {
		return nil
	}
	out := []int{}
	currentPID := os.Getpid()
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 || pid == currentPID {
			continue
		}
		command := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), fields[0]))
		if codexAppServerTmuxCommandMatchesSocketV0(command, socketPath) {
			out = append(out, pid)
		}
	}
	return out
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxConfiguredOrphanCleanupAllowedV0() bool {
	sessionName := strings.TrimSpace(backend.SessionName)
	if !strings.HasPrefix(sessionName, "orquesta-goal-") ||
		len(strings.TrimPrefix(sessionName, "orquesta-goal-")) < 16 {
		return false
	}
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" || filepath.Base(socketPath) == "." || filepath.Dir(socketPath) == "." {
		return false
	}
	if filepath.Ext(socketPath) != ".sock" && filepath.Base(socketPath) != "s.sock" {
		return false
	}
	return true
}

func (backend serverCodexAppServerTmuxBackendV0) cleanupTmuxSessionAfterStartupFailureV0(
	tmuxPath string,
) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), codexAppServerTmuxDefaultTimeoutV0)
	defer cancel()

	if hasSession, err := backend.tmuxHasSessionV0(cleanupCtx, tmuxPath); err == nil && hasSession {
		_ = backend.tmuxKillSessionV0(cleanupCtx, tmuxPath)
	}
	_ = os.Remove(strings.TrimSpace(backend.SocketPath))
	_ = os.Remove(backend.tmuxOwnerMarkerPathV0())
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxStartSessionV0(
	ctx context.Context,
	tmuxPath string,
) error {
	commandPath := strings.TrimSpace(backend.CommandPath)
	if commandPath == "" {
		commandPath = codexCommandPathV0()
	}
	socketURL := "unix://" + strings.TrimSpace(backend.SocketPath)
	pathEnv := strings.TrimSpace(backend.PathEnv)
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	envAssignments := []string{
		"PATH=" + shellQuoteCodexAppServerTmuxV0(pathEnv),
	}
	if homeDir := strings.TrimSpace(backend.HomeDir); homeDir != "" {
		envAssignments = append(envAssignments, "HOME="+shellQuoteCodexAppServerTmuxV0(homeDir))
	}
	if codeHomeDir := strings.TrimSpace(backend.CodeHomeDir); codeHomeDir != "" {
		envAssignments = append(envAssignments, "CODEX_HOME="+shellQuoteCodexAppServerTmuxV0(codeHomeDir))
	}
	for _, key := range []string{
		"CODEX_MANAGED_BY_NPM",
		"CODEX_MANAGED_PACKAGE_ROOT",
		"CODEX_CI",
		"XDG_RUNTIME_DIR",
	} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			envAssignments = append(envAssignments, key+"="+shellQuoteCodexAppServerTmuxV0(value))
		}
	}
	logPath := backend.tmuxLogPathV0()
	_ = os.Remove(logPath)
	shellCommand := strings.Join(envAssignments, " ") +
		" exec " + shellQuoteCodexAppServerTmuxV0(commandPath) +
		" app-server --listen " + shellQuoteCodexAppServerTmuxV0(socketURL) +
		" >> " + shellQuoteCodexAppServerTmuxV0(logPath) + " 2>&1"
	output, err := backend.runTmuxCommandV0(
		ctx,
		tmuxPath,
		"new-session",
		"-d",
		"-s",
		strings.TrimSpace(backend.SessionName),
		shellCommand,
	)
	if err != nil {
		return codexAppServerTmuxCommandErrorV0("codex_app_server_tmux_start_failed", output, err)
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) runTmuxCommandV0(
	ctx context.Context,
	tmuxPath string,
	args ...string,
) (string, error) {
	cmd := exec.CommandContext(ctx, tmuxPath, args...)
	if strings.TrimSpace(backend.PathEnv) != "" {
		cmd.Env = append(os.Environ(), "PATH="+strings.TrimSpace(backend.PathEnv))
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if ctx.Err() != nil {
		return output.String(), codexAppServerCallErrorV0{Code: "codex_app_server_tmux_timeout", Err: ctx.Err()}
	}
	return output.String(), err
}

func (backend serverCodexAppServerTmuxBackendV0) waitForTmuxSocketV0(
	ctx context.Context,
	tmuxPath string,
	preflight serverCodexAppServerProbePortV0,
) error {
	socketPath := strings.TrimSpace(backend.SocketPath)
	nextSessionCheck := time.Now()
	for {
		if ctx.Err() != nil {
			return backend.tmuxStartupFailureV0("codex_app_server_tmux_socket_timeout", ctx.Err())
		}
		if _, err := os.Stat(socketPath); err == nil &&
			backend.ensureTmuxSocketPrivateV0() == nil &&
			preflight.ProbeV0(ctx) == nil {
			return nil
		}
		if !nextSessionCheck.After(time.Now()) {
			hasSession, err := backend.tmuxHasSessionV0(ctx, tmuxPath)
			if ctx.Err() != nil {
				return backend.tmuxStartupFailureV0("codex_app_server_tmux_socket_timeout", ctx.Err())
			}
			if err != nil {
				return err
			}
			if !hasSession {
				return backend.tmuxStartupFailureV0(
					"codex_app_server_tmux_session_exited",
					errors.New("codex_app_server_tmux_session_exited"),
				)
			}
			nextSessionCheck = time.Now().Add(250 * time.Millisecond)
		}
		select {
		case <-ctx.Done():
			return backend.tmuxStartupFailureV0("codex_app_server_tmux_socket_timeout", ctx.Err())
		case <-time.After(codexAppServerTmuxSocketPollEveryV0):
		}
	}
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxStartupFailureV0(
	fallback string,
	err error,
) error {
	if code := codexAppServerIssueCodeFromLogFileV0(backend.tmuxLogPathV0()); code != "" {
		return codexAppServerCallErrorV0{Code: code, Err: err}
	}
	code := strings.TrimSpace(fallback)
	if code == "" {
		code = "codex_app_server_tmux_failed"
	}
	return codexAppServerCallErrorV0{Code: code, Err: err}
}

func ensureCodexAppServerTmuxRuntimeDirV0(socketPath string) error {
	dir := filepath.Dir(strings.TrimSpace(socketPath))
	if dir == "." || dir == "" {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_runtime_dir_unavailable",
			Err:  errors.New("codex_app_server_tmux_runtime_dir_unavailable"),
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_dir_unavailable", Err: err}
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_dir_unavailable", Err: err}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_dir_unavailable", Err: errors.New("codex_app_server_tmux_runtime_dir_symlink")}
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_dir_unavailable", Err: err}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) ensureTmuxSocketPrivateV0() error {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_config_missing", Err: errors.New("codex_app_server_tmux_config_missing")}
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_socket_permissions_unavailable", Err: err}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxOwnerMarkerPathV0() string {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(socketPath), codexAppServerTmuxMarkerFileV0)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxLogPathV0() string {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return ""
	}
	sessionName := strings.TrimSpace(backend.SessionName)
	if sessionName == "" {
		sessionName = "codex-app-server"
	}
	return filepath.Join(filepath.Dir(socketPath), sessionName+".log")
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxOwnerMarkerExistsV0() bool {
	_, ok := backend.readTmuxOwnerMarkerV0()
	return ok
}

func (backend serverCodexAppServerTmuxBackendV0) readTmuxOwnerMarkerV0() (codexAppServerTmuxOwnerMarkerV0, bool) {
	path := backend.tmuxOwnerMarkerPathV0()
	if path == "" {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	marker, ok := readCodexAppServerTmuxOwnerMarkerPathV0(path)
	if !ok {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	if strings.TrimSpace(marker.SchemaVersion) != codexAppServerTmuxOwnerSchemaV0 ||
		strings.TrimSpace(marker.OwnerRef) != codexAppServerTmuxEvidenceOwnedV0 ||
		strings.TrimSpace(marker.SessionName) != strings.TrimSpace(backend.SessionName) {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	return marker, true
}

func readCodexAppServerTmuxOwnerMarkerPathV0(path string) (codexAppServerTmuxOwnerMarkerV0, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	info, err := os.Lstat(path)
	if err != nil || info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	var marker codexAppServerTmuxOwnerMarkerV0
	if err := json.Unmarshal(raw, &marker); err != nil {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	if strings.TrimSpace(marker.SchemaVersion) != codexAppServerTmuxOwnerSchemaV0 ||
		strings.TrimSpace(marker.OwnerRef) != codexAppServerTmuxEvidenceOwnedV0 {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	return marker, true
}

func shellQuoteCodexAppServerTmuxV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func (backend serverCodexAppServerTmuxBackendV0) writeTmuxOwnerMarkerV0() error {
	path := backend.tmuxOwnerMarkerPathV0()
	if path == "" {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_config_missing", Err: errors.New("codex_app_server_tmux_config_missing")}
	}
	marker := codexAppServerTmuxOwnerMarkerV0{
		SchemaVersion: codexAppServerTmuxOwnerSchemaV0,
		OwnerRef:      codexAppServerTmuxEvidenceOwnedV0,
		SessionName:   strings.TrimSpace(backend.SessionName),
		SocketRef:     "socket-ref-codex-goal-app-server-tmux",
	}
	raw, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_marker_unavailable", Err: err}
	}
	return nil
}

func codexAppServerTmuxCommandPathV0(pathEnv string) (string, error) {
	if strings.TrimSpace(pathEnv) != "" {
		for _, dir := range filepath.SplitList(pathEnv) {
			candidate := filepath.Join(dir, "tmux")
			info, err := os.Stat(candidate)
			if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
				return candidate, nil
			}
		}
	}
	path, err := exec.LookPath("tmux")
	if err != nil {
		return "", codexAppServerCallErrorV0{Code: "codex_app_server_tmux_command_missing", Err: err}
	}
	return path, nil
}

func codexAppServerTmuxCommandErrorV0(fallback string, output string, err error) error {
	if callErr, ok := err.(codexAppServerCallErrorV0); ok {
		return callErr
	}
	code := codexAppServerIssueCodeFromCommandFailureV0(output, err)
	if code == "codex_app_server_call_failed" {
		code = strings.TrimSpace(fallback)
	}
	if code == "" {
		code = "codex_app_server_tmux_failed"
	}
	return codexAppServerCallErrorV0{Code: code, Err: err}
}

func codexAppServerTmuxSocketPathV0(config ConfigV0) (string, error) {
	runtimeDir := strings.TrimSpace(config.RuntimeWorkDir)
	if runtimeDir == "" {
		return "", codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_workdir_required", Err: errors.New("codex_app_server_tmux_runtime_workdir_required")}
	}
	socketPath := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, codexAppServerTmuxSocketFileNameV0(config))
	if len(socketPath) <= codexAppServerTmuxMaxSocketPathV0 {
		return socketPath, nil
	}
	shortPath := codexAppServerTmuxShortSocketPathV0(config)
	if len(shortPath) > codexAppServerTmuxMaxSocketPathV0 {
		return "", codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_socket_path_too_long",
			Err:  fmt.Errorf("codex_app_server_tmux_socket_path_too_long:%d", len(shortPath)),
		}
	}
	return shortPath, nil
}

func codexAppServerTmuxShortSocketPathV0(config ConfigV0) string {
	tempDir := strings.TrimSpace(os.TempDir())
	if tempDir == "" {
		tempDir = "/tmp"
	}
	hashPart := codexAppServerTmuxSocketHashPartV0(config)
	shortPath := filepath.Join(tempDir, codexAppServerTmuxShortSocketDirNameV0(hashPart), "s.sock")
	if len(shortPath) <= codexAppServerTmuxMaxSocketPathV0 {
		return shortPath
	}
	return filepath.Join("/tmp", codexAppServerTmuxShortSocketDirNameV0(hashPart), "s.sock")
}

func codexAppServerTmuxShortSocketDirNameV0(hashPart string) string {
	return fmt.Sprintf("oq-gsrv-%d-%s", os.Getuid(), hashPart)
}

func codexAppServerTmuxCodeHomePathV0(config ConfigV0) (string, error) {
	runtimeDir := strings.TrimSpace(config.RuntimeWorkDir)
	if runtimeDir == "" {
		return "", codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_workdir_required", Err: errors.New("codex_app_server_tmux_runtime_workdir_required")}
	}
	return filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "codex-home"), nil
}

func codexAppServerTmuxSessionNameV0(config ConfigV0) string {
	runtimeDir := filepath.Clean(strings.TrimSpace(config.RuntimeWorkDir))
	projectDir := filepath.Clean(strings.TrimSpace(config.ProjectWorkDir))
	sum := sha256.Sum256([]byte(strings.Join([]string{runtimeDir, projectDir}, "\x00")))
	return fmt.Sprintf("orquesta-goal-%x", sum[:8])
}

func codexAppServerTmuxSocketFileNameV0(config ConfigV0) string {
	return "g-" + codexAppServerTmuxSocketHashPartV0(config) + ".sock"
}

func codexAppServerTmuxSocketHashPartV0(config ConfigV0) string {
	sessionName := codexAppServerTmuxSessionNameV0(config)
	hashPart := strings.TrimPrefix(sessionName, "orquesta-goal-")
	if hashPart == "" || hashPart == sessionName {
		sum := sha256.Sum256([]byte(sessionName))
		hashPart = fmt.Sprintf("%x", sum[:8])
	}
	return hashPart
}

func codexAppServerTmuxStartupTimeoutV0(preflightTimeout time.Duration) time.Duration {
	if preflightTimeout < codexAppServerTmuxMinStartupV0 {
		return codexAppServerTmuxMinStartupV0
	}
	return preflightTimeout
}
