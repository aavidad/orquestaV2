//go:build linux

package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestLegacyV6CgroupQuarantineDrainsSetsidBeforeTerminal(t *testing.T) {
	config := supervisorTestConfig(t)
	request := supervisorTestRequest(
		"legacy-v6-quarantine-cgroup",
		"helper:setsid-late-output-block",
		1024,
	)
	request.ReasoningEffort = governance.ReasoningEffortHigh
	adapter, record, runPath := startLiveV6CgroupProcess(t, config, request)
	descendant := captureExactTestProcess(t, awaitSetsidPID(t, config, request.ExecutionRef))

	if _, err := adapter.Launch(
		context.Background(), request,
	); ErrorCode(err) != CodeLegacyExecutionRequiresNewAttempt {
		t.Fatalf("Launch(V6 quarantine) error=%v code=%q", err, ErrorCode(err))
	}
	time.Sleep(350 * time.Millisecond)
	assertLegacyCgroupQuarantineClosed(
		t, adapter, config, request, record, descendant, runPath,
	)
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyV6CgroupQuarantineResumesIntentAfterRestartAndEscalates(t *testing.T) {
	config := supervisorTestConfig(t)
	request := supervisorTestRequest(
		"legacy-v6-quarantine-restart",
		"helper:setsid-late-output-block",
		1024,
	)
	request.ReasoningEffort = governance.ReasoningEffortXHigh
	owner, record, runPath := startLiveV6CgroupProcess(t, config, request)
	descendant := captureExactTestProcess(t, awaitSetsidPID(t, config, request.ExecutionRef))
	operation := detachLegacyQuarantineOwner(t, owner, request, record, true)

	if err := platformKillExactProcess(record); err != nil && !errors.Is(err, os.ErrProcessDone) {
		t.Fatal(err)
	}
	awaitExactSupervisorGone(t, record)
	if populated, err := owner.cgroups.populated(record); err != nil || !populated {
		t.Fatalf("restart fixture did not retain populated cgroup: populated=%v error=%v", populated, err)
	}
	if identity, err := platformInspectProcess(descendant); err != nil ||
		identity != processIdentityAlive {
		t.Fatalf("restart fixture descendant identity=%v error=%v", identity, err)
	}

	recoveryConfig := config
	recoveryConfig.RuntimeScope = ""
	recoveryConfig.SupervisorStartTimeout = 50 * time.Millisecond
	reopened, err := New(recoveryConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.BindRuntimeScope(context.Background(), config.RuntimeScope); err != nil {
		t.Fatalf("BindRuntimeScope(resume quarantine): %v", err)
	}
	time.Sleep(350 * time.Millisecond)
	assertLegacyCgroupQuarantineClosed(
		t, reopened, config, request, record, descendant, runPath,
	)
	closeDetachedLegacyOwner(t, owner, request, operation, config)
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyV6CgroupQuarantineHasSingleWinnerAcrossAdapters(t *testing.T) {
	config := supervisorTestConfig(t)
	request := supervisorTestRequest(
		"legacy-v6-quarantine-race",
		"helper:setsid-late-output-block",
		1024,
	)
	owner, record, runPath := startLiveV6CgroupProcess(t, config, request)
	descendant := captureExactTestProcess(t, awaitSetsidPID(t, config, request.ExecutionRef))
	operation := detachLegacyQuarantineOwner(t, owner, request, record, false)

	adapters := make([]*Adapter, 2)
	for index := range adapters {
		var err error
		adapters[index], err = New(config)
		if err != nil {
			t.Fatal(err)
		}
	}
	results := make([]error, len(adapters))
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range adapters {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			_, results[index] = adapters[index].Launch(context.Background(), request)
		}(index)
	}
	close(start)
	group.Wait()
	completed := 0
	for index, result := range results {
		switch ErrorCode(result) {
		case CodeLegacyExecutionRequiresNewAttempt:
			completed++
		case CodeProcessOwnershipBusy:
		default:
			t.Fatalf("adapter %d quarantine error=%v code=%q", index, result, ErrorCode(result))
		}
	}
	if completed == 0 {
		t.Fatalf("no adapter completed quarantine: %+v", results)
	}
	for index, adapter := range adapters {
		if _, err := adapter.Launch(
			context.Background(), request,
		); ErrorCode(err) != CodeLegacyExecutionRequiresNewAttempt {
			t.Fatalf("adapter %d replay error=%v code=%q", index, err, ErrorCode(err))
		}
	}
	time.Sleep(350 * time.Millisecond)
	assertLegacyCgroupQuarantineClosed(
		t, adapters[0], config, request, record, descendant, runPath,
	)
	closeDetachedLegacyOwner(t, owner, request, operation, config)
	for _, adapter := range adapters {
		if err := adapter.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLegacyCgroupQuarantineReleasesAdapterMutexWhileCleaning(t *testing.T) {
	config := supervisorTestConfig(t)
	request := supervisorTestRequest(
		"legacy-v6-quarantine-unlocked-cleanup",
		"helper:setsid-block",
		1024,
	)
	adapter, _, runPath := startLiveV6CgroupProcess(t, config, request)
	entered := make(chan struct{})
	release := make(chan struct{})
	adapter.beforeCgroupCleanup = func(candidate string) error {
		if candidate == runPath {
			close(entered)
			<-release
		}
		return nil
	}
	result := make(chan error, 1)
	go func() {
		_, err := adapter.Launch(context.Background(), request)
		result <- err
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("quarantine never reached cgroup cleanup")
	}
	capabilityContext, cancelCapabilities := context.WithTimeout(
		context.Background(), 250*time.Millisecond,
	)
	_, err := adapter.ControlCapabilities(capabilityContext)
	cancelCapabilities()
	if err != nil {
		t.Fatalf("adapter mutex remained blocked during cgroup cleanup: %v", err)
	}
	close(release)
	if err := <-result; ErrorCode(err) != CodeLegacyExecutionRequiresNewAttempt {
		t.Fatalf("quarantine result=%v code=%q", err, ErrorCode(err))
	}
	adapter.beforeCgroupCleanup = nil
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestForcedStopLegacyV4ThroughV6RejectsMissingFenceBeforePhysicalEffect(t *testing.T) {
	for _, schemaVersion := range []int{
		intermediateStateSchemaVersion,
		accountlessStateSchemaVersion,
		profileStateSchemaVersion,
	} {
		t.Run(schemaName(schemaVersion), func(t *testing.T) {
			config := supervisorTestConfig(t)
			request := supervisorTestRequest(
				"stop-live-cgroup-"+schemaName(schemaVersion),
				"helper:setsid-block",
				1024,
			)
			adapter, record, runPath := startLiveLegacyCgroupProcess(
				t, config, request, schemaVersion,
			)
			descendant := captureExactTestProcess(
				t, awaitSetsidPID(t, config, request.ExecutionRef),
			)
			launch, err := adapter.observationReceipt(
				mustLoadLaunchRecord(t, adapter, request.ExecutionRef), request.ExecutionRef,
			)
			if err != nil {
				t.Fatal(err)
			}
			stop := stopRequestForLaunch(
				launch, ports.AgentStopForced,
				"stop:live-cgroup:"+schemaName(schemaVersion),
			)
			stopContext, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
			receipt, err := adapter.Stop(stopContext, stop)
			cancelStop()
			if ports.AgentContractErrorCode(err) != "agent.stop_launch_action_fence_required" ||
				receipt != (ports.AgentStopReceipt{}) {
				t.Fatalf("Stop(V%d) receipt=%+v error=%v", schemaVersion, receipt, err)
			}
			for name, candidate := range map[string]processRecord{
				"supervisor": record,
				"descendant": descendant,
			} {
				if identity, err := platformInspectProcess(candidate); err != nil ||
					identity != processIdentityAlive {
					t.Fatalf("V%d %s identity=%v error=%v", schemaVersion, name, identity, err)
				}
			}
			requestName, receiptName := stopRecordNames(stop.IdempotencyKey)
			for _, name := range []string{requestName, receiptName} {
				if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), name)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("Stop(V%d) created %s: %v", schemaVersion, name, err)
				}
			}
			if _, err := os.Stat(filepath.Join(
				config.WorkRoot, filepath.FromSlash(runPath), quarantineIntentFileName,
			)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Stop(V%d) created private quarantine: %v", schemaVersion, err)
			}
			if _, err := os.Stat(filepath.Join(
				config.WorkRoot, filepath.FromSlash(runPath), launchUpgradeFileName,
			)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Stop(V%d) created V7 binding: %v", schemaVersion, err)
			}
			if err := adapter.Close(); err != nil {
				t.Fatal(err)
			}
			for name, candidate := range map[string]processRecord{"supervisor": record, "descendant": descendant} {
				if identity, err := platformInspectProcess(candidate); err != nil || identity != processIdentityGone {
					t.Fatalf("shutdown V%d %s identity=%v error=%v", schemaVersion, name, identity, err)
				}
			}
		})
	}
}

func startLiveV6CgroupProcess(
	t *testing.T,
	config Config,
	request ports.AgentLaunchRequest,
) (*Adapter, processRecord, string) {
	t.Helper()
	return startLiveLegacyCgroupProcess(
		t, config, request, profileStateSchemaVersion,
	)
}

func startLiveLegacyCgroupProcess(
	t *testing.T,
	config Config,
	request ports.AgentLaunchRequest,
	schemaVersion int,
) (*Adapter, processRecord, string) {
	t.Helper()
	runPath := executionPath(request.ExecutionRef)
	if schemaVersion == profileStateSchemaVersion {
		runPath = seedPersistedV6Launch(t, config, request)
	} else {
		seedAccountlessLaunchRecord(t, config, request, schemaVersion)
	}
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	record, _, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil || !found || record.SchemaVersion != schemaVersion {
		t.Fatalf("load V%d found=%v record=%+v error=%v", schemaVersion, found, record, err)
	}
	receipt, err := record.receipt(request.ExecutionRef)
	if err != nil {
		t.Fatal(err)
	}
	state := &executionState{
		requestHash: record.RequestHash, terminalRequestHash: record.RequestHash,
		receipt: receipt, maxOutput: record.MaxOutputBytes,
		artifactMediaType: request.ArtifactMediaType,
		runPath:           runPath, status: ports.AgentPending,
	}
	adapter.mu.Lock()
	adapter.executions[request.ExecutionRef.String()] = state
	start := adapter.startExecutionLocked(
		context.Background(), context.Background(), request, state,
		append([]string(nil), adapter.environment...), nil,
	)
	adapter.mu.Unlock()
	if start == nil {
		t.Fatal("legacy V6 supervisor did not start")
	}
	adapter.resolveExecutionStart(start)
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if state.status != ports.AgentRunning || state.process == nil ||
		state.process.SchemaVersion != cgroupProcessSchemaVersion ||
		state.ownerLock == nil {
		t.Fatalf("legacy V6 supervisor state=%+v", state)
	}
	return adapter, *state.process, runPath
}

func mustLoadLaunchRecord(
	t *testing.T,
	adapter *Adapter,
	executionRef goal.ExecutionRef,
) launchRecord {
	t.Helper()
	record, _, found, err := adapter.loadLaunchRecord(executionRef)
	if err != nil || !found {
		t.Fatalf("load launch found=%v error=%v", found, err)
	}
	return record
}

func detachLegacyQuarantineOwner(
	t *testing.T,
	adapter *Adapter,
	request ports.AgentLaunchRequest,
	record processRecord,
	persistIntent bool,
) *quarantineOperation {
	t.Helper()
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	state := adapter.executions[request.ExecutionRef.String()]
	if state == nil || state.process == nil || *state.process != record ||
		state.ownerLock == nil {
		t.Fatalf("legacy owner state=%+v record=%+v", state, record)
	}
	intent := quarantineIntent{ErrorCode: CodeLegacyExecutionRequiresNewAttempt}
	if persistIntent {
		var err error
		intent, err = adapter.ensureQuarantineIntent(
			state, record, CodeLegacyExecutionRequiresNewAttempt,
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	operation := &quarantineOperation{intent: intent, done: make(chan struct{})}
	state.quarantine = operation
	adapter.releaseProcessOwnershipLocked(state)
	return operation
}

func closeDetachedLegacyOwner(
	t *testing.T,
	adapter *Adapter,
	request ports.AgentLaunchRequest,
	operation *quarantineOperation,
	config Config,
) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		adapter.mu.Lock()
		waits := adapter.activeWaits
		adapter.mu.Unlock()
		if waits == 0 {
			break
		}
		if !time.Now().Before(deadline) {
			t.Fatal("detached legacy owner waiter did not settle")
		}
		time.Sleep(time.Millisecond)
	}
	terminal := readPersistedTerminal(
		t, config, executionPath(request.ExecutionRef),
	)
	adapter.mu.Lock()
	state := adapter.executions[request.ExecutionRef.String()]
	if state != nil {
		state.quarantine = nil
		state.status, state.terminal, state.terminalDurable =
			terminal.Status, &terminal, true
	}
	operation.err = &Error{Code: terminal.ErrorCode}
	close(operation.done)
	adapter.mu.Unlock()
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
}

func captureExactTestProcess(t *testing.T, pid int) processRecord {
	t.Helper()
	_, pgid, birth, err := readLinuxProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	bootID, err := linuxBootID()
	if err != nil {
		t.Fatal(err)
	}
	return processRecord{
		PID: pid, PGID: pgid, BootID: bootID, BirthMarker: birth,
	}
}

func assertLegacyCgroupQuarantineClosed(
	t *testing.T,
	adapter *Adapter,
	config Config,
	request ports.AgentLaunchRequest,
	record, descendant processRecord,
	runPath string,
) {
	t.Helper()
	for name, candidate := range map[string]processRecord{
		"supervisor": record,
		"descendant": descendant,
	} {
		if identity, err := platformInspectProcess(candidate); err != nil ||
			identity != processIdentityGone {
			t.Fatalf("%s identity=%v error=%v", name, identity, err)
		}
	}
	if _, err := adapter.cgroups.leafForRecord(record); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("quarantine left cgroup leaf: %v", err)
	}
	runRoot := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath))
	if output, err := os.ReadFile(filepath.Join(runRoot, lastMessageFileName)); err != nil ||
		len(output) != 0 {
		t.Fatalf("late output=%q error=%v", output, err)
	}
	if _, err := os.Stat(filepath.Join(runRoot, completionProofFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("quarantine retained completion proof: %v", err)
	}
	intent, found, err := adapter.loadQuarantineIntent(runPath, record)
	if err != nil || !found ||
		intent.ExecutionRef != request.ExecutionRef.String() ||
		intent.RequestHash != record.RequestHash ||
		intent.RuntimeScope != record.RuntimeScope ||
		intent.SupervisorInstance != record.SupervisorInstance ||
		intent.ErrorCode != CodeLegacyExecutionRequiresNewAttempt {
		t.Fatalf("quarantine intent=%+v found=%v error=%v", intent, found, err)
	}
	terminal := readPersistedTerminal(t, config, runPath)
	if terminal.RequestHash != record.RequestHash ||
		terminal.Status != ports.AgentFailed ||
		terminal.ErrorCode != CodeLegacyExecutionRequiresNewAttempt {
		t.Fatalf("quarantine terminal=%+v", terminal)
	}
	entries, err := os.ReadDir(runRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "stop-") {
			t.Fatalf("private quarantine created public Stop journal %s", entry.Name())
		}
	}
}
