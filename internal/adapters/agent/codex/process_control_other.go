//go:build !linux

package codex

import (
	"errors"
	"os"

	"orquesta/internal/ports"
)

func platformProcessControlSupported() bool { return false }

func platformTryOwnerLock(*os.File) (bool, error) { return false, errors.New(CodeControlUnsupported) }

func platformOwnerLockCLOEXEC(*os.File) bool { return false }

func platformUnlockOwner(*os.File) {}

func platformCaptureProcess(int) (int, string, string, error) {
	return 0, "", "", errors.New(CodeControlUnsupported)
}

func platformInspectProcess(processRecord) (processIdentityState, error) {
	return processIdentityMismatch, errors.New(CodeControlUnsupported)
}

func platformSignalProcess(processRecord, ports.AgentStopMode) error {
	return errors.New(CodeControlUnsupported)
}
