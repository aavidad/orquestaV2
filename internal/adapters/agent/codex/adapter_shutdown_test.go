package codex

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"orquesta/internal/ports"
)

func TestAdapterShutdownCancelsBlockedSupervisorReadyBeforeMutex(t *testing.T) {
	adapter, err := New(testConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	operationContext, endOperation, err := adapter.beginOperation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = operationContext
	runContext, cancelRun := context.WithCancelCause(adapter.lifecycle)
	readyObserved := make(chan struct{})
	observedContext := &readyObservedContext{
		Context:  runContext,
		observed: readyObserved,
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		endOperation()
		t.Fatal(err)
	}
	defer writer.Close()
	state := &executionState{status: ports.AgentPending}
	start := &executionStart{
		state: state, runContext: observedContext, cancel: cancelRun,
		readyReader: reader, resolved: make(chan struct{}),
	}
	state.starting = start
	resolverDone := make(chan struct{})
	go func() {
		defer close(resolverDone)
		defer endOperation()
		adapter.resolveExecutionStart(start)
	}()
	select {
	case <-readyObserved:
	case <-time.After(time.Second):
		t.Fatal("READY wait was not installed")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), time.Second)
	defer cancelShutdown()
	if err := adapter.Shutdown(shutdownContext); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	select {
	case <-resolverDone:
	case <-time.After(time.Second):
		t.Fatal("blocked READY resolver did not cooperate with shutdown")
	}
	if !errors.Is(context.Cause(runContext), errAdapterShutdown) {
		t.Fatalf("run lifecycle cause = %v, want adapter shutdown", context.Cause(runContext))
	}
}

func TestAdapterShutdownDeadlineKeepsDescriptorsUntilOperationReleases(t *testing.T) {
	adapter, err := New(testConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	_, endOperation, err := adapter.beginOperation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	deadline, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := adapter.Shutdown(deadline); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("Shutdown() exceeded caller deadline by %s", elapsed)
	}
	if file, err := adapter.root.Open("."); err != nil {
		t.Fatalf("shutdown closed root while operation owned it: %v", err)
	} else {
		_ = file.Close()
	}

	endOperation()
	select {
	case <-adapter.shutdownDone:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finalize cooperatively after operation release")
	}
	if _, err := adapter.root.Open("."); err == nil {
		t.Fatal("shutdown left root descriptor usable after operation drain")
	}
}

func TestAdapterShutdownCancelsStopAndObserveWaitingForSupervisorReady(t *testing.T) {
	adapter, err := New(testConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	request := testRequest(t, "shutdown-stop-observe-ready", "helper:block", 1024)
	requestHash := mustRequestHash(t, request)
	record, runPath, _, err := adapter.ensureLaunchRecord(request, requestHash)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := record.receipt(request.ExecutionRef)
	if err != nil {
		t.Fatal(err)
	}
	runContext, cancelRun := context.WithCancelCause(adapter.lifecycle)
	readyObserved := make(chan struct{})
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	state := &executionState{
		requestHash: requestHash, terminalRequestHash: requestHash,
		receipt: receipt, maxOutput: request.MaxOutputBytes,
		runPath: runPath, status: ports.AgentPending, cancel: cancelRun,
	}
	start := &executionStart{
		state: state,
		runContext: &readyObservedContext{
			Context:  runContext,
			observed: readyObserved,
		},
		cancel: cancelRun, readyReader: reader, resolved: make(chan struct{}),
	}
	state.starting = start
	adapter.executions[request.ExecutionRef.String()] = state

	_, endLaunchOperation, err := adapter.beginOperation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	adapter.mu.Lock()
	adapter.beginExecutionWaitLocked()
	adapter.mu.Unlock()
	waiterDone := make(chan struct{})
	go func() {
		<-start.resolved
		adapter.endExecutionWait()
		close(waiterDone)
	}()
	resolverDone := make(chan struct{})
	go func() {
		defer close(resolverDone)
		defer endLaunchOperation()
		adapter.resolveExecutionStart(start)
	}()
	select {
	case <-readyObserved:
	case <-time.After(time.Second):
		t.Fatal("READY wait was not installed")
	}

	stop := stopRequestFromReceipt(
		receipt, ports.AgentStopForced, "stop:shutdown-stop-observe-ready",
	)
	stopResult := make(chan error, 1)
	go func() {
		_, stopErr := adapter.Stop(context.Background(), stop)
		stopResult <- stopErr
	}()
	observeResult := make(chan error, 1)
	go func() {
		_, observeErr := adapter.Observe(context.Background(), request.ExecutionRef)
		observeResult <- observeErr
	}()
	waitDeadline := time.Now().Add(time.Second)
	for {
		adapter.mu.Lock()
		operations := adapter.operations
		adapter.mu.Unlock()
		if operations == 3 {
			break
		}
		if !time.Now().Before(waitDeadline) {
			t.Fatal("Launch/Stop/Observe did not enter operation fence")
		}
		time.Sleep(time.Millisecond)
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), time.Second)
	defer cancelShutdown()
	if err := adapter.Shutdown(shutdownContext); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	for name, result := range map[string]<-chan error{
		"Stop": stopResult, "Observe": observeResult,
	} {
		if err := <-result; ErrorCode(err) != CodeUnavailable {
			t.Fatalf("%s error = %v code=%q, want unavailable", name, err, ErrorCode(err))
		}
	}
	for name, done := range map[string]<-chan struct{}{
		"resolver": resolverDone, "waiter": waiterDone,
	} {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("%s did not drain", name)
		}
	}
	adapter.mu.Lock()
	operations, waits := adapter.operations, adapter.activeWaits
	adapter.mu.Unlock()
	if operations != 0 || waits != 0 {
		t.Fatalf("shutdown owners remain: operations=%d waits=%d", operations, waits)
	}
	if _, err := adapter.root.Open("."); err == nil {
		t.Fatal("shutdown left root descriptor usable after all owners drained")
	}
}

func TestAdapterShutdownDeadlineClosesResourcesWhenAdoptedProcessStaysAlive(t *testing.T) {
	adapter, state := stuckAdoptedProcessAdapter(t)
	deadline, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	if err := adapter.Shutdown(deadline); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error = %v, want deadline exceeded", err)
	}
	assertShutdownFinalized(t, adapter, state)
}

type readyObservedContext struct {
	context.Context
	once     sync.Once
	observed chan struct{}
}

func stopRequestFromReceipt(
	launch ports.AgentLaunchReceipt,
	mode ports.AgentStopMode,
	idempotency string,
) ports.AgentStopRequest {
	return ports.AgentStopRequest{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef,
		WorkItemRef: launch.WorkItemRef, PlanGeneration: launch.PlanGeneration,
		AppSpecGeneration: launch.AppSpecGeneration, ExecutionAttempt: launch.ExecutionAttempt,
		LaunchActionFence: launch.LaunchActionFence,
		SpecHash:          launch.SpecHash, ProviderRef: launch.ProviderRef,
		ModelRef: launch.ModelRef, AgentRef: launch.AgentRef,
		ExternalRef: launch.ExternalRef, Mode: mode, IdempotencyKey: idempotency,
	}
}

func (ctx *readyObservedContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.observed) })
	return ctx.Context.Done()
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
