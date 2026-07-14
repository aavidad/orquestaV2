package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	helperExactEnvironment   = "CODEX_TEST_EXACT=present"
	backgroundSuccessPIDFile = "background-success.pid"
)

func init() {
	if len(os.Args) < 2 || os.Args[1] != "exec" {
		return
	}
	if err := runCodexHelper(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(91)
	}
	os.Exit(0)
}

func TestAdapterSuccessfulExecutionUsesHardenedCommandAndPrivateTerminal(t *testing.T) {
	config := testConfig(t)
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "success", "helper:success", 1024)

	receipt, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		t.Fatalf("receipt contract error = %v", err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentCompleted || observation.MediaType != request.ArtifactMediaType {
		t.Fatalf("observation = %+v", observation)
	}
	if got := string(observation.Content); got != "artifact:success" {
		t.Fatalf("artifact = %q", got)
	}
	if observation.ErrorCode != "" {
		t.Fatalf("completed error code = %q", observation.ErrorCode)
	}

	runDirectory := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)))
	for _, name := range []string{requestFileName, terminalFileName, outputSchemaFileName, lastMessageFileName} {
		info, err := os.Stat(filepath.Join(runDirectory, name))
		if err != nil {
			t.Fatalf("Stat(%s) error = %v", name, err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("%s mode = %o, want 600", name, got)
		}
	}
}

func TestAdapterRetriesTerminalPublicationBeforeExposingObservation(t *testing.T) {
	config := testConfig(t)
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "terminal-sync-retry", "helper:success", 1024)
	runPath := executionPath(request.ExecutionRef)
	realSync := adapter.syncDirectoryFn
	injected := errors.New("injected terminal directory sync failure")
	var failedOnce atomic.Bool
	adapter.syncDirectoryFn = func(root *os.Root, directory string) error {
		if directory == runPath && !failedOnce.Load() {
			if _, err := root.Lstat(path.Join(runPath, terminalFileName)); err == nil && failedOnce.CompareAndSwap(false, true) {
				return injected
			}
		}
		return realSync(root, directory)
	}

	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if !failedOnce.Load() || observation.Status != ports.AgentCompleted || string(observation.Content) != "artifact:success" {
		t.Fatalf("terminal retry failed: injected=%v observation=%+v", failedOnce.Load(), observation)
	}

	reopened := openTestAdapter(t, config)
	recovered, err := reopened.Observe(context.Background(), request.ExecutionRef)
	if err != nil || recovered.Status != ports.AgentCompleted || string(recovered.Content) != "artifact:success" {
		t.Fatalf("durable terminal after retry = %+v error=%v", recovered, err)
	}
}

func TestAdapterLaunchIsIdempotentAcrossRestart(t *testing.T) {
	config := testConfig(t)
	first, err := New(config)
	if err != nil {
		t.Fatalf("New(first) error = %v", err)
	}
	request := testRequest(t, "restart", "helper:success", 1024)
	firstReceipt, err := first.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(first) error = %v", err)
	}
	_ = awaitTerminal(t, first, request.ExecutionRef)
	if err := first.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	config.Now = func() time.Time { return time.Date(2030, 2, 3, 4, 5, 6, 0, time.UTC) }
	second := openTestAdapter(t, config)
	secondReceipt, err := second.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(second) error = %v", err)
	}
	if firstReceipt != secondReceipt {
		t.Fatalf("receipt changed after restart: first=%+v second=%+v", firstReceipt, secondReceipt)
	}
	observation := awaitTerminal(t, second, request.ExecutionRef)
	if observation.Status != ports.AgentCompleted || string(observation.Content) != "artifact:success" {
		t.Fatalf("restarted observation = %+v", observation)
	}

	invocationPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), "helper-invocations")
	invocations, err := os.ReadFile(invocationPath)
	if err != nil {
		t.Fatalf("ReadFile(invocations) error = %v", err)
	}
	if len(invocations) != 1 {
		t.Fatalf("helper invocations = %d, want 1", len(invocations))
	}
}

