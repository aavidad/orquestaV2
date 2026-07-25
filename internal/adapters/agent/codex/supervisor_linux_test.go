//go:build linux

package codex

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

const (
	supervisorOwnerTestArgument    = "__orquesta_test_codex_supervisor_owner_v1"
	supervisorObserverTestArgument = "__orquesta_test_codex_supervisor_observer_v1"
	supervisorOwnerReadyFileName   = "owner-ready"
)

func TestSupervisorSurvivesAbruptOwnerExitAndDoubleRestart(t *testing.T) {
	config := supervisorTestConfig(t)
	const suffix = "supervisor-owner-crash"
	objective := "helper:delayed-success"
	if strings.Contains(suffix, "diagnostic") {
		objective = "helper:delayed-failure"
	}
	request := supervisorTestRequest(suffix, objective, 1024)

	runSupervisorTestSubprocess(t, supervisorOwnerTestArgument, config.WorkRoot, suffix, request.Objective)
	recordBefore := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
	if recordBefore.SchemaVersion != cgroupProcessSchemaVersion ||
		!validSupervisorToken(recordBefore.SupervisorInstance, "supervisor:") {
		t.Fatalf("supervisor process record = %+v", recordBefore)
	}
	if identity, err := platformInspectProcess(recordBefore); err != nil || identity != processIdentityAlive {
		t.Fatalf("supervisor after owner SIGKILL-equivalent exit: identity=%v error=%v", identity, err)
	}
	assertSupervisorProcessIsSecretFree(t, recordBefore)

	// A second server adopts the same live supervisor and then disappears too.
	runSupervisorTestSubprocess(t, supervisorObserverTestArgument, config.WorkRoot, suffix, request.Objective)
	recordAfterSecond := readSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
	if recordAfterSecond != recordBefore {
		t.Fatalf("second restart replaced supervisor: before=%+v after=%+v", recordBefore, recordAfterSecond)
	}

	third := openTestAdapter(t, config)
	observation := awaitTerminal(t, third, request.ExecutionRef)
	if observation.Status != ports.AgentCompleted ||
		string(observation.Content) != "artifact:delayed-success" {
		t.Fatalf("adopted natural completion = %+v", observation)
	}
	recordAfterCompletion := readSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
	if recordAfterCompletion != recordBefore {
		t.Fatalf("completion changed supervisor identity: before=%+v after=%+v", recordBefore, recordAfterCompletion)
	}
	invocations, err := os.ReadFile(filepath.Join(
		config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), "helper-invocations",
	))
	if err != nil || len(invocations) != 1 {
		t.Fatalf("worker was relaunched: invocations=%d error=%v", len(invocations), err)
	}
	if identity, err := platformInspectProcess(recordBefore); err != nil || identity != processIdentityGone {
		t.Fatalf("completed supervisor remains live: identity=%v error=%v", identity, err)
	}
}

func TestSupervisorIndependentTimeoutPublishesCausalProof(t *testing.T) {
	config := supervisorTestConfig(t)
	config.Timeout = 50 * time.Millisecond
	adapter := openTestAdapter(t, config)
	request := supervisorTestRequest("supervisor-timeout", "helper:block", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	record := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionTimeout {
		t.Fatalf("timeout observation = %+v", observation)
	}
	if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityGone {
		t.Fatalf("timeout left supervisor group live: identity=%v error=%v", identity, err)
	}
}

func TestSupervisorCgroupDrainsSetsidNaturalAndPreservesExternalSibling(t *testing.T) {
	config := supervisorTestConfig(t)
	setsid, err := exec.LookPath("setsid")
	if err != nil {
		t.Skipf("setsid unavailable: %v", err)
	}
	sibling := exec.Command(setsid, "/bin/sleep", "30")
	if err := sibling.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sibling.Process.Kill()
		_, _ = sibling.Process.Wait()
	})

	adapter := openTestAdapter(t, config)
	request := supervisorTestRequest(
		"setsid-natural", "helper:setsid-background-success", 1024,
	)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	descendant := awaitSetsidPID(t, config, request.ExecutionRef)
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentCompleted ||
		string(observation.Content) != "artifact:setsid-background-success" {
		t.Fatalf("setsid natural observation = %+v", observation)
	}
	assertProcessGoneWithESRCH(t, descendant)
	if err := sibling.Process.Signal(os.Signal(syscall.Signal(0))); err != nil {
		t.Fatalf("external sibling was signaled: %v", err)
	}
}

