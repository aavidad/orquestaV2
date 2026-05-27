package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	cmd := exec.Command(exe, "run")
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
			if serverReadinessOKV0(state.Addr) {
				return state, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	if last.Addr != "" {
		return last, fmt.Errorf("readiness_timeout")
	}
	return last, fmt.Errorf("statefile_timeout")
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
	return response.StatusCode == http.StatusOK
}
