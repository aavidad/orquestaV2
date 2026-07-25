//go:build linux

package firecrackerguest

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

const privatePID1SupervisorArgument = "orquesta-firecracker-guest-pid1-v1"

type pid1SupervisorResult struct {
	ExitCode int
	Reaped   uint64
	Empty    bool
	Err      error
}

// DispatchPID1Supervisor handles the private self-exec used by OSExecutor.
// It is exported only because the guest bootstrap lives in cmd/.
func DispatchPID1Supervisor(arguments []string) (int, bool) {
	if len(arguments) < 2 || arguments[1] != privatePID1SupervisorArgument {
		return 0, false
	}
	if os.Getpid() != 1 || len(arguments) < 4 {
		return 255, true
	}
	environmentCount, err := strconv.Atoi(arguments[2])
	if err != nil || environmentCount < 0 || environmentCount > 64 ||
		len(arguments) < environmentCount+4 {
		return 255, true
	}
	targetIndex := environmentCount + 3
	target := arguments[targetIndex]
	if !filepath.IsAbs(target) {
		return 255, true
	}
	environment := append([]string(nil), arguments[3:targetIndex]...)
	result := runPID1Supervisor(target, arguments[targetIndex+1:], environment)
	if result.Err != nil || !result.Empty {
		return 255, true
	}
	return result.ExitCode, true
}

// runPID1Supervisor is the sole waiter inside the per-test PID namespace.
// The subject is started with raw ForkExec so no exec.Cmd waiter can race with
// Wait4(-1). Once the subject exits, every remaining process is killed
// regardless of process group/session and reaped before the supervisor exits.
func runPID1Supervisor(
	target string,
	arguments []string,
	environment []string,
) pid1SupervisorResult {
	result := pid1SupervisorResult{ExitCode: 255}
	if os.Getpid() != 1 || target == "" || !filepath.IsAbs(target) {
		result.Err = unix.EINVAL
		return result
	}
	targetArguments := append([]string{target}, arguments...)
	targetPID, err := syscall.ForkExec(target, targetArguments, &syscall.ProcAttr{
		Dir:   ".",
		Env:   append([]string(nil), environment...),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
		Sys: &syscall.SysProcAttr{
			// PID1 owns the namespace lifecycle and reaps every child.
			// Pdeathsig is intentionally absent: Linux binds it to the
			// creating OS thread, while this supervisor may migrate between
			// Wait4 calls.
			Setpgid: true,
		},
	})
	if err != nil {
		result.Err = err
		return result
	}

	targetReaped := false
	for {
		var status unix.WaitStatus
		reapedPID, waitErr := unix.Wait4(-1, &status, 0, nil)
		if waitErr != nil {
			if errors.Is(waitErr, unix.EINTR) {
				continue
			}
			if errors.Is(waitErr, unix.ECHILD) {
				result.Empty = targetReaped
				if !targetReaped {
					result.Err = unix.ECHILD
				}
				return result
			}
			_ = killPIDNamespaceProcesses()
			result.Err = waitErr
			return drainPID1Children(result)
		}
		if reapedPID <= 0 {
			continue
		}
		result.Reaped++
		if reapedPID != targetPID {
			continue
		}
		targetReaped = true
		result.ExitCode = pid1ExitCode(status)
		if err := killPIDNamespaceProcesses(); err != nil {
			result.Err = err
		}
	}
}

func drainPID1Children(result pid1SupervisorResult) pid1SupervisorResult {
	for {
		var status unix.WaitStatus
		reapedPID, err := unix.Wait4(-1, &status, 0, nil)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if errors.Is(err, unix.ECHILD) {
			result.Empty = true
			return result
		}
		if err != nil {
			result.Err = errors.Join(result.Err, err)
			return result
		}
		if reapedPID > 0 {
			result.Reaped++
		}
	}
}

func killPIDNamespaceProcesses() error {
	err := unix.Kill(-1, unix.SIGKILL)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	return err
}

func pid1ExitCode(status unix.WaitStatus) int {
	if status.Exited() {
		code := status.ExitStatus()
		if code >= 0 && code <= 255 {
			return code
		}
		return 255
	}
	if status.Signaled() {
		code := 128 + int(status.Signal())
		if code <= 255 {
			return code
		}
	}
	return 255
}
