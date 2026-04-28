package controlruntime

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultTMUXCommandTimeout = 2 * time.Second

func tmuxCommandTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("ORQUESTA_TMUX_COMMAND_TIMEOUT_MS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return time.Duration(n) * time.Millisecond
		}
	}
	return defaultTMUXCommandTimeout
}

func tmuxRunCommand(tmuxCommand string, args ...string) error {
	return tmuxRunCommandTimeout(tmuxCommand, tmuxCommandTimeout(), args...)
}

func tmuxRunCommandTimeout(tmuxCommand string, timeout time.Duration, args ...string) error {
	_, err := tmuxCombinedOutputCommandTimeout(tmuxCommand, timeout, args...)
	return err
}

func tmuxOutputCommand(tmuxCommand string, args ...string) ([]byte, error) {
	return tmuxOutputCommandTimeout(tmuxCommand, tmuxCommandTimeout(), args...)
}

func tmuxOutputCommandTimeout(tmuxCommand string, timeout time.Duration, args ...string) ([]byte, error) {
	return tmuxExecCommandTimeout(tmuxCommand, timeout, func(cmd *exec.Cmd) ([]byte, error) {
		return cmd.Output()
	}, args...)
}

func tmuxCombinedOutputCommand(tmuxCommand string, args ...string) ([]byte, error) {
	return tmuxCombinedOutputCommandTimeout(tmuxCommand, tmuxCommandTimeout(), args...)
}

func tmuxCombinedOutputCommandTimeout(tmuxCommand string, timeout time.Duration, args ...string) ([]byte, error) {
	return tmuxExecCommandTimeout(tmuxCommand, timeout, func(cmd *exec.Cmd) ([]byte, error) {
		return cmd.CombinedOutput()
	}, args...)
}

func tmuxExecCommandTimeout(tmuxCommand string, timeout time.Duration, run func(*exec.Cmd) ([]byte, error), args ...string) ([]byte, error) {
	tmuxCommand = strings.TrimSpace(tmuxCommand)
	if tmuxCommand == "" {
		return nil, fmt.Errorf("tmux command vacio")
	}
	if timeout <= 0 {
		timeout = tmuxCommandTimeout()
	}
	cmd := exec.Command(tmuxCommand, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	type result struct {
		out []byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := run(cmd)
		done <- result{out: out, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case res := <-done:
		return res.out, res.err
	case <-timer.C:
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			_ = cmd.Process.Kill()
		}
		res := <-done
		return res.out, fmt.Errorf("tmux %s timeout", strings.TrimSpace(strings.Join(args, " ")))
	}
}
