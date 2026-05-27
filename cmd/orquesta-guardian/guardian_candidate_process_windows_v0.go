//go:build windows

package main

import (
	"os"
	"os/exec"
	"time"
)

func guardianPrepareCandidateCommandV0(cmd *exec.Cmd) guardianCandidateProcessPolicyV0 {
	return guardianCandidateProcessPolicyV0{
		Platform:       "windows",
		Scope:          "parent_process",
		TreeStop:       false,
		PublicContract: "candidate_parent_process_stopped_before_promotion",
	}
}

func stopGuardianCandidateProcessPlatformV0(
	cmd *exec.Cmd,
	waitDone <-chan error,
	timeout time.Duration,
) guardianCandidateStopReceiptV0 {
	receipt := guardianCandidateStopReceiptV0{
		Scope:      "parent_process",
		Status:     "stop_requested",
		ReasonCode: "candidate_stop_requested",
	}
	if cmd == nil || cmd.Process == nil {
		receipt.Status = "stopped"
		receipt.ReasonCode = "candidate_parent_stopped"
		receipt.TreeStopConfirmed = true
		return receipt
	}
	_ = cmd.Process.Signal(os.Interrupt)
	select {
	case <-waitDone:
		receipt.Status = "stopped"
		receipt.ReasonCode = "candidate_parent_stopped"
		receipt.TreeStopConfirmed = true
		return receipt
	case <-time.After(timeout):
		receipt.DeadlineExceeded = true
		receipt.Escalated = true
		receipt.ReasonCode = "candidate_stop_timeout"
		_ = cmd.Process.Kill()
	}
	select {
	case <-waitDone:
		receipt.Status = "stopped"
		receipt.ReasonCode = "candidate_killed"
		receipt.TreeStopConfirmed = true
	case <-time.After(500 * time.Millisecond):
		receipt.Ambiguous = true
		receipt.Status = "ambiguous"
		receipt.ReasonCode = "candidate_process_left_alive"
	}
	return receipt
}
