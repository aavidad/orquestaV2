//go:build linux

package codex

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/ports"
)

func TestStopCanceledBeforeSupervisorReadyLeavesNaturalCompletionUnfenced(t *testing.T) {
	for _, mode := range []ports.AgentStopMode{
		ports.AgentStopCooperative,
		ports.AgentStopForced,
	} {
		t.Run(string(mode), func(t *testing.T) {
			config := testConfig(t)
			adapter, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			request := testRequest(t, "pre-ready-"+string(mode), "helper:success", 1024)
			requestHash := mustRequestHash(t, request)
			record, runPath, _, err := adapter.ensureLaunchRecord(request, requestHash)
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := record.receipt(request.ExecutionRef)
			if err != nil {
				t.Fatal(err)
			}
			if err := adapter.prepareRuntimeFiles(runPath); err != nil {
				t.Fatal(err)
			}
			if err := adapter.writePrivateRuntimeFile(
				runPath+"/"+lastMessageFileName, []byte(`{"artifact":"natural"}`),
			); err != nil {
				t.Fatal(err)
			}
			pgid, bootID, birthMarker, err := platformCaptureProcess(os.Getpid())
			if err != nil {
				t.Fatal(err)
			}
			owner, err := adapter.acquireOwnerLock(runPath)
			if err != nil {
				t.Fatal(err)
			}
			process := &processRecord{
				SchemaVersion: processSchemaVersion,
				ExecutionRef:  request.ExecutionRef.String(),
				RequestHash:   requestHash,
				RuntimeScope:  config.RuntimeScope,
				PID:           os.Getpid(),
				PGID:          pgid,
				BootID:        bootID,
				BirthMarker:   birthMarker,
			}
			state := &executionState{
				requestHash:         requestHash,
				terminalRequestHash: requestHash,
				receipt:             receipt,
				maxOutput:           request.MaxOutputBytes,
				artifactMediaType:   request.ArtifactMediaType,
				runPath:             runPath,
				status:              ports.AgentPending,
				process:             process,
				ownerLock:           owner,
				cancel:              func(error) {},
			}
			start := &executionStart{state: state, resolved: make(chan struct{})}
			state.starting = start
			adapter.executions[request.ExecutionRef.String()] = state
			var signals atomic.Int32
			adapter.shutdownSignal = func(processRecord, ports.AgentStopMode) error {
				signals.Add(1)
				return nil
			}

			stop := stopRequestForLaunch(receipt, mode, "stop:pre-ready:"+string(mode))
			stopContext, cancelStop := context.WithCancel(context.Background())
			stopDone := make(chan ports.AgentStopReceipt, 1)
			stopErrors := make(chan error, 1)
			go func() {
				stopReceipt, stopErr := adapter.Stop(stopContext, stop)
				stopDone <- stopReceipt
				stopErrors <- stopErr
			}()

			// Observe waiting through the same public STARTING fence. Neither
			// operation may inspect or signal the pre-READY supervisor.
			observeContext, cancelObserve := context.WithCancel(context.Background())
			observeErrors := make(chan error, 1)
			go func() {
				_, observeErr := adapter.Observe(observeContext, request.ExecutionRef)
				observeErrors <- observeErr
			}()
			waitDeadline := time.Now().Add(time.Second)
			for {
				adapter.mu.Lock()
				operations := adapter.operations
				adapter.mu.Unlock()
				if operations == 2 {
					break
				}
				if !time.Now().Before(waitDeadline) {
					t.Fatal("Stop/Observe did not enter public operation fence")
				}
				time.Sleep(time.Millisecond)
			}
			cancelStop()
			cancelObserve()
			stopReceipt := <-stopDone
			if stopErr := <-stopErrors; stopErr != nil {
				t.Fatalf("Stop() error = %v", stopErr)
			}
			if stopReceipt.Status != ports.AgentStopPending {
				t.Fatalf("Stop() receipt = %+v, want pending", stopReceipt)
			}
			if observeErr := <-observeErrors; observeErr != context.Canceled {
				t.Fatalf("Observe() error = %v, want canceled", observeErr)
			}
			if signals.Load() != 0 {
				t.Fatalf("pre-READY signals = %d, want zero", signals.Load())
			}
			stopHash, err := hashStopRequest(stop)
			if err != nil {
				t.Fatal(err)
			}
			if _, found, err := adapter.loadStopRequest(runPath, stopHash, stop); err != nil || found {
				t.Fatalf("pre-READY stop journal found=%v error=%v", found, err)
			}

			adapter.mu.Lock()
			state.starting = nil
			close(start.resolved)
			adapter.completeExecutionLocked(request, state, nil, nil, nil, nil, nil, false)
			if state.terminal == nil || state.status != ports.AgentCompleted {
				adapter.mu.Unlock()
				t.Fatalf("natural completion after canceled pre-READY stop = %+v", state.terminal)
			}
			delete(adapter.executions, request.ExecutionRef.String())
			adapter.mu.Unlock()
			if err := adapter.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
