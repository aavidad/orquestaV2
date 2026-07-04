package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const serverDaemonConfigSnapshotDirV0 = "config-snapshots"

const serverStartReadinessTimeoutV0 = 45 * time.Second

func startServerCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	options, err := parseServerCommandConfigOptionsV0("start", args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server start: %v\n", err)
		return 2
	}
	config, err := serverConfigFromEnvWithProjectConfigPathV0(options.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server start: %v\n", err)
		return 1
	}
	if err := validateCodexCommandAvailableV0(); err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server start: %v\n", err)
		return 1
	}
	pid, err := startDetachedRunV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server start: %v\n", err)
		return 1
	}
	state, err := waitForStateHealthyV0(config, serverStartReadinessTimeoutV0)
	if err != nil {
		cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, pid)
		_, _ = fmt.Fprintf(stderr, "orquesta-server start pid=%d: %v\n", pid, err)
		return 1
	}
	if err := writeCommandTextOutputV0(stdout, fmt.Sprintf("orquesta-server activo pid=%d addr=%s\n", state.PID, state.Addr)); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "start", "stdout", "text_write", err)
	}
	return 0
}

func startDetachedRunV0(config orquestaserver.ConfigV0) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("executable_unavailable")
	}
	stdoutLog, stderrLog, err := openDaemonOutputFilesV0(config)
	if err != nil {
		return 0, err
	}
	defer stdoutLog.Close()
	defer stderrLog.Close()
	runArgs, err := serverDaemonRunArgsV0(config)
	if err != nil {
		return 0, err
	}
	cmd := exec.Command(exe, runArgs...)
	cmd.Env = serverDaemonStartEnvironmentV0(os.Environ(), config)
	cmd.Stdin = nil
	cmd.Stdout = stdoutLog
	cmd.Stderr = stderrLog
	configureDetachedProcessV0(cmd)
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("daemon_start_failed")
	}
	return cmd.Process.Pid, cmd.Process.Release()
}

func serverDaemonRunArgsV0(config orquestaserver.ConfigV0) ([]string, error) {
	snapshotPath, err := writeServerDaemonConfigSnapshotV0(config)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(snapshotPath) == "" {
		return []string{"run"}, nil
	}
	return []string{"run", "--config", snapshotPath}, nil
}

func writeServerDaemonConfigSnapshotV0(config orquestaserver.ConfigV0) (string, error) {
	config = orquestaserver.NormalizeConfigV0(config)
	sourcePath := strings.TrimSpace(config.ProjectConfigFilePath)
	if sourcePath == "" {
		return "", nil
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("%s: read", configFileInvalidPublicCodeV0)
	}
	if _, _, err := loadServerProjectConfigPathV0(sourcePath); err != nil {
		return "", err
	}
	stateDir := strings.TrimSpace(config.StateDir)
	if stateDir == "" {
		return "", fmt.Errorf("server_state_dir_unavailable")
	}
	snapshotDir := filepath.Join(stateDir, serverDaemonConfigSnapshotDirV0)
	if err := os.MkdirAll(snapshotDir, 0o700); err != nil {
		return "", fmt.Errorf("daemon_config_snapshot_unavailable")
	}
	snapshotPath := filepath.Join(snapshotDir, serverProjectConfigFileNameV0)
	tmpPath := snapshotPath + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o600); err != nil {
		return "", fmt.Errorf("daemon_config_snapshot_write_failed")
	}
	if err := os.Rename(tmpPath, snapshotPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("daemon_config_snapshot_write_failed")
	}
	return snapshotPath, nil
}

var (
	serverReadinessPollEveryV0   = 200 * time.Millisecond
	serverReadinessStableDelayV0 = 250 * time.Millisecond
)

