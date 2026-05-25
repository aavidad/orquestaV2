//go:build !windows

package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
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

func processAliveV0(pid int) bool {
	if pid <= 0 {
		return false
	}
	if processZombieV0(pid) {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func processZombieV0(pid int) bool {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return false
	}
	stat := strings.TrimSpace(string(data))
	commandEnd := strings.LastIndex(stat, ")")
	if commandEnd < 0 || commandEnd+2 >= len(stat) {
		return false
	}
	return stat[commandEnd+2] == 'Z'
}
