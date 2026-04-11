package controlruntime

import "os"

func init() {
	args := os.Args[1:]
	if MaybeRunEmbeddedBroker(args) {
		return
	}
	MaybeRunEmbeddedTmuxMonitor(args)
}
