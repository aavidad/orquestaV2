package orquestaserver

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRuntimeV0ShutdownPorSenalPublicaStoppingYTimeoutV0(t *testing.T) {
	requireLocalTCPForServerTestV0(t)
	now := time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)
	store := &threadSafeStateStoreV0{}
	supervisor := newBlockingIdlePrepareSupervisorV0()
	defer close(supervisor.releasePrepare)
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:                "127.0.0.1:0",
		StateDir:            t.TempDir(),
		AuditDisabled:       true,
		TickInterval:        time.Hour,
		ShutdownGracePeriod: 30 * time.Millisecond,
		ShutdownSignalPolicy: ShutdownSignalPolicyV0{
			HandledSignals:     []string{"signal_interrupt", "signal_terminate"},
			CooperativeSignal:  "signal_interrupt",
			SecondSignalAction: "signal_escalated",
		},
		IdleSelfImprovementAfter:    time.Millisecond,
		IdleSelfImprovementWriteSet: []string{"modulos/orquesta-server"},
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runtime.RunWithShutdownCauseV0(ctx, func() ShutdownSignalCauseV0 {
			return ShutdownSignalCauseV0{SignalName: "signal_interrupt", Count: 1}
		})
	}()
	waitForRuntimeTestV0(t, supervisor.prepareStarted)

	cancel()
	err = waitRuntimeDoneV0(t, done)
	if err == nil || !strings.Contains(err.Error(), "async_work_timeout") {
		t.Fatalf("RunWithShutdownCauseV0 err=%v", err)
	}
	state := runtime.StateV0()
	if state.Status != "stop_timeout" ||
		state.ShutdownStatus != "stop_timeout" ||
		state.ShutdownSignalName != "signal_interrupt" ||
		state.ShutdownSignalCount != 1 ||
		!containsServerStringForTestV0(state.ShutdownSignalPolicy.HandledSignals, "signal_terminate") {
		t.Fatalf("estado signal timeout=%+v", state)
	}
	public := NewServerPublicStatusV0(state)
	if public.ShutdownSignalName != "signal_interrupt" ||
		!containsServerStringForTestV0(public.ShutdownSignalPolicy.HandledSignals, "signal_interrupt") {
		t.Fatalf("public=%+v", public)
	}
}

func containsServerStringForTestV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
