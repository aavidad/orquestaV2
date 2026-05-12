//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func configureDetachedProcessV0(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func signalProcessV0(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(os.Interrupt)
}
