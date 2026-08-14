//go:build linux

package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

func TestReopenedLiveProcessRetainsExactSecretGuards(t *testing.T) {
	for _, test := range []struct {
		name, objective, environment string
		launchReplay                 bool
		provider                     bool
	}{
		{"launch-provider", "helper:restart-provider-leak", codexAPIKeyEnvironment + "=" + helperCredentialInitial, true, true},
		{"observe-session", "helper:restart-session-leak", helperSessionEnvironment + "=" + helperSessionBearer, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig(t)
			if test.provider {
				config.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1}
				config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
			} else {
				config.SessionResolver = sessionResolverFunc(func(_ context.Context, request ports.AgentLaunchRequest) (Session, error) {
					secret, _ := credentials.NewSecret([]byte(helperSessionBearer))
					return Session{Ref: request.SessionRef, Endpoint: "http://127.0.0.1:7777/mcp", BearerToken: secret}, nil
				})
			}
			request := testRequest(t, "recovery-"+test.name, test.objective, 1024)
			if !test.provider {
				request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:recovery-" + test.name)
			}
			command, _, _ := seedUnownedLiveProcess(t, config, request, test.environment)
			reopened := openTestAdapter(t, config)
			if test.launchReplay {
				if _, err := reopened.Launch(context.Background(), request); err != nil {
					t.Fatalf("Launch(replay): %v", err)
				}
			}
			observation := awaitTerminal(t, reopened, request.ExecutionRef)
			if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeSecretLeak || len(observation.Content) != 0 {
				t.Fatalf("recovered terminal=%+v", observation)
			}
			if err := command.Wait(); err != nil {
				t.Fatalf("Wait: %v", err)
			}
			runRoot := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)))
			if output, err := os.ReadFile(filepath.Join(runRoot, lastMessageFileName)); err != nil || len(output) != 0 {
				t.Fatalf("last message=%q error=%v", output, err)
			}
			if terminal, err := os.ReadFile(filepath.Join(runRoot, terminalFileName)); err != nil ||
				strings.Contains(string(terminal), helperCredentialInitial) ||
				strings.Contains(string(terminal), helperSessionBearer) {
				t.Fatalf("terminal retained secret: %q error=%v", terminal, err)
			}
		})
	}
}

func TestRecoveryFailureQuarantinesLiveProcessBeforeFinalScrub(t *testing.T) {
	for _, test := range []struct {
		name, secret, environment, code string
		launch                          bool
	}{
		{"launch-session", helperSessionBearer, helperSessionEnvironment + "=" + helperSessionBearer, CodeSessionUnavailable, true},
		{"observe-provider", helperCredentialInitial, codexAPIKeyEnvironment + "=" + helperCredentialInitial, CodeCredentialUnavailable, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			helperPath := filepath.Join(t.TempDir(), "late-secret.sh")
			script := "#!/bin/sh\nsecret=\"${CODEX_API_KEY:-${ORQUESTA_MCP_BEARER_TOKEN:-}}\"\n" +
				"trap 'printf \"%s\" \"$secret\" > " + lastMessageFileName +
				"; printf late > recovery-late.marker; exit 0' TERM\n" +
				"printf ready > recovery-ready.marker\nwhile :; do sleep 1; done\n"
			if err := os.WriteFile(helperPath, []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			config := testConfig(t)
			config.Command = helperPath
			request := testRequest(t, "quarantine-"+test.name, "late secret after recovery scrub", 1024)
			if test.launch {
				request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:quarantine-launch")
				config.SessionResolver = sessionResolverFunc(func(context.Context, ports.AgentLaunchRequest) (Session, error) {
					return Session{}, errors.New("authority unavailable")
				})
			} else {
				config.CredentialStore = &credentialTestStore{material: test.secret, version: 1, revoked: true}
				config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
			}
			command, process, _ := seedUnownedLiveProcess(t, config, request, test.environment)
			runRoot := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)))
			awaitPath(t, filepath.Join(runRoot, "recovery-ready.marker"))
			outputPath := filepath.Join(runRoot, lastMessageFileName)
			if err := os.WriteFile(outputPath, []byte("before-first-scrub"), 0o600); err != nil {
				t.Fatal(err)
			}
			reopened := openTestAdapter(t, config)
			if test.launch {
				conflict := request
				conflict.Objective += " conflicting replay"
				if _, err := reopened.Launch(context.Background(), conflict); ErrorCode(err) != CodeExecutionConflict {
					t.Fatalf("conflicting Launch error=%v", err)
				}
				if output, _ := os.ReadFile(outputPath); string(output) != "before-first-scrub" {
					t.Fatalf("conflicting Launch touched output=%q", output)
				}
				if identity, _ := platformInspectProcess(process); identity != processIdentityAlive {
					t.Fatalf("conflicting Launch process identity=%v", identity)
				}
			}
			var err error
			if test.launch {
				_, err = reopened.Launch(context.Background(), request)
			} else {
				_, err = reopened.Observe(context.Background(), request.ExecutionRef)
			}
			if ErrorCode(err) != test.code {
				t.Fatalf("recovery error=%v code=%q", err, ErrorCode(err))
			}
			_ = command.Wait()
			awaitProcessIdentityGone(t, process)
			if _, err := os.Stat(filepath.Join(runRoot, "recovery-late.marker")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("quarantine allowed late TERM handler output: %v", err)
			}
			if output, err := os.ReadFile(outputPath); err != nil || len(output) != 0 {
				t.Fatalf("quarantine output=%q error=%v", output, err)
			}
			assertNoMaterialInTree(t, config.WorkRoot, helperCredentialInitial, helperSessionBearer)
			replay := openTestAdapter(t, config)
			if _, err := replay.Launch(context.Background(), request); err != nil {
				t.Fatalf("terminal Launch replay: %v", err)
			}
			if observation, err := replay.Observe(context.Background(), request.ExecutionRef); err != nil ||
				observation.Status != ports.AgentFailed || observation.ErrorCode != test.code || len(observation.Content) != 0 {
				t.Fatalf("terminal replay=%+v error=%v", observation, err)
			}
		})
	}
}

func TestCanceledRecoveryDoesNotQuarantineLiveProcess(t *testing.T) {
	for _, launch := range []bool{true, false} {
		name := "observe"
		if launch {
			name = "launch"
		}
		t.Run(name, func(t *testing.T) {
			config := processTreeTestConfig(t)
			ctx, cancel := context.WithCancel(context.Background())
			config.SessionResolver = sessionResolverFunc(func(context.Context, ports.AgentLaunchRequest) (Session, error) {
				cancel()
				return Session{}, ctx.Err()
			})
			request := testRequest(t, "canceled-recovery-"+name, "canceled recovery process tree", 1024)
			request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:canceled-recovery-" + name)
			command, process, _ := seedUnownedLiveProcess(t, config, request)
			reopened := openTestAdapter(t, config)
			var err error
			if launch {
				_, err = reopened.Launch(ctx, request)
			} else {
				_, err = reopened.Observe(ctx, request.ExecutionRef)
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("recovery error=%v", err)
			}
			if identity, inspectErr := platformInspectProcess(process); inspectErr != nil || identity != processIdentityAlive {
				t.Fatalf("canceled recovery process identity=%v error=%v", identity, inspectErr)
			}
			terminalPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), terminalFileName)
			if _, err := os.Stat(terminalPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("canceled recovery invented terminal: %v", err)
			}
			_ = platformSignalProcess(process, ports.AgentStopForced)
			_ = command.Wait()
			awaitProcessIdentityGone(t, process)
		})
	}
}

