package orquestaruntime

import (
	"errors"
	"os"
	"syscall"
)

func processRuntimeProcessAliveV0(process *os.Process) bool {
	if process == nil {
		return false
	}
	err := process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	if errors.Is(err, syscall.ESRCH) {
		return false
	}
	return !errors.Is(err, os.ErrProcessDone)
}
