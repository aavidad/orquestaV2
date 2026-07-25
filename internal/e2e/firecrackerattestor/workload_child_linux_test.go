//go:build linux

package firecrackerattestor

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestTerminateCommandProcessGroupReapsChildGroup(t *testing.T) {
	command := exec.Command("/bin/sh", "-c", "trap 'exit 0' TERM; sleep 30 & wait")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { finished <- command.Wait() }()
	if err := terminateCommandProcessGroup(command); err != nil &&
		!errors.Is(err, os.ErrProcessDone) {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		t.Fatal("process group survived cancellation")
	}
	if err := syscall.Kill(-command.Process.Pid, 0); !errors.Is(err, syscall.ESRCH) {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		t.Fatalf("process group remains: %v", err)
	}
}
