//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func configureDetachedProcessV0(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

type serverDaemonProcIdentityV0 struct {
	State     string
	GroupID   int
	SessionID int
	StartRef  string
}

func captureServerDaemonProcessIdentityV0(pid int) (serverDaemonProcessIdentityV0, error) {
	observed, err := readServerDaemonProcIdentityV0(pid)
	if err != nil || observed.State == "Z" || observed.StartRef == "" || observed.GroupID != pid || observed.SessionID != pid {
		return serverDaemonProcessIdentityV0{}, fmt.Errorf("daemon_process_identity_unavailable")
	}
	return serverDaemonProcessIdentityV0{
		PID: pid, StartRef: observed.StartRef, GroupID: observed.GroupID, SessionID: observed.SessionID,
	}, nil
}

func readServerDaemonProcIdentityV0(pid int) (serverDaemonProcIdentityV0, error) {
	if pid <= 0 {
		return serverDaemonProcIdentityV0{}, os.ErrProcessDone
	}
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return serverDaemonProcIdentityV0{}, err
	}
	stat := strings.TrimSpace(string(raw))
	commandEnd := strings.LastIndex(stat, ")")
	if commandEnd < 0 || commandEnd+2 >= len(stat) {
		return serverDaemonProcIdentityV0{}, fmt.Errorf("daemon_process_identity_unavailable")
	}
	fields := strings.Fields(stat[commandEnd+2:])
	if len(fields) < 20 {
		return serverDaemonProcIdentityV0{}, fmt.Errorf("daemon_process_identity_unavailable")
	}
	groupID, groupErr := strconv.Atoi(fields[2])
	sessionID, sessionErr := strconv.Atoi(fields[3])
	if groupErr != nil || sessionErr != nil || strings.TrimSpace(fields[19]) == "" {
		return serverDaemonProcIdentityV0{}, fmt.Errorf("daemon_process_identity_unavailable")
	}
	return serverDaemonProcIdentityV0{
		State: fields[0], GroupID: groupID, SessionID: sessionID, StartRef: strings.TrimSpace(fields[19]),
	}, nil
}

func serverDaemonProcessIdentityAliveV0(identity serverDaemonProcessIdentityV0) (bool, error) {
	observed, err := readServerDaemonProcIdentityV0(identity.PID)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("daemon_process_identity_unavailable")
	}
	if observed.State == "Z" {
		return false, nil
	}
	if identity.StartRef == "" || observed.StartRef != identity.StartRef ||
		observed.GroupID != identity.GroupID || observed.SessionID != identity.SessionID {
		return false, fmt.Errorf("daemon_process_identity_changed")
	}
	return true, nil
}

func serverDaemonProcessGroupActiveV0(identity serverDaemonProcessIdentityV0) (bool, error) {
	if identity.PID <= 0 || identity.GroupID <= 0 || identity.SessionID <= 0 {
		return false, fmt.Errorf("daemon_process_identity_invalid")
	}
	if leader, err := readServerDaemonProcIdentityV0(identity.PID); err == nil && leader.State != "Z" {
		if identity.StartRef == "" || leader.StartRef != identity.StartRef ||
			leader.GroupID != identity.GroupID || leader.SessionID != identity.SessionID {
			return false, fmt.Errorf("daemon_process_identity_changed")
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("daemon_process_identity_unavailable")
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false, fmt.Errorf("daemon_process_identity_unavailable")
	}
	for _, entry := range entries {
		pid, parseErr := strconv.Atoi(entry.Name())
		if parseErr != nil || pid <= 0 {
			continue
		}
		observed, readErr := readServerDaemonProcIdentityV0(pid)
		if readErr != nil || observed.State == "Z" || observed.GroupID != identity.GroupID {
			continue
		}
		if observed.SessionID != identity.SessionID {
			return false, fmt.Errorf("daemon_process_identity_changed")
		}
		return true, nil
	}
	return false, nil
}

func signalServerDaemonProcessIdentityV0(identity serverDaemonProcessIdentityV0) error {
	alive, err := serverDaemonProcessIdentityAliveV0(identity)
	if err != nil || !alive {
		return err
	}
	if err := syscall.Kill(identity.PID, syscall.SIGINT); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			_, verifyErr := serverDaemonProcessIdentityAliveV0(identity)
			return verifyErr
		}
		return fmt.Errorf("daemon_process_signal_failed")
	}
	return nil
}

func terminateServerDaemonProcessGroupIdentityV0(identity serverDaemonProcessIdentityV0) error {
	return signalServerDaemonProcessGroupIdentityV0(identity, syscall.SIGTERM)
}

func killServerDaemonProcessGroupIdentityV0(identity serverDaemonProcessIdentityV0) error {
	return signalServerDaemonProcessGroupIdentityV0(identity, syscall.SIGKILL)
}

func signalServerDaemonProcessGroupIdentityV0(identity serverDaemonProcessIdentityV0, signal syscall.Signal) error {
	active, err := serverDaemonProcessGroupActiveV0(identity)
	if err != nil || !active {
		return err
	}
	if err := syscall.Kill(-identity.GroupID, signal); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			_, verifyErr := serverDaemonProcessGroupActiveV0(identity)
			return verifyErr
		}
		return fmt.Errorf("daemon_process_group_signal_failed")
	}
	return nil
}

func signalProcessV0(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(serverCooperativeStopSignalV0())
}

func signalProcessGroupV0(pid int) error {
	if pid <= 0 {
		return os.ErrProcessDone
	}
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		return signalProcessV0(pid)
	}
	return nil
}

func signalProcessGroupKillV0(pid int) error {
	if pid <= 0 {
		return os.ErrProcessDone
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		process, findErr := os.FindProcess(pid)
		if findErr != nil {
			return findErr
		}
		return process.Kill()
	}
	return nil
}

func processGroupAliveV0(pid int) bool {
	if pid <= 0 {
		return false
	}
	if syscall.Kill(-pid, syscall.Signal(0)) != nil {
		return false
	}
	if alive, known := procProcessGroupHasNonZombieV0(pid); known {
		return alive
	}
	return true
}

func procProcessGroupHasNonZombieV0(groupID int) (bool, bool) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false, false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		data, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if err != nil {
			continue
		}
		stat := strings.TrimSpace(string(data))
		commandEnd := strings.LastIndex(stat, ")")
		if commandEnd < 0 || commandEnd+2 >= len(stat) {
			continue
		}
		fields := strings.Fields(stat[commandEnd+2:])
		if len(fields) < 3 {
			continue
		}
		processGroup, err := strconv.Atoi(fields[2])
		if err == nil && processGroup == groupID && fields[0] != "Z" {
			return true, true
		}
	}
	return false, true
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
