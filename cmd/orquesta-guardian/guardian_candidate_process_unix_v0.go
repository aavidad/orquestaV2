//go:build !windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func guardianPrepareCandidateCommandV0(cmd *exec.Cmd) guardianCandidateProcessPolicyV0 {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return guardianCandidateProcessPolicyV0{
		Platform:       "unix",
		Scope:          "process_group",
		TreeStop:       true,
		PublicContract: "candidate_process_group_stopped_before_promotion",
	}
}

func stopGuardianCandidateProcessPlatformV0(
	cmd *exec.Cmd,
	waitDone <-chan error,
	timeout time.Duration,
) guardianCandidateStopReceiptV0 {
	receipt := guardianCandidateStopReceiptV0{
		Scope:      "process_group",
		Status:     "stop_requested",
		ReasonCode: "candidate_stop_requested",
	}
	if cmd == nil || cmd.Process == nil {
		receipt.Status = "stopped"
		receipt.ReasonCode = "candidate_stopped"
		receipt.TreeStopConfirmed = true
		return receipt
	}
	pid := cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGINT)
	select {
	case <-waitDone:
		return confirmGuardianCandidateProcessGroupStoppedV0(pid, receipt)
	case <-time.After(timeout):
		receipt.DeadlineExceeded = true
		receipt.Escalated = true
		receipt.ReasonCode = "candidate_stop_timeout"
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
	select {
	case <-waitDone:
	case <-time.After(500 * time.Millisecond):
		receipt.Ambiguous = true
		receipt.Status = "ambiguous"
		receipt.ReasonCode = "candidate_process_left_alive"
		return receipt
	}
	return confirmGuardianCandidateProcessGroupStoppedV0(pid, receipt)
}

func confirmGuardianCandidateProcessGroupStoppedV0(
	pid int,
	receipt guardianCandidateStopReceiptV0,
) guardianCandidateStopReceiptV0 {
	time.Sleep(25 * time.Millisecond)
	err := syscall.Kill(-pid, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		receipt.Ambiguous = true
		receipt.Status = "ambiguous"
		receipt.ReasonCode = "candidate_process_left_alive"
		return receipt
	}
	if errors.Is(err, syscall.ESRCH) {
		receipt.Status = "stopped"
		if receipt.Escalated {
			receipt.ReasonCode = "candidate_killed"
		} else {
			receipt.ReasonCode = "candidate_stopped"
		}
		receipt.TreeStopConfirmed = true
		return receipt
	}
	if errors.Is(err, os.ErrProcessDone) {
		receipt.Status = "stopped"
		receipt.ReasonCode = "candidate_stopped"
		receipt.TreeStopConfirmed = true
		return receipt
	}
	receipt.Ambiguous = true
	receipt.Status = "ambiguous"
	receipt.ReasonCode = "candidate_process_left_alive"
	return receipt
}
