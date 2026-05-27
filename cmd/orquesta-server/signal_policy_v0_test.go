package main

import (
	"context"
	"os"
	"runtime"
	"testing"
)

func TestServerShutdownSignalPolicyV0DeclaraPlataformaYEscalado(t *testing.T) {
	policy := serverShutdownSignalPolicyV0()

	if !containsStringForTestV0(policy.HandledSignals, "signal_interrupt") {
		t.Fatalf("handled_signals=%v sin interrupt", policy.HandledSignals)
	}
	if runtime.GOOS != "windows" && !containsStringForTestV0(policy.HandledSignals, "signal_terminate") {
		t.Fatalf("handled_signals=%v sin terminate", policy.HandledSignals)
	}
	if policy.CooperativeSignal != "signal_interrupt" ||
		policy.SecondSignalAction != "signal_escalated" {
		t.Fatalf("policy=%+v", policy)
	}
	if serverSignalNameV0(serverCooperativeStopSignalV0()) != policy.CooperativeSignal {
		t.Fatalf("stop_signal=%s policy=%+v",
			serverSignalNameV0(serverCooperativeStopSignalV0()),
			policy,
		)
	}
}

func TestServerSignalControllerV0PrimeraSenalCancelaYSegundaEscala(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	escalated := false
	controller := &serverSignalControllerV0{
		ctx:      ctx,
		cancel:   cancel,
		signals:  make(chan os.Signal, 2),
		escalate: func(os.Signal) { escalated = true },
	}

	controller.observeSignalV0(os.Interrupt)
	if err := controller.contextV0().Err(); err == nil {
		t.Fatalf("primera senal no cancelo contexto")
	}
	cause := controller.shutdownCauseV0()
	if cause.SignalName != "signal_interrupt" ||
		cause.Count != 1 ||
		cause.Escalated {
		t.Fatalf("cause primera=%+v", cause)
	}

	controller.observeSignalV0(os.Interrupt)
	cause = controller.shutdownCauseV0()
	if cause.Count != 2 || !cause.Escalated || !escalated {
		t.Fatalf("cause segunda=%+v escalated=%v", cause, escalated)
	}
}
