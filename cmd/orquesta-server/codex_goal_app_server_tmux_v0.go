package main

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
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	codexAppServerTmuxDirV0             = "goal-srv"
	codexAppServerTmuxMarkerFileV0      = "owner.json"
	codexAppServerTmuxEvidenceOwnedV0   = "orquesta-codex-goal-app-server-tmux-v0"
	codexAppServerTmuxDefaultTimeoutV0  = 3 * time.Second
	codexAppServerTmuxMinStartupV0      = 60 * time.Second
	codexAppServerTmuxSocketPollEveryV0 = 50 * time.Millisecond
)

type serverCodexAppServerTmuxBackendV0 struct {
	CommandPath       string
	PathEnv           string
	SocketPath        string
	SessionName       string
	HomeDir           string
	CodeHomeDir       string
	SourceCodeHomeDir string
	Timeout           time.Duration
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
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_dir_unavailable", Err: err}
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
		if preflight.ProbeV0(runCtx) == nil {
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
		return err
	}
	if err := backend.writeTmuxOwnerMarkerV0(); err != nil {
		return err
	}
	return backend.waitForTmuxSocketV0(runCtx, tmuxPath, preflight)
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
		if _, err := os.Stat(socketPath); err == nil && preflight.ProbeV0(ctx) == nil {
			return nil
		}
		if !nextSessionCheck.After(time.Now()) {
			hasSession, err := backend.tmuxHasSessionV0(ctx, tmuxPath)
			if err != nil {
				return err
			}
			if !hasSession {
				return codexAppServerCallErrorV0{
					Code: "codex_app_server_tmux_session_exited",
					Err:  errors.New("codex_app_server_tmux_session_exited"),
				}
			}
			nextSessionCheck = time.Now().Add(250 * time.Millisecond)
		}
		select {
		case <-ctx.Done():
			return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_socket_timeout", Err: ctx.Err()}
		case <-time.After(codexAppServerTmuxSocketPollEveryV0):
		}
	}
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

func (backend serverCodexAppServerTmuxBackendV0) prepareTmuxCodeHomeV0() error {
	codeHomeDir := strings.TrimSpace(backend.CodeHomeDir)
	if codeHomeDir == "" {
		return nil
	}
	if err := os.MkdirAll(codeHomeDir, 0o700); err != nil {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_unavailable",
			Err:  err,
		}
	}
	sourceDir := strings.TrimSpace(backend.SourceCodeHomeDir)
	if sourceDir == "" {
		return nil
	}
	for _, name := range []string{"auth.json", "config.toml"} {
		if err := copyCodexAppServerTmuxCodeHomeFileV0(sourceDir, codeHomeDir, name); err != nil {
			return err
		}
	}
	return nil
}

func copyCodexAppServerTmuxCodeHomeFileV0(sourceDir string, targetDir string, name string) error {
	sourcePath := filepath.Join(sourceDir, name)
	targetPath := filepath.Join(targetDir, name)
	if samePathCodexAppServerTmuxV0(sourcePath, targetPath) {
		return nil
	}
	info, err := os.Lstat(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_source_unavailable",
			Err:  err,
		}
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_source_unavailable",
			Err:  err,
		}
	}
	if err := os.WriteFile(targetPath, raw, 0o600); err != nil {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_unavailable",
			Err:  err,
		}
	}
	return nil
}

func samePathCodexAppServerTmuxV0(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(strings.TrimSpace(left))
	rightAbs, rightErr := filepath.Abs(strings.TrimSpace(right))
	if leftErr != nil || rightErr != nil {
		return filepath.Clean(left) == filepath.Clean(right)
	}
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxOwnerMarkerExistsV0() bool {
	path := backend.tmuxOwnerMarkerPathV0()
	if path == "" {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var marker codexAppServerTmuxOwnerMarkerV0
	if err := json.Unmarshal(raw, &marker); err != nil {
		return false
	}
	return strings.TrimSpace(marker.OwnerRef) == codexAppServerTmuxEvidenceOwnedV0 &&
		strings.TrimSpace(marker.SessionName) == strings.TrimSpace(backend.SessionName)
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
		SchemaVersion: "orquesta_codex_app_server_tmux_owner.v0",
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

func codexAppServerTmuxSocketPathV0(config orquestaserver.ConfigV0) (string, error) {
	runtimeDir := strings.TrimSpace(config.RuntimeWorkDir)
	if runtimeDir == "" {
		return "", codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_workdir_required", Err: errors.New("codex_app_server_tmux_runtime_workdir_required")}
	}
	return filepath.Join(runtimeDir, codexAppServerTmuxDirV0, codexAppServerTmuxSocketFileNameV0(config)), nil
}

func codexAppServerTmuxCodeHomePathV0(config orquestaserver.ConfigV0) (string, error) {
	runtimeDir := strings.TrimSpace(config.RuntimeWorkDir)
	if runtimeDir == "" {
		return "", codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_workdir_required", Err: errors.New("codex_app_server_tmux_runtime_workdir_required")}
	}
	return filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "codex-home"), nil
}

func codexAppServerTmuxSessionNameV0(config orquestaserver.ConfigV0) string {
	runtimeDir := filepath.Clean(strings.TrimSpace(config.RuntimeWorkDir))
	projectDir := filepath.Clean(strings.TrimSpace(config.ProjectWorkDir))
	sum := sha256.Sum256([]byte(strings.Join([]string{runtimeDir, projectDir}, "\x00")))
	return fmt.Sprintf("orquesta-goal-%x", sum[:8])
}

func codexAppServerTmuxSocketFileNameV0(config orquestaserver.ConfigV0) string {
	sessionName := codexAppServerTmuxSessionNameV0(config)
	hashPart := strings.TrimPrefix(sessionName, "orquesta-goal-")
	if hashPart == "" || hashPart == sessionName {
		sum := sha256.Sum256([]byte(sessionName))
		hashPart = fmt.Sprintf("%x", sum[:8])
	}
	return "g-" + hashPart + ".sock"
}

func codexAppServerTmuxStartupTimeoutV0(preflightTimeout time.Duration) time.Duration {
	if preflightTimeout < codexAppServerTmuxMinStartupV0 {
		return codexAppServerTmuxMinStartupV0
	}
	return preflightTimeout
}
