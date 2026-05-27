package orquestaruntime

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestProcessRuntimeConnectorV0ExponeStoppingAntesDeEscalar(t *testing.T) {
	connector := NewProcessRuntimeConnectorWithPolicyV0(ProcessRuntimeEffectPolicyV0{
		StopTimeout: 300 * time.Millisecond,
		KillWait:    2 * time.Second,
		StopSignal:  syscall.SIGUSR1,
	})
	req := processRuntimeLaunchRequestForTestV0(t, "ignore")
	launched, err := connector.LaunchV0(context.Background(), req)
	requireNoProcessRuntimeErrorV0(t, err)
	waitForProcessRuntimeChildFileV0(t, req.WorkingDir, "ignore-started.txt")

	done := make(chan stopRuntimeTestResultV0, 1)
	go func() {
		stopped, stopErr := connector.StopV0(context.Background(), launched.ProcessRef)
		done <- stopRuntimeTestResultV0{snapshot: stopped, err: stopErr}
	}()

	stopping := waitForProcessRuntimeStatusV0(
		t,
		connector,
		launched.ProcessRef,
		ProcessRuntimeStoppingV0,
	)
	if stopping.StopRef == "" || stopping.StopGraceDeadline == "" {
		t.Fatalf("stopping sin stop_ref/deadline: %+v", stopping)
	}
	if stopping.StopReasonCode != ProcessRuntimeStopCooperativeSignalSentV0 {
		t.Fatalf("stop_reason_code stopping=%q", stopping.StopReasonCode)
	}

	var result stopRuntimeTestResultV0
	select {
	case result = <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("stop escalado no termino")
	}
	requireNoProcessRuntimeErrorV0(t, result.err)
	if result.snapshot.Status != ProcessRuntimeStoppedV0 {
		t.Fatalf("status final=%q", result.snapshot.Status)
	}
	if result.snapshot.StopReasonCode != ProcessRuntimeStopGraceTimeoutV0 {
		t.Fatalf("stop_reason_code final=%q", result.snapshot.StopReasonCode)
	}
	assertProcessRuntimeSnapshotDoesNotLeakV0(t, result.snapshot, req)
}

func TestProcessRuntimeConnectorV0ProcesoQueIgnoraSenalEscalaPorTimeout(t *testing.T) {
	connector := NewProcessRuntimeConnectorWithPolicyV0(ProcessRuntimeEffectPolicyV0{
		StopTimeout: 50 * time.Millisecond,
		KillWait:    2 * time.Second,
		StopSignal:  syscall.SIGUSR1,
	})
	req := processRuntimeLaunchRequestForTestV0(t, "ignore")
	launched, err := connector.LaunchV0(context.Background(), req)
	requireNoProcessRuntimeErrorV0(t, err)
	waitForProcessRuntimeChildFileV0(t, req.WorkingDir, "ignore-started.txt")

	stopped, err := connector.StopV0(context.Background(), launched.ProcessRef)
	requireNoProcessRuntimeErrorV0(t, err)
	if stopped.Status != ProcessRuntimeStoppedV0 || stopped.StopRef == "" {
		t.Fatalf("stop escalado invalido: %+v", stopped)
	}
	if stopped.StopReasonCode != ProcessRuntimeStopGraceTimeoutV0 {
		t.Fatalf("stop_reason_code=%q", stopped.StopReasonCode)
	}
	if stopped.StopGraceDeadline == "" {
		t.Fatalf("stop_grace_deadline vacio: %+v", stopped)
	}
	assertProcessRuntimeSnapshotDoesNotLeakV0(t, stopped, req)
}

func TestProcessRuntimeConnectorV0StopDeProcesoYaDetenidoDeclaraCodigoPublico(t *testing.T) {
	connector := NewProcessRuntimeConnectorV0()
	req := processRuntimeLaunchRequestForTestV0(t, "exit")
	launched, err := connector.LaunchV0(context.Background(), req)
	requireNoProcessRuntimeErrorV0(t, err)
	waitForProcessRuntimeStatusV0(t, connector, launched.ProcessRef, ProcessRuntimeStoppedV0)

	stopped, err := connector.StopV0(context.Background(), launched.ProcessRef)
	requireNoProcessRuntimeErrorV0(t, err)
	if stopped.Status != ProcessRuntimeStoppedV0 || stopped.StopRef == "" {
		t.Fatalf("stop ya detenido invalido: %+v", stopped)
	}
	if stopped.StopReasonCode != ProcessRuntimeStopAlreadyStoppedV0 {
		t.Fatalf("stop_reason_code=%q", stopped.StopReasonCode)
	}
	assertProcessRuntimeSnapshotDoesNotLeakV0(t, stopped, req)
}

func TestProcessRuntimeConnectorV0KillFallidoTieneCodigoPublico(t *testing.T) {
	err := processRuntimeErrorV0(ProcessRuntimeKillFallidoV0, "process")
	if err.Code != ProcessRuntimeKillFallidoV0 || !err.Retryable {
		t.Fatalf("kill error publico invalido: %+v", err)
	}
}

type stopRuntimeTestResultV0 struct {
	snapshot ProcessRuntimeSnapshotV0
	err      error
}

func waitForProcessRuntimeChildFileV0(t *testing.T, dir string, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("child marker ausente: %s", name)
}