func TestPromptFailureDoesNotQuarantineLiveProcess(t *testing.T) {
	config := processTreeTestConfig(t)
	request := testRequest(t, "prompt-failure-recovery", "prompt failure process tree", 1024)
	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:prompt-failure-recovery")
	command, process, _ := seedUnownedLiveProcess(t, config, request)
	config.SessionResolver = sessionResolverFunc(func(context.Context, ports.AgentLaunchRequest) (Session, error) {
		secret, _ := credentials.NewSecret([]byte(helperSessionBearer))
		return Session{Ref: request.SessionRef, Endpoint: "http://127.0.0.1:7777/mcp", BearerToken: secret}, nil
	})
	config.PromptRenderer = testPromptRenderer{render: func(AgentPrompt) (string, error) {
		return "", errors.New("render unavailable")
	}}
	reopened := openTestAdapter(t, config)
	if _, err := reopened.Launch(context.Background(), request); ErrorCode(err) != CodePromptRenderFailed {
		t.Fatalf("prompt recovery error=%v code=%q", err, ErrorCode(err))
	}
	if identity, inspectErr := platformInspectProcess(process); inspectErr != nil || identity != processIdentityAlive {
		t.Fatalf("prompt failure process identity=%v error=%v", identity, inspectErr)
	}
	terminalPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), terminalFileName)
	if _, err := os.Stat(terminalPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("prompt failure invented terminal: %v", err)
	}
	_ = platformSignalProcess(process, ports.AgentStopForced)
	_ = command.Wait()
	awaitProcessIdentityGone(t, process)
}

func TestLegacyLiveReplayQuarantinesWithoutInventingAuthority(t *testing.T) {
	config := processTreeTestConfig(t)
	request := testRequest(t, "legacy-authority-binding", "legacy authority process tree", 1024)
	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:legacy-authority-binding")
	command, process, _ := seedUnownedLiveProcess(t, config, request)
	_ = awaitGrandchildPID(t, config, request)
	runPath := executionPath(request.ExecutionRef)
	requestPath := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), requestFileName)
	currentPayload, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var current launchRecord
	if err := json.Unmarshal(currentPayload, &current); err != nil {
		t.Fatal(err)
	}
	legacyHash, err := hashLegacyLaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	legacy := persistedLaunchRecordV3{
		SchemaVersion: legacyStateSchemaVersion, RequestHash: legacyHash,
		ExecutionRef: current.ExecutionRef, SpecHash: current.SpecHash, ProviderRef: current.ProviderRef,
		ExternalRef: current.ExternalRef, IdempotencyKey: current.IdempotencyKey,
		AcceptedAt: current.AcceptedAt, MaxOutputBytes: current.MaxOutputBytes,
	}
	legacyPayload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requestPath, legacyPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	legacyProcess := process
	legacyProcess.RequestHash = legacyHash
	processPayload, err := json.Marshal(legacyProcess)
	if err != nil {
		t.Fatal(err)
	}
	processPath := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), processFileName)
	if err := os.WriteFile(processPath, processPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	config.SessionResolver = sessionResolverFunc(func(_ context.Context, candidate ports.AgentLaunchRequest) (Session, error) {
		if candidate.SessionRef != request.SessionRef || candidate.PlanGeneration != request.PlanGeneration {
			return Session{}, errors.New("authority unavailable")
		}
		secret, _ := credentials.NewSecret([]byte(helperSessionBearer))
		return Session{Ref: candidate.SessionRef, Endpoint: "http://127.0.0.1:7777/mcp", BearerToken: secret}, nil
	})
	outputPath := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), lastMessageFileName)
	if err := os.WriteFile(outputPath, []byte("untrusted replay must not scrub"), 0o600); err != nil {
		t.Fatal(err)
	}
	reopened := openTestAdapter(t, config)
	untrusted := request
	untrusted.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:legacy-authority-attacker")
	untrusted.PlanGeneration++
	if _, err := reopened.Launch(context.Background(), untrusted); ErrorCode(err) != CodeLegacyExecutionRequiresNewAttempt {
		t.Fatalf("untrusted replay error=%v code=%q", err, ErrorCode(err))
	}
	if payload, err := os.ReadFile(outputPath); err != nil || len(payload) != 0 {
		t.Fatalf("legacy quarantine left output=%q error=%v", payload, err)
	}
	_ = command.Wait()
	awaitProcessIdentityGone(t, process)
	if _, err := reopened.Launch(context.Background(), request); ErrorCode(err) != CodeLegacyExecutionRequiresNewAttempt {
		t.Fatalf("exact replay error=%v code=%q", err, ErrorCode(err))
	}
	if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), launchUpgradeFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy quarantine created upgrade: %v", err)
	}
	terminal := readPersistedTerminal(t, config, runPath)
	if terminal.RequestHash != legacyHash || terminal.Status != ports.AgentFailed {
		t.Fatalf("legacy terminal identity=%+v", terminal)
	}
}

func TestCodexSelectiveStopPreservesSiblingProcessTrees(t *testing.T) {
	config := durableControlProcessTreeTestConfig(t)
	adapter := openTestAdapter(t, config)
	requests := make([]ports.AgentLaunchRequest, 4)
	grandchildren := make([]int, 4)
	for index, suffix := range []string{"control-a", "control-b", "control-c", "control-d"} {
		requests[index] = testRequest(t, suffix, "process tree control "+suffix, 1024)
		if _, err := adapter.Launch(context.Background(), requests[index]); err != nil {
			t.Fatalf("Launch(%s) error = %v", suffix, err)
		}
		grandchildren[index] = awaitGrandchildPID(t, config, requests[index])
	}

	launch := launchReceiptForRequest(t, adapter, requests[1])
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:control-b")
	crossed := stop
	crossed.LaunchActionFence++
	stopHash, _ := hashStopRequest(stop)
	crossedHash, _ := hashStopRequest(crossed)
	if stopHash == crossedHash {
		t.Fatal("stop hash omitted launch fence")
	}
	if _, err := adapter.Stop(ctx, crossed); ports.AgentContractErrorCode(err) != "agent.stop_launch_action_fence_mismatch" {
		t.Fatalf("Stop(crossed fence) error = %v", err)
	}
	for name, mutate := range map[string]func(*ports.AgentStopRequest){
		"effect attempt": func(value *ports.AgentStopRequest) { value.StopEffectAttemptRef = "effect-attempt:codex-stop:other" },
		"action fence":   func(value *ports.AgentStopRequest) { value.StopActionFence++ },
	} {
		candidate := stop
		mutate(&candidate)
		candidateHash, err := hashStopRequest(candidate)
		if err != nil || candidateHash == stopHash {
			t.Fatalf("stop hash omitted %s: hash=%q err=%v", name, candidateHash, err)
		}
	}
	receipt, err := adapter.Stop(ctx, stop)
	if err != nil || receipt.Status != ports.AgentStopped {
		t.Fatalf("Stop(B) = %+v, %v", receipt, err)
	}
	assertProcessGoneWithESRCH(t, grandchildren[1])
	for _, index := range []int{0, 2, 3} {
		if err := syscall.Kill(grandchildren[index], 0); err != nil {
			t.Fatalf("sibling %d was stopped: %v", index, err)
		}
	}
}

