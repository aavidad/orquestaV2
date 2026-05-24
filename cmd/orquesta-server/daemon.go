package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func startServerCommandV0(stdout io.Writer, stderr io.Writer) int {
	config, err := serverConfigFromEnvV0()
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
	state, err := waitForStateHealthyV0(config, 15*time.Second)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server start pid=%d: %v\n", pid, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "orquesta-server activo pid=%d addr=%s\n", state.PID, state.Addr)
	return 0
}

func startDetachedRunV0(config orquestaserver.ConfigV0) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("executable_unavailable")
	}
	stdoutLog, err := openDaemonLogV0(config.StateDir, "stdout.log")
	if err != nil {
		return 0, err
	}
	defer stdoutLog.Close()
	stderrLog, err := openDaemonLogV0(config.StateDir, "stderr.log")
	if err != nil {
		return 0, err
	}
	defer stderrLog.Close()
	cmd := exec.Command(exe, "run")
	cmd.Env = serverEnvironmentWithDetailRailsDefaultV0(os.Environ())
	cmd.Stdin = nil
	cmd.Stdout = stdoutLog
	cmd.Stderr = stderrLog
	configureDetachedProcessV0(cmd)
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("daemon_start_failed")
	}
	return cmd.Process.Pid, cmd.Process.Release()
}

func openDaemonLogV0(dir string, name string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("daemon_log_dir_unavailable")
	}
	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("daemon_log_unavailable")
	}
	return file, nil
}

func waitForStateHealthyV0(
	config orquestaserver.ConfigV0,
	timeout time.Duration,
) (orquestaserver.StateV0, error) {
	deadline := time.Now().Add(timeout)
	var last orquestaserver.StateV0
	for time.Now().Before(deadline) {
		state, err := loadStateV0(config)
		if err == nil {
			last = state
			if healthOKV0(state.Addr) {
				return state, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	if last.Addr != "" {
		return last, fmt.Errorf("healthz_timeout")
	}
	return last, fmt.Errorf("statefile_timeout")
}

func healthOKV0(addr string) bool {
	client := http.Client{Timeout: 1 * time.Second}
	response, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK
}
