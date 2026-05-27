//go:build windows

package main

import (
	"os"
	"os/exec"
)

func configureDetachedProcessV0(_ *exec.Cmd) {}

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

func processAliveV0(pid int) bool {
	if pid <= 0 {
		return false
	}
	_, err := os.FindProcess(pid)
	return err == nil
}
