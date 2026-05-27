//go:build windows

package main

import (
	"os"
	"os/signal"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverShutdownSignalPolicyV0() orquestaserver.ShutdownSignalPolicyV0 {
	return orquestaserver.ShutdownSignalPolicyV0{
		HandledSignals:     []string{"signal_interrupt"},
		CooperativeSignal:  "signal_interrupt",
		SecondSignalAction: "signal_escalated",
	}
}

func serverShutdownSignalsV0() []os.Signal {
	return []os.Signal{os.Interrupt}
}

func serverCooperativeStopSignalV0() os.Signal {
	return os.Interrupt
}

func serverSignalNameV0(sig os.Signal) string {
	if sig == os.Interrupt {
		return "signal_interrupt"
	}
	return "signal_unknown"
}

func serverEscalateSignalV0(os.Signal) {
	signal.Reset(serverShutdownSignalsV0()...)
	if process, err := os.FindProcess(os.Getpid()); err == nil {
		_ = process.Kill()
	}
}
