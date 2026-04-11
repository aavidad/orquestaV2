package controlruntime

import (
	"os"
	"testing"
)

func TestEmbeddedBrokerHelperProcess(t *testing.T) {
	if !isEmbeddedBrokerHelperTestProcess() {
		return
	}
	args, err := embeddedBrokerHelperArgsFromTestProcess()
	if err != nil {
		os.Exit(2)
	}
	if err := runEmbeddedBroker(args); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
