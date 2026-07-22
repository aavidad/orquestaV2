package codex

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"orquesta/internal/ports"
)

func TestAdapterShutdownDeadlineClosesResourcesWhenAdoptedProcessStaysAlive(t *testing.T) {
	adapter, state := stuckAdoptedProcessAdapter(t)
	deadline, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	if err := adapter.Shutdown(deadline); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error = %v, want deadline exceeded", err)
	}
	assertShutdownFinalized(t, adapter, state)
}

func TestAdapterShutdownConcurrentDeadlineClosesResourcesOnce(t *testing.T) {
	adapter, state := stuckAdoptedProcessAdapter(t)
	deadline, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	const callers = 16
	errorsByCaller := make(chan error, callers)
	var callersDone sync.WaitGroup
	callersDone.Add(callers)
	for range callers {
		go func() {
			defer callersDone.Done()
			errorsByCaller <- adapter.Shutdown(deadline)
		}()
	}
	callersDone.Wait()
	close(errorsByCaller)
	for err := range errorsByCaller {
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("concurrent Shutdown() error = %v, want deadline exceeded", err)
		}
	}
	assertShutdownFinalized(t, adapter, state)
}

func stuckAdoptedProcessAdapter(t *testing.T) (*Adapter, *executionState) {
	t.Helper()
	adapter, err := New(testConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	runPath := "executions/stuck-adopted-process"
	if err := adapter.ensurePrivateDirectory(runPath); err != nil {
		t.Fatal(err)
	}
	owner, err := adapter.acquireOwnerLock(runPath)
	if err != nil {
		t.Fatal(err)
	}
	record := processRecord{ExecutionRef: "execution:stuck-adopted-process", PID: 1, PGID: 1}
	state := &executionState{runPath: runPath, process: &record, ownerLock: owner}
	adapter.executions[record.ExecutionRef] = state
	adapter.shutdownSignal = func(processRecord, ports.AgentStopMode) error { return nil }
	adapter.shutdownInspect = func(processRecord) (bool, error) { return false, nil }
	return adapter, state
}

func assertShutdownFinalized(t *testing.T, adapter *Adapter, state *executionState) {
	t.Helper()
	select {
	case <-adapter.shutdownDone:
	case <-time.After(time.Second):
		t.Fatal("shutdown goroutine did not finish")
	}
	if _, err := adapter.root.Open("."); err == nil {
		t.Fatal("shutdown left root descriptor usable")
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if state.ownerLock != nil {
		t.Fatal("shutdown left adopted owner descriptor open")
	}
}
