//go:build linux

package codex

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestLiveV6RecoveryNeverInventsEffortAndRequiresV7Attempt(t *testing.T) {
	for _, test := range []struct {
		name, authority string
		legacyEffort    governance.ReasoningEffort
		recoveryEffort  governance.ReasoningEffort
	}{
		{"high_to_xhigh_session", "session", governance.ReasoningEffortHigh, governance.ReasoningEffortXHigh},
		{"xhigh_to_high_credential", "credential", governance.ReasoningEffortXHigh, governance.ReasoningEffortHigh},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig(t)
			config.ReasoningEffort = string(test.legacyEffort)
			legacy := testRequest(t, "live-v6-"+test.name, "helper:block", 1024)
			legacy.ReasoningEffort = test.legacyEffort

			var authorityRequests []ports.AgentLaunchRequest
			if test.authority == "session" {
				legacy.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:live-v6-" + test.name)
				config.SessionResolver = sessionResolverFunc(func(
					_ context.Context,
					request ports.AgentLaunchRequest,
				) (Session, error) {
					authorityRequests = append(authorityRequests, request)
					secret, err := credentials.NewSecret([]byte(helperSessionBearer))
					if err != nil {
						return Session{}, err
					}
					return Session{
						Ref: request.SessionRef, Endpoint: "http://127.0.0.1:7777/mcp",
						BearerToken: secret,
					}, nil
				})
			} else {
				config.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1}
				config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
			}

			command, process, runPath := seedLiveV6Process(t, config, legacy)
			t.Cleanup(func() { _ = platformSignalProcess(process, ports.AgentStopForced) })

			recoveryConfig := config
			recoveryConfig.RuntimeScope = ""
			recoveryConfig.ReasoningEffort = string(test.recoveryEffort)
			adapter, err := New(recoveryConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := adapter.BindRuntimeScope(context.Background(), config.RuntimeScope); err != nil {
				t.Fatalf("BindRuntimeScope: %v", err)
			}
			assertLiveV6Adoption(t, adapter, legacy, process)
			if test.authority == "session" {
				if len(authorityRequests) != 1 || authorityRequests[0].ReasoningEffort != "" {
					t.Fatalf("legacy recovery invented effort: %+v", authorityRequests)
				}
			} else {
				adapter.mu.Lock()
				guarded := adapter.executions[legacy.ExecutionRef.String()].credentialGuard != nil
				adapter.mu.Unlock()
				if !guarded {
					t.Fatal("legacy credential authority was not recovered")
				}
			}

			if _, err := adapter.Launch(
				context.Background(), legacy,
			); ErrorCode(err) != CodeLegacyExecutionRequiresNewAttempt {
				t.Fatalf("Launch(V6 live) error=%v code=%q", err, ErrorCode(err))
			}
			_ = command.Wait()
			if identity, err := platformInspectProcess(process); err != nil || identity != processIdentityGone {
				t.Fatalf("legacy process survived quarantine identity=%v error=%v", identity, err)
			}
			if found, err := adapter.readPrivateJSON(
				filepath.ToSlash(filepath.Join(runPath, launchUpgradeFileName)),
				&launchUpgradeRecord{},
			); err != nil || found {
				t.Fatalf("legacy quarantine persisted false V7 binding found=%v error=%v", found, err)
			}
			terminal := readPersistedTerminal(t, recoveryConfig, runPath)
			if terminal.RequestHash != process.RequestHash ||
				terminal.ErrorCode != CodeLegacyExecutionRequiresNewAttempt {
				t.Fatalf("legacy quarantine terminal=%+v process=%+v", terminal, process)
			}

			retry := newV7RetryRequest(t, legacy, "retry-"+test.name, test.legacyEffort)
			if test.authority == "session" {
				retry.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:retry-" + test.name)
				retry.Objective = "helper:session helper:reasoning:" +
					string(test.legacyEffort) + " helper:success"
			} else {
				retry.Objective = "helper:credential-initial helper:reasoning:" +
					string(test.legacyEffort)
			}
			receipt, err := adapter.Launch(context.Background(), retry)
			if err != nil {
				t.Fatalf("Launch(V7 retry): %v", err)
			}
			if observation := awaitTerminal(t, adapter, retry.ExecutionRef); observation.Status != ports.AgentCompleted {
				t.Fatalf(
					"V7 retry observation=%+v terminal=%+v",
					observation, readPersistedTerminal(t, recoveryConfig, executionPath(retry.ExecutionRef)),
				)
			}
			record, _, found, err := adapter.loadLaunchRecord(retry.ExecutionRef)
			if err != nil || !found || record.SchemaVersion != stateSchemaVersion ||
				record.ReasoningEffort != test.legacyEffort ||
				record.RequestHash != mustRequestHash(t, retry) ||
				receipt.ExecutionAttempt != legacy.ExecutionAttempt+1 {
				t.Fatalf("durable V7 retry record=%+v receipt=%+v found=%v error=%v", record, receipt, found, err)
			}
			if test.authority == "session" {
				if len(authorityRequests) != 2 ||
					authorityRequests[1].ReasoningEffort != test.legacyEffort {
					t.Fatalf("V7 authority effort=%+v", authorityRequests)
				}
			}
			if err := adapter.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown: %v", err)
			}
		})
	}
}