func TestAdapterRejectsExecutionPayloadConflict(t *testing.T) {
	adapter := openTestAdapter(t, testConfig(t))
	request := testRequest(t, "conflict", "helper:success", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	conflicting := request
	conflicting.Objective = "helper:success changed payload"
	if _, err := adapter.Launch(context.Background(), conflicting); ErrorCode(err) != CodeExecutionConflict {
		t.Fatalf("conflicting Launch() error = %v, code = %q", err, ErrorCode(err))
	}
}

func TestLaunchHashAndPromptCarryPlanMetadata(t *testing.T) {
	request := testRequest(t, "plan-metadata", "helper:success", 1024)
	baseHash := mustRequestHash(t, request)
	mutations := map[string]func(*ports.AgentLaunchRequest){
		"phase":  func(value *ports.AgentLaunchRequest) { value.PhaseKey = "phase:review" },
		"role":   func(value *ports.AgentLaunchRequest) { value.RoleKey = "role:reviewer" },
		"writes": func(value *ports.AgentLaunchRequest) { value.WriteSet = []string{"internal/other"} },
		"output": func(value *ports.AgentLaunchRequest) { value.OutputContract = string(goal.OutputContractArtifact) },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := request
			changed.WriteSet = append([]string(nil), request.WriteSet...)
			mutate(&changed)
			if got := mustRequestHash(t, changed); got == baseHash {
				t.Fatalf("%s omitted from launch hash", name)
			}
		})
	}
	prompt := agentPrompt(request)
	for _, value := range []string{request.PhaseKey, request.RoleKey, request.WriteSet[0], request.OutputContract} {
		if !strings.Contains(prompt, value) {
			t.Fatalf("plan metadata %q omitted from prompt: %q", value, prompt)
		}
	}
}

func TestAdapterRejectsOversizeResultBeforeParsing(t *testing.T) {
	adapter := openTestAdapter(t, testConfig(t))
	request := testRequest(t, "oversize", "helper:oversize", 128)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeOutputTooLarge || len(observation.Content) != 0 {
		t.Fatalf("oversize observation = %+v", observation)
	}
}

func TestAdapterBoundsFailureDiagnosticAndReturnsOnlyTypedCode(t *testing.T) {
	config := testConfig(t)
	config.MaxDiagnosticBytes = 64
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "failure", "helper:failure", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeProcessFailed || len(observation.Content) != 0 {
		t.Fatalf("failure observation = %+v", observation)
	}

	terminal, found, err := adapter.loadTerminal(executionPath(request.ExecutionRef), mustRequestHash(t, request), request.MaxOutputBytes)
	if err != nil || !found {
		t.Fatalf("loadTerminal() found=%v error=%v", found, err)
	}
	if len(terminal.Diagnostic) != int(config.MaxDiagnosticBytes) || !terminal.DiagnosticTruncated {
		t.Fatalf("diagnostic bytes=%d truncated=%v", len(terminal.Diagnostic), terminal.DiagnosticTruncated)
	}
	if strings.Contains(observation.ErrorCode, string(terminal.Diagnostic)) {
		t.Fatal("diagnostic leaked through domain error code")
	}
}

func TestAdapterCleanupFailureCannotProduceSuccessfulTerminal(t *testing.T) {
	adapter := openTestAdapter(t, testConfig(t))
	adapter.processCleanup = func(*exec.Cmd) error { return errors.New("injected cleanup failure") }
	request := testRequest(t, "cleanup-failure", "helper:success", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeProcessCleanupFailed || len(observation.Content) != 0 {
		t.Fatalf("cleanup failure observation = %+v", observation)
	}
}

func TestAdapterLaunchContextCancellationKillsProcess(t *testing.T) {
	adapter := openTestAdapter(t, testConfig(t))
	request := testRequest(t, "cancel", "helper:block", 1024)
	ctx, cancel := context.WithCancel(context.Background())
	if _, err := adapter.Launch(ctx, request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	cancel()
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionCanceled {
		t.Fatalf("canceled observation = %+v", observation)
	}
}

func TestAdapterShutdownKillsProcessAndPersistsTerminal(t *testing.T) {
	config := testConfig(t)
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	request := testRequest(t, "shutdown", "helper:block", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if err := adapter.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	reopened := openTestAdapter(t, config)
	observation, err := reopened.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe(reopened) error = %v", err)
	}
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionCanceled {
		t.Fatalf("shutdown observation = %+v", observation)
	}
}

func TestAdapterPassesModelOnlyWhenConfigured(t *testing.T) {
	config := testConfig(t)
	config.Model = "test-model"
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "model", "helper:model", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentCompleted {
		t.Fatalf("model observation = %+v", observation)
	}
}

func TestAdapterRejectsNonPrivateOrSymlinkWorkRoot(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}
	base := Config{
		Command:                 executable,
		ReasoningEffort:         "medium",
		Timeout:                 5 * time.Second,
		ProcessPipeDrainDelay:   250 * time.Millisecond,
		MaxDiagnosticBytes:      64,
		MaxConcurrentExecutions: 1,
		Environment:             map[string]string{"CODEX_TEST_EXACT": "present"},
		Now:                     func() time.Time { return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC) },
	}

	t.Run("permissions", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Chmod(root, 0o755); err != nil {
			t.Fatalf("Chmod() error = %v", err)
		}
		config := base
		config.WorkRoot = root
		if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeWorkRootPermissions {
			t.Fatalf("New() adapter=%v error=%v code=%q", adapter, err, ErrorCode(err))
		}
	})

	t.Run("symlink", func(t *testing.T) {
		target := t.TempDir()
		link := filepath.Join(t.TempDir(), "work-link")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlink unsupported: %v", err)
		}
		config := base
		config.WorkRoot = link
		if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeWorkRootInvalid {
			t.Fatalf("New() adapter=%v error=%v code=%q", adapter, err, ErrorCode(err))
		}
	})
}

