//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverShutdownSignalPolicyV0() orquestaserver.ShutdownSignalPolicyV0 {
	return orquestaserver.ShutdownSignalPolicyV0{
		HandledSignals:     []string{"signal_interrupt", "signal_terminate"},
		CooperativeSignal:  "signal_interrupt",
		SecondSignalAction: "signal_escalated",
	}
}

func serverShutdownSignalsV0() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}

func serverCooperativeStopSignalV0() os.Signal {
	return os.Interrupt
}

func serverSignalNameV0(sig os.Signal) string {
	switch sig {
	case os.Interrupt:
		return "signal_interrupt"
	case syscall.SIGTERM:
		return "signal_terminate"
	default:
		return "signal_unknown"
	}
}

func serverEscalateSignalV0(sig os.Signal) {
	signal.Reset(serverShutdownSignalsV0()...)
	if process, err := os.FindProcess(os.Getpid()); err == nil {
		_ = process.Signal(sig)
	}
}