func TestStopControlsOneLiveV6WithoutV7Binding(t *testing.T) {
	config := testConfig(t)
	first := testRequest(t, "stop-live-v6-first", "helper:block", 1024)
	second := testRequest(t, "stop-live-v6-second", "helper:block", 1024)
	_ = seedPersistedV6Launch(t, config, first)
	_ = seedPersistedV6Launch(t, config, second)
	firstCommand, firstProcess, firstRunPath := seedLiveV6Process(t, config, first)
	secondCommand, secondProcess, _ := seedLiveV6Process(t, config, second)
	t.Cleanup(func() {
		_ = platformSignalProcess(firstProcess, ports.AgentStopForced)
		_ = platformSignalProcess(secondProcess, ports.AgentStopForced)
	})

	recoveryConfig := config
	recoveryConfig.RuntimeScope = ""
	adapter, err := New(recoveryConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.BindRuntimeScope(context.Background(), config.RuntimeScope); err != nil {
		t.Fatalf("BindRuntimeScope: %v", err)
	}
	launch := launchReceiptForRequest(t, adapter, first)
	stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:legacy-v6-individual")
	stopContext, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	stopReceipt, err := adapter.Stop(stopContext, stop)
	cancelStop()
	if err != nil || stopReceipt.Status != ports.AgentStopped {
		t.Fatalf("Stop(V6) receipt=%+v error=%v", stopReceipt, err)
	}
	_ = firstCommand.Wait()
	if identity, err := platformInspectProcess(firstProcess); err != nil || identity != processIdentityGone {
		t.Fatalf("stopped V6 identity=%v error=%v", identity, err)
	}
	if identity, err := platformInspectProcess(secondProcess); err != nil || identity != processIdentityAlive {
		t.Fatalf("unrelated V6 was touched identity=%v error=%v", identity, err)
	}
	if _, err := os.Stat(filepath.Join(
		config.WorkRoot, filepath.FromSlash(firstRunPath), launchUpgradeFileName,
	)); !os.IsNotExist(err) {
		t.Fatalf("Stop(V6) created V7 binding: %v", err)
	}
	if err := adapter.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	_ = secondCommand.Wait()
}

func TestStopV3QuarantinesWithoutFabricatingCausalMetadata(t *testing.T) {
	config := testConfig(t)
	request := testRequest(t, "stop-live-v3-unknown-metadata", "helper:block", 1024)
	legacyHash, runPath := seedPersistedV3Execution(t, config, request, nil)
	command, process, _ := seedLivePersistedProcess(t, config, request)
	t.Cleanup(func() { _ = platformSignalProcess(process, ports.AgentStopForced) })

	recoveryConfig := config
	recoveryConfig.RuntimeScope = ""
	adapter, err := New(recoveryConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.BindRuntimeScope(context.Background(), config.RuntimeScope); err != nil {
		t.Fatalf("BindRuntimeScope: %v", err)
	}
	otherGoal, _ := goal.NewGoalRef("goal:metadata-not-persisted")
	otherWorkItem, _ := goal.NewWorkItemRef("work-item:metadata-not-persisted")
	stop := ports.AgentStopRequest{
		ExecutionRef: request.ExecutionRef, GoalRef: otherGoal, WorkItemRef: otherWorkItem,
		PlanGeneration: request.PlanGeneration + 10, AppSpecGeneration: request.AppSpecGeneration + 10,
		ExecutionAttempt: request.ExecutionAttempt + 10, SpecHash: request.SpecHash,
		ProviderRef: ProviderRef, ModelRef: adapter.modelRef(), AgentRef: AgentRef,
		ExternalRef: "codex:" + filepath.Base(runPath), Mode: ports.AgentStopForced,
		IdempotencyKey: "stop:v3-unknown-causal-metadata",
	}
	stopContext, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	receipt, err := adapter.Stop(stopContext, stop)
	cancelStop()
	if ErrorCode(err) != CodeLegacyControlMetadataUnknown ||
		receipt != (ports.AgentStopReceipt{}) {
		t.Fatalf("Stop(V3 quarantine) receipt=%+v error=%v code=%q", receipt, err, ErrorCode(err))
	}
	_ = command.Wait()
	if identity, err := platformInspectProcess(process); err != nil || identity != processIdentityGone {
		t.Fatalf("V3 quarantine identity=%v error=%v", identity, err)
	}
	terminal := readPersistedTerminal(t, config, runPath)
	if terminal.RequestHash != legacyHash ||
		terminal.ErrorCode != CodeLegacyControlMetadataUnknown {
		t.Fatalf("V3 quarantine terminal=%+v", terminal)
	}
	requestName, receiptName := stopRecordNames(stop.IdempotencyKey)
	for _, name := range []string{requestName, receiptName} {
		if _, err := os.Stat(filepath.Join(
			config.WorkRoot, filepath.FromSlash(runPath), name,
		)); !os.IsNotExist(err) {
			t.Fatalf("V3 quarantine fabricated stop journal %s: %v", name, err)
		}
	}
	if err := adapter.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func seedLiveV6Process(
	t *testing.T,
	config Config,
	request ports.AgentLaunchRequest,
) (*exec.Cmd, processRecord, string) {
	t.Helper()
	runPath := executionPath(request.ExecutionRef)
	if _, err := os.Stat(filepath.Join(
		config.WorkRoot, filepath.FromSlash(runPath), requestFileName,
	)); os.IsNotExist(err) {
		seedPersistedV6Launch(t, config, request)
	} else if err != nil {
		t.Fatal(err)
	}
	return seedLivePersistedProcess(t, config, request)
}

func seedLivePersistedProcess(
	t *testing.T,
	config Config,
	request ports.AgentLaunchRequest,
) (*exec.Cmd, processRecord, string) {
	return seedLivePersistedProcessWithRequestHash(t, config, request, "")
}

func seedLivePersistedProcessWithRequestHash(
	t *testing.T,
	config Config,
	request ports.AgentLaunchRequest,
	processRequestHash string,
) (*exec.Cmd, processRecord, string) {
	t.Helper()
	runPath := executionPath(request.ExecutionRef)
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	record, _, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil || !found {
		t.Fatalf("load persisted launch found=%v record=%+v error=%v", found, record, err)
	}
	if err := adapter.prepareRuntimeFiles(runPath); err != nil {
		t.Fatal(err)
	}
	command, reader, writer, err := adapter.executionCommand(context.Background(), runPath)
	if err != nil {
		t.Fatal(err)
	}
	command.Dir = filepath.Join(config.WorkRoot, filepath.FromSlash(runPath))
	command.Env = append([]string(nil), adapter.environment...)
	prompt, err := adapter.renderAgentPrompt(request)
	if err != nil {
		t.Fatal(err)
	}
	command.Stdin, command.Stdout, command.Stderr = strings.NewReader(prompt), io.Discard, io.Discard
	configureProcessGroup(command, nil)
	owner, err := adapter.acquireOwnerLock(runPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		releaseOwnerLock(owner)
		t.Fatal(err)
	}
	_ = reader.Close()
	pgid, bootID, birthMarker, err := platformCaptureProcess(command.Process.Pid)
	if err != nil {
		releaseOwnerLock(owner)
		t.Fatal(err)
	}
	process := processRecord{
		SchemaVersion: processSchemaVersion, ExecutionRef: request.ExecutionRef.String(),
		RequestHash: record.RequestHash, RuntimeScope: config.RuntimeScope,
		PID: command.Process.Pid, PGID: pgid, BootID: bootID, BirthMarker: birthMarker,
	}
	if processRequestHash != "" {
		process.RequestHash = processRequestHash
	}
	if err := adapter.persistProcessRecord(runPath, process); err != nil {
		releaseOwnerLock(owner)
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("run\n")); err != nil {
		releaseOwnerLock(owner)
		t.Fatal(err)
	}
	_ = writer.Close()
	releaseOwnerLock(owner)
	adapter.cancelLifecycle(errAdapterShutdown)
	if err := adapter.root.Close(); err != nil {
		t.Fatal(err)
	}
	return command, process, runPath
}

func assertLiveV6Adoption(
	t *testing.T,
	adapter *Adapter,
	request ports.AgentLaunchRequest,
	process processRecord,
) {
	t.Helper()
	v6Hash, err := hashV6LaunchRequest(request, adapter.accountProfileRef())
	if err != nil {
		t.Fatal(err)
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	state := adapter.executions[request.ExecutionRef.String()]
	if state == nil || state.process == nil || *state.process != process ||
		state.requestHash != v6Hash || state.terminalRequestHash != v6Hash {
		t.Fatalf("live V6 adoption state=%+v process=%+v", state, process)
	}
}
