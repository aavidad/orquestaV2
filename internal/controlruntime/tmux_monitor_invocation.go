package controlruntime

import (
	"fmt"
	"os"
)

func embeddedTmuxMonitorInvocation(specPath string) (string, []string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", nil, err
	}
	return exePath, []string{"__tmux_monitor", "--spec", specPath}, nil
}

func isEmbeddedTmuxMonitorHelperTestProcess() bool {
	return false
}

func embeddedTmuxMonitorHelperArgsFromTestProcess() ([]string, error) {
	return nil, fmt.Errorf("helper tmux monitor test deshabilitado")
}
