package main

import (
	"errors"
	"os/exec"
	"strings"
	"time"
)

func errGuardianCandidateStopReceiptV0(receipt guardianCandidateStopReceiptV0) error {
	reason := strings.TrimSpace(receipt.ReasonCode)
	if reason == "" {
		reason = "candidate_process_left_alive"
	}
	return errors.New(reason)
}

func guardianCandidateStopPassedV0(receipt guardianCandidateStopReceiptV0) bool {
	return !receipt.Ambiguous && receipt.TreeStopConfirmed
}

func stopGuardianCandidateProcessV0(cmd *exec.Cmd, waitDone <-chan error, timeout time.Duration) guardianCandidateStopReceiptV0 {
	return stopGuardianCandidateProcessPlatformV0(cmd, waitDone, timeout)
}
