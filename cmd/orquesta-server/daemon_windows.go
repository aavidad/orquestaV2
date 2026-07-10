//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func configureDetachedProcessV0(_ *exec.Cmd) {}

func captureServerDaemonProcessIdentityV0(pid int) (serverDaemonProcessIdentityV0, error) {
	if pid <= 0 {
		return serverDaemonProcessIdentityV0{}, fmt.Errorf("daemon_process_identity_unavailable")
	}
	return serverDaemonProcessIdentityV0{PID: pid, StartRef: fmt.Sprintf("pid-%d", pid), GroupID: pid, SessionID: pid}, nil
}

func serverDaemonProcessIdentityAliveV0(identity serverDaemonProcessIdentityV0) (bool, error) {
	return processAliveV0(identity.PID), nil
}

func serverDaemonProcessGroupActiveV0(identity serverDaemonProcessIdentityV0) (bool, error) {
	return processAliveV0(identity.PID), nil
}

func signalServerDaemonProcessIdentityV0(identity serverDaemonProcessIdentityV0) error {
	return signalProcessV0(identity.PID)
}

func terminateServerDaemonProcessGroupIdentityV0(identity serverDaemonProcessIdentityV0) error {
	return signalProcessGroupV0(identity.PID)
}

func killServerDaemonProcessGroupIdentityV0(identity serverDaemonProcessIdentityV0) error {
	return signalProcessGroupKillV0(identity.PID)
}

func signalProcessV0(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(serverCooperativeStopSignalV0())
}

func signalProcessGroupV0(pid int) error {
	return signalProcessV0(pid)
}

func signalProcessGroupKillV0(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

func processGroupAliveV0(pid int) bool {
	return processAliveV0(pid)
}

func processAliveV0(pid int) bool {
	if pid <= 0 {
		return false
	}
	_, err := os.FindProcess(pid)
	return err == nil
}