func runCodexHelper(arguments []string) error {
	options, err := parseCodexHelperArguments(arguments)
	if err != nil {
		return err
	}
	prompt, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read prompt: %w", err)
	}
	if !strings.Contains(string(prompt), "Return only the JSON object required by the supplied schema") {
		return fmt.Errorf("structured-output instruction missing")
	}
	if err := validateHelperEnvironment(); err != nil {
		return err
	}
	schema, err := os.ReadFile(options.schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	if bytes.Contains(schema, []byte(`"summary"`)) || !bytes.Contains(schema, []byte(`"artifact"`)) {
		return fmt.Errorf("output schema missing fields")
	}
	invocations, err := os.OpenFile("helper-invocations", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open invocation marker: %w", err)
	}
	if _, err := invocations.Write([]byte{'1'}); err != nil {
		_ = invocations.Close()
		return fmt.Errorf("write invocation marker: %w", err)
	}
	if err := invocations.Close(); err != nil {
		return fmt.Errorf("close invocation marker: %w", err)
	}

	mode := string(prompt)
	switch {
	case strings.Contains(mode, "helper:background-success"):
		child := exec.Command("/bin/sleep", "30")
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			return fmt.Errorf("start background child: %w", err)
		}
		if err := os.WriteFile(backgroundSuccessPIDFile, []byte(strconv.Itoa(child.Process.Pid)), 0o600); err != nil {
			return fmt.Errorf("write background pid: %w", err)
		}
		if err := child.Process.Release(); err != nil {
			return fmt.Errorf("release background child: %w", err)
		}
		return writeHelperResult(options.outputPath, "artifact:background-success")
	case strings.Contains(mode, "helper:model"):
		if options.model != "test-model" {
			return fmt.Errorf("model = %q, want test-model", options.model)
		}
		return writeHelperResult(options.outputPath, "artifact:model")
	case strings.Contains(mode, "helper:success"):
		if options.model != "" {
			return fmt.Errorf("unexpected model %q", options.model)
		}
		_, _ = fmt.Fprint(os.Stdout, "stdout must be discarded")
		_, _ = fmt.Fprint(os.Stderr, "private success diagnostic")
		return writeHelperResult(options.outputPath, "artifact:success")
	case strings.Contains(mode, "helper:oversize"):
		return os.WriteFile(options.outputPath, bytes.Repeat([]byte{'x'}, 2048), 0o600)
	case strings.Contains(mode, "helper:failure"):
		_, _ = os.Stderr.Write(bytes.Repeat([]byte{'d'}, 4096))
		os.Exit(23)
	case strings.Contains(mode, "helper:block"):
		time.Sleep(30 * time.Second)
		return fmt.Errorf("block helper was not canceled")
	default:
		return fmt.Errorf("unknown helper objective")
	}
	return nil
}

type helperOptions struct {
	schemaPath string
	outputPath string
	model      string
}