func TestSupervisorCgroupDrainsSetsidTimeout(t *testing.T) {
	config := supervisorTestConfig(t)
	config.Timeout = 750 * time.Millisecond
	adapter := openTestAdapter(t, config)
	request := supervisorTestRequest("setsid-timeout", "helper:setsid-block", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	descendant := awaitSetsidPID(t, config, request.ExecutionRef)
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionTimeout {
		t.Fatalf("setsid timeout observation = %+v", observation)
	}
	assertProcessGoneWithESRCH(t, descendant)
}

func TestSupervisorCgroupForcedFallbackAfterCooperativeLeavesSetsid(t *testing.T) {
	config := supervisorTestConfig(t)
	config.ProcessPipeDrainDelay = 50 * time.Millisecond
	adapter := openTestAdapter(t, config)
	request := supervisorTestRequest("setsid-cooperative-forced", "helper:setsid-block", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	descendant := awaitSetsidPID(t, config, request.ExecutionRef)
	record := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
	launch := launchReceiptForRequest(t, adapter, request)
	cooperative := stopRequestForLaunch(
		launch, ports.AgentStopCooperative, "stop:setsid-cooperative",
	)
	if receipt, err := adapter.Stop(context.Background(), cooperative); err != nil ||
		receipt.Status != ports.AgentStopPending {
		t.Fatalf("cooperative Stop = %+v, %v", receipt, err)
	}
	awaitExactSupervisorGone(t, record)
	forced := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:setsid-forced")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	receipt, err := adapter.Stop(ctx, forced)
	if err != nil || receipt.Status != ports.AgentStopped {
		t.Fatalf("forced fallback Stop = %+v, %v", receipt, err)
	}
	assertProcessGoneWithESRCH(t, descendant)
}

func TestSupervisorImmediateStopAfterReadyIsSignalSafe(t *testing.T) {
	for _, mode := range []ports.AgentStopMode{ports.AgentStopCooperative, ports.AgentStopForced} {
		for iteration := 0; iteration < 4; iteration++ {
			config := supervisorTestConfig(t)
			adapter := openTestAdapter(t, config)
			suffix := fmt.Sprintf("immediate-%s-%d", mode, iteration)
			request := supervisorTestRequest(suffix, "helper:block", 1024)
			if _, err := adapter.Launch(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			launch := launchReceiptForRequest(t, adapter, request)
			stop := stopRequestForLaunch(launch, mode, "stop:"+suffix)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			receipt, err := adapter.Stop(ctx, stop)
			cancel()
			if err != nil {
				t.Fatalf("immediate %s Stop: %v", mode, err)
			}
			if mode == ports.AgentStopForced && receipt.Status != ports.AgentStopped {
				t.Fatalf("immediate forced receipt = %+v", receipt)
			}
			if mode == ports.AgentStopCooperative && receipt.Status == ports.AgentStopPending {
				forced := stopRequestForLaunch(
					launch, ports.AgentStopForced, "stop:"+suffix+":forced",
				)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, err = adapter.Stop(ctx, forced)
				cancel()
				if err != nil {
					t.Fatalf(
						"immediate cooperative cleanup: %v code=%q cause=%v",
						err, ErrorCode(err), errors.Unwrap(err),
					)
				}
			}
		}
	}
}

func TestSupervisorWorkerDoesNotInheritPrivateControlDescriptors(t *testing.T) {
	config := supervisorTestConfig(t)
	adapter := openTestAdapter(t, config)
	request := supervisorTestRequest("worker-fd-audit", "helper:fd-audit-block", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	runDirectory := filepath.Join(
		config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)),
	)
	auditPath := filepath.Join(runDirectory, workerFDAuditFile)
	var audit struct {
		PID     int               `json:"pid"`
		Targets map[string]string `json:"targets"`
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		payload, err := os.ReadFile(auditPath)
		if err == nil && json.Unmarshal(payload, &audit) == nil && audit.PID > 0 {
			break
		}
		if !time.Now().Before(deadline) {
			t.Fatal("worker fd audit unavailable")
		}
		time.Sleep(time.Millisecond)
	}
	record := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
	privateTargets := make(map[string]struct{})
	for _, descriptor := range []string{"5", "7"} {
		target, err := os.Readlink(filepath.Join(
			"/proc", strconv.Itoa(record.PID), "fd", descriptor,
		))
		if err != nil {
			t.Fatalf("supervisor private fd %s unavailable: %v", descriptor, err)
		}
		privateTargets[target] = struct{}{}
	}
	for descriptor, target := range audit.Targets {
		if _, private := privateTargets[target]; private {
			t.Fatalf("worker inherited supervisor fd %s target %q", descriptor, target)
		}
		if strings.Contains(target, "memfd:orquesta-codex-supervisor") ||
			strings.HasPrefix(target, "/sys/fs/cgroup") {
			t.Fatalf("worker inherited private fd %s target %q", descriptor, target)
		}
		if (descriptor == "5" || descriptor == "8") && strings.HasPrefix(target, "pipe:[") {
			t.Fatalf("worker inherited diagnostic/READY pipe on fd %s: %q", descriptor, target)
		}
	}

	launch := launchReceiptForRequest(t, adapter, request)
	stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:worker-fd-audit")
	stopContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if receipt, err := adapter.Stop(stopContext, stop); err != nil ||
		receipt.Status != ports.AgentStopped {
		t.Fatalf("forced cleanup = %+v, %v", receipt, err)
	}
}

func TestSupervisorForcedIntentCrashRecoversPopulatedAndEmptyLeaf(t *testing.T) {
	for _, emptyBeforeRestart := range []bool{false, true} {
		name := "populated"
		if emptyBeforeRestart {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			config := supervisorTestConfig(t)
			suffix := "forced-intent-" + name
			request := supervisorTestRequest(suffix, "helper:block", 1024)
			runSupervisorTestSubprocess(
				t, supervisorOwnerTestArgument, config.WorkRoot, suffix, request.Objective,
			)
			record := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
			t.Cleanup(func() {
				_ = unix.Kill(record.PID, unix.SIGKILL)
				if controller, err := openCodexCgroupRoot(config.CgroupRoot); err == nil {
					_ = controller.kill(record)
					_ = controller.remove(record)
					_ = controller.close()
				}
			})

			seeder, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			launchRecord, runPath, found, err := seeder.loadLaunchRecord(request.ExecutionRef)
			if err != nil || !found {
				t.Fatalf("load launch found=%v error=%v", found, err)
			}
			launch, err := launchRecord.receipt(request.ExecutionRef)
			if err != nil {
				t.Fatal(err)
			}
			cooperative := stopRequestForLaunch(
				launch, ports.AgentStopCooperative, "stop:stale-cooperative-"+name,
			)
			cooperativeHash, err := hashStopRequest(cooperative)
			if err != nil {
				t.Fatal(err)
			}
			stop := stopRequestForLaunch(
				launch, ports.AgentStopForced, "stop:forced-intent-"+name,
			)
			stopHash, err := hashStopRequest(stop)
			if err != nil {
				t.Fatal(err)
			}
			seeder.mu.Lock()
			state, _, err := seeder.controlTargetLocked(stop)
			if err == nil {
				_, err = seeder.ensureStopRequest(runPath, cooperativeHash, cooperative)
			}
			if err == nil {
				var cooperativeIntent stopSignalIntent
				var created bool
				cooperativeIntent, created, err = seeder.prepareStopSignalIntent(
					runPath, cooperativeHash, cooperative,
				)
				if err == nil && !created {
					err = errors.New("stale cooperative intent was not created")
				}
				if err == nil {
					_, err = seeder.persistStopSignalProof(runPath, cooperativeIntent)
				}
			}
			if err == nil {
				_, err = seeder.ensureStopRequest(runPath, stopHash, stop)
			}
			if err == nil {
				_, _, err = seeder.ownProcessLocked(state)
			}
			if err == nil {
				var created bool
				_, created, err = seeder.prepareStopSignalIntent(runPath, stopHash, stop)
				if err == nil && !created {
					err = errors.New("forced signal intent was not created")
				}
			}
			seeder.releaseProcessOwnershipLocked(state)
			seeder.mu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			if emptyBeforeRestart {
				if err := seeder.cgroups.kill(record); err != nil {
					t.Fatal(err)
				}
				awaitExactSupervisorGone(t, record)
				if err := seeder.cgroups.drain(record, config.SupervisorStartTimeout); err != nil {
					t.Fatal(err)
				}
			}
			if err := seeder.root.Close(); err != nil {
				t.Fatal(err)
			}
			if err := seeder.cgroups.close(); err != nil {
				t.Fatal(err)
			}

			recoveryConfig := config
			recoveryConfig.RuntimeScope = ""
			reopened, err := New(recoveryConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := reopened.BindRuntimeScope(config.RuntimeScope); err != nil {
				t.Fatal(err)
			}
			stopContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			receipt, err := reopened.Stop(stopContext, stop)
			cancel()
			if err != nil || (receipt.Status != ports.AgentStopped &&
				receipt.Status != ports.AgentStopAlreadyStopped) {
				t.Fatalf("recovered forced intent = %+v, %v", receipt, err)
			}
			if _, found, err := reopened.loadStopSignalProof(runPath, stopHash, stop); err != nil || !found {
				t.Fatalf("forced proof found=%v error=%v", found, err)
			}
			winner, found, err := reopened.loadWinningStopSignalProof(runPath)
			if err != nil || !found || winner.Mode != ports.AgentStopForced ||
				winner.RequestHash != stopHash {
				t.Fatalf("winning proof = %+v found=%v error=%v", winner, found, err)
			}
			if _, err := reopened.cgroups.leafForRecord(record); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("terminal forced intent left leaf: %v", err)
			}
			if err := reopened.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProductionCgroupAdapterRejectsLegacyProcessSchemas(t *testing.T) {
	adapter := openTestAdapter(t, supervisorTestConfig(t))
	for _, schema := range []int{processSchemaVersion, supervisedProcessSchemaVersion} {
		t.Run(strconv.Itoa(schema), func(t *testing.T) {
			runPath := "executions/legacy-cgroup-schema-" + strconv.Itoa(schema)
			if err := adapter.ensurePrivateDirectory(runPath); err != nil {
				t.Fatal(err)
			}
			var err error
			record := processRecord{
				SchemaVersion: schema, ExecutionRef: "execution:legacy-cgroup-schema",
				RequestHash: "sha256:legacy-cgroup-schema", RuntimeScope: adapter.config.RuntimeScope,
				PID: 1, PGID: 1, BootID: "boot", BirthMarker: "birth",
			}
			if schema == supervisedProcessSchemaVersion {
				record.SupervisorInstance, err = newSupervisorInstance()
				if err != nil {
					t.Fatal(err)
				}
				record.CompletionPublicKey, _, err = newCompletionSigningKey()
				if err != nil {
					t.Fatal(err)
				}
			}
			payload, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := adapter.writePrivateRuntimeFile(
				filepath.ToSlash(filepath.Join(runPath, processFileName)), payload,
			); err != nil {
				t.Fatal(err)
			}
			if baseline, found, err := adapter.readProcessRecord(runPath); err != nil ||
				!found || baseline != record {
				t.Fatalf("baseline schema %d found=%v record=%+v error=%v", schema, found, baseline, err)
			}
			if err := adapter.root.Remove(
				filepath.ToSlash(filepath.Join(runPath, processFileName)),
			); err != nil {
				t.Fatal(err)
			}
			record.CgroupName = "forbidden-cgroup-field"
			payload, err = json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := adapter.writePrivateRuntimeFile(
				filepath.ToSlash(filepath.Join(runPath, processFileName)), payload,
			); err != nil {
				t.Fatal(err)
			}
			if _, found, err := adapter.readProcessRecord(runPath); found ||
				ErrorCode(err) != CodeProcessOwnershipInvalid {
				t.Fatalf("read schema %d found=%v error=%v code=%q", schema, found, err, ErrorCode(err))
			}
			if err := adapter.root.Remove(
				filepath.ToSlash(filepath.Join(runPath, processFileName)),
			); err != nil {
				t.Fatal(err)
			}
			if _, err := adapter.inspectProcessTree(record); ErrorCode(err) != CodeControlUnsupported {
				t.Fatalf("inspect schema %d error=%v code=%q", schema, err, ErrorCode(err))
			}
			if err := adapter.signalProcessTree(record, ports.AgentStopForced); ErrorCode(err) != CodeControlUnsupported {
				t.Fatalf("signal schema %d error=%v code=%q", schema, err, ErrorCode(err))
			}
		})
	}
}

func TestTerminalCgroupCleanupFailureRetriesAfterRestart(t *testing.T) {
	config := supervisorTestConfig(t)
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	request := supervisorTestRequest("cleanup-restart", "helper:success", 1024)
	requestHash := mustRequestHash(t, request)
	launchRecord, runPath, _, err := adapter.ensureLaunchRecord(request, requestHash)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := adapter.allocateExecutionCgroup(runPath)
	if err != nil {
		t.Fatal(err)
	}
	record := processRecord{SchemaVersion: cgroupProcessSchemaVersion}
	adapter.cgroups.populateRecord(&record, leaf)
	if err := leaf.close(); err != nil {
		t.Fatal(err)
	}
	terminal := terminalRecord{
		SchemaVersion: stateSchemaVersion, RequestHash: launchRecord.RequestHash,
		Status: ports.AgentCompleted, MediaType: request.ArtifactMediaType,
		Artifact: "terminal-before-cleanup", ObservedAt: config.Now(),
	}
	if _, err := adapter.persistTerminal(
		runPath, terminal, launchRecord.SpecHash, launchRecord.MaxOutputBytes,
	); err != nil {
		t.Fatal(err)
	}

	// The terminal is already durable. Losing the root cgroup descriptor must
	// fail cleanup without deleting the boundary journals or terminal.
	if err := adapter.cgroups.root.Close(); err != nil {
		t.Fatal(err)
	}
	if err := adapter.cleanupTerminalCgroup(&executionState{runPath: runPath}); err == nil {
		t.Fatal("cleanup unexpectedly succeeded with closed cgroup root")
	}
	if _, err := os.Stat(filepath.Join(
		config.WorkRoot, filepath.FromSlash(runPath), terminalFileName,
	)); err != nil {
		t.Fatalf("durable terminal lost after cleanup failure: %v", err)
	}
	for _, name := range []string{cgroupAllocationFileName, cgroupBoundaryFileName} {
		if _, err := os.Stat(filepath.Join(
			config.WorkRoot, filepath.FromSlash(runPath), name,
		)); err != nil {
			t.Fatalf("cleanup journal %s lost after failure: %v", name, err)
		}
	}
	_ = adapter.root.Close()
	_ = adapter.cgroups.close()

	recoveryConfig := config
	recoveryConfig.RuntimeScope = ""
	reopened, err := New(recoveryConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.BindRuntimeScope(config.RuntimeScope); err != nil {
		t.Fatal(err)
	}
	observation, err := reopened.Observe(context.Background(), request.ExecutionRef)
	if err != nil || observation.Status != ports.AgentCompleted ||
		string(observation.Content) != terminal.Artifact {
		t.Fatalf("terminal after cleanup restart = %+v, %v", observation, err)
	}
	if _, err := reopened.cgroups.leafForRecord(record); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("restart left terminal cgroup leaf: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCgroupLaunchRollbackStagesPersistTerminalBeforeCleanupAndRestart(t *testing.T) {
	for _, stage := range []string{"pre_gate", "start", "persist", "ready"} {
		t.Run(stage, func(t *testing.T) {
			config := supervisorTestConfig(t)
			config.launchFailureStageForTests = stage
			adapter, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			baselineDescriptors := processFileDescriptorTargets(t)
			request := supervisorTestRequest("rollback-"+stage, "helper:block", 1024)
			runPath := executionPath(request.ExecutionRef)
			injectedCleanup := errors.New("injected cleanup pause")
			var terminalBeforeCleanup atomic.Bool
			adapter.beforeCgroupCleanup = func(candidate string) error {
				if candidate != runPath {
					return nil
				}
				if _, err := adapter.root.Lstat(
					filepath.ToSlash(filepath.Join(runPath, cgroupAllocationFileName)),
				); errors.Is(err, os.ErrNotExist) {
					// Initial stale-allocation preflight before this launch has
					// published any cgroup resource.
					return nil
				}
				if _, err := adapter.root.Lstat(
					filepath.ToSlash(filepath.Join(runPath, terminalFileName)),
				); err == nil {
					terminalBeforeCleanup.Store(true)
				}
				return injectedCleanup
			}
			if _, err := adapter.Launch(context.Background(), request); err != nil {
				t.Fatalf("Launch(%s) error=%v", stage, err)
			}

			deadline := time.Now().Add(5 * time.Second)
			for {
				adapter.mu.Lock()
				waits := adapter.activeWaits
				state := adapter.executions[request.ExecutionRef.String()]
				durable := state != nil && state.terminal != nil && state.terminalDurable
				adapter.mu.Unlock()
				if durable && waits == 0 {
					break
				}
				if !time.Now().Before(deadline) {
					t.Fatalf("rollback %s did not settle durable terminal", stage)
				}
				time.Sleep(time.Millisecond)
			}
			if !terminalBeforeCleanup.Load() {
				t.Fatalf("rollback %s attempted cleanup before terminal durability", stage)
			}
			var boundary cgroupBoundaryRecord
			found, err := adapter.readPrivateJSON(
				filepath.ToSlash(filepath.Join(runPath, cgroupBoundaryFileName)), &boundary,
			)
			if err != nil || !found {
				t.Fatalf("rollback %s boundary found=%v error=%v", stage, found, err)
			}
			cgroupRecord := processRecord{
				SchemaVersion:    cgroupProcessSchemaVersion,
				CgroupName:       boundary.Name,
				CgroupRootDevice: boundary.RootDevice, CgroupRootInode: boundary.RootInode,
				CgroupControlDevice: boundary.ControlDevice, CgroupControlInode: boundary.ControlInode,
				CgroupDevice: boundary.Device, CgroupInode: boundary.Inode,
			}
			leaf, err := adapter.cgroups.leafForRecord(cgroupRecord)
			if err != nil {
				t.Fatalf("rollback %s lost leaf before retry: %v", stage, err)
			}
			populated, populatedErr := cgroupPopulated(int(leaf.Fd()))
			empty, processesErr := cgroupProcessesEmpty(int(leaf.Fd()))
			closeErr := leaf.Close()
			if populatedErr != nil || processesErr != nil || closeErr != nil ||
				populated || !empty {
				t.Fatalf(
					"rollback %s leaf not empty before restart: populated=%v empty=%v errors=%v",
					stage, populated, empty,
					errors.Join(populatedErr, processesErr, closeErr),
				)
			}
			if stage == "ready" {
				process := readSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
				if identity, err := platformInspectProcess(process); err != nil ||
					identity != processIdentityGone {
					t.Fatalf(
						"rollback READY supervisor remains: identity=%v error=%v record=%+v",
						identity, err, process,
					)
				}
			}
			for _, name := range []string{
				terminalFileName, cgroupAllocationFileName, cgroupBoundaryFileName,
			} {
				if _, err := adapter.root.Lstat(
					filepath.ToSlash(filepath.Join(runPath, name)),
				); err != nil {
					t.Fatalf("rollback %s census missing %s: %v", stage, name, err)
				}
			}
			adapter.mu.Lock()
			state := adapter.executions[request.ExecutionRef.String()]
			adapter.releaseProcessOwnershipLocked(state)
			operations, waits := adapter.operations, adapter.activeWaits
			adapter.mu.Unlock()
			if operations != 0 || waits != 0 {
				t.Fatalf("rollback %s owners remain: operations=%d waits=%d", stage, operations, waits)
			}
			settledDescriptors := processFileDescriptorTargets(t)
			if !slices.Equal(settledDescriptors, baselineDescriptors) {
				t.Fatalf(
					"rollback %s leaked descriptors: baseline=%v settled=%v",
					stage, baselineDescriptors, settledDescriptors,
				)
			}
			if err := adapter.root.Close(); err != nil {
				t.Fatal(err)
			}
			if err := adapter.cgroups.close(); err != nil {
				t.Fatal(err)
			}

			recoveryConfig := config
			recoveryConfig.RuntimeScope = ""
			recoveryConfig.launchFailureStageForTests = ""
			reopened, err := New(recoveryConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := reopened.BindRuntimeScope(config.RuntimeScope); err != nil {
				t.Fatal(err)
			}
			observation, err := reopened.Observe(context.Background(), request.ExecutionRef)
			if err != nil || observation.Status != ports.AgentFailed {
				t.Fatalf("rollback %s restart terminal=%+v error=%v", stage, observation, err)
			}
			if _, err := reopened.cgroups.leafForRecord(cgroupRecord); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("rollback %s restart left leaf: %v", stage, err)
			}
			for _, name := range []string{cgroupAllocationFileName, cgroupBoundaryFileName} {
				if _, err := reopened.root.Lstat(
					filepath.ToSlash(filepath.Join(runPath, name)),
				); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("rollback %s restart left %s: %v", stage, name, err)
				}
			}
			if err := reopened.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func awaitSetsidPID(t *testing.T, config Config, execution goal.ExecutionRef) int {
	t.Helper()
	filePath := filepath.Join(
		config.WorkRoot, filepath.FromSlash(executionPath(execution)), setsidPIDFile,
	)
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		payload, err := os.ReadFile(filePath)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(payload)))
			if parseErr == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("setsid descendant PID unavailable for %s", execution)
	return 0
}

func awaitExactSupervisorGone(t *testing.T, record processRecord) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		identity, err := platformInspectProcess(record)
		if err == nil && identity == processIdentityGone {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("supervisor remained alive: %+v", record)
}

func TestSupervisorProofFailsClosedAfterAbruptOwnerExit(t *testing.T) {
	tests := map[string]func(*testing.T, string){
		"missing": func(t *testing.T, proofPath string) {
			result, err := os.ReadFile(filepath.Join(filepath.Dir(proofPath), lastMessageFileName))
			if err != nil || len(result) == 0 {
				t.Fatalf("missing-proof negative lacks tempting result: bytes=%d error=%v", len(result), err)
			}
			if err := os.Remove(proofPath); err != nil {
				t.Fatal(err)
			}
		},
		"truncated": func(t *testing.T, proofPath string) {
			if err := os.WriteFile(proofPath, []byte(`{"schema_version":1`), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"stale_request": func(t *testing.T, proofPath string) {
			proof := readCompletionProofFile(t, proofPath)
			proof.RequestHash = "sha256:" + strings.Repeat("0", 64)
			writeCompletionProofFile(t, proofPath, proof)
		},
		"tampered_result": func(t *testing.T, proofPath string) {
			runDirectory := filepath.Dir(proofPath)
			resultPath := filepath.Join(runDirectory, lastMessageFileName)
			file, err := os.OpenFile(resultPath, os.O_WRONLY|os.O_APPEND, 0o600)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write([]byte("tampered")); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		},
		"forged_signature": func(t *testing.T, proofPath string) {
			proof := readCompletionProofFile(t, proofPath)
			proof.Signature = base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
			writeCompletionProofFile(t, proofPath, proof)
		},
		"unresolved_stop_request": func(t *testing.T, proofPath string) {
			stopPath := filepath.Join(filepath.Dir(proofPath), "stop-fence.request.json")
			if err := os.WriteFile(stopPath, []byte("{}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"orphan_signal_intent": func(t *testing.T, proofPath string) {
			stopPath := filepath.Join(filepath.Dir(proofPath), "stop-orphan.signal-intent.json")
			if err := os.WriteFile(stopPath, []byte("{}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := supervisorTestConfig(t)
			suffix := "proof-" + name
			request := supervisorTestRequest(suffix, "helper:delayed-success", 1024)
			runSupervisorTestSubprocess(t, supervisorOwnerTestArgument, config.WorkRoot, suffix, request.Objective)
			record := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
			proofPath := awaitCompletionProofAndGone(t, config.WorkRoot, request.ExecutionRef, record)
			mutate(t, proofPath)

			restarted := openTestAdapter(t, config)
			observation, err := restarted.Observe(context.Background(), request.ExecutionRef)
			if err != nil {
				t.Fatalf("Observe() error = %v cause=%v", err, errors.Unwrap(err))
			}
			if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionInterrupted {
				t.Fatalf("invalid proof was accepted: %+v", observation)
			}
		})
	}
}

func TestSupervisorStopProofWinsNaturalCompletionProof(t *testing.T) {
	config := supervisorTestConfig(t)
	const suffix = "stop-proof-wins-completion"
	request := supervisorTestRequest(suffix, "helper:delayed-success", 1024)
	runSupervisorTestSubprocess(t, supervisorOwnerTestArgument, config.WorkRoot, suffix, request.Objective)
	record := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
	_ = awaitCompletionProofAndGone(t, config.WorkRoot, request.ExecutionRef, record)

	seeder, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	launchRecord, runPath, found, err := seeder.loadLaunchRecord(request.ExecutionRef)
	if err != nil || !found {
		t.Fatalf("load launch: found=%v error=%v", found, err)
	}
	launch, err := launchRecord.receipt(request.ExecutionRef)
	if err != nil {
		t.Fatal(err)
	}
	stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:proof-wins-completion")
	stopHash, err := hashStopRequest(stop)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seeder.ensureStopRequest(runPath, stopHash, stop); err != nil {
		t.Fatal(err)
	}
	intent, _, err := seeder.prepareStopSignalIntent(runPath, stopHash, stop)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seeder.persistStopSignalProof(runPath, intent); err != nil {
		t.Fatal(err)
	}
	if err := seeder.Close(); err != nil {
		t.Fatal(err)
	}

	restarted := openTestAdapter(t, config)
	observation := awaitTerminal(t, restarted, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionStopped {
		t.Fatalf("completion proof beat stop proof: %+v", observation)
	}
}

func TestSupervisorInMemoryObservationFencesUnresolvedStopRequest(t *testing.T) {
	config := supervisorTestConfig(t)
	const suffix = "in-memory-stop-fence"
	request := supervisorTestRequest(suffix, "helper:delayed-success", 1024)
	runSupervisorTestSubprocess(t, supervisorOwnerTestArgument, config.WorkRoot, suffix, request.Objective)
	record := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)

	restarted := openTestAdapter(t, config)
	running, err := restarted.Observe(context.Background(), request.ExecutionRef)
	if err != nil || running.Status != ports.AgentRunning {
		t.Fatalf("initial adoption = %+v error=%v", running, err)
	}
	proofPath := awaitCompletionProofAndGone(t, config.WorkRoot, request.ExecutionRef, record)
	stopPath := filepath.Join(filepath.Dir(proofPath), "stop-observation-fence.request.json")
	if err := os.WriteFile(stopPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	observation, err := restarted.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionInterrupted {
		t.Fatalf("in-memory observation accepted completion across unresolved stop: %+v", observation)
	}
}

func TestSupervisorProofDoesNotPersistRawDiagnostic(t *testing.T) {
	config := supervisorTestConfig(t)
	config.MaxDiagnosticBytes = 64
	const suffix = "proof-diagnostic"
	request := supervisorTestRequest(suffix, "helper:delayed-failure", 1024)
	runSupervisorTestSubprocess(t, supervisorOwnerTestArgument, config.WorkRoot, suffix, request.Objective)
	record := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
	proofPath := awaitCompletionProofAndGone(t, config.WorkRoot, request.ExecutionRef, record)
	proofPayload, err := os.ReadFile(proofPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(proofPayload), "private delayed failure diagnostic") {
		t.Fatal("completion proof persisted raw diagnostic")
	}
	runDirectory := filepath.Dir(proofPath)
	entries, err := os.ReadDir(runDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		payload, readErr := os.ReadFile(filepath.Join(runDirectory, entry.Name()))
		if readErr == nil && strings.Contains(string(payload), "private delayed failure diagnostic") {
			t.Fatalf("raw diagnostic persisted in %s", entry.Name())
		}
	}
	proof := readCompletionProofFile(t, proofPath)
	if proof.DiagnosticSize != config.MaxDiagnosticBytes || !proof.DiagnosticTruncated {
		t.Fatalf("diagnostic proof metadata = %+v", proof)
	}

	restarted := openTestAdapter(t, config)
	observation := awaitTerminal(t, restarted, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeProcessFailed {
		t.Fatalf("nonzero exit after restart = %+v", observation)
	}
	for _, name := range []string{completionProofFileName} {
		if _, err := os.Stat(filepath.Join(runDirectory, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s survived terminal publication: %v", name, err)
		}
	}
}

func TestSupervisorLaunchClosesSensitiveDescriptorsWhenOwnerLockIsBusy(t *testing.T) {
	config := supervisorTestConfig(t)
	adapter := openTestAdapter(t, config)
	request := supervisorTestRequest("supervisor-owner-lock-busy", "helper:delayed-success", 1024)
	runPath := executionPath(request.ExecutionRef)
	if err := adapter.ensurePrivateDirectory("executions"); err != nil {
		t.Fatal(err)
	}
	if err := adapter.ensurePrivateDirectory(runPath); err != nil {
		t.Fatal(err)
	}
	held, err := adapter.acquireOwnerLock(runPath)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseOwnerLock(held)
	before := processFileDescriptorTargets(t)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() should expose durable failed receipt, got %v", err)
	}
	after := processFileDescriptorTargets(t)
	if len(after) != len(before) {
		t.Fatalf("owner-lock failure leaked descriptors: before=%v after=%v", before, after)
	}
	for _, target := range after {
		if strings.Contains(target, "memfd:orquesta-codex-supervisor") {
			t.Fatalf("sealed launch envelope leaked after owner-lock failure: %q", target)
		}
	}
}

func runSupervisorTestProcess(arguments []string) (int, bool) {
	if len(arguments) < 6 || arguments[len(arguments)-4] != "--" {
		return 0, false
	}
	mode := arguments[len(arguments)-3]
	if mode != supervisorOwnerTestArgument && mode != supervisorObserverTestArgument {
		return 0, false
	}
	workRoot, suffix := arguments[len(arguments)-2], arguments[len(arguments)-1]
	config, err := supervisorSubprocessConfig(workRoot)
	if err != nil {
		return 91, true
	}
	objective := "helper:delayed-success"
	if strings.Contains(suffix, "diagnostic") {
		objective = "helper:delayed-failure"
		config.MaxDiagnosticBytes = 64
	} else if strings.Contains(suffix, "forced-intent-") {
		objective = "helper:block"
	}
	adapter, err := New(config)
	if err != nil {
		return 92, true
	}
	request := supervisorTestRequest(suffix, objective, 1024)
	if mode == supervisorOwnerTestArgument {
		if _, err := adapter.Launch(context.Background(), request); err != nil {
			return 93, true
		}
		invocations := filepath.Join(
			workRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), "helper-invocations",
		)
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if info, err := os.Stat(invocations); err == nil && info.Size() == 1 {
				readyPath := filepath.Join(filepath.Dir(invocations), supervisorOwnerReadyFileName)
				if err := os.WriteFile(readyPath, []byte("ready\n"), 0o600); err != nil {
					return 96, true
				}
				select {}
			}
			time.Sleep(time.Millisecond)
		}
		return 94, true
	}
	observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
	if err != nil || (observation.Status != ports.AgentRunning && observation.Status != ports.AgentCompleted) {
		return 95, true
	}
	return 0, true
}

func supervisorSubprocessConfig(workRoot string) (Config, error) {
	executable, err := os.Executable()
	if err != nil {
		return Config{}, err
	}
	cgroupRoot, err := supervisorTestCgroupRoot()
	if err != nil {
		return Config{}, err
	}
	return Config{
		Command: executable, WorkRoot: workRoot, CgroupRoot: cgroupRoot,
		RuntimeScope:    "runtime-scope:test",
		ReasoningEffort: "medium", Timeout: 5 * time.Second,
		ProcessPipeDrainDelay: 250 * time.Millisecond, SupervisorStartTimeout: 5 * time.Second,
		MaxDiagnosticBytes:      256,
		MaxConcurrentExecutions: 4, MCPBearerTokenEnvVar: "ORQUESTA_MCP_BEARER_TOKEN",
		PromptRenderer: testPromptRenderer{}, Environment: map[string]string{"CODEX_TEST_EXACT": "present"},
		Now: func() time.Time { return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC) },
	}, nil
}

func supervisorTestConfig(t *testing.T) Config {
	t.Helper()
	root, err := supervisorTestCgroupRoot()
	if err != nil {
		t.Skipf("delegated cgroup v2 unavailable: %v", err)
	}
	controller, err := openCodexCgroupRoot(root)
	if err != nil {
		var adapterErr *Error
		if errors.As(err, &adapterErr) {
			t.Skipf("delegated cgroup v2 unavailable: %v: %v", err, adapterErr.Cause)
		}
		t.Skipf("delegated cgroup v2 unavailable: %v", err)
	}
	_ = controller.close()
	config := testConfig(t)
	config.CgroupRoot = root
	config.allowLegacyProcessControlForTests = false
	return config
}

func supervisorTestCgroupRoot() (string, error) {
	payload, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(payload))
	if !strings.HasPrefix(line, "0::/") || strings.ContainsRune(line, '\n') {
		return "", errors.New("unified cgroup v2 required")
	}
	control := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(line, "0::/"))
	return filepath.Dir(control), nil
}

func runSupervisorTestSubprocess(t *testing.T, mode, workRoot, suffix, objective string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// Objective is deliberately not transported: subprocess fixtures use the
	// fixed delayed-success objective and never carry credentials in argv.
	if objective != "helper:delayed-success" {
		// Delayed failure uses the same owner fixture with a suffix convention.
		if objective != "helper:delayed-failure" && objective != "helper:block" {
			t.Fatalf("unsupported supervisor subprocess objective %q", objective)
		}
	}
	command := exec.Command(executable, "-test.run=^$", "--", mode, workRoot, suffix)
	command.Env = []string{}
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if mode != supervisorOwnerTestArgument {
		if err := command.Run(); err != nil {
			t.Fatalf("supervisor test subprocess: %v", err)
		}
		return
	}
	if err := command.Start(); err != nil {
		t.Fatalf("supervisor test subprocess: %v", err)
	}
	executionRef, _ := goal.NewExecutionRef("execution:" + suffix)
	readyPath := filepath.Join(
		workRoot, filepath.FromSlash(executionPath(executionRef)), supervisorOwnerReadyFileName,
	)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(readyPath); err == nil {
			if err := command.Process.Kill(); err != nil {
				t.Fatalf("kill owner process: %v", err)
			}
			err := command.Wait()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("owner process did not die abruptly: %v", err)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	_ = command.Process.Kill()
	_ = command.Wait()
	t.Fatal("owner process never reached launched state")
}

func supervisorTestRequest(suffix, objective string, maxOutput int64) ports.AgentLaunchRequest {
	executionRef, _ := goal.NewExecutionRef("execution:" + suffix)
	goalRef, _ := goal.NewGoalRef("goal:" + suffix)
	workItemRef, _ := goal.NewWorkItemRef("work-item:" + suffix)
	actorRef, _ := goal.NewActorRef("actor:local-owner")
	projectRef, _ := goal.NewProjectRef("project:default")
	return ports.AgentLaunchRequest{
		ExecutionRef: executionRef, GoalRef: goalRef, WorkItemRef: workItemRef,
		PlanGeneration: 2, AppSpecGeneration: 3, ExecutionAttempt: 1,
		SpecHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ActorRef: actorRef, ProjectRef: projectRef, Objective: objective,
		PhaseRef: "phase-instance:build", PhaseKey: "phase:build",
		PhaseTemplateRef: "phase-template:program", PhaseInputRefs: []string{"input:app-spec"},
		PhaseCriterionRefs: []string{"criterion:tests-green"}, RoleKey: "role:worker",
		SkillRefs: []string{"skill:go"}, ToolRefs: []string{"tool:go-test"},
		CapabilityRefs: []string{"capability:patch"}, WriteSet: []string{"internal/adapters/agent/codex"},
		OutputContract: string(goal.OutputContractEvidenceBundle), ArtifactMediaType: "text/markdown",
		IdempotencyKey: "launch:" + suffix, MaxOutputBytes: maxOutput,
		BudgetDemand: governance.BudgetDemand{
			Ref: "demand:" + suffix,
			Resources: governance.ResourceVector{
				Tokens: 100_000, ActiveTimeNS: int64(5 * time.Second),
				ProcessSlots: 1, DiskBytes: maxOutput,
			},
		},
		SecurityCriticality: governance.SecurityCriticalityNormal,
		ReasoningEffort:     governance.ReasoningEffortMedium,
	}
}

func awaitSupervisorProcessRecord(t *testing.T, workRoot string, executionRef goal.ExecutionRef) processRecord {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		record, err := tryReadSupervisorProcessRecord(workRoot, executionRef)
		if err == nil {
			return record
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("supervisor process record unavailable")
	return processRecord{}
}

func readSupervisorProcessRecord(t *testing.T, workRoot string, executionRef goal.ExecutionRef) processRecord {
	t.Helper()
	record, err := tryReadSupervisorProcessRecord(workRoot, executionRef)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func tryReadSupervisorProcessRecord(workRoot string, executionRef goal.ExecutionRef) (processRecord, error) {
	payload, err := os.ReadFile(filepath.Join(
		workRoot, filepath.FromSlash(executionPath(executionRef)), processFileName,
	))
	if err != nil {
		return processRecord{}, err
	}
	var record processRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return processRecord{}, err
	}
	return record, nil
}

func awaitCompletionProofAndGone(
	t *testing.T,
	workRoot string,
	executionRef goal.ExecutionRef,
	record processRecord,
) string {
	t.Helper()
	proofPath := filepath.Join(
		workRoot, filepath.FromSlash(executionPath(executionRef)), completionProofFileName,
	)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, proofErr := os.Stat(proofPath)
		identity, identityErr := platformInspectProcess(record)
		if proofErr == nil && identityErr == nil && identity == processIdentityGone {
			return proofPath
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("completion proof or supervisor exit unavailable")
	return ""
}

func readCompletionProofFile(t *testing.T, proofPath string) completionProof {
	t.Helper()
	payload, err := os.ReadFile(proofPath)
	if err != nil {
		t.Fatal(err)
	}
	var proof completionProof
	if err := json.Unmarshal(payload, &proof); err != nil {
		t.Fatal(err)
	}
	return proof
}

func writeCompletionProofFile(t *testing.T, proofPath string, proof completionProof) {
	t.Helper()
	payload, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proofPath, append(payload, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertSupervisorProcessIsSecretFree(t *testing.T, record processRecord) {
	t.Helper()
	commandLine, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(record.PID), "cmdline"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(commandLine), localSupervisorArgument) ||
		strings.Contains(string(commandLine), "CODEX_TEST_EXACT") ||
		strings.Contains(string(commandLine), "helper:") {
		t.Fatalf("unsafe supervisor argv = %q", commandLine)
	}
	environment, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(record.PID), "environ"))
	if err != nil {
		t.Fatal(err)
	}
	if len(environment) != 0 {
		t.Fatalf("supervisor inherited environment = %q", environment)
	}
	fdDirectory := filepath.Join("/proc", strconv.Itoa(record.PID), "fd")
	entries, err := os.ReadDir(fdDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join(fdDirectory, entry.Name()))
		if err == nil && strings.Contains(target, "memfd:orquesta-codex-supervisor") {
			t.Fatalf("supervisor retained sealed private envelope on fd %s: %q", entry.Name(), target)
		}
	}
}

func processFileDescriptorTargets(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	targets := make([]string, 0, len(entries))
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err == nil {
			targets = append(targets, target)
		}
	}
	sort.Strings(targets)
	return targets
}
