//go:build linux

package codex

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
	"orquesta/internal/ports"
)

func platformProcessControlSupported() bool {
	for _, name := range []string{"/bin/sh", "/proc/self/stat", "/proc/sys/kernel/random/boot_id"} {
		if _, err := os.Stat(name); err != nil {
			return false
		}
	}
	return true
}

func platformTryOwnerLock(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	flags, err := unix.FcntlInt(file.Fd(), unix.F_GETFD, 0)
	if err != nil {
		return false, err
	}
	if flags&unix.FD_CLOEXEC == 0 {
		_, err = unix.FcntlInt(file.Fd(), unix.F_SETFD, flags|unix.FD_CLOEXEC)
	}
	return err == nil, err
}

func platformOwnerLockCLOEXEC(file *os.File) bool {
	flags, err := unix.FcntlInt(file.Fd(), unix.F_GETFD, 0)
	return err == nil && flags&unix.FD_CLOEXEC != 0
}

func platformUnlockOwner(file *os.File) {
	_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

func platformCaptureProcess(pid int) (int, string, string, error) {
	state, pgid, birth, err := readLinuxProcess(pid)
	if err != nil {
		return 0, "", "", fmt.Errorf("capture process: %w", err)
	}
	if state == "Z" || state == "X" {
		return 0, "", "", errors.New("capture process: process already exited")
	}
	bootID, err := linuxBootID()
	if err != nil {
		return 0, "", "", err
	}
	return pgid, bootID, birth, nil
}

func platformInspectProcess(record processRecord) (processIdentityState, error) {
	bootID, err := linuxBootID()
	if err != nil {
		return processIdentityMismatch, err
	}
	if bootID != record.BootID {
		return processIdentityMismatch, nil
	}
	state, pgid, birth, err := readLinuxProcess(record.PID)
	if processGoneError(err) {
		return processIdentityGone, nil
	}
	if err != nil {
		return processIdentityMismatch, err
	}
	if pgid != record.PGID || birth != record.BirthMarker {
		return processIdentityMismatch, nil
	}
	if state == "Z" || state == "X" {
		return processIdentityGone, nil
	}
	return processIdentityAlive, nil
}

func platformSignalProcess(record processRecord, mode ports.AgentStopMode) error {
	state, err := platformInspectProcess(record)
	if err != nil {
		return err
	}
	if state == processIdentityMismatch {
		return &Error{Code: CodeProcessIdentityMismatch}
	}
	if state == processIdentityGone {
		return os.ErrProcessDone
	}
	signal := unix.SIGTERM
	if mode == ports.AgentStopForced {
		signal = unix.SIGKILL
	}
	if err := unix.Kill(-record.PGID, signal); errors.Is(err, unix.ESRCH) {
		return os.ErrProcessDone
	} else {
		return err
	}
}

func platformSignalCgroupSupervisor(record processRecord, mode ports.AgentStopMode) error {
	signal := unix.SIGTERM
	if mode == ports.AgentStopForced {
		signal = unix.SIGUSR1
	}
	return pidfdSignalExact(
		record.PID, record.PGID, record.BootID, record.BirthMarker, signal,
	)
}

func pidfdSignalExact(pid, pgid int, bootID, birth string, signal unix.Signal) error {
	fd, err := unix.PidfdOpen(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return os.ErrProcessDone
	}
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	state, currentPGID, currentBirth, readErr := readLinuxProcess(pid)
	currentBoot, bootErr := linuxBootID()
	if processGoneError(readErr) || state == "Z" || state == "X" {
		return os.ErrProcessDone
	}
	if readErr != nil || bootErr != nil || currentPGID != pgid ||
		currentBoot != bootID || currentBirth != birth {
		return &Error{Code: CodeProcessIdentityMismatch, Cause: errors.Join(readErr, bootErr)}
	}
	if err := unix.PidfdSendSignal(fd, signal, nil, 0); errors.Is(err, unix.ESRCH) {
		return os.ErrProcessDone
	} else {
		return err
	}
}

func processGoneError(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, unix.ESRCH)
}

func readLinuxProcess(pid int) (string, int, string, error) {
	payload, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", 0, "", err
	}
	end := strings.LastIndex(string(payload), ")")
	if end < 0 || end+1 >= len(payload) {
		return "", 0, "", errors.New("invalid proc stat")
	}
	fields := strings.Fields(string(payload[end+1:]))
	if len(fields) <= 19 {
		return "", 0, "", errors.New("short proc stat")
	}
	pgid, err := strconv.Atoi(fields[2])
	if err != nil || pgid <= 0 {
		return "", 0, "", errors.New("invalid process group")
	}
	return fields[0], pgid, fields[19], nil
}

func linuxBootID() (string, error) {
	payload, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(payload))
	if value == "" {
		return "", errors.New("empty boot id")
	}
	return value, nil
}