func TestControlLaunchReceiptRejectsMissingDurableFence(t *testing.T) {
	adapter := &Adapter{}
	if _, err := adapter.controlLaunchReceipt(launchRecord{SchemaVersion: stateSchemaVersion}, ports.AgentStopRequest{}); ErrorCode(err) != CodeLegacyControlMetadataUnknown {
		t.Fatalf("missing launch fence error = %v", err)
	}
}

func TestReopenedStopRejectsCrossedEffectAuthorityBeforeSignal(t *testing.T) {
	for name, mutate := range map[string]func(*ports.AgentStopRequest){
		"effect-attempt": func(value *ports.AgentStopRequest) { value.StopEffectAttemptRef = "effect-attempt:codex-stop:other" },
		"action-fence":   func(value *ports.AgentStopRequest) { value.StopActionFence++ },
	} {
		t.Run(name, func(t *testing.T) {
			config := durableControlProcessTreeTestConfig(t)
			request := testRequest(t, "crossed-stop-"+name, "crossed stop authority", 1024)
			command, process, launch := seedUnownedLiveProcess(t, config, request)
			cleanupSeededProcess(t, command, process)
			stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:crossed-authority")
			requestHash, err := hashStopRequest(stop)
			if err != nil {
				t.Fatal(err)
			}
			seeder, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			runPath := executionPath(request.ExecutionRef)
			if _, err := seeder.ensureStopRequest(runPath, requestHash, stop); err != nil {
				t.Fatal(err)
			}
			if err := seeder.root.Close(); err != nil {
				t.Fatal(err)
			}

			crossed := stop
			mutate(&crossed)
			reopened := openTestAdapter(t, config)
			receipt, err := reopened.Stop(context.Background(), crossed)
			if ErrorCode(err) != CodeStopConflict || receipt != (ports.AgentStopReceipt{}) {
				t.Fatalf("Stop(crossed authority)=%+v err=%v code=%q", receipt, err, ErrorCode(err))
			}
			if identity, inspectErr := platformInspectProcess(process); inspectErr != nil || identity != processIdentityAlive {
				t.Fatalf("crossed authority touched process identity=%v err=%v", identity, inspectErr)
			}
			for _, fileName := range []string{
				stopSignalIntentName(stop.IdempotencyKey), stopSignalName(stop.IdempotencyKey),
				stopCompletionName, terminalFileName,
			} {
				if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), fileName)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("crossed authority created %s: %v", fileName, err)
				}
			}
		})
	}
}

func TestReopenedV7StopRejectsMissingDurableFenceBeforeJournalOrSignal(t *testing.T) {
	for _, preloadObservation := range []bool{false, true} {
		name := "cold"
		if preloadObservation {
			name = "observation-loaded"
		}
		t.Run(name, func(t *testing.T) {
			config := durableControlProcessTreeTestConfig(t)
			request := testRequest(t, "v7-missing-fence-"+name, "historical V7 without launch fence", 1024)
			command, process, launch := seedUnownedLiveProcess(t, config, request)
			cleanupSeededProcess(t, command, process)
			grandchild := awaitGrandchildPID(t, config, request)
			runPath := executionPath(request.ExecutionRef)
			requestPath := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), requestFileName)
			payload, err := os.ReadFile(requestPath)
			if err != nil {
				t.Fatal(err)
			}
			var historical launchRecord
			if err := json.Unmarshal(payload, &historical); err != nil ||
				historical.SchemaVersion != stateSchemaVersion || historical.LaunchActionFence == 0 {
				t.Fatalf("current V7 record=%+v err=%v", historical, err)
			}
			historical.LaunchActionFence = 0
			payload, err = json.Marshal(historical)
			if err != nil || strings.Contains(string(payload), "launch_action_fence") {
				t.Fatalf("historical V7 payload=%s err=%v", payload, err)
			}
			if err := os.WriteFile(requestPath, payload, 0o600); err != nil {
				t.Fatal(err)
			}

			adapter := openTestAdapter(t, config)
			if preloadObservation {
				observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
				if err != nil || observation.Status != ports.AgentRunning {
					t.Fatalf("Observe(V7 missing fence)=%+v err=%v", observation, err)
				}
				state := adapter.executions[request.ExecutionRef.String()]
				if state == nil || state.receipt.LaunchActionFence != 0 {
					t.Fatalf("recovered V7 state=%+v", state)
				}
			}
			stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:v7-missing-fence:"+name)
			receipt, err := adapter.Stop(context.Background(), stop)
			if ErrorCode(err) != CodeLegacyControlMetadataUnknown || receipt != (ports.AgentStopReceipt{}) {
				t.Fatalf("Stop(V7 missing fence)=%+v err=%v code=%q", receipt, err, ErrorCode(err))
			}
			if identity, inspectErr := platformInspectProcess(process); inspectErr != nil || identity != processIdentityAlive {
				t.Fatalf("Stop touched V7 process identity=%v err=%v", identity, inspectErr)
			}
			if err := syscall.Kill(grandchild, 0); err != nil {
				t.Fatalf("Stop touched V7 descendant: %v", err)
			}
			requestName, receiptName := stopRecordNames(stop.IdempotencyKey)
			for _, fileName := range []string{
				requestName, receiptName, stopSignalIntentName(stop.IdempotencyKey),
				stopSignalName(stop.IdempotencyKey), stopCompletionName, terminalFileName,
			} {
				if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), fileName)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("Stop(V7 missing fence) created %s: %v", fileName, err)
				}
			}
		})
	}
}

