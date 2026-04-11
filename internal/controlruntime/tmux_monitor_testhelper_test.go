package controlruntime

import (
	"os"
	"testing"
)

func TestEmbeddedTmuxMonitorHelperProcess(t *testing.T) {
	if !isEmbeddedTmuxMonitorHelperTestProcess() {
		return
	}
	args, err := embeddedTmuxMonitorHelperArgsFromTestProcess()
	if err != nil {
		os.Exit(2)
	}
	if err := runEmbeddedTmuxMonitor(args); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
