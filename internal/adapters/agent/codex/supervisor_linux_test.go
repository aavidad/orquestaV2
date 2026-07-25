//go:build linux

package codex

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

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
	config := testConfig(t)
	const suffix = "supervisor-owner-crash"
	objective := "helper:delayed-success"
	if strings.Contains(suffix, "diagnostic") {
		objective = "helper:delayed-failure"
	}
	request := supervisorTestRequest(suffix, objective, 1024)

	runSupervisorTestSubprocess(t, supervisorOwnerTestArgument, config.WorkRoot, suffix, request.Objective)
	recordBefore := awaitSupervisorProcessRecord(t, config.WorkRoot, request.ExecutionRef)
	if recordBefore.SchemaVersion != supervisedProcessSchemaVersion ||
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
	config := testConfig(t)
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
			config := testConfig(t)
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
	config := testConfig(t)
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
	config := testConfig(t)
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
	config := testConfig(t)
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
	config := testConfig(t)
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
	return Config{
		Command: executable, WorkRoot: workRoot, RuntimeScope: "runtime-scope:test",
		ReasoningEffort: "medium", Timeout: 5 * time.Second,
		ProcessPipeDrainDelay: 250 * time.Millisecond, MaxDiagnosticBytes: 256,
		MaxConcurrentExecutions: 4, MCPBearerTokenEnvVar: "ORQUESTA_MCP_BEARER_TOKEN",
		PromptRenderer: testPromptRenderer{}, Environment: map[string]string{"CODEX_TEST_EXACT": "present"},
		Now: func() time.Time { return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC) },
	}, nil
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
		if objective != "helper:delayed-failure" {
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