func TestForcedStopSettlementUsesCallerDeadlineNotSupervisorStartTimeout(t *testing.T) {
	config := testConfig(t)
	config.SupervisorStartTimeout = 5 * time.Millisecond
	adapter := openTestAdapter(t, config)

	const settlementDelay = 75 * time.Millisecond
	cleanupStarted := make(chan struct{})
	processCleanup := adapter.processCleanup
	adapter.processCleanup = func(command *exec.Cmd) error {
		close(cleanupStarted)
		time.Sleep(settlementDelay)
		return processCleanup(command)
	}

	request := testRequest(t, "forced-stop-settlement-context", "helper:block", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	launch := launchReceiptForRequest(t, adapter, request)
	stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:forced-settlement-context")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	started := time.Now()
	receipt, err := adapter.Stop(ctx, stop)
	elapsed := time.Since(started)
	if err != nil || receipt.Status != ports.AgentStopped {
		t.Fatalf("Stop() = %+v, %v after %s; want stopped", receipt, err, elapsed)
	}
	select {
	case <-cleanupStarted:
	default:
		t.Fatal("Stop() returned before delayed settlement started")
	}
	if elapsed < settlementDelay {
		t.Fatalf("Stop() returned after %s before delayed settlement %s", elapsed, settlementDelay)
	}
	if receipt.ReceiptRef == "" || receipt.ConfirmedAt.IsZero() {
		t.Fatalf("Stop() returned incomplete durable receipt: %+v", receipt)
	}
	observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
	if err != nil || observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionStopped {
		t.Fatalf("Observe() = %+v, %v; want durable stopped terminal", observation, err)
	}
}

func TestCodexSelectiveStopAdoptsAfterCrashAndRejectsReusedPID(t *testing.T) {
	t.Run("adopts exact live process", func(t *testing.T) {
		config := durableControlProcessTreeTestConfig(t)
		request := testRequest(t, "adopted-control", "adopted process tree", 1024)
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, record)
		grandchild := awaitGrandchildPID(t, config, request)

		adapter := openTestAdapter(t, config)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		receipt, err := adapter.Stop(ctx, stopRequestForLaunch(launch, ports.AgentStopForced, "stop:adopted"))
		if err != nil || receipt.Status != ports.AgentStopped {
			t.Fatalf("adopted Stop() = %+v, %v", receipt, err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityGone {
			t.Fatalf("adopted process identity = %v, %v", identity, err)
		}
	})

	t.Run("retries forced signal after crash between intent and syscall", func(t *testing.T) {
		config := durableControlProcessTreeTestConfig(t)
		request := testRequest(t, "forced-intent-before-syscall", "recover forced intent", 1024)
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, record)
		grandchild := awaitGrandchildPID(t, config, request)
		stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:forced-intent-before-syscall")
		requestHash, err := hashStopRequest(stop)
		if err != nil {
			t.Fatal(err)
		}

		seeder, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		seeder.mu.Lock()
		state, runPath, err := seeder.controlTargetLocked(stop)
		if err == nil {
			_, err = seeder.ensureStopRequest(runPath, requestHash, stop)
		}
		if err == nil {
			_, _, err = seeder.ownProcessLocked(state)
		}
		if err == nil {
			var created bool
			_, created, err = seeder.prepareStopSignalIntent(runPath, requestHash, stop)
			if err == nil && !created {
				err = errors.New("forced signal intent was not created")
			}
		}
		// Simulate process crash after intent fsync and before SIGKILL. A real
		// crash releases both owner flock and private root descriptor.
		seeder.releaseProcessOwnershipLocked(state)
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if err := seeder.root.Close(); err != nil {
			t.Fatal(err)
		}
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityAlive {
			t.Fatalf("process before forced replay = %v, %v", identity, err)
		}

		reopened := openTestAdapter(t, config)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		receipt, err := reopened.Stop(ctx, stop)
		if err != nil || receipt.Status != ports.AgentStopped {
			t.Fatalf("Stop(recovered forced intent) = %+v, %v", receipt, err)
		}
		if receipt.ReceiptRef == "" {
			t.Fatal("forced stop receipt ref missing")
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		if _, found, err := reopened.loadStopSignalProof(runPath, requestHash, stop); err != nil || !found {
			t.Fatalf("forced replay proof found=%v err=%v", found, err)
		}
		replay, err := reopened.Stop(context.Background(), stop)
		if err != nil || replay != receipt || replay.ReceiptRef != receipt.ReceiptRef {
			t.Fatalf("forced replay receipt = %+v, %v; want %+v", replay, err, receipt)
		}
	})

	t.Run("request without effect plus natural exit is not stopped", func(t *testing.T) {
		controlRoot := t.TempDir()
		readyPath := filepath.Join(controlRoot, "ready")
		releasePath := filepath.Join(controlRoot, "release")
		helperPath := filepath.Join(controlRoot, "natural-exit.sh")
		helper := "#!/bin/sh\n" +
			"printf ready > \"$CODEX_NATURAL_READY\"\n" +
			"while [ ! -f \"$CODEX_NATURAL_RELEASE\" ]; do /bin/sleep 0.01; done\n"
		if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
			t.Fatal(err)
		}
		config := testConfig(t)
		config.Command = helperPath
		config.Environment = map[string]string{
			"CODEX_NATURAL_READY": readyPath, "CODEX_NATURAL_RELEASE": releasePath,
		}
		request := testRequest(t, "request-before-natural-exit", "natural exit after request", 1024)
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, record)
		awaitPath(t, readyPath)

		stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:request-without-effect")
		requestHash, err := hashStopRequest(stop)
		if err != nil {
			t.Fatal(err)
		}
		journal, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := journal.ensureStopRequest(executionPath(request.ExecutionRef), requestHash, stop); err != nil {
			t.Fatal(err)
		}
		// Simulate crash after request fsync and before any signal.
		if err := journal.root.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(releasePath, []byte("natural\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := command.Wait(); err != nil {
			t.Fatalf("natural process exit = %v", err)
		}
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityGone {
			t.Fatalf("natural process identity = %v, %v", identity, err)
		}

		adapter := openTestAdapter(t, config)
		receipt, err := adapter.Stop(context.Background(), stop)
		if err != nil || receipt.Status != ports.AgentStopAlreadyFailed {
			t.Fatalf("Stop(after natural exit) = %+v, %v", receipt, err)
		}
		observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
		if err != nil || observation.ErrorCode != CodeExecutionInterrupted {
			t.Fatalf("Observe(after natural exit) = %+v, %v", observation, err)
		}
	})

	t.Run("durable stopped terminal survives crash before receipt", func(t *testing.T) {
		config := durableControlProcessTreeTestConfig(t)
		request := testRequest(t, "crash-after-stopped-terminal", "crash after stopped terminal", 1024)
		command, process, launch := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, process)
		grandchild := awaitGrandchildPID(t, config, request)
		stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:crash-after-terminal")
		requestHash, err := hashStopRequest(stop)
		if err != nil {
			t.Fatal(err)
		}
		seeder, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		seeder.mu.Lock()
		state, runPath, err := seeder.controlTargetLocked(stop)
		if err == nil {
			_, err = seeder.ensureStopRequest(runPath, requestHash, stop)
		}
		var record processRecord
		var identity processIdentityState
		if err == nil {
			record, identity, err = seeder.ownProcessLocked(state)
		}
		var intent stopSignalIntent
		if err == nil {
			var created bool
			intent, created, err = seeder.prepareStopSignalIntent(runPath, requestHash, stop)
			if err == nil && !created {
				err = errors.New("signal intent was not created")
			}
		}
		if err == nil && identity != processIdentityAlive {
			err = errors.New("process was not alive before stop")
		}
		if err == nil {
			err = signalProcessTree(record, ports.AgentStopForced)
		}
		var proof stopSignalProof
		if err == nil {
			proof, err = seeder.persistStopSignalProof(runPath, intent)
			state.stopProof.Store(&proof)
		}
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if gone, err := waitForExactProcess(context.Background(), record, nil); err != nil || !gone {
			t.Fatalf("waitForExactProcess() = %v, %v", gone, err)
		}
		seeder.mu.Lock()
		err = seeder.finishStoppedProcessLocked(state, proof)
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		// Crash after terminal fsync: no stop receipt was written.
		if err := seeder.root.Close(); err != nil {
			t.Fatal(err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)

		reopened := openTestAdapter(t, config)
		receipt, err := reopened.Stop(context.Background(), stop)
		if err != nil || receipt.Status != ports.AgentStopped {
			t.Fatalf("Stop(after stopped terminal crash) = %+v, %v", receipt, err)
		}
	})

	t.Run("crash after signal effect fences replay without inventing proof", func(t *testing.T) {
		config, termLog := ignoreTermTreeTestConfig(t)
		request := testRequest(t, "crash-after-signal-effect", "ignore TERM across owner crash", 1024)
		command, process, launch := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, process)
		grandchild := awaitGrandchildPID(t, config, request)
		cooperative := stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:crash-after-effect")
		requestHash, err := hashStopRequest(cooperative)
		if err != nil {
			t.Fatal(err)
		}

		seeder, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		seeder.mu.Lock()
		state, runPath, err := seeder.controlTargetLocked(cooperative)
		if err == nil {
			_, err = seeder.ensureStopRequest(runPath, requestHash, cooperative)
		}
		var record processRecord
		if err == nil {
			record, _, err = seeder.ownProcessLocked(state)
		}
		if err == nil {
			var created bool
			_, created, err = seeder.prepareStopSignalIntent(runPath, requestHash, cooperative)
			if err == nil && !created {
				err = errors.New("signal intent was not created")
			}
		}
		if err == nil {
			err = signalProcessTree(record, cooperative.Mode)
		}
		// Simulate the adapter process dying after the syscall but before proof
		// or receipt fsync. The OS would release this flock on a real crash.
		seeder.releaseProcessOwnershipLocked(state)
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if err := seeder.root.Close(); err != nil {
			t.Fatal(err)
		}
		awaitFileSize(t, termLog, 1)

		reopened := openTestAdapter(t, config)
		replayCtx, cancelReplay := context.WithTimeout(context.Background(), 50*time.Millisecond)
		replay, err := reopened.Stop(replayCtx, cooperative)
		cancelReplay()
		if err != nil || replay.Status != ports.AgentStopPending {
			t.Fatalf("Stop(replay after uncertain effect) = %+v, %v", replay, err)
		}
		alternateCtx, cancelAlternate := context.WithTimeout(context.Background(), 50*time.Millisecond)
		alternate := stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:alternate-after-uncertain-effect")
		alternateReceipt, err := reopened.Stop(alternateCtx, alternate)
		cancelAlternate()
		if err != nil || alternateReceipt.Status != ports.AgentStopPending {
			t.Fatalf("Stop(alternate after uncertain effect) = %+v, %v", alternateReceipt, err)
		}
		if payload, err := os.ReadFile(termLog); err != nil || string(payload) != "x" {
			t.Fatalf("signal log after replay = %q, %v; want exactly one signal", payload, err)
		}

		forcedCtx, cancelForced := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelForced()
		forced := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:forced-after-uncertain-effect")
		forcedReceipt, err := reopened.Stop(forcedCtx, forced)
		if err != nil || forcedReceipt.Status != ports.AgentStopped {
			t.Fatalf("Stop(forced after uncertain effect) = %+v, %v", forcedReceipt, err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
	})

	t.Run("latest uncertain intent does not erase an older durable proof", func(t *testing.T) {
		controlRoot := t.TempDir()
		readyPath := filepath.Join(controlRoot, "ready")
		releasePath := filepath.Join(controlRoot, "release")
		termLog := filepath.Join(controlRoot, "term.log")
		helperPath := filepath.Join(controlRoot, "natural-after-term.sh")
		helper := "#!/bin/sh\n" +
			"trap 'printf x >> \"$CODEX_TERM_LOG\"' TERM\n" +
			"printf ready > \"$CODEX_NATURAL_READY\"\n" +
			"while [ ! -f \"$CODEX_NATURAL_RELEASE\" ]; do /bin/sleep 0.01; done\n"
		if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
			t.Fatal(err)
		}
		config := testConfig(t)
		config.Command = helperPath
		config.Timeout = 10 * time.Second
		config.Environment = map[string]string{
			"CODEX_TERM_LOG": termLog, "CODEX_NATURAL_READY": readyPath, "CODEX_NATURAL_RELEASE": releasePath,
		}
		request := testRequest(t, "proof-before-uncertain-intent", "natural exit after durable proof", 1024)
		command, process, launch := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, process)
		awaitPath(t, readyPath)

		seeder, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		cooperative := stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:durable-cooperative-proof")
		cooperativeCtx, cancelCooperative := context.WithTimeout(context.Background(), 50*time.Millisecond)
		cooperativeReceipt, err := seeder.Stop(cooperativeCtx, cooperative)
		cancelCooperative()
		if err != nil || cooperativeReceipt.Status != ports.AgentStopPending {
			t.Fatalf("Stop(durable cooperative) = %+v, %v", cooperativeReceipt, err)
		}
		awaitFileSize(t, termLog, 1)

		forced := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:uncertain-forced-intent")
		forcedHash, err := hashStopRequest(forced)
		if err != nil {
			t.Fatal(err)
		}
		seeder.mu.Lock()
		state, runPath, err := seeder.controlTargetLocked(forced)
		if err == nil {
			_, err = seeder.ensureStopRequest(runPath, forcedHash, forced)
		}
		if err == nil {
			var created bool
			_, created, err = seeder.prepareStopSignalIntent(runPath, forcedHash, forced)
			if err == nil && !created {
				err = errors.New("forced signal intent was not created")
			}
		}
		// Simulate crash before the newer forced syscall. The older cooperative
		// proof remains durable evidence even though the newer intent is the
		// at-most-once fence.
		seeder.releaseProcessOwnershipLocked(state)
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if err := seeder.root.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(releasePath, []byte("natural\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := command.Wait(); err != nil {
			t.Fatalf("natural exit = %v", err)
		}

		reopened := openTestAdapter(t, config)
		observation, err := reopened.Observe(context.Background(), request.ExecutionRef)
		if err != nil || observation.ErrorCode != CodeExecutionStopped {
			t.Fatalf("Observe(highest durable proof) = %+v, %v", observation, err)
		}
		winnerReceipt, err := reopened.Stop(context.Background(), cooperative)
		if err != nil || winnerReceipt.Status != ports.AgentStopped {
			t.Fatalf("Stop(cooperative winner) = %+v, %v", winnerReceipt, err)
		}
		loserReceipt, err := reopened.Stop(context.Background(), forced)
		if err != nil || loserReceipt.Status != ports.AgentStopAlreadyStopped {
			t.Fatalf("Stop(uncertain forced loser) = %+v, %v", loserReceipt, err)
		}
		if payload, err := os.ReadFile(termLog); err != nil || string(payload) != "x" {
			t.Fatalf("cooperative signal log = %q, %v; want one signal", payload, err)
		}
	})

	t.Run("leader gone with live child stays owned until forced", func(t *testing.T) {
		config := testConfig(t)
		config.Timeout = 10 * time.Second
		request := testRequest(t, "leader-gone-child-live", "helper:background-success", 1024)
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, record)
		childPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), backgroundSuccessPIDFile)
		child := awaitPIDFile(t, childPath)
		// Do not reap the leader yet: the detached child inherited the test
		// helper's diagnostic descriptor, so exec.Cmd.Wait cannot return until
		// that child exits. A zombie leader is already identity-gone and must not
		// hide the still-live process-group member.
		awaitProcessIdentityGone(t, record)

		adapter := openTestAdapter(t, config)
		observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
		if err != nil || observation.Status != ports.AgentRunning {
			t.Fatalf("Observe(live child) = %+v, %v", observation, err)
		}
		busyConfig := config
		busyConfig.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1, revoked: true}
		busyConfig.CredentialRef = credentials.CredentialRef("credential:codex-primary")
		contender, err := New(busyConfig)
		if err != nil {
			t.Fatal(err)
		}
		if lock, err := contender.acquireOwnerLock(executionPath(request.ExecutionRef)); lock != nil || !errors.Is(err, errOwnerLockBusy) {
			t.Fatalf("owner lock after leader exit = %v, %v", lock, err)
		}
		if _, err := contender.Observe(context.Background(), request.ExecutionRef); ErrorCode(err) != CodeProcessOwnershipBusy {
			t.Fatalf("busy recovery error=%v code=%q", err, ErrorCode(err))
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(childPath), terminalFileName)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("busy recovery invented terminal: %v", err)
		}
		_ = contender.root.Close()

		forcedCtx, cancelForced := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelForced()
		forced, err := adapter.Stop(forcedCtx, stopRequestForLaunch(launch, ports.AgentStopForced, "stop:child-forced"))
		if err != nil || forced.Status != ports.AgentStopped {
			t.Fatalf("Stop(live child forced) = %+v, %v", forced, err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, child)
	})

	t.Run("forced stop escalates a pending cooperative stop", func(t *testing.T) {
		config, termLog := ignoreTermTreeTestConfig(t)
		adapter := openTestAdapter(t, config)
		request := testRequest(t, "cooperative-then-forced", "ignore TERM until forced", 1024)
		if _, err := adapter.Launch(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		grandchild := awaitGrandchildPID(t, config, request)
		launch := launchReceiptForRequest(t, adapter, request)

		started := time.Now()
		cooperative, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:cooperative-pending"))
		if err != nil || cooperative.Status != ports.AgentStopPending {
			t.Fatalf("cooperative Stop() = %+v, %v", cooperative, err)
		}
		// The real identity inspection scans /proc once before returning
		// pending. Its duration depends on host process pressure; the contract
		// forbids waiting for provider settlement, not a sub-second /proc scan.
		if elapsed := time.Since(started); elapsed > 2*time.Second {
			t.Fatalf("cooperative Stop() blocked resident caller for %s", elapsed)
		}
		if err := syscall.Kill(grandchild, 0); err != nil {
			t.Fatalf("cooperative stop unexpectedly killed tree: %v", err)
		}
		awaitFileSize(t, termLog, 1)

		replay, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:cooperative-pending"))
		if err != nil || replay.Status != ports.AgentStopPending {
			t.Fatalf("cooperative replay = %+v, %v", replay, err)
		}
		alternate, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:cooperative-alternate"))
		if err != nil || alternate.Status != ports.AgentStopPending {
			t.Fatalf("alternate cooperative Stop() = %+v, %v", alternate, err)
		}
		if payload, err := os.ReadFile(termLog); err != nil || string(payload) != "x" {
			t.Fatalf("cooperative signal log = %q, %v; want one signal", payload, err)
		}

		forcedCtx, cancelForced := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelForced()
		forced, err := adapter.Stop(forcedCtx, stopRequestForLaunch(launch, ports.AgentStopForced, "stop:forced-escalation"))
		if err != nil || forced.Status != ports.AgentStopped {
			t.Fatalf("forced Stop() = %+v, %v", forced, err)
		}
		assertProcessGoneWithESRCH(t, grandchild)
		loser, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:cooperative-pending"))
		if err != nil || loser.Status != ports.AgentStopAlreadyStopped {
			t.Fatalf("losing cooperative receipt = %+v, %v", loser, err)
		}
	})

	t.Run("supervisor proof bypasses obsolete parent cleanup hook", func(t *testing.T) {
		config := processTreeTestConfig(t)
		root, rootErr := supervisorTestCgroupRoot()
		if rootErr != nil {
			t.Skipf("delegated cgroup v2 unavailable: %v", rootErr)
		}
		controller, rootErr := openCodexCgroupRoot(root)
		if rootErr != nil {
			t.Skipf("delegated cgroup v2 unavailable: %v", rootErr)
		}
		_ = controller.close()
		config.CgroupRoot = root
		config.allowLegacyProcessControlForTests = false
		adapter := openTestAdapter(t, config)
		adapter.processCleanup = func(*exec.Cmd) error { return errors.New("test cleanup failed") }
		request := testRequest(t, "stop-cleanup-failure", "cleanup failure after forced stop", 1024)
		if _, err := adapter.Launch(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		grandchild := awaitGrandchildPID(t, config, request)
		launch := launchReceiptForRequest(t, adapter, request)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		receipt, err := adapter.Stop(ctx, stopRequestForLaunch(launch, ports.AgentStopForced, "stop:cleanup-failure"))
		if err != nil || receipt.Status != ports.AgentStopped {
			t.Fatalf("Stop(supervised cleanup) = %+v, %v", receipt, err)
		}
		observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
		if err != nil || observation.ErrorCode != CodeExecutionStopped {
			t.Fatalf("Observe(supervised cleanup) = %+v, %v", observation, err)
		}
		assertProcessGoneWithESRCH(t, grandchild)
	})

	t.Run("shutdown discovers and waits for process tree after restart", func(t *testing.T) {
		config := durableControlProcessTreeTestConfig(t)
		request := testRequest(t, "adopted-shutdown", "adopted shutdown process tree", 1024)
		command, record, _ := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, record)
		grandchild := awaitGrandchildPID(t, config, request)
		adapter := openTestAdapter(t, config)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := adapter.Shutdown(ctx); err != nil {
			t.Fatalf("Shutdown(adopted) = %v", err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityGone {
			t.Fatalf("shutdown process identity = %v, %v", identity, err)
		}
	})

	t.Run("shutdown continues after corrupt journal and stops later valid tree", func(t *testing.T) {
		config := durableControlProcessTreeTestConfig(t)
		request := testRequest(t, "shutdown-after-corrupt-journal", "valid tree after corrupt journal", 1024)
		command, record, _ := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, record)
		grandchild := awaitGrandchildPID(t, config, request)
		corruptPath := filepath.Join(config.WorkRoot, "executions", "000-corrupt", processFileName)
		if err := os.MkdirAll(filepath.Dir(corruptPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(corruptPath, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		adapter, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := adapter.Shutdown(ctx); err == nil {
			t.Fatal("Shutdown(corrupt plus valid) unexpectedly succeeded")
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityGone {
			t.Fatalf("valid process identity after best-effort shutdown = %v, %v", identity, err)
		}

		contender, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		lock, err := contender.acquireOwnerLock(executionPath(request.ExecutionRef))
		if err != nil {
			t.Fatalf("valid owner lock was not released = %v", err)
		}
		releaseOwnerLock(lock)
		_ = contender.root.Close()
	})

	t.Run("shutdown cannot report success after adopted signal failure", func(t *testing.T) {
		config := durableControlProcessTreeTestConfig(t)
		request := testRequest(t, "adopted-shutdown-signal-failure", "adopted shutdown signal failure", 1024)
		command, record, _ := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, record)
		grandchild := awaitGrandchildPID(t, config, request)
		adapter, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := adapter.Observe(context.Background(), request.ExecutionRef); err != nil {
			t.Fatal(err)
		}
		adapter.mu.Lock()
		adapter.executions[request.ExecutionRef.String()].process.BirthMarker += "-stale"
		adapter.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := adapter.Shutdown(ctx); ErrorCode(err) != CodeProcessIdentityMismatch {
			t.Fatalf("Shutdown(signal failure) = %v, code = %q", err, ErrorCode(err))
		}
		if err := syscall.Kill(grandchild, 0); err != nil {
			t.Fatalf("failed shutdown unexpectedly stopped target: %v", err)
		}
		if err := platformSignalProcess(record, ports.AgentStopForced); err != nil {
			t.Fatalf("manual cleanup signal = %v", err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		adapter.mu.Lock()
		adapter.releaseProcessOwnershipLocked(adapter.executions[request.ExecutionRef.String()])
		adapter.mu.Unlock()
	})

	t.Run("stop journal read errors surface", func(t *testing.T) {
		adapter, err := New(testConfig(t))
		if err != nil {
			t.Fatal(err)
		}
		if err := adapter.root.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := adapter.hasDurableStopRequest("executions/unreadable"); ErrorCode(err) != CodeStateInvalid {
			t.Fatalf("hasDurableStopRequest() = %v, code = %q", err, ErrorCode(err))
		}
	})

	t.Run("rejects reused pid marker", func(t *testing.T) {
		config := durableControlProcessTreeTestConfig(t)
		request := testRequest(t, "reused-pid-control", "reused pid process tree", 1024)
		request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:reused-pid")
		config.SessionResolver = sessionResolverFunc(func(context.Context, ports.AgentLaunchRequest) (Session, error) {
			return Session{}, errors.New("authority unavailable")
		})
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		cleanupSeededProcess(t, command, record)
		grandchild := awaitGrandchildPID(t, config, request)
		tampered := record
		tampered.BirthMarker += "-stale"
		processPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), processFileName)
		payload, _ := json.Marshal(tampered)
		if err := os.WriteFile(processPath, append(payload, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}

		adapter := openTestAdapter(t, config)
		_, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopForced, "stop:stale"))
		if ErrorCode(err) != CodeProcessIdentityMismatch {
			t.Fatalf("stale PID Stop() error = %v, code = %q", err, ErrorCode(err))
		}
		if err := syscall.Kill(grandchild, 0); err != nil {
			t.Fatalf("stale marker signaled target: %v", err)
		}
		recovery := openTestAdapter(t, config)
		if _, err := recovery.Observe(context.Background(), request.ExecutionRef); ErrorCode(err) != CodeProcessIdentityMismatch {
			t.Fatalf("mismatch recovery error=%v code=%q", err, ErrorCode(err))
		}
		if err := syscall.Kill(grandchild, 0); err != nil {
			t.Fatalf("mismatch recovery signaled target: %v", err)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(processPath), terminalFileName)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("mismatch recovery invented terminal: %v", err)
		}
		_ = platformSignalProcess(record, ports.AgentStopForced)
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
	})
}

func durableControlProcessTreeTestConfig(t *testing.T) Config {
	t.Helper()
	config := processTreeTestConfig(t)
	helper := "#!/bin/sh\n" +
		"/bin/sleep 600 &\n" +
		"grandchild=$!\n" +
		"printf '%s\\n' \"$grandchild\" > " + processTreePIDFile + "\n" +
		"wait \"$grandchild\"\n"
	if err := os.WriteFile(config.Command, []byte(helper), 0o700); err != nil {
		t.Fatalf("WriteFile(durable process tree helper) error = %v", err)
	}
	return config
}

func cleanupSeededProcess(t *testing.T, command *exec.Cmd, record processRecord) {
	t.Helper()
	t.Cleanup(func() {
		if err := signalProcessTree(record, ports.AgentStopForced); err != nil && !errors.Is(err, os.ErrProcessDone) {
			t.Errorf("cleanup seeded process tree: %v", err)
			return
		}
		_ = command.Wait()
	})
}

func awaitPath(t *testing.T, filePath string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filePath); err == nil {
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Stat(%s) error = %v", filePath, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("path was not published: %s", filePath)
}

func awaitFileSize(t *testing.T, filePath string, minimum int64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(filePath); err == nil && info.Size() >= minimum {
			return
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Stat(%s) error = %v", filePath, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("file did not reach %d bytes: %s", minimum, filePath)
}

func awaitPIDFile(t *testing.T, filePath string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		payload, err := os.ReadFile(filePath)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(payload)))
			if parseErr != nil || pid <= 0 {
				t.Fatalf("pid file %q is invalid: %v", payload, parseErr)
			}
			return pid
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("ReadFile(%s) error = %v", filePath, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("pid was not published: %s", filePath)
	return 0
}

func awaitProcessIdentityGone(t *testing.T, record processRecord) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		identity, err := platformInspectProcess(record)
		if err != nil {
			t.Fatalf("platformInspectProcess() error = %v", err)
		}
		if identity == processIdentityGone {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("process leader did not exit: pid=%d", record.PID)
}

func ignoreTermTreeTestConfig(t *testing.T) (Config, string) {
	t.Helper()
	controlRoot := t.TempDir()
	helperPath := filepath.Join(controlRoot, "ignore-term-tree.sh")
	termLog := filepath.Join(controlRoot, "term.log")
	helper := "#!/bin/sh\n" +
		"trap 'printf x >> \"$CODEX_TERM_LOG\"' TERM\n" +
		"( trap '' TERM; exec /bin/sleep 30 ) &\n" +
		"grandchild=$!\n" +
		"printf '%s\\n' \"$grandchild\" > " + processTreePIDFile + "\n" +
		"while kill -0 \"$grandchild\" 2>/dev/null; do wait \"$grandchild\"; done\n"
	if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
		t.Fatal(err)
	}
	config := testConfig(t)
	config.Command = helperPath
	config.Timeout = 10 * time.Second
	config.ProcessPipeDrainDelay = 10 * time.Millisecond
	config.Environment = map[string]string{"CODEX_TERM_LOG": termLog}
	return config, termLog
}

func TestCodexLaunchGateCrashNeverOrphansProcess(t *testing.T) {
	for _, afterIdentity := range []bool{false, true} {
		name := "before identity publish"
		if afterIdentity {
			name = "after identity publish"
		}
		t.Run(name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "target-started")
			commandPath := filepath.Join(t.TempDir(), "target.sh")
			if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nprintf started > \"$CODEX_GATE_MARKER\"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			config := testConfig(t)
			config.Command = commandPath
			config.Environment = map[string]string{"CODEX_GATE_MARKER": marker}
			adapter, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			runPath := "executions/gate-crash"
			if err := adapter.ensurePrivateDirectory(runPath); err != nil {
				t.Fatal(err)
			}
			command, reader, writer, err := adapter.executionCommand(context.Background(), runPath)
			if err != nil {
				t.Fatal(err)
			}
			command.Env = []string{"CODEX_GATE_MARKER=" + marker}
			command.Stdout, command.Stderr = io.Discard, io.Discard
			configureProcessGroup(command, nil)
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			_ = reader.Close()
			if afterIdentity {
				pgid, bootID, birth, err := platformCaptureProcess(command.Process.Pid)
				if err != nil {
					t.Fatal(err)
				}
				record := processRecord{
					SchemaVersion: processSchemaVersion, ExecutionRef: "execution:gate-crash",
					RequestHash: "sha256:gate-crash", RuntimeScope: config.RuntimeScope,
					PID: command.Process.Pid, PGID: pgid, BootID: bootID, BirthMarker: birth,
				}
				if err := adapter.persistProcessRecord(runPath, record); err != nil {
					t.Fatal(err)
				}
			}
			// Simulated crash: writer disappears without release token.
			_ = writer.Close()
			if err := command.Wait(); err != nil {
				t.Fatalf("gate wrapper exit = %v", err)
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("target crossed unreleased gate: %v", err)
			}
			if err := adapter.root.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("exec closes inherited gate fd", func(t *testing.T) {
		marker := filepath.Join(t.TempDir(), "fd3")
		commandPath := filepath.Join(t.TempDir(), "fd3.sh")
		script := "#!/bin/sh\nif [ -e /proc/self/fd/3 ]; then printf open; else printf closed; fi > \"$CODEX_GATE_MARKER\"\n"
		if err := os.WriteFile(commandPath, []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
		config := testConfig(t)
		config.Command = commandPath
		adapter := openTestAdapter(t, config)
		command, reader, writer, err := adapter.executionCommand(context.Background(), "unused")
		if err != nil {
			t.Fatal(err)
		}
		command.Env = []string{"CODEX_GATE_MARKER=" + marker}
		command.Stdout, command.Stderr = io.Discard, io.Discard
		configureProcessGroup(command, nil)
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		_ = reader.Close()
		if _, err := writer.Write([]byte("run\n")); err != nil {
			t.Fatal(err)
		}
		_ = writer.Close()
		if err := command.Wait(); err != nil {
			t.Fatal(err)
		}
		if payload, err := os.ReadFile(marker); err != nil || string(payload) != "closed" {
			t.Fatalf("gate fd marker = %q, %v", payload, err)
		}
	})
}

func TestCodexOwnerLockIsExclusiveAndCLOEXEC(t *testing.T) {
	adapter := openTestAdapter(t, testConfig(t))
	runPath := "executions/owner-lock-test"
	if err := adapter.ensurePrivateDirectory(runPath); err != nil {
		t.Fatal(err)
	}
	first, err := adapter.acquireOwnerLock(runPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { releaseOwnerLock(first) })
	if !platformOwnerLockCLOEXEC(first) {
		t.Fatal("owner lock lacks FD_CLOEXEC")
	}
	if second, err := adapter.acquireOwnerLock(runPath); second != nil || !errors.Is(err, errOwnerLockBusy) {
		t.Fatalf("second owner lock = %v, %v", second, err)
	}
	releaseOwnerLock(first)
	first = nil
	third, err := adapter.acquireOwnerLock(runPath)
	if err != nil {
		t.Fatalf("lock after release = %v", err)
	}
	releaseOwnerLock(third)
}

func TestCodexRestoredDatabaseCannotAdoptSourceProcess(t *testing.T) {
	config := processTreeTestConfig(t)
	config.RuntimeScope = "runtime-scope:source-db-device-inode"
	request := testRequest(t, "restored-db-control", "source process tree", 1024)
	command, record, launch := seedUnownedLiveProcess(t, config, request)
	grandchild := awaitGrandchildPID(t, config, request)

	restored := config
	restored.RuntimeScope = "runtime-scope:restored-db-device-inode"
	adapter, err := New(restored)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopForced, "stop:restored"))
	if ErrorCode(err) != CodeProcessOwnershipInvalid {
		t.Fatalf("restored scope Stop() error = %v, code = %q", err, ErrorCode(err))
	}
	if err := syscall.Kill(grandchild, 0); err != nil {
		t.Fatalf("restored scope touched source process: %v", err)
	}
	_ = platformSignalProcess(record, ports.AgentStopForced)
	_ = command.Wait()
	assertProcessGoneWithESRCH(t, grandchild)
	if err := adapter.root.Close(); err != nil {
		t.Fatal(err)
	}
}

func seedUnownedLiveProcess(t *testing.T, config Config, request ports.AgentLaunchRequest, privateEnvironment ...string) (*exec.Cmd, processRecord, ports.AgentLaunchReceipt) {
	t.Helper()
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	requestHash := mustRequestHash(t, request)
	launchRecord, runPath, created, err := adapter.ensureLaunchRecord(request, requestHash)
	if err != nil || !created {
		t.Fatalf("ensure launch created=%v err=%v", created, err)
	}
	receipt, err := launchRecord.receipt(request.ExecutionRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.prepareRuntimeFiles(runPath); err != nil {
		t.Fatal(err)
	}
	command, reader, writer, err := adapter.executionCommand(context.Background(), runPath)
	if err != nil {
		t.Fatal(err)
	}
	command.Dir = filepath.Join(config.WorkRoot, filepath.FromSlash(runPath))
	command.Env = append(append([]string(nil), adapter.environment...), privateEnvironment...)
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
		t.Fatal(err)
	}
	_ = reader.Close()
	pgid, bootID, birthMarker, err := platformCaptureProcess(command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	record := processRecord{
		SchemaVersion: processSchemaVersion, ExecutionRef: request.ExecutionRef.String(),
		RequestHash: requestHash, RuntimeScope: config.RuntimeScope,
		PID: command.Process.Pid, PGID: pgid, BootID: bootID, BirthMarker: birthMarker,
	}
	if err := adapter.persistProcessRecord(runPath, record); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("run\n")); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	releaseOwnerLock(owner)
	if err := adapter.root.Close(); err != nil {
		t.Fatal(err)
	}
	return command, record, receipt
}

func launchReceiptForRequest(t *testing.T, adapter *Adapter, request ports.AgentLaunchRequest) ports.AgentLaunchReceipt {
	t.Helper()
	record, _, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil || !found {
		t.Fatalf("load launch found=%v err=%v", found, err)
	}
	receipt, err := record.receipt(request.ExecutionRef)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func stopRequestForLaunch(launch ports.AgentLaunchReceipt, mode ports.AgentStopMode, idempotency string) ports.AgentStopRequest {
	return ports.AgentStopRequest{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef,
		PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration,
		ExecutionAttempt: launch.ExecutionAttempt, LaunchActionFence: launch.LaunchActionFence,
		StopEffectAttemptRef: "effect-attempt:codex-stop", StopActionFence: 13, SpecHash: launch.SpecHash,
		ProviderRef: launch.ProviderRef, ModelRef: launch.ModelRef, AgentRef: launch.AgentRef,
		ExternalRef: launch.ExternalRef, Mode: mode, IdempotencyKey: idempotency,
	}
}
