//go:build linux

package codex

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestLiveV6AdoptionBindsHistoricalEffortAcrossRestart(t *testing.T) {
	for _, authority := range []string{"session", "credential"} {
		t.Run(authority, func(t *testing.T) {
			config := testConfig(t)
			config.ReasoningEffort = string(governance.ReasoningEffortHigh)
			request := testRequest(t, "live-v6-"+authority, "helper:block", 1024)
			request.ReasoningEffort = governance.ReasoningEffortHigh

			var recoveredRequests []ports.AgentLaunchRequest
			if authority == "session" {
				request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:live-v6-session")
				config.SessionResolver = sessionResolverFunc(func(
					_ context.Context,
					recovery ports.AgentLaunchRequest,
				) (Session, error) {
					recoveredRequests = append(recoveredRequests, recovery)
					secret, err := credentials.NewSecret([]byte(helperSessionBearer))
					if err != nil {
						return Session{}, err
					}
					return Session{
						Ref: recovery.SessionRef, Endpoint: "http://127.0.0.1:7777/mcp",
						BearerToken: secret,
					}, nil
				})
			} else {
				config.CredentialStore = &credentialTestStore{
					material: helperCredentialInitial,
					version:  1,
				}
				config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
			}

			command, process, runPath := seedLiveV6Process(t, config, request)
			t.Cleanup(func() {
				_ = platformSignalProcess(process, ports.AgentStopForced)
			})

			recoveryConfig := config
			recoveryConfig.RuntimeScope = ""
			first, err := New(recoveryConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := first.BindRuntimeScope(context.Background(), config.RuntimeScope); err != nil {
				t.Fatalf("BindRuntimeScope(first): %v", err)
			}
			assertLiveV6Adoption(t, first, request, process, mustV6RequestHash(t, first, request))
			if authority == "session" {
				assertRecoveredHistoricalEffort(t, recoveredRequests, 1)
			} else {
				first.mu.Lock()
				guarded := first.executions[request.ExecutionRef.String()].credentialGuard != nil
				first.mu.Unlock()
				if !guarded {
					t.Fatal("V6 credential authority was not recovered")
				}
			}

			conflict := request
			conflict.ReasoningEffort = governance.ReasoningEffortXHigh
			if _, err := first.Launch(context.Background(), conflict); ErrorCode(err) != CodeExecutionConflict {
				t.Fatalf("live V6 effort change error=%v code=%q", err, ErrorCode(err))
			}
			if found, err := first.readPrivateJSON(
				filepath.ToSlash(filepath.Join(runPath, launchUpgradeFileName)),
				&launchUpgradeRecord{},
			); err != nil || found {
				t.Fatalf("conflicting live replay persisted upgrade found=%v error=%v", found, err)
			}

			wantReceipt, err := first.Launch(context.Background(), request)
			if err != nil {
				t.Fatalf("Launch(live V6): %v", err)
			}
			v7Hash := mustRequestHash(t, request)
			assertMigratedLiveState(t, first, request, process, v7Hash)
			abandonAdoptedAdapter(t, first, request)

			secondConfig := recoveryConfig
			secondConfig.ReasoningEffort = string(governance.ReasoningEffortXHigh)
			second, err := New(secondConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := second.BindRuntimeScope(context.Background(), config.RuntimeScope); err != nil {
				t.Fatalf("BindRuntimeScope(second): %v", err)
			}
			assertMigratedLiveState(t, second, request, process, v7Hash)
			if authority == "session" {
				assertRecoveredHistoricalEffort(t, recoveredRequests, 2)
			}
			gotReceipt, err := second.Launch(context.Background(), request)
			if err != nil || gotReceipt != wantReceipt {
				t.Fatalf("durable V7 replay receipt=%+v error=%v want=%+v", gotReceipt, err, wantReceipt)
			}
			if _, err := second.Launch(context.Background(), conflict); ErrorCode(err) != CodeExecutionConflict {
				t.Fatalf("durable V7 effort conflict error=%v code=%q", err, ErrorCode(err))
			}

			stop := stopRequestForLaunch(
				wantReceipt, ports.AgentStopForced, "stop:live-v6-"+authority,
			)
			stopContext, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
			stopReceipt, err := second.Stop(stopContext, stop)
			cancelStop()
			if err != nil || stopReceipt.Status != ports.AgentStopped {
				t.Fatalf("Stop(migrated V6) receipt=%+v error=%v", stopReceipt, err)
			}
			if err := second.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown(migrated V6): %v", err)
			}
			_ = command.Wait()
		})
	}
}

func seedLiveV6Process(
	t *testing.T,
	config Config,
	request ports.AgentLaunchRequest,
) (*exec.Cmd, processRecord, string) {
	t.Helper()
	runPath := seedPersistedV6Launch(t, config, request)
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	record, _, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil || !found || record.SchemaVersion != profileStateSchemaVersion {
		t.Fatalf("load V6 launch found=%v record=%+v error=%v", found, record, err)
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

func mustV6RequestHash(t *testing.T, adapter *Adapter, request ports.AgentLaunchRequest) string {
	t.Helper()
	hash, err := hashV6LaunchRequest(request, adapter.accountProfileRef())
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func assertLiveV6Adoption(
	t *testing.T,
	adapter *Adapter,
	request ports.AgentLaunchRequest,
	process processRecord,
	v6Hash string,
) {
	t.Helper()
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	state := adapter.executions[request.ExecutionRef.String()]
	if state == nil || state.process == nil || *state.process != process ||
		state.requestHash != v6Hash || state.terminalRequestHash != v6Hash {
		t.Fatalf("live V6 adoption state=%+v process=%+v", state, process)
	}
}

func assertMigratedLiveState(
	t *testing.T,
	adapter *Adapter,
	request ports.AgentLaunchRequest,
	process processRecord,
	v7Hash string,
) {
	t.Helper()
	v6Hash := mustV6RequestHash(t, adapter, request)
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	state := adapter.executions[request.ExecutionRef.String()]
	if state == nil || state.process == nil || *state.process != process ||
		state.requestHash != v7Hash || state.terminalRequestHash != v6Hash {
		t.Fatalf("migrated live state=%+v process=%+v", state, process)
	}
}

func assertRecoveredHistoricalEffort(
	t *testing.T,
	requests []ports.AgentLaunchRequest,
	wantCount int,
) {
	t.Helper()
	if len(requests) != wantCount ||
		requests[len(requests)-1].ReasoningEffort != governance.ReasoningEffortHigh {
		t.Fatalf("legacy recovery requests=%+v want count=%d effort=high", requests, wantCount)
	}
}

func abandonAdoptedAdapter(t *testing.T, adapter *Adapter, request ports.AgentLaunchRequest) {
	t.Helper()
	adapter.mu.Lock()
	state := adapter.executions[request.ExecutionRef.String()]
	if state == nil || state.process == nil || state.ownerLock == nil {
		adapter.mu.Unlock()
		t.Fatal("adopted process unavailable before simulated crash")
	}
	adapter.releaseProcessOwnershipLocked(state)
	destroyExecutionGuards(state)
	adapter.mu.Unlock()
	adapter.cancelLifecycle(errAdapterShutdown)
	if err := adapter.root.Close(); err != nil {
		t.Fatal(err)
	}
}
