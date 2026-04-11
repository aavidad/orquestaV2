package controlruntime

import (
	"fmt"
	"os"
)

func embeddedBrokerInvocation(specPath string) (string, []string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", nil, err
	}
	return exePath, []string{"__pty_broker", "--spec", specPath}, nil
}

func isEmbeddedBrokerHelperTestProcess() bool {
	return false
}

func embeddedBrokerHelperArgsFromTestProcess() ([]string, error) {
	return nil, fmt.Errorf("helper broker test deshabilitado")
}