func parseCodexHelperArguments(arguments []string) (helperOptions, error) {
	if len(arguments) == 0 || arguments[0] != "exec" {
		return helperOptions{}, fmt.Errorf("missing exec subcommand")
	}
	booleans := map[string]bool{
		"--ephemeral":           false,
		"--ignore-user-config":  false,
		"--ignore-rules":        false,
		"--skip-git-repo-check": false,
	}
	var options helperOptions
	seenPairs := make(map[string]bool)
	for index := 1; index < len(arguments); index++ {
		argument := arguments[index]
		if _, found := booleans[argument]; found {
			if booleans[argument] {
				return helperOptions{}, fmt.Errorf("duplicate flag %s", argument)
			}
			booleans[argument] = true
			continue
		}
		if argument != "--sandbox" && argument != "--output-schema" && argument != "--output-last-message" && argument != "--config" && argument != "--model" {
			return helperOptions{}, fmt.Errorf("unexpected argument %q", argument)
		}
		if seenPairs[argument] || index+1 >= len(arguments) {
			return helperOptions{}, fmt.Errorf("invalid paired flag %s", argument)
		}
		seenPairs[argument] = true
		index++
		value := arguments[index]
		switch argument {
		case "--sandbox":
			if value != "read-only" {
				return helperOptions{}, fmt.Errorf("sandbox = %q", value)
			}
		case "--output-schema":
			options.schemaPath = value
		case "--output-last-message":
			options.outputPath = value
		case "--config":
			if value != `model_reasoning_effort="medium"` {
				return helperOptions{}, fmt.Errorf("reasoning config = %q", value)
			}
		case "--model":
			options.model = value
		}
	}
	for flag, present := range booleans {
		if !present {
			return helperOptions{}, fmt.Errorf("required flag missing: %s", flag)
		}
	}
	for _, flag := range []string{"--sandbox", "--output-schema", "--output-last-message", "--config"} {
		if !seenPairs[flag] {
			return helperOptions{}, fmt.Errorf("required paired flag missing: %s", flag)
		}
	}
	if !filepath.IsAbs(options.schemaPath) || !filepath.IsAbs(options.outputPath) {
		return helperOptions{}, fmt.Errorf("runtime paths must be absolute")
	}
	return options, nil
}

func validateHelperEnvironment() error {
	payload, err := os.ReadFile("/proc/self/environ")
	if err != nil {
		return fmt.Errorf("read exact environment: %w", err)
	}
	entries := strings.Split(strings.TrimSuffix(string(payload), "\x00"), "\x00")
	sort.Strings(entries)
	if want := []string{helperExactEnvironment}; !reflect.DeepEqual(entries, want) {
		return fmt.Errorf("environment = %#v, want %#v", entries, want)
	}
	return nil
}

func writeHelperResult(outputPath, artifact string) error {
	payload, err := json.Marshal(modelResult{Artifact: artifact})
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, payload, 0o600)
}

func testConfig(t *testing.T) Config {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}
	workRoot := t.TempDir()
	if err := os.Chmod(workRoot, 0o700); err != nil {
		t.Fatalf("Chmod(work root) error = %v", err)
	}
	return Config{
		Command:                 executable,
		WorkRoot:                workRoot,
		ReasoningEffort:         "medium",
		Timeout:                 5 * time.Second,
		ProcessPipeDrainDelay:   250 * time.Millisecond,
		MaxDiagnosticBytes:      256,
		MaxConcurrentExecutions: 4,
		Environment:             map[string]string{"CODEX_TEST_EXACT": "present"},
		Now:                     func() time.Time { return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC) },
	}
}

func openTestAdapter(t *testing.T, config Config) *Adapter {
	t.Helper()
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return adapter
}

func testRequest(t *testing.T, suffix, objective string, maxOutput int64) ports.AgentLaunchRequest {
	t.Helper()
	executionRef, _ := goal.NewExecutionRef("execution:" + suffix)
	goalRef, _ := goal.NewGoalRef("goal:" + suffix)
	workItemRef, _ := goal.NewWorkItemRef("work-item:" + suffix)
	actorRef, _ := goal.NewActorRef("actor:local-owner")
	projectRef, _ := goal.NewProjectRef("project:default")
	return ports.AgentLaunchRequest{
		ExecutionRef:      executionRef,
		GoalRef:           goalRef,
		WorkItemRef:       workItemRef,
		ActorRef:          actorRef,
		ProjectRef:        projectRef,
		Objective:         objective,
		PhaseKey:          "phase:build",
		RoleKey:           "role:worker",
		WriteSet:          []string{"internal/adapters/agent/codex"},
		OutputContract:    string(goal.OutputContractEvidenceBundle),
		ArtifactMediaType: "text/markdown",
		IdempotencyKey:    "launch:" + suffix,
		MaxOutputBytes:    maxOutput,
	}
}

func awaitTerminal(t *testing.T, adapter *Adapter, executionRef goal.ExecutionRef) ports.AgentObservation {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		observation, err := adapter.Observe(context.Background(), executionRef)
		if err != nil {
			t.Fatalf("Observe() error = %v", err)
		}
		if observation.Status == ports.AgentCompleted || observation.Status == ports.AgentFailed {
			return observation
		}
		if time.Now().After(deadline) {
			t.Fatalf("execution did not become terminal: %+v", observation)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func mustRequestHash(t *testing.T, request ports.AgentLaunchRequest) string {
	t.Helper()
	hash, err := hashLaunchRequest(request)
	if err != nil {
		t.Fatalf("hashLaunchRequest() error = %v", err)
	}
	return hash
}