func waitForStateHealthyV0(
	config orquestaserver.ConfigV0,
	timeout time.Duration,
) (orquestaserver.StateV0, error) {
	deadline := time.Now().Add(timeout)
	var last orquestaserver.StateV0
	serverExitedAfterReadiness := false
	for time.Now().Before(deadline) {
		state, err := loadStateV0(config)
		if err == nil {
			loadedState := state
			if reconciled, ok := reconcileStatefileSnapshotLivenessV0(config, state); ok {
				state = reconciled
				serverExitedAfterReadiness = serverExitedAfterReadiness || serverStateWasStartupReadyV0(loadedState)
			}
			last = state
			if serverStateSnapshotReadyForReadinessV0(state) && serverReadinessOKV0(state.Addr) {
				stableState, stableReconciled, ok := waitForStableReadinessV0(config, state, deadline)
				if stableState.Addr != "" {
					last = stableState
				}
				serverExitedAfterReadiness = serverExitedAfterReadiness || stableReconciled
				if ok {
					return stableState, nil
				}
			}
		}
		sleepUntilDeadlineV0(serverReadinessPollEveryV0, deadline)
	}
	if serverExitedAfterReadiness && serverStateIsProcessStaleV0(last) {
		cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, last.PID)
		return last, fmt.Errorf("server_exited_after_readiness")
	}
	if last.Addr != "" {
		return last, fmt.Errorf("readiness_timeout")
	}
	return last, fmt.Errorf("statefile_timeout")
}

func waitForStableReadinessV0(
	config orquestaserver.ConfigV0,
	readyState orquestaserver.StateV0,
	deadline time.Time,
) (orquestaserver.StateV0, bool, bool) {
	if !sleepUntilDeadlineV0(serverReadinessStableDelayV0, deadline) {
		return readyState, false, false
	}
	stableState, err := loadStateV0(config)
	if err != nil {
		return readyState, false, false
	}
	loadedState := stableState
	if reconciled, ok := reconcileStatefileSnapshotLivenessV0(config, stableState); ok {
		return reconciled, serverStateWasStartupReadyV0(loadedState), false
	}
	if !serverStateSnapshotReadyForReadinessV0(stableState) {
		return stableState, false, false
	}
	if !sameStartupDaemonIdentityV0(readyState, stableState) {
		return stableState, false, false
	}
	if !serverReadinessOKV0(stableState.Addr) {
		return stableState, false, false
	}
	return stableState, false, true
}

func sleepUntilDeadlineV0(delay time.Duration, deadline time.Time) bool {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return false
	}
	if delay <= 0 {
		return true
	}
	if delay > remaining {
		delay = remaining
	}
	time.Sleep(delay)
	return true
}

func sameStartupDaemonIdentityV0(a orquestaserver.StateV0, b orquestaserver.StateV0) bool {
	return a.Addr == b.Addr &&
		a.PID == b.PID &&
		a.ProcessRef == b.ProcessRef &&
		a.DaemonEpochRef == b.DaemonEpochRef &&
		a.StartedAt == b.StartedAt &&
		a.RuntimeIdentity.BinarySHA256 == b.RuntimeIdentity.BinarySHA256 &&
		a.RuntimeIdentity.BuildRef == b.RuntimeIdentity.BuildRef &&
		a.RuntimeIdentity.CommitRef == b.RuntimeIdentity.CommitRef &&
		a.RuntimeIdentity.StartedAt == b.RuntimeIdentity.StartedAt
}

func serverStateWasStartupReadyV0(state orquestaserver.StateV0) bool {
	return state.StartupReady ||
		strings.TrimSpace(state.StartupStatus) == orquestaserver.StartupCheckStatusReadyV0
}

func serverStateSnapshotReadyForReadinessV0(state orquestaserver.StateV0) bool {
	return strings.TrimSpace(state.Status) == "running" && serverStateWasStartupReadyV0(state)
}

func serverStateIsProcessStaleV0(state orquestaserver.StateV0) bool {
	return strings.TrimSpace(state.Status) == "stale" &&
		strings.TrimSpace(state.StartupStatus) == orquestaserver.ServerProcessStaleReasonCodeV0
}

func serverReadinessOKV0(addr string) bool {
	baseURL, err := commandRESTBaseURLFromAddrV0(addr)
	if err != nil {
		return false
	}
	target, err := commandRESTEndpointURLV0(baseURL, orquestaserver.ServerReadinessEndpointV0)
	if err != nil {
		return false
	}
	client := commandHTTPClientWithRedirectPolicyV0(time.Second, baseURL)
	response, err := client.Get(target)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true
	}
	if response.StatusCode != http.StatusServiceUnavailable {
		return false
	}
	var readiness orquestaserver.ServerReadinessV0
	if err := json.NewDecoder(response.Body).Decode(&readiness); err != nil {
		return false
	}
	return readiness.StartupReady &&
		strings.TrimSpace(readiness.Status) == "running" &&
		strings.TrimSpace(readiness.AvailabilityStatus) == "running"
}
